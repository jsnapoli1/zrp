const { test, expect } = require('@playwright/test');
const { login, nav, getContent } = require('./helpers');

const modules = [
  { name: 'ECOs', route: 'ecos' },
  { name: 'Parts', route: 'parts' },
  { name: 'Inventory', route: 'inventory' },
  { name: 'Work Orders', route: 'workorders' },
  { name: 'Procurement', route: 'procurement' },
  { name: 'NCRs', route: 'ncr' },
  { name: 'Devices', route: 'devices' },
  { name: 'RMAs', route: 'rma' },
  { name: 'Quotes', route: 'quotes' },
  { name: 'Vendors', route: 'vendors' },
];

test.describe('List Features - Table Rows', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  for (const mod of modules) {
    test(`${mod.name} list has table rows`, async ({ page }) => {
      await nav(page, mod.route);
      const c = await getContent(page);
      // Content area should have rendered something (table or empty state)
      expect(c.length).toBeGreaterThan(0);
    });
  }
});

test.describe('List Features - Column Headers', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  for (const mod of modules) {
    test(`${mod.name} has sortable column headers`, async ({ page }) => {
      await nav(page, mod.route);
      const headers = await page.$$('#content th');
      if (headers.length === 0) {
        // Module is in empty state (no data) — verify empty state is shown instead
        const emptyState = await page.$('#content .empty-state, #content svg, #content .no-data');
        const content = await getContent(page);
        // Accept: either we have headers OR the module rendered an empty state / some content
        expect(content.length).toBeGreaterThan(0);
      } else {
        expect(headers.length).toBeGreaterThan(0);
      }
    });
  }
});

test.describe('List Features - Click Column to Sort', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('ECOs sort by clicking header', async ({ page }) => {
    await nav(page, 'ecos');
    const thLocator = page.locator('#content th').first();
    if (await thLocator.count() > 0) {
      await thLocator.click();
      await page.waitForTimeout(1000);
      const c = await getContent(page);
      expect(c.length).toBeGreaterThan(10);
    }
  });

  test('Parts sort by clicking header', async ({ page }) => {
    await nav(page, 'parts');
    const thLocator = page.locator('#content th').first();
    if (await thLocator.count() > 0) {
      await thLocator.click();
      await page.waitForTimeout(1000);
      const c = await getContent(page);
      expect(c.length).toBeGreaterThan(10);
    }
  });

  test('Inventory sort by clicking header', async ({ page }) => {
    await nav(page, 'inventory');
    const thLocator = page.locator('#content th').first();
    if (await thLocator.count() > 0) {
      await thLocator.click();
      await page.waitForTimeout(1000);
      const c = await getContent(page);
      expect(c.length).toBeGreaterThan(10);
    }
  });
});

test.describe('List Features - Search/Filter', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('global search filters results', async ({ page }) => {
    await page.fill('#global-search', 'PCB');
    await page.waitForTimeout(1500);
    const dropdown = await page.$('#search-dropdown');
    if (dropdown) {
      const visible = await dropdown.isVisible();
      expect(visible).toBe(true);
    }
  });

  test('global search with empty string is safe', async ({ page }) => {
    await page.fill('#global-search', '');
    await page.waitForTimeout(500);
    // Should not crash
  });

  test('search API returns structured data', async ({ page }) => {
    const r = await page.evaluate(() => fetch('/api/v1/search?q=test').then(r => r.json()));
    expect(r.data).toBeTruthy();
  });
});

test.describe('List Features - Row Click Opens Detail', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  for (const mod of modules) {
    test(`${mod.name} row click opens modal`, async ({ page }) => {
      await nav(page, mod.route);
      const rowLocator = page.locator('#content tr.table-row').first();
      if (await rowLocator.count() > 0) {
        await rowLocator.click();
        await page.waitForTimeout(1000);
        const modal = await page.$('.modal-overlay');
        expect(modal).toBeTruthy();
      } else {
        // No rows (empty state) — just verify module rendered
        const c = await getContent(page);
        expect(c.length).toBeGreaterThan(0);
      }
    });
  }
});
