// CF-17 — prefers-reduced-motion disables the flight and shows a
// cross-fade instead.
import { test, expect } from '@playwright/test';
import { uniqueName, addItem, cleanupItem } from './helpers.js';

test('prefers-reduced-motion shows a cross-fade instead of a flying transform', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });

  const name = uniqueName('Reduced');
  await page.goto('/');
  await addItem(page, name);

  const row = page.locator('.item-row', { hasText: name });

  // Intercept the Web Animations API calls made on this row to confirm
  // no `transform` keyframe is used under reduced motion (§8.6.2).
  const usedTransform = await page.evaluate(async (itemName) => {
    const rows = Array.from(document.querySelectorAll('.item-row'));
    const target = rows.find((r) => r.textContent.includes(itemName));
    if (!target) return null;

    let sawTransform = false;
    const originalAnimate = Element.prototype.animate;
    Element.prototype.animate = function (keyframes, options) {
      const frames = Array.isArray(keyframes) ? keyframes : [keyframes];
      if (frames.some((f) => f && Object.prototype.hasOwnProperty.call(f, 'transform'))) {
        sawTransform = true;
      }
      return originalAnimate.call(this, keyframes, options);
    };

    // The row itself is a non-interactive container (WCAG 4.1.2 fix,
    // T-33) — the click handler lives on the full-cover move button.
    target.querySelector('.item-row-move-target').click();
    await new Promise((resolve) => setTimeout(resolve, 400));
    Element.prototype.animate = originalAnimate;
    return sawTransform;
  }, name);

  expect(usedTransform).toBe(false);

  await cleanupItem(page, name);
});
