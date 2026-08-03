// T-30 — AC-11 (full keyboard navigation), AC-12 (exact aria-live
// wording on own and remote-reflected changes), EC-14 (empty state on
// first use), EC-15 (mobile viewport keeps all functionality).
import { test, expect } from '@playwright/test';
import { uniqueProjectName, addProject, cleanupProject } from './projects-helpers.js';

// maxPresses is generous: the projects list is shared across the whole
// Playwright run (single DB, no per-test isolation, same pattern as
// NIU-1's keyboard-navigation.spec.js) and syncProjectsFromServer()'s
// 10s poll can re-sort rows to server order (oldest first) mid-test,
// pushing a newly-added row further from the focus start point than a
// tight budget would tolerate. A large ceiling costs nothing when the
// target is reached early (the loop returns as soon as it is).
async function tabUntilFocused(page, locator, maxPresses = 200) {
  for (let i = 0; i < maxPresses; i++) {
    if (await locator.evaluate((el) => el === document.activeElement).catch(() => false)) {
      return true;
    }
    await page.keyboard.press('Tab');
  }
  return locator.evaluate((el) => el === document.activeElement).catch(() => false);
}

// AC-11 — add, change state (any direction) and delete without a pointer
// device.
test('add, change state and delete a project using only the keyboard', async ({ page }) => {
  const name = uniqueProjectName('Keyboard');
  await page.goto('/projects.html');

  // Add via keyboard: focus input, type, Enter.
  await page.locator('#add-project-name').focus();
  await page.keyboard.type(name);
  await page.keyboard.press('Enter');

  const row = page.locator('.project-row', { hasText: name });
  await expect(row).toBeVisible();

  // Change state via keyboard: Tab to the "Decidit" badge button and
  // activate with Enter — a real three-option selector, not a "next
  // state" toggle (AC-09).
  const decidesBtn = row.locator('.state-badge.state-decidit');
  await page.locator('#add-project-name').focus();
  const reachedDecidit = await tabUntilFocused(page, decidesBtn);
  expect(reachedDecidit).toBe(true);
  await page.keyboard.press('Enter');
  await expect(decidesBtn).toHaveClass(/is-current/);

  // Reverse direction: back to "Idea", still via keyboard.
  const ideaBtn = row.locator('.state-badge.state-idea');
  await page.locator('#add-project-name').focus();
  const reachedIdea = await tabUntilFocused(page, ideaBtn);
  expect(reachedIdea).toBe(true);
  await page.keyboard.press('Enter');
  await expect(ideaBtn).toHaveClass(/is-current/);

  // Delete via keyboard.
  const deleteBtn = row.locator('.delete-btn');
  await page.locator('#add-project-name').focus();
  const reachedDelete = await tabUntilFocused(page, deleteBtn);
  expect(reachedDelete).toBe(true);
  await page.keyboard.press('Enter');

  await expect(page.locator('.project-row', { hasText: name })).toHaveCount(0);
});

// AC-12 — aria-live announces the exact "{nom} ara està {estat}." wording
// on a local state change.
test('aria-live announces the exact state-change wording for a local action', async ({ page }) => {
  const name = uniqueProjectName('Announce');
  await addProject(page, name);

  const row = page.locator('.project-row', { hasText: name });
  await row.locator('.state-badge.state-decidit').click();

  const liveRegion = page.locator('#live-region');
  await expect(liveRegion).toHaveText(`${name} ara està decidit.`, { timeout: 2000 });

  await cleanupProject(page, name);
});

// EC-14 — first use / zero projects shows a clear "no projects yet"
// message, never an error or a bare empty table.
test('empty state shows a clear message, no error', async ({ page }) => {
  await page.goto('/projects.html');

  const rows = page.locator('.project-row');
  const rowCount = await rows.count();

  if (rowCount === 0) {
    const empty = page.locator('#empty-projects');
    await expect(empty).toBeVisible();
    await expect(empty).toContainText('Cap projecte encara');
  } else {
    // Suite may run after other specs seeded projects — delete them all
    // to force and verify the empty state deterministically.
    for (let i = 0; i < rowCount; i++) {
      await rows.first().locator('.delete-btn').click({ force: true });
    }
    const empty = page.locator('#empty-projects');
    await expect(empty).toBeVisible();
    await expect(empty).toContainText('Cap projecte encara');
  }
});

// EC-15 — mobile viewport (375x667) keeps every action (add, change
// state, delete) available, same pattern as NIU-1's mobile-viewport.spec.js.
test.describe('mobile viewport', () => {
  test.use({ viewport: { width: 375, height: 667 } });

  test('mobile viewport supports add/change-state/delete', async ({ page }) => {
    const name = uniqueProjectName('Mobile');
    await page.goto('/projects.html');

    await page.fill('#add-project-name', name);
    await page.click('#add-project-btn');
    const row = page.locator('.project-row', { hasText: name });
    await expect(row).toBeVisible();

    await row.locator('.state-badge.state-fet').click();
    await expect(row.locator('.state-badge.state-fet')).toHaveClass(/is-current/);

    await row.locator('.delete-btn').click({ force: true });
    await expect(page.locator('.project-row', { hasText: name })).toHaveCount(0);
  });
});
