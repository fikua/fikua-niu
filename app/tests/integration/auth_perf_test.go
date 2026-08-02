package integration

import (
	"net/http"
	"testing"
	"time"
)

// T-34 — NFR-01/NFR-06: the happy-path login is bcrypt-dominated (>200ms,
// cost 12, not mocked) yet stays comfortably under the <1s p95 budget
// across several repetitions.
func TestLogin_HappyPath_TimingWithinNFRBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive test in -short mode")
	}
	if raceEnabled {
		// The race detector instruments every memory access, which
		// inflates CPU-bound work like bcrypt's cost-12 hashing by a
		// large, well-documented factor (observed here: ~2.75s under
		// -race vs ~350ms without it, on identical hardware and the same
		// binary otherwise). That is -race's known overhead on hashing
		// workloads, not a regression in login latency — NFR-06's <1s
		// budget is a real-world production number and was never meant
		// to hold under an instrumented build. Skip rather than either
		// loosen the budget (which would hide a genuine slowdown in a
		// normal build) or leave this failing every -race run.
		t.Skip("bcrypt timing is not meaningful under -race (see comment); run without -race to validate NFR-06")
	}

	srv := newAuthTestServer(t)

	const samples = 5
	durations := make([]time.Duration, 0, samples)

	for i := 0; i < samples; i++ {
		// A fresh login each time: log out (or just ignore that a session
		// accumulates — logging in again is a legitimate flow and its cost
		// is what NFR-06 cares about, session reuse is irrelevant here).
		start := time.Now()
		res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
			"username": testUsernameA,
			"password": testPasswordA,
		})
		elapsed := time.Since(start)
		res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("attempt %d: login status = %d, want 200", i+1, res.StatusCode)
		}
		durations = append(durations, elapsed)
	}

	for i, d := range durations {
		// NFR-01: bcrypt cost 12 alone costs >200ms; a login that returns
		// faster than that suggests bcrypt was skipped or badly configured.
		if d < 200*time.Millisecond {
			t.Errorf("sample %d: login took %v, want > 200ms (NFR-01: real bcrypt cost-12, not mocked)", i+1, d)
		}
		// NFR-06: even with the bcrypt cost, the whole request must stay
		// well under the 1s p95 budget.
		if d > 1*time.Second {
			t.Errorf("sample %d: login took %v, want < 1s (NFR-06)", i+1, d)
		}
	}
}
