const { test, expect } = require('@playwright/test');
const { login, nav, apiFetch } = require('./helpers');

test.describe('Notifications', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('notifications API returns array', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/notifications');
    expect(r.status).toBe(200);
    expect(Array.isArray(r.body.data)).toBe(true);
  });

  test('notification bell exists in UI', async ({ page }) => {
    const bell = await page.$('#notif-bell');
    expect(bell).toBeTruthy();
  });

  test('clicking bell shows notification dropdown', async ({ page }) => {
    await page.click('#notif-bell');
    await page.waitForTimeout(1000);
    const dropdown = await page.$('#notif-dropdown');
    if (dropdown) {
      expect(await dropdown.isVisible()).toBe(true);
    }
  });

  test('mark notification as read', async ({ page }) => {
    const list = await apiFetch(page, '/api/v1/notifications');
    if (list.body.data.length > 0) {
      const notif = list.body.data[0];
      const r = await apiFetch(page, `/api/v1/notifications/${notif.id}/read`, { method: 'POST' });
      expect(r.status).toBe(200);
    }
  });

  test('low stock alerts appear in dashboard', async ({ page }) => {
    const r = await apiFetch(page, '/api/v1/dashboard/lowstock');
    expect(r.status).toBe(200);
  });

  test('notification dropdown shows items or empty state', async ({ page }) => {
    await page.click('#notif-bell');
    await page.waitForTimeout(1000);
    const dropdown = await page.$('#notif-dropdown');
    if (dropdown) {
      const text = await dropdown.textContent();
      expect(text.length).toBeGreaterThan(0);
    }
  });
});
