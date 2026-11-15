// Package store implements items.Repository and items.EventSink (and the
// ideas and projects repositories) on top of PostgreSQL: the platform
// cluster, schema "niu" (fikua-platform-iac).
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
)

// MigrationsFS is populated by the caller (cmd/niu) via //go:embed and
// passed to Open so goose can apply versioned migrations from the
// embedded binary, without requiring a migrations/ directory on disk at
// runtime.
type MigrationsFS = embed.FS

// Store wraps the connection pool used by the application.
type Store struct {
	DB *sql.DB
}

// Open connects to PostgreSQL with databaseURL (the role's search_path — or
// a search_path parameter in the URL — selects the schema) and runs any
// pending goose migrations embedded in migrationsFS under "migrations".
//
// Open holds a session advisory lock while goose runs, so two processes
// starting together (two replicas, or the app and the sqlite2pg tool) never
// apply the same migration twice.
func Open(databaseURL string, migrationsFS embed.FS) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: open postgres: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping postgres: %w", err)
	}

	if err := migrateUp(db, migrationsFS); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{DB: db}, nil
}

// migrationLockID is the arbitrary advisory-lock key of the migrator.
const migrationLockID = 7_310_244

func migrateUp(db *sql.DB, migrationsFS embed.FS) error {
	conn, err := db.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("store: acquire migration conn: %w", err)
	}
	defer conn.Close() //nolint:errcheck // returning to the pool

	if _, err := conn.ExecContext(context.Background(), `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("store: take migration lock: %w", err)
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLockID) //nolint:errcheck

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("store: set goose dialect: %w", err)
	}
	// The migrations run on the pool, not on conn: the advisory lock only has
	// to be held (by conn's session) while they do.
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("store: run migrations: %w", err)
	}
	return nil
}

// IsUniqueViolation reports whether err is PostgreSQL's unique_violation
// (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Close releases the underlying database connection pool.
func (s *Store) Close() error {
	return s.DB.Close()
}

// Healthy runs a trivial query against PostgreSQL to verify the dependency is
// reachable — used by GET /healthz (REL-03, NFR-08).
func (s *Store) Healthy() error {
	var one int
	return s.DB.QueryRow("SELECT 1").Scan(&one)
}
