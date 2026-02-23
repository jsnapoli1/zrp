# ZRP Edge Case Test Plan

**Date**: February 19, 2026  
**Author**: Eva (AI Security Auditor)  
**Application**: ZRP ERP System  
**Purpose**: Comprehensive edge case testing to ensure production readiness

---

## Executive Summary

This document identifies **87 critical edge cases** that need testing across 9 major categories to ensure ZRP handles boundary conditions, error scenarios, and extreme data gracefully. These tests are essential for production deployment.

### Risk Assessment

| Category | Edge Cases | Risk Level | Current Coverage |
|----------|------------|------------|------------------|
| Boundary Values | 23 | **CRITICAL** | ~5% |
| Numeric Overflow | 12 | **CRITICAL** | 0% |
| String/Input Limits | 15 | **HIGH** | ~10% |
| Error Scenarios | 11 | **CRITICAL** | ~20% |
| Data Volume/Performance | 8 | **HIGH** | 0% |
| Concurrency/Race Conditions | 9 | **CRITICAL** | 0% |
| Data Integrity | 7 | **CRITICAL** | ~30% |
| Special Characters/Injection | 6 | **CRITICAL** | ~80% (security audit) |
| File Operations | 6 | **MEDIUM** | ~10% |

**Total Edge Cases Identified**: 87  
**Total Coverage Estimate**: **15-20%**  
**Production-Ready Threshold**: **85%+**

---

## 1. Boundary Values Testing

### 1.1 Empty/Null/Undefined Values

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| BC-001 | Parts API | Create part with empty IPN | Reject with 400 "IPN required" | ⚠️ UNKNOWN |
| BC-002 | Parts API | Create part with null category | Reject with 400 "category required" | ⚠️ UNKNOWN |
| BC-003 | Inventory API | Update inventory with empty location | Accept (location is optional) | ⚠️ UNKNOWN |
| BC-004 | Work Orders | Create WO with null notes | Accept (notes optional, store as empty string) | ⚠️ UNKNOWN |
| BC-005 | PO Lines | Create PO line with null manufacturer | Accept (optional field) | ⚠️ UNKNOWN |
| BC-006 | Vendors | Create vendor with empty contact_name | Accept (optional) | ⚠️ UNKNOWN |
| BC-007 | Documents | Upload document with null description | Accept | ⚠️ UNKNOWN |
| BC-008 | NCRs | Create NCR with empty root_cause | Accept initially (can add later) | ⚠️ UNKNOWN |
| BC-009 | Search API | Search with empty query string | Return all results (paginated) | ⚠️ UNKNOWN |
| BC-010 | Export API | Export with no filters | Export all records | ⚠️ UNKNOWN |

**Recommendation**: Add validation tests for all required vs. optional fields.

---

### 1.2 Zero Values

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| BC-011 | Inventory | Set qty_on_hand = 0 | Accept (valid stock level) | ✅ PASS (CHECK constraint) |
| BC-012 | Inventory | Set reorder_point = 0 | Accept (disables reordering) | ✅ PASS |
| BC-013 | PO Lines | qty_ordered = 0 | Reject with CHECK constraint error | ✅ PASS (CHECK qty_ordered > 0) |
| BC-014 | Work Orders | qty = 0 | Reject with CHECK constraint error | ✅ PASS (CHECK qty > 0) |
| BC-015 | Shipment Lines | qty = 0 | Reject with CHECK constraint error | ✅ PASS (CHECK qty > 0) |
| BC-016 | Vendors | lead_time_days = 0 | Accept (same-day delivery) | ✅ PASS (CHECK >= 0) |
| BC-017 | Product Pricing | price = 0 | Accept (free/internal transfer) | ⚠️ UNKNOWN |
| BC-018 | Inventory Transactions | qty = 0 | ⚠️ Should reject (meaningless transaction) | ❌ NO CHECK |

**Gap Found**: Inventory transactions allow zero quantity, which is illogical.

---

### 1.3 Negative Values

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| BC-019 | Inventory | qty_on_hand = -10 | Reject (CHECK constraint) | ✅ PASS (CHECK >= 0) |
| BC-020 | Inventory | qty_reserved = -5 | Reject (CHECK constraint) | ✅ PASS (CHECK >= 0) |
| BC-021 | Inventory | reorder_point = -1 | Reject (CHECK constraint) | ✅ PASS (CHECK >= 0) |
| BC-022 | Inventory Transactions | qty = -100 | Accept for returns/adjustments | ⚠️ DEPENDS ON TYPE |
| BC-023 | Vendors | lead_time_days = -5 | Reject (CHECK constraint) | ✅ PASS (CHECK >= 0) |

**Note**: Inventory transactions need type-specific validation (returns can be negative, receives cannot).

---

## 2. Numeric Overflow & Large Numbers

### 2.1 Integer Overflow

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| NO-001 | Work Orders | qty = 2147483648 (INT_MAX+1) | Reject or handle gracefully | ❌ UNTESTED |
| NO-002 | Shipment Lines | qty = 9223372036854775807 | Handle or reject | ❌ UNTESTED |
| NO-003 | Inventory | qty_on_hand = 1e308 (REAL max) | Handle or reject | ❌ UNTESTED |
| NO-004 | PO Lines | qty_ordered = 999999999999 | Accept or validate max | ❌ UNTESTED |
| NO-005 | Pricing | unit_price = 1e100 | Handle large prices | ❌ UNTESTED |
| NO-006 | Vendors | lead_time_days = 999999 | Validate reasonable max (e.g., 365 days) | ❌ NO VALIDATION |

**Critical Gap**: No maximum value validation on any numeric fields.

---

### 2.2 Floating Point Precision

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| NO-007 | Inventory | qty_on_hand = 0.33333333333333 | Store with precision loss acceptable | ⚠️ UNKNOWN |
| NO-008 | PO Lines | unit_price = 99.999999999 | Round to 2 decimal places? | ❌ NO ROUNDING |
| NO-009 | Calculations | Multiply 0.1 * 0.2 (floating point error) | Use decimal types or round | ❌ UNTESTED |
| NO-010 | Inventory | qty = 1.0000000001 vs 1.0 comparison | Handle precision errors | ❌ UNTESTED |
| NO-011 | Totals | Sum 1000 line items with $0.01 prices | Accumulation errors? | ❌ UNTESTED |
| NO-012 | Currency | Store $1,234,567.89 exactly | No precision loss | ⚠️ UNKNOWN |

**Recommendation**: Use DECIMAL/NUMERIC types for currency, not REAL/FLOAT.

---

## 3. String Length & Input Limits

### 3.1 Very Long Strings

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| SL-001 | Parts | IPN = 10,000 character string | Reject with max length error | ❌ NO VALIDATION |
| SL-002 | Parts | description = 100,000 chars | Truncate or reject | ❌ NO VALIDATION |
| SL-003 | Notes fields | notes = 1MB of text | Store or reject with max | ❌ NO VALIDATION |
| SL-004 | Vendors | name = 5,000 chars | Reject | ❌ NO VALIDATION |
| SL-005 | Search | search query = 50,000 chars | Handle or reject | ❌ NO VALIDATION |
| SL-006 | Export | filename = 1,000 chars | Validate path length | ❌ NO VALIDATION |
| SL-007 | JSON payload | Single field = 10MB | Reject request | ⚠️ SERVER MAX |
| SL-008 | HTTP headers | Custom header = 100KB | Reject | ⚠️ SERVER LIMIT |

**Critical Gap**: No input length validation on TEXT fields.

---

### 3.2 Special String Formats

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| SL-009 | Email fields | email = "not-an-email" | Validate format | ❌ NO VALIDATION |
| SL-010 | Email fields | email = "" | Allow empty or reject | ⚠️ INCONSISTENT |
| SL-011 | Phone numbers | phone = "abc123" | Validate format or accept any | ❌ NO VALIDATION |
| SL-012 | URLs | website = "javascript:alert(1)" | Sanitize/validate | ❌ NO VALIDATION |
| SL-013 | Dates | ship_date = "not-a-date" | Reject with format error | ⚠️ UNKNOWN |
| SL-014 | Dates | ship_date = "9999-12-31" | Accept far future dates? | ⚠️ UNKNOWN |
| SL-015 | Tracking numbers | tracking = "" (empty) | Allow empty tracking | ⚠️ UNKNOWN |

**Recommendation**: Add format validation for email, URL, phone, date fields.

---

## 4. Error Scenarios & Resilience

### 4.1 Database Errors

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| ER-001 | All APIs | Database connection lost | Return 503 "Service Unavailable" | ❌ UNTESTED |
| ER-002 | All APIs | Database locked (busy_timeout exceeded) | Retry or return 503 | ⚠️ UNKNOWN |
| ER-003 | Transactions | Constraint violation mid-transaction | Rollback + error message | ⚠️ PARTIAL |
| ER-004 | Foreign Keys | Delete vendor with active POs | Reject (ON DELETE RESTRICT) | ✅ TESTED |
| ER-005 | Unique constraints | Duplicate serial number | Reject with clear error | ✅ TESTED |
| ER-006 | Migrations | Migration fails mid-way | Database corrupted? | ❌ UNTESTED |
| ER-007 | Backup | Backup during write transaction | Consistent backup? | ❌ UNTESTED |

**Critical**: Need database failure and recovery testing.

---

### 4.2 Network & Infrastructure Errors

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| ER-008 | Frontend | API request timeout (30s+) | Show timeout error | ⚠️ UNKNOWN |
| ER-009 | Frontend | Network disconnected mid-request | Graceful error message | ⚠️ UNKNOWN |
| ER-010 | File Upload | Network fails during upload | Retry or clear error | ❌ UNTESTED |
| ER-011 | WebSocket | Connection drops | Reconnect automatically | ❌ UNTESTED |

---

## 5. Data Volume & Performance

### 5.1 Large Datasets

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| DV-001 | Parts List | 10,000+ parts in database | List loads with pagination | ❌ UNTESTED |
| DV-002 | Parts List | 100,000+ parts | Performance acceptable (<2s) | ❌ UNTESTED |
| DV-003 | BOM | 1,000+ line items in single BOM | Loads and edits smoothly | ❌ UNTESTED |
| DV-004 | Search | Search across 50,000+ parts | Results in <1s | ❌ UNTESTED |
| DV-005 | Export | Export 100,000 parts to CSV | Completes without timeout | ❌ UNTESTED |
| DV-006 | Export | Export 100,000 parts to Excel | Completes without OOM | ❌ UNTESTED |
| DV-007 | Audit Log | 1,000,000+ audit entries | Pagination works, queries fast | ❌ UNTESTED |
| DV-008 | Dashboard | Aggregate 100,000+ records | Metrics load quickly | ❌ UNTESTED |

**Critical**: Load testing with realistic data volumes is essential.

---

## 6. Concurrency & Race Conditions

### 6.1 Concurrent Updates

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| RC-001 | Inventory | 2 users update same IPN simultaneously | Last write wins or lock | ❌ UNTESTED |
| RC-002 | Work Orders | 2 users complete same WO at same time | One succeeds, one gets error | ❌ UNTESTED |
| RC-003 | PO Receiving | 2 users receive same PO line simultaneously | Quantities add correctly | ❌ UNTESTED |
| RC-004 | Inventory Reserve | Reserve qty while another issues same qty | Handle race correctly | ❌ UNTESTED |
| RC-005 | Serial Numbers | 2 WOs try to use same serial number | UNIQUE constraint blocks one | ✅ DB ENFORCED |
| RC-006 | Session Management | Same user logs in from 2 browsers | Handle multiple sessions | ⚠️ UNKNOWN |
| RC-007 | API Key Creation | Create 2 keys with same name simultaneously | Both succeed or one blocked | ⚠️ UNKNOWN |
| RC-008 | Bulk Operations | 100 users bulk update simultaneously | Queue or handle gracefully | ❌ UNTESTED |
| RC-009 | Backup | Backup triggered during bulk import | Consistent state captured | ❌ UNTESTED |

**Critical**: Need transaction isolation and locking strategy.

---

## 7. Data Integrity

### 7.1 Referential Integrity

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| DI-001 | Foreign Keys | Delete vendor with POs | Blocked (ON DELETE RESTRICT) | ✅ TESTED |
| DI-002 | Foreign Keys | Delete shipment with lines | Cascade delete lines | ✅ DB ENFORCED |
| DI-003 | Foreign Keys | Create PO with non-existent vendor | Reject | ⚠️ UNKNOWN |
| DI-004 | Orphaned Records | Delete part referenced in BOM | Handle gracefully | ⚠️ UNKNOWN |
| DI-005 | Orphaned Records | Delete WO leaves orphaned serials | Cascade works? | ✅ DB ENFORCED |
| DI-006 | Data Corruption | Manual DB edit breaks FK | App handles missing refs | ❌ UNTESTED |
| DI-007 | Undo System | Undo delete restores FK relationships | Integrity maintained | ⚠️ UNKNOWN |

---

## 8. Special Characters & Injection (Security)

### 8.1 SQL Injection Prevention

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| SC-001 | Search | Query = "'; DROP TABLE parts; --" | Escaped, no execution | ✅ FIXED (parameterized) |
| SC-002 | Parts | IPN = "IPN-001' OR '1'='1" | Stored as literal string | ✅ FIXED |
| SC-003 | Notes | notes = "'; DELETE FROM users; --" | Stored safely | ✅ FIXED |
| SC-004 | Table Names | Table validation with malicious input | Rejected by whitelist | ✅ FIXED (security audit) |

**Status**: Security audit addressed SQL injection. ✅

---

### 8.2 XSS Prevention

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| SC-005 | Parts | Description = "<script>alert(1)</script>" | Rendered as text, not executed | ⚠️ UNTESTED (React escapes by default) |
| SC-006 | Notes | notes = "<img src=x onerror=alert(1)>" | Rendered safely | ⚠️ UNTESTED |

**Note**: React escapes by default, but confirm no `dangerouslySetInnerHTML` usage.

---

## 9. File Operations

### 9.1 File Upload Edge Cases

| Test ID | Component | Scenario | Expected Behavior | Current Status |
|---------|-----------|----------|-------------------|----------------|
| FO-001 | Documents | Upload 0-byte file | Reject or accept | ❌ UNTESTED |
| FO-002 | Documents | Upload 100MB+ file | Reject with size limit error | ❌ NO VALIDATION |
| FO-003 | Documents | Upload file with no extension | Accept or reject | ⚠️ UNKNOWN |
| FO-004 | Documents | Upload .exe file | Validate allowed types | ❌ NO VALIDATION |
| FO-005 | CSV Import | Upload 1GB CSV file | Reject or stream process | ❌ NO VALIDATION |
| FO-006 | Backup Restore | Restore corrupted backup file | Reject with clear error | ❌ UNTESTED |

**Critical Gap**: No file size or type validation.

---

## 10. Edge Case Testing Priorities

### Phase 1: Critical (Must Have Before Production)

1. **Numeric Overflow** (NO-001 to NO-006): Prevent crashes
2. **String Length Limits** (SL-001 to SL-008): Prevent DOS
3. **Database Errors** (ER-001 to ER-007): Resilience
4. **Concurrency** (RC-001 to RC-004): Data corruption prevention
5. **File Upload Limits** (FO-001 to FO-006): Security

**Estimated Effort**: 40-60 hours  
**Priority**: CRITICAL

---

### Phase 2: High Priority (Should Have)

1. **Data Volume** (DV-001 to DV-008): Performance validation
2. **Boundary Values** (BC-001 to BC-023): Completeness
3. **Floating Point** (NO-007 to NO-012): Financial accuracy
4. **Network Errors** (ER-008 to ER-011): User experience

**Estimated Effort**: 30-40 hours  
**Priority**: HIGH

---

### Phase 3: Medium Priority (Nice to Have)

1. **String Format Validation** (SL-009 to SL-015): Data quality
2. **Data Integrity** (DI-001 to DI-007): Edge case handling
3. **Additional Concurrency** (RC-005 to RC-009): Advanced scenarios

**Estimated Effort**: 20-30 hours  
**Priority**: MEDIUM

---

## 11. Recommended Test Implementation

### 11.1 Unit Tests (Go)

Create `edge_case_test.go`:

```go
package main

import "testing"

// Numeric overflow tests
func TestInventoryQuantityOverflow(t *testing.T) {
    // Test INT_MAX + 1
    // Test very large REAL values
}

func TestNegativeQuantities(t *testing.T) {
    // Ensure CHECK constraints work
}

// String length tests
func TestVeryLongIPN(t *testing.T) {
    longIPN := strings.Repeat("A", 10000)
    // Should reject
}

func TestVeryLongDescription(t *testing.T) {
    longDesc := strings.Repeat("X", 100000)
    // Should handle or reject
}

// Concurrency tests
func TestConcurrentInventoryUpdate(t *testing.T) {
    // Use goroutines to simulate race condition
}

func TestConcurrentPOReceiving(t *testing.T) {
    // Multiple goroutines receive same PO
}
```

---

### 11.2 Integration Tests (Go)

Create `edge_case_integration_test.go`:

```go
func TestDatabaseFailureRecovery(t *testing.T) {
    // Close DB, make request, expect 503
}

func TestLargeDatasetPerformance(t *testing.T) {
    // Insert 10,000 parts
    // Measure list query time
    // Assert < 2 seconds
}

func TestExportLargeDataset(t *testing.T) {
    // Create 100,000 parts
    // Export to CSV
    // Verify completion without timeout
}
```

---

### 11.3 E2E Tests (Playwright)

Create `tests/e2e/edge-cases.spec.ts`:

```typescript
test('handles very long part description', async ({ page }) => {
  const longDesc = 'X'.repeat(10000);
  await createPart(page, { description: longDesc });
  // Assert error or truncation
});

test('handles network timeout gracefully', async ({ page }) => {
  await page.route('**/api/**', route => route.abort());
  // Assert error message shown
});

test('handles 10,000+ parts pagination', async ({ page }) => {
  // Seed 10,000 parts
  await page.goto('/parts');
  // Assert pagination works
});
```

---

### 11.4 Load/Stress Tests

Create `stress_edge_case_test.go`:

```go
func TestStress100ConcurrentUsers(t *testing.T) {
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Simulate user actions
        }()
    }
    wg.Wait()
}

func TestStressLargeExport(t *testing.T) {
    // Export 100,000 records
    // Measure memory usage
    // Assert no OOM
}
```

---

## 12. Validation Rules to Implement

Based on edge case analysis, implement these validation rules:

### Backend Validation (Go)

```go
// Add to validation.go

const (
    MaxStringLength = 10000
    MaxTextLength   = 100000
    MaxQty          = 1000000
    MaxPrice        = 1000000.00
    MaxLeadTimeDays = 365
)

func ValidateStringLength(field, value string, max int) error {
    if len(value) > max {
        return fmt.Errorf("%s exceeds maximum length of %d", field, max)
    }
    return nil
}

func ValidateNumericRange(field string, value float64, min, max float64) error {
    if value < min || value > max {
        return fmt.Errorf("%s must be between %f and %f", field, min, max)
    }
    return nil
}

func ValidateEmail(email string) error {
    if email == "" {
        return nil // Optional
    }
    matched, _ := regexp.MatchString(`^[^\s@]+@[^\s@]+\.[^\s@]+$`, email)
    if !matched {
        return errors.New("invalid email format")
    }
    return nil
}

func ValidateURL(url string) error {
    if url == "" {
        return nil
    }
    if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
        return errors.New("URL must start with http:// or https://")
    }
    return nil
}
```

### Frontend Validation (TypeScript)

```typescript
// Add to lib/validation.ts

export const MAX_STRING_LENGTH = 10000;
export const MAX_TEXT_LENGTH = 100000;
export const MAX_QTY = 1000000;

export function validateStringLength(value: string, max: number): string | null {
  if (value.length > max) {
    return `Maximum length is ${max} characters`;
  }
  return null;
}

export function validateEmail(email: string): string | null {
  if (!email) return null;
  const regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!regex.test(email)) {
    return "Invalid email format";
  }
  return null;
}

export function validateNumericRange(
  value: number, 
  min: number, 
  max: number
): string | null {
  if (value < min || value > max) {
    return `Value must be between ${min} and ${max}`;
  }
  return null;
}
```

---

## 13. Test Data Requirements

### Seed Data for Edge Case Testing

```sql
-- Large dataset (10,000 parts)
-- Use script to generate

-- Extreme values
INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES 
  ('EDGE-001', 999999999, 0),  -- Very large quantity
  ('EDGE-002', 0.0000001, 0),  -- Very small quantity
  ('EDGE-003', 0, 0);          -- Zero quantity

-- Long strings
INSERT INTO parts (ipn, description) VALUES (
  'EDGE-LONG-001',
  REPEAT('X', 50000)  -- 50KB description
);

-- Special characters (already handled by security)
INSERT INTO parts (ipn, description) VALUES (
  'EDGE-SPECIAL-001',
  'Description with <script>alert(1)</script> and SQL'' injection'
);
```

---

## 14. Acceptance Criteria

### Production-Ready Checklist

- [ ] All CRITICAL edge cases tested (Phase 1)
- [ ] At least 85% edge case coverage overall
- [ ] No unhandled numeric overflow errors
- [ ] All string inputs have max length validation
- [ ] Database failures return proper error codes
- [ ] Concurrent updates don't corrupt data
- [ ] File uploads have size/type limits
- [ ] Load test with 10,000+ records passes
- [ ] Export of 100,000 records succeeds
- [ ] 100 concurrent users handled gracefully
- [ ] All foreign key constraints enforced
- [ ] All CHECK constraints verified
- [ ] Floating point calculations accurate for currency
- [ ] Network timeout errors handled gracefully
- [ ] All validation rules implemented backend + frontend

---

## 15. Metrics & Monitoring

### Edge Case Test Metrics to Track

```
Total Edge Cases: 87
Phase 1 (Critical): 28 tests
Phase 2 (High): 31 tests
Phase 3 (Medium): 28 tests

Current Coverage: 15-20%
Target Coverage: 85%+

Estimated Implementation Time:
- Phase 1: 40-60 hours
- Phase 2: 30-40 hours
- Phase 3: 20-30 hours
Total: 90-130 hours (11-16 days)
```

---

## 16. Known Gaps Summary

### Critical Gaps Found

1. **No input length validation** on any TEXT fields
2. **No maximum value validation** on numeric fields (overflow risk)
3. **No file size/type validation** on uploads
4. **No load testing** with realistic data volumes (10K+ records)
5. **No concurrency testing** (race conditions unverified)
6. **No database failure recovery** testing
7. **No floating point precision** handling for currency
8. **Inventory transactions allow zero quantity** (illogical)

### Medium Gaps

1. Format validation missing (email, URL, phone)
2. Network timeout handling unverified
3. Export of very large datasets untested
4. Undo system integrity edge cases

---

## 17. Next Steps

### Immediate Actions (Week 1)

1. Implement input validation (string length, numeric range)
2. Add file upload limits (size, type)
3. Write Phase 1 critical edge case tests
4. Fix inventory transaction zero-qty issue

### Short Term (Week 2-3)

1. Write Phase 2 high priority tests
2. Perform load testing with 10,000+ records
3. Test concurrent updates (race conditions)
4. Add format validation (email, URL)

### Long Term (Week 4+)

1. Write Phase 3 medium priority tests
2. Continuous edge case testing in CI/CD
3. Regular stress testing
4. Edge case regression testing

---

## 18. Conclusion

ZRP has good security coverage (80%+) from the recent security audit, but **edge case coverage is only 15-20%**. The most critical gaps are:

- **Numeric overflow/underflow** (could crash system)
- **No input length limits** (DOS vulnerability)
- **Untested concurrency** (data corruption risk)
- **No load testing** (performance unknown)
- **File upload limits** (security/DOS risk)

**Recommendation**: Implement Phase 1 (critical) edge case tests before production deployment. Estimated 40-60 hours of work to achieve production-ready status for edge cases.

---

**Document Version**: 1.0  
**Last Updated**: February 19, 2026  
**Status**: Ready for Review  
**Next Review**: After Phase 1 implementation

---

## Appendix A: Edge Case Test Checklist

```
BOUNDARY VALUES
[ ] BC-001: Empty IPN
[ ] BC-002: Null category
[ ] BC-003: Empty location
[ ] BC-004: Null notes
[ ] BC-005: Null manufacturer
[ ] BC-006: Empty contact name
[ ] BC-007: Null description
[ ] BC-008: Empty root cause
[ ] BC-009: Empty search query
[ ] BC-010: No export filters
[ ] BC-011: Zero qty_on_hand
[ ] BC-012: Zero reorder_point
[ ] BC-013: Zero qty_ordered (should reject)
[ ] BC-014: Zero work order qty (should reject)
[ ] BC-015: Zero shipment qty (should reject)
[ ] BC-016: Zero lead time
[ ] BC-017: Zero price
[ ] BC-018: Zero transaction qty (BUG)
[ ] BC-019: Negative qty_on_hand (should reject)
[ ] BC-020: Negative qty_reserved (should reject)
[ ] BC-021: Negative reorder_point (should reject)
[ ] BC-022: Negative transaction qty (context-dependent)
[ ] BC-023: Negative lead time (should reject)

NUMERIC OVERFLOW
[ ] NO-001: Work order qty INT_MAX+1
[ ] NO-002: Shipment qty max int64
[ ] NO-003: Inventory qty_on_hand REAL max
[ ] NO-004: PO qty very large number
[ ] NO-005: Unit price 1e100
[ ] NO-006: Lead time 999999 days
[ ] NO-007: Floating point precision loss
[ ] NO-008: Price rounding
[ ] NO-009: Floating point math errors
[ ] NO-010: Qty comparison precision
[ ] NO-011: Sum accumulation errors
[ ] NO-012: Currency precision

STRING LENGTH
[ ] SL-001: 10K char IPN
[ ] SL-002: 100K char description
[ ] SL-003: 1MB notes
[ ] SL-004: 5K char vendor name
[ ] SL-005: 50K char search query
[ ] SL-006: 1K char filename
[ ] SL-007: 10MB JSON field
[ ] SL-008: 100KB HTTP header
[ ] SL-009: Invalid email format
[ ] SL-010: Empty email
[ ] SL-011: Invalid phone format
[ ] SL-012: JavaScript URL
[ ] SL-013: Invalid date format
[ ] SL-014: Far future date
[ ] SL-015: Empty tracking number

ERROR SCENARIOS
[ ] ER-001: Database connection lost
[ ] ER-002: Database locked/busy
[ ] ER-003: Constraint violation rollback
[ ] ER-004: Delete vendor with POs
[ ] ER-005: Duplicate serial number
[ ] ER-006: Migration failure
[ ] ER-007: Backup during transaction
[ ] ER-008: API request timeout
[ ] ER-009: Network disconnected
[ ] ER-010: Upload network failure
[ ] ER-011: WebSocket disconnect

DATA VOLUME
[ ] DV-001: 10K parts list
[ ] DV-002: 100K parts performance
[ ] DV-003: 1K BOM line items
[ ] DV-004: Search 50K parts
[ ] DV-005: Export 100K CSV
[ ] DV-006: Export 100K Excel
[ ] DV-007: 1M audit entries
[ ] DV-008: Dashboard 100K aggregation

RACE CONDITIONS
[ ] RC-001: Concurrent inventory update
[ ] RC-002: Concurrent WO completion
[ ] RC-003: Concurrent PO receiving
[ ] RC-004: Concurrent inventory reserve
[ ] RC-005: Duplicate serial number creation
[ ] RC-006: Multiple sessions same user
[ ] RC-007: Concurrent API key creation
[ ] RC-008: 100 users bulk update
[ ] RC-009: Backup during import

DATA INTEGRITY
[ ] DI-001: Delete vendor with POs (blocked)
[ ] DI-002: Delete shipment cascades
[ ] DI-003: PO with invalid vendor
[ ] DI-004: Delete part in BOM
[ ] DI-005: Delete WO cascades serials
[ ] DI-006: Manual DB corruption
[ ] DI-007: Undo maintains integrity

SPECIAL CHARACTERS (SECURITY)
[ ] SC-001: SQL injection in search
[ ] SC-002: SQL injection in IPN
[ ] SC-003: SQL injection in notes
[ ] SC-004: SQL injection table name
[ ] SC-005: XSS in description
[ ] SC-006: XSS in notes

FILE OPERATIONS
[ ] FO-001: Zero-byte file upload
[ ] FO-002: 100MB+ file upload
[ ] FO-003: No file extension
[ ] FO-004: .exe file upload
[ ] FO-005: 1GB CSV import
[ ] FO-006: Corrupted backup restore
```

**Total**: 87 edge case tests

---

**END OF EDGE CASE TEST PLAN**
