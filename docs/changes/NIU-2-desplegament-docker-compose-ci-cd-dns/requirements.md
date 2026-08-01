---
artefact: requirements
key: "NIU-2"                  # REQUIRED — must match proposal.key
title: "<Same title as proposal>"     # REQUIRED
status: "draft"                       # draft | in_review | approved | superseded
owner: "product-manager + qa-engineer"
proposal_path: "./proposal.md"        # REQUIRED — relative path
ac_count: 0                           # REQUIRED — must equal the number of AC IDs below
nfr_count: 0                          # REQUIRED — non-functional requirements count
sources:
  - "User Story format (Mike Cohn) — As a / I want / So that"
  - "INVEST — Independent, Negotiable, Valuable, Estimable, Small, Testable"
  - "Given/When/Then — Gherkin / Cucumber"
created: "<YYYY-MM-DD>"
updated: "<YYYY-MM-DD>"
---

# Requirements — <Short title>

> **What this is.** The contract between product and engineering. Every
> acceptance criterion is testable and traced to at least one task in
> `tasks.md`. **Functional behaviour only — no implementation detail.**

## 1. User story

> Mike Cohn format. One story per artefact. Multi-role stories: split into
> separate items at `/capture` time, do not bundle here.

- **As a** <canonical role>
- **I want** <capability>
- **So that** <outcome / value>

## 2. INVEST self-check

> Mark each property `✅ | ⚠️ | ❌`. Any `❌` blocks Stage 1 approval.

- [ ] **Independent** — can be planned and shipped without depending on another in-flight item
- [ ] **Negotiable** — open to design alternatives, not a fixed plan
- [ ] **Valuable** — directly serves the user or business outcome stated above
- [ ] **Estimable** — engineering can size it within an order of magnitude
- [ ] **Small** — fits in one sprint or planning unit; otherwise split
- [ ] **Testable** — every AC is observable from outside the system

## 3. Acceptance criteria

> Given/When/Then. **Each AC has a stable ID** (`AC-01`, `AC-02`, …) used
> by `tasks.md` for traceability. Keep ACs externally observable; do not
> reference internal classes or schemas.

### AC-01 — <Short name>

- **Given** <initial state>
- **When** <event or action>
- **Then** <observable outcome>

### AC-02 — <Short name>

- **Given** …
- **When** …
- **Then** …

> Add as many ACs as needed. Keep them atomic — one observable outcome each.

## 4. Edge cases and negative scenarios

> Scenarios that **must** behave well even though they are not the happy
> path. Use the same `EC-NN` ID pattern; tasks may reference them too.

### EC-01 — <Short name>

- **Given** …
- **When** …
- **Then** …

## 5. Non-functional requirements (NFRs)

> Performance, security, accessibility, compliance, observability,
> i18n/l10n. **Each NFR has a stable ID** (`NFR-01`, …) and a measurable
> target. Soft NFRs ("fast") are not acceptable.

| ID | Category | Statement | Target / threshold |
|----|----------|-----------|---------------------|
| NFR-01 | <perf \| sec \| a11y \| obs \| compliance \| …> | <statement> | <e.g. p95 < 300 ms> |

## 6. Testing strategy (drafted by `qa-engineer`)

> How each AC and NFR will be verified. The full test matrix is built in
> `/audit`; this section captures the **strategy**, not concrete cases.

- **Unit:** which ACs covered by unit tests
- **Integration:** which ACs need cross-component tests
- **E2E:** which ACs need full-stack tests
- **Manual / exploratory:** what remains for manual validation
- **NFR validation:** how each NFR is measured (load test, audit, manual review…)

## 7. Out of scope (explicit)

> Items explicitly excluded from this requirements set. Anything not
> listed here AND not in an AC is undefined behaviour — must not be
> assumed.

- …

## 8. Open questions

> Unresolved decisions that block Stage 2 approval. Each question must
> have a named owner.

- [ ] <question> — owner: <person>
