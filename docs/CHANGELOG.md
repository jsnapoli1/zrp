# CHANGELOG

## [Unreleased]

### Added - Comprehensive Integration Test Documentation (2026-02-19)

**Context:** Following the initial integration test planning, conducted a deep audit of ZRP's test coverage to identify the highest-value improvements needed for production readiness.

**Key Findings:**
- **Unit test coverage:** Excellent (1,136 frontend + 40 backend test files, all passing)
- **Integration test coverage:** Missing entirely for cross-module workflows
- **Highest risk:** Bugs at module boundaries (BOM→Procurement, WO→Inventory, NCR→ECO)

**Created:** `docs/INTEGRATION_TESTS_NEEDED.md` - Implementation guide containing:

1. **Current Test Coverage Assessment:**
   - Detailed breakdown of what's well-tested vs. missing
   - Identified 7 critical workflow gaps (3x P0, 4x P1)

2. **Critical Integration Test Cases (Fully Specified):**
   - **TC-INT-001:** BOM Shortage → PO → Inventory (P0)
   - **TC-INT-002:** WO Completion → Inventory Update (P0)
   - **TC-INT-003:** Material Reservation on WO Creation (P0)
   - **TC-INT-004:** NCR → ECO → Implementation (P1)
   - **TC-INT-005:** WO Scrap/Yield Tracking (P1)
   - **TC-INT-006:** Partial PO Receiving (P1)

3. **Implementation Roadmap:**
   - Phase 1: Documentation (✅ COMPLETE)
   - Phase 2: Test infrastructure setup (NEXT)
   - Phase 3: Fix critical gaps (after tests surface them)
   - Phase 4: Expand coverage long-term

4. **Testing Best Practices:**
   - ✅ DO: Use real database, test edge cases, document gaps explicitly
   - ❌ DON'T: Mock everything, test only happy path, ignore known gaps

**Documented Known Gaps (Cross-Referenced):**
- 🔴 **GAP #4.5:** WO completion doesn't update inventory (P0 BLOCKER)
- 🔴 **GAP #4.1:** Material reservation not implemented (P0 BLOCKER)
- 🔴 **GAP #3.1:** PO receiving → inventory update unclear (P0 FRAGILE)
- ⚠️ **GAP #9.1:** URL-param based linking (NCR→ECO/CAPA) instead of DB relations (P1)
- 🔴 **GAP #8.1:** No sales order module - quote workflow incomplete (P0 BLOCKER)

**Success Criteria Defined:**
- Target: 5 P0 integration tests passing
- Target: 4 P0 workflow gaps fixed
- Target: Integration tests in CI pipeline

**Impact:**
- Provides actionable roadmap for achieving production readiness
- Documents exact expected behavior for all critical workflows
- Establishes testing standards for future development
- Surfaces the 3 highest-priority features needed: inventory auto-update, material reservation, sales orders

**Recommendation:** Implement Phase 2 (test infrastructure) immediately to surface exact gaps, then systematically fix P0 blockers.

---

### Added - Integration Test Planning (2026-02-19)

**Context:** ZRP has excellent unit test coverage (1,224 frontend tests + 40 backend test files, all passing), but integration tests for cross-module workflows were missing. This creates risk for regressions when modules interact.

**Created:** `docs/INTEGRATION_TEST_PLAN.md` - Comprehensive test plan documenting:

1. **Critical Integration Flows Identified:**
   - BOM shortage → Procurement → PO → Receiving → Inventory (P0)
   - Work Order → Material Reservation → Completion → Inventory Update (P0)
   - NCR → ECO / CAPA Creation (P1)
   - Device → RMA → Repair → Return (P1)
   - Quote → Sales Order → Work Order → Shipment (P0 BLOCKER)

2. **Test Cases Documented:**
   - TC-INT-001 through TC-INT-011 covering end-to-end workflows
   - Expected behavior vs. actual behavior
   - Known gaps cross-referenced with WORKFLOW_GAPS.md

3. **Implementation Guidance:**
   - Test database setup patterns
   - HTTP test patterns using httptest
   - Strategy for documenting known gaps without failing tests

4. **Gaps Identified and Documented:**
   - ⚠️ GAP #4.1: Creating WO does NOT reserve materials (`qty_reserved` stays 0)
   - ⚠️ GAP #4.5: Completing WO does NOT update inventory (no auto add finished goods / consume materials)
   - ⚠️ GAP #9.1: URL-param based linking (NCR→ECO, NCR→CAPA, Device→RMA) - fragile pattern
   - 🔴 GAP #8.1: No sales order module exists - quote acceptance is a dead end
   - ⚠️ GAP #7.4: Device status not auto-updated when RMA created

**Impact:**
- Provides roadmap for integration test implementation
- Documents expected behavior for critical workflows
- Flags P0 blockers (sales orders, inventory updates) for prioritization
- Establishes testing patterns for future development

**Next Steps:**
1. Implement tests for working flows (BOM check, PO generation)
2. Address P0 gaps (WO inventory updates, sales orders)
3. Migrate URL-param linking to database relations
4. Add tests to CI pipeline for regression prevention

### Fixed - Procurement Handler Tests (2026-02-19)

**Issue:** Three procurement handler tests were failing due to incorrect API response decoding.

**Root Cause:** Tests were attempting to decode responses directly into domain structs, but handlers wrap all responses in an `APIResponse{Data: ...}` envelope. This caused:
- `TestHandleCreatePO_Success`: Empty ID and vendor_id fields
- `TestHandleCreatePO_DefaultStatus`: Empty status field  
- `TestHandleGeneratePOFromWO_Success`: Panic from nil interface conversion

**Fix:**
- Added helper functions `parsePO()` and `parsePOGenerateResponse()` in `handler_procurement_test.go`
- Updated failing tests to decode envelope first, then extract data
- All three tests now pass ✓

**Impact:** Procurement test suite now passes reliably. Pattern matches existing test helpers in `handler_devices_test.go` and `handler_doc_versions_test.go`.

---

### Fixed - Backend Test Suite (2026-02-19)

**Context:** Multiple backend test suites were failing due to schema mismatches and NULL handling issues.

**Root Causes Identified:**
1. **Test database schema drift** - Test setup functions used outdated column names:
   - `audit_log` table: used `timestamp` instead of `created_at`
   - Missing `user_id` column in test `audit_log` tables
   - `changes` table: used `timestamp` instead of `created_at`
   
2. **NULL value scanning errors** - Handlers attempted to scan potentially-NULL database columns directly into Go strings instead of using `COALESCE()` or `sql.NullString`

**Changes Made:**

#### Test Schema Fixes
- `handler_devices_test.go`: Fixed `audit_log` and `changes` table schemas to match production schema
- `handler_vendors_test.go`: Fixed `audit_log`, `changes`, and `undo_stack` table schemas
- `api_health_test.go`: Removed unused `fmt` import causing compilation errors

#### Handler Fixes
- `handler_eco.go`:
  - Added `COALESCE()` to all potentially-NULL TEXT/DATETIME columns in SELECT queries
  - Fixed `handleListECOs()` query
  - Fixed `handleGetECO()` query
  - **Impact:** ECO endpoints now properly handle records with NULL fields

**Test Results:**
- ✅ All device handler tests now passing (16/16)
- ✅ ECO list/filter tests now passing
- ✅ Eliminated ~5+ test failures related to schema mismatches
- ✅ Frontend tests: All 1224 tests passing (unchanged)

**Pattern for Future Tests:**
When creating test database setup functions:
1. Copy schema from `db.go` migrations, not from memory
2. Use `COALESCE(column, '')` for all columns that allow NULL when scanning into strings
3. Alternatively, use `sql.NullString` for nullable columns
4. Run `go test -v -run SpecificTest` to debug individual test failures

---

## Previous Entries

