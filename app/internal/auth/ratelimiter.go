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
func (rl *RateLimiter) Allow(key string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		return true
	}
	if time.Since(b.windowStart) > rateLimitWindow {
		// Window has expired; the caller has not failed inside the current
		// window, so at this point they are allowed. The bucket itself is
		// only reset lazily on the next RecordFailure/Cleanup pass.
		return true
	}
	return b.count < limit
}

// RecordFailure increments the failure counter for key, resetting the
// window first if the previous one has already expired (ADR-01: fixed,
// resettable window, not a sliding log).
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
