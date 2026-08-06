package integration

import (
	"encoding/json"
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
//
// F-23: beyond "no 5xx" and "converges to one of the two states", this
// also asserts the *audit trail itself* is correct under the race — the
// two project_state_changed events must chain validly from the project's
// real starting state ("idea") to its two distinct requested destinations
// ("decidit", "fet"): whichever UpdateState transaction actually committed
// first must have read "idea" as its "from", and whichever committed
// second must have read the first one's "to" as its own "from". Note this
// check does NOT assume events.id order reflects transaction-commit order
// — Service.ChangeState calls sink.Record (a separate, non-transactional
// INSERT into events) only after UpdateState's transaction has already
// committed, so a goroutine that wins the UpdateState commit race can
// still lose the Record race if it is descheduled in between; the
// invariant under test is therefore checked order-independently. Before
// the F-23 fix, both events could independently record "from":"idea" even
// though only one of them could have truly observed that value at its own
// commit — this asserts that specific corruption cannot happen, regardless
// of insertion order.
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

		assertStateChangedEventChainIsConsistent(t, srv, created.ID, round)
	}
}

// assertStateChangedEventChainIsConsistent queries every
// project_state_changed event for projectID and asserts the from/to values
// form a valid 2-step chain starting at "idea", regardless of the order
// the two rows were inserted in (see the note on TestProjects_
// ConcurrentStateChange_Repeated for why events.id order is NOT a proxy
// for UpdateState transaction-commit order). The two requests in this test
// always target fixed, distinct destinations ("decidit" and "fet"), so a
// valid chain looks like: one event has from="idea" (the transaction that
// truly committed first), and the other has from=<the first event's to>
// (the transaction that committed second, which correctly observed what
// the first one left behind). A wrong "from" under the race (F-23) breaks
// this chain even though the HTTP responses and final state look fine.
func assertStateChangedEventChainIsConsistent(t *testing.T, srv *testServer, projectID int64, round int) {
	t.Helper()

	rows, err := srv.Store.DB.Query(
		`SELECT payload FROM events
		 WHERE kind = 'project_state_changed'
		   AND json_extract(payload, '$.project_id') = ?`,
		projectID,
	)
	if err != nil {
		t.Fatalf("round %d: query project_state_changed events: %v", round, err)
	}
	defer rows.Close()

	type fromTo struct{ From, To string }
	var chain []fromTo
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			t.Fatalf("round %d: scan event payload: %v", round, err)
		}
		var decoded struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
			t.Fatalf("round %d: unmarshal event payload %q: %v", round, payload, err)
		}
		chain = append(chain, fromTo{From: decoded.From, To: decoded.To})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("round %d: iterate event rows: %v", round, err)
	}

	if len(chain) != 2 {
		t.Fatalf("round %d: expected exactly 2 project_state_changed events, got %d (%+v)", round, len(chain), chain)
	}

	// Identify which event committed first: it is the one whose "to" does
	// NOT equal the other event's "from" (the second-committing event's
	// "from" must equal the first one's "to" under a correct fix).
	a, b := chain[0], chain[1]
	var first, second fromTo
	switch {
	case b.From == a.To:
		first, second = a, b
	case a.From == b.To:
		first, second = b, a
	default:
		t.Fatalf("round %d: neither event's from matches the other's to — chain does not connect at all (chain: %+v)", round, chain)
	}

	if first.From != "idea" {
		t.Fatalf("round %d: first-committed event has from=%q, want %q — the audit trail chain is broken, exactly the corruption F-23 guards against (chain: %+v)", round, first.From, "idea", chain)
	}
	if second.From != first.To {
		// Unreachable given the switch above, kept as an explicit
		// invariant statement for readability.
		t.Fatalf("round %d: second-committed event has from=%q, want %q (chain: %+v)", round, second.From, first.To, chain)
	}

	gotDestinations := map[string]bool{first.To: true, second.To: true}
	if !gotDestinations["decidit"] || !gotDestinations["fet"] {
		t.Fatalf("round %d: expected the two events' destinations to be exactly {decidit, fet}, got %+v", round, chain)
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

// ---- T-12 (NIU-11): POST /api/v1/projects with url ----

// waitForProjectPreviewStatus polls GET /api/v1/projects until id reaches
// a non-nil, non-pending preview_status or the deadline passes — the
// worker pool resolves asynchronously (same ADR-03 rule reused from
// NIU-6), so tests cannot assert on the outcome synchronously after POST
// returns.
func waitForProjectPreviewStatus(t *testing.T, srv *httptest.Server, id int64, timeout time.Duration) projectDTO {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, p := range listProjects(t, srv) {
			if p.ID == id && p.PreviewStatus != nil && *p.PreviewStatus != "pending" {
				return p
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("project %d still pending after %s", id, timeout)
	return projectDTO{}
}

// TestProjects_Add_WithURL_Returns201WithoutWaitingForScrape covers
// tasks.md T-12's first half: POST with a url returns 201 immediately
// with preview_status="pending" (never waiting for the scrape, mirrors
// ideas' ADR-03), and the scrape eventually resolves to "ready".
func TestProjects_Add_WithURL_Returns201WithoutWaitingForScrape(t *testing.T) {
	mock := newMockPreviewServer(t)
	mock.SetOpenGraph("Televisor 4K fantàstic", mock.URL+"/img.jpg", "La millor oferta de la ciutat.")

	srv := newProjectsHTTPTestServer(t, seedUserAID)

	res := doJSON(t, http.MethodPost, srv.Server.URL+"/api/v1/projects", map[string]any{
		"name": "Televisor 4K amb enllaç", "url": mock.URL,
	})
	if res.StatusCode != http.StatusCreated {
		var errBody errorResponse
		decodeJSON(t, res, &errBody)
		t.Fatalf("POST /projects with url status = %d, error = %+v", res.StatusCode, errBody)
	}
	var withURL projectDTO
	decodeJSON(t, res, &withURL)

	if withURL.URL == nil || *withURL.URL != mock.URL {
		t.Fatalf("created.URL = %v, want %q", withURL.URL, mock.URL)
	}
	if withURL.PreviewStatus == nil || *withURL.PreviewStatus != "pending" {
		t.Fatalf("POST response preview_status = %v, want pending (201 never waits for the scrape)", withURL.PreviewStatus)
	}

	resolved := waitForProjectPreviewStatus(t, srv.Server, withURL.ID, 2*time.Second)
	if *resolved.PreviewStatus != "ready" {
		t.Fatalf("resolved preview_status = %q, want ready", *resolved.PreviewStatus)
	}
	if resolved.Title == nil || *resolved.Title != "Televisor 4K fantàstic" {
		t.Errorf("Title = %v", resolved.Title)
	}
	if resolved.ImageURL == nil || *resolved.ImageURL != mock.URL+"/img.jpg" {
		t.Errorf("ImageURL = %v", resolved.ImageURL)
	}
}

// TestProjects_Add_WithURL_DescriptionNeverExposed covers tasks.md T-12's
// second half (T-05's contract): description is persisted (it comes free
// from the same Preview as title/image_url) but the JSON response must
// never contain the key at all — checked against the raw response bytes,
// not just a struct field that would silently ignore an unknown key.
func TestProjects_Add_WithURL_DescriptionNeverExposed(t *testing.T) {
	mock := newMockPreviewServer(t)
	mock.SetOpenGraph("Sofà de disseny", mock.URL+"/sofa.jpg", "Descripció detallada del sofà.")

	srv := newProjectsHTTPTestServer(t, seedUserAID)

	res := doJSON(t, http.MethodPost, srv.Server.URL+"/api/v1/projects", map[string]any{
		"name": "Sofà nou", "url": mock.URL,
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /projects with url status = %d", res.StatusCode)
	}
	var created projectDTO
	body := readAllBytes(t, res)
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("unmarshal created project: %v", err)
	}
	if strings.Contains(string(body), `"description"`) {
		t.Fatalf("POST /projects response contains a \"description\" key, want it entirely absent: %s", body)
	}

	resolved := waitForProjectPreviewStatus(t, srv.Server, created.ID, 2*time.Second)
	if *resolved.PreviewStatus != "ready" {
		t.Fatalf("resolved preview_status = %q, want ready", *resolved.PreviewStatus)
	}

	getRes, err := http.Get(srv.Server.URL + "/api/v1/projects")
	if err != nil {
		t.Fatalf("GET /projects: %v", err)
	}
	getBody := readAllBytes(t, getRes)
	if strings.Contains(string(getBody), `"description"`) {
		t.Fatalf("GET /projects response contains a \"description\" key, want it entirely absent: %s", getBody)
	}
}

// TestProjects_Add_WithoutURL_PreviewFieldsAllNull covers the "sense URL"
// half of decision 2: a project created without a url exposes url/title/
// image_url/preview_status all as null, exactly like before this feature.
func TestProjects_Add_WithoutURL_PreviewFieldsAllNull(t *testing.T) {
	srv := newProjectsHTTPTestServer(t, seedUserAID)

	created := createProject(t, srv.Server, "Prestatgeria sense enllaç")

	if created.URL != nil {
		t.Fatalf("created.URL = %v, want nil", created.URL)
	}
	if created.PreviewStatus != nil {
		t.Fatalf("created.PreviewStatus = %v, want nil (never pending — no preview to resolve)", created.PreviewStatus)
	}
	if created.Title != nil || created.ImageURL != nil {
		t.Fatalf("created preview fields = %+v, want all nil", created)
	}
}
