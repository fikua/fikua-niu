// Package auth defines the authentication seam (ADR-03). NIU-1 ships only
// a stub implementation that always resolves to the same seeded user;
// NIU-4 will add a SessionAuthenticator that reads a cookie and validates
// it against the sessions table — without touching any handler in
// internal/httpapi.
package auth

import (
	"context"
	"errors"
	"net/http"
)

// User is the identity resolved for the current request.
type User struct {
	ID int64
	// SessionTokenHash is the SHA-256 hash of the session token that
	// resolved this user, when resolved via a real session (empty for
	// StubAuthenticator). httpapi.RequireCSRF (ADR-05) derives the
	// expected CSRF value from this field instead of re-reading the
	// session cookie a second time.
	SessionTokenHash string
}

// ErrUnauthenticated is returned by Authenticator.CurrentUser when no
// valid session is present (absent cookie, mutated token, unknown token,
// or expired session — EC-01/EC-02/EC-11/AC-05/AC-12). httpapi maps this
// specific error to 401; any other error is a genuine internal failure and
// maps to 500.
var ErrUnauthenticated = errors.New("auth: unauthenticated")

// Authenticator resolves the current user for an incoming request.
// httpapi calls this exclusively through the WithCurrentUser middleware —
// handlers never read cookies/headers directly (ADR-03).
type Authenticator interface {
	CurrentUser(r *http.Request) (User, error)
}

// StubAuthenticator ignores the request entirely and always returns the
// same seeded user. This is NIU-1's implementation of the Authenticator
// seam; NIU-4 replaces it with a real session-backed authenticator.
type StubAuthenticator struct {
	UserID int64
}

// CurrentUser always returns the configured stub user, regardless of the
// incoming request.
func (a StubAuthenticator) CurrentUser(_ *http.Request) (User, error) {
	return User{ID: a.UserID}, nil
}

type contextKey int

const userContextKey contextKey = iota

// WithUser returns a new context carrying the given user, for use by the
// WithCurrentUser middleware in internal/httpapi.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

// FromContext extracts the current user injected by the WithCurrentUser
// middleware. ok is false if no user was injected (should not happen in
// practice once the middleware is wired).
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userContextKey).(User)
	return u, ok
}
