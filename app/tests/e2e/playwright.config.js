// Playwright config for T-29 (E2E: FLIP animation, reduced-motion,
// confetti one-shot, mobile viewport, optimistic rollback, keyboard nav,
// aria-live). Starts the REAL Go binary against a temp SQLite DB — no
// mocked backend, per requirements.md §6 ("large" tier of the pyramid).
import { defineConfig, devices } from '@playwright/test';
import path from 'node:path';
import os from 'node:os';
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const dbDir = fs.mkdtempSync(path.join(os.tmpdir(), 'niu-e2e-'));
const dbPath = path.join(dbDir, 'niu.db');
const port = 8098;

// NIU-4: config.Load() now fails fast without NIU_SESSION_SECRET +
// NIU_USER_A_HASH/NIU_USER_B_HASH (EC-12) — fixture-only values, never
// real credentials (S11/NFR-09). Non-secret identity (username, display
// name, avatar) is no longer read from the environment — it comes from
// the committed users.json (config.go LoadUsers), so E2E_USERNAME_A must
// match users.json's user_a.name ("ocanades"), not an arbitrary fixture
// name. login-cycle.spec.js authenticates as
// E2E_USERNAME_A/E2E_PASSWORD_A below.
export const E2E_USERNAME_A = 'ocanades';
export const E2E_PASSWORD_A = 'e2e-fixture-password-a';
const E2E_HASH_A = '$2a$12$HLMGKaY6RMuvie4Vi9kZRenYSfwthJvV0YLziWHr7RNfShHakPejS';
const E2E_HASH_B = '$2a$12$vWKHyLuv0lMsBoHSMxx4X.05K./.SOfGBYs3xerxqTXmnsoesXjk6';

export default defineConfig({
  testDir: './specs',
  timeout: 30_000,
  fullyParallel: false, // shared single-server SQLite DB across the suite
  workers: 1,
  reporter: [['list']],
  // NIU-4: log in once via global-setup.js and reuse the resulting
  // session/CSRF cookies for every spec (see global-setup.js for why —
  // main.js now redirects an unauthenticated "/" to /login.html).
  // login-cycle.spec.js (T-36) explicitly overrides storageState to start
  // logged OUT, since it exercises the login/logout flow itself.
  globalSetup: './global-setup.js',
  use: {
    baseURL: `http://localhost:${port}`,
    trace: 'retain-on-failure',
    storageState: './.auth-state.json',
  },
  webServer: {
    command: `go run ../../cmd/niu`,
    cwd: __dirname,
    env: {
      NIU_PORT: String(port),
      NIU_DB_PATH: dbPath,
      NIU_ENV: 'development',
      NIU_SESSION_SECRET: 'e2e-fixture-session-secret-32bytes!!',
      NIU_USER_A_HASH: E2E_HASH_A,
      NIU_USER_B_HASH: E2E_HASH_B,
      PATH: process.env.PATH,
    },
    url: `http://localhost:${port}/healthz`,
    reuseExistingServer: false,
    timeout: 30_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
