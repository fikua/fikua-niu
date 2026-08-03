// main.js — SPA entry point + hand-rolled client-side router. Resolves
// identity EXACTLY ONCE per page load (GET /api/v1/me), then mounts the
// shopping-list view (the default route) and wires the router; the
// projects view is loaded and mounted right after via a dynamic import
// (see the note above that import below).
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
//
// The router lives in this same file (rather than its own module) so the
// SPA shell's critical path is one file shorter — perf-3g.spec.js's <1s
// NFR-06 budget on a throttled connection is tight enough that every
// extra modulepreload'd file measurably matters (verified empirically:
// splitting it out cost ~100ms on a simulated 3G profile).

import * as api from './api.js';
import { initAnnounce } from './a11y.js';
import { logout } from './auth.js';
import { initShoppingView, prefetchShoppingItems } from './shopping-view.js';
// projects-view.js is intentionally loaded via a dynamic import() below,
// not a static one here — a static "import ... from './projects-view.js'"
// is fetched eagerly by the browser's module loader the moment main.js
// itself is parsed, regardless of when the imported binding is actually
// used. That eager fetch (plus the module graph it pulls in:
// projects-store.js, projects-render.js, projects-api.js) was found
// competing for bandwidth with the shopping view's critical first paint
// on a throttled 3G-class connection and pushed initial interactivity
// past the <1s NFR-06 budget (perf-3g.spec.js). A dynamic import() is the
// one construct that genuinely defers the fetch to when it is called.

// ================= Router =================
//
// Small hand-rolled client-side router for Niu's single-page shell.
// Three routes: "/" (shopping list), "/projects" (home projects), and
// "/ideas" (activity ideas, NIU-6) — no route params, no nested routes,
// no external library (design.md §8: no build step, no framework).

const ROUTES = {
  '/': { view: 'shopping', title: 'niu' },
  '/projects': { view: 'projects', title: 'niu — Projectes' },
  '/ideas': { view: 'ideas', title: 'niu — Idees' },
};

function routeFor(pathname) {
  return ROUTES[pathname] || ROUTES['/'];
}

// showView() toggles which <main data-view="..."> is visible via the
// `hidden` attribute, updates document.title, and syncs the nav's
// is-active class/aria-current — the only DOM effects of a route change.
function showView(pathname) {
  const route = routeFor(pathname);

  document.querySelectorAll('main[data-view]').forEach((main) => {
    main.hidden = main.dataset.view !== route.view;
  });

  document.title = route.title;

  document.querySelectorAll('.app-nav-link').forEach((link) => {
    const isActive = link.dataset.route === pathname
      || (link.dataset.route === '/' && !ROUTES[pathname]);
    link.classList.toggle('is-active', isActive);
    if (isActive) {
      link.setAttribute('aria-current', 'page');
    } else {
      link.removeAttribute('aria-current');
    }
  });
}

// navigate() pushes a new history entry (unless replace is requested) and
// renders the target route — used both by link clicks and by any
// programmatic navigation.
function navigate(pathname, { replace = false } = {}) {
  if (replace) {
    history.replaceState({}, '', pathname);
  } else if (window.location.pathname !== pathname) {
    history.pushState({}, '', pathname);
  }
  showView(pathname);
}

// wireRouter() intercepts clicks on same-origin .app-nav-link elements
// (preventDefault + pushState instead of a full navigation) and renders
// whatever route the browser lands on initially or via back/forward
// (popstate). Call once, at bootstrap.
function wireRouter() {
  document.querySelectorAll('.app-nav-link').forEach((link) => {
    link.addEventListener('click', (e) => {
      // Let modified clicks (open in new tab, etc.) behave natively.
      if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) {
        return;
      }
      const href = link.getAttribute('href');
      if (!ROUTES[href]) return; // unknown route — let the browser handle it
      e.preventDefault();
      navigate(href);
    });
  });

  window.addEventListener('popstate', () => {
    showView(window.location.pathname);
  });

  // Initial render for whatever URL the page was loaded with (direct
  // visit, bookmark, or a refresh while on /projects — the Go-side SPA
  // fallback in router.go serves this same shell for that request).
  showView(window.location.pathname);
}

// ================= Bootstrap =================

async function main() {
  // AC-05/design.md §7: resolve identity BEFORE mounting any UI. On 401,
  // api.getMe() already triggers the redirect to /login.html?next=... (via
  // api.js's centralized handleUnauthenticated) — returning here avoids
  // rendering so much as an empty list first (no flicker).
  //
  // GET /api/v1/items is kicked off in parallel with GET /api/v1/me (not
  // awaited here) so the two requests share one round trip's worth of
  // latency instead of being serialized — NFR-06 budgets initial load at
  // <1s even on a throttled connection.
  //
  // GET /api/v1/projects is deliberately NOT started here: on a throttled
  // 3G-class link, three concurrent requests (me + items + projects) plus
  // two module graphs worth of modulepreload compete for the same
  // constrained bandwidth and pushed initial interactivity past the <1s
  // budget (measured via perf-3g.spec.js after first merging both views'
  // prefetches eagerly). The shopping list is the default route (AC-05
  // lands on "/"), so it gets priority; the projects view — and its
  // GET /api/v1/projects — is initialized right after, once the shopping
  // view's own render pass has been scheduled, which in practice still
  // starts well within the same initial load and is invisible to the user
  // (only the shopping view is on screen at that point, per the SPA
  // route).
  prefetchShoppingItems();

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

  // The default-route view (shopping) is mounted first and wins the
  // initial-load bandwidth race. Both stores keep polling on their
  // existing 10s intervals once mounted, regardless of which view is
  // currently visible — this is deliberate: it is what keeps "other
  // user's changes appear promptly" working even for the view you're not
  // looking at (design.md §5 Flux 3), and per-view start/stop polling
  // logic would be unnecessary complexity for a 2-person household app
  // with cheap SQLite reads.
  initShoppingView(me);

  // Defer the projects view's module fetch, its own GET /api/v1/projects
  // prefetch, and its mount to right after the shopping view's critical
  // first render — see the import comment above for why this must be a
  // dynamic import(), not a static one.
  import('./projects-view.js').then(({ initProjectsView, prefetchProjectsList }) => {
    prefetchProjectsList();
    initProjectsView(me);
  });

  // NIU-6: same deferred-dynamic-import treatment as the projects view
  // above — the ideas view's own module graph (ideas-store.js,
  // ideas-render.js, ideas-api.js) and its GET /api/v1/ideas prefetch
  // must not compete with the shopping view's critical first paint on a
  // slow connection (NFR-06 budget, perf-3g.spec.js).
  import('./ideas-view.js').then(({ initIdeasView, prefetchIdeasList }) => {
    prefetchIdeasList();
    initIdeasView(me);
  });
}

function wireLogoutButton() {
  const btn = document.getElementById('logout-btn');
  if (!btn) return;
  btn.addEventListener('click', () => {
    logout();
  });
}

main();
