// AC-14 — full cycle: login (real form) -> protected action (list items,
// already visible after login) -> logout (real button/link) -> a reload
// after logout redirects to /login.html, no unexpected error at any step.
import { test, expect } from '@playwright/test';
import { E2E_USERNAME_A, E2E_PASSWORD_A } from '../playwright.config.js';

// This spec exercises the login/logout flow itself, so it must start
// logged OUT — overriding the suite-wide storageState set up by
// global-setup.js (every other spec relies on that authenticated state).
test.use({ storageState: { cookies: [], origins: [] } });

test('login -> list items -> logout -> reload redirects to /login.html', async ({ page }) => {
  // AC-05: visiting the app unauthenticated redirects to the login screen.
  await page.goto('/');
  await page.waitForURL(/\/login\.html/);

  // AC-01: real login form, real credentials.
  await page.fill('#username-input', E2E_USERNAME_A);
  await page.fill('#password-input', E2E_PASSWORD_A);
  await page.click('#login-submit');

  // Successful login redirects to "/" (default next=/) and mounts the UI.
  await page.waitForURL('/');
  await expect(page.locator('#user-name')).toHaveText('Usuari A');

  // Protected action: the shopping list is visible and populated (GET
  // /api/v1/items succeeded under the new session).
  await expect(page.locator('.boxes')).toBeVisible();

  // AC-04: logout via the real button.
  await page.click('#logout-btn');
  await page.waitForURL(/\/login\.html/);

  // A reload with the session gone must land back on the login screen,
  // not silently show stale content or throw an unhandled error.
  await page.reload();
  await expect(page).toHaveURL(/\/login\.html/);
  await expect(page.locator('#login-form')).toBeVisible();
});
