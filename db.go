package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB(path string) error {
	// Close previous connection if any (prevents goroutine leaks in tests)
	if db != nil {
		db.Close()
	}
	var err error
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	db, err = sql.Open("sqlite", path+sep+"_journal_mode=WAL&_busy_timeout=10000&_foreign_keys=1")
	if err != nil {
		return err
	}
	
	// Configure connection pool for better concurrency
	// SQLite can handle 1 writer + multiple readers with WAL mode
	db.SetMaxOpenConns(10)  // Allow up to 10 concurrent connections
	db.SetMaxIdleConns(5)   // Keep 5 connections alive
	db.SetConnMaxLifetime(0) // Connections don't expire
	
	// Explicitly set WAL mode (some drivers don't parse connection string params correctly)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("enable WAL mode: %w", err)
	}
	
	// Set busy timeout explicitly (30 seconds for high concurrency)
	if _, err := db.Exec("PRAGMA busy_timeout=30000"); err != nil {
		return fmt.Errorf("set busy_timeout: %w", err)
	}
	
	// Ensure foreign keys are enforced for every connection
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	
	return runMigrations()
}

func runMigrations() error {
	shipmentTables := []string{
		`CREATE TABLE IF NOT EXISTS shipments (
			id TEXT PRIMARY KEY, type TEXT NOT NULL DEFAULT 'outbound' CHECK(type IN ('inbound','outbound','transfer')),
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','packed','shipped','delivered','cancelled')),
			tracking_number TEXT DEFAULT '',
			carrier TEXT DEFAULT '', ship_date DATETIME, delivery_date DATETIME,
			from_address TEXT DEFAULT '', to_address TEXT DEFAULT '',
			notes TEXT DEFAULT '', created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS shipment_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT, shipment_id TEXT NOT NULL,
			ipn TEXT DEFAULT '', serial_number TEXT DEFAULT '', qty INTEGER DEFAULT 1 CHECK(qty > 0),
			work_order_id TEXT DEFAULT '', rma_id TEXT DEFAULT '',
			FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS pack_lists (
			id INTEGER PRIMARY KEY AUTOINCREMENT, shipment_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
		)`,
	}
	for _, t := range shipmentTables {
		if _, err := db.Exec(t); err != nil {
			return fmt.Errorf("shipment migration: %w", err)
		}
	}

	fieldReportTable := `CREATE TABLE IF NOT EXISTS field_reports (
		id TEXT PRIMARY KEY, title TEXT NOT NULL,
		report_type TEXT DEFAULT 'failure' CHECK(report_type IN ('failure','performance','safety','visit','other')),
		status TEXT DEFAULT 'open' CHECK(status IN ('open','investigating','resolved','closed')),
		priority TEXT DEFAULT 'medium' CHECK(priority IN ('low','medium','high','critical')),
		customer_name TEXT DEFAULT '', site_location TEXT DEFAULT '',
		device_ipn TEXT DEFAULT '', device_serial TEXT DEFAULT '',
		reported_by TEXT DEFAULT '', reported_at DATETIME,
		description TEXT DEFAULT '', root_cause TEXT DEFAULT '',
		resolution TEXT DEFAULT '', resolved_at DATETIME,
		ncr_id TEXT DEFAULT '', eco_id TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.Exec(fieldReportTable); err != nil {
		return fmt.Errorf("field_reports migration: %w", err)
	}

	tables := []string{
		`CREATE TABLE IF NOT EXISTS ecos (
			id TEXT PRIMARY KEY, title TEXT NOT NULL, description TEXT,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','review','approved','implemented','rejected','cancelled')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			affected_ipns TEXT, created_by TEXT DEFAULT 'engineer',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_at DATETIME, approved_by TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY, title TEXT NOT NULL, category TEXT, ipn TEXT,
			revision TEXT DEFAULT 'A', status TEXT DEFAULT 'draft' CHECK(status IN ('draft','review','approved','released','obsolete')),
			content TEXT, file_path TEXT, created_by TEXT DEFAULT 'engineer',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS vendors (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, website TEXT,
			contact_name TEXT, contact_email TEXT, contact_phone TEXT,
			address TEXT DEFAULT '', payment_terms TEXT DEFAULT '',
			notes TEXT, status TEXT DEFAULT 'active' CHECK(status IN ('active','preferred','inactive','blocked')),
			lead_time_days INTEGER DEFAULT 0 CHECK(lead_time_days >= 0),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS inventory (
			ipn TEXT PRIMARY KEY, qty_on_hand REAL DEFAULT 0 CHECK(qty_on_hand >= 0),
			qty_reserved REAL DEFAULT 0 CHECK(qty_reserved >= 0), location TEXT,
			reorder_point REAL DEFAULT 0 CHECK(reorder_point >= 0),
			reorder_qty REAL DEFAULT 0 CHECK(reorder_qty >= 0),
			description TEXT DEFAULT '', mpn TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT, ipn TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('receive','issue','adjust','transfer','return','scrap')),
			qty REAL NOT NULL, reference TEXT, notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS purchase_orders (
			id TEXT PRIMARY KEY, vendor_id TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','sent','confirmed','partial','received','cancelled')),
			notes TEXT, created_by TEXT DEFAULT '',
			total REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date DATE, received_at DATETIME,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE RESTRICT
		)`,
		`CREATE TABLE IF NOT EXISTS po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT, po_id TEXT NOT NULL,
			ipn TEXT NOT NULL, mpn TEXT, manufacturer TEXT,
			qty_ordered REAL NOT NULL CHECK(qty_ordered > 0),
			qty_received REAL DEFAULT 0 CHECK(qty_received >= 0),
			unit_price REAL CHECK(unit_price >= 0), notes TEXT,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS work_orders (
			id TEXT PRIMARY KEY, assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL DEFAULT 1 CHECK(qty > 0),
			status TEXT DEFAULT 'open' CHECK(status IN ('open','in_progress','complete','cancelled','on_hold')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME, completed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS wo_serials (
			id INTEGER PRIMARY KEY AUTOINCREMENT, wo_id TEXT NOT NULL,
			serial_number TEXT NOT NULL,
			status TEXT DEFAULT 'building' CHECK(status IN ('building','testing','complete','failed','scrapped')),
			notes TEXT, UNIQUE(serial_number),
			FOREIGN KEY (wo_id) REFERENCES work_orders(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS test_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT, serial_number TEXT NOT NULL,
			ipn TEXT NOT NULL, firmware_version TEXT,
			test_type TEXT CHECK(test_type IN ('factory','incoming','final','field','calibration')),
			result TEXT NOT NULL CHECK(result IN ('pass','fail','conditional')),
			measurements TEXT, notes TEXT,
			tested_by TEXT DEFAULT 'operator',
			tested_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ncrs (
			id TEXT PRIMARY KEY, title TEXT NOT NULL, description TEXT,
			ipn TEXT, serial_number TEXT, defect_type TEXT,
			severity TEXT DEFAULT 'minor' CHECK(severity IN ('minor','major','critical')),
			status TEXT DEFAULT 'open' CHECK(status IN ('open','investigating','resolved','closed')),
			root_cause TEXT, corrective_action TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS devices (
			serial_number TEXT PRIMARY KEY, ipn TEXT NOT NULL,
			firmware_version TEXT, customer TEXT, location TEXT,
			status TEXT DEFAULT 'active' CHECK(status IN ('active','inactive','rma','decommissioned','maintenance')),
			install_date DATE,
			last_seen DATETIME, notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS firmware_campaigns (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, version TEXT NOT NULL,
			category TEXT DEFAULT 'public' CHECK(category IN ('public','beta','internal')),
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','active','paused','completed','cancelled')),
			target_filter TEXT, notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME, completed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS campaign_devices (
			campaign_id TEXT NOT NULL, serial_number TEXT NOT NULL,
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending','in_progress','success','failed','skipped')),
			updated_at DATETIME,
			PRIMARY KEY(campaign_id, serial_number),
			FOREIGN KEY (campaign_id) REFERENCES firmware_campaigns(id) ON DELETE CASCADE,
			FOREIGN KEY (serial_number) REFERENCES devices(serial_number) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS rmas (
			id TEXT PRIMARY KEY, serial_number TEXT NOT NULL,
			customer TEXT, reason TEXT,
			status TEXT DEFAULT 'open' CHECK(status IN ('open','received','diagnosing','repairing','resolved','closed','scrapped')),
			defect_description TEXT, resolution TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			received_at DATETIME, resolved_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS quotes (
			id TEXT PRIMARY KEY, customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','sent','accepted','rejected','expired','cancelled')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			valid_until DATE, accepted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS quote_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT, quote_id TEXT NOT NULL,
			ipn TEXT NOT NULL, description TEXT, qty INTEGER NOT NULL CHECK(qty > 0),
			unit_price REAL CHECK(unit_price >= 0), notes TEXT,
			FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS change_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_name TEXT NOT NULL,
			record_id TEXT NOT NULL,
			operation TEXT NOT NULL,
			old_data TEXT,
			new_data TEXT,
			user_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			undone INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS undo_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			action TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			previous_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			role TEXT DEFAULT 'user',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS csrf_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			before_value TEXT,
			after_value TEXT,
			ip_address TEXT,
			user_agent TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			created_by TEXT DEFAULT 'admin',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_used DATETIME,
			expires_at DATETIME,
			enabled INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			original_name TEXT NOT NULL,
			size_bytes INTEGER,
			mime_type TEXT,
			uploaded_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			severity TEXT DEFAULT 'info',
			title TEXT NOT NULL,
			message TEXT,
			record_id TEXT,
			module TEXT,
			user_id TEXT DEFAULT '',
			emailed INTEGER DEFAULT 0,
			read_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS price_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			vendor_id TEXT,
			vendor_name TEXT,
			unit_price REAL NOT NULL,
			currency TEXT DEFAULT 'USD',
			min_qty INTEGER DEFAULT 1,
			lead_time_days INTEGER,
			po_id TEXT,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			notes TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS email_config (
			id INTEGER PRIMARY KEY DEFAULT 1,
			smtp_host TEXT,
			smtp_port INTEGER DEFAULT 587,
			smtp_user TEXT,
			smtp_password TEXT,
			from_address TEXT,
			from_name TEXT DEFAULT 'ZRP',
			enabled INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS email_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			to_address TEXT NOT NULL,
			recipient TEXT DEFAULT '',
			subject TEXT NOT NULL,
			body TEXT,
			event_type TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'sent',
			error TEXT,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS email_subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			UNIQUE(user_id, event_type)
		)`,
		`CREATE TABLE IF NOT EXISTS eco_revisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			eco_id TEXT NOT NULL,
			revision TEXT NOT NULL DEFAULT 'A',
			status TEXT NOT NULL DEFAULT 'created',
			changes_summary TEXT,
			created_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_by TEXT,
			approved_at DATETIME,
			implemented_by TEXT,
			implemented_at DATETIME,
			effectivity_date TEXT,
			notes TEXT,
			FOREIGN KEY (eco_id) REFERENCES ecos(id)
		)`,
		`CREATE TABLE IF NOT EXISTS dashboard_widgets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER DEFAULT 0,
			widget_type TEXT NOT NULL,
			position INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS receiving_inspections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			po_line_id INTEGER NOT NULL,
			ipn TEXT NOT NULL,
			qty_received REAL NOT NULL DEFAULT 0 CHECK(qty_received >= 0),
			qty_passed REAL NOT NULL DEFAULT 0 CHECK(qty_passed >= 0),
			qty_failed REAL NOT NULL DEFAULT 0 CHECK(qty_failed >= 0),
			qty_on_hold REAL NOT NULL DEFAULT 0 CHECK(qty_on_hold >= 0),
			inspector TEXT,
			inspected_at DATETIME,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE RESTRICT,
			FOREIGN KEY (po_line_id) REFERENCES po_lines(id) ON DELETE RESTRICT
		)`,
		`CREATE TABLE IF NOT EXISTS rfqs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','sent','quoting','awarded','cancelled','closed')),
			created_by TEXT DEFAULT 'system',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			due_date TEXT,
			notes TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS rfq_vendors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			vendor_id TEXT NOT NULL,
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending','quoted','declined','awarded')),
			quoted_at DATETIME,
			notes TEXT,
			FOREIGN KEY (rfq_id) REFERENCES rfqs(id) ON DELETE CASCADE,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE RESTRICT
		)`,
		`CREATE TABLE IF NOT EXISTS rfq_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty REAL NOT NULL DEFAULT 0 CHECK(qty >= 0),
			unit TEXT DEFAULT 'ea',
			FOREIGN KEY (rfq_id) REFERENCES rfqs(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS rfq_quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			rfq_vendor_id INTEGER NOT NULL,
			rfq_line_id INTEGER NOT NULL,
			unit_price REAL DEFAULT 0 CHECK(unit_price >= 0),
			lead_time_days INTEGER DEFAULT 0 CHECK(lead_time_days >= 0),
			moq INTEGER DEFAULT 0 CHECK(moq >= 0),
			notes TEXT,
			FOREIGN KEY (rfq_id) REFERENCES rfqs(id) ON DELETE CASCADE,
			FOREIGN KEY (rfq_vendor_id) REFERENCES rfq_vendors(id) ON DELETE CASCADE,
			FOREIGN KEY (rfq_line_id) REFERENCES rfq_lines(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS product_pricing (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_ipn TEXT NOT NULL,
			pricing_tier TEXT NOT NULL DEFAULT 'standard' CHECK(pricing_tier IN ('standard','volume','distributor','oem')),
			min_qty INTEGER DEFAULT 0 CHECK(min_qty >= 0),
			max_qty INTEGER DEFAULT 0 CHECK(max_qty >= 0),
			unit_price REAL NOT NULL DEFAULT 0 CHECK(unit_price >= 0),
			currency TEXT DEFAULT 'USD',
			effective_date TEXT DEFAULT '',
			expiry_date TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS cost_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_ipn TEXT NOT NULL UNIQUE,
			bom_cost REAL DEFAULT 0,
			labor_cost REAL DEFAULT 0,
			overhead_cost REAL DEFAULT 0,
			total_cost REAL DEFAULT 0,
			margin_pct REAL DEFAULT 0,
			last_calculated DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS document_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT NOT NULL,
			revision TEXT NOT NULL,
			content TEXT DEFAULT '',
			file_path TEXT DEFAULT '',
			change_summary TEXT DEFAULT '',
			status TEXT DEFAULT 'draft',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			eco_id TEXT,
			FOREIGN KEY (document_id) REFERENCES documents(id)
		)`,
		`CREATE TABLE IF NOT EXISTS market_pricing (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			part_ipn TEXT NOT NULL,
			mpn TEXT NOT NULL,
			distributor TEXT NOT NULL,
			distributor_pn TEXT DEFAULT '',
			manufacturer TEXT DEFAULT '',
			description TEXT DEFAULT '',
			stock_qty INTEGER DEFAULT 0,
			lead_time_days INTEGER DEFAULT 0,
			currency TEXT DEFAULT 'USD',
			price_breaks TEXT DEFAULT '[]',
			product_url TEXT DEFAULT '',
			datasheet_url TEXT DEFAULT '',
			fetched_at TEXT NOT NULL,
			UNIQUE(part_ipn, distributor)
		)`,
	}
	tables = append(tables, `CREATE TABLE IF NOT EXISTS capas (
		id TEXT PRIMARY KEY, title TEXT NOT NULL,
		type TEXT DEFAULT 'corrective' CHECK(type IN ('corrective','preventive')),
		linked_ncr_id TEXT DEFAULT '', linked_rma_id TEXT DEFAULT '',
		root_cause TEXT DEFAULT '', action_plan TEXT DEFAULT '',
		owner TEXT DEFAULT '', due_date TEXT DEFAULT '',
		status TEXT DEFAULT 'open' CHECK(status IN ('open','in_progress','pending_review','closed','cancelled')),
		effectiveness_check TEXT DEFAULT '',
		approved_by_qe TEXT DEFAULT '', approved_by_qe_at DATETIME,
		approved_by_mgr TEXT DEFAULT '', approved_by_mgr_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	tables = append(tables, `CREATE TABLE IF NOT EXISTS parts (
		ipn TEXT PRIMARY KEY,
		category TEXT DEFAULT '',
		description TEXT DEFAULT '',
		mpn TEXT DEFAULT '',
		manufacturer TEXT DEFAULT '',
		lifecycle TEXT DEFAULT 'active',
		status TEXT DEFAULT 'active',
		notes TEXT DEFAULT '',
		fields TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	tables = append(tables, `CREATE TABLE IF NOT EXISTS part_changes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		part_ipn TEXT NOT NULL,
		eco_id TEXT DEFAULT '',
		field_name TEXT NOT NULL,
		old_value TEXT DEFAULT '',
		new_value TEXT DEFAULT '',
		status TEXT DEFAULT 'draft',
		created_by TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	tables = append(tables, `CREATE TABLE IF NOT EXISTS sales_orders (
		id TEXT PRIMARY KEY,
		quote_id TEXT DEFAULT '',
		customer TEXT NOT NULL,
		status TEXT DEFAULT 'draft' CHECK(status IN ('draft','confirmed','allocated','picked','shipped','invoiced','closed')),
		notes TEXT DEFAULT '',
		created_by TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	tables = append(tables, `CREATE TABLE IF NOT EXISTS sales_order_lines (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sales_order_id TEXT NOT NULL,
		ipn TEXT NOT NULL,
		description TEXT DEFAULT '',
		qty INTEGER NOT NULL CHECK(qty > 0),
		qty_allocated INTEGER DEFAULT 0 CHECK(qty_allocated >= 0),
		qty_picked INTEGER DEFAULT 0 CHECK(qty_picked >= 0),
		qty_shipped INTEGER DEFAULT 0 CHECK(qty_shipped >= 0),
		unit_price REAL DEFAULT 0 CHECK(unit_price >= 0),
		notes TEXT DEFAULT '',
		FOREIGN KEY (sales_order_id) REFERENCES sales_orders(id) ON DELETE CASCADE
	)`)

	tables = append(tables, `CREATE TABLE IF NOT EXISTS invoices (
		id TEXT PRIMARY KEY,
		invoice_number TEXT NOT NULL UNIQUE,
		sales_order_id TEXT NOT NULL,
		customer TEXT NOT NULL,
		issue_date DATE NOT NULL,
		due_date DATE NOT NULL,
		status TEXT DEFAULT 'draft' CHECK(status IN ('draft','sent','paid','overdue','cancelled')),
		total REAL DEFAULT 0,
		tax REAL DEFAULT 0,
		notes TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		paid_at DATETIME,
		FOREIGN KEY (sales_order_id) REFERENCES sales_orders(id) ON DELETE RESTRICT
	)`)

	tables = append(tables, `CREATE TABLE IF NOT EXISTS invoice_lines (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		invoice_id TEXT NOT NULL,
		ipn TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL,
		quantity INTEGER NOT NULL CHECK(quantity > 0),
		unit_price REAL NOT NULL CHECK(unit_price >= 0),
		total REAL NOT NULL CHECK(total >= 0),
		FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
	)`)

	for _, t := range tables {
		if _, err := db.Exec(t); err != nil {
			return fmt.Errorf("migration error: %w\nSQL: %s", err, t)
		}
	}
	// Add columns to existing tables if missing
	alterStmts := []string{
		"ALTER TABLE inventory ADD COLUMN description TEXT DEFAULT ''",
		"ALTER TABLE inventory ADD COLUMN mpn TEXT DEFAULT ''",
		"ALTER TABLE users ADD COLUMN active INTEGER DEFAULT 1",
		"ALTER TABLE ecos ADD COLUMN ncr_id TEXT DEFAULT ''",
		"ALTER TABLE notifications ADD COLUMN emailed INTEGER DEFAULT 0",
		"ALTER TABLE notifications ADD COLUMN user_id TEXT DEFAULT ''",
		"ALTER TABLE work_orders ADD COLUMN due_date TEXT DEFAULT ''",
		"ALTER TABLE work_orders ADD COLUMN qty_good INTEGER DEFAULT 0",
		"ALTER TABLE work_orders ADD COLUMN qty_scrap INTEGER DEFAULT 0",
		"ALTER TABLE users ADD COLUMN email TEXT DEFAULT ''",
		"ALTER TABLE email_log ADD COLUMN event_type TEXT DEFAULT ''",
		"ALTER TABLE email_log ADD COLUMN recipient TEXT DEFAULT ''",
		"ALTER TABLE purchase_orders ADD COLUMN created_by TEXT DEFAULT ''",
		"ALTER TABLE purchase_orders ADD COLUMN total REAL DEFAULT 0",
		"ALTER TABLE vendors ADD COLUMN address TEXT DEFAULT ''",
		"ALTER TABLE vendors ADD COLUMN payment_terms TEXT DEFAULT ''",
		"ALTER TABLE shipment_lines ADD COLUMN sales_order_id TEXT DEFAULT ''",
		// Invoice table migrations for enhanced invoicing
		"ALTER TABLE invoices ADD COLUMN invoice_number TEXT DEFAULT ''",
		"ALTER TABLE invoices ADD COLUMN issue_date DATE",
		"ALTER TABLE invoices ADD COLUMN tax REAL DEFAULT 0",
		"ALTER TABLE invoices ADD COLUMN notes TEXT DEFAULT ''",
		"ALTER TABLE invoices RENAME COLUMN total_amount TO total",
	}
	for _, s := range alterStmts {
		db.Exec(s) // ignore errors (column already exists)
	}

	// Enhanced audit logging migrations - MUST run BEFORE indexes
	auditMigrations := []string{
		`ALTER TABLE audit_log ADD COLUMN before_value TEXT`,
		`ALTER TABLE audit_log ADD COLUMN after_value TEXT`,
		`ALTER TABLE audit_log ADD COLUMN ip_address TEXT`,
		`ALTER TABLE audit_log ADD COLUMN user_agent TEXT`,
	}
	
	for _, migration := range auditMigrations {
		if _, err := db.Exec(migration); err != nil {
			// Ignore "duplicate column" errors - column already exists
			if !strings.Contains(err.Error(), "duplicate column") {
				log.Printf("Audit migration warning: %v\nSQL: %s", err, migration)
			}
		}
	}

	// Create indexes on frequently queried columns
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_ecos_status ON ecos(status)",
		"CREATE INDEX IF NOT EXISTS idx_ecos_created_at ON ecos(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_documents_category ON documents(category)",
		"CREATE INDEX IF NOT EXISTS idx_documents_ipn ON documents(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status)",
		"CREATE INDEX IF NOT EXISTS idx_vendors_status ON vendors(status)",
		"CREATE INDEX IF NOT EXISTS idx_inventory_location ON inventory(location)",
		"CREATE INDEX IF NOT EXISTS idx_inventory_transactions_ipn ON inventory_transactions(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_inventory_transactions_created_at ON inventory_transactions(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_purchase_orders_vendor_id ON purchase_orders(vendor_id)",
		"CREATE INDEX IF NOT EXISTS idx_purchase_orders_status ON purchase_orders(status)",
		"CREATE INDEX IF NOT EXISTS idx_po_lines_po_id ON po_lines(po_id)",
		"CREATE INDEX IF NOT EXISTS idx_po_lines_ipn ON po_lines(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_work_orders_status ON work_orders(status)",
		"CREATE INDEX IF NOT EXISTS idx_work_orders_assembly_ipn ON work_orders(assembly_ipn)",
		"CREATE INDEX IF NOT EXISTS idx_wo_serials_wo_id ON wo_serials(wo_id)",
		"CREATE INDEX IF NOT EXISTS idx_test_records_serial_number ON test_records(serial_number)",
		"CREATE INDEX IF NOT EXISTS idx_test_records_ipn ON test_records(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_test_records_result ON test_records(result)",
		"CREATE INDEX IF NOT EXISTS idx_ncrs_status ON ncrs(status)",
		"CREATE INDEX IF NOT EXISTS idx_ncrs_ipn ON ncrs(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_devices_ipn ON devices(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status)",
		"CREATE INDEX IF NOT EXISTS idx_devices_customer ON devices(customer)",
		"CREATE INDEX IF NOT EXISTS idx_campaign_devices_campaign_id ON campaign_devices(campaign_id)",
		"CREATE INDEX IF NOT EXISTS idx_rmas_status ON rmas(status)",
		"CREATE INDEX IF NOT EXISTS idx_rmas_serial_number ON rmas(serial_number)",
		"CREATE INDEX IF NOT EXISTS idx_quotes_status ON quotes(status)",
		"CREATE INDEX IF NOT EXISTS idx_quote_lines_quote_id ON quote_lines(quote_id)",
		"CREATE INDEX IF NOT EXISTS idx_change_history_table_record ON change_history(table_name, record_id)",
		"CREATE INDEX IF NOT EXISTS idx_change_history_created_at ON change_history(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_undo_log_user_id ON undo_log(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_undo_log_expires_at ON undo_log(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_module ON audit_log(module)",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_record_id ON audit_log(record_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action)",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_ip_address ON audit_log(ip_address)",
		"CREATE INDEX IF NOT EXISTS idx_attachments_module_record ON attachments(module, record_id)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_read_at ON notifications(read_at)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_price_history_ipn ON price_history(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_price_history_vendor_id ON price_history(vendor_id)",
		"CREATE INDEX IF NOT EXISTS idx_eco_revisions_eco_id ON eco_revisions(eco_id)",
		"CREATE INDEX IF NOT EXISTS idx_sales_orders_status ON sales_orders(status)",
		"CREATE INDEX IF NOT EXISTS idx_sales_orders_quote_id ON sales_orders(quote_id)",
		"CREATE INDEX IF NOT EXISTS idx_sales_orders_customer ON sales_orders(customer)",
		"CREATE INDEX IF NOT EXISTS idx_sales_order_lines_order_id ON sales_order_lines(sales_order_id)",
		"CREATE INDEX IF NOT EXISTS idx_invoices_sales_order_id ON invoices(sales_order_id)",
		"CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status)",
		"CREATE INDEX IF NOT EXISTS idx_invoices_customer ON invoices(customer)",
		"CREATE INDEX IF NOT EXISTS idx_invoices_due_date ON invoices(due_date)",
		"CREATE INDEX IF NOT EXISTS idx_invoices_invoice_number ON invoices(invoice_number)",
		"CREATE INDEX IF NOT EXISTS idx_invoice_lines_invoice_id ON invoice_lines(invoice_id)",
		"CREATE INDEX IF NOT EXISTS idx_shipment_lines_sales_order_id ON shipment_lines(sales_order_id)",
		"CREATE INDEX IF NOT EXISTS idx_receiving_inspections_po_id ON receiving_inspections(po_id)",
		"CREATE INDEX IF NOT EXISTS idx_rfq_vendors_rfq_id ON rfq_vendors(rfq_id)",
		"CREATE INDEX IF NOT EXISTS idx_rfq_lines_rfq_id ON rfq_lines(rfq_id)",
		"CREATE INDEX IF NOT EXISTS idx_rfq_quotes_rfq_id ON rfq_quotes(rfq_id)",
		"CREATE INDEX IF NOT EXISTS idx_product_pricing_product_ipn ON product_pricing(product_ipn)",
		"CREATE INDEX IF NOT EXISTS idx_document_versions_document_id ON document_versions(document_id)",
		"CREATE INDEX IF NOT EXISTS idx_market_pricing_part_ipn ON market_pricing(part_ipn)",
		"CREATE INDEX IF NOT EXISTS idx_capas_status ON capas(status)",
		"CREATE INDEX IF NOT EXISTS idx_part_changes_part_ipn ON part_changes(part_ipn)",
		"CREATE INDEX IF NOT EXISTS idx_part_changes_eco_id ON part_changes(eco_id)",
		"CREATE INDEX IF NOT EXISTS idx_shipment_lines_shipment_id ON shipment_lines(shipment_id)",
		"CREATE INDEX IF NOT EXISTS idx_field_reports_status ON field_reports(status)",
		"CREATE INDEX IF NOT EXISTS idx_email_log_sent_at ON email_log(sent_at)",
		
		// Performance optimization: Composite indexes for common query patterns
		"CREATE INDEX IF NOT EXISTS idx_inventory_ipn_qty_on_hand ON inventory(ipn, qty_on_hand)",
		"CREATE INDEX IF NOT EXISTS idx_inventory_ipn_qty_reserved ON inventory(ipn, qty_reserved)",
		"CREATE INDEX IF NOT EXISTS idx_po_lines_ipn_unit_price ON po_lines(ipn, unit_price)",
		"CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications(user_id, read_at)",
		"CREATE INDEX IF NOT EXISTS idx_test_records_ipn_result_tested ON test_records(ipn, result, tested_at)",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_user_created ON audit_log(user_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_change_history_user_created ON change_history(user_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_email_log_address_sent ON email_log(to_address, sent_at)",
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("index creation: %w\nSQL: %s", err, idx)
		}
	}

	// Quality workflow improvements - add missing columns
	qualityMigrations := []string{
		// Add created_by to NCR table (Gap 5.2)
		`ALTER TABLE ncrs ADD COLUMN created_by TEXT DEFAULT ''`,
		
		// Add ncr_id to ECO table to link ECOs back to NCRs
		`ALTER TABLE ecos ADD COLUMN ncr_id TEXT DEFAULT ''`,
		
		// Update CAPA status values to match workflow (Gap 5.5)
		// Note: SQLite doesn't support modifying CHECK constraints, so this is documented
		// Current: ('open','in_progress','pending_review','closed','cancelled')
		// Should be: ('draft','pending_approval','approved','in_progress','verification','closed')
		
		// Add indexes for better performance
		`CREATE INDEX IF NOT EXISTS idx_ncrs_created_by ON ncrs(created_by)`,
		`CREATE INDEX IF NOT EXISTS idx_ecos_ncr_id ON ecos(ncr_id)`,
		`CREATE INDEX IF NOT EXISTS idx_capas_linked_ncr_id ON capas(linked_ncr_id)`,
	}
	
	for _, migration := range qualityMigrations {
		// Use IF NOT EXISTS pattern for ALTER TABLE
		if _, err := db.Exec(migration); err != nil {
			// Ignore "duplicate column" errors - column already exists
			if !strings.Contains(err.Error(), "duplicate column") {
				log.Printf("Quality migration warning: %v\nSQL: %s", err, migration)
			}
		}
	}

	// Initialize advanced search tables
	if err := InitSearchTables(db); err != nil {
		log.Printf("Search tables migration warning: %v", err)
	}

	return nil
}

func seedDB() {
	// Always ensure admin user exists
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&userCount)
	if userCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("changeme"), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash admin password: %v", err)
		} else {
			db.Exec("INSERT INTO users (username, password_hash, display_name, role) VALUES (?, ?, ?, ?)",
				"admin", string(hash), "Administrator", "admin")
		}
	}

	// Seed engineer user
	var engCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'engineer'").Scan(&engCount)
	if engCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("changeme"), bcrypt.DefaultCost)
		if err == nil {
			db.Exec("INSERT INTO users (username, password_hash, display_name, role, active) VALUES (?, ?, ?, ?, 1)",
				"engineer", string(hash), "Engineer", "user")
		}
	}
	// Seed viewer user
	var viewCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'viewer'").Scan(&viewCount)
	if viewCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("changeme"), bcrypt.DefaultCost)
		if err == nil {
			db.Exec("INSERT INTO users (username, password_hash, display_name, role, active) VALUES (?, ?, ?, ?, 1)",
				"viewer", string(hash), "Viewer", "readonly")
		}
	}

	// Seed email config
	var emailCount int
	db.QueryRow("SELECT COUNT(*) FROM email_config").Scan(&emailCount)
	if emailCount == 0 {
		db.Exec("INSERT INTO email_config (id, enabled) VALUES (1, 0)")
	}

	// Seed dashboard widgets
	var widgetCount int
	db.QueryRow("SELECT COUNT(*) FROM dashboard_widgets").Scan(&widgetCount)
	if widgetCount == 0 {
		widgets := []string{
			"kpi_open_ecos", "kpi_low_stock", "kpi_open_pos", "kpi_active_wos",
			"kpi_open_ncrs", "kpi_open_rmas", "kpi_total_parts", "kpi_total_devices",
			"chart_eco_status", "chart_wo_status", "chart_inventory",
		}
		for i, w := range widgets {
			db.Exec("INSERT INTO dashboard_widgets (user_id, widget_type, position, enabled) VALUES (0, ?, ?, 1)", w, i)
		}
	}

	// Check if already seeded
	var count int
	db.QueryRow("SELECT COUNT(*) FROM ecos").Scan(&count)
	if count > 0 {
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	year := time.Now().Format("2006")

	// ECOs
	db.Exec(`INSERT INTO ecos (id,title,description,status,priority,affected_ipns,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		"ECO-"+year+"-001", "Update power supply capacitor", "Replace C12 with higher voltage rating", "draft", "high", `["CAP-001-0001"]`, now, now)
	db.Exec(`INSERT INTO ecos (id,title,description,status,priority,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		"ECO-"+year+"-002", "Add conformal coating to PCB", "Environmental protection improvement", "review", "normal", now, now)

	// Documents
	db.Exec(`INSERT INTO documents (id,title,category,revision,status,content,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		"DOC-"+year+"-001", "Assembly Procedure - Z1000", "procedure", "B", "approved", "# Assembly Procedure\n\nStep 1: Place PCB...", now, now)
	db.Exec(`INSERT INTO documents (id,title,category,status,content,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		"DOC-"+year+"-002", "Test Specification - Power Module", "spec", "draft", "# Test Spec\n\nVoltage range: 100-240VAC", now, now)

	// Vendors
	db.Exec(`INSERT INTO vendors (id,name,website,contact_name,contact_email,status,lead_time_days) VALUES (?,?,?,?,?,?,?)`,
		"V-001", "DigiKey", "https://digikey.com", "Sales Team", "sales@digikey.com", "preferred", 3)
	db.Exec(`INSERT INTO vendors (id,name,website,contact_name,contact_email,status,lead_time_days) VALUES (?,?,?,?,?,?,?)`,
		"V-002", "Mouser Electronics", "https://mouser.com", "Account Rep", "rep@mouser.com", "active", 5)
	db.Exec(`INSERT INTO vendors (id,name,website,status,lead_time_days) VALUES (?,?,?,?,?)`,
		"V-003", "JLCPCB", "https://jlcpcb.com", "active", 14)

	// Inventory
	db.Exec(`INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location,reorder_point,reorder_qty) VALUES (?,?,?,?,?,?)`,
		"CAP-001-0001", 500, 50, "Bin A-12", 100, 1000)
	db.Exec(`INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location,reorder_point,reorder_qty) VALUES (?,?,?,?,?,?)`,
		"RES-001-0001", 25, 0, "Bin B-03", 100, 500)
	db.Exec(`INSERT INTO inventory (ipn,qty_on_hand,location,reorder_point,reorder_qty) VALUES (?,?,?,?,?)`,
		"PCB-001-0001", 150, "Shelf C-1", 20, 50)

	// Inventory transactions
	db.Exec(`INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)`,
		"CAP-001-0001", "receive", 500, "PO-"+year+"-0001", "Initial stock", now)
	db.Exec(`INSERT INTO inventory_transactions (ipn,type,qty,reference,created_at) VALUES (?,?,?,?,?)`,
		"CAP-001-0001", "issue", -50, "WO-"+year+"-0001", now)

	// Purchase Orders
	db.Exec(`INSERT INTO purchase_orders (id,vendor_id,status,notes,expected_date) VALUES (?,?,?,?,?)`,
		"PO-"+year+"-0001", "V-001", "received", "Capacitor order", "2026-03-01")
	db.Exec(`INSERT INTO purchase_orders (id,vendor_id,status,notes,expected_date) VALUES (?,?,?,?,?)`,
		"PO-"+year+"-0002", "V-002", "sent", "Resistors restock", "2026-03-15")

	db.Exec(`INSERT INTO po_lines (po_id,ipn,mpn,manufacturer,qty_ordered,qty_received,unit_price) VALUES (?,?,?,?,?,?,?)`,
		"PO-"+year+"-0001", "CAP-001-0001", "GRM188R71C104KA01", "Murata", 1000, 1000, 0.02)
	db.Exec(`INSERT INTO po_lines (po_id,ipn,mpn,manufacturer,qty_ordered,unit_price) VALUES (?,?,?,?,?,?)`,
		"PO-"+year+"-0002", "RES-001-0001", "RC0402FR-0710KL", "Yageo", 500, 0.005)

	// Work Orders
	db.Exec(`INSERT INTO work_orders (id,assembly_ipn,qty,status,priority,notes) VALUES (?,?,?,?,?,?)`,
		"WO-"+year+"-0001", "PCB-001-0001", 10, "in_progress", "high", "Production batch 1")
	db.Exec(`INSERT INTO work_orders (id,assembly_ipn,qty,status,priority) VALUES (?,?,?,?,?)`,
		"WO-"+year+"-0002", "PCB-001-0001", 5, "open", "normal")

	// Test Records
	db.Exec(`INSERT INTO test_records (serial_number,ipn,firmware_version,test_type,result,measurements,tested_by) VALUES (?,?,?,?,?,?,?)`,
		"SN-001", "PCB-001-0001", "1.2.0", "factory", "pass", `{"voltage":12.1,"current":0.5}`, "operator")
	db.Exec(`INSERT INTO test_records (serial_number,ipn,firmware_version,test_type,result,notes,tested_by) VALUES (?,?,?,?,?,?,?)`,
		"SN-002", "PCB-001-0001", "1.2.0", "factory", "fail", "Voltage out of spec", "operator")

	// NCRs
	db.Exec(`INSERT INTO ncrs (id,title,description,ipn,serial_number,defect_type,severity,status) VALUES (?,?,?,?,?,?,?,?)`,
		"NCR-"+year+"-001", "Solder bridge on U3", "Solder bridge between pins 3-4 of U3", "PCB-001-0001", "SN-002", "workmanship", "major", "open")
	db.Exec(`INSERT INTO ncrs (id,title,description,defect_type,severity,status,root_cause,corrective_action,resolved_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		"NCR-"+year+"-002", "Wrong resistor value", "R15 populated with 1K instead of 10K", "component", "minor", "resolved", "BOM mismatch", "Updated BOM and retrained operator", now)

	// Devices
	db.Exec(`INSERT INTO devices (serial_number,ipn,firmware_version,customer,location,status,install_date) VALUES (?,?,?,?,?,?,?)`,
		"SN-001", "PCB-001-0001", "1.2.0", "Acme Corp", "Data Center A", "active", "2026-01-15")
	db.Exec(`INSERT INTO devices (serial_number,ipn,firmware_version,customer,location,status,install_date) VALUES (?,?,?,?,?,?,?)`,
		"SN-003", "PCB-001-0001", "1.1.0", "TechStart Inc", "Server Room B", "active", "2025-11-20")
	db.Exec(`INSERT INTO devices (serial_number,ipn,firmware_version,customer,status) VALUES (?,?,?,?,?)`,
		"SN-002", "PCB-001-0001", "1.2.0", "Acme Corp", "rma")

	// Firmware Campaigns
	db.Exec(`INSERT INTO firmware_campaigns (id,name,version,category,status,notes) VALUES (?,?,?,?,?,?)`,
		"FW-"+year+"-001", "v1.3.0 Security Patch", "1.3.0", "public", "draft", "Critical security update")

	// RMAs
	db.Exec(`INSERT INTO rmas (id,serial_number,customer,reason,status,defect_description) VALUES (?,?,?,?,?,?)`,
		"RMA-"+year+"-001", "SN-002", "Acme Corp", "Unit not powering on", "diagnosing", "Customer reports no LED activity")
	db.Exec(`INSERT INTO rmas (id,serial_number,customer,reason,status,defect_description,resolution,resolved_at) VALUES (?,?,?,?,?,?,?,?)`,
		"RMA-"+year+"-002", "SN-004", "BigCo", "Intermittent connectivity", "closed", "WiFi drops under load", "Replaced antenna module", now)

	// Quotes
	db.Exec(`INSERT INTO quotes (id,customer,status,notes,valid_until) VALUES (?,?,?,?,?)`,
		"Q-"+year+"-001", "Acme Corp", "sent", "50 units Z1000", "2026-04-01")
	db.Exec(`INSERT INTO quotes (id,customer,status,notes,valid_until) VALUES (?,?,?,?,?)`,
		"Q-"+year+"-002", "TechStart Inc", "draft", "Custom configuration", "2026-05-01")

	db.Exec(`INSERT INTO quote_lines (quote_id,ipn,description,qty,unit_price) VALUES (?,?,?,?,?)`,
		"Q-"+year+"-001", "PCB-001-0001", "Z1000 Power Module", 50, 149.99)
	db.Exec(`INSERT INTO quote_lines (quote_id,ipn,description,qty,unit_price) VALUES (?,?,?,?,?)`,
		"Q-"+year+"-002", "PCB-001-0001", "Z1000 Custom Config", 20, 179.99)
}

// ID generation helpers
func nextID(prefix string, table string, digits int) string {
	year := time.Now().Format("2006")
	pattern := prefix + "-" + year + "-%"
	var maxID sql.NullString
	db.QueryRow("SELECT id FROM "+table+" WHERE id LIKE ? ORDER BY id DESC LIMIT 1", pattern).Scan(&maxID)

	next := 1
	if maxID.Valid {
		parts := strings.Split(maxID.String, "-")
		if len(parts) >= 3 {
			if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
				next = n + 1
			}
		}
	}
	return fmt.Sprintf("%s-%s-%0*d", prefix, year, digits, next)
}

func ns(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func sp(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}
