# ZRP API Reference

Base URL: `http://localhost:9000`

## Authentication

ZRP uses session-based authentication with cookie tokens. API keys (Bearer tokens) are also supported for programmatic access.

- **Session auth:** POST to `/auth/login`, receive a `zrp_session` cookie
- **API key auth:** Send `Authorization: Bearer zrp_...` header
- **Exempt paths:** `/`, `/static/*`, `/auth/*`, `/files/*` do not require auth

### Roles

| Role | Permissions |
|------|-------------|
| `admin` | Full access, can manage users and API keys |
| `user` | Read and write access to all modules |
| `readonly` | GET requests only — POST/PUT/DELETE return 403 |

## Response Format

Successful responses wrap data in an envelope:

```json
{
  "data": { ... },
  "meta": { "total": 100, "page": 1, "limit": 50 }
}
```

`meta` is included only for paginated list endpoints. Error responses:

```json
{ "error": "not found" }
```

## HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Invalid request body |
| 401 | Unauthorized (no session or invalid API key) |
| 403 | Forbidden (deactivated account or readonly role) |
| 404 | Resource not found |
| 409 | Conflict (e.g., duplicate username) |
| 500 | Server error |
| 501 | Not implemented (e.g., CSV write operations) |

---

## Auth

### POST /auth/login

Authenticate and create a session. No auth required.

**Request Body:**

```json
{ "username": "admin", "password": "changeme" }
```

**Response (200):**

```json
{
  "user": { "id": 1, "username": "admin", "display_name": "Administrator", "role": "admin" }
}
```

Sets `zrp_session` cookie (HttpOnly, 24h expiry).

```bash
curl -X POST http://localhost:9000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme"}' \
  -c cookies.txt
```

### POST /auth/logout

Destroy the current session. No auth required.

**Response (200):**

```json
{ "status": "ok" }
```

```bash
curl -X POST http://localhost:9000/auth/logout -b cookies.txt
```

### GET /auth/me

Get the currently authenticated user. No auth required (returns 401 if not logged in).

**Response (200):**

```json
{
  "user": { "id": 1, "username": "admin", "display_name": "Administrator", "role": "admin" }
}
```

**Response (401):**

```json
{ "error": "Unauthorized", "code": "UNAUTHORIZED" }
```

```bash
curl http://localhost:9000/auth/me -b cookies.txt
```

---

## Users

Admin-only endpoints for managing user accounts.

### GET /api/v1/users

List all users. Requires admin role.

**Response:**

```json
{
  "data": [
    { "id": 1, "username": "admin", "display_name": "Administrator", "role": "admin", "active": 1, "created_at": "2026-02-17 20:46:55", "last_login": "2026-02-17 21:00:00" }
  ]
}
```

```bash
curl http://localhost:9000/api/v1/users -b cookies.txt
```

### POST /api/v1/users

Create a new user. Requires admin role.

**Request Body:**

```json
{ "username": "jdoe", "display_name": "Jane Doe", "password": "secret123", "role": "user" }
```

| Field | Required | Description |
|-------|----------|-------------|
| `username` | Yes | Unique login name |
| `password` | Yes | Plain-text password (hashed with bcrypt) |
| `display_name` | No | Friendly name |
| `role` | No | `admin`, `user`, or `readonly` (default: `user`) |

**Response (201):**

```json
{ "data": { "id": 5, "username": "jdoe", "display_name": "Jane Doe", "role": "user" } }
```

```bash
curl -X POST http://localhost:9000/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username":"jdoe","display_name":"Jane Doe","password":"secret123","role":"user"}' \
  -b cookies.txt
```

### PUT /api/v1/users/:id

Update a user's display name, role, or active status. Requires admin role. Admin cannot deactivate themselves.

**Request Body:**

```json
{ "display_name": "Jane Smith", "role": "admin", "active": 1 }
```

```bash
curl -X PUT http://localhost:9000/api/v1/users/5 \
  -H "Content-Type: application/json" \
  -d '{"display_name":"Jane Smith","role":"admin","active":1}' \
  -b cookies.txt
```

### PUT /api/v1/users/:id/password

Reset a user's password. Requires admin role.

**Request Body:**

```json
{ "password": "newpassword456" }
```

```bash
curl -X PUT http://localhost:9000/api/v1/users/5/password \
  -H "Content-Type: application/json" \
  -d '{"password":"newpassword456"}' \
  -b cookies.txt
```

---

## API Keys

Manage API keys for programmatic (Bearer token) access. Authenticated users can manage keys.

### GET /api/v1/apikeys

List all API keys. Keys show only the prefix (first 12 chars), never the full key.

**Response:**

```json
{
  "data": [
    { "id": 1, "name": "CI Pipeline", "key_prefix": "zrp_a1b2c3d4", "created_by": "admin", "created_at": "2026-02-17", "last_used": null, "expires_at": null, "enabled": 1 }
  ]
}
```

```bash
curl http://localhost:9000/api/v1/apikeys -b cookies.txt
```

### POST /api/v1/apikeys

Generate a new API key. The full key is returned only once.

**Request Body:**

```json
{ "name": "CI Pipeline", "expires_at": "2027-01-01" }
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Descriptive name |
| `expires_at` | No | Expiration date (ISO 8601) |

**Response (201):**

```json
{ "id": 1, "name": "CI Pipeline", "key": "zrp_a1b2c3d4e5f67890abcdef12345678", "key_prefix": "zrp_a1b2c3d4", "message": "Store this key securely. It will not be shown again." }
```

```bash
curl -X POST http://localhost:9000/api/v1/apikeys \
  -H "Content-Type: application/json" \
  -d '{"name":"CI Pipeline"}' \
  -b cookies.txt
```

### DELETE /api/v1/apikeys/:id

Revoke (permanently delete) an API key.

```bash
curl -X DELETE http://localhost:9000/api/v1/apikeys/1 -b cookies.txt
```

### PUT /api/v1/apikeys/:id

Enable or disable an API key without deleting it.

**Request Body:**

```json
{ "enabled": 0 }
```

```bash
curl -X PUT http://localhost:9000/api/v1/apikeys/1 \
  -H "Content-Type: application/json" \
  -d '{"enabled":0}' \
  -b cookies.txt
```

---

## Notifications

System-generated notifications for low stock, overdue work orders, open NCRs, and new RMAs.

### GET /api/v1/notifications

List notifications (most recent 50).

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `unread` | string | `true` to show only unread notifications |

**Response:**

```json
{
  "data": [
    { "id": 1, "type": "low_stock", "severity": "warning", "title": "Low Stock: CAP-001-0001", "message": "25 on hand, reorder point 100", "record_id": "CAP-001-0001", "module": "inventory", "read_at": null, "created_at": "2026-02-17 21:00:00" }
  ]
}
```

```bash
curl "http://localhost:9000/api/v1/notifications?unread=true" -b cookies.txt
```

### POST /api/v1/notifications/:id/read

Mark a notification as read.

**Response:**

```json
{ "status": "read" }
```

```bash
curl -X POST http://localhost:9000/api/v1/notifications/1/read -b cookies.txt
```

---

## Attachments

File attachments linked to any module record.

### POST /api/v1/attachments

Upload a file. Uses `multipart/form-data`. Max 32MB.

| Field | Required | Description |
|-------|----------|-------------|
| `module` | Yes | Module name (e.g., `eco`, `ncr`, `workorder`) |
| `record_id` | Yes | ID of the parent record |
| `file` | Yes | The file to upload |

**Response (201):**

```json
{
  "data": { "id": 1, "module": "eco", "record_id": "ECO-2026-001", "filename": "eco-ECO-2026-001-1708300000000-schematic.pdf", "original_name": "schematic.pdf", "size_bytes": 245000, "mime_type": "application/pdf", "uploaded_by": "admin" }
}
```

```bash
curl -X POST http://localhost:9000/api/v1/attachments \
  -F "module=eco" \
  -F "record_id=ECO-2026-001" \
  -F "file=@schematic.pdf" \
  -b cookies.txt
```

### GET /api/v1/attachments

List attachments for a specific record.

**Query Parameters (both required):**

| Param | Type | Description |
|-------|------|-------------|
| `module` | string | Module name |
| `record_id` | string | Record ID |

```bash
curl "http://localhost:9000/api/v1/attachments?module=eco&record_id=ECO-2026-001" -b cookies.txt
```

### DELETE /api/v1/attachments/:id

Delete an attachment (removes file from disk).

```bash
curl -X DELETE http://localhost:9000/api/v1/attachments/1 -b cookies.txt
```

---

## File Serving

### GET /files/:filename

Serve an uploaded file. No auth required.

```bash
curl http://localhost:9000/files/eco-ECO-2026-001-1708300000000-schematic.pdf
```

---

## Audit Log

### GET /api/v1/audit

Query the audit log. All create/update/delete/bulk operations are logged.

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `module` | string | Filter by module (e.g., `eco`, `workorder`, `po`) |
| `user` | string | Filter by username |
| `from` | string | Start date (YYYY-MM-DD) |
| `to` | string | End date (YYYY-MM-DD) |
| `limit` | int | Max results (default: 50) |

**Response:**

```json
{
  "data": [
    { "id": 1, "username": "admin", "action": "created", "module": "eco", "record_id": "ECO-2026-001", "summary": "Created ECO ECO-2026-001", "created_at": "2026-02-17 21:00:00" }
  ]
}
```

```bash
curl "http://localhost:9000/api/v1/audit?module=eco&limit=10" -b cookies.txt
```

---

## Dashboard

### GET /api/v1/dashboard

Returns summary KPIs.

```bash
curl http://localhost:9000/api/v1/dashboard -b cookies.txt
```

```json
{
  "open_ecos": 2, "low_stock": 1, "open_pos": 1, "active_wos": 1,
  "open_ncrs": 1, "open_rmas": 1, "total_parts": 150, "total_devices": 3
}
```

Note: The dashboard response is **not** wrapped in the `{data}` envelope.

### GET /api/v1/dashboard/charts

Chart data for the dashboard: ECOs by status, work orders by status, top inventory by value.

**Response:**

```json
{
  "data": {
    "ecos_by_status": { "draft": 1, "review": 0, "approved": 1, "implemented": 0 },
    "wos_by_status": { "open": 1, "in_progress": 0, "completed": 0 },
    "inventory_value": [ { "ipn": "CAP-001-0001", "value": 10.0 } ]
  }
}
```

```bash
curl http://localhost:9000/api/v1/dashboard/charts -b cookies.txt
```

### GET /api/v1/dashboard/lowstock

Low stock items (where qty on hand < reorder point).

**Response:**

```json
{
  "data": [ { "ipn": "RES-001-0001", "qty_on_hand": 25, "reorder_point": 100 } ]
}
```

```bash
curl http://localhost:9000/api/v1/dashboard/lowstock -b cookies.txt
```

---

## Global Search

### GET /api/v1/search

Search across all modules simultaneously.

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `q` | string | Search query (case-insensitive) |
| `limit` | int | Max results per module (default: 20) |

**Response:**

```json
{
  "data": {
    "parts": [ { "IPN": "CAP-001-0001", "Manufacturer": "Murata", ... } ],
    "ecos": [ { "id": "ECO-2026-001", "title": "...", "status": "draft" } ],
    "workorders": [ { "id": "WO-2026-0001", "assembly_ipn": "...", "status": "open" } ],
    "devices": [ { "serial_number": "SN-001", "ipn": "...", "customer": "...", "status": "active" } ],
    "ncrs": [ { "id": "NCR-2026-001", "title": "...", "status": "open" } ],
    "pos": [ { "id": "PO-2026-0001", "status": "received" } ],
    "quotes": [ { "id": "Q-2026-001", "customer": "...", "status": "draft" } ]
  },
  "meta": { "total": 5, "query": "murata" }
}
```

```bash
curl "http://localhost:9000/api/v1/search?q=murata&limit=10" -b cookies.txt
```

---

## Calendar

### GET /api/v1/calendar

Calendar events for a given month: WO due dates, PO expected deliveries, quote expirations.

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `year` | int | Year (default: current) |
| `month` | int | Month 1-12 (default: current) |

**Response:**

```json
{
  "data": [
    { "date": "2026-03-15", "type": "workorder", "id": "WO-2026-0001", "title": "Build PCB-001 ×25", "color": "blue" },
    { "date": "2026-03-20", "type": "po", "id": "PO-2026-0001", "title": "PO expected delivery", "color": "green" },
    { "date": "2026-03-31", "type": "quote", "id": "Q-2026-001", "title": "Quote for MegaCorp expires", "color": "orange" }
  ]
}
```

```bash
curl "http://localhost:9000/api/v1/calendar?year=2026&month=3" -b cookies.txt
```

---

## Parts

Parts are read from gitplm CSV files on disk. Write operations return 501.

### GET /api/v1/parts

List parts with optional filtering and pagination.

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `category` | string | Filter by category name |
| `q` | string | Full-text search across IPN and all fields |
| `page` | int | Page number (default: 1) |
| `limit` | int | Results per page (default: 50) |

```bash
curl "http://localhost:9000/api/v1/parts?category=capacitors&q=murata&page=1&limit=20" -b cookies.txt
```

### GET /api/v1/parts/:ipn

Get a single part by IPN.

```bash
curl http://localhost:9000/api/v1/parts/CAP-001-0001 -b cookies.txt
```

### GET /api/v1/parts/:ipn/bom

Get the Bill of Materials for an assembly IPN. Returns sub-components parsed from the parts database with quantities and descriptions.

```bash
curl http://localhost:9000/api/v1/parts/PCB-001-0001/bom -b cookies.txt
```

### GET /api/v1/parts/:ipn/cost

BOM cost rollup for a part. Returns component costs, totals, and cost breakdown.

```bash
curl http://localhost:9000/api/v1/parts/PCB-001-0001/cost -b cookies.txt
```

### POST /api/v1/parts → 501

### PUT /api/v1/parts/:ipn → 501

### DELETE /api/v1/parts/:ipn → 501

Parts are managed through gitplm CSV files. Edit the CSVs directly.

---

## Categories

### GET /api/v1/categories

List all part categories with column schemas and part counts.

```bash
curl http://localhost:9000/api/v1/categories -b cookies.txt
```

### POST /api/v1/categories/:id/columns

Add a column to a category.

### DELETE /api/v1/categories/:id/columns/:colName

Remove a column.

---

## ECOs

### GET /api/v1/ecos

List all ECOs, optionally filtered by status.

**Query Parameters:** `status` (draft, review, approved, implemented, rejected)

```bash
curl http://localhost:9000/api/v1/ecos?status=draft -b cookies.txt
```

### GET /api/v1/ecos/:id

Get a single ECO with enriched part details for affected IPNs.

### POST /api/v1/ecos

Create an ECO. Auto-generates ID as `ECO-{YEAR}-{NNN}`.

**Fields:** `title` (required), `description`, `status` (default: draft), `priority` (default: normal), `affected_ipns` (JSON string array), `ncr_id` (links to an NCR)

```bash
curl -X POST http://localhost:9000/api/v1/ecos \
  -H "Content-Type: application/json" \
  -d '{"title":"Change resistor value","description":"R15 should be 10K","priority":"normal","affected_ipns":"[\"RES-001-0001\"]"}' \
  -b cookies.txt
```

### PUT /api/v1/ecos/:id

Update an ECO.

### POST /api/v1/ecos/:id/approve

Sets status to `approved`, records approver and timestamp.

### POST /api/v1/ecos/:id/implement

Sets status to `implemented`.

### POST /api/v1/ecos/bulk

Bulk operations on ECOs.

**Request Body:**

```json
{ "ids": ["ECO-2026-001", "ECO-2026-002"], "action": "approve" }
```

**Supported actions:** `approve`, `implement`, `reject`, `delete`

**Response:**

```json
{ "data": { "success": 2, "failed": 0, "errors": [] } }
```

```bash
curl -X POST http://localhost:9000/api/v1/ecos/bulk \
  -H "Content-Type: application/json" \
  -d '{"ids":["ECO-2026-001","ECO-2026-002"],"action":"approve"}' \
  -b cookies.txt
```

---

## Documents

### GET /api/v1/docs

List all documents.

### GET /api/v1/docs/:id

### POST /api/v1/docs

**Fields:** `title` (required), `category`, `ipn`, `revision` (default: A), `status` (default: draft), `content`, `file_path`

### PUT /api/v1/docs/:id

### POST /api/v1/docs/:id/approve

---

## Vendors

### GET /api/v1/vendors

### GET /api/v1/vendors/:id

### POST /api/v1/vendors

**Fields:** `name` (required), `website`, `contact_name`, `contact_email`, `contact_phone`, `notes`, `status` (default: active), `lead_time_days`

### PUT /api/v1/vendors/:id

### DELETE /api/v1/vendors/:id

---

## Inventory

### GET /api/v1/inventory

List all inventory items. Use `low_stock=true` to filter items at or below reorder point.

### GET /api/v1/inventory/:ipn

### POST /api/v1/inventory/transact

Create an inventory transaction. Types: `receive`, `issue`, `return`, `adjust`.

```bash
curl -X POST http://localhost:9000/api/v1/inventory/transact \
  -H "Content-Type: application/json" \
  -d '{"ipn":"CAP-001-0001","type":"receive","qty":500,"reference":"PO-2026-0003"}' \
  -b cookies.txt
```

### POST /api/v1/inventory/bulk

Bulk operations on inventory. **Supported actions:** `delete`

### GET /api/v1/inventory/:ipn/history

Transaction history for a specific IPN.

---

## Purchase Orders

### GET /api/v1/pos

List all POs.

### GET /api/v1/pos/:id

Returns PO with line items.

### POST /api/v1/pos

Create a PO with line items.

```bash
curl -X POST http://localhost:9000/api/v1/pos \
  -H "Content-Type: application/json" \
  -d '{"vendor_id":"V-001","expected_date":"2026-04-01","lines":[{"ipn":"CAP-001-0001","qty_ordered":500,"unit_price":0.02}]}' \
  -b cookies.txt
```

### PUT /api/v1/pos/:id

### POST /api/v1/pos/:id/receive

Receive line items. Auto-updates inventory and creates transactions.

```bash
curl -X POST http://localhost:9000/api/v1/pos/PO-2026-0002/receive \
  -H "Content-Type: application/json" \
  -d '{"lines":[{"id":2,"qty":250}]}' \
  -b cookies.txt
```

### POST /api/v1/pos/generate-from-wo

Generate a PO from work order shortages. Analyzes BOM needs vs inventory and creates a draft PO for items with shortages.

**Request Body:**

```json
{ "wo_id": "WO-2026-0001", "vendor_id": "V-001" }
```

**Response:**

```json
{ "data": { "po_id": "PO-2026-0005", "lines": 3 } }
```

```bash
curl -X POST http://localhost:9000/api/v1/pos/generate-from-wo \
  -H "Content-Type: application/json" \
  -d '{"wo_id":"WO-2026-0001","vendor_id":"V-001"}' \
  -b cookies.txt
```

---

## Work Orders

### GET /api/v1/workorders

### GET /api/v1/workorders/:id

### POST /api/v1/workorders

**Fields:** `assembly_ipn` (required), `qty` (default: 1), `status` (default: open), `priority` (default: normal), `notes`

### PUT /api/v1/workorders/:id

Setting status to `in_progress` auto-sets `started_at`. Setting to `completed` auto-sets `completed_at`.

### GET /api/v1/workorders/:id/bom

BOM availability check with shortage highlighting. Returns inventory levels, required quantities, shortage amounts, and status (`ok`, `low`, `shortage`) for each component.

**Response:**

```json
{
  "data": {
    "wo_id": "WO-2026-0001", "assembly_ipn": "PCB-001-0001", "qty": 10,
    "bom": [
      { "ipn": "CAP-001-0001", "description": "100nF MLCC", "qty_required": 10, "qty_on_hand": 500, "shortage": 0, "status": "ok" },
      { "ipn": "RES-001-0001", "description": "10K 0402", "qty_required": 10, "qty_on_hand": 5, "shortage": 5, "status": "low" }
    ]
  }
}
```

```bash
curl http://localhost:9000/api/v1/workorders/WO-2026-0001/bom -b cookies.txt
```

### GET /api/v1/workorders/:id/pdf

Generate a printable Work Order Traveler as HTML. Opens print dialog automatically. Includes assembly info, BOM table, and sign-off section.

```bash
curl http://localhost:9000/api/v1/workorders/WO-2026-0001/pdf -b cookies.txt
# Returns text/html — open in a browser to print
```

### POST /api/v1/workorders/bulk

Bulk operations on work orders.

**Supported actions:** `complete`, `cancel`, `delete`

```bash
curl -X POST http://localhost:9000/api/v1/workorders/bulk \
  -H "Content-Type: application/json" \
  -d '{"ids":["WO-2026-0001","WO-2026-0002"],"action":"complete"}' \
  -b cookies.txt
```

---

## Test Records

### GET /api/v1/tests

### GET /api/v1/tests/:serial_number

### POST /api/v1/tests

**Fields:** `serial_number` (required), `ipn` (required), `result` (required: pass/fail), `firmware_version`, `test_type`, `measurements` (JSON string), `notes`

---

## NCRs

### GET /api/v1/ncrs

### GET /api/v1/ncrs/:id

### POST /api/v1/ncrs

**Fields:** `title` (required), `description`, `ipn`, `serial_number`, `defect_type`, `severity` (default: minor), `status` (default: open)

### PUT /api/v1/ncrs/:id

Setting status to `resolved` or `closed` auto-sets `resolved_at`.

### POST /api/v1/ncrs/bulk

Bulk operations on NCRs. **Supported actions:** `close`, `resolve`, `delete`

---

## Devices

### GET /api/v1/devices

### GET /api/v1/devices/:serial_number

### POST /api/v1/devices

### PUT /api/v1/devices/:serial_number

### GET /api/v1/devices/:serial_number/history

Combined test records and firmware campaign participation.

### GET /api/v1/devices/export

Export all devices as CSV.

```bash
curl http://localhost:9000/api/v1/devices/export -b cookies.txt -o devices.csv
```

### POST /api/v1/devices/import

Import devices from CSV. Uses `multipart/form-data`. Upserts on `serial_number`.

**Required CSV columns:** `serial_number`, `ipn`
**Optional columns:** `firmware_version`, `customer`, `location`, `status`, `install_date`, `notes`

**Response:**

```json
{ "data": { "imported": 15, "skipped": 2, "errors": [] } }
```

```bash
curl -X POST http://localhost:9000/api/v1/devices/import \
  -F "file=@devices.csv" \
  -b cookies.txt
```

### POST /api/v1/devices/bulk

Bulk operations on devices. **Supported actions:** `decommission`, `delete`

---

## Firmware Campaigns

### GET /api/v1/campaigns

### GET /api/v1/campaigns/:id

### POST /api/v1/campaigns

**Fields:** `name` (required), `version` (required), `category` (default: public), `status` (default: draft), `target_filter`, `notes`

### PUT /api/v1/campaigns/:id

### POST /api/v1/campaigns/:id/launch

Enrolls all active devices and sets campaign to `active`.

### GET /api/v1/campaigns/:id/progress

Returns device counts by status.

### GET /api/v1/campaigns/:id/devices

List all devices in a campaign with their update status.

### GET /api/v1/campaigns/:id/stream

Server-Sent Events (SSE) stream of campaign progress. Sends progress JSON every 2 seconds until all devices are updated or failed.

**Response (text/event-stream):**

```
data: {"pending":5,"sent":2,"updated":3,"failed":0,"total":10,"pct":30}
```

```bash
curl -N http://localhost:9000/api/v1/campaigns/FW-2026-001/stream -b cookies.txt
```

### POST /api/v1/campaigns/:id/devices/:serial/mark

Mark a device's campaign status.

**Request Body:**

```json
{ "status": "updated" }
```

Status must be `updated` or `failed`.

```bash
curl -X POST http://localhost:9000/api/v1/campaigns/FW-2026-001/devices/SN-001/mark \
  -H "Content-Type: application/json" \
  -d '{"status":"updated"}' \
  -b cookies.txt
```

---

## RMAs

### GET /api/v1/rmas

### GET /api/v1/rmas/:id

### POST /api/v1/rmas

### PUT /api/v1/rmas/:id

### POST /api/v1/rmas/bulk

Bulk operations on RMAs. **Supported actions:** `close`, `delete`

---

## Quotes

### GET /api/v1/quotes

### GET /api/v1/quotes/:id

Returns quote with line items.

### POST /api/v1/quotes

### PUT /api/v1/quotes/:id

### GET /api/v1/quotes/:id/cost

Cost rollup — calculates line totals and grand total.

**Response:**

```json
{
  "data": {
    "lines": [ { "ipn": "PCB-001-0001", "description": "Z1000 Power Module", "qty": 50, "unit_price": 149.99, "line_total": 7499.50 } ],
    "total": 7499.50
  }
}
```

```bash
curl http://localhost:9000/api/v1/quotes/Q-2026-001/cost -b cookies.txt
```

### GET /api/v1/quotes/:id/pdf

Generate a printable quote as HTML. Includes customer info, line items with pricing, subtotal, terms, and contact info. Opens print dialog automatically.

```bash
curl http://localhost:9000/api/v1/quotes/Q-2026-001/pdf -b cookies.txt
# Returns text/html — open in a browser to print
```

---

## Prices (Supplier Price Catalog)

### GET /api/v1/prices/:ipn

Price history for an IPN, sorted newest first. Includes vendor name via join.

**Response:**

```json
{
  "data": [
    { "id": 1, "ipn": "CAP-001-0001", "vendor_id": "V-001", "vendor_name": "DigiKey", "unit_price": 0.025, "currency": "USD", "min_qty": 500, "lead_time_days": 3, "po_id": null, "recorded_at": "2026-02-18T06:00:00Z", "notes": null }
  ]
}
```

```bash
curl http://localhost:9000/api/v1/prices/CAP-001-0001 -b cookies.txt
```

### POST /api/v1/prices

Manually add a price entry.

**Request Body:**

```json
{ "ipn": "CAP-001-0001", "vendor_id": "V-001", "unit_price": 0.025, "min_qty": 500, "lead_time_days": 3 }
```

| Field | Required | Description |
|-------|----------|-------------|
| `ipn` | Yes | Internal Part Number |
| `vendor_id` | No | Vendor ID |
| `unit_price` | Yes | Price per unit (must be > 0) |
| `min_qty` | No | Minimum order quantity (default: 1) |
| `lead_time_days` | No | Vendor lead time in days |
| `notes` | No | Free-text notes |

```bash
curl -X POST http://localhost:9000/api/v1/prices \
  -H "Content-Type: application/json" \
  -d '{"ipn":"CAP-001-0001","vendor_id":"V-001","unit_price":0.025,"min_qty":500}' \
  -b cookies.txt
```

### DELETE /api/v1/prices/:id

Remove a price entry by ID.

```bash
curl -X DELETE http://localhost:9000/api/v1/prices/1 -b cookies.txt
```

### GET /api/v1/prices/:ipn/trend

Price trend data for charting, sorted by date ascending.

**Response:**

```json
{
  "data": [
    { "date": "2026-01-15", "price": 0.03, "vendor": "DigiKey" },
    { "date": "2026-02-18", "price": 0.025, "vendor": "DigiKey" }
  ]
}
```

```bash
curl http://localhost:9000/api/v1/prices/CAP-001-0001/trend -b cookies.txt
```

**Note:** Prices are also automatically recorded when PO lines are received (POST /api/v1/pos/:id/receive) if the line has a unit_price set.

---

## Email Configuration

### GET /api/v1/email/config

Returns SMTP configuration. Password is masked as "****" if set.

**Response:**

```json
{
  "data": { "id": 1, "smtp_host": "smtp.gmail.com", "smtp_port": 587, "smtp_user": "user@example.com", "smtp_password": "****", "from_address": "noreply@example.com", "from_name": "ZRP", "enabled": 1 }
}
```

```bash
curl http://localhost:9000/api/v1/email/config -b cookies.txt
```

### PUT /api/v1/email/config

Update SMTP configuration. Send password as "****" to keep existing password unchanged.

**Request Body:**

```json
{ "smtp_host": "smtp.gmail.com", "smtp_port": 587, "smtp_user": "user@example.com", "smtp_password": "app-password", "from_address": "noreply@example.com", "from_name": "ZRP", "enabled": 1 }
```

```bash
curl -X PUT http://localhost:9000/api/v1/email/config \
  -H "Content-Type: application/json" \
  -d '{"smtp_host":"smtp.gmail.com","smtp_port":587,"smtp_user":"user@example.com","smtp_password":"app-password","from_address":"noreply@example.com","enabled":1}' \
  -b cookies.txt
```

### POST /api/v1/email/test

Send a test email to verify configuration.

**Request Body:**

```json
{ "to": "recipient@example.com" }
```

```bash
curl -X POST http://localhost:9000/api/v1/email/test \
  -H "Content-Type: application/json" \
  -d '{"to":"recipient@example.com"}' \
  -b cookies.txt
```

### GET /api/v1/email-log

Returns recent email log entries (up to 100), newest first.

**Response:**

```json
{
  "data": [
    { "id": 1, "to_address": "user@example.com", "subject": "ZRP Test Email", "body": "...", "status": "sent", "error": "", "sent_at": "2026-02-17 22:00:00" }
  ]
}
```

```bash
curl http://localhost:9000/api/v1/email-log -b cookies.txt
```

### Settings Aliases

The following aliases are also available for the email configuration endpoints:

- `GET /api/v1/settings/email` → same as `GET /api/v1/email/config`
- `PUT /api/v1/settings/email` → same as `PUT /api/v1/email/config`
- `POST /api/v1/settings/email/test` → same as `POST /api/v1/email/test`

### Email Triggers

When email is enabled, ZRP automatically sends emails on these events:

- **ECO Approved** → Emails the ECO creator (or admin) when an ECO status changes to "approved"
- **Low Stock** → Emails admin when an inventory item drops below its reorder point after a transaction
- **Overdue Work Order** → Emails admin when a work order is updated and its due date has passed (status ≠ closed/completed)

---

## Dashboard Widgets

### GET /api/v1/dashboard/widgets

Returns all dashboard widgets with their position and enabled status.

**Response:**

```json
{
  "data": [
    { "id": 1, "user_id": 0, "widget_type": "kpi_open_ecos", "position": 0, "enabled": 1 },
    { "id": 2, "user_id": 0, "widget_type": "kpi_low_stock", "position": 1, "enabled": 1 }
  ]
}
```

```bash
curl http://localhost:9000/api/v1/dashboard/widgets -b cookies.txt
```

### PUT /api/v1/dashboard/widgets

Update widget positions and visibility. Send an array of widget updates.

**Request Body:**

```json
[
  { "widget_type": "kpi_open_ecos", "position": 0, "enabled": 1 },
  { "widget_type": "kpi_low_stock", "position": 1, "enabled": 0 }
]
```

**Available widget types:** `kpi_open_ecos`, `kpi_low_stock`, `kpi_open_pos`, `kpi_active_wos`, `kpi_open_ncrs`, `kpi_open_rmas`, `kpi_total_parts`, `kpi_total_devices`, `chart_eco_status`, `chart_wo_status`, `chart_inventory`

```bash
curl -X PUT http://localhost:9000/api/v1/dashboard/widgets \
  -H "Content-Type: application/json" \
  -d '[{"widget_type":"kpi_open_ecos","position":0,"enabled":1}]' \
  -b cookies.txt
```

---

## Config

### GET /api/v1/config

Returns application configuration.

**Response:**
```json
{
  "data": {
    "gitplm_ui_url": "http://localhost:8888"
  }
}
```

---

## Reports

All report endpoints support `?format=csv` to download CSV instead of JSON.

### GET /api/v1/reports/inventory-valuation

Inventory items with qty × latest PO unit price, grouped by category.

**Response:**
```json
{
  "data": {
    "groups": [
      {
        "category": "RES",
        "items": [
          { "ipn": "RES-001-0001", "description": "...", "category": "RES", "qty_on_hand": 100, "unit_price": 0.05, "subtotal": 5.00, "po_ref": "PO-2026-0001" }
        ],
        "subtotal": 5.00
      }
    ],
    "grand_total": 5.00
  }
}
```

### GET /api/v1/reports/open-ecos

Open ECOs (draft/review) sorted by priority (critical→low) with age in days.

**Response:**
```json
{
  "data": [
    { "id": "ECO-2026-001", "title": "...", "status": "draft", "priority": "critical", "created_by": "engineer", "created_at": "...", "age_days": 5 }
  ]
}
```

### GET /api/v1/reports/wo-throughput?days=30

Work order throughput for the last 30/60/90 days.

**Query Parameters:**
- `days` — 30, 60, or 90 (default: 30)

**Response:**
```json
{
  "data": {
    "days": 30,
    "count_by_status": { "completed": 5 },
    "total_completed": 5,
    "avg_cycle_time_days": 3.5
  }
}
```

### GET /api/v1/reports/low-stock

Items where qty_on_hand < reorder_point.

**Response:**
```json
{
  "data": [
    { "ipn": "RES-001-0001", "description": "...", "qty_on_hand": 5, "reorder_point": 50, "reorder_qty": 100, "suggested_order": 100 }
  ]
}
```

### GET /api/v1/reports/ncr-summary

Open NCR summary by severity and defect type with average resolution time.

**Response:**
```json
{
  "data": {
    "by_severity": { "minor": 2, "major": 1 },
    "by_defect_type": { "cosmetic": 1, "functional": 2 },
    "total_open": 3,
    "avg_resolve_days": 7.5
  }
}
```

---

## Supplier Prices

### GET /api/v1/supplier-prices

List/filter supplier price quotes.

**Query Parameters:**
- `ipn` — filter by IPN (exact match)
- `vendor` — filter by vendor name (partial match)
- `limit` — max results (default 100, max 1000)
- `offset` — pagination offset

**Response:**
```json
{
  "data": [
    { "id": 1, "ipn": "RES-001-0001", "vendor_name": "Acme Corp", "unit_price": 0.0523, "currency": "USD", "quantity_break": 100, "quote_date": "2026-01-15", "lead_time_days": 14, "notes": "...", "created_at": "..." }
  ],
  "meta": { "total": 1, "page": 1, "limit": 100 }
}
```

### POST /api/v1/supplier-prices

Create a new supplier price quote.

**Body:**
```json
{ "ipn": "RES-001-0001", "vendor_name": "Acme Corp", "unit_price": 0.0523, "currency": "USD", "quantity_break": 100, "quote_date": "2026-01-15", "lead_time_days": 14, "notes": "optional" }
```

Required: `ipn`, `vendor_name`, `unit_price` (> 0). Defaults: currency=USD, quantity_break=1.

### GET /api/v1/supplier-prices/:id

Get a single supplier price quote by ID.

### PUT /api/v1/supplier-prices/:id

Update a supplier price quote. Send only fields to change.

**Body (all optional):**
```json
{ "ipn": "...", "vendor_name": "...", "unit_price": 0.05, "currency": "EUR", "quantity_break": 250, "quote_date": "2026-02-01", "lead_time_days": 10, "notes": "updated" }
```

### DELETE /api/v1/supplier-prices/:id

Delete a supplier price quote.

### GET /api/v1/supplier-prices/trend

Get price trend data for charting.

**Query Parameters:**
- `ipn` (required) — IPN to get trend for

**Response:**
```json
{
  "data": [
    { "date": "2026-01-01", "price": 0.0523, "vendor": "Acme Corp", "quantity_break": 100 }
  ]
}
```
