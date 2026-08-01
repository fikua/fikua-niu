// CF-20/AC-13 — optimistic move + mocked server error reverts the item
// visually and shows a non-blocking toast.
import { test, expect } from '@playwright/test';
import { uniqueName, addItem, cleanupItem } from './helpers.js';

test('server error on move reverts the item and shows a toast', async ({ page }) => {
  const name = uniqueName('Rollback');
  await page.goto('/');
  await addItem(page, name);

  const row = page.locator('#box-shopping .item-row', { hasText: name });
  await expect(row).toBeVisible();

  // Intercept the PATCH for this item and force a server error.
  await page.route('**/api/v1/items/*', async (route) => {
    if (route.request().method() === 'PATCH') {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: { code: 'internal_error', message: "S'ha produït un error inesperat." } }),
      });
      return;
    }
    await route.continue();
  });

  await row.click();

  // Toast must appear.
  await expect(page.locator('.toast')).toBeVisible({ timeout: 2000 });
  await expect(page.locator('.toast')).toContainText(name);

  // Item must have rolled back to "A comprar".
  await expect(page.locator('#box-shopping .item-row', { hasText: name })).toBeVisible();
  await expect(page.locator('#box-pantry .item-row', { hasText: name })).toHaveCount(0);

  await page.unroute('**/api/v1/items/*');
  await cleanupItem(page, name);
});
