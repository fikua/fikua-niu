---
name: niu-known-preexisting-issues
description: Known pre-existing (non-blocking) security items in Niu — open redirect via `next`, static dir listings. Check origin before filing as new findings.
metadata:
  type: project
---

Items found in Niu that are **pre-existing**, not introduced by whatever diff
is currently under review. Verify with `git diff main...HEAD -- <file>` before
filing as a finding against a branch.

1. **Open redirect via `next` (CWE-601)** — `app/web/js/auth.js` does
   `window.location.href = params.get('next') || '/'` with no validation that
   `next` is a relative path. `//evil.example` would redirect off-site after
   login. Introduced by NIU-4. Low impact (needs a crafted login URL). Not yet
   captured as a backlog item as of 2026-08-03.

2. **Static directory listings** — `GET /js/`, `/assets/`, `/fonts/` return
   `http.FileServer` autoindex pages. Only exposes names of already-public
   static assets. Pre-dates the SPA conversion.

**Why:** both surfaced during the 2026-08-03 ad-hoc review of the
`niu-spa-conversion` branch and were explicitly scoped out as not caused by
that diff.

**How to apply:** mention as informational context, not as blocking/major
findings against an unrelated branch. If the user asks to fix them, they are
small and independent.

Related: [[niu-auth-model]]
