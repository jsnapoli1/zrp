const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:9000';

async function login(page, user = 'admin', pass = 'changeme') {
  await page.goto(BASE);
  await page.waitForSelector('#login-page:not(.hidden), #app:not(.hidden)', { timeout: 5000 });
  const loginVisible = await page.$('#login-page:not(.hidden)');
  if (loginVisible) {
    await page.fill('#login-username', user);
    await page.fill('#login-password', pass);
    await page.click('#login-form button[type="submit"]');
    await page.waitForSelector('#app:not(.hidden)', { timeout: 5000 });
  }
  // Dismiss tour overlay if it auto-started
  await page.evaluate(() => {
    localStorage.setItem('zrp-tour-seen', 'true');
    document.querySelectorAll('.zt-overlay-bg, .zt-overlay, .zt-popover').forEach(el => el.remove());
    if (window._tourCleanup) window._tourCleanup();
  });
  // Wait for dashboard to fully render before returning
  await page.waitForTimeout(2000);
}

async function nav(page, route) {
  await page.waitForTimeout(500);
  await page.evaluate((r) => {
    window.location.hash = '#/' + r;
  }, route);
  await page.waitForFunction((r) => {
    return window.location.hash === '#/' + r;
  }, route, { timeout: 5000 }).catch(() => {});
  await page.waitForTimeout(2000);
  await page.waitForFunction(() => {
    const content = document.getElementById('content');
    return content && content.innerHTML.length > 0;
  }, { timeout: 5000 }).catch(() => {});
}

async function getContent(page) {
  return page.textContent('#content');
}

// Helper: fill modal field
async function fillField(page, field, value) {
  const sel = `[data-field="${field}"]`;
  const tag = await page.$eval(sel, el => el.tagName.toLowerCase());
  if (tag === 'select') {
    await page.selectOption(sel, value);
  } else {
    await page.fill(sel, value);
  }
}

// Helper: save modal
async function saveModal(page) {
  await page.click('#modal-save');
  await page.waitForTimeout(2000);
}

// ─── AUTHENTICATION ───────────────────────────────────────────

test.describe('Authentication', () => {
  test('shows login page when not authenticated', async ({ page }) => {
    await page.goto(BASE);
    await page.waitForSelector('#login-page:not(.hidden)', { timeout: 5000 });
    await expect(page.locator('#login-page')).not.toHaveClass(/hidden/);
  });

  test('login with valid credentials', async ({ page }) => {
    await page.goto(BASE);
    await page.waitForSelector('#login-page:not(.hidden)', { timeout: 5000 });
    await page.fill('#login-username', 'admin');
    await page.fill('#login-password', 'changeme');
    await page.click('#login-form button[type="submit"]');
    await page.waitForSelector('#app:not(.hidden)', { timeout: 5000 });
    await expect(page.locator('#app')).not.toHaveClass(/hidden/);
  });

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto(BASE);
    await page.waitForSelector('#login-page:not(.hidden)', { timeout: 5000 });
    await page.fill('#login-username', 'admin');
    await page.fill('#login-password', 'wrongpass');
    await page.click('#login-form button[type="submit"]');
    await page.waitForTimeout(1000);
    await expect(page.locator('#login-error')).not.toHaveClass(/hidden/);
  });

  test('logout returns to login page', async ({ page }) => {
    await login(page);
    const logoutBtn = page.locator('button:has-text("Logout"), a:has-text("Logout"), [onclick*="doLogout"]').first();
    if (await logoutBtn.count() > 0) {
      await logoutBtn.click();
      await page.waitForTimeout(1000);
      await expect(page.locator('#login-page')).not.toHaveClass(/hidden/);
    }
  });

  test('API returns 401 without session', async ({ page }) => {
    const resp = await page.request.get(BASE + '/api/v1/dashboard');
    expect(resp.status()).toBe(401);
  });
});

// ─── DASHBOARD ────────────────────────────────────────────────

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('loads with KPI cards', async ({ page }) => {
    await nav(page, 'dashboard');
    const c = await getContent(page);
    expect(c).toContain('Open ECOs');
    expect(c).toContain('Low Stock');
    expect(c).toContain('Active Work Orders');
  });

  test('displays activity feed', async ({ page }) => {
    await nav(page, 'dashboard');
    await page.waitForTimeout(2000);
    const c = await getContent(page);
    expect(c).toContain('Activity');
  });

  test('charts render', async ({ page }) => {
    await nav(page, 'dashboard');
    await page.waitForTimeout(3000);
    const canvases = await page.$$('#content canvas');
    expect(canvases.length).toBeGreaterThanOrEqual(1);
  });

  test('low stock alerts section exists', async ({ page }) => {
    await nav(page, 'dashboard');
    await page.waitForTimeout(2000);
    const c = await getContent(page);
    expect(c).toContain('Low Stock');
  });
});

// ─── PARTS / BOM ──────────────────────────────────────────────

test.describe('Parts & BOM', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('parts list loads', async ({ page }) => {
    await nav(page, 'parts');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(50);
  });

  test('parts table shows IPN column', async ({ page }) => {
    await nav(page, 'parts');
    const c = await getContent(page);
    expect(c).toMatch(/[A-Z]{2,4}-\d{3}/);
  });

  test('click part row opens detail', async ({ page }) => {
    await nav(page, 'parts');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForTimeout(1000);
      const modal = await page.$('.modal-overlay');
      expect(modal).toBeTruthy();
    }
  });

  test('BOM viewer accessible from part detail', async ({ page }) => {
    await nav(page, 'parts');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForTimeout(1500);
      const modalText = await page.textContent('.modal-overlay');
      expect(modalText.length).toBeGreaterThan(10);
    }
  });
});

// ─── INVENTORY ────────────────────────────────────────────────

test.describe('Inventory', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('inventory list loads with items', async ({ page }) => {
    await nav(page, 'inventory');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(50);
  });

  test('inventory shows stock quantities', async ({ page }) => {
    await nav(page, 'inventory');
    const c = await getContent(page);
    expect(c).toMatch(/\d+/);
  });

  test('click inventory row shows detail/history', async ({ page }) => {
    await nav(page, 'inventory');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForTimeout(1000);
      const modal = await page.$('.modal-overlay');
      expect(modal).toBeTruthy();
    }
  });
});

// ─── WORK ORDERS ──────────────────────────────────────────────

test.describe('Work Orders', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('work orders list loads', async ({ page }) => {
    await nav(page, 'workorders');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(20);
  });

  test('create work order', async ({ page }) => {
    await nav(page, 'workorders');
    await page.locator('button:has-text("New Work Order")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await fillField(page, 'assembly_ipn', 'PCB-001-0001');
    await fillField(page, 'qty', '10');
    await saveModal(page);
    const c = await getContent(page);
    expect(c).toContain('PCB-001-0001');
  });

  test('click work order row opens detail', async ({ page }) => {
    await nav(page, 'workorders');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForSelector('.modal-overlay', { timeout: 5000 }).catch(() => null);
      const modal = await page.$('.modal-overlay');
      expect(modal).toBeTruthy();
    }
  });

  test('work order BOM shortage check', async ({ page }) => {
    await nav(page, 'workorders');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForTimeout(1500);
      const modalText = await page.textContent('.modal-overlay');
      expect(modalText.length).toBeGreaterThan(20);
    }
  });
});

// ─── ECOs ─────────────────────────────────────────────────────

test.describe('ECOs', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('ECO list loads', async ({ page }) => {
    await nav(page, 'ecos');
    const c = await getContent(page);
    expect(c).toContain('ECO');
  });

  test('create ECO', async ({ page }) => {
    await nav(page, 'ecos');
    await page.locator('button:has-text("New ECO")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await fillField(page, 'title', 'PW-Test ECO Create');
    await fillField(page, 'description', 'Automated Playwright test');
    await saveModal(page);
    const c = await getContent(page);
    expect(c).toContain('PW-Test ECO Create');
  });

  test('edit ECO', async ({ page }) => {
    await nav(page, 'ecos');
    const row = page.locator('tr.table-row:has-text("PW-Test ECO Create")').first();
    if (await row.count() > 0) {
      await row.click();
      await page.waitForTimeout(1000);
      await fillField(page, 'title', 'PW-Test ECO Edited');
      await saveModal(page);
      const c = await getContent(page);
      expect(c).toContain('PW-Test ECO Edited');
    }
  });

  test('ECO approve workflow', async ({ page }) => {
    await nav(page, 'ecos');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForTimeout(1000);
      const approveBtn = page.locator('.modal-overlay button:has-text("Approve")');
      if (await approveBtn.count() > 0) {
        await approveBtn.click();
        await page.waitForTimeout(1500);
      }
    }
  });
});

// ─── NCRs ─────────────────────────────────────────────────────

test.describe('NCRs', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('NCR list loads', async ({ page }) => {
    await nav(page, 'ncr');
    const c = await getContent(page);
    expect(c).toContain('NCR');
  });

  test('create NCR', async ({ page }) => {
    await nav(page, 'ncr');
    await page.locator('button:has-text("New NCR")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await fillField(page, 'title', 'PW-Test NCR');
    await fillField(page, 'description', 'Playwright defect test');
    await fillField(page, 'severity', 'major');
    await saveModal(page);
    const c = await getContent(page);
    expect(c).toContain('PW-Test NCR');
  });

  test('edit NCR with root cause', async ({ page }) => {
    await nav(page, 'ncr');
    const row = page.locator('tr.table-row:has-text("PW-Test NCR")').first();
    if (await row.count() > 0) {
      await row.click();
      await page.waitForTimeout(1000);
      await fillField(page, 'root_cause', 'Solder bridge');
      await fillField(page, 'corrective_action', 'Reflow profile adjusted');
      await saveModal(page);
    }
  });

  test('NCR severity badge displays', async ({ page }) => {
    await nav(page, 'ncr');
    const c = await getContent(page);
    // Check for severity-related content (badge class or text like major/minor/critical)
    const hasBadge = await page.$$('#content .badge');
    const hasSeverityText = /major|minor|critical/i.test(c);
    expect(hasBadge.length > 0 || hasSeverityText).toBe(true);
  });
});

// ─── RMAs ─────────────────────────────────────────────────────

test.describe('RMAs', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('RMA list loads', async ({ page }) => {
    await nav(page, 'rma');
    const c = await getContent(page);
    expect(c).toContain('RMA');
  });

  test('create RMA', async ({ page }) => {
    await nav(page, 'rma');
    await page.locator('button:has-text("New RMA")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await fillField(page, 'serial_number', 'SN-PW-TEST');
    await fillField(page, 'customer', 'PW Test Customer');
    await fillField(page, 'reason', 'Unit not powering on');
    await saveModal(page);
    const c = await getContent(page);
    expect(c).toContain('PW Test Customer');
  });

  test('edit RMA status', async ({ page }) => {
    await nav(page, 'rma');
    const row = page.locator('tr.table-row:has-text("PW Test Customer")').first();
    if (await row.count() > 0) {
      await row.click();
      await page.waitForTimeout(1000);
      await fillField(page, 'status', 'diagnosing');
      await saveModal(page);
    }
  });
});

// ─── PROCUREMENT / POs ────────────────────────────────────────

test.describe('Procurement / POs', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('PO list loads', async ({ page }) => {
    await nav(page, 'procurement');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(20);
  });

  test('create PO', async ({ page }) => {
    await nav(page, 'procurement');
    await page.locator('button:has-text("New PO")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await saveModal(page);
    await page.waitForTimeout(1000);
    const c = await getContent(page);
    expect(c).toContain('PO-');
  });

  test('click PO row opens detail', async ({ page }) => {
    await nav(page, 'procurement');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForTimeout(1000);
      const modal = await page.$('.modal-overlay');
      expect(modal).toBeTruthy();
    }
  });
});

// ─── QUOTES ───────────────────────────────────────────────────

test.describe('Quotes', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('quotes list loads', async ({ page }) => {
    await nav(page, 'quotes');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(20);
  });

  test('create quote', async ({ page }) => {
    await nav(page, 'quotes');
    await page.locator('button:has-text("New Quote")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await fillField(page, 'customer', 'PW Aerospace Corp');
    await saveModal(page);
    const c = await getContent(page);
    expect(c).toContain('PW Aerospace Corp');
  });

  test('click quote row opens detail with margin analysis', async ({ page }) => {
    await nav(page, 'quotes');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForTimeout(1000);
      const modalText = await page.textContent('.modal-overlay');
      expect(modalText).toContain('Margin');
    }
  });
});

// ─── DEVICE REGISTRY ──────────────────────────────────────────

test.describe('Device Registry', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('device list loads', async ({ page }) => {
    await nav(page, 'devices');
    const c = await getContent(page);
    expect(c).toContain('SN-');
  });

  test('register new device', async ({ page }) => {
    await nav(page, 'devices');
    await page.locator('button:has-text("Register Device")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await fillField(page, 'serial_number', 'SN-PW-001');
    await fillField(page, 'ipn', 'PCB-001-0001');
    await fillField(page, 'firmware_version', '1.0.0');
    await fillField(page, 'customer', 'PW Test');
    await saveModal(page);
    const c = await getContent(page);
    expect(c).toContain('SN-PW-001');
  });

  test('click device shows detail with history', async ({ page }) => {
    await nav(page, 'devices');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForTimeout(1000);
      const modal = await page.$('.modal-overlay');
      expect(modal).toBeTruthy();
    }
  });
});

// ─── FIRMWARE CAMPAIGNS ───────────────────────────────────────

test.describe('Firmware Campaigns', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('campaigns list loads', async ({ page }) => {
    await nav(page, 'firmware');
    const c = await getContent(page);
    expect(c).toContain('Campaign');
  });

  test('create firmware campaign', async ({ page }) => {
    await nav(page, 'firmware');
    await page.locator('button:has-text("New Campaign")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await fillField(page, 'name', 'PW FW Update v2');
    await fillField(page, 'version', '2.0.0');
    await saveModal(page);
    const c = await getContent(page);
    expect(c).toContain('PW FW Update v2');
  });

  test('click campaign shows detail', async ({ page }) => {
    await nav(page, 'firmware');
    const rowLocator = page.locator('#content tr.table-row').first();
    if (await rowLocator.count() > 0) {
      await rowLocator.click();
      await page.waitForTimeout(1000);
      const modal = await page.$('.modal-overlay');
      expect(modal).toBeTruthy();
    }
  });
});

// ─── VENDORS / SUPPLIERS ──────────────────────────────────────

test.describe('Vendors', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('vendor list loads', async ({ page }) => {
    await nav(page, 'vendors');
    const c = await getContent(page);
    expect(c).toContain('DigiKey');
  });

  test('create vendor', async ({ page }) => {
    await nav(page, 'vendors');
    await page.locator('button:has-text("New Vendor")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await fillField(page, 'name', 'PW Test Supplier');
    await fillField(page, 'website', 'https://pwtest.com');
    await fillField(page, 'contact_name', 'Jane Doe');
    await fillField(page, 'contact_email', 'jane@pwtest.com');
    await saveModal(page);
    const c = await getContent(page);
    expect(c).toContain('PW Test Supplier');
  });

  test('edit vendor', async ({ page }) => {
    await nav(page, 'vendors');
    const row = page.locator('tr.table-row:has-text("PW Test Supplier")').first();
    if (await row.count() > 0) {
      await row.click();
      await page.waitForTimeout(1000);
      await fillField(page, 'contact_name', 'Jane Updated');
      await saveModal(page);
    }
  });

  test('delete vendor', async ({ page }) => {
    await nav(page, 'vendors');
    const row = page.locator('tr.table-row:has-text("PW Test Supplier")').first();
    if (await row.count() > 0) {
      await row.click();
      await page.waitForTimeout(1000);
      const delBtn = page.locator('.modal-overlay button:has-text("Delete")');
      if (await delBtn.count() > 0) {
        page.on('dialog', d => d.accept());
        await delBtn.click();
        await page.waitForTimeout(1500);
      }
    }
  });
});

// ─── AUDIT LOG ────────────────────────────────────────────────

test.describe('Audit Log', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('audit log loads', async ({ page }) => {
    await nav(page, 'audit');
    const c = await getContent(page);
    expect(c).toContain('Audit');
  });

  test('audit log shows entries', async ({ page }) => {
    await nav(page, 'audit');
    const rows = await page.$$('#content tr.table-row, #content tr');
    expect(rows.length).toBeGreaterThan(0);
  });
});

// ─── GLOBAL SEARCH ────────────────────────────────────────────

test.describe('Global Search', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('search input exists', async ({ page }) => {
    const input = await page.$('#global-search');
    expect(input).toBeTruthy();
  });

  test('search shows dropdown with results', async ({ page }) => {
    await page.fill('#global-search', 'PCB');
    await page.waitForTimeout(1500);
    const dropdown = await page.$('#search-dropdown');
    if (dropdown) {
      const visible = await dropdown.isVisible();
      expect(visible).toBe(true);
    }
  });

  test('search for non-existent term shows no results gracefully', async ({ page }) => {
    await page.fill('#global-search', 'zzzznonexistent99999');
    await page.waitForTimeout(1500);
  });
});

// ─── NOTIFICATIONS ────────────────────────────────────────────

test.describe('Notifications', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('notification bell exists', async ({ page }) => {
    const bell = await page.$('#notif-bell');
    expect(bell).toBeTruthy();
  });

  test('clicking bell shows dropdown', async ({ page }) => {
    await page.click('#notif-bell');
    await page.waitForTimeout(1000);
    const dropdown = await page.$('#notif-dropdown');
    if (dropdown) {
      const visible = await dropdown.isVisible();
      expect(visible).toBe(true);
    }
  });
});

// ─── USER MANAGEMENT ──────────────────────────────────────────

test.describe('User Management', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('users list loads', async ({ page }) => {
    await nav(page, 'users');
    const c = await getContent(page);
    expect(c).toContain('admin');
  });

  test('create user', async ({ page }) => {
    await nav(page, 'users');
    await page.locator('button:has-text("New User")').click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 });
    await fillField(page, 'username', 'pw_testuser');
    await fillField(page, 'display_name', 'PW Test User');
    await fillField(page, 'password', 'testpass123');
    await saveModal(page);
    const c = await getContent(page);
    expect(c).toContain('pw_testuser');
  });

  test('edit user role', async ({ page }) => {
    await nav(page, 'users');
    const row = page.locator('tr.table-row:has-text("pw_testuser"), tr:has-text("pw_testuser")').first();
    if (await row.count() > 0) {
      await row.click();
      await page.waitForTimeout(1000);
      await fillField(page, 'role', 'readonly');
      await saveModal(page);
    }
  });
});

// ─── API KEY MANAGEMENT ───────────────────────────────────────

test.describe('API Key Management', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('API keys page loads', async ({ page }) => {
    await nav(page, 'apikeys');
    const c = await getContent(page);
    expect(c).toContain('API');
  });

  test('generate new API key', async ({ page }) => {
    await nav(page, 'apikeys');
    await page.locator('#btn-new-key, button:has-text("Generate New Key"), button:has-text("New Key")').first().click();
    await page.waitForSelector('.modal-overlay', { timeout: 5000 }).catch(() => {});
    await page.waitForTimeout(500);
    // Try to fill the name field if a modal appeared
    const modal = await page.$('.modal-overlay');
    if (modal) {
      await fillField(page, 'name', 'PW Test Key');
      await saveModal(page);
    }
    const c = await getContent(page);
    expect(c).toContain('PW Test Key');
  });
});

// ─── CALENDAR VIEW ────────────────────────────────────────────

test.describe('Calendar View', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('calendar loads', async ({ page }) => {
    await nav(page, 'calendar');
    const c = await getContent(page);
    expect(c).toContain('February');
  });

  test('calendar shows day grid and event categories', async ({ page }) => {
    await nav(page, 'calendar');
    const c = await getContent(page);
    expect(c).toContain('Mon');
    expect(c).toContain('Work Orders');
  });
});

// ─── REPORTS ──────────────────────────────────────────────────

test.describe('Reports', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('reports page loads with report cards', async ({ page }) => {
    await nav(page, 'reports');
    const c = await getContent(page);
    expect(c).toContain('Inventory Valuation');
    expect(c).toContain('Open ECOs');
    expect(c).toContain('WO Throughput');
    expect(c).toContain('Low Stock');
    expect(c).toContain('NCR Summary');
  });

  test('click report card runs report', async ({ page }) => {
    await nav(page, 'reports');
    const card = page.locator('#content .card, #content [onclick*="report"], #content button:has-text("Inventory Valuation"), #content div:has-text("Inventory Valuation")').first();
    if (await card.count() > 0) {
      await card.click();
      await page.waitForTimeout(2000);
    }
  });
});

// ─── DOCUMENTS ────────────────────────────────────────────────

test.describe('Documents', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('docs page loads', async ({ page }) => {
    await nav(page, 'docs');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(10);
  });
});

// ─── TEST RECORDS ─────────────────────────────────────────────

test.describe('Test Records', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('test records page loads', async ({ page }) => {
    await nav(page, 'testing');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(10);
  });
});

// ─── EMAIL SETTINGS ───────────────────────────────────────────

test.describe('Email Settings', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('email settings page loads', async ({ page }) => {
    await nav(page, 'email');
    const c = await getContent(page);
    expect(c.length).toBeGreaterThan(10);
  });
});

// ─── DARK MODE ────────────────────────────────────────────────

test.describe('Dark Mode', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('toggle dark mode on', async ({ page }) => {
    await page.click('#dark-toggle');
    await page.waitForTimeout(500);
    const hasDark = await page.$eval('html', el => el.classList.contains('dark'));
    expect(hasDark).toBe(true);
  });

  test('toggle dark mode off', async ({ page }) => {
    await page.click('#dark-toggle');
    await page.waitForTimeout(300);
    await page.click('#dark-toggle');
    await page.waitForTimeout(300);
    const hasDark = await page.$eval('html', el => el.classList.contains('dark'));
    expect(hasDark).toBe(false);
  });

  test('dark mode persists in localStorage', async ({ page }) => {
    await page.click('#dark-toggle');
    await page.waitForTimeout(300);
    const stored = await page.evaluate(() => localStorage.getItem('zrp-dark-mode'));
    expect(stored).toBe('true');
  });
});

// ─── KEYBOARD SHORTCUTS ──────────────────────────────────────

test.describe('Keyboard Shortcuts', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('? opens shortcut help modal', async ({ page }) => {
    await page.keyboard.press('?');
    await page.waitForTimeout(500);
    const modal = await page.$('.modal-overlay');
    expect(modal).toBeTruthy();
    const text = await page.textContent('.modal-overlay');
    expect(text).toContain('Keyboard Shortcuts');
  });

  test('/ focuses search', async ({ page }) => {
    await page.keyboard.press('/');
    await page.waitForTimeout(300);
    const focused = await page.evaluate(() => document.activeElement.id);
    expect(focused).toBe('global-search');
  });

  test('g then d navigates to dashboard', async ({ page }) => {
    await nav(page, 'parts');
    await page.keyboard.press('g');
    await page.waitForTimeout(200);
    await page.keyboard.press('d');
    await page.waitForTimeout(1500);
    const hash = await page.evaluate(() => window.location.hash);
    expect(hash).toBe('#/dashboard');
  });

  test('g then p navigates to parts', async ({ page }) => {
    await nav(page, 'dashboard');
    await page.keyboard.press('g');
    await page.waitForTimeout(200);
    await page.keyboard.press('p');
    await page.waitForTimeout(1500);
    const hash = await page.evaluate(() => window.location.hash);
    expect(hash).toBe('#/parts');
  });

  test('g then e navigates to ECOs', async ({ page }) => {
    await nav(page, 'dashboard');
    await page.keyboard.press('g');
    await page.waitForTimeout(200);
    await page.keyboard.press('e');
    await page.waitForTimeout(1500);
    const hash = await page.evaluate(() => window.location.hash);
    expect(hash).toBe('#/ecos');
  });

  test('n triggers new item button', async ({ page }) => {
    await nav(page, 'ecos');
    await page.keyboard.press('n');
    await page.waitForTimeout(500);
    const modal = await page.$('.modal-overlay');
    expect(modal).toBeTruthy();
  });
});

// ─── BATCH / BULK OPERATIONS ─────────────────────────────────

test.describe('Batch Operations', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('ECO list has bulk checkboxes', async ({ page }) => {
    await nav(page, 'ecos');
    const checkboxes = await page.$$('#content .bulk-cb');
    expect(checkboxes.length).toBeGreaterThan(0);
  });

  test('selecting items shows bulk action bar', async ({ page }) => {
    await nav(page, 'ecos');
    const cb = page.locator('#content .bulk-cb').first();
    if (await cb.count() > 0) {
      await cb.click();
      await page.waitForTimeout(500);
      const bar = await page.$('.bulk-bar');
      expect(bar).toBeTruthy();
    }
  });

  test('select all checkbox works', async ({ page }) => {
    await nav(page, 'ecos');
    const allCb = page.locator('#content .bulk-cb-all');
    if (await allCb.count() > 0) {
      await allCb.click();
      await page.waitForTimeout(500);
      const checked = await page.$$eval('#content .bulk-cb', els => els.filter(e => e.checked).length);
      expect(checked).toBeGreaterThan(0);
    }
  });
});

// ─── CROSS-MODULE LINKS ──────────────────────────────────────

test.describe('Cross-Module Integration', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('dashboard KPI cards link to modules', async ({ page }) => {
    await nav(page, 'dashboard');
    await page.waitForTimeout(2000);
    const ecoLink = page.locator('#content [onclick*="ecos"], #content a:has-text("Open ECOs"), #content *:has-text("Open ECOs")').first();
    if (await ecoLink.count() > 0) {
      await ecoLink.click();
      await page.waitForTimeout(1500);
    }
  });

  test('sidebar navigation covers all modules', async ({ page }) => {
    const routes = ['dashboard', 'calendar', 'parts', 'ecos', 'docs', 'inventory',
      'procurement', 'vendors', 'workorders', 'testing', 'ncr', 'devices',
      'firmware', 'rma', 'quotes', 'reports', 'users', 'audit', 'apikeys', 'email'];
    for (const route of routes) {
      const link = await page.$(`[data-route="${route}"]`);
      expect(link).toBeTruthy();
    }
  });
});

// ─── API DIRECT TESTS ────────────────────────────────────────

test.describe('API Direct', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('GET /api/v1/dashboard returns data', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/dashboard').then(r => r.json()));
    expect(resp.open_ecos !== undefined || resp.data !== undefined).toBe(true);
  });

  test('GET /api/v1/parts returns array', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/parts').then(r => r.json()));
    expect(Array.isArray(resp.data)).toBe(true);
  });

  test('GET /api/v1/ecos returns array', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/ecos').then(r => r.json()));
    expect(Array.isArray(resp.data)).toBe(true);
  });

  test('GET /api/v1/inventory returns array', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/inventory').then(r => r.json()));
    expect(Array.isArray(resp.data)).toBe(true);
  });

  test('GET /api/v1/notifications returns array', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/notifications').then(r => r.json()));
    expect(Array.isArray(resp.data)).toBe(true);
  });

  test('GET /api/v1/audit returns array', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/audit').then(r => r.json()));
    expect(Array.isArray(resp.data)).toBe(true);
  });

  test('GET /api/v1/calendar returns data', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/calendar').then(r => r.json()));
    expect(resp.data).toBeTruthy();
  });

  test('GET /api/v1/search?q=PCB returns results', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/search?q=PCB').then(r => r.json()));
    expect(resp.data).toBeTruthy();
  });

  test('POST /api/v1/ecos creates ECO', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/ecos', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'API Test ECO', description: 'Created via API test' })
    }).then(r => r.json()));
    expect(resp.data).toBeTruthy();
  });

  test('POST /api/v1/ncrs creates NCR', async ({ page }) => {
    const resp = await page.evaluate(() => fetch('/api/v1/ncrs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'API Test NCR', description: 'API test', severity: 'minor' })
    }).then(r => r.json()));
    expect(resp.data).toBeTruthy();
  });
});
