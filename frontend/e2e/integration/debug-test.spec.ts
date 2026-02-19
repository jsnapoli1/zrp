import { test, expect } from '@playwright/test';

test.describe('Debug Tests', () => {
  test('can login and navigate', async ({ page }) => {
    // Try to connect to ZRP
    await page.goto('/');
    console.log('Loaded page:', page.url());
    
    // Take screenshot
    await page.screenshot({ path: '/tmp/zrp-login-page.png' });
    
    // Try to login
    await page.fill('input[type="text"], input[name="username"]', 'admin');
    await page.fill('input[type="password"], input[name="password"]', 'changeme');
    await page.click('button[type="submit"]');
    
    // Wait for redirect
    await page.waitForTimeout(3000);
    console.log('After login:', page.url());
    
    // Take screenshot after login
    await page.screenshot({ path: '/tmp/zrp-after-login.png' });
    
    // Try to navigate to vendors
    await page.goto('/vendors');
    await page.waitForTimeout(2000);
    console.log('Vendors page:', page.url());
    
    // Take screenshot
    await page.screenshot({ path: '/tmp/zrp-vendors-page.png' });
    
    // Check if page loaded
    const pageText = await page.textContent('body');
    console.log('Page contains vendor text:', pageText?.includes('vendor') || pageText?.includes('Vendor'));
  });
});
