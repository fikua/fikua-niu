package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestNoSprintfSQL is the static check recommended by design.md §9/tasks.md
// T-28 (S8/NFR-03): grep internal/store/ for fmt.Sprintf calls that build
// SELECT/INSERT/UPDATE/DELETE statements. All queries must use bound (?)
// parameters, never string formatting into SQL.
func TestNoSprintfSQL(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	// tests/integration/sql_static_test.go -> app/internal/store
	storeDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "internal", "store")

	pattern := regexp.MustCompile(`(?i)fmt\.Sprintf\([^)]*\b(SELECT|INSERT|UPDATE|DELETE)\b`)

	entries, err := os.ReadDir(storeDir)
	if err != nil {
		t.Fatalf("read internal/store: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(storeDir, entry.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if pattern.Match(body) {
			t.Errorf("%s: found fmt.Sprintf building a SQL statement — use bound parameters (?) instead", entry.Name())
		}
	}
}
