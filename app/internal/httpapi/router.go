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
	"niu/internal/projects"
)

// HealthChecker is implemented by internal/store — a trivial query
// against SQLite (REL-03).
type HealthChecker interface {
	Healthy() error
}

// credentialAuthenticator is implemented by auth.PasswordAuthenticator
// (NIU-4). NewRouter type-asserts the given auth.Authenticator against
// this interface: when satisfied (production, and any test that opts in),
// the login/logout routes and RequireCSRF are mounted; when not (NIU-1's
// auth.StubAuthenticator and the various test fakes), those routes are
// simply absent — the existing NIU-1 surface is completely unchanged, so
// no NIU-1 test call site needs to change (tasks.md T-18 note: router.go
// gets a surgical addition, not a rewrite).
type credentialAuthenticator interface {
	auth.Authenticator
	Login(username, password, ip string) (token string, userID int64, err error)
	Logout(token string) error
	SessionSecret() string
}

// Server holds the dependencies needed by the HTTP handlers.
type Server struct {
	items         *items.Service
	projects      *projects.Service
	health        HealthChecker
	authenticator credentialAuthenticator
	cookiesSecure bool
}

// NewRouter builds the complete chi/v5 router: security headers first
// (before any other middleware, design.md §9), then auth injection, then
// routes for /api/v1/*, /healthz, and the embedded static frontend for
// everything else.
//
// cookiesSecure controls the Secure attribute on the session/CSRF cookies
// (design.md §6.1) — false only for local development over plain HTTP.
func NewRouter(svc *items.Service, projectsSvc *projects.Service, health HealthChecker, authenticator auth.Authenticator, webFS fs.FS, cookiesSecure bool) http.Handler {
	s := &Server{items: svc, projects: projectsSvc, health: health, cookiesSecure: cookiesSecure}
	s.authenticator, _ = authenticator.(credentialAuthenticator)

	r := chi.NewRouter()

	// SecurityHeaders MUST be the outermost middleware — applies to API
	// and static responses alike (S7, NFR-02).
	r.Use(SecurityHeaders)
	// Body cap sits high in the chain so it applies before any handler
	// reads — the point is to never materialise a huge body at all.
	r.Use(LimitBody)
	r.Use(Recoverer)
	r.Use(chimw.Logger)

	// /healthz is intentionally outside /api/v1 and outside the auth
	// middleware — liveness must not depend on identity.
	r.Get("/healthz", s.handleHealthz)

	// POST /api/v1/auth/login is mounted on the OUTER router, before
	// entering the /api/v1 group's WithCurrentUser middleware (no session
	// exists yet) and outside RequireCSRF (design.md §4, risk R-07). It is
	// registered here — rather than inside r.Route("/api/v1", ...) — only
	// because chi requires every Use() on a sub-router to precede that
	// sub-router's own route registrations; mounting login on the outer
	// mux sidesteps that ordering constraint without weakening the
	// contract (still exactly "/api/v1/auth/login", still outside
	// WithCurrentUser/RequireCSRF). Only mounted when the authenticator
	// supports real credentials — StubAuthenticator (NIU-1) never exposes
	// this route.
	if s.authenticator != nil {
		r.Post("/api/v1/auth/login", s.handleLogin)
	}

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(WithCurrentUser(authenticator))

		api.Get("/me", s.handleMe)

		if s.authenticator != nil {
			api.With(RequireCSRF(s.authenticator.SessionSecret())).Post("/auth/logout", s.handleLogout)
		}

		api.Route("/items", func(items chi.Router) {
			items.Get("/", s.handleListItems)
			if s.authenticator != nil {
				items.With(RequireCSRF(s.authenticator.SessionSecret())).Post("/", s.handleCreateItem)
				items.Route("/{id}", func(item chi.Router) {
					item.With(RequireCSRF(s.authenticator.SessionSecret())).Patch("/", s.handleUpdateItem)
					item.With(RequireCSRF(s.authenticator.SessionSecret())).Delete("/", s.handleDeleteItem)
				})
			} else {
				// NIU-1 fixtures/tests without a credentialAuthenticator —
				// no CSRF layer, identical to the pre-NIU-4 router.
				items.Post("/", s.handleCreateItem)
				items.Route("/{id}", func(item chi.Router) {
					item.Patch("/", s.handleUpdateItem)
					item.Delete("/", s.handleDeleteItem)
				})
			}
		})

		// NIU-5: "compres grans i projectes de casa" — same
		// WithCurrentUser/RequireCSRF pattern as /items above, no new auth
		// surface (design.md §5/§8, EC-10/EC-11/NFR-04/NFR-05).
		api.Route("/projects", func(proj chi.Router) {
			proj.Get("/", s.handleListProjects)
			if s.authenticator != nil {
				proj.With(RequireCSRF(s.authenticator.SessionSecret())).Post("/", s.handleCreateProject)
				proj.Route("/{id}", func(project chi.Router) {
					project.With(RequireCSRF(s.authenticator.SessionSecret())).Patch("/", s.handlePatchProjectState)
					project.With(RequireCSRF(s.authenticator.SessionSecret())).Delete("/", s.handleDeleteProject)
				})
			} else {
				proj.Post("/", s.handleCreateProject)
				proj.Route("/{id}", func(project chi.Router) {
					project.Patch("/", s.handlePatchProjectState)
					project.Delete("/", s.handleDeleteProject)
				})
			}
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
