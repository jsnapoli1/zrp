package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

// parsePO extracts a PurchaseOrder from APIResponse-wrapped JSON
func parsePO(t *testing.T, body []byte) PurchaseOrder {
	t.Helper()
	var wrap struct {
		Data PurchaseOrder `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("parse PO: %v", err)
	}
	return wrap.Data
}

// parsePOGenerateResponse extracts the po_id from the generate PO response
func parsePOGenerateResponse(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var wrap struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("parse PO generate response: %v", err)
	}
	return wrap.Data
}

func setupProcurementTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create vendors table
	_, err = testDB.Exec(`
		CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'active'
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create vendors table: %v", err)
	}

	// Create purchase_orders table
	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','ordered','received','cancelled')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date TEXT,
			received_at DATETIME,
			created_by TEXT,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id)
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
			qty_ordered REAL NOT NULL CHECK(qty_ordered > 0),
			qty_received REAL DEFAULT 0,
			unit_price REAL DEFAULT 0 CHECK(unit_price >= 0),
			notes TEXT,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create po_lines table: %v", err)
	}

	// Create work_orders table (for PO generation)
	_, err = testDB.Exec(`
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT,
			qty INTEGER,
			status TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Create inventory table
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create audit_log table (match production schema)
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

	// Create part_changes table
	_, err = testDB.Exec(`
		CREATE TABLE part_changes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user TEXT,
			table_name TEXT,
			record_id TEXT,
			operation TEXT,
			old_snapshot TEXT,
			new_snapshot TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create part_changes table: %v", err)
	}

	// Create id_sequences table
	_, err = testDB.Exec(`
		CREATE TABLE id_sequences (
			prefix TEXT PRIMARY KEY,
			next_num INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create id_sequences table: %v", err)
	}

	return testDB
}

func TestHandleListPOs_Empty(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/pos", nil)
	w := httptest.NewRecorder()

	handleListPOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Data []PurchaseOrder `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := response.Data

	if len(result) != 0 {
		t.Errorf("Expected empty list, got %d items", len(result))
	}
}

func TestHandleListPOs_WithData(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	_, err := db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, notes) VALUES 
		('PO-0001', 'VEN-001', 'draft', 'Test PO 1'),
		('PO-0002', 'VEN-001', 'ordered', 'Test PO 2')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/pos", nil)
	w := httptest.NewRecorder()

	handleListPOs(w, req)

	var response struct {
		Data []PurchaseOrder `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := response.Data

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	// Should be ordered by created_at DESC
	if len(result) > 0 && result[0].ID != "PO-0002" {
		t.Errorf("Expected PO-0002 first, got %s", result[0].ID)
	}
}

func TestHandleGetPO_Success(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)
	db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, notes, expected_date) VALUES 
		('PO-0001', 'VEN-001', 'draft', 'Test PO', '2024-12-31')
	`)
	db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES 
		('PO-0001', 'IPN-001', 100, 1.50),
		('PO-0001', 'IPN-002', 50, 2.00)
	`)

	req := httptest.NewRequest("GET", "/api/v1/pos/PO-0001", nil)
	w := httptest.NewRecorder()

	handleGetPO(w, req, "PO-0001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Data PurchaseOrder `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := response.Data

	if result.ID != "PO-0001" {
		t.Errorf("Expected ID PO-0001, got %s", result.ID)
	}
	if len(result.Lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(result.Lines))
	}
	if len(result.Lines) > 0 && result.Lines[0].IPN != "IPN-001" {
		t.Errorf("Expected first line IPN-001, got %s", result.Lines[0].IPN)
	}
}

func TestHandleGetPO_NotFound(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/pos/PO-9999", nil)
	w := httptest.NewRecorder()

	handleGetPO(w, req, "PO-9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreatePO_Success(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	reqBody := `{
		"vendor_id": "VEN-001",
		"status": "draft",
		"notes": "Test PO",
		"expected_date": "2024-12-31",
		"lines": [
			{"ipn": "IPN-001", "qty_ordered": 100, "unit_price": 1.50},
			{"ipn": "IPN-002", "qty_ordered": 50, "unit_price": 2.00}
		]
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreatePO(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parsePO(t, w.Body.Bytes())

	if result.ID == "" {
		t.Error("Expected ID to be generated")
	}
	if result.VendorID != "VEN-001" {
		t.Errorf("Expected vendor_id VEN-001, got %s", result.VendorID)
	}

	// Verify lines were created
	var lineCount int
	db.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id=?", result.ID).Scan(&lineCount)
	if lineCount != 2 {
		t.Errorf("Expected 2 lines, got %d", lineCount)
	}
}

func TestHandleCreatePO_DefaultStatus(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	reqBody := `{
		"vendor_id": "VEN-001",
		"lines": [{"ipn": "IPN-001", "qty_ordered": 10, "unit_price": 1.0}]
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreatePO(w, req)

	result := parsePO(t, w.Body.Bytes())

	if result.Status != "draft" {
		t.Errorf("Expected default status 'draft', got %s", result.Status)
	}
}

func TestHandleCreatePO_InvalidVendor(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"vendor_id": "VEN-999",
		"lines": [{"ipn": "IPN-001", "qty_ordered": 10, "unit_price": 1.0}]
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreatePO(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid vendor, got %d", w.Code)
	}
}

func TestHandleCreatePO_NegativeQty(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	reqBody := `{
		"vendor_id": "VEN-001",
		"lines": [{"ipn": "IPN-001", "qty_ordered": -10, "unit_price": 1.0}]
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreatePO(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for negative qty, got %d", w.Code)
	}
}

func TestHandleCreatePO_NegativePrice(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	reqBody := `{
		"vendor_id": "VEN-001",
		"lines": [{"ipn": "IPN-001", "qty_ordered": 10, "unit_price": -5.0}]
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreatePO(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for negative price, got %d", w.Code)
	}
}

func TestHandleUpdatePO_Success(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Vendor 1'), ('VEN-002', 'Vendor 2')`)
	db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, notes) VALUES 
		('PO-0001', 'VEN-001', 'draft', 'Original notes')
	`)

	reqBody := `{
		"vendor_id": "VEN-002",
		"status": "ordered",
		"notes": "Updated notes",
		"expected_date": "2024-12-31"
	}`
	req := httptest.NewRequest("PUT", "/api/v1/pos/PO-0001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdatePO(w, req, "PO-0001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify updates
	var vendorID, status, notes string
	db.QueryRow("SELECT vendor_id, status, notes FROM purchase_orders WHERE id=?", "PO-0001").
		Scan(&vendorID, &status, &notes)

	if vendorID != "VEN-002" {
		t.Errorf("Expected vendor_id VEN-002, got %s", vendorID)
	}
	if status != "ordered" {
		t.Errorf("Expected status 'ordered', got %s", status)
	}
	if notes != "Updated notes" {
		t.Errorf("Expected notes 'Updated notes', got %s", notes)
	}
}

func TestHandleGeneratePOFromWO_Success(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Setup work order
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status) VALUES 
		('WO-001', 'ASSY-001', 10, 'draft')
	`)

	// Setup inventory with shortages
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES 
		('IPN-001', 5),
		('IPN-002', 8)
	`)

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	reqBody := `{
		"wo_id": "WO-001",
		"vendor_id": "VEN-001"
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos/generate", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleGeneratePOFromWO(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parsePOGenerateResponse(t, w.Body.Bytes())

	if result["po_id"] == nil || result["po_id"] == "" {
		t.Error("Expected po_id in response")
	}

	// Verify PO was created
	poID, ok := result["po_id"].(string)
	if !ok {
		t.Fatalf("po_id is not a string: %v", result["po_id"])
	}
	var vendorID string
	err := db.QueryRow("SELECT vendor_id FROM purchase_orders WHERE id=?", poID).Scan(&vendorID)
	if err != nil {
		t.Fatalf("Expected PO to exist: %v", err)
	}

	if vendorID != "VEN-001" {
		t.Errorf("Expected vendor VEN-001, got %s", vendorID)
	}
}

func TestHandleGeneratePOFromWO_MissingWOID(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{"vendor_id": "VEN-001"}`
	req := httptest.NewRequest("POST", "/api/v1/pos/generate", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleGeneratePOFromWO(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleGeneratePOFromWO_WONotFound(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"wo_id": "WO-999",
		"vendor_id": "VEN-001"
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos/generate", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleGeneratePOFromWO(w, req)

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleGeneratePOFromWO_NoShortages(t *testing.T) {
	oldDB := db
	db = setupProcurementTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Setup work order
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status) VALUES 
		('WO-001', 'ASSY-001', 10, 'draft')
	`)

	// Setup inventory with no shortages (sufficient stock)
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES 
		('IPN-001', 100),
		('IPN-002', 200)
	`)

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'Test Vendor')`)

	reqBody := `{
		"wo_id": "WO-001",
		"vendor_id": "VEN-001"
	}`
	req := httptest.NewRequest("POST", "/api/v1/pos/generate", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleGeneratePOFromWO(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for no shortages, got %d", w.Code)
	}
}
