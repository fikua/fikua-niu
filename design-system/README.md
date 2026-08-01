# Niu — design system preview

This directory is a **design reference**, not the shipped application. It
exists to let the product owner see and interact with the visual system
approved in `docs/changes/NIU-1-llista-de-la-compra-rebost-auth/proposal.md`
§8 (Visuals) before any backend or real client exists.

Every file here is a standalone HTML document — vanilla HTML, CSS and ES
modules, no build step, no framework, no CDN calls. Open any `.html` file
directly in a browser.

## What's here

| File | Contents |
|---|---|
| `tokens.css` | Every design token from §8.1/§8.2 as CSS custom properties. All other files import this — no file hardcodes a hex value. |
| `foundations-colors.html` | Palette swatches with the verified WCAG contrast ratio (§8.1.1) printed on each pair. |
| `foundations-type.html` | Type scale, radius scale, shadow scale, spacing scale (§8.1/§8.2). |
| `component-item-row.html` | `ItemRow` (§8.4.2): default, hover, focus-visible, active, pending, rollback, single- and dual-avatar variants. |
| `component-add-input.html` | `AddItemInput` (§8.4.3): default, focus, submitting, and every validation/duplicate error state with its exact Catalan message. |
| `component-toast-empty.html` | `Toast` (§8.4.4) and `EmptyState` (§8.4.5), including both "A comprar" and "Rebost" empty variants. |
| `component-tabbar-avatar.html` | `TabBar` (§8.4.6) and `Avatar` (§8.4.7), including the dual-avatar `↩` pattern. |
| `screen-desktop.html` | Full desktop two-box layout (§8.3.1) — **interactive**: click a row to move it between boxes with the real FLIP animation, honouring `prefers-reduced-motion`, firing confetti once when "A comprar" empties. |
| `screen-mobile.html` | Mobile stacked/tabs layout (§8.3.2), same interactivity, 44×44px touch targets. |

## What this is NOT

- Not the shipped app. The real implementation lives in `app/web/` and
  talks to the Go API described in `PLAN.md` §2.5.
- Not a Figma replacement — §8 of the proposal is still the single
  source of truth for exact values; these files are its executable proof.
- Not production-ready fonts. Nunito is declared as self-hosted per the
  spec, but the actual `.woff2` binaries aren't present here — see the
  comment at the top of `tokens.css` for what to drop into
  `app/web/fonts/` at implementation time.

## Fonts note

The spec (§8.2) mandates self-hosted Nunito, two static weights only
(Regular 400, Bold 700), served by the Go binary — no Google Fonts, no
external `font-src` (the CSP forbids it and it leaks the user's IP). This
preview cannot ship font binaries, so it relies on the spec's own fallback
stack: `"Nunito", "Segoe UI", system-ui, -apple-system, sans-serif`. The
`@font-face` rule is present in `tokens.css`, commented out, ready to
uncomment once the two `.woff2` files exist.
