// T-28 — AC-07 (visual differentiation from Compra/Projectes), AC-09
// (full keyboard navigation: add + delete without a pointer device),
// AC-10 (accessible content — alt text, announced title/description/
// link), EC-18 (mobile viewport keeps all functionality). Same
// conventions as projects-visual-differentiation.spec.js/
// keyboard-navigation.spec.js/mobile-viewport.spec.js — no new harness.
//
// Every idea added here resolves to Estat B (fallback) — see
// ideas-helpers.js's header comment for why: fetchsafe correctly refuses
// to reach any destination available in this local E2E environment. That
// is the SSRF mitigation working as designed, not a limitation of these
// tests — Estat B is a fully valid AC-02 outcome and already exercises
// title/link/avatar/delete for AC-09/AC-10 purposes; Estat A/C's OG
// rendering contract is covered by the Go integration suite instead
// (tests/integration/ideas_test.go, against a controllable mock).
import { test, expect } from '@playwright/test';
import { uniqueIdeaURL, addIdea, waitForFallback, cleanupIdea, cardFor } from './ideas-helpers.js';

// AC-07 — the ideas space is clearly, visually differentiated from both
// the shopping list (moss) and the projects space (terracotta): its own
// nav accent (--color-mel) and its own grid layout (neither dual-box nor
// single-list).
test('navigation entry and accent colour clearly differ from Compra and Projectes', async ({ page }) => {
  await page.goto('/');

  const shoppingLink = page.locator('.app-nav-link', { hasText: 'Compra' });
  const projectsLink = page.locator('.app-nav-link', { hasText: 'Projectes' });
  const ideasLink = page.locator('.app-nav-link', { hasText: 'Idees' });
  await expect(shoppingLink).toBeVisible();
  await expect(projectsLink).toBeVisible();
  await expect(ideasLink).toBeVisible();

  const shoppingBorderColor = await shoppingLink.evaluate((el) => getComputedStyle(el).borderBottomColor);

  await projectsLink.click();
  await page.waitForURL('/projects');
  const activeProjectsLink = page.locator('.app-nav-link.is-active', { hasText: 'Projectes' });
  const projectsBorderColor = await activeProjectsLink.evaluate((el) => getComputedStyle(el).borderBottomColor);

  await ideasLink.click();
  await page.waitForURL('/ideas');
  await expect(page.locator('main[data-view="ideas"]')).toBeVisible();
  const activeIdeasLink = page.locator('.app-nav-link.is-active', { hasText: 'Idees' });
  await expect(activeIdeasLink).toBeVisible();
  const ideasBorderColor = await activeIdeasLink.evaluate((el) => getComputedStyle(el).borderBottomColor);

  // Three distinct accent colours, not just "different from one sibling".
  expect(ideasBorderColor).not.toBe(shoppingBorderColor);
  expect(ideasBorderColor).not.toBe(projectsBorderColor);

  const boxTitle = page.locator('.ideas-box .box-title');
  await expect(boxTitle).toBeVisible();
  const titleColor = await boxTitle.evaluate((el) => getComputedStyle(el).color);
  // --color-mel-hover is #A87D2A -> rgb(168, 125, 42).
  expect(titleColor).toBe('rgb(168, 125, 42)');

  // Layout differentiation: a CSS grid (auto-fill card grid), not the
  // dual-list (#list-shopping/#list-pantry) or single-list
  // (#projects-list) layouts of the other two spaces.
  const gridDisplay = await page.locator('#ideas-grid').evaluate((el) => getComputedStyle(el).display);
  expect(gridDisplay).toBe('grid');
});

// AC-09 — add a link and delete a card using only the keyboard.
async function tabUntilFocused(page, locator, maxPresses = 200) {
  for (let i = 0; i < maxPresses; i++) {
    if (await locator.evaluate((el) => el === document.activeElement).catch(() => false)) {
      return true;
    }
    await page.keyboard.press('Tab');
  }
  return locator.evaluate((el) => el === document.activeElement).catch(() => false);
}

test('add and delete an idea using only the keyboard', async ({ page }) => {
  const url = uniqueIdeaURL('keyboard');
  await page.goto('/ideas');

  await page.locator('#add-idea-url').focus();
  await page.keyboard.type(url);
  await page.keyboard.press('Enter');

  const card = cardFor(page, url);
  await expect(card).toBeVisible();

  const deleteBtn = card.locator('.delete-btn');
  await page.locator('#add-idea-url').focus();
  const reachedDelete = await tabUntilFocused(page, deleteBtn);
  expect(reachedDelete).toBe(true);
  await page.keyboard.press('Enter');

  await expect(cardFor(page, url)).toHaveCount(0);
});

// AC-10 — a card (with or without preview) announces title/description/
// link comprehensibly, and any image carries non-empty or deliberately
// empty (decorative, alt="") alternative text — never a missing alt.
test('fallback card is accessible to screen readers: title, link, and alt text', async ({ page }) => {
  const url = uniqueIdeaURL('a11y');
  await addIdea(page, url);
  const card = await waitForFallback(page, url);

  // Estat B: domain-as-title is present (never an empty string), the
  // full URL is shown as the only identifying link, and the fallback
  // message is present (not silence, not an error-styled announcement).
  await expect(card.locator('.idea-title')).not.toHaveText('');
  await expect(card.locator('.idea-link')).toContainText(url);
  await expect(card.locator('.idea-fallback-message')).toBeVisible();

  // No <img> at all in Estat B (proposal.md §8.2) — the substitute icon
  // is aria-hidden, never presented as meaningful content.
  await expect(card.locator('img')).toHaveCount(0);
  await expect(card.locator('.idea-fallback-icon')).toHaveAttribute('aria-hidden', 'true');

  await cleanupIdea(page, url);
});

// AC-01/AC-11 base contract: aria-live announces submission and
// resolution — verified here end-to-end (the exact wording is a
// documented, deliberate design choice, not incidental).
test('aria-live announces saving and the fallback resolution', async ({ page }) => {
  const url = uniqueIdeaURL('announce');
  await page.goto('/ideas');
  await page.fill('#add-idea-url', url);
  await page.click('#add-idea-btn');

  const live = page.locator('#live-region');
  await expect(live).toHaveText('Desant idea, recuperant previsualització…', { timeout: 2000 });

  await waitForFallback(page, url);
  await expect(live).toHaveText('Idea desada sense previsualització.', { timeout: 12000 });

  await cleanupIdea(page, url);
});

// EC-18 — mobile viewport (375x667) keeps every function working: add,
// view as a card, delete.
test.describe('mobile viewport', () => {
  test.use({ viewport: { width: 375, height: 667 } });

  test('mobile viewport supports add and delete, single-column grid', async ({ page }) => {
    const url = uniqueIdeaURL('mobile');
    await page.goto('/ideas');

    const gridColumns = await page.locator('#ideas-grid').evaluate((el) => getComputedStyle(el).gridTemplateColumns);
    // A single-column grid on mobile has exactly one column width listed.
    expect(gridColumns.trim().split(/\s+/).length).toBe(1);

    await page.fill('#add-idea-url', url);
    await page.click('#add-idea-btn');

    const card = cardFor(page, url);
    await expect(card).toBeVisible();

    await card.hover();
    await card.locator('.delete-btn').click({ force: true });
    await expect(cardFor(page, url)).toHaveCount(0);
  });
});
