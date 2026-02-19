# Delete Operations E2E Tests

## Overview

Comprehensive end-to-end tests for delete operations across all ZRP modules.

## Test Coverage

### ✅ Implemented & Tested
- **Vendors**: Full delete with constraint checking
  - Confirmation dialog verification
  - Foreign key constraint enforcement (POs, RFQs)
  - Success/error notification validation
  
- **Inventory**: Bulk delete operations
  - Checkbox selection
  - Bulk action buttons
  - Confirmation dialogs

### ⚠️ Documented (Endpoint Exists but Not Functional)
- **Parts**: Endpoint exists but returns 501 (not implemented)
  - Backend returns: "deleting parts via API not yet supported — edit CSVs directly"
  - Tests are written but skipped
  - Ready to enable when backend implementation is complete

### ❌ Not Yet Implemented (No Endpoints)
The following modules have NO delete endpoints in the backend. Tests are written as skipped/documented tests ready to be enabled when endpoints are implemented:

- **Work Orders**
  - Should allow delete in draft/planned states
  - Should prevent delete in progress/completed states
  - Should release reserved inventory on delete

- **Purchase Orders**
  - Should allow delete of draft POs
  - Should prevent delete of sent/received POs
  - Should clean up PO line items

- **ECOs**
  - Should allow delete of draft ECOs
  - Should prevent delete of approved/implemented ECOs
  - Should clean up related part changes

## Running the Tests

### Prerequisites
1. **ZRP Server Running**: The tests expect the server to be running on `http://localhost:9000`
   ```bash
   cd /Users/jsnapoli1/.openclaw/workspace/zrp
   ./zrp -db zrp.db -pmDir uploads/parts -port 9000
   ```

2. **Test Data**: Tests create their own test data but require:
   - Admin user credentials (admin/changeme)
   - Fresh database state (recommended)

### Run All Delete Tests
```bash
cd frontend
npx playwright test delete-operations.spec.ts
```

### Run Specific Test Suites
```bash
# Only vendor delete tests
npx playwright test delete-operations.spec.ts -g "Delete Operations - Vendors"

# Only inventory delete tests
npx playwright test delete-operations.spec.ts -g "Delete Operations - Inventory"

# View summary
npx playwright test delete-operations.spec.ts -g "Summary"
```

### Debug Mode
```bash
npx playwright test delete-operations.spec.ts --debug
```

## Test Structure

Each test follows this pattern:

1. **Setup**: Login and navigate to module
2. **Create Test Data**: Create entities to delete
3. **Trigger Delete**: Click delete button/menu
4. **Verify Confirmation**: Check dialog appears with proper warnings
5. **Execute Delete**: Confirm deletion
6. **Verify Results**: Check success/error notifications and data state

## Constraint Testing

Tests verify foreign key constraints prevent orphaned data:

- **Vendor with POs**: Cannot delete vendor with existing purchase orders
- **Part in BOM**: Cannot delete part used in bill of materials (when implemented)
- **Work Order in Progress**: Cannot delete active work orders (when implemented)
- **Received PO**: Cannot delete purchase orders with received items (when implemented)

## Success Criteria

✅ **Completed**:
- [x] Vendor delete with confirmation dialog
- [x] Vendor delete constraint enforcement (POs, RFQs)
- [x] Inventory bulk delete UI checks
- [x] Comprehensive documentation of missing endpoints
- [x] Tests written for all future implementations

⏳ **Pending Backend Implementation**:
- [ ] Part delete implementation (remove 501 status)
- [ ] Work order delete endpoint
- [ ] Purchase order delete endpoint
- [ ] ECO delete endpoint

## Screenshot Documentation

Failed tests generate screenshots in `test-results/`:
- `delete-vendor-confirmation.png`: Vendor delete dialog
- `delete-vendor-constraint.png`: Constraint violation error
- `delete-inventory-confirmation.png`: Inventory bulk delete dialog

## Future Enhancements

When implementing new delete endpoints:

1. **Update Test**: Remove `.skip` from corresponding test
2. **Add Constraints**: Ensure proper foreign key checks
3. **Audit Logging**: Verify deletes are logged
4. **Soft Delete**: Consider soft delete for traceability
5. **Undo Support**: Implement undo functionality where appropriate

## Related Files

- `frontend/src/pages/Vendors.tsx`: Vendor delete implementation
- `frontend/src/components/ConfirmDialog.tsx`: Reusable confirmation dialog
- `handler_vendors.go`: Backend vendor delete with constraints
- `handler_inventory.go`: Backend inventory bulk delete
- `handler_parts.go`: Backend part delete (501 stub)

## References

- **Task**: MISSING_E2E_TESTS.md - Delete Operations
- **Backend Constraints**: See `handleDeleteVendor()` in handler_vendors.go for reference implementation
- **API Spec**: frontend/src/lib/api.ts
