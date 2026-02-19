import { test, expect } from '@playwright/test';

/**
 * TC-INT-003: Purchase Order Receipt → Inventory Increase
 * 
 * Tests that receiving a purchase order automatically increases inventory quantities.
 * 
 * Workflow:
 * 1. Create vendor, part, and initial inventory
 * 2. Create a PO for additional quantity
 * 3. Record initial inventory qty_on_hand
 * 4. Mark PO as received
 * 5. Verify inventory qty_on_hand increased by PO quantity
 * 
 * This is a critical integration test that validates the procurement → inventory workflow.
 * If this test fails, it indicates that PO receiving does not properly update inventory,
 * which would be a production-blocking bug.
 */

test.describe('TC-INT-003: PO Receipt → Inventory Integration', () => {
  // Login before each test
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    
    // Wait for login form to be ready
    await page.waitForSelector('input[type="text"], input[name="username"]', { timeout: 10000 });
    
    await page.fill('input[type="text"], input[name="username"]', 'admin');
    await page.fill('input[type="password"], input[name="password"]', 'changeme');
    await page.click('button[type="submit"]');
    
    // Wait for redirect after login - increase timeout
    try {
      await page.waitForURL(/dashboard|home|parts|inventory/i, { timeout: 10000 });
    } catch (error) {
      // If login fails, take screenshot for debugging
      await page.screenshot({ path: `/tmp/login-failure-${Date.now()}.png` });
      const currentUrl = page.url();
      throw new Error(`Login failed. Current URL: ${currentUrl}. Check /tmp/login-failure-*.png`);
    }
    
    console.log('✓ Login successful, URL:', page.url());
  });

  test('should increase inventory quantity when PO is received', async ({ page }) => {
    const testPartIPN = `TEST-PO-${Date.now()}`;
    const testVendorCode = `V-${Date.now()}`;
    const initialQty = 50;
    const poQty = 100;
    const expectedFinalQty = initialQty + poQty;

    // ==========================================
    // STEP 1: Create Vendor
    // ==========================================
    console.log('Step 1: Creating vendor...');
    await page.goto('/vendors');
    
    const newVendorButton = page.locator('button:has-text("New Vendor"), button:has-text("Add Vendor")').first();
    await newVendorButton.click();
    
    await page.fill('input[name="vendor_code"], input[placeholder*="vendor"]', testVendorCode);
    await page.fill('input[name="name"], input[placeholder*="name"]', `Test Vendor ${testVendorCode}`);
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    await page.waitForTimeout(1000);
    
    // Verify vendor was created
    await expect(page.locator(`text="${testVendorCode}"`)).toBeVisible({ timeout: 5000 });
    console.log(`✓ Vendor ${testVendorCode} created`);

    // ==========================================
    // STEP 2: Create Category and Part
    // ==========================================
    console.log('Step 2: Creating part...');
    await page.goto('/parts');
    
    // Create category first
    const newCategoryButton = page.locator('button:has-text("New Category"), button:has-text("Add Category")').first();
    await newCategoryButton.click();
    
    await page.fill('input[name="title"], input[placeholder*="title"]', 'Integration Test Parts');
    await page.fill('input[name="prefix"], input[placeholder*="prefix"]', 'test');
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    await page.waitForTimeout(1000);
    
    // Create the test part
    const newPartButton = page.locator('button:has-text("New Part"), button:has-text("Add Part")').first();
    await newPartButton.click();
    
    await page.fill('input[name="ipn"], input[placeholder*="IPN"]', testPartIPN);
    
    // Select the category we just created
    const categorySelect = page.locator('select[name="category"], [role="combobox"]').first();
    if (await categorySelect.isVisible()) {
      await categorySelect.click();
      await page.click('text="Integration Test Parts", text="test"');
    }
    
    const descField = page.locator('input[name="description"], textarea[name="description"]');
    if (await descField.isVisible()) {
      await descField.fill('Test part for PO integration test');
    }
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    await page.waitForTimeout(1000);
    
    await expect(page.locator(`text="${testPartIPN}"`)).toBeVisible({ timeout: 5000 });
    console.log(`✓ Part ${testPartIPN} created`);

    // ==========================================
    // STEP 3: Create Initial Inventory Record
    // ==========================================
    console.log(`Step 3: Creating initial inventory (qty=${initialQty})...`);
    await page.goto('/inventory');
    
    const newInventoryButton = page.locator('button:has-text("New"), button:has-text("Add Inventory")').first();
    await newInventoryButton.click();
    
    // Select the part (might be dropdown or autocomplete)
    const partSelect = page.locator('select[name="ipn"], [role="combobox"]').first();
    if (await partSelect.isVisible()) {
      await partSelect.click();
      await page.click(`text="${testPartIPN}"`);
    } else {
      // Try autocomplete/search input
      const partInput = page.locator('input[name="ipn"], input[placeholder*="IPN"]').first();
      await partInput.fill(testPartIPN);
      await page.waitForTimeout(500);
      await page.click(`text="${testPartIPN}"`);
    }
    
    // Set initial quantity
    await page.fill('input[name="qty_on_hand"], input[placeholder*="quantity"]', initialQty.toString());
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    await page.waitForTimeout(1000);
    
    // Verify initial inventory
    await expect(page.locator(`text="${testPartIPN}"`)).toBeVisible({ timeout: 5000 });
    console.log(`✓ Initial inventory created: ${initialQty} units`);

    // ==========================================
    // STEP 4: Record Initial Quantity (via API or UI verification)
    // ==========================================
    console.log('Step 4: Verifying initial inventory quantity...');
    
    // Navigate to inventory page and find the part
    await page.goto('/inventory');
    
    // Wait for inventory table to load
    await page.waitForTimeout(1000);
    
    // Find the row with our test part and verify initial quantity
    const inventoryRow = page.locator(`tr:has-text("${testPartIPN}")`).first();
    await expect(inventoryRow).toBeVisible({ timeout: 5000 });
    
    // Verify the initial quantity is displayed
    const qtyCell = inventoryRow.locator('td').filter({ hasText: initialQty.toString() });
    await expect(qtyCell).toBeVisible({ timeout: 3000 });
    console.log(`✓ Verified initial quantity: ${initialQty}`);

    // ==========================================
    // STEP 5: Create Purchase Order
    // ==========================================
    console.log(`Step 5: Creating PO for ${poQty} units...`);
    await page.goto('/pos');
    
    const newPOButton = page.locator('button:has-text("New PO"), button:has-text("New Purchase Order")').first();
    await newPOButton.click();
    
    // Select vendor
    const vendorSelect = page.locator('select[name="vendor"], select[name="vendor_code"], [role="combobox"]').first();
    if (await vendorSelect.isVisible()) {
      await vendorSelect.click();
      await page.click(`text="${testVendorCode}"`);
    }
    
    // Add line item
    await page.waitForTimeout(500);
    
    // Look for "Add Line" or "Add Item" button
    const addLineButton = page.locator('button:has-text("Add Line"), button:has-text("Add Item")').first();
    if (await addLineButton.isVisible()) {
      await addLineButton.click();
      await page.waitForTimeout(500);
    }
    
    // Fill in line item details
    // Select part for line item
    const linePartSelect = page.locator('select[name*="ipn"], [role="combobox"]').last();
    if (await linePartSelect.isVisible()) {
      await linePartSelect.click();
      await page.click(`text="${testPartIPN}"`);
    } else {
      const linePartInput = page.locator('input[name*="ipn"]').last();
      await linePartInput.fill(testPartIPN);
      await page.waitForTimeout(500);
      await page.click(`text="${testPartIPN}"`);
    }
    
    // Fill quantity
    const qtyInput = page.locator('input[name*="quantity"], input[name*="qty"]').last();
    await qtyInput.fill(poQty.toString());
    
    // Fill unit price (if required)
    const priceInput = page.locator('input[name*="price"], input[name*="unit_price"]').last();
    if (await priceInput.isVisible()) {
      await priceInput.fill('10.00');
    }
    
    // Save the PO
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save PO")');
    await page.waitForTimeout(1500);
    
    console.log(`✓ PO created for ${poQty} units of ${testPartIPN}`);

    // ==========================================
    // STEP 6: Receive the PO
    // ==========================================
    console.log('Step 6: Receiving the PO...');
    
    // The PO should be visible in the list
    // Find the PO row and click it to open details
    await page.waitForTimeout(1000);
    
    // Look for the PO in the table (most recent one)
    const poRow = page.locator('tr').filter({ hasText: testVendorCode }).first();
    await expect(poRow).toBeVisible({ timeout: 5000 });
    
    // Click on the PO to open details
    await poRow.click();
    await page.waitForTimeout(1000);
    
    // Look for "Receive" or "Mark as Received" button
    const receiveButton = page.locator(
      'button:has-text("Receive"), button:has-text("Mark Received"), button:has-text("Received")'
    ).first();
    
    await expect(receiveButton).toBeVisible({ timeout: 5000 });
    await receiveButton.click();
    
    // Wait for the receive action to complete
    await page.waitForTimeout(2000);
    
    console.log('✓ PO marked as received');

    // ==========================================
    // STEP 7: Verify Inventory Increased
    // ==========================================
    console.log('Step 7: Verifying inventory increase...');
    
    await page.goto('/inventory');
    await page.waitForTimeout(1000);
    
    // Find the inventory row for our part
    const updatedInventoryRow = page.locator(`tr:has-text("${testPartIPN}")`).first();
    await expect(updatedInventoryRow).toBeVisible({ timeout: 5000 });
    
    // Check if the quantity increased
    // The cell should now show the expectedFinalQty
    const updatedQtyCell = updatedInventoryRow.locator('td').filter({ 
      hasText: expectedFinalQty.toString() 
    });
    
    // This is the critical assertion - if this fails, PO receiving doesn't update inventory
    await expect(updatedQtyCell).toBeVisible({ 
      timeout: 5000,
      // Custom error message for debugging
    });
    
    console.log(`✓ PASS: Inventory increased from ${initialQty} to ${expectedFinalQty}`);
    console.log(`✓ TC-INT-003 COMPLETE: PO receipt successfully increased inventory`);
  });

  test('should handle receiving PO when inventory record does not exist', async ({ page }) => {
    /**
     * Edge case: What happens if we receive a PO for a part that has no inventory record?
     * Expected: System should create inventory record with qty = PO qty
     */
    
    const testPartIPN = `TEST-PO-NOINV-${Date.now()}`;
    const testVendorCode = `V-NOINV-${Date.now()}`;
    const poQty = 75;

    // Create vendor
    await page.goto('/vendors');
    const newVendorButton = page.locator('button:has-text("New Vendor"), button:has-text("Add Vendor")').first();
    await newVendorButton.click();
    await page.fill('input[name="vendor_code"], input[placeholder*="vendor"]', testVendorCode);
    await page.fill('input[name="name"], input[placeholder*="name"]', `Test Vendor ${testVendorCode}`);
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    await page.waitForTimeout(1000);

    // Create part (but NO inventory record)
    await page.goto('/parts');
    const newPartButton = page.locator('button:has-text("New Part"), button:has-text("Add Part")').first();
    await newPartButton.click();
    await page.fill('input[name="ipn"], input[placeholder*="IPN"]', testPartIPN);
    
    const categorySelect = page.locator('select[name="category"], [role="combobox"]').first();
    if (await categorySelect.isVisible()) {
      await categorySelect.click();
      await page.locator('option, [role="option"]').first().click();
    }
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    await page.waitForTimeout(1000);

    // Create PO
    await page.goto('/pos');
    const newPOButton = page.locator('button:has-text("New PO"), button:has-text("New Purchase Order")').first();
    await newPOButton.click();
    
    const vendorSelect = page.locator('select[name="vendor"], select[name="vendor_code"]').first();
    if (await vendorSelect.isVisible()) {
      await vendorSelect.click();
      await page.click(`text="${testVendorCode}"`);
    }
    
    await page.waitForTimeout(500);
    
    const addLineButton = page.locator('button:has-text("Add Line"), button:has-text("Add Item")').first();
    if (await addLineButton.isVisible()) {
      await addLineButton.click();
    }
    
    const linePartSelect = page.locator('select[name*="ipn"]').last();
    if (await linePartSelect.isVisible()) {
      await linePartSelect.click();
      await page.click(`text="${testPartIPN}"`);
    }
    
    const qtyInput = page.locator('input[name*="quantity"], input[name*="qty"]').last();
    await qtyInput.fill(poQty.toString());
    
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    await page.waitForTimeout(1500);

    // Receive PO
    const poRow = page.locator('tr').filter({ hasText: testVendorCode }).first();
    await poRow.click();
    await page.waitForTimeout(1000);
    
    const receiveButton = page.locator('button:has-text("Receive")').first();
    await receiveButton.click();
    await page.waitForTimeout(2000);

    // Verify inventory was created
    await page.goto('/inventory');
    await page.waitForTimeout(1000);
    
    const inventoryRow = page.locator(`tr:has-text("${testPartIPN}")`).first();
    await expect(inventoryRow).toBeVisible({ timeout: 5000 });
    
    // Should show the PO quantity
    const qtyCell = inventoryRow.locator('td').filter({ hasText: poQty.toString() });
    await expect(qtyCell).toBeVisible({ timeout: 3000 });
    
    console.log(`✓ PASS: Inventory record auto-created when receiving PO (qty=${poQty})`);
  });
});
