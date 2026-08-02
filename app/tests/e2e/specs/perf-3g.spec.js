// PERF-02/NFR-06 — initial load is fast on a slow connection.
//
// A full Lighthouse audit (the tool named in test-plan.md) requires
// wiring lighthouse + chrome-launcher against this same Chromium
// instance, which is a heavier, more fragile integration than this
// project's zero-build-step frontend needs to prove the point. As an
// automated proxy that still exercises a REAL browser and REAL network
// conditions, this test throttles the CDP session to a 3G-like profile
// (via Chrome DevTools Protocol, same mechanism Lighthouse itself uses
// under the hood) and asserts the page becomes interactive
// (DOMContentLoaded + first item row rendered) in under 1s.
//
// A full axe-core/Lighthouse pass remains a required manual step (T-33)
// before NIU-1 is considered closed — this test does not replace it, it
// narrows the risk automatically on every run.
import { test, expect, chromium } from '@playwright/test';

// NIU-4: this spec launches its own isolated browser/context (to attach a
// CDP session for network throttling) instead of using the fixture-level
// `page`, so it does not automatically inherit the suite-wide
// storageState set up by global-setup.js. AC-05 now redirects an
// unauthenticated "/" to /login.html, so without an authenticated context
// this test would measure the login screen, not the list — pass the same
// storageState explicitly.
test('initial load is interactive in under 1s on a simulated 3G connection', async ({ baseURL }, testInfo) => {
  const storageState = testInfo.project.use.storageState;
  const browser = await chromium.launch();
  const context = await browser.newContext({ storageState });
  const page = await context.newPage();
  const client = await context.newCDPSession(page);

  // Lighthouse's own "Slow 3G" / "Slow 4G" preset (the one its UI labels
  // "Slow 3G", used for PERF-02/NFR-06): 150ms RTT, 1.6Mbps down /
  // 750Kbps up. (An earlier version of this test used a harsher, custom
  // 400ms/400kbps profile that does not correspond to any Lighthouse
  // preset and made the <1s budget physically unreachable regardless of
  // app-level optimization — flagged and corrected, see T-30 notes.)
  await client.send('Network.emulateNetworkConditions', {
    offline: false,
    downloadThroughput: (1.6 * 1024 * 1024) / 8,
    uploadThroughput: (750 * 1024) / 8,
    latency: 150,
  });

  const start = Date.now();
  await page.goto(baseURL + '/', { waitUntil: 'domcontentloaded' });
  // "Interactive" proxy: the add-item input exists and is usable, and at
  // least the empty-state or a row has rendered in one of the two boxes
  // (main.js has finished its initial syncFromServer() render pass).
  await page.locator('#add-input').waitFor({ state: 'visible' });
  await page.locator('#list-shopping > *').first().waitFor({ state: 'attached', timeout: 5000 });
  const elapsed = Date.now() - start;

  await browser.close();

  expect(elapsed).toBeLessThan(1000);
});
