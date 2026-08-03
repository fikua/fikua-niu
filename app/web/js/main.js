// main.js — SPA entry point. Resolves identity EXACTLY ONCE per page
// load (GET /api/v1/me), then mounts both the shopping-list view and the
// projects view up front (both are cheap to keep in memory — the DOM
// visibility toggles per route, not the JS module lifecycle) and wires
// the client-side router.
//
// Before this merge, index.html and projects.html were two independent
// full pages, each with its own entry point that called GET /api/v1/me
// on every load — switching between "Compra" and "Projectes" meant two
// full page reloads, each re-resolving identity and re-fetching its list
// from scratch, on top of re-fetching manifest.json etc. Under normal
// two-person household usage this tripped Traefik's rate limit (429s,
// confirmed via browser console). Merging into one shell with a hand-
// rolled router means identity is resolved once, and clicking between
// views is a DOM toggle, not a network round trip.

import * as api from './api.js';
import { initAnnounce } from './a11y.js';
import { wireRouter } from './router.js';
import { logout } from './auth.js';
import { initShoppingView, prefetchShoppingItems } from './shopping-view.js';
import { initProjectsView, prefetchProjectsList } from './projects-view.js';

async function main() {
  // AC-05/design.md §7: resolve identity BEFORE mounting any UI. On 401,
  // api.getMe() already triggers the redirect to /login.html?next=... (via
  // api.js's centralized handleUnauthenticated) — returning here avoids
  // rendering so much as an empty list first (no flicker).
  //
  // GET /api/v1/items and GET /api/v1/projects are both kicked off in
  // parallel (not awaited here) so all three requests share one round
  // trip's worth of latency instead of being serialized — NFR-06 budgets
  // initial load at <1s even on a throttled connection.
  prefetchShoppingItems();
  prefetchProjectsList();

  let me;
  try {
    me = await api.getMe();
  } catch {
    return;
  }

  const liveRegion = document.getElementById('live-region');
  initAnnounce(liveRegion);

  wireRouter();

  const nameEl = document.getElementById('user-name');
  const avatarEl = document.getElementById('user-avatar');
  if (nameEl) nameEl.textContent = me.display_name;
  if (avatarEl) avatarEl.textContent = me.avatar_emoji;
  wireLogoutButton();

  // Both views are mounted up front regardless of which route is active —
  // both stores keep polling on their existing 10s intervals no matter
  // which view is currently visible. This is deliberate: it is what keeps
  // "other user's changes appear promptly" working even for the view
  // you're not looking at (design.md §5 Flux 3), and per-view start/stop
  // polling logic would be unnecessary complexity for a 2-person
  // household app with cheap SQLite reads.
  initShoppingView(me);
  initProjectsView(me);
}

function wireLogoutButton() {
  const btn = document.getElementById('logout-btn');
  if (!btn) return;
  btn.addEventListener('click', () => {
    logout();
  });
}

main();
