// client.go implements T-03d (DisableKeepAlives), T-03e (CheckRedirect —
// hop limit + per-hop scheme re-validation), and T-03h (the dedicated,
// credential-free http.Client, constructed once at app startup).
package fetchsafe

import (
	"fmt"
	"net/http"
)

// maxRedirects is the hard cap on the redirect chain length (T-03e,
// NFR-06/NFR-07/EC-04) — a redirect chain arriving here has already
// followed this many hops.
const maxRedirects = 5

// userAgent is the ONLY outgoing header fetchsafe's client ever attaches
// beyond what net/http sets by default (T-03h, NFR-08) — no Cookie, no
// Authorization, no Niu secret of any kind travels to the external
// destination.
const userAgent = "Niu-LinkPreview/1.0 (+https://niu.fikua.com)"

// checkRedirect implements T-03e's two explicit responsibilities:
//
//  1. Limit the chain to maxRedirects hops (an error past that).
//  2. Re-validate, explicitly and again, that the NEXT URL the client is
//     about to follow has an http/https scheme — rejecting any 30x toward
//     file://, javascript:, or any other scheme a hostile server might
//     attempt.
//
// This check does NOT replace T-03c's IP validation — it is a cheap,
// independent defense-in-depth layer: if DisableKeepAlives (T-03d) were
// ever accidentally reverted in a future change, this still blocks
// dangerous schemes before a connection is attempted for that hop.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return ErrTooManyRedirects
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("%w: redirect to non-http(s) scheme %q", ErrSchemeRejected, req.URL.Scheme)
	}
	return nil
}

// newTransport builds the http.Transport dedicated to fetchsafe.
//
// DisableKeepAlives: true (T-03d, F-01/F-06) is load-bearing, not an
// optional performance tweak: without it, a chain of redirects toward the
// same host reuses the already-open TCP connection and NEVER calls
// DialContext/ControlContext again for those subsequent hops — silently
// bypassing IP validation entirely for that case. security-engineer
// verified this empirically: a 4-hop redirect chain to the same host
// triggered ControlContext exactly once without this flag. A one-shot,
// 5-second-budget preview fetch pays no meaningful performance cost for
// disabling keep-alives — this is not a client reused repeatedly against
// the same host.
//
// ForceAttemptHTTP2: true is required for real-world coverage, not a
// performance tweak: major sites (Instagram, and Meta properties
// generally) treat an HTTP/1.1 request as a bot and answer 429 with a
// redirect to their login page, so no Open Graph tag is ever reachable
// over HTTP/1.1 — every such preview resolved to 'failed'. Verified
// empirically from the deployment host: identical request, 3/3 runs 429
// with ForceAttemptHTTP2:false vs 3/3 runs 200 + full OG tags with it
// true.
//
// This does NOT weaken the DisableKeepAlives guarantee above, which was
// the reason to be careful here (HTTP/2 multiplexes many requests over a
// single connection, exactly the shape that could skip per-hop dialing).
// Verified explicitly: a 4-hop redirect chain to the same host triggers
// ControlContext 5 times under HTTP/2, identical to HTTP/1.1 — every hop
// still re-validates its IP.
func newTransport() *http.Transport {
	return &http.Transport{
		DialContext:         newDialer().DialContext,
		DisableKeepAlives:   true,
		ForceAttemptHTTP2:   true,
		MaxIdleConnsPerHost: -1,
	}
}

// NewClient builds fetchsafe's dedicated http.Client (T-03h) — constructed
// ONCE at application startup by main.go and reused for every FetchPreview
// call, never one client per request. It shares no Transport with any
// other part of Niu (there is no other outgoing HTTP client today; the
// separation is explicit so no future addition reuses it by accident).
func NewClient() *http.Client {
	return &http.Client{
		Transport:     newTransport(),
		CheckRedirect: checkRedirect,
		// Timeout is a second-level safety net alongside the
		// context.WithTimeout(5s) that FetchPreview wraps the whole call
		// with (T-03f) — kept in sync with fetchTimeout.
		Timeout: fetchTimeout,
	}
}
