# Niu — Master Execution Plan

> **Audience:** the Fikua SDD agents (`product-manager`, `qa-engineer`,
> `software-architect`, `task-planner`, `fullstack-developer`,
> `code-reviewer`, `security-engineer`, `platform-engineer`,
> `ux-ui-designer`) and the human driving them.
>
> **Status:** approved by the human owner on 2026-08-01. This document is
> the frozen input to `/define`. It is NOT an artefact of the SDD loop —
> it is the brief that feeds it.
>
> **Language note:** this file is model-facing, so it is written in
> English per the framework convention. Human-facing product docs under
> `docs/` follow `language.team: ca`.

---

## 0. How to use this document

This plan is deliberately **prescriptive about decisions already made**
and **silent about decisions not yet made**. Agents must:

1. Treat §2 (Architecture), §3 (Security), §5 (Deployment) and §7 (Test
   plan) as **binding constraints**. They were decided with the human and
   are not open for re-litigation during `/define`.
2. Treat §8 (Backlog) as the **execution order**. One item at a time.
3. Treat anything marked `[OPEN]` as a genuine decision for `/define` to
   resolve with the human.
4. **Stop and ask** rather than guess when this document is silent. The
   framework rule stands: ambiguity is a bug.

### Decision log (why things are the way they are)

Recorded so no agent re-opens a settled question:

| Decision | Chosen | Rejected alternative | Reason |
|---|---|---|---|
| App name | `Niu` | Lar&Niu, NossaNostra, Junt(o)s | Short, reads identically in CA and PT, needs no gloss |
| Database | SQLite | PostgreSQL | Two users, one household. Postgres is infra cost with no benefit. Migration path stays open |
| Frontend hosting | Served by the Go binary via `embed.FS` | Cloudflare Pages/Workers | Same-origin kills CSRF structurally; `SameSite=None` would have been a permanent security cost for zero gain. See §2.1 |
| Repo layout | Single standalone repo, code in `app/` | Separate front/back repos | One deployable unit, one pipeline, no version skew |
| Auth mechanism | Opaque session token, hash stored in DB | JWT | JWTs cannot be revoked without a blacklist — which is exactly the `sessions` table. Opaque token is both simpler and a better path to mobile (§9) |
| Execution order | ~~List → Deploy → OTEL → Auth~~ **List → Auth → Deploy → OTEL** | Auth first (original) | Original order chosen for early visible value, stopgapped with Cloudflare Access. **Superseded 2026-08-02** — see the two rows below and §8 |
| API prefix | `/api/v1/` from day one | `/api/` | Costs nothing today, avoids a painful dual-URL period if a mobile app ever ships (§9) |
| Duplicate items | **Blocked** (trimmed, case-insensitive, across both boxes) | Allow / warn-but-allow | Human chose a clean list over quantity-in-list. Resolved 2026-08-01 |
| Sync between users | **Polling ~10s + refetch on focus** | SSE | Two people; polling is indistinguishable from real time and has far fewer moving parts. Resolved 2026-08-01 |
| Quantity | **No field** — written inside the name (`"2 llets"`) | Numeric column + ± UI | Avoids a column, a migration, UI and test cases for something a string already solves. Resolved 2026-08-01 |
| User identity in docs | **Generic** (`Usuari A` / `Usuari B`) | Real names | **The repo is public.** Real names and avatars are injected via env at deploy time. Resolved 2026-08-01 |
| Auth mechanism (NIU-4) | **Google OAuth only, email allowlist** | Username/password (bcrypt) · Both, user's choice | Delegating auth to Google removes S4 (brute force), S5 (user enumeration) and S6 (session fixation) as *code the app must get right* — Google already solves them. For two users, "both options" would double the auth surface for no benefit. Resolved 2026-08-02 |
| Public exposure strategy | **Ship NIU-4 before any public deploy** | Deploy behind Cloudflare Access as a stopgap, add real auth later | Access-as-stopgap required inventing a deployment pattern with no precedent on the platform (proxied A record + Access; every existing Access app fronts a Tunnel). Real auth removes the exposure problem at its root instead of working around it. Resolved 2026-08-02 — reorders the backlog: NIU-4 now ships before NIU-2's public deploy |

---

## 1. Product

**Niu** is a private household app for exactly two people (a couple). It
is not a product for sale, has no growth ambitions, and will never have
user registration.

**v1 scope:** a shared shopping list with two boxes — *to buy* and
*pantry* — where selecting an item moves it between them.

**Explicitly out of scope for v1:** registration, password reset, user
management, invitations, roles, multi-household, notifications, mobile
apps.

**Future (not now):** chores, meal planning, expenses, gamification
layers. The data model in §2.4 anticipates these without implementing
them.

### 1.1 Naming and wordmark

- Wordmark: lowercase **`niu`** everywhere.
- Domain: `niu.fikua.com`.
- Slug (repo, manifest `name`, `OTEL_SERVICE_NAME`, container name): `niu`.

---

## 2. Architecture

### 2.1 The single-binary decision (binding)

The Go binary serves **both** the JSON API and the static frontend:

```
┌─────────────────────────────────────────┐
│  niu (single Go binary, single container)│
│                                          │
│  ┌──────────────────┐  ┌──────────────┐  │
│  │  JSON API        │  │  embed.FS    │  │
│  │  /api/v1/*       │  │  web/*       │  │
│  └──────────────────┘  └──────────────┘  │
│           │                    │         │
│           └────────┬───────────┘         │
│                    ▼                     │
│              SQLite @ /data/niu.db       │
└─────────────────────────────────────────┘
```

**This is a deployment decision, not a coupling decision.** The API and
the static file server are independent concerns that happen to share a
process. Agents MUST keep them separable:

- The API layer MUST NOT render HTML.
- The frontend MUST talk to the API only over `fetch` against
  `/api/v1/*`, exactly as any other client would.
- No server-side templating. No mixing.

Consequence: the same origin serves everything → `SameSite=Strict`
cookies work → CSRF is structurally impossible, not merely mitigated.
This is why §3 is short. Do not undo it.

### 2.2 Repository layout

```text
niu/                              ← this repo (project meta-folder + code)
├── fikua.yml                     ← SDD manifest
├── PLAN.md                       ← this file
├── CLAUDE.md                     ← generated by /init inside app/
├── BACKLOG.md                    ← markdown backlog adapter
├── CHANGELOG.md
├── README.md
├── docs/
│   ├── overview.md
│   ├── architecture.md
│   ├── test-plan.md              ← §7 — the human's control mechanism
│   ├── conventions/
│   └── changes/<KEY>-slug/       ← SDD artefacts per item
└── app/                          ← the Go module
    ├── go.mod
    ├── cmd/niu/main.go
    ├── internal/
    │   ├── config/               ← env parsing, fail-fast validation
    │   ├── store/                ← SQLite access, migrations
    │   ├── items/                ← domain: shopping list + pantry
    │   ├── auth/                 ← sessions, password verify (NIU-4)
    │   ├── httpapi/              ← handlers, middleware, routing
    │   └── observability/        ← OTEL wiring (NIU-3)
    ├── migrations/               ← goose SQL, embedded
    ├── web/                      ← HTML + CSS + JS, embedded
    │   ├── index.html
    │   ├── app.css
    │   └── js/
    ├── tests/                    ← integration + security tests
    ├── Dockerfile
    ├── compose.yaml              ← synced copy; source of truth in IaC repo
    ├── .env.example
    └── Makefile
```

### 2.3 Stack (binding)

| Concern | Choice | Constraint |
|---|---|---|
| Language | Go 1.25 | Match `fikua-yes-very-well-fandango` (verified: its `build.yml` uses Go 1.25) |
| Router | `net/http` + `chi/v5` | stdlib does most of it; chi for routing + middleware only |
| SQLite driver | `modernc.org/sqlite` | **Pure Go, no CGO.** Required for a static binary and a `distroless`/`scratch` image. Do NOT use `mattn/go-sqlite3` |
| Migrations | `pressly/goose/v3`, embedded | Versioned from day one |
| Frontend | Vanilla HTML + CSS + ES modules | **No build step. No npm. No framework.** |
| Animation | Web Animations API (FLIP) | ~250ms move between boxes |
| Testing | stdlib `testing` + `httptest` | Plus Playwright for E2E only if §7 demands it |

**SQLite configuration (binding):** open with `_pragma=journal_mode(WAL)`,
`_pragma=busy_timeout(5000)`, `_pragma=foreign_keys(on)`. WAL matters
because two users may write concurrently.

### 2.4 Data model (v1)

```sql
CREATE TABLE users (
  id            INTEGER PRIMARY KEY,
  name          TEXT NOT NULL UNIQUE,      -- login handle
  display_name  TEXT NOT NULL,
  password_hash TEXT NOT NULL,             -- bcrypt; seeded, never registered
  avatar_emoji  TEXT NOT NULL DEFAULT '🐦',
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
  token_hash  TEXT PRIMARY KEY,            -- SHA-256 of the token, NEVER the token
  user_id     INTEGER NOT NULL REFERENCES users(id),
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at  TIMESTAMP NOT NULL
);

CREATE TABLE items (
  id         INTEGER PRIMARY KEY,
  name       TEXT NOT NULL,
  location   TEXT NOT NULL CHECK (location IN ('shopping','pantry')),
  position   REAL NOT NULL,                -- fractional ordering, see note
  added_by   INTEGER REFERENCES users(id),
  moved_by   INTEGER REFERENCES users(id),
  moved_at   TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE events (                      -- append-only, substrate for gamification
  id         INTEGER PRIMARY KEY,
  user_id    INTEGER REFERENCES users(id),
  kind       TEXT NOT NULL,                -- 'item_added','item_moved','item_deleted'
  payload    TEXT NOT NULL,                -- JSON
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Design notes agents must respect:**

- An item is **one row that changes `location`**, never two tables and
  never a delete+insert. Moving is an `UPDATE`. This preserves authorship
  history and keeps the door open for gamification without a refactor.
- `position` is `REAL` so reordering is an update to one row (insert
  between `1.0` and `2.0` → `1.5`), not a renumbering of the whole list.
- `events` is written from day one even though nothing reads it in v1. It
  costs ~20 lines now and saves a painful backfill later.
- `users.password_hash` exists from the first migration even though
  NIU-4 is last; the seed just inserts placeholder rows until then.
- **No quantity column.** Quantity lives inside `name` (`"2 llets"`).
  Resolved with the human 2026-08-01 — a string already solves this, and
  a column would cost a migration, UI and test cases for no gain.
- `items` carries a **case-insensitive uniqueness constraint on the
  trimmed name across both locations** (duplicates are blocked, §3).

### 2.5 API contract (v1)

All endpoints under `/api/v1/`. All responses JSON. All mutations are
`POST`/`PATCH`/`DELETE` — **never `GET`** (a `GET` mutation would
reintroduce CSRF).

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/items` | List all items, both locations |
| `POST` | `/api/v1/items` | Create item in `shopping` |
| `PATCH` | `/api/v1/items/{id}` | Move (`location`) and/or rename |
| `DELETE` | `/api/v1/items/{id}` | Remove item |
| `GET` | `/api/v1/me` | Current user (stub until NIU-4) |
| `POST` | `/api/v1/auth/login` | NIU-4 |
| `POST` | `/api/v1/auth/logout` | NIU-4 |
| `GET` | `/healthz` | Liveness — **not** under `/api/v1`, no auth |

**Error format (binding, uniform):**

```json
{ "error": { "code": "validation_failed", "message": "Item name is required" } }
```

Never leak internal errors, stack traces, SQL, or file paths to the
client. Log detail server-side; return a generic message.

### 2.6 Concurrency between the two users

**RESOLVED 2026-08-01: polling.** `GET /api/v1/items` every ~10s plus a
refetch on window focus. SSE was considered and rejected for v1 — for two
people, polling is indistinguishable from real time and carries far fewer
moving parts (no reconnection logic, no server-side connection state, no
proxy buffering surprises). Revisit only if it feels stale in real use.

Moves must be **idempotent**: `PATCH` sets
`location` to an absolute value, never toggles. If both users move the
same item, last write wins and both clients converge on refetch.

---

## 3. Security (binding)

The human explicitly required a secure app despite the simple login.
Every row below is a requirement, and every row has a corresponding test
in §7.2.

| # | Threat | Mitigation | Enforced in |
|---|---|---|---|
| S1 | CSRF | Same-origin + `SameSite=Strict`. **Plus** double-submit token on all mutations as defence in depth. No `GET` mutations | NIU-4 (token), NIU-1 (no GET mutations) |
| S2 | Session hijacking | Cookie `HttpOnly; Secure; Path=/; SameSite=Strict`. Token = 256 bits from `crypto/rand`. **DB stores SHA-256 of the token, never the token** | NIU-4 |
| S3 | XSS | CSP with no `unsafe-inline` (all JS in files). **Zero `innerHTML` with user data — `textContent` only** | NIU-1 |
| ~~S4~~ | ~~Brute force~~ | **Removed 2026-08-02.** Google OAuth is the only login path — there is no password to brute-force. Google's own account security (2FA, anomaly detection) applies | — |
| S5 | Identity verification | The ID token's `email` claim MUST be checked against the allowlist (`NIU_ALLOWED_EMAILS`) **and** its issuer/audience/signature verified via Google's OIDC discovery document — never trust a client-supplied email. Reject anyone not on the list with a generic "not authorised" page, no detail about why | NIU-4 |
| ~~S6~~ | ~~Session fixation~~ | **Unchanged in substance** — still applies to Niu's own session cookie issued after a successful Google login, renumbered nowhere else. New token on every login; invalidate on logout; expire server-side | NIU-4 |
| S7 | Missing headers | `HSTS`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, CSP | NIU-1 |
| S8 | SQL injection | Prepared statements only. **Zero string concatenation into SQL** | NIU-1 |
| S9 | Secrets in image/repo | OAuth client secret and `NIU_SESSION_SECRET` injected via env at runtime. Nothing secret in the image or git | NIU-2, NIU-4 |
| ~~S10~~ | ~~Unauthenticated exposure window~~ | **Removed 2026-08-02.** NIU-4 now ships *before* the first public deploy (backlog reorder, §8) — there is no window to cover with a stopgap. Cloudflare Access is no longer part of the deployment plan | — |

**Why S4/S6(brute-force)/S10 disappear rather than move.** They existed
to cover a self-hosted username/password scheme and a temporary exposure
window created by shipping auth last. Both premises are gone: Google
handles credential security, and the backlog now ships auth before any
public traffic reaches the app. Removing the row is more honest than
marking it "done elsewhere" — there is nothing left to do.

**S5 is now the one that could actually bite.** A misconfigured OAuth
client (wrong `redirect_uri`, missing token verification) is the
realistic failure mode, not a guessed password. `security-engineer` MUST
verify the ID token is cryptographically validated, not merely decoded
and trusted.

**Input validation (NIU-1):** item name trimmed, 1–200 chars after trim,
reject empty/whitespace-only, allow Unicode (accents, emoji, apostrophes
like `O'Neill`). Reject or coerce control characters. **Duplicates
rejected** — trimmed, case-insensitive, across both boxes.

**S11 — public repository.** `fikua/fikua-niu` is public. No real names,
emails, household details, hostnames beyond `niu.fikua.com`, or any
personal data may appear in any committed file. Documents refer to
`Usuari A` / `Usuari B`; real identities are injected via environment
variables at deploy time and never committed. This applies to code
comments, seed data, test fixtures and screenshots.

---

## 4. Look & feel

Approved direction: **warm, not childish.**

- **Palette:** cream/sand background, moss green and terracotta accents.
  No corporate blue, no SaaS grey.
- **Shape:** generous corner radius (16–20px), soft shadows.
- **Type:** rounded sans (Nunito or Quicksand), self-hosted in `web/` —
  no Google Fonts CDN call (CSP + privacy).
- **Layout:** two boxes side by side on desktop, stacked with tabs on
  mobile.

```
┌─────────────────────┐  ┌─────────────────────┐
│  🛒 A comprar        │  │  🥫 Rebost          │
│  ─────────────       │  │  ─────────────      │
│  ○ Llet          🐦  │  │  ● Arròs      🦊 ↩  │
│  ○ Pa            🦊  │  │  ● Oli        🐦 ↩  │
│  [ + afegir…      ]  │  │                     │
└─────────────────────┘  └─────────────────────┘
```

**Interaction (approved):**

- Tap an item → FLIP animation flying to the other box, ~250ms,
  `ease-out`.
- Optimistic UI: move immediately, reconcile with the server after. On
  failure, animate back and show a non-blocking toast.
- Haptic feedback on mobile where available.
- `prefers-reduced-motion` → cross-fade instead of flight. **Required**,
  not optional.

**Gamification in v1 — deliberately minimal:**

- ✅ Per-user avatar emoji on each item (who added / who moved it).
- ✅ Discreet confetti when the shopping list empties.
- ❌ Streaks, points, leaderboards — the `events` table collects the data;
  build these only once there is real usage. Gamifying before the habit
  exists is speculative design.

Accessibility is a v1 requirement, not a nice-to-have: keyboard
navigable, AA contrast, `aria-live` announcements when items move.

---

## 5. Deployment (binding — verified against the real infra)

Everything below was verified by reading `fikua-platform-iac` and the
sibling repos. Agents MUST NOT invent alternatives.

### 5.1 Target

Single OVH VPS, Docker Compose, Traefik v3.4 routing by labels on the
external `traefik-public` network. Host path:
`/opt/vps/projects/niu/`.

### 5.2 Compose

Copy the structure of
`fikua-platform-iac/projects/exam-room/compose.yaml` — a Go service, no
published host port, Traefik-routed. Differences for Niu:

- **No database container.** Niu is the first SQLite app on this
  platform. Use a named volume `niu-data` mounted at `/data`, with the DB
  at `/data/niu.db`.
- Add the label `fikua.backup.sqlite=/data/niu.db` — this is the hook the
  backup script will discover (§5.5).
- Direct Cloudflare-proxied **A record** → Traefik `websecure` (443),
  `tls: "true"`, cert from the existing `*.fikua.com` wildcard. **Not**
  the tunnel `web` entrypoint (that is for internal tools like dozzle and
  openobserve).
- Include a Traefik rate-limit middleware mirroring exam-room's, with
  `sourcecriterion.requestheadername: "Cf-Connecting-Ip"` — without this
  header the limiter sees rotating Cloudflare edge IPs and never fires.
  This is a documented, production-confirmed gotcha.
- Resource limits: memory 128M, cpus '0.5'. Logging json-file, 25m × 3.

Source of truth for the compose file is
`fikua-platform-iac/projects/niu/compose.yaml`; a synced copy lives at
`niu/app/compose.yaml` so `deploy.yml` can push it. Keep both in sync —
same convention as exam-room.

### 5.3 CI/CD (GitHub Actions)

Copy the **`fikua-yes-very-well-fandango`** pattern (the newer, cleaner
of the two in the org). Three workflows:

| Workflow | Trigger | Does |
|---|---|---|
| `build.yml` | PR + push to main | `go vet`, `go test ./...`, `go build`; frontend lint if any |
| `release.yml` | `release: [published]` | Multi-arch (`linux/amd64,linux/arm64`) build → **Docker Hub** `fikua/niu`, tags semver + latest + `sha-`, GHA cache |
| `deploy.yml` | `release: [published]` + `workflow_dispatch` | `cloudflared access ssh` → VPS, scp compose, `docker compose pull && up -d`, health poll, Traefik smoke test |

**Organization secrets (verified names — these are `secrets.*`, NOT
`vars.*`; no workflow in the org uses org variables):**

- `DOCKER_USERNAME`, `DOCKER_TOKEN` — Docker Hub push
- `VPS_SSH_PRIVATE_KEY` — SSH key
- `CF_ACCESS_CLIENT_ID`, `CF_ACCESS_CLIENT_SECRET` — Cloudflare Access
  service token for the SSH tunnel

SSH is **not raw** — it goes through Cloudflare Access:

```
Host vps.fikua.com
  User ubuntu
  ProxyCommand cloudflared access ssh --hostname %h
```

`deploy.yml` MUST use `environment: prd` with required reviewers and
`concurrency: group: deploy-niu-prd` — matching the existing pattern so a
deploy cannot race itself.

### 5.4 OTEL → OpenObserve

Niu will be **the platform's first real Go OpenTelemetry instrumentation**
(existing instrumented apps are JVM; the Go exam-room service only sets
env vars without an SDK). This has value beyond Niu — write it as a
reference implementation.

App env (exact):

```yaml
OTEL_EXPORTER_OTLP_ENDPOINT: http://otel-collector:4318
OTEL_EXPORTER_OTLP_PROTOCOL: http/protobuf
OTEL_SERVICE_NAME: niu
OTEL_TRACES_EXPORTER: otlp
OTEL_METRICS_EXPORTER: otlp
```

⚠️ **Two documented traps — both cause silent data loss:**

1. Port is **4318** (OTLP/HTTP), not 4317. Port 4317 accepts the
   connection then resets, and nothing ever arrives.
2. The Go SDK defaults to **gRPC**. `OTEL_EXPORTER_OTLP_PROTOCOL` must be
   set explicitly to `http/protobuf`.

The app needs **no auth headers** — the shared collector holds the Basic
auth and forwards to OpenObserve. Do not set
`OTEL_EXPORTER_OTLP_HEADERS`.

⚠️ **Required change in the IaC repo:** edit
`fikua-platform-iac/platform-services/otel-collector/config.yaml` to add
an `otlphttp/niu` exporter, three pipelines (`traces|metrics|logs/niu`),
and a match rule on `service.name == "niu"` in **each** of
`routing/traces`, `routing/metrics`, `routing/logs`. Without this the
app exports happily and the collector silently discards everything.

The container must join `traefik-public` to reach `otel-collector`.

Instrumentation scope: `otelhttp` middleware on all routes, manual spans
around DB operations, and a `user.id` attribute on every span (useful for
debugging *and* it feeds future gamification).

UI: `https://observability.fikua.com`.

### 5.5 SQLite backup

`fikua-platform-iac/scripts/backup-db.sh` is Postgres-only today. It
discovers containers with:

```bash
mapfile -t DB_CONTAINERS < <(docker ps --format '{{.Names}}' | grep -E -- '-db$' || true)
```

Niu's container is not named `*-db` and has no `pg_dump`, so **it would
be silently skipped**. Required changes:

1. Add a second discovery pass for containers carrying the label
   `fikua.backup.sqlite=<path>`.
2. Back up with `sqlite3 "$db" ".backup /tmp/niu-backup.db"` then gzip.
   **Never `cp` a live SQLite file** — a copy taken mid-write is
   corrupt.
3. Reuse the existing `rclone copy` to
   `ovh-backups:fikua-db-backups/<container>/` and the same
   `${container}_${TIMESTAMP}.sql.gz` naming.
4. ⚠️ **The retention/prune loop iterates `DB_CONTAINERS`.** Any new
   container list must be folded into that loop or given its own prune
   pass — otherwise Niu backs up forever and never prunes, quietly
   filling the bucket.
5. Scheduling is Ansible-managed cron
   (`ansible/roles/cron-db-backup/`), daily at 03:00. No change needed
   beyond the script.

**Verification is part of the story:** restore a backup into an empty
database and confirm the items come back. A backup that has never been
restored is not a backup.

---

## 6. Configuration

All config via environment, parsed once at startup with **fail-fast
validation** — the process must refuse to start on missing/invalid
required config rather than failing at first request.

| Var | Required | Notes |
|---|---|---|
| `NIU_PORT` | no (8080) | |
| `NIU_DB_PATH` | no (`/data/niu.db`) | |
| `NIU_SESSION_SECRET` | **yes** (NIU-4) | ≥32 bytes; refuse to start if short |
| `NIU_GOOGLE_CLIENT_ID` | **yes** (NIU-4) | OAuth 2.0 client ID from Google Cloud Console |
| `NIU_GOOGLE_CLIENT_SECRET` | **yes** (NIU-4) | never logged, never in any response |
| `NIU_ALLOWED_EMAILS` | **yes** (NIU-4) | comma-separated allowlist; refuse to start if empty — an app with no allowed emails is a bug, not a valid state |
| `NIU_USER_A_DISPLAY` / `NIU_USER_B_DISPLAY` | yes (NIU-4) | display name + avatar shown in the UI, keyed by the verified Google email — no password material anywhere |
| `NIU_ENV` | no (`production`) | `development` relaxes `Secure` cookie for localhost |
| `OTEL_*` | no | Absent → tracing disabled, app still runs |

Secrets live in a gitignored `.env` on the host next to the compose file,
with a committed `.env.example` showing shape only. This is the
established pattern for small projects (SOPS is for infra-level secrets).

---

## 7. Test plan — `docs/test-plan.md`

**This is the human's primary control mechanism.** The human has stated
they will not review the code. Therefore this document is not
documentation — it is *the contract*.

Rules:

- Owned by `qa-engineer`. Written in Gherkin (Given/When/Then), in
  Catalan (`language.team: ca`) so the human can actually read it.
- **Every case maps to an automated test.** A case with no test is a
  fiction.
- CI blocks on failure. A red suite means the story is not done.
- Written during `/define` **before** implementation, audited in
  `/audit`.

### 7.1 Functional cases (NIU-1 unless noted)

**Shopping list**
- Add an item → appears in *to buy*, persists across reload.
- Add with empty/whitespace-only name → rejected with a clear message.
- Add a 200-char name → accepted; 201 chars → rejected.
- Add a name with accents, emoji, and an apostrophe (`O'Neill`) → stored
  and displayed verbatim.
- Add a duplicate name → **rejected** with a clear message. Matching is
  trimmed and case-insensitive, and spans **both** boxes (an item already
  in the pantry cannot be re-added to the shopping list — it is moved).
  Edge cases to cover: `"llet"` vs `"Llet"` vs `"  LLET  "`.

**Moving between boxes**
- Select an item in *to buy* → moves to *pantry* in one operation;
  authorship (`moved_by`, `moved_at`) updated.
- Select an item in *pantry* → returns to *to buy*.
- Move persists across reload.
- Delete an item → gone from both boxes, does not reappear.

**Two users**
- A adds an item; B sees it after refresh (or within the poll interval).
- Both move the same item concurrently → no error, both converge on the
  same state after refetch.
- Actions are attributed to the right avatar.

**Persistence**
- Restart the container → all data intact (volume).
- Restart mid-write → database not corrupt, app starts cleanly.

**UI**
- Move animation runs and lands in the correct box.
- `prefers-reduced-motion` → cross-fade, no flight.
- Empty shopping list → confetti fires once, not on every render.
- Mobile viewport → boxes stack, tabs work.
- Optimistic move + server error → item animates back, toast shown.

### 7.2 Non-functional cases

**Security — each test performs the attack and asserts it fails.**

| ID | Test |
|---|---|
| S1 | `POST /api/v1/items` without CSRF token → 403 |
| S1 | No mutation is reachable via `GET` (route table assertion) |
| S2 | Request with no cookie → 401; tampered cookie → 401 |
| S2 | DB inspection: `sessions` contains no value equal to a live token |
| S3 | Item named `<img src=x onerror=alert(1)>` renders as literal text; asserted in a real browser |
| S3 | Response carries a CSP with no `unsafe-inline` |
| S5 | Unverified/forged ID token (bad signature, wrong audience) → rejected before the email is even read (NIU-4) |
| S5 | Valid Google login with an email not on `NIU_ALLOWED_EMAILS` → generic "not authorised", no session issued (NIU-4) |
| S6 | Token before login ≠ token after; logout invalidates server-side (NIU-4) |
| S7 | HSTS, nosniff, X-Frame-Options, Referrer-Policy all present |
| S8 | Item named `'; DROP TABLE items;--` is stored literally; table survives |
| S9 | `docker history` / image scan shows no secret (including the OAuth client secret); repo has no `.env` |
| S11 | Repo scan finds no real names, emails or personal data in any committed file |

> S4 and S10 have no test — both rows were removed from §3, not merely
> marked done. There is no password to brute-force, and no exposure
> window to verify closed.

**Performance**
- p95 `GET /api/v1/items` < 200ms with 500 items.
- Initial page load < 1s on simulated 3G.
- Image size < 30MB (static binary + distroless).

**Reliability**
- `docker kill` mid-`UPDATE` → no corruption (WAL).
- Backup restores into an empty DB and yields identical item set.
- `/healthz` returns 200 only when the DB is reachable.

**Accessibility**
- Full keyboard operation: add, move, delete.
- AA contrast on all text.
- `aria-live` announces moves to a screen reader.

**Observability (NIU-3)**
- One request produces a complete trace visible in OpenObserve with
  `service.name = niu`.
- Trace includes the HTTP span and the DB span.
- Collector is not dropping Niu data (verify after the config.yaml
  change — this is the trap in §5.4).

---

## 8. Backlog and execution order

**Order revised 2026-08-02.** Originally: list → deploy → OTEL → auth,
accepting a public-but-unauthenticated window closed by Cloudflare
Access. That stopgap required inventing a deployment pattern with no
precedent on the platform (a proxied A record with Access in front —
every existing Access app fronts a Tunnel instead). Shipping real auth
before the first public deploy removes the exposure problem at its root
instead of working around it, and Google OAuth (§3, decision log) makes
NIU-4 small enough that reordering costs little.

**Current order: list → auth → deploy (public) → OTEL.**

| Key | Type | Title | Depends on |
|---|---|---|---|
| `NIU-1` | story | Shopping list ↔ pantry with stubbed auth | — (done) |
| `NIU-4` | story | Google OAuth login, email allowlist | NIU-1 |
| `NIU-2` | task | Deployment: Docker, compose, CI/CD, DNS, SQLite backup | NIU-4 |
| `NIU-3` | task | OTEL instrumentation → OpenObserve | NIU-2 |

### NIU-1 — Shopping list (story) — DONE

Scope: data model + migrations, `/api/v1/items` CRUD, `events` writes,
the full warm UI with FLIP animation, optimistic updates, accessibility,
security items S3/S7/S8, and the functional test suite from §7.1.

Auth is **stubbed**: a hardcoded current user, `GET /api/v1/me` returns
it. The `users` table exists from migration 1 with two placeholder rows.
`httpapi` already routes through an auth middleware seam (ADR-03) so
NIU-4 swaps the implementation, not the shape.

Shipped: two boxes work end to end, 31 Go tests + 18 Playwright tests
green, audit findings resolved.

### NIU-4 — Authentication (story)

Scope: "Sign in with Google" button matching the warm design (§4), OAuth
2.0 / OpenID Connect authorization-code flow, ID token verification
against Google's OIDC discovery document (issuer, audience, signature —
never trust a client-supplied claim), email checked against
`NIU_ALLOWED_EMAILS`, Niu's own session cookie issued after a successful
verified login (`HttpOnly; Secure; Path=/; SameSite=Strict`, opaque
token, SHA-256 at rest — unchanged from the original design), logout,
session expiry and cleanup.

**Explicitly removed from scope** (§3 decision log): password storage,
bcrypt, brute-force rate limiting, password-reset flow — there is no
password. `users.password_hash` (migration 1) becomes dead and should be
dropped in this item's migration, not left as an unused column.

Replaces `auth.StubAuthenticator` (ADR-03) with a `GoogleAuthenticator`
behind the same `Authenticator` interface — `items_handlers.go` should
need zero changes.

Done when: every remaining §7.2 security test passes (S1, S2, S5, S7,
S8), an unauthorised email is rejected with a generic message, and a
person on the allowlist can complete the full login → use → logout cycle.

### NIU-2 — Deployment (task)

Scope: Dockerfile (multi-stage, distroless, non-root, static binary),
`compose.yaml` in both places, three GitHub Actions workflows, DNS A
record via OpenTofu, `backup-db.sh` SQLite support + prune fix, and a
documented restore test.

**Cloudflare Access is no longer part of this item.** With NIU-4 shipped
first, the app requires a verified Google login before any mutation is
possible — a direct proxied A record to Traefik's `websecure` entrypoint
is the same pattern `exam-room` already uses on this platform, no new
pattern required.

Needs `platform-engineer`. Touches the `fikua-platform-iac` repo — a
**separate PR in a separate repo**, coordinated with this one.

Done when: `https://niu.fikua.com` serves the app, only allowlisted
Google accounts can use it, a release triggers a deploy, a backup exists
in object storage and has been restored once.

> **Migration note for the in-flight PR.** `fikua-platform-iac` PR #15
> was written against the old plan (DNS committed commented out, gated on
> a Cloudflare Access application that no longer applies). It needs
> revision: drop the Access dependency, uncomment the DNS record — but
> only once NIU-4 is actually deployed, not before.

### NIU-3 — Observability (task)

Scope: OTEL SDK wiring, `otelhttp` middleware, DB spans, `user.id`
attribute, collector `config.yaml` change in the IaC repo, and
verification that traces actually land.

Explicitly a learning exercise as well as a deliverable — write it as the
reference Go instrumentation for the platform.

---

## 9. Mobile-readiness (explicitly not building this now)

Recorded so no agent "prepares" for it speculatively. The current design
is already mobile-ready by construction:

- API is JSON and separate from presentation → any client can call it.
- Session tokens are opaque and server-stored → moving from cookie to
  `Authorization: Bearer` is ~10 lines in one middleware, reusing all the
  token generation, hashing, expiry and revocation logic.
- `/api/v1/` prefix already in place → no dual-URL migration later.

If a Flutter app ever ships, the work is the app itself. The backend
changes are: read the bearer header as an alternative token source, and
keep the contract stable for old app versions (mobile users do not
auto-update — this is the one real difference from web).

**Do not implement bearer auth, offline sync, or API versioning beyond
`/v1` now.** Those decisions need information that does not exist yet.

---

## 10. Manifest configuration

`fikua.yml` at the repo root:

- `name: niu`, `language: { team: ca, model: en }`
- `backlog.type: markdown`, `issue_prefix: NIU`, `file: BACKLOG.md`
- `stack.backend: [go, chi]`, `frontend: [html, css, javascript]`,
  `database: [sqlite]`, `infra: [docker-compose, traefik, github-actions]`
- `architecture_pattern: hexagonal` (light — `internal/` boundaries)
- Commands: `test: "cd app && go test ./..."`,
  `lint: "cd app && gofmt -l"`, `typecheck: "cd app && go vet ./..."`,
  `build: "cd app && go build ./..."`, `up: "cd app && docker compose up -d"`
- Agents enabled beyond core: `ux_ui_designer: true` (look & feel is a
  first-class requirement), `platform_engineer: true` (NIU-2/3),
  `security_engineer: true` (NIU-4 and the §3 audit)

---

## 11. Rules for the executing agents

1. **Do not re-litigate §2, §3, §5, §7.** They were settled with the
   human. If you believe one is wrong, say so and stop — do not silently
   deviate.
2. **One item at a time**, in the §8 order. `single_item_per_session:
   true`.
3. **Stop at the first ambiguity.** Anything marked `[OPEN]` needs a
   human answer, not a guess.
4. **The test plan is the contract.** No story is done with a red suite,
   and no security row in §3 ships without its §7.2 test.
5. **Never commit or push without explicit approval** (framework
   invariant).
6. **Changes to `fikua-platform-iac` are separate PRs in that repo.** Do
   not mix them into Niu's history.
7. **Human-facing docs in Catalan; `.claude/` and this plan in English.**
