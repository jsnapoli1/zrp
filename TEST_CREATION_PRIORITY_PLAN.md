# Test Creation Priority Plan

**Generated:** February 20, 2026  
**Purpose:** Prioritized roadmap for adding missing test coverage

---

## Phase 1: Critical Security & Infrastructure (Week 1)

### 🔴 STOP! Fix These Bugs FIRST

These aren't missing tests - these are **critical security bugs** found BY tests:

1. **RBAC Permission Enforcement (CRITICAL)**
   - **Bug:** Readonly users can create/modify resources
   - **File:** `middleware.go` or `permissions.go`
   - **Evidence:** `security_permissions_test.go` line 111, 257
   - **Fix:** Implement proper permission checking before write operations
   - **DO NOT add tests - FIX THE CODE!**

2. **Rate Limiting Broken (HIGH)**
   - **Bug:** Rate limiting not preventing brute force
   - **File:** `security.go` or `middleware.go`
   - **Evidence:** `handler_auth_test.go` line 258, `ws_auth_test.go` line 430
   - **Fix:** Implement/fix rate limiting middleware
   - **DO NOT add tests - FIX THE CODE!**

3. **Session Management Issues (HIGH)**
   - **Bug:** Valid sessions return 401, session sliding not working
   - **File:** `middleware.go`, `handler_auth.go`
   - **Evidence:** Multiple `middleware_test.go` failures
   - **Fix:** Debug session validation logic
   - **DO NOT add tests - FIX THE CODE!**

### Database Test Infrastructure

**Before writing ANY new tests, fix test database setup:**

**Create:** `test_helpers.go` (enhance existing)

```go
package main

import (
    "database/sql"
    "testing"
    "os"
)

// setupTestDB creates isolated test database with migrations
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    
    // Use in-memory database for speed
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("Failed to open test db: %v", err)
    }
    
    // Run all migrations
    if err := runMigrations(db); err != nil {
        t.Fatalf("Failed to run migrations: %v", err)
    }
    
    // Register cleanup
    t.Cleanup(func() {
        db.Close()
    })
    
    return db
}

// runMigrations applies all schema migrations
func runMigrations(db *sql.DB) error {
    // Read schema from main.go or separate migration files
    schema := `
        CREATE TABLE IF NOT EXISTS users (...);
        CREATE TABLE IF NOT EXISTS audit_log (...);
        CREATE TABLE IF NOT EXISTS password_history (...);
        CREATE TABLE IF NOT EXISTS parts (...);
        CREATE TABLE IF NOT EXISTS inventory (...);
        -- ... all tables
    `
    _, err := db.Exec(schema)
    return err
}

// seedTestData inserts common test fixtures
func seedTestData(t *testing.T, db *sql.DB) {
    // Insert test users, parts, etc.
}
```

**Update ALL existing tests to use:**
```go
func TestSomething(t *testing.T) {
    db := setupTestDB(t)  // Auto-cleanup, isolated
    // ... rest of test
}
```

**Estimated Time:** 2-3 days (will fix ~50% of failing tests)

---

## Phase 2: Critical Untested Handlers (Week 2)

### Priority 1: Revenue & Security Critical

#### Test 1: `handler_quotes_test.go`
**Why:** Revenue-critical, customer-facing  
**Estimated LOC:** 400-500 lines  
**Time:** 1 day

```go
// Test structure
func TestHandleGetQuotes_Success(t *testing.T) { ... }
func TestHandleGetQuotes_EmptyResult(t *testing.T) { ... }
func TestHandleGetQuotes_Pagination(t *testing.T) { ... }
func TestHandleGetQuote_ByID_Success(t *testing.T) { ... }
func TestHandleGetQuote_NotFound(t *testing.T) { ... }
func TestHandleCreateQuote_Success(t *testing.T) { ... }
func TestHandleCreateQuote_ValidationFails(t *testing.T) { ... }
func TestHandleCreateQuote_DuplicateQuoteID(t *testing.T) { ... }
func TestHandleUpdateQuote_Success(t *testing.T) { ... }
func TestHandleUpdateQuote_NotFound(t *testing.T) { ... }
func TestHandleDeleteQuote_Success(t *testing.T) { ... }
func TestHandleDeleteQuote_WithDependencies(t *testing.T) { ... }
func TestHandleQuotePDF_Export(t *testing.T) { ... }
func TestHandleQuotePDF_NotFound(t *testing.T) { ... }
func TestQuoteStatusTransitions(t *testing.T) { ... }
func TestQuoteApprovalWorkflow(t *testing.T) { ... }
```

**Coverage Target:** 90%+

---

#### Test 2: `handler_permissions_test.go`
**Why:** Security-critical, authorization foundation  
**Estimated LOC:** 300-400 lines  
**Time:** 1 day

```go
func TestHandleGetPermissions_Success(t *testing.T) { ... }
func TestHandleGetPermissions_ByRole(t *testing.T) { ... }
func TestHandleUpdatePermissions_Success(t *testing.T) { ... }
func TestHandleUpdatePermissions_InvalidRole(t *testing.T) { ... }
func TestHandleUpdatePermissions_RemoveLastAdmin(t *testing.T) { ... }
func TestPermissionCheck_ReadAccess(t *testing.T) { ... }
func TestPermissionCheck_WriteAccess(t *testing.T) { ... }
func TestPermissionCheck_AdminOnly(t *testing.T) { ... }
func TestRoleHierarchy(t *testing.T) { ... }
func TestPermissionInheritance(t *testing.T) { ... }
```

**Coverage Target:** 95%+ (security critical)

---

#### Test 3: `handler_attachments_test.go`
**Why:** Security risk (file uploads)  
**Estimated LOC:** 400-500 lines  
**Time:** 1 day

```go
func TestHandleUploadAttachment_Success(t *testing.T) { ... }
func TestHandleUploadAttachment_FileSizeLimit(t *testing.T) { ... }
func TestHandleUploadAttachment_InvalidMIME(t *testing.T) { ... }
func TestHandleUploadAttachment_MaliciousFile(t *testing.T) { ... }
func TestHandleUploadAttachment_PathTraversal(t *testing.T) { ... }
func TestHandleGetAttachment_Success(t *testing.T) { ... }
func TestHandleGetAttachment_NotFound(t *testing.T) { ... }
func TestHandleDeleteAttachment_Success(t *testing.T) { ... }
func TestHandleDeleteAttachment_Unauthorized(t *testing.T) { ... }
func TestAttachmentVirus_Scan(t *testing.T) { ... }
func TestAttachmentLinking_ToPart(t *testing.T) { ... }
func TestAttachmentLinking_ToECO(t *testing.T) { ... }
```

**Coverage Target:** 90%+ (security focus on upload validation)

---

### Priority 2: Core Business Logic

#### Test 4: `handler_rma_test.go`
**Why:** Customer service critical  
**Estimated LOC:** 400-500 lines  
**Time:** 1 day

```go
func TestHandleGetRMAs_Success(t *testing.T) { ... }
func TestHandleGetRMA_ByID(t *testing.T) { ... }
func TestHandleCreateRMA_Success(t *testing.T) { ... }
func TestHandleCreateRMA_AutoGenerateID(t *testing.T) { ... }
func TestHandleUpdateRMA_Success(t *testing.T) { ... }
func TestHandleUpdateRMA_StatusTransition(t *testing.T) { ... }
func TestHandleApproveRMA_Success(t *testing.T) { ... }
func TestHandleRejectRMA_Success(t *testing.T) { ... }
func TestRMAStatusWorkflow_Complete(t *testing.T) { ... }
func TestRMANotification_ToCustomer(t *testing.T) { ... }
func TestRMALinkedToNCR(t *testing.T) { ... }
func TestRMARefund_Processing(t *testing.T) { ... }
```

**Coverage Target:** 85%+

---

#### Test 5: `handler_receiving_test.go`
**Why:** Inventory accuracy depends on it  
**Estimated LOC:** 400-500 lines  
**Time:** 1 day

```go
func TestHandleReceivePO_Success(t *testing.T) { ... }
func TestHandleReceivePO_PartialQuantity(t *testing.T) { ... }
func TestHandleReceivePO_OverReceive(t *testing.T) { ... }
func TestHandleReceivePO_LotTracking(t *testing.T) { ... }
func TestHandleReceivePO_SerialNumbers(t *testing.T) { ... }
func TestHandleReceivePO_InventoryUpdate(t *testing.T) { ... }
func TestHandleReceivePO_QualityInspection(t *testing.T) { ... }
func TestHandleReceivePO_POClosure(t *testing.T) { ... }
func TestHandleReceiveHistory_ByPO(t *testing.T) { ... }
func TestHandleReceiveHistory_ByPart(t *testing.T) { ... }
func TestReceivingDisc Discrepancy_Alert(t *testing.T) { ... }
```

**Coverage Target:** 85%+

---

## Phase 3: Remaining Handlers (Week 3)

### Test 6-10: Medium Priority (1 day each)

6. **handler_reports_test.go** (BI critical)
   - Report generation
   - Export formats (PDF, CSV, HTML)
   - Data accuracy
   - Permission-based filtering

7. **handler_advanced_search_test.go** (UX critical)
   - Multi-field search
   - Filter combinations
   - Search performance
   - Result relevance

8. **handler_bulk_test.go** (operational efficiency)
   - Bulk create/update/delete
   - Error handling (partial failures)
   - Transaction rollback
   - Progress reporting

9. **handler_email_test.go** (notifications)
   - Email sending
   - Template rendering
   - Attachment handling
   - Queue management

10. **handler_export_test.go** (data portability)
    - CSV export
    - Excel export
    - JSON export
    - Large dataset handling

### Test 11-15: Lower Priority (1/2 day each)

11. **handler_apikeys_test.go**
12. **handler_calendar_test.go**
13. **handler_docs_test.go**
14. **handler_firmware_test.go**
15. **handler_git_docs_test.go**

### Test 16-25: Nice-to-Have (1/2 day each)

16. **handler_costing_test.go**
17. **handler_ncr_integration_test.go** (may be covered by quality_workflow_test.go)
18. **handler_notifications_test.go**
19. **handler_prices_test.go**
20. **handler_query_profiler_test.go** (dev tool)
21. **handler_scan_test.go**
22. **handler_search_test.go**
23. **handler_testing_test.go**
24. **handler_widgets_test.go**
25. **handler_rfq_test.go** (has .skip file - may exist but disabled)

---

## Phase 4: Integration Test Expansion (Week 4)

### Integration Test 1: Complete Procurement Flow
**File:** `integration_procurement_flow_test.go`  
**Time:** 2 days

```go
func TestProcurementFlow_RFQ_to_Inventory(t *testing.T) {
    // 1. Create RFQ
    // 2. Convert RFQ to Quote
    // 3. Approve Quote → PO
    // 4. Receive PO
    // 5. Verify inventory updated
    // 6. Verify costs recorded
}
```

### Integration Test 2: Sales to Fulfillment Flow
**File:** `integration_sales_fulfillment_test.go`  
**Time:** 2 days

```go
func TestSalesFlow_Order_to_Shipment(t *testing.T) {
    // 1. Create Sales Order
    // 2. Generate Work Order
    // 3. Kit materials (deduct inventory)
    // 4. Complete Work Order
    // 5. Create Shipment
    // 6. Verify inventory updated
}
```

### Integration Test 3: Quality Management Flow
**File:** `integration_quality_flow_test.go`  
**Time:** 2 days

```go
func TestQualityFlow_FieldReport_to_Resolution(t *testing.T) {
    // 1. Create Field Report
    // 2. Generate NCR
    // 3. Create CAPA from NCR
    // 4. Create ECO from CAPA
    // 5. Update BOM via ECO
    // 6. Close NCR/CAPA
}
```

---

## Phase 5: Edge Case Hardening (Week 5)

### Edge Case Test Suite 1: Concurrency
**File:** `edge_case_concurrency_test.go`  
**Time:** 2 days

```go
func TestConcurrentPartCreation_SameIPN(t *testing.T) { ... }
func TestConcurrentInventoryUpdates_SamePart(t *testing.T) { ... }
func TestConcurrentOrderPlacement_LimitedStock(t *testing.T) { ... }
func TestConcurrentUserLogout_ActiveSession(t *testing.T) { ... }
```

### Edge Case Test Suite 2: Limits & Boundaries
**File:** `edge_case_limits_test.go`  
**Time:** 2 days

```go
func TestLargeDataset_10000Parts(t *testing.T) { ... }
func TestVeryLongText_DescriptionField(t *testing.T) { ... }
func TestMaxFileSize_Upload(t *testing.T) { ... }
func TestDeepBOM_100Levels(t *testing.T) { ... }
func TestPagination_EdgeCases(t *testing.T) { ... }
```

### Edge Case Test Suite 3: Failure Recovery
**File:** `edge_case_recovery_test.go`  
**Time:** 2 days

```go
func TestDatabaseConnection_Lost(t *testing.T) { ... }
func TestTransaction_Deadlock(t *testing.T) { ... }
func TestPartialAPI_Failure(t *testing.T) { ... }
func TestEmail_SendFailed_Retry(t *testing.T) { ... }
```

---

## Timeline Summary

| Phase | Tasks | Duration | Cumulative |
|-------|-------|----------|------------|
| **Phase 1** | Fix bugs + test infrastructure | 1 week | Week 1 |
| **Phase 2** | 5 critical handler tests | 1 week | Week 2 |
| **Phase 3** | Remaining 20 handler tests | 2 weeks | Week 4 |
| **Phase 4** | Integration tests | 1 week | Week 5 |
| **Phase 5** | Edge cases | 1 week | Week 6 |
| **Buffer** | Documentation, fixes | 1 week | Week 7 |

**Total Estimated Time:** 6-7 weeks for complete coverage

---

## Test Writing Best Practices

### 1. Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"Valid email", "user@example.com", false},
        {"Invalid format", "not-an-email", true},
        {"Empty string", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateEmail(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error = %v, want error = %v", err, tt.wantErr)
            }
        })
    }
}
```

### 2. Test Helpers

```go
// createTestPart creates a part with sensible defaults
func createTestPart(t *testing.T, db *sql.DB, overrides map[string]interface{}) Part {
    t.Helper()
    part := Part{
        IPN: "TEST-001",
        Description: "Test Part",
        Category: "Electronics",
    }
    // Apply overrides
    // Insert into DB
    return part
}
```

### 3. Cleanup

```go
func TestSomething(t *testing.T) {
    db := setupTestDB(t)
    
    // Cleanup registered automatically via t.Cleanup()
    t.Cleanup(func() {
        db.Exec("DELETE FROM parts WHERE ipn LIKE 'TEST-%'")
    })
    
    // ... test code
}
```

### 4. Assertion Helpers

```go
func assertEqual(t *testing.T, got, want interface{}) {
    t.Helper()
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}

func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

---

## Coverage Tracking

### Measure Progress

```bash
# Baseline
go test -cover ./... > coverage_baseline.txt

# After Phase 2
go test -cover ./... > coverage_phase2.txt

# After Phase 3
go test -cover ./... > coverage_phase3.txt

# Generate HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Goals

| Phase | Target Coverage | Current |
|-------|----------------|---------|
| Baseline | - | 58.5% |
| Phase 1 | 60% | - |
| Phase 2 | 70% | - |
| Phase 3 | 80% | - |
| Phase 4 | 85% | - |
| Phase 5 | 90% | - |

---

## Success Criteria

✅ **Phase 1 Complete When:**
- All critical bugs fixed
- Test database isolation working
- 95%+ of existing tests passing

✅ **Phase 2 Complete When:**
- 5 critical handlers have 85%+ test coverage
- All tests passing
- No security vulnerabilities in tested code

✅ **Phase 3 Complete When:**
- All handlers have test files
- Overall backend coverage >80%
- All tests passing

✅ **Phase 4 Complete When:**
- 3 major integration flows fully tested
- Cross-module dependencies verified
- All integration tests passing

✅ **Phase 5 Complete When:**
- Edge cases documented and tested
- Concurrency issues identified and fixed
- System robust under stress

---

## Questions to Answer with Tests

1. **What happens when two users update the same part simultaneously?**
2. **Can a work order be kitted with insufficient inventory?**
3. **What if a PO is received without matching line items?**
4. **Can circular BOMs be created through multi-step updates?**
5. **What's the maximum safe BOM depth?**
6. **How does the system handle 10,000+ parts?**
7. **Can permissions be escalated through API manipulation?**
8. **What happens when database connections are exhausted?**
9. **Are all state transitions reversible?**
10. **Is data integrity maintained across transaction failures?**

---

**Generated:** February 20, 2026  
**Next Review:** After Phase 1 completion  
**Owner:** Engineering Team
