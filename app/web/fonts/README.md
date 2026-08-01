# Fonts — self-hosted Nunito (TODO)

This directory must contain `Nunito-Regular.woff2` (weight 400) and
`Nunito-Bold.woff2` (weight 700), per `proposal.md` §8.2 and `design.md`
§8. They are NOT present yet — the implementation environment used for
NIU-1 had no network access to fetch the real Google Fonts OFL binaries,
and per project rules a CDN link is never an acceptable substitute.

**TODO before shipping to production:** download the real Nunito
Regular/Bold `.woff2` files (OFL-licensed, downloadable for self-hosting
from Google Fonts) and place them here as:

- `Nunito-Regular.woff2`
- `Nunito-Bold.woff2`

Until then, `app.css` relies on the spec's own fallback stack
(`"Nunito", "Segoe UI", system-ui, -apple-system, sans-serif`) — the
`@font-face` rule is commented out, exactly as in
`design-system/tokens.css`, so the app still renders correctly with a
system rounded sans, just not pixel-identical to the final typography.
