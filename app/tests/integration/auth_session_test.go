package integration

import (
	"bytes"
	"net/http"
	"testing"
)

// T-27 — AC-04/AC-05/AC-09/EC-01/EC-02/EC-05/EC-11/S2a/S2b: session
// lifecycle around protected endpoints.

// S2a/EC-02: no Cookie header at all.
func TestProtectedEndpoint_NoCookie_Rejected(t *testing.T) {
	srv := newAuthTestServer(t)

	res := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/items", "", "", nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
	body := readAllBytes(t, res)
	if bytes.Contains(body, []byte(`"items"`)) {
		t.Fatalf("401 body leaked protected data: %s", body)
	}
}

// S2b/EC-01: a session cookie with one mutated character.
func TestProtectedEndpoint_MutatedCookie_Rejected(t *testing.T) {
	srv := newAuthTestServer(t)

	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}

	mutated := mutateOneChar(login.SessionToken)

	noCookieRes := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/items", "", "", nil)
	defer noCookieRes.Body.Close()
	noCookieBody := readAllBytes(t, noCookieRes)

	mutatedRes := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/items", mutated, "", nil)
	defer mutatedRes.Body.Close()
	mutatedBody := readAllBytes(t, mutatedRes)

	if mutatedRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("mutated-cookie status = %d, want 401", mutatedRes.StatusCode)
	}
	if mutatedRes.StatusCode != noCookieRes.StatusCode || !bytes.Equal(mutatedBody, noCookieBody) {
		t.Fatalf("mutated-cookie response (%d %s) differs from no-cookie response (%d %s) — "+
			"must not reveal a token was \"almost valid\"",
			mutatedRes.StatusCode, mutatedBody, noCookieRes.StatusCode, noCookieBody)
	}
}

func mutateOneChar(s string) string {
	if s == "" {
		return "x"
	}
	b := []byte(s)
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	return string(b)
}

// AC-04/AC-09: logout invalidates the session server-side; reusing the
// token afterward is rejected.
func TestLogout_InvalidatesSession(t *testing.T) {
	srv := newAuthTestServer(t)

	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}

	logoutReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/auth/logout", nil)
	logoutReq.Header.Set("X-CSRF-Token", login.CSRFToken)
	logoutRes, err := login.Client.Do(logoutReq)
	if err != nil {
		t.Fatalf("logout request: %v", err)
	}
	defer logoutRes.Body.Close()
	if logoutRes.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", logoutRes.StatusCode)
	}

	// EC-05: reusing the now-invalidated token is rejected.
	reuseRes := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/items", login.SessionToken, "", nil)
	defer reuseRes.Body.Close()
	if reuseRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("reusing post-logout token status = %d, want 401", reuseRes.StatusCode)
	}
}

// EC-11: a token that was once valid, but whose session row has since
// been deleted directly (simulating logout-elsewhere/expiry/cleanup),
// is rejected exactly like a token that was never valid.
func TestProtectedEndpoint_SessionDeletedServerSide_Rejected(t *testing.T) {
	srv := newAuthTestServer(t)

	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}

	if _, err := srv.Store.DB.Exec(`DELETE FROM sessions`); err != nil {
		t.Fatalf("delete sessions directly: %v", err)
	}

	res := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/items", login.SessionToken, "", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status after server-side session deletion = %d, want 401", res.StatusCode)
	}
}

// AC-05: GET /api/v1/me, another protected endpoint, is equally rejected
// without a valid session.
func TestGetMe_NoSession_Rejected(t *testing.T) {
	srv := newAuthTestServer(t)

	res := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/me", "", "", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /me without session status = %d, want 401", res.StatusCode)
	}
}
