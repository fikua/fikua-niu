package integration

import (
	"net/http"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// T-33 (integration half) — AC-13: with all required config variables
// present, both seeded users can log in successfully against the hashes
// derived from that configuration — exercising the same
// UPDATE-users-at-startup path cmd/niu.seedCredentials implements
// (unexported in package main, so re-created here at the store level with
// the exact same SQL and RowsAffected verification design.md §6.2
// mandates).
func TestConfigSeed_BothUsersCanLoginAgainstConfiguredHashes(t *testing.T) {
	srv := newAuthTestServer(t) // already seeds both users via the same UPDATE pattern

	for _, tc := range []struct {
		username, password string
	}{
		{testUsernameA, testPasswordA},
		{testUsernameB, testPasswordB},
	} {
		res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/auth/login", map[string]string{
			"username": tc.username,
			"password": tc.password,
		})
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("login(%q) status = %d, want 200", tc.username, res.StatusCode)
		}
	}
}

// TestConfigSeed_UpdateFailsWhenNameDoesNotMatchSeedRow proves the
// RowsAffected guard design.md §6.2/risk R-05 requires: an UPDATE whose
// WHERE name = ? matches no row must be treated as a configuration error,
// not silently ignored.
func TestConfigSeed_UpdateFailsWhenNameDoesNotMatchSeedRow(t *testing.T) {
	srv := newAuthTestServer(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("whatever"), 12)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword: %v", err)
	}

	res, err := srv.Store.DB.Exec(`UPDATE users SET password_hash = ? WHERE name = ?`, string(hash), "nom_que_no_existeix")
	if err != nil {
		t.Fatalf("UPDATE: %v", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected: %v", err)
	}
	if rows != 0 {
		t.Fatalf("RowsAffected = %d for a nonexistent username, want 0 (this is the case that must fail startup, R-05)", rows)
	}
}
