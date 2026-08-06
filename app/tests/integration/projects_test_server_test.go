// projects_test_server_test.go builds the projects-domain equivalent of
// ideas_test_server_test.go's newIdeasHTTPTestServer (NIU-11, tasks.md
// T-12): a full httpapi.NewRouter-backed server whose projects.Service
// performs controlledFetch instead of the real fetchsafe.FetchPreview —
// same local-mock rationale as ideas' flavour (fetchsafe's own IP
// allowlist rejects 127.0.0.1 by design, which is exactly what a local
// httptest.Server binds to).
package integration

import (
	"context"
	"io/fs"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	niu "niu"
	"niu/internal/auth"
	"niu/internal/httpapi"
	"niu/internal/ideas"
	"niu/internal/items"
	"niu/internal/projects"
	"niu/internal/store"
)

// controlledFetchForProjects adapts controlledFetch's ideas.Preview
// result into projects.Preview — same underlying HTTP behaviour
// (timeout, size cap, content-type gate, real fetchsafe OG parser), just
// a different exported shape per package (design.md §4: internal/projects
// does not import internal/ideas' Preview type into its own surface).
func controlledFetchForProjects(ctx context.Context, targetURL string) (projects.Preview, error) {
	preview, err := controlledFetch(ctx, targetURL)
	return projects.Preview{
		Title:       preview.Title,
		ImageURL:    preview.ImageURL,
		Description: preview.Description,
		Partial:     preview.Partial,
	}, err
}

// newProjectsHTTPTestServer wires a full httpapi.NewRouter-backed server
// whose projects.Service performs controlledFetchForProjects — used by
// T-12, which exercises the /api/v1/projects HTTP surface's link-preview
// behaviour (async resolution, description never exposed) against a
// local mock, not the SSRF mechanism itself (that is ideas_ssrf_test.go's
// job, exercised via the real fetchsafe.FetchPreview).
func newProjectsHTTPTestServer(t *testing.T, userID int64) *testServer {
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
	pool := ideas.NewWorkerPool(context.Background())
	t.Cleanup(pool.Close)
	projectsSvc := projects.NewService(projectsRepo, projectsRepo, controlledFetchForProjects, pool)

	ideasSvc := newIdeasService(t, st)

	authenticator := auth.StubAuthenticator{UserID: userID}
	var emptyFS fs.FS = fstest.MapFS{}
	router := httpapi.NewRouter(svc, projectsSvc, ideasSvc, st, authenticator, emptyFS, true)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testServer{Server: srv, Store: st}
}
