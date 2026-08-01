// Command killtest implements ADR-04 (design.md §3): a standalone Go
// program (not `go test`) that repeatedly SIGKILLs the real `niu` binary
// mid-write and verifies SQLite integrity + a clean restart afterwards
// (EC-15/REL-01, NFR-07).
//
// This is a manual, mandatory procedure — it does NOT run in CI by
// default (see design.md ADR-04 for the reasoning: real SIGKILL timing
// is non-deterministic and fragile inside shared CI runners). Run it
// with:
//
//	make killtest N=10
//
// Expected result: after N iterations, every run prints
// "PRAGMA integrity_check: ok" and "healthz after restart: 200", and the
// program exits 0. Any other outcome is a REL-01/NFR-07 regression.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	n := flag.Int("n", 10, "number of kill iterations to run")
	flag.Parse()

	if err := run(*n); err != nil {
		fmt.Fprintln(os.Stderr, "killtest: FAILED:", err)
		os.Exit(1)
	}
	fmt.Printf("killtest: PASSED all %d iterations\n", *n)
}

func run(n int) error {
	dir, err := os.MkdirTemp("", "niu-killtest-")
	if err != nil {
		return fmt.Errorf("mkdtemp: %w", err)
	}
	defer os.RemoveAll(dir)

	dbPath := filepath.Join(dir, "niu.db")
	binPath := filepath.Join(dir, "niu-killtest-bin")

	fmt.Println("killtest: building niu binary...")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/niu")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	const port = "18099"

	for i := 1; i <= n; i++ {
		fmt.Printf("killtest: iteration %d/%d\n", i, n)
		if err := iteration(binPath, dbPath, port, i); err != nil {
			return fmt.Errorf("iteration %d: %w", i, err)
		}
	}
	return nil
}

func iteration(binPath, dbPath, port string, iterationNum int) error {
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"NIU_DB_PATH="+dbPath,
		"NIU_PORT="+port,
		"NIU_ENV=development",
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start niu: %w", err)
	}

	if err := waitForHealthz(port, 5*time.Second); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return fmt.Errorf("first startup did not become healthy: %w", err)
	}

	// Seed one item to move repeatedly. Each iteration reuses the same
	// persistent DB file (to genuinely exercise "restart against
	// existing data", not a fresh DB every time), so the name must be
	// unique per iteration — EC-06 correctly rejects a repeat.
	itemID, err := seedItem(port, iterationNum)
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return fmt.Errorf("seed item: %w", err)
	}

	stop := make(chan struct{})
	done := make(chan struct{})
	go hammerMoves(port, itemID, stop, done)

	// Random 50-500ms delay before SIGKILL, per ADR-04.
	delay := time.Duration(50+rand.Intn(450)) * time.Millisecond
	time.Sleep(delay)

	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		close(stop)
		<-done
		return fmt.Errorf("SIGKILL: %w", err)
	}
	_, _ = cmd.Process.Wait()
	close(stop)
	<-done

	// Reopen the same DB file directly and check integrity.
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	var result string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		db.Close()
		return fmt.Errorf("integrity_check query: %w", err)
	}
	db.Close()
	if result != "ok" {
		return fmt.Errorf("PRAGMA integrity_check = %q, want ok", result)
	}
	fmt.Println("  PRAGMA integrity_check: ok")

	// Restart the binary and verify /healthz.
	cmd2 := exec.Command(binPath)
	cmd2.Env = append(os.Environ(),
		"NIU_DB_PATH="+dbPath,
		"NIU_PORT="+port,
		"NIU_ENV=development",
	)
	cmd2.Stdout = io.Discard
	cmd2.Stderr = io.Discard
	if err := cmd2.Start(); err != nil {
		return fmt.Errorf("restart niu: %w", err)
	}
	defer func() {
		_ = cmd2.Process.Kill()
		_, _ = cmd2.Process.Wait()
	}()

	if err := waitForHealthz(port, 5*time.Second); err != nil {
		return fmt.Errorf("restart did not become healthy: %w", err)
	}
	fmt.Println("  healthz after restart: 200")

	return nil
}

func waitForHealthz(port string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := "http://localhost:" + port + "/healthz"
	for time.Now().Before(deadline) {
		res, err := http.Get(url)
		if err == nil {
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for /healthz on port %s", port)
}

func seedItem(port string, iterationNum int) (int64, error) {
	name := fmt.Sprintf("Killtest item %d", iterationNum)
	res, err := http.Post(
		"http://localhost:"+port+"/api/v1/items",
		"application/json",
		jsonBody(fmt.Sprintf(`{"name":%q}`, name)),
	)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("seed item status = %d", res.StatusCode)
	}
	var body struct {
		ID int64 `json:"id"`
	}
	if err := decodeJSON(res.Body, &body); err != nil {
		return 0, err
	}
	return body.ID, nil
}

// hammerMoves sends PATCH requests continuously until stop is closed,
// alternating the item's location — this is the "write in progress" that
// SIGKILL is meant to interrupt (ADR-04).
func hammerMoves(port string, itemID int64, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	locations := []string{"pantry", "shopping"}
	i := 0
	for {
		select {
		case <-stop:
			return
		default:
		}
		loc := locations[i%2]
		i++
		req, err := http.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("http://localhost:%s/api/v1/items/%d", port, itemID),
			jsonBody(fmt.Sprintf(`{"location":%q}`, loc)),
		)
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err == nil {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}
	}
}
