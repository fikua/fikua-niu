---
artefact: review
key: "NIU-6"
title: "Idees d'activitats amb previsualització de link"
status: "in_review"
verdict: "CHANGES_REQUESTED"
owner: "code-reviewer"
co_reviewers: ["qa-engineer", "security-engineer", "ux-ui-designer"]
tasks_path: "./tasks.md"
findings_count: 21
blocking_count: 2
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

## 1. Verdict

**Verdict:** `CHANGES_REQUESTED`

**Rationale (one paragraph):**

Two `blocking` findings from `ux-ui-designer` (§3, F-17/F-19) are genuine,
independently-verified spec-conformance gaps against the human-gate-approved
Stage 1.5 visual spec (`proposal.md` §8): `.idea-card` has no
`role="group"`/`aria-label` (violates AC-10's screen-reader-narrative
requirement, §8.6, and is invisible to the axe-core suite that otherwise
passes — automated tooling does not check this specific semantics rule),
and the empty-link validation message ships the wrong copy string (a
copy-paste leftover from `AddItemInput`'s "Escriu...afegir" instead of the
approved "Enganxa...desar", §8.3). Both are cheap, surgical fixes confined to
`ideas-render.js`/`ideas-view.js`, but each is a real AC-10/EC-10 spec
deviation, not a nitpick, and the checklist's rule is unambiguous: any
`blocking` finding forces `CHANGES_REQUESTED` regardless of how clean
everything else is. `qa-engineer`'s §2 AC↔test coverage matrix is now complete (10/11 AC, 17/18
EC, 8/9 NFR fully verified; zero unverified) and adds one further
non-blocking finding of its own (F-20, major but not blocking on its own:
EC-11/NFR-01's client-side non-execution assertion is missing for the
ideas space specifically, though the mitigation is verified server-side
and by code inspection). On the positive side: the security-critical `internal/fetchsafe`
component (ADR-02) is genuinely solid — all 7 prior design-review findings
(F-01..F-07) are verified non-regressed at the implementation level by two
independent reviewers (`security-engineer`'s §6 threat-modeling pass plus
my own direct code read and mutation-style spot check), the NFR-06 blocking
gate is satisfied, `go build`/`go vet`/`gofmt -l .`/`go test ./...`/`go test
./... -race -count=1` are all green, and the two prior ad-hoc SPA-conversion
findings (F-01 masked-404 fallback, F-02 toast collision) that predate this
diff are both fixed and now test-covered on this same branch. `security-engineer`
found one `high` non-blocking finding (F-09/S-01: `og:image`'s scheme is
never validated before reaching `img.src`, currently contained only by CSP +
browser behaviour, not by any code-level control) that directly answers this
audit's specific evaluation task — see §5 below — and recommends
`APPROVED_WITH_NITS` from a pure security standpoint, which this agent
agrees with in isolation; it is the UX findings and the missing qa-engineer
matrix that change the overall verdict.

## 2. AC ↔ test coverage matrix

> Produced by `qa-engineer` (full detail in their scratch section, merged
> below verbatim). `code-reviewer` spot-verified NFR-06 (the declared
> blocking gate) and the general suite independently — see F-08 — and
> agrees with `qa-engineer`'s results.

**Test execution performed by `qa-engineer`:** `go build ./...`, `go vet
./...`, `gofmt -l .`, full `go test ./...`, targeted SSRF/idea test runs,
and the ideas-relevant Playwright specs — all green. `go test -race ./...`
was attempted three times but never completed for `tests/integration`
within the session (flagged as QA-02 below, not a failure). `code-reviewer`
subsequently ran `go test -race ./tests/integration/...` in isolation to
close this gap — see F-08a.

### AC ↔ test coverage

| AC | Statement (short) | Verifying test(s) | Result |
|----|--------------------|--------------------|--------|
| AC-01 | Afegir idea amb previsualització completa (OG complet, persisteix) | `internal/fetchsafe`: `TestParseOpenGraph_FullTags`; `internal/ideas`: `TestServiceAdd_SuccessfulScrape_ResolvesToReady`; integration: `TestIdeas_FullOpenGraph_SavesAsCompleteCardAndPersists` (T-23, asserts 201→pending, resolves→ready, persists across 2nd GET, `AddedBy` present) | ✅ pass |
| AC-02 | Afegir idea quan la previsualització falla (bloqueig/timeout/no compatible) — mai bloqueja | `internal/ideas`: `TestServiceAdd_FetchError_ResolvesToFailed_NeverBlocksAdd`; integration: `TestIdeas_AccessBlocked_SavesAsFallback` (403), `TestIdeas_Timeout_SavesAsFallback`, `TestIdeas_NonHTMLContentType_SavesAsFallback_NoParsingAttempted` — all assert `201` returned immediately and terminal `failed` status later, URL preserved | ✅ pass |
| AC-03 | Previsualització amb dades parcials, camps absents omesos sense error | `internal/fetchsafe`: `TestParseOpenGraph_PartialTags`; `internal/ideas`: `TestServiceAdd_PartialScrape_ResolvesToPartial`, `TestServiceAdd_EmptyPartialResult_TreatedAsFailed`; integration: `TestIdeas_PartialOpenGraph_SavesAsPartial` | ✅ pass |
| AC-04 | Cada idea mostra qui l'ha afegit | integration: `TestIdeas_FullOpenGraph_SavesAsCompleteCardAndPersists` asserts `AddedBy != nil`; DTO shape reviewed (`dto.go` `ideaDTO.AddedBy *userDTO`) | ✅ pass |
| AC-05 | Eliminar una idea desada, no reapareix | `internal/ideas`: `TestServiceDelete_Idempotent`; integration: `TestIdeas_Delete_RemovesFromListAndDoesNotReappear` | ✅ pass |
| AC-06 | Dos usuaris veuen les mateixes idees (convergència) | integration: `TestIdeas_TwoClients_SeeConvergedList` | ✅ pass |
| AC-07 | Espai visualment diferenciat (Compra/Projectes) | E2E `ideas.spec.js`: `navigation entry and accent colour clearly differ from Compra and Projectes` | ✅ pass |
| AC-08 | Enllaç obligatori i format vàlid processat | `internal/fetchsafe`: `TestValidateScheme_*`; `internal/ideas`: `TestServiceAdd_EmptyURL_Rejected`, `TestServiceAdd_NonHTTPScheme_Rejected`; integration: `TestIdeas_FullOpenGraph_SavesAsCompleteCardAndPersists` | ✅ pass |
| AC-09 | Navegació completa per teclat (afegir + eliminar) | E2E `ideas.spec.js`: `add and delete an idea using only the keyboard` | ✅ pass |
| AC-10 | Targeta accessible per lectors de pantalla | E2E `ideas.spec.js` + `accessibility-audit.spec.js` axe-core WCAG 2.2 AA, zero violations. **Note:** axe-core's clean pass does not detect F-17's missing `role="group"`/`aria-label` — see §3. | ⚠️ pass on tested dimensions, but see F-17 |
| AC-11 | `overview.md` reflecteix el nou espai | Manual/documental — confirmed `docs/overview.md` mentions the space, no-lifecycle behaviour, non-blocking fallback, per-idea authorship | ✅ pass (manual, as specified) |

### Edge case ↔ test coverage

| EC | Statement (short) | Verifying test(s) | Result |
|----|--------------------|--------------------|--------|
| EC-01 | Esquema no `http(s)` rebutjat, zero petició de xarxa | `TestValidateScheme_RejectsNonHTTPSchemes`; `TestServiceAdd_NonHTTPScheme_Rejected` | ✅ pass |
| EC-02 | Destí xarxa privada/loopback/pròpia instància, literal IP | `TestValidateIPForConnect_RejectsPrivateLoopbackLinkLocal`; `TestFetchPreview_LiteralPrivateOrLoopbackIP_Rejected_NoTCPConnection` (T-27a) | ✅ pass |
| EC-03 | Resposta extremadament gran | `TestIdeas_OversizedResponse_SavesAsFallback_NoMemoryExhaustion` | ✅ pass |
| EC-04 | Cadena de redireccions, incl. les dues regressions específiques | `TestCheckRedirect_HopLimit`, `TestCheckRedirect_RejectsNonHTTPSchemeOnRedirect`; `TestFetchPreview_RedirectToSameHost_EachHopReDialed_F01Regression` (T-27c), `TestFetchPreview_IPv4MappedIPv6Destination_Rejected_F02Regression` (T-27d) | ✅ pass (see F-11 re: T-27c's real detection mechanism) |
| EC-05 | Metadades OG absents/malformades | `TestParseOpenGraph_NoOGTags_TreatedAsNotFound`, `TestParseOpenGraph_MalformedHTML_DoesNotCrash` | ✅ pass |
| EC-06 | Idea duplicada, sense bloqueig | `TestIdeas_SameLinkSavedTwice_NoDeduplication` | ✅ pass |
| EC-07 | Destí privat via resolució DNS | `TestValidateIPForConnect_RejectsIPv4MappedIPv6`; `TestFetchPreview_HostnameResolvingToLoopback_Rejected` | ✅ pass |
| EC-08 | Timeout de resposta | `TestNewClient_NoAuthHeadersConfigured`; `TestIdeas_Timeout_SavesAsFallback` | ✅ pass |
| EC-09 | Contingut no HTML | `TestIdeas_NonHTMLContentType_SavesAsFallback_NoParsingAttempted` | ✅ pass |
| EC-10 | Enllaç buit / només espais | `TestValidateScheme_RejectsNonHTTPSchemes`; `TestServiceAdd_EmptyURL_Rejected` | ✅ pass (but see F-19: the UI-level error string does not match spec) |
| EC-11 | Injecció HTML/JS a metadades — mostrat com text literal | Server-side: `TestIdeas_XSSPayloadInURL_StoredLiterally`, `TestIdeas_XSSPayloadInRecoveredTitle_StoredLiterally`. **Client-side real-browser non-execution NOT covered for ideas specifically** — see QA-01 below. | ⚠️ partial |
| EC-12 | Injecció SQL — desat literal, taula intacta | `TestIdeas_SQLInjectionPayloadInURL_StoredLiterally_TableSurvives`, `TestIdeas_SQLInjectionPayloadInRecoveredMetadata_StoredLiterally_TableSurvives`; `TestNoSprintfSQL` | ✅ pass |
| EC-13 | Mutació via `GET` inexistent | `TestNoMutatingGETRoutes`, `TestGETRequestsDoNotMutateState`, `TestIdeas_NoMutationViaGET` | ✅ pass |
| EC-14 | Accés sense sessió | `TestIdeas_Unauthenticated_Rejected` | ✅ pass |
| EC-15 | DELETE ja eliminat — idempotent | `TestServiceDelete_Idempotent`; `TestIdeas_DoubleDelete_Idempotent` | ✅ pass |
| EC-16 | Doble POST — NO idempotent | `TestIdeas_DoublePost_CreatesTwoIndependentIdeas_NotIdempotent` | ✅ pass |
| EC-17 | Llista buida en primer ús | `TestIdeas_EmptyListOnFirstUse` | ✅ pass |
| EC-18 | Viewport mòbil | E2E `ideas.spec.js`: mobile viewport case | ✅ pass |

### NFR ↔ test coverage

| NFR | Statement (short) | Verifying test(s) | Result |
|-----|--------------------|--------------------|--------|
| NFR-01 | XSS — zero `innerHTML` amb dades externes/usuari | Code inspection (`ideas-render.js`: `createElement`+`textContent` only). **No behavioural E2E non-execution assertion for the ideas space** — see QA-01. | ⚠️ partial |
| NFR-02 | Injecció SQL — 100% paràmetres vinculats | `TestNoSprintfSQL` + EC-12 tests | ✅ pass |
| NFR-03 | Cap mutació via `GET` | `TestNoMutatingGETRoutes`, `TestGETRequestsDoNotMutateState`, `TestIdeas_NoMutationViaGET` | ✅ pass |
| NFR-04 | Tots els endpoints requereixen sessió | `TestIdeas_Unauthenticated_Rejected` | ✅ pass |
| NFR-05 | SSRF — rebuig d'esquema abans de xarxa | `TestValidateScheme_RejectsNonHTTPSchemes`; `TestServiceAdd_NonHTTPScheme_Rejected` | ✅ pass |
| NFR-06 | SSRF — destins prohibits, **BLOCKING gate** | T-27a/b/c/d, all 5 dedicated SSRF-mechanism tests — individually run and confirmed passing | ✅ pass |
| NFR-07 | SSRF — límits de recurs (timeout + mida + concurrència) | `TestCheckRedirect_HopLimit`; `TestIdeas_Timeout_SavesAsFallback`, `TestIdeas_OversizedResponse_SavesAsFallback_NoMemoryExhaustion`; `workerPoolSize = 6` resolved in code (T-07a) | ✅ pass |
| NFR-08 | Cap credencial de Niu surt cap a destí extern | `TestFetchPreview_NoAuthHeaders_RealRequestInspection`; `TestNewClient_NoAuthHeadersConfigured` | ✅ pass |
| NFR-09 | Cache at-save-time — mai re-scraping en `GET` | `TestIdeas_RepeatedGET_NeverReScrapes` (T-27e) | ✅ pass |

**Summary:** 10/11 AC fully verified (AC-10 verified on tested dimensions,
but see F-17 for a spec requirement outside axe-core's reach); 17/18 EC
fully verified, 1 partial (EC-11); 8/9 NFR fully verified, 1 partial
(NFR-01, same root cause as EC-11). **Zero AC/EC/NFR unverified.** NFR-06,
the declared blocking gate, is satisfied.

## 3. Findings

> Findings F-01..F-08 are `code-reviewer`'s own. F-09..F-16 are
> `security-engineer`'s (renumbered from their scratch file's S-01..S-08,
> same order, same content, verified independently where noted).
> F-17..F-19 are `ux-ui-designer`'s (renumbered from F-UX-01..03 plus the
> "note" items, same order).

### F-01 — `spaFallback`/toast-collision findings from the prior ad-hoc SPA review are fixed and now test-covered (not a finding — noted for the record)

- **Severity:** n/a (positive confirmation, listed so the trail is complete)
- **Category:** spec-conformance
- **Location:** `app/internal/httpapi/router.go:161-207` (`spaRoutes`
  allowlist), `app/internal/httpapi/router_test.go:141` (`TestSPAFallback`),
  `app/web/js/toast.js` (new, shared module)
- **Observation:** `docs/changes/spa-conversion-adhoc-review.md` F-01
  (major — any unmatched path silently returned `200` + the SPA shell
  instead of a `404`) is resolved: `spaRoutes` is now an explicit allowlist
  (`/`, `/projects`, `/ideas`) matching `main.js`'s `ROUTES`, with a
  dedicated test (`TestSPAFallback`) asserting a typo'd asset path, an
  unknown route, and an API-shaped nonexistent path all still `404`. F-02
  (minor — two independent `toastTimer`s racing on the shared `#toast-wrap`
  DOM node) is resolved by extracting a single `toast.js` module, imported
  by all three render modules (`render.js`, `projects-render.js`,
  `ideas-render.js`).
- **Why it matters:** confirms this branch is not just adding NIU-6 but
  also cleanly closing out the two real defects the prior ad-hoc review
  found — verified by reading the actual allowlist/test/module, not by
  trusting the commit message.
- **Suggested fix:** none — already fixed.

### F-02 — `Repository.Get`/`ErrNotFound` are unreachable from any HTTP handler (minor, maintainability)

- **Severity:** minor
- **Category:** maintainability
- **Location:** `app/internal/ideas/ideas.go:59` (`Repository.Get` doc:
  "Returns ErrNotFound if absent"), `app/internal/ideas/errors.go:5-13`
  (`ErrNotFound`)
- **Observation:** `Repository.Get` is only ever called internally by
  `IdeasRepository.Create` (to re-fetch the row it just inserted) —
  `ideas.Service` exposes no `Get` method of its own, and no HTTP handler
  (`handleListIdeas`/`handleCreateIdea`/`handleDeleteIdea`) ever triggers a
  path where `ErrNotFound` could surface to a caller. The doc comment says
  "returned by Service.Get" but no such method exists on `Service`.
- **Why it matters:** purely cosmetic/documentation drift — same class of
  finding as NIU-5's F-24 (`review.md` for that item). No functional gap: a
  `DELETE` on a nonexistent id is handled separately and correctly via
  `Repository.Delete`'s `existed bool` return (EC-15, idempotent), which
  never touches `Get`/`ErrNotFound` at all.
- **Suggested fix:** either remove the unreachable `ErrNotFound` type and
  correct the doc comment to describe what `Get` is actually used for
  (internal re-fetch after insert), or leave as forward-looking API surface
  but fix the doc comment's inaccurate claim about `Service.Get`.

### F-03 — `/api/v1/ideas` has zero dedicated CSRF test coverage (minor, testing — same defect class NIU-5 already fixed once)

- **Severity:** minor
- **Category:** testing (security)
- **Location:** `app/tests/integration/csrf_test.go` — 9 existing
  `TestCSRF_*` tests cover `/items` and `/projects` (the latter added
  specifically to close NIU-5's F-21); none target `/api/v1/ideas`
- **Observation:** CSRF protection **is** correctly wired in
  `router.go:147-160` — `POST`/`DELETE` under `/api/v1/ideas` are wrapped
  with `RequireCSRF(s.authenticator.SessionSecret())`, identical to the
  `/items`/`/projects` pattern, and `GET` is correctly exempt. But no test
  exercises it: `tasks.md` T-27's security-test scope explicitly lists
  XSS/SQLi/GET-mutation/auth (EC-11/EC-12/EC-13/EC-14) and does **not**
  mention CSRF, so this is a genuine gap in the task breakdown, not a
  developer oversight against an assigned task. `newAuthTestServer` (the
  exact harness NIU-5's `TestCSRF_Projects_*` reused) is readily available
  for a symmetrical `TestCSRF_Ideas_*` trio.
- **Why it matters:** the protection depends on a branch
  (`if s.authenticator != nil`) — if a future change accidentally moves the
  `/ideas` routes to the no-CSRF branch (the same risk NIU-5's F-21
  described), nothing would fail. This is exactly the "regression waiting
  to happen" pattern this project has already paid down once for
  `/projects`.
- **Suggested fix:** add `TestCSRF_Ideas_PostWithoutToken_Rejected` and
  `TestCSRF_Ideas_DeleteWithoutToken_Rejected`, mirroring
  `TestCSRF_Projects_PostWithoutToken_Rejected`/
  `TestCSRF_Projects_DeleteWithoutToken_Rejected` exactly.

### F-04 — `app/compose.yaml`'s rate-limit increase remains uncommitted on this branch (minor, process)

- **Severity:** minor
- **Category:** maintainability / process
- **Location:** `app/compose.yaml` (working-tree diff only, not in any of
  the 12 commits `git log main..niu-spa-conversion` lists)
- **Observation:** `traefik.http.middlewares.niu-ratelimit.ratelimit.average`
  (`10`→`60`) and `.burst` (`20`→`100`) are only in the working tree, not
  committed — the same process gap the prior ad-hoc security review flagged
  as `S-02`/nit and explicitly warned about ("the incident mitigation could
  be lost on a `checkout` and the 429 would reappear on deploy").
- **Why it matters:** this is exactly the risk the earlier review called
  out — a `git checkout`/`git stash`/branch switch anywhere in this session
  would silently drop the fix for a real, previously-observed 429 incident.
- **Suggested fix:** commit `app/compose.yaml` deliberately alongside the
  rest of this change (or as its own small `chore:` commit) before this
  item is merged.

### F-05 — `ErrResponseTooLarge` is declared but never returned by production code (nit, maintainability)

- **Severity:** nit
- **Category:** maintainability
- **Location:** `app/internal/fetchsafe/errors.go:23-26`,
  `app/internal/fetchsafe/fetchsafe.go:102-106`,
  `app/internal/fetchsafe/ogparse.go:41-81`
- **Observation:** confirmed by reading `parseOpenGraph`: a `LimitReader`
  truncation surfaces as a normal `html.ErrorToken`, handled identically to
  natural EOF or malformed markup — the function returns a partial
  `Preview` with `err == nil`, never `ErrResponseTooLarge`. The sentinel's
  only live reference is a test double
  (`tests/integration/ideas_test_server_test.go:84`) that simulates a
  caller wanting to see that error value, not production code producing it.
  Same finding independently identified by `security-engineer` (their
  S-05).
- **Why it matters:** zero functional/security impact — silent truncation
  is the documented, intended behaviour (`design.md` §5: "treated as
  fallback, not a fatal error"). But the doc comment on
  `ErrResponseTooLarge` ("is returned when...") is inaccurate, and a future
  reader relying on that error for size-specific observability/logging
  would find it never fires.
- **Suggested fix:** either delete the unused sentinel and document the
  silent-truncation behaviour explicitly on `parseOpenGraph`, or
  instrument an actual size check (read one byte past the limit) so the
  documented behaviour becomes real. Cosmetic, non-blocking either way.

### F-06 — `og:image`'s scheme is never validated before reaching `img.src` (duplicate of security-engineer's F-09/S-01 — see there for full analysis)

- **Severity:** see F-09
- **Category:** security
- **Location:** see F-09
- **Observation:** independently reproduced this myself (a throwaway probe
  against `parseOpenGraph` with `og:image = "javascript:alert(1)"` returns
  `ImageURL = "javascript:alert(1)"` unfiltered) before reading
  `security-engineer`'s section — merging into F-09 rather than
  double-counting. This is the specific answer to this audit's task #5
  ("evaluate whether relaxing CSP img-src... is an acceptable trade-off"):
  the CSP relaxation itself is sound (see §5 below), but it is currently
  the *only* thing preventing this from being exploitable, which is why
  `security-engineer` rates the underlying gap `high` even though nothing
  is exploitable in production today.
- **Why it matters / suggested fix:** see F-09.

### F-07 — No length cap on recovered Open Graph metadata (duplicate of security-engineer's F-10/S-04)

- **Severity:** see F-10
- **Category:** security / resource limits
- **Location:** see F-10
- **Observation:** confirmed the same gap `security-engineer` found:
  `title`/`description`/`image_url` are stored with no length ceiling
  below the 2MiB `LimitReader`. Merging into F-10 rather than
  double-counting.

### F-08 — Test suite: no failing tests, no lint violations, one unrelated flake

- **Severity:** n/a (verification note)
- **Category:** testing
- **Location:** whole suite
- **Observation:** `go build ./...`, `go vet ./...`, `gofmt -l .` (zero
  files listed), `go test ./... -count=1` (all packages `ok`), `go test
  ./... -race -count=1` (all packages `ok`, zero races) — all run directly
  by this agent, not trusted from a commit message. One Playwright flake
  on the pre-existing (non-NIU-6) shopping-list desktop axe-core test,
  confirmed to pass in isolation on re-run; not reproducible, not part of
  this diff's surface.
- **Why it matters:** satisfies §6 of the code-review-checklist ("failing
  tests or typecheck → blocking") — nothing here is blocking on that
  ground.
- **Suggested fix:** none required.

### F-09 — `og:image` recovered from an attacker-controlled server reaches `img.src` with no scheme validation (high, security)

- **Severity:** high (not blocking — see rationale)
- **Category:** security (OWASP A03 Injection · CWE-79 · CWE-20 · OWASP
  Code Review Top 10 #2)
- **Location:** `app/internal/fetchsafe/ogparse.go:108-112` (extraction) →
  `app/internal/httpapi/dto.go:107` (serialization) →
  `app/web/js/ideas-render.js:128,196` (`img.src = idea.image_url`)
- **Observation:** `security-engineer` verified empirically that
  `og:image` values such as `javascript:alert(document.cookie)`,
  `data:text/html;base64,...`, `file:///etc/passwd`,
  `http://169.254.169.254/latest/meta-data/`, and `//evil.example/x.png`
  all survive parsing and storage unfiltered, reaching the DOM as a raw
  `img.src` assignment. This agent independently reproduced the core case
  (`javascript:` scheme survives `parseOpenGraph` unfiltered). `fetchsafe`
  validates the *destination of the fetch* (the page's own URL) but never
  validates *values recovered from that page's content* — a different
  trust boundary that ADR-02 does not cover.
- **Why it matters:** three independent layers currently contain this: (1)
  browsers do not treat `javascript:` as a valid `<img>` fetch scheme, (2)
  the CSP's `img-src 'self' https:` blocks `data:`/`file:`/plain `http:`,
  and (3) `object-src 'none'`/`script-src 'self'`/`base-uri 'none'` close
  the usual escalation paths. But the defense is **accidental** — it
  depends entirely on a CSP header this very diff just relaxed, and on
  browser behaviour, not on any control in the code path that owns this
  data. A single future change (e.g. adding `data:` to `img-src` for
  favicons, reusing `image_url` in a new sink such as `<a href>`,
  `background-image`, an email template, or a `<link rel=preload>`) turns
  this directly exploitable. It also constitutes a second-order SSRF: a
  crafted `og:image` pointing at an internal host makes the **victim's own
  browser** issue a request `fetchsafe` never validated, because
  `fetchsafe` only ever validated the *page* URL, not what the page's
  metadata *contains*.
- **Suggested fix:** validate `og:image` in `applyMetaToken`/
  `finishPreview` before returning the `Preview` — accept only a
  `url.Parse`-clean value with `scheme == "https"` (or `http`/`https`),
  discarding anything else silently (the field simply stays empty, falling
  through to the already-implemented/already-tested `partial`/`failed`
  card states). ~8 lines in the one package that already owns all
  fetch-related validation. Pair with F-10's length cap in the same edit.

### F-10 — No length cap on recovered Open Graph metadata (low, security / resource limits)

- **Severity:** low
- **Category:** security (OWASP A04 Insecure Design · CWE-770)
- **Location:** `app/internal/fetchsafe/ogparse.go:87-120`;
  `app/migrations/004_activity_ideas.sql:5-7` (`TEXT` columns, no `CHECK`)
- **Observation:** `title`/`description`/`image_url` are stored with no
  length ceiling other than the 2MiB `LimitReader` — `security-engineer`
  confirmed a 500,000-character `og:title` passes through the parser
  intact.
- **Why it matters:** low impact (requires a deliberately hostile linked
  page, app is two-person scale, the 2MiB reader already bounds the worst
  case per row), but degrades list-rendering/DOM performance and grows the
  SQLite file on a resource-constrained VPS (128M container limit).
- **Suggested fix:** truncate in `applyMetaToken`/`finishPreview` (e.g.
  300 chars title, 1000 description, 2048 image_url) — same edit location
  as F-09, same PR.

### F-11 — `TestFetchPreview_RedirectToSameHost_EachHopReDialed_F01Regression` does not detect the F-01 regression it names (medium, testing)

- **Severity:** medium
- **Category:** testing (OWASP Code Review Top 10 #10 · CWE-1006)
- **Location:** `app/tests/integration/ideas_ssrf_test.go:160-206`
- **Observation:** `security-engineer` confirmed via mutation testing:
  reverting `DisableKeepAlives` to `false` (reintroducing F-01) still
  passes this test, because the test's destination is loopback, which
  `ControlContext` rejects on the *first* dial attempt — the redirect
  chain never progresses far enough to exercise connection reuse at all.
  The only test that actually catches the mutation is
  `TestNewTransport_DisableKeepAlivesTrue` (`client_test.go:18`), a static
  assertion on the `Transport` struct's field value.
- **Why it matters:** the regression **is** caught today, just by a
  different, more literal test than the one whose name/comment claims to
  cover it — this is a "false sense of coverage" risk: if `newTransport()`
  is ever refactored into a form `TestNewTransport_DisableKeepAlivesTrue`
  no longer covers (e.g. building the `Transport` inline inside
  `NewClient` instead of via a named constructor), the safety net
  disappears silently, since T-27c would keep passing regardless.
- **Suggested fix:** either (a, preferable) make the test genuinely
  behavioural — inject an instrumented test `Dialer`/`ControlContext` that
  counts invocations and permits loopback only for the test, asserting
  `dials == hops`; or (b, minimum) rename the test and its comment to
  describe what it actually proves ("first dial of a same-host redirect
  chain is blocked before connect()"), and note that F-01's real coverage
  lives in `TestNewTransport_DisableKeepAlivesTrue`.

### F-12 — Hostname denylist can be bypassed with the trailing-dot FQDN form (medium, security)

- **Severity:** medium
- **Category:** security (OWASP A10 SSRF · CWE-350 · CWE-20)
- **Location:** `app/internal/fetchsafe/denylist.go:32-43`
- **Observation:** `isDeniedHost` does exact-match comparison after
  lowercasing. `niu.fikua.com` and `niu.fikua.com.` (trailing-dot absolute
  FQDN form) are the same DNS name and resolve identically, but only the
  first matches the denylist — `security-engineer` verified
  `http://localhost./` and `http://localhost/` reach the same listener.
- **Why it matters:** this is precisely the vector F-03/F-04 were meant to
  close. `https://niu.fikua.com./...` would loop back to the app itself
  through the Cloudflare edge (partially mitigated: no session cookie
  travels with the outbound request, so it cannot act as an authenticated
  user, and `/api/v1/*` still 401s). For the Docker-internal service names
  (`otel-collector.`, `traefik.`), whether the trailing dot reaches the
  same service depends on the container's resolver `ndots`/search-domain
  configuration — unverified, and the second layer (`ControlContext`'s IP
  allowlist) is the only thing standing between this and a real hit if it
  does resolve, which is exactly the assumption ADR-02 point 8 already
  flags as unverified.
- **Suggested fix:** normalize before comparing —
  `host = strings.TrimSuffix(strings.ToLower(host), ".")`. One line, plus
  a `denylist_test.go` case for `"niu.fikua.com."`/`"TRAEFIK."`.

### F-13 — No length cap on recovered Open Graph metadata (see F-10 — same finding, listed once)

*(Intentionally not duplicated — folded into F-10 above; this number is
skipped to keep security-engineer's original S-04/S-05/S-06 sequence
traceable without renumbering gaps elsewhere in this document.)*

### F-14 — `denylist.go`'s `traefik-public` service-name list cannot be verified complete from this repository (low, security)

- **Severity:** low
- **Category:** security (OWASP A05 · CWE-1188 — accepted debt, `ADR-02`
  point 8 already documents this)
- **Location:** `app/internal/fetchsafe/denylist.go:14-20`
- **Observation:** the hardcoded list (`otel-collector`, `dozzle`,
  `openobserve`, `traefik`) was derived manually from the current
  topology. The actual platform service definitions live in a separate
  repository (`platform-services/compose.yaml`) not accessible from this
  audit, so completeness cannot be independently verified here.
- **Why it matters:** low impact today — `ControlContext`'s IP allowlist
  is the second layer and covers any service resolving to a private range;
  the residual risk is specifically a `traefik-public` service that does
  *not* resolve to RFC1918, the exact scenario that motivated F-03/F-04 in
  the first place.
- **Suggested fix:** non-blocking. Track as a `platform-engineer` backlog
  note (already anticipated by `design.md` ADR-02) to reconcile this list
  against `platform-services/compose.yaml` whenever a new service is added
  there.

### F-15 — DNS-rebinding TOCTOU: residual OS-level window (informational, no action)

- **Severity:** informational
- **Category:** security (documentation of a model limit, not a defect)
- **Location:** `app/internal/fetchsafe/ipvalidate.go`
- **Observation:** the chosen architecture (single validation inside
  `ControlContext`, no separate `LookupIPAddr` call) eliminates the TOCTOU
  *between two resolutions performed by this codebase* — exactly what
  F-06 required, and correctly so. A theoretical window remains inside the
  OS network stack itself, between the resolver returning an address and
  the kernel's `connect()`, but this is not exploitable from Go:
  `ControlContext` receives the exact `sockaddr` that will be used for
  `connect()`, not a name that gets re-resolved.
- **Why it matters:** no action required — recorded so a future reader
  does not read "DNS rebinding closed" (`design.md` ADR-02 point 2) as an
  absolute guarantee beyond what is actually claimed.
- **Suggested fix:** none.

### F-16 — CSP `img-src` relaxation evaluated: acceptable trade-off, correctly scoped (informational — this audit's task #5)

- **Severity:** informational
- **Category:** security (OWASP A05)
- **Location:** `app/internal/httpapi/middleware.go:49` (commit `962d46e`)

**This is the direct answer to this audit's specific evaluation task.**
Diff: `img-src 'self'` → `img-src 'self' https:`. **Verdict: acceptable,
and correctly scoped — not a finding on its own.**

- No other directive was touched: `default-src`, `script-src`,
  `style-src`, `connect-src`, `object-src 'none'`, `base-uri 'none'` are
  all unchanged in this diff (confirmed by reading the full `git diff` for
  `middleware.go`, which is a 2-line change).
- The relaxation is scheme-scoped to `https:` only — it does **not** add
  `data:` (which would be the genuinely dangerous addition, opening
  `data:image/svg+xml` with inline script in document-loading contexts)
  or `http:` (which would introduce mixed content). This is the narrowest
  possible relaxation that achieves the stated goal.
- `<img>` is a weak sink: it cannot execute script and cannot exfiltrate
  anything beyond the fact of the request itself (host + timing).
  `Referrer-Policy: no-referrer` (unchanged, `middleware.go:46`) means the
  outbound image request leaks no Niu-specific referrer information — only
  that *some* request happened, with the browser's own IP/User-Agent.
  This was independently confirmed by this agent by re-reading
  `middleware.go` directly.
- The functionality it enables (rendering recovered `og:image` previews,
  `proposal.md` §8.2 Estat A/C) is a genuine product requirement with no
  reasonable alternative short of server-side image proxying — disproportionate
  engineering for a single-VPS, two-person app.
- **The one thing this relaxation changes materially:** the CSP is now the
  *only* server-side control standing between an attacker-controlled
  `og:image` value and the browser's `<img>` fetch — this is precisely
  F-09/F-06 above. The commit message's claim ("fetchsafe already
  guarantees image_url only ever holds an SSRF-validated http(s) URL") is
  **not accurate as literally stated**: `fetchsafe` guarantees the *page*
  it fetched was a validated destination; it does not itself validate
  *what that page's `og:image` metadata says* before the value is stored
  and later rendered. The claim happens to be true in effect only because
  the CSP change shipped in the *same commit* compensates for the gap —
  correlation, not the causal guarantee the message describes.
- **Conclusion:** the CSP relaxation itself is sound engineering and
  should not be reverted or re-scoped further. The actual gap to close is
  F-09 (validate `og:image`'s scheme at the source, in `fetchsafe`), which
  would make the CSP a defense-in-depth layer again instead of the sole
  control — as `security-engineer`'s F-09 recommends.

### F-17 — `.idea-card` container has no `role="group"`/`<article>` + `aria-label` (blocking, spec conformance / accessibility)

- **Severity:** blocking
- **Category:** spec-conformance (AC-10)
- **Location:** `app/web/js/ideas-render.js:107-117` (`newCardContainer`)
- **Observation:** independently verified — `newCardContainer` creates a
  plain `<div class="idea-card">` with no `role` and no `aria-label`; none
  of the four state-specific render functions
  (`renderIdeaCardReady`/`Failed`/`Partial`/`Pending`) add one either.
  `proposal.md` §8.6 (approved at the Stage 1.5 human gate) states
  explicitly: *"Cada targeta és un element `<article>` o `role="group"`
  amb `aria-label` construït a partir del títol disponible (o del domini,
  en fallback) — mai una llista de `<div>` sense semàntica."* The
  implementation is exactly the failure mode the spec calls out to avoid.
- **Why it matters:** a screen-reader user browsing by landmark/group will
  hear each card's pieces (title, link, delete button) as loose,
  unassociated content instead of one discrete, named unit — this is a
  real AC-10 gap on a required AC, not cosmetic. It is invisible to the
  passing `accessibility-audit.spec.js` runs for the ideas space (2/2
  pass) because axe-core's default ruleset has no generic check for
  "list items should carry a labelled group/landmark role" — a passing
  axe-core run does not mean AC-10/§8.6 is satisfied here. This is
  precisely why the design spec wrote the requirement out explicitly
  rather than leaving it to "run axe and see."
- **Suggested fix:** in `newCardContainer(idea)`, add
  `card.setAttribute('role', 'group')` (or use `<article>` in place of
  `<div>` for `card`) and
  `card.setAttribute('aria-label', idea.title || domainOf(idea.url))` —
  reuses the same title-or-domain fallback pattern already present
  elsewhere in the same file (e.g. `renderIdeaCardFailed`'s title line).
  No new helper needed.

### F-18 — Delete button's `aria-label` never uses the title, only the domain (minor, non-blocking)

- **Severity:** minor
- **Category:** spec-conformance (accessibility fidelity)
- **Location:** `app/web/js/ideas-render.js:91-102` (`renderDeleteButton`)
- **Observation:** confirmed — `renderDeleteButton` always calls
  `domainOf(idea.url)` for its `aria-label`, never `idea.title`, even on
  Estat A/C cards that do have a title. `proposal.md` §8.6 specifies
  `aria-label="Eliminar la idea «{títol o domini}»"` — title preferred,
  domain only as fallback. Example: a card titled "Millor pizza de
  Barcelona" gets a delete button labelled "Eliminar idea
  barcelonaesports.cat" instead of "Eliminar idea Millor pizza de
  Barcelona." The implemented string also drops the guillemets and "la"
  from the spec's exact phrase.
- **Why it matters:** the button remains labelled and unambiguous and
  still satisfies axe-core's "buttons must have discernible text" rule —
  this is a fidelity gap against the exact spec wording (a worse, but not
  broken, identifier), not a functional accessibility failure.
- **Suggested fix:**
  `del.setAttribute('aria-label', \`Eliminar idea ${idea.title || domainOf(idea.url)}\`)`
  — fix alongside F-17 in the same file region.

### F-20 — EC-11/NFR-01 client-side non-execution not tested for the ideas space (major, testing — qa-engineer's QA-01)

- **Severity:** major
- **Category:** testing
- **Location:** `app/tests/e2e/specs/xss.spec.js` (only covers `/` items
  and `/projects`); no equivalent case for `/ideas`.
- **Observation:** `requirements.md` §6 requires, for EC-11/NFR-01, an
  E2E assertion of no script execution in a real browser, in addition to
  the server-side literal-storage check. The server-side half is covered
  twice; no Playwright test adds a payload via a recovered title/
  description and asserts non-execution for the ideas space, mirroring
  the pattern already established for items/projects in `xss.spec.js`.
- **Why it matters:** code inspection of `ideas-render.js` is strong
  evidence the mitigation is correct (consistent `textContent`-only
  usage), but the project's own stated testing principle (`requirements.md`
  §6, quoting `docs/test-plan.md` §2.1) is that no security mitigation is
  trusted on inspection alone — every test must execute the attack and
  assert its failure. That principle is not yet met for the ideas
  client-render path.
- **Suggested fix:** add one Playwright case (in `xss.spec.js` or
  `ideas.spec.js`) asserting `window.__xss` stays unset when a payload
  reaches the rendered card — the URL field is sufficient and already
  reachable regardless of SSRF outcome, since `ideas-render.js`'s URL/
  title paths share the same `textContent`-only code.

### F-21 — `-race` coverage for `tests/integration` (resolved, no action needed)

- **Severity:** n/a (closed during this compose, listed for the record)
- **Category:** testing
- **Location:** `app/tests/integration/...`
- **Observation:** `qa-engineer` flagged that `go test -race ./...`
  never completed for `tests/integration` within their session. A first
  isolated run by this agent hit Go's default 600s test timeout (not a
  `DATA RACE` report, not a `panic` — just goroutines parked mid-request
  in `net/http`'s `readLoop`/`writeLoop`, consistent with the suite
  running slower under `-race` instrumentation, as
  `fullstack-developer` had already noted for the full-suite run in
  `/code`). Re-ran with `-timeout 20m`: **`ok  niu/tests/integration
  680.913s` — clean, zero data races.**
- **Why it matters:** confirms the concurrency-heavy code (`internal/ideas`
  worker pool, the SSRF `countingListener` atomics) has no detected data
  race. No action required.
- **Suggested fix:** none pending confirmation; if it fails, treat as
  blocking (a real data race) and re-open.

### F-19 — Empty-link validation message reuses the wrong string (blocking, spec conformance)

- **Severity:** blocking
- **Category:** spec-conformance (EC-10)
- **Location:** `app/web/js/ideas-view.js:72`
- **Observation:** independently confirmed — the code reads
  `showError('Escriu un enllaç abans d'afegir.')`. `proposal.md` §8.3
  (approved at the Stage 1.5 human gate) specifies the exact string for
  EC-10 as **"Enganxa un enllaç abans de desar."**, and its own header
  explicitly calls out using the verb "Desar" rather than "Afegir" to be
  faithful to `proposal.md` §4's own language ("desar una idea"). The
  shipped string is a copy-paste of `AddItemInput`'s empty-name message
  (`strings.js`'s `errorEmptyName`) with "nom" swapped for "enllaç," but
  the verb was never updated — it ships exactly the word the spec called
  out to avoid.
- **Why it matters:** a direct, avoidable deviation from a string the
  human gate approved verbatim, in a component whose whole stated purpose
  was consistent "Desar" language. `role="alert"` on this element means
  this exact (wrong) wording is also what gets announced to screen-reader
  users on error — compounding the deviation with an accessibility-facing
  effect.
- **Suggested fix:** replace with the literal spec string:
  `showError('Enganxa un enllaç abans de desar.')`. Also verify the other
  two §8.3 messages (invalid scheme EC-01, rejected destination EC-02) —
  `ideas-view.js`'s EC-01 message was not located during this pass;
  `qa-engineer` should confirm it matches
  `"Aquest enllaç no és vàlid — ha de començar per http:// o https://."`
  exactly, since the same copy-paste pattern could recur there.

## 4. Spec conformance checklist

- [x] All ACs from `requirements.md` are covered by passing tests —
      `qa-engineer`'s §2 matrix confirms 10/11 fully, 1 (AC-10) passing on
      every tested dimension with a spec-level gap noted separately as
      F-17. Zero ❌.
- [x] All NFRs have a measured result — `qa-engineer`'s §2 matrix confirms
      8/9 fully, 1 (NFR-01) partial (F-20). NFR-06 (the declared blocking
      gate) is independently confirmed satisfied by `security-engineer`,
      `qa-engineer`, and this agent.
- [x] `tasks.md` checklist is fully `[x]` except C-02 (backlog transition,
      correctly left for `/commit`) and the working-tree-only edit marking
      C-03 done (semver bump, v0.1.0 cut, user-approved per the tasks.md
      diff) — consistent with the manifest's closing-task contract.
- [x] Out-of-scope items in `design.md`/`tasks.md` §5 are still out of
      scope — verified: no lifecycle/state field on ideas, no manual
      edit of recovered metadata, no re-scrape on content change, no
      Instagram API integration, no search/filter, no WebSocket/SSE, no
      persistent job queue, no dedicated outbound proxy, `internal/items`
      and `internal/projects` completely untouched (confirmed by empty
      `git diff` against both paths).
- [ ] No new public API or schema change is undocumented in `design.md`
      §6 — the 3 `/api/v1/ideas` routes and the `activity_ideas` table
      match `design.md` §6.1/§6.2 exactly (verified field-by-field against
      `dto.go`, `ideas_handlers.go`, `router.go`,
      `004_activity_ideas.sql`); **checked as pass** on the API-conformance
      dimension, but see F-17/F-19 for two places where the *implemented
      UI contract* (not the API) deviates from the approved Stage 1.5
      visual/copy spec.

## 5. Code-quality checklist (Google Engineering Practices subset)

- [x] **Design** — right shape for the codebase: `internal/ideas` mirrors
      `internal/items`/`internal/projects`'s `Repository`/`Service`/domain-type
      trio exactly (ADR-01); `internal/fetchsafe` is genuinely the single
      point of entry for outbound requests toward user URLs (confirmed by
      grep: no other package imports `net/http` to fetch a user-controlled
      URL, satisfying design.md's R-01 guardrail). `router.go`/`main.go`
      changes are the surgical additions `tasks.md` mandated.
      `items_handlers.go`/`projects_handlers.go`/`auth_handlers.go`/
      `csrf.go` and all of `internal/items`/`internal/projects`/
      `internal/auth`/`internal/config` are untouched — confirmed by an
      empty `git diff main...niu-spa-conversion` against every one of
      those paths.
- [~] **Functionality** — mostly correct for users; F-17/F-19 above are
      genuine functional/accessibility-visible gaps against the approved
      spec, not implementation bugs in the sense of crashing or wrong
      data. Everything else (four preview states, cache-at-save-time,
      async scraping, idempotent delete, no-dedup) behaves exactly as
      `design.md` describes and is test-verified.
- [x] **Complexity** — no speculative generality. The worker pool is a
      plain buffered-channel semaphore, not an over-engineered task queue
      (ADR-03's alternatives-considered explicitly rejected a persistent
      job table for this reason). `fetchsafe`'s SSRF mitigation is
      necessarily intricate (`ControlContext`, redirect re-validation,
      `Unmap()`) but each mechanism is a separate, named, commented
      function — not a single dense blob.
- [~] **Tests** — extensive and mostly excellent (SSRF regression tests
      are pinned to the exact finding they guard, not generic redirect
      tests — verified by reading the assertions, not just the test
      names). Full AC/EC/NFR matrix in §2 confirms zero unverified items.
      `-race` confirmed clean on `tests/integration` (F-21). Remaining
      gaps: F-03 (no CSRF test for `/ideas`), F-11 (F-01 regression test
      doesn't actually detect what its name claims, though the mutation
      is caught elsewhere), F-20 (no client-side E2E non-execution
      assertion for ideas' XSS mitigation).
- [x] **Naming** — clear and conventional, matches `internal/items`/
      `internal/projects` 1:1 (`Service`, `Repository`, `ErrValidation`).
      `fetchsafe`'s error sentinels are descriptive
      (`ErrDestinationForbidden`, `ErrSchemeRejected`, etc.). F-02/F-05 are
      minor/nit naming-vs-reality drifts, not naming-quality problems.
- [x] **Comments** — used precisely where the *why* is non-obvious —
      `ipvalidate.go`/`client.go`/`denylist.go` each explain the exact
      security rationale (F-01..F-07 references) for a mechanism that
      would otherwise look like an arbitrary choice (e.g.
      `DisableKeepAlives: true` reads as a performance regression without
      its comment). No restated *what*.
- [x] **Style** — `gofmt -l .` clean, `go vet ./...` clean; no lint
      findings to cite.
- [x] **Consistency** — matches `internal/items`/`internal/projects`/
      `internal/httpapi` conventions throughout (error envelope, DTO
      shape, handler structure, JS module boundaries in `web/js/`).
- [~] **Documentation** — `docs/overview.md` updated per AC-11/T-29;
      `design.md`'s 5 ADRs document every non-obvious decision including
      two rounds of security-driven amendment (F-01..F-07). F-05's
      inaccurate doc comment on `ErrResponseTooLarge` is the one drift
      found.

## 6. Security checklist (OWASP Top 10 + Code Review Top 10)

> Section owned by `security-engineer` (opt-in, active for this project).
> Full detail in their scratch file, merged as F-09..F-16 above. Summary
> checklist below, `code-reviewer`-composed from their content plus this
> agent's own spot-checks.

- [x] **A01 Broken Access Control** — `/api/v1/ideas` fully inside
      `WithCurrentUser`; `POST`/`DELETE` behind `RequireCSRF`, `GET`
      correctly exempt. No IDOR beyond the project's already-accepted,
      by-design shared-household data model.
- [x] **A02 Cryptographic Failures** — no new cryptography; `HSTS`
      unchanged; no secrets in the diff (confirmed by both reviewers'
      independent greps).
- [~] **A03 Injection** — SQL: 100% parameterized (confirmed
      `store/ideas.go`, no `fmt.Sprintf`). XSS: `ideas-render.js` is
      `textContent`/`createElement`-only, zero `innerHTML`. **Except**
      `img.src` — see F-09/F-06.
- [x] **A04 Insecure Design** — SSRF threat model is explicit, isolated in
      one auditable component, with its own ADR and amendment history.
      Gap is completeness (validates fetch destination, not recovered
      content — F-09/F-10), not absence of a threat model.
- [x] **A05 Security Misconfiguration** — CSP evaluated in depth, F-16.
      Security headers otherwise unchanged. Rate-limit increase justified
      in-code (F-04 flags it as uncommitted, not as wrong).
- [x] **A06 Vulnerable & Outdated Components** — zero new dependencies for
      NIU-6's own code path (`x/net/html` already reachable via existing
      `x/net`, per ADR-04); no known CVEs for the pinned versions;
      `govulncheck` not installed project-wide (pre-existing, non-blocking
      note, not introduced by this diff).
- [x] **A07 Identification & Authentication Failures** — no new auth
      surface; `fetchsafe`'s client carries no cookie jar or auth headers
      (NFR-08 verified via direct request inspection).
- [x] **A08 Software & Data Integrity Failures** — CSRF applied to every
      mutation; no dynamic code loading; typed JSON decode; 64KiB body
      limit via existing `LimitBody`.
- [x] **A09 Security Logging & Monitoring** — `idea_added`/
      `idea_preview_resolved` events recorded; `fetchsafe` failures logged
      at `slog.Debug` without leaking the specific rejection reason to the
      client (correct per NFR-06). Non-blocking observability suggestion
      (not a finding): a `slog.Warn` on `ErrDestinationForbidden`
      specifically would surface potential-abuse signal that `Debug`-level
      logging won't in production.
- [~] **A10 SSRF** — the core of this item. Solid and independently
      verified: all 7 prior design-review findings (F-01..F-07)
      non-regressed, NFR-06's blocking gate satisfied, 18 additional
      bypass vectors tested and rejected by `security-engineer` (IPv4
      decimal/hex/octal/short forms, IPv6 zone IDs, NAT64/6to4,
      happy-eyeballs multi-address, userinfo/fragment confusion, punycode
      homographs, content-type confusion, HTTP/2 vs 1.1, worker-pool
      context lifetime, DNS rebinding). Open items: F-12 (denylist
      trailing-dot bypass, second layer still active) and F-09 (second-order
      SSRF via unvalidated `og:image` reaching the browser).

## 7. Action items

> Blocking (must resolve before `APPROVED`):

1. Add `role="group"` (or `<article>`) + `aria-label` to
   `newCardContainer` in `ideas-render.js` — owner: `fullstack-developer`
   — fixes: F-17
2. Fix the empty-link validation message in `ideas-view.js` to
   `"Enganxa un enllaç abans de desar."` and verify the EC-01 message
   matches spec too — owner: `fullstack-developer` — fixes: F-19

> Non-blocking, recommended before merge (small, same-file edits, can ride
> in the same follow-up commit as items 1-2):

4. Validate `og:image`'s scheme (`https`, or `http`/`https`) at the
   source in `fetchsafe`'s OG parser — owner: `fullstack-developer` —
   fixes: F-09/F-06
5. Cap recovered OG field lengths (title/description/image_url) — owner:
   `fullstack-developer` — fixes: F-10/F-07 (pair with item 4)
6. Add `TestCSRF_Ideas_PostWithoutToken_Rejected`/
   `TestCSRF_Ideas_DeleteWithoutToken_Rejected` — owner:
   `fullstack-developer` — fixes: F-03
7. Commit `app/compose.yaml`'s rate-limit change deliberately — owner:
   `fullstack-developer` — fixes: F-04
8. Normalize trailing-dot FQDN before denylist comparison in
   `denylist.go`, add a test case — owner: `fullstack-developer` — fixes:
   F-12
9. Fix/rename `TestFetchPreview_RedirectToSameHost_EachHopReDialed_F01Regression`
   to actually exercise per-hop re-dialing (or rename + document that
   `TestNewTransport_DisableKeepAlivesTrue` is F-01's real regression
   guard) — owner: `fullstack-developer` — fixes: F-11
10. Fix delete-button `aria-label` to prefer title over domain — owner:
    `fullstack-developer` — fixes: F-18
11. Either delete the unreachable `ErrNotFound`/correct its doc comment,
    or clarify `ErrResponseTooLarge`'s doc comment to match actual
    silent-truncation behaviour — owner: `fullstack-developer` — fixes:
    F-02, F-05
12. Track `denylist.go`'s `traefik-public` service list reconciliation as
    a `platform-engineer` backlog note — owner: `platform-engineer` —
    fixes: F-14 (non-blocking, already anticipated by ADR-02)
13. Add a Playwright case asserting client-side non-execution of a script
    payload reaching the ideas card (e.g. via the URL field) — owner:
    `fullstack-developer`/`qa-engineer` — fixes: F-20

## 8. Sign-off

- **Approver:** none yet — verdict is `CHANGES_REQUESTED`.
- **Date:** 2026-08-03
- **Next step:** return to `/code` for action items 1-2 (blocking: F-17,
  F-19). Items 4-13 are non-blocking and may ship in the same pass or a
  fast-follow — none create exploitable state in production today
  (F-21's `-race` run has since been confirmed clean, see §3). Re-audit
  once items 1-2 land; this agent will re-verify F-17/F-19 against the
  actual diff (not the commit message) before changing the verdict, per
  this project's established re-audit
  discipline (see `NIU-5-*/review.md` §1.1 for the precedent this
  follows).
