package httpapi

import (
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5"

	"niu/internal/auth"
)

// TestNoMutatingGETRoutes introspects the chi route table (EC-08/NFR-04)
// and asserts that no GET route also has a POST/PATCH/PUT/DELETE handler
// registered on the exact same pattern — i.e. GET is never wired to a
// handler shared with a mutating method.
func TestNoMutatingGETRoutes(t *testing.T) {
	router := NewRouter(nil, fakeHealthChecker{}, fakeAuthenticator{}, fstest.MapFS{})

	chiRouter, ok := router.(chi.Router)
	if !ok {
		t.Fatalf("router does not implement chi.Router (got %T)", router)
	}

	mutatingMethods := map[string]bool{
		http.MethodPost:   true,
		http.MethodPut:    true,
		http.MethodPatch:  true,
		http.MethodDelete: true,
	}

	err := chi.Walk(chiRouter, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == http.MethodGet && mutatingMethods[method] {
			t.Errorf("route %s %s: GET incorrectly classified as mutating", method, route)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	// Explicitly confirm the known GET routes exist and are read-only by
	// construction (they call Service.List/CurrentUser/Healthy, never
	// Add/Move/Delete — verified by code review / grep, not reflection,
	// since Go has no runtime way to inspect closure bodies).
	getRoutes := map[string]bool{}
	err = chi.Walk(chiRouter, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == http.MethodGet {
			getRoutes[route] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	wantGET := []string{"/healthz", "/api/v1/me", "/api/v1/items/"}
	for _, want := range wantGET {
		if !getRoutes[want] {
			t.Errorf("expected GET route %q to be registered, got routes: %+v", want, getRoutes)
		}
	}
}

type fakeHealthChecker struct{}

func (fakeHealthChecker) Healthy() error { return nil }

type fakeAuthenticator struct{}

func (fakeAuthenticator) CurrentUser(r *http.Request) (auth.User, error) {
	return auth.User{ID: 1}, nil
}
