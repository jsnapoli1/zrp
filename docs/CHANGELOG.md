# Changelog

All notable changes to ZRP are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/).

## [0.5.1] - 2026-02-18

### Added
- **Settings hub page** — unified tabbed settings page at `/settings` with sections for General, Email/SMTP, Distributor APIs, GitPLM, Backups, and Users/Auth
- **General settings API** — `GET/PUT /api/v1/settings/general` for app name, company info, currency, and date format
- General settings stored in `app_settings` table with `general_` key prefix

## [0.5.0] - 2026-02-18

### Added
- **Field Reports module** — full CRUD for tracking field failures, customer complaints, site visits, and installation notes
- Field report types: failure, complaint, visit, installation
- Status workflow: open → investigating → resolved → closed
- Priority levels: low, medium, high, critical
- Filter by status, priority, type, and date range
- Create NCR directly from a field report (POST /api/v1/field-reports/:id/create-ncr)
- Link field reports to ECOs for corrective action tracking
- Detail view with edit mode, status workflow buttons, and linked NCR/ECO display
- Go tests for all CRUD endpoints and NCR creation
- Vitest tests for FieldReports list page component

## [0.4.0] - 2026-02-18

### Added
- **Real Digikey API v4 integration** — OAuth2 client credentials flow, keyword search via POST /products/v4/search/keyword
- **Real Mouser API v2 integration** — Part number search via POST /api/v2/search/partnumber
- Separate `digikey.go` and `mouser.go` files with interface-based HTTP clients
- Distributor settings now use Digikey client_id + client_secret (OAuth2) instead of API key
- "Not configured" message on Market Pricing section when no API keys are set, with link to settings
- `not_configured` field in market pricing API response
- `hasDistributorKeys()` helper function
- API error details returned per-distributor in `errors` response field
- Helper text with links to developer portals on Distributor Settings page

### Changed
- Removed mock/demo distributor clients — no fallback to fake data when keys are missing
- Digikey settings: replaced `api_key` field with `client_secret` for OAuth2 flow
- Market pricing handler returns structured error info per distributor

### Fixed
- Market pricing no longer returns fake data when API keys are not configured

## [0.3.3] - 2026-02-18

### Added
- Digikey (v4 Product Search, OAuth2) and Mouser (v2 Search API) real API integration for live pricing
- Market Pricing section on Part Detail page showing price breaks, stock levels, lead times per distributor
- 24-hour caching of market pricing results in `market_pricing` DB table
- Refresh button to re-fetch live pricing on demand
- Admin Distributor Settings page (`/distributor-settings`) for configuring API keys (stored in DB, not env vars)
- API endpoints: `GET /api/v1/parts/:ipn/market-pricing`, `POST /api/v1/settings/digikey`, `POST /api/v1/settings/mouser`, `GET /api/v1/settings/distributors`
- Full test coverage for market pricing backend and frontend components

## [0.3.2] - 2026-02-18

### Added
- Enhanced supplier RFQ workflow with full lifecycle: Draft → Sent → Awarded → Closed
- Multi-supplier RFQ: send same RFQ to multiple vendors, compare responses side-by-side
- RFQ line items linked to parts/BOM with per-line vendor quote tracking
- Per-line award: select winning vendor per line item, auto-creates POs per vendor
- Whole-RFQ award: award all lines to single vendor with auto PO creation
- RFQ email body generation for copy-to-clipboard vendor communication
- RFQ dashboard API with open RFQs, pending responses, and monthly award stats
- Frontend: RFQ list page with create dialog
- Frontend: RFQ detail page with tabbed sections (Lines, Vendors, Responses, Compare, Award)
- Side-by-side vendor comparison matrix view
- Quote entry dialog with vendor/line item selection
- RFQ sidebar navigation under Supply Chain
- API endpoints: close, email, award-lines, rfq-dashboard
- 11 backend tests covering all RFQ workflows (CRUD, send, close, quotes, compare, award, per-line award, dashboard, email)

## [0.3.1] - 2026-02-18

### Fixed
- Calendar page now fetches from real API instead of showing template error
- Devices page uses inline modal dialog instead of navigating to separate page
- Docker healthcheck uses `/healthz` endpoint (bypasses auth)
- Healthcheck command syntax corrected in Portainer stack
- Portainer stack updated to build from Dockerfile via git repo
- Hover states for sidebar, buttons, and table rows

## [0.3.0] - 2026-02-18

### Added
- **Email Notifications — Event Triggers & Email Log**
  - New `email_log` table records all sent/failed emails with recipient, subject, status, error, and timestamp
  - Email Log table visible on the Email Settings page
  - `GET /api/v1/email-log` — list recent email log entries
  - Settings aliases: `GET/PUT /api/v1/settings/email`, `POST /api/v1/settings/email/test`
  - ECO approval trigger: emails the ECO creator when an ECO is approved
  - Low stock trigger: emails admin when inventory drops below reorder point after a transaction
  - Overdue work order trigger: emails admin when a WO with a past due date is updated
  - `due_date` column added to work_orders table
  - `email` column added to users table
  - Audit logging on email config save and test send
  - Go unit tests for all email handlers and triggers (mock SMTP)

### Added
- **Supplier Price Catalog** — new module (`/supplier-prices`) under Supply Chain for tracking vendor price quotes per IPN
  - Full CRUD API: `GET/POST/PUT/DELETE /api/v1/supplier-prices`, plus `/trend` endpoint for chart data
  - Sortable price table with "Best Price" highlight (green) per IPN
  - Add Price Quote modal with IPN autocomplete from parts database
  - Price history view with SVG line chart showing price trends per vendor over time
  - Parts detail modal integration: new "📊 Price Quotes" tab
  - Audit logging on create/update/delete
  - Go unit tests for all handlers (CRUD, validation, trend, edge cases)


### Added
- **Supplier Price Catalog** — track price history per IPN across vendors with automatic recording from PO receipts:
  - `GET /api/v1/prices/:ipn` — price history sorted newest first
  - `POST /api/v1/prices` — manually add price entries
  - `DELETE /api/v1/prices/:id` — remove entries
  - `GET /api/v1/prices/:ipn/trend` — trend data for charting
  - Pricing tab in part detail modal with history table, SVG sparkline, and "Add Price" form
  - Best price highlighted with green badge
  - Automatic price recording when PO lines are received
- **Email Notifications** — SMTP-based email alerts for system notifications:
  - `GET /api/v1/email/config` — view SMTP config (password masked)
  - `PUT /api/v1/email/config` — update SMTP settings
  - `POST /api/v1/email/test` — send test email
  - Email Settings page under Admin (📧 Email Settings sidebar link)
  - Background goroutine sends emails for new notifications when enabled
  - Each notification emailed only once (tracked by `emailed` column)
- **Custom Dashboard Widgets** — configurable KPI cards and charts:
  - `GET /api/v1/dashboard/widgets` — list widgets with positions
  - `PUT /api/v1/dashboard/widgets` — update positions and visibility
  - "Customize" button opens drag-to-reorder modal with toggle switches
  - 11 widgets: 8 KPI cards + 3 charts, all individually hideable
  - Settings persist server-side
- **Report Builder** — new Reports page with 5 built-in reports, all exportable to CSV:
  - Inventory Valuation (qty × latest PO price, grouped by category)
  - Open ECOs by Priority (sorted critical→low, with age in days)
  - WO Throughput (30/60/90 day windows, count by status, avg cycle time)
  - Low Stock Report (items below reorder point, suggested order qty)
  - NCR Summary (by severity and defect type, avg resolve time)
- **Quote Margin Analysis** — new "Margin Analysis" tab in quote detail showing BOM cost vs quoted price per line item with color-coded margin % (green >50%, yellow 20-50%, red <20%)
- **gitplm-ui Deep Links** — parts table and detail modal link directly to gitplm-ui; configurable via `--gitplm-ui` flag
- **Config endpoint** — `GET /api/v1/config` returns configurable settings (gitplm_ui_url)

## [0.2.0] - 2026-02-18

### Added
- **Authentication** — session-based login/logout with bcrypt password hashing, 24-hour session tokens, and role-based access control (admin, user, readonly)
- **User Management** — admin panel for creating, editing, deactivating users and resetting passwords
- **API Keys** — generate Bearer tokens for programmatic access with optional expiration, enable/disable toggle, and automatic last-used tracking
- **Readonly Role Enforcement** — readonly users can view all data but POST/PUT/DELETE requests return 403
- **Notifications** — automatic notification generation for low stock, overdue work orders (>7 days), aging NCRs (>14 days), and new RMAs; bell icon with unread count; mark-as-read
- **File Attachments** — upload files (up to 32MB) to any module record; file serving at `/files/`; delete with disk cleanup
- **Audit Log** — all create/update/delete/bulk operations logged with username, action, module, record ID, and summary; filterable by module, user, and date range
- **Global Search** — search across parts, ECOs, work orders, devices, NCRs, POs, and quotes simultaneously from the top bar
- **Calendar View** — monthly calendar showing WO due dates (blue), PO expected deliveries (green), and quote expirations (orange)
- **Dashboard Charts** — ECOs by status, work orders by status, and top inventory items by value
- **Dashboard Low Stock Panel** — dedicated view of items below reorder point
- **Bulk Operations** — multi-select with checkboxes for ECOs (approve/implement/reject/delete), work orders (complete/cancel/delete), NCRs (close/resolve/delete), devices (decommission/delete), RMAs (close/delete), and inventory (delete)
- **Work Order PDF Traveler** — printable HTML traveler with assembly info, BOM table, and sign-off section
- **Quote PDF** — printable HTML quote with customer info, line items, subtotal, terms
- **WO BOM Shortage Highlighting** — color-coded status (ok/low/shortage) for each BOM component
- **Parts BOM & Cost Rollup** — view Bill of Materials and cost breakdown for assembly IPNs
- **ECO→Parts Enrichment** — affected IPNs enriched with part details when viewing an ECO
- **NCR→ECO Auto-Link** — ECOs can reference an NCR ID for traceability
- **Generate PO from WO Shortages** — automatically create draft POs for work order BOM shortages
- **Device Import/Export** — CSV import (upsert on serial number) and export for the device registry
- **Campaign SSE Stream** — real-time Server-Sent Events for firmware campaign progress monitoring
- **Campaign Device Marking** — mark individual devices as updated or failed during rollout
- **Campaign Device Listing** — view all devices enrolled in a campaign with status
- **IPN Autocomplete** — suggested IPNs when entering part numbers in inventory and other modules
- **Dark Mode** — toggle with localStorage persistence
- **Keyboard Shortcuts** — `/` or `Ctrl+K` for search, `n` for new record, `Escape` to close, `?` for help

### Changed
- All API endpoints now require authentication (session cookie or Bearer token), except `/auth/*`, `/static/*`, `/files/*`
- API responses include 401/403 status codes for auth failures
- CORS headers include Authorization in allowed headers

## [0.1.0] - 2026-02-18

### Added
- Single-binary Go server with embedded SQLite (WAL mode)
- SPA frontend with Tailwind CSS and hash-based routing
- **Dashboard** with 8 KPI cards (open ECOs, low stock, active WOs, open NCRs/RMAs, etc.)
- **Parts (PLM)** — read-only browsing of gitplm CSV files with category filtering, full-text search, pagination
- **ECOs** — engineering change order lifecycle (draft → review → approved → implemented)
- **Documents** — revision-controlled documents with approval workflow
- **Inventory** — per-IPN stock tracking with reorder points, locations, and full transaction history
- **Purchase Orders** — PO lifecycle with line items and partial receiving (auto-updates inventory)
- **Vendors** — supplier directory with contacts, lead times, and status
- **Work Orders** — production tracking with BOM availability checks
- **Test Records** — factory test results with serial numbers, measurements, and pass/fail
- **NCRs** — non-conformance reports with defect classification, root cause, and corrective actions
- **Device Registry** — field device tracking with firmware versions, customers, and history
- **Firmware Campaigns** — OTA rollout management with per-device progress tracking
- **RMAs** — return processing from complaint through resolution
- **Quotes** — customer quotes with line items and cost rollup
- CORS support for cross-origin API access
- Request logging middleware
- Seed data for all modules (demo-ready out of the box)
- Auto-generated IDs with year prefix (ECO-2026-001, PO-2026-0001, etc.)

## 2026-02-18

### Added
- **Pricing Management Page**: Centralized product pricing with cost analysis and margin tracking
  - New DB tables: `product_pricing`, `cost_analysis`
  - CRUD endpoints: GET/POST /api/v1/pricing, GET/PUT/DELETE /api/v1/pricing/:id
  - Cost analysis: GET/POST /api/v1/pricing/analysis
  - Price history: GET /api/v1/pricing/history/:ipn
  - Bulk price update: POST /api/v1/pricing/bulk-update
  - Frontend: Pricing list with color-coded margins (red <15%, yellow 15-30%, green >30%)
  - Create/edit pricing tiers (standard, volume, distributor, OEM)
  - Cost analysis tab with BOM cost breakdown vs selling price
  - Bulk price update with percentage or absolute adjustments
  - Replaced PlaceholderPage in App.tsx
