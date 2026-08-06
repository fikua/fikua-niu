package integration

// migration_005_test.go closes the one gap /audit flagged as blocking for
// NIU-11 (qa-verification.md G-1): every other test in this suite builds
// its database by running ALL migrations at once against an empty file,
// so none of them exercises the scenario migration 005's central design
// decision actually exists for — a `projects` row that already existed
// *before* the column was added.
//
// That decision (tasks.md T-01) is that `preview_status` is NULL-able
// here, unlike activity_ideas' `NOT NULL DEFAULT 'pending'`. If it had
// been copied verbatim from the ideas table, every pre-existing project
// would have silently become 'pending' on deploy — a preview that is
// never coming, rendering as a permanent spinner in the UI. The test
// below is what proves the deployed migration does not do that.

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	niu "niu"
)

// openAt migrates a fresh database up to (and including) the given
// version, mirroring what store.Open does except for stopping partway —
// goose.UpTo is the only way to reconstruct "the schema as it was before
// this change shipped".
func openAt(t *testing.T, dbPath string, version int64) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(on)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	goose.SetBaseFS(niu.MigrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	// goose logs every applied migration to stdout otherwise, which buries
	// the actual test output.
	goose.SetLogger(goose.NopLogger())

	if err := goose.UpTo(db, "migrations", version); err != nil {
		t.Fatalf("goose up to %d: %v", version, err)
	}
	return db
}

// TestMigration005_PreExistingRow_KeepsNullPreviewStatus is the
// production-shaped scenario: a project created before NIU-11 shipped,
// then migrated. It must come out the other side with every new column
// NULL — most importantly preview_status, which must NOT be 'pending'.
func TestMigration005_PreExistingRow_KeepsNullPreviewStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "niu.db")

	// 1. The world as it was before this change: schema at migration 004.
	db := openAt(t, dbPath, 4)

	// Guard the premise: if 004 already had preview_status, this test
	// would be silently vacuous forever after.
	var hasColumn int
	if err := db.QueryRow(
		`SELECT count(*) FROM pragma_table_info('projects') WHERE name = 'preview_status'`,
	).Scan(&hasColumn); err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	if hasColumn != 0 {
		t.Fatalf("projects.preview_status already exists at migration 004 — this test no longer tests what it claims")
	}

	if _, err := db.Exec(
		`INSERT INTO projects (name, name_normalized, state, budget) VALUES (?, ?, ?, ?)`,
		"Sofà Kivik", "sofà kivik", "idea", "599 EUR",
	); err != nil {
		t.Fatalf("insert pre-migration project: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// 2. Deploy: migrate the same file the rest of the way up.
	db = openAt(t, dbPath, 5)
	defer db.Close()

	var (
		name          string
		budget        sql.NullString
		state         string
		url           sql.NullString
		title         sql.NullString
		imageURL      sql.NullString
		description   sql.NullString
		previewStatus sql.NullString
	)
	if err := db.QueryRow(
		`SELECT name, budget, state, url, title, image_url, description, preview_status FROM projects WHERE name_normalized = ?`,
		"sofà kivik",
	).Scan(&name, &budget, &state, &url, &title, &imageURL, &description, &previewStatus); err != nil {
		t.Fatalf("select migrated project: %v", err)
	}

	// The pre-existing data must survive untouched.
	if name != "Sofà Kivik" {
		t.Errorf("name = %q, want %q — migration altered existing data", name, "Sofà Kivik")
	}
	if !budget.Valid || budget.String != "599 EUR" {
		t.Errorf("budget = %v, want %q — migration altered existing data", budget, "599 EUR")
	}
	if state != "idea" {
		t.Errorf("state = %q, want %q — migration altered existing data", state, "idea")
	}

	// The whole point of T-01: NULL, never 'pending'.
	if previewStatus.Valid {
		t.Errorf("preview_status = %q, want NULL — a project that predates NIU-11 has no preview pending, and 'pending' would render an eternal spinner", previewStatus.String)
	}
	for _, c := range []struct {
		name string
		val  sql.NullString
	}{
		{"url", url},
		{"title", title},
		{"image_url", imageURL},
		{"description", description},
	} {
		if c.val.Valid {
			t.Errorf("%s = %q, want NULL for a pre-existing row", c.name, c.val.String)
		}
	}
}

// TestMigration005_DownUp_RoundTrip proves the Down migration is real
// (the project has no goose Down coverage at all — qa-verification.md
// B-09) and that re-applying Up afterwards works, which is what a
// rollback-then-redeploy actually does.
func TestMigration005_DownUp_RoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "niu.db")
	db := openAt(t, dbPath, 5)
	defer db.Close()

	if _, err := db.Exec(
		`INSERT INTO projects (name, name_normalized, state, url, preview_status) VALUES (?, ?, ?, ?, ?)`,
		"Televisor", "televisor", "decidit", "https://example.com/tv", "ready",
	); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	if err := goose.Down(db, "migrations"); err != nil {
		t.Fatalf("goose down: %v", err)
	}

	// The row survives the rollback — only the columns go away.
	var name, state string
	if err := db.QueryRow(
		`SELECT name, state FROM projects WHERE name_normalized = ?`, "televisor",
	).Scan(&name, &state); err != nil {
		t.Fatalf("select after down: %v", err)
	}
	if name != "Televisor" || state != "decidit" {
		t.Errorf("after Down: name=%q state=%q, want Televisor/decidit — rollback destroyed data", name, state)
	}

	// The unique index from migration 003 must survive too: SQLite's
	// ALTER TABLE DROP COLUMN can rebuild a table, and losing this index
	// would silently allow duplicate project names.
	var indexCount int
	if err := db.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type='index' AND name='idx_projects_name_normalized'`,
	).Scan(&indexCount); err != nil {
		t.Fatalf("check index: %v", err)
	}
	if indexCount != 1 {
		t.Error("idx_projects_name_normalized did not survive the Down migration — duplicate project names would become possible")
	}

	if err := goose.UpTo(db, "migrations", 5); err != nil {
		t.Fatalf("goose up again after down: %v", err)
	}

	// And the columns are back, still NULL for the row that predates them.
	var previewStatus sql.NullString
	if err := db.QueryRow(
		`SELECT preview_status FROM projects WHERE name_normalized = ?`, "televisor",
	).Scan(&previewStatus); err != nil {
		t.Fatalf("select after re-up: %v", err)
	}
	if previewStatus.Valid {
		t.Errorf("preview_status = %q after Down+Up, want NULL", previewStatus.String)
	}
}
