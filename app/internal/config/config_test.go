package config

import (
	"strings"
	"testing"
	"testing/fstest"
)

// T-33 — AC-13/EC-12: config.Load() fails fast without every required
// secret var, and never starts in a partial state. Non-secret identity
// fields (name/display/avatar) now come from users.json rather than the
// environment — see TestLoadUsers_* below for their own validation.

const validUsersJSON = `{
  "user_a": {"name": "usuari_a", "display_name": "Usuari A", "avatar_emoji": "🐦"},
  "user_b": {"name": "usuari_b", "display_name": "Usuari B", "avatar_emoji": "🦊"}
}`

func fixtureUsersFS(t *testing.T, contents string) fstest.MapFS {
	t.Helper()
	return fstest.MapFS{
		"users.json": {Data: []byte(contents)},
	}
}

func setAllRequiredEnv(t *testing.T) {
	t.Helper()
	vars := map[string]string{
		"NIU_SESSION_SECRET": "01234567890123456789012345678901", // 34 bytes
		"NIU_USER_A_HASH":    "$2a$12$fixtureonlyfixtureonlyfixtureonly",
		"NIU_USER_B_HASH":    "$2a$12$fixtureonlyfixtureonlyfixtureonlyy",
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func TestLoad_AllRequiredVarsPresent_Succeeds(t *testing.T) {
	setAllRequiredEnv(t)

	cfg, err := Load(fixtureUsersFS(t, validUsersJSON))
	if err != nil {
		t.Fatalf("Load() with all required vars set: %v", err)
	}
	if cfg.SessionSecret == "" || cfg.UserAHash == "" || cfg.UserBHash == "" {
		t.Fatalf("Load() returned incomplete config: %+v", cfg)
	}
	if cfg.UserA.Name != "usuari_a" || cfg.UserA.DisplayName != "Usuari A" || cfg.UserA.AvatarEmoji != "🐦" {
		t.Fatalf("Load() UserA identity mismatch: %+v", cfg.UserA)
	}
	if cfg.UserB.Name != "usuari_b" || cfg.UserB.DisplayName != "Usuari B" || cfg.UserB.AvatarEmoji != "🦊" {
		t.Fatalf("Load() UserB identity mismatch: %+v", cfg.UserB)
	}
}

func TestLoad_MissingSessionSecret_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_SESSION_SECRET", "")

	if _, err := Load(fixtureUsersFS(t, validUsersJSON)); err == nil {
		t.Fatal("Load() succeeded without NIU_SESSION_SECRET, want an error (EC-12)")
	}
}

func TestLoad_SessionSecretTooShort_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_SESSION_SECRET", strings.Repeat("x", minSessionSecretBytes-1))

	if _, err := Load(fixtureUsersFS(t, validUsersJSON)); err == nil {
		t.Fatalf("Load() succeeded with a %d-byte secret (below the %d-byte floor), want an error",
			minSessionSecretBytes-1, minSessionSecretBytes)
	}
}

func TestLoad_MissingUserAHash_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_USER_A_HASH", "")

	if _, err := Load(fixtureUsersFS(t, validUsersJSON)); err == nil {
		t.Fatal("Load() succeeded without NIU_USER_A_HASH, want an error (EC-12)")
	}
}

func TestLoad_MissingUserBHash_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_USER_B_HASH", "")

	if _, err := Load(fixtureUsersFS(t, validUsersJSON)); err == nil {
		t.Fatal("Load() succeeded without NIU_USER_B_HASH, want an error (EC-12)")
	}
}

// TestLoadUsers_MissingName_FailsFast covers EC-12's intent for the
// fields that moved from the environment to users.json: a malformed or
// incomplete committed file must still fail fast at startup, not start
// with a blank username.
func TestLoadUsers_MissingName_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	badJSON := `{
  "user_a": {"name": "", "display_name": "Usuari A", "avatar_emoji": "🐦"},
  "user_b": {"name": "usuari_b", "display_name": "Usuari B", "avatar_emoji": "🦊"}
}`

	if _, err := Load(fixtureUsersFS(t, badJSON)); err == nil {
		t.Fatal("Load() succeeded with an empty user_a.name in users.json, want an error")
	}
}

func TestLoadUsers_MissingDisplay_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	badJSON := `{
  "user_a": {"name": "usuari_a", "display_name": "", "avatar_emoji": "🐦"},
  "user_b": {"name": "usuari_b", "display_name": "Usuari B", "avatar_emoji": "🦊"}
}`

	if _, err := Load(fixtureUsersFS(t, badJSON)); err == nil {
		t.Fatal("Load() succeeded with an empty user_a.display_name in users.json, want an error")
	}
}

func TestLoadUsers_MalformedJSON_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)

	if _, err := Load(fixtureUsersFS(t, `not json`)); err == nil {
		t.Fatal("Load() succeeded with malformed users.json, want an error")
	}
}
