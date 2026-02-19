import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for integration tests
 * Uses the existing ZRP server (localhost:9000) instead of starting a new one
 */
export default defineConfig({
  testDir: './e2e/integration',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'html',
  timeout: 60000,
  use: {
    // Connect to existing ZRP server
    baseURL: 'http://localhost:9000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    // Connect to Playwright server if available
    ...(process.env.PLAYWRIGHT_WS_ENDPOINT && {
      connectOptions: {
        wsEndpoint: process.env.PLAYWRIGHT_WS_ENDPOINT,
      },
    }),
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  
  // No webServer - use existing ZRP instance
  // No globalSetup - existing server should already be initialized
});
