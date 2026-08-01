# Runbook — Deploy Niu

**Status of this doc:** written during NIU-2. Cloudflare Access (S10) is
**not yet configured** as of this writing — see the warning below before
doing anything that makes `niu.fikua.com` publicly resolvable.

## Overview

Niu deploys as a single Docker container (`fikua/niu`) behind Traefik on
the shared Fikua VPS, following the same pattern as `exam-room`. Full
detail: `PLAN.md` §5, and `fikua-platform-iac/projects/niu/README.md` for
the infra side.

```text
GitHub Release published
        │
        ▼
release.yml  ──build + push──▶  docker.io/fikua/niu (multi-arch)
        │
        ▼
deploy.yml   ──ssh (Cloudflare Access tunnel)──▶  VPS
        │
        ▼
docker compose pull && up -d   (niu-data-init chowns the volume first)
        │
        ▼
health poll (/healthz) + Traefik smoke test
```

## ⚠️ Before the first public deploy — Cloudflare Access (S10, blocking)

Niu ships with **stubbed authentication** (NIU-1) — no real login exists
until NIU-4. If `niu.fikua.com` resolves and is proxied through
Cloudflare without an Access policy in front of it, **anyone who
discovers the hostname can read and write the shared household list.**

This is not a follow-up task — it must exist before the DNS record is
created/proxied. Exact manual steps are documented in
`fikua-platform-iac/projects/niu/README.md` ("Cloudflare Access — read
before first deploy" section). In short: a self-hosted Cloudflare Access
application at `niu.fikua.com`, with an access policy scoped to the two
household members' real emails (not the `fikua` GitHub org — this is a
personal app, not a work tool).

**Do not manually enable the `deploy.yml` workflow trigger or manually
create the DNS record until that policy is confirmed live** (`curl -sI
https://niu.fikua.com/` should redirect to Cloudflare's Access login,
not reach the app).

## Cutting a release

1. Merge the change to `main` (build.yml runs `go vet` / `go test` /
   `go build` on every PR and push to `main`).
2. Tag and publish a GitHub Release (`vX.Y.Z`, semver). This triggers:
   - `release.yml` — builds and pushes `docker.io/fikua/niu` for
     `linux/amd64,linux/arm64` tagged `vX.Y.Z`, `X.Y`, `X`, `latest`,
     and `sha-<short-sha>`.
   - `deploy.yml` — deploys to the `prd` GitHub environment. If the
     environment has required reviewers configured, the run pauses
     until approved.
3. Watch the deploy job: it polls `/healthz` for up to 150s, then runs a
   Traefik smoke test with the `Host: niu.fikua.com` header.
4. Confirm from outside: `curl -I https://niu.fikua.com/healthz`
   (expect a redirect to Cloudflare Access once S10 is configured, or a
   direct `200` only if you're already authenticated in the browser
   session used for `curl`... in practice, verify via a real browser).

### Manual deploy (without a new release)

```bash
# From the GitHub UI: Actions -> "Deploy Niu to VPS" -> Run workflow
# Pick the image_tag input (defaults to "latest").
```

## Rolling back

There is no automatic rollback. To roll back to a previous image tag:

1. GitHub UI -> Actions -> "Deploy Niu to VPS" -> Run workflow.
2. Set `image_tag` to the previous known-good tag (e.g. `v1.2.2` or a
   `sha-xxxxxxx` tag from a prior successful `release.yml` run — check
   the Docker Hub tags list or the Actions history for the last good
   SHA).
3. Confirm the health poll and smoke test pass against the rolled-back
   version.

If the deploy is broken badly enough that even a redeploy of an old tag
won't apply cleanly, fall back to manual operations on the VPS (see
"Where the data lives" below for the SSH path), inspect
`docker compose logs niu`, and fix forward rather than trying to hand-edit
container state.

## Where the data lives

- **Database:** SQLite at `/data/niu.db` inside the `niu` container,
  backed by the named Docker volume `niu-data` on the VPS. Surviving a
  container recreate (`docker compose up -d`) is automatic — the volume
  is untouched by a redeploy. Surviving a **volume** loss requires a
  restore from backup (see below).
- **Backup:** daily, 03:00, via `scripts/backup-db.sh` in
  `fikua-platform-iac` (Ansible-managed cron). Niu is discovered via the
  `fikua.backup.sqlite=/data/niu.db` container label (a separate
  discovery pass from the existing Postgres `*-db` containers — see
  PLAN.md §5.5 for why the original script would have silently skipped
  it otherwise). Backups land in OVH Object Storage
  (`ovh-backups:fikua-db-backups/niu/`), pruned after 30 days by the
  same retention loop the Postgres backups use.
- **Restore:** see `fikua-platform-iac/docs/runbooks/disaster-recovery.md`
  for the general procedure. For Niu specifically:
  ```bash
  rclone copy ovh-backups:fikua-db-backups/niu/<latest>.sql.gz /tmp/
  # Note: unlike the Postgres dumps, this is a SQLite .backup file,
  # not SQL text — despite the .sql.gz extension inherited from the
  # shared naming convention, treat it as a binary SQLite database file.
  gunzip /tmp/<latest>.sql.gz
  # Stop the app, swap the file in the volume, restart:
  ssh vps.fikua.com 'sudo docker compose -f /opt/vps/projects/niu/compose.yaml stop niu'
  # copy the restored file into the niu-data volume, e.g. via a throwaway
  # container mounting the same volume, then:
  ssh vps.fikua.com 'sudo docker compose -f /opt/vps/projects/niu/compose.yaml start niu'
  ```
  **A backup that has never been restored is not a backup** (PLAN.md
  §5.5) — this restore path must be exercised at least once before
  NIU-2 is considered done, with the container's item count verified
  against a known-good state afterwards.
- **Config / secrets:** `.env` on the host at
  `/opt/vps/projects/niu/.env` (gitignored, never committed). Shape only
  is documented in `.env.example` in both this repo and
  `fikua-platform-iac/projects/niu/`.

## Observability

Once NIU-3 lands, traces/metrics/logs for `service.name = niu` are
visible at `https://observability.fikua.com` (Cloudflare Access,
existing GitHub IdP). Until NIU-3 ships, the only signal is
`docker compose logs niu` on the VPS and the `/healthz` endpoint.

## Related

- `PLAN.md` §5 — the binding deployment spec this runbook implements.
- `fikua-platform-iac/projects/niu/README.md` — infra-side setup
  checklist and the Cloudflare Access manual steps in full.
- `fikua-platform-iac/docs/runbooks/disaster-recovery.md` — general VPS
  and database recovery procedure.
- `fikua-platform-iac/docs/runbooks/access-vps.md` — how the SSH tunnel
  `deploy.yml` uses actually works.
