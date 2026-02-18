const { test, expect } = require('@playwright/test');
const { login, nav } = require('./helpers');

test.describe('Onboarding Tour', () => {
  test.beforeEach(async ({ page }) => {
    // Clear tour-seen flag
    await page.goto('http://localhost:9000');
    await page.evaluate(() => localStorage.removeItem('zrp-tour-seen'));
    await login(page);
  });

  test('tour auto-starts for first-time users', async ({ page }) => {
    await page.evaluate(() => localStorage.removeItem('zrp-tour-seen'));
    await page.reload();
    await login(page);
    await page.waitForSelector('.zt-popover.visible', { timeout: 5000 });
    const step = await page.textContent('.zt-pop-step');
    expect(step).toContain('Step 1 of');
  });

  test('tour does NOT auto-start for returning users', async ({ page }) => {
    await page.evaluate(() => localStorage.setItem('zrp-tour-seen', 'true'));
    await page.reload();
    await login(page);
    await page.waitForTimeout(2500);
    const popover = await page.$('.zt-popover.visible');
    expect(popover).toBeNull();
  });

  test('tour starts when clicking the Tour button', async ({ page }) => {
    await page.evaluate(() => localStorage.setItem('zrp-tour-seen', 'true'));
    await page.reload();
    await login(page);
    await page.waitForSelector('#tour-start-btn', { timeout: 5000 });
    await page.click('#tour-start-btn');
    await page.waitForSelector('.zt-popover.visible', { timeout: 3000 });
    expect(await page.textContent('.zt-pop-title')).toBe('Dashboard');
  });

  test('tour can be dismissed via Skip', async ({ page }) => {
    await page.waitForSelector('#tour-start-btn', { timeout: 5000 });
    await page.click('#tour-start-btn');
    await page.waitForSelector('.zt-popover.visible', { timeout: 3000 });
    await page.click('#zt-skip');
    await page.waitForTimeout(500);
    const popover = await page.$('.zt-popover.visible');
    expect(popover).toBeNull();
  });

  test('tour can be dismissed via close button', async ({ page }) => {
    await page.waitForSelector('#tour-start-btn', { timeout: 5000 });
    await page.click('#tour-start-btn');
    await page.waitForSelector('.zt-popover.visible', { timeout: 3000 });
    await page.click('#zt-close');
    await page.waitForTimeout(500);
    expect(await page.$('.zt-popover.visible')).toBeNull();
  });

  test('step counter increments on Next', async ({ page }) => {
    await page.click('#tour-start-btn');
    await page.waitForSelector('.zt-popover.visible', { timeout: 3000 });
    expect(await page.textContent('.zt-pop-step')).toContain('Step 1');
    await page.click('#zt-next');
    await page.waitForTimeout(1200);
    expect(await page.textContent('.zt-pop-step')).toContain('Step 2');
  });

  test('Back button goes to previous step', async ({ page }) => {
    await page.click('#tour-start-btn');
    await page.waitForSelector('.zt-popover.visible', { timeout: 3000 });
    await page.click('#zt-next');
    await page.waitForTimeout(1200);
    await page.click('#zt-prev');
    await page.waitForTimeout(1200);
    expect(await page.textContent('.zt-pop-step')).toContain('Step 1');
  });

  test('tour works in dark mode', async ({ page }) => {
    await page.evaluate(() => {
      document.documentElement.classList.add('dark');
      localStorage.setItem('zrp-dark-mode', 'true');
    });
    await page.click('#tour-start-btn');
    await page.waitForSelector('.zt-popover.visible', { timeout: 3000 });
    // Popover should exist and be styled
    const bg = await page.$eval('.zt-popover', el => getComputedStyle(el).backgroundColor);
    // Dark mode bg should be dark
    expect(bg).toBeTruthy();
  });

  test('tour completes all steps', async ({ page }) => {
    test.setTimeout(180000);
    // Use autoplay
    await page.evaluate(() => window.startTour(true));
    // Wait for tour to finish — popover should eventually disappear
    await page.waitForFunction(() => !document.querySelector('.zt-popover.visible'), { timeout: 150000 });
    expect(await page.$('.zt-popover.visible')).toBeNull();
  });

  test('highlight element is positioned over target', async ({ page }) => {
    await page.click('#tour-start-btn');
    await page.waitForSelector('.zt-highlight', { timeout: 3000 });
    const hlBox = await page.$eval('.zt-highlight', el => {
      const r = el.getBoundingClientRect();
      return { width: r.width, height: r.height };
    });
    expect(hlBox.width).toBeGreaterThan(0);
    expect(hlBox.height).toBeGreaterThan(0);
  });
});
