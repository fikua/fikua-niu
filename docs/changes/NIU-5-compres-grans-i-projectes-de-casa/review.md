---
artefact: review
key: "NIU-5"                  # REQUIRED — must match tasks.key
title: "Compres grans i projectes de casa"
status: "changes_requested"
verdict: "CHANGES_REQUESTED"
owner: "code-reviewer"
co_reviewers: ["qa-engineer", "security-engineer", "ux-ui-designer"]
tasks_path: "./tasks.md"              # REQUIRED — relative
findings_count: 5                     # F-20, F-21, F-22 (security-engineer) + F-23, F-24 (code-reviewer)
blocking_count: 0                     # 0 blocking; 2 major (F-20, F-23) trigger CHANGES_REQUESTED per the severity-cluster rule
sources:
  - "OWASP Code Review Top 10 (2017)"
  - "Google Engineering Practices — Code Review Developer Guide"
created: "2026-08-03"
updated: "2026-08-03"
---

# Review — Compres grans i projectes de casa

> **What this is.** The pre-PR audit. Produced by `/audit`, consumed by
> `/commit` (which requires `verdict: APPROVED`). Read-only: this file
> never edits code, only reports.

## 1. Verdict

> One of: `APPROVED` · `CHANGES_REQUESTED` · `PENDING`.
> Mirror this into the front-matter `verdict` field.

**Verdict:** `CHANGES_REQUESTED`

**Rationale (one paragraph):**

`qa-engineer` (§2) and `security-engineer` (§6) both ran in parallel and
neither raised a `blocking` finding or a ❌ in the AC↔test matrix (15/15 AC
✅, 17/17 EC ✅, 6/8 NFR ✅ with 2 ⚠️ partial coverage gaps, not broken
behaviour). `go test ./...`, `go test ./... -race -count=1`, `gofmt -l .`
and `go vet ./...` are all clean (see §4/§5 below) — no failing
test/lint/typecheck gate. However, my own review (§3) found **F-23**, a
`major` correctness defect: `projects.Service.ChangeState` reads the
project's prior state via a separate, non-transactional `Get()` call
before the atomic `UpdateState()` transaction commits, so under the exact
AC-07 concurrent-state-change scenario this item explicitly tests for, the
`"from"` value written to the `project_state_changed` event can be wrong
for one of the two racing requests — silently corrupting the audit trail
NFR-01 exists to guarantee. Combined with `security-engineer`'s **F-20**
(`major` — public-repo PII exposure, partly introduced by this diff's
`BACKLOG.md`/`proposal.md` entries), this is a cluster of 2+ `major`
findings, which per the `code-review-checklist` severity ladder forces
`CHANGES_REQUESTED` even with zero `blocking` findings. Both are
straightforward to fix (wrap `Get`+`UpdateState` in one transaction;
redact two names) and do not require touching approved scope, design, or
tests.

## 2. AC ↔ test coverage matrix

> Every AC and NFR from `requirements.md` validated against the
> implementation. `qa-engineer` owns this section. All tests below were
> executed directly for this audit: `cd app && go test ./...` (all
> packages `ok`, verbose run inspected test-by-test) and
> `cd app/tests/e2e && npx playwright test specs/projects*.spec.js
> specs/xss.spec.js specs/accessibility-audit.spec.js` (all 16 specs
> `passed`). `gofmt -l .` and `go vet ./...` both clean (T-33 honored).

| AC / NFR | Statement (short) | Verifying test(s) | Result |
|----------|--------------------|---------------------|--------|
| AC-01 | Afegir una idea nova | `internal/httpapi` DTO tests + `integration/projects_test.go:TestProjects_Add_PersistsAndListed` (POST→GET persistence), `internal/projects/service_test.go` (Add validation) | ✅ pass |
| AC-02 | Marcar una idea com a decidida | `integration/projects_test.go:TestProjects_ChangeState_AllDirections` (`idea`→`decidit` leg, asserts state + `last_updated_by`) | ✅ pass |
| AC-03 | Marcar un element decidit com a fet | `integration/projects_test.go:TestProjects_ChangeState_AllDirections` (`decidit`→`fet` leg) | ✅ pass |
| AC-04 | Cada element mostra qui l'ha tocat i quan | `TestProjects_ChangeState_AllDirections` (asserts `LastUpdatedBy.ID` after every PATCH); DTO always includes `added_by`/`last_updated_by` (`dto_test.go`) | ✅ pass |
| AC-05 | Eliminar un element | `integration/projects_test.go:TestProjects_Delete_RemovesFromList` | ✅ pass |
| AC-06 | Dos usuaris veuen el mateix estat | `TestProjects_ChangeState_WritesEventWithFromTo` (GET-after-PATCH convergence) + `TestProjects_ConcurrentStateChange_*` (GET reflects final state) | ✅ pass |
| AC-07 | Canvi d'estat concurrent convergeix sense error | `TestProjects_ConcurrentStateChange_NoErrorConverges` **and** `TestProjects_ConcurrentStateChange_Repeated` (25 rounds via goroutines, asserts no 5xx + single converged final state) | ✅ pass |
| AC-08 | Espai visualment diferenciat | `e2e/specs/projects-visual-differentiation.spec.js` — asserts nav accent colour AND box-title colour differ from the shopping list (real computed-style comparison, not a snapshot) | ✅ pass |
| AC-09 | Retrocedir un estat és possible, ambdues direccions | `integration/projects_test.go:TestProjects_ChangeState_AllDirections` exercises `idea→decidit→fet→decidit→idea` (forward **and both backward** legs) + `internal/projects/service_test.go:TestService_ChangeState_AnyDirectionValid` (6-step sequence including reversions) | ✅ pass — verified genuinely bidirectional, not just forward |
| AC-10 | Nom d'element obligatori i acotat | `service_test.go:TestService_Add_NameLengthBoundary` (200 ok / 201 rejected) + `TestService_Add_EmptyOrWhitespaceName` | ✅ pass |
| AC-11 | Navegació completa per teclat | `e2e/specs/projects.spec.js:"add, change state and delete a project using only the keyboard"` — Tab-only add/change-state(both directions)/delete | ✅ pass |
| AC-12 | Anunci per lectors de pantalla en canviar d'estat | `e2e/specs/projects.spec.js:"aria-live announces the exact state-change wording..."` — asserts exact text `"{nom} ara està decidit."` | ✅ pass |
| AC-13 | `overview.md` reflecteix el nou espai | Manual doc check: `docs/overview.md` updated with the 3-state lifecycle, authorship, optional budget/target date — not automatable, matches AC-13's own note in `requirements.md` | ✅ pass (manual/documentary, as scoped) |
| AC-14 | Afegir pressupost opcional (text lliure) | `integration/projects_test.go:TestProjects_Add_BudgetAndTargetDatePersisted` + `TestProjects_Add_BudgetAndTargetDateOmittedAreNull` + `service_test.go:TestService_Add_BudgetOptional` | ✅ pass |
| AC-15 | Afegir data objectiu opcional | Same two integration tests above (budget+target_date always tested together) + `service_test.go:TestService_Add_TargetDateOptional` | ✅ pass |
| EC-01 | Nom buit o només espais | `service_test.go:TestService_Add_EmptyOrWhitespaceName` (`""`, `"   "`, tabs, newline) | ✅ pass |
| EC-02 | Nom al límit de longitud (200/201) | `service_test.go:TestService_Add_NameLengthBoundary` | ✅ pass |
| EC-03 | Nom d'idea duplicat, **independentment de l'estat** | `service_test.go:TestService_Add_DuplicateTrimmedCaseInsensitive` (casing/whitespace) **and** `TestService_Add_DuplicateAcrossAllStates` — explicitly seeds a project, transitions it to `idea`/`decidit`/`fet` (subtests per state), and asserts a new duplicate is still rejected **when the existing one is `fet`** | ✅ pass — confirmed the `fet`-state duplicate case specifically, not just `idea` |
| EC-04 | Duplicat exacte permès post-eliminació | `integration/projects_test.go:TestProjects_Duplicate_ExactNameAllowedAfterDelete` + `service_test.go:TestService_Add_DuplicateAllowedAfterDelete` | ✅ pass |
| EC-05 | Element estancat indefinidament a `decidit` | `integration/projects_test.go:TestProjects_StaleDecidit_NoAutomaticChange` — simulates 6-month-old `updated_at`, asserts state/timestamp untouched by a mere GET | ✅ pass |
| EC-06 | No existeix estat "abandonat" | `integration/projects_test.go:TestProjects_NoAbandonedState_OnlyDeleteAvailable` — rejects via API (400) **and** via direct SQL `UPDATE` against the CHECK constraint | ✅ pass |
| EC-07 | Format del camp de pressupost (resolt: text lliure) | Resolved via AC-14/EC-16 coverage; no separate test needed (documentary EC) | ✅ pass (n/a as a distinct test — folded into AC-14/EC-16) |
| EC-08 | Injecció HTML/JS al nom (XSS) | `integration/projects_test.go:TestProjects_XSSPayload_StoredLiterally` (server-side literal storage) **+** `e2e/specs/xss.spec.js:"img onerror does not execute in the projects space"` — real browser, asserts `window.__xssExecuted` (or equivalent) stays `false` | ✅ pass — genuine attack executed against a real browser, not just a mitigation-exists check |
| EC-09 | Injecció SQL al nom | `integration/projects_test.go:TestProjects_SQLInjectionPayload_StoredLiterally_TableSurvives` — sends `'; DROP TABLE projects;--`, asserts literal storage + table row count + `sqlite_master` still lists `projects` | ✅ pass |
| EC-10 | Intent de mutació via `GET` | `integration/projects_test.go:TestProjects_NoMutationViaGET` (runtime) **+** `internal/httpapi/router_test.go` (static route-table assertion via `spyProjectsRepo`, asserts zero mutating calls from any `GET`) | ✅ pass |
| EC-11 | Accés sense sessió autenticada | `integration/projects_test.go:TestProjects_Unauthenticated_Rejected` — all 4 verbs (GET/POST/PATCH/DELETE) without cookie → 401 | ✅ pass |
| EC-12 | Canviar l'estat d'un element ja eliminat | `integration/projects_test.go:TestProjects_ChangeState_NotFound` — PATCH on nonexistent id → 404, other project unaffected | ✅ pass |
| EC-13 | Eliminar un element ja eliminat (idempotent) | `integration/projects_test.go:TestProjects_Delete_IdempotentDoubleDelete` (204 both times) + `service_test.go:TestService_Delete_Idempotent` | ✅ pass |
| EC-14 | Llista buida en primer ús | `e2e/specs/projects.spec.js:"empty state shows a clear message, no error"` | ✅ pass |
| EC-15 | Viewport mòbil | `e2e/specs/projects.spec.js` mobile-viewport describe block (375×667, add/change-state/delete) | ✅ pass |
| EC-16 | Pressupost al límit de longitud (200/201) | `service_test.go:TestService_Add_BudgetLengthBoundary` | ✅ pass |
| EC-17 | Data objectiu en el passat | `service_test.go:TestService_Add_TargetDatePastAccepted` (accepts `2000-01-01` without error) + `TestService_Add_TargetDateInvalidFormat` (format still validated) | ✅ pass |
| NFR-01 | 100% de canvis d'estat reflectits a `events` | `integration/projects_test.go:TestProjects_ChangeState_WritesEventWithFromTo` — asserts a real `events` row with `kind="project_state_changed"` and `payload` containing `from`/`to`, queried directly post-transaction. Covers AC-02/AC-03/AC-09 transitions generically (state-agnostic assertion) | ✅ pass |
| NFR-02 | Zero XSS via `innerHTML` | E2E `xss.spec.js` (real-browser non-execution, see EC-08) + code-level convention: `projects-render.js` uses `document.createElement`/`.textContent` exclusively (verified by reading; no automated grep-in-CI test found — see finding) | ⚠️ partial — behavioural proof is solid (E2E), but the "zero `innerHTML` occurrences" static-grep half of the NFR threshold has no automated CI check |
| NFR-03 | 100% de consultes amb paràmetres vinculats | `integration/sql_static_test.go:TestNoSprintfSQL` (repo-wide static check, pre-existing, extended scope covers `internal/store/projects.go`) + `TestProjects_SQLInjectionPayload_StoredLiterally_TableSurvives` (behavioural) | ✅ pass |
| NFR-04 | Zero rutes `GET` amb efecte mutador | `internal/httpapi/router_test.go` (static route assertion via `spyProjectsRepo`, zero mutating calls recorded across all `GET`s) + `TestProjects_NoMutationViaGET` (behavioural) | ✅ pass |
| NFR-05 | Sessió vàlida requerida a tots els endpoints | `TestProjects_Unauthenticated_Rejected` (all 4 verbs, CI-run) | ✅ pass |
| NFR-06 | Contrast AA + operabilitat per teclat | `e2e/specs/accessibility-audit.spec.js:"projects space has no automatically-detectable accessibility violations"` (axe-core, WCAG 2.2 AA tag) + `projects.spec.js` keyboard-only flow (AC-11) | ✅ pass |
| NFR-07 | Canvis d'estat anunciats a `aria-live` | `projects.spec.js:"aria-live announces the exact state-change wording..."` (own action) — **no test found for the remote/poll-reflected change case** (requirements.md AC-12 explicitly requires "per acció pròpia **o per sondeig que reflecteix un canvi remot**") | ⚠️ partial — own-action path proven; remote/poll-reflected announcement path untested |
| NFR-08 | `prefers-reduced-motion` (no aplicable) | Documented as not-applicable in `design.md` (ADR-03, no animated transition shipped) — no test required per requirements.md §6 itself | ✅ pass (correctly n/a, no test expected) |

**Coverage summary:** 15/15 AC ✅, 17/17 EC ✅ (16 direct + EC-07 correctly folded into AC-14/EC-16), 6/8 NFR ✅, 2/8 NFR ⚠️ partial (NFR-02, NFR-07). No ❌ (missing/vacuous/failing) found — the two ⚠️ are coverage gaps in an otherwise-passing area, not broken behaviour.

## 3. Findings

> Each finding has a stable ID (`F-NN`), a severity, and a clear next
> action. `blocking` findings force `CHANGES_REQUESTED`. Numbered from
> `F-23` to avoid colliding with `security-engineer`'s F-20/F-21/F-22 in
> §6.

### F-23 — `ChangeState`'s event "from" is a check-then-act race, not covered by the atomic state transition

- **Severity:** major
- **Category:** correctness
- **Location:** `app/internal/projects/service.go:180-205` (`Service.ChangeState`), with the atomic half correctly implemented in `app/internal/store/projects.go:193-232` (`ProjectsRepository.UpdateState`)
- **Observation:** `ChangeState` calls `s.repo.Get(ctx, id)` to read `previous.State` for the event payload, then separately calls `s.repo.UpdateState(...)`, which opens its own `BEGIN IMMEDIATE` transaction. The `Get` and the `UpdateState` are **two independent round-trips**, not one critical section — a classic instance of this project's recurring check-then-act defect class already found on NIU-1 (F-02, non-transactional move) and NIU-4 (F-01, rate limiter `Allow`/`RecordFailure`). I confirmed this empirically with a throwaway concurrent probe against a fake repo mirroring the real locking shape (200 rounds, two goroutines racing `ChangeState` on the same id from `idea`): both concurrent calls' events recorded `"from":"idea"`, even though only one of the two `UpdateState` transactions could have genuinely observed `idea` as the row's state immediately before its own commit — the second one's true "from" was actually the first one's "to". `TestProjects_ChangeState_WritesEventWithFromTo` and `TestProjects_ConcurrentStateChange_*` (T-27/T-28) only assert that *a* `project_state_changed` row exists with *some* `from`/`to`, or that the *final* state converges without a 5xx — none of them assert the `from` is correct under concurrent access to the same row, so this gap is currently invisible to the green suite.
- **Why it matters:** `design.md` §2.3/ADR context and `requirements.md` NFR-01 both anchor this item's data-integrity story on `events` being a trustworthy, inspectable audit trail of every state transition ("100% dels canvis d'estat queden reflectits com a esdeveniment... verificable per inspecció directa"). A wrong `from` under exactly the concurrency scenario this item's own AC-07/T-28 exists to exercise means the audit trail can silently lie about what happened — the same class of guarantee this project has already had to fix twice (NIU-1, NIU-4), and the human owner's stated control mechanism (`docs/test-plan.md`) trusts these events to be accurate, not just present.
- **Suggested fix:** move the "read previous state" step inside the same `BEGIN IMMEDIATE` transaction as the `UPDATE` in `ProjectsRepository.UpdateState` (e.g. `SELECT state FROM projects WHERE id = ?` on the same `conn` right before the `UPDATE`, returning both old and new state to the service layer), so the event's `from` is guaranteed to be the value truly overwritten by that specific commit. `Service.ChangeState` then drops its separate `repo.Get` call for this purpose entirely.

### F-24 — `errors.go` codes are declared but two are unreachable from any caller path shown in the diff

- **Severity:** nit
- **Category:** maintainability
- **Location:** `app/internal/projects/errors.go:39-46` (`ValidationBudgetTooLong`, `ValidationInvalidDate`)
- **Observation:** Both codes are correctly produced by `validateBudget`/`validateTargetDate` (`service.go:94`, `:124`) and correctly covered by tests (EC-16/EC-17) — this is not a dead-code finding, just a naming/consistency observation: `ValidationTooLong` (name) and `ValidationBudgetTooLong` (budget) read as parallel siblings, but there's no `ValidationTargetDateInvalid` parallel to `ValidationInvalidDate` for symmetry with the other two `Validation*` constants' `<Field><Problem>` order.
- **Why it matters:** Cosmetic only — no functional impact, no test gap. Listed as a nit per the checklist's "no personal style preferences" rule; not a blocker.
- **Suggested fix:** optional rename to `ValidationTargetDateInvalid` for naming symmetry, only if this file is touched again for another reason.

## 3.5 Visual conformance (ux-ui-designer)

> Compares the implemented UI (`app/web/projects.html`, `app/web/app.css`
> §"Projects space (NIU-5)", `app/web/js/projects-render.js`,
> `projects-store.js`, `projects-main.js`, `a11y.js`) against the binding
> visual decisions in `design.md` §7/§10 (ADR-03, ADR-04) and AC-08/AC-11/
> AC-12/NFR-06/NFR-07/NFR-08.

**Verdict for this section: no blocking or major findings. Two minor
observations below.**

**Single list + badge (design.md §10, confirmed at human gate) — MATCHES.**
`projects.html` renders one `<ul id="projects-list">`; no three-column
board, no drag-and-drop anywhere in `projects-render.js`. Each row
(`renderProjectRow`) is a single `<li>` with a `project-state-selector`
group of three native `<button>` elements (`idea`/`decidit`/`fet`), the
current state marked with `.is-current` + `aria-pressed="true"`. This is
a three-option selector rather than a single passive "badge" — but it
satisfies the AC-09 requirement ("canviar l'estat en qualsevol direcció,
mai només 'següent estat'") more directly than a single-badge-with-menu
would, and design.md §7 itself allows "un selector/menú de tres opcions"
as the vinculant pattern for direction-agnostic transitions. Not a
deviation from the approved decision — the approved decision described
the constraint (single list, no columns/DnD), not a specific control
widget.

**Terracotta accent (ADR-04) — MATCHES, reuses existing tokens.**
`app.css` uses `--color-terracotta` / `--color-terracotta-hover` /
`--color-terracotta-active` — the same tokens already defined for NIU-1
(`app.css:32-34`), not new ad-hoc values. Applied consistently: box title
(`.projects-box .box-title`), active nav tab underline
(`.app-nav-link.is-active[href$="projects.html"]`), current-state badge
background, and the "Afegir" button. No new color token introduced —
confirms ADR-04 §2 ("no calen valors nous, només una assignació
diferent"). Computed contrast: terracotta-on-white text on the active
badge is 4.56:1, inactive badge (surface/text-secondary) is 6.86:1 — both
pass WCAG AA for normal text (AC-11's "text llegible, no només color").

**No state-transition animation (ADR-03) — MATCHES.** No `transition`,
`animation`, or `@keyframes` rule targets `.state-badge`,
`.project-row`, or any projects-specific selector in `app.css` — only a
static `:hover`/`:focus-visible` color swap. `changeProjectState()`
(`projects-store.js`) does a plain server round-trip + re-render, no
optimistic local flip and no FLIP/movement, consistent with ADR-03's
"actualització de text/badge... mai un vol o un desplaçament de fila".
`prefers-reduced-motion` correctly has nothing to gate here — NFR-08 is
rightly not applicable, matching `design.md` §8's explicit statement.

**Navigation / AC-08 differentiation — MATCHES.** Persistent top-level
`<nav aria-label="Seccions de Niu">` with two entries ("🛒 Compra" / "🏠
Projectes") appears on both `index.html` and `projects.html`, same-origin
`<a>` links (no SPA router — within the "decisió d'implementació lliure"
`design.md` §7 explicitly allows). Combined with the terracotta accent,
the space is clearly and immediately distinguishable from the shopping
list on sight, satisfying AC-08/R-01's mitigation plan.

**Empty state (EC-14) — MATCHES.** `renderEmptyProjectsState()` renders a
dedicated `.empty-state` block with icon + explicit Catalan copy ("Cap
projecte encara — afegeix una idea de compra gran o un projecte de
casa."), never an empty `<ul>` or an error state.

**Keyboard path (AC-11) — MATCHES by DOM inspection.** Every interactive
control is a native, unnested `<button>` or `<input>`: three state
buttons per row, one delete button per row, add-form inputs + add button,
nav links, logout button. No nested-interactive-control pattern (the
project comment in `a11y.js` documents this was deliberately fixed after
an axe-core finding on the shopping list, and the same non-nested
row-with-sibling-buttons shape is reused here). All focusable elements
have visible `:focus-visible` outlines using `--color-focus-ring`. No
`tabindex` manipulation hides any control from Tab order.

**Screen-reader narrative (AC-12/NFR-07) — MATCHES.** A single
`#live-region[aria-live="polite"][aria-atomic="true"]` is shared for all
announcements. `announceProjectStateChange()` emits the exact
`"{nom} ara està {estat}."` wording specified in design.md §7/§8, fired
both on the local actor's own change (`changeProjectState`) and on a
remote change detected during `syncProjectsFromServer()` polling (guarded
to skip re-announcing the current user's own confirmed change). The
`announce()` helper in `a11y.js` clears then re-sets `textContent` inside
a `requestAnimationFrame` specifically so repeated identical text is
still re-announced — a correct, non-obvious a11y detail.

**Minor (non-blocking) observations:**

- **M-1 (minor, docs):** `design.md` §7 describes the state control as
  "un selector/menú de tres opcions" without picturing three always-visible
  buttons; a first-time reader comparing prose to implementation might
  expect a `<select>`/menu rather than a three-button toggle group. No
  functional or accessibility gap — `role="group"` +
  `aria-label="Estat de {nom}"` on the wrapper is a correct pattern — but
  worth a one-line addendum to design.md if this space is revisited, so
  future readers aren't surprised by the widget shape.
- **M-2 (minor, out of NIU-5 scope):** the shared `.toast` component
  (`toast-in` keyframe, `translateY(8px)` + opacity, 200ms) used for
  projects-page error toasts is not gated by `prefers-reduced-motion`.
  This is a pre-existing NIU-1 component reused as-is on this page, not a
  new animation introduced by NIU-5, and NFR-08 in this item's
  `requirements.md` is scoped to *state-transition* animation — so this is
  not a NIU-5 regression. Flagging for awareness only; if picked up, it
  belongs to NIU-1's surface, not this item's action items.

No design-system drift found: every new visual (badge, nav entry, empty
state, form) is built from tokens/patterns already defined in `PLAN.md`
§4 and `app.css`'s existing custom-property set — no new color, radius,
shadow or type-scale value was introduced.

## 4. Spec conformance checklist

> Lightweight gate before signing off.

- [x] All ACs from `requirements.md` are covered by passing tests (§2, qa-engineer: 15/15 ✅)
- [x] All NFRs have a measured result, not just "implemented" (§2: 6/8 ✅, 2/8 ⚠️ partial coverage gap — NFR-02 static grep, NFR-07 remote-announce path — not a ❌, not a broken behaviour)
- [x] `tasks.md` checklist is fully `[x]` (35/36 lines checked; the one open line is `C-03` semver bump, correctly left for human ASK-USER per contract)
- [x] Out-of-scope items in `design.md`/`tasks.md` §5 are still out of scope — verified: no three-column board, no drag-and-drop, no soft-delete/`deleted_at`, no notes field, no state-transition animation, no cross-domain coupling with `internal/items` beyond the single `NormalizeName` import (ADR-02)
- [x] No new public API or schema change is undocumented in `design.md` §6 — the 4 `/api/v1/projects` routes and the `projects` table match `design.md` §6.1/§6.2 exactly (verified field-by-field against `dto.go`, `projects_handlers.go`, `router.go`, `003_projects.sql`)

## 5. Code-quality checklist (Google Engineering Practices subset)

- [x] **Design** — right shape for the codebase: `internal/projects` mirrors `internal/items`' `Repository`/`Service`/domain-type trio exactly (ADR-01), reuses `items.NormalizeName` as the sole intentional coupling (ADR-02), reuses `WithCurrentUser`/`RequireCSRF`/`events` verbatim. `router.go`/`main.go` changes are the surgical additions `tasks.md` §6 mandated — `items_handlers.go`/`auth_handlers.go`/`csrf.go` untouched, confirmed by diff.
- [ ] **Functionality** — correct for users with one gap: F-23 (major) — the event audit trail's `from` can be wrong under the exact concurrency scenario AC-07 targets, even though the actual state transition and HTTP response codes are correct.
- [x] **Complexity** — no speculative generality; `ChangeState` correctly has no forbidden-transition state machine (AC-09 requires none). `UpdateState`'s `BEGIN IMMEDIATE` pattern is copy-consistent with `ItemsRepository.Update`, not reinvented.
- [x] **Tests** — present, sized appropriately, assert behaviour not implementation (e.g. `TestProjects_ConcurrentStateChange_Repeated` runs 25 real-goroutine rounds, matching this project's own established anti-flake bar from NIU-1's `TestTwoUsers_ConcurrentMove_Repeated`). No tautological "test that can't fail" pattern found this time (checked `router_test.go`'s route-table assertion specifically, given the NIU-1 history) — `wantGET` is a real allowlist that would break if a mutating route were added to it.
- [x] **Naming** — clear and conventional; matches `internal/items`' naming 1:1 (`Service`, `Repository`, `ErrDuplicate`, `ErrNotFound`, `ErrValidation`). One nit (F-24) on `Validation*` constant naming symmetry.
- [x] **Comments** — used precisely where the *why* is non-obvious (e.g. `service.go`'s doc-comments citing the exact AC/EC each function covers, `router.go`'s explanation of why `/auth/login` is mounted on the outer router). No restated *what*.
- [x] **Style** — `gofmt -l .` and `go vet ./...` both clean; no lint findings to cite.
- [x] **Consistency** — matches `internal/items`/`internal/httpapi` conventions throughout (error envelope, DTO shape, handler structure, JS module boundaries in `web/js/`).
- [x] **Documentation** — `docs/overview.md` updated per AC-13/T-32; `design.md`'s 4 ADRs document every non-obvious decision (ADR-01..04) with alternatives considered.

## 6. Security checklist (OWASP Top 10 + Code Review Top 10)

> Secció propietat de `security-engineer` (opt-in actiu per a NIU-5).
> Auditoria feta contra el codi real del diff `main...HEAD`, no contra les
> descripcions de `tasks.md`. Cada fila diu **què s'ha verificat i com**.

### 6.1 Model d'amenaces del diff

**Fronteres de confiança creuades per NIU-5:**

| Frontera | Novetat | Superfície |
| --- | --- | --- |
| HTTP públic → API | 4 rutes noves sota `/api/v1/projects` | `POST`, `GET`, `PATCH /{id}`, `DELETE /{id}` |
| API → SQLite | Taula nova `projects` + escriptures a `events` | 7 consultes a `internal/store/projects.go` |
| Servidor → DOM | Mòdul de render nou (`projects-render.js`) | `name`, `budget`, `target_date`, `display_name` |

**Classificació de dades tocades:** cap PII nova, cap credencial, cap dada
de pagament. `budget` és **text lliure** (AC-14), no un import estructurat
— no entra en cap àmbit de compliment financer.

**Transicions d'estat afegides:** `idea ↔ decidit ↔ fet`, totes
reversibles, sense màquina d'estats restrictiva a fer complir (AC-09).
Cap sessió, token, cua ni job nou.

**Cap superfície de sortida (outbound).** NIU-5 no fa cap petició HTTP
sortint — **A10 SSRF no aplica** en aquest ítem. (Aquesta superfície
arriba amb NIU-6/`fetchsafe`, auditada per separat.)

### 6.2 OWASP Top 10 (edició vigent) — escombrada

- [x] **A01 Broken Access Control** — ✅ **Verificat al codi, no assumit.**
  Les 4 rutes es registren dins de `r.Route("/api/v1", …)`, que aplica
  `api.Use(WithCurrentUser(authenticator))` (`router.go:91`) abans de
  qualsevol registre de ruta. `WithCurrentUser` retorna `401` amb cos
  mínim davant de `auth.ErrUnauthenticated` (`middleware.go`). **Cap
  excepció ni ruta pública nova** — l'única ruta fora del grup autenticat
  segueix sent `POST /api/v1/auth/login` (NIU-4) i `/healthz`, cap de les
  dues tocada per aquest diff. Cobreix EC-11/NFR-05, amb test
  `TestProjects_Unauthenticated_Rejected`.
  **Nota d'abast (no és una troballa):** Niu és una app de dos usuaris amb
  un únic àmbit compartit per disseny — no hi ha autorització *per
  recurs*, qualsevol usuari autenticat pot modificar qualsevol projecte.
  Això és **intencionat** (`requirements.md` §7 "Multi-llar, rols o
  permisos" explícitament fora d'abast), no un IDOR: `PATCH /{id}` i
  `DELETE /{id}` accepten qualsevol `id` d'un usuari autenticat perquè el
  model de domini no té propietat per fila. Es documenta aquí perquè
  qualsevol canvi futur que introdueixi més d'una llar convertiria
  aquestes dues rutes en un IDOR real (CWE-639) — el punt d'inflexió és
  exactament aquest.
- [x] **A02 Cryptographic Failures** — ✅ NIU-5 no introdueix cap
  primitiva criptogràfica. Reutilitza sense modificar la derivació CSRF
  HMAC-SHA256 i les cookies `HttpOnly; Secure; SameSite=Strict` de NIU-4
  (`csrf.go`). `git diff` no toca `auth/` ni `csrf.go`.
- [x] **A03 Injection** — ✅ **Verificat consulta a consulta.** Les 7
  consultes de `internal/store/projects.go` (`:39`, `:50`, `:152`, `:165`,
  `:210`, `:237`, `:253`, `:273`) usen **exclusivament paràmetres
  vinculats `?`**. `grep -rE 'Sprintf' internal/` creuat amb paraules clau
  SQL (`select|insert|update|delete|from|where|table`) retorna **zero
  coincidències**. Les úniques concatenacions de cadena en SQL són
  `projectSelectColumns`/`projectSelectFrom` (`:152`, `:166`) — constants
  literals del propi codi font, **sense cap entrada d'usuari**, patró
  idèntic al ja establert a `items.go:182`. Compleix NFR-03/EC-09/S8, amb
  test `TestProjects_SQLInjectionPayload_StoredLiterally_TableSurvives`.
  Validació d'entrada addicional (defensa en profunditat, no la mitigació
  principal): `hasControlChars` rebutja controls, `\n\r\t` i **`unicode.Cf`
  (format/zero-width)** — això tanca el bypass de la regla de duplicats via
  zero-width joiners i mitiga Trojan Source (CWE-94).
- [x] **A04 Insecure Design** — ✅ El disseny reutilitza els patrons ja
  provats de NIU-1/NIU-4 en lloc d'inventar-ne de nous: mateix middleware,
  mateix sobre d'error, mateixa disciplina de render. La comprovació de
  duplicats es fa **dins de la mateixa transacció** que l'`INSERT`
  (`projects.go:32-71`) i, sobretot, **l'índex únic
  `idx_projects_name_normalized` és l'autoritat final**, no la
  pre-comprovació — el TOCTOU check-then-insert (CWE-367) està tancat a
  nivell de BD, no només a nivell d'aplicació.
- [x] **A05 Security Misconfiguration** — ✅ `SecurityHeaders` continua
  sent el middleware més extern (`router.go:64`), de manera que HSTS,
  `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer` i la
  CSP **sense `unsafe-inline`** s'apliquen també a `/projects.html` i als
  seus mòduls JS. `LimitBody` (64 KiB) cobreix els nous handlers per
  construcció. Superfície nova mínima: cap flag, cap variable d'entorn,
  cap port.
- [x] **A06 Vulnerable & Outdated Components** — ✅ **NIU-5 no afegeix cap
  dependència.** `git diff main...HEAD -- app/go.mod` és buit; el bloc
  `require` es manté idèntic (chi v5.3.1, goose v3.27.3, x/text v0.40.0,
  modernc/sqlite v1.55.0). `go vet ./...` net. `govulncheck` **no està
  instal·lat en aquest entorn** i, per la regla de no introduir eines que
  el projecte no fa servir, no s'ha instal·lat — veure F-22 (nit).
- [x] **A07 Identification & Authentication Failures** — ✅ Cap camí
  d'autenticació nou. `auth.FromContext` és l'única font d'identitat als
  handlers (`projects_handlers.go:34`, `:68`, `:103`); el `userID` que
  s'escriu a `added_by`/`last_updated_by` prové **sempre del context de
  sessió**, mai del cos de la petició — `addProjectRequest` i
  `patchProjectRequest` no tenen cap camp d'usuari, així que no hi ha
  mass-assignment d'identitat (CWE-915).
- [x] **A08 Software & Data Integrity (CSRF)** — ⚠️ **Implementació
  correcta, cobertura de test incompleta.** Les tres mutacions estan
  embolcallades amb `RequireCSRF(s.authenticator.SessionSecret())`
  (`router.go:124`, `:126`, `:127`), idèntic al patró d'`/items`; el `GET`
  n'està exempt correctament. `RequireCSRF` compara en temps constant
  (`subtle.ConstantTimeCompare`) i rebutja el token buit. **Cap test
  d'integració exercita CSRF contra les rutes de `/projects`** — veure
  F-21 (minor).
- [x] **A09 Logging & Monitoring** — ✅ Cada mutació escriu a la taula
  append-only `events` (`project_added`, `project_state_changed`,
  `project_deleted`) amb `user_id`, satisfent NFR-01. Els errors interns es
  registren amb `slog` **al servidor** i mai al client.
- [n/a] **A10 SSRF** — **No aplica.** NIU-5 no fa cap petició sortint.

### 6.3 OWASP Code Review Top 10 (2017) — complement de revisió

| Senyal | Resultat |
| --- | --- |
| Validació d'entrada | ✅ `validateName` (1–200 runes, trim, controls), `validateBudget` (mateix llindar), `validateTargetDate` (`time.Parse` estricte ISO-8601). Longitud comptada amb `utf8.RuneCountInString`, no `len()` — no es pot desbordar el límit amb multibyte. |
| Codificació de sortida | ✅ Zero `innerHTML`/`outerHTML`/`insertAdjacentHTML`/`document.write`/`eval` a tot `app/web/` (grep exhaustiu: només aparicions en **comentaris** que documenten la prohibició). Neteja de llista amb `replaceChildren()`. |
| Parametrització | ✅ Veure A03. |
| Gestió d'errors | ✅ Veure 6.4. |
| Gestió de secrets | ✅ Veure 6.5. |

**Superfícies de render auditades una a una** (EC-08/NFR-02) — la manera
com sobreviu un XSS és aplicar la regla en un lloc i oblidar-la en un
altre, així que s'ha revisat **cada** camí que toca dades d'usuari:

| Camí de render | Fitxer:línia | Mecanisme |
| --- | --- | --- |
| Nom del projecte | `projects-render.js:121` | `textContent` |
| Pressupost | `projects-render.js:129` | `textContent` |
| Data objectiu | `projects-render.js:136` | `textContent` (via `formatDate`) |
| `aria-label` del selector d'estat | `projects-render.js:90`, `:103` | `setAttribute` — no és context HTML |
| `aria-label` d'eliminar | `projects-render.js:145` | `setAttribute` |
| `title` dels avatars | `projects-render.js:55`, `:69` | propietat DOM, no parseig HTML |
| Emoji d'avatar | `projects-render.js:56`, `:70` | `textContent` |
| Toast (nom en missatge d'error) | `projects-render.js:195` | `textContent` |
| Regió `aria-live` | `a11y.js:22`, `:27` | `textContent` |

**Cap dels nou camins usa una API que interpreti HTML.** Els atributs
`aria-label`/`title` porten text controlat per l'usuari, però
`setAttribute` i la propietat `.title` no parsegen HTML, de manera que no
són un vector d'injecció aquí.

### 6.4 Fuita d'informació als errors (CWE-209)

✅ **Verificat exhaustivament.** Els 9 `writeError` de
`projects_handlers.go` emeten **cadenes literals fixes** en català
(`"S'ha produït un error inesperat."`, `"El projecte ja no existeix."`,
`"Identificador de projecte no vàlid."`) o el `val.Message` d'`ErrValidation`
— que també és una constant del domini (`errors.go:39-46`), mai un
`err.Error()` embolcallat. El `%w` de `fmt.Errorf` a `store/projects.go`
**mai arriba al client**: el `switch` dels handlers cau al cas `default`
genèric. `Recoverer` captura els panics i respon amb el mateix sobre
genèric, registrant el detall només amb `slog`. **Zero rutes de fuita de
SQL, stack traces o rutes de fitxer.**

Únic reflex d'entrada de l'usuari: `«" + trimForMessage(req.Name) + "» ja
existeix."` (`:48`) — retorna al client **el seu propi input**, ja
rebutjat, renderitzat després amb `textContent`. No és auto-XSS explotable
(l'atacant només s'injecta a si mateix) ni fuita de dades d'altri.

### 6.5 Secrets al diff

✅ **Cap secret introduït.** Grep del diff complet contra patrons de clau
d'API, token, clau privada, `AKIA`, `ghp_`: cap coincidència real — totes
les aparicions de "token" són el token CSRF (derivat, no emmagatzemat) o
la paraula "token" en el sentit de *design token* CSS. Cap bloc `.env`,
cap credencial.

### 6.6 Estàndard propi del projecte — `PLAN.md` §3 (vinculant)

| # | Amenaça | Estat a NIU-5 | Evidència |
| --- | --- | --- | --- |
| S1 | CSRF | ✅ implementat / ⚠️ sense test propi | `router.go:124-127`; F-21 |
| S2 | Segrest de sessió | ✅ sense canvis | `csrf.go` intacte |
| S3 | XSS | ✅ | §6.3, 9/9 camins amb `textContent`; CSP sense `unsafe-inline` |
| S4 | Força bruta | n/a | cap camí de login nou |
| S5 | Enumeració d'usuaris | n/a | cap camí de login nou |
| S6 | Fixació de sessió | n/a | cap camí de login nou |
| S7 | Capçaleres | ✅ | `SecurityHeaders` és el middleware més extern |
| S8 | Injecció SQL | ✅ | §6.2 A03, 7/7 consultes parametritzades |
| S9 | Secrets a la imatge/repo | ✅ | §6.5 |
| **S11** | **Repositori públic sense dades personals** | ❌ **incomplert** | **F-20 (major)** |

### 6.7 Troballes de seguretat

> Numerades des de `F-20` per no col·lidir amb `code-reviewer` (§3).

#### F-20 — Noms reals de persones en fitxers versionats d'un repositori públic

- **Severitat:** `major`
- **Categoria:** security (exposició de dades personals)
- **Estàndard:** `PLAN.md` §3 **S11** (regla vinculant del projecte) ·
  OWASP A01/A02 (exposició de dades sensibles) · CWE-359 *Exposure of
  Private Personal Information to an Unauthorized Actor*
- **Ubicació:** `BACKLOG.md` (files NIU-5 i NIU-6, afegides per aquest
  diff) i `docs/changes/NIU-5-…/proposal.md` §"Primari" — **més** el
  preexistent `app/users.json` i `app/tests/e2e/specs/login-cycle.spec.js`
- **Observació:** `PLAN.md` §3 S11 diu, textualment: *"No real names,
  emails, household details … may appear in any committed file. Documents
  refer to `Usuari A` / `Usuari B`."* Verificat que
  `github.com/fikua/fikua-niu` és **`"visibility": "PUBLIC"`** (`gh repo
  view`). El diff d'aquest ítem **afegeix noves ocurrències** de dos noms
  reals de persona a `BACKLOG.md` i `proposal.md`. Independentment, ja hi
  ha a `main` un `app/users.json` versionat que associa nom real ↔ nom
  d'usuari de login per als dos usuaris — i **no està a cap `.gitignore`**.
- **Per què importa:** la regla no és decorativa: el repositori és públic,
  i `users.json` publica dos **noms d'usuari de login vàlids** juntament
  amb els noms reals. Combinat amb `POST /api/v1/auth/login`, això elimina
  el pas d'enumeració d'usuaris que S5 mira de protegir — el rate-limit de
  S4 passa a ser l'única barrera restant. Es marca `major` i no `blocking`
  perquè (a) l'exposició és **preexistent a `main`**, no creada per NIU-5, i
  (b) la contramesura de contrasenya + rate-limit continua vigent; però
  NIU-5 **amplia** l'exposició i no s'hauria de tancar en silenci.
- **Correcció suggerida:** substituir els noms per `Usuari A`/`Usuari B` a
  `BACKLOG.md` i `proposal.md` abans del PR; i, com a seguiment separat
  (no bloqueja NIU-5), treure `app/users.json` del control de versions
  injectant-lo per entorn tal com ja fa `PLAN.md` §3 S9/S11, valorant la
  reescriptura d'historial o la rotació de noms d'usuari.

#### F-21 — Les rutes de `/api/v1/projects` no tenen cap test de CSRF propi

- **Severitat:** `minor`
- **Categoria:** testing (seguretat)
- **Estàndard:** OWASP A08 · `PLAN.md` §3 **S1** · `requirements.md`
  §6 (*"Cap AC de seguretat es dona per bo només perquè existeix una
  mitigació: cada test de seguretat executa l'atac i n'afirma el fracàs"*)
- **Ubicació:** `app/tests/integration/csrf_test.go` — els 6 tests
  (`TestCSRF_*`) apunten tots a `/api/v1/items`
- **Observació:** la protecció CSRF **està correctament implementada** a
  les tres mutacions de projectes (verificat a `router.go:124-127`), però
  cap test l'exercita. NIU-5 sí que ha replicat els altres tres patrons de
  seguretat a `security_test.go` (XSS, SQLi, GET, no-auth) — CSRF és
  l'únic que falta. Contrasta amb el fet que la protecció depèn d'una
  branca `if s.authenticator != nil`: si algú mou una ruta al bloc `else`
  (el camí sense CSRF per a fixtures NIU-1), **cap test fallaria**.
- **Per què importa:** és una regressió silenciosa esperant a passar, en
  una defensa que el propi `PLAN.md` marca com a vinculant. El risc actual
  d'explotació és baix (`SameSite=Strict` és la primera línia i segueix
  activa), per això és `minor` i no `major`.
- **Correcció suggerida:** afegir a `csrf_test.go` l'equivalent de
  `TestCSRF_PatchWithoutToken_Rejected` i
  `TestCSRF_DeleteWithoutToken_Rejected` apuntant a `/api/v1/projects/{id}`.

#### F-22 — Sense escaneig de vulnerabilitats de dependències al projecte

- **Severitat:** `nit`
- **Categoria:** security (cadena de subministrament)
- **Estàndard:** OWASP **A06** Vulnerable & Outdated Components
- **Ubicació:** projecte sencer (no específic de NIU-5)
- **Observació:** NIU-5 **no afegeix cap dependència** (`go.mod` intacte),
  de manera que aquest ítem no empitjora res. Es deixa constància que el
  projecte no té `govulncheck` ni cap equivalent a CI, així que cap CVE de
  la cadena transitiva (chi, goose, modernc/sqlite, x/crypto) es detectaria
  automàticament. No s'ha instal·lat cap eina per auditar-ho, per la regla
  de no introduir eines que el projecte no fa servir.
- **Correcció suggerida:** seguiment separat — afegir `govulncheck ./...`
  al pipeline. **No bloqueja NIU-5.**

### 6.8 Hipòtesis comprovades que **no** són troballes

> Explicitat perquè confirmar que un mecanisme és sòlid val tant com
> trobar-hi un forat.

- **Fuita d'SQL als errors 500** → no. El `%w` de la capa `store` mai
  travessa el `switch` dels handlers (§6.4).
- **Mass-assignment d'`added_by`/`last_updated_by`** → no. Els DTOs
  d'entrada no tenen camp d'usuari; l'identitat ve sempre del context.
- **TOCTOU a la comprovació de duplicats** → no explotable. L'índex únic
  de BD és l'autoritat final i `isUniqueConstraintErr` reconverteix la
  cursa a `ErrDuplicate` (`projects.go:56-60`).
- **`DELETE` no idempotent com a oracle d'existència** → no. `Delete`
  retorna sempre `204` existeixi o no (EC-13); no filtra si un `id` existia.
- **Mutació via `GET`** → no. Doblement verificat: introspecció de la
  taula de rutes de chi (`router_test.go`, `wantGET` és una allowlist
  explícita que trenca si algú hi afegeix res) **més** un spy repo que
  afirma zero crides mutants (`spyProjectsRepo`). Cobreix EC-10/NFR-04.
- **Injecció al `target_date`** → no. `time.Parse` amb layout estricte
  rebutja qualsevol cosa que no sigui una data de calendari vàlida abans
  d'arribar a la BD.
- **Desbordament del límit de longitud amb Unicode multibyte** → no. El
  recompte és per runes (`utf8.RuneCountInString`), no per bytes.
- **Bypass de duplicats amb caràcters zero-width** → no.
  `hasControlChars` rebutja la categoria `unicode.Cf` sencera.

### 6.9 Veredicte de seguretat

**Cap troballa `blocking`.** El nucli tècnic de la superfície CRUD és
sòlid: A03 (injecció) i A01 (autenticació) estan tancats i **verificats
al codi**, i els 9 camins de render usen `textContent`. L'única troballa
rellevant (**F-20**, `major`) no és un defecte de codi sinó l'incompliment
d'una regla vinculant de dades personals en un repositori públic — la
part atribuïble a NIU-5 (`BACKLOG.md`, `proposal.md`) es resol amb una
edició de text abans del PR.

## 7. Action items (only if `CHANGES_REQUESTED`)

> Numbered, with owner and pointer back to the finding.

1. Make `ChangeState`'s read of the prior state part of the same `BEGIN IMMEDIATE` transaction as `UpdateState`, so the `project_state_changed` event's `from` field is guaranteed correct under concurrent access to the same project — owner: `fullstack-developer` — fixes: F-23
2. Replace the two real person names newly introduced in `BACKLOG.md`/`proposal.md` with `Usuari A`/`Usuari B` before `/commit` (public repo, `PLAN.md` §3 S11) — owner: `fullstack-developer` — fixes: F-20 (security-engineer, §6)
3. (Optional, non-blocking) Add `TestCSRF_PatchWithoutToken_Rejected`/`TestCSRF_DeleteWithoutToken_Rejected` for `/api/v1/projects/{id}` — owner: `fullstack-developer` — fixes: F-21 (security-engineer, §6, minor — does not block this verdict alone)
4. (Optional, non-blocking) Add a CI-level `grep -r innerHTML app/web/` style static check to close the measurement half of NFR-02, and a poll-reflected-announcement E2E case for NFR-07 — owner: `fullstack-developer` — fixes: qa-engineer's two ⚠️ partial rows in §2 (do not block this verdict alone)

## 8. Sign-off

> Filled in once `APPROVED`. Not applicable this round — verdict is
> `CHANGES_REQUESTED` (see §1). Re-run `/audit NIU-5` after action items
> 1–2 land; items 3–4 are non-blocking and may ship in the same or a
> follow-up pass at the developer's discretion.

- **Approver:** —
- **Date:** —
- **Next step:** `/code NIU-5` to address F-23 (and F-20 from `security-engineer`), then re-run `/audit NIU-5`
