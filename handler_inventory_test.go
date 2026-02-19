package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupInventoryTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create inventory table
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0 CHECK(qty_on_hand >= 0),
			qty_reserved REAL DEFAULT 0 CHECK(qty_reserved >= 0),
			location TEXT,
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
		CREATE TABLE inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			type TEXT NOT NULL,
			qty REAL NOT NULL,
			reference TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory_transactions table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user TEXT,
			action TEXT,
			entity_type TEXT,
			entity_id TEXT,
			details TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	return testDB
}

func TestHandleListInventory_Empty(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/inventory", nil)
	w := httptest.NewRecorder()

	handleListInventory(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []InventoryItem
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty list, got %d items", len(result))
	}
}

func TestHandleListInventory_WithData(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert test inventory items
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved, location, description) VALUES 
		('IPN-001', 100, 10, 'A1', 'Test Part 1'),
		('IPN-002', 50, 5, 'B2', 'Test Part 2')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/inventory", nil)
	w := httptest.NewRecorder()

	handleListInventory(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []InventoryItem
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result[0].IPN != "IPN-001" {
		t.Errorf("Expected first IPN to be IPN-001, got %s", result[0].IPN)
	}
	if result[0].QtyOnHand != 100 {
		t.Errorf("Expected qty_on_hand 100, got %f", result[0].QtyOnHand)
	}
}

func TestHandleListInventory_LowStock(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES 
		('IPN-001', 5, 10),
		('IPN-002', 50, 10),
		('IPN-003', 8, 15)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/inventory?low_stock=true", nil)
	w := httptest.NewRecorder()

	handleListInventory(w, req)

	var result []InventoryItem
	json.NewDecoder(w.Body).Decode(&result)

	if len(result) != 2 {
		t.Errorf("Expected 2 low stock items, got %d", len(result))
	}

	for _, item := range result {
		if item.QtyOnHand > item.ReorderPoint {
			t.Errorf("Expected low stock items only, got %s with qty %f > reorder %f",
				item.IPN, item.QtyOnHand, item.ReorderPoint)
		}
	}
}

func TestHandleGetInventory_Success(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, location, description) VALUES 
		('IPN-001', 100, 'A1', 'Test Part')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/inventory/IPN-001", nil)
	w := httptest.NewRecorder()

	handleGetInventory(w, req, "IPN-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result InventoryItem
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.IPN != "IPN-001" {
		t.Errorf("Expected IPN IPN-001, got %s", result.IPN)
	}
	if result.QtyOnHand != 100 {
		t.Errorf("Expected qty_on_hand 100, got %f", result.QtyOnHand)
	}
}

func TestHandleGetInventory_NotFound(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/inventory/IPN-999", nil)
	w := httptest.NewRecorder()

	handleGetInventory(w, req, "IPN-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleInventoryTransact_Receive(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert initial inventory
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('IPN-001', 100)`)

	reqBody := `{
		"ipn": "IPN-001",
		"type": "receive",
		"qty": 50,
		"reference": "PO-123",
		"notes": "Received shipment"
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify quantity was updated
	var qty float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-001").Scan(&qty)

	if qty != 150 {
		t.Errorf("Expected qty_on_hand 150, got %f", qty)
	}

	// Verify transaction was recorded
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn=?", "IPN-001").Scan(&txCount)
	if txCount != 1 {
		t.Errorf("Expected 1 transaction, got %d", txCount)
	}
}

func TestHandleInventoryTransact_Issue(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('IPN-001', 100)`)

	reqBody := `{
		"ipn": "IPN-001",
		"type": "issue",
		"qty": 30,
		"reference": "WO-456"
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify quantity was decreased
	var qty float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-001").Scan(&qty)

	if qty != 70 {
		t.Errorf("Expected qty_on_hand 70, got %f", qty)
	}
}

func TestHandleInventoryTransact_Adjust(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('IPN-001', 100)`)

	reqBody := `{
		"ipn": "IPN-001",
		"type": "adjust",
		"qty": 85,
		"notes": "Inventory count adjustment"
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify quantity was set to exact value
	var qty float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-001").Scan(&qty)

	if qty != 85 {
		t.Errorf("Expected qty_on_hand 85, got %f", qty)
	}
}

func TestHandleInventoryTransact_NewItem(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Don't insert anything - test auto-creation
	reqBody := `{
		"ipn": "IPN-NEW",
		"type": "receive",
		"qty": 25,
		"reference": "PO-789"
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify inventory record was created
	var qty float64
	err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-NEW").Scan(&qty)
	if err != nil {
		t.Fatalf("Expected inventory record to be created: %v", err)
	}

	if qty != 25 {
		t.Errorf("Expected qty_on_hand 25, got %f", qty)
	}
}

func TestHandleInventoryTransact_MissingIPN(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"type": "receive",
		"qty": 50
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleInventoryTransact_MissingType(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"ipn": "IPN-001",
		"qty": 50
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleInventoryTransact_InvalidType(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"ipn": "IPN-001",
		"type": "invalid_type",
		"qty": 50
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleInventoryTransact_NegativeQty(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"ipn": "IPN-001",
		"type": "receive",
		"qty": -10
	}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInventoryTransact(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleInventoryHistory_Empty(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO inventory (ipn) VALUES ('IPN-001')`)

	req := httptest.NewRequest("GET", "/api/v1/inventory/IPN-001/history", nil)
	w := httptest.NewRecorder()

	handleInventoryHistory(w, req, "IPN-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []InventoryTransaction
	json.NewDecoder(w.Body).Decode(&result)

	if len(result) != 0 {
		t.Errorf("Expected empty history, got %d items", len(result))
	}
}

func TestHandleInventoryHistory_WithData(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO inventory (ipn) VALUES ('IPN-001')`)
	db.Exec(`INSERT INTO inventory_transactions (ipn, type, qty, reference) VALUES 
		('IPN-001', 'receive', 100, 'PO-123'),
		('IPN-001', 'issue', 20, 'WO-456'),
		('IPN-001', 'receive', 50, 'PO-789')
	`)

	req := httptest.NewRequest("GET", "/api/v1/inventory/IPN-001/history", nil)
	w := httptest.NewRecorder()

	handleInventoryHistory(w, req, "IPN-001")

	var result []InventoryTransaction
	json.NewDecoder(w.Body).Decode(&result)

	if len(result) != 3 {
		t.Errorf("Expected 3 transactions, got %d", len(result))
	}

	// Should be ordered by created_at DESC
	if result[0].Reference != "PO-789" {
		t.Errorf("Expected most recent transaction first, got %s", result[0].Reference)
	}
}

func TestHandleBulkDeleteInventory_Success(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO inventory (ipn) VALUES ('IPN-001'), ('IPN-002'), ('IPN-003')`)

	reqBody := `{
		"ipns": ["IPN-001", "IPN-003"]
	}`
	req := httptest.NewRequest("DELETE", "/api/v1/inventory/bulk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleBulkDeleteInventory(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]int
	json.NewDecoder(w.Body).Decode(&result)

	if result["deleted"] != 2 {
		t.Errorf("Expected 2 deleted, got %d", result["deleted"])
	}

	// Verify items were deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM inventory").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 remaining item, got %d", count)
	}
}

func TestHandleBulkDeleteInventory_EmptyList(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{"ipns": []}`
	req := httptest.NewRequest("DELETE", "/api/v1/inventory/bulk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleBulkDeleteInventory(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleBulkDeleteInventory_NonexistentIPNs(t *testing.T) {
	oldDB := db
	db = setupInventoryTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO inventory (ipn) VALUES ('IPN-001')`)

	reqBody := `{
		"ipns": ["IPN-999", "IPN-888"]
	}`
	req := httptest.NewRequest("DELETE", "/api/v1/inventory/bulk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleBulkDeleteInventory(w, req)

	var result map[string]int
	json.NewDecoder(w.Body).Decode(&result)

	if result["deleted"] != 0 {
		t.Errorf("Expected 0 deleted, got %d", result["deleted"])
	}
}
