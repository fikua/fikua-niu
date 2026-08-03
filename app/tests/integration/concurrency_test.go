package integration

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"testing/fstest"

	niu "niu"
	"niu/internal/auth"
	"niu/internal/httpapi"
	"niu/internal/items"
	"niu/internal/projects"
	"niu/internal/store"
)

// twoUserServers wires two independent httptest.Server instances backed
// by the SAME SQLite database file, one authenticating as Usuari A and
// the other as Usuari B — simulating two browser sessions (CF-11,
// CF-12/AC-09, CF-13).
type twoUserServers struct {
	A *httptest.Server
	B *httptest.Server
}

func newTwoUserServers(t *testing.T) *twoUserServers {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "niu.db")

	stA, err := store.Open(dbPath, niu.MigrationsFS)
	if err != nil {
		t.Fatalf("store.Open (A): %v", err)
	}
	t.Cleanup(func() { stA.Close() })

	repoA := store.NewItemsRepository(stA.DB)
	svcA := items.NewService(repoA, repoA, repoA)
	projectsRepoA := store.NewProjectsRepository(stA.DB)
	projectsSvcA := projects.NewService(projectsRepoA, projectsRepoA)
	ideasSvcA := newIdeasService(t, stA)
	var emptyFS = fstest.MapFS{}
	routerA := httpapi.NewRouter(svcA, projectsSvcA, ideasSvcA, stA, auth.StubAuthenticator{UserID: seedUserAID}, emptyFS, true)
	srvA := httptest.NewServer(routerA)
	t.Cleanup(srvA.Close)

	// A second Store instance opens the same underlying SQLite file
	// (WAL mode allows this) so goose does not re-run migrations against
	// a different logical database.
	stB, err := store.Open(dbPath, niu.MigrationsFS)
	if err != nil {
		t.Fatalf("store.Open (B): %v", err)
	}
	t.Cleanup(func() { stB.Close() })

	repoB := store.NewItemsRepository(stB.DB)
	svcB := items.NewService(repoB, repoB, repoB)
	projectsRepoB := store.NewProjectsRepository(stB.DB)
	projectsSvcB := projects.NewService(projectsRepoB, projectsRepoB)
	ideasSvcB := newIdeasService(t, stB)
	routerB := httpapi.NewRouter(svcB, projectsSvcB, ideasSvcB, stB, auth.StubAuthenticator{UserID: seedUserBID}, emptyFS, true)
	srvB := httptest.NewServer(routerB)
	t.Cleanup(srvB.Close)

	return &twoUserServers{A: srvA, B: srvB}
}

// CF-11 — A adds an item, B sees it via GET (convergence, AC-08).
func TestTwoUsers_Convergence(t *testing.T) {
	srv := newTwoUserServers(t)

	res := doJSON(t, http.MethodPost, srv.A.URL+"/api/v1/items", map[string]string{"name": "Mel"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST as A status = %d, want 201", res.StatusCode)
	}

	getRes, err := http.Get(srv.B.URL + "/api/v1/items")
	if err != nil {
		t.Fatalf("GET as B: %v", err)
	}
	var list itemsListResponse
	decodeJSON(t, getRes, &list)
	if len(list.Items) != 1 || list.Items[0].Name != "Mel" {
		t.Fatalf("B's view = %+v, want single Mel item added by A", list.Items)
	}
}

// CF-12/AC-09 — two concurrent PATCH from A and B on the same item never
// return 5xx, and a subsequent GET shows a single consistent state
// matching whichever PATCH response has the latest updated_at (ADR-01).
func TestTwoUsers_ConcurrentMove_NoErrorConverges(t *testing.T) {
	srv := newTwoUserServers(t)

	created := createItem(t, srv.A, "Sucre")

	var wg sync.WaitGroup
	statuses := make([]int, 2)
	wg.Add(2)

	go func() {
		defer wg.Done()
		res := doJSON(t, http.MethodPatch, srv.A.URL+"/api/v1/items/"+idStr(created.ID), map[string]string{"location": "pantry"})
		statuses[0] = res.StatusCode
	}()
	go func() {
		defer wg.Done()
		res := doJSON(t, http.MethodPatch, srv.B.URL+"/api/v1/items/"+idStr(created.ID), map[string]string{"location": "shopping"})
		statuses[1] = res.StatusCode
	}()
	wg.Wait()

	for i, status := range statuses {
		if status >= 500 {
			t.Fatalf("concurrent PATCH #%d returned 5xx: %d", i, status)
		}
	}

	// After both requests complete, both clients must converge on the
	// same single state via GET (ADR-01: last write wins by server
	// timestamp).
	listA := listItems(t, srv.A)
	listB := listItemsFrom(t, srv.B)
	if len(listA) != 1 || len(listB) != 1 {
		t.Fatalf("expected exactly one item on both sides, got A=%d B=%d", len(listA), len(listB))
	}
	if listA[0].Location != listB[0].Location {
		t.Fatalf("A and B diverge after concurrent move: A=%q B=%q", listA[0].Location, listB[0].Location)
	}
}

// TestTwoUsers_ConcurrentMove_Repeated is the same race run many times.
//
// A single-shot concurrency test is close to useless: the two goroutines
// often do not actually overlap, so the test passes without ever
// exercising the contention it claims to cover. This one failed 5 runs
// out of 5 when a deferred SQLite transaction made the second writer
// return 500 (SQLITE_BUSY on lock upgrade) — a defect a single run could
// easily have missed.
func TestTwoUsers_ConcurrentMove_Repeated(t *testing.T) {
	const rounds = 25

	for round := 0; round < rounds; round++ {
		srv := newTwoUserServers(t)
		created := createItem(t, srv.A, "Cursa")

		var wg sync.WaitGroup
		statuses := make([]int, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			res := doJSON(t, http.MethodPatch, srv.A.URL+"/api/v1/items/"+idStr(created.ID), map[string]string{"location": "pantry"})
			statuses[0] = res.StatusCode
		}()
		go func() {
			defer wg.Done()
			res := doJSON(t, http.MethodPatch, srv.B.URL+"/api/v1/items/"+idStr(created.ID), map[string]string{"location": "shopping"})
			statuses[1] = res.StatusCode
		}()
		wg.Wait()

		for i, status := range statuses {
			if status >= 500 {
				t.Fatalf("round %d: concurrent PATCH #%d returned %d — AC-09 requires that neither request fails",
					round, i, status)
			}
		}

		listA := listItems(t, srv.A)
		listB := listItemsFrom(t, srv.B)
		if len(listA) != 1 || len(listB) != 1 {
			t.Fatalf("round %d: expected one item on both sides, got A=%d B=%d", round, len(listA), len(listB))
		}
		if listA[0].Location != listB[0].Location {
			t.Fatalf("round %d: clients diverge: A=%q B=%q", round, listA[0].Location, listB[0].Location)
		}
	}
}

// CF-13 — item added by A, moved by B: response identifies A as creator
// and B as the last mover.
func TestTwoUsers_AuthorshipAttribution(t *testing.T) {
	srv := newTwoUserServers(t)

	created := createItem(t, srv.A, "Cafè")

	res := doJSON(t, http.MethodPatch, srv.B.URL+"/api/v1/items/"+idStr(created.ID), map[string]string{"location": "pantry"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH as B status = %d, want 200", res.StatusCode)
	}
	var updated itemDTO
	decodeJSON(t, res, &updated)

	if updated.AddedBy == nil || updated.AddedBy.ID != seedUserAID {
		t.Fatalf("added_by = %+v, want user %d (A)", updated.AddedBy, seedUserAID)
	}
	if updated.MovedBy == nil || updated.MovedBy.ID != seedUserBID {
		t.Fatalf("moved_by = %+v, want user %d (B)", updated.MovedBy, seedUserBID)
	}
}

func listItemsFrom(t *testing.T, srv *httptest.Server) []itemDTO {
	t.Helper()
	res, err := http.Get(srv.URL + "/api/v1/items")
	if err != nil {
		t.Fatalf("GET /items: %v", err)
	}
	var list itemsListResponse
	decodeJSON(t, res, &list)
	return list.Items
}

func idStr(id int64) string {
	return strconv.FormatInt(id, 10)
}
