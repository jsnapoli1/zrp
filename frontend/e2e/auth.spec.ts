import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test('should show login page by default', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1, h2')).toContainText(/login|sign in/i);
  });

  test('should reject invalid credentials', async ({ page }) => {
    await page.goto('/');
    
    // Fill in invalid credentials
    await page.fill('input[type="text"], input[name="username"]', 'wronguser');
    await page.fill('input[type="password"], input[name="password"]', 'wrongpass');
    await page.click('button[type="submit"]');
    
    // Should show error or stay on login page
    await expect(page.locator('body')).toContainText(/invalid|incorrect|failed/i);
  });

  test('should login with valid credentials', async ({ page }) => {
    await page.goto('/');
    
    // Fill in default admin credentials
    await page.fill('input[type="text"], input[name="username"]', 'admin');
    await page.fill('input[type="password"], input[name="password"]', 'changeme');
    await page.click('button[type="submit"]');
    
    // Should redirect to dashboard
    await expect(page).toHaveURL(/dashboard|home/i);
  });

  test('should logout successfully', async ({ page }) => {
    // Login first
    await page.goto('/');
    await page.fill('input[type="text"], input[name="username"]', 'admin');
    await page.fill('input[type="password"], input[name="password"]', 'changeme');
    await page.click('button[type="submit"]');
    
    // Wait for dashboard
    await expect(page).toHaveURL(/dashboard|home/i);
    
    // Click logout (look for logout button/link)
    await page.click('text=/logout/i');
    
    // Should redirect to login
    await expect(page).toHaveURL(/login|auth/i);
  });
});
