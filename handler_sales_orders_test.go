package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupSalesOrdersTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create sales_orders table
	_, err = testDB.Exec(`
		CREATE TABLE sales_orders (
			id TEXT PRIMARY KEY,
			quote_id TEXT,
			customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','confirmed','allocated','picked','shipped','invoiced','closed')),
			notes TEXT,
			created_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sales_orders table: %v", err)
	}

	// Create sales_order_lines table
	_, err = testDB.Exec(`
		CREATE TABLE sales_order_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sales_order_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty INTEGER NOT NULL CHECK(qty > 0),
			qty_allocated INTEGER DEFAULT 0,
			qty_picked INTEGER DEFAULT 0,
			qty_shipped INTEGER DEFAULT 0,
			unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (sales_order_id) REFERENCES sales_orders(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sales_order_lines table: %v", err)
	}

	// Create quotes table
	_, err = testDB.Exec(`
		CREATE TABLE quotes (
			id TEXT PRIMARY KEY,
			customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			valid_until TEXT,
			accepted_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create quotes table: %v", err)
	}

	// Create quote_lines table
	_, err = testDB.Exec(`
		CREATE TABLE quote_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			quote_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty INTEGER NOT NULL,
			unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create quote_lines table: %v", err)
	}

	// Create inventory table
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT UNIQUE NOT NULL,
			description TEXT,
			qty_on_hand REAL DEFAULT 0,
			qty_reserved REAL DEFAULT 0,
			min_qty REAL DEFAULT 0,
			location TEXT,
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
			type TEXT CHECK(type IN ('receive','issue','adjust','transfer','return')),
			qty REAL NOT NULL,
			reference TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory_transactions table: %v", err)
	}

	// Create shipments table
	_, err = testDB.Exec(`
		CREATE TABLE shipments (
			id TEXT PRIMARY KEY,
			type TEXT DEFAULT 'outbound',
			status TEXT DEFAULT 'draft',
			to_address TEXT,
			notes TEXT,
			created_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create shipments table: %v", err)
	}

	// Create shipment_lines table
	_, err = testDB.Exec(`
		CREATE TABLE shipment_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			shipment_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			qty INTEGER NOT NULL,
			sales_order_id TEXT,
			FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create shipment_lines table: %v", err)
	}

	// Create invoices table
	_, err = testDB.Exec(`
		CREATE TABLE invoices (
			id TEXT PRIMARY KEY,
			invoice_number TEXT UNIQUE NOT NULL,
			sales_order_id TEXT,
			customer TEXT NOT NULL,
			issue_date TEXT NOT NULL,
			due_date TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			total REAL DEFAULT 0,
			tax REAL DEFAULT 0,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			paid_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create invoices table: %v", err)
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

	return testDB
}

func insertTestSalesOrder(t *testing.T, db *sql.DB, id, customer, status string) {
	_, err := db.Exec(
		"INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))",
		id, customer, status, "testuser",
	)
	if err != nil {
		t.Fatalf("Failed to insert test sales order: %v", err)
	}
}

func insertTestSalesOrderLine(t *testing.T, db *sql.DB, salesOrderID, ipn, description string, qty int, unitPrice float64) {
	_, err := db.Exec(
		"INSERT INTO sales_order_lines (sales_order_id, ipn, description, qty, unit_price) VALUES (?, ?, ?, ?, ?)",
		salesOrderID, ipn, description, qty, unitPrice,
	)
	if err != nil {
		t.Fatalf("Failed to insert test sales order line: %v", err)
	}
}

func insertTestQuoteForSO(t *testing.T, db *sql.DB, id, customer, status string) {
	_, err := db.Exec(
		"INSERT INTO quotes (id, customer, status, created_at) VALUES (?, ?, ?, datetime('now'))",
		id, customer, status,
	)
	if err != nil {
		t.Fatalf("Failed to insert test quote: %v", err)
	}
}

func insertTestQuoteLineForSO(t *testing.T, db *sql.DB, quoteID, ipn, description string, qty int, unitPrice float64) {
	_, err := db.Exec(
		"INSERT INTO quote_lines (quote_id, ipn, description, qty, unit_price) VALUES (?, ?, ?, ?, ?)",
		quoteID, ipn, description, qty, unitPrice,
	)
	if err != nil {
		t.Fatalf("Failed to insert test quote line: %v", err)
	}
}

func insertTestInventorySO(t *testing.T, db *sql.DB, ipn string, qtyOnHand, qtyReserved float64) {
	_, err := db.Exec(
		"INSERT INTO inventory (ipn, qty_on_hand, qty_reserved, updated_at) VALUES (?, ?, ?, datetime('now'))",
		ipn, qtyOnHand, qtyReserved,
	)
	if err != nil {
		t.Fatalf("Failed to insert test inventory: %v", err)
	}
}

// Test List Sales Orders - Empty
func TestHandleListSalesOrders_Empty(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/sales_orders", nil)
	w := httptest.NewRecorder()

	handleListSalesOrders(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	orders, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(orders) != 0 {
		t.Errorf("Expected empty array, got %d sales orders", len(orders))
	}
}

// Test List Sales Orders - With Data
func TestHandleListSalesOrders_WithData(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "draft")
	insertTestSalesOrder(t, db, "SO-0002", "Beta Inc", "confirmed")
	insertTestSalesOrder(t, db, "SO-0003", "Gamma LLC", "shipped")

	req := httptest.NewRequest("GET", "/api/sales_orders", nil)
	w := httptest.NewRecorder()

	handleListSalesOrders(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	orders, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(orders) != 3 {
		t.Errorf("Expected 3 sales orders, got %d", len(orders))
	}
}

// Test List Sales Orders - Filter by Status
func TestHandleListSalesOrders_FilterByStatus(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "draft")
	insertTestSalesOrder(t, db, "SO-0002", "Beta Inc", "confirmed")
	insertTestSalesOrder(t, db, "SO-0003", "Gamma LLC", "confirmed")

	req := httptest.NewRequest("GET", "/api/sales_orders?status=confirmed", nil)
	w := httptest.NewRecorder()

	handleListSalesOrders(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	orders := resp.Data.([]interface{})

	if len(orders) != 2 {
		t.Errorf("Expected 2 confirmed orders, got %d", len(orders))
	}
}

// Test Get Sales Order - Success
func TestHandleGetSalesOrder_Success(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "draft")
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-001", "Widget", 10, 100.0)

	req := httptest.NewRequest("GET", "/api/sales_orders/SO-0001", nil)
	w := httptest.NewRecorder()

	handleGetSalesOrder(w, req, "SO-0001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	order := resp.Data.(map[string]interface{})

	if order["id"] != "SO-0001" {
		t.Errorf("Expected ID SO-0001, got %v", order["id"])
	}
	if order["customer"] != "Acme Corp" {
		t.Errorf("Expected customer Acme Corp, got %v", order["customer"])
	}

	lines := order["lines"].([]interface{})
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}
}

// Test Get Sales Order - Not Found
func TestHandleGetSalesOrder_NotFound(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/sales_orders/SO-9999", nil)
	w := httptest.NewRecorder()

	handleGetSalesOrder(w, req, "SO-9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test Create Sales Order - Valid
func TestHandleCreateSalesOrder_Valid(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	order := SalesOrder{
		Customer: "Acme Corp",
		Lines: []SalesOrderLine{
			{IPN: "PART-001", Description: "Widget", Qty: 10, UnitPrice: 100.0},
			{IPN: "PART-002", Description: "Gadget", Qty: 5, UnitPrice: 50.0},
		},
	}

	body, _ := json.Marshal(order)
	req := httptest.NewRequest("POST", "/api/sales_orders", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateSalesOrder(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	created := resp.Data.(map[string]interface{})

	if created["customer"] != "Acme Corp" {
		t.Errorf("Expected customer Acme Corp, got %v", created["customer"])
	}
	if created["status"] != "draft" {
		t.Errorf("Expected default status 'draft', got %v", created["status"])
	}
	if created["id"] == nil {
		t.Error("Expected ID to be generated")
	}
}

// Test Create Sales Order - Missing Customer
func TestHandleCreateSalesOrder_MissingCustomer(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	order := SalesOrder{
		Lines: []SalesOrderLine{
			{IPN: "PART-001", Qty: 10, UnitPrice: 100.0},
		},
	}

	body, _ := json.Marshal(order)
	req := httptest.NewRequest("POST", "/api/sales_orders", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateSalesOrder(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test Create Sales Order - Invalid Line Qty
func TestHandleCreateSalesOrder_InvalidLineQty(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	order := SalesOrder{
		Customer: "Acme Corp",
		Lines: []SalesOrderLine{
			{IPN: "PART-001", Qty: 0, UnitPrice: 100.0}, // Invalid qty
		},
	}

	body, _ := json.Marshal(order)
	req := httptest.NewRequest("POST", "/api/sales_orders", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateSalesOrder(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test Update Sales Order
func TestHandleUpdateSalesOrder_Success(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "draft")

	order := SalesOrder{
		Customer: "Updated Corp",
		Status:   "confirmed",
		Notes:    "Updated notes",
	}

	body, _ := json.Marshal(order)
	req := httptest.NewRequest("PUT", "/api/sales_orders/SO-0001", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleUpdateSalesOrder(w, req, "SO-0001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify update
	var customer, status string
	db.QueryRow("SELECT customer, status FROM sales_orders WHERE id = ?", "SO-0001").Scan(&customer, &status)
	if customer != "Updated Corp" {
		t.Errorf("Expected customer 'Updated Corp', got %s", customer)
	}
	if status != "confirmed" {
		t.Errorf("Expected status 'confirmed', got %s", status)
	}
}

// Test Convert Quote to Sales Order - Success
func TestHandleConvertQuoteToOrder_Success(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuoteForSO(t, db, "Q-001", "Acme Corp", "accepted")
	insertTestQuoteLineForSO(t, db, "Q-001", "PART-001", "Widget", 10, 100.0)
	insertTestQuoteLineForSO(t, db, "Q-001", "PART-002", "Gadget", 5, 50.0)

	req := httptest.NewRequest("POST", "/api/quotes/Q-001/convert", nil)
	w := httptest.NewRecorder()

	handleConvertQuoteToOrder(w, req, "Q-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	order := resp.Data.(map[string]interface{})

	// Check order was created from quote
	if order["quote_id"] != "Q-001" {
		t.Errorf("Expected quote_id Q-001, got %v", order["quote_id"])
	}
	if order["customer"] != "Acme Corp" {
		t.Errorf("Expected customer Acme Corp, got %v", order["customer"])
	}

	lines := order["lines"].([]interface{})
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

// Test Convert Quote to Sales Order - Not Accepted
func TestHandleConvertQuoteToOrder_NotAccepted(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuoteForSO(t, db, "Q-001", "Acme Corp", "draft") // Not accepted

	req := httptest.NewRequest("POST", "/api/quotes/Q-001/convert", nil)
	w := httptest.NewRecorder()

	handleConvertQuoteToOrder(w, req, "Q-001")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("must be in 'accepted' status")) {
		t.Error("Expected error about quote not being accepted")
	}
}

// Test Convert Quote to Sales Order - Already Converted
func TestHandleConvertQuoteToOrder_AlreadyConverted(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestQuoteForSO(t, db, "Q-001", "Acme Corp", "accepted")
	db.Exec("INSERT INTO sales_orders (id, quote_id, customer, status, created_at, updated_at) VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))",
		"SO-0001", "Q-001", "Acme Corp", "draft")

	req := httptest.NewRequest("POST", "/api/quotes/Q-001/convert", nil)
	w := httptest.NewRecorder()

	handleConvertQuoteToOrder(w, req, "Q-001")

	if w.Code != 409 {
		t.Errorf("Expected status 409 (conflict), got %d", w.Code)
	}
}

// Test Confirm Sales Order
func TestHandleConfirmSalesOrder_Success(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "draft")

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/confirm", nil)
	w := httptest.NewRecorder()

	handleConfirmSalesOrder(w, req, "SO-0001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status was updated
	var status string
	db.QueryRow("SELECT status FROM sales_orders WHERE id = ?", "SO-0001").Scan(&status)
	if status != "confirmed" {
		t.Errorf("Expected status 'confirmed', got %s", status)
	}
}

// Test Allocate Sales Order - Success
func TestHandleAllocateSalesOrder_Success(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "confirmed")
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-001", "Widget", 10, 100.0)
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-002", "Gadget", 5, 50.0)
	insertTestInventorySO(t, db, "PART-001", 100, 0)
	insertTestInventorySO(t, db, "PART-002", 50, 0)

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/allocate", nil)
	w := httptest.NewRecorder()

	handleAllocateSalesOrder(w, req, "SO-0001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status was updated
	var status string
	db.QueryRow("SELECT status FROM sales_orders WHERE id = ?", "SO-0001").Scan(&status)
	if status != "allocated" {
		t.Errorf("Expected status 'allocated', got %s", status)
	}

	// Verify inventory was reserved
	var reserved1, reserved2 float64
	db.QueryRow("SELECT qty_reserved FROM inventory WHERE ipn = ?", "PART-001").Scan(&reserved1)
	db.QueryRow("SELECT qty_reserved FROM inventory WHERE ipn = ?", "PART-002").Scan(&reserved2)

	if reserved1 != 10 {
		t.Errorf("Expected 10 reserved for PART-001, got %.0f", reserved1)
	}
	if reserved2 != 5 {
		t.Errorf("Expected 5 reserved for PART-002, got %.0f", reserved2)
	}
}

// Test Allocate Sales Order - Insufficient Inventory
func TestHandleAllocateSalesOrder_InsufficientInventory(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "confirmed")
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-001", "Widget", 10, 100.0)
	insertTestInventorySO(t, db, "PART-001", 5, 0) // Only 5 available, need 10

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/allocate", nil)
	w := httptest.NewRecorder()

	handleAllocateSalesOrder(w, req, "SO-0001")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("insufficient inventory")) {
		t.Error("Expected error about insufficient inventory")
	}
}

// Test Pick Sales Order
func TestHandlePickSalesOrder_Success(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "allocated")
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-001", "Widget", 10, 100.0)

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/pick", nil)
	w := httptest.NewRecorder()

	handlePickSalesOrder(w, req, "SO-0001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status was updated
	var status string
	db.QueryRow("SELECT status FROM sales_orders WHERE id = ?", "SO-0001").Scan(&status)
	if status != "picked" {
		t.Errorf("Expected status 'picked', got %s", status)
	}

	// Verify qty_picked was set
	var qtyPicked int
	db.QueryRow("SELECT qty_picked FROM sales_order_lines WHERE sales_order_id = ?", "SO-0001").Scan(&qtyPicked)
	if qtyPicked != 10 {
		t.Errorf("Expected qty_picked 10, got %d", qtyPicked)
	}
}

// Test Ship Sales Order - Success
func TestHandleShipSalesOrder_Success(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "picked")
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-001", "Widget", 10, 100.0)
	insertTestInventorySO(t, db, "PART-001", 100, 10) // On hand with 10 reserved

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/ship", nil)
	w := httptest.NewRecorder()

	handleShipSalesOrder(w, req, "SO-0001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status was updated
	var status string
	db.QueryRow("SELECT status FROM sales_orders WHERE id = ?", "SO-0001").Scan(&status)
	if status != "shipped" {
		t.Errorf("Expected status 'shipped', got %s", status)
	}

	// Verify shipment was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM shipments WHERE type = 'outbound'").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 shipment to be created, got %d", count)
	}

	// Verify inventory was reduced
	var onHand, reserved float64
	db.QueryRow("SELECT qty_on_hand, qty_reserved FROM inventory WHERE ipn = ?", "PART-001").Scan(&onHand, &reserved)
	if onHand != 90 {
		t.Errorf("Expected qty_on_hand 90 (100-10), got %.0f", onHand)
	}
	if reserved != 0 {
		t.Errorf("Expected qty_reserved 0 (released after shipment), got %.0f", reserved)
	}
}

// Test Ship Sales Order - Wrong Status
func TestHandleShipSalesOrder_WrongStatus(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "confirmed") // Not picked yet

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/ship", nil)
	w := httptest.NewRecorder()

	handleShipSalesOrder(w, req, "SO-0001")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("must be in 'picked' status")) {
		t.Error("Expected error about order not being picked")
	}
}

// Test Invoice Sales Order - Success
func TestHandleInvoiceSalesOrder_Success(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "shipped")
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-001", "Widget", 10, 100.0)
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-002", "Gadget", 5, 50.0)

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/invoice", nil)
	w := httptest.NewRecorder()

	handleInvoiceSalesOrder(w, req, "SO-0001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status was updated
	var status string
	db.QueryRow("SELECT status FROM sales_orders WHERE id = ?", "SO-0001").Scan(&status)
	if status != "invoiced" {
		t.Errorf("Expected status 'invoiced', got %s", status)
	}

	// Verify invoice was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM invoices WHERE sales_order_id = ?", "SO-0001").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 invoice to be created, got %d", count)
	}

	// Verify total calculation
	var total float64
	db.QueryRow("SELECT total FROM invoices WHERE sales_order_id = ?", "SO-0001").Scan(&total)
	expectedTotal := (10 * 100.0) + (5 * 50.0) // 1250
	if total != expectedTotal {
		t.Errorf("Expected invoice total %.2f, got %.2f", expectedTotal, total)
	}
}

// Test Invoice Sales Order - Wrong Status
func TestHandleInvoiceSalesOrder_WrongStatus(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "picked") // Not shipped yet

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/invoice", nil)
	w := httptest.NewRecorder()

	handleInvoiceSalesOrder(w, req, "SO-0001")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("must be in 'shipped' status")) {
		t.Error("Expected error about order not being shipped")
	}
}

// Test Order-to-Cash Workflow - Full Flow
func TestSalesOrderWorkflow_FullFlow(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Step 1: Create sales order
	order := SalesOrder{
		Customer: "Acme Corp",
		Lines: []SalesOrderLine{
			{IPN: "PART-001", Description: "Widget", Qty: 10, UnitPrice: 100.0},
		},
	}

	body, _ := json.Marshal(order)
	req := httptest.NewRequest("POST", "/api/sales_orders", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)

	if w.Code != 200 {
		t.Fatalf("Failed to create sales order: %s", w.Body.String())
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	created := resp.Data.(map[string]interface{})
	orderID := created["id"].(string)

	// Step 2: Confirm
	req = httptest.NewRequest("POST", "/api/sales_orders/"+orderID+"/confirm", nil)
	w = httptest.NewRecorder()
	handleConfirmSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("Failed to confirm: %s", w.Body.String())
	}

	// Step 3: Allocate (requires inventory)
	insertTestInventorySO(t, db, "PART-001", 100, 0)
	req = httptest.NewRequest("POST", "/api/sales_orders/"+orderID+"/allocate", nil)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("Failed to allocate: %s", w.Body.String())
	}

	// Step 4: Pick
	req = httptest.NewRequest("POST", "/api/sales_orders/"+orderID+"/pick", nil)
	w = httptest.NewRecorder()
	handlePickSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("Failed to pick: %s", w.Body.String())
	}

	// Step 5: Ship
	req = httptest.NewRequest("POST", "/api/sales_orders/"+orderID+"/ship", nil)
	w = httptest.NewRecorder()
	handleShipSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("Failed to ship: %s", w.Body.String())
	}

	// Step 6: Invoice
	req = httptest.NewRequest("POST", "/api/sales_orders/"+orderID+"/invoice", nil)
	w = httptest.NewRecorder()
	handleInvoiceSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("Failed to invoice: %s", w.Body.String())
	}

	// Verify final state
	var finalStatus string
	db.QueryRow("SELECT status FROM sales_orders WHERE id = ?", orderID).Scan(&finalStatus)
	if finalStatus != "invoiced" {
		t.Errorf("Expected final status 'invoiced', got %s", finalStatus)
	}

	// Verify invoice exists
	var invoiceCount int
	db.QueryRow("SELECT COUNT(*) FROM invoices WHERE sales_order_id = ?", orderID).Scan(&invoiceCount)
	if invoiceCount != 1 {
		t.Errorf("Expected 1 invoice, got %d", invoiceCount)
	}
}

// Test Sales Order Price Calculation
func TestSalesOrderPriceCalculation(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "shipped")
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-001", "Widget", 10, 100.0)
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-002", "Gadget", 5, 50.0)
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-003", "Doohickey", 3, 33.33)

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/invoice", nil)
	w := httptest.NewRecorder()
	handleInvoiceSalesOrder(w, req, "SO-0001")

	if w.Code != 200 {
		t.Fatalf("Failed to invoice: %s", w.Body.String())
	}

	// Verify total calculation
	var total float64
	db.QueryRow("SELECT total FROM invoices WHERE sales_order_id = ?", "SO-0001").Scan(&total)
	
	expectedTotal := (10 * 100.0) + (5 * 50.0) + (3 * 33.33) // 1349.99
	// Allow small floating point differences
	diff := total - expectedTotal
	if diff > 0.01 || diff < -0.01 {
		t.Errorf("Expected total %.2f, got %.2f", expectedTotal, total)
	}
}

// Test Filter Sales Orders by Customer
func TestHandleListSalesOrders_FilterByCustomer(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "draft")
	insertTestSalesOrder(t, db, "SO-0002", "Beta Inc", "confirmed")
	insertTestSalesOrder(t, db, "SO-0003", "Acme Corp", "shipped")

	req := httptest.NewRequest("GET", "/api/sales_orders?customer=Acme", nil)
	w := httptest.NewRecorder()

	handleListSalesOrders(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	orders := resp.Data.([]interface{})

	if len(orders) != 2 {
		t.Errorf("Expected 2 Acme Corp orders, got %d", len(orders))
	}
}

// Test Inventory Transaction Logging
func TestSalesOrderInventoryTransactions(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "picked")
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-001", "Widget", 10, 100.0)
	insertTestInventorySO(t, db, "PART-001", 100, 10)

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/ship", nil)
	w := httptest.NewRecorder()
	handleShipSalesOrder(w, req, "SO-0001")

	if w.Code != 200 {
		t.Fatalf("Failed to ship: %s", w.Body.String())
	}

	// Verify inventory transaction was logged
	var count int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn = ? AND type = 'issue'", "PART-001").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 inventory transaction to be logged, got %d", count)
	}

	// Verify transaction details
	var reference string
	db.QueryRow("SELECT reference FROM inventory_transactions WHERE ipn = ? AND type = 'issue'", "PART-001").Scan(&reference)
	if reference != "SO:SO-0001" {
		t.Errorf("Expected reference 'SO:SO-0001', got %s", reference)
	}
}

// Test Get Sales Order Lines Helper
func TestGetSalesOrderLines(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "draft")
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-001", "Widget", 10, 100.0)
	insertTestSalesOrderLine(t, db, "SO-0001", "PART-002", "Gadget", 5, 50.0)

	lines := getSalesOrderLines("SO-0001")

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	if lines[0].IPN != "PART-001" {
		t.Errorf("Expected first line IPN PART-001, got %s", lines[0].IPN)
	}
	if lines[1].IPN != "PART-002" {
		t.Errorf("Expected second line IPN PART-002, got %s", lines[1].IPN)
	}
}

// Test Transition Sales Order Helper
func TestTransitionSalesOrder(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "draft")

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/confirm", nil)
	w := httptest.NewRecorder()

	transitionSalesOrder(w, req, "SO-0001", "draft", "confirmed")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var status string
	db.QueryRow("SELECT status FROM sales_orders WHERE id = ?", "SO-0001").Scan(&status)
	if status != "confirmed" {
		t.Errorf("Expected status 'confirmed', got %s", status)
	}
}

// Test Transition Sales Order - Wrong Current Status
func TestTransitionSalesOrder_WrongStatus(t *testing.T) {
	oldDB := db
	db = setupSalesOrdersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestSalesOrder(t, db, "SO-0001", "Acme Corp", "shipped") // Already shipped

	req := httptest.NewRequest("POST", "/api/sales_orders/SO-0001/confirm", nil)
	w := httptest.NewRecorder()

	transitionSalesOrder(w, req, "SO-0001", "draft", "confirmed")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
