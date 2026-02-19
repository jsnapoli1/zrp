import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.fill('input[type="text"], input[name="username"]', 'admin');
    await page.fill('input[type="password"], input[name="password"]', 'changeme');
    await page.click('button[type="submit"]');
  });

  test('should load dashboard without errors', async ({ page }) => {
    // Should redirect to dashboard after login
    await expect(page).toHaveURL(/dashboard|home/i);
    
    // Check for common dashboard elements
    const dashboardText = page.locator('h1, h2, [role="heading"]');
    await expect(dashboardText).toBeVisible({ timeout: 5000 });
    
    // Should not have any console errors (check via page.on('pageerror'))
    const errors: Error[] = [];
    page.on('pageerror', (err) => errors.push(err));
    
    // Wait a bit for any async errors
    await page.waitForTimeout(2000);
    
    // No errors should have been logged
    expect(errors.length).toBe(0);
  });

  test('should display dashboard metrics', async ({ page }) => {
    await expect(page).toHaveURL(/dashboard|home/i);
    
    // Look for common metric cards (parts count, etc.)
    // This is a basic check - adjust based on actual dashboard structure
    const metricsContainer = page.locator('[data-testid="metrics"], .dashboard, [role="main"]');
    await expect(metricsContainer).toBeVisible({ timeout: 5000 });
  });
});
