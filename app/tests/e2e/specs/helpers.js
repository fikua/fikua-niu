// Shared Playwright test helpers for the NIU-1 E2E suite (T-29). The DB
// is shared across the whole run (single webServer instance), so each
// test creates a uniquely-named item and cleans it up afterwards to
// avoid cross-test interference and EC-06 duplicate collisions.

let counter = 0;

export function uniqueName(prefix) {
  counter += 1;
  return `${prefix}-${Date.now()}-${counter}`;
}

export async function addItem(page, name) {
  await page.fill('#add-input', name);
  await page.click('#add-btn');
  await page.locator('.item-row', { hasText: name }).first().waitFor({ state: 'visible' });
}

export async function cleanupItem(page, name) {
  const row = page.locator('.item-row', { hasText: name }).first();
  if (await row.count() === 0) return;
  await row.hover();
  const deleteBtn = row.locator('.delete-btn');
  await deleteBtn.click({ force: true });
}
