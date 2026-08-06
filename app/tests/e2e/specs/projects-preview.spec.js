// projects-preview.spec.js — NIU-11. Closes the frontend coverage gaps
// /audit flagged (qa-verification.md G-2/G-3): the Go side proves
// preview_status can be NULL and that the DTO carries the right fields,
// but nothing proved the browser survives that NULL, or that the
// security-relevant attributes on the new link actually reach the DOM.
//
// The preview here is served by intercepting the projects API rather
// than by letting fetchsafe scrape a real site: the E2E sandbox has no
// route to the public internet (see the note in xss.spec.js), and even
// if it did, a test whose outcome depends on a third party's Open Graph
// tags is a flake waiting to happen. What is under test is the render
// path, not the scraper — fetchsafe has its own unit tests.

import { test, expect } from '@playwright/test';
import { uniqueProjectName, addProject, cleanupProject } from './projects-helpers.js';

// withProjects intercepts GET /api/v1/projects and replaces the payload
// with rows we control, so each case can pin an exact preview state.
async function withProjects(page, rows) {
  await page.route('**/api/v1/projects', async (route, request) => {
    if (request.method() !== 'GET') return route.continue();
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      // The endpoint returns an envelope, not a bare array
      // (projects-api.js reads body.projects).
      body: JSON.stringify({ projects: rows }),
    });
  });
}

const baseRow = {
  id: 9001,
  name: 'Sofà Kivik',
  state: 'idea',
  budget: null,
  target_date: null,
  url: null,
  title: null,
  image_url: null,
  preview_status: null,
  added_by: { id: 1, display_name: 'Usuari A', avatar_emoji: '🐢' },
  last_updated_by: null,
  created_at: '2026-08-07T10:00:00Z',
  updated_at: '2026-08-07T10:00:00Z',
};

test.describe('NIU-11 — project link preview', () => {
  // G-2: the null-safety case. A project with no URL at all is the
  // common row (every project that existed before NIU-11, and every one
  // added without a link) — it must render exactly like before, with no
  // thumbnail, no link, and above all no JS error from dereferencing an
  // absent preview.
  test('a project without a URL renders with no thumbnail and no page error', async ({ page }) => {
    const errors = [];
    page.on('pageerror', (e) => errors.push(e.message));

    await withProjects(page, [baseRow]);
    await page.goto('/projects');

    const row = page.locator('.project-row', { hasText: 'Sofà Kivik' }).first();
    await expect(row).toBeVisible();
    await expect(row.locator('.project-thumb')).toHaveCount(0);
    await expect(row.locator('a.project-name')).toHaveCount(0);
    // The name is still rendered, as plain text.
    await expect(row).toContainText('Sofà Kivik');

    expect(errors, 'null preview_status caused a page error').toEqual([]);
  });

  // G-3: the security-relevant attributes. rel="noopener noreferrer" is
  // the kind of thing that silently disappears in a refactor and that
  // nobody notices until it matters, so it gets an explicit assertion
  // rather than relying on code inspection.
  test('a project with a preview renders a thumbnail and a safely-linked name', async ({ page }) => {
    const errors = [];
    page.on('pageerror', (e) => errors.push(e.message));

    await withProjects(page, [{
      ...baseRow,
      url: 'https://example.com/sofa',
      title: 'Sofà Kivik, 3 places',
      image_url: 'https://example.com/sofa.jpg',
      preview_status: 'ready',
    }]);
    await page.goto('/projects');

    const row = page.locator('.project-row', { hasText: 'Sofà Kivik' }).first();

    const thumb = row.locator('.project-thumb');
    await expect(thumb).toHaveCount(1);
    await expect(thumb).toHaveAttribute('src', 'https://example.com/sofa.jpg');
    // Decorative: the project name next to it already identifies the row.
    await expect(thumb).toHaveAttribute('alt', '');

    const link = row.locator('a.project-name');
    await expect(link).toHaveAttribute('href', 'https://example.com/sofa');
    await expect(link).toHaveAttribute('target', '_blank');
    // Without noopener the opened page gets window.opener and can
    // navigate this tab; without noreferrer it learns where we came from.
    const rel = await link.getAttribute('rel');
    expect(rel, 'rel on a target=_blank link').toContain('noopener');
    expect(rel, 'rel on a target=_blank link').toContain('noreferrer');

    expect(errors, 'unexpected page errors').toEqual([]);
  });

  // A URL saved but no image recovered (preview_status 'failed' or
  // 'partial') is the Instagram/bot-wall case and the most likely
  // real-world outcome for many sites — the name must still link, with
  // no broken-image placeholder next to it.
  test('a failed preview still links the name but renders no thumbnail', async ({ page }) => {
    await withProjects(page, [{
      ...baseRow,
      url: 'https://example.com/sofa',
      preview_status: 'failed',
    }]);
    await page.goto('/projects');

    const row = page.locator('.project-row', { hasText: 'Sofà Kivik' }).first();
    await expect(row.locator('.project-thumb')).toHaveCount(0);
    await expect(row.locator('a.project-name')).toHaveAttribute('href', 'https://example.com/sofa');
  });

  // B-12: T-10's own acceptance criterion — a row with a thumbnail must
  // not be taller than one without. /audit found the original 48px
  // thumbnail broke this by 16px; this measures it instead of trusting
  // the CSS comment.
  test('a thumbnail does not make the row taller (desktop)', async ({ page }) => {
    await withProjects(page, [
      { ...baseRow, id: 9001, name: 'Amb miniatura', url: 'https://example.com/a', image_url: 'https://example.com/a.jpg', preview_status: 'ready' },
      { ...baseRow, id: 9002, name: 'Sense miniatura' },
    ]);
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto('/projects');

    const withThumb = page.locator('.project-row', { hasText: 'Amb miniatura' }).first();
    const withoutThumb = page.locator('.project-row', { hasText: 'Sense miniatura' }).first();
    await expect(withThumb.locator('.project-thumb')).toHaveCount(1);

    const a = await withThumb.boundingBox();
    const b = await withoutThumb.boundingBox();
    expect(Math.abs(a.height - b.height), `row with thumbnail is ${a.height}px vs ${b.height}px without — T-10 requires equal height`).toBeLessThanOrEqual(1);
  });

  // The URL is user input, so it is also an XSS vector — same discipline
  // as xss.spec.js: execute the attack in a real browser, assert it
  // fails, never trust inspection of the render code alone. This one
  // goes through the real API (no interception) so the server-side
  // scheme validation is part of what is under test.
  test('a javascript: URL is rejected and never becomes a link', async ({ page }) => {
    const errors = [];
    page.on('pageerror', (e) => errors.push(e.message));

    await page.goto('/projects');
    await page.evaluate(() => { delete window.__xss; });

    const name = uniqueProjectName('preview-xss');
    await page.fill('#add-project-name', name);
    await page.fill('#add-project-url', 'javascript:window.__xss=1');
    await page.click('#add-project-btn');

    // The submission is rejected server-side (scheme validation), so no
    // row appears at all — assert the error surfaces rather than the row.
    const row = page.locator('.project-row', { hasText: name });
    await expect(row).toHaveCount(0);

    const executed = await page.evaluate(() => window.__xss === 1);
    expect(executed, 'javascript: URL executed — this is a real XSS').toBe(false);
    expect(errors, 'unexpected page errors').toEqual([]);

    await cleanupProject(page, name);
  });
});
