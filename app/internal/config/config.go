// Package config parses process configuration from the environment with
// fail-fast validation (PLAN.md §6). In NIU-1, almost nothing is required
// yet: NIU_SESSION_SECRET and NIU_USER_*_HASH become mandatory only in
// NIU-4 (real auth). This item seeds placeholder users directly via a
// migration, not via environment variables.
package config

import (
	"fmt"
	"os"
)

// Config holds the process-wide configuration resolved once at startup.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string
	// DBPath is the filesystem path to the SQLite database file.
	DBPath string
	// Env is the running environment ("production" by default).
	Env string
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
	}

	if cfg.Port == "" {
		return Config{}, fmt.Errorf("config: NIU_PORT must not be empty")
	}
	if cfg.DBPath == "" {
		return Config{}, fmt.Errorf("config: NIU_DB_PATH must not be empty")
	}

	// NIU_SESSION_SECRET and NIU_USER_*_HASH are required starting NIU-4
	// (real authentication). Not validated here — auth is stubbed in
	// NIU-1 (ADR-03) and users are seeded via migration 002.

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
