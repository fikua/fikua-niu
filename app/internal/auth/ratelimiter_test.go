package auth

import (
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
