package integration

import (
	"net/http"
	"strconv"
	"testing"
)

// T-30 — AC-06/AC-07/EC-03/S1b: CSRF double-submit guard on the mutation
// surface, explicitly exercised against POST /api/v1/items (create) and at
// least one PATCH/DELETE (move/delete) — the exact routes retrofitted onto
// NIU-1's already-shipped handlers.

func TestCSRF_ValidToken_MutationSucceeds(t *testing.T) {
	srv := newAuthTestServer(t)
	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}

	res := doJSONWithCookie(t, http.MethodPost, srv.URL+"/api/v1/items", login.SessionToken, login.CSRFToken,
		map[string]string{"name": "Llet CSRF vàlid"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /items with a valid CSRF token status = %d, want 201", res.StatusCode)
	}
}

func TestCSRF_MissingToken_MutationRejected(t *testing.T) {
	srv := newAuthTestServer(t)
	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}

	res := doJSONWithCookie(t, http.MethodPost, srv.URL+"/api/v1/items", login.SessionToken, "",
		map[string]string{"name": "No hauria d'existir"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("POST /items without a CSRF token status = %d, want 403", res.StatusCode)
	}

	assertNoItemWithName(t, srv, login, "No hauria d'existir")
}

func TestCSRF_ArbitraryToken_MutationRejected(t *testing.T) {
	srv := newAuthTestServer(t)
	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}

	res := doJSONWithCookie(t, http.MethodPost, srv.URL+"/api/v1/items", login.SessionToken, "arbitrary-not-issued-token",
		map[string]string{"name": "Tampoc hauria d'existir"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("POST /items with an arbitrary CSRF token status = %d, want 403", res.StatusCode)
	}

	assertNoItemWithName(t, srv, login, "Tampoc hauria d'existir")
}

// EC-03: a value that looks plausible (well-formed base64) but was not
// issued by the server for this session must fail exactly like any other
// mismatch — the CSRF token is bound to the session's token_hash, not a
// guessable/global secret.
func TestCSRF_PlausibleButUnissuedToken_Rejected(t *testing.T) {
	srv := newAuthTestServer(t)
	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}

	// A second, independent login (same user) yields a CSRF token bound to
	// a DIFFERENT session — well-formed, but not valid for the first
	// session's requests.
	otherLogin := doLogin(t, srv, testUsernameA, testPasswordA)
	if otherLogin.StatusCode != http.StatusOK {
		t.Fatalf("second login status = %d, want 200", otherLogin.StatusCode)
	}
	if otherLogin.CSRFToken == login.CSRFToken {
		t.Fatal("two independent logins produced the same CSRF token — cannot exercise EC-03")
	}

	res := doJSONWithCookie(t, http.MethodPost, srv.URL+"/api/v1/items", login.SessionToken, otherLogin.CSRFToken,
		map[string]string{"name": "CSRF d'una altra sessió"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("POST /items with another session's CSRF token status = %d, want 403 (EC-03)", res.StatusCode)
	}
}

// Retrofit coverage on PATCH/DELETE, not just POST — items_handlers.go
// was shipped in NIU-1 without any CSRF check.
func TestCSRF_PatchWithoutToken_Rejected(t *testing.T) {
	srv := newAuthTestServer(t)
	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}

	created := createItemAuthenticated(t, srv, login, "Ítem per moure")

	res := doJSONWithCookie(t, http.MethodPatch, srv.URL+"/api/v1/items/"+strconv.FormatInt(created.ID, 10), login.SessionToken, "",
		map[string]string{"location": "pantry"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("PATCH without CSRF token status = %d, want 403", res.StatusCode)
	}

	// Confirm no effect: still in the original location.
	items := listItemsAuthenticated(t, srv, login)
	for _, it := range items {
		if it.ID == created.ID && it.Location != "shopping" {
			t.Fatalf("item moved despite missing CSRF token: %+v", it)
		}
	}
}

func TestCSRF_DeleteWithoutToken_Rejected(t *testing.T) {
	srv := newAuthTestServer(t)
	login := doLogin(t, srv, testUsernameA, testPasswordA)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}

	created := createItemAuthenticated(t, srv, login, "Ítem per eliminar")

	res := doJSONWithCookie(t, http.MethodDelete, srv.URL+"/api/v1/items/"+strconv.FormatInt(created.ID, 10), login.SessionToken, "", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("DELETE without CSRF token status = %d, want 403", res.StatusCode)
	}

	items := listItemsAuthenticated(t, srv, login)
	found := false
	for _, it := range items {
		if it.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("item disappeared despite missing CSRF token on DELETE")
	}
}
