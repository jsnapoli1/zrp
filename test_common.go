package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// setupTestDB creates a standard in-memory SQLite database for testing
// with foreign keys enabled and common tables created
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create core users table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1,
			last_login TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create sessions table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	// Create capas table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS capas (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create capas table: %v", err)
	}

	// Create sales_orders table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS sales_orders (
			id TEXT PRIMARY KEY,
			quote_id TEXT DEFAULT '',
			customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','confirmed','allocated','picked','shipped','invoiced','closed')),
			notes TEXT DEFAULT '',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sales_orders table: %v", err)
	}

	// Create sales_order_lines table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS sales_order_lines (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sales_order_lines table: %v", err)
	}

	// Create quotes table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS quotes (
			id TEXT PRIMARY KEY,
			customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			notes TEXT DEFAULT '',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create quotes table: %v", err)
	}

	// Create quote_lines table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS quote_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			quote_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT DEFAULT '',
			qty INTEGER NOT NULL CHECK(qty > 0),
			unit_price REAL DEFAULT 0 CHECK(unit_price >= 0),
			notes TEXT DEFAULT '',
			FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create quote_lines table: %v", err)
	}

	// Create invoices table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS invoices (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create invoices table: %v", err)
	}

	// Create invoice_lines table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS invoice_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			invoice_id TEXT NOT NULL,
			ipn TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL,
			quantity INTEGER NOT NULL CHECK(quantity > 0),
			unit_price REAL NOT NULL CHECK(unit_price >= 0),
			total REAL NOT NULL CHECK(total >= 0),
			FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create invoice_lines table: %v", err)
	}

	// Create inventory table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0 CHECK(qty_on_hand >= 0),
			qty_reserved REAL DEFAULT 0 CHECK(qty_reserved >= 0),
			location TEXT DEFAULT '',
			reorder_point REAL DEFAULT 0 CHECK(reorder_point >= 0),
			reorder_qty REAL DEFAULT 0 CHECK(reorder_qty >= 0),
			description TEXT DEFAULT '',
			mpn TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create inventory_transactions table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT, ipn TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('receive','issue','adjust','transfer','return','scrap')),
			qty REAL NOT NULL, reference TEXT, notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory_transactions table: %v", err)
	}

	// Create ncrs table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS ncrs (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
	}

	// Create ecos table (for NCR integration tests)
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT DEFAULT '',
			linked_ncr_id TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	// Create work_orders table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS work_orders (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Create wo_serials table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS wo_serials (
			id INTEGER PRIMARY KEY AUTOINCREMENT, wo_id TEXT NOT NULL,
			serial_number TEXT NOT NULL,
			status TEXT DEFAULT 'building' CHECK(status IN ('building','testing','complete','failed','scrapped')),
			notes TEXT, UNIQUE(serial_number),
			FOREIGN KEY (wo_id) REFERENCES work_orders(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create wo_serials table: %v", err)
	}

	// Create parts table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS parts (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	// Create bom table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS bom (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_ipn TEXT NOT NULL,
			child_ipn TEXT NOT NULL,
			quantity REAL NOT NULL DEFAULT 1,
			reference_designator TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			UNIQUE(parent_ipn, child_ipn),
			FOREIGN KEY (parent_ipn) REFERENCES parts(ipn),
			FOREIGN KEY (child_ipn) REFERENCES parts(ipn)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create bom table: %v", err)
	}

	// Create documents table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create documents table: %v", err)
	}

	// Create document_versions table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS document_versions (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create document_versions table: %v", err)
	}

	// Create app_settings table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create app_settings table: %v", err)
	}

	// Create audit_log table - CRITICAL: Used by almost every handler via logAudit()
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS audit_log (
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	return testDB
}

// createTestUser creates a test user with the given credentials
func createTestUser(t *testing.T, db *sql.DB, username, password, role string, active bool) int {
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

// createTestSessionSimple creates a session token for the given user with default 24h expiry
// Note: Some test files may have their own createTestSession with custom duration parameter
func createTestSessionSimple(t *testing.T, db *sql.DB, userID int) string {
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

// loginAdmin creates an admin user and returns their session token
func loginAdmin(t *testing.T, db *sql.DB) string {
	t.Helper()
	adminID := createTestUser(t, db, "admin", "password", "admin", true)
	return createTestSessionSimple(t, db, adminID)
}

// loginUser creates a regular user and returns their session token
func loginUser(t *testing.T, db *sql.DB, username string) string {
	t.Helper()
	userID := createTestUser(t, db, username, "password", "user", true)
	return createTestSessionSimple(t, db, userID)
}

// authedRequest creates an authenticated HTTP request with a session cookie
func authedRequest(method, path string, body []byte, sessionToken string) *http.Request {
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

// authedJSONRequest creates an authenticated HTTP request with JSON content type
func authedJSONRequest(method, path string, body interface{}, sessionToken string) *http.Request {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	
	req := authedRequest(method, path, bodyBytes, sessionToken)
	req.Header.Set("Content-Type", "application/json")
	
	return req
}

// decodeAPIResponse decodes an APIResponse from a ResponseRecorder
func decodeAPIResponse(t *testing.T, w *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var response APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode API response: %v", err)
	}
	return response
}

// assertStatus checks that the HTTP status code matches expected
func assertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("Expected status %d, got %d. Body: %s", expected, w.Code, w.Body.String())
	}
}
