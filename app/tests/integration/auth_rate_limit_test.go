package integration

import (
	"fmt"
	"net/http"
	"testing"
)

// T-29 — AC-10/EC-06/EC-07/S4: brute-force rate limiting, per-username and
// per-IP, including the case where the 11th attempt uses the CORRECT
// password and is still rejected by the limiter (proving the limiter acts
// before bcrypt, not just that some rejection happens).

// S4/AC-10: 10 consecutive failed attempts against the same user, then an
// 11th attempt WITH THE CORRECT PASSWORD is still rejected by the rate
// limiter, not by credential failure.
func TestLogin_RateLimit_PerUsername_EleventhRejectedEvenWithCorrectPassword(t *testing.T) {
	srv := newAuthTestServer(t)

	// This test's own login attempts are indistinguishable to the
	// per-username limiter from any other client, so use a username not
	// touched by other tests running against this fresh server instance
	// (each test gets its own SQLite DB and in-process rate limiter).
	for i := 0; i < 10; i++ {
		res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
			"username": testUsernameA,
			"password": "wrong",
		})
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401 (still under threshold)", i+1, res.StatusCode)
		}
	}

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": testUsernameA,
		"password": testPasswordA, // correct password
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("11th attempt (correct password) status = %d, want 429 — "+
			"rate limiter must reject before bcrypt even runs", res.StatusCode)
	}
}

// EC-06: a burst against DIFFERENT usernames from the SAME IP trips the
// per-IP threshold (20/5min) even though no single username individually
// exceeds its own 10/5min threshold. Each request in this test explicitly
// sets the same Cf-Connecting-Ip header — httptest.Server's r.RemoteAddr
// varies per connection (distinct ephemeral ports), which would otherwise
// make every request look like a different IP under the RemoteAddr
// fallback and never exercise the per-IP path at all.
func TestLogin_RateLimit_PerIP_AcrossDifferentUsernames(t *testing.T) {
	srv := newAuthTestServer(t)

	const sharedIP = "198.51.100.7"
	var lastStatus int
	for i := 0; i < 21; i++ {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/auth/login", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cf-Connecting-Ip", sharedIP)
		req.Body = jsonBody(t, map[string]string{
			"username": fmt.Sprintf("no_existeix_%d", i),
			"password": "wrong",
		})

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		lastStatus = res.StatusCode
		res.Body.Close()
	}

	if lastStatus != http.StatusTooManyRequests {
		t.Fatalf("21st attempt (distinct usernames, same IP) status = %d, want 429 (per-IP threshold, EC-06)", lastStatus)
	}
}

// EC-07: a burst against the SAME username, simulated as coming from
// different IPs via Cf-Connecting-Ip, trips the per-username threshold
// (10/5min) even though no single IP individually exceeds its own
// 20/5min threshold.
func TestLogin_RateLimit_PerUsername_AcrossDifferentIPs(t *testing.T) {
	srv := newAuthTestServer(t)

	var lastStatus int
	for i := 0; i < 11; i++ {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/auth/login", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Method = http.MethodPost
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cf-Connecting-Ip", fmt.Sprintf("203.0.113.%d", i))
		req.Body = jsonBody(t, map[string]string{"username": testUsernameA, "password": "wrong"})

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		lastStatus = res.StatusCode
		res.Body.Close()
	}

	if lastStatus != http.StatusTooManyRequests {
		t.Fatalf("11th attempt (same username, distinct IPs) status = %d, want 429 (per-username threshold, EC-07)", lastStatus)
	}
}
