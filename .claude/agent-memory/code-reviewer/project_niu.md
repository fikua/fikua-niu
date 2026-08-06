---
name: project-niu
description: Niu project — ad-hoc review conventions, SPA routing gotcha, and where the rigor bar is set
metadata:
  type: project
---

Niu (`/Users/ocanades/Projects/fikua/niu`) sometimes ships changes outside
the full `/capture → /define → /code` pipeline at the user's explicit
request, when they want speed over ceremony for a small fix (e.g. the
`niu-spa-conversion` branch, reviewed 2026-08-03, converting two static
HTML pages into a single-page app with a hand-rolled router). When this
happens there is no `requirements.md`/`design.md`/`tasks.md` for that
branch — review it as a straight code-quality + correctness +
security-adjacent audit against `PLAN.md` (binding architecture/security)
and the rigor bar set by the most recent full-pipeline audit
(`docs/changes/NIU-5-*/review.md` is the reference example: it does line-
level empirical verification, not trust-the-commit-message summarizing).
Output goes to a flat file under `docs/changes/`, not a formal change
folder, since there's no `tasks.md` to key off.

**Why:** the user wants fast iteration without giving up review rigor —
ad-hoc scope doesn't mean lower verification standard, just no formal
artefacts to check the diff against.

**How to apply:** when asked to review a niu branch with no
`docs/changes/<KEY>-slug/` folder, don't try to force the formal
`/audit` template (front-matter `key`/`tasks_path`/AC-matrix) — write
plain findings with severity/location/why/fix, verified empirically
(read the actual code, run real probes against it) rather than trusting
implementation-task self-reports.

**SPA-fallback gotcha — RESOLVED 2026-08-03 (NIU-6 audit).**
`app/internal/httpapi/router.go`'s `spaFallback` now checks the request
path against an explicit `spaRoutes` allowlist (`/`, `/projects`,
`/ideas`, mirrored from `ROUTES` in `main.js`) before serving
`index.html` — a typo'd/missing asset path correctly 404s again,
confirmed by the dedicated `TestSPAFallback` in `router_test.go` (cases:
unknown route, typo'd asset path, API-shaped nonexistent path — all
404). If this router changes again, confirm `spaRoutes` still mirrors
`main.js`'s `ROUTES` map 1:1 (nothing enforces that sync automatically).

**Recurring finding pattern across NIU-5 and NIU-6: doc/code drift on
unreachable sentinels.** Both items independently produced the same
class of nit/minor finding — a named error type or doc comment claims a
code path exists ("returned by Service.Get", "returned when size
exceeds...") that turns out to be unreachable from any real caller
(NIU-5 F-24: `ValidationInvalidDate` naming asymmetry; NIU-6 F-02
`ErrNotFound`/`Repository.Get` unreachable from any HTTP handler, F-05
`ErrResponseTooLarge` never actually returned — `io.LimitReader`
truncates silently instead). None of these are functional bugs — the
actual behavior is correct — but worth grep-checking every new typed
sentinel/error against its actual call sites before trusting a "returns
X when Y" doc comment at face value.

**NIU-6 (2026-08-03): `internal/fetchsafe` SSRF mitigation — genuinely
solid, verified via three independent passes (design review →
code-reviewer → security-engineer), each catching things the prior pass
missed.** Key structural pattern worth reusing as a review template if
Niu ever adds another server-side-fetch surface: (1) single-package
isolation with one public function (`FetchPreview`), (2) IP validation
via `net.Dialer.ControlContext` (not `Control` — Go silently ignores
`Control` when `ControlContext` is set) with `Unmap()` before any
`Is*()` call (IPv4-mapped-IPv6 bypass), (3) `DisableKeepAlives: true` is
load-bearing, not a perf knob — without it, connection reuse across
redirect hops to the same host skips `ControlContext` entirely (a real,
reproduced bypass, not theoretical), (4) allowlist-based IP
classification (`IsGlobalUnicast() && !IsPrivate()`) beats enumerating
denylist ranges (misses non-obvious blocks like `240.0.0.0/4`,
`198.18.0.0/15`), (5) a same-origin-edge denylist (hostname-based, e.g.
`niu.fikua.com` resolving to a public Cloudflare IP) is a *separate*
mechanism from IP validation — no IP-range check can catch a public edge
proxying back to the same app. Even with all this, `security-engineer`'s
audit still found a real gap two prior passes missed: `og:image`
recovered from the fetched page's *content* is never scheme-validated
before reaching `img.src` in the browser — `fetchsafe` validates the
*fetch destination*, not *values embedded in the response body*, a
different trust boundary entirely. Worth checking this exact blind spot
("did we validate the destination but forget to validate the payload
that gets rendered/reused downstream") on any future scraping/fetch
feature.

**Mutation-testing a security regression test is worth doing, not just
reading it.** NIU-6's `security-engineer` found that
`TestFetchPreview_RedirectToSameHost_EachHopReDialed_F01Regression`
doesn't actually detect the F-01 regression its name claims — the test's
destination is loopback, so `ControlContext` rejects the *first* dial,
and the redirect chain never progresses far enough to exercise
connection-reuse at all. Reverting the fix under test (`DisableKeepAlives:
false`) still passes it; the regression is only caught by a *different*,
more literal test (`TestNewTransport_DisableKeepAlivesTrue`). Lesson:
for any test whose name claims "regression test for finding X," actually
flip the fix back and confirm the test goes red — don't trust that a
passing green test with the right name/comment is proof it exercises the
mechanism it claims to.

**Project scale reminder:** [[niu-project]] (personal memory) has the
architecture context — two-person household app, single Go binary,
SQLite, no build step/npm/framework for the frontend (binding per
`PLAN.md` §2.3). Any review of Niu frontend work should confirm that
constraint holds (no bundler config, no `package.json` build script
introduced).
