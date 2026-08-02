package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"niu/internal/auth"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// clientIP resolves the request's originating address for rate-limiting
// purposes: Cf-Connecting-Ip when present (the same header PLAN.md §5.2
// already requires Traefik to forward — without it, requests would appear
// to come from Cloudflare's rotating edge IPs), falling back to
// r.RemoteAddr for local development (ADR-01). The port is stripped from
// RemoteAddr — otherwise every connection would carry a distinct ephemeral
// port and never share a rate-limit key with any other request from the
// same host.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("Cf-Connecting-Ip"); ip != "" {
		return ip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// handleLogin is POST /api/v1/auth/login, mounted OUTSIDE WithCurrentUser
// (no session exists yet) and OUTSIDE RequireCSRF (design.md §4, risk
// R-07). It follows ADR-03's strict pipeline order: JSON decode -> pure
// input validation (never touches the rate limiter) -> auth.Login, which
// itself checks the rate limiter before bcrypt (ADR-01/ADR-02/ADR-03).
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	// Step 1 — JSON decode.
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", msgLoginValidation)
		return
	}

	// Step 2 — pure input validation, previous to and independent of the
	// rate limiter (EC-08/EC-09): a payload missing either field never
	// consumes a brute-force attempt.
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", msgLoginValidation)
		return
	}

	normalizedUser := auth.NormalizeUsername(username)

	// Steps 3-4 — rate limiter then bcrypt, both inside auth.Login
	// (ADR-01/ADR-02/ADR-03).
	token, userID, err := s.authenticator.Login(username, req.Password, ip)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrRateLimited):
			slog.Info("login attempt", "username", normalizedUser, "result", "rate_limited", "ip", ip)
			writeError(w, http.StatusTooManyRequests, "rate_limited", msgRateLimited)
		case errors.Is(err, auth.ErrInvalidCredentials):
			slog.Info("login attempt", "username", normalizedUser, "result", "failure", "ip", ip)
			writeError(w, http.StatusUnauthorized, "invalid_credentials", msgInvalidCredentials)
		default:
			slog.Error("login: internal failure", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		}
		return
	}

	// Step 5 — success: cookies + 200 with the user payload. A success is
	// not a failed attempt, so the rate limiter counters are left as-is —
	// they simply expire with the window (ADR-03 step 5).
	slog.Info("login attempt", "username", normalizedUser, "result", "success", "ip", ip)

	setSessionCookies(w, token, s.authenticator.SessionSecret(), s.cookiesSecure)

	full, err := s.items.CurrentUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": map[string]any{
		"id":           full.ID,
		"display_name": full.DisplayName,
		"avatar_emoji": full.AvatarEmoji,
	}})
}

// handleLogout is POST /api/v1/auth/logout, mounted INSIDE WithCurrentUser
// (needs to resolve which session to invalidate) and INSIDE RequireCSRF
// (it is a mutation, design.md §5 Flux 3).
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil && cookie.Value != "" {
		if err := s.authenticator.Logout(cookie.Value); err != nil {
			slog.Error("logout: internal failure", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "S'ha produït un error inesperat.")
			return
		}
	}

	clearSessionCookies(w, s.cookiesSecure)
	w.WriteHeader(http.StatusNoContent)
}
