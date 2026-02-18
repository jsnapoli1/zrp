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
	var err error
	db, err = sql.Open("sqlite", path+"?_journal_mode=WAL")
	if err != nil {
		return err
	}
	return runMigrations()
}

func runMigrations() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS ecos (
			id TEXT PRIMARY KEY, title TEXT NOT NULL, description TEXT,
			status TEXT DEFAULT 'draft', priority TEXT DEFAULT 'normal',
			affected_ipns TEXT, created_by TEXT DEFAULT 'engineer',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_at DATETIME, approved_by TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY, title TEXT NOT NULL, category TEXT, ipn TEXT,
			revision TEXT DEFAULT 'A', status TEXT DEFAULT 'draft',
			content TEXT, file_path TEXT, created_by TEXT DEFAULT 'engineer',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS vendors (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, website TEXT,
			contact_name TEXT, contact_email TEXT, contact_phone TEXT,
			notes TEXT, status TEXT DEFAULT 'active', lead_time_days INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS inventory (
			ipn TEXT PRIMARY KEY, qty_on_hand REAL DEFAULT 0,
			qty_reserved REAL DEFAULT 0, location TEXT,
			reorder_point REAL DEFAULT 0, reorder_qty REAL DEFAULT 0,
			description TEXT DEFAULT '', mpn TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT, ipn TEXT NOT NULL,
			type TEXT NOT NULL, qty REAL NOT NULL, reference TEXT, notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS purchase_orders (
			id TEXT PRIMARY KEY, vendor_id TEXT, status TEXT DEFAULT 'draft',
			notes TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date DATE, received_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT, po_id TEXT NOT NULL,
			ipn TEXT NOT NULL, mpn TEXT, manufacturer TEXT,
			qty_ordered REAL NOT NULL, qty_received REAL DEFAULT 0,
			unit_price REAL, notes TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS work_orders (
			id TEXT PRIMARY KEY, assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL DEFAULT 1, status TEXT DEFAULT 'open',
			priority TEXT DEFAULT 'normal', notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME, completed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS wo_serials (
			id INTEGER PRIMARY KEY AUTOINCREMENT, wo_id TEXT NOT NULL,
			serial_number TEXT NOT NULL, status TEXT DEFAULT 'building',
			notes TEXT, UNIQUE(serial_number)
		)`,
		`CREATE TABLE IF NOT EXISTS test_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT, serial_number TEXT NOT NULL,
			ipn TEXT NOT NULL, firmware_version TEXT, test_type TEXT,
			result TEXT NOT NULL, measurements TEXT, notes TEXT,
			tested_by TEXT DEFAULT 'operator',
			tested_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ncrs (
			id TEXT PRIMARY KEY, title TEXT NOT NULL, description TEXT,
			ipn TEXT, serial_number TEXT, defect_type TEXT,
			severity TEXT DEFAULT 'minor', status TEXT DEFAULT 'open',
			root_cause TEXT, corrective_action TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS devices (
			serial_number TEXT PRIMARY KEY, ipn TEXT NOT NULL,
			firmware_version TEXT, customer TEXT, location TEXT,
			status TEXT DEFAULT 'active', install_date DATE,
			last_seen DATETIME, notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS firmware_campaigns (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, version TEXT NOT NULL,
			category TEXT DEFAULT 'public', status TEXT DEFAULT 'draft',
			target_filter TEXT, notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME, completed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS campaign_devices (
			campaign_id TEXT NOT NULL, serial_number TEXT NOT NULL,
			status TEXT DEFAULT 'pending', updated_at DATETIME,
			PRIMARY KEY(campaign_id, serial_number)
		)`,
		`CREATE TABLE IF NOT EXISTS rmas (
			id TEXT PRIMARY KEY, serial_number TEXT NOT NULL,
			customer TEXT, reason TEXT, status TEXT DEFAULT 'open',
			defect_description TEXT, resolution TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			received_at DATETIME, resolved_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS quotes (
			id TEXT PRIMARY KEY, customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft', notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			valid_until DATE, accepted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS quote_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT, quote_id TEXT NOT NULL,
			ipn TEXT NOT NULL, description TEXT, qty INTEGER NOT NULL,
			unit_price REAL, notes TEXT
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
			expires_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, t := range tables {
		if _, err := db.Exec(t); err != nil {
			return fmt.Errorf("migration error: %w\nSQL: %s", err, t)
		}
	}
	// Add columns to existing tables if missing
	alterStmts := []string{
		"ALTER TABLE inventory ADD COLUMN description TEXT DEFAULT ''",
		"ALTER TABLE inventory ADD COLUMN mpn TEXT DEFAULT ''",
	}
	for _, s := range alterStmts {
		db.Exec(s) // ignore errors (column already exists)
	}
	return nil
}

func seedDB() {
	// Always ensure admin user exists
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&userCount)
	if userCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("zonit123"), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash admin password: %v", err)
		} else {
			db.Exec("INSERT INTO users (username, password_hash, display_name, role) VALUES (?, ?, ?, ?)",
				"admin", string(hash), "Administrator", "admin")
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
