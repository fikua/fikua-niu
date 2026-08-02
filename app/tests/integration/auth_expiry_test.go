package integration

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"niu/internal/auth"
)

// T-32 — AC-12/EC-10/NFR-08: a session past its expires_at is rejected
// even though the row still exists, and CleanupExpired removes it.

func TestExpiredSession_RejectedAndCleanedUp(t *testing.T) {
	srv := newAuthTestServer(t)

	// Seed an already-expired session directly, bypassing login, so the
	// test controls expires_at precisely.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("generate token: %v", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(sum[:])

	// Stored in the same UTC "YYYY-MM-DD HH:MM:SS" text format
	// CURRENT_TIMESTAMP produces — a raw time.Time (RFC3339 with a
	// timezone offset) would not compare correctly against it.
	expiredAt := time.Now().Add(-1 * time.Hour).UTC().Format("2006-01-02 15:04:05")
	if _, err := srv.Store.DB.Exec(
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		tokenHash, seedUserAID, expiredAt,
	); err != nil {
		t.Fatalf("seed expired session: %v", err)
	}

	// AC-12: a request using the expired token is rejected as
	// unauthenticated even though the row is still present.
	res := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/items", token, "", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("request with an expired session token status = %d, want 401", res.StatusCode)
	}

	var countBefore int
	if err := srv.Store.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token_hash = ?`, tokenHash).Scan(&countBefore); err != nil {
		t.Fatalf("count sessions before cleanup: %v", err)
	}
	if countBefore != 1 {
		t.Fatalf("expected the expired session row to still exist before cleanup, got count=%d", countBefore)
	}

	// EC-10/NFR-08: force CleanupExpired directly (not waiting for the
	// real hourly ticker) and confirm the row disappears.
	authenticator, err := auth.NewPasswordAuthenticator(srv.Store.DB, srv.SessionSecret)
	if err != nil {
		t.Fatalf("NewPasswordAuthenticator: %v", err)
	}
	if err := authenticator.CleanupExpired(); err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}

	var countAfter int
	if err := srv.Store.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token_hash = ?`, tokenHash).Scan(&countAfter); err != nil {
		t.Fatalf("count sessions after cleanup: %v", err)
	}
	if countAfter != 0 {
		t.Fatalf("expired session row still present after CleanupExpired, count=%d", countAfter)
	}
}
