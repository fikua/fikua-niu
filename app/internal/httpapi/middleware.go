package httpapi

import (
	"log/slog"
	"net/http"

	"niu/internal/auth"
)

// maxRequestBody caps how much of a request body the server will read.
// An item name is at most 200 characters, so even a generous JSON
// envelope fits in a few KiB; 64 KiB leaves room to spare while keeping
// the ceiling far below anything that could hurt.
const maxRequestBody = 64 << 10

// LimitBody caps every request body at maxRequestBody.
//
// Without it the handlers decode whatever arrives before any length check
// runs, so a 128 MiB body was allocated in full and only then rejected
// for exceeding 200 characters — measured at ~896 MiB of memory for a
// single request. A handful in parallel would OOM the process on the
// shared VPS and take SQLite down with it.
//
// Note this is not mitigated by putting the app behind Cloudflare Access:
// the request comes from an already-authenticated legitimate user, and a
// buggy client loop would do it by accident.
func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders applies the mandatory security headers (S7, NFR-02) to
// every response — API and static files alike. It MUST be mounted before
// any other middleware on the root router (design.md §9).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self'; "+
				"font-src 'self'; img-src 'self'; connect-src 'self'; "+
				"object-src 'none'; base-uri 'none'")
		next.ServeHTTP(w, r)
	})
}

// WithCurrentUser injects the current user (resolved via authenticator)
// into the request context, so handlers can call auth.FromContext instead
// of reading cookies/headers directly (ADR-03).
func WithCurrentUser(authenticator auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := authenticator.CurrentUser(r)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
				slog.Error("auth: failed to resolve current user", "error", err)
				return
			}
			ctx := auth.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Recoverer wraps a handler, recovering from panics and responding with a
// generic 500 envelope — the panic detail is logged server-side only,
// never serialized to the client (design.md §9, PLAN.md §2.5).
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
