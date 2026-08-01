# killtest — mid-write SIGKILL resilience (EC-15/REL-01, NFR-07)

> ADR-04 (`design.md` §3). This is a **manual, mandatory procedure**, not
> part of the automated CI suite. `/audit` blocks approval of NIU-1
> without recorded evidence of 10 successful runs.

## What it does

For each iteration:

1. Builds the real `niu` binary and starts it as a child process against
   a fresh temporary SQLite database.
2. Waits for `/healthz` to report healthy.
3. Seeds one item and starts a goroutine that sends `PATCH
   /api/v1/items/{id}` continuously, alternating `location` between
   `shopping` and `pantry`.
4. Waits a random interval between 50–500ms.
5. Sends `SIGKILL` to the child process — an abrupt, real kill, not a
   simulated one, equivalent to `docker kill`.
6. Reopens the same SQLite file directly and runs `PRAGMA
   integrity_check`, asserting the result is exactly `ok`.
7. Restarts the `niu` binary against the same database file and asserts
   `GET /healthz` returns `200`.

Any failure at any step aborts the whole run non-zero.

## How to run it

```bash
cd app
make killtest N=10
```

Or directly:

```bash
cd app
go run ./tests/killtest -n 10
```

## Expected output

For each of the `N` iterations:

```text
killtest: iteration <i>/<N>
  PRAGMA integrity_check: ok
  healthz after restart: 200
```

Followed by, at the very end:

```text
killtest: PASSED all <N> iterations
```

Any other output (a Go panic, a non-`ok` integrity_check result, a
timeout waiting for `/healthz`) is a **NFR-07 regression** — do not close
NIU-1 with a red killtest run.

## Recorded evidence (T-31 execution log)

Executed 2026-08-01, 10 consecutive iterations, on this development
machine (macOS, Go 1.25.7 toolchain via `go1.26.5`):

```text
$ make killtest N=10
killtest: building niu binary...
killtest: iteration 1/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: iteration 2/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: iteration 3/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: iteration 4/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: iteration 5/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: iteration 6/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: iteration 7/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: iteration 8/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: iteration 9/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: iteration 10/10
  PRAGMA integrity_check: ok
  healthz after restart: 200
killtest: PASSED all 10 iterations
```

No corruption detected in any of the 10 SIGKILL cycles — NFR-07
satisfied. Re-run this procedure any time `internal/store` or the
migration set changes.
