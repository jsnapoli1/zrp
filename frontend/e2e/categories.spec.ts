import { test, expect } from '@playwright/test';

// Login helper function
async function login(page: any) {
  await page.goto('/login');
  await page.fill('#username', 'admin');
  await page.fill('#password', 'changeme');
  await page.click('button[type="submit"]');
  await page.waitForURL(/dashboard/);
}

// Helper to login before each test
test.beforeEach(async ({ page }) => {
  await login(page);
});

test.describe('Category Management', () => {
  test('should create a new category', async ({ page }) => {
    // Navigate to parts page
    await page.goto('/parts');
    
    // Look for "New Category" or "Add Category" button
    const newCategoryButton = page.locator('button:has-text("New Category"), button:has-text("Add Category")').first();
    await newCategoryButton.click();
    
    // Fill in category details
    await page.fill('input[name="title"], input[placeholder*="title"]', 'Test Category');
    await page.fill('input[name="prefix"], input[placeholder*="prefix"]', 'tst');
    
    // Submit the form
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    
    // Wait for success (toast or redirect)
    await page.waitForTimeout(1000);
    
    // Verify category appears in the list
    await expect(page.locator('text="Test Category"')).toBeVisible({ timeout: 5000 });
  });

  test('should display created category with correct name', async ({ page }) => {
    // First create a category
    await page.goto('/parts');
    
    const newCategoryButton = page.locator('button:has-text("New Category"), button:has-text("Add Category")').first();
    await newCategoryButton.click();
    
    await page.fill('input[name="title"], input[placeholder*="title"]', 'Display Name Test');
    await page.fill('input[name="prefix"], input[placeholder*="prefix"]', 'dnt');
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    
    // Wait a bit for the category to be created
    await page.waitForTimeout(1500);
    
    // Reload or navigate to categories
    await page.goto('/parts');
    await page.waitForTimeout(500);
    
    // The category should show the human-readable title, not the filename
    await expect(page.locator('text="Display Name Test"')).toBeVisible({ timeout: 5000 });
    
    // Should NOT show the raw filename like "z-dnt"
    const rawFilename = page.locator('text="z-dnt"');
    await expect(rawFilename).not.toBeVisible();
  });
});
