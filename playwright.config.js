const { defineConfig } = require('@playwright/test');
module.exports = defineConfig({
  testDir: './tests',
  workers: 4,
  use: {
    video: 'on',
    viewport: { width: 1440, height: 900 },
    baseURL: 'http://localhost:9000',
  },
  outputDir: 'test-results',
});
