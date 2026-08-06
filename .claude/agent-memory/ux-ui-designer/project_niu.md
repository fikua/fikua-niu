---
name: project-niu
description: Niu project (household app for two) — visual spec conventions, design-system tokens, and audit findings history for the ux-ui-designer role
metadata:
  type: project
---

Niu is a two-user household app (Go + SQLite single binary, web SPA at
`app/web/`). Visual specs are written inline in `proposal.md` §8 (Markdown/
ASCII spec, no Figma) — the project uses the "otherwise" branch of the
Stage 1.5 procedure. Tokens live in `app/web/app.css` `:root`, sourced
from `design-system/tokens.css`-style naming (`--color-*`, `--radius-*`,
`--shadow-*`, `--space-*`, `--type-*`).

**Established accent tokens per top-level space** (each `/define` item
that adds a new top-level nav entry needs its OWN accent, confirmed
pattern as of NIU-6):
- `color.moss` — Compra/Rebost (NIU-1)
- `color.terracotta` / `-hover` / `-active` — Projectes (NIU-5, ADR-04)
- `color.mel` (#C99A3A family) — Idees (NIU-6, approved at Stage 1.5 gate
  2026-08-03) — proposed explicitly as a NEW token because reusing
  terracotta across two top-level spaces would fail AC-07-style visual
  differentiation checks. **How to apply:** if a future item adds a
  fourth top-level space, do the same §8.0-style flag-and-propose
  pattern rather than reusing an existing accent — this project has
  already hit "all existing accents are spoken for" once.

**Contrast rule for warm/dark accents on `color.bg` (#FBF6EC):** both
`terracotta` and `mel` fail 4.5:1 as plain text/pale-surface colour —
only the `-hover`/`-active` shades clear AA for text-sized use, and even
those need per-token verification against actual button-fill contrast,
not just against the page background (NIU-6's `mel-hover`/`mel-active`
hex values were retuned slightly darker than my Stage 1.5 proposal after
an axe-core contrast finding — see [[feedback-token-values-may-drift-post-approval]]).

**Design-spec accessibility items axe-core will NOT catch:** `role="group"`/
`aria-label` on card-like list items is a spec-level requirement this
project writes explicitly into §8.6 precisely because generic
WCAG-automated tooling (axe-core) does not have a rule for "list item
needs a named group role." A passing axe-core run in this project's E2E
suite is not sufficient evidence that an AC-10-style "screen reader
narrative per item" requirement is met — always check the actual DOM
node for `role`/`aria-label`, not just the axe report. First observed:
NIU-6 `/audit` (2026-08-03), `.idea-card` shipped without it despite
being explicit in the approved spec.

E2E specs live at `app/tests/e2e/specs/*.spec.js` (Playwright), run via
`npx playwright test specs/<name>.spec.js` from `app/tests/e2e/`. An
axe-core WCAG 2.2 AA sweep lives in `accessibility-audit.spec.js`,
extended per-space as each new item ships (grep for the item's own
`test(...)` block rather than assuming full coverage exists).

**NIU-6 F-17/F-18/F-19 re-audit (2026-08-03, commit 647f66f): all three
genuinely RESOLVED** — verified by direct code read + byte-level string
comparison against `proposal.md` §8.3/§8.6, not just re-running the E2E
suite (which stayed green before AND would have stayed green if the fix
had been incomplete — see below). Confirms the fix-report's own claims
were accurate, including a same-shape copy-paste bug found server-side
in `internal/ideas/service.go`'s `ValidationEmpty` message: the client
already short-circuits the empty-URL case before any network call, so
the server string is dead-in-practice for that exact path, but fixing it
was still the right call as defense-in-depth (a future client-guard
regression would otherwise resurface the old wrong copy via the
`ideas-api.js` → `ApiError.message` → `showError()` passthrough, which
propagates `body.error.message` verbatim on any non-OK response). **How
to apply:** when a fix report claims "also fixed the same bug
server-side," always trace whether the client-side guard makes that
path reachable from the UI today — if not, the fix is still correct
(defense-in-depth) but say so explicitly rather than implying the user
could currently see the wrong string both ways.

**Confirmed again: E2E green is not evidence for `role`/`aria-label`
gaps specifically — recommend a dedicated attribute assertion.** Neither
`ideas.spec.js` nor `accessibility-audit.spec.js` asserts
`.idea-card`'s `role`/`aria-label` via `toHaveAttribute(...)` — the
axe-core spec only asserts `violations === []`, which passes for a plain
`<div>` with no ARIA just as easily as for correctly-annotated markup.
Flagged as a minor, non-blocking recommendation each time this class of
finding comes up (`[[project-niu]]` axe-core gap note above) — worth
proposing the concrete assertion line next time this file is touched
rather than re-discovering the gap from scratch.
