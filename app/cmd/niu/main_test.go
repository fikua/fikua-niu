package main

import (
	"testing"

	niu "niu"
	"niu/internal/config"
	"niu/internal/store"
	"niu/tests/testdb"
)

// seedCredentials runs at every start against the real database: it must
// overwrite the two placeholder users from migration 002.
func TestSeedCredentials_UpdatesBothUsers(t *testing.T) {
	st, err := store.Open(testdb.URL(t), niu.MigrationsFS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	cfg := config.Config{
		UserA:     config.UserIdentity{Name: "Anna", DisplayName: "Anna", AvatarEmoji: "🐦"},
		UserAHash: "$2a$12$fixtureonlyfixtureonlyfixtureonly",
		UserB:     config.UserIdentity{Name: "bernat", DisplayName: "Bernat", AvatarEmoji: "🦊"},
		UserBHash: "$2a$12$fixtureonlyfixtureonlyfixtureonlyy",
	}
	if err := seedCredentials(st.DB, cfg); err != nil {
		t.Fatalf("seedCredentials: %v", err)
	}

	var name, display, hash string
	if err := st.DB.QueryRow(`SELECT name, display_name, password_hash FROM users WHERE id = 2`).Scan(&name, &display, &hash); err != nil {
		t.Fatal(err)
	}
	if name != "bernat" || display != "Bernat" || hash != cfg.UserBHash {
		t.Fatalf("user 2 = %q/%q/%q, want the seeded values", name, display, hash)
	}
	if err := st.DB.QueryRow(`SELECT name FROM users WHERE id = 1`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "anna" { // NormalizeUsername lower-cases
		t.Fatalf("user 1 name = %q, want %q", name, "anna")
	}
}
