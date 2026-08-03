---
artefact: review (ad-hoc, no SDD artefacts — see note below)
branch: niu-spa-conversion
compared_to: main
verdict: "CHANGES_REQUESTED"
findings_count: 4
blocking_count: 0
major_count: 1
minor_count: 2
nit_count: 1
reviewer: code-reviewer
created: "2026-08-03"
---

# Ad-hoc review — SPA conversion (`niu-spa-conversion` vs `main`)

## Context note

This change was made outside the `/capture → /define → /code` pipeline at
the user's explicit request (speed over ceremony for this fix). There is
no `requirements.md`/`design.md`/`tasks.md` to audit against, so this
review is a straight code-quality + correctness + security-adjacent audit
of the diff, judged against `PLAN.md` (binding architecture/security/look
& feel) and the standard this project has set for itself in
`docs/changes/NIU-5-*/review.md`.

## Verdict

**CHANGES_REQUESTED** — one `major` correctness/robustness defect (F-01,
the SPA fallback silently converts any missing-asset 404 into a 200 soft
navigation, with zero test coverage for the new routing logic), plus two
`minor` findings. Nothing here is a security hole and nothing regresses
functional parity — the fixes are small and don't touch approved scope.
`go build`, `go vet`, `gofmt -l .`, and `go test ./...` are all clean.

## What was verified as correct (worth stating explicitly)

- **`GET /api/v1/me` fires exactly once per page load**, verified by
  reading `app/web/js/main.js`'s `main()`: `api.getMe()` is called once;
  `initShoppingView`/`initProjectsView` receive the resolved `me` object
  and never call `getMe()` again. `api.js` is untouched by this diff (not
  in `git diff --stat`), so there's no hidden internal caching to distrust
  — this is a genuine fix of the "two pages, two `/me` calls" problem the
  commit messages describe, not a self-report.
- **History/routing correctness for direct URL entry and refresh** is
  handled correctly. `router.go`'s `spaFallback` serves `index.html`'s
  bytes in place (via `http.ServeContent`, not a redirect) for any
  GET/HEAD request whose path doesn't resolve to a real file in the
  embedded `webFS` — confirmed this is not just commented intent but
  actual behavior (see F-01 below for the price of that permissiveness).
  Client-side `popstate` handling and back-button behavior are wired
  correctly in `main.js`'s `wireRouter()`. All 4 updated E2E specs use
  `page.goto('/projects')` (a real HTTP navigation, not a client
  transition), so the fallback path is genuinely exercised by the test
  suite on every run — not just the "click a nav link" happy path.
- **CSP compliance confirmed.** Zero occurrences of `innerHTML`,
  `outerHTML`, `insertAdjacentHTML`, `document.write`, `eval(`, or
  `new Function(` in `main.js`, `shopping-view.js`, `projects-view.js`, or
  `index.html`. `SecurityHeaders` remains the outermost middleware
  (`router.go:67`) and, because chi applies router-level `Use()`
  middleware to `NotFound` handlers too, the SPA fallback responses still
  carry the full header set (CSP, HSTS, nosniff, etc.) — verified this is
  not a gap.
- **Functional parity confirmed by diffing old `main.js`/`projects-main.js`
  against the new `shopping-view.js`/`projects-view.js` line by line.**
  Add-form validation, counter/aria-live threshold, Enter-to-submit,
  Escape-to-dismiss-toast, `wireTabs`/`setActivePanel`, `onLocaleChange`,
  the 10s poll + focus-refetch cycle — all preserved verbatim in the split
  files. FLIP animation and confetti live in `store.js`/`flip.js`/
  `confetti.js`, none of which are touched by this diff, so no regression
  there either. The mobile tabbar is correctly nested inside
  `<main data-view="shopping">` in the merged shell, so it's hidden
  automatically when the `/projects` view is active — no CSS rule
  overrides the native `hidden` attribute's `display: none`.
- No dead code from the merge: `projects.html`/`projects-main.js`/
  `router.js` are fully deleted, no leftover imports, no orphaned
  `[href$="projects.html"]`/`.projects-page` CSS selectors (the latter was
  actually already-dead on `main`, so its removal isn't a regression
  either way).

## Findings

### F-01 — SPA fallback treats every unmatched path as a client route, masking real 404s as soft navigations (major, correctness/robustness)

- **Location:** `app/internal/httpapi/router.go:161-183` (`spaFallback`)
- **Observation:** verified empirically (throwaway probe against the real
  `spaFallback` function with an `fstest.MapFS`): a request for a
  genuinely nonexistent static asset (e.g. a typo'd
  `/js/shoping-view.js`) returns **`200` with `index.html`'s bytes**, not
  a `404`. `spaFallback` only distinguishes "real file exists in `webFS`"
  from "doesn't exist" — it does not check the request against the app's
  own small, closed route table (`ROUTES = {'/': ..., '/projects': ...}`
  in `main.js`) before deciding to serve the shell. Any unmatched path —
  a renamed/moved JS module, a stale cached reference to a deleted asset,
  a bad link, a bot probing common paths — gets a `200` HTML response
  instead of a clear `404`.
- **Why it matters:** this is the classic SPA anti-pattern the task
  description called out, and it is real here, not hypothetical. The
  browser's own module loader will still fail to *execute* an HTML
  response as JS (so it's not silently swallowed client-side — a console
  error appears), but the **server-side signal is gone**: access logs,
  uptime/synthetic checks, and any tooling that greps for `404` to catch
  broken deploys will see a `200` and miss it. Given `router.go`'s own
  route table is tiny and known (`/` and `/projects` today), there's no
  need for maximal permissiveness.
- **Zero test coverage for `spaFallback` itself.** `router_test.go` has no
  test exercising this function at all — not the missing-asset case, not
  the "known client route with no file" case, not method handling
  (POST/PUT falling through to `fileServer.ServeHTTP` unchanged). The
  E2E specs exercise the *happy path* of the fallback (`/projects`
  resolves to the shell) but nothing asserts the negative case (a
  missing asset 404s).
- **Suggested fix:** have `spaFallback` check the request path against an
  explicit, small allowlist of known client-side routes (mirroring
  `ROUTES` in `main.js` — could be passed in or duplicated server-side
  as a `[]string{"/", "/projects"}`) and only serve the shell for paths
  in that list (or paths with no file extension, as a looser heuristic);
  fall through to a real `404` for everything else. Add a
  `router_test.go` case asserting a nonexistent path with a file-like
  shape (e.g. `/js/does-not-exist.js`) returns `404`, and one asserting
  a known client route with no matching file still returns the shell.

### F-02 — Cross-view toast interference: two independent `toastTimer`s now share one DOM node (minor, correctness)

- **Location:** `app/web/js/render.js:239-278` and
  `app/web/js/projects-render.js:171-210`
- **Observation:** both modules independently manage `#toast-wrap` — the
  *same* DOM element in the merged single-page shell — but each keeps its
  own module-level `toastTimer` variable and its own `showToast`/
  `dismissToast` pair. Before this merge this was harmless: `index.html`
  and `projects.html` were separate pages, each with its own isolated
  `#toast-wrap`. Now that both views' JS lives in the same page (and both
  keep polling/mutating in the background regardless of which view is
  visible, per `main.js`'s own stated design), it's possible for the
  shopping view to show a toast, then the projects view's independent
  poll/error path to call its own `showToast()` shortly after — which
  calls `wrap.replaceChildren()` (wiping the shopping toast's DOM) and
  starts its *own* `toastTimer`, while the shopping view's original
  `toastTimer` is still live and will later fire `dismissToast()` against
  what is now the *other* view's toast. Net effect: a toast can be
  dismissed early, or a stale timer can clear a toast that isn't the one
  it was scheduled for.
- **Why it matters:** low-severity in practice (it's a background-error
  UI glitch, not a data or security issue, and both users interacting
  with error-triggering conditions on both views within a 5s window is an
  edge case) — but it's a genuine new class of bug introduced specifically
  by the SPA merge (both view modules assuming exclusive ownership of a
  DOM node that's no longer exclusively theirs), and it's exactly the
  kind of thing this task asked to check for.
- **Suggested fix:** extract the toast module into a single shared
  `toast.js` used by both views (there is already a
  "shared visual pattern with render.js" comment in
  `projects-render.js` acknowledging the duplication — this is the
  natural place to actually share the implementation, not just the CSS
  pattern), so there's one `toastTimer` and one source of truth for
  `#toast-wrap`.

### F-03 — No E2E assertion of the SPA fallback's negative case (minor, testing)

- **Location:** `app/tests/e2e/specs/*` (none)
- **Observation:** related to F-01 — while the positive path (`/projects`
  resolves correctly) is well covered by the updated specs, there's no
  test (Go or Playwright) asserting that a request for a genuinely
  missing static asset still 404s. This is the same gap called out at
  the unit level in F-01, restated here because it's also an E2E-level
  testing gap relevant to `qa-engineer`'s usual coverage expectations for
  this project.
- **Suggested fix:** covered by the `router_test.go` addition suggested
  under F-01; a dedicated E2E case is optional given the Go-level test
  would already pin the behavior.

### F-04 — `spaFallback`/`serveIndex` duplicate `index.html`-serving logic that already existed once in `http.FileServer`'s default behavior (nit, maintainability)

- **Location:** `app/internal/httpapi/router.go:185-207` (`serveIndex`)
- **Observation:** the doc comment on `serveIndex` correctly explains
  *why* it can't just delegate to `fileServer` with a rewritten path
  (chi/`http.FileServer` would 301-redirect any request resolving to a
  file named `index.html` back to `/`, which is wrong for `/projects`).
  This is a real constraint, not an oversight — noted as a nit only
  because the resulting ~25 lines of manual `fs.Open`/`Stat`/
  `io.ReadSeeker`-assertion/`http.ServeContent` plumbing is exactly the
  kind of code that's easy to get subtly wrong (e.g. it would panic-free
  but silently 500 if `webFS`'s `Open` ever returned something that isn't
  an `io.ReadSeeker` — currently unreachable since `embed.FS` always
  satisfies it, but worth a one-line comment noting that invariant so a
  future change to how `webFS` is constructed doesn't quietly reopen this
  path).
- **Why it matters:** cosmetic/documentation only — no functional gap
  today.
- **Suggested fix:** optional — add a one-line comment on the
  `io.ReadSeeker` type assertion noting it's guaranteed by `embed.FS` and
  would need re-verification if `webFS`'s construction ever changes.

## Build/test gate results

```
cd app && go build ./...   → clean
cd app && go vet ./...     → clean
cd app && gofmt -l .       → clean (no files listed)
cd app && go test ./...    → all packages ok
```

No failing test, no typecheck/vet/lint issue, no secret found in the
diff.

## Action items

1. **F-01 (major)** — scope `spaFallback` to the app's known client-side
   route set (or a file-extension heuristic) instead of treating every
   unmatched path as a route; add `router_test.go` coverage for both the
   missing-asset-404 case and the known-route-no-file case.
2. **F-02 (minor)** — extract a shared `toast.js` module so
   `shopping-view.js` and `projects-view.js` don't independently own the
   same `#toast-wrap` DOM node with separate timers.
3. **F-03 (minor)** — optional E2E case for the fallback's negative path,
   likely subsumed by the Go-level test from item 1.
4. **F-04 (nit)** — optional comment noting the `io.ReadSeeker` assertion's
   `embed.FS` invariant.

None of these block merging to `main` in the sense of "the app is broken"
— the app works correctly for every scenario a real user hits. F-01 is
flagged `major` rather than `blocking` because the risk is operational
(masked signal in logs/monitoring, not a functional or security defect
for end users) and is cheap to fix without touching approved scope.
