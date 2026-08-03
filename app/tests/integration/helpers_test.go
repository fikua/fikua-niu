// Package integration holds medium (integration) tests: real SQLite
// database (temp file per test), real httpapi router, httptest.Server —
// no mocks (qa-engineer test pyramid, requirements.md §6).
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"testing/fstest"

	"golang.org/x/crypto/bcrypt"

	niu "niu"
	"niu/internal/auth"
	"niu/internal/fetchsafe"
	"niu/internal/httpapi"
	"niu/internal/ideas"
	"niu/internal/items"
	"niu/internal/projects"
	"niu/internal/store"
)

const (
	seedUserAID = 1
	seedUserBID = 2
)

// Fixture-only test credentials (S11/NFR-09 — never a real
// name/password/hash, generated fresh for the test suite). usernameA/B
// mirror migration 002's seeded rows (usuari_a/usuari_b); the plaintext
// passwords below are only ever compared in-memory during a test run.
const (
	testUsernameA = "usuari_a"
	testPasswordA = "correct-horse-battery-staple-a"
	testUsernameB = "usuari_b"
	testPasswordB = "correct-horse-battery-staple-b"
)

// testServer bundles an httptest.Server backed by a fresh temporary
// SQLite database with migrations applied, using the given fixed user id
// for the auth stub (so tests can simulate "logged in as A" or "as B").
type testServer struct {
	*httptest.Server
	Store *store.Store
}

// newIdeasService wires a real ideas.Service against st, using the real
// fetchsafe.FetchPreview + a bounded worker pool — the same wiring as
// cmd/niu/main.go's production path, so integration tests exercise the
// genuine SSRF mitigation, not a stub. Callers that need to intercept the
// fetch step for a specific unit-level scenario use ideas.NewService
// directly instead (see internal/ideas/service_test.go).
func newIdeasService(t *testing.T, st *store.Store) *ideas.Service {
	t.Helper()
	ideasRepo := store.NewIdeasRepository(st.DB)
	client := fetchsafe.NewClient()
	pool := ideas.NewWorkerPool(context.Background())
	t.Cleanup(pool.Close)
	fetch := func(ctx context.Context, rawURL string) (ideas.Preview, error) {
		preview, err := fetchsafe.FetchPreview(ctx, client, rawURL)
		return ideas.Preview{
			Title:       preview.Title,
			ImageURL:    preview.ImageURL,
			Description: preview.Description,
			Partial:     preview.Partial,
		}, err
	}
	return ideas.NewService(ideasRepo, ideasRepo, fetch, pool)
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
	projectsRepo := store.NewProjectsRepository(st.DB)
	projectsSvc := projects.NewService(projectsRepo, projectsRepo)
	ideasSvc := newIdeasService(t, st)
	authenticator := auth.StubAuthenticator{UserID: userID}

	var emptyFS fs.FS = fstest.MapFS{}
	router := httpapi.NewRouter(svc, projectsSvc, ideasSvc, st, authenticator, emptyFS, true)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testServer{Server: srv, Store: st}
}

// authTestServer bundles an httptest.Server wired with a real
// auth.PasswordAuthenticator (not the NIU-1 stub) — used by every NIU-4
// test (T-26 to T-35). Both seeded users (usuari_a/usuari_b) have their
// password_hash overwritten with a bcrypt hash of a fixture-only
// plaintext (S11/NFR-09: not a real credential).
type authTestServer struct {
	*httptest.Server
	Store         *store.Store
	SessionSecret string
}

func newAuthTestServer(t *testing.T) *authTestServer {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "niu.db")

	st, err := store.Open(dbPath, niu.MigrationsFS)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	hashA, err := bcrypt.GenerateFromPassword([]byte(testPasswordA), 12)
	if err != nil {
		t.Fatalf("bcrypt hash A: %v", err)
	}
	hashB, err := bcrypt.GenerateFromPassword([]byte(testPasswordB), 12)
	if err != nil {
		t.Fatalf("bcrypt hash B: %v", err)
	}
	if _, err := st.DB.Exec(`UPDATE users SET password_hash = ? WHERE name = ?`, string(hashA), testUsernameA); err != nil {
		t.Fatalf("seed password hash A: %v", err)
	}
	if _, err := st.DB.Exec(`UPDATE users SET password_hash = ? WHERE name = ?`, string(hashB), testUsernameB); err != nil {
		t.Fatalf("seed password hash B: %v", err)
	}

	// Fixture-only session secret (not a real deployment secret) — must
	// satisfy the same >=32-byte floor config.Load() enforces.
	const sessionSecret = "test-only-session-secret-32bytes!!"

	repo := store.NewItemsRepository(st.DB)
	svc := items.NewService(repo, repo, repo)
	projectsRepo := store.NewProjectsRepository(st.DB)
	projectsSvc := projects.NewService(projectsRepo, projectsRepo)
	ideasSvc := newIdeasService(t, st)

	authenticator, err := auth.NewPasswordAuthenticator(st.DB, sessionSecret)
	if err != nil {
		t.Fatalf("auth.NewPasswordAuthenticator: %v", err)
	}

	var emptyFS fs.FS = fstest.MapFS{}
	router := httpapi.NewRouter(svc, projectsSvc, ideasSvc, st, authenticator, emptyFS, true)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &authTestServer{Server: srv, Store: st, SessionSecret: sessionSecret}
}

// loginResult captures everything a test typically needs after a
// successful login: the raw plaintext session token (for S2c/manual
// cookie manipulation), the CSRF token, and an *http.Client whose cookie
// jar already carries both cookies (for tests that just want to act as a
// logged-in user without re-handling cookies by hand).
type loginResult struct {
	SessionToken string
	CSRFToken    string
	Client       *http.Client
	StatusCode   int
	Body         []byte
}

// doLogin issues POST /api/v1/auth/login with a cookie jar attached, so
// Set-Cookie is captured the same way a real browser would handle it.
func doLogin(t *testing.T, srv *authTestServer, username, password string) loginResult {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New: %v", err)
	}
	client := &http.Client{Jar: jar}

	b, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/auth/login", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("new login request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("do login request: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read login response body: %v", err)
	}

	result := loginResult{Client: client, StatusCode: res.StatusCode, Body: body}

	reqURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	for _, c := range jar.Cookies(reqURL) {
		switch c.Name {
		case auth.SessionCookieName:
			result.SessionToken = c.Value
		case httpapi.CSRFCookieName:
			result.CSRFToken = c.Value
		}
	}

	return result
}

// doJSONWithCookie issues a request carrying an explicit raw session
// cookie value (not via a cookie jar) — used by tests that need to send a
// specific, possibly-invalid, token (EC-01/S2b/S6/EC-04).
func doJSONWithCookie(t *testing.T, method, url, sessionToken, csrfToken string, body any) *http.Response {
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
	if sessionToken != "" {
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessionToken})
	}
	if csrfToken != "" {
		req.Header.Set("X-CSRF-Token", csrfToken)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return res
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

// jsonBody marshals v to JSON and wraps it as an io.ReadCloser, for tests
// that need to build an *http.Request by hand (e.g. to set a custom
// header like Cf-Connecting-Ip before sending).
func jsonBody(t *testing.T, v any) io.ReadCloser {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json body: %v", err)
	}
	return io.NopCloser(bytes.NewReader(b))
}

// createItemAuthenticated performs POST /api/v1/items using an already
// logged-in session's cookie/CSRF token (loginResult from doLogin) — used
// by CSRF tests that need at least one real item to move/delete.
func createItemAuthenticated(t *testing.T, srv *authTestServer, login loginResult, name string) itemDTO {
	t.Helper()
	res := doJSONWithCookie(t, http.MethodPost, srv.URL+"/api/v1/items", login.SessionToken, login.CSRFToken,
		map[string]string{"name": name})
	if res.StatusCode != http.StatusCreated {
		var errBody errorResponse
		decodeJSON(t, res, &errBody)
		t.Fatalf("createItemAuthenticated(%q) status = %d, error = %+v", name, res.StatusCode, errBody)
	}
	var created itemDTO
	decodeJSON(t, res, &created)
	return created
}

// listItemsAuthenticated performs GET /api/v1/items with a session cookie.
func listItemsAuthenticated(t *testing.T, srv *authTestServer, login loginResult) []itemDTO {
	t.Helper()
	res := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/items", login.SessionToken, "", nil)
	var list itemsListResponse
	decodeJSON(t, res, &list)
	return list.Items
}

// createProjectAuthenticated performs POST /api/v1/projects using an
// already logged-in session's cookie/CSRF token (loginResult from
// doLogin) — mirrors createItemAuthenticated, used by CSRF tests that need
// at least one real project to change state on / delete (F-21).
func createProjectAuthenticated(t *testing.T, srv *authTestServer, login loginResult, name string) projectDTO {
	t.Helper()
	res := doJSONWithCookie(t, http.MethodPost, srv.URL+"/api/v1/projects", login.SessionToken, login.CSRFToken,
		map[string]string{"name": name})
	if res.StatusCode != http.StatusCreated {
		var errBody errorResponse
		decodeJSON(t, res, &errBody)
		t.Fatalf("createProjectAuthenticated(%q) status = %d, error = %+v", name, res.StatusCode, errBody)
	}
	var created projectDTO
	decodeJSON(t, res, &created)
	return created
}

// listProjectsAuthenticated performs GET /api/v1/projects with a session
// cookie.
func listProjectsAuthenticated(t *testing.T, srv *authTestServer, login loginResult) []projectDTO {
	t.Helper()
	res := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/projects", login.SessionToken, "", nil)
	var list projectsListResponse
	decodeJSON(t, res, &list)
	return list.Projects
}

// createIdeaAuthenticated performs POST /api/v1/ideas using an already
// logged-in session's cookie/CSRF token (loginResult from doLogin) —
// mirrors createProjectAuthenticated, used by CSRF tests that need at
// least one real idea to delete (F-03).
func createIdeaAuthenticated(t *testing.T, srv *authTestServer, login loginResult, url string) ideaDTO {
	t.Helper()
	res := doJSONWithCookie(t, http.MethodPost, srv.URL+"/api/v1/ideas", login.SessionToken, login.CSRFToken,
		map[string]string{"url": url})
	if res.StatusCode != http.StatusCreated {
		var errBody errorResponse
		decodeJSON(t, res, &errBody)
		t.Fatalf("createIdeaAuthenticated(%q) status = %d, error = %+v", url, res.StatusCode, errBody)
	}
	var created ideaDTO
	decodeJSON(t, res, &created)
	return created
}

// listIdeasAuthenticated performs GET /api/v1/ideas with a session
// cookie.
func listIdeasAuthenticated(t *testing.T, srv *authTestServer, login loginResult) []ideaDTO {
	t.Helper()
	res := doJSONWithCookie(t, http.MethodGet, srv.URL+"/api/v1/ideas", login.SessionToken, "", nil)
	var list ideasListResponse
	decodeJSON(t, res, &list)
	return list.Ideas
}

// assertNoItemWithName fails the test if any item with the given name
// exists — used after an expected-to-be-rejected mutation to confirm it
// truly had no effect (AC-07/S1b: "no produeix cap efecte").
func assertNoItemWithName(t *testing.T, srv *authTestServer, login loginResult, name string) {
	t.Helper()
	for _, it := range listItemsAuthenticated(t, srv, login) {
		if it.Name == name {
			t.Fatalf("item %q exists despite the mutation that created it being rejected", name)
		}
	}
}

// readAllBytes reads a response body as raw bytes without closing it
// (callers already `defer res.Body.Close()`) — used by tests asserting
// byte-for-byte equality (AC-11/S5) rather than field-by-field.
func readAllBytes(t *testing.T, res *http.Response) []byte {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return b
}
