// CF-16 — FLIP animation runs (~250ms) and lands correctly in the
// destination box.
import { test, expect } from '@playwright/test';
import { uniqueName, addItem, cleanupItem } from './helpers.js';

test('moving an item plays a ~250ms FLIP and lands in the destination box', async ({ page }) => {
  const name = uniqueName('Flip');
  await page.goto('/');
  await addItem(page, name);

  const row = page.locator('.item-row', { hasText: name });
  await expect(row).toBeVisible();

  // Capture the actual Web Animations API call duration directly, rather
  // than wall-clock time around the click (which is dominated by
  // Playwright/network overhead, not the animation itself, and gave
  // false negatives when the row settled in <50ms of real time despite a
  // 250ms animation being scheduled).
  const animationDuration = await page.evaluate(async (itemName) => {
    const rows = Array.from(document.querySelectorAll('.item-row'));
    const target = rows.find((r) => r.textContent.includes(itemName));
    if (!target) return null;

    // playFlip() re-queries the row by data-id AFTER re-rendering, so the
    // element it calls .animate() on is a different (but data-id-equal)
    // node than `target` — match by data-id, not by reference identity.
    const itemId = target.dataset.id;
    let capturedDuration = null;
    const originalAnimate = Element.prototype.animate;
    Element.prototype.animate = function (keyframes, options) {
      if (this.classList && this.classList.contains('item-row') && this.dataset.id === itemId) {
        capturedDuration = typeof options === 'number' ? options : options.duration;
      }
      return originalAnimate.call(this, keyframes, options);
    };

    // The row itself is a non-interactive container (WCAG 4.1.2 fix,
    // T-33) — the click handler lives on the full-cover move button.
    target.querySelector('.item-row-move-target').click();
    await new Promise((resolve) => setTimeout(resolve, 400));
    Element.prototype.animate = originalAnimate;
    return capturedDuration;
  }, name);

  expect(animationDuration).toBe(250);

  const pantryRow = page.locator('#box-pantry .item-row', { hasText: name });
  await expect(pantryRow).toBeVisible({ timeout: 2000 });

  await cleanupItem(page, name);
});
