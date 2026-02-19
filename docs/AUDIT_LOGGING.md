# Audit Logging System

## Overview

ZRP includes a comprehensive audit logging system for compliance, security, and accountability. The system tracks all significant user actions, data modifications, and sensitive data access.

## Features

### 1. Comprehensive Tracking
- **User Actions**: Track who did what, when
- **Data Changes**: Before/after values for updates
- **Sensitive Data Access**: Log viewing of pricing, costs, and confidential information
- **Data Exports**: Track all data export operations
- **System Context**: IP address, user agent, timestamp

### 2. Action Types

| Action Type | Description | Use Case |
|------------|-------------|----------|
| `CREATE` | New record created | Part added, PO created |
| `UPDATE` | Record modified | Part updated, status changed |
| `DELETE` | Record removed | Part deleted, PO cancelled |
| `VIEW` | Record viewed | View part details |
| `VIEW_SENSITIVE` | Sensitive data accessed | View pricing, cost data |
| `EXPORT` | Data exported | Export parts list, audit logs |
| `LOGIN` | User logged in | Session started |
| `LOGOUT` | User logged out | Session ended |
| `APPROVE` | Approval granted | ECO approved, PO approved |
| `REJECT` | Approval rejected | ECO rejected |

### 3. Retention Policy

- **Configurable**: Set retention period from 30 days to 10 years
- **Default**: 365 days (1 year)
- **Cleanup**: Manual or scheduled cleanup of old logs
- **Compliance**: Adjust to meet regulatory requirements

### 4. Audit Log Viewer

The audit log viewer provides powerful filtering and search capabilities:

#### Filters
- **Search**: Full-text search across all fields
- **Entity Type**: Filter by module (parts, work orders, etc.)
- **Action Type**: Filter by action (CREATE, UPDATE, DELETE, etc.)
- **User**: Filter by username
- **Date Range**: From/to date filtering

#### Export
- **CSV Export**: Download filtered logs for analysis
- **Complete History**: Up to 10,000 records per export
- **Compliance Reports**: Generate audit reports for compliance

#### Details View
- **Before/After Values**: See exact changes made
- **IP Address**: Track where actions originated
- **User Agent**: Browser/application information
- **Full Context**: Complete audit trail

## Database Schema

```sql
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    username TEXT DEFAULT 'system',
    action TEXT NOT NULL,
    module TEXT NOT NULL,
    record_id TEXT NOT NULL,
    summary TEXT,
    before_value TEXT,      -- JSON of record before change
    after_value TEXT,       -- JSON of record after change
    ip_address TEXT,        -- Client IP address
    user_agent TEXT,        -- Browser/client user agent
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_log_module ON audit_log(module);
CREATE INDEX idx_audit_log_record_id ON audit_log(record_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_ip_address ON audit_log(ip_address);
CREATE INDEX idx_audit_log_user_created ON audit_log(user_id, created_at);
```

## API Usage

### Backend (Go)

#### Simple Audit Logging

```go
// Basic audit log (backward compatible)
logAudit(db, username, "CREATE", "part", ipn, "Created part "+ipn)
```

#### Enhanced Audit Logging

```go
// Full featured audit logging
LogSimpleAudit(db, r, AuditActionCreate, "part", ipn, "Created part "+ipn)

// Log with before/after values
LogUpdateWithDiff(db, r, "part", ipn, oldPart, newPart)

// Log sensitive data access
LogSensitiveDataAccess(db, r, "part", ipn, "pricing/cost")

// Log data export
LogDataExport(db, r, "parts", "CSV", 150)
```

#### Helper Functions

```go
// Get user context from request
userID, username := GetUserContext(r, db)

// Get client IP (handles proxies)
ip := GetClientIP(r)

// Cleanup old logs
deleted, err := CleanupOldAuditLogs(db, retentionDays)

// Get/Set retention policy
days := GetAuditRetentionDays(db)
err := SetAuditRetentionDays(db, 730) // 2 years
```

### Frontend (TypeScript)

#### Fetch Audit Logs

```typescript
const result = await api.getAuditLogs({
  search: "part-001",
  entityType: "part",
  user: "admin",
  page: 1,
  limit: 50
});

console.log(result.entries); // Array of AuditLogEntry
console.log(result.total);   // Total count
```

#### Export Audit Logs

```typescript
// CSV export
const url = `/api/audit/export?entity_type=part&from=2024-01-01&to=2024-12-31`;
window.open(url, '_blank');
```

#### Manage Retention

```typescript
// Get current retention
const response = await fetch('/api/audit/retention');
const { retention_days } = await response.json();

// Update retention
await fetch('/api/audit/retention', {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ retention_days: 730 })
});

// Cleanup old logs
await fetch('/api/audit/cleanup', { method: 'POST' });
```

## API Endpoints

### GET `/api/audit`
Get audit log entries with optional filters.

**Query Parameters:**
- `search` - Full-text search
- `entity_type` / `module` - Filter by entity type
- `user` - Filter by username
- `action` - Filter by action type
- `from` - Start date (YYYY-MM-DD)
- `to` - End date (YYYY-MM-DD)
- `page` - Page number (default: 1)
- `limit` - Results per page (default: 50)

**Response:**
```json
{
  "entries": [
    {
      "id": 1,
      "user_id": 5,
      "username": "admin",
      "action": "CREATE",
      "module": "part",
      "record_id": "RES-0001",
      "summary": "Created part RES-0001",
      "before_value": null,
      "after_value": "{\"ipn\":\"RES-0001\",\"description\":\"Resistor\"}",
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0...",
      "created_at": "2024-02-19 13:45:00"
    }
  ],
  "total": 1
}
```

### GET `/api/audit/export`
Export audit logs to CSV.

**Query Parameters:** Same as GET `/api/audit`

**Response:** CSV file download

### GET `/api/audit/retention`
Get current retention policy.

**Response:**
```json
{
  "retention_days": 365
}
```

### PUT `/api/audit/retention`
Update retention policy.

**Request:**
```json
{
  "retention_days": 730
}
```

**Response:**
```json
{
  "success": true,
  "retention_days": 730
}
```

### POST `/api/audit/cleanup`
Delete audit logs older than retention period.

**Response:**
```json
{
  "success": true,
  "deleted": 1523,
  "message": "Deleted 1523 audit log entries older than 365 days"
}
```

## Best Practices

### 1. What to Log

**Always Log:**
- CREATE, UPDATE, DELETE operations
- Sensitive data access (pricing, costs, salaries)
- Data exports
- Configuration changes
- User management actions
- Approval/rejection workflows

**Consider Logging:**
- VIEW operations for highly sensitive data
- Bulk operations
- Failed authentication attempts
- Permission changes

**Don't Log:**
- Normal read operations on non-sensitive data
- Health checks
- Static asset requests
- Automated background tasks (unless they modify data)

### 2. When to Use Before/After Values

Use `LogUpdateWithDiff()` when:
- Changes need detailed audit trail
- Compliance requires change tracking
- Debugging data changes
- Rollback might be needed

Skip before/after for:
- Large records (>1KB)
- Binary data
- Frequently changing data (e.g., timestamps)
- Non-critical updates

### 3. Retention Policy

**Guidelines:**
- Financial data: 7 years minimum
- Quality records: Per ISO requirements
- General operations: 1-2 years
- Debug logs: 30-90 days

**Compliance:**
- ISO 9001: Quality records per procedure
- SOX: 7 years for financial
- GDPR: Minimum necessary period
- Industry-specific: Check regulations

### 4. Performance Considerations

**Indexes:**
- Keep existing indexes on commonly filtered fields
- Monitor query performance
- Add indexes if filtering is slow

**Cleanup:**
- Run cleanup during off-hours
- Schedule monthly or quarterly
- Monitor database size
- Archive old logs if needed

## Security

### Access Control
- Only admins can view full audit logs
- Users can view their own actions
- Export requires admin permissions
- Cleanup requires admin permissions

### Data Privacy
- Hash sensitive values in before/after if needed
- Exclude passwords from audit logs
- Mask PII in exported logs if required
- Follow data retention regulations

### Integrity
- Audit logs should not be editable
- Protect against tampering
- Regular backups
- Monitor for suspicious patterns

## Compliance

### ISO 9001
- Track all quality-related actions
- Document approvals and changes
- Maintain traceability
- Keep records per procedure

### SOX (If Applicable)
- Track financial data access
- Log all data exports
- Maintain 7-year retention
- Protect log integrity

### GDPR
- Log personal data access
- Support data deletion requests
- Maintain minimum necessary logs
- Provide audit trail on request

## Troubleshooting

### Logs Not Appearing
1. Check database migration ran successfully
2. Verify `LogSimpleAudit()` calls exist
3. Check for errors in server logs
4. Confirm user session is valid

### Export Not Working
1. Verify export endpoint is registered
2. Check file permissions
3. Review error logs
4. Test with simple query first

### Performance Issues
1. Review index usage
2. Check retention policy
3. Run cleanup if database is large
4. Consider archiving old logs

### Missing Fields
1. Run database migrations
2. Check schema matches documentation
3. Verify API returns all fields
4. Update frontend types

## Future Enhancements

Potential improvements for future versions:

- **Scheduled Cleanup**: Automatic daily/weekly cleanup
- **Archival**: Move old logs to archive storage
- **Advanced Search**: Full-text search with highlighting
- **Anomaly Detection**: Alert on suspicious patterns
- **Compliance Reports**: Pre-built reports for auditors
- **Real-time Monitoring**: Live audit log stream
- **Integration**: Export to SIEM systems
- **Rollback**: Automatic rollback using before values

## Migration Guide

### From Legacy to Enhanced

If upgrading from the simple audit system:

1. **Database**: Migrations run automatically on startup
2. **Backend**: Replace `logAudit()` calls gradually:
   ```go
   // Old
   logAudit(db, username, "create", "part", ipn, "Created part")
   
   // New
   LogSimpleAudit(db, r, AuditActionCreate, "part", ipn, "Created part")
   ```
3. **Frontend**: Update `AuditLogEntry` interface in `api.ts`
4. **Testing**: Verify logs appear in UI
5. **Rollout**: Can run both systems in parallel

## Support

For issues or questions:
1. Check this documentation
2. Review example code in handlers
3. Check server logs for errors
4. Contact development team
