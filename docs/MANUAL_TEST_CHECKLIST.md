# ZRP Manual Testing Checklist

**Generated**: 2026-02-19  
**Purpose**: Step-by-step manual test scenarios for features not covered by automated tests  
**Use Case**: Acceptance testing, regression verification, pre-release validation

---

## How to Use This Checklist

1. **Start with Critical Features** — test 🔴 items first
2. **Create Test Data** — use the data setup instructions for each scenario
3. **Check Expected Results** — verify both success cases and error handling
4. **Document Issues** — note any deviations from expected behavior
5. **Retest Fixes** — use this checklist to verify bug fixes

---

## Critical Features (🔴 High Risk)

### TC-001: BOM Cost Rollup Calculation

**Priority**: 🔴 Critical  
**Risk**: Incorrect costing leads to bad quotes and financial loss  
**Frequency**: Run before every release

#### Prerequisites
- At least 3 parts in gitplm: resistor (RES-001), capacitor (CAP-001), assembly (ASY-001)
- ASY-001 BOM includes RES-001 (qty 10) and CAP-001 (qty 5)
- PO received for RES-001 @ $0.10/unit
- PO received for CAP-001 @ $0.50/unit

#### Test Steps
1. Navigate to Parts → Search for ASY-001
2. Click on ASY-001 to open detail modal
3. Click "View BOM" tab
4. Click "Cost Analysis" button

#### Expected Results
- ✅ Total BOM cost = (10 × $0.10) + (5 × $0.50) = $3.50
- ✅ Per-unit cost breakdown shown for each component
- ✅ If a component has no price history, shows "No cost data" instead of $0
- ✅ Multi-level BOM (assembly within assembly) calculates recursively

#### Error Cases to Test
- BOM with circular reference (ASY-001 contains ASY-001) → should show error, not crash
- BOM component with no price history → shows warning, continues calculation
- Very deep BOM tree (5+ levels) → calculates correctly without timeout

---

### TC-002: Work Order Kitting (Inventory Reservation)

**Priority**: 🔴 Critical  
**Risk**: Double-allocating inventory to multiple work orders  
**Frequency**: Every release

#### Prerequisites
- Part RES-001 exists
- Inventory for RES-001: qty_on_hand = 100, qty_reserved = 0, reorder_point = 20
- Two work orders created:
  - WO-001: ASY-001 qty 5 (requires 50 units of RES-001)
  - WO-002: ASY-001 qty 3 (requires 30 units of RES-001)

#### Test Steps
1. Open WO-001 detail
2. Click "Kit Materials" button
3. Confirm kitting action
4. Verify inventory update
5. Open WO-002 detail
6. Click "Kit Materials" button
7. Verify inventory reservation logic

#### Expected Results
- ✅ After kitting WO-001:
  - RES-001 qty_on_hand = 100 (unchanged)
  - RES-001 qty_reserved = 50
  - WO-001 status shows "Kitted" or "Ready"
- ✅ After kitting WO-002:
  - RES-001 qty_on_hand = 100
  - RES-001 qty_reserved = 80
  - WO-002 status shows "Kitted"
- ✅ Attempting to kit WO requiring 30 more units (total 110) → shows error "Insufficient inventory (available: 20)"

#### Error Cases
- Kit materials when insufficient stock → error message, no partial reservation
- Kit materials twice on same WO → error "Already kitted" or idempotent (no double-reserve)
- Delete work order → unreserve inventory automatically

---

### TC-003: ECO Part Revision Cascade

**Priority**: 🔴 Critical  
**Risk**: Lost change history, incorrect part data  
**Frequency**: Every release

#### Prerequisites
- Part RES-001 rev A exists in gitplm
- ECO-001 created with affected IPN: RES-001
- ECO-001 status: draft

#### Test Steps
1. Open ECO-001 detail
2. Click "Approve" button (confirm in modal)
3. Verify approval recorded
4. Click "Implement" button
5. Confirm implementation
6. Check if part revision updated

#### Expected Results
- ✅ After approval:
  - ECO-001 status = "approved"
  - `approved_by` = current user
  - `approved_at` = current timestamp
  - Email sent to ECO creator (if subscribed)
- ✅ After implementation:
  - ECO-001 status = "implemented"
  - If gitplm sync enabled, RES-001 revision → B
  - Part change history shows ECO-001 reference
  - All documents referencing RES-001 rev A get notification/warning
  - BOM using RES-001 shows "ECO pending" or revision mismatch warning

#### Error Cases
- Implement ECO without approval → error "Must be approved first"
- Implement ECO with invalid IPNs → shows warning, continues for valid ones
- Concurrent ECO implementation on same part → second one waits or errors

---

### TC-004: Low Stock Alert Generation & Deduplication

**Priority**: 🔴 Critical  
**Risk**: Missed reorder → stockout / Alert spam → ignored warnings  
**Frequency**: Every release

#### Prerequisites
- Part RES-001: qty_on_hand = 25, reorder_point = 20
- Part CAP-001: qty_on_hand = 100, reorder_point = 50
- No existing low stock notifications in last 24 hours

#### Test Steps
1. Create inventory transaction: issue 10 units of RES-001
2. Wait 5 seconds (for background notification goroutine)
3. Check notifications bell icon
4. Verify notification content
5. Create another issue transaction: issue 5 more units of RES-001
6. Check if duplicate notification appears

#### Expected Results
- ✅ After first transaction:
  - New notification: "Low stock alert: RES-001 (qty: 15, reorder: 20)"
  - Notification severity: Warning
  - Dashboard "Low Stock" KPI increments by 1
- ✅ After second transaction:
  - No new notification (deduplicated)
  - Existing notification updates qty if real-time enabled
- ✅ After 24 hours, another low stock transaction → new notification allowed

#### Error Cases
- Transaction bringing qty from below reorder to above reorder → notification auto-clears or marks resolved
- Multiple parts drop below reorder simultaneously → all get notifications
- Notification goroutine crash → logs error, doesn't affect main app

---

### TC-005: Purchase Order → Inventory Auto-Update

**Priority**: 🔴 Critical  
**Risk**: Inventory mismatch, lost receiving records  
**Frequency**: Every release

#### Prerequisites
- Vendor V-001 exists
- Part RES-001 exists
- Inventory for RES-001: qty_on_hand = 10
- PO-001 created with line: RES-001 qty 100 @ $0.10/unit

#### Test Steps
1. Open PO-001 detail
2. Click "Receive Shipment" button
3. Enter received quantities:
   - RES-001: 100 (full shipment)
4. Confirm receipt
5. Navigate to Inventory → find RES-001
6. Check transaction history

#### Expected Results
- ✅ After receiving:
  - RES-001 qty_on_hand = 110 (10 + 100)
  - Inventory transaction created:
    - Type: "receive"
    - Qty: +100
    - Reference: "PO-001"
    - Unit price: $0.10
  - Supplier price recorded in price history:
    - Vendor: V-001
    - Unit price: $0.10
    - Date: today
  - PO-001 status → "received"
  - Email notification sent (if configured)

#### Partial Receive Test
1. Create PO-002 with RES-001 qty 50
2. Receive partial: RES-001 qty 30
3. Verify:
   - Inventory +30
   - PO-002 status = "partial"
   - Line item shows 30/50 received
4. Receive remaining 20
5. Verify PO-002 status → "received"

#### Error Cases
- Receive qty > ordered qty → warning "Over-receive: confirm?" or error
- Receive negative qty → error "Invalid quantity"
- Concurrent receives on same PO → second one errors or merges

---

### TC-006: Serial Number Auto-Generation

**Priority**: 🔴 Critical  
**Risk**: Duplicate serials, untracked devices  
**Frequency**: Every release

#### Prerequisites
- Assembly ASY-001 exists
- Work order WO-001 created for ASY-001 qty 10
- WO-001 status: "in_progress"

#### Test Steps
1. Open WO-001 detail
2. Click "Serials" tab
3. Click "Add Serial" button (without entering serial number)
4. Verify auto-generated serial
5. Add another serial (auto-generated)
6. Manually enter serial "TEST-001"
7. Try to add duplicate serial "TEST-001"

#### Expected Results
- ✅ First auto-generated serial:
  - Format: ASY001-YYYYMMDD-001 (or similar pattern)
  - Status: "assigned"
  - Linked to WO-001
- ✅ Second auto-generated serial:
  - Format: ASY001-YYYYMMDD-002 (increments)
  - No collision with existing serials
- ✅ Manual serial "TEST-001":
  - Accepted if unique
  - Linked to WO-001
- ✅ Duplicate serial attempt:
  - Error: "Serial number already exists"
  - No database write

#### Error Cases
- Auto-generate 1000 serials rapidly → no duplicates, no collisions
- Two users add serial simultaneously → one succeeds, other gets error
- Delete work order → serials remain but status → "orphaned" or deleted (configurable)

---

### TC-007: Notification Deduplication Logic

**Priority**: 🔴 Critical  
**Risk**: User gets 100 identical alerts  
**Frequency**: Every release

#### Prerequisites
- Part RES-001: qty_on_hand = 15, reorder_point = 20
- No notifications exist

#### Test Steps
1. Create 5 inventory transactions rapidly (issue 1 unit each)
2. Wait for notification goroutine (5 sec)
3. Check notification count
4. Wait 23 hours, 59 minutes
5. Issue 1 more unit
6. Check notification count
7. Wait 2 more minutes (24 hours total)
8. Issue 1 more unit
9. Check notification count

#### Expected Results
- ✅ After 5 transactions: **1 notification** (deduplicated)
- ✅ After transaction at 23h59m: **still 1 notification** (within 24h window)
- ✅ After transaction at 24h01m: **2 notifications total** (new 24h window started)

#### Deduplication Key
- Same type (low_stock) + same IPN + within 24 hours → deduplicate
- Different type (overdue_wo) + same WO ID + within 24 hours → deduplicate

---

### TC-008: Audit Log Writes for All CRUD Operations

**Priority**: 🔴 Critical  
**Risk**: Missing accountability, compliance failure  
**Frequency**: Spot-check each release

#### Test Steps
1. Login as user "alice"
2. Navigate to Parts → Create part RES-999
3. Navigate to Audit Log
4. Verify create entry exists
5. Edit RES-999 (change description)
6. Refresh audit log
7. Verify update entry exists
8. Delete RES-999
9. Verify delete entry exists
10. Logout, login as "bob"
11. Bulk delete 3 ECOs
12. Verify bulk action logged with count

#### Expected Results
- ✅ Create audit entry:
  - Module: "parts"
  - Action: "create"
  - User: "alice"
  - Record ID: RES-999
  - Summary: "Created part RES-999"
  - Timestamp: accurate
- ✅ Update audit entry:
  - Action: "update"
  - Summary includes changed field ("description")
- ✅ Delete audit entry:
  - Action: "delete"
  - Summary: "Deleted part RES-999"
- ✅ Bulk delete:
  - Action: "bulk_delete"
  - User: "bob"
  - Summary: "Deleted 3 ECOs"

#### Error Cases
- Audit write fails → logs error, doesn't block operation
- User deletes audit log entries → prevented (read-only)

---

### TC-009: Session Expiration

**Priority**: 🔴 Critical  
**Risk**: Session hijacking, unauthorized access  
**Frequency**: Every release

#### Prerequisites
- Session timeout configured to 24 hours
- User logged in

#### Test Steps
1. Login as user "alice" at 10:00 AM
2. Note session cookie value
3. Make API call at 10:30 AM (within session) → should succeed
4. Wait 25 hours (or change system time)
5. Make API call at 11:01 AM next day (session expired) → should fail
6. Verify redirect to login page

#### Expected Results
- ✅ Active session (within 24h): API calls return 200 OK
- ✅ Expired session (>24h): API calls return 401 Unauthorized
- ✅ Frontend detects 401, redirects to `/login`
- ✅ After re-login, new session cookie issued

#### Error Cases
- Session cookie tampered with → 401 Unauthorized
- Session cookie from different domain → rejected

---

### TC-010: Email Notification Delivery

**Priority**: 🔴 Critical  
**Risk**: Critical alerts missed  
**Frequency**: Every release (requires SMTP config)

#### Prerequisites
- Email settings configured (SMTP host, port, credentials)
- Email enabled = true
- User "alice" email = alice@example.com
- User subscribed to ECO notifications

#### Test Steps
1. Login as "alice"
2. Create ECO-TEST
3. Approve ECO-TEST
4. Check email inbox (alice@example.com)
5. Check Email Log in ZRP (Admin → Email Settings → Email Log)

#### Expected Results
- ✅ Email received within 30 seconds:
  - Subject: "ECO Approved: ECO-TEST"
  - Body includes ECO ID, title, who approved, when
  - "View ECO" link works
- ✅ Email log entry:
  - Status: "sent"
  - Recipient: alice@example.com
  - Sent at: timestamp
  - Event type: "eco_approved"

#### Error Cases
- SMTP server unreachable → email log shows "failed", error message logged
- Invalid recipient email → email log shows "failed: invalid address"
- User unsubscribed from ECO notifications → no email sent, log shows "skipped: unsubscribed"

---

## High Priority Features (🟡)

### TC-011: IPN Autocomplete

**Priority**: 🟡 High  
**Test Frequency**: Each release

#### Test Steps
1. Navigate to Work Orders → Create New
2. In "Assembly IPN" field, type "RES"
3. Observe dropdown suggestions
4. Select RES-001 from dropdown
5. Verify IPN field populated correctly

#### Expected Results
- ✅ Typing "RES" shows parts with IPN starting with "RES" (RES-001, RES-002, etc.)
- ✅ Shows max 10 suggestions
- ✅ Selecting suggestion populates field
- ✅ Shows part description in suggestion (e.g., "RES-001 - 10k Resistor")

---

### TC-012: NCR → ECO Auto-Link

**Priority**: 🟡 High  
**Test Frequency**: Each release

#### Test Steps
1. Create NCR-001 describing defect in RES-001
2. From NCR-001 detail modal, click "Create ECO"
3. Verify ECO pre-populated with NCR data
4. Save ECO
5. Navigate back to NCR-001
6. Verify ECO link shown

#### Expected Results
- ✅ ECO creation modal pre-filled:
  - Title: "ECO from NCR-001: [NCR title]"
  - Description: includes NCR description
  - Affected IPNs: includes NCR's affected part
  - ncr_id field: NCR-001
- ✅ NCR detail shows "Related ECO: ECO-XXX" link
- ✅ Clicking link navigates to ECO

---

### TC-013: Work Order PDF Traveler

**Priority**: 🟡 High  
**Test Frequency**: Spot-check

#### Test Steps
1. Create WO-TEST for ASY-001 qty 5
2. Open WO-TEST detail
3. Click "Print Traveler" button
4. Verify PDF opens in new tab
5. Check PDF content

#### Expected Results
- ✅ PDF contains:
  - Work Order ID, Assembly IPN, Quantity
  - Full BOM table (IPN, Desc, MPN, Manufacturer, Qty, Ref Des)
  - Sign-off section (Kitted by, Built by, Tested by, QA)
  - Company logo/header (if configured)
- ✅ PDF is printable (Ctrl+P works)
- ✅ File downloads with name "WO-TEST-traveler.pdf"

---

### TC-014: Quote Margin Analysis

**Priority**: 🟡 High  
**Test Frequency**: Each release

#### Prerequisites
- Part RES-001: latest PO price $0.10
- Part CAP-001: latest PO price $0.50
- Quote Q-001 with lines:
  - RES-001 qty 100 @ $0.20 (unit price)
  - CAP-001 qty 50 @ $0.60

#### Test Steps
1. Open Quote Q-001 detail
2. Click "Margin Analysis" tab
3. Review margin calculations

#### Expected Results
- ✅ RES-001 line:
  - BOM cost: $0.10
  - Quoted price: $0.20
  - Margin per unit: $0.10
  - Margin %: 50% (green)
- ✅ CAP-001 line:
  - BOM cost: $0.50
  - Quoted price: $0.60
  - Margin per unit: $0.10
  - Margin %: 16.7% (red, <20%)
- ✅ Summary:
  - Total quoted: (100×$0.20 + 50×$0.60) = $50
  - Total BOM cost: (100×$0.10 + 50×$0.50) = $35
  - Total margin: $15
  - Overall margin %: 30% (yellow, 20-50%)

---

### TC-015: Batch Operations (Bulk Approve ECOs)

**Priority**: 🟡 High  
**Test Frequency**: Spot-check

#### Test Steps
1. Create 5 ECOs (ECO-001 to ECO-005), all status "review"
2. Navigate to ECOs list
3. Select checkboxes for ECO-001, ECO-003, ECO-005
4. Click "Bulk Approve" button
5. Confirm action
6. Refresh list

#### Expected Results
- ✅ ECO-001, ECO-003, ECO-005 status = "approved"
- ✅ ECO-002, ECO-004 status = "review" (unchanged)
- ✅ Audit log shows 3 separate "approve" entries or 1 "bulk_approve" entry
- ✅ Emails sent to creators of approved ECOs (if subscribed)
- ✅ Success message: "3 ECOs approved"

#### Error Cases
- Select ECO already approved → skipped, no error
- Select ECO in "draft" → error "Cannot approve draft ECO"

---

### TC-016: Calendar Event Aggregation

**Priority**: 🟡 High  
**Test Frequency**: Each release

#### Prerequisites
- Work Order WO-001 due date: 2026-02-25
- Purchase Order PO-001 expected delivery: 2026-02-25
- Quote Q-001 valid until: 2026-02-28

#### Test Steps
1. Navigate to Calendar
2. View February 2026
3. Click on day 25
4. Verify events listed

#### Expected Results
- ✅ Feb 25 shows:
  - WO-001 (blue badge)
  - PO-001 (green badge)
- ✅ Feb 28 shows:
  - Q-001 (orange badge)
- ✅ Clicking WO-001 event navigates to WO-001 detail
- ✅ Hovering over event shows tooltip with details

#### Error Cases
- Event with no date → not shown on calendar
- Event date in past → shown with "overdue" styling

---

### TC-017: Advanced Search Filters

**Priority**: 🟡 High  
**Test Frequency**: New feature, test thoroughly

#### Test Steps
1. Navigate to Advanced Search (if UI exists)
2. Select module: "Parts"
3. Add filters:
   - Category = "Resistors"
   - Field "value" contains "10k"
   - Field "package" = "0603"
4. Click Search
5. Verify results

#### Expected Results
- ✅ Results include only parts matching all filters
- ✅ Results show part IPN, description, category
- ✅ Clicking result navigates to part detail
- ✅ "Clear filters" button resets form

---

### TC-018: Reports - Inventory Valuation

**Priority**: 🟡 High  
**Test Frequency**: Each release

#### Prerequisites
- Inventory:
  - RES-001: qty 100 @ latest PO price $0.10
  - CAP-001: qty 50 @ latest PO price $0.50

#### Test Steps
1. Navigate to Reports → Inventory Valuation
2. Run report
3. Check calculations
4. Click "Export CSV"
5. Verify CSV download

#### Expected Results
- ✅ Report table shows:
  - RES-001: qty 100, unit price $0.10, value $10.00
  - CAP-001: qty 50, unit price $0.50, value $25.00
  - **Total inventory value: $35.00**
- ✅ CSV export contains same data
- ✅ CSV filename: "inventory-valuation-YYYY-MM-DD.csv"

---

### TC-019: GitPLM CSV Sync

**Priority**: 🟡 High  
**Risk**: Part data out of sync  
**Test Frequency**: Spot-check

#### Prerequisites
- GitPLM URL configured
- gitplm CSV file exists with 10 parts

#### Test Steps
1. Navigate to Settings → GitPLM
2. Click "Sync Now" button
3. Wait for sync completion
4. Verify success message
5. Navigate to Parts list
6. Verify parts loaded

#### Expected Results
- ✅ Sync success message: "Synced 10 parts"
- ✅ Parts list shows all 10 parts from CSV
- ✅ Part fields match CSV columns
- ✅ Sync timestamp updated

#### Error Cases
- GitPLM URL unreachable → error message, no partial sync
- CSV malformed → shows error, line number
- Duplicate IPNs in CSV → error or last-one-wins (document behavior)

---

## Medium Priority Features (🟢)

### TC-020: Dark Mode Toggle

**Priority**: 🟢 Low  
**Test Frequency**: Spot-check

#### Test Steps
1. Click theme toggle button (sun/moon icon)
2. Verify dark mode applied
3. Refresh page
4. Verify dark mode persists
5. Toggle back to light mode
6. Verify light mode applied

#### Expected Results
- ✅ Dark mode: dark background, light text
- ✅ Light mode: light background, dark text
- ✅ Preference saved in localStorage
- ✅ All UI elements readable in both modes

---

### TC-021: Widget Customization

**Priority**: 🟢 Low  
**Test Frequency**: Spot-check

#### Test Steps
1. Navigate to Dashboard
2. Click "Customize" button
3. Drag "Open ECOs" widget to bottom
4. Toggle "Low Stock" widget off
5. Click Save
6. Refresh page
7. Verify changes persist

#### Expected Results
- ✅ "Open ECOs" widget moved to bottom
- ✅ "Low Stock" widget hidden
- ✅ Changes persist after refresh
- ✅ Other users see default layout (per-user setting)

---

## Data Corruption & Race Condition Tests

### TC-022: Concurrent Inventory Updates

**Priority**: 🔴 Critical  
**Test Frequency**: Before major releases

#### Prerequisites
- Part RES-001: qty_on_hand = 100
- Two browser tabs open, both logged in as same user

#### Test Steps
1. Tab 1: Navigate to Inventory → Create transaction
   - IPN: RES-001
   - Type: issue
   - Qty: 50
   - Don't submit yet
2. Tab 2: Navigate to Inventory → Create transaction
   - IPN: RES-001
   - Type: issue
   - Qty: 60
   - Don't submit yet
3. Tab 1: Click Submit (transaction 1)
4. Tab 2: Immediately click Submit (transaction 2)
5. Check RES-001 qty_on_hand

#### Expected Results
- ✅ **Pessimistic Locking**: Tab 2 gets error "Inventory locked, please retry"
- ✅ **Optimistic Locking**: Both succeed, final qty = 100 - 50 - 60 = -10 → error on second transaction "Insufficient inventory"
- ✅ **Serialized**: Second transaction waits, then executes if sufficient qty remains

#### Unacceptable Result
- ❌ Final qty = 50 or 40 (lost update)
- ❌ Both transactions succeed, final qty = -10 (negative inventory, violates constraint)

---

### TC-023: BOM Circular Dependency Detection

**Priority**: 🔴 Critical  
**Test Frequency**: Before major releases

#### Test Steps
1. Create part ASY-001 (assembly)
2. Create part ASY-002 (assembly)
3. Add BOM line to ASY-001: includes ASY-002
4. Attempt to add BOM line to ASY-002: includes ASY-001

#### Expected Results
- ✅ Error when adding circular BOM: "Circular dependency detected: ASY-002 → ASY-001 → ASY-002"
- ✅ BOM not saved
- ✅ System remains stable (no infinite loop)

#### Multi-Level Circular Test
- ASY-001 → ASY-002 → ASY-003 → ASY-001
- Should detect and reject at ASY-003 BOM edit

---

## Acceptance Criteria Summary

### ✅ Pass Criteria for Release

- All 🔴 Critical tests pass (TC-001 through TC-010)
- At least 80% of 🟡 High Priority tests pass
- No data corruption in race condition tests (TC-022, TC-023)
- No security vulnerabilities (session expiration works, audit log complete)

### 🔴 Blocker Criteria (Do Not Release)

- Any 🔴 Critical test fails
- Data corruption detected (negative inventory, lost audit logs)
- Session hijacking possible
- Email delivery completely broken

### ⚠️ Warning (Release with Caution)

- Multiple 🟡 High Priority tests fail
- Performance degradation (BOM calculation timeout, slow search)
- UI bugs in low-traffic modules

---

## Test Data Setup Scripts

### Quick Test Data Setup

```bash
# Create test user
curl -X POST http://localhost:3000/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "Test123!", "role": "user"}'

# Create vendor
curl -X POST http://localhost:3000/api/v1/vendors \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Vendor", "code": "V-TEST", "lead_time_days": 7}'

# Create inventory item
curl -X POST http://localhost:3000/api/v1/inventory/transact \
  -H "Content-Type: application/json" \
  -d '{"ipn": "RES-001", "type": "receive", "qty": 100, "reference": "Initial stock"}'
```

---

## Test Environment Recommendations

### Dedicated Test Instance
- Separate database (not production!)
- Isolated file uploads directory
- Test SMTP server (mailtrap.io or similar)

### Test Data Reset
Before each test run:
```bash
# Backup current DB
cp zrp.db zrp_backup.db

# Restore clean test DB
cp zrp_test_clean.db zrp.db

# Restart server
./zrp
```

### Browser Setup
- Use incognito/private mode to avoid cached sessions
- Test in multiple browsers (Chrome, Firefox, Safari)
- Test mobile viewport (375px width)

---

**Document Status**: Ready for use  
**Maintainer**: QA Team  
**Next Review**: After each sprint/release
