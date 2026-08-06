// Package fetchsafe is the SINGLE point of entry for any outgoing HTTP
// request toward a URL supplied by a user (design.md ADR-02, tasks.md
// T-03..T-03h). No other package in this application may import net/http
// to fetch a user-controlled URL — every SSRF mitigation lives here, and
// only here, so no other call site can "forget" to apply it (design.md
// §4/§8, risk R-01).
//
// FetchPreview is the package's only public function. It encapsulates, in
// order:
//
//  1. Scheme validation (T-03a) — http/https only, before any network
//     activity.
//  2. Hostname denylist (T-03b) — niu.fikua.com / NIU_PUBLIC_HOST /
//     known traefik-public service names, checked before any DNS
//     resolution.
//  3. IP validation via net.Dialer.ControlContext (T-03c) — the ONLY
//     point of IP classification, combined with DisableKeepAlives
//     (T-03d) so every redirect hop re-triggers it.
//  4. Per-hop redirect re-validation + 5-hop cap (T-03e).
//  5. A hard 5s timeout wrapping the entire call (T-03f).
//  6. A 2MiB streaming size limit (T-03g).
//  7. A dedicated, credential-free http.Client (T-03h).
package fetchsafe

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// fetchTimeout is the hard 5-second budget for the entire fetch — DNS,
// connect, TLS, headers, and body (T-03f, NFR-07/EC-08). Wrapping the
// whole call in context.WithTimeout (rather than relying solely on
// http.Client.Timeout, which is kept as a second-level safety net, see
// client.go) ensures a server that sends headers promptly but then
// stalls the body indefinitely is still cut off at 5s.
const fetchTimeout = 5 * time.Second

// maxResponseBytes is the streaming size cap (T-03g, NFR-07/EC-03): 2MiB,
// applied via io.LimitReader BEFORE the body reaches the HTML parser —
// never io.ReadAll without a limit.
const maxResponseBytes = 2 << 20

// Preview is the result of a successful or partially-successful fetch.
// Partial is true when the response was read and parsed but one or more
// Open Graph fields were absent (EC-05) — the caller (ideas.Service)
// distinguishes preview_status='ready' from 'partial' using this flag.
type Preview struct {
	Title       string
	ImageURL    string
	Description string
	Partial     bool
}

// FetchPreview retrieves Open Graph metadata for rawURL, applying every
// SSRF mitigation described in design.md ADR-02. client MUST be the
// single, dedicated *http.Client built once by NewClient() at application
// startup (T-03h) — FetchPreview never constructs its own client per call.
func FetchPreview(ctx context.Context, client *http.Client, rawURL string) (Preview, error) {
	parsed, err := validateScheme(rawURL)
	if err != nil {
		return Preview{}, err
	}

	// T-03b: hostname denylist, checked before and independently of any
	// DNS resolution/IP validation — zero network requests if it matches.
	if isDeniedHost(parsed.Hostname()) {
		return Preview{}, ErrDestinationForbidden
	}

	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return Preview{}, ErrSchemeRejected
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		if fetchCtx.Err() != nil {
			return Preview{}, ErrTimeout
		}
		// T-03c/T-03e surface ErrDestinationForbidden/ErrSchemeRejected/
		// ErrTooManyRedirects wrapped inside a *url.Error by net/http —
		// unwrap conservatively by checking the known sentinels via
		// errors.Is through the client's own error chain.
		return Preview{}, classifyClientError(err)
	}
	defer resp.Body.Close()

	// A non-2xx response carries no preview worth parsing: its body is an
	// error page, which the OG parser would happily consume and report as
	// "no tags found" — the same outcome as a real page without metadata,
	// with no way to tell them apart afterwards.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Preview{}, ErrHTTPStatus
	}

	contentType := resp.Header.Get("Content-Type")
	if !isHTMLCompatible(contentType) {
		// EC-09: non-HTML content (PDF, image, video, ...) — treated as a
		// fallback directly, no parsing attempt.
		return Preview{}, ErrUnsupportedContentType
	}

	limited := io.LimitReader(resp.Body, maxResponseBytes)
	preview, err := parseOpenGraph(limited)
	if err != nil {
		return Preview{}, err
	}
	return preview, nil
}

// isHTMLCompatible reports whether contentType is compatible with HTML
// parsing (EC-09). An empty Content-Type is treated as HTML-compatible —
// some legitimate servers omit it, and the OG parser itself falls back
// safely (EC-05) if the body turns out not to contain any recognizable
// markup.
func isHTMLCompatible(contentType string) bool {
	if contentType == "" {
		return true
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	switch mediaType {
	case "text/html", "application/xhtml+xml":
		return true
	default:
		return false
	}
}

// classifyClientError maps errors surfaced by http.Client.Do (which
// wraps CheckRedirect/DialContext errors inside *url.Error) back to
// fetchsafe's own typed sentinels, so callers never need to know about
// net/url's wrapping.
func classifyClientError(err error) error {
	switch {
	case containsErr(err, ErrDestinationForbidden):
		return ErrDestinationForbidden
	case containsErr(err, ErrTooManyRedirects):
		return ErrTooManyRedirects
	case containsErr(err, ErrSchemeRejected):
		return ErrSchemeRejected
	default:
		return ErrTimeout
	}
}

func containsErr(err error, target error) bool {
	for err != nil {
		if err == target || err.Error() == target.Error() {
			return true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapper.Unwrap()
	}
	return false
}
