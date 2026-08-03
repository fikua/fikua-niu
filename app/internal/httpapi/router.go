// Package httpapi wires the chi/v5 router: handlers, middleware and the
// embedded-static file server. Handlers are thin — they deserialize/
// serialize JSON and delegate to items.Service; the domain never imports
// net/http (design.md §4/§5).
package httpapi

import (
	"io"
	"io/fs"
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"niu/internal/auth"
	"niu/internal/ideas"
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
	ideas         *ideas.Service
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
func NewRouter(svc *items.Service, projectsSvc *projects.Service, ideasSvc *ideas.Service, health HealthChecker, authenticator auth.Authenticator, webFS fs.FS, cookiesSecure bool) http.Handler {
	s := &Server{items: svc, projects: projectsSvc, ideas: ideasSvc, health: health, cookiesSecure: cookiesSecure}
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

		// NIU-6: "idees d'activitats amb previsualització de link" — same
		// WithCurrentUser/RequireCSRF pattern as /items and /projects above,
		// no new auth surface (design.md §6.1, EC-13/EC-14/NFR-03/NFR-04).
		// GET never mutates (no CSRF); POST/DELETE require it. There is no
		// PATCH endpoint in this space (no lifecycle/state field, ADR-01).
		api.Route("/ideas", func(idea chi.Router) {
			idea.Get("/", s.handleListIdeas)
			if s.authenticator != nil {
				idea.With(RequireCSRF(s.authenticator.SessionSecret())).Post("/", s.handleCreateIdea)
				idea.Route("/{id}", func(one chi.Router) {
					one.With(RequireCSRF(s.authenticator.SessionSecret())).Delete("/", s.handleDeleteIdea)
				})
			} else {
				idea.Post("/", s.handleCreateIdea)
				idea.Route("/{id}", func(one chi.Router) {
					one.Delete("/", s.handleDeleteIdea)
				})
			}
		})
	})

	// Static frontend: anything not matching /api/v1/* or /healthz is
	// served from the embedded web/ tree (design.md §8, PLAN.md §2.1 — no
	// server-side templating).
	//
	// SPA fallback (frontend SPA merge): client-side routing means a real
	// browser navigation to e.g. /projects (bookmark, manual URL entry, or
	// a page refresh while on that view) is a genuine GET /projects
	// request that has no corresponding file in webFS — only the router
	// running INSIDE index.html (js/main.js) knows how to render that
	// route. Without this fallback such a request 404s. spaFallback serves
	// index.html for any GET/HEAD request whose path does not resolve to a
	// real file in webFS, so CSS/JS modules/manifest/icons continue to
	// resolve normally (they exist in webFS and are served as-is) and only
	// unmatched, HTML-navigation-shaped routes fall back to the shell.
	fileServer := http.FileServer(http.FS(webFS))
	r.NotFound(spaFallback(webFS, fileServer))

	return r
}

// spaRoutes is the exact set of client-side routes js/main.js's router
// knows how to render (mirrors the ROUTES map in app/web/js/main.js — keep
// both in sync). Only these paths fall back to the SPA shell; every other
// unmatched GET/HEAD still 404s normally. Without this allowlist, ANY
// missing asset (e.g. a typo'd import path) would silently return 200 +
// the shell instead of a clear 404, masking real bugs as soft navigations
// (found in code review, see spa-conversion-adhoc-review.md F-01).
var spaRoutes = map[string]bool{
	"/":         true,
	"/projects": true,
	"/ideas":    true,
}

// spaFallback wraps the embedded-FS file server: if the request is a
// GET/HEAD for one of spaRoutes (a client-side route, not a static asset),
// it serves index.html instead of a 404 (classic SPA server-side
// fallback) — necessary because a real browser navigation to e.g.
// /projects (bookmark, manual URL entry, or a page refresh while on that
// view) is a genuine GET /projects request with no corresponding file in
// webFS; only the router running INSIDE index.html (js/main.js) knows how
// to render that route. Any other unmatched path (a genuinely missing
// asset, a typo, an unknown route) falls through to fileServer, which
// 404s exactly as it did before this fallback existed.
//
// The fallback branch serves index.html's bytes directly via
// http.ServeContent rather than delegating to fileServer with a rewritten
// URL.Path — http.FileServer special-cases any request whose served file
// is named "index.html" and issues a 301 to strip it from the URL
// ("Location: ./"), which is correct for a real directory index but wrong
// here: a request for /projects must render the SPA shell IN PLACE, at
// the /projects URL, not redirect the browser back to /.
func spaFallback(webFS fs.FS, fileServer http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fileServer.ServeHTTP(w, r)
			return
		}

		if spaRoutes[path.Clean(r.URL.Path)] {
			serveIndex(w, r, webFS)
			return
		}

		fileServer.ServeHTTP(w, r)
	}
}

// serveIndex reads index.html from webFS and serves it directly, keeping
// the request's original URL (so the browser's address bar and
// history.pushState-based router both stay at e.g. /projects, not / ).
func serveIndex(w http.ResponseWriter, r *http.Request, webFS fs.FS) {
	f, err := webFS.Open("index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	rs, ok := f.(io.ReadSeeker)
	if !ok {
		http.Error(w, "internal_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", stat.ModTime(), rs)
}
