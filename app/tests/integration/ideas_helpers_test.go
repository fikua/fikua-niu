package integration

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// ---- shared ideas HTTP helpers (mirrors createItem/createProject) ----

func createIdea(t *testing.T, srv *httptest.Server, url string) ideaDTO {
	t.Helper()
	res := doJSON(t, http.MethodPost, srv.URL+"/api/v1/ideas", map[string]string{"url": url})
	if res.StatusCode != http.StatusCreated {
		var errBody errorResponse
		decodeJSON(t, res, &errBody)
		t.Fatalf("createIdea(%q) status = %d, error = %+v", url, res.StatusCode, errBody)
	}
	var created ideaDTO
	decodeJSON(t, res, &created)
	return created
}

func listIdeas(t *testing.T, srv *httptest.Server) []ideaDTO {
	t.Helper()
	res, err := http.Get(srv.URL + "/api/v1/ideas")
	if err != nil {
		t.Fatalf("GET /ideas: %v", err)
	}
	var list ideasListResponse
	decodeJSON(t, res, &list)
	return list.Ideas
}

func deleteIdeaHTTP(t *testing.T, srv *httptest.Server, id int64) *http.Response {
	t.Helper()
	return doJSON(t, http.MethodDelete, srv.URL+"/api/v1/ideas/"+strconv.FormatInt(id, 10), nil)
}
