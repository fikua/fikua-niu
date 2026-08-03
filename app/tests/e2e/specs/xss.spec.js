// S3a — XSS. The test-plan rule is that a security case must *perform the
// attack and assert it fails*, not assert that a mitigation exists.
//
// The Go integration suite already proves the server stores hostile input
// verbatim rather than escaping it. That is deliberately only half of
// S3a: storing `<img src=x onerror=...>` as literal text is correct, and
// the actual question is what the browser does when that text is
// rendered. Only a real browser can answer it — which is why this file
// exists.
//
// This gap was found during /audit: S3a was marked green while nothing
// executed the payload in a browser, and the design-system previews these
// screens were ported from do use innerHTML.

import { test, expect } from '@playwright/test';
import { uniqueName, addItem, cleanupItem } from './helpers.js';
import { uniqueProjectName, addProject, cleanupProject } from './projects-helpers.js';

// Each payload is a different injection vector, not a variation of one.
const PAYLOADS = [
  { label: 'img onerror', value: '<img src=x onerror="window.__xss=1">' },
  { label: 'script tag', value: '<script>window.__xss=1</script>' },
  { label: 'svg onload', value: '<svg onload="window.__xss=1">' },
  { label: 'anchor javascript: URI', value: '<a href="javascript:window.__xss=1">clica</a>' },
  { label: 'closing-tag breakout', value: '"><script>window.__xss=1</script>' },
];

test.describe('S3a — XSS payloads render as literal text', () => {
  for (const payload of PAYLOADS) {
    test(`${payload.label} does not execute`, async ({ page }) => {
      const errors = [];
      page.on('pageerror', (e) => errors.push(e.message));

      await page.goto('/');
      await page.evaluate(() => { delete window.__xss; });

      const name = `${uniqueName('xss')} ${payload.value}`;
      await addItem(page, name);

      // 1. The payload did not run.
      const executed = await page.evaluate(() => window.__xss === 1);
      expect(executed, `${payload.label} executed — this is a real XSS`).toBe(false);

      // 2. No element was created from the payload. If the markup had
      //    been parsed as HTML rather than inserted as text, these
      //    selectors would match.
      const row = page.locator('.item-row', { hasText: name }).first();
      await expect(row.locator('img, script, svg, a')).toHaveCount(0);

      // 3. The text is present verbatim — the item is still usable, we
      //    have not silently mangled the user's input.
      await expect(row).toContainText(payload.value);

      expect(errors, 'unexpected page errors').toEqual([]);

      await cleanupItem(page, name);
    });
  }

  // The payload also passes through the toast and the aria-live region,
  // which are separate render paths from the item row. A textContent-only
  // rule applied in one place and forgotten in another is exactly how XSS
  // survives a review.
  test('payload stays inert in the aria-live announcement', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => { delete window.__xss; });

    const name = `${uniqueName('xss-live')} <img src=x onerror="window.__xss=1">`;
    await addItem(page, name);

    const row = page.locator('.item-row', { hasText: name }).first();
    await row.click();

    const live = page.locator('[aria-live]');
    await expect(live).toContainText('mogut a', { timeout: 3000 });

    const executed = await page.evaluate(() => window.__xss === 1);
    expect(executed, 'payload executed via the aria-live path').toBe(false);
    await expect(live.locator('img, script, svg')).toHaveCount(0);

    await cleanupItem(page, name);
  });

  // S3b belongs here too: the CSP is what makes an injected inline script
  // fail even if a rendering bug ever let one through. Asserting it from
  // the browser (not just from a Go header test) confirms the policy the
  // page actually runs under.
  test('CSP served to the browser forbids inline script', async ({ page }) => {
    const response = await page.goto('/');
    const csp = response.headers()['content-security-policy'];

    expect(csp, 'no CSP header on the document').toBeTruthy();
    expect(csp).not.toContain("'unsafe-inline'");
    expect(csp).not.toContain("'unsafe-eval'");
    expect(csp).toContain("default-src 'self'");
  });

  // NIU-5/EC-08/NFR-02: same attack, same assertion, applied to the
  // projects space's own render path (projects-render.js), a separate
  // module from render.js.
  test('img onerror does not execute in the projects space', async ({ page }) => {
    const errors = [];
    page.on('pageerror', (e) => errors.push(e.message));

    await page.goto('/projects.html');
    await page.evaluate(() => { delete window.__xss; });

    const payload = '<img src=x onerror="window.__xss=1">';
    const name = `${uniqueProjectName('xss-project')} ${payload}`;
    await addProject(page, name);

    const executed = await page.evaluate(() => window.__xss === 1);
    expect(executed, 'projects XSS payload executed — this is a real XSS').toBe(false);

    const row = page.locator('.project-row', { hasText: name }).first();
    await expect(row.locator('img, script, svg, a')).toHaveCount(0);
    await expect(row).toContainText(payload);

    expect(errors, 'unexpected page errors').toEqual([]);

    await cleanupProject(page, name);
  });
});
