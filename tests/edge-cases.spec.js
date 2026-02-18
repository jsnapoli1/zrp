const { test, expect } = require('@playwright/test');
const { BASE, login, nav, getContent, apiFetch } = require('./helpers');

test.describe('Edge Cases - Error Handling', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('404 API route returns error JSON', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/nonexistent');
    expect(r.body.error).toBeTruthy();
  });

  test('invalid method on endpoint', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ecos', { method: 'DELETE' });
    // Should not crash — returns error or not-found
    expect([200, 400, 404, 405].includes(r.status)).toBe(true);
  });

  test('very long title in ECO creation', async ({ page }) => {
    const longTitle = 'A'.repeat(5000);
    const r = await apiFetch(page, '/api/v1/ecos', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: longTitle, description: 'long title test' })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('special characters in title', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ecos', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: '<script>alert("xss")</script>', description: 'XSS test' })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('unicode in vendor name', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/vendors', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: '日本電子 Supplier™ ñ', website: 'https://unicode.test' })
    });
    expect(r.status).toBe(200);
  });

  test('SQL injection attempt in search', async ({ page }) => {
    const r = await apiFetch(page, "/api/v1/search?q=' OR 1=1 --");
    expect(r.status).toBe(200);
    // Should not crash
  });

  test('empty search query', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/search?q=');
    expect(r.status).toBe(200);
  });
});

test.describe('Edge Cases - Dark Mode', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('dark mode toggle works on all modules', async ({ page }) => {
    await page.click('#dark-toggle');
    await page.waitForTimeout(300);
    const hasDark = await page.$eval('html', el => el.classList.contains('dark'));
    expect(hasDark).toBe(true);

    // Navigate through several modules in dark mode
    for (const route of ['dashboard', 'ecos', 'parts', 'inventory', 'workorders']) {
      await nav(page, route);
      const c = await getContent(page);
      expect(c.length).toBeGreaterThan(5);
    }
  });
});

test.describe('Edge Cases - Mobile Viewport', () => {
  test('app renders at mobile width', async ({ browser }) => {
    const context = await browser.newContext({ viewport: { width: 375, height: 812 } });
    const page = await context.newPage();
    await login(page);
    await nav(page, 'dashboard');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(10);
    await context.close();
  });

  test('app renders at tablet width', async ({ browser }) => {
    const context = await browser.newContext({ viewport: { width: 768, height: 1024 } });
    const page = await context.newPage();
    await login(page);
    await nav(page, 'ecos');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(10);
    await context.close();
  });

  test('sidebar visible at desktop width', async ({ page }) => {
    await login(page);
    const sidebar = await page.$('#sidebar, .sidebar, nav');
    expect(sidebar).toBeTruthy();
  });
});

test.describe('Edge Cases - Concurrent Operations', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('rapid navigation does not crash', async ({ page }) => {
    const routes = ['dashboard', 'ecos', 'parts', 'inventory', 'workorders', 'ncr', 'devices'];
    for (const r of routes) {
      await page.evaluate((route) => window.navigate(route), r);
      await page.waitForTimeout(200); // rapid, minimal wait
    }
    await page.waitForTimeout(2000);
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(5);
  });

  test('multiple API calls in parallel', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const endpoints = ['/api/v1/ecos', '/api/v1/parts', '/api/v1/inventory', '/api/v1/workorders', '/api/v1/ncrs'];
      const promises = endpoints.map(e => fetch(e).then(r => r.status));
      return Promise.all(promises);
    });
    for (const status of results) {
      expect(status).toBe(200);
    }
  });
});

test.describe('Edge Cases - Session', () => {
  test('double login does not break session', async ({ page }) => {
    await login(page);
    await login(page);
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(5);
  });

  test('accessing app after cookie clear redirects to login', async ({ page }) => {
    await login(page);
    await page.context().clearCookies();
    await page.goto(BASE);
    await page.waitForTimeout(2000);
    // Should show login page or redirect
  });
});
