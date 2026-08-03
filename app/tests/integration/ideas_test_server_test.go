// ideas_test_server_test.go builds the two flavours of test server the
// ideas integration suite needs (tasks.md §6 notes — "one HTTP mock
// server built once, reused across T-22/T-23/T-24/T-26/T-27e", plus a
// separate flavour for the SSRF-specific regressions T-27a-d).
//
// newIdeasHTTPTestServer wires ideas.Service with an INJECTED fetch
// function that talks directly (via a plain http.Client, no fetchsafe
// involved) to a controllable httptest.Server — necessary because
// fetchsafe's own IP allowlist (design.md ADR-02) rejects 127.0.0.1 by
// design, which is exactly what any local httptest.Server binds to.
// These tests exercise the full /api/v1/ideas HTTP surface (create,
// list, delete, async resolution, caching) with a fetch step that
// behaves EXACTLY like fetchsafe.FetchPreview would (same Preview shape,
// same size/timeout/content-type semantics reused from fetchsafe's own
// exported helpers where possible) but is allowed to reach loopback,
// since these tests are not about the SSRF mechanism itself — that is
// T-27a-d's job, exercised against the REAL fetchsafe.FetchPreview via
// newSSRFTestServer below.
package integration

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	niu "niu"
	"niu/internal/auth"
	"niu/internal/fetchsafe"
	"niu/internal/httpapi"
	"niu/internal/ideas"
	"niu/internal/items"
	"niu/internal/projects"
	"niu/internal/store"
)

const (
	testFetchTimeout  = 200 * time.Millisecond
	testFetchMaxBytes = 2 << 20
)

// controlledFetch performs a plain HTTP GET (no fetchsafe/SSRF mitigation
// — deliberately, see file comment) against targetURL, applying the same
// observable behaviour fetchsafe.FetchPreview promises to its callers:
// a hard timeout, a streaming size cap, a Content-Type gate before
// parsing, and delegating to fetchsafe's own OG parser so the parsing
// logic under test is the real production code, not a re-implementation.
func controlledFetch(ctx context.Context, targetURL string) (ideas.Preview, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, testFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return ideas.Preview{}, err
	}

	client := &http.Client{Timeout: testFetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return ideas.Preview{}, fetchsafe.ErrTimeout
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && contentType != "text/html" && contentType != "text/html; charset=utf-8" {
		return ideas.Preview{}, fetchsafe.ErrUnsupportedContentType
	}

	limited := io.LimitReader(resp.Body, testFetchMaxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return ideas.Preview{}, err
	}
	// Detect the LimitReader having cut a response that intended to keep
	// going: a real fetchsafe.ErrResponseTooLarge equivalent for these
	// HTTP-behaviour-focused tests (EC-03).
	if int64(len(body)) > testFetchMaxBytes {
		return ideas.Preview{}, fetchsafe.ErrResponseTooLarge
	}

	preview, err := fetchsafe.ParseOpenGraphForTesting(bytes.NewReader(body))
	if err != nil {
		return ideas.Preview{}, err
	}
	return ideas.Preview{
		Title:       preview.Title,
		ImageURL:    preview.ImageURL,
		Description: preview.Description,
		Partial:     preview.Partial,
	}, nil
}

// newIdeasHTTPTestServer wires a full httpapi.NewRouter-backed server
// whose ideas.Service performs controlledFetch instead of the real
// fetchsafe.FetchPreview — used by T-22/T-23/T-24/T-26/T-27e, which
// exercise the /api/v1/ideas HTTP surface's behaviour (async resolution,
// caching, two-client convergence) against a local mock, not the SSRF
// mechanism itself.
func newIdeasHTTPTestServer(t *testing.T, userID int64) *testServer {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "niu.db")

	st, err := store.Open(dbPath, niu.MigrationsFS)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	repo := store.NewItemsRepository(st.DB)
	svc := items.NewService(repo, repo, repo)
	projectsRepo := store.NewProjectsRepository(st.DB)
	projectsSvc := projects.NewService(projectsRepo, projectsRepo)

	ideasRepo := store.NewIdeasRepository(st.DB)
	pool := ideas.NewWorkerPool(context.Background())
	t.Cleanup(pool.Close)
	ideasSvc := ideas.NewService(ideasRepo, ideasRepo, controlledFetch, pool)

	authenticator := auth.StubAuthenticator{UserID: userID}
	var emptyFS fs.FS = fstest.MapFS{}
	router := httpapi.NewRouter(svc, projectsSvc, ideasSvc, st, authenticator, emptyFS, true)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testServer{Server: srv, Store: st}
}

// newHTTPTestServer wraps a bare http.Handler in an httptest.Server with
// automatic cleanup — used by mockPreviewServer (ideas_test.go) and the
// SSRF-focused doubles (ideas_ssrf_test.go) that need full control over
// status code, headers, latency, body size, and redirects.
func newHTTPTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}
