import { test, expect } from '@playwright/test';

// Helper to login before each test
test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await page.fill('input[type="text"], input[name="username"]', 'admin');
  await page.fill('input[type="password"], input[name="password"]', 'changeme');
  await page.click('button[type="submit"]');
  await page.waitForURL(/dashboard|home/i);
});

test.describe('Parts Management', () => {
  test('should create a category and then add a part to it', async ({ page }) => {
    // Navigate to parts page
    await page.goto('/parts');
    
    // Create a new category first
    const newCategoryButton = page.locator('button:has-text("New Category"), button:has-text("Add Category")').first();
    await newCategoryButton.click();
    
    await page.fill('input[name="title"], input[placeholder*="title"]', 'Resistors');
    await page.fill('input[name="prefix"], input[placeholder*="prefix"]', 'res');
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    
    // Wait for category creation
    await page.waitForTimeout(1500);
    
    // Now create a part in this category
    const newPartButton = page.locator('button:has-text("New Part"), button:has-text("Add Part")').first();
    await newPartButton.click();
    
    // Fill in part details
    await page.fill('input[name="ipn"], input[placeholder*="IPN"]', 'RES-001');
    
    // Select category (might be a select dropdown or autocomplete)
    const categorySelect = page.locator('select[name="category"], [role="combobox"]').first();
    if (await categorySelect.isVisible()) {
      await categorySelect.click();
      await page.click('text="Resistors", text="z-res"');
    }
    
    // Fill other fields if available
    const descField = page.locator('input[name="description"], textarea[name="description"]');
    if (await descField.isVisible()) {
      await descField.fill('10k Ohm Resistor');
    }
    
    // Submit the form
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    
    // Wait for success
    await page.waitForTimeout(1500);
    
    // Verify part appears in the list
    await expect(page.locator('text="RES-001"')).toBeVisible({ timeout: 5000 });
  });

  test('should display parts list', async ({ page }) => {
    await page.goto('/parts');
    
    // Should show a table or list of parts
    await expect(page.locator('table, [role="table"]')).toBeVisible({ timeout: 5000 });
  });

  test('should search for a part', async ({ page }) => {
    // First, create a part to search for
    await page.goto('/parts');
    
    // Create category
    const newCategoryButton = page.locator('button:has-text("New Category"), button:has-text("Add Category")').first();
    if (await newCategoryButton.isVisible()) {
      await newCategoryButton.click();
      await page.fill('input[name="title"], input[placeholder*="title"]', 'Capacitors');
      await page.fill('input[name="prefix"], input[placeholder*="prefix"]', 'cap');
      await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
      await page.waitForTimeout(1500);
    }
    
    // Create part
    const newPartButton = page.locator('button:has-text("New Part"), button:has-text("Add Part")').first();
    if (await newPartButton.isVisible()) {
      await newPartButton.click();
      await page.fill('input[name="ipn"], input[placeholder*="IPN"]', 'CAP-SEARCH-001');
      
      const categorySelect = page.locator('select[name="category"], [role="combobox"]').first();
      if (await categorySelect.isVisible()) {
        await categorySelect.click();
        await page.click('text="Capacitors", text="z-cap"');
      }
      
      await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
      await page.waitForTimeout(1500);
    }
    
    // Now search for the part
    await page.goto('/parts');
    const searchInput = page.locator('input[type="search"], input[placeholder*="search"], input[name="search"]').first();
    await searchInput.fill('CAP-SEARCH');
    
    // Wait for search results
    await page.waitForTimeout(1000);
    
    // Should show the part
    await expect(page.locator('text="CAP-SEARCH-001"')).toBeVisible({ timeout: 5000 });
  });
});
