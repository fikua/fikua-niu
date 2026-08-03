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
	"niu/internal/items"
	"niu/internal/projects"
)

// TestNoMutatingGETRoutes introspects the chi route table (EC-08/NFR-04)
// and asserts that no GET route also has a POST/PATCH/PUT/DELETE handler
// registered on the exact same pattern — i.e. GET is never wired to a
// handler shared with a mutating method.
func TestNoMutatingGETRoutes(t *testing.T) {
	router := NewRouter(nil, nil, fakeHealthChecker{}, fakeAuthenticator{}, fstest.MapFS{}, true)

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
	projectsRepo := &spyProjectsRepo{}
	projectsSvc := projects.NewService(projectsRepo, &spySink{})
	router := NewRouter(svc, projectsSvc, fakeHealthChecker{}, fakeAuthenticator{}, fstest.MapFS{}, true)

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

func (r *spyProjectsRepo) Create(_ context.Context, _ int64, _, _ string, _, _ *string) (projects.Project, error) {
	r.mutations = append(r.mutations, "Create")
	return projects.Project{}, nil
}

func (r *spyProjectsRepo) UpdateState(_ context.Context, _, _ int64, _ projects.State) (projects.Project, error) {
	r.mutations = append(r.mutations, "UpdateState")
	return projects.Project{}, nil
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
