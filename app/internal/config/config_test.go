package config

import (
	"strings"
	"testing"
)

// T-33 — AC-13/EC-12: config.Load() fails fast without every required
// credential/secret var, and never starts in a partial state.

func setAllRequiredEnv(t *testing.T) {
	t.Helper()
	vars := map[string]string{
		"NIU_SESSION_SECRET": "01234567890123456789012345678901", // 34 bytes
		"NIU_USER_A_NAME":    "usuari_a",
		"NIU_USER_A_DISPLAY": "Usuari A",
		"NIU_USER_A_HASH":    "$2a$12$fixtureonlyfixtureonlyfixtureonly",
		"NIU_USER_B_NAME":    "usuari_b",
		"NIU_USER_B_DISPLAY": "Usuari B",
		"NIU_USER_B_HASH":    "$2a$12$fixtureonlyfixtureonlyfixtureonlyy",
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func TestLoad_AllRequiredVarsPresent_Succeeds(t *testing.T) {
	setAllRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with all required vars set: %v", err)
	}
	if cfg.SessionSecret == "" || cfg.UserAHash == "" || cfg.UserBHash == "" {
		t.Fatalf("Load() returned incomplete config: %+v", cfg)
	}
}

func TestLoad_MissingSessionSecret_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_SESSION_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded without NIU_SESSION_SECRET, want an error (EC-12)")
	}
}

func TestLoad_SessionSecretTooShort_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_SESSION_SECRET", strings.Repeat("x", minSessionSecretBytes-1))

	if _, err := Load(); err == nil {
		t.Fatalf("Load() succeeded with a %d-byte secret (below the %d-byte floor), want an error",
			minSessionSecretBytes-1, minSessionSecretBytes)
	}
}

func TestLoad_MissingUserAHash_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_USER_A_HASH", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded without NIU_USER_A_HASH, want an error (EC-12)")
	}
}

func TestLoad_MissingUserBHash_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_USER_B_HASH", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded without NIU_USER_B_HASH, want an error (EC-12)")
	}
}

func TestLoad_MissingUserAName_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_USER_A_NAME", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded without NIU_USER_A_NAME, want an error (EC-12)")
	}
}

func TestLoad_MissingUserADisplay_FailsFast(t *testing.T) {
	setAllRequiredEnv(t)
	t.Setenv("NIU_USER_A_DISPLAY", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded without NIU_USER_A_DISPLAY, want an error (EC-12)")
	}
}
