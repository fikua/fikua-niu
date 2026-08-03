package fetchsafe

import (
	"context"
	"net/http"
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
