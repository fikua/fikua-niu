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

	// ErrResponseTooLarge is declared for an oversized-response case that
	// is, in current production code, handled silently rather than by
	// returning this error (F-05, review.md): the io.LimitReader (T-03g)
	// truncates the stream before the html tokenizer, which then
	// surfaces the cutoff as an ordinary html.ErrorToken — parseOpenGraph
	// returns whatever partial Preview it already recovered with err ==
	// nil, exactly like any other EC-05 malformed/absent-tag case
	// (design.md §5: "treated as fallback, not a fatal error"). This
	// sentinel's only live reference is a test double
	// (tests/integration/ideas_test_server_test.go) simulating a caller
	// that wants to see this value — it is not currently produced by
	// FetchPreview itself. Kept as documented API surface in case a
	// future change instruments an explicit size check instead of
	// relying on silent truncation.
	ErrResponseTooLarge = errors.New("fetchsafe: response exceeded size limit")

	// ErrUnsupportedContentType is returned when the response's
	// Content-Type is not HTML-compatible (T-04, EC-09) — no parsing is
	// attempted.
	ErrUnsupportedContentType = errors.New("fetchsafe: unsupported content type")

	// ErrTooManyRedirects is returned when the redirect chain exceeds the
	// 5-hop limit (T-03e, NFR-06/NFR-07/EC-04).
	ErrTooManyRedirects = errors.New("fetchsafe: too many redirects")

	// ErrHTTPStatus is returned when the destination answers with a
	// non-2xx status. Before this existed, an error page (a 429 bot wall,
	// a 404, a 500) reached the Open Graph parser as if it were an
	// ordinary page: it parsed cleanly, found no og: tags, and resolved to
	// preview_status='failed' — indistinguishable from a legitimate page
	// that simply has no Open Graph metadata. Rejecting explicitly keeps
	// the two apart in logs. It stays a plain fallback for the user
	// (NFR-06 — the reason is never surfaced), like every sentinel here.
	ErrHTTPStatus = errors.New("fetchsafe: non-success HTTP status")
)
