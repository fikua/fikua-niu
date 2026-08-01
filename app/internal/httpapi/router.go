// Package httpapi wires the chi/v5 router: handlers, middleware and the
// embedded-static file server. Handlers are thin — they deserialize/
// serialize JSON and delegate to items.Service; the domain never imports
// net/http (design.md §4/§5).
package httpapi

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"niu/internal/auth"
	"niu/internal/items"
)

// HealthChecker is implemented by internal/store — a trivial query
// against SQLite (REL-03).
type HealthChecker interface {
	Healthy() error
}

// Server holds the dependencies needed by the HTTP handlers.
type Server struct {
	items  *items.Service
	health HealthChecker
}

// NewRouter builds the complete chi/v5 router: security headers first
// (before any other middleware, design.md §9), then auth injection, then
// routes for /api/v1/*, /healthz, and the embedded static frontend for
// everything else.
func NewRouter(svc *items.Service, health HealthChecker, authenticator auth.Authenticator, webFS fs.FS) http.Handler {
	s := &Server{items: svc, health: health}

	r := chi.NewRouter()

	// SecurityHeaders MUST be the outermost middleware — applies to API
	// and static responses alike (S7, NFR-02).
	r.Use(SecurityHeaders)
	r.Use(Recoverer)
	r.Use(chimw.Logger)

	// /healthz is intentionally outside /api/v1 and outside the auth
	// middleware — liveness must not depend on identity.
	r.Get("/healthz", s.handleHealthz)

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(WithCurrentUser(authenticator))

		api.Get("/me", s.handleMe)

		api.Route("/items", func(items chi.Router) {
			items.Get("/", s.handleListItems)
			items.Post("/", s.handleCreateItem)
			items.Route("/{id}", func(item chi.Router) {
				item.Patch("/", s.handleUpdateItem)
				item.Delete("/", s.handleDeleteItem)
			})
		})
	})

	// Static frontend: anything not matching /api/v1/* or /healthz is
	// served from the embedded web/ tree (design.md §8, PLAN.md §2.1 — no
	// server-side templating).
	fileServer := http.FileServer(http.FS(webFS))
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	})

	return r
}
