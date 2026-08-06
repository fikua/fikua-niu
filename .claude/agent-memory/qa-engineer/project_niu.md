---
name: project-niu
description: Niu project context — household app for two, re-audit discipline, test-plan principle
metadata:
  type: project
---

Niu is a Go + chi + SQLite household app for two people (see the
work-coordination/niu memory in the user's global MEMORY.md for the
broader architecture picture). QA-relevant specifics:

- **Testing principle (requirements.md §6, quoting `docs/test-plan.md`
  §2.1):** no security mitigation is trusted on code inspection alone —
  every security-relevant test must actually execute the attack (XSS,
  SQLi, SSRF, CSRF) in a real environment and assert its failure. This is
  the standard `qa-engineer` audits against; a mitigation that "looks
  correct" in the source (e.g. `textContent`-only rendering) is not
  sufficient without a behavioural test proving it.
- **Re-audit discipline:** established precedent (referenced in NIU-6
  `review.md` as following `NIU-5-*/review.md` §1.1) — on a re-audit pass,
  every previously-flagged fix must be re-verified from the actual diff/
  test run, not trusted from the commit message or from the prior round's
  narrative. This applies especially to renamed tests (see F-11 pattern
  below) where the test name can silently drift from what it proves.
- **F-11 pattern (worth watching for again):** a test can pass while not
  actually detecting the regression its name/comment claims to guard
  against, if its specific setup (e.g. loopback destination) trips an
  earlier check before the code path under test is ever reached. Caught
  by literal mutation testing (revert the fix, see if the test still
  passes). Renaming the test to describe what it actually proves (rather
  than force it to become behavioural) was accepted as a valid fix here.
- **E2E suite** (`app/tests/e2e`) boots a real Go binary + temp SQLite via
  `playwright.config.js`'s `webServer`, not a mocked backend — consistent
  with the "large tier of the pyramid" framing this project uses in
  `requirements.md` §6.
- **Multi-agent audit structure:** `code-reviewer` composes the final
  `review.md`; `qa-engineer`, `security-engineer`, and `ux-ui-designer`
  each write to their own scratch file first (e.g.
  `review-qa-section.md`), merged in afterward. Never write directly to
  `review.md` when running as `qa-engineer` in this project's `/audit`.

See also [[niu-project]] (global memory, architecture/execution-order
decisions) for the product-level picture this QA context sits on top of.
