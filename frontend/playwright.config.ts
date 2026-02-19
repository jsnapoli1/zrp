import { defineConfig, devices } from '@playwright/test';

// Use /tmp for test data (absolute paths work better with shell commands)
const testDbPath = '/tmp/zrp-test/zrp-test.db';
const testPartsDir = '/tmp/zrp-test/parts';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'html',
  timeout: 60000,
  use: {
    baseURL: 'http://localhost:9001',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  
  globalSetup: './e2e/global-setup.ts',
  
  webServer: {
    command: `cd .. && go run . -db ${testDbPath} -pmDir ${testPartsDir} -port 9001`,
    url: 'http://localhost:9001',
    reuseExistingServer: !process.env.CI,
    timeout: 30000,
  },
});
