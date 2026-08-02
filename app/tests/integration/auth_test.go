package integration

import (
	"bytes"
	"net/http"
	"testing"
)

// T-26 — AC-01/AC-02/AC-03/AC-11/S5: successful login opens a session
// with the correct cookie attributes; wrong password and unknown username
// produce a byte-identical response body.

func TestLogin_CorrectCredentials_OpensSession(t *testing.T) {
	srv := newAuthTestServer(t)

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": testUsernameA,
		"password": testPasswordA,
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", res.StatusCode)
	}

	var sessionCookie, csrfCookie *http.Cookie
	for _, c := range res.Cookies() {
		switch c.Name {
		case "niu_session":
			sessionCookie = c
		case "niu_csrf":
			csrfCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("no niu_session cookie in login response")
	}
	if !sessionCookie.HttpOnly {
		t.Error("niu_session cookie is not HttpOnly")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("niu_session SameSite = %v, want Strict", sessionCookie.SameSite)
	}
	if sessionCookie.Path != "/" {
		t.Errorf("niu_session Path = %q, want \"/\"", sessionCookie.Path)
	}
	if csrfCookie == nil {
		t.Fatal("no niu_csrf cookie in login response")
	}
	if csrfCookie.HttpOnly {
		t.Error("niu_csrf cookie must NOT be HttpOnly (ADR-05 — JS must read it)")
	}

	// AC-01: the cookie authenticates subsequent requests.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/me", nil)
	req.AddCookie(sessionCookie)
	meRes, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /me: %v", err)
	}
	defer meRes.Body.Close()
	if meRes.StatusCode != http.StatusOK {
		t.Errorf("GET /me with fresh session cookie status = %d, want 200", meRes.StatusCode)
	}
}

func TestLogin_WrongPassword_Rejected(t *testing.T) {
	srv := newAuthTestServer(t)

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": testUsernameA,
		"password": "definitely-not-the-password",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
	for _, c := range res.Cookies() {
		if c.Name == "niu_session" && c.Value != "" {
			t.Fatalf("session cookie set on failed login: %+v", c)
		}
	}
}

func TestLogin_UnknownUsername_Rejected(t *testing.T) {
	srv := newAuthTestServer(t)

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": "no_existeix",
		"password": "whatever",
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
}

// AC-11/S5: the two failure bodies must be byte-identical — read as raw
// bytes, not individual fields.
func TestLogin_UnknownUserAndWrongPassword_ByteIdenticalBody(t *testing.T) {
	srv := newAuthTestServer(t)

	wrongPasswordRes := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": testUsernameA,
		"password": "definitely-not-the-password",
	})
	defer wrongPasswordRes.Body.Close()
	wrongPasswordBody := readAllBytes(t, wrongPasswordRes)

	unknownUserRes := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": "no_existeix",
		"password": "whatever",
	})
	defer unknownUserRes.Body.Close()
	unknownUserBody := readAllBytes(t, unknownUserRes)

	if wrongPasswordRes.StatusCode != unknownUserRes.StatusCode {
		t.Fatalf("status codes differ: wrong-password=%d unknown-user=%d",
			wrongPasswordRes.StatusCode, unknownUserRes.StatusCode)
	}
	if !bytes.Equal(wrongPasswordBody, unknownUserBody) {
		t.Fatalf("response bodies differ:\nwrong-password: %s\nunknown-user:   %s",
			wrongPasswordBody, unknownUserBody)
	}
}
