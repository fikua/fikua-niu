---
key: NIU-11
type: qa-verification
status: draft
owner: qa-engineer
source: tasks.md (no requirements.md — shipped via /quick)
---

# NIU-11 — QA verification (test coverage matrix)

Suite run: `cd app && go build ./... && go test ./... -count=1` — **all packages pass**, `gofmt -l .` clean. No flakes observed across repeated runs of the async-resolution tests (`waitForPreviewStatus` / `waitForProjectPreviewStatus`, polling with a 1–2s deadline).

## 1. Coverage matrix — behaviour promised → test that proves it

| # | Behaviour (tasks.md) | Test(s) | Verdict | Notes |
|---|---|---|---|---|
| B-01 | URL is optional; project without url enqueues nothing, `preview_status` stays `nil` | `TestService_Add_WithoutURL_NeverEnqueuesAndPreviewStatusStaysNil` (unit, fake repo); `TestProjects_Add_WithoutURL_PreviewFieldsAllNull` (HTTP integration, real SQLite) | ✅ pass | Unit test additionally fails the test via `t.Fatal` inside the fetch stub if it's ever called — a real behavioural assertion, not just "no error". Sleeps 20ms then re-reads the fake repo to catch a wrongly-enqueued job. Good. |
| B-02 | Whitespace-only URL treated as no URL (not a validation error) | `TestService_Add_EmptyURLTreatedAsNoURL` | ✅ pass | Not explicitly in tasks.md's "Fet quan" but is a real edge case correctly covered. |
| B-03 | Project WITH url returns 201 immediately, without waiting for the scrape (response shows `pending`) | `TestService_Add_WithURL_EnqueuesAndResolvesToReady` (unit); `TestProjects_Add_WithURL_Returns201WithoutWaitingForScrape` (HTTP integration, asserts `preview_status == "pending"` on the POST response itself, then polls to `ready`) | ✅ pass | Asserts real field values (title, image_url) post-resolution, not just status code. |
| B-04 | Fetch error → `failed` | `TestService_Add_FetchError_ResolvesToFailed_NeverBlocksAdd` | ✅ pass | Also asserts `Add` itself never returns an error because of the scrape outcome — correct separation of concerns. |
| B-05 | Partial result with zero recovered fields → `failed` (never a hollow "complete" card) | `TestService_Add_PartialWithZeroFieldsRecovered_TreatedAsFailed` | ✅ pass | Positive counterpart `TestService_Add_PartialWithSomeFieldsRecovered_ResolvesToPartial` also present and asserts the recovered field value. |
| B-06 | Rejected URL scheme (e.g. `javascript:`) → validation error, no row created | `TestService_Add_RejectedURLScheme_NoRowCreated` | ✅ pass | Not in tasks.md's explicit T-11 list but correctly derived from "reutilitzar la validació d'esquema" (T-04) and worth having. |
| B-07 | `description` persisted but NOT exposed in HTTP response | `TestProjects_Add_WithURL_DescriptionNeverExposed` (asserts on raw JSON bytes via `strings.Contains(body, "\"description\"")`, both on the POST response and a follow-up GET) + unit test `TestService_Add_WithURL_EnqueuesAndResolvesToReady` asserts `Description` **is** populated in the domain object after resolution | ✅ pass | This is exactly the right test shape — asserting on raw bytes catches an accidental future `omitempty` removal or JSON tag typo that a struct-decode-only test would miss. |
| B-08 | Migration 005 applies (Up) | Implicit only — every test DB is built via `store.Open` against the full embedded `MigrationsFS`, so migration 005's Up runs on every single test in the suite. | ⚠️ partial | Proves Up doesn't error out today. Does **not** prove Up is idempotent-safe against pre-existing data, because no test seeds data at migration 004 first (see Gap G-1 below). |
| B-09 | Migration 005 reverts (Down) cleanly | none | ❌ missing | No test invokes goose Down at all, for this migration or any other in the repo (`internal/store` has zero `_test.go` files). See Gap G-2. |
| B-10 | Frontend: thumbnail rendered when `image_url` present, using `setAttribute('src', ...)` (not string interpolation), `alt=""` | none (JS) | ❌ missing | No JS unit tests exist in the repo for any module (`projects-render.js` included) — this is consistent with the project's existing testing posture, not a NIU-11 regression, but it is still the only place T-09's actual DOM behaviour is checked at all. |
| B-11 | Frontend: clickable name `<a target="_blank" rel="noopener noreferrer">`, `textContent` for name (never `innerHTML`) | none | ❌ missing | Same as B-10 — no automated check that `rel="noopener noreferrer"` is actually present on the rendered anchor. This is a security-relevant DOM attribute (NFR-02 in tasks.md's own words) with zero automated coverage. |
| B-12 | Frontend: row without url/image_url renders identically to pre-NIU-11 (no visual gap, same row height) | `app/tests/e2e/specs/projects-visual-differentiation.spec.js` (existing, unrelated to url) — does not cover this | ❌ missing | tasks.md T-10 explicitly calls out "verificar-ho amb una fila sense miniatura al costat d'una amb miniatura" as a to-do — there is no test (visual regression or otherwise) that does this comparison. It was verified manually at best. |
| B-13 | `POST /projects` request DTO accepts optional `url` field end-to-end (index.html → projects-api.js → HTTP) | `TestProjects_Add_WithURL_Returns201WithoutWaitingForScrape` covers API layer only | ⚠️ partial | The wiring from the new `#add-project-url` input through `projects-view.js` → `projects-store.js` → `projects-api.js` is untested at any level (no JS tests exist project-wide, and no E2E fills the new input). |

## 2. NULL `preview_status` — the highest-risk area (per your framing)

This is the one deliberate deviation from the proven `ideas` pattern (NOT NULL DEFAULT 'pending' there vs NULL-able here), so it deserves its own breakdown.

| Question | Answer |
|---|---|
| Is there a test proving a URL-less project round-trips through **scan → JSON** without a nil deref or bogus `'pending'`? | **Yes, and it's solid.** `scanProject` (`app/internal/store/projects.go`) uses `sql.NullString` for `preview_status` and only dereferences into `*string` when `.Valid` — this is exercised by `TestProjects_Add_WithoutURL_PreviewFieldsAllNull`, which goes through the *real* SQLite repository (`newProjectsHTTPTestServer` / `newProjectsService`), not the fake repo — so the actual `NULL` → `sql.NullString{Valid:false}` → `nil *string` → JSON `null` path is genuinely proven, not just simulated in a Go map. |
| Is there a test proving it round-trips through **JSON → frontend** without crashing? | **No.** There is no JS test (unit or E2E) that feeds a `preview_status: null` project into `projects-render.js` and asserts it renders without throwing or without a stray "pending" badge appearing. `renderProjectThumbnail`/`renderProjectName` only branch on `image_url`/`url` (both correctly `null`-safe via truthy checks), and nothing in the diff appears to render a `preview_status` badge/spinner at all for projects (unlike `ideas`, which does show a pending spinner) — so there may be no dedicated UI for `preview_status` in projects yet, which would make this a non-issue. **This needs a quick manual/code check**, not a test-coverage judgment — see Gap G-3. |
| Is there a test for an **existing project row created before migration 005** still working? | **No — and this is the real gap.** Every test database in the entire suite is created fresh via `store.Open(dbPath, niu.MigrationsFS)`, which applies **all** migrations 001→005 in one shot before any row is ever inserted. There is no test that: (a) opens a DB pinned at migration 004, (b) inserts a project row the "old way" (no url/title/image_url/description/preview_status columns touched), (c) runs migration 005's Up, (d) reads that row back and asserts `PreviewStatus == nil` (not `"pending"`, not an error). The production risk this covers — a real pre-existing `niu.db` file on Oriol's actual deployment gaining these columns for the first time — is **exactly the scenario the whole NULL-vs-NOT-NULL design decision exists to protect**, and it is untested. |

**Verdict: this is the single most important gap in the whole change.** The reasoning in the migration comment and in `projects.go`'s doc comments is sound and the code (`ALTER TABLE ... ADD COLUMN preview_status TEXT` with no `DEFAULT`, correctly NULL-able) will, in SQLite, in fact leave existing rows with `preview_status = NULL` — that part is standard, well-understood SQLite behavior, not exotic. But nothing *proves* it here, and it's cheap to prove (one test, no new fixtures needed: seed via raw SQL against a DB opened with only migrations up to 004, or simply insert a row via a raw `INSERT INTO projects (...) VALUES (...)` that omits the new columns after opening the full DB, since SQLite's `ADD COLUMN` semantics are what's actually being trusted here — either approach validates the real risk).

## 3. Do the new tests assert real values, or just "no error ran"?

Reviewed every new/changed assertion in `preview_test.go`, `projects_test.go` (integration), and `projects_test_server_test.go`.

- **Real-value assertions, not just no-error:** all six tests in `preview_test.go` assert concrete field values (`Title == "T"`, `ImageURL == "https://example.com/i.jpg"`, `PreviewStatus == PreviewReady/PreviewFailed/PreviewPartial`, etc.) and, in three cases, use `t.Fatal` **inside the fetch stub itself** to positively prove the fetch was never called (`TestService_Add_WithoutURL_...`, `TestService_Add_EmptyURLTreatedAsNoURL`, `TestService_Add_RejectedURLScheme_NoRowCreated`) — this is a stronger assertion than checking a boolean afterwards; it fails loudly and precisely at the call site if the code path changes.
- **The description-absence test is the strongest test in the whole change** — it checks `strings.Contains` against the raw JSON bytes rather than decoding into a struct, which is the only way to actually catch "the field leaked" (decoding into a struct that doesn't declare the field would silently hide a leak).
- **No rubber-stamp tests found** — I did not find any new test in this diff that only checks `err == nil` without also checking a concrete resulting value.

## 4. Frontend / E2E coverage

- `app/tests/e2e/specs/projects.spec.js` and `projects-helpers.js` locate rows via `.project-row` + `hasText` and control elements via `.state-badge`/`.delete-btn` — none of these selectors are affected by wrapping the name in `<a class="project-name">` instead of `<span class="project-name">`, so **existing E2E specs will conceptually keep passing** unmodified against the new markup. Confirmed by reading the diff to `projects-render.js`: `class="project-name"` is preserved on both the `<a>` and `<span>` variants, and no existing locator depends on the tag name.
- **Zero new E2E coverage was added** for: the thumbnail appearing when `image_url` is set, the name becoming a clickable link when `url` is set, `target="_blank"` + `rel="noopener noreferrer"` being present, or the row-height-unchanged claim in T-10's "Fet quan" section. Given this project's E2E suite already covers comparable UI states for `ideas` (which has the same preview pattern), the absence here is a real inconsistency, not an acceptable "we don't do E2E for visuals" project norm.

## 5. Migration 005 Up/Down vs existing rows

- **Up:** runs successfully in every test (implicitly, via `store.Open`), but as detailed in §2, never against a DB that already has rows predating the migration. SQL itself (`ALTER TABLE ... ADD COLUMN ... TEXT` with no default) is correct and idiomatic for what's intended — this is a code-review-level confidence, not a test-proven one.
- **Down:** `-- +goose Down` drops the five columns in reverse order (symmetric, correct-looking). **No test exercises Down at all** — not for this migration, not for any other migration in the project (`internal/store` has no test files). This is a pre-existing gap in the project's overall migration-testing posture, not something NIU-11 introduced or made worse — flagging it for completeness, but it is not a NIU-11-specific regression.

## Gaps ranked by real production risk

1. **[BLOCKING-CANDIDATE] G-1 — No test proves migration 005 behaves correctly against a pre-existing project row.** This is the one deliberate, non-obvious design decision in the whole change (NULL vs NOT NULL DEFAULT 'pending', diverging from the proven `ideas` pattern on purpose) and it is exactly the kind of thing that's obvious in a code review and silently wrong in production. Oriol's real `niu.db` has real pre-005 project rows today. If the `ALTER TABLE` behavior, a SQLite driver quirk, or a future migration squash ever changed this silently, nothing in the suite would catch it. Cheap to close: one test opening a store, inserting a row before "conceptually" gaining the new columns (or verifying directly against the shipped `niu.db`-shaped fixture), asserting `PreviewStatus == nil` after scan.
2. **[MAJOR] G-2 — No JS-level or E2E proof that a `nil`/`null` `preview_status` and absent `image_url`/`url` render safely on the actual page.** The Go-side null-safety is proven; the browser-side is not. Given this project apparently has zero JS unit tests project-wide, the realistic fix is a small E2E addition (add a project without url, assert row renders with no thumbnail and a plain, non-link name — mirrors T-10's own "Fet quan" checklist item verbatim) rather than introducing a new JS test framework.
3. **[MAJOR] G-3 — Zero E2E coverage of the two new interactive/security-relevant DOM behaviors**: thumbnail appears only when `image_url` present, and the name link carries `rel="noopener noreferrer"` + `target="_blank"`. The `rel` attribute in particular is a real tabnabbing mitigation (correctly implemented in the diff) that has no regression protection — a future refactor of `projects-render.js` could drop it silently and nothing would fail.
4. **[NICE-TO-HAVE] G-4 — No goose Down test**, for this migration or any prior one. Pre-existing project-wide gap, not introduced by NIU-11. Worth a backlog item, not a blocker for this change.
5. **[NICE-TO-HAVE] G-5 — No test exercises the full input-to-request wiring for the new `#add-project-url` field** (`index.html` → `projects-view.js` → `projects-store.js` → `projects-api.js`). Low risk: the code is a straight mechanical copy of the existing budget/target_date wiring, and an E2E test filling the URL field would also incidentally cover G-2/G-3 if added.

## Process note — `docs/test-plan.md` was not updated

`docs/test-plan.md` is this project's declared **binding contract**
(`PLAN.md §7`): "el propietari no revisarà el codi... aquest document no
és documentació — és el contracte," with rule 4 stating it is written
**during `/define`, before implementation**, and rule 2 that a case
without a matching automated test is fiction. `grep`-ing the file for
`NIU-11`/`preview` returns **zero matches** — none of the behaviours in
this matrix were added to it.

This is explained, not necessarily wrong: NIU-11 shipped via `/quick`,
which by design skips `/define` (no `requirements.md`, and evidently no
test-plan update either — `/quick`'s contract only produces a minimal
`tasks.md`). But it means the project's own stated "only real control
mechanism" for the human owner currently has a blind spot for the entire
URL-preview feature. If Oriol later re-reads `test-plan.md` expecting it
to be the exhaustive project contract it claims to be, NIU-11 is
invisible in it. Flagging as a **process gap**, not a code/test gap: the
fix is either to backfill `test-plan.md` entries for NIU-11 now, or to
explicitly amend `PLAN.md §7` to scope the contract to `/define`-path
items only.

## What is NOT a gap (confirmed solid)

- Async resolution timing (pending → ready/partial/failed) is well covered at both unit and HTTP-integration level, with real polling against real timeouts, not sleeps-and-hope.
- `description` persisted-but-not-exposed is proven at the byte level — this is the best-tested behaviour in the whole change.
- The `Repository`/`Service`/`httpapi` signature changes (`Create(..., url *string)`, `NewService(..., fetch, pool)`) were consistently propagated to every call site (`main.go`, all integration test servers, `router_test.go`'s spy repo) — confirmed by `go build ./...` and `go vet` succeeding and all pre-existing tests still passing unmodified in behaviour.
- Existing `projects` E2E specs will continue to pass conceptually against the new row markup (verified by reading selectors, not just asserting it).
