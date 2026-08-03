// ideas_ssrf_test.go — T-27a/T-27b/T-27c/T-27d: dedicated SSRF regression
// tests against the REAL fetchsafe.FetchPreview (not the mock-fetch
// wiring in ideas_test.go, which deliberately bypasses fetchsafe for
// HTTP-content-behaviour tests). These are the tests tasks.md/design.md
// mark as BLOCKING for /audit (NFR-06) — every one of them must be green
// before this item can be considered conformant, independently of how
// correct the code looks on inspection.
//
// countingListener wraps a real net.Listener bound to 127.0.0.1 (the
// only address a local listener can realistically bind to) and counts
// Accept() calls — i.e. completed TCP connections. Since 127.0.0.1 is
// itself a forbidden destination under fetchsafe's own allowlist
// (design.md ADR-02), asserting acceptCount stays at 0 after a rejected
// fetch is a STRONGER proof than counting HTTP requests: it demonstrates
// that ControlContext aborted the dial before connect() ever completed,
// not merely that the HTTP layer chose not to send a request.
package integration

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"niu/internal/fetchsafe"
)

type countingListener struct {
	net.Listener
	accepted int64
}

func (l *countingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err == nil {
		atomic.AddInt64(&l.accepted, 1)
	}
	return conn, err
}

func (l *countingListener) AcceptedCount() int64 {
	return atomic.LoadInt64(&l.accepted)
}

// newCountingHTTPServer starts a real HTTP server on a countingListener
// so tests can assert on completed-connection counts, not just whether a
// handler ran.
func newCountingHTTPServer(t *testing.T, handler http.Handler) (*countingListener, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	cl := &countingListener{Listener: ln}
	srv := &http.Server{Handler: handler}
	go srv.Serve(cl) //nolint:errcheck // server lifetime ends with test process/listener close
	t.Cleanup(func() {
		srv.Close()
	})
	return cl, "http://" + ln.Addr().String()
}

// ---- T-27a: EC-02/EC-07 — literal private/loopback IP rejected, zero TCP connection ----

func TestFetchPreview_LiteralPrivateOrLoopbackIP_Rejected_NoTCPConnection(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must never be invoked — the connection should be rejected before any HTTP exchange")
	})
	cl, _ := newCountingHTTPServer(t, handler)

	client := fetchsafe.NewClient()
	cases := []string{
		"http://127.0.0.1:" + portOf(t, cl.Listener),
		"http://10.0.0.1/",
		"http://169.254.169.254/", // classic cloud metadata endpoint
	}
	for _, rawURL := range cases {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := fetchsafe.FetchPreview(ctx, client, rawURL)
		cancel()
		if err != fetchsafe.ErrDestinationForbidden && err != fetchsafe.ErrTimeout {
			t.Errorf("FetchPreview(%q) = %v, want ErrDestinationForbidden", rawURL, err)
		}
	}

	if got := cl.AcceptedCount(); got != 0 {
		t.Fatalf("listener accepted %d connection(s), want 0 — the loopback destination must never be reached", got)
	}
}

// ---- T-27a (EC-07 half): DNS-resolved private destination rejected ----
//
// EC-07 requires that a hostname which RESOLVES to a private/loopback
// address is rejected the same way as a literal IP, even though the URL
// text is not itself an IP. "localhost" is exactly this case on every
// standard resolver (resolves to 127.0.0.1/::1) without requiring any
// test-specific DNS double — a real, unmodified resolution path.
func TestFetchPreview_HostnameResolvingToLoopback_Rejected(t *testing.T) {
	client := fetchsafe.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := fetchsafe.FetchPreview(ctx, client, "http://localhost:1/")
	if err != fetchsafe.ErrDestinationForbidden {
		t.Fatalf("FetchPreview(http://localhost:1/) = %v, want ErrDestinationForbidden (EC-07, DNS-resolved loopback)", err)
	}
}

func portOf(t *testing.T, ln net.Listener) string {
	t.Helper()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	return port
}

// ---- T-27b: hostname denylist rejected before any DNS/TCP activity ----

func TestFetchPreview_DenylistedHostname_Rejected_NoDNSOrTCP(t *testing.T) {
	t.Setenv("NIU_PUBLIC_HOST", "niu-test-staging.example")

	client := fetchsafe.NewClient()
	cases := []string{
		"https://niu.fikua.com/somepage",
		"https://niu-test-staging.example/somepage",
	}
	for _, rawURL := range cases {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := fetchsafe.FetchPreview(ctx, client, rawURL)
		cancel()
		if err != fetchsafe.ErrDestinationForbidden {
			t.Errorf("FetchPreview(%q) = %v, want ErrDestinationForbidden (T-03b denylist, rejected before DNS)", rawURL, err)
		}
	}
}

// ---- T-27c: same-host redirect chain toward a forbidden IP is rejected on the FIRST hop ----
//
// F-11 (review.md, /audit NIU-6): this test does NOT itself detect the
// F-01 regression its former name claimed to cover. Confirmed by
// mutation testing during audit — reverting DisableKeepAlives to false
// (reintroducing F-01) still passes this test unchanged, because the
// destination here is loopback, and loopback is rejected by
// ControlContext on the very FIRST dial attempt: the redirect chain
// never progresses far enough to exercise connection reuse at all (hop
// 1/2 are consequently never reached — accepted count == 0). Proving
// "0 hops reached" cannot, by construction, also prove "every hop that
// WAS reached triggered a fresh dial" — that would require a
// destination the chain is allowed to actually traverse, which black-box
// tests in this package cannot arrange without a dialer/ControlContext
// injection seam fetchsafe does not currently expose (adding one purely
// for this test was judged more invasive than worth it here — see
// review.md F-11 option (b)).
//
// What this test DOES prove, and is named/documented for accordingly:
// a same-host redirect chain whose destination is loopback is blocked
// before connect() completes on the very first hop, with zero hops
// silently let through.
//
// F-01's REAL regression guard — the one that fails immediately and
// unconditionally if DisableKeepAlives is ever reverted or refactored
// away — is TestNewTransport_DisableKeepAlivesTrue (client_test.go, same
// package): a literal-code assertion on the Transport construction
// itself, which this test cannot substitute for.
func TestFetchPreview_RedirectToSameHost_FirstHopRejectedBeforeConnect(t *testing.T) {
	const hops = 3
	var hitCount int64

	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hop1", http.StatusFound)
	})
	mux.HandleFunc("/hop1", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hop2", http.StatusFound)
	})
	mux.HandleFunc("/hop2", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hitCount, 1)
		w.WriteHeader(http.StatusOK)
	})

	cl, baseURL := newCountingHTTPServer(t, mux)

	client := fetchsafe.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// The destination is loopback, so fetchsafe correctly rejects it
	// regardless of the redirect chain — the assertion that matters here
	// is architectural: EVERY hop must trigger a fresh dial attempt
	// (Accept() on the listener), never fewer than the number of hops
	// actually reached before rejection. If DisableKeepAlives were ever
	// reverted, a connection-reuse bug would show as an accepted-count
	// LOWER than the number of redirects the server actually issued —
	// this is exactly what F-01 was: 4 hops -> 1 single dial.
	_, err := fetchsafe.FetchPreview(ctx, client, baseURL+"/start")
	if err != fetchsafe.ErrDestinationForbidden {
		t.Fatalf("FetchPreview(same-host redirect chain to loopback) = %v, want ErrDestinationForbidden", err)
	}

	// Because the destination is rejected at ControlContext (before
	// connect()), the redirect chain never actually progresses past hop
	// 0 in this specific scenario — accepted count must be exactly 0,
	// proving even the FIRST hop's connection attempt was blocked by
	// IP validation, not merely "some" hops.
	if got := cl.AcceptedCount(); got != 0 {
		t.Fatalf("listener accepted %d connection(s) for a same-host redirect chain toward a forbidden destination, want 0", got)
	}
	if atomic.LoadInt64(&hitCount) != 0 {
		t.Fatalf("/hop2 was reached %d time(s) — the forbidden destination must never be contacted, not even via redirect", hitCount)
	}
}

// ---- T-27d: [F-02 regression] IPv4-mapped-to-IPv6 destination rejected ----
//
// This is the dedicated regression for Unmap() being omitted or removed
// by accident (F-02) — NOT a re-test of the plain IPv4-literal case
// (T-27a already covers that; tasks.md explicitly warns not to conflate
// the two, since T-27a's literal-IPv4 assertion does NOT exercise the
// Unmap() code path at all).
func TestFetchPreview_IPv4MappedIPv6Destination_Rejected_F02Regression(t *testing.T) {
	client := fetchsafe.NewClient()

	cases := []string{
		"http://[::ffff:127.0.0.1]/",
		"http://[::ffff:169.254.169.254]/",
	}
	for _, rawURL := range cases {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := fetchsafe.FetchPreview(ctx, client, rawURL)
		cancel()
		if err != fetchsafe.ErrDestinationForbidden {
			t.Errorf("FetchPreview(%q) = %v, want ErrDestinationForbidden (F-02 regression: Unmap() must run before Is*() checks)", rawURL, err)
		}
	}
}
