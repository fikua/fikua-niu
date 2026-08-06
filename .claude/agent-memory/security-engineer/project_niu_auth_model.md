---
name: niu-auth-model
description: Niu gates auth client-side — the static shell is public by design; only /api/v1/* is server-protected. Critical context for any "does X leak when unauthenticated" review.
metadata:
  type: project
---

Niu's authentication is **client-side gating**, not server-side page protection.
`index.html` (the SPA shell) is served to **anyone**, authenticated or not.
`js/main.js` calls `GET /api/v1/me`; on 401, `api.js` redirects to
`/login.html?next=...`. All real data lives behind `/api/v1/*`, which is the
only surface wrapped by `WithCurrentUser`.

Router structure (`app/internal/httpapi/router.go`):
- `SecurityHeaders` is the **outermost** `r.Use()` — CSP/XFO/nosniff apply to
  every response including the `NotFound` handler.
- `WithCurrentUser` is mounted **only** inside `r.Route("/api/v1", ...)`.
- chi's `/api/v1` subrouter captures the whole namespace with a subtree
  wildcard, so `/api/v1/<typo>` returns 401/404 JSON and **never** reaches the
  root `NotFound` handler. This is why the SPA fallback cannot serve HTML for
  API-shaped paths.
- Static assets are served from `embed.FS` rooted via `fs.Sub(WebFS, "web")`.

**Why:** two-person household app; the shell contains no secrets, no CSRF
token, no user data — only empty containers and nav labels. Gating it
server-side would buy nothing.

**How to apply:** when reviewing "does an unauthenticated request leak X",
first check whether X was *already* public on `main` (`GET /` served the full
shell long before the SPA fallback existed). The bar is whether `/api/v1/*`
protection changed — not whether static HTML is reachable. Do not file
findings that assume the shell is meant to be auth-gated.

Related: [[niu-known-preexisting-issues]]
