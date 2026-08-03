// Shared Playwright helpers for the NIU-6 "idees" E2E suite. Same
// rationale as helpers.js/projects-helpers.js: the DB is shared across
// the whole run, so each test creates a uniquely-suffixed link and cleans
// it up afterwards.
//
// Every URL used here deliberately resolves to preview_status='failed'
// (fetchsafe correctly refuses any destination reachable from this local
// E2E environment — there is no publicly-routable test fixture to point
// at, and there should not be: SSRF mitigation is exactly what makes that
// true). Estat B (fallback) is a fully valid, spec-compliant state
// (AC-02) and is what these tests exercise; NIU-6's Go integration suite
// (tests/integration/ideas_test.go) already covers the OG-success/
// partial rendering contract at the API level with a controllable mock.

let counter = 0;

export function uniqueIdeaURL(prefix) {
  counter += 1;
  return `https://example.com/${prefix}-${Date.now()}-${counter}`;
}

// cardFor locates the idea's card by its link href, not by visible text —
// Estat D/A/C show only the domain ("example.com"), while Estat B shows
// the full URL (proposal.md §8.2); the href attribute is stable across
// every state, so this is the one selector that works regardless of
// which state the card is currently in.
export function cardFor(page, url) {
  return page.locator('.idea-card').filter({ has: page.locator(`a.idea-link[href="${url}"]`) }).first();
}

export async function addIdea(page, url) {
  await page.goto('/ideas');
  await page.fill('#add-idea-url', url);
  await page.click('#add-idea-btn');
  await cardFor(page, url).waitFor({ state: 'visible' });
}

// waitForFallback waits for the idea's transient "Recuperant..." card
// (Estat D) to resolve to Estat B (fallback) — fetchsafe's own 5s hard
// timeout bounds how long this can take (measured ~5.0s against a
// destination this sandboxed E2E environment cannot reach, e.g.
// example.com with no real internet egress); the 12s budget below gives
// comfortable margin over that floor plus poll-interval/network jitter.
export async function waitForFallback(page, url) {
  const card = cardFor(page, url);
  await card.locator('.idea-fallback-message').waitFor({ state: 'visible', timeout: 12000 });
  return card;
}

export async function cleanupIdea(page, url) {
  const card = cardFor(page, url);
  if (await card.count() === 0) return;
  await card.locator('.delete-btn').click({ force: true });
}
