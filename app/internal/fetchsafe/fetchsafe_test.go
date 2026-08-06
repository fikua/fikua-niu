package fetchsafe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// roundTripFunc adapts a function to http.RoundTripper, for tests that
// need to inspect the *http.Request FetchPreview constructs without
// depending on a real dial completing (avoids the loopback-rejection
// problem any locally-bound test server runs into against fetchsafe's
// own IP allowlist).
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestFetchPreview_NoAuthHeaders_RealRequestInspection is the literal
// NFR-08 assertion: the *http.Request FetchPreview builds carries no
// Cookie/Authorization header, and its only custom header is the
// identifiable User-Agent — inspected directly on the request object
// FetchPreview hands to http.Client.Do, via a client whose Transport is
// swapped for an inspecting RoundTripper (bypassing the real dialer, so
// this test needs no network access and is not affected by
// ControlContext's IP allowlist at all).
func TestFetchPreview_NoAuthHeaders_RealRequestInspection(t *testing.T) {
	var captured *http.Request
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			captured = req.Clone(req.Context())
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       http.NoBody,
			}, nil
		}),
	}

	_, err := FetchPreview(context.Background(), client, "https://example.com/page")
	if err != nil {
		t.Fatalf("FetchPreview: %v", err)
	}
	if captured == nil {
		t.Fatal("RoundTrip was never invoked — FetchPreview did not attempt the request")
	}

	if got := captured.Header.Get("Cookie"); got != "" {
		t.Errorf("outgoing request carries a Cookie header: %q (NFR-08 violation)", got)
	}
	if got := captured.Header.Get("Authorization"); got != "" {
		t.Errorf("outgoing request carries an Authorization header: %q (NFR-08 violation)", got)
	}
	if got := captured.Header.Get("User-Agent"); got != userAgent {
		t.Errorf("User-Agent = %q, want %q (NFR-08 — identifiable UA, no auth)", got, userAgent)
	}

	// Only Content-Type + User-Agent should ever be present — assert the
	// full header set is small and known, not just the two we check above,
	// so a stray future header addition (e.g. accidentally forwarding a
	// session cookie) is caught even if it is not named Cookie/Authorization.
	allowedHeaders := map[string]bool{"User-Agent": true, "Accept-Encoding": true}
	for name := range captured.Header {
		if !allowedHeaders[name] {
			t.Errorf("unexpected outgoing header %q = %v — fetchsafe must only ever send User-Agent (NFR-08)", name, captured.Header[name])
		}
	}
}

// TestFetchPreview_NonSuccessStatus_Rejected covers the regression that
// made Instagram previews silently fail: the destination answered 429
// with a bot-wall HTML page, which parsed cleanly, yielded no og: tags,
// and became preview_status='failed' — identical to a page with no
// metadata, so the real cause was invisible. A non-2xx response must be
// rejected explicitly instead of being fed to the parser.
func TestFetchPreview_NonSuccessStatus_Rejected(t *testing.T) {
	// Body a bot wall would realistically return: valid HTML that parses
	// fine, with no Open Graph tags — the case that previously masked the
	// error as an ordinary "no metadata" result.
	const botWall = `<html><head><title>Login</title></head><body>Log in to continue</body></html>`

	for _, status := range []int{
		http.StatusTooManyRequests,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusUnauthorized,
		http.StatusMovedPermanently,
	} {
		client := &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Header:     http.Header{"Content-Type": []string{"text/html"}},
					Body:       io.NopCloser(strings.NewReader(botWall)),
				}, nil
			}),
		}

		_, err := FetchPreview(context.Background(), client, "https://example.com/page")
		if !errors.Is(err, ErrHTTPStatus) {
			t.Errorf("status %d: err = %v, want ErrHTTPStatus", status, err)
		}
	}
}

// TestFetchPreview_SuccessStatuses_Parsed guards the other side of the
// check above: a 2xx must still be parsed normally, so the new rejection
// cannot swallow legitimate responses.
func TestFetchPreview_SuccessStatuses_Parsed(t *testing.T) {
	const page = `<html><head>` +
		`<meta property="og:title" content="Un títol">` +
		`<meta property="og:image" content="https://example.com/i.jpg">` +
		`<meta property="og:description" content="Una descripció">` +
		`</head></html>`

	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted} {
		client := &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Header:     http.Header{"Content-Type": []string{"text/html"}},
					Body:       io.NopCloser(strings.NewReader(page)),
				}, nil
			}),
		}

		preview, err := FetchPreview(context.Background(), client, "https://example.com/page")
		if err != nil {
			t.Fatalf("status %d: FetchPreview: %v", status, err)
		}
		if preview.Title != "Un títol" {
			t.Errorf("status %d: Title = %q, want %q", status, preview.Title, "Un títol")
		}
	}
}

// TestNewTransport_ForceAttemptHTTP2 pins the setting whose absence made
// every Instagram preview resolve to 'failed' (see client.go). HTTP/1.1
// requests are answered with a 429 bot wall by major sites; HTTP/2 gets
// the real page with its Open Graph tags.
func TestNewTransport_ForceAttemptHTTP2(t *testing.T) {
	if !newTransport().ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 = false, want true — HTTP/1.1 gets 429 bot walls from major sites, making every preview fail")
	}
}

// TestNewTransport_DisableKeepAlives pins the SSRF-load-bearing setting
// alongside it: without it, redirect hops toward the same host reuse the
// open connection and never re-trigger ControlContext, skipping IP
// validation for those hops. This must hold under HTTP/2 too, which
// multiplexes over a single connection.
func TestNewTransport_DisableKeepAlives(t *testing.T) {
	if !newTransport().DisableKeepAlives {
		t.Error("DisableKeepAlives = false, want true — redirect hops would skip ControlContext IP validation")
	}
}
