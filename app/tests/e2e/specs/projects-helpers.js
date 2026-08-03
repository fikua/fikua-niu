// Shared Playwright helpers for the NIU-5 "projectes" E2E suite. Same
// rationale as helpers.js: the DB is shared across the whole run, so
// each test creates a uniquely-named project and cleans it up
// afterwards.

let counter = 0;

export function uniqueProjectName(prefix) {
  counter += 1;
  return `${prefix}-${Date.now()}-${counter}`;
}

export async function addProject(page, name) {
  await page.goto('/projects.html');
  await page.fill('#add-project-name', name);
  await page.click('#add-project-btn');
  await page.locator('.project-row', { hasText: name }).first().waitFor({ state: 'visible' });
}

export async function cleanupProject(page, name) {
  const row = page.locator('.project-row', { hasText: name }).first();
  if (await row.count() === 0) return;
  await row.locator('.delete-btn').click({ force: true });
}
