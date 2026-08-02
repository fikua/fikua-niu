// global-setup.js — NIU-4: since cmd/niu now always wires
// auth.PasswordAuthenticator (no more StubAuthenticator), every E2E spec
// that navigates straight to "/" needs an authenticated session already
// in place, or main.js's AC-05 redirect sends it to /login.html instead.
//
// Rather than editing all ten pre-existing NIU-1 specs to perform a login
// step (which would be an unrelated behavioural change to already-shipped
// tests), this logs in once via the real HTTP API before the suite runs
// and persists the resulting cookies to a storageState file that every
// project/spec loads automatically (Playwright's standard pattern) — the
// specs' page.goto('/') calls are unmodified.
import { chromium } from '@playwright/test';
import { E2E_USERNAME_A, E2E_PASSWORD_A } from './playwright.config.js';

const STORAGE_STATE_PATH = './.auth-state.json';

export default async function globalSetup(config) {
  const { baseURL } = config.projects[0].use;

  const browser = await chromium.launch();
  const page = await browser.newPage({ baseURL });

  await page.goto(`${baseURL}/login.html`);
  await page.fill('#username-input', E2E_USERNAME_A);
  await page.fill('#password-input', E2E_PASSWORD_A);
  await page.click('#login-submit');
  // Successful login redirects to "/" (default next=/).
  await page.waitForURL(`${baseURL}/`);

  await page.context().storageState({ path: STORAGE_STATE_PATH });
  await browser.close();

  return STORAGE_STATE_PATH;
}

export { STORAGE_STATE_PATH };
