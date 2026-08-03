package integration

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- shared test helpers (mirror items_move_test.go's createItem/move/listItems) ----

func createProject(t *testing.T, srv *httptest.Server, name string) projectDTO {
	t.Helper()
	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects", map[string]any{"name": name})
	if res.StatusCode != http.StatusCreated {
		var errBody errorResponse
		decodeJSON(t, res, &errBody)
		t.Fatalf("createProject(%q) status = %d, error = %+v", name, res.StatusCode, errBody)
	}
	var created projectDTO
	decodeJSON(t, res, &created)
	return created
}

func createProjectWithFields(t *testing.T, srv *httptest.Server, name string, budget, targetDate *string) projectDTO {
	t.Helper()
	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/projects", map[string]any{
		"name": name, "budget": budget, "target_date": targetDate,
	})
	if res.StatusCode != http.StatusCreated {
		var errBody errorResponse
		decodeJSON(t, res, &errBody)
		t.Fatalf("createProjectWithFields(%q) status = %d, error = %+v", name, res.StatusCode, errBody)
	}
	var created projectDTO
	decodeJSON(t, res, &created)
	return created
}

func changeProjectStateHTTP(t *testing.T, srv *httptest.Server, id int64, state string) *http.Response {
	t.Helper()
	return doJSON(t, http.MethodPatch, srv.URL+"/api/v1/projects/"+strconv.FormatInt(id, 10), map[string]string{"state": state})
}

func listProjects(t *testing.T, srv *httptest.Server) []projectDTO {
	t.Helper()
	res, err := http.Get(srv.URL + "/api/v1/projects")
	if err != nil {
		t.Fatalf("GET /projects: %v", err)
	}
	var list projectsListResponse
	decodeJSON(t, res, &list)
	return list.Projects
}

// ---- T-24: AC-02/AC-03/AC-09 all directions, EC-12/EC-13 ----

func TestProjects_ChangeState_AllDirections(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Televisor nou")
	if created.State != "idea" {
		t.Fatalf("initial state = %q, want idea (AC-01)", created.State)
	}

	transitions := []string{"decidit", "fet", "decidit", "idea"}
	for _, next := range transitions {
		res := changeProjectStateHTTP(t, srv.Server, created.ID, next)
		if res.StatusCode != http.StatusOK {
			var errBody errorResponse
			decodeJSON(t, res, &errBody)
			t.Fatalf("PATCH state=%q status = %d, error = %+v", next, res.StatusCode, errBody)
		}
		var updated projectDTO
		decodeJSON(t, res, &updated)
		if updated.State != next {
			t.Fatalf("PATCH state=%q resulting state = %q, want %q", next, updated.State, next)
		}
		if updated.LastUpdatedBy == nil || updated.LastUpdatedBy.ID != seedUserAID {
			t.Fatalf("PATCH state=%q last_updated_by = %+v, want user %d", next, updated.LastUpdatedBy, seedUserAID)
		}
	}
}

// EC-12: PATCH on a nonexistent id -> 404, other elements unaffected.
func TestProjects_ChangeState_NotFound(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	other := createProject(t, srv.Server, "Sofà")

	res := changeProjectStateHTTP(t, srv.Server, 999999, "decidit")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("PATCH nonexistent project status = %d, want 404", res.StatusCode)
	}
	var body errorResponse
	decodeJSON(t, res, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("PATCH nonexistent project error code = %q, want not_found", body.Error.Code)
	}

	list := listProjects(t, srv.Server)
	if len(list) != 1 || list[0].ID != other.ID || list[0].State != "idea" {
		t.Fatalf("other project affected by failed PATCH: %+v", list)
	}
}

// EC-13: double DELETE is idempotent, no 5xx.
func TestProjects_Delete_IdempotentDoubleDelete(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Rentavaixelles")

	first := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/projects/"+strconv.FormatInt(created.ID, 10), nil)
	if first.StatusCode != http.StatusNoContent {
		t.Fatalf("first DELETE status = %d, want 204", first.StatusCode)
	}

	second := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/projects/"+strconv.FormatInt(created.ID, 10), nil)
	if second.StatusCode != http.StatusNoContent {
		t.Fatalf("second DELETE status = %d, want 204 (idempotent, EC-13)", second.StatusCode)
	}

	list := listProjects(t, srv.Server)
	if len(list) != 0 {
		t.Fatalf("projects after double delete = %+v, want empty", list)
	}
}

// ---- T-25: AC-05, EC-04 ----

func TestProjects_Add_PersistsAndListed(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Bicicleta elèctrica")

	list := listProjects(t, srv.Server)
	if len(list) != 1 || list[0].ID != created.ID || list[0].Name != "Bicicleta elèctrica" {
		t.Fatalf("GET /projects after create = %+v, want the created project (AC-01)", list)
	}
}

func TestProjects_Delete_RemovesFromList(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Forn nou")

	res := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/projects/"+strconv.FormatInt(created.ID, 10), nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", res.StatusCode)
	}

	list := listProjects(t, srv.Server)
	if len(list) != 0 {
		t.Fatalf("projects after delete = %+v, want empty (AC-05)", list)
	}
}

// EC-04: create -> delete -> recreate same name is accepted as new.
func TestProjects_Duplicate_ExactNameAllowedAfterDelete(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Cadira ergonòmica")

	delRes := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/projects/"+strconv.FormatInt(created.ID, 10), nil)
	if delRes.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", delRes.StatusCode)
	}

	recreated := createProject(t, srv.Server, "Cadira ergonòmica")
	if recreated.Name != "Cadira ergonòmica" {
		t.Fatalf("recreated project name = %q, want Cadira ergonòmica", recreated.Name)
	}
	list := listProjects(t, srv.Server)
	if len(list) != 1 || list[0].Name != "Cadira ergonòmica" {
		t.Fatalf("projects after recreate = %+v, want exactly one", list)
	}
}

// ---- T-27: NFR-01 events + AC-06 convergence ----

func TestProjects_ChangeState_WritesEventWithFromTo(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Escriptori nou")

	res := changeProjectStateHTTP(t, srv.Server, created.ID, "decidit")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 200", res.StatusCode)
	}

	var kind, payload string
	err := srv.Store.DB.QueryRow(
		`SELECT kind, payload FROM events WHERE kind = 'project_state_changed' ORDER BY id DESC LIMIT 1`,
	).Scan(&kind, &payload)
	if err != nil {
		t.Fatalf("query events: %v", err)
	}
	if kind != "project_state_changed" {
		t.Fatalf("event kind = %q, want project_state_changed", kind)
	}
	if !strings.Contains(payload, `"from":"idea"`) || !strings.Contains(payload, `"to":"decidit"`) {
		t.Fatalf("event payload = %q, want it to contain from=idea and to=decidit", payload)
	}
}

// ---- T-28: AC-07 concurrency ----

func TestProjects_ConcurrentStateChange_NoErrorConverges(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Renovar terrassa")

	var wg sync.WaitGroup
	statuses := make([]int, 2)
	wg.Add(2)

	go func() {
		defer wg.Done()
		res := changeProjectStateHTTP(t, srv.Server, created.ID, "decidit")
		statuses[0] = res.StatusCode
	}()
	go func() {
		defer wg.Done()
		res := changeProjectStateHTTP(t, srv.Server, created.ID, "fet")
		statuses[1] = res.StatusCode
	}()
	wg.Wait()

	for i, status := range statuses {
		if status >= 500 {
			t.Fatalf("concurrent PATCH #%d returned 5xx: %d", i, status)
		}
	}

	list := listProjects(t, srv.Server)
	if len(list) != 1 {
		t.Fatalf("expected exactly one project, got %d", len(list))
	}
	if list[0].State != "decidit" && list[0].State != "fet" {
		t.Fatalf("final state = %q, want one of the two PATCHed states", list[0].State)
	}
}

// TestProjects_ConcurrentStateChange_Repeated is the same race run many
// times — a single-shot concurrency test is close to useless if the two
// goroutines happen not to overlap (same rationale as NIU-1's
// TestTwoUsers_ConcurrentMove_Repeated).
func TestProjects_ConcurrentStateChange_Repeated(t *testing.T) {
	const rounds = 25

	for round := 0; round < rounds; round++ {
		srv := newTestServer(t, seedUserAID)
		created := createProject(t, srv.Server, "Renovar cuina")

		var wg sync.WaitGroup
		statuses := make([]int, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			res := changeProjectStateHTTP(t, srv.Server, created.ID, "decidit")
			statuses[0] = res.StatusCode
		}()
		go func() {
			defer wg.Done()
			res := changeProjectStateHTTP(t, srv.Server, created.ID, "fet")
			statuses[1] = res.StatusCode
		}()
		wg.Wait()

		for i, status := range statuses {
			if status >= 500 {
				t.Fatalf("round %d: concurrent PATCH #%d returned %d — AC-07 requires that neither request fails", round, i, status)
			}
		}

		list := listProjects(t, srv.Server)
		if len(list) != 1 {
			t.Fatalf("round %d: expected exactly one project, got %d", round, len(list))
		}
	}
}

// ---- T-29: EC-05 (no automatic expiry) and EC-06 (no "abandoned" state) ----

// EC-05: a project sitting in "decidit" for weeks/months (simulated via a
// stale updated_at) is still visible with its real state and last-change
// timestamp — no automatic expiry, archival, or silent hiding.
func TestProjects_StaleDecidit_NoAutomaticChange(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Reformar bany")
	res := changeProjectStateHTTP(t, srv.Server, created.ID, "decidit")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH state=decidit status = %d, want 200", res.StatusCode)
	}

	// Simulate the project having sat untouched for six months.
	staleTimestamp := time.Now().AddDate(0, -6, 0).UTC().Format("2006-01-02 15:04:05")
	if _, err := srv.Store.DB.Exec(
		`UPDATE projects SET updated_at = ? WHERE id = ?`, staleTimestamp, created.ID,
	); err != nil {
		t.Fatalf("simulate stale updated_at: %v", err)
	}

	// A read (GET, "qualsevol dels dos usuaris el consulta") must not
	// trigger any automatic state change, archival, or hiding — the row
	// stays visible with its real state and the simulated stale timestamp.
	list := listProjects(t, srv.Server)
	if len(list) != 1 {
		t.Fatalf("stale project missing from list — got %d project(s), want 1 (EC-05: no silent hiding)", len(list))
	}
	if list[0].State != "decidit" {
		t.Fatalf("stale project state = %q, want decidit unchanged (EC-05: no automatic expiry)", list[0].State)
	}
	if list[0].UpdatedAt.Format("2006-01-02") != staleTimestamp[:10] {
		t.Fatalf("stale project updated_at = %v, want the simulated stale timestamp preserved (no silent touch)", list[0].UpdatedAt)
	}
}

// EC-06: there is no "abandonat/descartat" state — the CHECK constraint
// only allows idea/decidit/fet, and the only way to remove an unwanted
// project is DELETE.
func TestProjects_NoAbandonedState_OnlyDeleteAvailable(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Idea descartada")

	res := changeProjectStateHTTP(t, srv.Server, created.ID, "abandonat")
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("PATCH state=abandonat status = %d, want 400 (EC-06: no such state exists)", res.StatusCode)
	}
	var body errorResponse
	decodeJSON(t, res, &body)
	if body.Error.Code != "validation_failed" {
		t.Fatalf("PATCH state=abandonat error code = %q, want validation_failed", body.Error.Code)
	}

	// Confirm the CHECK constraint itself rejects a fourth state directly
	// at the database level too — not just at the Go validation layer.
	_, err := srv.Store.DB.Exec(`UPDATE projects SET state = 'abandonat' WHERE id = ?`, created.ID)
	if err == nil {
		t.Fatal("direct SQL UPDATE to state='abandonat' succeeded, want CHECK constraint violation (EC-06)")
	}

	// The only available action for an unwanted project is DELETE.
	delRes := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/projects/"+strconv.FormatInt(created.ID, 10), nil)
	if delRes.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", delRes.StatusCode)
	}
	if list := listProjects(t, srv.Server); len(list) != 0 {
		t.Fatalf("projects after delete = %+v, want empty", list)
	}
}

// ---- T-31: AC-14/AC-15 budget/target_date persistence ----

func TestProjects_Add_BudgetAndTargetDatePersisted(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	budget := "uns 600€, potser més"
	targetDate := "2026-12-01"
	created := createProjectWithFields(t, srv.Server, "Televisor 4K", &budget, &targetDate)

	if created.Budget == nil || *created.Budget != budget {
		t.Fatalf("created.Budget = %+v, want %q", created.Budget, budget)
	}
	if created.TargetDate == nil || *created.TargetDate != targetDate {
		t.Fatalf("created.TargetDate = %+v, want %q", created.TargetDate, targetDate)
	}

	list := listProjects(t, srv.Server)
	if len(list) != 1 || list[0].Budget == nil || *list[0].Budget != budget {
		t.Fatalf("GET /projects budget mismatch: %+v", list)
	}
	if list[0].TargetDate == nil || *list[0].TargetDate != targetDate {
		t.Fatalf("GET /projects target_date mismatch: %+v", list)
	}
}

func TestProjects_Add_BudgetAndTargetDateOmittedAreNull(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Prestatgeria")

	if created.Budget != nil {
		t.Fatalf("created.Budget = %+v, want nil when omitted (AC-14)", created.Budget)
	}
	if created.TargetDate != nil {
		t.Fatalf("created.TargetDate = %+v, want nil when omitted (AC-15)", created.TargetDate)
	}
}
