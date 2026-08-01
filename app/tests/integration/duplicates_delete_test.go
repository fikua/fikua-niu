package integration

import (
	"net/http"
	"strconv"
	"testing"
)

// T-26 — CF-05 (6 duplicate combinations across both boxes), CF-06
// (create -> delete -> recreate same name), CF-10/EC-11 (DELETE + double
// DELETE idempotent), EC-12 (move nonexistent item -> clear error).

func TestDuplicate_TrimmedCaseInsensitive_AcrossBothBoxes(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	// Seed "Llet" in shopping, then move a second seed item ("Formatge")
	// into pantry so we can also test duplicate rejection against a
	// pantry-resident name.
	created := createItem(t, srv.Server, "Llet")
	pantryItem := createItem(t, srv.Server, "Formatge")
	move(t, srv.Server, pantryItem.ID, "pantry")

	variants := []string{"llet", "Llet ", "LLET", "  llet  ", "formatge", "FORMATGE"}
	for _, name := range variants {
		res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/items", map[string]string{"name": name})
		if res.StatusCode != http.StatusConflict {
			var body errorResponse
			decodeJSON(t, res, &body)
			t.Fatalf("POST %q status = %d, want 409 (body=%+v)", name, res.StatusCode, body)
		}
		var body errorResponse
		decodeJSON(t, res, &body)
		if body.Error.Code != "duplicate_item" {
			t.Fatalf("POST %q error code = %q, want duplicate_item", name, body.Error.Code)
		}
	}

	// Sanity: only the two original items exist.
	list := listItems(t, srv.Server)
	if len(list) != 2 {
		t.Fatalf("expected exactly 2 items after duplicate rejections, got %d: %+v", len(list), list)
	}
	_ = created
}

func TestDuplicate_ExactNameAllowedAfterDelete(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createItem(t, srv.Server, "Pa")

	delRes := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/items/"+strconv.FormatInt(created.ID, 10), nil)
	if delRes.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, want 204", delRes.StatusCode)
	}

	// EC-07: same name is now accepted as a brand new item (the check
	// only looks at active items, not history). SQLite's INTEGER PRIMARY
	// KEY may legitimately reuse the freed rowid, so we assert acceptance
	// and correct persisted state, not a distinct id.
	recreated := createItem(t, srv.Server, "Pa")
	if recreated.Name != "Pa" {
		t.Fatalf("recreated item name = %q, want Pa", recreated.Name)
	}
	list := listItems(t, srv.Server)
	if len(list) != 1 || list[0].Name != "Pa" {
		t.Fatalf("items after recreate = %+v, want exactly one Pa", list)
	}
	_ = created
}

func TestDeleteItem_IdempotentDoubleDelete(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createItem(t, srv.Server, "Oli")

	first := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/items/"+strconv.FormatInt(created.ID, 10), nil)
	if first.StatusCode != http.StatusNoContent {
		t.Fatalf("first DELETE status = %d, want 204", first.StatusCode)
	}

	second := doJSON(t, http.MethodDelete, srv.URL+"/api/v1/items/"+strconv.FormatInt(created.ID, 10), nil)
	if second.StatusCode != http.StatusNoContent {
		t.Fatalf("second DELETE status = %d, want 204 (idempotent, EC-11)", second.StatusCode)
	}

	list := listItems(t, srv.Server)
	if len(list) != 0 {
		t.Fatalf("items after double delete = %+v, want empty", list)
	}
}

func TestMoveItem_NotFound(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	res := doJSON(t, http.MethodPatch, srv.URL+"/api/v1/items/999999", map[string]string{"location": "pantry"})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("PATCH nonexistent item status = %d, want 404", res.StatusCode)
	}
	var body errorResponse
	decodeJSON(t, res, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("PATCH nonexistent item error code = %q, want not_found", body.Error.Code)
	}
}
