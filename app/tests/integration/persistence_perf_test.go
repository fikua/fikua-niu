package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"testing"
	"testing/fstest"
	"time"

	niu "niu"
	"niu/internal/auth"
	"niu/internal/httpapi"
	"niu/internal/items"
	"niu/internal/store"
)

// T-30 — EC-14 (restart persists data) and PERF-01/NFR-05 (p95 < 200ms
// with 500 items). PERF-02/NFR-06 (Lighthouse 3G <1s TTI) is a browser
// audit outside Go's reach — tracked as a manual/T-33 concern, not
// duplicated here as a fake automated result.

// EC-14: seed data, close and reopen the SQLite file (equivalent to a
// container restart against the same volume), assert the item set is
// identical.
func TestRestart_DataPersists(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "niu.db")

	st1, err := store.Open(dbPath, niu.MigrationsFS)
	if err != nil {
		t.Fatalf("store.Open (first): %v", err)
	}
	repo1 := store.NewItemsRepository(st1.DB)
	svc1 := items.NewService(repo1, repo1, repo1)

	names := []string{"Llet", "Pa", "Formatge"}
	var ids []int64
	for _, name := range names {
		item, err := svc1.Add(t.Context(), seedUserAID, name)
		if err != nil {
			t.Fatalf("seed Add(%q): %v", name, err)
		}
		ids = append(ids, item.ID)
	}
	// Move one item to pantry to also exercise moved_by/moved_at survival.
	if _, err := svc1.Move(t.Context(), seedUserBID, ids[0], items.LocationPantry); err != nil {
		t.Fatalf("seed Move: %v", err)
	}

	before, err := svc1.List(t.Context())
	if err != nil {
		t.Fatalf("List before restart: %v", err)
	}
	if err := st1.Close(); err != nil {
		t.Fatalf("close store (simulating restart): %v", err)
	}

	// Reopen the same file — equivalent to the container starting again
	// against the same volume.
	st2, err := store.Open(dbPath, niu.MigrationsFS)
	if err != nil {
		t.Fatalf("store.Open (second): %v", err)
	}
	t.Cleanup(func() { st2.Close() })
	repo2 := store.NewItemsRepository(st2.DB)
	svc2 := items.NewService(repo2, repo2, repo2)

	after, err := svc2.List(t.Context())
	if err != nil {
		t.Fatalf("List after restart: %v", err)
	}

	if len(before) != len(after) {
		t.Fatalf("item count changed across restart: before=%d after=%d", len(before), len(after))
	}
	beforeByID := map[int64]items.Item{}
	for _, it := range before {
		beforeByID[it.ID] = it
	}
	for _, it := range after {
		prev, ok := beforeByID[it.ID]
		if !ok {
			t.Fatalf("item %d present after restart but not before", it.ID)
		}
		if prev.Name != it.Name || prev.Location != it.Location {
			t.Fatalf("item %d state changed across restart: before=%+v after=%+v", it.ID, prev, it)
		}
	}
}

// PERF-01/NFR-05: p95 of GET /api/v1/items < 200ms with 500 items seeded.
func TestListItems_P95Latency_With500Items(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping perf test in -short mode")
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "niu.db")

	st, err := store.Open(dbPath, niu.MigrationsFS)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	repo := store.NewItemsRepository(st.DB)
	svc := items.NewService(repo, repo, repo)
	authenticator := auth.StubAuthenticator{UserID: seedUserAID}
	var emptyFS = fstest.MapFS{}
	router := httpapi.NewRouter(svc, st, authenticator, emptyFS, true)
	testSrv := httptest.NewServer(router)
	defer testSrv.Close()

	const itemCount = 500
	for i := 0; i < itemCount; i++ {
		if _, err := svc.Add(t.Context(), seedUserAID, fmt.Sprintf("Item %04d", i)); err != nil {
			t.Fatalf("seed item %d: %v", i, err)
		}
	}

	const sampleCount = 30
	durations := make([]time.Duration, 0, sampleCount)
	for i := 0; i < sampleCount; i++ {
		start := time.Now()
		res, err := http.Get(testSrv.URL + "/api/v1/items")
		if err != nil {
			t.Fatalf("GET /items sample %d: %v", i, err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET /items sample %d status = %d", i, res.StatusCode)
		}
		durations = append(durations, time.Since(start))
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95Index := int(float64(len(durations)) * 0.95)
	if p95Index >= len(durations) {
		p95Index = len(durations) - 1
	}
	p95 := durations[p95Index]

	t.Logf("p95 latency over %d samples with %d items: %v", sampleCount, itemCount, p95)
	if p95 > 200*time.Millisecond {
		t.Errorf("p95 latency = %v, want < 200ms (NFR-05)", p95)
	}
}
