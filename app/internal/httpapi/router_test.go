package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5"

	"niu/internal/auth"
	"niu/internal/ideas"
	"niu/internal/items"
	"niu/internal/projects"
)

// noopIdeasFetch never actually calls fetchsafe — used by router-level
// tests that only assert on routing/GET-mutation behaviour, never on
// scrape outcomes.
func noopIdeasFetch(_ context.Context, _ string) (ideas.Preview, error) {
	return ideas.Preview{}, nil
}

// noopProjectsFetch mirrors noopIdeasFetch for the projects domain
// (NIU-11) — same rationale, router-level tests never exercise scrape
// outcomes.
func noopProjectsFetch(_ context.Context, _ string) (projects.Preview, error) {
	return projects.Preview{}, nil
}

// TestNoMutatingGETRoutes introspects the chi route table (EC-08/NFR-04)
// and asserts that no GET route also has a POST/PATCH/PUT/DELETE handler
// registered on the exact same pattern — i.e. GET is never wired to a
// handler shared with a mutating method.
func TestNoMutatingGETRoutes(t *testing.T) {
	router := NewRouter(nil, nil, nil, fakeHealthChecker{}, fakeAuthenticator{}, fstest.MapFS{}, true)

	chiRouter, ok := router.(chi.Router)
	if !ok {
		t.Fatalf("router does not implement chi.Router (got %T)", router)
	}

	// Collect the actual routing table, so the assertions below are about
	// what is really registered rather than about the shape of this test.
	getRoutes := map[string]bool{}
	allRoutes := map[string][]string{}
	err := chi.Walk(chiRouter, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == http.MethodGet {
			getRoutes[route] = true
		}
		allRoutes[route] = append(allRoutes[route], method)
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	// The GET surface is a closed set. A new GET route added without a
	// deliberate update here fails the test — which is the point: EC-08
	// must break loudly if someone wires a mutation behind a GET.
	wantGET := map[string]bool{
		"/healthz":          true,
		"/api/v1/me":        true,
		"/api/v1/items/":    true,
		"/api/v1/projects/": true,
		"/api/v1/ideas/":    true,
	}
	for route := range getRoutes {
		if !wantGET[route] {
			t.Errorf("unexpected GET route %q registered — every GET must be read-only (EC-08/NFR-04); "+
				"if this route is genuinely read-only, add it to wantGET deliberately", route)
		}
	}
	for route := range wantGET {
		if !getRoutes[route] {
			t.Errorf("expected GET route %q to be registered, got: %+v", route, getRoutes)
		}
	}
}

// TestGETRequestsDoNotMutateState is the behavioural half of EC-08/NFR-04.
// The routing-table check above proves which GET routes exist; this proves
// that exercising every one of them leaves the data untouched. Both are
// needed: a route can be registered as GET and still mutate inside its
// handler, and Go offers no way to inspect a closure body at runtime.
func TestGETRequestsDoNotMutateState(t *testing.T) {
	repo := &spyRepo{}
	svc := items.NewService(repo, &spySink{}, &spyUsers{})
	previewPool := ideas.NewWorkerPool(context.Background())
	t.Cleanup(previewPool.Close)
	projectsRepo := &spyProjectsRepo{}
	projectsSvc := projects.NewService(projectsRepo, &spySink{}, noopProjectsFetch, previewPool)
	ideasRepo := &spyIdeasRepo{}
	ideasSvc := ideas.NewService(ideasRepo, &spySink{}, noopIdeasFetch, previewPool)
	router := NewRouter(svc, projectsSvc, ideasSvc, fakeHealthChecker{}, fakeAuthenticator{}, fstest.MapFS{}, true)

	chiRouter, ok := router.(chi.Router)
	if !ok {
		t.Fatalf("router does not implement chi.Router (got %T)", router)
	}

	var getPaths []string
	if err := chi.Walk(chiRouter, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method == http.MethodGet {
			// chi reports the pattern; turn "/api/v1/items/" into a
			// concrete request path.
			getPaths = append(getPaths, strings.TrimSuffix(route, "/"))
		}
		return nil
	}); err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	if len(getPaths) == 0 {
		t.Fatal("no GET routes discovered — the walk is not seeing the routing table")
	}

	for _, path := range getPaths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	if len(repo.mutations) != 0 {
		t.Errorf("GET requests triggered %d mutating repository call(s) (%v); "+
			"no GET may create, move or delete (EC-08/NFR-04)",
			len(repo.mutations), repo.mutations)
	}
	if len(projectsRepo.mutations) != 0 {
		t.Errorf("GET requests triggered %d mutating projects repository call(s) (%v); "+
			"no GET may create, change state or delete (EC-10/NFR-04)",
			len(projectsRepo.mutations), projectsRepo.mutations)
	}
	if len(ideasRepo.mutations) != 0 {
		t.Errorf("GET requests triggered %d mutating ideas repository call(s) (%v); "+
			"no GET may create or delete (EC-13/NFR-03)",
			len(ideasRepo.mutations), ideasRepo.mutations)
	}
}

// TestSPAFallback exercises the client-side-route allowlist added
// alongside the SPA merge (see spa-conversion-adhoc-review.md F-01): only
// paths in spaRoutes fall back to index.html; everything else, including a
// plausible-looking but nonexistent asset path, must still 404 cleanly
// rather than silently returning 200 + the shell (which would mask a real
// bug, e.g. a typo'd import path, as a soft navigation).
func TestSPAFallback(t *testing.T) {
	webFS := fstest.MapFS{
		"index.html":      {Data: []byte("<html>shell</html>")},
		"app.css":         {Data: []byte("body{}")},
		"js/main.js":      {Data: []byte("export {}")},
		"js/router.js":    {Data: []byte("export {}")},
		"manifest.json":   {Data: []byte("{}")},
		"assets/icon.png": {Data: []byte("fake-png")},
	}
	router := NewRouter(nil, nil, nil, fakeHealthChecker{}, fakeAuthenticator{}, webFS, true)

	cases := []struct {
		name       string
		path       string
		wantStatus int
		wantShell  bool // body should be index.html's content
	}{
		{"root serves shell", "/", http.StatusOK, true},
		{"known client route serves shell in place", "/projects", http.StatusOK, true},
		{"ideas client route serves shell in place", "/ideas", http.StatusOK, true},
		{"real asset serves as-is, not the shell", "/app.css", http.StatusOK, false},
		{"real nested asset serves as-is", "/js/main.js", http.StatusOK, false},
		{"unknown route still 404s, not the shell", "/does-not-exist", http.StatusNotFound, false},
		{"typo'd asset path still 404s, not the shell", "/js/shoping-view.js", http.StatusNotFound, false},
		{"api-shaped nonexistent path 404s, not the shell", "/api/v1/itms", http.StatusNotFound, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("GET %s: got status %d, want %d (body: %s)", tc.path, rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantShell && rec.Body.String() != "<html>shell</html>" {
				t.Errorf("GET %s: expected shell content, got %q", tc.path, rec.Body.String())
			}
			if !tc.wantShell && tc.wantStatus == http.StatusOK && rec.Body.String() == "<html>shell</html>" {
				t.Errorf("GET %s: unexpectedly got the SPA shell instead of the real asset", tc.path)
			}
		})
	}
}

// spyRepo records every mutating call so a test can assert none happened.
// Read methods return empty results; the point is what gets written, not
// what comes back.
type spyRepo struct {
	mutations []string
}

func (r *spyRepo) Create(_ context.Context, _ int64, _, _ string) (items.Item, error) {
	r.mutations = append(r.mutations, "Create")
	return items.Item{}, nil
}

func (r *spyRepo) Update(_ context.Context, _, _ int64, _ items.Location, _ float64) (items.Item, error) {
	r.mutations = append(r.mutations, "Update")
	return items.Item{}, nil
}

func (r *spyRepo) Delete(_ context.Context, _ int64) (bool, error) {
	r.mutations = append(r.mutations, "Delete")
	return false, nil
}

func (r *spyRepo) Get(_ context.Context, _ int64) (items.Item, error) {
	return items.Item{}, items.ErrNotFound{}
}

func (r *spyRepo) List(_ context.Context) ([]items.Item, error) { return nil, nil }

func (r *spyRepo) ExistsByNormalizedName(_ context.Context, _ string) (bool, items.Location, error) {
	return false, "", nil
}

func (r *spyRepo) MaxPosition(_ context.Context, _ items.Location) (float64, bool, error) {
	return 0, false, nil
}

// spyProjectsRepo mirrors spyRepo for the projects domain (EC-10/NFR-04):
// records every mutating call so a test can assert none happened.
type spyProjectsRepo struct {
	mutations []string
}

func (r *spyProjectsRepo) Create(_ context.Context, _ int64, _, _ string, _, _, _ *string) (projects.Project, error) {
	r.mutations = append(r.mutations, "Create")
	return projects.Project{}, nil
}

func (r *spyProjectsRepo) UpdatePreview(_ context.Context, _ int64, _, _, _ *string, _ string) error {
	r.mutations = append(r.mutations, "UpdatePreview")
	return nil
}

func (r *spyProjectsRepo) UpdateState(_ context.Context, _, _ int64, _ projects.State) (projects.Project, projects.State, error) {
	r.mutations = append(r.mutations, "UpdateState")
	return projects.Project{}, "", nil
}

func (r *spyProjectsRepo) Delete(_ context.Context, _ int64) (bool, error) {
	r.mutations = append(r.mutations, "Delete")
	return false, nil
}

func (r *spyProjectsRepo) Get(_ context.Context, _ int64) (projects.Project, error) {
	return projects.Project{}, projects.ErrNotFound{}
}

func (r *spyProjectsRepo) List(_ context.Context) ([]projects.Project, error) { return nil, nil }

func (r *spyProjectsRepo) ExistsByNormalizedName(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// spyIdeasRepo mirrors spyRepo/spyProjectsRepo for the ideas domain
// (EC-13/NFR-03): records every mutating call so a test can assert none
// happened.
type spyIdeasRepo struct {
	mutations []string
}

func (r *spyIdeasRepo) Create(_ context.Context, _ int64, _ string) (ideas.Idea, error) {
	r.mutations = append(r.mutations, "Create")
	return ideas.Idea{}, nil
}

func (r *spyIdeasRepo) UpdatePreview(_ context.Context, _ int64, _, _, _ *string, _ ideas.PreviewStatus) error {
	r.mutations = append(r.mutations, "UpdatePreview")
	return nil
}

func (r *spyIdeasRepo) Delete(_ context.Context, _ int64) (bool, error) {
	r.mutations = append(r.mutations, "Delete")
	return false, nil
}

func (r *spyIdeasRepo) Get(_ context.Context, _ int64) (ideas.Idea, error) {
	return ideas.Idea{}, ideas.ErrNotFound{}
}

func (r *spyIdeasRepo) List(_ context.Context) ([]ideas.Idea, error) { return nil, nil }

type spySink struct{}

func (spySink) Record(_ context.Context, _ int64, _ string, _ any) error { return nil }

type spyUsers struct{}

func (spyUsers) GetUser(_ context.Context, id int64) (items.User, error) {
	return items.User{ID: id, DisplayName: "Usuari A", AvatarEmoji: "🐦"}, nil
}

type fakeHealthChecker struct{}

func (fakeHealthChecker) Healthy() error { return nil }

type fakeAuthenticator struct{}

func (fakeAuthenticator) CurrentUser(r *http.Request) (auth.User, error) {
	return auth.User{ID: 1}, nil
}
