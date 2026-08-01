package integration

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// T-25 — CF-01/CF-07/CF-08/CF-09/AC-04: POST -> GET persistence, PATCH
// location=pantry/shopping, assert new location and authorship fields
// both in the PATCH response and in a subsequent GET.

func TestAddItem_PersistsAcrossGet(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/items", map[string]string{"name": "Llet"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /items status = %d, want 201", res.StatusCode)
	}
	var created itemDTO
	decodeJSON(t, res, &created)
	if created.Name != "Llet" || created.Location != "shopping" {
		t.Fatalf("created item mismatch: %+v", created)
	}

	getRes, err := http.Get(srv.URL + "/api/v1/items")
	if err != nil {
		t.Fatalf("GET /items: %v", err)
	}
	var list itemsListResponse
	decodeJSON(t, getRes, &list)
	if len(list.Items) != 1 || list.Items[0].Name != "Llet" {
		t.Fatalf("GET /items after POST = %+v, want single Llet item", list.Items)
	}
}

func TestMoveItem_ShoppingToPantry(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createItem(t, srv.Server, "Arròs")

	res := doJSON(t, http.MethodPatch, srv.URL+"/api/v1/items/"+strconv.FormatInt(created.ID, 10), map[string]string{"location": "pantry"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 200", res.StatusCode)
	}
	var updated itemDTO
	decodeJSON(t, res, &updated)

	if updated.Location != "pantry" {
		t.Fatalf("location after PATCH = %q, want pantry", updated.Location)
	}
	if updated.MovedBy == nil || updated.MovedBy.ID != seedUserAID {
		t.Fatalf("moved_by after PATCH = %+v, want user %d", updated.MovedBy, seedUserAID)
	}
	if updated.MovedAt == nil {
		t.Fatalf("moved_at after PATCH is nil, want a timestamp")
	}

	// CF-09: persists across an independent GET.
	list := listItems(t, srv.Server)
	if len(list) != 1 || list[0].Location != "pantry" {
		t.Fatalf("GET /items after PATCH = %+v, want single pantry item", list)
	}
}

func TestMoveItem_PantryToShopping(t *testing.T) {
	srv := newTestServer(t, seedUserAID)

	created := createItem(t, srv.Server, "Oli")
	move(t, srv.Server, created.ID, "pantry")

	res := doJSON(t, http.MethodPatch, srv.URL+"/api/v1/items/"+strconv.FormatInt(created.ID, 10), map[string]string{"location": "shopping"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH back to shopping status = %d, want 200", res.StatusCode)
	}
	var updated itemDTO
	decodeJSON(t, res, &updated)
	if updated.Location != "shopping" {
		t.Fatalf("location after second PATCH = %q, want shopping", updated.Location)
	}
}

// ---- shared test helpers ----

func createItem(t *testing.T, srv *httptest.Server, name string) itemDTO {
	t.Helper()
	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/items", map[string]string{"name": name})
	if res.StatusCode != http.StatusCreated {
		var errBody errorResponse
		decodeJSON(t, res, &errBody)
		t.Fatalf("createItem(%q) status = %d, error = %+v", name, res.StatusCode, errBody)
	}
	var created itemDTO
	decodeJSON(t, res, &created)
	return created
}

func move(t *testing.T, srv *httptest.Server, id int64, location string) itemDTO {
	t.Helper()
	res := doJSON(t, http.MethodPatch, srv.URL+"/api/v1/items/"+strconv.FormatInt(id, 10), map[string]string{"location": location})
	if res.StatusCode != http.StatusOK {
		var errBody errorResponse
		decodeJSON(t, res, &errBody)
		t.Fatalf("move(%d, %q) status = %d, error = %+v", id, location, res.StatusCode, errBody)
	}
	var updated itemDTO
	decodeJSON(t, res, &updated)
	return updated
}

func listItems(t *testing.T, srv *httptest.Server) []itemDTO {
	t.Helper()
	res, err := http.Get(srv.URL + "/api/v1/items")
	if err != nil {
		t.Fatalf("GET /items: %v", err)
	}
	var list itemsListResponse
	decodeJSON(t, res, &list)
	return list.Items
}
