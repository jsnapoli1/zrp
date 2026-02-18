# ZRP Modules Guide

## Authentication

ZRP requires authentication. On first launch, a default admin account is created:

- **Username:** `admin`
- **Password:** `changeme`

### Login

Enter your credentials on the login screen. Sessions last 24 hours.

### Roles

| Role | Can View | Can Create/Edit | Can Manage Users |
|------|----------|-----------------|------------------|
| Admin | Everything | Everything | Yes |
| User | Everything | Everything | No |
| Read-only | Everything | Nothing | No |

### Logout

Click your username in the top-right corner and select "Logout."

---

## User Management

*Admin only.* Access from the gear icon in the sidebar.

- **Create users:** Set username, display name, password, and role
- **Edit users:** Change display name, role, or active status
- **Deactivate:** Set a user to inactive — they can no longer log in (admins cannot deactivate themselves)
- **Reset password:** Set a new password for any user

---

## API Keys

Generate API keys for programmatic access (scripts, CI pipelines, integrations).

- **Create:** Give the key a name and optional expiration date
- **Copy key:** The full key is shown only once — store it securely
- **Use:** Send as `Authorization: Bearer zrp_...` header
- **Disable/Enable:** Toggle a key without deleting it
- **Revoke:** Permanently delete a key

---

## Dashboard

The landing page. Shows eight KPI cards:

| Card | What it shows |
|------|---------------|
| Open ECOs | ECOs not yet implemented or rejected |
| Low Stock | Items at or below reorder point |
| Open POs | Purchase orders not received or cancelled |
| Active WOs | Work orders that are open or in progress |
| Open NCRs | Quality issues not yet resolved |
| Open RMAs | Customer returns still being processed |
| Total Parts | Count of all parts in the gitplm database |
| Total Devices | Count of registered field devices |

Click any card to jump to that module.

### Charts

The dashboard includes visual charts:
- **ECOs by status** — bar/pie chart of draft, review, approved, implemented
- **Work orders by status** — open, in progress, completed
- **Top inventory by value** — highest-value items in stock

### Low Stock Alerts

A dedicated panel showing items where quantity on hand has fallen below the reorder point.

---

## Notifications

The bell icon in the top bar shows unread notifications. ZRP automatically generates notifications for:

| Type | Trigger | Severity |
|------|---------|----------|
| Low Stock | Qty on hand < reorder point | Warning |
| Overdue WO | In progress > 7 days | Warning |
| Open NCR | Open > 14 days | Error |
| New RMA | Created in last hour | Info |

Notifications are deduplicated (same type + record only once per 24 hours). Click a notification to navigate to the relevant record. Click the checkmark to mark as read.

---

## Global Search

The search bar at the top of every page searches across **all modules** simultaneously:

- **Parts** — IPN, manufacturer, MPN, all field values
- **ECOs** — ID, title, description
- **Work Orders** — ID, assembly IPN
- **Devices** — serial number, IPN, customer
- **NCRs** — ID, title
- **Purchase Orders** — ID
- **Quotes** — ID, customer

Type at least one character and results appear grouped by module. Click any result to navigate to it.

---

## Parts (PLM)

**What it's for:** Browsing and searching your parts database. Parts are managed externally in gitplm CSV files — ZRP reads them and provides a searchable web interface.

**Key concepts:**
- **IPN** (Internal Part Number) — your unique identifier for each part
- **Category** — groups parts by type (capacitors, resistors, connectors, etc.)
- **Fields** — each category has its own columns from the CSV headers

**Common workflows:**

1. **Find a part:** Type in the search box or filter by category. Search matches against IPN and all field values.
2. **View details:** Click a part row to see all its fields.
3. **View BOM:** For assemblies, view the Bill of Materials showing sub-components.
4. **View cost:** See BOM cost rollup for assemblies.

**IPN Autocomplete:** When entering an IPN in other modules (inventory, work orders, etc.), the system suggests matching IPNs as you type.

**Integration:** Parts IPNs are referenced by Inventory, Work Orders, Purchase Orders, Documents, NCRs, Quotes, and ECOs.

---

## ECOs (Engineering Change Orders)

**What it's for:** Tracking proposed changes to parts, assemblies, or processes before they're implemented.

**Status workflow:** `draft` → `review` → `approved` → `implemented`

**Common workflows:**

1. **Propose a change:** Create a new ECO with a title, description, priority, and list of affected IPNs.
2. **Submit for review:** Update the status to `review`.
3. **Approve:** Click the approve action. Records who approved and when.
4. **Implement:** After changes are made, mark as implemented.

**ECO→Parts enrichment:** When viewing an ECO, affected IPNs are enriched with part details (description, manufacturer, MPN) from the parts database.

**NCR→ECO auto-link:** ECOs can reference an NCR ID. When creating an ECO from an NCR, the link is maintained for traceability.

**Batch operations:** Select multiple ECOs with checkboxes, then bulk approve, implement, reject, or delete.

---

## Documents

**What it's for:** Managing revision-controlled engineering documents — assembly procedures, test specs, drawings, process instructions.

**Common workflows:**

1. **Create a document:** Give it a title, category, optional IPN link, and write the content.
2. **Edit and revise:** Update content and bump the revision letter.
3. **Approve:** Move from draft to approved when reviewed.
4. **Attach files:** Upload supporting files (PDFs, images) via the attachments panel.

---

## Inventory

**What it's for:** Tracking how much stock you have of each part, where it's stored, and when to reorder.

**Key concepts:**
- **Qty On Hand** — physical count in stock
- **Qty Reserved** — allocated to work orders
- **Reorder Point** — when on-hand drops to this level, a notification is generated
- **Transaction History** — every receive, issue, return, and adjustment is logged

**IPN Autocomplete:** When entering an IPN for a transaction, matching IPNs from the parts database are suggested.

**Integration:** Automatically updated when POs are received. Referenced by Work Order BOM checks. Low stock triggers dashboard alerts and notifications.

---

## Purchase Orders

**What it's for:** Ordering parts from vendors and tracking deliveries.

**Status workflow:** `draft` → `sent` → `partial` → `received`

**Common workflows:**

1. **Create a PO:** Select a vendor, add line items with part numbers and quantities.
2. **Generate from WO:** Automatically create a PO for work order shortages (see Procurement below).
3. **Send to vendor:** Update status to `sent`.
4. **Receive shipment:** Use the receive action with quantities for each line. Inventory updates automatically.

### Generate PO from Work Order Shortages

When a work order has BOM shortages, you can automatically generate a draft PO:

1. Open the work order and view the BOM
2. Click "Generate PO from Shortages"
3. Select a vendor
4. A draft PO is created with line items for all short components

---

## Vendors

**What it's for:** Keeping a directory of your suppliers with contact information and lead times.

---

## Work Orders

**What it's for:** Tracking production runs — building assemblies from components.

**Status workflow:** `open` → `in_progress` → `completed`

**Common workflows:**

1. **Plan production:** Create a WO specifying the assembly and quantity.
2. **Check materials:** Use the BOM view to verify all components are in stock.
3. **Start production:** Update status to `in_progress` (auto-records start time).
4. **Complete:** Update status to `completed` (auto-records completion time).

### BOM Shortage Highlighting

The BOM view color-codes each component:
- **Green (ok):** Sufficient stock
- **Yellow (low):** Some stock but not enough for the full build
- **Red (shortage):** Zero stock

### PDF Traveler

Click "Print Traveler" to generate a printable Work Order Traveler with:
- Assembly information and notes
- Full BOM table (IPN, description, MPN, manufacturer, qty, ref des)
- Sign-off section (kitted by, built by, tested by, QA approved by)

The traveler opens in a new tab with the print dialog ready.

**Batch operations:** Select multiple WOs to bulk complete, cancel, or delete.

---

## Test Records

**What it's for:** Recording factory test results for individual units identified by serial number.

---

## NCRs (Non-Conformance Reports)

**What it's for:** Documenting and tracking quality issues.

**NCR→ECO auto-link:** When an NCR identifies a design issue, create an ECO directly from the NCR. The ECO's `ncr_id` field maintains the link.

**Batch operations:** Select multiple NCRs to bulk close, resolve, or delete.

---

## Device Registry

**What it's for:** Tracking deployed devices in the field.

**Common workflows:**

1. **Register a device:** After production and testing, register the serial number.
2. **Import devices:** Upload a CSV file to bulk-register devices.
3. **Export devices:** Download all devices as a CSV.
4. **View history:** See full lifecycle — factory tests, firmware updates, campaigns.

**Batch operations:** Select multiple devices to bulk decommission or delete.

---

## Firmware Campaigns

**What it's for:** Managing OTA firmware rollouts to deployed devices.

### Live Streaming

Use the SSE stream endpoint to monitor campaign progress in real-time. The UI shows a live progress bar with device counts.

### Mark Individual Devices

Mark each device as `updated` or `failed` as the firmware rollout proceeds.

---

## RMAs (Return Merchandise Authorization)

**What it's for:** Processing customer returns.

**Batch operations:** Select multiple RMAs to bulk close or delete.

---

## Quotes

**What it's for:** Creating and tracking customer quotes with itemized pricing.

### Cost Rollup

The cost view calculates line totals (qty × unit price) and a grand total.

### PDF Quote

Click "Print Quote" to generate a professional quote document with:
- Quote number, date, validity period
- Customer information
- Line items with IPN, description, quantity, unit price, and line totals
- Subtotal
- Terms (Net 30) and contact info

---

## File Attachments

Upload files to any record in any module. Supported on ECOs, NCRs, work orders, documents, and more.

- **Upload:** Click the paperclip icon on any record detail view
- **Supported types:** Any file type (PDF, images, spreadsheets, etc.) up to 32MB
- **Storage:** Files are stored in the `uploads/` directory alongside the database
- **Access:** Files are served at `/files/{filename}` (no auth required for direct file URLs)
- **Delete:** Click the trash icon next to any attachment

---

## Audit Log

Every create, update, delete, and bulk operation is logged with:
- **Who** (username)
- **What** (action and module)
- **Which record** (record ID)
- **Summary** (human-readable description)
- **When** (timestamp)

Access the audit log from the sidebar. Filter by module, user, and date range.

---

## Batch Operations

Most list views support multi-select with checkboxes:

| Module | Bulk Actions |
|--------|-------------|
| ECOs | Approve, Implement, Reject, Delete |
| Work Orders | Complete, Cancel, Delete |
| NCRs | Close, Resolve, Delete |
| Devices | Decommission, Delete |
| RMAs | Close, Delete |
| Inventory | Delete |

Select items with checkboxes, then click the bulk action button.

---

## Calendar View

The calendar shows upcoming dates across modules:

| Color | Source | Date Shown |
|-------|--------|------------|
| Blue | Work Orders | Due date (completed_at or created + 30 days) |
| Green | Purchase Orders | Expected delivery date |
| Orange | Quotes | Expiration date (valid_until) |

Navigate between months using the arrow buttons. Click any event to jump to that record.

---

## Dark Mode

Toggle dark mode from the theme switcher in the top bar. Your preference is saved in the browser (localStorage) and persists across sessions.

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `/` or `Ctrl+K` | Focus global search |
| `Escape` | Close modal / clear search |
| `n` | New record (when in a module list view) |
| `?` | Show keyboard shortcuts help |

---

## Reports

Access from the **Reports** section in the sidebar. Five built-in reports:

### Inventory Valuation
Table of all inventory items showing quantity × unit price (from the latest PO line), subtotal per item, and grand total. Grouped by IPN category prefix (e.g., RES, CAP, PCA).

### Open ECOs by Priority
All open ECOs (draft/review status) sorted by priority from critical to low. Shows age in days since creation.

### WO Throughput
Work orders completed in a selectable time window (30/60/90 days). Shows count by status and average cycle time (started → completed).

### Low Stock Report
All inventory items where qty_on_hand is below reorder_point. Includes suggested reorder quantity and a link to create a purchase order.

### NCR Summary
Open NCRs broken down by severity and defect type. Shows total open count and average time to resolve for closed NCRs.

### CSV Export
Every report has an **Export CSV** button that downloads the report data as a CSV file.

---

## Quote Margin Analysis

In the Quote detail modal, click the **📊 Margin Analysis** tab to see:

- **BOM cost per line item** — sourced from the latest PO unit price for each IPN
- **Margin per unit** — quoted price minus BOM cost
- **Margin %** — color-coded: green (>50%), yellow (20–50%), red (<20%)
- **Summary totals** — total quoted, total BOM cost, total margin $, total margin %
- If no PO history exists for an IPN, the cost shows "—" with "No cost data"

---

## Supplier Price Catalog

Track price history for every part across vendors. Prices are recorded automatically when PO lines are received, and can also be added manually.

### Pricing Tab in Part Detail

When viewing any part, click the **💲 Pricing** tab to see:

- **Price history table** — vendor, unit price, min qty, lead time, date
- **Sparkline chart** — inline SVG showing price trend over time (hover points for details)
- **Best price highlight** — the lowest current unit price is marked with a green "Best" badge
- **Add Price button** — manually add a price entry with vendor dropdown, price, min qty, and lead time

### Automatic Price Recording

When receiving items on a Purchase Order (POST /api/v1/pos/:id/receive), the unit price from each PO line is automatically recorded in the price history, linked to the PO and vendor.

---

## Email Notifications

Configure SMTP to receive email notifications for system alerts (low stock, overdue work orders, aging NCRs, new RMAs).

### Email Settings (Admin → 📧 Email Settings)

- **SMTP host, port, username, password** — configure your mail server
- **From address and name** — the sender identity for outgoing emails
- **Enable/disable toggle** — turn email notifications on or off
- **Send Test Email** — verify configuration by sending a test message

### Email Log

The Email Settings page includes an **Email Log** table showing recent sent and failed emails with recipient, subject, status, error details, and timestamp.

### How It Works

When email is enabled, the background notification goroutine (runs every 5 minutes) will also send an email for each new notification. Each notification is only emailed once (tracked by an `emailed` flag). Emails are sent to the configured "from address" as the default recipient.

### Event-Driven Email Triggers

In addition to periodic notification emails, ZRP sends targeted emails on specific events:

- **ECO Approved** — When an ECO is approved, an email is sent to the ECO creator (falls back to admin address if no user email is set)
- **Low Stock Alert** — When an inventory transaction causes an item to drop below its reorder point, an email is sent to the admin
- **Overdue Work Order** — When a work order is updated and its due date has passed (and status is not closed/completed), an email is sent to the admin

All sent/failed emails are recorded in the email log for auditing.

---

## Custom Dashboard Widgets

The dashboard KPI cards and charts are now configurable. You can reorder, show, or hide any widget.

### Customizing the Dashboard

1. Click the **⚙️ Customize** button in the top-right of the dashboard
2. A modal shows all available widgets with toggle switches
3. **Drag to reorder** — grab the ⠿ handle and drag widgets up/down
4. **Toggle visibility** — use the switch to show/hide individual widgets
5. Click **Save** to apply changes

### Available Widgets

| Widget | Type |
|--------|------|
| Open ECOs | KPI card |
| Low Stock Items | KPI card |
| Open POs | KPI card |
| Active Work Orders | KPI card |
| Open NCRs | KPI card |
| Open RMAs | KPI card |
| Total Parts | KPI card |
| Total Devices | KPI card |
| ECO Status Chart | Chart |
| WO Status Chart | Chart |
| Inventory Chart | Chart |

Settings are saved server-side and persist across sessions.

---

## gitplm-ui Integration

ZRP links to the gitplm-ui for deeper part data:

- **Part detail modal** — "Open in gitplm-ui →" link in the header
- **Parts table** — ↗ icon next to each IPN opens that part in gitplm-ui
- **Configuration** — the gitplm-ui URL is configurable via `--gitplm-ui` flag (default: `http://localhost:8888`)

## Supplier Prices

**Route:** `/supplier-prices` (sidebar: Supply Chain → Supplier Prices)

Track and compare supplier price quotes per IPN across vendors.

### Features
- **Price catalog table** — sortable by IPN, vendor, price, date. "Best Price" per IPN highlighted in green.
- **Add Price Quote** — modal form with IPN autocomplete from parts DB, vendor name, unit price, currency, quantity break, lead time, quote date, and notes.
- **Price History** — click any IPN to see all quotes across vendors with an SVG line chart showing price trends over time (one line per vendor, color-coded legend).
- **Parts Integration** — Parts detail modal includes a "📊 Price Quotes" tab showing the supplier price chart and table for that IPN.
