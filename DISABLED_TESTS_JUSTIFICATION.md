# Disabled Tests Justification

This document tracks all disabled test files and their disposition.

## Files Deleted (Backups/Duplicates)

### 1. handler_apikeys_test.go.backup
**Action:** DELETED  
**Reason:** Exact duplicate of active `handler_apikeys_test.go` (649 vs 644 lines, identical test functions)  
**Date:** 2026-02-20

### 2. handler_auth_test.go.broken.old
**Action:** DELETED  
**Reason:** Superseded by `handler_auth_test.go.skip.temp` which has more comprehensive tests (827 vs 760 lines)  
**Date:** 2026-02-20

### 3. handler_export_test.go.tmp
**Action:** Will be determined after comparing with active handler_export_test.go

### 4. test_debug.go.skip
**Action:** Will be determined (not in original list but found)

## Files Activated (Fixed & Renamed)

### 1. handler_auth_test.go.skip.temp → handler_auth_test.go
**Action:** ACTIVATED  
**Reason:** No active auth tests exist, this is the most comprehensive version  
**Status:** Needs compilation check and fixes

### 2-18. [To be filled as we process each file]

## Files Still Under Review

[To be updated as we work through the list]
