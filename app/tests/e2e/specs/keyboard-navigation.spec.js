// CF-21/AC-15 — full keyboard navigation: add, move, delete without a
// pointer device. Uses real keyboard Tab presses (not .focus()).
//
// Each row is a non-interactive container (WCAG 4.1.2 fix, T-33) with
// two independent sibling <button>s: `.item-row-move-target` (a
// full-cover invisible button that makes the whole row act as the
// "move" control, matching the approved visual spec) and `.delete-btn`.
// Keyboard interaction targets these buttons directly, not the row.
import { test, expect } from '@playwright/test';
import { uniqueName } from './helpers.js';

async function tabUntilFocused(page, locator, maxPresses = 30) {
  for (let i = 0; i < maxPresses; i++) {
    if (await locator.evaluate((el) => el === document.activeElement).catch(() => false)) {
      return true;
    }
    await page.keyboard.press('Tab');
  }
  return locator.evaluate((el) => el === document.activeElement).catch(() => false);
}

test('add, move and delete an item using only the keyboard', async ({ page }) => {
  const name = uniqueName('Keyboard');
  await page.goto('/');

  // Add via keyboard: focus input, type, Enter.
  await page.locator('#add-input').focus();
  await page.keyboard.type(name);
  await page.keyboard.press('Enter');

  const row = page.locator('#box-shopping .item-row', { hasText: name });
  await expect(row).toBeVisible();
  const moveTarget = row.locator('.item-row-move-target');

  // Move via keyboard: Tab from the add input until our new row's move
  // button (added at the top of "A comprar") is focused, and activate
  // with Enter.
  await page.locator('#add-input').focus();
  const reachedMoveTarget = await tabUntilFocused(page, moveTarget);
  expect(reachedMoveTarget).toBe(true);
  await page.keyboard.press('Enter');

  const pantryRow = page.locator('#box-pantry .item-row', { hasText: name });
  await expect(pantryRow).toBeVisible();
  // Wait for the optimistic move to settle (server confirmation swaps
  // the row's DOM node once) before interacting further.
  await expect(pantryRow).not.toHaveClass(/is-pending/);

  // Real Tab traversal down to the pantry row's move button, then Tab
  // once more onto its delete button — what an actual keyboard user
  // does, so :focus-visible (and therefore the delete button's opacity)
  // behaves correctly.
  const pantryMoveTarget = pantryRow.locator('.item-row-move-target');
  await page.locator('#add-input').focus();
  const reachedPantryMoveTarget = await tabUntilFocused(page, pantryMoveTarget);
  expect(reachedPantryMoveTarget).toBe(true);

  await page.keyboard.press('Tab');
  const deleteBtn = pantryRow.locator('.delete-btn');
  await expect(deleteBtn).toBeFocused();
  await page.keyboard.press(' ');

  await expect(page.locator('.item-row', { hasText: name })).toHaveCount(0);
});
