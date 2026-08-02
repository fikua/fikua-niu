package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"

	"niu/internal/auth"
)

// CSRFCookieName is the non-HttpOnly cookie carrying the double-submit
// CSRF token (ADR-05) — deliberately readable by JS.
const CSRFCookieName = "niu_csrf"

// csrfHeaderName is the header the client echoes the CSRF cookie value
// into on every mutating request (ADR-05).
const csrfHeaderName = "X-CSRF-Token"

// csrfCookieMaxAgeSeconds mirrors the session cookie's lifetime (30 days,
// design.md §6.1) — the CSRF cookie has no independent lifetime of its
// own, it is only ever regenerated alongside a new session.
const csrfCookieMaxAgeSeconds = 30 * 24 * 60 * 60

// deriveCSRFToken computes HMAC-SHA256(sessionSecret, tokenHash) and
// base64 URL-safe encodes it (ADR-05). It is a pure function of the
// session's token_hash and the server-wide secret — nothing is persisted,
// the value is recomputed on demand both when issuing the cookie at login
// and when verifying it on every mutation.
func deriveCSRFToken(sessionSecret, tokenHash string) string {
	mac := hmac.New(sha256.New, []byte(sessionSecret))
	mac.Write([]byte(tokenHash))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// RequireCSRF is a double-submit CSRF guard (ADR-05). It must be mounted
// AFTER WithCurrentUser (it reads the user resolved by that middleware),
// only on mutating routes (POST/PATCH/DELETE) — never on GET, never on
// POST /api/v1/auth/login (design.md §4, risk R-07).
//
// It recomputes the expected CSRF value from the session's token_hash
// (auth.User.SessionTokenHash, populated by WithCurrentUser) and compares
// it against the X-CSRF-Token header in constant time. It does NOT re-read
// the session cookie itself — it trusts the user already resolved by
// WithCurrentUser, so an absent/invalid session already produced a 401
// before this middleware runs.
func RequireCSRF(sessionSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.FromContext(r.Context())
			if !ok || user.SessionTokenHash == "" {
				writeError(w, http.StatusUnauthorized, "unauthenticated", "Cal iniciar sessió.")
				return
			}

			expected := deriveCSRFToken(sessionSecret, user.SessionTokenHash)
			got := r.Header.Get(csrfHeaderName)

			if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
				writeError(w, http.StatusForbidden, "csrf_failed", msgCSRFFailed)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// setSessionCookies writes both the HttpOnly session cookie and the
// non-HttpOnly CSRF cookie on a successful login (design.md §6.1). secure
// controls the Secure attribute — omitted only in local development
// (NIU_ENV=development) against plain HTTP, matching NIU-1's existing
// behaviour.
func setSessionCookies(w http.ResponseWriter, sessionToken, sessionSecret string, secure bool) {
	tokenHash := auth.TokenHashForSession(sessionToken)
	csrfToken := deriveCSRFToken(sessionSecret, tokenHash)

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   csrfCookieMaxAgeSeconds,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   csrfCookieMaxAgeSeconds,
	})
}

// clearSessionCookies expires both cookies client-side on logout — belt
// and braces alongside the server-side session deletion (AC-04/AC-09).
func clearSessionCookies(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
