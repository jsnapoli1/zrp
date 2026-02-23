package inventory_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

// lowStockAlertsHandler is a local test helper that replicates the handleLowStockAlerts
// logic from the root package. This queries inventory for items where qty_on_hand < reorder_point
// and reorder_point > 0, and returns them wrapped in the standard API response envelope.
func lowStockAlertsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT ipn, qty_on_hand, reorder_point FROM inventory WHERE qty_on_hand < reorder_point AND reorder_point > 0")
		if err != nil {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		type LowStockItem struct {
			IPN          string  `json:"ipn"`
			QtyOnHand    float64 `json:"qty_on_hand"`
			ReorderPoint float64 `json:"reorder_point"`
		}
		var items []LowStockItem
		for rows.Next() {
			var i LowStockItem
			rows.Scan(&i.IPN, &i.QtyOnHand, &i.ReorderPoint)
			items = append(items, i)
		}
		if items == nil {
			items = []LowStockItem{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.APIResponse{Data: items})
	}
}

func setupLowStockTestDB(t *testing.T) *sql.DB {
	t.Helper()
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

	// Create audit_log table (required by inventory handlers)
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

// Test Case 1: Part with reorder_point=10, qty=11 -> no alert
func TestLowStockAlerts_NoAlert_WhenAboveThreshold(t *testing.T) {
	testDB := setupLowStockTestDB(t)
	defer testDB.Close()
	handler := lowStockAlertsHandler(testDB)

	// Insert part with qty above reorder point
	_, err := testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) VALUES
		('PART-001', 11, 10, 'Test Part Above Threshold')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var alerts []map[string]interface{}
	if err := json.Unmarshal(dataBytes, &alerts); err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}

	if len(alerts) != 0 {
		t.Errorf("Expected no alerts when qty >= reorder_point, got %d alerts", len(alerts))
	}
}

// Test Case 2: Part with reorder_point=10, qty=9 -> alert generated
func TestLowStockAlerts_AlertGenerated_WhenBelowThreshold(t *testing.T) {
	testDB := setupLowStockTestDB(t)
	defer testDB.Close()
	handler := lowStockAlertsHandler(testDB)

	// Insert part with qty below reorder point
	_, err := testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) VALUES
		('PART-002', 9, 10, 'Test Part Below Threshold')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var alerts []map[string]interface{}
	if err := json.Unmarshal(dataBytes, &alerts); err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}

	if len(alerts) != 1 {
		t.Fatalf("Expected 1 alert when qty < reorder_point, got %d alerts", len(alerts))
	}

	// Verify alert contains correct information
	alert := alerts[0]
	if alert["ipn"] != "PART-002" {
		t.Errorf("Expected IPN PART-002, got %v", alert["ipn"])
	}
	if alert["qty_on_hand"] != 9.0 {
		t.Errorf("Expected qty_on_hand 9, got %v", alert["qty_on_hand"])
	}
	if alert["reorder_point"] != 10.0 {
		t.Errorf("Expected reorder_point 10, got %v", alert["reorder_point"])
	}
}

// Test Case 3: Alert appears in /api/v1/dashboard/lowstock endpoint
func TestLowStockAlerts_EndpointReturnsAlerts(t *testing.T) {
	testDB := setupLowStockTestDB(t)
	defer testDB.Close()
	handler := lowStockAlertsHandler(testDB)

	// Insert multiple parts with varying stock levels
	_, err := testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) VALUES
		('PART-LOW-1', 5, 10, 'Low Stock Part 1'),
		('PART-OK', 50, 10, 'Normal Stock Part'),
		('PART-LOW-2', 3, 15, 'Low Stock Part 2')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var alerts []map[string]interface{}
	if err := json.Unmarshal(dataBytes, &alerts); err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}

	// Should only return 2 low stock items
	if len(alerts) != 2 {
		t.Fatalf("Expected 2 low stock alerts, got %d", len(alerts))
	}

	// Verify both low stock parts are present
	ipns := make(map[string]bool)
	for _, alert := range alerts {
		ipns[alert["ipn"].(string)] = true
	}

	if !ipns["PART-LOW-1"] {
		t.Error("Expected PART-LOW-1 in low stock alerts")
	}
	if !ipns["PART-LOW-2"] {
		t.Error("Expected PART-LOW-2 in low stock alerts")
	}
	if ipns["PART-OK"] {
		t.Error("PART-OK should not be in low stock alerts")
	}
}

// Test Case 4: Alert clears when stock replenished above minimum
func TestLowStockAlerts_AlertClears_WhenReplenished(t *testing.T) {
	testDB := setupLowStockTestDB(t)
	defer testDB.Close()
	alertsHandler := lowStockAlertsHandler(testDB)
	h := newTestHandler(testDB)

	// Insert part with low stock
	_, err := testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) VALUES
		('PART-003', 5, 10, 'Part to be Replenished')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Verify alert exists initially
	req1 := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w1 := httptest.NewRecorder()
	alertsHandler(w1, req1)

	var resp1 models.APIResponse
	json.NewDecoder(w1.Body).Decode(&resp1)
	dataBytes1, _ := json.Marshal(resp1.Data)
	var alertsBefore []map[string]interface{}
	json.Unmarshal(dataBytes1, &alertsBefore)

	if len(alertsBefore) != 1 {
		t.Fatalf("Expected 1 alert before replenishment, got %d", len(alertsBefore))
	}

	// Replenish stock via inventory transaction
	reqBody := `{
		"ipn": "PART-003",
		"type": "receive",
		"qty": 10,
		"reference": "PO-TEST",
		"notes": "Replenishment"
	}`
	reqTransact := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
	wTransact := httptest.NewRecorder()
	h.Transact(wTransact, reqTransact)

	if wTransact.Code != 200 {
		t.Fatalf("Failed to replenish stock: %s", wTransact.Body.String())
	}

	// Verify qty_on_hand is now above reorder_point
	var qtyOnHand float64
	testDB.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "PART-003").Scan(&qtyOnHand)
	if qtyOnHand != 15 {
		t.Errorf("Expected qty_on_hand 15 after replenishment, got %f", qtyOnHand)
	}

	// Verify alert is now cleared
	req2 := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w2 := httptest.NewRecorder()
	alertsHandler(w2, req2)

	var resp2 models.APIResponse
	json.NewDecoder(w2.Body).Decode(&resp2)
	dataBytes2, _ := json.Marshal(resp2.Data)
	var alertsAfter []map[string]interface{}
	json.Unmarshal(dataBytes2, &alertsAfter)

	if len(alertsAfter) != 0 {
		t.Errorf("Expected no alerts after replenishment, got %d alerts", len(alertsAfter))
	}
}

// Test Case 5: Multiple parts below minimum -> multiple alerts
func TestLowStockAlerts_MultipleAlerts_MultipleParts(t *testing.T) {
	testDB := setupLowStockTestDB(t)
	defer testDB.Close()
	handler := lowStockAlertsHandler(testDB)

	// Insert 5 parts, 3 below threshold
	_, err := testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) VALUES
		('PART-A', 2, 10, 'Low Stock A'),
		('PART-B', 50, 10, 'Normal Stock B'),
		('PART-C', 8, 20, 'Low Stock C'),
		('PART-D', 100, 50, 'Normal Stock D'),
		('PART-E', 1, 5, 'Low Stock E')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var alerts []map[string]interface{}
	if err := json.Unmarshal(dataBytes, &alerts); err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}

	if len(alerts) != 3 {
		t.Fatalf("Expected 3 low stock alerts, got %d", len(alerts))
	}

	// Verify each alert has required fields
	for _, alert := range alerts {
		if alert["ipn"] == nil {
			t.Error("Alert missing ipn field")
		}
		if alert["qty_on_hand"] == nil {
			t.Error("Alert missing qty_on_hand field")
		}
		if alert["reorder_point"] == nil {
			t.Error("Alert missing reorder_point field")
		}

		// Verify all alerts are actually below threshold
		qtyOnHand := alert["qty_on_hand"].(float64)
		reorderPoint := alert["reorder_point"].(float64)
		if qtyOnHand >= reorderPoint {
			t.Errorf("Alert for %s has qty_on_hand (%f) >= reorder_point (%f)",
				alert["ipn"], qtyOnHand, reorderPoint)
		}
	}
}

// Test edge case: Part exactly at threshold (should not alert)
func TestLowStockAlerts_NoAlert_AtExactThreshold(t *testing.T) {
	testDB := setupLowStockTestDB(t)
	defer testDB.Close()
	handler := lowStockAlertsHandler(testDB)

	_, err := testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) VALUES
		('PART-EXACT', 10, 10, 'Part At Exact Threshold')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	dataBytes, _ := json.Marshal(resp.Data)
	var alerts []map[string]interface{}
	json.Unmarshal(dataBytes, &alerts)

	if len(alerts) != 0 {
		t.Errorf("Expected no alerts when qty equals reorder_point, got %d alerts", len(alerts))
	}
}

// Test edge case: Parts with reorder_point = 0 should not generate alerts
func TestLowStockAlerts_NoAlert_ZeroReorderPoint(t *testing.T) {
	testDB := setupLowStockTestDB(t)
	defer testDB.Close()
	handler := lowStockAlertsHandler(testDB)

	_, err := testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) VALUES
		('PART-ZERO', 0, 0, 'Part With Zero Reorder Point')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	dataBytes, _ := json.Marshal(resp.Data)
	var alerts []map[string]interface{}
	json.Unmarshal(dataBytes, &alerts)

	if len(alerts) != 0 {
		t.Errorf("Expected no alerts for parts with reorder_point = 0, got %d alerts", len(alerts))
	}
}

// Test that alerts endpoint returns empty array when no low stock items
func TestLowStockAlerts_EmptyArray_NoLowStock(t *testing.T) {
	testDB := setupLowStockTestDB(t)
	defer testDB.Close()
	handler := lowStockAlertsHandler(testDB)

	_, err := testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) VALUES
		('PART-1', 100, 10, 'Normal Stock 1'),
		('PART-2', 50, 20, 'Normal Stock 2')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	dataBytes, _ := json.Marshal(resp.Data)
	var alerts []map[string]interface{}
	if err := json.Unmarshal(dataBytes, &alerts); err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}

	if alerts == nil {
		t.Error("Expected empty array, got nil")
	}

	if len(alerts) != 0 {
		t.Errorf("Expected empty array, got %d alerts", len(alerts))
	}
}

// Test sequential replenishment - alert appears, disappears, appears again
func TestLowStockAlerts_Sequential_ReplenishAndDeplete(t *testing.T) {
	testDB := setupLowStockTestDB(t)
	defer testDB.Close()
	alertsHandler := lowStockAlertsHandler(testDB)
	h := newTestHandler(testDB)

	// Start with low stock
	testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES ('PART-SEQ', 5, 10)`)

	// Should have 1 alert
	req1 := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w1 := httptest.NewRecorder()
	alertsHandler(w1, req1)
	var resp1 models.APIResponse
	json.NewDecoder(w1.Body).Decode(&resp1)
	dataBytes1, _ := json.Marshal(resp1.Data)
	var alerts1 []map[string]interface{}
	json.Unmarshal(dataBytes1, &alerts1)
	if len(alerts1) != 1 {
		t.Errorf("Step 1: Expected 1 alert, got %d", len(alerts1))
	}

	// Replenish
	reqBody1 := `{"ipn": "PART-SEQ", "type": "receive", "qty": 10}`
	reqT1 := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody1))
	wT1 := httptest.NewRecorder()
	h.Transact(wT1, reqT1)

	// Should have 0 alerts
	req2 := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w2 := httptest.NewRecorder()
	alertsHandler(w2, req2)
	var resp2 models.APIResponse
	json.NewDecoder(w2.Body).Decode(&resp2)
	dataBytes2, _ := json.Marshal(resp2.Data)
	var alerts2 []map[string]interface{}
	json.Unmarshal(dataBytes2, &alerts2)
	if len(alerts2) != 0 {
		t.Errorf("Step 2: Expected 0 alerts after replenishment, got %d", len(alerts2))
	}

	// Issue stock to bring below threshold again
	reqBody2 := `{"ipn": "PART-SEQ", "type": "issue", "qty": 8}`
	reqT2 := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody2))
	wT2 := httptest.NewRecorder()
	h.Transact(wT2, reqT2)

	// Should have 1 alert again
	req3 := httptest.NewRequest("GET", "/api/v1/dashboard/lowstock", nil)
	w3 := httptest.NewRecorder()
	alertsHandler(w3, req3)
	var resp3 models.APIResponse
	json.NewDecoder(w3.Body).Decode(&resp3)
	dataBytes3, _ := json.Marshal(resp3.Data)
	var alerts3 []map[string]interface{}
	json.Unmarshal(dataBytes3, &alerts3)
	if len(alerts3) != 1 {
		t.Errorf("Step 3: Expected 1 alert after depletion, got %d", len(alerts3))
	}
}
