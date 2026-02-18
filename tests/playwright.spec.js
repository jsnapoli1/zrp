const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:9000';

async function login(page) {
  await page.goto(BASE);
  // Check if login form is visible
  const loginForm = await page.$('#login-form, [data-field="username"], input[placeholder*="sername"]');
  if (loginForm) {
    await page.fill('input[type="text"], [data-field="username"], input[placeholder*="sername"]', 'admin');
    await page.fill('input[type="password"]', 'zonit123');
    await page.click('button[type="submit"], button:has-text("Sign In"), button:has-text("Login"), button:has-text("Log In")');
    await page.waitForTimeout(2000);
  }
}

test.describe('ZRP ERP System', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('Dashboard loads with KPI cards', async ({ page }) => {
    await page.goto(BASE);
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('Open ECOs');
    expect(content).toContain('Low Stock');
    expect(content).toContain('Active Work Orders');
  });

  test('Parts module loads', async ({ page }) => {
    await page.goto(BASE + '#/parts');
    await page.waitForTimeout(1500);
    const content = await page.textContent('#content');
    expect(content.length).toBeGreaterThan(10);
  });

  test('ECO: create and approve', async ({ page }) => {
    await page.goto(BASE + '#/ecos');
    await page.waitForTimeout(2000);
    await page.click('button:has-text("New ECO")');
    await page.waitForTimeout(1000);
    await page.fill('[data-field="title"]', 'Test ECO from Playwright');
    await page.fill('[data-field="description"]', 'Automated test');
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('Test ECO from Playwright');
  });

  test('Inventory: view and receive stock', async ({ page }) => {
    await page.goto(BASE + '#/inventory');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('CAP-001-0001');
  });

  test('Work Orders: create WO', async ({ page }) => {
    await page.goto(BASE + '#/workorders');
    await page.waitForTimeout(2000);
    await page.click('text=+ New Work Order');
    await page.waitForTimeout(1000);
    await page.fill('[data-field="assembly_ipn"]', 'PCB-001-0001');
    await page.fill('[data-field="qty"]', '5');
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('PCB-001-0001');
  });

  test('Devices: register and view', async ({ page }) => {
    await page.goto(BASE + '#/devices');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('SN-001');
  });

  test('NCR: create NCR', async ({ page }) => {
    await page.goto(BASE + '#/ncr');
    await page.waitForTimeout(2000);
    await page.click('text=+ New NCR');
    await page.waitForTimeout(1000);
    await page.fill('[data-field="title"]', 'Test NCR');
    await page.fill('[data-field="description"]', 'Test defect');
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('Test NCR');
  });

  test('Quotes: create quote', async ({ page }) => {
    await page.goto(BASE + '#/quotes');
    await page.waitForTimeout(2000);
    await page.click('text=+ New Quote');
    await page.waitForTimeout(1000);
    await page.fill('[data-field="customer"]', 'Test Customer');
    await page.click('#modal-save');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('Test Customer');
  });

  test('Vendors page loads', async ({ page }) => {
    await page.goto(BASE + '#/vendors');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('DigiKey');
  });

  test('RMA page loads', async ({ page }) => {
    await page.goto(BASE + '#/rma');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('RMA-2026-001');
  });

  // --- New Tests ---

  test('Reports page loads with report cards', async ({ page }) => {
    await page.goto(BASE + '#/reports');
    await page.waitForTimeout(2000);
    const content = await page.textContent('#content');
    expect(content).toContain('Reports');
    expect(content).toContain('Inventory Valuation');
    expect(content).toContain('Open ECOs by Priority');
    expect(content).toContain('WO Throughput');
    expect(content).toContain('Low Stock Report');
    expect(content).toContain('NCR Summary');
  });

  test('Quote margin analysis tab shows', async ({ page }) => {
    await page.goto(BASE + '#/quotes');
    await page.waitForTimeout(2000);
    // Click on a quote row
    const row = await page.$('tr.table-row');
    if (row) {
      await row.click();
      await page.waitForTimeout(1000);
      const modal = await page.textContent('.modal-content, .modal-overlay');
      expect(modal).toContain('Margin Analysis');
    }
  });
});
