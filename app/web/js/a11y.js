// a11y.js — aria-live announcements (§8.7 exact wording) and mobile
// managed-tabindex helpers for the "delete" button (only tabbable while
// its row has focus).

import { t } from './strings.js';

let liveRegion = null;

export function initAnnounce(el) {
  liveRegion = el;
}

export function boxLabel(location) {
  return location === 'shopping' ? t('boxShopping') : t('boxPantry');
}

// announce writes the exact wording from proposal.md §8.7:
//   "{Nom} mogut a {caixa}."                      — local action
//   "{Nom} mogut a {caixa} per {usuari}."          — remote change (poll)
export function announce(text) {
  if (!liveRegion) return;
  liveRegion.textContent = '';
  // Force a DOM mutation even if the text is identical to the previous
  // announcement, so assistive tech re-announces it.
  requestAnimationFrame(() => {
    liveRegion.textContent = text;
  });
}

export function announceMove(itemName, location, movedByDisplayName) {
  const target = boxLabel(location);
  if (movedByDisplayName) {
    announce(t('announceMovedBy', itemName, target, movedByDisplayName));
  } else {
    announce(t('announceMoved', itemName, target));
  }
}

// announceProjectStateChange writes the exact wording required by
// design.md §7/§8 (AC-12/NFR-07): "{nom} ara està {estat}." — on both a
// local action and a remote change reflected via polling.
const PROJECT_STATE_LABELS = {
  idea: 'idea',
  decidit: 'decidit',
  fet: 'fet',
};

export function announceProjectStateChange(projectName, state) {
  const label = PROJECT_STATE_LABELS[state] || state;
  announce(`${projectName} ara està ${label}.`);
}

// wireRowFocusTabindex used to manage the delete button's `tabindex`
// dynamically so it would only join the tab order while its row had
// focus. That approach relied on the row ITSELF being the sole
// interactive control (a <button> containing a nested <button>), which
// axe-core flags as WCAG 4.1.2 "Interactive controls must not be nested"
// (found via the T-33 accessibility audit) — and separately, hiding a
// focusable element with `visibility: hidden` turned out to exclude it
// from Chromium's Tab order entirely, even once a CSS rule made it
// visible again (found via the T-29 keyboard-navigation spec).
//
// render.js now builds each row as a non-interactive container with TWO
// independent sibling <button>s (the full-cover "move" target and the
// delete button) — both are native buttons, always genuinely focusable,
// with no tabindex gymnastics required. Their VISUAL prominence (the
// delete button is invisible until hovered/focused) is handled purely in
// CSS via `opacity`/`pointer-events`, which does not affect focusability
// or tab order at all. This function is kept as a no-op shim so
// render.js's call site does not need to change shape, but there is
// nothing left to wire.
export function wireRowFocusTabindex(_row, _moveBtn, _deleteBtn) {
  // Intentionally empty — see comment above.
}
