---
artefact: tasks
key: "NIU-2"                  # REQUIRED — must match design.key
title: "<Same title as design>"       # REQUIRED
status: "draft"                       # draft | in_review | approved | in_progress | done
owner: "task-planner"
design_path: "./design.md"            # REQUIRED — relative
requirements_path: "./requirements.md" # REQUIRED — relative
task_count: 0                         # REQUIRED — total task items (T-NN)
ac_coverage:                          # REQUIRED — every AC from requirements MUST appear here
  - ac: "AC-01"
    tasks: []                         # list of T-NN ids that satisfy it
ec_coverage: []                       # optional — edge cases mapped to tasks
nfr_coverage: []                      # optional — NFRs mapped to tasks
sources:
  - "GitHub-style checklist (Markdown task lists)"
  - "Fikua AC↔tasks traceability matrix"
created: "<YYYY-MM-DD>"
updated: "<YYYY-MM-DD>"
---

# Tasks — <Short title>

> **What this is.** The implementation checklist. Each task is small
> (≤ ~1 hour), self-contained, and explicitly linked to at least one
> acceptance criterion. **No task without a covering AC; no AC without
> at least one task.** This file is the only mutable artefact during
> `/code` — the others are locked.

## 1. Task list

> Group tasks by phase or component if useful. Each task uses a stable
> `T-NN` ID. Mark `[x]` when done; the `/code` agent updates this in
> real time and the PostToolUse hook reads it.

### Foundations

- [ ] **T-01** — <what to do> · *covers:* AC-01
- [ ] **T-02** — <what to do> · *covers:* AC-01, AC-02
- [ ] **T-03** — <what to do> · *covers:* NFR-01

### Implementation

- [ ] **T-04** — …
- [ ] **T-05** — …

### Verification

- [ ] **T-06** — Add unit tests for AC-01 · *covers:* AC-01
- [ ] **T-07** — Add integration test for the AC-02 flow · *covers:* AC-02
- [ ] **T-08** — Validate NFR-01 with <method> · *covers:* NFR-01

### Closing (universal — all changes)

- [ ] **C-01** — Append changelog entry (`docs.changelog` from manifest)
- [ ] **C-02** — Transition backlog item to `Human Review` via the adapter
- [ ] **C-03** — Propose semver bump (ASK USER — never apply unattended)

## 2. AC ↔ tasks traceability matrix

> Mirrored in `ac_coverage` front-matter. Every AC must have at least
> one task. Every task in §1 must appear in at least one AC's `tasks` list.

| AC | Statement (short) | Covering tasks |
|----|--------------------|----------------|
| AC-01 | … | T-01, T-02, T-06 |
| AC-02 | … | T-02, T-07 |

## 3. Edge cases ↔ tasks

> Only if `requirements.md` defines edge cases.

| EC | Statement (short) | Covering tasks |
|----|--------------------|----------------|
| EC-01 | … | T-04 |

## 4. NFRs ↔ tasks

| NFR | Statement (short) | Covering tasks |
|-----|--------------------|----------------|
| NFR-01 | … | T-03, T-08 |

## 5. Out of scope (mirrored from design)

> Tasks-shaped reminder of what is NOT going to be done. Prevents scope
> creep during `/code`.

- …

## 6. Notes for the developer

> Anything the developer needs to know before starting, but that doesn't
> belong in `design.md`. Setup quirks, recommended local commands,
> debugging tips specific to this change. Keep it short.

- …
