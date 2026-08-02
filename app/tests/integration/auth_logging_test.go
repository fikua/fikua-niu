package integration

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

// T-35 — NFR-07: a failed login and a rate-limited login are both logged
// with username + result + IP, and NEVER the plaintext password or the
// session token.

// captureSlog redirects the default slog logger to an in-memory buffer for
// the duration of the test, restoring the previous default afterward —
// same mechanism NIU-1 already relies on for its own log-based assertions
// (chi's request logger uses the same default logger).
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return buf
}

func TestLoginLogging_FailedAttempt_NeverLogsPassword(t *testing.T) {
	buf := captureSlog(t)
	srv := newAuthTestServer(t)

	const secretPassword = "this-must-never-appear-in-logs"
	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": testUsernameA,
		"password": secretPassword,
	})
	res.Body.Close()

	output := buf.String()
	if !strings.Contains(output, "login attempt") {
		t.Fatalf("no \"login attempt\" log entry found; output:\n%s", output)
	}
	if !strings.Contains(output, testUsernameA) {
		t.Errorf("log output does not contain the attempted username %q; output:\n%s", testUsernameA, output)
	}
	if !strings.Contains(output, "failure") {
		t.Errorf("log output does not contain result=failure; output:\n%s", output)
	}
	if strings.Contains(output, secretPassword) {
		t.Fatalf("log output leaked the plaintext password: %q; output:\n%s", secretPassword, output)
	}
}

func TestLoginLogging_RateLimited_NeverLogsPassword(t *testing.T) {
	srv := newAuthTestServer(t)

	// Exhaust the per-username threshold first (not yet capturing logs —
	// only the 11th attempt's log line is under test).
	for i := 0; i < 10; i++ {
		res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
			"username": testUsernameA,
			"password": "wrong",
		})
		res.Body.Close()
	}

	buf := captureSlog(t)
	const secretPassword = "this-must-never-appear-in-logs-either"
	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
		"username": testUsernameA,
		"password": secretPassword,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("11th attempt status = %d, want 429 (test setup assumption)", res.StatusCode)
	}

	output := buf.String()
	if !strings.Contains(output, "rate_limited") {
		t.Errorf("log output does not contain result=rate_limited; output:\n%s", output)
	}
	if strings.Contains(output, secretPassword) {
		t.Fatalf("log output leaked the plaintext password on the rate-limited path: %q; output:\n%s", secretPassword, output)
	}
}

// Successful login must also never log the session token in plaintext.
func TestLoginLogging_Success_NeverLogsSessionToken(t *testing.T) {
	buf := captureSlog(t)
	srv := newAuthTestServer(t)

	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}
	if login.SessionToken == "" {
		t.Fatal("no session token captured")
	}

	output := buf.String()
	if strings.Contains(output, login.SessionToken) {
		t.Fatalf("log output leaked the plaintext session token; output:\n%s", output)
	}
}
