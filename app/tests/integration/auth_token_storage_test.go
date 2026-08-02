package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
)

// T-28 — AC-08/S2c: the sessions table never stores the plaintext token,
// only SHA-256(token). Same DB-inspection pattern as
// TestSQLInjectionPayload_StoredLiterally_TableSurvives (sql_static_test.go).
func TestSessionStorage_NeverContainsPlaintextToken(t *testing.T) {
	srv := newAuthTestServer(t)

	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}
	if login.SessionToken == "" {
		t.Fatal("no session token captured from login response")
	}

	rows, err := srv.Store.DB.Query(`SELECT token_hash FROM sessions`)
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			t.Fatalf("scan token_hash: %v", err)
		}
		hashes = append(hashes, h)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}

	if len(hashes) == 0 {
		t.Fatal("no rows in sessions table after a successful login")
	}

	sum := sha256.Sum256([]byte(login.SessionToken))
	expectedHash := hex.EncodeToString(sum[:])

	foundExpectedHash := false
	for _, h := range hashes {
		if h == login.SessionToken {
			t.Fatalf("sessions.token_hash contains the PLAINTEXT token: %q", h)
		}
		if h == expectedHash {
			foundExpectedHash = true
		}
	}
	if !foundExpectedHash {
		t.Fatalf("no row matches SHA-256(token) = %q — session does not appear to be stored at all", expectedHash)
	}
}
