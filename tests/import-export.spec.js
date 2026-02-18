const { test, expect } = require('@playwright/test');
const { login, apiFetch } = require('./helpers');

test.describe('Import/Export - Device CSV', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('export devices CSV', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const resp = await fetch('/api/v1/devices/export');
      return { status: resp.status, contentType: resp.headers.get('content-type'), text: await resp.text() };
    });
    expect(r.status).toBe(200);
    expect(r.contentType).toContain('csv');
    expect(r.text.length).toBeGreaterThan(0);
  });

  test('export devices CSV has header row', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const resp = await fetch('/api/v1/devices/export');
      return await resp.text();
    });
    // CSV should have headers
    expect(r.toLowerCase()).toContain('serial');
  });

  test('import devices CSV', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const csv = 'serial_number,ipn,firmware_version,customer\nSN-IMPORT-001,PCB-001-0001,1.0.0,Import Test\nSN-IMPORT-002,PCB-001-0001,1.0.0,Import Test 2';
      const formData = new FormData();
      formData.append('file', new Blob([csv], { type: 'text/csv' }), 'devices.csv');
      const resp = await fetch('/api/v1/devices/import', { method: 'POST', body: formData });
      return { status: resp.status, body: await resp.json() };
    });
    expect(r.status).toBe(200);
  });

  test('imported devices appear in list', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/devices');
    const imported = r.body.data.find(d => d.serial_number === 'SN-IMPORT-001');
    expect(imported).toBeTruthy();
  });
});

test.describe('Import/Export - Reports CSV', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('inventory valuation CSV export', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const resp = await fetch('/api/v1/reports/inventory-valuation?format=csv');
      return { status: resp.status, contentType: resp.headers.get('content-type'), text: await resp.text() };
    });
    expect(r.status).toBe(200);
    expect(r.contentType).toContain('csv');
  });

  test('open ECOs CSV export', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const resp = await fetch('/api/v1/reports/open-ecos?format=csv');
      return { status: resp.status, contentType: resp.headers.get('content-type') };
    });
    expect(r.status).toBe(200);
    expect(r.contentType).toContain('csv');
  });

  test('WO throughput CSV export', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const resp = await fetch('/api/v1/reports/wo-throughput?format=csv');
      return { status: resp.status, contentType: resp.headers.get('content-type') };
    });
    expect(r.status).toBe(200);
    expect(r.contentType).toContain('csv');
  });

  test('low stock CSV export', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const resp = await fetch('/api/v1/reports/low-stock?format=csv');
      return { status: resp.status, contentType: resp.headers.get('content-type') };
    });
    expect(r.status).toBe(200);
    expect(r.contentType).toContain('csv');
  });

  test('NCR summary CSV export', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const resp = await fetch('/api/v1/reports/ncr-summary?format=csv');
      return { status: resp.status, contentType: resp.headers.get('content-type') };
    });
    expect(r.status).toBe(200);
    expect(r.contentType).toContain('csv');
  });
});

test.describe('Import/Export - Reports JSON', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('inventory valuation JSON', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/reports/inventory-valuation');
    expect(r.status).toBe(200);
    expect(r.body.data).toBeTruthy();
  });

  test('open ECOs JSON', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/reports/open-ecos');
    expect(r.status).toBe(200);
  });

  test('WO throughput JSON', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/reports/wo-throughput');
    expect(r.status).toBe(200);
  });

  test('low stock JSON', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/reports/low-stock');
    expect(r.status).toBe(200);
  });

  test('NCR summary JSON', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/reports/ncr-summary');
    expect(r.status).toBe(200);
  });
});

test.describe('Import/Export - Bulk Operations', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('bulk devices import via API', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/devices/bulk', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ devices: [
        { serial_number: 'SN-BULK-001', ipn: 'PCB-001-0001', firmware_version: '1.0.0' },
        { serial_number: 'SN-BULK-002', ipn: 'PCB-001-0001', firmware_version: '1.0.0' }
      ]})
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });

  test('bulk inventory update', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/inventory/bulk', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ items: [] })
    });
    expect([200, 400].includes(r.status)).toBe(true);
  });
});
