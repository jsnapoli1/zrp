package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// =============================================================================
// TEST DATABASE SETUP
// =============================================================================

func setupReportsTestDB(t *testing.T) *sql.DB {
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
			description TEXT,
			mpn TEXT,
			qty_on_hand REAL DEFAULT 0,
			reorder_point REAL DEFAULT 0,
			reorder_qty REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create purchase_orders table
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

	// Create ecos table
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','review','approved','rejected','implemented')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			created_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_at DATETIME,
			implemented_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	// Create work_orders table
	_, err = testDB.Exec(`
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			ipn TEXT,
			qty REAL NOT NULL,
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending','ready','in-progress','completed','cancelled','on-hold')),
			priority TEXT DEFAULT 'normal',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Create ncrs table (Non-Conformance Reports)
	_, err = testDB.Exec(`
		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			severity TEXT DEFAULT 'minor' CHECK(severity IN ('minor','major','critical')),
			defect_type TEXT,
			status TEXT DEFAULT 'open' CHECK(status IN ('open','investigating','resolved','closed')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
	}

	return testDB
}

// Helper functions for test data insertion
func insertTestInventoryItem(t *testing.T, db *sql.DB, ipn, description, mpn string, qtyOnHand, reorderPoint, reorderQty float64) {
	_, err := db.Exec(
		"INSERT INTO inventory (ipn, description, mpn, qty_on_hand, reorder_point, reorder_qty) VALUES (?, ?, ?, ?, ?, ?)",
		ipn, description, mpn, qtyOnHand, reorderPoint, reorderQty,
	)
	if err != nil {
		t.Fatalf("Failed to insert inventory item %s: %v", ipn, err)
	}
}

func insertTestPO(t *testing.T, db *sql.DB, poID, vendorID, status, createdAt string) {
	_, err := db.Exec(
		"INSERT INTO purchase_orders (id, vendor_id, status, created_at) VALUES (?, ?, ?, ?)",
		poID, vendorID, status, createdAt,
	)
	if err != nil {
		t.Fatalf("Failed to insert PO %s: %v", poID, err)
	}
}

func insertTestPOLineForReports(t *testing.T, db *sql.DB, poID, ipn string, unitPrice float64) {
	_, err := db.Exec(
		"INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES (?, ?, 100, ?)",
		poID, ipn, unitPrice,
	)
	if err != nil {
		t.Fatalf("Failed to insert PO line for %s: %v", ipn, err)
	}
}

func insertTestECO(t *testing.T, db *sql.DB, id, title, status, priority, createdBy, createdAt string) {
	_, err := db.Exec(
		"INSERT INTO ecos (id, title, status, priority, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, title, status, priority, createdBy, createdAt,
	)
	if err != nil {
		t.Fatalf("Failed to insert ECO %s: %v", id, err)
	}
}

func insertTestWorkOrder(t *testing.T, db *sql.DB, id, status string, startedAt, completedAt *string) {
	query := "INSERT INTO work_orders (id, ipn, qty, status, started_at, completed_at) VALUES (?, 'TEST-IPN', 100, ?, ?, ?)"
	_, err := db.Exec(query, id, status, startedAt, completedAt)
	if err != nil {
		t.Fatalf("Failed to insert work order %s: %v", id, err)
	}
}

func insertTestNCR(t *testing.T, db *sql.DB, id, severity, defectType, status string, resolvedAt *string) {
	_, err := db.Exec(
		"INSERT INTO ncrs (id, title, severity, defect_type, status, resolved_at, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
		id, "NCR "+id, severity, defectType, status, resolvedAt,
	)
	if err != nil {
		t.Fatalf("Failed to insert NCR %s: %v", id, err)
	}
}

// =============================================================================
// INVENTORY VALUATION REPORT TESTS
// =============================================================================

func TestReportInventoryValuation_Empty(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
	w := httptest.NewRecorder()

	handleReportInventoryValuation(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report InvValuationReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(report.Groups) != 0 {
		t.Errorf("Expected 0 groups for empty inventory, got %d", len(report.Groups))
	}

	if report.GrandTotal != 0 {
		t.Errorf("Expected grand total 0 for empty inventory, got %.2f", report.GrandTotal)
	}
}

func TestReportInventoryValuation_WithData(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert test inventory items with different categories (IPN prefix)
	insertTestInventoryItem(t, db, "RES-001", "100 Ohm Resistor", "RC0805FR-07100RL", 5000, 1000, 5000)
	insertTestInventoryItem(t, db, "RES-002", "220 Ohm Resistor", "RC0805FR-07220RL", 3000, 1000, 5000)
	insertTestInventoryItem(t, db, "CAP-001", "10uF Capacitor", "GRM21BR61C106KE15L", 2000, 500, 2000)
	insertTestInventoryItem(t, db, "IC-001", "ATmega328P MCU", "ATMEGA328P-PU", 100, 20, 50)

	// Insert PO pricing data
	insertTestPO(t, db, "PO-001", "VENDOR-001", "received", "2024-01-01 10:00:00")
	insertTestPOLineForReports(t, db, "PO-001", "RES-001", 0.05)  // $0.05 per resistor
	insertTestPOLineForReports(t, db, "PO-001", "RES-002", 0.06)  // $0.06 per resistor
	insertTestPOLineForReports(t, db, "PO-001", "CAP-001", 0.15)  // $0.15 per cap
	insertTestPOLineForReports(t, db, "PO-001", "IC-001", 2.50)   // $2.50 per MCU

	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
	w := httptest.NewRecorder()

	handleReportInventoryValuation(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report InvValuationReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have 3 categories: RES, CAP, IC
	if len(report.Groups) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(report.Groups))
	}

	// Verify calculations
	// RES: (5000 * 0.05) + (3000 * 0.06) = 250 + 180 = 430
	// CAP: (2000 * 0.15) = 300
	// IC: (100 * 2.50) = 250
	// Total: 430 + 300 + 250 = 980

	expectedTotal := 980.0
	if report.GrandTotal != expectedTotal {
		t.Errorf("Expected grand total %.2f, got %.2f", expectedTotal, report.GrandTotal)
	}

	// Find RES group and verify its subtotal
	var resGroup *InvValuationGroup
	for i := range report.Groups {
		if report.Groups[i].Category == "RES" {
			resGroup = &report.Groups[i]
			break
		}
	}

	if resGroup == nil {
		t.Fatal("Expected to find RES category group")
	}

	expectedResTotal := 430.0
	if resGroup.Subtotal != expectedResTotal {
		t.Errorf("Expected RES subtotal %.2f, got %.2f", expectedResTotal, resGroup.Subtotal)
	}

	if len(resGroup.Items) != 2 {
		t.Errorf("Expected 2 RES items, got %d", len(resGroup.Items))
	}

	// Verify individual item calculations
	for _, item := range resGroup.Items {
		if item.IPN == "RES-001" {
			expectedSubtotal := 5000 * 0.05 // 250
			if item.Subtotal != expectedSubtotal {
				t.Errorf("Expected RES-001 subtotal %.2f, got %.2f", expectedSubtotal, item.Subtotal)
			}
			if item.QtyOnHand != 5000 {
				t.Errorf("Expected RES-001 qty 5000, got %.2f", item.QtyOnHand)
			}
			if item.UnitPrice != 0.05 {
				t.Errorf("Expected RES-001 unit price 0.05, got %.4f", item.UnitPrice)
			}
		}
	}
}

func TestReportInventoryValuation_NoPricing(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert inventory without corresponding PO pricing
	insertTestInventoryItem(t, db, "RES-001", "100 Ohm Resistor", "RC0805FR-07100RL", 5000, 1000, 5000)

	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
	w := httptest.NewRecorder()

	handleReportInventoryValuation(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report InvValuationReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should still show item, but with 0 unit price
	if len(report.Groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(report.Groups))
	}

	if len(report.Groups[0].Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(report.Groups[0].Items))
	}

	item := report.Groups[0].Items[0]
	if item.UnitPrice != 0 {
		t.Errorf("Expected unit price 0 (no PO data), got %.4f", item.UnitPrice)
	}

	if item.Subtotal != 0 {
		t.Errorf("Expected subtotal 0, got %.2f", item.Subtotal)
	}

	if report.GrandTotal != 0 {
		t.Errorf("Expected grand total 0, got %.2f", report.GrandTotal)
	}
}

func TestReportInventoryValuation_CSV(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestInventoryItem(t, db, "RES-001", "100 Ohm Resistor", "RC0805FR-07100RL", 100, 50, 100)
	insertTestPO(t, db, "PO-001", "VENDOR-001", "received", "2024-01-01 10:00:00")
	insertTestPOLineForReports(t, db, "PO-001", "RES-001", 0.05)

	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation?format=csv", nil)
	w := httptest.NewRecorder()

	handleReportInventoryValuation(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", ct)
	}

	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "inventory-valuation.csv") {
		t.Errorf("Expected Content-Disposition to contain filename, got '%s'", cd)
	}

	// Parse CSV
	csvData := w.Body.String()
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have header + at least 1 data row
	if len(records) < 2 {
		t.Errorf("Expected at least 2 rows (header + data), got %d", len(records))
	}

	// Verify header
	expectedHeaders := []string{"IPN", "Description", "Category", "Qty On Hand", "Unit Price", "Subtotal", "PO Ref"}
	header := records[0]
	for i, expected := range expectedHeaders {
		if i >= len(header) || header[i] != expected {
			t.Errorf("Expected header[%d] = '%s', got '%s'", i, expected, header[i])
		}
	}

	// Verify data row
	if len(records) > 1 {
		dataRow := records[1]
		if dataRow[0] != "RES-001" {
			t.Errorf("Expected IPN 'RES-001', got '%s'", dataRow[0])
		}
		if dataRow[1] != "100 Ohm Resistor" {
			t.Errorf("Expected description, got '%s'", dataRow[1])
		}
		if dataRow[2] != "RES" {
			t.Errorf("Expected category 'RES', got '%s'", dataRow[2])
		}
		// Qty = 100, Price = 0.05, Subtotal = 5.00
		if dataRow[5] != "5.00" {
			t.Errorf("Expected subtotal '5.00', got '%s'", dataRow[5])
		}
	}
}

func TestReportInventoryValuation_CategoryGrouping(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test category extraction from IPN prefix
	tests := []struct {
		ipn              string
		expectedCategory string
	}{
		{"RES-001", "RES"},
		{"CAP-ABC-123", "CAP"},
		{"IC-ATmega", "IC"},
		{"CONN-USB-TypeC", "CONN"},
		{"NoHyphen", "NoHyphen"}, // Edge case: no hyphen
		{"", "Other"},            // Empty IPN edge case
	}

	for _, tt := range tests {
		if tt.ipn != "" {
			insertTestInventoryItem(t, db, tt.ipn, "Test Item", "MPN", 1, 0, 0)
		}
	}

	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
	w := httptest.NewRecorder()

	handleReportInventoryValuation(w, req)

	var report InvValuationReport
	json.NewDecoder(w.Body).Decode(&report)

	// Verify categories were extracted correctly
	categoryMap := make(map[string]bool)
	for _, group := range report.Groups {
		categoryMap[group.Category] = true
	}

	for _, tt := range tests {
		if tt.ipn != "" && tt.expectedCategory != "Other" {
			if !categoryMap[tt.expectedCategory] {
				t.Errorf("Expected category '%s' to be present for IPN '%s'", tt.expectedCategory, tt.ipn)
			}
		}
	}
}

func TestReportInventoryValuation_LatestPOPrice(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestInventoryItem(t, db, "RES-001", "100 Ohm Resistor", "RC0805FR-07100RL", 100, 50, 100)

	// Insert multiple POs with different prices (should use latest)
	insertTestPO(t, db, "PO-001", "VENDOR-001", "received", "2024-01-01 10:00:00")
	insertTestPOLineForReports(t, db, "PO-001", "RES-001", 0.05)

	insertTestPO(t, db, "PO-002", "VENDOR-001", "received", "2024-02-01 10:00:00")
	insertTestPOLineForReports(t, db, "PO-002", "RES-001", 0.04) // Newer, cheaper price

	insertTestPO(t, db, "PO-003", "VENDOR-001", "received", "2024-03-01 10:00:00")
	insertTestPOLineForReports(t, db, "PO-003", "RES-001", 0.06) // Latest price (most recent)

	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
	w := httptest.NewRecorder()

	handleReportInventoryValuation(w, req)

	var report InvValuationReport
	json.NewDecoder(w.Body).Decode(&report)

	// Should use latest PO price (0.06 from PO-003)
	if len(report.Groups) != 1 || len(report.Groups[0].Items) != 1 {
		t.Fatal("Expected 1 group with 1 item")
	}

	item := report.Groups[0].Items[0]
	if item.UnitPrice != 0.06 {
		t.Errorf("Expected latest unit price 0.06, got %.4f", item.UnitPrice)
	}

	expectedSubtotal := 100 * 0.06 // 6.00
	if item.Subtotal != expectedSubtotal {
		t.Errorf("Expected subtotal %.2f, got %.2f", expectedSubtotal, item.Subtotal)
	}
}

func TestReportInventoryValuation_ZeroQty(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestInventoryItem(t, db, "RES-001", "100 Ohm Resistor", "RC0805FR-07100RL", 0, 50, 100)
	insertTestPO(t, db, "PO-001", "VENDOR-001", "received", "2024-01-01 10:00:00")
	insertTestPOLineForReports(t, db, "PO-001", "RES-001", 0.05)

	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
	w := httptest.NewRecorder()

	handleReportInventoryValuation(w, req)

	var report InvValuationReport
	json.NewDecoder(w.Body).Decode(&report)

	if len(report.Groups) != 1 || len(report.Groups[0].Items) != 1 {
		t.Fatal("Expected 1 group with 1 item")
	}

	item := report.Groups[0].Items[0]
	if item.Subtotal != 0 {
		t.Errorf("Expected subtotal 0 for zero qty, got %.2f", item.Subtotal)
	}

	if report.GrandTotal != 0 {
		t.Errorf("Expected grand total 0, got %.2f", report.GrandTotal)
	}
}

// =============================================================================
// OPEN ECOs REPORT TESTS
// =============================================================================

func TestReportOpenECOs_Empty(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/reports/open-ecos", nil)
	w := httptest.NewRecorder()

	handleReportOpenECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	itemsData, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array, got %T", resp.Data)
	}

	if len(itemsData) != 0 {
		t.Errorf("Expected empty array, got %d items", len(itemsData))
	}
}

func TestReportOpenECOs_WithData(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert ECOs with different statuses and priorities
	insertTestECO(t, db, "ECO-001", "Critical production fix", "draft", "critical", "user1", "2024-01-01 10:00:00")
	insertTestECO(t, db, "ECO-002", "High priority change", "review", "high", "user2", "2024-01-02 10:00:00")
	insertTestECO(t, db, "ECO-003", "Normal change", "draft", "normal", "user1", "2024-01-03 10:00:00")
	insertTestECO(t, db, "ECO-004", "Low priority", "review", "low", "user3", "2024-01-04 10:00:00")
	insertTestECO(t, db, "ECO-005", "Approved ECO", "approved", "high", "user1", "2024-01-05 10:00:00") // Should NOT appear
	insertTestECO(t, db, "ECO-006", "Implemented ECO", "implemented", "critical", "user2", "2024-01-06 10:00:00") // Should NOT appear

	req := httptest.NewRequest("GET", "/api/reports/open-ecos", nil)
	w := httptest.NewRecorder()

	handleReportOpenECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	itemsData, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array, got %T", resp.Data)
	}

	// Convert to typed items
	items := make([]map[string]interface{}, len(itemsData))
	for i, item := range itemsData {
		items[i] = item.(map[string]interface{})
	}

	// Should only show draft/review (4 ECOs)
	if len(items) != 4 {
		t.Errorf("Expected 4 open ECOs, got %d", len(items))
	}

	// Verify sorting by priority (critical, high, normal, low)
	expectedOrder := []string{"ECO-001", "ECO-002", "ECO-003", "ECO-004"}
	for i, expected := range expectedOrder {
		if items[i]["id"] != expected {
			t.Errorf("Expected item[%d] ID '%s', got '%v'", i, expected, items[i]["id"])
		}
	}

	// Verify priority sorting
	expectedPriorities := []string{"critical", "high", "normal", "low"}
	for i, expected := range expectedPriorities {
		if items[i]["priority"] != expected {
			t.Errorf("Expected item[%d] priority '%s', got '%v'", i, expected, items[i]["priority"])
		}
	}

	// Verify age calculation
	for i, item := range items {
		ageDays, ok := item["age_days"].(float64)
		if !ok || ageDays < 0 {
			t.Errorf("Expected non-negative age_days for item %d, got %v", i, item["age_days"])
		}
	}
}

func TestReportOpenECOs_PriorityOrdering(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert ECOs in random order to verify sorting
	insertTestECO(t, db, "ECO-LOW", "Low priority", "draft", "low", "user1", "2024-01-01 10:00:00")
	insertTestECO(t, db, "ECO-CRIT", "Critical", "draft", "critical", "user1", "2024-01-02 10:00:00")
	insertTestECO(t, db, "ECO-NORM", "Normal", "draft", "normal", "user1", "2024-01-03 10:00:00")
	insertTestECO(t, db, "ECO-HIGH", "High", "draft", "high", "user1", "2024-01-04 10:00:00")

	req := httptest.NewRequest("GET", "/api/reports/open-ecos", nil)
	w := httptest.NewRecorder()

	handleReportOpenECOs(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	itemsData, ok := resp.Data.([]interface{})
	if !ok || len(itemsData) == 0 {
		t.Fatalf("Expected array of items, got %T with length %d", resp.Data, len(itemsData))
	}

	items := make([]map[string]interface{}, len(itemsData))
	for i, item := range itemsData {
		items[i] = item.(map[string]interface{})
	}

	// Should be ordered: critical, high, normal, low
	expectedOrder := []string{"ECO-CRIT", "ECO-HIGH", "ECO-NORM", "ECO-LOW"}
	for i, expected := range expectedOrder {
		if items[i]["id"] != expected {
			t.Errorf("Expected item[%d] = '%s', got '%v'", i, expected, items[i]["id"])
		}
	}
}

func TestReportOpenECOs_AgeDaysCalculation(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert ECO from 10 days ago
	tenDaysAgo := "2024-01-01 10:00:00"
	insertTestECO(t, db, "ECO-001", "Old ECO", "draft", "normal", "user1", tenDaysAgo)

	req := httptest.NewRequest("GET", "/api/reports/open-ecos", nil)
	w := httptest.NewRecorder()

	handleReportOpenECOs(w, req)

	var items []OpenECOItem
	json.NewDecoder(w.Body).Decode(&items)

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	// AgeDays should be positive and reasonable (not testing exact value due to time passage)
	if items[0].AgeDays < 0 {
		t.Errorf("Expected positive AgeDays, got %d", items[0].AgeDays)
	}
}

func TestReportOpenECOs_CSV(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestECO(t, db, "ECO-001", "Test ECO", "draft", "critical", "user1", "2024-01-01 10:00:00")

	req := httptest.NewRequest("GET", "/api/reports/open-ecos?format=csv", nil)
	w := httptest.NewRecorder()

	handleReportOpenECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", ct)
	}

	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "open-ecos.csv") {
		t.Errorf("Expected filename in Content-Disposition, got '%s'", cd)
	}

	csvData := w.Body.String()
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) < 2 {
		t.Errorf("Expected at least 2 rows, got %d", len(records))
	}

	// Verify header
	expectedHeaders := []string{"ID", "Title", "Status", "Priority", "Created By", "Created At", "Age (Days)"}
	header := records[0]
	for i, expected := range expectedHeaders {
		if i >= len(header) || header[i] != expected {
			t.Errorf("Expected header[%d] = '%s', got '%s'", i, expected, header[i])
		}
	}
}

func TestReportOpenECOs_OnlyDraftAndReview(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert ECOs with all possible statuses
	statuses := []string{"draft", "review", "approved", "rejected", "implemented"}
	for i, status := range statuses {
		id := fmt.Sprintf("ECO-%03d", i+1)
		insertTestECO(t, db, id, "Test "+status, status, "normal", "user1", "2024-01-01 10:00:00")
	}

	req := httptest.NewRequest("GET", "/api/reports/open-ecos", nil)
	w := httptest.NewRecorder()

	handleReportOpenECOs(w, req)

	var items []OpenECOItem
	json.NewDecoder(w.Body).Decode(&items)

	// Should only show draft and review (2 items)
	if len(items) != 2 {
		t.Errorf("Expected 2 items (draft+review), got %d", len(items))
	}

	for _, item := range items {
		if item.Status != "draft" && item.Status != "review" {
			t.Errorf("Unexpected status '%s' in open ECOs report", item.Status)
		}
	}
}

// =============================================================================
// WO THROUGHPUT REPORT TESTS
// =============================================================================

func TestReportWOThroughput_Empty(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/reports/wo-throughput", nil)
	w := httptest.NewRecorder()

	handleReportWOThroughput(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report WOThroughputReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if report.Days != 30 {
		t.Errorf("Expected default days=30, got %d", report.Days)
	}

	if report.TotalCompleted != 0 {
		t.Errorf("Expected total_completed=0, got %d", report.TotalCompleted)
	}

	if report.AvgCycleTimeDays != 0 {
		t.Errorf("Expected avg_cycle_time=0, got %.2f", report.AvgCycleTimeDays)
	}

	if len(report.CountByStatus) != 0 {
		t.Errorf("Expected empty count_by_status, got %d entries", len(report.CountByStatus))
	}
}

func TestReportWOThroughput_WithData(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert completed work orders within the last 30 days
	recentStart := "2024-02-01 08:00:00"
	recentEnd := "2024-02-03 17:00:00"

	// WO with 2-day cycle time (48 hours = 2 days)
	insertTestWorkOrder(t, db, "WO-001", "completed", &recentStart, &recentEnd)

	// WO with 1-day cycle time
	start2 := "2024-02-05 08:00:00"
	end2 := "2024-02-06 08:00:00"
	insertTestWorkOrder(t, db, "WO-002", "completed", &start2, &end2)

	// WO with 3-day cycle time
	start3 := "2024-02-10 08:00:00"
	end3 := "2024-02-13 08:00:00"
	insertTestWorkOrder(t, db, "WO-003", "completed", &start3, &end3)

	req := httptest.NewRequest("GET", "/api/reports/wo-throughput", nil)
	w := httptest.NewRecorder()

	handleReportWOThroughput(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var report WOThroughputReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if report.TotalCompleted != 3 {
		t.Errorf("Expected total_completed=3, got %d", report.TotalCompleted)
	}

	// Average cycle time: (2 + 1 + 3) / 3 = 2 days
	expectedAvg := 2.0
	if report.AvgCycleTimeDays != expectedAvg {
		t.Errorf("Expected avg_cycle_time=%.2f, got %.2f", expectedAvg, report.AvgCycleTimeDays)
	}

	if report.CountByStatus["completed"] != 3 {
		t.Errorf("Expected count_by_status['completed']=3, got %d", report.CountByStatus["completed"])
	}
}

func TestReportWOThroughput_DateFiltering(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert WOs at different times
	// Old WO (91 days ago) - should NOT be included in default 30-day report
	oldStart := "2023-10-01 08:00:00"
	oldEnd := "2023-10-02 08:00:00"
	insertTestWorkOrder(t, db, "WO-OLD", "completed", &oldStart, &oldEnd)

	// Recent WO (within 30 days)
	recentStart := "2024-02-01 08:00:00"
	recentEnd := "2024-02-02 08:00:00"
	insertTestWorkOrder(t, db, "WO-RECENT", "completed", &recentStart, &recentEnd)

	tests := []struct {
		name          string
		days          string
		expectedCount int
	}{
		{"default 30 days", "", 1},     // Only recent WO
		{"30 days explicit", "30", 1},  // Only recent WO
		{"60 days", "60", 1},           // Only recent WO
		{"90 days", "90", 1},           // Only recent WO (old is 91 days ago)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/reports/wo-throughput"
			if tt.days != "" {
				url += "?days=" + tt.days
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handleReportWOThroughput(w, req)

			var report WOThroughputReport
			json.NewDecoder(w.Body).Decode(&report)

			if report.TotalCompleted != tt.expectedCount {
				t.Errorf("Expected %d completed WOs, got %d", tt.expectedCount, report.TotalCompleted)
			}
		})
	}
}

func TestReportWOThroughput_DaysParameter(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name         string
		daysParam    string
		expectedDays int
	}{
		{"default", "", 30},
		{"30 days", "30", 30},
		{"60 days", "60", 60},
		{"90 days", "90", 90},
		{"invalid number", "abc", 30},      // Should default to 30
		{"invalid value", "45", 30},        // Only 30/60/90 allowed
		{"negative", "-10", 30},            // Should default to 30
		{"zero", "0", 30},                  // Should default to 30
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/reports/wo-throughput"
			if tt.daysParam != "" {
				url += "?days=" + tt.daysParam
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handleReportWOThroughput(w, req)

			var report WOThroughputReport
			json.NewDecoder(w.Body).Decode(&report)

			if report.Days != tt.expectedDays {
				t.Errorf("Expected days=%d, got %d", tt.expectedDays, report.Days)
			}
		})
	}
}

func TestReportWOThroughput_CycleTimeCalculation(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name             string
		startedAt        string
		completedAt      string
		expectedCycleDays float64
	}{
		{"1 day", "2024-02-01 08:00:00", "2024-02-02 08:00:00", 1.0},
		{"2 days", "2024-02-01 08:00:00", "2024-02-03 08:00:00", 2.0},
		{"12 hours", "2024-02-01 08:00:00", "2024-02-01 20:00:00", 0.5},
		{"3.5 days", "2024-02-01 08:00:00", "2024-02-04 20:00:00", 3.5},
	}

	for i, tt := range tests {
		woID := fmt.Sprintf("WO-%03d", i+1)
		insertTestWorkOrder(t, db, woID, "completed", &tt.startedAt, &tt.completedAt)
	}

	req := httptest.NewRequest("GET", "/api/reports/wo-throughput", nil)
	w := httptest.NewRecorder()

	handleReportWOThroughput(w, req)

	var report WOThroughputReport
	json.NewDecoder(w.Body).Decode(&report)

	// Average: (1.0 + 2.0 + 0.5 + 3.5) / 4 = 1.75
	expectedAvg := 1.75
	if report.AvgCycleTimeDays != expectedAvg {
		t.Errorf("Expected avg_cycle_time=%.2f, got %.2f", expectedAvg, report.AvgCycleTimeDays)
	}

	if report.TotalCompleted != 4 {
		t.Errorf("Expected 4 completed WOs, got %d", report.TotalCompleted)
	}
}

func TestReportWOThroughput_MissingTimestamps(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// WO with completed_at but no started_at
	completedAt := "2024-02-01 08:00:00"
	insertTestWorkOrder(t, db, "WO-001", "completed", nil, &completedAt)

	// WO with both timestamps
	start2 := "2024-02-01 08:00:00"
	end2 := "2024-02-02 08:00:00"
	insertTestWorkOrder(t, db, "WO-002", "completed", &start2, &end2)

	req := httptest.NewRequest("GET", "/api/reports/wo-throughput", nil)
	w := httptest.NewRecorder()

	handleReportWOThroughput(w, req)

	var report WOThroughputReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.TotalCompleted != 2 {
		t.Errorf("Expected total_completed=2, got %d", report.TotalCompleted)
	}

	// Only WO-002 should contribute to cycle time average
	expectedAvg := 1.0
	if report.AvgCycleTimeDays != expectedAvg {
		t.Errorf("Expected avg_cycle_time=%.2f (only WO-002), got %.2f", expectedAvg, report.AvgCycleTimeDays)
	}
}

func TestReportWOThroughput_CSV(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	start := "2024-02-01 08:00:00"
	end := "2024-02-02 08:00:00"
	insertTestWorkOrder(t, db, "WO-001", "completed", &start, &end)

	req := httptest.NewRequest("GET", "/api/reports/wo-throughput?format=csv", nil)
	w := httptest.NewRecorder()

	handleReportWOThroughput(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", ct)
	}

	csvData := w.Body.String()
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have summary row + blank + status breakdown header + data
	if len(records) < 4 {
		t.Errorf("Expected at least 4 rows, got %d", len(records))
	}

	// First row: summary header
	if records[0][0] != "Days" {
		t.Errorf("Expected first header 'Days', got '%s'", records[0][0])
	}

	// Verify summary data row
	if records[1][0] != "30" {
		t.Errorf("Expected days=30, got '%s'", records[1][0])
	}
	if records[1][1] != "1" {
		t.Errorf("Expected total_completed=1, got '%s'", records[1][1])
	}
}

func TestReportWOThroughput_MultipleStatuses(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	start := "2024-02-01 08:00:00"
	end := "2024-02-02 08:00:00"

	// Create WOs with different final statuses
	insertTestWorkOrder(t, db, "WO-001", "completed", &start, &end)
	insertTestWorkOrder(t, db, "WO-002", "completed", &start, &end)
	insertTestWorkOrder(t, db, "WO-003", "cancelled", &start, &end) // Different status
	
	// Note: The query filters WHERE completed_at IS NOT NULL, so status doesn't matter
	// All 3 should be counted

	req := httptest.NewRequest("GET", "/api/reports/wo-throughput", nil)
	w := httptest.NewRecorder()

	handleReportWOThroughput(w, req)

	var report WOThroughputReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.TotalCompleted != 3 {
		t.Errorf("Expected total_completed=3, got %d", report.TotalCompleted)
	}

	if report.CountByStatus["completed"] != 2 {
		t.Errorf("Expected 2 completed status, got %d", report.CountByStatus["completed"])
	}

	if report.CountByStatus["cancelled"] != 1 {
		t.Errorf("Expected 1 cancelled status, got %d", report.CountByStatus["cancelled"])
	}
}

// =============================================================================
// LOW STOCK REPORT TESTS
// =============================================================================

func TestReportLowStock_Empty(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/reports/low-stock", nil)
	w := httptest.NewRecorder()

	handleReportLowStock(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var items []LowStockItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected empty array, got %d items", len(items))
	}
}

func TestReportLowStock_WithData(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Items below reorder point
	insertTestInventoryItem(t, db, "RES-001", "100 Ohm Resistor", "MPN1", 50, 100, 500)   // Below reorder (diff: 50)
	insertTestInventoryItem(t, db, "CAP-001", "10uF Capacitor", "MPN2", 10, 50, 200)      // Below reorder (diff: 40)
	insertTestInventoryItem(t, db, "IC-001", "ATmega328P", "MPN3", 5, 20, 50)             // Below reorder (diff: 15)

	// Items at or above reorder point (should NOT appear)
	insertTestInventoryItem(t, db, "RES-002", "220 Ohm Resistor", "MPN4", 100, 100, 500)  // At reorder
	insertTestInventoryItem(t, db, "CAP-002", "22uF Capacitor", "MPN5", 200, 50, 200)     // Above reorder

	// Item with reorder_point=0 (should NOT appear)
	insertTestInventoryItem(t, db, "CONN-001", "USB Connector", "MPN6", 0, 0, 0)

	req := httptest.NewRequest("GET", "/api/reports/low-stock", nil)
	w := httptest.NewRecorder()

	handleReportLowStock(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var items []LowStockItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should show only 3 low-stock items
	if len(items) != 3 {
		t.Errorf("Expected 3 low-stock items, got %d", len(items))
	}

	// Verify ordering by shortage (DESC): RES-001 (50), CAP-001 (40), IC-001 (15)
	expectedOrder := []string{"RES-001", "CAP-001", "IC-001"}
	for i, expected := range expectedOrder {
		if items[i].IPN != expected {
			t.Errorf("Expected item[%d] IPN '%s', got '%s'", i, expected, items[i].IPN)
		}
	}

	// Verify suggested order calculation
	for _, item := range items {
		if item.IPN == "RES-001" {
			if item.SuggestedOrder != 500 {
				t.Errorf("Expected RES-001 suggested_order=500 (reorder_qty), got %.2f", item.SuggestedOrder)
			}
		}
		if item.IPN == "IC-001" {
			if item.SuggestedOrder != 50 {
				t.Errorf("Expected IC-001 suggested_order=50, got %.2f", item.SuggestedOrder)
			}
		}
	}
}

func TestReportLowStock_SuggestedOrderCalculation(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Item with reorder_qty set
	insertTestInventoryItem(t, db, "RES-001", "Test 1", "MPN1", 10, 100, 500)

	// Item with reorder_qty=0 (should calculate as reorder_point - qty_on_hand)
	insertTestInventoryItem(t, db, "RES-002", "Test 2", "MPN2", 10, 100, 0)

	req := httptest.NewRequest("GET", "/api/reports/low-stock", nil)
	w := httptest.NewRecorder()

	handleReportLowStock(w, req)

	var items []LowStockItem
	json.NewDecoder(w.Body).Decode(&items)

	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	for _, item := range items {
		if item.IPN == "RES-001" {
			// Should use reorder_qty
			if item.SuggestedOrder != 500 {
				t.Errorf("Expected RES-001 suggested_order=500, got %.2f", item.SuggestedOrder)
			}
		}
		if item.IPN == "RES-002" {
			// Should calculate: reorder_point - qty_on_hand = 100 - 10 = 90
			expectedSuggested := 90.0
			if item.SuggestedOrder != expectedSuggested {
				t.Errorf("Expected RES-002 suggested_order=%.2f, got %.2f", expectedSuggested, item.SuggestedOrder)
			}
		}
	}
}

func TestReportLowStock_OrderingByShortage(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert items with different shortage levels
	insertTestInventoryItem(t, db, "ITEM-A", "Small shortage", "MPN1", 90, 100, 100)  // shortage: 10
	insertTestInventoryItem(t, db, "ITEM-B", "Large shortage", "MPN2", 10, 100, 100)  // shortage: 90
	insertTestInventoryItem(t, db, "ITEM-C", "Medium shortage", "MPN3", 60, 100, 100) // shortage: 40

	req := httptest.NewRequest("GET", "/api/reports/low-stock", nil)
	w := httptest.NewRecorder()

	handleReportLowStock(w, req)

	var items []LowStockItem
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(items) == 0 {
		t.Fatalf("Expected items to be returned, got empty slice")
	}

	// Should be ordered by shortage DESC: ITEM-B (90), ITEM-C (40), ITEM-A (10)
	expectedOrder := []string{"ITEM-B", "ITEM-C", "ITEM-A"}
	if len(items) != len(expectedOrder) {
		t.Fatalf("Expected %d items, got %d", len(expectedOrder), len(items))
	}
	for i, expected := range expectedOrder {
		if items[i].IPN != expected {
			t.Errorf("Expected item[%d] = '%s', got '%s'", i, expected, items[i].IPN)
		}
	}
}

func TestReportLowStock_CSV(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestInventoryItem(t, db, "RES-001", "100 Ohm Resistor", "MPN1", 50, 100, 500)

	req := httptest.NewRequest("GET", "/api/reports/low-stock?format=csv", nil)
	w := httptest.NewRecorder()

	handleReportLowStock(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", ct)
	}

	csvData := w.Body.String()
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) < 2 {
		t.Errorf("Expected at least 2 rows, got %d", len(records))
	}

	// Verify header
	expectedHeaders := []string{"IPN", "Description", "Qty On Hand", "Reorder Point", "Reorder Qty", "Suggested Order"}
	header := records[0]
	for i, expected := range expectedHeaders {
		if i >= len(header) || header[i] != expected {
			t.Errorf("Expected header[%d] = '%s', got '%s'", i, expected, header[i])
		}
	}

	// Verify data
	if len(records) > 1 {
		dataRow := records[1]
		if dataRow[0] != "RES-001" {
			t.Errorf("Expected IPN 'RES-001', got '%s'", dataRow[0])
		}
		if dataRow[5] != "500.00" {
			t.Errorf("Expected suggested_order '500.00', got '%s'", dataRow[5])
		}
	}
}

func TestReportLowStock_ZeroReorderPoint(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Items with reorder_point=0 should NOT appear
	insertTestInventoryItem(t, db, "RES-001", "No reorder point", "MPN1", 0, 0, 0)
	insertTestInventoryItem(t, db, "RES-002", "With reorder point", "MPN2", 10, 100, 100)

	req := httptest.NewRequest("GET", "/api/reports/low-stock", nil)
	w := httptest.NewRecorder()

	handleReportLowStock(w, req)

	var items []LowStockItem
	json.NewDecoder(w.Body).Decode(&items)

	// Should only show RES-002
	if len(items) != 1 {
		t.Errorf("Expected 1 item (excluding zero reorder_point), got %d", len(items))
	}

	if len(items) > 0 && items[0].IPN != "RES-002" {
		t.Errorf("Expected RES-002, got '%s'", items[0].IPN)
	}
}

// =============================================================================
// NCR SUMMARY REPORT TESTS
// =============================================================================

func TestReportNCRSummary_Empty(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/reports/ncr-summary", nil)
	w := httptest.NewRecorder()

	handleReportNCRSummary(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report NCRSummaryReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if report.TotalOpen != 0 {
		t.Errorf("Expected total_open=0, got %d", report.TotalOpen)
	}

	if report.AvgResolveDays != 0 {
		t.Errorf("Expected avg_resolve_days=0, got %.2f", report.AvgResolveDays)
	}

	if len(report.BySeverity) != 0 {
		t.Errorf("Expected empty by_severity, got %d entries", len(report.BySeverity))
	}

	if len(report.ByDefectType) != 0 {
		t.Errorf("Expected empty by_defect_type, got %d entries", len(report.ByDefectType))
	}
}

func TestReportNCRSummary_WithData(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Open NCRs (should be counted)
	insertTestNCR(t, db, "NCR-001", "critical", "dimensional", "open", nil)
	insertTestNCR(t, db, "NCR-002", "major", "cosmetic", "investigating", nil)
	insertTestNCR(t, db, "NCR-003", "minor", "dimensional", "open", nil)
	insertTestNCR(t, db, "NCR-004", "critical", "electrical", "open", nil)

	// Closed/Resolved NCRs (should NOT be counted in open)
	resolvedAt := "2024-01-10 15:00:00"
	insertTestNCR(t, db, "NCR-005", "major", "cosmetic", "closed", &resolvedAt)
	insertTestNCR(t, db, "NCR-006", "minor", "dimensional", "resolved", &resolvedAt)

	req := httptest.NewRequest("GET", "/api/reports/ncr-summary", nil)
	w := httptest.NewRecorder()

	handleReportNCRSummary(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report NCRSummaryReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should count 4 open NCRs
	if report.TotalOpen != 4 {
		t.Errorf("Expected total_open=4, got %d", report.TotalOpen)
	}

	// Verify by_severity counts
	if report.BySeverity["critical"] != 2 {
		t.Errorf("Expected 2 critical NCRs, got %d", report.BySeverity["critical"])
	}
	if report.BySeverity["major"] != 1 {
		t.Errorf("Expected 1 major NCR, got %d", report.BySeverity["major"])
	}
	if report.BySeverity["minor"] != 1 {
		t.Errorf("Expected 1 minor NCR, got %d", report.BySeverity["minor"])
	}

	// Verify by_defect_type counts
	if report.ByDefectType["dimensional"] != 2 {
		t.Errorf("Expected 2 dimensional NCRs, got %d", report.ByDefectType["dimensional"])
	}
	if report.ByDefectType["cosmetic"] != 1 {
		t.Errorf("Expected 1 cosmetic NCR, got %d", report.ByDefectType["cosmetic"])
	}
	if report.ByDefectType["electrical"] != 1 {
		t.Errorf("Expected 1 electrical NCR, got %d", report.ByDefectType["electrical"])
	}
}

func TestReportNCRSummary_AvgResolveTime(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create resolved NCRs with known durations
	// NCR created on 2024-01-01, resolved on 2024-01-03 = 2 days
	db.Exec(`INSERT INTO ncrs (id, title, severity, defect_type, status, created_at, resolved_at) VALUES 
		('NCR-001', 'Test 1', 'major', 'dimensional', 'closed', '2024-01-01 10:00:00', '2024-01-03 10:00:00')`)

	// NCR created on 2024-01-01, resolved on 2024-01-05 = 4 days
	db.Exec(`INSERT INTO ncrs (id, title, severity, defect_type, status, created_at, resolved_at) VALUES 
		('NCR-002', 'Test 2', 'minor', 'cosmetic', 'resolved', '2024-01-01 10:00:00', '2024-01-05 10:00:00')`)

	req := httptest.NewRequest("GET", "/api/reports/ncr-summary", nil)
	w := httptest.NewRecorder()

	handleReportNCRSummary(w, req)

	var report NCRSummaryReport
	json.NewDecoder(w.Body).Decode(&report)

	// Average: (2 + 4) / 2 = 3 days
	expectedAvg := 3.0
	if report.AvgResolveDays != expectedAvg {
		t.Errorf("Expected avg_resolve_days=%.2f, got %.2f", expectedAvg, report.AvgResolveDays)
	}
}

func TestReportNCRSummary_NullSeverityAndDefectType(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert NCR with NULL severity and defect_type (COALESCE should convert to 'unknown')
	db.Exec(`INSERT INTO ncrs (id, title, status, created_at) VALUES 
		('NCR-001', 'Test NCR', 'open', datetime('now'))`)

	req := httptest.NewRequest("GET", "/api/reports/ncr-summary", nil)
	w := httptest.NewRecorder()

	handleReportNCRSummary(w, req)

	var report NCRSummaryReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.TotalOpen != 1 {
		t.Errorf("Expected total_open=1, got %d", report.TotalOpen)
	}

	// Should map NULL to 'unknown' via COALESCE
	if report.BySeverity["unknown"] != 1 {
		t.Errorf("Expected 1 'unknown' severity, got %d", report.BySeverity["unknown"])
	}

	if report.ByDefectType["unknown"] != 1 {
		t.Errorf("Expected 1 'unknown' defect_type, got %d", report.ByDefectType["unknown"])
	}
}

func TestReportNCRSummary_OnlyOpenNCRs(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test all possible NCR statuses
	statuses := []struct {
		status      string
		shouldCount bool
	}{
		{"open", true},
		{"investigating", true},
		{"resolved", false},
		{"closed", false},
	}

	for i, st := range statuses {
		id := fmt.Sprintf("NCR-%03d", i+1)
		insertTestNCR(t, db, id, "major", "dimensional", st.status, nil)
	}

	req := httptest.NewRequest("GET", "/api/reports/ncr-summary", nil)
	w := httptest.NewRecorder()

	handleReportNCRSummary(w, req)

	var report NCRSummaryReport
	json.NewDecoder(w.Body).Decode(&report)

	// Should count only open and investigating (2 NCRs)
	expectedOpen := 2
	if report.TotalOpen != expectedOpen {
		t.Errorf("Expected total_open=%d, got %d", expectedOpen, report.TotalOpen)
	}
}

func TestReportNCRSummary_CSV(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestNCR(t, db, "NCR-001", "critical", "dimensional", "open", nil)
	insertTestNCR(t, db, "NCR-002", "major", "cosmetic", "open", nil)

	req := httptest.NewRequest("GET", "/api/reports/ncr-summary?format=csv", nil)
	w := httptest.NewRecorder()

	handleReportNCRSummary(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", ct)
	}

	csvData := w.Body.String()
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have summary + severity breakdown + defect type breakdown
	if len(records) < 6 {
		t.Errorf("Expected at least 6 rows, got %d", len(records))
	}

	// Verify structure
	if records[0][0] != "Metric" {
		t.Errorf("Expected first row header 'Metric', got '%s'", records[0][0])
	}
}

func TestReportNCRSummary_ResolveTimeWithMissingTimestamps(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// NCR with resolved_at but created_at missing (shouldn't happen in practice, but test robustness)
	// NCR with both timestamps
	db.Exec(`INSERT INTO ncrs (id, title, severity, defect_type, status, created_at, resolved_at) VALUES 
		('NCR-001', 'Test', 'major', 'dimensional', 'closed', '2024-01-01 10:00:00', '2024-01-03 10:00:00')`)

	// NCR with resolved_at but NULL created_at (edge case)
	db.Exec(`INSERT INTO ncrs (id, title, severity, defect_type, status, resolved_at) VALUES 
		('NCR-002', 'Test', 'minor', 'cosmetic', 'closed', '2024-01-05 10:00:00')`)

	req := httptest.NewRequest("GET", "/api/reports/ncr-summary", nil)
	w := httptest.NewRecorder()

	handleReportNCRSummary(w, req)

	var report NCRSummaryReport
	json.NewDecoder(w.Body).Decode(&report)

	// Should calculate average from only NCR-001 (2 days)
	expectedAvg := 2.0
	if report.AvgResolveDays != expectedAvg {
		t.Errorf("Expected avg_resolve_days=%.2f (only NCR-001), got %.2f", expectedAvg, report.AvgResolveDays)
	}
}

// =============================================================================
// SECURITY & EDGE CASE TESTS
// =============================================================================

func TestReports_XSS_Prevention(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert data with XSS payloads
	insertTestInventoryItem(t, db, "<script>alert('xss')</script>", "XSS test", "MPN", 10, 50, 100)
	insertTestECO(t, db, "ECO-XSS", "<img src=x onerror=alert('xss')>", "draft", "normal", "user1", "2024-01-01 10:00:00")

	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		url     string
	}{
		{"inventory valuation", handleReportInventoryValuation, "/api/reports/inventory-valuation"},
		{"open ecos", handleReportOpenECOs, "/api/reports/open-ecos"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// JSON responses should contain escaped data
			body := w.Body.String()
			if strings.Contains(body, "<script>") && !strings.Contains(body, "\\u003c") {
				t.Error("Response may contain unescaped script tags")
			}
		})
	}
}

func TestReports_SQLInjection_Prevention(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// The reports don't take user input params beyond ?days and ?format
	// But test that malicious values don't break anything

	maliciousTests := []struct {
		name  string
		param string
		value string
	}{
		{"days_sql_injection", "days", "30'; DROP TABLE ncrs; --"},
		{"format_sql_injection", "format", "csv'; DELETE FROM inventory; --"},
		{"days_or_injection", "days", "30 OR 1=1"},
	}

	for _, tt := range maliciousTests {
		t.Run(tt.name, func(t *testing.T) {
			urlStr := "/api/reports/wo-throughput?" + url.QueryEscape(tt.param) + "=" + url.QueryEscape(tt.value)
			req := httptest.NewRequest("GET", urlStr, nil)
			w := httptest.NewRecorder()

			handleReportWOThroughput(w, req)

			// Should handle gracefully (either 200 with default values or 400)
			if w.Code != 200 && w.Code != 400 {
				t.Errorf("Expected status 200 or 400, got %d", w.Code)
			}

			// Verify tables still exist
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM work_orders").Scan(&count)
			if err != nil {
				t.Error("Table appears damaged - SQL injection vulnerability!")
			}
		})
	}
}

func TestReports_ConcurrentAccess(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert some test data
	insertTestInventoryItem(t, db, "RES-001", "Test", "MPN", 100, 50, 100)

	// Run multiple concurrent requests
	const concurrency = 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
			w := httptest.NewRecorder()
			handleReportInventoryValuation(w, req)
			done <- w.Code == 200
		}()
	}

	// Wait for all to complete
	for i := 0; i < concurrency; i++ {
		success := <-done
		if !success {
			t.Error("Concurrent request failed")
		}
	}
}

func TestReports_LargeDataset(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert 100 inventory items
	for i := 1; i <= 100; i++ {
		ipn := fmt.Sprintf("IPN-%05d", i)
		insertTestInventoryItem(t, db, ipn, "Test Item "+ipn, "MPN"+ipn, 100, 50, 100)
	}

	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
	w := httptest.NewRecorder()

	handleReportInventoryValuation(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report InvValuationReport
	json.NewDecoder(w.Body).Decode(&report)

	// Verify all items are included
	totalItems := 0
	for _, group := range report.Groups {
		totalItems += len(group.Items)
	}

	if totalItems != 100 {
		t.Errorf("Expected 100 items in report, got %d", totalItems)
	}
}

// =============================================================================
// CALCULATION ACCURACY TESTS
// =============================================================================

func TestInventoryValuation_CalculationAccuracy(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test with precise decimal values
	insertTestInventoryItem(t, db, "RES-001", "Test", "MPN", 1234.56, 0, 0)
	insertTestPO(t, db, "PO-001", "VENDOR", "received", "2024-01-01 10:00:00")
	insertTestPOLineForReports(t, db, "PO-001", "RES-001", 0.0789) // Precise price

	req := httptest.NewRequest("GET", "/api/reports/inventory-valuation", nil)
	w := httptest.NewRecorder()

	handleReportInventoryValuation(w, req)

	var report InvValuationReport
	json.NewDecoder(w.Body).Decode(&report)

	// Expected: 1234.56 * 0.0789 = 97.4067
	expectedSubtotal := 1234.56 * 0.0789
	actualSubtotal := report.Groups[0].Items[0].Subtotal

	// Allow small floating point tolerance
	tolerance := 0.01
	if actualSubtotal < expectedSubtotal-tolerance || actualSubtotal > expectedSubtotal+tolerance {
		t.Errorf("Calculation inaccuracy: expected %.4f, got %.4f", expectedSubtotal, actualSubtotal)
	}

	if report.GrandTotal != actualSubtotal {
		t.Errorf("Grand total mismatch: expected %.4f, got %.4f", actualSubtotal, report.GrandTotal)
	}
}

func TestWOThroughput_CycleTimeRounding(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test cycle time rounding (should round to 2 decimal places)
	// 25 hours = 1.041666... days, should round to 1.04
	start := "2024-02-01 08:00:00"
	end := "2024-02-02 09:00:00" // 25 hours later

	insertTestWorkOrder(t, db, "WO-001", "completed", &start, &end)

	req := httptest.NewRequest("GET", "/api/reports/wo-throughput", nil)
	w := httptest.NewRecorder()

	handleReportWOThroughput(w, req)

	var report WOThroughputReport
	json.NewDecoder(w.Body).Decode(&report)

	// 25 hours / 24 = 1.041666..., rounded to 1.04
	expected := 1.04
	if report.AvgCycleTimeDays != expected {
		t.Errorf("Expected cycle time %.2f, got %.2f", expected, report.AvgCycleTimeDays)
	}
}

func TestNCRSummary_ResolveTimeRounding(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test resolve time rounding
	// 25 hours = 1.041666... days, should round to 1.04
	db.Exec(`INSERT INTO ncrs (id, title, severity, defect_type, status, created_at, resolved_at) VALUES 
		('NCR-001', 'Test', 'major', 'dimensional', 'closed', '2024-01-01 08:00:00', '2024-01-02 09:00:00')`)

	req := httptest.NewRequest("GET", "/api/reports/ncr-summary", nil)
	w := httptest.NewRecorder()

	handleReportNCRSummary(w, req)

	var report NCRSummaryReport
	json.NewDecoder(w.Body).Decode(&report)

	expected := 1.04
	if report.AvgResolveDays != expected {
		t.Errorf("Expected resolve time %.2f, got %.2f", expected, report.AvgResolveDays)
	}
}

// =============================================================================
// CSV FORMAT VALIDATION TESTS
// =============================================================================

func TestCSV_SpecialCharacters(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert data with CSV special characters
	insertTestInventoryItem(t, db, "RES-001", "Test, with \"quotes\" and, commas", "MPN", 100, 50, 100)

	req := httptest.NewRequest("GET", "/api/reports/low-stock?format=csv", nil)
	w := httptest.NewRecorder()

	handleReportLowStock(w, req)

	csvData := w.Body.String()
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV with special characters: %v", err)
	}

	// Verify CSV parser handled the quoted field correctly
	if len(records) < 2 {
		t.Fatal("Expected at least 2 rows")
	}

	// The description should be properly escaped/quoted
	dataRow := records[1]
	if !strings.Contains(dataRow[1], "quotes") {
		t.Error("CSV parsing lost special characters")
	}
}

func TestCSV_Encoding(t *testing.T) {
	oldDB := db
	db = setupReportsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert data with Unicode characters
	insertTestInventoryItem(t, db, "RES-001", "Test Ω Resistor 100Ω ±5%", "MPN", 100, 50, 100)

	req := httptest.NewRequest("GET", "/api/reports/low-stock?format=csv", nil)
	w := httptest.NewRecorder()

	handleReportLowStock(w, req)

	csvData := w.Body.String()
	
	// Verify Unicode characters are preserved
	if !strings.Contains(csvData, "Ω") {
		t.Error("CSV lost Unicode characters")
	}
}
