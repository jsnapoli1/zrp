import { test, expect } from '@playwright/test';

/**
 * TC-INT-002: BOM Shortage Detection → Procurement Flow Integration Test
 * 
 * Tests that creating a work order with insufficient inventory:
 * 1. Detects BOM shortages correctly
 * 2. Allows generation of PO from shortages
 * 3. Receiving PO updates inventory
 * 4. Re-checking shows no shortages after receiving
 * 
 * This is a critical P0 integration test that validates the end-to-end
 * procurement workflow from shortage detection through PO receiving.
 */

test.describe('TC-INT-002: BOM Shortage → Procurement Flow', () => {
  
  // Login before each test
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.fill('input[type="text"], input[name="username"]', 'admin');
    await page.fill('input[type="password"], input[name="password"]', 'changeme');
    await page.click('button[type="submit"]');
    await page.waitForURL(/dashboard|home/i);
  });

  test('complete BOM shortage to procurement flow', async ({ page }) => {
    const timestamp = Date.now();
    const vendorId = `V-TEST-${timestamp}`;
    const assemblyIpn = `ASY-TEST-${timestamp}`;
    const resistorIpn = `RES-TEST-${timestamp}`;
    const capacitorIpn = `CAP-TEST-${timestamp}`;
    const woNumber = `WO-TEST-${timestamp}`;
    
    // ============================================================
    // SETUP: Create test data via API
    // ============================================================
    
    console.log('Setting up test data...');
    
    // Create vendor
    const vendorResponse = await page.request.post('/api/v1/vendors', {
      data: {
        vendor_id: vendorId,
        name: 'Test Vendor for BOM Integration',
        status: 'active',
        lead_time_days: 7
      }
    });
    
    if (!vendorResponse.ok()) {
      const errorBody = await vendorResponse.text();
      console.error(`Failed to create vendor. Status: ${vendorResponse.status()}, Body: ${errorBody}`);
    }
    expect(vendorResponse.ok()).toBeTruthy();
    console.log(`✓ Created vendor: ${vendorId}`);
    
    // Create part categories
    await page.request.post('/api/v1/categories', {
      data: {
        title: `Assembly-${timestamp}`,
        prefix: `asy-${timestamp}`,
        type: 'assembly'
      }
    });
    
    await page.request.post('/api/v1/categories', {
      data: {
        title: `Resistor-${timestamp}`,
        prefix: `res-${timestamp}`,
        type: 'resistor'
      }
    });
    
    await page.request.post('/api/v1/categories', {
      data: {
        title: `Capacitor-${timestamp}`,
        prefix: `cap-${timestamp}`,
        type: 'capacitor'
      }
    });
    console.log('✓ Created categories');
    
    // Create parts
    await page.request.post('/api/v1/parts', {
      data: {
        ipn: assemblyIpn,
        category: `asy-${timestamp}`,
        status: 'production',
        description: 'Test Assembly for BOM Integration'
      }
    });
    
    await page.request.post('/api/v1/parts', {
      data: {
        ipn: resistorIpn,
        category: `res-${timestamp}`,
        status: 'production',
        description: '10k Ohm Test Resistor'
      }
    });
    
    await page.request.post('/api/v1/parts', {
      data: {
        ipn: capacitorIpn,
        category: `cap-${timestamp}`,
        status: 'production',
        description: '100uF Test Capacitor'
      }
    });
    console.log(`✓ Created parts: ${assemblyIpn}, ${resistorIpn}, ${capacitorIpn}`);
    
    // Create BOM for assembly
    // Assembly requires: 10x resistors, 5x capacitors per unit
    await page.request.post('/api/v1/boms', {
      data: {
        parent_ipn: assemblyIpn,
        child_ipn: resistorIpn,
        quantity: 10.0,
        notes: 'Test BOM line - resistors'
      }
    });
    
    await page.request.post('/api/v1/boms', {
      data: {
        parent_ipn: assemblyIpn,
        child_ipn: capacitorIpn,
        quantity: 5.0,
        notes: 'Test BOM line - capacitors'
      }
    });
    console.log('✓ Created BOM (10x resistors, 5x capacitors per assembly)');
    
    // Create inventory with INSUFFICIENT stock
    // Need: 100 resistors (10 units * 10 per unit) - have only 5
    // Need: 50 capacitors (10 units * 5 per unit) - have only 2
    await page.request.post('/api/v1/inventory', {
      data: {
        ipn: resistorIpn,
        qty_on_hand: 5.0,
        qty_reserved: 0.0,
        reorder_point: 50.0
      }
    });
    
    await page.request.post('/api/v1/inventory', {
      data: {
        ipn: capacitorIpn,
        qty_on_hand: 2.0,
        qty_reserved: 0.0,
        reorder_point: 25.0
      }
    });
    
    await page.request.post('/api/v1/inventory', {
      data: {
        ipn: assemblyIpn,
        qty_on_hand: 0.0,
        qty_reserved: 0.0,
        reorder_point: 10.0
      }
    });
    console.log('✓ Created inventory with shortages (resistor: 5, capacitor: 2)');
    
    // ============================================================
    // STEP 1: Create Work Order for 10 units of assembly
    // ============================================================
    
    console.log('\n--- Step 1: Create Work Order ---');
    await page.goto('/work-orders');
    await page.waitForTimeout(1000);
    
    const newWoButton = page.locator('button:has-text("New Work Order"), button:has-text("Add Work Order"), button:has-text("Create Work Order")').first();
    await newWoButton.click();
    await page.waitForTimeout(500);
    
    // Fill work order form
    await page.fill('input[name="wo_number"], input[placeholder*="WO"]', woNumber);
    await page.fill('input[name="ipn"], input[placeholder*="IPN"]', assemblyIpn);
    await page.fill('input[name="quantity"], input[type="number"]', '10');
    
    // Set status to 'open' or 'in_progress'
    const statusSelect = page.locator('select[name="status"]');
    if (await statusSelect.isVisible()) {
      await statusSelect.selectOption('open');
    }
    
    // Submit work order
    await page.click('button[type="submit"]:has-text("Create"), button:has-text("Save")');
    await page.waitForTimeout(2000);
    
    console.log(`✓ Created work order: ${woNumber}`);
    
    // ============================================================
    // STEP 2: Check BOM Shortages
    // ============================================================
    
    console.log('\n--- Step 2: Check BOM Shortages ---');
    
    // Navigate to work order detail page
    await page.goto('/work-orders');
    await page.waitForTimeout(1000);
    
    // Click on the work order to view details
    const woLink = page.locator(`text="${woNumber}"`).first();
    await woLink.click();
    await page.waitForTimeout(1500);
    
    // Look for BOM check button or section
    const bomCheckButton = page.locator('button:has-text("Check BOM"), button:has-text("BOM Check"), button:has-text("Check Shortages")').first();
    
    // If button exists, click it; otherwise shortages might be shown automatically
    if (await bomCheckButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await bomCheckButton.click();
      await page.waitForTimeout(1500);
    }
    
    // Check via API to verify shortages are detected
    const bomCheckResponse = await page.request.get(`/api/v1/workorders/${woNumber}/bom-check`);
    expect(bomCheckResponse.ok()).toBeTruthy();
    
    const shortages = await bomCheckResponse.json();
    console.log('BOM Check Response:', JSON.stringify(shortages, null, 2));
    
    // Verify shortages detected
    expect(Array.isArray(shortages)).toBeTruthy();
    expect(shortages.length).toBe(2);
    
    // Find shortage for resistor and capacitor
    const resistorShortage = shortages.find((s: any) => 
      s.ipn === resistorIpn || s.child_ipn === resistorIpn
    );
    const capacitorShortage = shortages.find((s: any) => 
      s.ipn === capacitorIpn || s.child_ipn === capacitorIpn
    );
    
    expect(resistorShortage).toBeDefined();
    expect(capacitorShortage).toBeDefined();
    
    // Expected shortages:
    // Resistor: need 100 (10 units * 10), have 5, shortage = 95
    // Capacitor: need 50 (10 units * 5), have 2, shortage = 48
    const resistorShortageQty = resistorShortage.shortage || resistorShortage.shortage_qty;
    const capacitorShortageQty = capacitorShortage.shortage || capacitorShortage.shortage_qty;
    
    expect(resistorShortageQty).toBe(95);
    expect(capacitorShortageQty).toBe(48);
    
    console.log(`✓ Detected shortages: resistor=${resistorShortageQty}, capacitor=${capacitorShortageQty}`);
    
    // Verify shortages visible in UI
    await expect(page.locator(`text="${resistorIpn}"`)).toBeVisible({ timeout: 5000 });
    await expect(page.locator(`text="${capacitorIpn}"`)).toBeVisible({ timeout: 5000 });
    
    // ============================================================
    // STEP 3: Generate PO from Shortages
    // ============================================================
    
    console.log('\n--- Step 3: Generate PO from Shortages ---');
    
    // Look for "Generate PO" button
    const generatePoButton = page.locator('button:has-text("Generate PO"), button:has-text("Create PO"), button:has-text("Generate Purchase Order")').first();
    await expect(generatePoButton).toBeVisible({ timeout: 5000 });
    await generatePoButton.click();
    await page.waitForTimeout(1000);
    
    // Select vendor (might be a modal or form)
    const vendorSelect = page.locator('select[name="vendor_id"], [role="combobox"]').first();
    if (await vendorSelect.isVisible({ timeout: 3000 }).catch(() => false)) {
      await vendorSelect.click();
      await page.click(`text="${vendorId}"`);
    } else {
      // Might be an input field
      const vendorInput = page.locator('input[name="vendor_id"], input[placeholder*="vendor"]').first();
      if (await vendorInput.isVisible()) {
        await vendorInput.fill(vendorId);
      }
    }
    
    // Submit PO generation
    const submitPoButton = page.locator('button[type="submit"]:has-text("Generate"), button:has-text("Create PO"), button:has-text("Save")').first();
    await submitPoButton.click();
    await page.waitForTimeout(2000);
    
    console.log('✓ Generated PO from shortages');
    
    // Navigate to procurement to find the generated PO
    await page.goto('/procurement');
    await page.waitForTimeout(1500);
    
    // Get the list of POs via API to find our generated PO
    const posResponse = await page.request.get('/api/v1/pos');
    expect(posResponse.ok()).toBeTruthy();
    const posList = await posResponse.json();
    
    // Find PO for our vendor created recently
    const generatedPo = posList.find((po: any) => 
      po.vendor_id === vendorId && 
      (po.status === 'draft' || po.status === 'sent' || po.status === 'open')
    );
    
    expect(generatedPo).toBeDefined();
    const poId = generatedPo.po_number || generatedPo.id;
    console.log(`✓ Found generated PO: ${poId}`);
    
    // Verify PO has correct line items
    const poDetailResponse = await page.request.get(`/api/v1/pos/${poId}`);
    expect(poDetailResponse.ok()).toBeTruthy();
    const poDetail = await poDetailResponse.json();
    
    expect(poDetail.lines).toBeDefined();
    expect(poDetail.lines.length).toBe(2);
    
    // Verify line items match shortages
    const resistorLine = poDetail.lines.find((line: any) => line.ipn === resistorIpn);
    const capacitorLine = poDetail.lines.find((line: any) => line.ipn === capacitorIpn);
    
    expect(resistorLine).toBeDefined();
    expect(capacitorLine).toBeDefined();
    expect(resistorLine.quantity).toBe(95);
    expect(capacitorLine.quantity).toBe(48);
    
    console.log(`✓ Verified PO line items: resistor=95, capacitor=48`);
    
    // ============================================================
    // STEP 4: Receive the PO
    // ============================================================
    
    console.log('\n--- Step 4: Receive PO ---');
    
    // Navigate to PO detail page
    await page.goto(`/procurement`);
    await page.waitForTimeout(1000);
    
    // Click on the PO
    const poLink = page.locator(`text="${poId}"`).first();
    if (await poLink.isVisible({ timeout: 3000 }).catch(() => false)) {
      await poLink.click();
      await page.waitForTimeout(1500);
    } else {
      // Might need to navigate directly
      await page.goto(`/procurement/${poId}`);
      await page.waitForTimeout(1500);
    }
    
    // Click "Receive" button
    const receiveButton = page.locator('button:has-text("Receive"), button:has-text("Mark Received"), button:has-text("Receive PO")').first();
    if (await receiveButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await receiveButton.click();
      await page.waitForTimeout(1500);
      
      // Confirm if there's a confirmation dialog
      const confirmButton = page.locator('button:has-text("Confirm"), button:has-text("Yes")').first();
      if (await confirmButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await confirmButton.click();
        await page.waitForTimeout(1500);
      }
    } else {
      // Receive via API if button not found
      const receiveResponse = await page.request.post(`/api/v1/pos/${poId}/receive`);
      if (!receiveResponse.ok()) {
        console.warn('⚠️ PO receive API call failed, status might not be updated');
      }
    }
    
    console.log('✓ Received PO');
    
    // ============================================================
    // STEP 5: Verify Inventory Updated
    // ============================================================
    
    console.log('\n--- Step 5: Verify Inventory Updated ---');
    
    // Get updated inventory for resistor
    const resistorInvResponse = await page.request.get(`/api/v1/inventory/${resistorIpn}`);
    expect(resistorInvResponse.ok()).toBeTruthy();
    const resistorInv = await resistorInvResponse.json();
    
    // Get updated inventory for capacitor
    const capacitorInvResponse = await page.request.get(`/api/v1/inventory/${capacitorIpn}`);
    expect(capacitorInvResponse.ok()).toBeTruthy();
    const capacitorInv = await capacitorInvResponse.json();
    
    // Expected inventory after receiving:
    // Resistor: 5 (initial) + 95 (received) = 100
    // Capacitor: 2 (initial) + 48 (received) = 50
    console.log('Resistor inventory:', resistorInv);
    console.log('Capacitor inventory:', capacitorInv);
    
    expect(resistorInv.qty_on_hand).toBe(100);
    expect(capacitorInv.qty_on_hand).toBe(50);
    
    console.log('✓ Inventory updated correctly: resistor=100, capacitor=50');
    
    // ============================================================
    // STEP 6: Re-check BOM Shortages (Should Be Empty)
    // ============================================================
    
    console.log('\n--- Step 6: Re-check BOM Shortages ---');
    
    const bomRecheckResponse = await page.request.get(`/api/v1/workorders/${woNumber}/bom-check`);
    expect(bomRecheckResponse.ok()).toBeTruthy();
    
    const recheckShortages = await bomRecheckResponse.json();
    console.log('Re-check BOM Response:', JSON.stringify(recheckShortages, null, 2));
    
    // Should have no shortages now
    expect(Array.isArray(recheckShortages)).toBeTruthy();
    expect(recheckShortages.length).toBe(0);
    
    console.log('✓ No shortages remaining after PO receiving');
    
    // ============================================================
    // SUCCESS! Integration test complete
    // ============================================================
    
    console.log('\n✅ TC-INT-002 PASSED: BOM Shortage → Procurement flow working end-to-end');
  });
  
  test('handles BOM check when no shortages exist', async ({ page }) => {
    const timestamp = Date.now();
    const assemblyIpn = `ASY-NOSHR-${timestamp}`;
    const resistorIpn = `RES-NOSHR-${timestamp}`;
    const woNumber = `WO-NOSHR-${timestamp}`;
    
    console.log('Setting up test data with sufficient inventory...');
    
    // Create category
    await page.request.post('/api/v1/categories', {
      data: {
        title: `Assembly-${timestamp}`,
        prefix: `asy-${timestamp}`,
        type: 'assembly'
      }
    });
    
    await page.request.post('/api/v1/categories', {
      data: {
        title: `Resistor-${timestamp}`,
        prefix: `res-${timestamp}`,
        type: 'resistor'
      }
    });
    
    // Create parts
    await page.request.post('/api/v1/parts', {
      data: {
        ipn: assemblyIpn,
        category: `asy-${timestamp}`,
        status: 'production',
        description: 'Test Assembly - No Shortage'
      }
    });
    
    await page.request.post('/api/v1/parts', {
      data: {
        ipn: resistorIpn,
        category: `res-${timestamp}`,
        status: 'production',
        description: 'Test Resistor - No Shortage'
      }
    });
    
    // Create BOM
    await page.request.post('/api/v1/boms', {
      data: {
        parent_ipn: assemblyIpn,
        child_ipn: resistorIpn,
        quantity: 5.0
      }
    });
    
    // Create inventory with SUFFICIENT stock
    await page.request.post('/api/v1/inventory', {
      data: {
        ipn: resistorIpn,
        qty_on_hand: 100.0,  // More than enough
        qty_reserved: 0.0
      }
    });
    
    await page.request.post('/api/v1/inventory', {
      data: {
        ipn: assemblyIpn,
        qty_on_hand: 0.0,
        qty_reserved: 0.0
      }
    });
    
    // Create work order via API
    await page.request.post('/api/v1/workorders', {
      data: {
        wo_number: woNumber,
        ipn: assemblyIpn,
        quantity: 10,
        status: 'open'
      }
    });
    
    console.log(`✓ Created work order: ${woNumber} with sufficient inventory`);
    
    // Check BOM
    const bomCheckResponse = await page.request.get(`/api/v1/workorders/${woNumber}/bom-check`);
    expect(bomCheckResponse.ok()).toBeTruthy();
    
    const shortages = await bomCheckResponse.json();
    console.log('BOM Check Response:', JSON.stringify(shortages, null, 2));
    
    // Should have no shortages
    expect(Array.isArray(shortages)).toBeTruthy();
    expect(shortages.length).toBe(0);
    
    console.log('✅ BOM check correctly reports no shortages when inventory is sufficient');
  });
});
