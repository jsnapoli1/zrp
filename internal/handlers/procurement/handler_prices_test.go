package procurement_test

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

func setupPricesTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	schemas := []string{
		`CREATE TABLE price_history (
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
		`CREATE TABLE vendors (
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
		)`,
		`CREATE TABLE audit_log (
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

	for _, schema := range schemas {
		if _, err := testDB.Exec(schema); err != nil {
			t.Fatalf("Failed to create table: %v\nSchema: %s", err, schema)
		}
	}

	return testDB
}

func insertTestVendorPrices(t *testing.T, db *sql.DB, id, name string) {
	t.Helper()
	_, err := db.Exec(
		"INSERT INTO vendors (id, name, created_at) VALUES (?, ?, datetime('now'))",
		id, name,
	)
	if err != nil {
		t.Fatalf("Failed to insert test vendor: %v", err)
	}
}

func insertTestPrice(t *testing.T, db *sql.DB, ipn, vendorID string, unitPrice float64, currency string, minQty int) int {
	t.Helper()
	res, err := db.Exec(
		"INSERT INTO price_history (ipn, vendor_id, unit_price, currency, min_qty, recorded_at) VALUES (?, ?, ?, ?, ?, datetime('now'))",
		ipn, vendorID, unitPrice, currency, minQty,
	)
	if err != nil {
		t.Fatalf("Failed to insert test price: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// Test ListPrices - Empty
func TestHandleListPrices_Empty(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/prices/IPN-001", nil)
	w := httptest.NewRecorder()

	h.ListPrices(w, req, "IPN-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	prices, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(prices) != 0 {
		t.Errorf("Expected empty array, got %d prices", len(prices))
	}
}

// Test ListPrices - With Data
func TestHandleListPrices_WithData(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	insertTestVendorPrices(t, db, "V-001", "Acme Corp")
	insertTestVendorPrices(t, db, "V-002", "Beta Inc")

	insertTestPrice(t, db, "IPN-001", "V-001", 10.50, "USD", 1)
	insertTestPrice(t, db, "IPN-001", "V-002", 9.75, "USD", 10)
	insertTestPrice(t, db, "IPN-002", "V-001", 25.00, "USD", 1)

	req := httptest.NewRequest("GET", "/api/prices/IPN-001", nil)
	w := httptest.NewRecorder()

	h.ListPrices(w, req, "IPN-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	pricesData, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(pricesData) != 2 {
		t.Errorf("Expected 2 prices for IPN-001, got %d", len(pricesData))
	}

	// Verify vendor names are resolved
	for _, pData := range pricesData {
		price := pData.(map[string]interface{})
		vendorName := price["vendor_name"]
		if vendorName == nil || vendorName == "" {
			t.Errorf("Expected vendor_name to be resolved, got %v", vendorName)
		}
	}
}

// Test CreatePrice - Success with vendor_id
func TestHandleCreatePrice_SuccessWithVendorID(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	insertTestVendorPrices(t, db, "V-001", "Acme Corp")

	vendorID := "V-001"
	price := map[string]interface{}{
		"ipn":        "IPN-TEST",
		"vendor_id":  &vendorID,
		"unit_price": 15.99,
		"currency":   "USD",
		"min_qty":    5,
	}

	body, _ := json.Marshal(price)
	req := httptest.NewRequest("POST", "/api/prices", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreatePrice(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	created := resp.Data.(map[string]interface{})
	if created["ipn"] != "IPN-TEST" {
		t.Errorf("Expected IPN 'IPN-TEST', got %v", created["ipn"])
	}
	if created["unit_price"] != 15.99 {
		t.Errorf("Expected unit_price 15.99, got %v", created["unit_price"])
	}

	// Verify vendor name was resolved and stored
	var vendorName string
	db.QueryRow("SELECT vendor_name FROM price_history WHERE id=?", int(created["id"].(float64))).Scan(&vendorName)
	if vendorName != "Acme Corp" {
		t.Errorf("Expected vendor_name 'Acme Corp', got %s", vendorName)
	}
}

// Test CreatePrice - Success with vendor_name
func TestHandleCreatePrice_SuccessWithVendorName(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	vendorName := "Generic Supplier"
	price := map[string]interface{}{
		"ipn":         "IPN-TEST2",
		"vendor_name": &vendorName,
		"unit_price":  12.50,
		"currency":    "EUR",
		"min_qty":     1,
	}

	body, _ := json.Marshal(price)
	req := httptest.NewRequest("POST", "/api/prices", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreatePrice(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	created := resp.Data.(map[string]interface{})
	if created["ipn"] != "IPN-TEST2" {
		t.Errorf("Expected IPN 'IPN-TEST2', got %v", created["ipn"])
	}

	// Verify vendor_name was stored
	var storedVendorName string
	db.QueryRow("SELECT vendor_name FROM price_history WHERE id=?", int(created["id"].(float64))).Scan(&storedVendorName)
	if storedVendorName != "Generic Supplier" {
		t.Errorf("Expected vendor_name 'Generic Supplier', got %s", storedVendorName)
	}
}

// Test CreatePrice - Default values
func TestHandleCreatePrice_Defaults(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	price := map[string]interface{}{
		"ipn":        "IPN-DEFAULTS",
		"unit_price": 99.99,
	}

	body, _ := json.Marshal(price)
	req := httptest.NewRequest("POST", "/api/prices", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreatePrice(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify defaults were applied
	var currency string
	var minQty int
	db.QueryRow("SELECT currency, min_qty FROM price_history WHERE ipn=?", "IPN-DEFAULTS").Scan(&currency, &minQty)

	if currency != "USD" {
		t.Errorf("Expected default currency 'USD', got %s", currency)
	}
	if minQty != 1 {
		t.Errorf("Expected default min_qty 1, got %d", minQty)
	}
}

// Test CreatePrice - Validation errors
func TestHandleCreatePrice_ValidationErrors(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	tests := []struct {
		name       string
		price      map[string]interface{}
		expectCode int
	}{
		{
			name:       "Missing IPN",
			price:      map[string]interface{}{"unit_price": 10.0},
			expectCode: 400,
		},
		{
			name:       "Missing unit_price",
			price:      map[string]interface{}{"ipn": "IPN-001"},
			expectCode: 400,
		},
		{
			name:       "Zero unit_price",
			price:      map[string]interface{}{"ipn": "IPN-001", "unit_price": 0},
			expectCode: 400,
		},
		{
			name:       "Negative unit_price",
			price:      map[string]interface{}{"ipn": "IPN-001", "unit_price": -5.0},
			expectCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.price)
			req := httptest.NewRequest("POST", "/api/prices", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.CreatePrice(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

// Test DeletePrice - Success
func TestHandleDeletePrice_Success(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	id := insertTestPrice(t, db, "IPN-DELETE", "", 10.0, "USD", 1)

	req := httptest.NewRequest("DELETE", "/api/prices/"+strconv.Itoa(id), nil)
	w := httptest.NewRecorder()

	h.DeletePrice(w, req, strconv.Itoa(id))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify it was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM price_history WHERE id=?", id).Scan(&count)
	if count != 0 {
		t.Errorf("Expected price to be deleted, but it still exists")
	}
}

// Test DeletePrice - Not Found
func TestHandleDeletePrice_NotFound(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	req := httptest.NewRequest("DELETE", "/api/prices/99999", nil)
	w := httptest.NewRecorder()

	h.DeletePrice(w, req, "99999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test PriceTrend - Empty
func TestHandlePriceTrend_Empty(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/prices/IPN-001/trend", nil)
	w := httptest.NewRecorder()

	h.PriceTrend(w, req, "IPN-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	points, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(points) != 0 {
		t.Errorf("Expected empty array, got %d points", len(points))
	}
}

// Test PriceTrend - With Data
func TestHandlePriceTrend_WithData(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	insertTestVendorPrices(t, db, "V-001", "Vendor A")
	insertTestVendorPrices(t, db, "V-002", "Vendor B")

	// Insert prices with different dates
	db.Exec("INSERT INTO price_history (ipn, vendor_id, unit_price, recorded_at) VALUES (?, ?, ?, datetime('2026-01-01'))", "IPN-TREND", "V-001", 10.0)
	db.Exec("INSERT INTO price_history (ipn, vendor_id, unit_price, recorded_at) VALUES (?, ?, ?, datetime('2026-02-01'))", "IPN-TREND", "V-002", 9.5)
	db.Exec("INSERT INTO price_history (ipn, vendor_id, unit_price, recorded_at) VALUES (?, ?, ?, datetime('2026-03-01'))", "IPN-TREND", "V-001", 9.0)

	req := httptest.NewRequest("GET", "/api/prices/IPN-TREND/trend", nil)
	w := httptest.NewRecorder()

	h.PriceTrend(w, req, "IPN-TREND")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	points, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(points) != 3 {
		t.Errorf("Expected 3 price points, got %d", len(points))
	}

	// Verify order is ascending by date
	firstPoint := points[0].(map[string]interface{})
	if firstPoint["date"] != "2026-01-01" {
		t.Errorf("Expected first date to be 2026-01-01, got %v", firstPoint["date"])
	}

	// Verify vendor names are included
	for _, pData := range points {
		point := pData.(map[string]interface{})
		if point["vendor"] == nil {
			t.Errorf("Expected vendor to be included in trend point")
		}
	}
}

// Test PriceTrend - Multiple currencies
func TestHandlePriceTrend_MultipleCurrencies(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	insertTestVendorPrices(t, db, "V-001", "US Vendor")
	insertTestVendorPrices(t, db, "V-002", "EU Vendor")

	db.Exec("INSERT INTO price_history (ipn, vendor_id, unit_price, currency, recorded_at) VALUES (?, ?, ?, ?, datetime('2026-01-01'))", "IPN-MULTI", "V-001", 10.0, "USD")
	db.Exec("INSERT INTO price_history (ipn, vendor_id, unit_price, currency, recorded_at) VALUES (?, ?, ?, ?, datetime('2026-02-01'))", "IPN-MULTI", "V-002", 8.5, "EUR")

	req := httptest.NewRequest("GET", "/api/prices/IPN-MULTI/trend", nil)
	w := httptest.NewRecorder()

	h.PriceTrend(w, req, "IPN-MULTI")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	points, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	// Should include both prices even with different currencies
	if len(points) != 2 {
		t.Errorf("Expected 2 price points, got %d", len(points))
	}
}

// Test price history accuracy - chronological ordering
func TestPriceHistory_ChronologicalOrdering(t *testing.T) {
	db := setupPricesTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	insertTestVendorPrices(t, db, "V-001", "Vendor")

	// Insert prices with specific timestamps
	db.Exec("INSERT INTO price_history (ipn, vendor_id, unit_price, recorded_at) VALUES (?, ?, ?, datetime('2026-01-15 10:00:00'))", "IPN-ORDER", "V-001", 15.0)
	db.Exec("INSERT INTO price_history (ipn, vendor_id, unit_price, recorded_at) VALUES (?, ?, ?, datetime('2026-01-10 10:00:00'))", "IPN-ORDER", "V-001", 12.0)
	db.Exec("INSERT INTO price_history (ipn, vendor_id, unit_price, recorded_at) VALUES (?, ?, ?, datetime('2026-01-20 10:00:00'))", "IPN-ORDER", "V-001", 18.0)

	req := httptest.NewRequest("GET", "/api/prices/IPN-ORDER", nil)
	w := httptest.NewRecorder()

	h.ListPrices(w, req, "IPN-ORDER")

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	prices := resp.Data.([]interface{})

	// Should be in descending order (most recent first)
	if len(prices) != 3 {
		t.Fatalf("Expected 3 prices, got %d", len(prices))
	}

	firstPrice := prices[0].(map[string]interface{})["unit_price"].(float64)
	lastPrice := prices[2].(map[string]interface{})["unit_price"].(float64)

	if firstPrice != 18.0 {
		t.Errorf("Expected most recent price (18.0) first, got %.2f", firstPrice)
	}
	if lastPrice != 12.0 {
		t.Errorf("Expected oldest price (12.0) last, got %.2f", lastPrice)
	}
}
