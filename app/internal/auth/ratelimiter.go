package auth

import (
	"sync"
	"time"
)

// rateLimitWindow is the fixed, resettable window used by RateLimiter
// (ADR-01) — both the per-user and per-IP keys share the same window
// duration; only the threshold differs.
const rateLimitWindow = 5 * time.Minute

// bucket tracks failed attempts for a single key within the current
// window. The window is a fixed, resettable counter (ADR-01) — not a
// sliding log of individual timestamps — which is sufficient for the
// exact thresholds required and much simpler to implement/test.
type bucket struct {
	count       int
	windowStart time.Time
}

// RateLimiter is an in-process, in-memory brute-force guard (ADR-01). It
// deliberately does not survive a process restart and does not coordinate
// across instances — Niu is a single process without horizontal scaling
// (PLAN.md §2.1).
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

// NewRateLimiter returns a ready-to-use RateLimiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: make(map[string]*bucket)}
}

// Allow reports whether a new attempt against key is permitted given
// limit failures within the current window. It does NOT record an
// attempt — callers call RecordFailure separately, only after a real
// failed attempt (ADR-03: Allow must be side-effect-free so validation
// failures and rate-limiter checks stay decoupled).
//
// Audit finding F-01 (NIU-4 /audit): Allow and RecordFailure used to be
// two independent lock/unlock pairs, so a caller's "check, then later
// record" sequence was not atomic across the two calls. Concurrent
// requests could all observe count < limit before any of them
// incremented it — a classic check-then-act race, the same defect class
// NIU-1's own audit found in the item-move transaction (F-02 there).
// Measured: 50 concurrent attempts against limit=10 let 11-15 through.
//
// The fix is Reserve below, which holds the lock across the read-then-
// write for the caller's whole "check and consume on failure" sequence.
// Allow is kept only for read-only inspection (tests, potential future
// UI); production code MUST use Reserve, not Allow+RecordFailure.
func (rl *RateLimiter) Allow(key string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.allowLocked(key, limit)
}

// allowLocked is Allow's logic without acquiring the lock — callers must
// already hold rl.mu.
func (rl *RateLimiter) allowLocked(key string, limit int) bool {
	b, ok := rl.buckets[key]
	if !ok {
		return true
	}
	if time.Since(b.windowStart) > rateLimitWindow {
		// Window has expired; the caller has not failed inside the current
		// window, so at this point they are allowed. The bucket itself is
		// only reset lazily on the next Reserve/Cleanup pass.
		return true
	}
	return b.count < limit
}

// Reserve atomically checks whether key is under limit and, if so,
// pre-increments its counter in the same critical section — closing the
// check-then-act race Allow+RecordFailure had (F-01). It returns true if
// the attempt is allowed (and has now been provisionally counted), false
// if the key is already rate-limited (nothing is incremented in that
// case — a rejected attempt does not consume further budget).
//
// Callers that want the failure only counted for *actually failed*
// login attempts (ADR-03: a correct password does not consume a slot)
// must call Rollback on success to undo the provisional increment.
func (rl *RateLimiter) Reserve(key string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.allowLocked(key, limit) {
		return false
	}
	b, ok := rl.buckets[key]
	if !ok || time.Since(b.windowStart) > rateLimitWindow {
		b = &bucket{windowStart: time.Now()}
		rl.buckets[key] = b
	}
	b.count++
	return true
}

// Rollback undoes a single provisional increment made by Reserve, for
// the case where the request that reserved a slot turned out to succeed
// (a correct password must not permanently consume a brute-force slot).
// It is a best-effort decrement: if the window has since rolled over or
// the bucket was cleaned up, there is nothing to undo and this is a
// no-op — never lets the count go negative.
func (rl *RateLimiter) Rollback(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok || time.Since(b.windowStart) > rateLimitWindow {
		return
	}
	if b.count > 0 {
		b.count--
	}
}

// RecordFailure increments the failure counter for key, resetting the
// window first if the previous one has already expired (ADR-01: fixed,
// resettable window, not a sliding log).
//
// Deprecated: kept only so existing unit tests that exercise bucket
// mechanics in isolation keep working. Production login code uses
// Reserve, which folds this into the same critical section as the
// limit check (F-01).
func (rl *RateLimiter) RecordFailure(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok || time.Since(b.windowStart) > rateLimitWindow {
		b = &bucket{windowStart: time.Now()}
		rl.buckets[key] = b
	}
	b.count++
}

// Cleanup removes buckets whose window has already expired, preventing
// unbounded growth of the map from usernames/IPs that stopped attacking
// (ADR-01, reused by the hourly ticker set up in cmd/niu, ADR-04).
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for key, b := range rl.buckets {
		if time.Since(b.windowStart) > rateLimitWindow {
			delete(rl.buckets, key)
		}
	}
}
