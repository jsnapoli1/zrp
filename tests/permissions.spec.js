const { test, expect } = require('@playwright/test');
const { BASE, login, nav, getContent, fillField, saveModal, apiFetch } = require('./helpers');

test.describe('Permissions - Readonly User', () => {
  // First ensure readonly user exists
  test.beforeEach(async ({ page }) => {
    await login(page, 'admin', 'changeme');
  });

  test('ensure readonly user exists', async ({ page }) => {
    // Create user if not exists
    const users = await apiFetch(page, '/api/v1/users');
    const exists = users.body.data.find(u => u.username === 'readonly_test');
    if (!exists) {
      await apiFetch(page, '/api/v1/users', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: 'readonly_test', display_name: 'Readonly Tester', password: 'readonly123', role: 'readonly' })
      });
    }
    const users2 = await apiFetch(page, '/api/v1/users');
    const user = users2.body.data.find(u => u.username === 'readonly_test');
    expect(user).toBeTruthy();
  });
});

test.describe('Permissions - Readonly API Enforcement', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, 'readonly_test', 'readonly123');
  });

  test('readonly can GET ecos', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ecos');
    expect(r.status).toBe(200);
  });

  test('readonly can GET parts', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/parts');
    expect(r.status).toBe(200);
  });

  test('readonly can GET inventory', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/inventory');
    expect(r.status).toBe(200);
  });

  test('readonly CANNOT POST ecos', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ecos', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'Should Fail', description: 'no' })
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT POST ncrs', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/ncrs', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'Should Fail', severity: 'minor' })
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT POST workorders', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/workorders', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ assembly_ipn: 'PCB-001-0001', qty: 1 })
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT POST rmas', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/rmas', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ serial_number: 'SN-FAIL', customer: 'No', reason: 'no' })
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT POST vendors', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/vendors', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'No Vendor' })
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT POST pos', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/pos', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT POST devices', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/devices', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ serial_number: 'SN-FAIL', ipn: 'PCB-001-0001' })
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT POST campaigns', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/campaigns', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'No Campaign', version: '1.0' })
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT POST quotes', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/quotes', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ customer: 'No Quote' })
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT DELETE vendors', async ({ page }) => {
    // Get a vendor ID first
    const list = await apiFetch(page, '/api/v1/vendors');
    if (list.body.data && list.body.data.length > 0) {
      const v = list.body.data[0];
      const r = await apiFetch(page, `/api/v1/vendors/${v.id}`, { method: 'DELETE' });
      expect(r.status).toBe(403);
    }
  });

  test('readonly CANNOT PUT ecos', async ({ page }) => {
    const list = await apiFetch(page, '/api/v1/ecos');
    if (list.body.data && list.body.data.length > 0) {
      const eco = list.body.data[0];
      const r = await apiFetch(page, `/api/v1/ecos/${eco.id}`, {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: 'Hacked' })
      });
      expect(r.status).toBe(403);
    }
  });

  test('readonly CANNOT create users', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/users', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: 'hack', password: 'hack123', role: 'admin' })
    });
    expect(r.status).toBe(403);
  });

  test('readonly CANNOT create API keys', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/apikeys', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'hack key' })
    });
    expect(r.status).toBe(403);
  });
});

test.describe('Permissions - Unauthenticated API Access', () => {
  test('unauthenticated GET /api/v1/ecos returns 401', async ({ page }) => {
    const resp = await page.request.get(BASE + '/api/v1/ecos');
    expect(resp.status()).toBe(401);
  });

  test('unauthenticated GET /api/v1/parts returns 401', async ({ page }) => {
    const resp = await page.request.get(BASE + '/api/v1/parts');
    expect(resp.status()).toBe(401);
  });

  test('unauthenticated POST /api/v1/ecos returns 401', async ({ page }) => {
    const resp = await page.request.post(BASE + '/api/v1/ecos', {
      data: { title: 'hack' }
    });
    expect(resp.status()).toBe(401);
  });

  test('unauthenticated GET /api/v1/users returns 401', async ({ page }) => {
    const resp = await page.request.get(BASE + '/api/v1/users');
    expect(resp.status()).toBe(401);
  });

  test('unauthenticated GET /api/v1/apikeys returns 401', async ({ page }) => {
    const resp = await page.request.get(BASE + '/api/v1/apikeys');
    expect(resp.status()).toBe(401);
  });
});
