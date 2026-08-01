// Command niu is the single Go binary serving both the JSON API
// (/api/v1/*) and the static frontend (embedded web/) from the same
// process (PLAN.md §2.1). Wiring order: config → store → items.Service →
// auth.StubAuthenticator → httpapi.NewRouter (design.md §5, T-15).
package main

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	niu "niu"
	"niu/internal/auth"
	"niu/internal/config"
	"niu/internal/httpapi"
	"niu/internal/items"
	"niu/internal/store"
)

// seedUserAID is the seeded "Usuari A" row inserted by migration 002 —
// the fixed identity returned by the NIU-1 auth stub (ADR-03).
const seedUserAID = 1

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

	repo := store.NewItemsRepository(st.DB)
	svc := items.NewService(repo, repo, repo)
	authenticator := auth.StubAuthenticator{UserID: seedUserAID}

	webFS, err := fs.Sub(niu.WebFS, "web")
	if err != nil {
		return err
	}

	router := httpapi.NewRouter(svc, st, authenticator, webFS)

	addr := ":" + cfg.Port
	slog.Info("niu: listening", "addr", addr, "env", cfg.Env)

	srv := &http.Server{Addr: addr, Handler: router}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
