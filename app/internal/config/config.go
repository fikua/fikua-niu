// Package config parses process configuration from the environment with
// fail-fast validation (PLAN.md §6). Starting NIU-4, NIU_SESSION_SECRET and
// the NIU_USER_*_HASH/NAME/DISPLAY variables are mandatory (real
// authentication, design.md §2 point 9) — the process refuses to start in
// any partially-configured state.
package config

import (
	"fmt"
	"os"
)

// minSessionSecretBytes is the minimum length of NIU_SESSION_SECRET
// (design.md §1, EC-12) — short enough to type by hand for local dev, long
// enough to make the HMAC-derived CSRF token (ADR-05) and any future use of
// the secret computationally sound.
const minSessionSecretBytes = 32

// Config holds the process-wide configuration resolved once at startup.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string
	// DBPath is the filesystem path to the SQLite database file.
	DBPath string
	// Env is the running environment ("production" by default).
	Env string

	// SessionSecret is the HMAC key used to derive the CSRF token from a
	// session's token_hash (ADR-05). Must be at least minSessionSecretBytes.
	SessionSecret string
	// UserAName/UserADisplay/UserAHash and UserBName/UserBDisplay/UserBHash
	// seed the two households' credentials at startup via an idempotent
	// UPDATE (design.md §6.2, T-19) — never committed to the repository
	// (S11/NFR-09).
	UserAName    string
	UserADisplay string
	UserAHash    string
	UserAAvatar  string
	UserBName    string
	UserBDisplay string
	UserBHash    string
	UserBAvatar  string
}

const (
	defaultPort   = "8080"
	defaultDBPath = "/data/niu.db"
	defaultEnv    = "production"
)

// Load reads configuration from the environment and validates it,
// following PLAN.md §6. It never panics; callers should treat a non-nil
// error as fatal and refuse to start the process.
func Load() (Config, error) {
	cfg := Config{
		Port:   getEnvOrDefault("NIU_PORT", defaultPort),
		DBPath: getEnvOrDefault("NIU_DB_PATH", defaultDBPath),
		Env:    getEnvOrDefault("NIU_ENV", defaultEnv),

		SessionSecret: os.Getenv("NIU_SESSION_SECRET"),
		UserAName:     os.Getenv("NIU_USER_A_NAME"),
		UserADisplay:  os.Getenv("NIU_USER_A_DISPLAY"),
		UserAHash:     os.Getenv("NIU_USER_A_HASH"),
		UserAAvatar:   getEnvOrDefault("NIU_USER_A_AVATAR", "🐦"),
		UserBName:     os.Getenv("NIU_USER_B_NAME"),
		UserBDisplay:  os.Getenv("NIU_USER_B_DISPLAY"),
		UserBHash:     os.Getenv("NIU_USER_B_HASH"),
		UserBAvatar:   getEnvOrDefault("NIU_USER_B_AVATAR", "🦊"),
	}

	if cfg.Port == "" {
		return Config{}, fmt.Errorf("config: NIU_PORT must not be empty")
	}
	if cfg.DBPath == "" {
		return Config{}, fmt.Errorf("config: NIU_DB_PATH must not be empty")
	}

	// EC-12/AC-13: the six credential fields and the session secret are
	// mandatory — no partial-start state. Checked individually (not just
	// "all empty") so the error message is actionable.
	required := map[string]string{
		"NIU_SESSION_SECRET": cfg.SessionSecret,
		"NIU_USER_A_NAME":    cfg.UserAName,
		"NIU_USER_A_DISPLAY": cfg.UserADisplay,
		"NIU_USER_A_HASH":    cfg.UserAHash,
		"NIU_USER_B_NAME":    cfg.UserBName,
		"NIU_USER_B_DISPLAY": cfg.UserBDisplay,
		"NIU_USER_B_HASH":    cfg.UserBHash,
	}
	for _, key := range []string{
		"NIU_SESSION_SECRET",
		"NIU_USER_A_NAME", "NIU_USER_A_DISPLAY", "NIU_USER_A_HASH",
		"NIU_USER_B_NAME", "NIU_USER_B_DISPLAY", "NIU_USER_B_HASH",
	} {
		if required[key] == "" {
			return Config{}, fmt.Errorf("config: %s must be set", key)
		}
	}
	if len(cfg.SessionSecret) < minSessionSecretBytes {
		return Config{}, fmt.Errorf("config: NIU_SESSION_SECRET must be at least %d bytes", minSessionSecretBytes)
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
