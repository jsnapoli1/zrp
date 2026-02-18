const { test, expect } = require('@playwright/test');
const { login, nav, getContent, apiFetch } = require('./helpers');

test.describe('Calendar', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('calendar API returns events array', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/calendar');
    expect(r.status).toBe(200);
    expect(Array.isArray(r.body.data)).toBe(true);
  });

  test('calendar API accepts year/month params', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/calendar?year=2026&month=2');
    expect(r.status).toBe(200);
  });

  test('calendar API with different month', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/calendar?year=2026&month=3');
    expect(r.status).toBe(200);
  });

  test('calendar UI loads month view', async ({ page }) => {
    // Navigate via hash directly and wait for calendar grid to render
    await page.evaluate(() => window.navigate('calendar'));
    await page.waitForFunction(() => {
      const c = document.getElementById('content');
      return c && /Mon|Tue|Wed/.test(c.textContent);
    }, { timeout: 10000 });
    const c = await getContent(page);
    expect(c).toMatch(/Mon|Tue|Wed|Thu|Fri|Sun|Sat/);
  });

  test('calendar UI shows day numbers', async ({ page }) => {
    await page.evaluate(() => window.navigate('calendar'));
    await page.waitForFunction(() => {
      const c = document.getElementById('content');
      return c && /Mon|Tue|Wed/.test(c.textContent);
    }, { timeout: 10000 });
    const c = await getContent(page);
    expect(c).toMatch(/\b1\b/);
    expect(c).toMatch(/\b15\b/);
  });

  test('calendar UI has event categories', async ({ page }) => {
    await nav(page, 'calendar');
    const c = await getContent(page);
    expect(c).toContain('Work Orders');
  });

  test('calendar navigate to previous month', async ({ page }) => {
    await nav(page, 'calendar');
    const prevBtn = await page.$('#content button:has-text("←"), #content button:has-text("Prev"), #content [onclick*="prev"]');
    if (prevBtn) {
      await prevBtn.click();
      await page.waitForTimeout(1500);
      const c = await getContent(page);
      expect(c).toContain('January');
    }
  });

  test('calendar navigate to next month', async ({ page }) => {
    await nav(page, 'calendar');
    const nextBtn = await page.$('#content button:has-text("→"), #content button:has-text("Next"), #content [onclick*="next"]');
    if (nextBtn) {
      await nextBtn.click();
      await page.waitForTimeout(1500);
      const c = await getContent(page);
      expect(c).toContain('March');
    }
  });

  test('calendar events have type field', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/calendar?year=2026&month=2');
    if (r.body.data.length > 0) {
      const event = r.body.data[0];
      expect(event.type).toBeTruthy();
      expect(event.date).toBeTruthy();
      expect(event.title).toBeTruthy();
    }
  });
});
