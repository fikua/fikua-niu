// Package integration holds medium (integration) tests: real SQLite
// database (temp file per test), real httpapi router, httptest.Server —
// no mocks (qa-engineer test pyramid, requirements.md §6).
package integration

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"

	niu "niu"
	"niu/internal/auth"
	"niu/internal/httpapi"
	"niu/internal/items"
	"niu/internal/store"
)

const (
	seedUserAID = 1
	seedUserBID = 2
)

// testServer bundles an httptest.Server backed by a fresh temporary
// SQLite database with migrations applied, using the given fixed user id
// for the auth stub (so tests can simulate "logged in as A" or "as B").
type testServer struct {
	*httptest.Server
	Store *store.Store
}

func newTestServer(t *testing.T, userID int64) *testServer {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "niu.db")

	st, err := store.Open(dbPath, niu.MigrationsFS)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	repo := store.NewItemsRepository(st.DB)
	svc := items.NewService(repo, repo, repo)
	authenticator := auth.StubAuthenticator{UserID: userID}

	var emptyFS fs.FS = fstest.MapFS{}
	router := httpapi.NewRouter(svc, st, authenticator, emptyFS)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testServer{Server: srv, Store: st}
}

func doJSON(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return res
}

func decodeJSON(t *testing.T, res *http.Response, out any) {
	t.Helper()
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}
