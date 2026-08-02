---
artefact: design
key: "NIU-4"                  # REQUIRED — must match requirements.key
title: "<Same title as requirements>" # REQUIRED
status: "draft"                       # draft | in_review | approved | superseded
owner: "software-architect"
requirements_path: "./requirements.md" # REQUIRED — relative
adr_count: 0                          # REQUIRED — must equal the number of ADRs below
sources:
  - "arc42 (subset: §1 introduction, §4 solution strategy, §5 building blocks, §6 runtime, §8 cross-cutting, §11 risks)"
  - "ADR format (Michael Nygard, 2011)"
  - "C4 model — Levels 1 (context) and 2 (containers)"
created: "<YYYY-MM-DD>"
updated: "<YYYY-MM-DD>"
---

# Design — <Short title>

> **What this is.** The technical answer to the requirements. Architecture,
> key decisions, and contracts. **Implementation lives in code; this
> document explains the *shape* and the *why*.** Subset of arc42; never a
> full arc42 (we don't need 12 sections for one change).

## 1. Introduction and constraints (arc42 §1)

> Why we are designing this, and what fixed constraints shape it.

- **Goal of this change:** <one sentence>
- **Constraints (cannot be negotiated):**
  - Technical: <e.g. must run on existing Postgres, no new infra>
  - Organisational: <e.g. PII handling rules>
  - Time / cost: <e.g. ship before X>

## 2. Solution strategy (arc42 §4)

> The 5–10 highest-leverage decisions, in plain language. **No code,
> no detailed schemas yet.** Why this approach and not the obvious
> alternatives.

- …
- …

## 3. Architectural decisions (ADRs, Nygard format)

> One ADR per non-obvious decision. Keep them short (4–6 lines each).
> Reference each ADR by stable ID (`ADR-01`, …) from `tasks.md` and code
> comments where useful.

### ADR-01 — <Decision title>

- **Status:** proposed | accepted | superseded
- **Context:** <what forces are in play>
- **Decision:** <what we chose>
- **Consequences:** <good and bad outcomes>
- **Alternatives considered:** <one-line each>

### ADR-02 — <Decision title>

- **Status:** …
- **Context:** …
- **Decision:** …
- **Consequences:** …
- **Alternatives considered:** …

## 4. Building blocks (arc42 §5 + C4 Level 2)

> The components affected by this change and how they relate. Use a C4
> Container-level diagram or an ASCII sketch. **Show only what the change
> touches** — not the whole system.

```text
┌──────────────┐    ┌──────────────┐
│   Web app    │───▶│   API svc    │
└──────────────┘    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Database   │
                    └──────────────┘
```

- **<Component A>** — responsibility, interfaces touched
- **<Component B>** — …

## 5. Runtime view (arc42 §6)

> The 2–3 key flows that demonstrate the change works end-to-end. One
> sequence per flow, plain prose or a brief diagram.

**Flow 1 — <name>:**

1. <step>
2. <step>
3. <step>

## 6. Contracts and data model

> APIs, events, schemas, and database tables affected. **Just the surface
> area** — full schemas live in code.

### API / events

| Endpoint or event | Method | Request shape (high level) | Response shape (high level) |
|-------------------|--------|----------------------------|------------------------------|
| … | … | … | … |

### Data model deltas

| Entity | Change | Migration risk |
|--------|--------|----------------|
| … | new column / new table / index / … | LOW \| MED \| HIGH |

## 7. Cross-cutting concerns (arc42 §8)

> Aspects that touch every component of the change. Tick each subsection
> with the relevant strategy, or write "N/A" if it does not apply.

- **Security:** authn / authz changes, secrets, threat surface
- **Observability:** new logs, metrics, traces, alerts
- **Performance:** expected p95 / throughput / cache strategy
- **Resilience:** failure modes, retries, idempotency
- **Compliance & privacy:** data classification, retention, audit trail
- **Accessibility (if UI):** WCAG level targeted, keyboard/AT considerations
- **i18n / l10n:** translation needs

## 8. Risks (arc42 §11)

> Carry over the proposal risks and add any new technical risks surfaced
> while designing. Tie each to a mitigation owner.

| ID | Risk | Severity | Mitigation | Owner |
|----|------|----------|------------|-------|
| R-01 | … | LOW \| MED \| HIGH | … | … |

## 9. Open questions for the human gate

> Anything the human reviewer must decide before Stage 3 (tasks) starts.

- …
