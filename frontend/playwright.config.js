import { defineConfig, devices } from '@playwright/test'

// These tests drive the real Compose stack (Go backend + live ZITADEL), not
// a mocked backend — see e2e/README.md for preconditions. They provision and
// tear down their own throwaway ZITADEL users via e2e/support/zitadel-admin.js,
// so they're safe to run against a shared local dev instance, but since they
// share that one instance, run serially to avoid cross-test interference.
export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: 'http://localhost:8090',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
