// Record a video walkthrough of the ZRP onboarding tour
// Usage: node tests/record-tour.js
const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');

const BASE = 'http://localhost:9000';
const DEMO_DIR = path.join(__dirname, '..', 'demo');

(async () => {
  fs.mkdirSync(DEMO_DIR, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
    recordVideo: { dir: DEMO_DIR, size: { width: 1920, height: 1080 } },
  });
  const page = await context.newPage();

  // Login
  await page.goto(BASE);
  await page.waitForSelector('#login-page:not(.hidden)', { timeout: 15000 });
  await page.fill('#login-username', 'admin');
  await page.fill('#login-password', 'changeme');
  await page.click('#login-form button[type="submit"]');
  await page.waitForSelector('#app:not(.hidden)', { timeout: 10000 });
  await page.waitForTimeout(1000);

  // Start tour in autoplay mode
  console.log('Starting tour autoplay...');
  await page.evaluate(() => window.startTour(true));

  // Poll for tour completion (23 steps × 3.5s ≈ 80s)
  for (let i = 0; i < 120; i++) {
    await page.waitForTimeout(2000);
    const state = await page.evaluate(() => window.getTourState());
    console.log(`  Tour state: step ${state.current + 1}/${state.total}, active=${state.active}`);
    if (!state.active) {
      console.log('Tour completed!');
      break;
    }
  }

  await page.waitForTimeout(2000);

  // Close to finalize video
  const video = page.video();
  await page.close();
  await context.close();

  if (video) {
    const videoPath = await video.path();
    const dest = path.join(DEMO_DIR, 'tour-video.webm');
    if (fs.existsSync(videoPath)) {
      fs.copyFileSync(videoPath, dest);
      console.log(`Video saved to: ${dest}`);
    }
  }

  await browser.close();
  console.log('Done!');
})().catch(err => {
  console.error('Recording failed:', err);
  process.exit(1);
});
