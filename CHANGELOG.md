# Changelog

All notable changes to ZRP are documented here.

## [0.2.0] - 2026-02-18

### 🚀 Major: React Frontend
- **Complete React + TypeScript + shadcn/ui frontend** replacing vanilla JS SPA
- 31 page components covering all 20+ modules
- Vite build system with code-splitting (lazy-loaded routes)
- shadcn/ui component library (New York style, zinc theme)
- Cmd+K global search, dark mode toggle, collapsible sidebar
- Typed API client with full TypeScript interfaces
- Responsive layout for desktop and mobile

### Pages Built
- **Engineering**: Parts (with BOM tree), ECOs (with status workflow), Documents (with upload)
- **Supply Chain**: Inventory (low stock alerts, quick receive), Purchase Orders (line items, status workflow), Vendors (price catalog)
- **Manufacturing**: Work Orders (BOM vs inventory shortage highlighting), Test Records, NCRs (severity badges, ECO linking)
- **Field & Service**: Devices (CSV import/export), Firmware Campaigns (progress polling), RMAs
- **Sales**: Quotes (margin analysis, PDF export)
- **Admin**: Users (role management), Audit Log (filterable), API Keys (generate/revoke), Email Settings (SMTP config), Calendar, Reports

### Infrastructure
- Vite dev server proxies `/api/*` to Go backend
- Go backend serves `frontend/dist/` in production
- Code-split bundles: ~410KB main + 31 lazy chunks (3-11KB each)
- Git repository cleaned (removed tracked binaries, 216MB → 16MB)

### White-Label (from 0.1.0-beta)
- All vendor references removed
- Company info configurable via `ZRP_COMPANY_NAME`, `ZRP_COMPANY_EMAIL` env vars
- MIT license, default password "changeme"
- 283 Playwright E2E tests

## [0.1.0-beta] - 2026-02-18

### Added
- Initial release: 20+ ERP modules (Parts, ECOs, Inventory, Procurement, Vendors, Work Orders, NCRs, Testing, RMAs, Devices, Firmware, Quotes, Calendar, Reports, Audit, Users, API Keys, Email, Documents, Dashboard)
- Authentication with session tokens
- SQLite database
- REST API (`/api/v1/*`)
- Supplier price catalog with trend charts
- Email notifications (ECO approval, low stock, overdue WO)
- File attachments for ECOs, NCRs, RMAs
- PDF export for Work Order travelers and Quotes
- Dark mode, keyboard shortcuts, global search
- Onboarding tour (Driver.js)
- Gitea CI workflow
