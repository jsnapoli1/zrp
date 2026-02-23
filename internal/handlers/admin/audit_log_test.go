package admin_test

// Skipped: handler_audit_log_test.go tests cross-cutting audit logging through
// root-level handler functions (handleCreateVendor, handleUpdateVendor,
// handleDeleteVendor, handleCreateECO, handleApproveECO, handleCreatePO,
// handleUpdatePO, handleInventoryTransact) and root-level helpers
// (LogUpdateWithDiff, LogAuditEnhanced, GetUserContext). These are not admin
// handler methods; they verify that audit trails are recorded when domain
// handlers (vendor, ECO, PO, inventory) perform mutations. This file belongs
// in the cross-cutting Batch 11 tests that stay in the root package.
