package fetchsafe

import (
	"net/http"
	"net/url"
	"testing"
)

// TestNewTransport_DisableKeepAlivesTrue is a direct, literal-code
// assertion for T-03d/F-01/F-06: without DisableKeepAlives: true, a
// redirect chain toward the same host reuses the already-open TCP
// connection and never calls DialContext/ControlContext again for later
// hops, silently bypassing IP validation for that case (verified
// empirically by security-engineer: 4 hops -> 1 single Control
// invocation). This test fails loudly and immediately if that flag is
// ever reverted or refactored away, independently of any integration
// test that exercises the resulting behaviour.
func TestNewTransport_DisableKeepAlivesTrue(t *testing.T) {
	transport := newTransport()
	if !transport.DisableKeepAlives {
		t.Fatal("fetchsafe's Transport.DisableKeepAlives = false, want true (F-01/F-06 regression: " +
			"a redirect chain to the same host would reuse a connection and skip ControlContext on later hops)")
	}
}

// TestCheckRedirect_HopLimit verifies T-03e(a): the chain is capped at
// maxRedirects hops.
func TestCheckRedirect_HopLimit(t *testing.T) {
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}}

	var via []*http.Request
	for i := 0; i < maxRedirects; i++ {
		if err := checkRedirect(req, via); err != nil {
			t.Fatalf("checkRedirect at hop %d (within limit) = %v, want nil", i, err)
		}
		via = append(via, req)
	}
	if err := checkRedirect(req, via); err != ErrTooManyRedirects {
		t.Fatalf("checkRedirect at hop %d (over limit) = %v, want ErrTooManyRedirects", maxRedirects, err)
	}
}

// TestCheckRedirect_RejectsNonHTTPSchemeOnRedirect verifies T-03e(b): a
// redirect toward a non-http(s) scheme is rejected as a defense-in-depth
// layer independent from T-03c's IP validation.
func TestCheckRedirect_RejectsNonHTTPSchemeOnRedirect(t *testing.T) {
	cases := []string{"file:///etc/passwd", "javascript:alert(1)", "ftp://example.com/file"}
	for _, raw := range cases {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("url.Parse(%q): %v", raw, err)
		}
		req := &http.Request{URL: u}
		if err := checkRedirect(req, nil); err == nil {
			t.Errorf("checkRedirect(%q) = nil, want a rejection (non-http(s) redirect target)", raw)
		}
	}
}

// TestCheckRedirect_AllowsHTTPAndHTTPS verifies the redirect scheme gate
// does not reject legitimate http(s) redirect targets.
func TestCheckRedirect_AllowsHTTPAndHTTPS(t *testing.T) {
	for _, raw := range []string{"http://example.com/next", "https://example.com/next"} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("url.Parse(%q): %v", raw, err)
		}
		req := &http.Request{URL: u}
		if err := checkRedirect(req, nil); err != nil {
			t.Errorf("checkRedirect(%q) = %v, want nil", raw, err)
		}
	}
}

// TestNewClient_NoAuthHeadersConfigured verifies T-03h/NFR-08 at
// construction time: the client's own configuration carries no
// Authorization/Cookie machinery — only the identifiable User-Agent is
// attached, and only inside FetchPreview's request construction (verified
// end-to-end by tests/integration/security_test.go's header-inspection
// test), never as a default header on the client/transport themselves.
func TestNewClient_NoAuthHeadersConfigured(t *testing.T) {
	client := NewClient()
	if client.Jar != nil {
		t.Error("fetchsafe's client has a non-nil cookie Jar — it must never carry cookies (NFR-08)")
	}
	if client.Timeout != fetchTimeout {
		t.Errorf("client.Timeout = %v, want %v (second-level safety net matching the context timeout)", client.Timeout, fetchTimeout)
	}
}
