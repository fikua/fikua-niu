---
name: feedback-token-values-may-drift-post-approval
description: Token hex values approved at Stage 1.5 may be legitimately retuned downstream by an automated contrast audit — check proposal.md against shipped CSS at /audit rather than assuming byte-for-byte fidelity
metadata:
  type: feedback
---

When auditing a `/define`d item's implementation against my own Stage 1.5
proposal.md §8, do not assume the exact hex values I proposed are what
ships. `fullstack-developer` may retune a token slightly darker/lighter
in response to a downstream automated contrast audit (axe-core) run
during `/code`, if my originally-proposed shade fails 4.5:1 against a
specific surface (e.g. white button text) that I did not explicitly
contrast-check for that specific use in §8.0.

**Why:** observed on NIU-6 — I proposed `--color-mel-hover: #B0842A` /
`--color-mel-active: #966E20`, verified only against `color.bg` as the
background. The implementation shipped `#A87D2A` / `#8C6721` instead,
after its own axe-core run (T-28) found the button-fill contrast case I
hadn't separately verified. This is a legitimate, well-documented,
accessibility-driven fix — not a rogue deviation — as long as: (1) it
stays within the same approved hue/token family (not a new colour), (2)
it's documented in-code with the actual contrast ratios, and (3) it
doesn't touch the *use* of the token (still nav underline + button fill,
same as approved), only the hex value.

**How to apply:** at `/audit`, flag this kind of drift as **non-blocking**
documentation hygiene (recommend syncing proposal.md's token table to the
shipped hex), not as a spec-conformance failure — but always verify by
reading the actual shipped CSS token values against my proposal's table,
since "looks like the same colour" is not the same as "is the value I
wrote down." Consider, going forward, contrast-checking new accent tokens
against BOTH the page background AND a representative filled-button/white-text
case in §8.0 itself, to reduce the odds of this drift happening at all.
