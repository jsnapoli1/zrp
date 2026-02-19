# CHANGELOG

## [Unreleased]

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

