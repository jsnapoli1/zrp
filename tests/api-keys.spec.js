const { test, expect } = require('@playwright/test');
const { BASE, login, nav, getContent, apiFetch } = require('./helpers');

test.describe('API Keys - Full Lifecycle', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('list API keys', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/apikeys');
    expect(r.status).toBe(200);
  });

  test('create and use API key', async ({ page }) => {
    // Create
    const r = await apiFetch(page, '/api/v1/apikeys', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'PW-Lifecycle-Key' })
    });
    expect([200, 201].includes(r.status)).toBe(true);
    const keyValue = r.body.key;
    const keyId = r.body.id;
    expect(keyValue).toBeTruthy();
    expect(keyValue.startsWith('zrp_')).toBe(true);

    // Use key to authenticate
    const authResp = await page.request.get(BASE + '/api/v1/dashboard', {
      headers: { 'Authorization': `Bearer ${keyValue}` }
    });
    expect(authResp.status()).toBe(200);

    // Disable key
    const disableResp = await apiFetch(page, `/api/v1/apikeys/${keyId}`, {
      method: 'PUT', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled: 0 })
    });
    expect(disableResp.status).toBe(200);

    // Verify disabled key returns 401
    const failResp = await page.request.get(BASE + '/api/v1/dashboard', {
      headers: { 'Authorization': `Bearer ${keyValue}` }
    });
    expect(failResp.status()).toBe(401);

    // Delete key
    const delResp = await apiFetch(page, `/api/v1/apikeys/${keyId}`, { method: 'DELETE' });
    expect(delResp.status).toBe(200);

    // Verify gone
    const listResp = await apiFetch(page, '/api/v1/apikeys');
    const keys = listResp.body.data || listResp.body;
    const found = (Array.isArray(keys) ? keys : []).find(k => k.name === 'PW-Lifecycle-Key');
    expect(found).toBeFalsy();
  });

  test('API key appears in UI after creation', async ({ page }) => {
    // Create
    await apiFetch(page, '/api/v1/apikeys', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'PW-UI-Key' })
    });
    await nav(page, 'apikeys');
    const c = await getContent(page);
    expect(c).toContain('PW-UI-Key');
  });

  test('invalid bearer token returns 401', async ({ page }) => {
    const resp = await page.request.get(BASE + '/api/v1/dashboard', {
      headers: { 'Authorization': 'Bearer zrp_invalidtoken123456' }
    });
    expect(resp.status()).toBe(401);
  });

  test('API key can access multiple endpoints', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/apikeys', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'PW-Multi-Key' })
    });
    const keyValue = r.body.key;
    if (!keyValue) { test.skip(true, 'No key returned'); return; }

    for (const endpoint of ['/api/v1/parts', '/api/v1/ecos', '/api/v1/inventory']) {
      const resp = await page.request.get(BASE + endpoint, {
        headers: { 'Authorization': `Bearer ${keyValue}` }
      });
      expect(resp.status()).toBe(200);
    }
  });
});
