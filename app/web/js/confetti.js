// confetti.js — confetti() with a one-shot guard (§8.6.3, AC-14). Called
// by store.js only on the client-detected transition "A comprar ≥1 item"
// → "A comprar 0 items" caused by an action — never on initial load
// (EC-13) and never re-fired while it stays empty.

const COLORS = ['#3F6B4A', '#C1552C', '#E8C468']; // moss, terracotta, confetti-yellow
const PARTICLE_COUNT = 27; // ~24-30 per §8.6.3

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

export function confetti() {
  const layer = document.getElementById('confetti-layer');
  if (!layer) return;

  if (prefersReducedMotion()) {
    // §8.6.3 reduced-motion alternative: a single static flash instead of
    // moving particles.
    const emptyEl = document.getElementById('empty-shopping');
    if (emptyEl) {
      emptyEl.classList.add('flash');
      setTimeout(() => emptyEl.classList.remove('flash'), 600);
    }
    return;
  }

  const boxEl = document.getElementById('box-shopping');
  if (!boxEl) return;
  const boxRect = boxEl.getBoundingClientRect();
  const originX = boxRect.left + boxRect.width / 2;
  const originY = boxRect.top;

  for (let i = 0; i < PARTICLE_COUNT; i++) {
    const piece = document.createElement('span');
    piece.className = 'confetti-piece';
    piece.style.background = COLORS[i % COLORS.length];
    piece.style.left = `${originX + (Math.random() - 0.5) * 160}px`;
    piece.style.top = `${originY}px`;
    piece.style.animationDuration = `${1000 + Math.random() * 400}ms`; // ~1200ms
    layer.appendChild(piece);
    piece.addEventListener('animationend', () => piece.remove());
  }
}
