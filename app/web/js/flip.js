// flip.js — FLIP animation (First-Last-Invert-Play), direct port of
// captureRects()/playFlip() from design-system/screen-desktop.html
// (§8.6.1 FLIP, §8.6.2 reduced-motion cross-fade).

const FLIP_DURATION = 250; // ms, §8.6.1
const FLIP_EASING = 'cubic-bezier(0, 0, 0.2, 1)';
const CROSSFADE_DURATION = 150; // ms, §8.6.2 (150 out + 150 in = ~300 total)

const reduceMotionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');

export function prefersReducedMotion() {
  return reduceMotionQuery.matches;
}

// captureRects snapshots the bounding rect of every currently-rendered
// .item-row, keyed by item id. Call this BEFORE mutating the DOM.
export function captureRects() {
  const rects = new Map();
  document.querySelectorAll('.item-row').forEach((row) => {
    rects.set(row.dataset.id, row.getBoundingClientRect());
  });
  return rects;
}

// playFlip animates every row that existed in firstRects and still exists
// in the DOM now, from its previous position to its current one
// (§8.6.1). Rows with no previous rect (newly created) are skipped here —
// they get their own .just-added fade-in instead (render.js).
export function playFlip(firstRects) {
  const lastRects = captureRects();
  lastRects.forEach((lastRect, id) => {
    const firstRect = firstRects.get(id);
    if (!firstRect) return; // newly created row, nothing to animate from
    const dx = firstRect.left - lastRect.left;
    const dy = firstRect.top - lastRect.top;
    if (dx === 0 && dy === 0) return;
    const row = document.querySelector(`.item-row[data-id="${id}"]`);
    if (!row) return;

    if (prefersReducedMotion()) {
      // §8.6.2 — cross-fade instead of flight, no transform/position shift.
      row.animate(
        [{ opacity: 0 }, { opacity: 1 }],
        { duration: CROSSFADE_DURATION, easing: 'ease-out', fill: 'both' }
      );
      return;
    }

    // §8.6.1 — FLIP via Web Animations API, transform only.
    row.animate(
      [
        { transform: `translate(${dx}px, ${dy}px) scale(1)`, boxShadow: 'var(--shadow-sm)', offset: 0 },
        { transform: `translate(${dx * 0.4}px, ${dy * 0.4}px) scale(1.03)`, boxShadow: 'var(--shadow-lg)', offset: 0.5 },
        { transform: 'translate(0, 0) scale(1)', boxShadow: 'var(--shadow-sm)', offset: 1 },
      ],
      { duration: FLIP_DURATION, easing: FLIP_EASING, fill: 'both' }
    );
  });
}

export { FLIP_DURATION, CROSSFADE_DURATION };
