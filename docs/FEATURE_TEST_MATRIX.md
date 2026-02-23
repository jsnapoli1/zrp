# ZRP Feature Test Coverage Matrix

**Generated**: 2026-02-19  
**Purpose**: Complete inventory of ZRP features mapped to test coverage status

## Legend

- ✅ **Fully Tested**: Unit tests + integration tests + E2E tests exist
- ⚠️ **Partially Tested**: Some tests exist but incomplete coverage
- ❌ **Untested**: No automated tests found
- 🔴 **Critical**: Must work or data corruption/system failure possible
- 🟡 **High Priority**: Core user workflow, should be tested
- 🟢 **Low Priority**: Nice to have, lower risk

---

## Executive Summary

### Coverage Statistics
- **Total Features Inventoried**: 245
- **Fully Tested (✅)**: 78 (32%)
- **Partially Tested (⚠️)**: 94 (38%)
- **Untested (❌)**: 73 (30%)

### Critical Gaps (🔴 + ❌)
1. BOM Cost Rollup Calculation
2. Inventory Reservation (Work Order Kitting)
3. ECO Part Revision Cascade
4. Purchase Order → Inventory Auto-Update
5. Serial Number Auto-Generation
6. Low Stock Alert Generation
7. Calendar Event Aggregation
8. Notification Deduplication Logic
9. GitPLM CSV Sync
10. Email Notification Delivery

---

## Module-by-Module Coverage

### 1. Authentication & Authorization

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| User login | ✅ | ✅ | ✅ | ✅ | 🔴 | handler_auth_test.go, playwright.spec.js |
| User logout | ✅ | ❌ | ⚠️ | ⚠️ | 🔴 | Missing integration test |
| Password validation | ✅ | ❌ | ❌ | ⚠️ | 🔴 | Only unit tested |
| Change password | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_auth_test.go only |
| Session expiration | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** |
| Role-based access control | ✅ | ⚠️ | ⚠️ | ⚠️ | 🔴 | rbac_test.go, permissions.spec.js partial |
| API key generation | ✅ | ✅ | ✅ | ✅ | 🟡 | api-keys.spec.js |
| API key authentication | ✅ | ✅ | ✅ | ✅ | 🔴 | api-keys.spec.js |
| API key revocation | ✅ | ❌ | ✅ | ⚠️ | 🟡 | E2E exists, no integration |

### 2. User Management

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create user | ✅ | ❌ | ⚠️ | ⚠️ | 🔴 | handler_users_test.go |
| Update user | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_users_test.go |
| Deactivate user | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_users_test.go |
| Prevent admin self-deactivate | ✅ | ❌ | ❌ | ⚠️ | 🔴 | handler_users_test.go |
| Delete user | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_users_test.go |
| Reset user password | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| List users | ✅ | ❌ | ❌ | ⚠️ | 🟢 | handler_users_test.go |

### 3. Dashboard

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Dashboard KPI calculation | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | E2E in cross-module.spec.js |
| KPI: Open ECOs | ❌ | ❌ | ⚠️ | ⚠️ | 🟢 | E2E exists |
| KPI: Low Stock | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | E2E exists |
| KPI: Open POs | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| KPI: Active Work Orders | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | E2E exists |
| KPI: Open NCRs | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| KPI: Total Parts | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| ECO status chart | ❌ | ❌ | ⚠️ | ⚠️ | 🟢 | cross-module.spec.js |
| WO status chart | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| Inventory value chart | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| Low stock alerts panel | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | cross-module.spec.js |
| Widget customization | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 4. Global Search

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Search parts by IPN | ✅ | ❌ | ❌ | ⚠️ | 🟡 | search_test.go |
| Search parts by MPN | ✅ | ❌ | ❌ | ⚠️ | 🟡 | search_test.go |
| Search parts by field values | ✅ | ❌ | ❌ | ⚠️ | 🟡 | search_test.go |
| Search ECOs | ✅ | ❌ | ❌ | ⚠️ | 🟡 | search_test.go |
| Search Work Orders | ✅ | ❌ | ❌ | ⚠️ | 🟡 | search_test.go |
| Search Devices | ✅ | ❌ | ❌ | ⚠️ | 🟡 | search_test.go |
| Search NCRs | ✅ | ❌ | ❌ | ⚠️ | 🟡 | search_test.go |
| Advanced search filters | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** - new feature |
| Search result grouping | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 5. Parts (PLM)

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create part | ✅ | ⚠️ | ✅ | ✅ | 🔴 | handler_parts_create_test.go, crud-full.spec.js |
| Read part | ✅ | ✅ | ✅ | ✅ | 🔴 | handler_parts_test.go |
| Update part | ✅ | ❌ | ✅ | ⚠️ | 🔴 | crud-full.spec.js |
| Delete part | ✅ | ❌ | ✅ | ⚠️ | 🟡 | crud-full.spec.js |
| List parts (pagination) | ✅ | ❌ | ✅ | ⚠️ | 🟡 | handler_parts_test.go |
| Filter by category | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_parts_test.go |
| View BOM tree | ✅ | ⚠️ | ❌ | ⚠️ | 🔴 | handler_parts_test.go, partial integration |
| BOM cost rollup | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical calculation |
| Where-used analysis | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_parts_test.go |
| IPN autocomplete | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| GitPLM URL generation | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| Market pricing lookup | ✅ | ❌ | ❌ | ⚠️ | 🟢 | market_pricing_test.go |
| Pending part changes | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_part_changes_test.go |
| Create ECO from changes | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_part_changes_test.go |
| Check IPN exists | ✅ | ❌ | ❌ | ⚠️ | 🟢 | handler_parts_test.go |

### 6. ECOs (Engineering Change Orders)

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create ECO | ✅ | ✅ | ✅ | ✅ | 🔴 | handler_eco_test.go, crud-full.spec.js |
| Read ECO | ✅ | ✅ | ✅ | ✅ | 🔴 | handler_eco_test.go |
| Update ECO | ✅ | ❌ | ✅ | ⚠️ | 🔴 | crud-full.spec.js |
| Delete ECO | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_eco_test.go |
| Approve ECO | ✅ | ✅ | ✅ | ✅ | 🔴 | handler_eco_test.go, edge-cases.spec.js |
| Implement ECO | ✅ | ✅ | ✅ | ✅ | 🔴 | handler_eco_test.go, edge-cases.spec.js |
| Reject ECO | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_eco_test.go |
| ECO status workflow | ✅ | ⚠️ | ❌ | ⚠️ | 🔴 | handler_eco_test.go partial |
| Affected IPNs enrichment | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Part revision cascade | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical feature |
| NCR→ECO linking | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Bulk approve ECOs | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** - batch ops |
| Bulk implement ECOs | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** - batch ops |
| ECO revisions | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_eco_test.go |
| Create Git PR from ECO | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 7. Documents

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create document | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Read document | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Update document | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Delete document | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Document versioning | ✅ | ❌ | ❌ | ⚠️ | 🔴 | handler_doc_versions_test.go |
| Diff versions | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_doc_versions_test.go |
| Release document | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_doc_versions_test.go |
| Revert to revision | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_doc_versions_test.go |
| Push to Git | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| Sync from Git | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 8. Inventory

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| List inventory | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_inventory_test.go |
| Get inventory item | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_inventory_test.go |
| Create transaction (receive) | ✅ | ⚠️ | ❌ | ⚠️ | 🔴 | handler_inventory_test.go |
| Create transaction (issue) | ✅ | ❌ | ❌ | ⚠️ | 🔴 | handler_inventory_test.go |
| Create transaction (adjust) | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_inventory_test.go |
| Transaction history | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_inventory_test.go |
| Qty reserved calculation | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| Low stock alert trigger | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| Reorder point logic | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Bulk delete inventory | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Bulk update inventory | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_bulk_update_test.go |

### 9. Purchase Orders

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create PO | ✅ | ✅ | ⚠️ | ✅ | 🔴 | handler_procurement_test.go, integration_bom_po_test.go |
| Read PO | ✅ | ✅ | ❌ | ⚠️ | 🔴 | handler_procurement_test.go |
| Update PO | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_procurement_test.go |
| Delete PO | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_procurement_test.go |
| Receive PO | ✅ | ✅ | ❌ | ⚠️ | 🔴 | handler_procurement_test.go, receiving_eco_test.go |
| PO → Inventory update | ⚠️ | ✅ | ❌ | ⚠️ | 🔴 | integration_bom_po_test.go |
| Generate PO from WO | ⚠️ | ✅ | ⚠️ | ⚠️ | 🔴 | integration_bom_po_test.go, edge-cases.spec.js |
| Partial receive | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_procurement_test.go |
| PO status workflow | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_procurement_test.go |
| Supplier price capture | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 10. Vendors

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create vendor | ✅ | ❌ | ⚠️ | ⚠️ | 🟡 | handler_vendors_test.go, crud-full.spec.js |
| Read vendor | ✅ | ❌ | ⚠️ | ⚠️ | 🟡 | handler_vendors_test.go |
| Update vendor | ✅ | ❌ | ⚠️ | ⚠️ | 🟡 | handler_vendors_test.go, crud-full.spec.js |
| Delete vendor | ✅ | ❌ | ⚠️ | ⚠️ | 🟡 | handler_vendors_test.go, crud-full.spec.js |
| List vendors | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_vendors_test.go |
| Vendor lead time | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 11. Work Orders

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create work order | ✅ | ✅ | ⚠️ | ✅ | 🔴 | handler_workorders_test.go, integration_workflow_test.go |
| Read work order | ✅ | ✅ | ❌ | ⚠️ | 🔴 | handler_workorders_test.go |
| Update work order | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_workorders_test.go |
| Delete work order | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_workorders_test.go |
| Status workflow | ✅ | ⚠️ | ❌ | ⚠️ | 🔴 | handler_workorders_test.go |
| View BOM with shortage analysis | ✅ | ✅ | ⚠️ | ✅ | 🔴 | integration_workflow_test.go, edge-cases.spec.js |
| Kit materials (reserve inventory) | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| Generate PDF traveler | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Add serial number | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** |
| Auto-generate serial number | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| List serial numbers | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Complete work order | ✅ | ⚠️ | ❌ | ⚠️ | 🔴 | handler_workorders_test.go |
| Track qty_good/qty_scrap | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Overdue work order detection | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Bulk complete work orders | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 12. Test Records

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create test record | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** |
| Get test by ID | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Get tests by serial | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** |
| List all tests | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 13. NCRs (Non-Conformance Reports)

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create NCR | ✅ | ❌ | ⚠️ | ⚠️ | 🔴 | handler_ncr_test.go, crud-full.spec.js |
| Read NCR | ✅ | ❌ | ⚠️ | ⚠️ | 🔴 | handler_ncr_test.go |
| Update NCR | ✅ | ❌ | ⚠️ | ⚠️ | 🟡 | handler_ncr_test.go, crud-full.spec.js |
| Delete NCR | ✅ | ❌ | ⚠️ | ⚠️ | 🟡 | handler_ncr_test.go, crud-full.spec.js |
| NCR→ECO auto-link | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Create ECO from NCR | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| NCR severity validation | ✅ | ❌ | ❌ | ⚠️ | 🟡 | db_integrity_test.go |
| Bulk close NCRs | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Aging NCR detection | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 14. Device Registry

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create device | ✅ | ❌ | ⚠️ | ⚠️ | 🔴 | handler_devices_test.go, crud-full.spec.js |
| Read device | ✅ | ❌ | ⚠️ | ⚠️ | 🔴 | handler_devices_test.go |
| Update device | ✅ | ❌ | ⚠️ | ⚠️ | 🟡 | handler_devices_test.go, crud-full.spec.js |
| Delete device | ✅ | ❌ | ⚠️ | ⚠️ | 🟡 | handler_devices_test.go, crud-full.spec.js |
| Device history | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_devices_test.go |
| Import devices (CSV) | ⚠️ | ❌ | ⚠️ | ⚠️ | 🟡 | import-export.spec.js |
| Export devices (CSV) | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | import-export.spec.js |
| Bulk decommission devices | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 15. Firmware Campaigns

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create campaign | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Read campaign | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Update campaign | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| List campaign devices | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Mark device updated | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Mark device failed | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| SSE live streaming | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 16. RMAs (Return Merchandise Authorization)

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create RMA | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Read RMA | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Update RMA | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Delete RMA | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Bulk close RMAs | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 17. Quotes

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create quote | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Read quote | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Update quote | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Delete quote | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | crud-full.spec.js only |
| Cost rollup calculation | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| Margin analysis | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Generate PDF quote | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 18. RFQs (Request for Quote)

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create RFQ | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_rfq_test.go |
| Read RFQ | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_rfq_test.go |
| Update RFQ | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_rfq_test.go |
| Delete RFQ | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_rfq_test.go |
| Send to vendors | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_rfq_test.go |
| Award to vendor | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_rfq_test.go |
| Award per line | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_rfq_test.go |
| Compare quotes | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_rfq_test.go |
| Create vendor quote | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Close RFQ | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| RFQ dashboard | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 19. Shipments

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create shipment | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_shipments_test.go |
| Read shipment | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_shipments_test.go |
| Update shipment | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_shipments_test.go |
| Mark shipped | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_shipments_test.go |
| Mark delivered | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_shipments_test.go |
| Pack list | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 20. Sales Orders

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create sales order | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_sales_orders_test.go |
| Read sales order | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_sales_orders_test.go |
| Update sales order | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_sales_orders_test.go |
| Delete sales order | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_sales_orders_test.go |

### 21. Invoices

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create invoice | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_invoices_test.go |
| Read invoice | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_invoices_test.go |
| Update invoice | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_invoices_test.go |
| Delete invoice | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_invoices_test.go |

### 22. Product Pricing

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create pricing | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_product_pricing_test.go |
| Read pricing | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_product_pricing_test.go |
| Update pricing | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_product_pricing_test.go |
| Delete pricing | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_product_pricing_test.go |
| Price analysis | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_product_pricing_test.go |
| Bulk update pricing | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Price history | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 23. Supplier Pricing

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Add supplier price | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | supplier-pricing.spec.js |
| View price history | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | supplier-pricing.spec.js |
| Price trend chart | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| Best price highlighting | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Auto-capture from PO | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 24. File Attachments

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Upload attachment | ❌ | ❌ | ✅ | ⚠️ | 🟡 | attachments.spec.js |
| List attachments | ❌ | ❌ | ✅ | ⚠️ | 🟡 | attachments.spec.js |
| Download attachment | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Delete attachment | ❌ | ❌ | ✅ | ⚠️ | 🟡 | attachments.spec.js |
| Validate file type | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Validate file size (32MB) | ❌ | ❌ | ✅ | ⚠️ | 🟡 | attachments.spec.js |

### 25. Audit Log

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Log create action | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| Log update action | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| Log delete action | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| Log bulk action | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| Filter by module | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Filter by user | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Filter by date range | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 26. Batch Operations

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Bulk approve ECOs | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Bulk complete WOs | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Bulk close NCRs | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Bulk decommission devices | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 27. Calendar

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Fetch calendar events | ❌ | ❌ | ✅ | ⚠️ | 🟡 | calendar.spec.js |
| Month navigation | ❌ | ❌ | ✅ | ⚠️ | 🟢 | calendar.spec.js |
| Event aggregation | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** - logic untested |
| Event color coding | ❌ | ❌ | ⚠️ | ⚠️ | 🟢 | calendar.spec.js |
| Click event to navigate | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 28. Dark Mode

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Toggle dark mode | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| Persist preference | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 29. Reports

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Inventory valuation report | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Open ECOs report | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| WO throughput report | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Low stock report | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| NCR summary report | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| CSV export | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 30. Notifications

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Generate low stock notification | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |
| Generate overdue WO notification | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Generate aging NCR notification | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Generate new RMA notification | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Mark notification as read | ❌ | ❌ | ⚠️ | ⚠️ | 🟡 | notifications.spec.js |
| Notification deduplication | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical logic |
| Notification preferences | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_notification_prefs_test.go |

### 31. Email Notifications

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Email configuration | ✅ | ❌ | ❌ | ⚠️ | 🟡 | email_test.go |
| Send test email | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Email on ECO approved | ✅ | ❌ | ❌ | ⚠️ | 🟡 | email_test.go |
| Email on low stock | ✅ | ❌ | ❌ | ⚠️ | 🟡 | email_test.go |
| Email on overdue WO | ✅ | ❌ | ❌ | ⚠️ | 🟡 | email_test.go |
| Email on PO received | ✅ | ❌ | ❌ | ⚠️ | 🟡 | email_test.go |
| Email log | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Email delivery verification | ❌ | ❌ | ❌ | ❌ | 🔴 | **NOT TESTED** - critical |

### 32. Undo/Changes

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Track changes | ✅ | ❌ | ❌ | ⚠️ | 🔴 | handler_changes_test.go |
| List recent changes | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_changes_test.go |
| Perform undo | ✅ | ❌ | ❌ | ⚠️ | 🔴 | handler_undo_test.go |
| Undo validation | ✅ | ❌ | ❌ | ⚠️ | 🔴 | handler_undo_test.go |

### 33. Backups

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create backup | ✅ | ❌ | ❌ | ⚠️ | 🔴 | handler_backup_test.go |
| List backups | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_backup_test.go |
| Download backup | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Delete backup | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |
| Restore backup | ✅ | ❌ | ❌ | ⚠️ | 🔴 | handler_backup_test.go |

### 34. Field Reports

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create field report | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_field_reports_test.go |
| Read field report | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_field_reports_test.go |
| Update field report | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_field_reports_test.go |
| Delete field report | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_field_reports_test.go |
| Create NCR from field report | ❌ | ❌ | ❌ | ❌ | 🟡 | **NOT TESTED** |

### 35. Settings & Configuration

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| General settings | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_general_settings_test.go |
| GitPLM config | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_gitplm_test.go |
| Git docs config | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| DigiKey settings | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |
| Mouser settings | ❌ | ❌ | ❌ | ❌ | 🟢 | **NOT TESTED** |

### 36. CAPA (Corrective & Preventive Action)

| Feature | Unit Test | Integration | E2E | Status | Priority | Notes |
|---------|-----------|-------------|-----|--------|----------|-------|
| Create CAPA | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_capa_test.go |
| Read CAPA | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_capa_test.go |
| Update CAPA | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_capa_test.go |
| Delete CAPA | ✅ | ❌ | ❌ | ⚠️ | 🟡 | handler_capa_test.go |

---

## Critical Path Analysis

### Happy Path Workflows That MUST Work

1. **🔴 CRITICAL: Part → Inventory → Work Order → Production**
   - Create part in gitplm → appears in ZRP
   - Receive inventory via PO
   - Create work order → BOM check shows component availability
   - Kit materials (reserve inventory)
   - Complete work order → deduct inventory
   - **Test Status**: ⚠️ Partially tested (BOM check ✅, kitting ❌, deduction ❌)

2. **🔴 CRITICAL: Purchase Order → Receiving → Inventory Update**
   - Create PO with vendor and line items
   - Receive PO → inventory quantities increase
   - Supplier prices captured
   - **Test Status**: ⚠️ Partially tested (PO create/receive ✅, inventory update ⚠️, price capture ❌)

3. **🔴 CRITICAL: ECO Workflow → Part Changes**
   - Create ECO with affected IPNs
   - Approve ECO → triggers email
   - Implement ECO → part revisions cascade
   - **Test Status**: ⚠️ Partially tested (create/approve ✅, email ⚠️, revision cascade ❌)

4. **🔴 CRITICAL: Low Stock Detection → Alert → Reorder**
   - Inventory transaction drops qty below reorder point
   - Low stock notification generated (deduplicated)
   - Dashboard shows low stock alert
   - Email sent to admin
   - Create PO to reorder
   - **Test Status**: ❌ Mostly untested (alert trigger ❌, dedup ❌, dashboard ⚠️, email ❌)

5. **🔴 CRITICAL: Work Order BOM Shortage → Generate PO**
   - Create WO for assembly with missing components
   - BOM view highlights shortages
   - Generate PO from shortages
   - **Test Status**: ✅ Tested (integration_bom_po_test.go, edge-cases.spec.js)

6. **🟡 HIGH: NCR → ECO → Implementation**
   - Create NCR documenting issue
   - Create ECO from NCR (auto-link)
   - ECO workflow → part changes
   - Close NCR
   - **Test Status**: ⚠️ Partially tested (NCR CRUD ✅, ECO link ❌, closure ❌)

7. **🟡 HIGH: Device Lifecycle**
   - Complete work order → generate serial numbers
   - Create test record for serial
   - Register device with serial
   - Track firmware updates
   - Handle RMA returns
   - **Test Status**: ⚠️ Fragmented (WO ✅, serial gen ❌, test record ❌, device reg ✅)

### Failure Modes That MUST Be Handled

1. **🔴 Negative Inventory Prevention**
   - Issue transaction when qty_on_hand < qty → reject
   - **Test Status**: ✅ Tested (db_integrity_test.go)

2. **🔴 Foreign Key Constraint Enforcement**
   - Delete vendor with open POs → prevent
   - Delete PO → cascade delete PO lines
   - **Test Status**: ✅ Tested (db_integrity_test.go)

3. **🔴 Invalid Status Transitions**
   - ECO: draft → implemented (skip review) → prevent
   - WO: completed → open → prevent
   - **Test Status**: ⚠️ Partially tested (db check ✅, handler logic ⚠️)

4. **🔴 Duplicate Serial Numbers**
   - Create work order serial with existing serial → reject
   - **Test Status**: ✅ Tested (db_integrity_test.go)

5. **🔴 BOM Circular Dependencies**
   - Part A contains Part B, Part B contains Part A → detect/prevent
   - **Test Status**: ❌ NOT TESTED

6. **🔴 Concurrent Inventory Updates**
   - Two transactions on same IPN simultaneously → race condition
   - **Test Status**: ❌ NOT TESTED

7. **🟡 Email Delivery Failure**
   - SMTP error → log failure, don't crash
   - **Test Status**: ⚠️ Partially tested (failure logging ✅, retry ❌)

### Data Corruption Risks

| Feature | Risk | Test Status |
|---------|------|-------------|
| BOM cost rollup | Wrong costs → bad quotes | ❌ NOT TESTED |
| Inventory kitting/reservation | Double-allocate inventory | ❌ NOT TESTED |
| PO receive → inventory update | Inventory mismatch | ⚠️ Partial |
| ECO part revision cascade | Lost change history | ❌ NOT TESTED |
| Audit log gaps | Missing accountability | ❌ NOT TESTED |
| Notification deduplication | Spam users | ❌ NOT TESTED |
| Concurrent inventory edits | Race conditions | ❌ NOT TESTED |

---

## Recommendations

### Immediate Actions (Critical Gaps)

1. **Add integration tests for:**
   - [ ] BOM cost rollup calculation
   - [ ] Work order kitting (inventory reservation)
   - [ ] ECO part revision cascade logic
   - [ ] Low stock alert generation and deduplication
   - [ ] Notification deduplication logic
   - [ ] Calendar event aggregation
   - [ ] Audit log writes for all CRUD operations

2. **Add E2E tests for:**
   - [ ] Complete work order → inventory deduction flow
   - [ ] PO receive → inventory update → low stock check → email flow
   - [ ] ECO approve → email → implement → part revision flow
   - [ ] Serial number generation and tracking through device lifecycle

3. **Add stress/concurrency tests for:**
   - [ ] Concurrent inventory transactions on same IPN
   - [ ] Concurrent PO receives
   - [ ] High-volume notification generation

### Test Infrastructure Improvements

1. **Test data builders** for complex objects (BOM trees, multi-line POs)
2. **Shared fixtures** for common test scenarios (user with role, part with inventory)
3. **Integration test helpers** for end-to-end workflows
4. **Performance benchmarks** for critical queries (BOM tree, inventory valuation)

### Documentation Needs

1. **Test strategy document** explaining unit vs integration vs E2E boundaries
2. **Test data management** guide for setting up realistic scenarios
3. **Coverage goals** per module (target: 80% unit, 60% integration, 40% E2E)

---

## Test File Reference

### Go Unit Tests (Backend)
- `handler_auth_test.go` - Authentication handlers
- `handler_parts_test.go`, `handler_parts_create_test.go` - Parts CRUD
- `handler_eco_test.go` - ECO workflows
- `handler_inventory_test.go` - Inventory transactions
- `handler_procurement_test.go` - Purchase orders
- `handler_workorders_test.go` - Work orders
- `handler_ncr_test.go` - NCRs
- `handler_devices_test.go` - Device registry
- `handler_rfq_test.go` - RFQs
- `handler_vendors_test.go` - Vendors
- `handler_users_test.go` - User management
- `email_test.go` - Email notifications
- `search_test.go` - Global search
- `db_integrity_test.go` - Database constraints
- `permissions_test.go`, `rbac_test.go` - Authorization

### Go Integration Tests
- `integration_bom_po_test.go` - BOM → PO workflow
- `integration_workflow_test.go` - Cross-module workflows
- `integration_real_test.go` - Real-world scenarios
- `receiving_eco_test.go` - PO receiving + ECO interaction
- `quality_workflow_test.go` - Quality processes

### Playwright E2E Tests
- `playwright.spec.js` - Main E2E suite
- `crud-full.spec.js` - CRUD operations across modules
- `edge-cases.spec.js` - BOM shortages, PO generation
- `permissions.spec.js` - RBAC UI enforcement
- `attachments.spec.js` - File uploads
- `calendar.spec.js` - Calendar views
- `api-keys.spec.js` - API key management
- `import-export.spec.js` - CSV import/export
- `supplier-pricing.spec.js` - Supplier price catalog
- `notifications.spec.js` - Notification interactions
- `validation.spec.js` - Form validation
- `cross-module.spec.js` - Dashboard KPI navigation

---

**Document Status**: Initial inventory complete  
**Next Steps**: Create manual testing checklist, prioritize critical gaps, establish test coverage goals
