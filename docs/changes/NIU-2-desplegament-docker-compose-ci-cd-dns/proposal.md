---
artefact: proposal
key: "NIU-2"                  # e.g. ACME-42 — REQUIRED
type: "story"                         # story | task | bug — REQUIRED
title: "<Short, action-led title>"    # REQUIRED
status: "draft"                       # draft | in_review | approved | superseded
owner: "<agent or person>"            # e.g. product-manager
parent_key: null                      # parent backlog item (for sub-items)
related_keys: []                      # other backlog items this proposal touches
sources:
  - "Lean Canvas (Ash Maurya) — problem/solution framing"
  - "Amazon PR/FAQ — narrative front half"
created: "<YYYY-MM-DD>"
updated: "<YYYY-MM-DD>"
---

# Proposal — <Short title>

> **What this is.** A one-page narrative that frames the problem, the
> proposed solution, the users it serves, and the value it delivers.
> Inspired by Amazon's PR/FAQ (front half only — no FAQ section) and the
> Lean Canvas blocks for problem/customer/value/solution. **Read top to
> bottom in under 3 minutes.**

## 1. Headline

> One sentence. Imagine this is the press-release headline for the change.

<Headline here>

## 2. Problem

> What pain or gap exists today, for whom, and why does it matter now?
> Bullet 2–4 problem statements with concrete evidence (data, quotes,
> incidents). Avoid solutioning here.

- …
- …

## 3. Customer

> Who experiences the problem? Use **canonical roles** from the project's
> glossary if it has one. Identify primary vs secondary users.

- **Primary:** <role>
- **Secondary:** <role>

## 4. Proposed solution

> What we are going to do, in plain language. **One paragraph max.** This
> is not the design — it is the *idea* of the change.

<Paragraph>

## 5. Value & success measure

> Why is this worth doing now? How will we know it worked?

- **Value:** <revenue | cost | risk | UX | strategic>
- **Success measure:** <one or two leading indicators, measurable>

## 6. Scope and non-scope

> Explicit boundaries. What is IN, what is OUT, what is DEFERRED. Prevents
> scope creep at `/define` Stage 2.

**In scope**

- …

**Out of scope (explicit)**

- …

**Deferred to a later change**

- …

## 7. Risks and unknowns

> Top 3–5 risks. Mark each as `LOW | MEDIUM | HIGH` and add a mitigation
> hypothesis. Unknowns to be resolved during `/define` go here too.

| Risk / unknown | Severity | Mitigation hypothesis |
|----------------|----------|------------------------|
| … | … | … |

## 8. Visuals (optional — populated by `ux-ui-designer` in Stage 1.5)

> Wireframes, screen flow, or mockup links. Empty if the change has no UI.

<empty | Figma link | inline diagram>

## 9. Open questions for the human gate

> Anything the human reviewer must decide before this proposal is
> approved and Stage 2 (design) starts.

- …
