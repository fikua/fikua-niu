package integration

import (
	"net/http"
	"testing"
)

// T-31 — EC-08/EC-09: empty/absent username or password is rejected by
// input validation and never consumes a brute-force attempt — proven by
// exhausting 10 empty-payload "attempts" and confirming a subsequent
// legitimate login still succeeds unthrottled.

func TestLogin_EmptyPassword_ValidationRejected(t *testing.T) {
	srv := newAuthTestServer(t)

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": testUsernameA,
		"password": "",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty password status = %d, want 400", res.StatusCode)
	}
	var errBody errorResponse
	decodeJSON(t, res, &errBody)
	if errBody.Error.Code != "validation_failed" {
		t.Fatalf("empty password error code = %q, want validation_failed", errBody.Error.Code)
	}
}

func TestLogin_AbsentPassword_ValidationRejected(t *testing.T) {
	srv := newAuthTestServer(t)

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": testUsernameA,
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("absent password status = %d, want 400", res.StatusCode)
	}
}

func TestLogin_EmptyUsername_ValidationRejected(t *testing.T) {
	srv := newAuthTestServer(t)

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": "",
		"password": "whatever",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty username status = %d, want 400", res.StatusCode)
	}
}

func TestLogin_AbsentUsername_ValidationRejected(t *testing.T) {
	srv := newAuthTestServer(t)

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"password": "whatever",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("absent username status = %d, want 400", res.StatusCode)
	}
}

// EC-08/EC-09/ADR-03: validation failures never touch the rate limiter —
// 10 empty-payload attempts against the same username, followed by a
// legitimate login, must still succeed (not 429).
func TestLogin_EmptyPayloadAttempts_DoNotConsumeRateLimitBudget(t *testing.T) {
	srv := newAuthTestServer(t)

	for i := 0; i < 10; i++ {
		res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
			"username": testUsernameA,
			"password": "",
		})
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("attempt %d: status = %d, want 400", i+1, res.StatusCode)
		}
	}

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": testUsernameA,
		"password": testPasswordA,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("legitimate login after 10 empty-payload attempts status = %d, want 200 — "+
			"validation failures must never consume rate-limiter budget (EC-08/EC-09)", res.StatusCode)
	}
}
