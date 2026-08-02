package auth

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	_ "modernc.org/sqlite"
)

// T-25 — AC-11/NFR-03 (ADR-02): bcrypt.CompareHashAndPassword must run
// exactly once per login attempt, whether the user exists or not, and the
// comparison against dummyHash must always fail. Uses a minimal in-memory
// SQLite schema (users/sessions only) so internal/auth stays decoupled
// from internal/store, matching design.md §4 ("auth never imports
// database access via internal/store — it owns its own *sql.DB queries").

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			avatar_emoji TEXT NOT NULL DEFAULT '🐦'
		);
		CREATE TABLE sessions (
			token_hash TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func seedUser(t *testing.T, db *sql.DB, name, password string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO users (name, display_name, password_hash) VALUES (?, ?, ?)`,
		name, "Test User", string(hash),
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// TestLogin_AlwaysComparesBcryptOnce measures wall-clock time for a login
// against a known user with a wrong password versus an unknown username,
// asserting both pay the same (bcrypt-dominated) cost within a generous
// margin — the same criterion NFR-03 specifies to avoid CI flakiness. A
// naive implementation that returns early when the user does not exist
// would make the "unknown user" path measurably faster.
func TestLogin_AlwaysComparesBcryptOnce(t *testing.T) {
	db := newTestDB(t)
	seedUser(t, db, "usuari_a", "correct horse battery staple")

	a, err := NewPasswordAuthenticator(db, "01234567890123456789012345678901")
	if err != nil {
		t.Fatalf("NewPasswordAuthenticator: %v", err)
	}

	const samples = 3
	var wrongPasswordTotal, unknownUserTotal time.Duration

	for i := 0; i < samples; i++ {
		start := time.Now()
		_, _, err := a.Login("usuari_a", "wrong password", "127.0.0.1")
		wrongPasswordTotal += time.Since(start)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Login(wrong password) error = %v, want ErrInvalidCredentials", err)
		}

		start = time.Now()
		_, _, err = a.Login("usuari_inexistent", "whatever", "127.0.0.1")
		unknownUserTotal += time.Since(start)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("Login(unknown user) error = %v, want ErrInvalidCredentials", err)
		}
	}

	wrongAvg := wrongPasswordTotal / samples
	unknownAvg := unknownUserTotal / samples

	// Both paths must be bcrypt-dominated (cost 12, NFR-01: >200ms).
	if wrongAvg < 100*time.Millisecond {
		t.Errorf("wrong-password path averaged %v — too fast for a real bcrypt cost-12 comparison; "+
			"is bcrypt actually running on every attempt?", wrongAvg)
	}
	if unknownAvg < 100*time.Millisecond {
		t.Errorf("unknown-user path averaged %v — too fast for a real bcrypt cost-12 comparison; "+
			"ADR-02 requires bcrypt to run even when the user does not exist", unknownAvg)
	}

	// Generous margin (NFR-03: "marge ampli, per evitar flakiness en CI") —
	// this is a coarse structural signal, not a precision timing attack
	// test. A naive short-circuit would show a difference on the order of
	// the full bcrypt cost (hundreds of ms), not a few tens of ms of
	// scheduling noise.
	diff := wrongAvg - unknownAvg
	if diff < 0 {
		diff = -diff
	}
	maxOf := wrongAvg
	if unknownAvg > maxOf {
		maxOf = unknownAvg
	}
	if diff > maxOf/2 {
		t.Errorf("timing gap too large: wrong-password avg=%v, unknown-user avg=%v (diff=%v) — "+
			"suggests bcrypt is being skipped on one of the two paths (S5)", wrongAvg, unknownAvg, diff)
	}
}

// TestLogin_DummyHashComparisonAlwaysFails asserts the dummy hash never
// matches any client-supplied password, including the exact dummy
// password constant itself (which a client can never legitimately send
// since it is not tied to any real account).
func TestLogin_DummyHashComparisonAlwaysFails(t *testing.T) {
	db := newTestDB(t)
	// No users seeded at all.

	a, err := NewPasswordAuthenticator(db, "01234567890123456789012345678901")
	if err != nil {
		t.Fatalf("NewPasswordAuthenticator: %v", err)
	}

	_, _, err = a.Login("nobody", dummyPassword, "127.0.0.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login with the literal dummy password against a nonexistent user = %v, want ErrInvalidCredentials", err)
	}
}
