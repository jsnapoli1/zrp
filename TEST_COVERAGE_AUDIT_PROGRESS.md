# ZRP Test Coverage Audit - In Progress

**Date:** 2026-02-20  
**Auditor:** Subagent (zrp-test-coverage-audit)

## Initial Findings

### Backend (Go)

**Test Files:** 70 test files found  
**Handler Files:** 50 handler files found  
**Status:** Running coverage analysis...

#### Handlers WITHOUT dedicated test files:
1. handler_advanced_search.go
2. handler_apikeys.go
3. handler_attachments.go
4. handler_bulk.go
5. handler_calendar.go
6. handler_costing.go
7. handler_docs.go
8. handler_email.go
9. handler_export.go
10. handler_firmware.go
11. handler_git_docs.go
12. handler_ncr_integration.go
13. handler_notifications.go
14. handler_permissions.go
15. handler_prices.go
16. handler_query_profiler.go
17. handler_quotes.go
18. handler_receiving.go
19. handler_reports.go
20. handler_rfq.go (has .skip file)
21. handler_rma.go
22. handler_scan.go
23. handler_search.go
24. handler_testing.go
25. handler_widgets.go

#### Test Issues Discovered:
- **Schema mismatch:** Tests failing due to missing database tables (audit_log, password_history, ncrs)
- **Schema version mismatch:** audit_log table missing 'module' column
- **Concurrency test issues:** Inventory table already exists errors
- **Authentication test failures:** Rate limiting and session management tests failing

### Frontend (Vitest)

**Page Components:** 59 .tsx files in src/pages  
**Test Files:** 60 .test.tsx files  
**Coverage:** All pages appear to have test files ✓

**Test structure found:**
- Component tests in src/components
- Hook tests in src/hooks
- Page tests in src/pages
- API tests in src/lib

**Status:** Need to run coverage analysis

## Next Steps

1. ✅ Identify handlers without tests
2. ⏳ Complete Go test coverage run
3. ⏳ Run frontend Vitest coverage
4. Create tests for uncovered handlers
5. Fix database schema issues in tests
6. Add integration tests for cross-module flows
7. Document findings

## Priority Test Additions Needed

Based on handler analysis, highest priority areas:
- **handler_quotes.go** - No tests (critical business logic)
- **handler_rma.go** - No tests (customer service critical)
- **handler_receiving.go** - No tests (inventory impact)
- **handler_reports.go** - No tests (business intelligence)
- **handler_permissions.go** - No tests (security critical)
- **handler_attachments.go** - No tests (file upload security)
