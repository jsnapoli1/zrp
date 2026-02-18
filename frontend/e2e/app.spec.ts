import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test('loads and shows dashboard heading', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  });

  test('shows sidebar navigation', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('link', { name: 'Parts' })).toBeVisible();
  });
});

test.describe('Navigation - Module List Pages', () => {
  const modules = [
    { url: '/parts', heading: 'Parts' },
    { url: '/ecos', heading: 'Engineering Change Orders' },
    { url: '/work-orders', heading: 'Work Orders' },
    { url: '/inventory', heading: 'Inventory' },
    { url: '/vendors', heading: 'Vendors' },
    { url: '/ncrs', heading: 'Non-Conformance Reports' },
    { url: '/rmas', heading: 'Return Merchandise Authorization' },
    { url: '/quotes', heading: 'Quotes' },
    { url: '/documents', heading: 'Documents' },
    { url: '/users', heading: 'User Management' },
  ];

  for (const mod of modules) {
    test(`navigates to ${mod.url} and shows heading`, async ({ page }) => {
      await page.goto(mod.url);
      await expect(page.getByRole('heading', { name: mod.heading })).toBeVisible({ timeout: 10000 });
    });
  }
});

test.describe('Parts CRUD', () => {
  test('shows parts table with IPN column', async ({ page }) => {
    await page.goto('/parts');
    await expect(page.getByRole('heading', { name: 'Parts' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('columnheader', { name: 'IPN' })).toBeVisible();
  });

  test('has search input', async ({ page }) => {
    await page.goto('/parts');
    await expect(page.getByPlaceholder(/search/i)).toBeVisible({ timeout: 10000 });
  });

  test('opens create dialog on Add Part click', async ({ page }) => {
    await page.goto('/parts');
    await page.getByRole('button', { name: 'Add Part' }).click();
    await expect(page.getByText('Add New Part')).toBeVisible();
  });
});

test.describe('ECOs', () => {
  test('shows ECOs list page', async ({ page }) => {
    await page.goto('/ecos');
    await expect(page.getByRole('heading', { name: 'Engineering Change Orders' })).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Work Orders', () => {
  test('shows work orders list', async ({ page }) => {
    await page.goto('/work-orders');
    await expect(page.getByRole('heading', { name: 'Work Orders' })).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Inventory', () => {
  test('shows inventory list', async ({ page }) => {
    await page.goto('/inventory');
    await expect(page.getByRole('heading', { name: 'Inventory' })).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Dark Mode Toggle', () => {
  test('toggles dark class on html element', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
    
    const html = page.locator('html');
    const hadDark = await html.evaluate(el => el.classList.contains('dark'));
    
    // Dark mode toggle is in sidebar footer, has Moon or Sun icon
    const toggleBtn = page.locator('button').filter({ has: page.locator('svg.lucide-moon, svg.lucide-sun') }).first();
    await toggleBtn.click();
    
    const hasDark = await html.evaluate(el => el.classList.contains('dark'));
    expect(hasDark).not.toBe(hadDark);
  });
});

test.describe('Global Search', () => {
  test('opens command dialog on search click', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
    
    // Click the search button - contains Search icon and/or text
    const searchBtn = page.locator('button').filter({ has: page.locator('svg.lucide-search') }).first();
    await searchBtn.click();
    
    // Command dialog should appear with search input
    await expect(page.getByPlaceholder(/command or search/i)).toBeVisible();
  });
});
