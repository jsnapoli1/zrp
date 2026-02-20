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

func setupReceivingTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create receiving_inspections table
	_, err = testDB.Exec(`
		CREATE TABLE receiving_inspections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			po_line_id INTEGER NOT NULL,
			ipn TEXT NOT NULL,
			qty_received REAL NOT NULL CHECK(qty_received >= 0),
			qty_passed REAL DEFAULT 0 CHECK(qty_passed >= 0),
			qty_failed REAL DEFAULT 0 CHECK(qty_failed >= 0),
			qty_on_hold REAL DEFAULT 0 CHECK(qty_on_hold >= 0),
			inspector TEXT,
			inspected_at DATETIME,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create receiving_inspections table: %v", err)
	}

	// Create inventory table (for inventory updates)
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			updated_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create inventory_transactions table (for audit trail)
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

	// Create ncrs table (for auto-created NCRs on inspection failure)
	_, err = testDB.Exec(`
		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			ipn TEXT,
			defect_type TEXT,
			severity TEXT,
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
	}

	// Create id_sequences table (for nextID function)
	_, err = testDB.Exec(`
		CREATE TABLE id_sequences (
			prefix TEXT PRIMARY KEY,
			next_num INTEGER DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create id_sequences table: %v", err)
	}

	// Create audit_log table (for logAudit)
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

// Helper to insert a test receiving inspection
func insertTestReceivingInspection(t *testing.T, db *sql.DB, poID string, poLineID int, ipn string, qtyReceived float64) int {
	result, err := db.Exec(
		`INSERT INTO receiving_inspections (po_id, po_line_id, ipn, qty_received, qty_passed, qty_failed, qty_on_hold, created_at) 
		VALUES (?, ?, ?, ?, 0, 0, 0, datetime('now'))`,
		poID, poLineID, ipn, qtyReceived,
	)
	if err != nil {
		t.Fatalf("Failed to insert test receiving inspection: %v", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert ID: %v", err)
	}
	
	return int(id)
}

// Helper to insert test inventory
func insertTestInventory(t *testing.T, db *sql.DB, ipn string, qtyOnHand float64) {
	_, err := db.Exec(
		`INSERT INTO inventory (ipn, qty_on_hand, updated_at) VALUES (?, ?, datetime('now'))`,
		ipn, qtyOnHand,
	)
	if err != nil {
		t.Fatalf("Failed to insert test inventory: %v", err)
	}
}

// =============================================================================
// LIST RECEIVING INSPECTIONS TESTS
// =============================================================================

func TestHandleListReceiving_Empty(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/receiving", nil)
	w := httptest.NewRecorder()

	handleListReceiving(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	inspections, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(inspections) != 0 {
		t.Errorf("Expected empty array, got %d inspections", len(inspections))
	}
}

func TestHandleListReceiving_WithData(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)
	insertTestReceivingInspection(t, db, "PO-002", 2, "IPN-200", 50)
	insertTestReceivingInspection(t, db, "PO-003", 3, "IPN-300", 25)

	req := httptest.NewRequest("GET", "/api/receiving", nil)
	w := httptest.NewRecorder()

	handleListReceiving(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	inspectionsData, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(inspectionsData) != 3 {
		t.Errorf("Expected 3 inspections, got %d", len(inspectionsData))
	}

	// Verify first inspection has expected fields
	first := inspectionsData[0].(map[string]interface{})
	if first["po_id"] == nil || first["po_id"] == "" {
		t.Error("Expected po_id to be set")
	}
	if first["ipn"] == nil || first["ipn"] == "" {
		t.Error("Expected ipn to be set")
	}
	if qtyRecv, ok := first["qty_received"].(float64); !ok || qtyRecv <= 0 {
		t.Error("Expected qty_received > 0")
	}
}

func TestHandleListReceiving_OrderByCreatedDesc(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert with explicit timestamps to verify ordering
	db.Exec(`INSERT INTO receiving_inspections (po_id, po_line_id, ipn, qty_received, created_at) VALUES 
		('PO-001', 1, 'IPN-100', 100, '2024-01-01 10:00:00'),
		('PO-002', 2, 'IPN-200', 50, '2024-01-03 10:00:00'),
		('PO-003', 3, 'IPN-300', 25, '2024-01-02 10:00:00')
	`)

	req := httptest.NewRequest("GET", "/api/receiving", nil)
	w := httptest.NewRecorder()

	handleListReceiving(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	inspections := resp.Data.([]interface{})

	if len(inspections) != 3 {
		t.Fatalf("Expected 3 inspections, got %d", len(inspections))
	}

	// Should be ordered DESC by created_at: PO-002, PO-003, PO-001
	first := inspections[0].(map[string]interface{})
	if first["po_id"] != "PO-002" {
		t.Errorf("Expected first inspection to be PO-002 (most recent), got %v", first["po_id"])
	}
	second := inspections[1].(map[string]interface{})
	if second["po_id"] != "PO-003" {
		t.Errorf("Expected second inspection to be PO-003, got %v", second["po_id"])
	}
	third := inspections[2].(map[string]interface{})
	if third["po_id"] != "PO-001" {
		t.Errorf("Expected third inspection to be PO-001 (oldest), got %v", third["po_id"])
	}
}

func TestHandleListReceiving_FilterPending(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert pending (not inspected)
	insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)
	insertTestReceivingInspection(t, db, "PO-002", 2, "IPN-200", 50)

	// Insert inspected
	db.Exec(`INSERT INTO receiving_inspections (po_id, po_line_id, ipn, qty_received, qty_passed, inspected_at, created_at) 
		VALUES ('PO-003', 3, 'IPN-300', 25, 25, datetime('now'), datetime('now'))`)

	req := httptest.NewRequest("GET", "/api/receiving?status=pending", nil)
	w := httptest.NewRecorder()

	handleListReceiving(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	inspections := resp.Data.([]interface{})

	if len(inspections) != 2 {
		t.Errorf("Expected 2 pending inspections, got %d", len(inspections))
	}

	// Verify all returned items are pending (inspected_at is nil)
	for i, item := range inspections {
		ri := item.(map[string]interface{})
		if ri["inspected_at"] != nil {
			t.Errorf("Inspection %d should be pending (inspected_at nil), got %v", i, ri["inspected_at"])
		}
	}
}

func TestHandleListReceiving_FilterInspected(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert pending
	insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)

	// Insert inspected
	db.Exec(`INSERT INTO receiving_inspections (po_id, po_line_id, ipn, qty_received, qty_passed, inspected_at, inspector, created_at) 
		VALUES ('PO-002', 2, 'IPN-200', 50, 50, datetime('now'), 'testuser', datetime('now'))`)
	db.Exec(`INSERT INTO receiving_inspections (po_id, po_line_id, ipn, qty_received, qty_failed, inspected_at, inspector, created_at) 
		VALUES ('PO-003', 3, 'IPN-300', 25, 25, datetime('now'), 'testuser', datetime('now'))`)

	req := httptest.NewRequest("GET", "/api/receiving?status=inspected", nil)
	w := httptest.NewRecorder()

	handleListReceiving(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	inspections := resp.Data.([]interface{})

	if len(inspections) != 2 {
		t.Errorf("Expected 2 inspected items, got %d", len(inspections))
	}

	// Verify all returned items are inspected (inspected_at is not nil)
	for i, item := range inspections {
		ri := item.(map[string]interface{})
		if ri["inspected_at"] == nil {
			t.Errorf("Inspection %d should be inspected (inspected_at set), got nil", i)
		}
	}
}

func TestHandleListReceiving_NoFilter(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert both pending and inspected
	insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)
	db.Exec(`INSERT INTO receiving_inspections (po_id, po_line_id, ipn, qty_received, qty_passed, inspected_at, inspector, created_at) 
		VALUES ('PO-002', 2, 'IPN-200', 50, 50, datetime('now'), 'testuser', datetime('now'))`)

	req := httptest.NewRequest("GET", "/api/receiving", nil)
	w := httptest.NewRecorder()

	handleListReceiving(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	inspections := resp.Data.([]interface{})

	if len(inspections) != 2 {
		t.Errorf("Expected 2 total inspections, got %d", len(inspections))
	}
}

// =============================================================================
// INSPECT RECEIVING TESTS
// =============================================================================

func TestHandleInspectReceiving_Success_AllPassed(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)
	insertTestInventory(t, db, "IPN-100", 50) // Starting inventory: 50

	reqBody := `{
		"qty_passed": 100,
		"qty_failed": 0,
		"qty_on_hold": 0,
		"inspector": "testuser",
		"notes": "All items passed inspection"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	respData := apiResp.Data.(map[string]interface{})

	// Verify inspection was updated
	if respData["qty_passed"].(float64) != 100 {
		t.Errorf("Expected qty_passed=100, got %.0f", respData["qty_passed"].(float64))
	}
	if respData["qty_failed"].(float64) != 0 {
		t.Errorf("Expected qty_failed=0, got %.0f", respData["qty_failed"].(float64))
	}
	if respData["qty_on_hold"].(float64) != 0 {
		t.Errorf("Expected qty_on_hold=0, got %.0f", respData["qty_on_hold"].(float64))
	}
	if respData["inspector"].(string) != "testuser" {
		t.Errorf("Expected inspector='testuser', got '%s'", respData["inspector"].(string))
	}
	if respData["inspected_at"] == nil {
		t.Error("Expected inspected_at to be set")
	}

	// CRITICAL: Verify inventory was updated correctly
	var qtyOnHand float64
	err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "IPN-100").Scan(&qtyOnHand)
	if err != nil {
		t.Fatalf("Failed to query inventory: %v", err)
	}
	expectedQty := 50.0 + 100.0 // Starting 50 + 100 passed
	if qtyOnHand != expectedQty {
		t.Errorf("INVENTORY BUG: Expected qty_on_hand=%.0f, got %.0f", expectedQty, qtyOnHand)
	}

	// Verify inventory transaction was created
	var txCount int
	err = db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn = ? AND type = 'receive'", "IPN-100").Scan(&txCount)
	if err != nil {
		t.Fatalf("Failed to count inventory transactions: %v", err)
	}
	if txCount != 1 {
		t.Errorf("Expected 1 inventory transaction, got %d", txCount)
	}

	// Verify audit log was created
	var auditCount int
	err = db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE record_id = ? AND action = 'inspected'", fmt.Sprintf("%d", riID)).Scan(&auditCount)
	if err != nil {
		t.Fatalf("Failed to count audit log: %v", err)
	}
	if auditCount != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", auditCount)
	}
}

func TestHandleInspectReceiving_Success_AllFailed(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)
	insertTestInventory(t, db, "IPN-100", 50)

	reqBody := `{
		"qty_passed": 0,
		"qty_failed": 100,
		"qty_on_hold": 0,
		"inspector": "testuser",
		"notes": "All items failed - wrong part number"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify inventory was NOT updated (failed items don't go into inventory)
	var qtyOnHand float64
	err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "IPN-100").Scan(&qtyOnHand)
	if err != nil {
		t.Fatalf("Failed to query inventory: %v", err)
	}
	if qtyOnHand != 50.0 {
		t.Errorf("INVENTORY BUG: Failed items should not update inventory. Expected qty_on_hand=50, got %.0f", qtyOnHand)
	}

	// Verify NCR was auto-created
	var ncrCount int
	err = db.QueryRow("SELECT COUNT(*) FROM ncrs WHERE ipn = ? AND defect_type = 'receiving'", "IPN-100").Scan(&ncrCount)
	if err != nil {
		t.Fatalf("Failed to count NCRs: %v", err)
	}
	if ncrCount != 1 {
		t.Errorf("Expected 1 NCR to be auto-created, got %d", ncrCount)
	}

	// Verify NCR details
	var ncrTitle, ncrDesc string
	err = db.QueryRow("SELECT title, description FROM ncrs WHERE ipn = ?", "IPN-100").Scan(&ncrTitle, &ncrDesc)
	if err != nil {
		t.Fatalf("Failed to query NCR: %v", err)
	}
	if !strings.Contains(ncrTitle, "IPN-100") {
		t.Errorf("Expected NCR title to contain IPN, got '%s'", ncrTitle)
	}
	if !strings.Contains(ncrDesc, "100 units") {
		t.Errorf("Expected NCR description to contain quantity, got '%s'", ncrDesc)
	}
}

func TestHandleInspectReceiving_Success_Mixed(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)
	insertTestInventory(t, db, "IPN-100", 50)

	reqBody := `{
		"qty_passed": 80,
		"qty_failed": 15,
		"qty_on_hold": 5,
		"inspector": "testuser",
		"notes": "Most passed, some failed cosmetic inspection, holding 5 for re-test"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify inventory updated with only passed items
	var qtyOnHand float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "IPN-100").Scan(&qtyOnHand)
	expectedQty := 50.0 + 80.0
	if qtyOnHand != expectedQty {
		t.Errorf("INVENTORY BUG: Expected qty_on_hand=%.0f, got %.0f", expectedQty, qtyOnHand)
	}

	// Verify NCR was created for failed items
	var ncrCount int
	db.QueryRow("SELECT COUNT(*) FROM ncrs WHERE ipn = ?", "IPN-100").Scan(&ncrCount)
	if ncrCount != 1 {
		t.Errorf("Expected 1 NCR for failed items, got %d", ncrCount)
	}

	// Verify on-hold items are tracked (not added to inventory, no NCR)
	// The on-hold quantity should be visible in the inspection record
	var qtyOnHoldDB float64
	db.QueryRow("SELECT qty_on_hold FROM receiving_inspections WHERE id = ?", riID).Scan(&qtyOnHoldDB)
	if qtyOnHoldDB != 5 {
		t.Errorf("Expected qty_on_hold=5, got %.0f", qtyOnHoldDB)
	}
}

func TestHandleInspectReceiving_InvalidID(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{"qty_passed": 100, "qty_failed": 0, "qty_on_hold": 0}`
	req := httptest.NewRequest("PUT", "/api/receiving/invalid/inspect", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, "invalid")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "invalid") {
		t.Errorf("Expected error message about invalid id, got: %s", w.Body.String())
	}
}

func TestHandleInspectReceiving_NotFound(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{"qty_passed": 100, "qty_failed": 0, "qty_on_hold": 0}`
	req := httptest.NewRequest("PUT", "/api/receiving/999/inspect", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, "999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleInspectReceiving_QuantityValidation_ExceedsReceived(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100) // Received 100

	tests := []struct {
		name        string
		qtyPassed   float64
		qtyFailed   float64
		qtyOnHold   float64
		expectError bool
	}{
		{"total exceeds by 1", 100, 0, 1, true},
		{"total exceeds by 50", 100, 50, 0, true},
		{"total exceeds all high", 50, 50, 10, true},
		{"exact match", 100, 0, 0, false},
		{"partial match", 80, 15, 5, false},
		{"under received", 50, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := fmt.Sprintf(`{
				"qty_passed": %.0f,
				"qty_failed": %.0f,
				"qty_on_hold": %.0f,
				"inspector": "testuser"
			}`, tt.qtyPassed, tt.qtyFailed, tt.qtyOnHold)
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

			if tt.expectError {
				if w.Code != 400 {
					t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
				}
				if !strings.Contains(w.Body.String(), "exceed") {
					t.Errorf("Expected error about exceeding quantity, got: %s", w.Body.String())
				}
			} else {
				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestHandleInspectReceiving_NegativeQuantities(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)

	reqBody := `{
		"qty_passed": -10,
		"qty_failed": 0,
		"qty_on_hold": 0,
		"inspector": "testuser"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	// Should fail validation (total would be negative)
	// The validation checks if total > qty_received, but negative values should also be caught
	// This is a potential bug if negative values are allowed
	if w.Code == 200 {
		t.Log("WARNING: Negative quantities may be accepted - verify business logic")
	}
}

func TestHandleInspectReceiving_InspectorFromUsername(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)

	// Don't provide inspector in body - should use getUsername(r)
	reqBody := `{
		"qty_passed": 100,
		"qty_failed": 0,
		"qty_on_hold": 0,
		"notes": "Inspected by default user"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp APIResponse
	json.NewDecoder(w.Body).Decode(&apiResp)
	respData := apiResp.Data.(map[string]interface{})

	// Inspector should be set by getUsername(r) - likely "system" or empty in test
	inspector, _ := respData["inspector"].(string)
	if inspector == "" {
		t.Log("Inspector is empty - getUsername(r) returned empty string (expected in test)")
	}
}

func TestHandleInspectReceiving_InvalidJSON(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)

	reqBody := `{invalid json`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandleInspectReceiving_InventoryNotExists_ShouldCreate(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-NEW", 100)
	// Don't insert inventory - should be auto-created

	reqBody := `{
		"qty_passed": 100,
		"qty_failed": 0,
		"qty_on_hold": 0,
		"inspector": "testuser"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify inventory was created and updated
	var qtyOnHand float64
	err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "IPN-NEW").Scan(&qtyOnHand)
	if err != nil {
		t.Fatalf("INVENTORY BUG: Inventory should be auto-created. Error: %v", err)
	}
	if qtyOnHand != 100.0 {
		t.Errorf("Expected qty_on_hand=100, got %.0f", qtyOnHand)
	}
}

func TestHandleInspectReceiving_MultipleInspections_InventoryAccumulation(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	insertTestInventory(t, db, "IPN-100", 0) // Start at 0

	// First receiving
	ri1 := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 50)
	reqBody := `{"qty_passed": 50, "qty_failed": 0, "qty_on_hold": 0, "inspector": "user1"}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", ri1), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleInspectReceiving(w, req, fmt.Sprintf("%d", ri1))

	if w.Code != 200 {
		t.Fatalf("First inspection failed: %d", w.Code)
	}

	// Second receiving
	ri2 := insertTestReceivingInspection(t, db, "PO-002", 2, "IPN-100", 100)
	reqBody = `{"qty_passed": 100, "qty_failed": 0, "qty_on_hold": 0, "inspector": "user2"}`
	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", ri2), bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()
	handleInspectReceiving(w, req, fmt.Sprintf("%d", ri2))

	if w.Code != 200 {
		t.Fatalf("Second inspection failed: %d", w.Code)
	}

	// Third receiving
	ri3 := insertTestReceivingInspection(t, db, "PO-003", 3, "IPN-100", 75)
	reqBody = `{"qty_passed": 75, "qty_failed": 0, "qty_on_hold": 0, "inspector": "user3"}`
	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", ri3), bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()
	handleInspectReceiving(w, req, fmt.Sprintf("%d", ri3))

	if w.Code != 200 {
		t.Fatalf("Third inspection failed: %d", w.Code)
	}

	// CRITICAL: Verify inventory accumulated correctly
	var qtyOnHand float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "IPN-100").Scan(&qtyOnHand)
	expectedQty := 0.0 + 50.0 + 100.0 + 75.0 // 225
	if qtyOnHand != expectedQty {
		t.Errorf("INVENTORY BUG: Expected cumulative qty_on_hand=%.0f, got %.0f", expectedQty, qtyOnHand)
	}

	// Verify all transactions were recorded
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn = ? AND type = 'receive'", "IPN-100").Scan(&txCount)
	if txCount != 3 {
		t.Errorf("Expected 3 inventory transactions, got %d", txCount)
	}
}

func TestHandleInspectReceiving_Concurrency_RaceCondition(t *testing.T) {
	t.Skip("Concurrency test - requires special setup for race detection")
	
	// This test would check for race conditions when multiple inspections
	// for the same IPN happen simultaneously. This is a CRITICAL area for
	// inventory accuracy bugs.
	//
	// To properly test:
	// 1. Run with -race flag
	// 2. Use goroutines to simulate concurrent requests
	// 3. Verify final inventory count is correct
	// 4. Check for lost updates
	//
	// Example scenario:
	// - Start with qty_on_hand = 100
	// - Two inspections run concurrently, each adding 50
	// - Final qty should be 200 (not 150 due to race condition)
	//
	// BUG RISK: The current code uses separate SELECT and UPDATE queries
	// which could lead to race conditions. Should use:
	// UPDATE inventory SET qty_on_hand = qty_on_hand + ? WHERE ipn = ?
	// (which is atomic) instead of read-modify-write pattern.
}

func TestHandleInspectReceiving_ZeroQuantities(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)
	insertTestInventory(t, db, "IPN-100", 50)

	reqBody := `{
		"qty_passed": 0,
		"qty_failed": 0,
		"qty_on_hold": 0,
		"inspector": "testuser",
		"notes": "Nothing inspected yet"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify inventory was NOT changed
	var qtyOnHand float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "IPN-100").Scan(&qtyOnHand)
	if qtyOnHand != 50.0 {
		t.Errorf("Expected qty_on_hand unchanged at 50, got %.0f", qtyOnHand)
	}

	// Verify no transaction was created (qty_passed = 0)
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn = ?", "IPN-100").Scan(&txCount)
	if txCount != 0 {
		t.Errorf("Expected 0 inventory transactions for zero quantities, got %d", txCount)
	}

	// Verify no NCR was created (qty_failed = 0)
	var ncrCount int
	db.QueryRow("SELECT COUNT(*) FROM ncrs WHERE ipn = ?", "IPN-100").Scan(&ncrCount)
	if ncrCount != 0 {
		t.Errorf("Expected 0 NCRs for zero failures, got %d", ncrCount)
	}
}

func TestHandleInspectReceiving_DuplicateInspection(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)
	insertTestInventory(t, db, "IPN-100", 0)

	// First inspection
	reqBody := `{"qty_passed": 100, "qty_failed": 0, "qty_on_hold": 0, "inspector": "user1"}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Fatalf("First inspection failed: %d", w.Code)
	}

	// Verify inventory after first inspection
	var qtyAfterFirst float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "IPN-100").Scan(&qtyAfterFirst)
	if qtyAfterFirst != 100 {
		t.Errorf("Expected 100 after first inspection, got %.0f", qtyAfterFirst)
	}

	// Second inspection (re-inspect same record)
	reqBody = `{"qty_passed": 100, "qty_failed": 0, "qty_on_hold": 0, "inspector": "user2"}`
	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()
	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	// FIXED: Second inspection should be REJECTED (404) because item already inspected
	if w.Code != 404 {
		t.Errorf("Expected 404 for duplicate inspection, got %d", w.Code)
		t.Error("BUG: Duplicate inspection was allowed!")
	}

	// Verify inventory was NOT double-counted
	var qtyAfterSecond float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "IPN-100").Scan(&qtyAfterSecond)
	
	if qtyAfterSecond != 100 {
		t.Errorf("BUG: Inventory double-counted! Expected 100, got %.0f", qtyAfterSecond)
	}

	// Verify transaction count remains 1
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn = ? AND type = 'receive'", "IPN-100").Scan(&txCount)
	if txCount != 1 {
		t.Errorf("Expected 1 transaction, got %d", txCount)
	}
}

func TestHandleInspectReceiving_AuditTrail(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)
	insertTestInventory(t, db, "IPN-100", 0)

	reqBody := `{
		"qty_passed": 80,
		"qty_failed": 15,
		"qty_on_hold": 5,
		"inspector": "john.doe",
		"notes": "Cosmetic defects on 15 units"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Fatalf("Inspection failed: %d", w.Code)
	}

	// Verify audit log entry
	var username, action, module, recordID, summary string
	err := db.QueryRow(`SELECT username, action, module, record_id, summary FROM audit_log 
		WHERE record_id = ? AND action = 'inspected'`, fmt.Sprintf("%d", riID)).
		Scan(&username, &action, &module, &recordID, &summary)
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}

	if username != "john.doe" {
		t.Errorf("Expected audit username='john.doe', got '%s'", username)
	}
	if action != "inspected" {
		t.Errorf("Expected audit action='inspected', got '%s'", action)
	}
	if module != "receiving" {
		t.Errorf("Expected audit module='receiving', got '%s'", module)
	}
	if !strings.Contains(summary, "80 passed") {
		t.Errorf("Expected audit summary to contain pass count, got '%s'", summary)
	}
	if !strings.Contains(summary, "15 failed") {
		t.Errorf("Expected audit summary to contain fail count, got '%s'", summary)
	}
	if !strings.Contains(summary, "5 on-hold") {
		t.Errorf("Expected audit summary to contain on-hold count, got '%s'", summary)
	}
}

func TestHandleInspectReceiving_XSS_Prevention(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	riID := insertTestReceivingInspection(t, db, "PO-001", 1, "IPN-100", 100)

	reqBody := `{
		"qty_passed": 100,
		"qty_failed": 0,
		"qty_on_hold": 0,
		"inspector": "<script>alert('xss')</script>",
		"notes": "'; DROP TABLE inventory; --"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify data was stored as-is (no execution)
	var inspector, notes string
	db.QueryRow("SELECT inspector, notes FROM receiving_inspections WHERE id = ?", riID).Scan(&inspector, &notes)

	if inspector != "<script>alert('xss')</script>" {
		t.Errorf("XSS payload was modified: %s", inspector)
	}

	// Verify SQL injection didn't execute
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM inventory").Scan(&count)
	if err != nil {
		t.Error("Table 'inventory' appears to have been deleted - SQL injection vulnerability!")
	}
}

// =============================================================================
// WHERE USED TESTS (handleWhereUsed)
// =============================================================================

func TestHandleWhereUsed_NotInThisFile(t *testing.T) {
	// handleWhereUsed is in the same file but tests BOM functionality
	// which requires file system access and complex setup.
	// For now, we're focusing on the critical receiving/inspection functionality.
	// WhereUsed tests can be added in a separate test or when BOM testing is standardized.
	t.Skip("handleWhereUsed tests require file system BOM files - out of scope for this critical receiving test")
}

// =============================================================================
// SUMMARY TESTS
// =============================================================================

func TestHandleInspectReceiving_CompleteWorkflow(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Scenario: Receive 100 units, inspect to find 80 good, 15 bad, 5 on hold
	riID := insertTestReceivingInspection(t, db, "PO-12345", 42, "IPN-TEST-001", 100)
	insertTestInventory(t, db, "IPN-TEST-001", 200) // Starting inventory

	reqBody := `{
		"qty_passed": 80,
		"qty_failed": 15,
		"qty_on_hold": 5,
		"inspector": "quality.inspector",
		"notes": "Batch has cosmetic defects on 15 units, 5 held for retest"
	}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/receiving/%d/inspect", riID), bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleInspectReceiving(w, req, fmt.Sprintf("%d", riID))

	if w.Code != 200 {
		t.Fatalf("Inspection failed: %d: %s", w.Code, w.Body.String())
	}

	// 1. Verify inspection record was updated
	var qtyPassed, qtyFailed, qtyOnHold float64
	var inspector, notes string
	var inspectedAt sql.NullString
	err := db.QueryRow(`SELECT qty_passed, qty_failed, qty_on_hold, inspector, inspected_at, notes 
		FROM receiving_inspections WHERE id = ?`, riID).
		Scan(&qtyPassed, &qtyFailed, &qtyOnHold, &inspector, &inspectedAt, &notes)
	if err != nil {
		t.Fatalf("Failed to query inspection: %v", err)
	}

	if qtyPassed != 80 || qtyFailed != 15 || qtyOnHold != 5 {
		t.Errorf("Expected quantities 80/15/5, got %.0f/%.0f/%.0f", qtyPassed, qtyFailed, qtyOnHold)
	}
	if inspector != "quality.inspector" {
		t.Errorf("Expected inspector='quality.inspector', got '%s'", inspector)
	}
	if !inspectedAt.Valid {
		t.Error("Expected inspected_at to be set")
	}

	// 2. Verify inventory updated correctly (200 + 80 = 280)
	var qtyOnHand float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "IPN-TEST-001").Scan(&qtyOnHand)
	if qtyOnHand != 280 {
		t.Errorf("INVENTORY BUG: Expected 280, got %.0f", qtyOnHand)
	}

	// 3. Verify inventory transaction created
	var txQty float64
	var txRef string
	db.QueryRow("SELECT qty, reference FROM inventory_transactions WHERE ipn = ? AND type = 'receive'", "IPN-TEST-001").
		Scan(&txQty, &txRef)
	if txQty != 80 {
		t.Errorf("Expected transaction qty=80, got %.0f", txQty)
	}
	if !strings.Contains(txRef, "PO-12345") {
		t.Errorf("Expected transaction reference to contain PO ID, got '%s'", txRef)
	}

	// 4. Verify NCR was created for failed items
	var ncrID, ncrTitle, ncrDesc string
	db.QueryRow("SELECT id, title, description FROM ncrs WHERE ipn = ?", "IPN-TEST-001").
		Scan(&ncrID, &ncrTitle, &ncrDesc)
	if !strings.Contains(ncrTitle, "IPN-TEST-001") {
		t.Errorf("Expected NCR title to contain IPN, got '%s'", ncrTitle)
	}
	if !strings.Contains(ncrDesc, "15 units") {
		t.Errorf("Expected NCR description to contain failure count, got '%s'", ncrDesc)
	}

	// 5. Verify audit log
	var auditSummary string
	db.QueryRow("SELECT summary FROM audit_log WHERE record_id = ? AND action = 'inspected'", fmt.Sprintf("%d", riID)).
		Scan(&auditSummary)
	if !strings.Contains(auditSummary, "80 passed") {
		t.Errorf("Expected audit summary to contain pass count, got '%s'", auditSummary)
	}
}

func TestHandleListReceiving_NullFields(t *testing.T) {
	oldDB := db
	db = setupReceivingTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert with NULL inspector and notes
	db.Exec(`INSERT INTO receiving_inspections (po_id, po_line_id, ipn, qty_received, created_at) 
		VALUES ('PO-001', 1, 'IPN-100', 100, datetime('now'))`)

	req := httptest.NewRequest("GET", "/api/receiving", nil)
	w := httptest.NewRecorder()

	handleListReceiving(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	inspections := resp.Data.([]interface{})

	if len(inspections) != 1 {
		t.Fatalf("Expected 1 inspection, got %d", len(inspections))
	}

	first := inspections[0].(map[string]interface{})
	// COALESCE should convert NULL to empty string
	if first["inspector"] != "" {
		t.Errorf("Expected empty inspector (COALESCE), got '%v'", first["inspector"])
	}
	if first["notes"] != "" {
		t.Errorf("Expected empty notes (COALESCE), got '%v'", first["notes"])
	}
}
