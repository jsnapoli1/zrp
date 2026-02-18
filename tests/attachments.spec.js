const { test, expect } = require('@playwright/test');
const { login, apiFetch } = require('./helpers');
const path = require('path');
const fs = require('fs');

test.describe('Attachments', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('upload attachment to ECO', async ({ page }) => {
    // First get an ECO ID
    const ecos = await apiFetch(page, '/api/v1/ecos');
    const ecoId = ecos.body.data[0]?.id;
    if (!ecoId) { test.skip(true, 'No ECOs exist'); return; }

    const r = await page.evaluate(async (id) => {
      const formData = new FormData();
      formData.append('module', 'eco');
      formData.append('record_id', String(id));
      formData.append('file', new Blob(['test file content'], { type: 'text/plain' }), 'test.txt');
      const resp = await fetch('/api/v1/attachments', { method: 'POST', body: formData });
      return { status: resp.status, body: await resp.json() };
    }, ecoId);
    expect([200, 201].includes(r.status)).toBe(true);
  });

  test('upload attachment to NCR', async ({ page }) => {
    const ncrs = await apiFetch(page, '/api/v1/ncrs');
    const ncrId = ncrs.body.data[0]?.id;
    if (!ncrId) { test.skip(true, 'No NCRs exist'); return; }

    const r = await page.evaluate(async (id) => {
      const formData = new FormData();
      formData.append('module', 'ncr');
      formData.append('record_id', String(id));
      formData.append('file', new Blob(['ncr attachment'], { type: 'text/plain' }), 'ncr-doc.txt');
      const resp = await fetch('/api/v1/attachments', { method: 'POST', body: formData });
      return { status: resp.status, body: await resp.json() };
    }, ncrId);
    expect([200, 201].includes(r.status)).toBe(true);
  });

  test('upload attachment to RMA', async ({ page }) => {
    const rmas = await apiFetch(page, '/api/v1/rmas');
    const rmaId = rmas.body.data[0]?.id;
    if (!rmaId) { test.skip(true, 'No RMAs exist'); return; }

    const r = await page.evaluate(async (id) => {
      const formData = new FormData();
      formData.append('module', 'rma');
      formData.append('record_id', String(id));
      formData.append('file', new Blob(['rma photo'], { type: 'image/png' }), 'photo.png');
      const resp = await fetch('/api/v1/attachments', { method: 'POST', body: formData });
      return { status: resp.status, body: await resp.json() };
    }, rmaId);
    expect([200, 201].includes(r.status)).toBe(true);
  });

  test('list attachments for module and record', async ({ page }) => {
    const ecos = await apiFetch(page, '/api/v1/ecos');
    const ecoId = ecos.body.data[0]?.id;
    if (!ecoId) { test.skip(true, 'No ECOs'); return; }
    const r = await apiFetch(page, `/api/v1/attachments?module=eco&record_id=${ecoId}`);
    expect([200, 201].includes(r.status)).toBe(true);
  });

  test('upload without module returns 400', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const formData = new FormData();
      formData.append('file', new Blob(['bad'], { type: 'text/plain' }), 'bad.txt');
      const resp = await fetch('/api/v1/attachments', { method: 'POST', body: formData });
      return { status: resp.status };
    });
    expect(r.status).toBe(400);
  });

  test('upload without file returns 400', async ({ page }) => {
    const r = await page.evaluate(async () => {
      const formData = new FormData();
      formData.append('module', 'eco');
      formData.append('record_id', '1');
      const resp = await fetch('/api/v1/attachments', { method: 'POST', body: formData });
      return { status: resp.status };
    });
    expect(r.status).toBe(400);
  });

  test('delete attachment', async ({ page }) => {
    // Upload then delete
    const ecos = await apiFetch(page, '/api/v1/ecos');
    const ecoId = ecos.body.data[0]?.id;
    if (!ecoId) { test.skip(true, 'No ECOs'); return; }

    const upload = await page.evaluate(async (id) => {
      const formData = new FormData();
      formData.append('module', 'eco');
      formData.append('record_id', String(id));
      formData.append('file', new Blob(['delete me'], { type: 'text/plain' }), 'delete-me.txt');
      const resp = await fetch('/api/v1/attachments', { method: 'POST', body: formData });
      return { status: resp.status, body: await resp.json() };
    }, ecoId);
    expect([200, 201].includes(upload.status)).toBe(true);

    if (upload.body.data?.id) {
      const del = await apiFetch(page, `/api/v1/attachments/${upload.body.data.id}`, { method: 'DELETE' });
      expect(del.status).toBe(200);
    }
  });

  test('upload large-ish content succeeds', async ({ page }) => {
    const ecos = await apiFetch(page, '/api/v1/ecos');
    const ecoId = ecos.body.data[0]?.id;
    if (!ecoId) { test.skip(true, 'No ECOs'); return; }

    const r = await page.evaluate(async (id) => {
      // 1MB of data
      const data = 'x'.repeat(1024 * 1024);
      const formData = new FormData();
      formData.append('module', 'eco');
      formData.append('record_id', String(id));
      formData.append('file', new Blob([data], { type: 'application/octet-stream' }), 'large.bin');
      const resp = await fetch('/api/v1/attachments', { method: 'POST', body: formData });
      return { status: resp.status };
    }, ecoId);
    expect([200, 201].includes(r.status)).toBe(true);
  });
});
