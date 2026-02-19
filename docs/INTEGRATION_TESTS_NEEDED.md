# Integration Tests Status & Implementation Guide

> **Created:** 2026-02-19  
> **Status:** Integration test coverage incomplete  
> **Priority:** P0 - Critical for production readiness

## Executive Summary

ZRP has **excellent unit test coverage** (1,136 frontend tests + 40 backend test files, all passing). However, **integration tests for cross-module workflows are missing**. This creates risk that bugs could hide at module boundaries.

**Key Finding:** While individual handlers work correctly in isolation, the end-to-end flows that span multiple modules (BOM → Procurement → Inventory, WO → Completion → Inventory) lack comprehensive testing.

## Current Test Coverage

### ✅ Well-Tested Areas

| Module | Unit Tests | Status |
|--------|-----------|--------|
| Frontend Pages | 68 test files | ✅ All passing (1,136 tests) |
| Backend Handlers | 40+ test files | ✅ All passing |
| Auth & Middleware | Complete | ✅ |
| Individual CRUD | Complete | ✅ |
| API Schema | Complete | ✅ |

### ❌ Missing Integration Tests

| Workflow | Priority | Gap Description |
|----------|----------|-----------------|
| BOM → Procurement → Inventory | **P0** | No end-to-end test from shortage detection through PO receiving to inventory update |
| WO Completion → Inventory | **P0** | No test verifying finished goods added & components consumed |
| Material Reservation | **P0** | No test for qty_reserved updates when WO created |
| NCR → ECO → Implementation | **P1** | No test for complete quality workflow |
| NCR → CAPA Creation | **P1** | No test for corrective action flow |
| RMA → Device Status Update | **P1** | No test for RMA creation updating device |
| Quote → Sales Order | **P0** | Cannot test - sales order module doesn't exist |

## Critical Integration Test Cases

### TC-INT-001: BOM Shortage → PO → Inventory (P0)

**Workflow:** Check shortages → Generate PO → Receive PO → Verify inventory updated

```go
// Pseudo-code test structure
func TestIntegration_BOM_To_Inventory(t *testing.T) {
    // Setup: Create WO for 10x ASY-001
    // ASY-001 BOM requires: 100x RES-001, 50x CAP-001
    // Inventory: RES-001=5, CAP-001=2 (shortages exist)
    
    // Step 1: Check BOM shortages
    shortages := checkWorkOrderBOM("WO-001")
    assert.Len(shortages, 2)
    assert.Equal(95.0, shortages["RES-001"]) // Need 100, have 5
    
    // Step 2: Generate PO from shortages
    poID := generatePOFromWO("WO-001", "V-001")
    
    // Step 3: Verify PO created with correct line items
    po := getPO(poID)
    assert.Equal(2, len(po.Lines))
    
    // Step 4: Receive the PO
    receivePO(poID)
    
    // Step 5: Verify inventory updated
    inv := getInventory("RES-001")
    assert.Equal(100.0, inv.QtyOnHand) // 5 + 95 = 100
    
    // Step 6: Re-check shortages (should be resolved)
    shortages = checkWorkOrderBOM("WO-001")
    assert.Len(shortages, 0) // No shortages remaining
}
```

**Current Status:** ⚠️ **FRAGILE** (WORKFLOW_GAPS.md #3.1)
- No test exists
- Unclear if PO receiving auto-updates inventory
- Multiple handoff points untested

**Expected Gaps to Surface:**
- Inventory may not update automatically after PO receiving
- Transaction records may not be created
- qty_reserved may not be properly managed

---

### TC-INT-002: WO Completion → Inventory Update (P0)

**Workflow:** Create WO → Reserve materials → Complete → Verify inventory

```go
func TestIntegration_WO_Completion_Inventory(t *testing.T) {
    // Setup: WO-001 for 10x ASY-001
    // BOM: 10x RES-001, 5x CAP-001 per unit
    // Initial inventory: RES-001=100, CAP-001=50, ASY-001=0
    
    // Step 1: Create WO
    createWO("WO-001", "ASY-001", qty=10)
    
    // Step 2: Verify materials reserved
    inv := getInventory("RES-001")
    assert.Equal(100.0, inv.QtyReserved) // 10 units * 10 per unit
    
    // Step 3: Complete WO
    updateWO("WO-001", status="completed")
    
    // Step 4: Verify inventory updated
    invRES := getInventory("RES-001")
    assert.Equal(0.0, invRES.QtyOnHand)      // Consumed all 100
    assert.Equal(0.0, invRES.QtyReserved)    // Released reservation
    
    invCAP := getInventory("CAP-001")
    assert.Equal(0.0, invCAP.QtyOnHand)      // Consumed all 50
    
    invASY := getInventory("ASY-001")
    assert.Equal(10.0, invASY.QtyOnHand)     // Added finished goods
}
```

**Current Status:** 🔴 **BROKEN** (WORKFLOW_GAPS.md #4.1, #4.5)
- Creating WO does NOT reserve inventory
- Completing WO does NOT update inventory
- No material kitting step

**Expected Gaps to Surface:**
- qty_reserved stays 0 when WO created
- Inventory unchanged after WO completion
- No inventory transactions created

---

### TC-INT-003: Material Reservation on WO Creation (P0)

**Focus:** Verify that creating a work order reserves required materials

```go
func TestIntegration_Material_Reservation(t *testing.T) {
    // Setup: Inventory has sufficient stock
    setInventory("RES-001", qtyOnHand=100, qtyReserved=0)
    
    // Create WO requiring 50x RES-001
    createWO("WO-002", "ASY-001", qty=5) // 5 units * 10 per unit = 50
    
    // Verify reservation
    inv := getInventory("RES-001")
    assert.Equal(100.0, inv.QtyOnHand)    // Unchanged
    assert.Equal(50.0, inv.QtyReserved)   // Reserved
    assert.Equal(50.0, inv.Available())   // 100 - 50 = 50 available
}
```

**Current Status:** 🔴 **NOT IMPLEMENTED** (WORKFLOW_GAPS.md #4.1)

---

### TC-INT-004: NCR → ECO → Implementation (P1)

**Workflow:** Defect detected → ECO created → Approved → Implemented

```go
func TestIntegration_NCR_ECO_Flow(t *testing.T) {
    // Step 1: Create NCR for design defect
    ncrID := createNCR(title="Design flaw", defectType="design", 
                       correctiveAction="Redesign required")
    
    // Step 2: Create ECO from NCR (with database relation)
    ecoID := createECO(title="Fix from NCR", sourceNCRID=ncrID)
    
    // Step 3: Verify ECO-NCR link
    eco := getECO(ecoID)
    assert.Equal(ncrID, eco.SourceNCRID) // Database relation, not URL param
    
    // Step 4: Workflow: draft → open → approved → implemented
    updateECO(ecoID, status="open")
    approveECO(ecoID, approvedBy="test-user")
    implementECO(ecoID)
    
    // Step 5: Verify final state
    eco = getECO(ecoID)
    assert.Equal("implemented", eco.Status)
    
    // Step 6: Verify traceability maintained
    relatedECOs := getNCRRelatedECOs(ncrID)
    assert.Contains(relatedECOs, ecoID)
}
```

**Current Status:** ⚠️ **URL-PARAM BASED** (WORKFLOW_GAPS.md #2.7, #9.1)
- "Create ECO from NCR" uses query params (`/ecos?from_ncr=NCR-001`)
- No database relation between NCR and ECO
- Frontend must parse query params and auto-fill

**Expected Gaps:**
- source_ncr_id field may exist but not be populated
- Traceability relies on URL navigation, not database
- Cannot query "all ECOs for this NCR"

---

### TC-INT-005: WO Scrap/Yield Tracking (P1)

**Focus:** Verify scrap quantities are tracked and affect inventory

```go
func TestIntegration_WO_Scrap_Tracking(t *testing.T) {
    // Create WO for 100 units
    createWO("WO-003", "ASY-001", qty=100)
    
    // Complete with scrap: 95 good, 5 scrap
    updateWO("WO-003", status="completed", qtyGood=95, qtyScrap=5)
    
    // Verify WO recorded scrap
    wo := getWO("WO-003")
    assert.Equal(95, wo.QtyGood)
    assert.Equal(5, wo.QtyScrap)
    
    // Verify inventory reflects good quantity only
    inv := getInventory("ASY-001")
    assert.Equal(95.0, inv.QtyOnHand) // Only good units added, not all 100
}
```

**Current Status:** ⚠️ **UNKNOWN** (WORKFLOW_GAPS.md #4.6)
- qty_good/qty_scrap fields exist in schema
- Unclear if inventory update logic honors these fields

---

### TC-INT-006: Partial PO Receiving (P1)

**Focus:** Receive PO in multiple shipments

```go
func TestIntegration_Partial_PO_Receiving(t *testing.T) {
    // Create PO for 100 units
    poID := createPO(vendorID="V-001", lines=[{ipn:"RES-001", qty:100}])
    
    // Receive 50 units (first shipment)
    receivePartialPO(poID, lineID=1, qty=50)
    
    // Verify partial state
    po := getPO(poID)
    assert.Equal("partial", po.Status)
    assert.Equal(50.0, po.Lines[0].QtyReceived)
    
    inv := getInventory("RES-001")
    assert.Equal(50.0, inv.QtyOnHand) // First shipment received
    
    // Receive remaining 50 units
    receivePartialPO(poID, lineID=1, qty=50)
    
    // Verify completed state
    po = getPO(poID)
    assert.Equal("received", po.Status)
    assert.Equal(100.0, po.Lines[0].QtyReceived)
    
    inv = getInventory("RES-001")
    assert.Equal(100.0, inv.QtyOnHand) // Total received
}
```

**Current Status:** 🔴 **NOT IMPLEMENTED** (WORKFLOW_GAPS.md #3.4)
- Current PO receiving is all-or-nothing
- No endpoint for partial receiving

---

## Implementation Roadmap

### Phase 1: Document Existing Behavior ✅ COMPLETE
- [x] Identify critical workflows (INTEGRATION_TEST_PLAN.md)
- [x] Document expected vs actual behavior
- [x] Flag known gaps from WORKFLOW_GAPS.md
- [x] Create this implementation guide

### Phase 2: Create Test Infrastructure (NEXT)

**Create:** `handler_integration_test.go`

This file will contain all cross-module integration tests using the same patterns as existing tests.

**Structure:**
```go
package main

import (
    "database/sql"
    "testing"
    _ "modernc.org/sqlite"
)

func setupIntegrationDB(t *testing.T) *sql.DB {
    // Create in-memory SQLite with full schema
    // Use actual table definitions from db.go
    // Seed minimal test data
}

// Test functions following naming: TestIntegration_<Workflow>_<Scenario>
func TestIntegration_BOM_To_Inventory_Complete_Flow(t *testing.T) { ... }
func TestIntegration_WO_Material_Reservation(t *testing.T) { ... }
func TestIntegration_NCR_ECO_Database_Relation(t *testing.T) { ... }
```

**Key Principles:**
1. Use `t.Skip()` with clear messages for tests blocked by missing features
2. Document gaps with `t.Logf("⚠️ KNOWN GAP #X.Y: ...")` format
3. Assert expected behavior even if current implementation doesn't match
4. Tests should pass once gaps are fixed (no code changes to tests)

### Phase 3: Fill Critical Gaps (After Tests Written)

**Priority Order:**
1. **WO completion → inventory update** (P0) — Most critical missing feature
2. **Material reservation on WO creation** (P0) — Prevents double-allocation
3. **PO receiving → inventory update** (P0) — Procurement flow broken without this
4. **Sales order module** (P0 BLOCKER) — Quote acceptance is a dead end
5. **NCR/ECO database relations** (P1) — Replace URL-param linking

### Phase 4: Expand Coverage (Long-term)

- RMA → Device status workflow
- Firmware campaign → Device update flow  
- Serial number genealogy (component traceability)
- Multi-vendor PO generation
- WO-to-WO dependencies

---

## Testing Anti-Patterns to Avoid

### ❌ Don't: Test Only the Happy Path
```go
// BAD: Only tests when everything works
func TestBOM(t *testing.T) {
    // Assume inventory exists, PO succeeds, etc.
}
```

### ✅ Do: Test Edge Cases and Failures
```go
// GOOD: Tests what happens when inventory is insufficient
func TestBOM_InsufficientInventory(t *testing.T) {
    // Create WO with qty > inventory
    // Verify shortage is detected
    // Verify PO generation handles it correctly
}
```

### ❌ Don't: Mock Everything
```go
// BAD: Mocking defeats the purpose of integration tests
func TestBOM(t *testing.T) {
    mockInventory.EXPECT().Get(...)
    mockPO.EXPECT().Create(...)
    // This is a unit test, not integration test
}
```

### ✅ Do: Use Real Database, Real Handlers
```go
// GOOD: Actually exercises the full code path
func TestBOM(t *testing.T) {
    db := setupIntegrationDB(t) // Real SQLite
    // Call actual handlers
    // Verify actual database state
}
```

### ❌ Don't: Ignore Known Gaps
```go
// BAD: Test fails, developer ignores it
func TestInventoryUpdate(t *testing.T) {
    // Test fails but is left broken
}
```

### ✅ Do: Document Gaps Explicitly
```go
// GOOD: Test documents the gap and can pass once feature is implemented
func TestInventoryUpdate(t *testing.T) {
    // ... test code ...
    
    if actualBehavior != expectedBehavior {
        t.Logf("⚠️  KNOWN GAP #4.5: WO completion doesn't update inventory")
        t.Logf("   Expected: Inventory updated automatically")
        t.Logf("   Actual: Inventory unchanged")
        t.Skip("Skipping until gap is addressed")
    }
}
```

---

## Metrics & Success Criteria

### Current State
- Unit test coverage: **Excellent** (1,136 frontend + 40 backend)
- Integration test coverage: **Missing** (0 cross-module workflow tests)
- Known workflow gaps: **12** documented (WORKFLOW_GAPS.md)

### Target State (Production Ready)
- [ ] All P0 integration tests passing (5 tests)
- [ ] All P0 workflow gaps fixed (4 gaps)
- [ ] All P1 integration tests written (3 tests)
- [ ] Integration test CI job added (blocks merge if failing)

### Metrics to Track
| Metric | Current | Target |
|--------|---------|--------|
| P0 integration tests | 0 | 5 |
| P1 integration tests | 0 | 3 |
| Known workflow gaps (P0) | 4 | 0 |
| Cross-module bugs in production | Unknown | 0 |

---

## Conclusion

**Bottom Line:** ZRP has solid foundations (excellent unit tests, clean architecture) but **integration testing is the missing piece** before production readiness.

**Immediate Action:** Implement Phase 2 (test infrastructure) to surface the exact gaps, then systematically fix them.

**Long-term:** Make integration tests part of CI/CD so cross-module regressions are caught before deployment.

---

**See Also:**
- [INTEGRATION_TEST_PLAN.md](./INTEGRATION_TEST_PLAN.md) — Detailed test case specifications
- [WORKFLOW_GAPS.md](./WORKFLOW_GAPS.md) — Complete gap analysis across all modules
- [TESTING.md](./TESTING.md) — General testing documentation and patterns
