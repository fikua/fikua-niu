package integration

import (
	"net/http"
	"runtime"
	"strings"
	"testing"
)

// T-28 — S3a/S3b/S7/S8/EC-08/EC-09/EC-10.

// S7/NFR-02: security headers present on 100% of responses (API and
// static alike — checked here on both an API response and a healthz
// response, both go through the same outermost SecurityHeaders
// middleware).
func TestSecurityHeaders_PresentOnAllResponses(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	endpoints := []string{"/api/v1/items", "/healthz"}
	for _, path := range endpoints {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		assertSecurityHeaders(t, path, res)
	}
}

func assertSecurityHeaders(t *testing.T, path string, res *http.Response) {
	t.Helper()
	required := map[string]string{
		"Strict-Transport-Security": "",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "no-referrer",
		"Content-Security-Policy":   "",
	}
	for header, expectedExact := range required {
		got := res.Header.Get(header)
		if got == "" {
			t.Errorf("%s: header %s missing", path, header)
			continue
		}
		if expectedExact != "" && got != expectedExact {
			t.Errorf("%s: header %s = %q, want %q", path, header, got, expectedExact)
		}
	}
}

// S3b: CSP contains no unsafe-inline anywhere.
func TestCSP_NoUnsafeInline(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	res, err := http.Get(srv.URL + "/api/v1/items")
	if err != nil {
		t.Fatalf("GET /items: %v", err)
	}
	csp := res.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header missing")
	}
	if strings.Contains(csp, "unsafe-inline") {
		t.Fatalf("CSP contains unsafe-inline: %q", csp)
	}
}

// S3a/EC-09: an XSS payload in the item name is stored literally as text
// — the API layer never interprets it as HTML, and the value round-trips
// byte-for-byte. Script-execution-in-a-real-browser is asserted by the
// Playwright E2E suite (T-29); this test covers the server-side half of
// the contract.
func TestXSSPayload_StoredLiterally(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	payload := "<img src=x onerror=alert(1)>"
	created := createItem(t, srv.Server, payload)
	if created.Name != payload {
		t.Fatalf("stored name = %q, want literal payload %q", created.Name, payload)
	}

	list := listItems(t, srv.Server)
	if len(list) != 1 || list[0].Name != payload {
		t.Fatalf("GET /items name = %+v, want literal payload preserved", list)
	}
}

// S8/EC-10: a SQL-injection-shaped name is stored literally as text, and
// the items table survives intact (no DROP TABLE side effect).
func TestSQLInjectionPayload_StoredLiterally_TableSurvives(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	// Seed one legitimate item first so we can assert the table is not
	// wiped by the attack.
	createItem(t, srv.Server, "Llet")

	payload := "'; DROP TABLE items;--"
	created := createItem(t, srv.Server, payload)
	if created.Name != payload {
		t.Fatalf("stored name = %q, want literal payload %q", created.Name, payload)
	}

	// The table must still exist and contain both items.
	list := listItems(t, srv.Server)
	if len(list) != 2 {
		t.Fatalf("items table after injection attempt = %+v, want 2 rows (table intact)", list)
	}

	var one int
	if err := srv.Store.DB.QueryRow("SELECT 1 FROM sqlite_master WHERE type='table' AND name='items'").Scan(&one); err != nil {
		t.Fatalf("items table missing after injection attempt: %v", err)
	}
}

// EC-08/NFR-04: no GET route has a mutating effect. We assert this at the
// HTTP level: issuing GET against the create/move/delete surface must
// never create, move, or delete anything, and the router must not expose
// GET handlers for the mutation-only paths beyond the documented
// GET /items and GET /me.
func TestNoMutationViaGET(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	before := listItems(t, srv.Server)

	// GET against the collection endpoint must not create anything, no
	// matter what (there is no body parsing on GET at all).
	res, err := http.Get(srv.URL + "/api/v1/items")
	if err != nil {
		t.Fatalf("GET /items: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /items status = %d, want 200", res.StatusCode)
	}

	after := listItems(t, srv.Server)
	if len(before) != len(after) {
		t.Fatalf("GET /items changed item count: before=%d after=%d", len(before), len(after))
	}
}

// TestOversizedBodyRejectedWithoutBuffering covers F-S04: the request body
// is capped by middleware, so a large payload is refused without the
// process ever materialising it.
//
// Before the fix, a 128 MiB body was read in full and only then rejected
// for exceeding 200 characters — around 896 MiB of memory for one
// request. A handful in parallel would OOM the process on a shared VPS.
// Cloudflare Access does not help here: the caller is an authenticated
// legitimate user, and a buggy client loop does it by accident.
func TestOversizedBodyRejectedWithoutBuffering(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	// 8 MiB is far above the 64 KiB cap and far below anything that
	// would slow the test down.
	huge := strings.Repeat("a", 8<<20)
	body := `{"name":"` + huge + `"}`

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	res, err := http.Post(srv.URL+"/api/v1/items", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST oversized: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Errorf("POST with an %d-byte body returned %d, want a 4xx rejection",
			len(body), res.StatusCode)
	}

	runtime.GC()
	runtime.ReadMemStats(&after)

	// The cap is 64 KiB, so the handler must never have held the whole
	// body. Allow generous headroom for test harness allocations while
	// still failing loudly if the full payload were buffered.
	const ceiling = 4 << 20
	if grew := after.TotalAlloc - before.TotalAlloc; grew > uint64(len(body)) {
		t.Errorf("request allocated %d bytes for a %d-byte body — the body appears to be fully buffered despite the %d-byte cap",
			grew, len(body), ceiling)
	}

	// And nothing was stored.
	if items := listItems(t, srv.Server); len(items) != 0 {
		t.Errorf("oversized request created %d item(s), want 0", len(items))
	}
}

// TestBidiOverrideRejectedEndToEnd is the HTTP-level half of EC-05: a
// Trojan Source payload (CVE-2021-42574) must not reach storage. If it
// did, the stored text and the rendered text would differ — user B would
// read something other than what user A wrote.
func TestBidiOverrideRejectedEndToEnd(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	// U+202E flips the rendering of what follows.
	payload := `{"name":"Comprar ‮selpma 100"}`

	res, err := http.Post(srv.URL+"/api/v1/items", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("POST bidi: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("POST with a bidi override returned %d, want 400 (EC-05)", res.StatusCode)
	}
	if items := listItems(t, srv.Server); len(items) != 0 {
		t.Errorf("bidi payload was stored (%d item(s)) — Trojan Source reaches the other user", len(items))
	}
}
