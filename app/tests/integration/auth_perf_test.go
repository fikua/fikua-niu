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
