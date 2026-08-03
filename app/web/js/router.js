// router.js — small hand-rolled client-side router for Niu's single-page
// shell (index.html). Two routes only: "/" (shopping list) and
// "/projects" (home projects) — no route params, no nested routes, no
// external library (design.md §8: no build step, no framework).
//
// Responsibilities: map the current path to which <main data-view> is
// visible, update document.title and the nav's is-active class, and
// intercept same-origin nav-link clicks so switching views never triggers
// a full page reload (that reload was the root cause of the duplicated
// GET /api/v1/me + list-endpoint calls fixed by the SPA merge, see
// main.js).

const ROUTES = {
  '/': { view: 'shopping', title: 'niu' },
  '/projects': { view: 'projects', title: 'niu — Projectes' },
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
export function navigate(pathname, { replace = false } = {}) {
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
export function wireRouter() {
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
