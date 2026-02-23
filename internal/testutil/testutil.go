package testutil

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zrp/internal/models"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// SetupTestDB creates a standard in-memory SQLite database for testing
// with foreign keys enabled and common tables created.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	createTables(t, testDB)
	seedAdminUser(t, testDB)

	return testDB
}

func createTables(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []struct {
		name string
		ddl  string
	}{
		{"users", `CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1,
			last_login TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`},
		{"sessions", `CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`},
		{"capas", `CREATE TABLE IF NOT EXISTS capas (
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
		)`},
		{"sales_orders", `CREATE TABLE IF NOT EXISTS sales_orders (
			id TEXT PRIMARY KEY,
			quote_id TEXT DEFAULT '',
			customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','confirmed','allocated','picked','shipped','invoiced','closed')),
			notes TEXT DEFAULT '',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`},
		{"sales_order_lines", `CREATE TABLE IF NOT EXISTS sales_order_lines (
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
		)`},
		{"quotes", `CREATE TABLE IF NOT EXISTS quotes (
			id TEXT PRIMARY KEY,
			customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			notes TEXT DEFAULT '',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			valid_until TEXT DEFAULT '',
			accepted_at DATETIME
		)`},
		{"quote_lines", `CREATE TABLE IF NOT EXISTS quote_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			quote_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT DEFAULT '',
			qty INTEGER NOT NULL CHECK(qty > 0),
			unit_price REAL DEFAULT 0 CHECK(unit_price >= 0),
			notes TEXT DEFAULT '',
			FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE
		)`},
		{"invoices", `CREATE TABLE IF NOT EXISTS invoices (
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
		)`},
		{"invoice_lines", `CREATE TABLE IF NOT EXISTS invoice_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			invoice_id TEXT NOT NULL,
			ipn TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL,
			quantity INTEGER NOT NULL CHECK(quantity > 0),
			unit_price REAL NOT NULL CHECK(unit_price >= 0),
			total REAL NOT NULL CHECK(total >= 0),
			FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
		)`},
		{"inventory", `CREATE TABLE IF NOT EXISTS inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0 CHECK(qty_on_hand >= 0),
			qty_reserved REAL DEFAULT 0 CHECK(qty_reserved >= 0),
			location TEXT DEFAULT '',
			reorder_point REAL DEFAULT 0 CHECK(reorder_point >= 0),
			reorder_qty REAL DEFAULT 0 CHECK(reorder_qty >= 0),
			description TEXT DEFAULT '',
			mpn TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`},
		{"inventory_transactions", `CREATE TABLE IF NOT EXISTS inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT, ipn TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('receive','issue','adjust','transfer','return','scrap')),
			qty REAL NOT NULL, reference TEXT, notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`},
		{"ncrs", `CREATE TABLE IF NOT EXISTS ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			ipn TEXT DEFAULT '',
			serial_number TEXT DEFAULT '',
			defect_type TEXT DEFAULT '',
			severity TEXT DEFAULT 'minor' CHECK(severity IN ('minor','major','critical')),
			status TEXT DEFAULT 'open' CHECK(status IN ('open','investigating','resolved','closed')),
			priority TEXT DEFAULT 'medium',
			root_cause TEXT DEFAULT '',
			corrective_action TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		)`},
		{"ecos", `CREATE TABLE IF NOT EXISTS ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','review','approved','implemented','rejected','cancelled')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			affected_ipns TEXT DEFAULT '',
			created_by TEXT DEFAULT 'engineer',
			linked_ncr_id TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_at DATETIME,
			approved_by TEXT DEFAULT ''
		)`},
		{"work_orders", `CREATE TABLE IF NOT EXISTS work_orders (
			id TEXT PRIMARY KEY, assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL DEFAULT 1 CHECK(qty > 0),
			qty_good INTEGER DEFAULT 0,
			qty_scrap INTEGER DEFAULT 0,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','open','in_progress','completed','cancelled','on_hold')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			notes TEXT,
			due_date TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME, completed_at DATETIME
		)`},
		{"wo_serials", `CREATE TABLE IF NOT EXISTS wo_serials (
			id INTEGER PRIMARY KEY AUTOINCREMENT, wo_id TEXT NOT NULL,
			serial_number TEXT NOT NULL,
			status TEXT DEFAULT 'building' CHECK(status IN ('building','testing','complete','failed','scrapped')),
			notes TEXT, UNIQUE(serial_number),
			FOREIGN KEY (wo_id) REFERENCES work_orders(id) ON DELETE CASCADE
		)`},
		{"parts", `CREATE TABLE IF NOT EXISTS parts (
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
		)`},
		{"bom", `CREATE TABLE IF NOT EXISTS bom (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_ipn TEXT NOT NULL,
			child_ipn TEXT NOT NULL,
			quantity REAL NOT NULL DEFAULT 1,
			reference_designator TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			UNIQUE(parent_ipn, child_ipn),
			FOREIGN KEY (parent_ipn) REFERENCES parts(ipn),
			FOREIGN KEY (child_ipn) REFERENCES parts(ipn)
		)`},
		{"documents", `CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			category TEXT DEFAULT '',
			ipn TEXT DEFAULT '',
			content TEXT DEFAULT '',
			revision TEXT DEFAULT '',
			status TEXT DEFAULT 'draft',
			file_path TEXT DEFAULT '',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_at DATETIME,
			approved_by TEXT DEFAULT ''
		)`},
		{"document_versions", `CREATE TABLE IF NOT EXISTS document_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT NOT NULL,
			revision TEXT NOT NULL,
			content TEXT DEFAULT '',
			file_path TEXT DEFAULT '',
			change_summary TEXT DEFAULT '',
			status TEXT DEFAULT 'draft',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			comment TEXT DEFAULT '',
			eco_id TEXT DEFAULT '',
			FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
		)`},
		{"app_settings", `CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`},
		{"audit_log", `CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			record_id TEXT NOT NULL,
			user_id INTEGER,
			username TEXT DEFAULT '',
			summary TEXT DEFAULT '',
			changes TEXT DEFAULT '{}',
			ip_address TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`},
		{"vendors", `CREATE TABLE IF NOT EXISTS vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			address TEXT DEFAULT '',
			payment_terms TEXT DEFAULT '',
			notes TEXT,
			status TEXT DEFAULT 'active' CHECK(status IN ('active','preferred','inactive','blocked')),
			lead_time_days INTEGER DEFAULT 0 CHECK(lead_time_days >= 0),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`},
		{"shipments", `CREATE TABLE IF NOT EXISTS shipments (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL DEFAULT 'outbound' CHECK(type IN ('inbound','outbound','transfer')),
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','packed','shipped','delivered','cancelled')),
			tracking_number TEXT DEFAULT '',
			carrier TEXT DEFAULT '',
			ship_date DATETIME,
			delivery_date DATETIME,
			from_address TEXT DEFAULT '',
			to_address TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`},
		{"shipment_lines", `CREATE TABLE IF NOT EXISTS shipment_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			shipment_id TEXT NOT NULL,
			ipn TEXT DEFAULT '',
			serial_number TEXT DEFAULT '',
			qty INTEGER DEFAULT 1 CHECK(qty > 0),
			work_order_id TEXT DEFAULT '',
			rma_id TEXT DEFAULT '',
			sales_order_id TEXT DEFAULT '',
			FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
		)`},
		{"pack_lists", `CREATE TABLE IF NOT EXISTS pack_lists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			shipment_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
		)`},
		{"part_changes", `CREATE TABLE IF NOT EXISTS part_changes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			part_ipn TEXT NOT NULL,
			eco_id TEXT DEFAULT '',
			field_name TEXT NOT NULL,
			old_value TEXT DEFAULT '',
			new_value TEXT DEFAULT '',
			status TEXT DEFAULT 'draft',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`},
		{"product_pricing", `CREATE TABLE IF NOT EXISTS product_pricing (
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
		)`},
		{"password_reset_tokens", `CREATE TABLE IF NOT EXISTS password_reset_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			used INTEGER DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`},
		{"notifications", `CREATE TABLE IF NOT EXISTS notifications (
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
		)`},
		{"market_pricing", `CREATE TABLE IF NOT EXISTS market_pricing (
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
		)`},
		{"eco_revisions", `CREATE TABLE IF NOT EXISTS eco_revisions (
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
		)`},
		{"password_history", `CREATE TABLE IF NOT EXISTS password_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`},
	}

	for _, tbl := range tables {
		if _, err := db.Exec(tbl.ddl); err != nil {
			t.Fatalf("Failed to create %s table: %v", tbl.name, err)
		}
	}
}

func seedAdminUser(t *testing.T, db *sql.DB) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("changeme"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash admin password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO users (username, password_hash, display_name, role) VALUES (?, ?, ?, ?)`,
		"admin", string(hash), "Administrator", "admin")
	if err != nil {
		t.Fatalf("Failed to create default admin user: %v", err)
	}
}

// CreateTestUser creates a test user with the given credentials.
func CreateTestUser(t *testing.T, db *sql.DB, username, password, role string, active bool) int {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	activeInt := 0
	if active {
		activeInt = 1
	}

	result, err := db.Exec(
		"INSERT INTO users (username, password_hash, display_name, role, active) VALUES (?, ?, ?, ?, ?)",
		username, string(hash), username+" Display", role, activeInt,
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// CreateTestSessionSimple creates a session token for the given user with default 24h expiry.
func CreateTestSessionSimple(t *testing.T, db *sql.DB, userID int) string {
	t.Helper()
	token := "test-session-token-" + time.Now().Format("20060102150405.000000")
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return token
}

// LoginAdmin returns a session token for the default admin user.
func LoginAdmin(t *testing.T, db *sql.DB) string {
	t.Helper()
	var adminID int
	err := db.QueryRow("SELECT id FROM users WHERE username = 'admin'").Scan(&adminID)
	if err != nil {
		t.Fatalf("Failed to find admin user: %v", err)
	}
	return CreateTestSessionSimple(t, db, adminID)
}

// LoginUser creates a regular user and returns their session token.
func LoginUser(t *testing.T, db *sql.DB, username string) string {
	t.Helper()
	userID := CreateTestUser(t, db, username, "password", "user", true)
	return CreateTestSessionSimple(t, db, userID)
}

// AuthedRequest creates an authenticated HTTP request with a session cookie.
func AuthedRequest(method, path string, body []byte, sessionToken string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	if sessionToken != "" {
		req.AddCookie(&http.Cookie{Name: "zrp_session", Value: sessionToken})
	}

	return req
}

// AuthedJSONRequest creates an authenticated HTTP request with JSON content type.
func AuthedJSONRequest(method, path string, body interface{}, sessionToken string) *http.Request {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	req := AuthedRequest(method, path, bodyBytes, sessionToken)
	req.Header.Set("Content-Type", "application/json")

	return req
}

// DecodeAPIResponse decodes an APIResponse from a ResponseRecorder.
func DecodeAPIResponse(t *testing.T, w *httptest.ResponseRecorder) models.APIResponse {
	t.Helper()
	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode API response: %v", err)
	}
	return response
}

// AssertStatus checks that the HTTP status code matches expected.
func AssertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("Expected status %d, got %d. Body: %s", expected, w.Code, w.Body.String())
	}
}

// DecodeEnvelope decodes an API response envelope and extracts the data.
func DecodeEnvelope(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode API envelope: %v", err)
	}
	dataBytes, _ := json.Marshal(resp.Data)
	if err := json.Unmarshal(dataBytes, v); err != nil {
		t.Fatalf("Failed to decode data from envelope: %v", err)
	}
}
