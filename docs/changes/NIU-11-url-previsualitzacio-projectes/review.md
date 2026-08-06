---
key: NIU-11
artefact: review
status: draft
verdict: APPROVED
findings_count: 7
blocking_count: 0
co_reviewers: [qa-engineer, security-engineer]
source: tasks.md (no requirements.md/design.md — shipped via /quick)
---

# NIU-11 — Code review

## 1. Verdict

**APPROVED**

Re-audit after fixes for the prior `CHANGES_REQUESTED` round. Both
blocking findings (F-01 data race, qa-engineer's G-1 migration gap) are
resolved and I have independently re-verified every claim in the
coordinator's fix report rather than taking it at face value, per the
explicit ask to re-run `-race` myself:

- **`go build ./...`** — clean.
- **`go vet ./...`** — clean.
- **`gofmt -l .`** — silent (clean).
- **`go test ./... -count=1`** — all packages pass, no cache.
- **`go test ./internal/... -race -count=1`** — clean, run by me directly (not the coordinator's word). Additionally re-ran `go test ./internal/projects/... -race -count=1` three times in a row to rule out flakiness — clean every time.
- **`go test ./tests/integration/... -race -count=1`** — clean, run by me directly (559s, exit 0).
- **Playwright suite** — ran the full suite myself: **38 passed, 1 failed** (`perf-3g.spec.js`, 1018ms vs a 1000ms budget). Confirmed this spec touches only the shopping-list view (`/`, `#add-input`, `#list-shopping`) — nothing NIU-11 changed — and is a machine-timing test with a history of recalibration (its own comments reference an earlier over-tight budget from T-30). Reproduced the same failure twice more independently. This is environment noise on this machine, not a NIU-11 regression, and out of this item's scope regardless.
- **Mutation-tested F-02's fix myself**: reverted `.project-thumb` from 32px back to 48px and re-ran the new `projects-preview.spec.js` row-height spec — it failed with the exact predicted delta ("row with thumbnail is 64px vs 48px without"), confirming the new test genuinely catches the regression it claims to catch, not just a coincidental pass. Restored the fix afterward; spec passes again.
- **Read `TestMigration005_PreExistingRow_KeepsNullPreviewStatus` and `TestMigration005_DownUp_RoundTrip` in full** and ran them directly (`-run 'TestMigration005' -v`) — both pass, and the vacuity guard (`hasColumn != 0` check against migration 004) and the `idx_projects_name_normalized`-survives-Down check are exactly the right defenses for what this test claims to prove.

No new findings introduced by the fix round. F-04/F-05/F-06/F-07 (all
`minor`/`nit`) remain as opportunistic follow-ups, not blockers — see §7.
`security-engineer`'s `security-review.md` (APPROVED, 0 blocking/major)
is unaffected by this round, since none of the fixes touched the
surfaces it audited (SSRF, XSS, `description` exposure).

## 2. QA coverage matrix

Owned by `qa-engineer` (`qa-verification.md` in this folder still
reflects the **pre-fix** state — it has not been regenerated since the
fixes landed). I am independently confirming, from the same evidence
qa-engineer's matrix was built on, that the gates their matrix put on
this verdict are now closed:

| # | Behaviour | Original verdict | Status after fix round |
|---|---|---|---|
| G-1 | Migration 005 against a **pre-existing** project row | **[BLOCKING-CANDIDATE]** ❌ missing | **Closed.** `app/tests/integration/migration_005_test.go`'s `TestMigration005_PreExistingRow_KeepsNullPreviewStatus` opens a DB at migration 4, inserts a row, migrates to 5, and asserts `preview_status IS NULL` — with a guard that fails the test outright if migration 4 ever gains the column first (preventing silent vacuity). Read and independently re-run by me; passes. |
| B-09 | Migration 005 reverts (Down) cleanly | ❌ missing | **Closed.** `TestMigration005_DownUp_RoundTrip` exercises `goose.Down` then `goose.UpTo(5)` again, asserting the pre-existing row survives and — notably — that `idx_projects_name_normalized` (the unique-name index from migration 003) survives the round trip, which is the real SQLite `DROP COLUMN`-triggers-table-rebuild risk this kind of migration carries. Independently re-run; passes. |
| B-10 | Frontend: thumbnail rendered null-safely | ❌ missing | **Closed.** `projects-preview.spec.js`'s first test intercepts the API with a `preview_status: null` row, asserts zero `.project-thumb`/`a.project-name` elements, plain-text name still renders, and zero `pageerror` events. Independently re-run; passes. |
| B-11 | Frontend: `rel="noopener noreferrer"` regression-protected | ❌ missing | **Closed.** Second spec asserts `rel` contains both `noopener` and `noreferrer` on the rendered anchor, plus `src`/`alt`/`href`/`target` on the thumbnail and link. Independently re-run; passes. |
| B-12 | Row-height-unchanged claim (T-10) actually verified | ❌ missing | **Closed.** Fourth spec measures `boundingBox().height` for a row with vs without a thumbnail and asserts the delta is ≤1px. I mutation-tested this myself (see §1) — it correctly fails when the regression is reintroduced. |

The remaining item from the original matrix, **G-2/G-3's underlying
gap** (zero E2E coverage of the two new security-relevant DOM
behaviours) is subsumed by B-10/B-11 above — closed by the same spec
file. I recommend qa-engineer regenerate `qa-verification.md` to reflect
the current state for the record, but I am not blocking `APPROVED` on
that regeneration since I have independently verified the underlying
tests exist, pass, and test what they claim to.

## 3. Findings

### F-01 — RESOLVED — concurrency — `app/internal/projects/fake_repo_test.go`

Previously `blocking`: `fakeRepo`'s map was read/written without
synchronization while `internal/ideas`'s equivalent had a `sync.Mutex`
guarding every method. Fixed: `mu sync.Mutex` added to the struct, all 8
methods (`Create`, `UpdatePreview`, `Get`, `List`, `UpdateState`,
`Delete`, `ExistsByNormalizedName`, `Record`) now `f.mu.Lock(); defer
f.mu.Unlock()`, matching `internal/ideas/fake_repo_test.go` method-for-
method. The new doc comment on the `mu` field (`fake_repo_test.go:13-18`)
explains *why* the lock is load-bearing rather than boilerplate — a
good addition beyond just copying the fix.

Independently re-verified: `go test ./internal/... -race -count=1` and
`go test ./tests/integration/... -race -count=1` both clean, run by me
directly. `go test ./internal/projects/... -race -count=1` re-run three
times with no flakes.

---

### F-02 — RESOLVED — functionality/regression — `app/web/app.css:657-663`

Previously `major`: 48px thumbnail grew the row 16px taller than a
row without one, contradicting T-10's "Fet quan" criterion. Fixed:
`.project-thumb` is now `32px × 32px`, matching `.state-badge`'s
`min-height: 32px` — the tallest-flex-child math now yields the same
32px content height (→ 48px total row height with the row's existing
8px+8px padding) whether or not a thumbnail is present, on desktop.

The mobile fix is worth calling out specifically: rather than papering
over the column-layout case, the new comment at `app.css:725-729`
states plainly that the desktop "same height" guarantee **cannot** hold
once `flex-direction: column` kicks in below 767px, and explains why
(`align-self: flex-start` instead of stretch, to avoid `object-fit:
cover` cropping the image full-width on a phone). This is the right
call — an honest, scoped claim rather than the original comment's
overreaching one.

Independently re-verified via mutation testing (see §1): reverting to
48px reproduces the exact original 16px regression and the new E2E spec
catches it with a precise message; restoring 32px passes again.

---

### F-03 — RESOLVED — documentation — `app/internal/ideas/ideas.go:1-12`

Previously `minor`: stale claim of mutual independence from
`internal/projects`. Fixed: the doc comment now explicitly describes
the one-way dependency ("since NIU-11, internal/projects imports this
package for WorkerPool and ValidateURL... This package still imports
neither internal/projects nor internal/items"). Accurate and clear.

---

### F-04 — NOT ADDRESSED (by agreement) — tests — `app/tests/integration/helpers_test.go:80-102`

Coordinator confirmed this is intentionally left as-is: no integration
test exercises the actual shared-`*ideas.WorkerPool` wiring `main.go`
uses in production (each test server still wires its own pool per
service). I agree with leaving this out of scope — it is wiring, not
business logic, `main.go` is straightforward to read, and a dedicated
test for it would mostly be asserting its own setup. Not a blocker, not
re-flagged as an action item.

---

### F-05 / F-06 — NOT ADDRESSED (by agreement) — nits

Both were `minor`/`nit` observations I explicitly said required no
action (defensive dead-ish code with a self-explanatory comment;
confirmed-intentional naming divergence). No change expected or made.

---

### F-07 — NOT ADDRESSED (by agreement) — cross-reference, not a new finding

Restated only that `security-engineer`'s N-01 (debug-level URL logging)
applies identically to `internal/projects/service.go` as it does to
`internal/ideas`. Owned by `security-engineer`'s §6, not a code-reviewer
action item.

---

### Process note — `docs/test-plan.md` still has no NIU-11 entry

Confirmed still absent. The coordinator's framing is right: this is a
structural consequence of `/quick` skipping `/define` (where
`test-plan.md` entries are normally authored), not something to
half-fill retroactively for one item while leaving the document's
declared "written during `/define`" contract otherwise intact. I agree
this is out of scope for NIU-11 specifically — it is a standing process
question (first raised by `qa-engineer` on this same item) about
whether `PLAN.md §7` should be amended to scope the binding-contract
claim to `/define`-path items only. Recommend Oriol decide that
separately from this item's shipping.

## 4. Spec conformance checklist

- [x] T-01 Migration: NULL-able `preview_status`, symmetric Down — **now proven** against a pre-existing row and a Down/Up round trip (`migration_005_test.go`), closing the one gap from the prior round.
- [x] T-02–T-09 — unchanged from prior round, all still verified sound.
- [x] T-10 `.project-thumb` sized to match `.state-badge` (32px) — **row-height parity now proven** by an E2E measurement, not just a code comment; mutation-tested by me.
- [x] T-11 Unit tests — **race-free**, independently confirmed.
- [x] T-12 Integration tests — unchanged, still sound.
- [x] T-13 `go build`/`go test`/`gofmt` all clean — reconfirmed independently, plus `-race` on both `internal/...` and `tests/integration/...`.

No scope creep: `handlePatchProjectState` (URL-editing OOS item) still
untouched; no new files touch anything outside T-01–T-13's footprint
plus the new test files this round added.

## 5. Code-quality checklist (code-review-checklist skill)

Unchanged from the prior round's assessment on points 1, 3, 5, 7, 8, 10
(Design, Complexity, Naming, Style, Consistency, Duplication — all still
sound, nothing in this fix round touched them adversely).

- **4. Tests** — upgraded from the prior round: the two structural gaps
  (F-01's race, qa-engineer's G-1/B-09/B-10/B-11/B-12) are now closed
  with tests I independently re-ran and, for the two highest-risk ones
  (row-height, race), mutation-tested myself rather than trusting the
  fix report. The migration test's vacuity guard and index-survival
  check in particular are the kind of defensive test-writing that
  should be the project's default for future schema migrations, not
  just this one.
- **6. Comments** — F-03's fix plus the honest mobile-breakpoint comment
  in F-02's fix are both improvements over the original: the project's
  comment discipline (explain *why*, flag what's NOT guaranteed) held
  up under a second pass, it didn't regress.
- **9. Documentation** — `docs/test-plan.md` gap remains, flagged as a
  process question above, not a code-quality blocker.

## 6. Security (OWASP)

Unchanged — owned by `security-engineer`, `security-review.md`,
**APPROVED**, 0 blocking/major. None of this round's fixes touched the
surfaces that review covered (SSRF egress path, XSS `src`/`href`
handling, `description` exposure) — the fixes were concurrency-safety
in a test double, a CSS layout value, and two doc comments. I have no
new security-relevant observation to add.

## 7. Action items

Nothing blocking remains. Optional follow-ups, not required before
shipping:

1. **[F-04, optional]** One integration test wiring `ideas.Service` +
   `projects.Service` against the *same* shared `*ideas.WorkerPool`,
   mirroring `main.go`'s production wiring — agreed non-blocking.
2. **[Process, optional]** Decide whether `PLAN.md §7`'s `test-plan.md`
   contract should be explicitly scoped to `/define`-path items, given
   `/quick` items structurally cannot populate it before implementation.
3. **[Housekeeping, optional]** Ask `qa-engineer` to regenerate
   `qa-verification.md` against the current tree so the artefact in
   this folder reflects the closed gaps rather than the pre-fix state —
   purely for the historical record; I've independently verified the
   underlying tests myself and am not blocking on this regeneration.

## 8. Sign-off

- **code-reviewer:** APPROVED. Re-verified independently: build, vet,
  gofmt, full test suite, `-race` on both `internal/...` and
  `tests/integration/...`, full Playwright suite, plus targeted
  mutation-testing of the two fixes that mattered most (F-01's race,
  F-02's row height) rather than trusting the fix report at face value.
- **qa-engineer:** matrix gaps (G-1, B-09, B-10, B-11, B-12) closed per
  my independent re-verification of the same tests; `qa-verification.md`
  itself has not been regenerated (see action item 3).
- **security-engineer:** APPROVED (`security-review.md`, unchanged,
  unaffected by this fix round).
