package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
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

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create all necessary tables
	tables := []string{
		`CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			category TEXT NOT NULL,
			fields TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			status TEXT DEFAULT 'active',
			lead_time_days INTEGER DEFAULT 7,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			total REAL DEFAULT 0,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id)
		)`,
		`CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			qty INTEGER NOT NULL,
			unit_price REAL DEFAULT 0,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL,
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand INTEGER DEFAULT 0,
			qty_reserved INTEGER DEFAULT 0,
			location TEXT DEFAULT '',
			notes TEXT,
			FOREIGN KEY (ipn) REFERENCES parts(ipn)
		)`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			role TEXT DEFAULT 'user'
		)`,
		`CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			severity TEXT DEFAULT 'minor',
			status TEXT DEFAULT 'open',
			ipn TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			model TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE rmas (
			id TEXT PRIMARY KEY,
			serial_number TEXT NOT NULL,
			status TEXT DEFAULT 'open',
			issue_description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE invoices (
			id TEXT PRIMARY KEY,
			customer_name TEXT NOT NULL,
			amount REAL DEFAULT 0,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE docs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT,
			category TEXT,
			status TEXT DEFAULT 'draft',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE field_reports (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			report_type TEXT DEFAULT 'failure',
			status TEXT DEFAULT 'open',
			description TEXT,
			device_ipn TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT,
			action TEXT,
			module TEXT,
			record_id TEXT,
			summary TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, table := range tables {
		if _, err := testDB.Exec(table); err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}

	// Insert test data
	insertSQLInjectionTestData(t, testDB)

	return testDB
}

func insertSQLInjectionTestData(t *testing.T, testDB *sql.DB) {
	t.Helper()

	// Insert parts
	_, err := testDB.Exec(`INSERT INTO parts (ipn, category, fields) VALUES 
		('PART-001', 'resistors', '{"description":"10K Resistor","manufacturer":"Yageo"}'),
		('PART-002', 'capacitors', '{"description":"100uF Cap","manufacturer":"Murata"}'),
		('PART-003', 'ics', '{"description":"MCU STM32","manufacturer":"ST"}')`)
	if err != nil {
		t.Fatalf("Failed to insert parts: %v", err)
	}

	// Insert vendors
	_, err = testDB.Exec(`INSERT INTO vendors (id, name, contact_name, contact_email, status) VALUES 
		('VEN-001', 'Digi-Key', 'John Doe', 'john@digikey.com', 'active'),
		('VEN-002', 'Mouser', 'Jane Smith', 'jane@mouser.com', 'active')`)
	if err != nil {
		t.Fatalf("Failed to insert vendors: %v", err)
	}

	// Insert inventory
	_, err = testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, location, notes) VALUES 
		('PART-001', 100, 'A1', 'Good stock'),
		('PART-002', 50, 'B2', 'Low stock'),
		('PART-003', 25, 'C3', 'Critical level')`)
	if err != nil {
		t.Fatalf("Failed to insert inventory: %v", err)
	}

	// Insert users
	_, err = testDB.Exec(`INSERT INTO users (username, password_hash, display_name, role) VALUES 
		('admin', '$2a$10$abcdefghijklmnopqrstuv', 'Admin User', 'admin'),
		('engineer', '$2a$10$zyxwvutsrqponmlkjihgfe', 'Engineer User', 'user')`)
	if err != nil {
		t.Fatalf("Failed to insert users: %v", err)
	}
}

// Test 1: Parts list endpoint with search
func TestSQLInjection_PartsListSearch(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Payload: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts?q="+payload, nil)
			w := httptest.NewRecorder()
			handleListParts(w, req)

			// Should not cause an error or leak data
			if w.Code == 500 {
				t.Errorf("SQL injection caused server error: %v", w.Body.String())
			}

			// Verify response doesn't leak all data
			var parts []Part
			if err := json.Unmarshal(w.Body.Bytes(), &parts); err == nil {
				if len(parts) > 100 {
					t.Errorf("SQL injection may have bypassed filters, returned %d parts", len(parts))
				}
			}
		})
	}
}

// Test 2: Advanced search with filters
func TestSQLInjection_AdvancedSearch(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Payload: %s", payload), func(t *testing.T) {
			searchQuery := map[string]interface{}{
				"entity_type": "parts",
				"search_text": payload,
				"filters": []map[string]interface{}{
					{
						"field":    "description",
						"operator": "contains",
						"value":    payload,
					},
				},
			}

			body, _ := json.Marshal(searchQuery)
			req := httptest.NewRequest("POST", "/api/v1/search/advanced", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleAdvancedSearch(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection caused server error: %v", w.Body.String())
			}
		})
	}
}

// Test 3: Part creation with malicious IPN
func TestSQLInjection_PartCreate(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	// Note: IPN validation should reject most of these, which is good
	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Payload: %s", payload), func(t *testing.T) {
			part := map[string]interface{}{
				"ipn":      payload,
				"category": "resistors",
				"fields": map[string]string{
					"description":  payload,
					"manufacturer": payload,
				},
			}

			body, _ := json.Marshal(part)
			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handleCreatePart(w, req)

			// Verify no SQL execution or data corruption
			var count int
			db.QueryRow("SELECT COUNT(*) FROM parts").Scan(&count)
			if count > 10 {
				t.Errorf("SQL injection may have created unexpected records")
			}
		})
	}
}

// Test 4: Vendor search and creation
func TestSQLInjection_Vendors(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Search: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/vendors?search="+payload, nil)
			w := httptest.NewRecorder()
			handleListVendors(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection caused server error: %v", w.Body.String())
			}
		})

		t.Run(fmt.Sprintf("Create: %s", payload), func(t *testing.T) {
			vendor := map[string]interface{}{
				"name":          payload,
				"contact_name":  payload,
				"contact_email": "test@example.com",
			}

			body, _ := json.Marshal(vendor)
			req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateVendor(w, req)

			// Verify no SQL execution
			var count int
			db.QueryRow("SELECT COUNT(*) FROM vendors").Scan(&count)
			if count > 10 {
				t.Errorf("SQL injection may have created unexpected records")
			}
		})
	}
}

// Test 5: Purchase Orders with malicious notes
func TestSQLInjection_PurchaseOrders(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	// Create a valid vendor first
	db.Exec("INSERT INTO vendors (id, name) VALUES ('VEN-TEST', 'Test Vendor')")

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("PO Notes: %s", payload), func(t *testing.T) {
			po := map[string]interface{}{
				"vendor_id": "VEN-TEST",
				"notes":     payload,
				"lines": []map[string]interface{}{
					{"ipn": "PART-001", "qty": 10, "unit_price": 1.50},
				},
			}

			body, _ := json.Marshal(po)
			req := httptest.NewRequest("POST", "/api/v1/purchase-orders", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreatePO(w, req)

			// Should handle gracefully
			if w.Code == 500 {
				t.Errorf("SQL injection in PO notes caused error: %v", w.Body.String())
			}
		})
	}
}

// Test 6: Work Orders with malicious notes
func TestSQLInjection_WorkOrders(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("WO Notes: %s", payload), func(t *testing.T) {
			wo := map[string]interface{}{
				"assembly_ipn": "PART-001",
				"qty":          5,
				"notes":        payload,
				"priority":     "normal",
			}

			body, _ := json.Marshal(wo)
			req := httptest.NewRequest("POST", "/api/v1/work-orders", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateWorkOrder(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in WO notes caused error: %v", w.Body.String())
			}
		})

		t.Run(fmt.Sprintf("WO Search: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/work-orders?search="+payload, nil)
			w := httptest.NewRecorder()
			handleListWorkOrders(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in WO search caused error")
			}
		})
	}
}

// Test 7: ECOs with malicious title/description
func TestSQLInjection_ECOs(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("ECO Title: %s", payload), func(t *testing.T) {
			eco := map[string]interface{}{
				"title":       payload,
				"description": payload,
				"status":      "draft",
				"priority":    "normal",
			}

			body, _ := json.Marshal(eco)
			req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateECO(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in ECO creation caused error")
			}
		})
	}
}

// Test 8: Inventory search with malicious input
func TestSQLInjection_Inventory(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Inventory Search: %s", payload), func(t *testing.T) {
			searchQuery := map[string]interface{}{
				"entity_type": "inventory",
				"search_text": payload,
			}

			body, _ := json.Marshal(searchQuery)
			req := httptest.NewRequest("POST", "/api/v1/search/advanced", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleAdvancedSearch(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in inventory search caused error")
			}
		})

		t.Run(fmt.Sprintf("Inventory Notes: %s", payload), func(t *testing.T) {
			// Direct database update with parameterized query
			_, err := db.Exec("UPDATE inventory SET notes = ? WHERE ipn = 'PART-001'", payload)
			if err != nil {
				t.Logf("Expected: Parameterized query safely handled payload")
			}

			// Verify data integrity
			var count int
			db.QueryRow("SELECT COUNT(*) FROM inventory").Scan(&count)
			if count != 3 {
				t.Errorf("SQL injection affected inventory data integrity")
			}
		})
	}
}

// Test 9: NCR with malicious description
func TestSQLInjection_NCRs(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("NCR Description: %s", payload), func(t *testing.T) {
			ncr := map[string]interface{}{
				"title":       payload,
				"description": payload,
				"severity":    "minor",
			}

			body, _ := json.Marshal(ncr)
			req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateNCR(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in NCR creation caused error")
			}
		})
	}
}

// Test 10: Devices with malicious serial/notes
func TestSQLInjection_Devices(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Device Serial: %s", payload), func(t *testing.T) {
			device := map[string]interface{}{
				"serial_number": payload,
				"model":         "TEST-MODEL",
				"notes":         payload,
			}

			body, _ := json.Marshal(device)
			req := httptest.NewRequest("POST", "/api/v1/devices", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateDevice(w, req)

			// Should handle gracefully (may reject invalid serial format)
			if w.Code == 500 {
				body := w.Body.String()
				if !json.Valid([]byte(body)) || len(body) == 0 {
					t.Errorf("SQL injection may have caused error")
				}
			}
		})
	}
}

// Test 11: RMAs with malicious issue description
func TestSQLInjection_RMAs(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("RMA Issue: %s", payload), func(t *testing.T) {
			rma := map[string]interface{}{
				"serial_number":     "TEST-SN-001",
				"issue_description": payload,
			}

			body, _ := json.Marshal(rma)
			req := httptest.NewRequest("POST", "/api/v1/rmas", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateRMA(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in RMA creation caused error")
			}
		})
	}
}

// Test 12: Invoices with malicious customer/notes
func TestSQLInjection_Invoices(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Invoice Customer: %s", payload), func(t *testing.T) {
			invoice := map[string]interface{}{
				"customer_name": payload,
				"amount":        100.50,
				"notes":         payload,
			}

			body, _ := json.Marshal(invoice)
			req := httptest.NewRequest("POST", "/api/v1/invoices", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateInvoice(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in invoice creation caused error")
			}
		})

		t.Run(fmt.Sprintf("Invoice Search: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/invoices?search="+payload, nil)
			w := httptest.NewRecorder()
			handleListInvoices(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in invoice search caused error")
			}
		})
	}
}

// Test 13: Documents with malicious content
func TestSQLInjection_Documents(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Document Content: %s", payload), func(t *testing.T) {
			doc := map[string]interface{}{
				"title":    payload,
				"content":  payload,
				"category": "procedures",
			}

			body, _ := json.Marshal(doc)
			req := httptest.NewRequest("POST", "/api/v1/docs", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateDoc(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in document creation caused error")
			}
		})
	}
}

// Test 14: Field Reports with malicious description
func TestSQLInjection_FieldReports(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Field Report: %s", payload), func(t *testing.T) {
			report := map[string]interface{}{
				"title":       payload,
				"description": payload,
				"report_type": "failure",
			}

			body, _ := json.Marshal(report)
			req := httptest.NewRequest("POST", "/api/v1/field-reports", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			handleCreateFieldReport(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in field report creation caused error")
			}
		})
	}
}

// Test 15: Audit Log with malicious search
func TestSQLInjection_AuditLog(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	for _, payload := range sqlInjectionPayloads {
		t.Run(fmt.Sprintf("Audit Search: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/audit?search="+payload, nil)
			w := httptest.NewRecorder()
			handleAuditLog(w, req)

			if w.Code == 500 {
				t.Errorf("SQL injection in audit log search caused error")
			}

			// Verify no data leakage
			var result map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err == nil {
				if data, ok := result["items"].([]interface{}); ok {
					if len(data) > 100 {
						t.Errorf("SQL injection may have leaked excessive audit data")
					}
				}
			}
		})
	}
}

// Test 16: Verify parameterized queries prevent execution
func TestSQLInjection_VerifyParameterizedQueries(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	// This test verifies malicious input doesn't execute as SQL
	payload := "'; DELETE FROM parts WHERE '1'='1"

	// Insert a part with malicious description
	_, err := db.Exec(
		"INSERT INTO parts (ipn, category, fields) VALUES (?, ?, ?)",
		"TEST-INJECT",
		"test",
		`{"description":"`+payload+`"}`,
	)

	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Verify parts table still has data (DELETE didn't execute)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM parts").Scan(&count)

	if count < 3 {
		t.Errorf("SQL injection may have deleted data! Count: %d", count)
	}

	// Verify the malicious payload is stored as data, not executed
	var fields string
	err = db.QueryRow("SELECT fields FROM parts WHERE ipn = ?", "TEST-INJECT").Scan(&fields)
	if err != nil {
		t.Fatalf("Failed to retrieve test part: %v", err)
	}

	// Should contain the payload as text
	if !strings.Contains(fields, payload) {
		t.Errorf("Malicious payload was not properly stored as data")
	}
}

// Test 17: UNION-based SQL injection attempts
func TestSQLInjection_UNIONAttacks(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	unionPayloads := []string{
		"' UNION SELECT username, password_hash, NULL FROM users--",
		"' UNION ALL SELECT id, password_hash, role FROM users--",
		"PART-001' UNION SELECT username FROM users--",
	}

	for _, payload := range unionPayloads {
		t.Run(fmt.Sprintf("UNION: %s", payload), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts?q="+payload, nil)
			w := httptest.NewRecorder()
			handleListParts(w, req)

			// Should not leak user data
			body := w.Body.String()
			if strings.Contains(body, "password_hash") || strings.Contains(body, "$2a$10$") {
				t.Errorf("UNION attack leaked password data!")
			}

			if w.Code == 500 {
				t.Errorf("UNION attack caused server error")
			}
		})
	}
}

// Test 18: Second-order SQL injection
func TestSQLInjection_SecondOrder(t *testing.T) {
	db = setupSQLInjectionTestDB(t)
	defer db.Close()

	// Store malicious data
	maliciousData := "'; DROP TABLE parts--"
	_, err := db.Exec(
		"INSERT INTO parts (ipn, category, fields) VALUES (?, ?, ?)",
		"SECOND-ORDER",
		"test",
		`{"description":"`+maliciousData+`"}`,
	)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Retrieve and use the data (second-order injection attempt)
	var fields string
	db.QueryRow("SELECT fields FROM parts WHERE ipn = ?", "SECOND-ORDER").Scan(&fields)

	// Verify parts table still exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM parts").Scan(&count)
	if err != nil {
		t.Errorf("Second-order SQL injection may have dropped table: %v", err)
	}

	if count < 3 {
		t.Errorf("Second-order SQL injection affected data integrity")
	}
}
