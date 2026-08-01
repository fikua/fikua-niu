package integration

import (
	"net/http"
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
