const { test, expect } = require('@playwright/test');
const { login, nav, getContent, apiFetch } = require('./helpers');

test.describe('Supplier Pricing', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('create price entry', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/prices', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ipn: 'PCB-001-0001', vendor_name: 'DigiKey', unit_price: 12.50, min_qty: 100, currency: 'USD' })
    });
    expect([200, 201].includes(r.status)).toBe(true);
  });

  test('create second price entry for same IPN different vendor', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/prices', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ipn: 'PCB-001-0001', vendor_name: 'Mouser', unit_price: 13.00, min_qty: 100, currency: 'USD' })
    });
    expect([200, 201].includes(r.status)).toBe(true);
  });

  test('list prices for IPN', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/prices/PCB-001-0001');
    expect(r.status).toBe(200);
  });

  test('price trend endpoint', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/prices/PCB-001-0001/trend');
    expect(r.status).toBe(200);
  });

  test('create price with different quantities', async ({ page }) => {
    for (const qty of [1, 10, 100, 1000]) {
      const r = await apiFetch(page, '/api/v1/prices', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ipn: 'PCB-001-0001', vendor_name: 'DigiKey', unit_price: 15.00 - qty * 0.005, min_qty: qty, currency: 'USD' })
      });
      expect([200, 201].includes(r.status)).toBe(true);
    }
  });

  test('delete price entry', async ({ page }) => {
    const list = await apiFetch(page, '/api/v1/prices/PCB-001-0001');
    const prices = list.body.data || list.body;
    if (Array.isArray(prices) && prices.length > 0) {
      const priceId = prices[0].id;
      const r = await apiFetch(page, `/api/v1/prices/${priceId}`, { method: 'DELETE' });
      expect(r.status).toBe(200);
    }
  });

  test('vendor list shows vendors', async ({ page }) => {
    await nav(page, 'vendors');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(20);
  });

  test('parts cost endpoint', async ({ page }) => {
    const parts = await apiFetch(page, '/api/v1/parts');
    if (parts.body.data && parts.body.data.length > 0) {
      const ipn = parts.body.data[0].ipn || parts.body.data[0].IPN;
      if (ipn) {
        const r = await apiFetch(page, `/api/v1/parts/${ipn}/cost`);
        expect([200, 404].includes(r.status)).toBe(true);
      }
    }
  });
});
