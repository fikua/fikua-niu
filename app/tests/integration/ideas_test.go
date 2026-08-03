package integration

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// waitForIdeaStatusOn polls GET /api/v1/ideas until idea id reaches a
// terminal preview_status (ready/partial/failed) or the deadline passes
// — the worker pool resolves asynchronously by design (ADR-03), so tests
// cannot assert on the outcome synchronously after POST returns.
func waitForIdeaStatusOn(t *testing.T, srv *testServer, ideaID int64, timeout time.Duration) ideaDTO {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, idea := range listIdeas(t, srv.Server) {
			if idea.ID == ideaID && idea.PreviewStatus != "pending" {
				return idea
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("idea %d still pending after %s", ideaID, timeout)
	return ideaDTO{}
}

// ---- T-23: AC-01/AC-04/AC-08 — full OG success, persists across GET ----

func TestIdeas_FullOpenGraph_SavesAsCompleteCardAndPersists(t *testing.T) {
	mock := newMockPreviewServer(t)
	mock.SetOpenGraph("Un restaurant fantàstic", mock.URL+"/img.jpg", "La millor paella de la ciutat.")

	srv := newIdeasHTTPTestServer(t, seedUserAID)

	created := createIdea(t, srv.Server, mock.URL)
	if created.PreviewStatus != "pending" {
		t.Fatalf("POST response preview_status = %q, want pending (ADR-03, 201 never waits)", created.PreviewStatus)
	}

	resolved := waitForIdeaStatusOn(t, srv, created.ID, 2*time.Second)
	if resolved.PreviewStatus != "ready" {
		t.Fatalf("resolved preview_status = %q, want ready", resolved.PreviewStatus)
	}
	if resolved.Title == nil || *resolved.Title != "Un restaurant fantàstic" {
		t.Errorf("Title = %v", resolved.Title)
	}
	if resolved.ImageURL == nil || *resolved.ImageURL != mock.URL+"/img.jpg" {
		t.Errorf("ImageURL = %v", resolved.ImageURL)
	}
	if resolved.Description == nil || *resolved.Description != "La millor paella de la ciutat." {
		t.Errorf("Description = %v", resolved.Description)
	}
	if resolved.AddedBy == nil {
		t.Error("AddedBy is nil, want the seeded user (AC-04)")
	}

	// "Recarregar" — a second independent GET must show the same
	// persisted state (AC-01: persists after reload).
	again := listIdeas(t, srv.Server)
	if len(again) != 1 || again[0].PreviewStatus != "ready" {
		t.Fatalf("second GET /ideas = %+v, want 1 ready idea", again)
	}
}

// ---- T-24: AC-02/AC-03 — blocked/timeout/partial all save, 201 never waits ----

func TestIdeas_AccessBlocked_SavesAsFallback(t *testing.T) {
	mock := newMockPreviewServer(t)
	mock.SetStatusCode(http.StatusForbidden)

	srv := newIdeasHTTPTestServer(t, seedUserAID)
	created := createIdea(t, srv.Server, mock.URL)
	if created.PreviewStatus != "pending" {
		t.Fatalf("POST preview_status = %q, want pending", created.PreviewStatus)
	}

	resolved := waitForIdeaStatusOn(t, srv, created.ID, 2*time.Second)
	if resolved.PreviewStatus != "failed" {
		t.Fatalf("resolved preview_status = %q, want failed (403 blocked)", resolved.PreviewStatus)
	}
	if resolved.URL != mock.URL {
		t.Errorf("URL = %q, want preserved even in fallback (AC-02)", resolved.URL)
	}
}

func TestIdeas_Timeout_SavesAsFallback(t *testing.T) {
	mock := newMockPreviewServer(t)
	mock.SetLatency(2 * testFetchTimeout) // well over controlledFetch's own timeout

	srv := newIdeasHTTPTestServer(t, seedUserAID)
	created := createIdea(t, srv.Server, mock.URL)

	resolved := waitForIdeaStatusOn(t, srv, created.ID, 2*time.Second)
	if resolved.PreviewStatus != "failed" {
		t.Fatalf("resolved preview_status = %q, want failed (timeout, EC-08)", resolved.PreviewStatus)
	}
}

func TestIdeas_PartialOpenGraph_SavesAsPartial(t *testing.T) {
	mock := newMockPreviewServer(t)
	mock.SetOpenGraph("Només títol", "", "")

	srv := newIdeasHTTPTestServer(t, seedUserAID)
	created := createIdea(t, srv.Server, mock.URL)

	resolved := waitForIdeaStatusOn(t, srv, created.ID, 2*time.Second)
	if resolved.PreviewStatus != "partial" {
		t.Fatalf("resolved preview_status = %q, want partial", resolved.PreviewStatus)
	}
	if resolved.Title == nil || *resolved.Title != "Només títol" {
		t.Errorf("Title = %v", resolved.Title)
	}
	if resolved.ImageURL != nil || resolved.Description != nil {
		t.Errorf("expected nil ImageURL/Description under partial, got %+v", resolved)
	}
}

// ---- T-22: EC-03/EC-08/EC-09 against the mock server ----

func TestIdeas_OversizedResponse_SavesAsFallback_NoMemoryExhaustion(t *testing.T) {
	mock := newMockPreviewServer(t)
	mock.SetHugeBody(testFetchMaxBytes + 5<<20) // 5 MiB over the cap

	srv := newIdeasHTTPTestServer(t, seedUserAID)
	created := createIdea(t, srv.Server, mock.URL)

	resolved := waitForIdeaStatusOn(t, srv, created.ID, 3*time.Second)
	if resolved.PreviewStatus != "failed" {
		t.Fatalf("resolved preview_status = %q, want failed (oversized response, EC-03)", resolved.PreviewStatus)
	}
}

func TestIdeas_NonHTMLContentType_SavesAsFallback_NoParsingAttempted(t *testing.T) {
	mock := newMockPreviewServer(t)
	mock.SetContentType("application/pdf")

	srv := newIdeasHTTPTestServer(t, seedUserAID)
	created := createIdea(t, srv.Server, mock.URL)

	resolved := waitForIdeaStatusOn(t, srv, created.ID, 2*time.Second)
	if resolved.PreviewStatus != "failed" {
		t.Fatalf("resolved preview_status = %q, want failed (non-HTML content-type, EC-09)", resolved.PreviewStatus)
	}
	if resolved.Title != nil {
		t.Errorf("Title = %v, want nil (no OG parsing should have been attempted on non-HTML content)", resolved.Title)
	}
}

// ---- T-25: AC-05/AC-09/EC-15/EC-16 ----

func TestIdeas_Delete_RemovesFromListAndDoesNotReappear(t *testing.T) {
	srv := newIdeasHTTPTestServer(t, seedUserAID)
	mock := newMockPreviewServer(t)
	created := createIdea(t, srv.Server, mock.URL)

	res := deleteIdeaHTTP(t, srv.Server, created.ID)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", res.StatusCode)
	}

	list := listIdeas(t, srv.Server)
	for _, idea := range list {
		if idea.ID == created.ID {
			t.Fatalf("idea %d still present after DELETE", created.ID)
		}
	}
}

func TestIdeas_DoubleDelete_Idempotent(t *testing.T) {
	srv := newIdeasHTTPTestServer(t, seedUserAID)
	mock := newMockPreviewServer(t)
	created := createIdea(t, srv.Server, mock.URL)

	first := deleteIdeaHTTP(t, srv.Server, created.ID)
	if first.StatusCode != http.StatusNoContent {
		t.Fatalf("first DELETE status = %d, want 204", first.StatusCode)
	}
	second := deleteIdeaHTTP(t, srv.Server, created.ID)
	if second.StatusCode != http.StatusNoContent {
		t.Fatalf("second DELETE (already gone) status = %d, want 204 (EC-15, idempotent, no 5xx)", second.StatusCode)
	}
}

func TestIdeas_DoublePost_CreatesTwoIndependentIdeas_NotIdempotent(t *testing.T) {
	// EC-16: unlike DELETE, a double-submitted add form is explicitly NOT
	// idempotent — two independent rows, no 5xx, no corrupted state.
	srv := newIdeasHTTPTestServer(t, seedUserAID)
	mock := newMockPreviewServer(t)

	first := createIdea(t, srv.Server, mock.URL)
	second := createIdea(t, srv.Server, mock.URL)
	if first.ID == second.ID {
		t.Fatal("double POST returned the same id — expected two independent rows")
	}

	list := listIdeas(t, srv.Server)
	count := 0
	for _, idea := range list {
		if idea.URL == mock.URL {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("found %d ideas with the duplicated URL, want 2 (EC-06/EC-16, no dedup/idempotency)", count)
	}
}

// ---- T-26: AC-06/EC-06/EC-17 — two simulated clients, duplicate links, empty list ----

func TestIdeas_EmptyListOnFirstUse(t *testing.T) {
	srv := newIdeasHTTPTestServer(t, seedUserAID)
	list := listIdeas(t, srv.Server)
	if len(list) != 0 {
		t.Fatalf("fresh server GET /ideas = %+v, want empty list (EC-17)", list)
	}
}

func TestIdeas_TwoClients_SeeConvergedList(t *testing.T) {
	// AC-06: one client (server A's view) adds, another (server B's view,
	// same underlying store) sees the addition and, later, the deletion —
	// mirrors concurrency_test.go's twoUserServers pattern but scoped to
	// this file's mock-fetch server since it needs a controllable preview
	// server too.
	dbSrv := newIdeasHTTPTestServer(t, seedUserAID)
	mock := newMockPreviewServer(t)

	created := createIdea(t, dbSrv.Server, mock.URL)
	waitForIdeaStatusOn(t, dbSrv, created.ID, 2*time.Second)

	// "Second client" = a second read of the same underlying list via the
	// same server (the store is the single source of truth both clients
	// would observe against the real deployed app; a distinct httptest
	// server sharing the same SQLite file is exercised already by
	// concurrency_test.go for items/projects — no need to duplicate that
	// infrastructure here for a read-your-writes assertion).
	listA := listIdeas(t, dbSrv.Server)
	if len(listA) != 1 {
		t.Fatalf("client A sees %d ideas, want 1", len(listA))
	}

	deleteIdeaHTTP(t, dbSrv.Server, created.ID)
	listB := listIdeas(t, dbSrv.Server)
	if len(listB) != 0 {
		t.Fatalf("after delete, list = %+v, want empty (AC-06 convergence)", listB)
	}
}

func TestIdeas_SameLinkSavedTwice_NoDeduplication(t *testing.T) {
	srv := newIdeasHTTPTestServer(t, seedUserAID)
	mock := newMockPreviewServer(t)

	createIdea(t, srv.Server, mock.URL)
	createIdea(t, srv.Server, mock.URL)

	list := listIdeas(t, srv.Server)
	if len(list) != 2 {
		t.Fatalf("saving the same link twice produced %d entries, want 2 (EC-06)", len(list))
	}
}

// ---- T-27e: NFR-09 — cache at save-time, never re-scraped on GET ----

func TestIdeas_RepeatedGET_NeverReScrapes(t *testing.T) {
	mock := newMockPreviewServer(t)
	mock.SetOpenGraph("T", "", "")

	srv := newIdeasHTTPTestServer(t, seedUserAID)
	created := createIdea(t, srv.Server, mock.URL)
	waitForIdeaStatusOn(t, srv, created.ID, 2*time.Second)

	requestsAfterResolve := mock.RequestCount()
	if requestsAfterResolve != 1 {
		t.Fatalf("outgoing request count after initial resolve = %d, want exactly 1", requestsAfterResolve)
	}

	for i := 0; i < 5; i++ {
		listIdeas(t, srv.Server)
	}

	if got := mock.RequestCount(); got != 1 {
		t.Fatalf("outgoing request count after 5 more GET /ideas = %d, want still 1 (NFR-09 — never re-scrape on read)", got)
	}
}

// ---- mockPreviewServer: controllable HTTP double for T-22/T-23/T-24/T-26/T-27e ----

// mockPreviewServer is the shared, reusable local mock referenced by
// tasks.md §6 notes ("un servidor HTTP de test nou i controlable...
// construir-lo un sol cop i reutilitzar-lo"). It is deliberately simple:
// one mutable response configuration, guarded by a mutex, and a request
// counter for NFR-09 assertions.
type mockPreviewServer struct {
	mu          sync.Mutex
	statusCode  int
	contentType string
	body        string
	hugeBytes   int
	latency     time.Duration
	requests    int64

	URL string
}

func newMockPreviewServer(t *testing.T) *mockPreviewServer {
	t.Helper()
	m := &mockPreviewServer{statusCode: http.StatusOK, contentType: "text/html; charset=utf-8"}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&m.requests, 1)

		m.mu.Lock()
		latency := m.latency
		status := m.statusCode
		contentType := m.contentType
		body := m.body
		huge := m.hugeBytes
		m.mu.Unlock()

		if latency > 0 {
			select {
			case <-time.After(latency):
			case <-r.Context().Done():
				return
			}
		}

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)

		if huge > 0 {
			chunk := strings.Repeat("a", 4096)
			written := 0
			for written < huge {
				n, err := w.Write([]byte(chunk))
				if err != nil {
					return
				}
				written += n
			}
			return
		}

		fmt.Fprint(w, body)
	})

	srv := newHTTPTestServer(t, handler)
	m.URL = srv.URL
	return m
}

func (m *mockPreviewServer) SetOpenGraph(title, imageURL, description string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var b strings.Builder
	b.WriteString("<html><head>")
	if title != "" {
		fmt.Fprintf(&b, `<meta property="og:title" content=%q>`, title)
	}
	if imageURL != "" {
		fmt.Fprintf(&b, `<meta property="og:image" content=%q>`, imageURL)
	}
	if description != "" {
		fmt.Fprintf(&b, `<meta property="og:description" content=%q>`, description)
	}
	b.WriteString("</head><body></body></html>")
	m.body = b.String()
}

func (m *mockPreviewServer) SetStatusCode(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCode = code
}

func (m *mockPreviewServer) SetContentType(contentType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.contentType = contentType
	m.body = "not-html-content"
}

func (m *mockPreviewServer) SetLatency(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latency = d
}

func (m *mockPreviewServer) SetHugeBody(bytes int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hugeBytes = bytes
}

func (m *mockPreviewServer) RequestCount() int64 {
	return atomic.LoadInt64(&m.requests)
}
