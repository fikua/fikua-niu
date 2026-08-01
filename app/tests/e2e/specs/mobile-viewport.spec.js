// CF-19 — mobile viewport (375x667) stacks the boxes with tabs, and all
// actions (add, move, delete) still work.
import { test, expect, devices } from '@playwright/test';
import { uniqueName } from './helpers.js';

test.use({ viewport: { width: 375, height: 667 } });

test('mobile viewport shows tabs and supports add/move/delete', async ({ page }) => {
  const name = uniqueName('Mobile');
  await page.goto('/');

  const tabbar = page.locator('.tabbar');
  await expect(tabbar).toBeVisible();

  // Add (shopping tab is active by default).
  await page.fill('#add-input', name);
  await page.click('#add-btn');
  const shoppingRow = page.locator('#box-shopping .item-row', { hasText: name });
  await expect(shoppingRow).toBeVisible();

  // Move to pantry.
  await shoppingRow.click();

  // Switch tabs to see it landed in the pantry box.
  await page.click('.tab[data-panel="pantry"]');
  const pantryRow = page.locator('#box-pantry .item-row', { hasText: name });
  await expect(pantryRow).toBeVisible();

  // Delete it.
  await pantryRow.hover();
  await pantryRow.locator('.delete-btn').click({ force: true });
  await expect(page.locator('.item-row', { hasText: name })).toHaveCount(0);
});
