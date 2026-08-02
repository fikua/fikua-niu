package auth

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// T-24 — AC-10/EC-06/EC-07/NFR-05: bucket window expiry, per-key
// independence, and the exact off-by-one at the threshold.

func TestRateLimiter_TenthAllowedEleventhRejected(t *testing.T) {
	rl := NewRateLimiter()
	const limit = 10

	for i := 0; i < limit; i++ {
		if !rl.Allow("user", limit) {
			t.Fatalf("attempt %d: Allow() = false, want true (under threshold)", i+1)
		}
		rl.RecordFailure("user")
	}

	// The 10th failure has just been recorded; the 11th attempt must be
	// rejected without ever reaching bcrypt.
	if rl.Allow("user", limit) {
		t.Fatal("11th Allow() = true, want false (threshold exceeded)")
	}
}

func TestRateLimiter_PerUserAndPerIPAreIndependent(t *testing.T) {
	rl := NewRateLimiter()
	const limit = 10

	for i := 0; i < limit; i++ {
		rl.RecordFailure("usuari_a")
	}
	if rl.Allow("usuari_a", limit) {
		t.Fatal("usuari_a: Allow() = true after exceeding threshold, want false")
	}

	// A different key must be entirely unaffected.
	if !rl.Allow("usuari_b", limit) {
		t.Fatal("usuari_b: Allow() = false, want true — blocking one key must not block another")
	}
	if !rl.Allow("203.0.113.5", limit) {
		t.Fatal("IP key: Allow() = false, want true — blocking a username key must not block an IP key")
	}
}

func TestRateLimiter_WindowExpiryResets(t *testing.T) {
	rl := NewRateLimiter()
	const limit = 10

	for i := 0; i < limit; i++ {
		rl.RecordFailure("user")
	}
	if rl.Allow("user", limit) {
		t.Fatal("Allow() = true immediately after exceeding threshold, want false")
	}

	// Simulate the window having elapsed by backdating the bucket
	// directly — waiting 5 real minutes in a unit test is not
	// reasonable.
	rl.mu.Lock()
	rl.buckets["user"].windowStart = time.Now().Add(-6 * time.Minute)
	rl.mu.Unlock()

	if !rl.Allow("user", limit) {
		t.Fatal("Allow() = false after the window elapsed, want true (fixed resettable window, ADR-01)")
	}
}

func TestRateLimiter_Cleanup_RemovesExpiredBucketsOnly(t *testing.T) {
	rl := NewRateLimiter()
	rl.RecordFailure("stale")
	rl.RecordFailure("fresh")

	rl.mu.Lock()
	rl.buckets["stale"].windowStart = time.Now().Add(-6 * time.Minute)
	rl.mu.Unlock()

	rl.Cleanup()

	rl.mu.Lock()
	_, staleStillPresent := rl.buckets["stale"]
	_, freshStillPresent := rl.buckets["fresh"]
	rl.mu.Unlock()

	if staleStillPresent {
		t.Error("Cleanup() left an expired bucket in place")
	}
	if !freshStillPresent {
		t.Error("Cleanup() removed a bucket still inside its window")
	}
}

// TestRateLimiter_ReserveIsAtomicUnderConcurrency is the regression test
// for audit finding F-01: Allow and RecordFailure used to be two
// independent lock/unlock pairs, so concurrent callers could all observe
// "under the limit" before any of them recorded a failure — a
// check-then-act race that let more attempts through than the configured
// threshold (measured: 11-15 admitted against limit=10 with 50 concurrent
// callers, before this fix). Reserve folds the check and the increment
// into one critical section, so the same scenario must now admit exactly
// the configured limit, no matter how many goroutines race for it.
//
// Run with -race to also confirm there is no data race in addition to no
// logic race.
func TestRateLimiter_ReserveIsAtomicUnderConcurrency(t *testing.T) {
	const limit = 10
	const attackers = 50
	// A pure "launch 50 goroutines and Wait" pattern lets the Go scheduler
	// serialise most of them before they ever reach the racy section, so
	// the original check-then-act bug rarely manifested even when it was
	// present (empirically: 0/20 failures on the buggy code without this
	// barrier). A shared start gate forces every goroutine to attempt
	// Reserve at effectively the same instant, and repeating the whole
	// burst raises the odds further that a real race gets caught if one
	// exists — this combination reliably failed on the pre-fix
	// implementation across repeated local runs.
	const rounds = 20

	for round := 0; round < rounds; round++ {
		rl := NewRateLimiter()
		var start sync.WaitGroup
		start.Add(1)
		var wg sync.WaitGroup
		var admitted int64
		wg.Add(attackers)
		for i := 0; i < attackers; i++ {
			go func() {
				defer wg.Done()
				start.Wait()
				if rl.Reserve("victim", limit) {
					atomic.AddInt64(&admitted, 1)
				}
			}()
		}
		start.Done() // release all goroutines at once
		wg.Wait()

		if admitted != limit {
			t.Fatalf("round %d: Reserve() admitted %d concurrent attempts against a limit of %d — the check-then-act race is back (F-01)",
				round, admitted, limit)
		}

		// The limit must still hold for a subsequent sequential attempt:
		// all of the limit's budget was consumed by the concurrent burst.
		if rl.Reserve("victim", limit) {
			t.Fatalf("round %d: Reserve() admitted an attempt beyond the limit after the concurrent burst already exhausted it", round)
		}
	}
}

// TestRateLimiter_RollbackUndoesReservation covers the success path: a
// correct password must not permanently consume a brute-force slot
// (AC-10/ADR-03), so Reserve's provisional increment must be fully
// undone by Rollback.
func TestRateLimiter_RollbackUndoesReservation(t *testing.T) {
	rl := NewRateLimiter()
	const limit = 3

	// Reserve up to the limit, rolling each one back immediately — like a
	// user who mistypes and then succeeds, repeatedly.
	for i := 0; i < limit*2; i++ {
		if !rl.Reserve("user", limit) {
			t.Fatalf("iteration %d: Reserve() = false, want true — Rollback should have kept this key under the limit indefinitely", i)
		}
		rl.Rollback("user")
	}

	// A reservation that is NOT rolled back (a genuine failure) must
	// still count normally afterwards.
	for i := 0; i < limit; i++ {
		if !rl.Reserve("user2", limit) {
			t.Fatalf("attempt %d: Reserve() = false, want true (under threshold)", i+1)
		}
	}
	if rl.Reserve("user2", limit) {
		t.Fatal("Reserve() admitted an attempt beyond the limit when none of the prior ones were rolled back")
	}
}
