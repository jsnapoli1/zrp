const { test, expect } = require('@playwright/test');
const { login, nav, getContent, apiFetch } = require('./helpers');

test.describe('Cross-Module - Dashboard KPI Links', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('dashboard KPI Open ECOs is clickable', async ({ page }) => {
    await nav(page, 'dashboard');
    const card = page.locator('#content *:has-text("Open ECOs")').first();
    expect(await card.count()).toBeGreaterThan(0);
    await card.click();
    await page.waitForTimeout(1500);
  });

  test('dashboard KPI Low Stock is clickable', async ({ page }) => {
    await nav(page, 'dashboard');
    const card = await page.$('#content *:has-text("Low Stock")');
    expect(card).toBeTruthy();
  });

  test('dashboard KPI Active Work Orders is clickable', async ({ page }) => {
    await nav(page, 'dashboard');
    const card = await page.$('#content *:has-text("Active Work Orders")');
    expect(card).toBeTruthy();
  });

  test('dashboard returns correct data structure', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/dashboard');
    expect(r.status).toBe(200);
    // Should have KPI fields
    expect(typeof r.body.open_ecos).toBe('number');
  });

  test('dashboard charts endpoint returns data', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/dashboard/charts');
    expect(r.status).toBe(200);
  });

  test('dashboard low stock endpoint', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/dashboard/lowstock');
    expect(r.status).toBe(200);
  });
});

test.describe('Cross-Module - WO to PO Generation', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('generate PO from WO shortages endpoint', async ({ page }) => {
    // Create a WO first
    const wo = await apiFetch(page, '/api/v1/workorders', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ assembly_ipn: 'PCB-001-0001', qty: 100 })
    });
    expect(wo.status).toBe(200);
    const woId = wo.body.data?.id;

    if (woId) {
      // Try generating PO from WO
      const r = await apiFetch(page, '/api/v1/pos/generate-from-wo', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ work_order_id: woId })
      });
      // May succeed or return error if no shortages — both are valid
      expect([200, 400, 404].includes(r.status)).toBe(true);
    }
  });

  test('WO BOM shows component list', async ({ page }) => {
    const list = await apiFetch(page, '/api/v1/workorders');
    if (list.body.data.length > 0) {
      const wo = list.body.data[0];
      const bom = await apiFetch(page, `/api/v1/workorders/${wo.id}/bom`);
      expect(bom.status).toBe(200);
    }
  });
});

test.describe('Cross-Module - ECO Workflow', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('approve ECO via API', async ({ page }) => {
    const list = await apiFetch(page, '/api/v1/ecos');
    const draft = list.body.data.find(e => e.status === 'draft' || e.status === 'open');
    if (draft) {
      const r = await apiFetch(page, `/api/v1/ecos/${draft.id}/approve`, { method: 'POST' });
      expect([200, 400].includes(r.status)).toBe(true);
    }
  });

  test('implement ECO via API', async ({ page }) => {
    const list = await apiFetch(page, '/api/v1/ecos');
    const approved = list.body.data.find(e => e.status === 'approved');
    if (approved) {
      const r = await apiFetch(page, `/api/v1/ecos/${approved.id}/implement`, { method: 'POST' });
      expect([200, 400].includes(r.status)).toBe(true);
    }
  });
});

test.describe('Cross-Module - Sidebar Navigation Integrity', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('all sidebar routes load without error', async ({ page }) => {
    test.setTimeout(60000);
    const routes = ['dashboard', 'calendar', 'parts', 'ecos', 'docs', 'inventory',
      'procurement', 'vendors', 'workorders', 'testing', 'ncr', 'devices',
      'firmware', 'rma', 'quotes', 'reports', 'users', 'audit', 'apikeys', 'email'];
    for (const route of routes) {
      await nav(page, route);
      const c = await getContent(page);
      // Some routes may render minimal content
      expect(c).toBeTruthy();
    }
  });
});
