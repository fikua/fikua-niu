// CF-18 — confetti fires exactly once when "A comprar" empties, and does
// not reappear on subsequent renders while it stays empty.
import { test, expect } from '@playwright/test';
import { uniqueName, addItem } from './helpers.js';

test('confetti fires exactly once when the shopping box empties, not again while still empty', async ({ page }) => {
  const name = uniqueName('Confetti');
  await page.goto('/');

  // Assert the emptying TRANSITION rather than assuming the shopping box
  // starts empty (shared DB across the E2E run): add exactly one item,
  // then remove it, and count confetti pieces spawned during that single
  // action only.
  await addItem(page, name);

  const row = page.locator('.item-row', { hasText: name });
  await row.hover();

  // Arm the MutationObserver BEFORE triggering the delete — confetti
  // pieces self-remove on animationend, so observing after the click
  // could miss pieces that already appeared and disappeared.
  const observePromise = page.evaluate(() => {
    return new Promise((resolve) => {
      const layer = document.getElementById('confetti-layer');
      let seen = 0;
      const observer = new MutationObserver((mutations) => {
        for (const m of mutations) {
          seen += m.addedNodes.length;
        }
      });
      observer.observe(layer, { childList: true });
      setTimeout(() => {
        observer.disconnect();
        resolve(seen);
      }, 800);
    });
  });

  await row.locator('.delete-btn').click({ force: true });

  const pieceCount = await observePromise;
  expect(pieceCount).toBeGreaterThan(0);
});

test('confetti does not fire again on a subsequent render while shopping stays empty', async ({ page }) => {
  const name = uniqueName('ConfettiNoRepeat');
  await page.goto('/');
  await addItem(page, name);

  const row = page.locator('.item-row', { hasText: name });
  await row.hover();
  await row.locator('.delete-btn').click({ force: true });

  // Let the first celebration's particles finish, then trigger another
  // render (a manual syncFromServer via window focus) while the shopping
  // box is still empty, and confirm no NEW particles appear (EC-13/AC-14
  // guard against repeat firing).
  await page.waitForTimeout(1300);

  const secondBurstCount = await page.evaluate(() => {
    return new Promise((resolve) => {
      const layer = document.getElementById('confetti-layer');
      let seen = 0;
      const observer = new MutationObserver((mutations) => {
        for (const m of mutations) seen += m.addedNodes.length;
      });
      observer.observe(layer, { childList: true });
      window.dispatchEvent(new Event('focus'));
      setTimeout(() => {
        observer.disconnect();
        resolve(seen);
      }, 800);
    });
  });

  expect(secondBurstCount).toBe(0);
});
