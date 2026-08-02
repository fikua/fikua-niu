// Command niu is the single Go binary serving both the JSON API
// (/api/v1/*) and the static frontend (embedded web/) from the same
// process (PLAN.md §2.1). Wiring order: config → store → credential seed →
// items.Service → auth.PasswordAuthenticator → cleanup goroutine →
// httpapi.NewRouter (design.md §5/§6.2, T-19).
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	niu "niu"
	"niu/internal/auth"
	"niu/internal/config"
	"niu/internal/httpapi"
	"niu/internal/items"
	"niu/internal/store"
)

// cleanupInterval is how often the background goroutine deletes expired
// sessions and stale rate-limit buckets (ADR-04).
const cleanupInterval = 1 * time.Hour

func main() {
	if err := run(); err != nil {
		slog.Error("niu: fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.DBPath, niu.MigrationsFS)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := seedCredentials(st.DB, cfg); err != nil {
		return err
	}

	repo := store.NewItemsRepository(st.DB)
	svc := items.NewService(repo, repo, repo)

	authenticator, err := auth.NewPasswordAuthenticator(st.DB, cfg.SessionSecret)
	if err != nil {
		return err
	}

	webFS, err := fs.Sub(niu.WebFS, "web")
	if err != nil {
		return err
	}

	cookiesSecure := cfg.Env != "development"
	router := httpapi.NewRouter(svc, st, authenticator, webFS, cookiesSecure)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startCleanupLoop(ctx, authenticator)

	addr := ":" + cfg.Port
	slog.Info("niu: listening", "addr", addr, "env", cfg.Env)

	// Timeouts are not optional on a public listener: without them a
	// single slow or stalled connection holds a goroutine and its buffers
	// open indefinitely, and enough of them exhaust the process. The
	// values are generous for a two-person app on a shared VPS.
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16, // 64 KiB
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// seedCredentials upserts the two households' credentials from
// configuration into the users table (design.md §6.2, AC-13, EC-12,
// risk R-05) — never a migration, since the users already exist from
// migration 002. Runs inside a transaction; fails the whole startup if
// either UPDATE affects zero rows (a name mismatch is a real
// configuration error, not an expected case, since migration 002 already
// seeded usuari_a/usuari_b).
func seedCredentials(db *sql.DB, cfg config.Config) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("niu: seed credentials: begin tx: %w", err)
	}
	defer tx.Rollback()

	seeds := []struct {
		name, display, hash string
	}{
		{auth.NormalizeUsername(cfg.UserAName), cfg.UserADisplay, cfg.UserAHash},
		{auth.NormalizeUsername(cfg.UserBName), cfg.UserBDisplay, cfg.UserBHash},
	}

	for _, seed := range seeds {
		res, err := tx.Exec(
			`UPDATE users SET password_hash = ?, display_name = ? WHERE name = ?`,
			seed.hash, seed.display, seed.name,
		)
		if err != nil {
			return fmt.Errorf("niu: seed credentials: update %q: %w", seed.name, err)
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("niu: seed credentials: rows affected for %q: %w", seed.name, err)
		}
		if rows != 1 {
			return fmt.Errorf("niu: seed credentials: expected to update exactly 1 row for user %q, got %d — "+
				"NIU_USER_*_NAME does not match a seeded user", seed.name, rows)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("niu: seed credentials: commit: %w", err)
	}
	return nil
}

// startCleanupLoop launches ADR-04's background goroutine: an immediate
// pass at startup (covers sessions that expired during a long process
// outage), then one pass per cleanupInterval, stopping cleanly when ctx is
// cancelled (shutdown).
func startCleanupLoop(ctx context.Context, authenticator *auth.PasswordAuthenticator) {
	runOnce := func() {
		if err := authenticator.CleanupExpired(); err != nil {
			slog.Error("niu: session cleanup failed", "error", err)
		}
	}

	runOnce()

	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runOnce()
			}
		}
	}()
}
