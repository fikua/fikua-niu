// CF-22/AC-16 — aria-live region announces the exact wording from
// proposal.md §8.7 when an item moves via local action.
import { test, expect } from '@playwright/test';
import { uniqueName, addItem, cleanupItem } from './helpers.js';

test('aria-live announces the exact move wording for a local action', async ({ page }) => {
  const name = uniqueName('Announce');
  await page.goto('/');
  await addItem(page, name);

  const row = page.locator('#box-shopping .item-row', { hasText: name });
  await row.click();

  const liveRegion = page.locator('#live-region');
  await expect(liveRegion).toHaveText(`${name} mogut a Rebost.`, { timeout: 2000 });

  await cleanupItem(page, name);
});
