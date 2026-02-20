package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupRFQTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create rfqs table
	_, err = testDB.Exec(`
		CREATE TABLE rfqs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			created_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			due_date TEXT,
			notes TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create rfqs table: %v", err)
	}

	// Create rfq_lines table
	_, err = testDB.Exec(`
		CREATE TABLE rfq_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty REAL NOT NULL,
			unit TEXT DEFAULT 'EA',
			FOREIGN KEY (rfq_id) REFERENCES rfqs(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create rfq_lines table: %v", err)
	}

	// Create rfq_vendors table
	_, err = testDB.Exec(`
		CREATE TABLE rfq_vendors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			vendor_id TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			quoted_at DATETIME,
			notes TEXT,
			FOREIGN KEY (rfq_id) REFERENCES rfqs(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create rfq_vendors table: %v", err)
	}

	// Create rfq_quotes table
	_, err = testDB.Exec(`
		CREATE TABLE rfq_quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			rfq_id TEXT NOT NULL,
			rfq_vendor_id INTEGER NOT NULL,
			rfq_line_id INTEGER NOT NULL,
			unit_price REAL NOT NULL,
			lead_time_days INTEGER DEFAULT 0,
			moq INTEGER DEFAULT 1,
			notes TEXT,
			FOREIGN KEY (rfq_id) REFERENCES rfqs(id) ON DELETE CASCADE,
			FOREIGN KEY (rfq_vendor_id) REFERENCES rfq_vendors(id) ON DELETE CASCADE,
			FOREIGN KEY (rfq_line_id) REFERENCES rfq_lines(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create rfq_quotes table: %v", err)
	}

	// Create vendors table
	_, err = testDB.Exec(`
		CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			notes TEXT,
			status TEXT DEFAULT 'active',
			lead_time_days INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create vendors table: %v", err)
	}

	// Create purchase_orders table (for award functionality)
	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date TEXT,
			received_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create purchase_orders table: %v", err)
	}

	// Create po_lines table
	_, err = testDB.Exec(`
		CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			mpn TEXT,
			manufacturer TEXT,
			qty_ordered REAL NOT NULL,
			qty_received REAL DEFAULT 0,
			unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create po_lines table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	return testDB
}

func insertTestRFQ(t *testing.T, db *sql.DB, id, title, status, createdBy string) {
	_, err := db.Exec(
		"INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))",
		id, title, status, createdBy,
	)
	if err != nil {
		t.Fatalf("Failed to insert test RFQ: %v", err)
	}
}

func insertTestRFQLine(t *testing.T, db *sql.DB, rfqID, ipn string, qty float64) int {
	res, err := db.Exec(
		"INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?, ?, ?, ?, ?)",
		rfqID, ipn, "Test part", qty, "EA",
	)
	if err != nil {
		t.Fatalf("Failed to insert test RFQ line: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func insertTestRFQVendor(t *testing.T, db *sql.DB, rfqID, vendorID, status string) int {
	res, err := db.Exec(
		"INSERT INTO rfq_vendors (rfq_id, vendor_id, status) VALUES (?, ?, ?)",
		rfqID, vendorID, status,
	)
	if err != nil {
		t.Fatalf("Failed to insert test RFQ vendor: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func insertTestVendorRFQ(t *testing.T, db *sql.DB, id, name string) {
	_, err := db.Exec(
		"INSERT INTO vendors (id, name, created_at) VALUES (?, ?, datetime('now'))",
		id, name,
	)
	if err != nil {
		t.Fatalf("Failed to insert test vendor: %v", err)
	}
}

// Test handleListRFQs - Empty
func TestHandleListRFQs_Empty(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/rfq", nil)
	w := httptest.NewRecorder()

	handleListRFQs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	rfqs, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(rfqs) != 0 {
		t.Errorf("Expected empty array, got %d RFQs", len(rfqs))
	}
}

// Test handleListRFQs - With Data
func TestHandleListRFQs_WithData(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRFQ(t, db, "RFQ-001", "Test RFQ 1", "draft", "user1")
	insertTestRFQ(t, db, "RFQ-002", "Test RFQ 2", "sent", "user1")
	insertTestRFQ(t, db, "RFQ-003", "Test RFQ 3", "awarded", "user2")

	req := httptest.NewRequest("GET", "/api/rfq", nil)
	w := httptest.NewRecorder()

	handleListRFQs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	rfqsData, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(rfqsData) != 3 {
		t.Errorf("Expected 3 RFQs, got %d", len(rfqsData))
	}
}

// Test handleGetRFQ - Success with full data
func TestHandleGetRFQ_Success(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestVendorRFQ(t, db, "V-001", "Vendor A")
	insertTestRFQ(t, db, "RFQ-001", "Test RFQ", "draft", "testuser")
	lineID := insertTestRFQLine(t, db, "RFQ-001", "IPN-001", 100)
	vendorID := insertTestRFQVendor(t, db, "RFQ-001", "V-001", "pending")

	// Insert a quote
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-001", vendorID, lineID, 10.50, 14, 50)

	req := httptest.NewRequest("GET", "/api/rfq/RFQ-001", nil)
	w := httptest.NewRecorder()

	handleGetRFQ(w, req, "RFQ-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	rfq := resp.Data.(map[string]interface{})
	if rfq["id"] != "RFQ-001" {
		t.Errorf("Expected ID RFQ-001, got %v", rfq["id"])
	}

	// Verify lines are included
	lines, ok := rfq["lines"].([]interface{})
	if !ok {
		t.Fatalf("Expected lines to be an array")
	}
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}

	// Verify vendors are included
	vendors, ok := rfq["vendors"].([]interface{})
	if !ok {
		t.Fatalf("Expected vendors to be an array")
	}
	if len(vendors) != 1 {
		t.Errorf("Expected 1 vendor, got %d", len(vendors))
	}

	// Verify quotes are included
	quotes, ok := rfq["quotes"].([]interface{})
	if !ok {
		t.Fatalf("Expected quotes to be an array")
	}
	if len(quotes) != 1 {
		t.Errorf("Expected 1 quote, got %d", len(quotes))
	}
}

// Test handleGetRFQ - Not Found
func TestHandleGetRFQ_NotFound(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/rfq/RFQ-999", nil)
	w := httptest.NewRecorder()

	handleGetRFQ(w, req, "RFQ-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test handleCreateRFQ - Success
func TestHandleCreateRFQ_Success(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestVendorRFQ(t, db, "V-001", "Vendor A")

	rfq := map[string]interface{}{
		"title":    "New RFQ",
		"due_date": "2026-12-31",
		"notes":    "Test notes",
		"lines": []map[string]interface{}{
			{"ipn": "IPN-001", "description": "Part 1", "qty": 100, "unit": "EA"},
			{"ipn": "IPN-002", "description": "Part 2", "qty": 50, "unit": "EA"},
		},
		"vendors": []map[string]interface{}{
			{"vendor_id": "V-001", "notes": "Preferred vendor"},
		},
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	created := resp.Data.(map[string]interface{})
	if created["title"] != "New RFQ" {
		t.Errorf("Expected title 'New RFQ', got %v", created["title"])
	}
	if created["status"] != "draft" {
		t.Errorf("Expected default status 'draft', got %v", created["status"])
	}

	// Verify lines were created
	lines := created["lines"].([]interface{})
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	// Verify vendors were created
	vendors := created["vendors"].([]interface{})
	if len(vendors) != 1 {
		t.Errorf("Expected 1 vendor, got %d", len(vendors))
	}
	vendor := vendors[0].(map[string]interface{})
	if vendor["status"] != "pending" {
		t.Errorf("Expected vendor status 'pending', got %v", vendor["status"])
	}
}

// Test handleCreateRFQ - Validation
func TestHandleCreateRFQ_Validation(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	rfq := map[string]interface{}{
		"notes": "Missing title",
	}

	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQ(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for missing title, got %d", w.Code)
	}
}

// Test handleUpdateRFQ - Success
func TestHandleUpdateRFQ_Success(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestVendorRFQ(t, db, "V-001", "Vendor A")
	insertTestRFQ(t, db, "RFQ-001", "Original Title", "draft", "user1")
	insertTestRFQLine(t, db, "RFQ-001", "IPN-001", 50)

	update := map[string]interface{}{
		"title":    "Updated Title",
		"due_date": "2026-06-30",
		"notes":    "Updated notes",
		"lines": []map[string]interface{}{
			{"ipn": "IPN-002", "description": "New part", "qty": 100, "unit": "EA"},
		},
		"vendors": []map[string]interface{}{
			{"vendor_id": "V-001"},
		},
	}

	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/rfq/RFQ-001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateRFQ(w, req, "RFQ-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify title was updated
	var title string
	db.QueryRow("SELECT title FROM rfqs WHERE id=?", "RFQ-001").Scan(&title)
	if title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %s", title)
	}

	// Verify lines were replaced
	var lineCount int
	db.QueryRow("SELECT COUNT(*) FROM rfq_lines WHERE rfq_id=?", "RFQ-001").Scan(&lineCount)
	if lineCount != 1 {
		t.Errorf("Expected 1 line after update, got %d", lineCount)
	}

	var lineIPN string
	db.QueryRow("SELECT ipn FROM rfq_lines WHERE rfq_id=?", "RFQ-001").Scan(&lineIPN)
	if lineIPN != "IPN-002" {
		t.Errorf("Expected line IPN 'IPN-002', got %s", lineIPN)
	}
}

// Test handleDeleteRFQ - Success
func TestHandleDeleteRFQ_Success(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRFQ(t, db, "RFQ-DELETE", "To Delete", "draft", "user1")
	insertTestRFQLine(t, db, "RFQ-DELETE", "IPN-001", 10)

	req := httptest.NewRequest("DELETE", "/api/rfq/RFQ-DELETE", nil)
	w := httptest.NewRecorder()

	handleDeleteRFQ(w, req, "RFQ-DELETE")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify RFQ was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM rfqs WHERE id=?", "RFQ-DELETE").Scan(&count)
	if count != 0 {
		t.Errorf("Expected RFQ to be deleted")
	}

	// Verify lines were cascade deleted
	db.QueryRow("SELECT COUNT(*) FROM rfq_lines WHERE rfq_id=?", "RFQ-DELETE").Scan(&count)
	if count != 0 {
		t.Errorf("Expected RFQ lines to be cascade deleted")
	}
}

// Test handleSendRFQ - Draft to Sent transition
func TestHandleSendRFQ_Success(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRFQ(t, db, "RFQ-SEND", "Send Test", "draft", "user1")
	insertTestRFQVendor(t, db, "RFQ-SEND", "V-001", "pending")

	req := httptest.NewRequest("POST", "/api/rfq/RFQ-SEND/send", nil)
	w := httptest.NewRecorder()

	handleSendRFQ(w, req, "RFQ-SEND")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify status changed to sent
	var status string
	db.QueryRow("SELECT status FROM rfqs WHERE id=?", "RFQ-SEND").Scan(&status)
	if status != "sent" {
		t.Errorf("Expected status 'sent', got %s", status)
	}
}

// Test handleSendRFQ - Invalid status transition
func TestHandleSendRFQ_InvalidStatus(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRFQ(t, db, "RFQ-SENT", "Already Sent", "sent", "user1")

	req := httptest.NewRequest("POST", "/api/rfq/RFQ-SENT/send", nil)
	w := httptest.NewRecorder()

	handleSendRFQ(w, req, "RFQ-SENT")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid transition, got %d", w.Code)
	}
}

// Test handleAwardRFQ - Success
func TestHandleAwardRFQ_Success(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestVendorRFQ(t, db, "V-001", "Winner Vendor")
	insertTestRFQ(t, db, "RFQ-AWARD", "Award Test", "sent", "user1")
	lineID := insertTestRFQLine(t, db, "RFQ-AWARD", "IPN-001", 100)
	vendorID := insertTestRFQVendor(t, db, "RFQ-AWARD", "V-001", "quoted")

	// Add a quote
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-AWARD", vendorID, lineID, 12.50, 10, 25)

	award := map[string]string{"vendor_id": "V-001"}
	body, _ := json.Marshal(award)
	req := httptest.NewRequest("POST", "/api/rfq/RFQ-AWARD/award", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleAwardRFQ(w, req, "RFQ-AWARD")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	result := resp.Data.(map[string]interface{})

	// Verify status changed to awarded
	var status string
	db.QueryRow("SELECT status FROM rfqs WHERE id=?", "RFQ-AWARD").Scan(&status)
	if status != "awarded" {
		t.Errorf("Expected status 'awarded', got %s", status)
	}

	// Verify PO was created
	poID := result["po_id"].(string)
	if poID == "" {
		t.Errorf("Expected PO ID to be returned")
	}

	var poCount int
	db.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE id=?", poID).Scan(&poCount)
	if poCount != 1 {
		t.Errorf("Expected PO to be created")
	}

	// Verify PO lines were created
	var poLineCount int
	db.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id=?", poID).Scan(&poLineCount)
	if poLineCount != 1 {
		t.Errorf("Expected 1 PO line, got %d", poLineCount)
	}
}

// Test handleAwardRFQ - Missing vendor
func TestHandleAwardRFQ_MissingVendor(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRFQ(t, db, "RFQ-001", "Test", "sent", "user1")

	award := map[string]string{}
	body, _ := json.Marshal(award)
	req := httptest.NewRequest("POST", "/api/rfq/RFQ-001/award", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleAwardRFQ(w, req, "RFQ-001")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for missing vendor_id, got %d", w.Code)
	}
}

// Test handleCloseRFQ - Success
func TestHandleCloseRFQ_Success(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRFQ(t, db, "RFQ-CLOSE", "Close Test", "awarded", "user1")

	req := httptest.NewRequest("POST", "/api/rfq/RFQ-CLOSE/close", nil)
	w := httptest.NewRecorder()

	handleCloseRFQ(w, req, "RFQ-CLOSE")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify status changed to closed
	var status string
	db.QueryRow("SELECT status FROM rfqs WHERE id=?", "RFQ-CLOSE").Scan(&status)
	if status != "closed" {
		t.Errorf("Expected status 'closed', got %s", status)
	}
}

// Test handleCloseRFQ - Invalid status
func TestHandleCloseRFQ_InvalidStatus(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRFQ(t, db, "RFQ-DRAFT", "Draft RFQ", "draft", "user1")

	req := httptest.NewRequest("POST", "/api/rfq/RFQ-DRAFT/close", nil)
	w := httptest.NewRecorder()

	handleCloseRFQ(w, req, "RFQ-DRAFT")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid status transition, got %d", w.Code)
	}
}

// Test handleCreateRFQQuote - Success
func TestHandleCreateRFQQuote_Success(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRFQ(t, db, "RFQ-QUOTE", "Quote Test", "sent", "user1")
	lineID := insertTestRFQLine(t, db, "RFQ-QUOTE", "IPN-001", 100)
	vendorID := insertTestRFQVendor(t, db, "RFQ-QUOTE", "V-001", "pending")

	quote := map[string]interface{}{
		"rfq_vendor_id":  vendorID,
		"rfq_line_id":    lineID,
		"unit_price":     15.75,
		"lead_time_days": 14,
		"moq":            50,
		"notes":          "Best price for qty 100+",
	}

	body, _ := json.Marshal(quote)
	req := httptest.NewRequest("POST", "/api/rfq/RFQ-QUOTE/quotes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateRFQQuote(w, req, "RFQ-QUOTE")

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	created := resp.Data.(map[string]interface{})

	if created["unit_price"] != 15.75 {
		t.Errorf("Expected unit_price 15.75, got %v", created["unit_price"])
	}

	// Verify vendor status changed to quoted
	var vendorStatus string
	db.QueryRow("SELECT status FROM rfq_vendors WHERE id=?", vendorID).Scan(&vendorStatus)
	if vendorStatus != "quoted" {
		t.Errorf("Expected vendor status 'quoted', got %s", vendorStatus)
	}
}

// Test handleUpdateRFQQuote - Success
func TestHandleUpdateRFQQuote_Success(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestRFQ(t, db, "RFQ-UPDATE", "Update Test", "sent", "user1")
	lineID := insertTestRFQLine(t, db, "RFQ-UPDATE", "IPN-001", 100)
	vendorID := insertTestRFQVendor(t, db, "RFQ-UPDATE", "V-001", "pending")

	res, _ := db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq) VALUES (?, ?, ?, ?, ?, ?)",
		"RFQ-UPDATE", vendorID, lineID, 10.0, 10, 25)
	quoteID, _ := res.LastInsertId()

	update := map[string]interface{}{
		"unit_price":     12.50,
		"lead_time_days": 7,
		"moq":            10,
		"notes":          "Updated quote",
	}

	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/rfq/RFQ-UPDATE/quotes/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateRFQQuote(w, req, "RFQ-UPDATE", string(rune(quoteID)))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify quote was updated
	var unitPrice float64
	db.QueryRow("SELECT unit_price FROM rfq_quotes WHERE id=?", quoteID).Scan(&unitPrice)
	if unitPrice != 12.50 {
		t.Errorf("Expected unit_price 12.50, got %.2f", unitPrice)
	}
}

// Test handleCompareRFQ
func TestHandleCompareRFQ(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestVendorRFQ(t, db, "V-001", "Vendor A")
	insertTestVendorRFQ(t, db, "V-002", "Vendor B")
	insertTestRFQ(t, db, "RFQ-COMPARE", "Compare Test", "sent", "user1")
	
	line1 := insertTestRFQLine(t, db, "RFQ-COMPARE", "IPN-001", 100)
	line2 := insertTestRFQLine(t, db, "RFQ-COMPARE", "IPN-002", 50)
	
	vendor1 := insertTestRFQVendor(t, db, "RFQ-COMPARE", "V-001", "quoted")
	vendor2 := insertTestRFQVendor(t, db, "RFQ-COMPARE", "V-002", "quoted")

	// Add quotes
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days) VALUES (?, ?, ?, ?, ?)",
		"RFQ-COMPARE", vendor1, line1, 10.0, 14)
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days) VALUES (?, ?, ?, ?, ?)",
		"RFQ-COMPARE", vendor2, line1, 9.5, 21)
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days) VALUES (?, ?, ?, ?, ?)",
		"RFQ-COMPARE", vendor1, line2, 15.0, 14)

	req := httptest.NewRequest("GET", "/api/rfq/RFQ-COMPARE/compare", nil)
	w := httptest.NewRecorder()

	handleCompareRFQ(w, req, "RFQ-COMPARE")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	result := resp.Data.(map[string]interface{})

	// Verify structure
	lines := result["lines"].([]interface{})
	vendors := result["vendors"].([]interface{})
	matrix := result["matrix"].(map[string]interface{})

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
	if len(vendors) != 2 {
		t.Errorf("Expected 2 vendors, got %d", len(vendors))
	}
	if len(matrix) == 0 {
		t.Errorf("Expected matrix to have entries")
	}
}

// Test RFQ workflow state transitions
func TestRFQWorkflow_StateTransitions(t *testing.T) {
	oldDB := db
	db = setupRFQTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestVendorRFQ(t, db, "V-001", "Test Vendor")
	
	// Create RFQ (draft)
	rfq := map[string]interface{}{
		"title": "Workflow Test",
		"lines": []map[string]interface{}{
			{"ipn": "IPN-001", "qty": 10},
		},
		"vendors": []map[string]interface{}{
			{"vendor_id": "V-001"},
		},
	}
	
	body, _ := json.Marshal(rfq)
	req := httptest.NewRequest("POST", "/api/rfq", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)
	
	var createResp APIResponse
	json.NewDecoder(w.Body).Decode(&createResp)
	rfqData := createResp.Data.(map[string]interface{})
	rfqID := rfqData["id"].(string)
	
	// Verify initial status is draft
	var status string
	db.QueryRow("SELECT status FROM rfqs WHERE id=?", rfqID).Scan(&status)
	if status != "draft" {
		t.Fatalf("Expected initial status 'draft', got %s", status)
	}
	
	// Send RFQ (draft -> sent)
	req = httptest.NewRequest("POST", "/api/rfq/"+rfqID+"/send", nil)
	w = httptest.NewRecorder()
	handleSendRFQ(w, req, rfqID)
	
	if w.Code != 200 {
		t.Errorf("Failed to send RFQ: %d", w.Code)
	}
	
	db.QueryRow("SELECT status FROM rfqs WHERE id=?", rfqID).Scan(&status)
	if status != "sent" {
		t.Errorf("Expected status 'sent' after sending, got %s", status)
	}
	
	// Award RFQ (sent -> awarded)
	vendorID := int(rfqData["vendors"].([]interface{})[0].(map[string]interface{})["id"].(float64))
	lineID := int(rfqData["lines"].([]interface{})[0].(map[string]interface{})["id"].(float64))
	
	db.Exec("INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price) VALUES (?, ?, ?, ?)",
		rfqID, vendorID, lineID, 10.0)
	
	award := map[string]string{"vendor_id": "V-001"}
	body, _ = json.Marshal(award)
	req = httptest.NewRequest("POST", "/api/rfq/"+rfqID+"/award", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handleAwardRFQ(w, req, rfqID)
	
	if w.Code != 200 {
		t.Errorf("Failed to award RFQ: %d", w.Code)
	}
	
	db.QueryRow("SELECT status FROM rfqs WHERE id=?", rfqID).Scan(&status)
	if status != "awarded" {
		t.Errorf("Expected status 'awarded' after awarding, got %s", status)
	}
	
	// Close RFQ (awarded -> closed)
	req = httptest.NewRequest("POST", "/api/rfq/"+rfqID+"/close", nil)
	w = httptest.NewRecorder()
	handleCloseRFQ(w, req, rfqID)
	
	if w.Code != 200 {
		t.Errorf("Failed to close RFQ: %d", w.Code)
	}
	
	db.QueryRow("SELECT status FROM rfqs WHERE id=?", rfqID).Scan(&status)
	if status != "closed" {
		t.Errorf("Expected status 'closed' after closing, got %s", status)
	}
}
