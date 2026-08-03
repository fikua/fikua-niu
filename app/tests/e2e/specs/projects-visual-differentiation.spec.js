// T-21/AC-08 — the projects space is clearly, visually differentiated
// from the shopping list (NIU-1): a distinct navigation entry and a
// distinct accent colour (terracotta vs. moss green), per ADR-04. This
// automates the "distingeix clarament, només pel visual" contract by
// comparing computed styles between the two spaces — complementing the
// human visual spot-check noted in requirements.md §6 (ux-ui-designer/
// code-reviewer at /audit).
import { test, expect } from '@playwright/test';

test('navigation entry and accent colour clearly differ from the shopping list', async ({ page }) => {
  await page.goto('/');

  // Both nav entries are visible from the shopping list.
  const shoppingLink = page.locator('.app-nav-link', { hasText: 'Compra' });
  const projectsLink = page.locator('.app-nav-link', { hasText: 'Projectes' });
  await expect(shoppingLink).toBeVisible();
  await expect(projectsLink).toBeVisible();

  const shoppingBorderColor = await shoppingLink.evaluate((el) => getComputedStyle(el).borderBottomColor);

  await projectsLink.click();
  await page.waitForURL(/projects\.html/);

  const activeProjectsLink = page.locator('.app-nav-link.is-active', { hasText: 'Projectes' });
  await expect(activeProjectsLink).toBeVisible();
  const projectsBorderColor = await activeProjectsLink.evaluate((el) => getComputedStyle(el).borderBottomColor);

  // ADR-04: terracotta accent for /projects vs. moss green for the
  // shopping list — the two active-tab accent colours must differ.
  expect(projectsBorderColor).not.toBe(shoppingBorderColor);

  // The box title itself also carries the terracotta accent (app.css:
  // .projects-box .box-title { color: var(--color-terracotta) }).
  const boxTitle = page.locator('.projects-box .box-title');
  await expect(boxTitle).toBeVisible();
  const titleColor = await boxTitle.evaluate((el) => getComputedStyle(el).color);
  // Terracotta is #C1552C -> rgb(193, 85, 44).
  expect(titleColor).toBe('rgb(193, 85, 44)');
});
