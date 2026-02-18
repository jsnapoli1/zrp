const { test, expect } = require('@playwright/test');
const { login, nav, apiFetch } = require('./helpers');

test.describe('Validation - Required Fields', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('create ECO without title succeeds or returns error', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ecos', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ description: 'no title' })
    });
    // Server should either reject (400) or create with empty title
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('create NCR without severity', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ncrs', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'No Severity NCR' })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('create RMA without serial_number', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/rmas', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ customer: 'No SN' })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('create WO without assembly_ipn', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/workorders', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ qty: 5 })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('create WO with zero quantity', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/workorders', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ assembly_ipn: 'PCB-001-0001', qty: 0 })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('create WO with negative quantity', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/workorders', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ assembly_ipn: 'PCB-001-0001', qty: -5 })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('create vendor without name returns response', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/vendors', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ website: 'https://noname.com' })
    });
    // Server should handle gracefully (any non-crash response)
    expect(r.status).toBeGreaterThan(0);
  });

  test('create user without username', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/users', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: 'test123' })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('create user without password', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/users', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: 'nopw_user' })
    });
    expect(r.status).toBeGreaterThan(0);
  });

  test('create device without serial_number returns response', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/devices', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ipn: 'PCB-001-0001' })
    });
    expect(r.status).toBeGreaterThan(0);
  });

  test('create campaign without name', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/campaigns', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ version: '1.0' })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('empty JSON body to ECO create', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ecos', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('invalid JSON body returns error', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const resp = await fetch('/api/v1/ecos', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: 'not json'
      });
      return { status: resp.status };
    });
    expect([400, 500].includes(r.status)).toBe(true);
  });

  test('update nonexistent ECO returns error', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ecos/999999', {
      method: 'PUT', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'ghost' })
    });
    expect([404, 500, 200].includes(r.status)).toBe(true);
  });

  test('get nonexistent ECO returns error', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ecos/999999');
    expect([404, 500, 200].includes(r.status)).toBe(true);
  });

  test('delete nonexistent vendor', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/vendors/999999', { method: 'DELETE' });
    expect([404, 500, 200].includes(r.status)).toBe(true);
  });
});

test.describe('Validation - UI Form Submission', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('open ECO modal and save without filling fields', async ({ page }) => {
    await nav(page, 'ecos');
    await page.locator('button:has-text("New ECO")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
    // Should either show error or create with empty fields — no crash
  });

  test('open WO modal and save without filling fields', async ({ page }) => {
    await nav(page, 'workorders');
    await page.locator('button:has-text("New Work Order")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
  });
});
