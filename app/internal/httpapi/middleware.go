package httpapi

import (
	"log/slog"
	"net/http"

	"niu/internal/auth"
)

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
