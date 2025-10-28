import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for testing Go server with embedded web UI
 *
 * This config is specifically for testing the Go server (not Next.js dev server).
 * Start the Go server first, then run:
 *
 * BASE_URL=http://localhost:8082 npx playwright test go-server.spec.ts --config=playwright.config.goserver.ts
 */
export default defineConfig({
  testDir: './tests/e2e',
  testMatch: 'go-server.spec.ts',

  // Maximum time one test can run
  timeout: 30 * 1000,

  // Test execution settings
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,

  // Reporter configuration
  reporter: [['html'], ['list']],

  // Shared settings for all projects
  use: {
    // Base URL for tests - should point to Go server
    baseURL: process.env.BASE_URL || 'http://localhost:8082',

    // Capture screenshots on failure
    screenshot: 'only-on-failure',

    // Record video on first retry
    video: 'retain-on-failure',

    // Collect trace on failure
    trace: 'on-first-retry',
  },

  // Configure projects for major browsers
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // NO webServer - we test against a running Go server
});
