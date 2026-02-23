# ZRP Test Recommendations & Implementation Roadmap

**Date**: February 19, 2026  
**Author**: Eva (AI Assistant)  
**Purpose**: Specific test recommendations with implementation outlines, priorities, and effort estimates

---

## Quick Reference

| Priority | Category | Test Count | Estimated Effort | Timeline |
|----------|----------|------------|------------------|----------|
| **P0** | Security Testing | 45 tests | 2 weeks | Week 1-2 |
| **P0** | Critical Handlers | 10 handlers | 2 weeks | Week 2-3 |
| **P0** | E2E Critical Journeys | 5 journeys | 1.5 weeks | Week 3-4 |
| **P0** | Integration Workflows | 6 workflows | 1 week | Week 4-5 |
| **P1** | Error Recovery | 20 tests | 1.5 weeks | Week 5-6 |
| **P1** | Load Testing | 12 tests | 1 week | Week 7 |
| **P1** | Handler Coverage | 40 handlers | 3 weeks | Week 8-10 |
| **P2** | Migration Testing | 10 tests | 1 week | Week 11 |
| **P2** | Visual Regression | 15 tests | 1 week | Week 12 |

**Total Estimated Effort**: 14 weeks (3.5 months) for complete coverage

---

## P0 - Critical Priority Tests

### 1. Security Test Suite (Priority: CRITICAL)

**File**: `security_test.go`  
**Estimated Effort**: 2 weeks  
**Priority Rationale**: 0% security testing coverage is unacceptable for production

#### Test Suite Outline

```go
package main

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"strings"
)

// SQL Injection Tests
func TestSQLInjection_Login(t *testing.T) {
	attacks := []string{
		"admin' OR '1'='1",
		"admin'--",
		"' OR 1=1--",
		"admin'; DROP TABLE users--",
	}
	for _, attack := range attacks {
		// Test login endpoint with SQL injection attempt
		// Verify: Returns error, no SQL error exposed, account not compromised
	}
}

func TestSQLInjection_SearchQueries(t *testing.T) {
	attacks := []string{
		"test' OR '1'='1",
		"test'; DELETE FROM parts--",
		"test' UNION SELECT * FROM users--",
	}
	// Test search endpoints with injection attempts
	// Verify: Queries are parameterized, no SQL injection possible
}

func TestSQLInjection_FilterParameters(t *testing.T) {
	// Test filter endpoints: /api/v1/parts?filter=...
	// Inject SQL in filter values
	// Verify: No SQL injection, safe parameterization
}

// XSS Tests
func TestXSS_PartName(t *testing.T) {
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"<iframe src='javascript:alert(1)'>",
		"javascript:alert(1)",
	}
	// Create parts with XSS payloads in name field
	// Verify: Content is escaped when retrieved
}

func TestXSS_PartDescription(t *testing.T) {
	// Test rich text fields for XSS
	// Verify: HTML sanitization applied
}

func TestXSS_URLParameters(t *testing.T) {
	// Test URL parameters for XSS reflection
	// GET /api/v1/parts?search=<script>alert(1)</script>
	// Verify: Parameters escaped in error messages
}

// Authentication Tests
func TestAuthBypass_DirectAPIAccess(t *testing.T) {
	// Attempt to access protected endpoints without session
	// Verify: 401 Unauthorized returned
	endpoints := []string{
		"/api/v1/parts",
		"/api/v1/boms",
		"/api/v1/users",
		"/api/v1/settings",
	}
	// Test each endpoint without authentication
}

func TestAuthBypass_SessionHijacking(t *testing.T) {
	// Create session for user A
	// Attempt to use session cookie in different browser/IP
	// Verify: Session invalidated or additional checks required
}

func TestAuthBypass_SessionFixation(t *testing.T) {
	// Create session before login
	// Login with session ID
	// Verify: New session ID generated after login
}

func TestBruteForce_LoginRateLimit(t *testing.T) {
	// Attempt 100 login attempts in 10 seconds
	// Verify: Rate limiting applied, account locked after N attempts
}

func TestPasswordStrength_WeakPasswords(t *testing.T) {
	weakPasswords := []string{
		"123456",
		"password",
		"admin",
		"test",
	}
	// Attempt to create user with weak passwords
	// Verify: Password strength requirements enforced
}

// Authorization Tests
func TestAuthZ_HorizontalAccess(t *testing.T) {
	// Create user A and user B
	// User A attempts to access user B's data
	// Verify: 403 Forbidden
	// Test on: /api/v1/users/{userB_id}, private data endpoints
}

func TestAuthZ_VerticalAccess(t *testing.T) {
	// Create regular user (no admin)
	// Attempt to access admin endpoints
	// Verify: 403 Forbidden
	// Test on: /api/v1/users, /api/v1/settings, /api/v1/audit
}

func TestAuthZ_PermissionEnforcement(t *testing.T) {
	// Create user with limited permissions (e.g., read-only)
	// Attempt write operations
	// Verify: 403 Forbidden
}

// API Security Tests
func TestAPIKey_Authentication(t *testing.T) {
	// Test API key authentication
	// Valid key: should work
	// Invalid key: should fail
	// No key: should fail
	// Expired key: should fail
}

func TestAPIKey_RateLimit(t *testing.T) {
	// Test rate limiting on API endpoints
	// Make 1000 requests in 1 minute
	// Verify: Rate limit enforced (e.g., 100 req/min)
}

func TestAPI_ContentTypeValidation(t *testing.T) {
	// Send requests with wrong Content-Type
	// POST with text/plain instead of application/json
	// Verify: 400 Bad Request
}

// File Security Tests
func TestFileUpload_PathTraversal(t *testing.T) {
	filenames := []string{
		"../../etc/passwd",
		"..\\..\\windows\\system32\\config\\sam",
		"test/../../../secret.txt",
	}
	// Attempt to upload files with malicious paths
	// Verify: Path sanitization prevents traversal
}

func TestFileUpload_TypeRestriction(t *testing.T) {
	// Upload executable files (.exe, .sh, .bat)
	// Verify: File type restrictions enforced
}

func TestFileUpload_SizeLimit(t *testing.T) {
	// Upload extremely large file (1GB+)
	// Verify: Size limit enforced (e.g., 100MB max)
}

func TestFileDownload_Authorization(t *testing.T) {
	// Create file for user A
	// User B attempts to download
	// Verify: 403 Forbidden
}

// CSRF Tests
func TestCSRF_TokenRequired(t *testing.T) {
	// Attempt state-changing operations without CSRF token
	// POST, PUT, DELETE requests
	// Verify: 403 Forbidden (if CSRF protection implemented)
}

func TestCSRF_TokenValidation(t *testing.T) {
	// Use invalid/expired CSRF token
	// Verify: Request rejected
}

// Data Exposure Tests
func TestDataExposure_ErrorMessages(t *testing.T) {
	// Trigger various errors
	// Verify: No sensitive data in error messages
	// No stack traces, no database schema info, no file paths
}

func TestDataExposure_Logs(t *testing.T) {
	// Perform operations with sensitive data (passwords, API keys)
	// Check logs
	// Verify: Sensitive data not logged
}
```

**Implementation Steps**:
1. Week 1: Implement SQL injection and XSS tests (20 tests)
2. Week 1-2: Implement authentication and authorization tests (15 tests)
3. Week 2: Implement file security and CSRF tests (10 tests)
4. Fix all discovered vulnerabilities as they're found

---

### 2. Critical Untested Handlers (Priority: CRITICAL)

**Estimated Effort**: 2 weeks (10 handlers × ~1 day each)

#### Handler 1: `handler_advanced_search_test.go`

**Priority**: P0 (Advanced search is a key feature)  
**Effort**: 1 day

```go
func TestHandleAdvancedSearch_BasicQuery(t *testing.T) {
	// Test basic search with single field
	// POST /api/v1/advanced-search
	// Body: {"filters": [{"field": "part_number", "operator": "contains", "value": "RES"}]}
	// Verify: Returns matching parts
}

func TestHandleAdvancedSearch_MultipleFilters(t *testing.T) {
	// Test AND logic with multiple filters
	// Filters: part_number contains "RES" AND category = "Resistors"
	// Verify: Returns parts matching both conditions
}

func TestHandleAdvancedSearch_DateRanges(t *testing.T) {
	// Test date range filters
	// Filter: created_at between 2024-01-01 and 2024-12-31
	// Verify: Returns parts within date range
}

func TestHandleAdvancedSearch_NumericRanges(t *testing.T) {
	// Test numeric range filters
	// Filter: qty_on_hand > 100 AND cost < 50
	// Verify: Returns parts matching numeric conditions
}

func TestHandleAdvancedSearch_Sorting(t *testing.T) {
	// Test result sorting
	// Sort by: part_number ASC, cost DESC
	// Verify: Results correctly sorted
}

func TestHandleAdvancedSearch_Pagination(t *testing.T) {
	// Test pagination with large result sets
	// Create 100 parts, request page 2 (limit 20)
	// Verify: Returns correct page, total count correct
}

func TestHandleAdvancedSearch_InvalidOperators(t *testing.T) {
	// Test with invalid operators
	// Operator: "invalid_op"
	// Verify: 400 Bad Request with clear error
}

func TestHandleAdvancedSearch_SQLInjection(t *testing.T) {
	// Test SQL injection in search values
	// Value: "test' OR '1'='1"
	// Verify: Query is parameterized, no injection
}

func TestHandleAdvancedSearch_EmptyResults(t *testing.T) {
	// Test search with no matches
	// Verify: Returns empty array, not error
}

func TestHandleAdvancedSearch_Performance(t *testing.T) {
	// Create 10,000 parts
	// Run complex search
	// Verify: Returns results in <1 second
}
```

#### Handler 2: `handler_apikeys_test.go`

**Priority**: P0 (API security critical)  
**Effort**: 1 day

```go
func TestHandleAPIKeys_Create(t *testing.T) {
	// POST /api/v1/apikeys
	// Body: {"name": "Test Key", "expires_at": "2025-12-31"}
	// Verify: Key created, returns key value (one-time display)
}

func TestHandleAPIKeys_List(t *testing.T) {
	// GET /api/v1/apikeys
	// Verify: Returns list of keys (without key values)
}

func TestHandleAPIKeys_Revoke(t *testing.T) {
	// DELETE /api/v1/apikeys/{id}
	// Verify: Key revoked, cannot be used for authentication
}

func TestAPIKeyAuthentication_ValidKey(t *testing.T) {
	// Create API key
	// Use key in Authorization header
	// Verify: Request succeeds
}

func TestAPIKeyAuthentication_InvalidKey(t *testing.T) {
	// Use invalid key
	// Verify: 401 Unauthorized
}

func TestAPIKeyAuthentication_ExpiredKey(t *testing.T) {
	// Create key with past expiration
	// Use key
	// Verify: 401 Unauthorized
}

func TestAPIKeyAuthentication_RevokedKey(t *testing.T) {
	// Create key, revoke it
	// Use key
	// Verify: 401 Unauthorized
}

func TestAPIKeys_RateLimit(t *testing.T) {
	// Create key with rate limit
	// Exceed rate limit
	// Verify: 429 Too Many Requests
}
```

#### Handler 3: `handler_attachments_test.go`

**Priority**: P0 (File handling security critical)  
**Effort**: 1 day

```go
func TestHandleAttachments_Upload(t *testing.T) {
	// POST /api/v1/attachments
	// Upload file with multipart form
	// Verify: File saved, metadata returned
}

func TestHandleAttachments_List(t *testing.T) {
	// GET /api/v1/attachments?entity_type=part&entity_id=123
	// Verify: Returns attachments for entity
}

func TestHandleAttachments_Download(t *testing.T) {
	// GET /api/v1/attachments/{id}/download
	// Verify: File downloaded with correct content
}

func TestHandleAttachments_Delete(t *testing.T) {
	// DELETE /api/v1/attachments/{id}
	// Verify: File deleted from disk and database
}

func TestAttachments_FileTypeRestriction(t *testing.T) {
	// Upload .exe, .sh, .bat files
	// Verify: Rejected (if restrictions exist)
}

func TestAttachments_SizeLimit(t *testing.T) {
	// Upload 200MB file
	// Verify: Rejected if over limit
}

func TestAttachments_PathTraversal(t *testing.T) {
	// Upload file with path traversal in filename
	// Filename: "../../etc/passwd"
	// Verify: Filename sanitized
}

func TestAttachments_Authorization(t *testing.T) {
	// User A uploads file for part 1
	// User B (no access to part 1) tries to download
	// Verify: 403 Forbidden
}

func TestAttachments_MimeTypeValidation(t *testing.T) {
	// Upload file with spoofed MIME type
	// Verify: Actual file type validated
}
```

#### Handler 4: `handler_export_test.go`

**Priority**: P0 (Data export critical)  
**Effort**: 1 day

```go
func TestHandleExport_CSV(t *testing.T) {
	// GET /api/v1/export/parts?format=csv
	// Verify: Returns CSV with all parts
}

func TestHandleExport_Excel(t *testing.T) {
	// GET /api/v1/export/parts?format=xlsx
	// Verify: Returns Excel file
}

func TestHandleExport_PDF(t *testing.T) {
	// GET /api/v1/export/bom/{id}?format=pdf
	// Verify: Returns PDF
}

func TestHandleExport_FilteredData(t *testing.T) {
	// Export with filters
	// GET /api/v1/export/parts?category=Resistors&format=csv
	// Verify: Only filtered parts exported
}

func TestHandleExport_LargeDataset(t *testing.T) {
	// Create 10,000 parts
	// Export to CSV
	// Verify: All parts exported, reasonable performance (<10s)
}

func TestHandleExport_Authorization(t *testing.T) {
	// User with no export permission
	// Attempt export
	// Verify: 403 Forbidden
}

func TestHandleExport_Encoding(t *testing.T) {
	// Create parts with Unicode characters
	// Export to CSV
	// Verify: Correct UTF-8 encoding
}
```

#### Handler 5: `handler_permissions_test.go`

**Priority**: P0 (RBAC critical)  
**Effort**: 1 day

```go
func TestHandlePermissions_List(t *testing.T) {
	// GET /api/v1/permissions
	// Verify: Returns all defined permissions
}

func TestHandlePermissions_AssignToRole(t *testing.T) {
	// POST /api/v1/roles/{id}/permissions
	// Body: {"permission_ids": [1, 2, 3]}
	// Verify: Permissions assigned to role
}

func TestHandlePermissions_CheckUser(t *testing.T) {
	// GET /api/v1/users/{id}/permissions
	// Verify: Returns effective permissions (role + user-specific)
}

func TestPermissionEnforcement_ReadOnly(t *testing.T) {
	// User with read-only permission
	// Attempt POST /api/v1/parts
	// Verify: 403 Forbidden
}

func TestPermissionEnforcement_ModuleLevel(t *testing.T) {
	// User with no "inventory" module access
	// Attempt GET /api/v1/inventory
	// Verify: 403 Forbidden
}

func TestPermissionEnforcement_EntityLevel(t *testing.T) {
	// User can only see parts in specific category
	// Attempt to view part in different category
	// Verify: 403 Forbidden
}
```

#### Handlers 6-10: Quick Outlines

**Handler 6: `handler_quotes_test.go`** (1 day)
- Create quote, list quotes, update quote, delete quote
- Quote approval workflow
- Quote → Sales Order conversion
- Quote expiration handling
- Quote version tracking

**Handler 7: `handler_receiving_test.go`** (1 day)
- Receive PO, partial receiving, over-receiving
- Inventory update verification
- Quality inspection integration
- Receiving discrepancies (NCR creation)

**Handler 8: `handler_reports_test.go`** (1 day)
- Generate inventory report, BOM cost report
- Custom report parameters
- Report scheduling (if exists)
- Report export formats

**Handler 9: `handler_rma_test.go`** (1 day)
- Create RMA, list RMAs, update status
- RMA → NCR linkage
- RMA receiving and disposition
- Credit/refund processing

**Handler 10: `handler_search_test.go`** (1 day)
- Basic search, wildcard search
- Search across multiple entities
- Search result ranking
- Search performance with large datasets

---

### 3. E2E Critical User Journeys (Priority: CRITICAL)

**File**: `tests/critical-journeys.spec.js`  
**Estimated Effort**: 1.5 weeks (5 journeys × 2 days each)

#### Journey 1: Complete Procurement Cycle (2 days)

```javascript
test.describe('Complete Procurement Cycle', () => {
  test('RFQ → Quote → PO → Receiving → Invoice → Payment', async ({ page }) => {
    await login(page);
    
    // Step 1: Create RFQ
    await page.goto('/rfqs');
    await page.click('text=New RFQ');
    await page.fill('#rfq_number', 'RFQ-001');
    await page.fill('#vendor', 'Acme Corp');
    await page.click('text=Add Line Item');
    await page.fill('#part_number', 'RES-001');
    await page.fill('#quantity', '100');
    await page.click('button:has-text("Send to Vendor")');
    await expect(page.locator('text=RFQ sent')).toBeVisible();
    
    // Step 2: Receive Quote from Vendor
    await page.goto('/quotes');
    await page.click('text=New Quote');
    await page.selectOption('#rfq', 'RFQ-001');
    await page.fill('#unit_price', '1.50');
    await page.fill('#lead_time', '2 weeks');
    await page.click('button:has-text("Submit Quote")');
    
    // Step 3: Approve Quote → Convert to PO
    await page.click('button:has-text("Approve")');
    await page.click('button:has-text("Convert to PO")');
    await expect(page).toHaveURL(/\/pos\/\d+/);
    
    // Step 4: Receive PO
    const poNumber = await page.locator('#po_number').textContent();
    await page.click('button:has-text("Receive")');
    await page.fill('#quantity_received', '100');
    await page.click('button:has-text("Confirm Receipt")');
    
    // Step 5: Verify Inventory Updated
    await page.goto('/inventory');
    await page.fill('#search', 'RES-001');
    const qtyOnHand = await page.locator('[data-testid="qty-on-hand"]').textContent();
    expect(parseInt(qtyOnHand)).toBeGreaterThan(0);
    
    // Step 6: Invoice Creation
    await page.goto(`/pos/${poNumber}`);
    await page.click('button:has-text("Create Invoice")');
    await page.fill('#invoice_number', 'INV-001');
    await page.fill('#invoice_amount', '150.00');
    await page.click('button:has-text("Submit Invoice")');
    
    // Step 7: Process Payment
    await page.goto('/invoices');
    await page.click('text=INV-001');
    await page.click('button:has-text("Mark as Paid")');
    await page.fill('#payment_date', '2024-12-15');
    await page.fill('#payment_method', 'Wire Transfer');
    await page.click('button:has-text("Confirm Payment")');
    
    // Verify: PO marked complete, invoice paid
    await expect(page.locator('#po_status')).toHaveText('Complete');
    await expect(page.locator('#invoice_status')).toHaveText('Paid');
  });
});
```

#### Journey 2: Manufacturing Workflow (2 days)

```javascript
test.describe('Manufacturing Workflow', () => {
  test('Work Order → Material Picking → Build → QC → Ship', async ({ page }) => {
    await login(page);
    
    // Step 1: Create Work Order
    await page.goto('/work-orders');
    await page.click('text=New Work Order');
    await page.selectOption('#assembly', 'ASM-001');
    await page.fill('#quantity', '10');
    await page.fill('#due_date', '2024-12-31');
    await page.click('button:has-text("Create Work Order")');
    const woNumber = await page.locator('#wo_number').textContent();
    
    // Step 2: Material Picking (verify BOM components)
    await page.click('button:has-text("Start Work Order")');
    await expect(page.locator('text=Material Picking List')).toBeVisible();
    
    // Verify BOM components listed
    await expect(page.locator('text=RES-001')).toBeVisible();
    await expect(page.locator('[data-testid="required-qty"]')).toHaveText('50'); // 5 per assy × 10
    
    // Record material consumption
    await page.fill('#actual_qty_used', '50');
    await page.click('button:has-text("Consume Materials")');
    
    // Step 3: Build Process
    await page.click('tab:has-text("Build Log")');
    await page.click('button:has-text("Log Build Progress")');
    await page.fill('#qty_built', '10');
    await page.fill('#operator', 'John Doe');
    await page.fill('#notes', 'All 10 units assembled successfully');
    await page.click('button:has-text("Save Progress")');
    
    // Step 4: Quality Control
    await page.click('tab:has-text("Quality")');
    await page.click('button:has-text("Start QC")');
    await page.fill('#qty_inspected', '10');
    await page.fill('#qty_passed', '10');
    await page.fill('#qty_failed', '0');
    await page.selectOption('#inspector', 'Jane Smith');
    await page.click('button:has-text("Complete QC")');
    
    // Step 5: Complete Work Order
    await page.click('button:has-text("Complete Work Order")');
    await expect(page.locator('#wo_status')).toHaveText('Complete');
    
    // Step 6: Verify Inventory
    await page.goto('/inventory');
    await page.fill('#search', 'ASM-001');
    const qtyOnHand = await page.locator('[data-testid="qty-on-hand"]').textContent();
    expect(parseInt(qtyOnHand)).toBeGreaterThanOrEqual(10);
    
    // Verify component inventory decreased
    await page.fill('#search', 'RES-001');
    const componentQty = await page.locator('[data-testid="qty-on-hand"]').textContent();
    // Original qty - 50 consumed
  });
});
```

#### Journey 3: Quality Workflow - NCR → CAPA → ECO (2 days)

```javascript
test.describe('Quality Workflow', () => {
  test('NCR → Investigation → CAPA → ECO → Verification', async ({ page }) => {
    await login(page);
    
    // Step 1: Create NCR for defective part
    await page.goto('/ncrs');
    await page.click('text=New NCR');
    await page.fill('#ncr_number', 'NCR-001');
    await page.selectOption('#part', 'RES-001');
    await page.fill('#defect_description', 'Resistance out of tolerance');
    await page.fill('#quantity_affected', '100');
    await page.selectOption('#severity', 'Major');
    await page.click('button:has-text("Submit NCR")');
    
    // Step 2: Assign for Investigation
    await page.selectOption('#assigned_to', 'Quality Engineer');
    await page.click('button:has-text("Assign")');
    
    // Step 3: Investigation & Root Cause Analysis
    await page.click('tab:has-text("Investigation")');
    await page.fill('#root_cause', 'Incorrect vendor specification');
    await page.fill('#investigation_notes', 'Vendor confirmed spec error');
    await page.selectOption('#disposition', 'Scrap');
    await page.click('button:has-text("Complete Investigation")');
    
    // Step 4: Create CAPA
    await page.click('button:has-text("Create CAPA")');
    await page.fill('#corrective_action', 'Update vendor specification');
    await page.fill('#preventive_action', 'Add incoming inspection step');
    await page.selectOption('#assigned_to', 'Engineering');
    await page.fill('#due_date', '2025-01-31');
    await page.click('button:has-text("Submit CAPA")');
    
    // Step 5: CAPA → ECO
    await page.click('button:has-text("Create ECO for Spec Change")');
    await expect(page).toHaveURL(/\/ecos\/\d+/);
    
    await page.fill('#eco_number', 'ECO-001');
    await page.fill('#change_description', 'Update resistor tolerance spec');
    await page.selectOption('#affected_part', 'RES-001');
    await page.fill('#technical_details', 'Change tolerance from ±5% to ±1%');
    await page.click('button:has-text("Submit for Review")');
    
    // Step 6: ECO Approval
    await page.click('button:has-text("Approve ECO")');
    await page.fill('#approver_notes', 'Approved - update part spec');
    await page.click('button:has-text("Confirm Approval")');
    
    // Step 7: Implement ECO (update part)
    await page.goto('/parts/RES-001');
    await page.click('button:has-text("Edit")');
    await page.fill('#tolerance', '±1%');
    await page.fill('#eco_number', 'ECO-001');
    await page.click('button:has-text("Save")');
    
    // Step 8: Verify ECO Closure
    await page.goto('/ecos/ECO-001');
    await page.click('button:has-text("Mark as Implemented")');
    await expect(page.locator('#eco_status')).toHaveText('Implemented');
    
    // Step 9: Close CAPA
    await page.goto('/capas');
    await page.click('text=CAPA from NCR-001');
    await page.click('button:has-text("Close CAPA")');
    await page.fill('#effectiveness_check', 'New parts received in tolerance');
    await page.click('button:has-text("Confirm Closure")');
    
    // Step 10: Close NCR
    await page.goto('/ncrs/NCR-001');
    await page.click('button:has-text("Close NCR")');
    await expect(page.locator('#ncr_status')).toHaveText('Closed');
    
    // Verify: All linked (NCR ↔ CAPA ↔ ECO)
    await expect(page.locator('text=CAPA')).toBeVisible();
    await expect(page.locator('text=ECO-001')).toBeVisible();
  });
});
```

#### Journeys 4-5: Quick Outlines

**Journey 4: Complete ECO Flow** (2 days)
- Draft ECO → Add affected parts/BOMs → Submit for review
- Multi-level approval workflow
- Implementation (part updates, BOM changes)
- Verification and closure
- Audit trail verification

**Journey 5: Multi-user Concurrent Editing** (2 days)
- Two users edit same part simultaneously
- Verify conflict detection
- Test optimistic locking
- Verify last-write-wins or merge logic
- WebSocket real-time updates

---

### 4. Integration Workflow Tests (Priority: P0)

**File**: `integration_workflows_extended_test.go`  
**Estimated Effort**: 1 week (6 workflows)

#### Workflow 1: Sales Order → Work Order → Shipment

```go
func TestIntegration_SalesOrderToShipment(t *testing.T) {
	// 1. Create sales order for 10 assemblies
	// 2. Create work order from sales order
	// 3. Complete work order (build assemblies)
	// 4. Create shipment from sales order
	// 5. Ship order
	// 6. Verify: SO marked complete, inventory decreased, shipment tracked
}
```

#### Workflow 2: Part Change → ECO → BOM Update → Work Order Impact

```go
func TestIntegration_PartChangeImpact(t *testing.T) {
	// 1. Create old part, new part, assembly with BOM
	// 2. Create ECO to replace old part with new part
	// 3. Approve ECO
	// 4. Update BOM to use new part
	// 5. Create work order for assembly
	// 6. Verify: Work order uses new part, material picking list correct
}
```

#### Workflow 3: NCR → CAPA → ECO → Part Update

```go
func TestIntegration_QualityCorrectiveAction(t *testing.T) {
	// 1. Create NCR for defective part
	// 2. Create CAPA with root cause analysis
	// 3. Create ECO from CAPA
	// 4. Approve ECO and update part
	// 5. Close CAPA
	// 6. Close NCR
	// 7. Verify: All linked, audit trail complete
}
```

#### Workflow 4: Quote → Sales Order → Invoice → Payment

```go
func TestIntegration_SalesToCash(t *testing.T) {
	// 1. Create quote
	// 2. Approve quote → Convert to sales order
	// 3. Ship order → Create invoice
	// 4. Receive payment
	// 5. Verify: Sales order complete, invoice paid, revenue recognized
}
```

#### Workflow 5: PO → Receiving → Inspection → NCR

```go
func TestIntegration_DefectDetection(t *testing.T) {
	// 1. Create and approve PO
	// 2. Receive shipment
	// 3. Perform quality inspection (fail)
	// 4. Create NCR for defective parts
	// 5. Link NCR to PO and receiving record
	// 6. Verify: Inventory not updated (rejected), NCR linked
}
```

#### Workflow 6: Part → BOM → Cost Rollup → Quote

```go
func TestIntegration_CostCalculation(t *testing.T) {
	// 1. Create component parts with costs
	// 2. Create assembly BOM with quantities
	// 3. Calculate BOM cost rollup
	// 4. Create quote for assembly
	// 5. Verify: Quote price includes BOM cost + margin
}
```

---

## P1 - High Priority Tests

### 5. Error Recovery Tests (Priority: HIGH)

**File**: `error_recovery_test.go`  
**Estimated Effort**: 1.5 weeks (20 tests)

#### Database Failure Tests

```go
func TestErrorRecovery_DBConnectionLost(t *testing.T) {
	// 1. Start operation
	// 2. Simulate DB connection lost (close DB)
	// 3. Verify: Graceful error, retry logic kicks in
	// 4. Reconnect DB
	// 5. Verify: Operation completes or reports clear error
}

func TestErrorRecovery_DBDiskFull(t *testing.T) {
	// Simulate disk full during write operation
	// Verify: Error caught, partial data rolled back
}

func TestErrorRecovery_DBLocked(t *testing.T) {
	// Simulate SQLITE_BUSY error
	// Verify: Retry logic with backoff, operation succeeds
}

func TestErrorRecovery_TransactionDeadlock(t *testing.T) {
	// Create deadlock scenario (two transactions waiting on each other)
	// Verify: Deadlock detected, one transaction rolled back
}
```

#### Network Failure Tests

```go
func TestErrorRecovery_NetworkTimeout(t *testing.T) {
	// Simulate network timeout during API call
	// Verify: Timeout error returned, no hanging requests
}

func TestErrorRecovery_WebSocketDisconnect(t *testing.T) {
	// Establish WebSocket connection
	// Simulate network interruption
	// Verify: Client auto-reconnects, missed messages handled
}

func TestErrorRecovery_FileUploadInterrupted(t *testing.T) {
	// Start large file upload
	// Interrupt network mid-upload
	// Verify: Partial file cleaned up, upload can be retried
}
```

#### System Failure Tests

```go
func TestErrorRecovery_OutOfMemory(t *testing.T) {
	// Simulate memory exhaustion
	// Attempt large operation (e.g., export 100k records)
	// Verify: Graceful degradation, clear error message
}

func TestErrorRecovery_DiskFull(t *testing.T) {
	// Simulate disk full during file upload
	// Verify: Error caught, partial file deleted
}

func TestErrorRecovery_GracefulShutdown(t *testing.T) {
	// Start long-running operation
	// Send shutdown signal
	// Verify: Operation completes or rolled back, clean shutdown
}
```

#### User Error Recovery

```go
func TestErrorRecovery_UnsavedChanges(t *testing.T) {
	// E2E test: Edit form, navigate away
	// Verify: Warning shown, changes can be saved or discarded
}

func TestErrorRecovery_ConflictResolution(t *testing.T) {
	// User A and B edit same record
	// User A saves first
	// User B attempts to save
	// Verify: Conflict detected, merge UI shown
}

func TestErrorRecovery_SessionTimeout(t *testing.T) {
	// Long-running operation
	// Session expires
	// Verify: Prompted to re-authenticate, operation can continue
}
```

---

### 6. Load & Performance Tests (Priority: HIGH)

**File**: `load_test.go`  
**Estimated Effort**: 1 week (12 tests)

```go
func TestLoad_ConcurrentUsers(t *testing.T) {
	// Simulate 100 concurrent users
	// Each performs typical operations (list, view, edit)
	// Verify: No errors, response time <500ms p95
}

func TestLoad_LargeDataset(t *testing.T) {
	// Create 10,000 parts
	// Test list/search/filter operations
	// Verify: Response time <1s, pagination works
}

func TestLoad_ComplexBOM(t *testing.T) {
	// Create BOM with 500 components
	// Test BOM display, cost calculation
	// Verify: Loads in <2s, calculations correct
}

func TestLoad_BulkOperations(t *testing.T) {
	// Bulk update 1,000 parts
	// Verify: Completes in <10s, all updates applied
}

func TestLoad_LargeFileUpload(t *testing.T) {
	// Upload 100MB file
	// Verify: Upload succeeds, memory usage reasonable
}

func TestLoad_ComplexSearch(t *testing.T) {
	// Create 10,000 parts
	// Run complex search (5+ filters, sorting, pagination)
	// Verify: Results in <1s
}

func TestLoad_ReportGeneration(t *testing.T) {
	// Generate report with 10,000 rows
	// Verify: Completes in <30s, output correct
}

func TestLoad_ExportLargeDataset(t *testing.T) {
	// Export 10,000 parts to CSV
	// Verify: Export completes in <30s, file size reasonable
}

func TestLoad_WebSocketConnections(t *testing.T) {
	// Open 1,000 WebSocket connections
	// Verify: All connections stable, no memory leaks
}

func TestLoad_DatabaseQueryPerformance(t *testing.T) {
	// Run common queries against large dataset
	// Verify: All queries <100ms
	// Identify slow queries (query profiler)
}

func TestLoad_CacheEffectiveness(t *testing.T) {
	// If caching implemented
	// Measure cache hit ratio
	// Verify: >80% cache hits on repeated queries
}

func TestLoad_MemoryUsage(t *testing.T) {
	// Run application for 1 hour under load
	// Verify: Memory usage stable (no leaks)
}
```

---

### 7. Remaining Handler Coverage (Priority: HIGH)

**Estimated Effort**: 3 weeks (40 handlers)

**Approach**: Create tests for remaining 40 untested handlers using templates from existing handler tests.

**Batch 1 (Week 1)**: High-traffic handlers
- handler_bulk
- handler_docs
- handler_email
- handler_notifications
- handler_scan
- handler_widgets
- handler_calendar
- handler_costing
- handler_firmware
- handler_git_docs
- handler_prices
- handler_query_profiler
- handler_testing

**Batch 2 (Week 2)**: Integration handlers
- handler_ncr_integration (extend existing)
- handler_gitplm (extend existing)
- handler_backup (extend existing)

**Batch 3 (Week 3)**: Remaining handlers
- All other untested handlers following standard CRUD test template

---

## P2 - Medium Priority Tests

### 8. Migration & Upgrade Testing (Priority: MEDIUM)

**File**: `migration_test.go`  
**Estimated Effort**: 1 week (10 tests)

```go
func TestMigration_ForwardMigration(t *testing.T) {
	// Start with v1 schema
	// Apply all migrations
	// Verify: Schema matches expected v2
}

func TestMigration_DataPreservation(t *testing.T) {
	// Insert data in v1 schema
	// Run migration
	// Verify: All data preserved, transformed correctly
}

func TestMigration_Rollback(t *testing.T) {
	// Run migration
	// Simulate failure mid-migration
	// Verify: Rollback to v1 schema
}

func TestMigration_IndexCreation(t *testing.T) {
	// Create 10,000 rows
	// Add index via migration
	// Verify: Index created, queries faster
}

func TestMigration_ColumnTypeChange(t *testing.T) {
	// Change column type (e.g., VARCHAR to INT)
	// Verify: Data converted correctly or flagged for manual review
}

func TestMigration_AddForeignKey(t *testing.T) {
	// Add foreign key to existing data
	// Verify: Orphaned rows detected, migration handles gracefully
}

func TestUpgrade_V1toV2(t *testing.T) {
	// Full upgrade test: backup, migrate, verify
	// Verify: Application works with new schema
}

func TestUpgrade_DataCompatibility(t *testing.T) {
	// Export data from v1
	// Import to v2
	// Verify: All data accessible, no corruption
}

func TestUpgrade_ZeroDowntime(t *testing.T) {
	// If zero-downtime required
	// Run migration while application running
	// Verify: No service interruption
}

func TestImport_LegacyData(t *testing.T) {
	// Import data from legacy system
	// Verify: Data mapped correctly, validation applied
}
```

---

### 9. Visual Regression Testing (Priority: MEDIUM)

**Tool**: Playwright + Visual Regression Plugin  
**File**: `tests/visual-regression.spec.js`  
**Estimated Effort**: 1 week (15 tests)

```javascript
test.describe('Visual Regression Tests', () => {
  test('Dashboard layout', async ({ page }) => {
    await page.goto('/dashboard');
    await expect(page).toHaveScreenshot('dashboard.png');
  });
  
  test('Parts list table', async ({ page }) => {
    await page.goto('/parts');
    await expect(page.locator('#parts-table')).toHaveScreenshot('parts-table.png');
  });
  
  test('BOM tree view', async ({ page }) => {
    await page.goto('/boms/1');
    await expect(page.locator('#bom-tree')).toHaveScreenshot('bom-tree.png');
  });
  
  test('Responsive - Mobile view', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/parts');
    await expect(page).toHaveScreenshot('parts-mobile.png');
  });
  
  test('Responsive - Tablet view', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/parts');
    await expect(page).toHaveScreenshot('parts-tablet.png');
  });
  
  test('Theme - Dark mode', async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });
    await page.goto('/dashboard');
    await expect(page).toHaveScreenshot('dashboard-dark.png');
  });
  
  // ... more visual tests for critical UI components
});
```

---

## Implementation Strategy

### Phase 1: Foundation (Weeks 1-2) - P0 Security

**Goal**: Establish security testing baseline

1. Create `security_test.go` with 45 tests
2. Run tests, document all vulnerabilities
3. Fix P0 security issues immediately
4. Re-run tests until 100% pass
5. **Deliverable**: Security test suite, vulnerability report

### Phase 2: Critical Coverage (Weeks 2-5) - P0 Handlers & E2E

**Goal**: Test critical untested functionality

1. Week 2-3: Test 10 critical handlers (20 tests each = 200 tests total)
2. Week 3-4: Implement 5 E2E critical journeys (10 tests each = 50 tests total)
3. Week 4-5: Add 6 integration workflows (5 tests each = 30 tests total)
4. **Deliverable**: 280 new tests, coverage report

### Phase 3: Resilience (Weeks 5-7) - P1 Error Recovery & Load

**Goal**: Ensure system resilience under stress

1. Week 5-6: Error recovery tests (20 tests)
2. Week 7: Load and performance tests (12 tests)
3. **Deliverable**: Stress test report, performance benchmarks

### Phase 4: Completion (Weeks 8-10) - P1 Handler Coverage

**Goal**: Achieve 80%+ handler coverage

1. Week 8-10: Test remaining 40 handlers (systematic approach)
2. **Deliverable**: Complete handler test coverage

### Phase 5: Polish (Weeks 11-12) - P2 Migration & Visual

**Goal**: Production-ready polish

1. Week 11: Migration tests (10 tests)
2. Week 12: Visual regression tests (15 tests)
3. **Deliverable**: Production deployment checklist

---

## Tracking & Metrics

### Success Metrics

**Weekly Tracking**:
- Tests added (count)
- Coverage increase (%)
- Bugs found and fixed (count)
- Test execution time (seconds)

**Milestone Goals**:
- End of Week 2: Security suite complete
- End of Week 5: P0 coverage complete
- End of Week 7: P1 coverage complete
- End of Week 12: 80%+ total coverage

### Dashboard

Create `TEST_METRICS.md` updated weekly:

```markdown
# Test Metrics Dashboard

**Week**: X of 14  
**Phase**: [Foundation|Critical|Resilience|Completion|Polish]

## Coverage Progress

| Category | Tests | Coverage | Change |
|----------|-------|----------|--------|
| Backend Handlers | X/77 | XX% | +X% |
| Security | X/45 | XX% | +X% |
| E2E Journeys | X/10 | XX% | +X% |
| Integration | X/15 | XX% | +X% |
| Error Recovery | X/20 | XX% | +X% |
| Load Tests | X/12 | XX% | +X% |

## Bugs Found This Week

1. [P0] Security: SQL injection in search
2. [P1] Bug: Inventory not updating on receive
3. ...

## Next Week Plan

- Complete X tests in Y category
- Fix all P0 bugs from this week
```

---

## Conclusion

This roadmap provides **specific, actionable tests** with **clear priorities** and **realistic effort estimates**. Following this plan will take ZRP from **~40% test coverage to 80%+ coverage** in 14 weeks, addressing all critical gaps in security, functionality, and resilience.

**Key Takeaways**:
1. **Start with security** - P0 priority, 0% coverage is unacceptable
2. **Test critical paths first** - Focus on high-impact, untested areas
3. **Systematic approach** - Use templates and patterns for efficiency
4. **Measure progress** - Track coverage weekly, celebrate wins
5. **Fix as you go** - Don't accumulate test failures, fix immediately

**Estimated Total Effort**: 14 weeks (3.5 months) with 1 engineer, or 7 weeks with 2 engineers working in parallel.

---

**Document created**: February 19, 2026  
**Next review**: After Phase 1 completion (Week 2)
