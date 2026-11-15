// Command sqlite2pg copies a Niu SQLite database into the PostgreSQL schema
// (one-off, for the move to the platform cluster).
//
//	sqlite2pg -sqlite /data/niu.db -pg "$NIU_DATABASE_URL"
//
// Run it with the application stopped. It applies the schema migrations,
// refuses to touch a target that already has data (the two seeded users are the
// exception: they are overwritten with the real rows), copies every table in
// one transaction, resets the identity counters and compares the row counts
// before committing. Nothing is deleted from the SQLite file.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	niu "niu"
	"niu/internal/store"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// tables in foreign-key order. users is special: migration 002 already seeded
// ids 1 and 2 in the target, so its rows are upserted.
var tables = []struct {
	name   string
	upsert bool
}{
	{"users", true},
	{"sessions", false},
	{"items", false},
	{"events", false},
	{"projects", false},
	{"activity_ideas", false},
}

func main() {
	sqlitePath := flag.String("sqlite", "", "path of the SQLite database (the app stopped)")
	pgURL := flag.String("pg", os.Getenv("NIU_DATABASE_URL"), "PostgreSQL URL (default $NIU_DATABASE_URL)")
	flag.Parse()
	if *sqlitePath == "" || *pgURL == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(context.Background(), *sqlitePath, *pgURL); err != nil {
		log.Fatalf("sqlite2pg: %v", err)
	}
}

func run(ctx context.Context, sqlitePath, pgURL string) error {
	if _, err := os.Stat(sqlitePath); err != nil {
		return err
	}
	src, err := sql.Open("sqlite", "file:"+sqlitePath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer src.Close()

	// Applies the schema migrations.
	st, err := store.Open(pgURL, niu.MigrationsFS)
	if err != nil {
		return err
	}
	dst := st.DB
	defer st.Close()

	for _, t := range tables {
		if t.upsert {
			continue
		}
		var n int
		if err := dst.QueryRowContext(ctx, "SELECT count(*) FROM "+t.name).Scan(&n); err != nil {
			return fmt.Errorf("count target %s: %w", t.name, err)
		}
		if n != 0 {
			return fmt.Errorf("target table %s already has %d rows: refusing to copy over it", t.name, n)
		}
	}

	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, t := range tables {
		copied, err := copyTable(ctx, src, tx, t.name, t.upsert)
		if err != nil {
			return fmt.Errorf("%s: %w", t.name, err)
		}
		var want, got int
		if err := src.QueryRowContext(ctx, "SELECT count(*) FROM "+t.name).Scan(&want); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+t.name).Scan(&got); err != nil {
			return err
		}
		if want != copied || got != want {
			return fmt.Errorf("%s: sqlite has %d rows, copied %d, postgres has %d", t.name, want, copied, got)
		}
		if t.name != "sessions" { // sessions has no identity column
			if _, err := tx.ExecContext(ctx, fmt.Sprintf(
				`SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE(MAX(id), 1), MAX(id) IS NOT NULL) FROM %s`, t.name, t.name)); err != nil {
				return fmt.Errorf("reset identity of %s: %w", t.name, err)
			}
		}
		log.Printf("%-16s %6d rows", t.name, copied)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	log.Print("done: every table copied and counted")
	return nil
}

func copyTable(ctx context.Context, src *sql.DB, tx *sql.Tx, table string, upsert bool) (int, error) {
	rows, err := src.QueryContext(ctx, "SELECT * FROM "+table+" ORDER BY rowid")
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	marks := make([]string, len(cols))
	sets := make([]string, 0, len(cols))
	for i, c := range cols {
		marks[i] = fmt.Sprintf("$%d", i+1)
		if c != "id" {
			sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
		}
	}
	insert := fmt.Sprintf("INSERT INTO %s (%s) OVERRIDING SYSTEM VALUE VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(marks, ", "))
	if upsert {
		insert += " ON CONFLICT (id) DO UPDATE SET " + strings.Join(sets, ", ")
	}
	stmt, err := tx.PrepareContext(ctx, insert)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	n := 0
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return n, err
		}
		for i, v := range vals {
			if b, ok := v.([]byte); ok {
				vals[i] = string(b)
			}
		}
		if _, err := stmt.ExecContext(ctx, vals...); err != nil {
			return n, fmt.Errorf("row %d: %w", n+1, err)
		}
		n++
	}
	return n, rows.Err()
}
