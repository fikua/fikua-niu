package fetchsafe

import "errors"

// Typed errors returned by FetchPreview (design.md §4/ADR-02, tasks.md
// T-03). Callers (ideas.Service, via the worker pool) map any of these to
// preview_status='failed' without distinguishing the reason to the user
// (NFR-06 — "missatge clar sense revelar detalls interns").
var (
	// ErrSchemeRejected is returned when the URL's scheme is not http or
	// https (T-03a, NFR-05/EC-01) — checked before any network activity.
	ErrSchemeRejected = errors.New("fetchsafe: URL scheme not allowed")

	// ErrDestinationForbidden is returned when the resolved destination
	// (by hostname denylist, T-03b, or by IP classification, T-03c) is
	// not allowed (NFR-06/EC-02/EC-07).
	ErrDestinationForbidden = errors.New("fetchsafe: destination not allowed")

	// ErrTimeout is returned when the fetch does not complete within the
	// hard deadline (T-03f, NFR-07/EC-08).
	ErrTimeout = errors.New("fetchsafe: fetch timed out")

	// ErrResponseTooLarge is returned when the response body exceeds the
	// streaming size limit before a usable <head> could be read (T-03g,
	// NFR-07/EC-03).
	ErrResponseTooLarge = errors.New("fetchsafe: response exceeded size limit")

	// ErrUnsupportedContentType is returned when the response's
	// Content-Type is not HTML-compatible (T-04, EC-09) — no parsing is
	// attempted.
	ErrUnsupportedContentType = errors.New("fetchsafe: unsupported content type")

	// ErrTooManyRedirects is returned when the redirect chain exceeds the
	// 5-hop limit (T-03e, NFR-06/NFR-07/EC-04).
	ErrTooManyRedirects = errors.New("fetchsafe: too many redirects")
)
