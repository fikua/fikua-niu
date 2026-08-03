// T-33 — automated accessibility audit (axe-core) covering EC-17/A11Y-02
// (colour contrast) and general WCAG 2.2 AA rule violations across both
// the desktop and mobile layouts, with at least one item present in each
// box (so ItemRow/Avatar/delete-btn contrast is actually exercised, not
// just the empty state).
//
// This automates the tooling half of T-33. The screen-reader-in-hand
// verification (A11Y-03) and the final human sign-off remain manual —
// documented in this file's header and in STATUS.md at archive time.
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { uniqueName, addItem, cleanupItem } from './helpers.js';
import { uniqueProjectName, addProject, cleanupProject } from './projects-helpers.js';
import { uniqueIdeaURL, addIdea, cleanupIdea } from './ideas-helpers.js';

test.describe('axe-core WCAG 2.2 AA audit', () => {
  test('desktop layout has no automatically-detectable accessibility violations', async ({ page }) => {
    const name = uniqueName('A11yDesktop');
    await page.goto('/');
    await addItem(page, name);

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag22aa'])
      .analyze();

    expect(results.violations, JSON.stringify(results.violations, null, 2)).toEqual([]);

    await cleanupItem(page, name);
  });

  test('mobile layout has no automatically-detectable accessibility violations', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    const name = uniqueName('A11yMobile');
    await page.goto('/');
    await addItem(page, name);

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag22aa'])
      .analyze();

    expect(results.violations, JSON.stringify(results.violations, null, 2)).toEqual([]);

    await cleanupItem(page, name);
  });

  // NIU-5/NFR-06: same automated audit applied to the projects space, with
  // at least one row present (state badges, avatars, delete-btn contrast).
  test('projects space has no automatically-detectable accessibility violations', async ({ page }) => {
    const name = uniqueProjectName('A11yProjects');
    await addProject(page, name);

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag22aa'])
      .analyze();

    expect(results.violations, JSON.stringify(results.violations, null, 2)).toEqual([]);

    await cleanupProject(page, name);
  });

  // NIU-6/AC-10: same automated audit applied to the ideas space. The
  // idea used here is added before the scrape has resolved, so both the
  // "Recuperant..." (Estat D) shimmer/placeholder AND, after axe runs
  // against it, the settled fallback (Estat B) card get exercised across
  // the two test cases below.
  test('ideas space has no automatically-detectable accessibility violations (pending card)', async ({ page }) => {
    const url = uniqueIdeaURL('A11yIdeasPending');
    await page.goto('/ideas');
    await page.fill('#add-idea-url', url);
    await page.click('#add-idea-btn');

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag22aa'])
      .analyze();

    expect(results.violations, JSON.stringify(results.violations, null, 2)).toEqual([]);

    await cleanupIdea(page, url);
  });

  test('ideas space has no automatically-detectable accessibility violations (fallback card)', async ({ page }) => {
    const url = uniqueIdeaURL('A11yIdeasFallback');
    await addIdea(page, url);
    // Give the scrape enough time to resolve to the fallback state before
    // auditing it (see ideas-helpers.js — no reachable OG target exists
    // in this environment, so this always settles to Estat B).
    await page.waitForTimeout(6000);

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag22aa'])
      .analyze();

    expect(results.violations, JSON.stringify(results.violations, null, 2)).toEqual([]);

    await cleanupIdea(page, url);
  });
});
