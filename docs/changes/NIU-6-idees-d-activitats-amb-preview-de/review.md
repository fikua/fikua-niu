---
artefact: review
key: "NIU-6"
title: "Idees d'activitats amb previsualització de link"
status: "reviewed"
verdict: "APPROVED"
owner: "code-reviewer"
co_reviewers: ["qa-engineer", "security-engineer", "ux-ui-designer"]
tasks_path: "./tasks.md"
findings_count: 21
blocking_count: 0
sources:
  - "OWASP Code Review Top 10 (2017)"
  - "OWASP Top 10 (2021)"
  - "OWASP SSRF Prevention Cheat Sheet"
  - "Google Engineering Practices — Code Review Developer Guide"
created: "2026-08-03"
updated: "2026-08-03"
---

# Review — Idees d'activitats amb previsualització de link

> **What this is.** The pre-PR audit. Produced by `/audit`, consumed by
> `/commit` (which requires `verdict: APPROVED`). Read-only: this file
> never edits code, only reports.
>
> **This is a re-audit.** The previous pass (this same file, dated
> 2026-08-03) produced `CHANGES_REQUESTED` with 2 blocking findings
> (F-17, F-19) and 11 non-blocking findings across 8 follow-up commits.
> Every one of the 13 action items has since been addressed except F-14,
> which was correctly deferred as a `platform-engineer` backlog note (not
> a code fix — this was the previous review's own suggested resolution).
> All 13 have been independently re-verified this round by
> `code-reviewer`, `qa-engineer`, `security-engineer`, and
> `ux-ui-designer` — each reading the actual diff and re-running tests,
> not trusting commit messages. Findings F-01 through F-21 are preserved
> below for traceability; each carries an updated status where this round
> changed it.

## 1. Verdict

**Verdict:** `APPROVED`

**Rationale (one paragraph):**

Both blocking findings from the prior pass are genuinely fixed, verified
independently by three agents reading the same diff separately and
converging on the same conclusion. F-17 (`.idea-card` missing
`role="group"`/`aria-label`): `newCardContainer()` in `ideas-render.js`
now sets both attributes, confirmed applied through all four state-render
functions (ready/failed/partial/pending) via the single shared factory —
no fifth/alternate card-construction path was left unfixed. F-19 (wrong
empty-link copy): the client string is now byte-exact against
`proposal.md` §8.3's EC-10 spec, and the fix commit additionally repaired
an identical copy-paste bug in `internal/ideas/service.go`'s server-side
`ValidationEmpty` message — a real fix, not overstated, since
`ideas-api.js` propagates server error messages verbatim to the same UI
region on any non-OK response, so the server string is a live fallback
path even though the client-side guard normally makes it unreachable
today. The EC-01 (invalid scheme) message was independently confirmed
byte-exact against spec by three separate reads (this agent,
`qa-engineer`, `ux-ui-designer`) — no fix was needed there, and the fix
commit's claim to that effect checks out. All 11 non-blocking findings
from the prior pass are also resolved, each spot-checked here at
increasing depth for the higher-risk ones: F-09/F-06 (`og:image` scheme
validation) is a clean, correctly-scoped fix — `isHTTPOrHTTPSURL` silently
discards non-http(s) schemes before storage, verified by direct code read,
by this agent's own probe, and by `security-engineer`'s independent
19-case probe (14 hostile schemes, including evasion variants the fix
commit didn't explicitly mention — mixed case, embedded whitespace,
tab/newline/NUL, `blob:` — all correctly discarded). F-10/F-07 (length
caps) are rune-aware and boundary-tested (exact post-truncation lengths
asserted, not just "shorter"). F-11 (misleadingly-named regression test)
took option (b) from the prior review — rename + honest doc comment — and
`security-engineer` re-ran the exact mutation test from the prior audit
(revert `DisableKeepAlives` to `false`) and got the identical result:
the renamed test still passes (correctly, since it never claimed to
detect this), and `TestNewTransport_DisableKeepAlivesTrue` still fails
loudly, confirming it remains the real guard. F-12 (denylist trailing-dot
bypass) is fixed at both call sites (hardcoded list and `NIU_PUBLIC_HOST`),
verified end-to-end by `security-engineer` including confirming the
original bypass was real (both host forms resolve to the same IP) and
that the one remaining residual (double-trailing-dot) is provably
non-exploitable (invalid DNS label, resolution fails). F-03 (CSRF tests),
F-04 (compose.yaml committed), F-20 (client-side XSS E2E for ideas), F-18
(delete-button aria-label), and F-02/F-05 (doc-comment drift) are all
confirmed fixed as claimed, each independently re-verified by at least one
other agent. `qa-engineer`'s AC↔test matrix is now **11/11 AC, 18/18 EC,
9/9 NFR — zero partial, zero unverified** (up from 10/11 AC, 17/18 EC,
8/9 NFR in the prior round; the sole gap, EC-11/NFR-01's missing
client-side non-execution test for ideas, F-20, is now closed with a
genuinely behavioral Playwright test — asserts non-execution, absence of
injected DOM nodes, literal-text presence, and zero page errors — that
`qa-engineer` read line-by-line and additionally ran directly (9/9 passing
in the file). `security-engineer` recommends `APPROVED` from a pure
security standpoint with zero open blocking/major findings.
`ux-ui-designer` confirms `APPROVED` on F-17/F-18/F-19 with one new minor,
non-blocking finding (no direct E2E attribute assertion for
`role`/`aria-label`, currently verified by manual code read only). This
agent ran the full verification suite independently: `go build ./...`,
`go vet ./...`, `gofmt -l .` (clean), `go test ./... -count=1` (all
packages `ok`), and `go test ./... -race -count=1 -timeout 20m` (all
packages `ok`, including `tests/integration` at 573.934s, zero data
races — a first attempt at this same run surfaced a build failure in
`internal/fetchsafe`, traced to a transient, untracked, already-deleted
probe test file left in the working tree from this session's own
verification process, not a defect in the branch; a clean re-run
confirmed the whole suite green). No new blocking or major findings
surfaced by this round of review. `internal/items`, `internal/projects`,
`internal/auth`, `internal/config` remain completely untouched
(`git diff main...niu-spa-conversion --stat` against all four paths is
empty). Two residual minor/informational items are recorded as follow-ups
(not blockers): the missing direct E2E attribute assertion for
`role="group"`/`aria-label` (`ux-ui-designer`'s new finding), and F-14
(the `traefik-public` service-name denylist completeness, correctly
tracked as a `platform-engineer` backlog note per the prior audit's own
suggested resolution, not addressed by code in this pass because none was
expected).

## 2. AC ↔ test coverage matrix

> Produced by `qa-engineer` (full detail in their scratch section,
> `review-qa-section.md`, merged below). `code-reviewer` independently
> re-ran `go build`/`go vet`/`gofmt -l`/`go test ./...`/`go test -race`
> and cross-checked the F-09/F-10/F-11/F-12 fixes at the code level —
> results agree.

**Test execution performed by `qa-engineer` this round:** `go build ./...`,
`go vet ./...`, `gofmt -l .`, full `go test ./... -count=1` (all green),
targeted re-runs of every fix (`TestParseOpenGraph_Image*`,
`TestParseOpenGraph_*ExceedsCap*`, `TestCSRF_Ideas_*`,
`TestIsDeniedHost_*`), the T-27a/c/d/e SSRF mechanism tests individually
(no regression from F-11/F-12's adjacent changes), and the full
`xss.spec.js` Playwright suite (9/9 passing, including the new F-20 case).
`code-reviewer` independently ran the same full suite plus
`go test ./... -race -count=1 -timeout 20m` (573.913s for
`tests/integration`, zero data races) — see §3 note on a transient,
now-deleted stray test file that caused one false-negative run before the
clean confirmation.

### AC ↔ test coverage

| AC | Statement (short) | Verifying test(s) | Result |
|----|--------------------|--------------------|--------|
| AC-01 | Afegir idea amb previsualització completa (OG complet, persisteix) | `internal/fetchsafe`: `TestParseOpenGraph_FullTags`; `internal/ideas`: `TestServiceAdd_SuccessfulScrape_ResolvesToReady`; integration: `TestIdeas_FullOpenGraph_SavesAsCompleteCardAndPersists` | ✅ pass |
| AC-02 | Afegir idea quan la previsualització falla — mai bloqueja | `internal/ideas`: `TestServiceAdd_FetchError_ResolvesToFailed_NeverBlocksAdd`; integration: `TestIdeas_AccessBlocked_SavesAsFallback`, `TestIdeas_Timeout_SavesAsFallback`, `TestIdeas_NonHTMLContentType_SavesAsFallback_NoParsingAttempted` | ✅ pass |
| AC-03 | Previsualització amb dades parcials, camps absents omesos sense error | `TestParseOpenGraph_PartialTags`; `TestServiceAdd_PartialScrape_ResolvesToPartial`, `TestServiceAdd_EmptyPartialResult_TreatedAsFailed`; `TestIdeas_PartialOpenGraph_SavesAsPartial` | ✅ pass |
| AC-04 | Cada idea mostra qui l'ha afegit | integration: `TestIdeas_FullOpenGraph_SavesAsCompleteCardAndPersists` asserts `AddedBy != nil` | ✅ pass |
| AC-05 | Eliminar una idea desada, no reapareix | `TestServiceDelete_Idempotent`; `TestIdeas_Delete_RemovesFromListAndDoesNotReappear` | ✅ pass |
| AC-06 | Dos usuaris veuen les mateixes idees (convergència) | `TestIdeas_TwoClients_SeeConvergedList` | ✅ pass |
| AC-07 | Espai visualment diferenciat (Compra/Projectes) | E2E `ideas.spec.js`: navigation entry/accent colour differ | ✅ pass |
| AC-08 | Enllaç obligatori i format vàlid processat | `TestValidateScheme_*`; `TestServiceAdd_EmptyURL_Rejected`, `TestServiceAdd_NonHTTPScheme_Rejected` — **EC-01 server message now confirmed byte-exact against spec** (3 independent reads) | ✅ pass |
| AC-09 | Navegació completa per teclat | E2E `ideas.spec.js`: keyboard add/delete | ✅ pass |
| **AC-10** | Targeta accessible per lectors de pantalla | axe-core WCAG 2.2 AA (unchanged, zero violations) **+ F-17/F-19 UI fixes confirmed applied and correct in code this round** (`role="group"`/`aria-label` present on all 4 states; correct copy shipped) | **✅ pass — no longer flagged** |
| AC-11 | `overview.md` reflecteix el nou espai | Manual/documental (unchanged) | ✅ pass (manual) |

**AC summary: 11/11 fully verified** (was 10/11 in the prior round — AC-10's
remaining gap, F-17/F-19, is now fixed and verified).

### Edge case ↔ test coverage

| EC | Statement (short) | Verifying test(s) | Result |
|----|--------------------|--------------------|--------|
| EC-01 | Esquema no `http(s)` rebutjat, zero petició de xarxa | `TestValidateScheme_RejectsNonHTTPSchemes`; `TestServiceAdd_NonHTTPScheme_Rejected` | ✅ pass |
| EC-02 | Destí xarxa privada/loopback/pròpia instància, literal IP | `TestValidateIPForConnect_RejectsPrivateLoopbackLinkLocal`; T-27a | ✅ pass |
| EC-03 | Resposta extremadament gran | `TestIdeas_OversizedResponse_SavesAsFallback_NoMemoryExhaustion` | ✅ pass |
| EC-04 | Cadena de redireccions, incl. les dues regressions específiques | `TestCheckRedirect_HopLimit`, `TestCheckRedirect_RejectsNonHTTPSchemeOnRedirect`; T-27c (renamed, F-11), T-27d (F-02 regression) | ✅ pass |
| EC-05 | Metadades OG absents/malformades | `TestParseOpenGraph_NoOGTags_TreatedAsNotFound`, `TestParseOpenGraph_MalformedHTML_DoesNotCrash` | ✅ pass |
| EC-06 | Idea duplicada, sense bloqueig | `TestIdeas_SameLinkSavedTwice_NoDeduplication` | ✅ pass |
| EC-07 | Destí privat via resolució DNS | `TestValidateIPForConnect_RejectsIPv4MappedIPv6`; `TestFetchPreview_HostnameResolvingToLoopback_Rejected` | ✅ pass |
| EC-08 | Timeout de resposta | `TestNewClient_NoAuthHeadersConfigured`; `TestIdeas_Timeout_SavesAsFallback` | ✅ pass |
| EC-09 | Contingut no HTML | `TestIdeas_NonHTMLContentType_SavesAsFallback_NoParsingAttempted` | ✅ pass |
| **EC-10** | Enllaç buit / només espais | `TestValidateScheme_RejectsNonHTTPSchemes`; `TestServiceAdd_EmptyURL_Rejected` — **client-side error string now byte-exact against spec** (F-19) | **✅ pass — no longer flagged** |
| **EC-11** | Injecció HTML/JS a metadades — mostrat com text literal | Server-side (unchanged): `TestIdeas_XSSPayloadInURL_StoredLiterally`, `TestIdeas_XSSPayloadInRecoveredTitle_StoredLiterally`. **Client-side (NEW, F-20):** `xss.spec.js` — `img onerror in the URL does not execute in the ideas space` — real-browser non-execution assertion, read and re-run independently by `qa-engineer` (9/9 pass) | **✅ pass — no longer flagged** |
| EC-12 | Injecció SQL — desat literal, taula intacta | `TestIdeas_SQLInjectionPayloadInURL_StoredLiterally_TableSurvives`; `TestNoSprintfSQL` | ✅ pass |
| EC-13 | Mutació via `GET` inexistent | `TestNoMutatingGETRoutes`, `TestGETRequestsDoNotMutateState`, `TestIdeas_NoMutationViaGET` | ✅ pass |
| EC-14 | Accés sense sessió | `TestIdeas_Unauthenticated_Rejected` | ✅ pass |
| EC-15 | DELETE ja eliminat — idempotent | `TestServiceDelete_Idempotent`; `TestIdeas_DoubleDelete_Idempotent` | ✅ pass |
| EC-16 | Doble POST — NO idempotent | `TestIdeas_DoublePost_CreatesTwoIndependentIdeas_NotIdempotent` | ✅ pass |
| EC-17 | Llista buida en primer ús | `TestIdeas_EmptyListOnFirstUse` | ✅ pass |
| EC-18 | Viewport mòbil | E2E `ideas.spec.js`: mobile viewport case | ✅ pass |

**EC summary: 18/18 fully verified** (was 17/18 in the prior round — EC-11
is now closed by F-20's new test).

### NFR ↔ test coverage

| NFR | Statement (short) | Verifying test(s) | Result |
|-----|--------------------|--------------------|--------|
| **NFR-01** | XSS — zero `innerHTML` amb dades externes/usuari | Code inspection (unchanged: zero `innerHTML` in `ideas-render.js`) **+ NEW behavioural E2E assertion** (F-20, same test as EC-11 above) | **✅ pass — no longer flagged** |
| NFR-02 | Injecció SQL — 100% paràmetres vinculats | `TestNoSprintfSQL` + EC-12 tests | ✅ pass |
| NFR-03 | Cap mutació via `GET` | `TestNoMutatingGETRoutes`, `TestGETRequestsDoNotMutateState` | ✅ pass |
| NFR-04 | Tots els endpoints requereixen sessió | `TestIdeas_Unauthenticated_Rejected` | ✅ pass |
| NFR-05 | SSRF — rebuig d'esquema abans de xarxa | `TestValidateScheme_RejectsNonHTTPSchemes` | ✅ pass |
| NFR-06 | SSRF — destins prohibits, **BLOCKING gate** | T-27a-e, all individually re-run this round by both `qa-engineer` and `security-engineer` — no regression from F-11/F-12's adjacent changes | ✅ pass |
| NFR-07 | SSRF — límits de recurs (timeout + mida + concurrència) | `TestCheckRedirect_HopLimit`; `TestIdeas_Timeout_SavesAsFallback`, `TestIdeas_OversizedResponse_SavesAsFallback_NoMemoryExhaustion` | ✅ pass |
| NFR-08 | Cap credencial de Niu surt cap a destí extern | `TestFetchPreview_NoAuthHeaders_RealRequestInspection` | ✅ pass |
| NFR-09 | Cache at-save-time — mai re-scraping en `GET` | `TestIdeas_RepeatedGET_NeverReScrapes` | ✅ pass |

**Summary: 11/11 AC · 18/18 EC · 9/9 NFR — zero partial, zero unverified.**
NFR-06, the declared blocking gate, is satisfied and re-confirmed
non-regressed by two independent agents this round.

## 3. Findings

> Findings F-01..F-08 are `code-reviewer`'s own. F-09..F-16 are
> `security-engineer`'s. F-17..F-19 are `ux-ui-designer`'s. F-20 is
> `qa-engineer`'s. F-21 was `code-reviewer`'s verification note. Numbering
> preserved from the prior round for traceability; each finding's status
> is updated to reflect this round's re-verification. New findings from
> this round (none blocking/major) are appended at the end as F-22/F-23.

### F-01 — `spaFallback`/toast-collision findings from the prior ad-hoc SPA review (RESOLVED, unchanged from prior round)

- **Severity:** n/a (positive confirmation)
- **Status:** unchanged — still resolved, not re-verified this round (no
  code in this area changed since).

### F-02 — `Repository.Get`/`ErrNotFound` doc-comment drift (RESOLVED this round)

- **Severity:** was minor
- **Category:** maintainability
- **Location:** `app/internal/ideas/errors.go:5-19`
- **Status:** **RESOLVED**, verified by direct read of commit `f49ea90`.
  `ErrNotFound`'s doc comment no longer claims a nonexistent `Service.Get`
  method — it now accurately describes the only real caller
  (`IdeasRepository.Create`'s internal re-fetch) and explicitly notes the
  type is unreachable from any HTTP handler today, kept as forward-looking
  API surface. No behavioural change, comment-only fix, exactly as scoped.

### F-03 — `/api/v1/ideas` CSRF test coverage (RESOLVED this round)

- **Severity:** was minor
- **Category:** testing (security)
- **Location:** `app/tests/integration/csrf_test.go:235-278`
- **Status:** **RESOLVED**, verified by direct read (structurally
  identical to the proven `/projects` pattern: login → mutate without
  token → assert `403` → assert no state change via a follow-up
  authenticated `GET`) and by running both tests directly
  (`TestCSRF_Ideas_PostWithoutToken_Rejected`,
  `TestCSRF_Ideas_DeleteWithoutToken_Rejected` — both pass,
  `qa-engineer`'s independent run confirms the same). Not a shortcut
  version of the mirrored pattern.

### F-04 — `compose.yaml` rate-limit increase uncommitted (RESOLVED this round)

- **Severity:** was minor
- **Category:** maintainability / process
- **Location:** `app/compose.yaml` (commit `d17938a`)
- **Status:** **RESOLVED**, verified — the average/burst change
  (`10/20` → `60/100`) is now a deliberate, documented commit with a clear
  rationale (two polled pages × two users pushing past 10 req/min in
  normal use, confirmed via browser console) and an explicit note that it
  had been sitting uncommitted since before this branch, at risk of being
  silently dropped on a `checkout`/`stash`. The risk this finding
  described no longer exists.

### F-05 — `ErrResponseTooLarge` doc-comment drift (RESOLVED this round)

- **Severity:** was nit
- **Category:** maintainability
- **Location:** `app/internal/fetchsafe/errors.go:20-35`
- **Status:** **RESOLVED**, verified by direct read of commit `f49ea90`.
  The comment no longer claims the sentinel "is returned when" the limit
  is hit — it now documents the actual silent-truncation behaviour
  (`LimitReader` cutoff surfaces as an ordinary `html.ErrorToken`, handled
  identically to any other EC-05 fallback case) and correctly notes the
  sentinel's only live reference is a test double. No behavioural change.

### F-06 — see F-09 (merged, unchanged numbering for traceability)

### F-07 — see F-10 (merged, unchanged numbering for traceability)

### F-08 — Test suite verification note (superseded by this round's re-run, see §3 note below and F-21)

- **Severity:** n/a (verification note, prior round)
- **Status:** superseded — this round's full independent re-run (this
  agent + `qa-engineer`) replaces it; see rationale in §1 and the note
  below F-21.

### F-09 — `og:image` scheme validation (RESOLVED this round)

- **Severity:** was high
- **Category:** security (OWASP A03 Injection · CWE-79 · CWE-20 ·
  CWE-918 second-order SSRF)
- **Location:** `app/internal/fetchsafe/ogparse.go:43-58,151-167`
  (`isHTTPOrHTTPSURL` + `applyMetaToken`)
- **Status:** **RESOLVED**, verified at three independent levels: (1) this
  agent's own direct code read plus a throwaway reproduction of the
  original bypass (now confirmed fixed); (2) new unit tests
  (`TestParseOpenGraph_Image{Javascript,Data,File}Scheme_Discarded`,
  `..._ImageSchemeRelative_Discarded`, `..._ImageHTTPAndHTTPSSchemes_Accepted`)
  read and confirmed boundary-correct; (3) `security-engineer`'s
  independent 19-case probe, covering not just the 3 vectors this audit
  named (`javascript:`, `data:`, `file:`) but additional evasion variants
  not explicitly requested (mixed case, leading whitespace, embedded
  tab/newline/NUL, `blob:`, scheme-relative `//host/path`) — all 14
  hostile cases discarded, all 3 legitimate http(s) cases stored correctly
  unmodified. The fix is scheme-only, not destination-validating (an
  attacker-controlled `http://127.0.0.1/x.png` is still stored) — correctly
  scoped, since the browser (not the server) makes this request and CSP
  `img-src 'self' https:` is the layer that governs it; `security-engineer`
  flags this as an informational note for future server-side sinks
  (image proxy, thumbnailing, email), not a gap in this fix. The
  previously-inaccurate doc-comment claim in `ideas-render.js:15-16`
  ("fetchsafe already guarantees image_url only ever holds an http(s)
  URL") is now **true** — it was false before this fix, which was
  precisely this finding's substance.

### F-10 — Length caps on recovered OG metadata (RESOLVED this round)

- **Severity:** was low
- **Category:** security (OWASP A04 Insecure Design · CWE-770)
- **Location:** `app/internal/fetchsafe/ogparse.go:18-41`
- **Status:** **RESOLVED**, verified: `maxTitleLen=300`,
  `maxDescriptionLen=1000`, `maxImageURLLen=2048`, applied via a
  rune-aware `truncate()` (converts to `[]rune` before slicing — verified
  this matters for this project's Catalan-diacritic titles, where a
  byte-level cut would produce invalid UTF-8 in SQLite). Boundary-tested:
  each over-cap test asserts the exact post-truncation length, and a
  separate within-cap test asserts normal-length fields are byte-for-byte
  unmodified — this proves "truncate, don't reject," not merely "some
  limit exists." `security-engineer`'s own probe with 5000-character
  inputs (including a multi-codepoint emoji ZWJ sequence) confirms the
  truncation never splits mid-codepoint.

### F-11 — Misleadingly-named F-01 regression test (RESOLVED this round)

- **Severity:** was medium
- **Category:** testing (OWASP Code Review Top 10 #10 · CWE-1006)
- **Location:** `app/tests/integration/ideas_ssrf_test.go:140-214`
- **Status:** **RESOLVED**, via option (b) from the prior review (rename +
  honest documentation, not a behavioural rewrite — judged not worth the
  API-surface cost of exposing a dialer-injection seam purely for this
  test). Renamed to
  `TestFetchPreview_RedirectToSameHost_FirstHopRejectedBeforeConnect`, with
  a new doc comment that explicitly states what it does NOT prove (per-hop
  re-dial detection — the destination is loopback, rejected on the first
  dial, so the redirect chain never progresses far enough to exercise
  connection reuse) and what it DOES prove (first hop blocked before
  `connect()`, zero hops silently let through), correctly pointing to
  `TestNewTransport_DisableKeepAlivesTrue` as the real F-01 guard.
  `security-engineer` **re-ran the exact mutation test from the prior
  audit** (reverted `DisableKeepAlives` to `false` in `client.go`,
  confirmed via `git diff` the file was restored clean afterward) and got
  the identical result: the renamed integration test still passes
  (correctly — it never claimed to catch this), and
  `TestNewTransport_DisableKeepAlivesTrue` fails immediately, confirming it
  remains the actual regression guard. The finding was about documentation
  honesty, not an exploitable gap — F-01's real coverage was never lost,
  only mislabeled — and this is now fixed.

### F-12 — Denylist trailing-dot FQDN bypass (RESOLVED this round)

- **Severity:** was medium
- **Category:** security (OWASP A10 SSRF · CWE-350 · CWE-20 · CWE-918)
- **Location:** `app/internal/fetchsafe/denylist.go:38-49`
- **Status:** **RESOLVED**, verified at both call sites (hardcoded-list
  comparison and `NIU_PUBLIC_HOST` comparison) via
  `strings.TrimSuffix(strings.ToLower(host), ".")`. `security-engineer`
  independently confirmed the original bypass was real (both
  `niu.fikua.com` and `niu.fikua.com.` resolve to the identical public IP)
  before confirming the fix closes it, including an end-to-end
  `FetchPreview` probe against the trailing-dot form (correctly rejected
  as `ErrDestinationForbidden`, both for the hardcoded list and for
  `NIU_PUBLIC_HOST`). One residual was checked and correctly judged
  non-exploitable, not reopened as a finding: the double-trailing-dot form
  (`niu.fikua.com..`) still evaluates `denied=false` (the single-suffix
  `TrimSuffix` only strips one dot), but this form is an invalid DNS label
  under RFC 1035 and does not resolve — confirmed both via the system
  resolver directly and via an end-to-end `FetchPreview` call timing out
  at resolution, never reaching a connection. No destination is reachable
  behind this residual form. Test coverage:
  `TestIsDeniedHost_TrailingDotFQDN_StillDenied`,
  `TestIsDeniedHost_NIUPublicHostEnv_TrailingDotFQDN_StillDenied`, both
  covering case-insensitive + trailing-dot combinations for both call
  sites.

### F-13 — (intentionally skipped, folded into F-10, unchanged from prior round)

### F-14 — `denylist.go` service-name list completeness (UNCHANGED, correctly deferred, not a code fix expected)

- **Severity:** low (unchanged)
- **Category:** security (OWASP A05 · CWE-1188 — accepted debt,
  `design.md` ADR-02 point 8 already documents this)
- **Location:** `app/internal/fetchsafe/denylist.go:14-20`
- **Status:** **not addressed by code this round, correctly so** — the
  prior review's own suggested fix for this finding was "track as a
  `platform-engineer` backlog note," not a code change, precisely because
  the actual platform service topology this list must track lives outside
  this repository. No commit in this round touches this list, and none
  was expected. Confirmed the note is consistent with `design.md`'s R-03
  row, which already assigns this to `platform-engineer` as an ongoing
  maintenance item, not a one-time fix. Non-blocking, as before.

### F-15 — DNS-rebinding TOCTOU residual (informational, unchanged)

- **Severity:** informational (unchanged)
- **Status:** no code in this area changed this round; not re-verified,
  carried forward unchanged from the prior audit.

### F-16 — CSP `img-src` relaxation evaluation (informational, unchanged)

- **Severity:** informational (unchanged)
- **Status:** no code in this area changed this round beyond F-09 (which
  this finding anticipated as the actual gap to close — now resolved, see
  F-09 above). The CSP relaxation itself is unchanged and remains sound;
  it is now correctly a defense-in-depth layer rather than the sole
  control, since F-09 added the code-level scheme validation this finding
  called for.

### F-17 — `.idea-card` container missing `role="group"`/`aria-label` (RESOLVED this round — was blocking)

- **Severity:** was blocking → **RESOLVED**
- **Category:** spec-conformance (AC-10)
- **Location:** `app/web/js/ideas-render.js:107-119` (`newCardContainer`)
- **Status:** **RESOLVED**, verified independently by three agents
  (`code-reviewer`, `qa-engineer`, `ux-ui-designer`), each reading the
  full file end-to-end, not just the diff. `newCardContainer` now sets
  `card.setAttribute('role', 'group')` and
  `card.setAttribute('aria-label', idea.title || domainOf(idea.url))`,
  matching `proposal.md` §8.6 exactly (title-preferred, domain fallback).
  Confirmed applied consistently through the single shared factory used
  by all four state-render functions (`renderIdeaCardReady`/`Failed`/
  `Partial`/`Pending`) — no alternate card-construction path was left
  unfixed. This is the correct, complete fix for what was previously a
  genuine, invisible-to-axe-core AC-10 gap.
- **New residual finding this round (minor, non-blocking, from
  `ux-ui-designer`):** no E2E test directly asserts `role`/`aria-label`
  on `.idea-card` (e.g. `toHaveAttribute('role', 'group')`) — coverage
  today is manual-code-read only. A future regression on this exact fix
  would only be caught at the next `/audit`, not by CI. See F-22 below.

### F-18 — Delete-button `aria-label` uses domain not title (RESOLVED this round — was minor)

- **Severity:** was minor → **RESOLVED**
- **Category:** spec-conformance (accessibility fidelity)
- **Location:** `app/web/js/ideas-render.js:91-102` (`renderDeleteButton`)
- **Status:** **RESOLVED**, verified — `renderDeleteButton` now reads
  `` `Eliminar la idea «${idea.title || domainOf(idea.url)}»` ``, matching
  `proposal.md` §8.6's exact phrase (guillemets present, "la" present,
  title-preferred with domain fallback). The previous shipped string
  (domain-only, no guillemets, no "la") is fully gone.

### F-19 — Empty-link validation message wrong string (RESOLVED this round — was blocking)

- **Severity:** was blocking → **RESOLVED**
- **Category:** spec-conformance (EC-10)
- **Location:** `app/web/js/ideas-view.js:72`,
  `app/internal/ideas/service.go:57-60`
- **Status:** **RESOLVED**, verified at both layers. Client-side
  (`ideas-view.js:72`): `showError('Enganxa un enllaç abans de desar.')`
  — byte-exact match to `proposal.md` §8.3's EC-10 string, confirmed via
  direct substring comparison against the proposal source (not
  eyeballing) by both this agent and `ux-ui-designer` independently.
  Server-side twin, fixed in the same commit and correctly scoped: this
  audit's task explicitly asked to check whether `internal/ideas/service.go`
  had an identical bug, and it did — `ValidationEmpty`'s message was the
  same "Escriu...afegir" copy-paste. The fix is a real, load-bearing
  change, not cosmetic: `ux-ui-designer` traced the full error-propagation
  path and confirmed `ideas-api.js`'s `request()` forwards
  `body?.error?.message` verbatim into `ApiError.message` for any non-OK
  response, and `ideas-view.js`'s `catch` block calls `showError(err.message)`
  with no client-side override — so a direct API call bypassing the UI
  guard, or any future relaxation of the client-side whitespace check,
  would have silently resurfaced the wrong copy via this server fallback
  path. The fix commit's own description ("same latent bug, server-side")
  is accurate, not overstated. **EC-01 (invalid scheme) message** — this
  audit's task specifically asked to verify this against spec: confirmed
  byte-exact by three independent reads (this agent, `qa-engineer`,
  `ux-ui-designer`) against `proposal.md` §8.3's exact string
  ("Aquest enllaç no és vàlid — ha de començar per http:// o https://.").
  This message was correctly left untouched by the fix commit — it was
  already correct — and the commit message's claim to that effect checks
  out against a direct string comparison, not a read-through.

### F-20 — Client-side XSS non-execution test for ideas space (RESOLVED this round — qa-engineer's QA-01, was major)

- **Severity:** was major → **RESOLVED**
- **Category:** testing
- **Location:** `app/tests/e2e/specs/xss.spec.js:125-177`
- **Status:** **RESOLVED**, verified by `qa-engineer` reading the test
  body line-by-line before running it, then running it directly (9/9
  passing in the file, including the new case). The test is genuinely
  behavioral, not a weakened proxy: it embeds a real `<img src=x
  onerror="window.__xss=1">` payload in the URL field (chosen because it
  is always stored regardless of SSRF/preview outcome, and shares the same
  `.textContent`-only rendering path as title/description, confirmed by
  direct read of `ideas-render.js` lines 70/139/146), submits it through
  the real UI, and asserts four independent signals: (1) `window.__xss`
  stays unset, (2) zero `img`/`script`/`svg` elements exist in the
  rendered card, (3) the literal payload string is present as visible
  text, (4) zero unexpected `pageerror` events. This satisfies the
  project's stated testing principle (`requirements.md` §6): execute the
  attack in a real browser and assert its failure, not infer it from code
  inspection alone.

### F-21 — `-race` coverage for `tests/integration` (RESOLVED, re-confirmed this round)

- **Severity:** n/a (closed during prior compose, re-confirmed here)
- **Category:** testing
- **Location:** `app/tests/integration/...`
- **Status:** re-confirmed clean this round by `code-reviewer`'s own
  independent run: `go test ./... -race -count=1 -timeout 20m` → all
  packages `ok`, `tests/integration` at 573.934s, zero data races. Note:
  a first attempt at this same run this session produced a build failure
  in `internal/fetchsafe` (`zz_probe2_test.go: not enough arguments in
  call to FetchPreview`) — investigated and traced to a transient,
  **untracked** probe test file left in the working tree from this
  session's own empirical verification of F-09/F-12 (calling an older,
  2-argument `FetchPreview` signature that no longer matches
  `client.go`). The file self-cleaned (deleted, presumably by an editor
  autosave/cleanup) between the first and second attempts; `git status`
  confirms the working tree for `app/` is fully clean with no uncommitted
  diffs anywhere in `internal/fetchsafe`. This was **not** a defect in the
  branch — a clean re-run from an untouched working tree passed
  completely. Recorded here for the audit trail since it briefly looked
  like a real build failure.

### F-22 — No direct E2E assertion for `.idea-card`'s `role`/`aria-label` (NEW this round, minor, non-blocking)

- **Severity:** minor
- **Category:** testing (regression-safety for F-17/F-18's fix)
- **Location:** `app/tests/e2e/specs/ideas.spec.js`,
  `app/tests/e2e/specs/accessibility-audit.spec.js`
- **Observation:** `ux-ui-designer` found that neither existing E2E spec
  directly asserts `.idea-card`'s `role="group"` or `aria-label`
  attributes — both specs exercise the fallback card's title/link/alt
  text, but not the container's own ARIA semantics. axe-core's generic
  WCAG ruleset has no rule requiring a card-like list item to carry a
  named group role, so a bare `<div>` with zero ARIA would have passed
  the same two specs just as cleanly as the current, correctly-fixed
  markup.
- **Why it matters:** the only thing currently guarding F-17/F-18 against
  regression is manual code reading at the next `/audit`, not CI. This is
  a real but narrow gap — the fix itself is verified correct today.
- **Suggested fix:** add one assertion to `ideas.spec.js`, e.g.
  `await expect(card).toHaveAttribute('role', 'group')` +
  `await expect(card).toHaveAttribute('aria-label', /.+/)`. Small,
  non-blocking, can ship in a fast-follow.

## 4. Spec conformance checklist

- [x] All ACs from `requirements.md` are covered by passing tests —
      `qa-engineer`'s §2 matrix confirms **11/11**, zero partial, zero ❌.
- [x] All NFRs have a measured result — **9/9** fully verified, zero
      partial. NFR-06 (the declared blocking gate) independently
      re-confirmed by `qa-engineer`, `security-engineer`, and this agent.
- [x] `tasks.md` checklist is fully `[x]` except C-02 (backlog transition,
      correctly left for `/commit`).
- [x] Out-of-scope items in `design.md`/`tasks.md` §5 are still out of
      scope — `internal/items`, `internal/projects`, `internal/auth`,
      `internal/config` completely untouched, confirmed by an empty
      `git diff main...niu-spa-conversion --stat` against all four paths.
- [x] No new public API or schema change is undocumented in `design.md`
      §6 — unchanged from the prior audit's pass on this dimension; the
      only API-adjacent change since (server-side error message text) is
      not a contract/shape change.

## 5. Code-quality checklist (Google Engineering Practices subset)

- [x] **Design** — unchanged from the prior audit's pass; this round's
      diffs are all surgical, same-file fixes (doc comments, string
      literals, one attribute set, one scheme check, one string trim, one
      length cap, one test rename, two new tests, one committed compose
      change). No new pattern, no scope creep — every commit maps 1:1 to
      a named finding.
- [x] **Functionality** — the two remaining functional/accessibility
      deviations (F-17, F-19) from the prior round are now correct and
      verified against the exact spec text. No new functional gaps found.
- [x] **Complexity** — all fixes are minimal, in the file/function that
      already owned the relevant logic. No new abstraction introduced
      where a direct fix would do (e.g. `isHTTPOrHTTPSURL` is a 9-line
      function reusing the exact scheme-check pattern already established
      elsewhere in the package).
- [x] **Tests** — every fix ships with a test, each boundary-correct
      (truncation caps assert exact lengths; scheme rejection is
      case/whitespace/control-character-evasion-tested beyond what was
      strictly asked). F-20 closes the one remaining testing gap with a
      genuinely behavioral assertion, not a weaker proxy.
- [x] **Naming** — `TestFetchPreview_RedirectToSameHost_FirstHopRejectedBeforeConnect`
      (F-11's rename) is a textbook example of a test name matching what
      it actually proves.
- [x] **Comments** — F-02/F-05's fixes are exactly what "fix the drift"
      should look like: the comment now states real behaviour, not
      aspirational behaviour. F-11's new doc comment explicitly states
      both what the test does NOT prove and what it DOES prove — an
      unusually honest and useful pattern worth reusing.
- [x] **Style** — `gofmt -l .` clean, `go vet ./...` clean.
- [x] **Consistency** — F-03's CSRF tests are a structural mirror of the
      `/projects` pattern, not a shortcut. F-09's scheme check reuses the
      exact validation idiom already used elsewhere in `fetchsafe`.
- [x] **Documentation** — all six doc-comment/copy fixes this round
      (F-02, F-05, F-11's rename comment, F-09's inline rationale) improve
      documentation accuracy with no behavioural change bundled in —
      correctly separated concerns.

## 6. Security checklist (OWASP Top 10 + Code Review Top 10)

> Section owned by `security-engineer`. Full detail in
> `review-security-section.md`, merged as F-09..F-16 above (updated
> status) plus this summary.

- [x] **A01 Broken Access Control** — unchanged, still correct.
- [x] **A02 Cryptographic Failures** — unchanged, no new secrets
      (confirmed by grep, repeated this round).
- [x] **A03 Injection** — XSS: now fully closed including the client-side
      real-browser assertion for the ideas space (F-20). `img.src` sink:
      now scheme-validated at the source (F-09), closing the gap this
      audit's task specifically asked to verify.
- [x] **A04 Insecure Design** — SSRF threat model completeness gap
      (validates fetch destination, not recovered content) is now closed
      by F-09.
- [x] **A05 Security Misconfiguration** — CSP evaluation from the prior
      round stands, now correctly a defense-in-depth layer rather than
      the sole control (F-09 closes the gap it was compensating for).
      Rate-limit fix (F-04) now committed, no longer at risk of being
      silently dropped.
- [x] **A06 Vulnerable & Outdated Components** — unchanged, no new
      dependencies.
- [x] **A07 Identification & Authentication Failures** — unchanged.
- [x] **A08 Software & Data Integrity Failures** — unchanged, CSRF now
      additionally test-covered for `/ideas` (F-03).
- [x] **A09 Security Logging & Monitoring** — unchanged.
- [x] **A10 SSRF** — all previously-open items (F-09 second-order SSRF,
      F-12 denylist bypass) now resolved and independently re-verified.
      NFR-06's blocking gate re-confirmed non-regressed by both
      `qa-engineer` and `security-engineer` after this round's changes to
      files adjacent to the SSRF core (`ogparse.go`, `denylist.go`).

**`security-engineer`'s verdict from a pure security standpoint:
`APPROVED` — zero blocking, zero major findings open.** Residual
informational notes (scheme-vs-destination validation distinction for
future server-side sinks; the non-exploitable double-trailing-dot
denylist edge case) are documented as context for future reviews, not
outstanding work.

## 7. Action items

> No blocking items remain.

**Non-blocking, recommended (small, can ship in a fast-follow, none
gate this item's merge):**

1. Add a direct E2E attribute assertion for `.idea-card`'s
   `role="group"`/`aria-label` in `ideas.spec.js` — owner:
   `fullstack-developer`/`qa-engineer` — fixes: F-22 (new this round,
   from `ux-ui-designer`).
2. Track `denylist.go`'s `traefik-public` service list reconciliation
   against `platform-services/compose.yaml` whenever a new service is
   added there — owner: `platform-engineer` — fixes: F-14 (unchanged,
   already anticipated by `design.md` ADR-02/R-03, not expected to be a
   code change in this repo).

## 8. Sign-off

- **Approver:** `code-reviewer` (composing on behalf of
  `qa-engineer`/`security-engineer`/`ux-ui-designer`, all four in
  agreement).
- **Date:** 2026-08-03
- **Next step:** `/commit NIU-6`. All blocking findings from the prior
  round are resolved and independently re-verified by four separate
  agents; the full verification suite (`go build`, `go vet`, `gofmt -l`,
  `go test ./... -count=1`, `go test ./... -race -count=1`) is green with
  zero failures and zero data races. The two remaining action items
  (F-22, F-14) are both non-blocking and explicitly deferred — F-22 to a
  fast-follow test addition, F-14 to `platform-engineer`'s ongoing
  maintenance backlog, consistent with this project's established
  practice of not gating a merge on tracked, non-exploitable residual
  debt.
