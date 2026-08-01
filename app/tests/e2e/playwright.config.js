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

export default defineConfig({
  testDir: './specs',
  timeout: 30_000,
  fullyParallel: false, // shared single-server SQLite DB across the suite
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: `http://localhost:${port}`,
    trace: 'retain-on-failure',
  },
  webServer: {
    command: `go run ../../cmd/niu`,
    cwd: __dirname,
    env: {
      NIU_PORT: String(port),
      NIU_DB_PATH: dbPath,
      NIU_ENV: 'development',
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
