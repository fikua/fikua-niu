// Command niu is the single Go binary serving both the JSON API
// (/api/v1/*) and the static frontend (embedded web/) from the same
// process (PLAN.md §2.1). Wiring order: config → store → credential seed →
// items.Service → fetchsafe.NewClient + shared preview worker pool →
// ideas.Service (NIU-6, tasks.md T-08/T-14) → projects.Service (NIU-5's
// three-state lifecycle, NIU-11's link preview reusing the same
// fetchsafeClient/previewPool as ideas, tasks.md T-06) →
// auth.PasswordAuthenticator → cleanup goroutine → httpapi.NewRouter
// (design.md §5/§6.2, T-19).
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
	"strings"
	"syscall"
	"time"

	niu "niu"
	"niu/internal/auth"
	"niu/internal/config"
	"niu/internal/fetchsafe"
	"niu/internal/httpapi"
	"niu/internal/ideas"
	"niu/internal/items"
	"niu/internal/projects"
	"niu/internal/store"
)

// cleanupInterval is how often the background goroutine deletes expired
// sessions and stale rate-limit buckets (ADR-04).
const cleanupInterval = 1 * time.Hour

// setupLogging installs the process-wide slog handler. Without this, Go's
// default handler discards every Debug record, so diagnostics written at
// that level — notably the per-idea reason a link preview failed
// (ideas.Service.resolvePreview) — never reached the logs at all, and a
// failing preview could only be diagnosed by reproducing it by hand.
//
// Level is read from NIU_LOG_LEVEL (debug|info|warn|error), defaulting to
// info so production verbosity is unchanged unless explicitly raised.
func setupLogging() {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(os.Getenv("NIU_LOG_LEVEL"))) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}

func main() {
	setupLogging()
	if err := run(); err != nil {
		slog.Error("niu: fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(niu.UsersConfigFS)
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

	// ctx/stop is the process' own shutdown context — cancelled on
	// SIGINT/SIGTERM. The preview worker pool (ADR-03, T-08 of NIU-6) is
	// deliberately wired against THIS context, never a per-request
	// context: a scrape must keep running after the POST that queued it
	// has already responded, and must only stop when the whole process is
	// shutting down.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// fetchsafeClient is THE single client for the whole process (T-03h of
	// NIU-6) — never one per service. previewPool is likewise shared
	// between ideas and projects (NIU-11, tasks.md T-06): 6 workers
	// already cover the load of a two-person household app, and a second
	// pool would just double the memory ceiling (see ideas.WorkerPool's
	// own sizing comment) without either space actually needing dedicated
	// capacity. Named "previewPool", not "ideasPool", precisely because it
	// is no longer ideas-exclusive — a name that lied about that would be
	// worse than no comment at all.
	fetchsafeClient := fetchsafe.NewClient()
	previewPool := ideas.NewWorkerPool(ctx)
	defer previewPool.Close()

	ideasRepo := store.NewIdeasRepository(st.DB)
	ideasFetch := func(fetchCtx context.Context, rawURL string) (ideas.Preview, error) {
		preview, err := fetchsafe.FetchPreview(fetchCtx, fetchsafeClient, rawURL)
		return ideas.Preview{
			Title:       preview.Title,
			ImageURL:    preview.ImageURL,
			Description: preview.Description,
			Partial:     preview.Partial,
		}, err
	}
	ideasSvc := ideas.NewService(ideasRepo, ideasRepo, ideasFetch, previewPool)

	projectsRepo := store.NewProjectsRepository(st.DB)
	projectsFetch := func(fetchCtx context.Context, rawURL string) (projects.Preview, error) {
		preview, err := fetchsafe.FetchPreview(fetchCtx, fetchsafeClient, rawURL)
		return projects.Preview{
			Title:       preview.Title,
			ImageURL:    preview.ImageURL,
			Description: preview.Description,
			Partial:     preview.Partial,
		}, err
	}
	projectsSvc := projects.NewService(projectsRepo, projectsRepo, projectsFetch, previewPool)

	authenticator, err := auth.NewPasswordAuthenticator(st.DB, cfg.SessionSecret)
	if err != nil {
		return err
	}

	webFS, err := fs.Sub(niu.WebFS, "web")
	if err != nil {
		return err
	}

	cookiesSecure := cfg.Env != "development"
	router := httpapi.NewRouter(svc, projectsSvc, ideasSvc, st, authenticator, webFS, cookiesSecure)

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
// either UPDATE affects zero rows.
//
// Keyed by id (1=A, 2=B), not by name — migration 002's ids never
// change, but name/display_name/avatar_emoji are all meant to be
// reconfigurable from the environment (e.g. renaming usuari_a's login
// to a real username, or swapping the avatar emoji), so matching on name
// would break the moment someone actually renames a user.
func seedCredentials(db *sql.DB, cfg config.Config) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("niu: seed credentials: begin tx: %w", err)
	}
	defer tx.Rollback()

	seeds := []struct {
		id                          int64
		name, display, hash, avatar string
	}{
		{1, auth.NormalizeUsername(cfg.UserA.Name), cfg.UserA.DisplayName, cfg.UserAHash, cfg.UserA.AvatarEmoji},
		{2, auth.NormalizeUsername(cfg.UserB.Name), cfg.UserB.DisplayName, cfg.UserBHash, cfg.UserB.AvatarEmoji},
	}

	for _, seed := range seeds {
		res, err := tx.Exec(
			`UPDATE users SET name = ?, password_hash = ?, display_name = ?, avatar_emoji = ? WHERE id = ?`,
			seed.name, seed.hash, seed.display, seed.avatar, seed.id,
		)
		if err != nil {
			return fmt.Errorf("niu: seed credentials: update user id=%d: %w", seed.id, err)
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("niu: seed credentials: rows affected for user id=%d: %w", seed.id, err)
		}
		if rows != 1 {
			return fmt.Errorf("niu: seed credentials: expected to update exactly 1 row for user id=%d, got %d — "+
				"migration 002 may not have run", seed.id, rows)
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
