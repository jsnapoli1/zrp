# ZRP — Resource Planning

A single-binary ERP system for hardware electronics manufacturing. Go backend, React frontend, SQLite database. No dependencies to deploy — just run the binary.

```
┌──────────────────────────────────────────────┐
│  ZRP                                    ERP  │
├──────────────┬───────────────────────────────┤
│ ▸ Dashboard  │  Dashboard                    │
│              │  ┌──────┐┌──────┐┌──────┐     │
│ ENGINEERING  │  │Open  ││Low   ││Active│     │
│ ▸ Parts      │  │ECOs  ││Stock ││WOs   │     │
│ ▸ ECOs       │  │  2   ││  1   ││  1   │     │
│ ▸ Documents  │  └──────┘└──────┘└──────┘     │
│              │  ┌──────┐┌──────┐┌──────┐     │
│ SUPPLY CHAIN │  │Open  ││Open  ││Total │     │
│ ▸ Inventory  │  │NCRs  ││RMAs  ││Parts │     │
│ ▸ POs        │  │  1   ││  1   ││ 150  │     │
│ ▸ Vendors    │  └──────┘└──────┘└──────┘     │
│              │                                │
│ MANUFACTURING│  Welcome to ZRP               │
│ ▸ Work Orders│  Resource Planning —     │
│ ▸ Testing    │  your complete ERP for         │
│ ▸ NCRs       │  hardware manufacturing.       │
│              │                                │
│ FIELD        │                                │
│ ▸ Devices    │                                │
│ ▸ Firmware   │                                │
│ ▸ RMAs       │                                │
│              │                                │
│ SALES        │                                │
│ ▸ Quotes     │                                │
└──────────────┴───────────────────────────────┘
```

## Features

**20+ modules** covering the full hardware product lifecycle:

### Core Modules
- **Dashboard** — KPI cards, charts (ECOs/WOs by status), low stock alerts, calendar view
- **Parts (PLM)** — Browse/search gitplm CSV files with BOM & cost rollup for assemblies
- **ECOs** — Engineering Change Orders with approval workflow and parts enrichment
- **Documents** — Revision-controlled procedures, specs, and drawings with file attachments
- **Inventory** — Stock tracking with reorder points, transaction history, and IPN autocomplete
- **Purchase Orders** — PO lifecycle with partial receiving and auto-generate from WO shortages
- **Vendors** — Supplier directory with contacts and lead times
- **Work Orders** — Production tracking with BOM shortage highlighting and printable PDF travelers
- **Test Records** — Factory test results with pass/fail, measurements, and firmware versions
- **NCRs** — Non-Conformance Reports with NCR→ECO auto-linking for traceability
- **Device Registry** — Field device tracking with CSV import/export and full lifecycle history
- **Firmware Campaigns** — OTA rollouts with live SSE progress streaming
- **RMAs** — Return processing from complaint through resolution
- **Quotes** — Customer quotes with cost rollup and printable PDF quotes

### Platform Features
- **Authentication** — Session-based login with bcrypt, role-based access (admin/user/readonly)
- **API Keys** — Bearer token auth for scripts and CI pipelines
- **Global Search** — Search across all modules from the top bar
- **Notifications** — Auto-generated alerts for low stock, overdue WOs, aging NCRs, new RMAs
- **File Attachments** — Upload files to any record (up to 32MB)
- **Audit Log** — Full audit trail of all changes, filterable by module/user/date
- **Bulk Operations** — Multi-select and batch actions across all list views
- **Calendar** — Monthly view of WO due dates, PO deliveries, quote expirations
- **Dark Mode** — Toggle with persistent preference
- **Keyboard Shortcuts** — `/` to search, `n` for new, `?` for help

## Quick Start

```bash
git clone https://github.com/yourusername/zrp.git
cd zrp
go build -o zrp .
./zrp --pmDir /path/to/gitplm/parts/database
# Open http://localhost:9000
```

## Frontend

**React frontend** in `frontend/` is the primary UI — fully responsive, modern components with shadcn/ui and Tailwind CSS. The legacy templates in `templates/` and `static/` are kept for backward compatibility but the React app provides the main user experience.

## Requirements

- Go 1.24+ (build only — the binary is self-contained)
- No external database required (uses embedded SQLite)
- Tested on macOS (arm64) and Linux (amd64)

## Installation

```bash
# From source
git clone https://github.com/yourusername/zrp.git
cd zrp
go build -o zrp .

# Or install directly
go install github.com/yourusername/zrp@latest
```

## Configuration

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-pmDir` | `""` | Path to gitplm parts database directory (CSV files) |
| `-port` | `9000` | HTTP port to listen on |
| `-db` | `zrp.db` | Path to SQLite database file |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ZRP_COMPANY_NAME` | `Your Company` | Company name shown on PDF quotes and work order travelers |
| `ZRP_COMPANY_EMAIL` | `admin@example.com` | Contact email shown on PDF quotes |

Example:
```bash
ZRP_COMPANY_NAME="Acme Corp" ZRP_COMPANY_EMAIL="sales@acme.com" \
  ./zrp -pmDir ./parts-database -port 8080 -db /var/data/zrp.db
```

## Running

```bash
./zrp --pmDir /path/to/parts
```

Open [http://localhost:9000](http://localhost:9000) in your browser. The database is created automatically on first run with seed data.

## Modules

### Dashboard
Summary cards showing open ECOs, low stock items, open POs, active work orders, open NCRs, open RMAs, total parts, and total devices. Click any card to jump to that module.

### Parts (PLM)
Reads parts data from [gitplm](https://github.com/git-plm/gitplm) CSV files on disk. gitplm is an open-source parts library manager by [Cliff Brake](https://github.com/cbrake) that stores component data as plain CSV files in Git — no database required. ZRP integrates directly with these CSV files, providing a web UI for browsing, searching, and viewing BOM trees. Parts are read-only through the API — edit the CSV files directly (or use the gitplm CLI) and changes are reflected immediately.

### ECOs (Engineering Change Orders)
Track proposed changes to parts and assemblies. Workflow: `draft` → `review` → `approved` → `implemented`. Each ECO can reference affected IPNs and tracks who approved it and when.

### Documents
Revision-controlled documentation: procedures, specs, test plans. Each document has a category, optional IPN link, revision letter, and markdown content. Supports `draft` → `approved` workflow.

### Inventory
Per-IPN stock tracking with quantities on hand, reserved quantities, bin locations, and reorder points. Transaction history records every receive, issue, return, and adjustment. Automatically updated when POs are received.

### Purchase Orders
Full PO lifecycle: `draft` → `sent` → `partial` → `received`. Each PO has line items with IPN, MPN, manufacturer, quantities, and unit pricing. Receiving a PO automatically creates inventory transactions and updates stock levels.

### Vendors
Supplier database with contact info, website, lead time in days, and status (`active`, `preferred`, `inactive`).

### Work Orders
Production tracking for assemblies. Each WO specifies an assembly IPN, quantity, and priority. Status flow: `open` → `in_progress` → `completed`. BOM availability endpoint checks inventory against required components.

### Test Records
Factory test results by serial number. Records include IPN, firmware version, test type, pass/fail result, JSON measurements, and tester identity. Queryable by serial number to see full test history.

### NCRs (Non-Conformance Reports)
Quality issue tracking with defect type classification (workmanship, component, design), severity levels (minor, major, critical), root cause analysis, and corrective actions. Links to specific IPNs and serial numbers.

### Device Registry
Field-deployed device tracking. Each device has a serial number, IPN, firmware version, customer, location, and status. History endpoint aggregates test records and firmware campaign participation.

### Firmware Campaigns
Manage OTA firmware rollouts. Create a campaign targeting a firmware version, launch it to automatically enroll all active devices, and track progress (pending, sent, updated, failed).

### RMAs
Return processing from customer complaint through diagnosis to resolution. Tracks serial number, customer, reason, defect description, and resolution. Status flow: `open` → `received` → `diagnosing` → `repaired` → `shipped` → `closed`.

### Quotes
Customer quotes with line items. Each line has an IPN, description, quantity, and unit price. Cost rollup endpoint calculates line totals and grand total. Status: `draft` → `sent` → `accepted`/`declined`/`expired`.

## API

All endpoints are under `/api/v1/`. Requests and responses use JSON. The standard response envelope is:

```json
{
  "data": { ... },
  "meta": { "total": 100, "page": 1, "limit": 50 }
}
```

See [docs/API.md](docs/API.md) for the complete API reference.

## Development

### Project Structure

```
zrp/
├── main.go              # HTTP server, routing, response helpers
├── db.go                # SQLite init, migrations, seed data, ID generation
├── types.go             # All Go struct types and API response types
├── middleware.go         # Auth middleware (session + Bearer), CORS, logging
├── audit.go             # Audit logging, dashboard charts, low stock alerts
├── handler_auth.go      # Login, logout, session (me) handlers
├── handler_users.go     # User management (CRUD, password reset)
├── handler_apikeys.go   # API key generation, validation, revocation
├── handler_parts.go     # Parts + Categories + Dashboard handlers
├── handler_eco.go       # ECO handlers
├── handler_docs.go      # Document handlers
├── handler_vendors.go   # Vendor handlers
├── handler_inventory.go # Inventory + transaction handlers
├── handler_procurement.go # PO handlers with receiving + generate-from-WO
├── handler_workorders.go  # Work order + BOM + PDF traveler handlers
├── handler_testing.go   # Test record handlers
├── handler_ncr.go       # NCR handlers
├── handler_devices.go   # Device registry + import/export handlers
├── handler_firmware.go  # Firmware campaign + SSE stream handlers
├── handler_rma.go       # RMA handlers
├── handler_quotes.go    # Quote + cost rollup + PDF handlers
├── handler_costing.go   # BOM cost rollup (shared logic)
├── handler_bulk.go      # Bulk operations for all modules
├── handler_search.go    # Global search across all modules
├── handler_calendar.go  # Calendar event aggregation
├── handler_notifications.go # Notification generation and management
├── handler_attachments.go   # File upload, list, delete
├── static/
│   ├── index.html       # SPA shell with Tailwind CSS
│   └── modules/         # One JS file per module
│       ├── dashboard.js
│       ├── parts.js
│       ├── eco.js
│       ├── docs.js
│       ├── inventory.js
│       ├── procurement.js
│       ├── vendors.js
│       ├── workorders.js
│       ├── testing.js
│       ├── ncr.js
│       ├── devices.js
│       ├── firmware.js
│       ├── rma.js
│       ├── quotes.js
│       └── costing.js
└── zrp.db               # SQLite database (auto-created)
```

### Running Tests

```bash
go test ./...
```

### Build Verification

Before committing or deploying, run the verification script to ensure all checks pass:

```bash
./scripts/verify.sh
```

This script runs:
- ✓ Backend build (`go build ./...`)
- ✓ Backend tests (`go test ./...`)
- ✓ Frontend tests (`npx vitest run`)
- ✓ Frontend build (`npm run build` — catches TypeScript errors)

The script exits with a non-zero status if any check fails, making it ideal for pre-commit hooks or CI pipelines.

**Rebuild and restart workflow:**

After making changes, use this script to verify, rebuild, and restart the server:

```bash
./scripts/rebuild-and-restart.sh
```

This script:
1. Runs full verification
2. Rebuilds the frontend
3. Rebuilds the backend binary
4. Kills the existing ZRP server
5. Starts the new server
6. Verifies the server is healthy

**Continuous Integration:**

The `.github/workflows/ci.yml` workflow runs on every push and pull request, ensuring:
- All backend code compiles
- All backend tests pass
- All frontend tests pass
- Frontend builds without TypeScript errors

No broken code reaches the main branch.

### Building

```bash
go build -o zrp .
```

## Contributing

See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md).

## Acknowledgments

- [gitplm](https://github.com/git-plm/gitplm) by [Cliff Brake](https://github.com/cbrake) — the open-source Git-based parts library manager that ZRP integrates with for parts/BOM data
- [Driver.js](https://driverjs.com/) — lightweight onboarding tour library
- [Chart.js](https://www.chartjs.org/) — dashboard charts
- [Tailwind CSS](https://tailwindcss.com/) — utility-first CSS framework

## License

MIT
