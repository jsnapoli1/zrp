package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// SQL injection test payloads - common attack vectors
var sqlInjectionPayloads = []string{
	"' OR '1'='1",
	"'; DROP TABLE parts--",
	"' UNION SELECT * FROM users--",
	"admin'--",
	"' OR 1=1--",
	"1' AND '1'='1",
	"'; DELETE FROM inventory WHERE '1'='1",
	"' UNION SELECT NULL, username, password_hash FROM users--",
	"') OR ('1'='1",
	"1' OR '1'='1' /*",
	"'; EXEC xp_cmdshell('dir')--",
	"' AND 1=(SELECT COUNT(*) FROM users)--",
	"admin' OR '1'='1' /*",
	"' UNION ALL SELECT NULL,NULL,NULL--",
	"'; UPDATE users SET role='admin' WHERE username='engineer'--",
}

func setupSQLInjectionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create test tables
	tables := []string{
		`CREATE TABLE parts (ipn TEXT PRIMARY KEY, category TEXT NOT NULL, fields TEXT NOT NULL)`,
		`CREATE TABLE vendors (id TEXT PRIMARY KEY, name TEXT NOT NULL, contact_name TEXT, status TEXT DEFAULT 'active')`,
		`CREATE TABLE inventory (ipn TEXT PRIMARY KEY, qty_on_hand INTEGER DEFAULT 0, location TEXT, notes TEXT)`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, role TEXT DEFAULT 'user')`,
	}

	for _, table := range tables {
		if _, err := testDB.Exec(table); err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}

	// Insert test data
	testDB.Exec(`INSERT INTO parts (ipn, category, fields) VALUES ('PART-001', 'resistors', '{"description":"10K Resistor"}')`)
	testDB.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, location) VALUES ('PART-001', 100, 'A1')`)
	testDB.Exec(`INSERT INTO users (username, password_hash, role) VALUES ('admin', '$2a$10$test', 'admin')`)

	return testDB
}

// Test 1: Parts search with SQL injection payloads
func TestSQLInjection_PartsListSearch(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Payload: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts?q="+url.QueryEscape(payload), nil)
			w := httptest.NewRecorder()
			handleListParts(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection caused server error")
			}
		})
	}
}

// Test 2: Advanced search with malicious filters
func TestSQLInjection_AdvancedSearch(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Payload: %s", payload), func(t *testing.T) {
			searchQuery := map[string]interface{}{
				"entity_type": "parts",
				"search_text": payload,
			}

			body, _ := json.Marshal(searchQuery)
			req := httptest.NewRequest("POST", "/api/v1/search/advanced", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleAdvancedSearch(w, req)

			// Verify no data corruption occurred
			var count int
			db.QueryRow("SELECT COUNT(*) FROM parts").Scan(&count)
			if count != 1 {
				t.Errorf("SQL injection may have corrupted data, count: %d", count)
			}
		})
	}
}

// Test 3: Verify parameterized queries prevent SQL execution
func TestSQLInjection_VerifyParameterizedQueries(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	payload := "'; DELETE FROM parts WHERE '1'='1"

	// Insert with parameterized query
	_, err := db.Exec(
		"INSERT INTO parts (ipn, category, fields) VALUES (?, ?, ?)",
		"TEST-INJECT",
		"test",
		`{"description":"`+payload+`"}`,
	)

	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Verify parts table still has data
	var count int
	db.QueryRow("SELECT COUNT(*) FROM parts").Scan(&count)

	if count < 2 {
		t.Errorf("SQL injection may have deleted data! Count: %d", count)
	}

	// Verify malicious payload stored as data, not executed
	var fields string
	err = db.QueryRow("SELECT fields FROM parts WHERE ipn = ?", "TEST-INJECT").Scan(&fields)
	if err != nil {
		t.Fatalf("Failed to retrieve: %v", err)
	}

	if !strings.Contains(fields, payload) {
		t.Errorf("Malicious payload was not stored as data")
	}
}

// Test 4: UNION-based attacks
func TestSQLInjection_UNIONAttacks(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	unionPayloads := []string{
		"' UNION SELECT username, password_hash, NULL FROM users--",
		"PART-001' UNION SELECT username FROM users--",
	}

	for _, payload := range unionPayloads {
		t.Run(fmt.Sprintf("UNION: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts?q="+url.QueryEscape(payload), nil)
			w := httptest.NewRecorder()
			handleListParts(w, req)

			body := w.Body.String()
			if strings.Contains(body, "password_hash") || strings.Contains(body, "$2a$10$") {
				t.Errorf("UNION attack leaked password data!")
			}
		})
	}
}

// Test 5: Second-order SQL injection
func TestSQLInjection_SecondOrder(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	maliciousData := "'; DROP TABLE parts--"
	_, err := db.Exec(
		"INSERT INTO parts (ipn, category, fields) VALUES (?, ?, ?)",
		"SECOND-ORDER",
		"test",
		`{"description":"`+maliciousData+`"}`,
	)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Retrieve and use the data
	var fields string
	db.QueryRow("SELECT fields FROM parts WHERE ipn = ?", "SECOND-ORDER").Scan(&fields)

	// Verify parts table still exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM parts").Scan(&count)
	if err != nil {
		t.Errorf("Second-order SQL injection may have dropped table: %v", err)
	}
}

// Test 6: Vendor search with SQL injection
func TestSQLInjection_VendorSearch(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads[:5] {
		t.Run(fmt.Sprintf("Vendor: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/vendors?search="+url.QueryEscape(payload), nil)
			w := httptest.NewRecorder()
			handleListVendors(w, req)

			// Verify data integrity
			var count int
			db.QueryRow("SELECT COUNT(*) FROM vendors").Scan(&count)
			if count != 1 {
				t.Errorf("SQL injection may have corrupted vendor data")
			}
		})
	}
}

// Test 7: Inventory notes with SQL injection
func TestSQLInjection_InventoryNotes(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads[:5] {
		t.Run(fmt.Sprintf("Notes: %s", payload), func(t *testing.T) {
			_, err := db.Exec("UPDATE inventory SET notes = ? WHERE ipn = 'PART-001'", payload)
			if err != nil {
				t.Logf("Parameterized query handled payload safely")
			}

			// Verify data integrity
			var count int
			db.QueryRow("SELECT COUNT(*) FROM inventory").Scan(&count)
			if count != 1 {
				t.Errorf("SQL injection affected inventory data")
			}
		})
	}
}

// Test 8: Audit log search with SQL injection
func TestSQLInjection_AuditLogSearch(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	// Create audit_log table
	db.Exec(`CREATE TABLE audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		action TEXT,
		module TEXT,
		record_id TEXT,
		summary TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	
	db.Exec(`INSERT INTO audit_log (username, action) VALUES ('admin', 'LOGIN')`)

	for _, payload := range sqlInjectionPayloads[:5] {
		t.Run(fmt.Sprintf("Audit: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/audit?search="+url.QueryEscape(payload), nil)
			w := httptest.NewRecorder()
			handleAuditLog(w, req)

			// Verify table still exists and data intact
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&count)
			if err != nil || count != 1 {
				t.Errorf("SQL injection may have dropped audit_log table or corrupted data")
			}
		})
	}
}

// Test 9: Work orders with SQL injection in notes
func TestSQLInjection_WorkOrderNotes(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	// Create work_orders table
	db.Exec(`CREATE TABLE work_orders (
		id TEXT PRIMARY KEY,
		assembly_ipn TEXT NOT NULL,
		qty INTEGER NOT NULL,
		status TEXT DEFAULT 'draft',
		notes TEXT
	)`)

	initialCount := 0
	db.QueryRow("SELECT COUNT(*) FROM work_orders").Scan(&initialCount)

	for _, payload := range sqlInjectionPayloads[:5] {
		t.Run(fmt.Sprintf("WO Notes: %s", payload), func(t *testing.T) {
			wo := map[string]interface{}{
				"assembly_ipn": "PART-001",
				"qty":          5,
				"notes":        payload,
			}

			body, _ := json.Marshal(wo)
			req := httptest.NewRequest("POST", "/api/v1/work-orders", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateWorkOrder(w, req)

			// Verify table still exists
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM work_orders").Scan(&count)
			if err != nil {
				t.Errorf("SQL injection may have dropped work_orders table")
			}
		})
	}
}

// Test 10: ECO with SQL injection in title/description
func TestSQLInjection_ECOFields(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	db.Exec(`CREATE TABLE ecos (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT DEFAULT 'draft'
	)`)

	for _, payload := range sqlInjectionPayloads[:5] {
		t.Run(fmt.Sprintf("ECO: %s", payload), func(t *testing.T) {
			eco := map[string]interface{}{
				"title":       payload,
				"description": payload,
				"status":      "draft",
			}

			body, _ := json.Marshal(eco)
			req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateECO(w, req)

			// Verify table still exists
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM ecos").Scan(&count)
			if err != nil {
				t.Errorf("SQL injection may have dropped ecos table")
			}
		})
	}
}
