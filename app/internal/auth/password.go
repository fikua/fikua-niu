package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// SessionTTL is the lifetime of a newly created session (30 days,
// human-confirmed, requirements.md §8 / design.md §1).
const SessionTTL = 30 * 24 * time.Hour

// SessionCookieName is the HttpOnly cookie carrying the opaque session
// token (design.md §6.1).
const SessionCookieName = "niu_session"

// dummyPassword is never a real credential — it exists purely so
// PasswordAuthenticator has a fixed bcrypt hash to compare against when a
// username does not exist (ADR-02). It is not a secret: it protects
// nothing, it only makes bcrypt run unconditionally.
const dummyPassword = "niu-dummy-password-for-timing-parity-only"

// bcryptCost is fixed at 12 (NFR-01: >200ms per verification on
// deployment-class hardware).
const bcryptCost = 12

// Rate-limit thresholds (ADR-01): 10 failed attempts per normalized
// username, 20 per IP, both within the same 5-minute fixed window
// (RateLimiter).
const (
	rateLimitUserThreshold = 10
	rateLimitIPThreshold   = 20
)

// ErrInvalidCredentials is returned by Login when the username does not
// exist or the password does not match. Callers must treat both cases
// identically (AC-11/S5) — the error never distinguishes which happened.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")

// ErrRateLimited is returned by Login when either the per-username or
// per-IP threshold has already been exceeded (AC-10) — bcrypt is never
// invoked in this path (ADR-01/ADR-03).
var ErrRateLimited = errors.New("auth: rate limited")

// PasswordAuthenticator implements Authenticator against the sessions/
// users tables, replacing StubAuthenticator behind the same interface
// (ADR-03 NIU-1). It never imports net/http for its business logic beyond
// what Authenticator itself already requires (design.md §4) — CurrentUser
// is the only method that reads a request, and it only reads a cookie.
type PasswordAuthenticator struct {
	db            *sql.DB
	sessionSecret string
	dummyHash     []byte
	rateLimiter   *RateLimiter
}

// NewPasswordAuthenticator builds a PasswordAuthenticator, precomputing
// the dummy bcrypt hash exactly once at construction time (ADR-02) — never
// per-request.
func NewPasswordAuthenticator(db *sql.DB, sessionSecret string) (*PasswordAuthenticator, error) {
	dummyHash, err := bcrypt.GenerateFromPassword([]byte(dummyPassword), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth: precompute dummy hash: %w", err)
	}
	return &PasswordAuthenticator{
		db:            db,
		sessionSecret: sessionSecret,
		dummyHash:     dummyHash,
		rateLimiter:   NewRateLimiter(),
	}, nil
}

// NormalizeUsername applies the same normalization the login pipeline uses
// for both the credential lookup and the rate-limiter key (design.md
// ADR-01 references NIU-1's EC-06 pattern): trim, then Unicode-aware
// lowercase. Usernames are ASCII by construction (seeded via config), but
// the same normalization function is used consistently everywhere a
// username is compared or keyed.
func NormalizeUsername(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// Login verifies username/password and returns a new opaque session token
// on success, following ADR-03's strict pipeline order: rate limiter check
// first (both keys, 429-equivalent ErrRateLimited without calling bcrypt),
// then bcrypt.CompareHashAndPassword exactly once regardless of whether
// the username exists (ADR-02) — never a conditional branch that skips the
// call. Only a real credential failure (not a validation failure — see
// httpapi.handleLogin step 2, which never reaches here) records an attempt
// against the rate limiter.
//
// ip is the caller's rate-limiting key (design.md's Cf-Connecting-Ip /
// RemoteAddr resolution happens in httpapi, which passes the result here).
func (a *PasswordAuthenticator) Login(username, password, ip string) (token string, userID int64, err error) {
	normalized := NormalizeUsername(username)

	// Reserve atomically checks-and-provisionally-counts this attempt
	// against both keys in one critical section per key (audit finding
	// F-01: the previous Allow-then-RecordFailure pair left a window
	// where concurrent requests could all pass the check before any of
	// them recorded a failure, letting more than the configured limit
	// through — measured at 11-15 admitted against a limit of 10).
	//
	// A successful login rolls the provisional count back below (ADR-03:
	// a correct password must not consume rate-limit budget), so the
	// pre-increment here is safe for the common case; it only sticks for
	// requests that go on to fail.
	userReserved := a.rateLimiter.Reserve(normalized, rateLimitUserThreshold)
	ipReserved := a.rateLimiter.Reserve(ip, rateLimitIPThreshold)
	if !userReserved || !ipReserved {
		// Whichever key WAS reserved (the other one might still have
		// succeeded) must be rolled back — this request is being
		// rejected outright, so neither key should end up double-counted
		// relative to a request that never reserved at all.
		if userReserved {
			a.rateLimiter.Rollback(normalized)
		}
		if ipReserved {
			a.rateLimiter.Rollback(ip)
		}
		return "", 0, ErrRateLimited
	}

	var id int64
	var hash string
	lookupErr := a.db.QueryRow(
		`SELECT id, password_hash FROM users WHERE name = ?`, normalized,
	).Scan(&id, &hash)

	compareHash := a.dummyHash
	found := lookupErr == nil
	if found {
		compareHash = []byte(hash)
	} else if !errors.Is(lookupErr, sql.ErrNoRows) {
		a.rateLimiter.Rollback(normalized)
		a.rateLimiter.Rollback(ip)
		return "", 0, fmt.Errorf("auth: lookup user: %w", lookupErr)
	}

	// ADR-02: bcrypt runs unconditionally, against the real hash if found,
	// the dummy hash otherwise — the dummy comparison always returns
	// false, but the CPU cost is paid identically either way.
	cmpErr := bcrypt.CompareHashAndPassword(compareHash, []byte(password))

	if !found || cmpErr != nil {
		// The provisional reservation above already counted this attempt
		// — nothing further to record. Kept as a genuine failure (not
		// rolled back), unlike the success path below.
		return "", 0, ErrInvalidCredentials
	}

	// Success: the reservation above provisionally counted this attempt
	// as a failure. Undo that — a correct password must never consume
	// brute-force budget (AC-10/ADR-03).
	a.rateLimiter.Rollback(normalized)
	a.rateLimiter.Rollback(ip)

	tok, err := a.CreateSession(id)
	if err != nil {
		return "", 0, err
	}
	return tok, id, nil
}

// sqliteTimestampFormat matches the text format SQLite's own
// CURRENT_TIMESTAMP produces (UTC, "YYYY-MM-DD HH:MM:SS") — every
// timestamp this package writes must use the same format, otherwise a
// naive Go time.Time (RFC3339, with a timezone offset suffix) sorts and
// compares incorrectly against a column populated by CURRENT_TIMESTAMP
// (ADR-04's CleanupExpired relies on this comparison being correct).
const sqliteTimestampFormat = "2006-01-02 15:04:05"

func formatSQLiteTimestamp(t time.Time) string {
	return t.UTC().Format(sqliteTimestampFormat)
}

// CreateSession generates a new 256-bit opaque token, persists only its
// SHA-256 hash (AC-08), and returns the plaintext token — which the caller
// must place in a cookie and never store anywhere else (design.md §5 Flux
// 1 step 7).
func (a *PasswordAuthenticator) CreateSession(userID int64) (token string, err error) {
	raw := make([]byte, 32) // 256 bits
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("auth: generate session token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	hash := hashToken(token)

	expiresAt := formatSQLiteTimestamp(time.Now().Add(SessionTTL))
	_, err = a.db.Exec(
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		hash, userID, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("auth: insert session: %w", err)
	}
	return token, nil
}

// Logout deletes the session identified by the given plaintext token
// (idempotent: deleting an absent hash is not an error).
func (a *PasswordAuthenticator) Logout(token string) error {
	_, err := a.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	if err != nil {
		return fmt.Errorf("auth: delete session: %w", err)
	}
	return nil
}

// CleanupExpired deletes every session past its expiry (ADR-04, EC-10/
// NFR-08), reusing the single *sql.DB connection already open, and prunes
// stale rate-limiter buckets on the same pass (ADR-01/ADR-04: one ticker,
// one goroutine, no second background process).
func (a *PasswordAuthenticator) CleanupExpired() error {
	_, err := a.db.Exec(`DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`)
	if err != nil {
		return fmt.Errorf("auth: cleanup expired sessions: %w", err)
	}
	a.rateLimiter.Cleanup()
	return nil
}

// CurrentUser implements Authenticator: resolves the session cookie,
// hashes it, and looks it up against sessions — rejecting anything absent,
// mutated, or expired with the exact same outcome (EC-01/EC-02/EC-11).
func (a *PasswordAuthenticator) CurrentUser(r *http.Request) (User, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return User{}, ErrUnauthenticated
	}

	hash := hashToken(cookie.Value)

	var userID int64
	var expiresAt time.Time
	qerr := a.db.QueryRow(
		`SELECT user_id, expires_at FROM sessions WHERE token_hash = ?`, hash,
	).Scan(&userID, &expiresAt)
	if errors.Is(qerr, sql.ErrNoRows) {
		return User{}, ErrUnauthenticated
	}
	if qerr != nil {
		return User{}, fmt.Errorf("auth: lookup session: %w", qerr)
	}
	if !time.Now().Before(expiresAt) {
		return User{}, ErrUnauthenticated
	}

	return User{ID: userID, SessionTokenHash: hash}, nil
}

// TokenHashForSession returns SHA-256(token) hex-encoded — exposed so
// httpapi can derive the CSRF token (ADR-05) from the session's token_hash
// without duplicating the hashing logic or reaching into internal state.
func TokenHashForSession(token string) string {
	return hashToken(token)
}

// SessionSecret exposes the HMAC key used for the CSRF token (ADR-05) so
// httpapi.RequireCSRF can recompute the expected value without a second
// copy of the secret.
func (a *PasswordAuthenticator) SessionSecret() string {
	return a.sessionSecret
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
