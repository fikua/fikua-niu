// Package store implements items.Repository and items.EventSink on top of
// SQLite (modernc.org/sqlite — pure Go, no CGO, design.md §2.3/§7).
package store

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"

	// Pure-Go SQLite driver registered under the "sqlite" name. Never
	// mattn/go-sqlite3 (requires CGO) — design.md §2.3, PLAN.md §2.3.
	_ "modernc.org/sqlite"
)

// MigrationsFS is populated by the caller (cmd/niu) via //go:embed and
// passed to Open so goose can apply versioned migrations from the
// embedded binary, without requiring a migrations/ directory on disk at
// runtime.
type MigrationsFS = embed.FS

// Store wraps the single SQLite connection pool used by the application.
type Store struct {
	DB *sql.DB
}

// Open opens (and creates, if missing) the SQLite database at dbPath,
// applies the three mandatory pragmas (design.md §7: WAL, busy_timeout,
// foreign_keys) and runs any pending goose migrations embedded in
// migrationsFS under the "migrations" subdirectory.
//
// No SetMaxOpenConns is applied here — human-confirmed decision (design.md
// §7, tasks.md T-04): revisit only if load testing (NFR-05) shows
// contention.
func Open(dbPath string, migrationsFS embed.FS) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)",
		dbPath,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping sqlite: %w", err)
	}

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: set goose dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: run migrations: %w", err)
	}

	return &Store{DB: db}, nil
}

// Close releases the underlying database connection pool.
func (s *Store) Close() error {
	return s.DB.Close()
}

// Healthy runs a trivial query against SQLite to verify the dependency is
// reachable — used by GET /healthz (REL-03, NFR-08).
func (s *Store) Healthy() error {
	var one int
	return s.DB.QueryRow("SELECT 1").Scan(&one)
}
