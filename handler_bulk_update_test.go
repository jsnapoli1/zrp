package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupBulkUpdateTestDB(t *testing.T) func() {
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
			location TEXT,
			reorder_point REAL DEFAULT 0,
			reorder_qty REAL DEFAULT 0,
			qty_on_hand REAL DEFAULT 0,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create work_orders table
	_, err = testDB.Exec(`
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','open','in_progress','completed','cancelled','on_hold')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			due_date TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Create devices table
	_, err = testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			ipn TEXT,
			status TEXT DEFAULT 'active' CHECK(status IN ('active','inactive','decommissioned','rma')),
			customer TEXT,
			location TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	// Create parts table
	_, err = testDB.Exec(`
		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			title TEXT,
			category TEXT,
			status TEXT DEFAULT 'active',
			lifecycle TEXT,
			min_stock REAL DEFAULT 0,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	// Create ecos table
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','open','approved','implemented','rejected')),
			priority TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			action TEXT,
			module TEXT,
			record_id TEXT,
			summary TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	// Save and swap db
	origDB := db
	db = testDB

	return func() {
		db.Close()
		db = origDB
	}
}

func TestBulkUpdateInventoryLocation(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	// Create test inventory items
	db.Exec("INSERT INTO inventory (ipn, location) VALUES ('CAP-001', 'A1'), ('RES-001', 'A2')")

	body := `{"ids":["CAP-001","RES-001"],"updates":{"location":"B3"}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})

	if data["success"].(float64) != 2 {
		t.Errorf("expected 2 success, got %v", data["success"])
	}

	// Verify updates
	var loc1, loc2 string
	db.QueryRow("SELECT location FROM inventory WHERE ipn='CAP-001'").Scan(&loc1)
	db.QueryRow("SELECT location FROM inventory WHERE ipn='RES-001'").Scan(&loc2)

	if loc1 != "B3" || loc2 != "B3" {
		t.Errorf("expected both locations to be B3, got %s and %s", loc1, loc2)
	}
}

func TestBulkUpdateInventoryReorderPoint(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO inventory (ipn, reorder_point) VALUES ('CAP-001', 10)")

	body := `{"ids":["CAP-001"],"updates":{"reorder_point":"50","reorder_qty":"100"}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var rp, rq float64
	db.QueryRow("SELECT reorder_point, reorder_qty FROM inventory WHERE ipn='CAP-001'").Scan(&rp, &rq)

	if rp != 50 {
		t.Errorf("expected reorder_point 50, got %v", rp)
	}
	if rq != 100 {
		t.Errorf("expected reorder_qty 100, got %v", rq)
	}
}

func TestBulkUpdateInventoryDisallowedField(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO inventory (ipn) VALUES ('CAP-001')")

	body := `{"ids":["CAP-001"],"updates":{"qty_on_hand":"999"}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for disallowed field, got %d", w.Code)
	}
}

func TestBulkUpdateInventoryEmptyIDs(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	body := `{"ids":[],"updates":{"location":"X"}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBulkUpdateInventoryEmptyUpdates(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO inventory (ipn) VALUES ('CAP-001')")

	body := `{"ids":["CAP-001"],"updates":{}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBulkUpdateInventoryNotFound(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO inventory (ipn) VALUES ('CAP-001')")

	body := `{"ids":["CAP-001","NONEXISTENT"],"updates":{"location":"X"}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})

	if data["success"].(float64) != 1 {
		t.Errorf("expected 1 success, got %v", data["success"])
	}
	if data["failed"].(float64) != 1 {
		t.Errorf("expected 1 failed, got %v", data["failed"])
	}
}

func TestBulkUpdateWorkOrdersStatus(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	year := fmt.Sprintf("%d", time.Now().Year())
	id1 := fmt.Sprintf("WO-%s-001", year)
	id2 := fmt.Sprintf("WO-%s-002", year)
	
	db.Exec("INSERT INTO work_orders (id, status) VALUES (?, 'draft'), (?, 'open')", id1, id2)

	body := fmt.Sprintf(`{"ids":["%s","%s"],"updates":{"status":"completed"}}`, id1, id2)
	req := httptest.NewRequest("POST", "/api/v1/workorders/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateWorkOrders(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var status1, status2 string
	db.QueryRow("SELECT status FROM work_orders WHERE id=?", id1).Scan(&status1)
	db.QueryRow("SELECT status FROM work_orders WHERE id=?", id2).Scan(&status2)

	if status1 != "completed" || status2 != "completed" {
		t.Errorf("expected both statuses to be completed, got %s and %s", status1, status2)
	}
}

func TestBulkUpdateWorkOrdersPriority(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	year := fmt.Sprintf("%d", time.Now().Year())
	id := fmt.Sprintf("WO-%s-001", year)
	
	db.Exec("INSERT INTO work_orders (id, priority) VALUES (?, 'normal')", id)

	body := fmt.Sprintf(`{"ids":["%s"],"updates":{"priority":"critical"}}`, id)
	req := httptest.NewRequest("POST", "/api/v1/workorders/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateWorkOrders(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var priority string
	db.QueryRow("SELECT priority FROM work_orders WHERE id=?", id).Scan(&priority)

	if priority != "critical" {
		t.Errorf("expected priority critical, got %s", priority)
	}
}

func TestBulkUpdateWorkOrdersInvalidStatus(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	year := fmt.Sprintf("%d", time.Now().Year())
	id := fmt.Sprintf("WO-%s-001", year)
	db.Exec("INSERT INTO work_orders (id) VALUES (?)", id)

	body := fmt.Sprintf(`{"ids":["%s"],"updates":{"status":"invalid"}}`, id)
	req := httptest.NewRequest("POST", "/api/v1/workorders/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateWorkOrders(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid status, got %d", w.Code)
	}
}

func TestBulkUpdateWorkOrdersInvalidPriority(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	year := fmt.Sprintf("%d", time.Now().Year())
	id := fmt.Sprintf("WO-%s-001", year)
	db.Exec("INSERT INTO work_orders (id) VALUES (?)", id)

	body := fmt.Sprintf(`{"ids":["%s"],"updates":{"priority":"invalid"}}`, id)
	req := httptest.NewRequest("POST", "/api/v1/workorders/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateWorkOrders(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid priority, got %d", w.Code)
	}
}

func TestBulkUpdateWorkOrdersDisallowedField(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	year := fmt.Sprintf("%d", time.Now().Year())
	id := fmt.Sprintf("WO-%s-001", year)
	db.Exec("INSERT INTO work_orders (id) VALUES (?)", id)

	body := fmt.Sprintf(`{"ids":["%s"],"updates":{"assembly_ipn":"HACK"}}`, id)
	req := httptest.NewRequest("POST", "/api/v1/workorders/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateWorkOrders(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for disallowed field, got %d", w.Code)
	}
}

func TestBulkUpdateDevicesStatus(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO devices (serial_number, status) VALUES ('SN-001', 'active'), ('SN-002', 'active')")

	body := `{"ids":["SN-001","SN-002"],"updates":{"status":"inactive"}}`
	req := httptest.NewRequest("POST", "/api/v1/devices/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateDevices(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var status1, status2 string
	db.QueryRow("SELECT status FROM devices WHERE serial_number='SN-001'").Scan(&status1)
	db.QueryRow("SELECT status FROM devices WHERE serial_number='SN-002'").Scan(&status2)

	if status1 != "inactive" || status2 != "inactive" {
		t.Errorf("expected both statuses to be inactive, got %s and %s", status1, status2)
	}
}

func TestBulkUpdateDevicesCustomerAndLocation(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO devices (serial_number) VALUES ('SN-001')")

	body := `{"ids":["SN-001"],"updates":{"customer":"NewCorp","location":"Building C"}}`
	req := httptest.NewRequest("POST", "/api/v1/devices/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateDevices(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var cust, loc string
	db.QueryRow("SELECT customer, location FROM devices WHERE serial_number='SN-001'").Scan(&cust, &loc)

	if cust != "NewCorp" {
		t.Errorf("expected customer NewCorp, got %s", cust)
	}
	if loc != "Building C" {
		t.Errorf("expected location 'Building C', got %s", loc)
	}
}

func TestBulkUpdateDevicesInvalidStatus(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO devices (serial_number) VALUES ('SN-001')")

	body := `{"ids":["SN-001"],"updates":{"status":"invalid"}}`
	req := httptest.NewRequest("POST", "/api/v1/devices/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateDevices(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid status, got %d", w.Code)
	}
}

func TestBulkUpdateDevicesDisallowedField(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO devices (serial_number) VALUES ('SN-001')")

	body := `{"ids":["SN-001"],"updates":{"serial_number":"HACK"}}`
	req := httptest.NewRequest("POST", "/api/v1/devices/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateDevices(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for disallowed field, got %d", w.Code)
	}
}

func TestBulkUpdatePartsCategory(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO parts (ipn, category) VALUES ('P-001', 'resistor'), ('P-002', 'capacitor')")

	body := `{"ids":["P-001","P-002"],"updates":{"category":"passive"}}`
	req := httptest.NewRequest("POST", "/api/v1/parts/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateParts(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var cat1, cat2 string
	db.QueryRow("SELECT category FROM parts WHERE ipn='P-001'").Scan(&cat1)
	db.QueryRow("SELECT category FROM parts WHERE ipn='P-002'").Scan(&cat2)

	if cat1 != "passive" || cat2 != "passive" {
		t.Errorf("expected both categories to be passive, got %s and %s", cat1, cat2)
	}
}

func TestBulkUpdatePartsLifecycle(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO parts (ipn, lifecycle) VALUES ('P-001', 'active')")

	body := `{"ids":["P-001"],"updates":{"lifecycle":"obsolete","status":"inactive"}}`
	req := httptest.NewRequest("POST", "/api/v1/parts/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateParts(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var lifecycle, status string
	db.QueryRow("SELECT lifecycle, status FROM parts WHERE ipn='P-001'").Scan(&lifecycle, &status)

	if lifecycle != "obsolete" {
		t.Errorf("expected lifecycle obsolete, got %s", lifecycle)
	}
	if status != "inactive" {
		t.Errorf("expected status inactive, got %s", status)
	}
}

func TestBulkUpdatePartsMinStock(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO parts (ipn, min_stock) VALUES ('P-001', 0)")

	body := `{"ids":["P-001"],"updates":{"min_stock":"100"}}`
	req := httptest.NewRequest("POST", "/api/v1/parts/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateParts(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var minStock float64
	db.QueryRow("SELECT min_stock FROM parts WHERE ipn='P-001'").Scan(&minStock)

	if minStock != 100 {
		t.Errorf("expected min_stock 100, got %v", minStock)
	}
}

func TestBulkUpdatePartsDisallowedField(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO parts (ipn) VALUES ('P-001')")

	body := `{"ids":["P-001"],"updates":{"ipn":"HACK"}}`
	req := httptest.NewRequest("POST", "/api/v1/parts/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateParts(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for disallowed field, got %d", w.Code)
	}
}

func TestBulkUpdateECOsStatus(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO ecos (id, status) VALUES ('ECO-001', 'draft'), ('ECO-002', 'open')")

	body := `{"ids":["ECO-001","ECO-002"],"updates":{"status":"approved"}}`
	req := httptest.NewRequest("POST", "/api/v1/ecos/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateECOs(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var status1, status2 string
	db.QueryRow("SELECT status FROM ecos WHERE id='ECO-001'").Scan(&status1)
	db.QueryRow("SELECT status FROM ecos WHERE id='ECO-002'").Scan(&status2)

	if status1 != "approved" || status2 != "approved" {
		t.Errorf("expected both statuses to be approved, got %s and %s", status1, status2)
	}
}

func TestBulkUpdateECOsPriority(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO ecos (id, priority) VALUES ('ECO-001', 'low')")

	body := `{"ids":["ECO-001"],"updates":{"priority":"high"}}`
	req := httptest.NewRequest("POST", "/api/v1/ecos/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateECOs(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var priority string
	db.QueryRow("SELECT priority FROM ecos WHERE id='ECO-001'").Scan(&priority)

	if priority != "high" {
		t.Errorf("expected priority high, got %s", priority)
	}
}

func TestBulkUpdateECOsInvalidStatus(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO ecos (id) VALUES ('ECO-001')")

	body := `{"ids":["ECO-001"],"updates":{"status":"invalid"}}`
	req := httptest.NewRequest("POST", "/api/v1/ecos/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateECOs(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid status, got %d", w.Code)
	}
}

func TestBulkUpdateECOsDisallowedField(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO ecos (id) VALUES ('ECO-001')")

	body := `{"ids":["ECO-001"],"updates":{"title":"HACK"}}`
	req := httptest.NewRequest("POST", "/api/v1/ecos/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateECOs(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 for disallowed field, got %d", w.Code)
	}
}

func TestBulkUpdateAuditLogging(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO inventory (ipn) VALUES ('CAP-001')")

	body := `{"ids":["CAP-001"],"updates":{"location":"B3"}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	// Verify audit log entry was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE action='bulk_update' AND module='inventory'").Scan(&count)

	if count != 1 {
		t.Errorf("expected 1 audit log entry, got %d", count)
	}
}

func TestBulkUpdatePartialFailure(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	// Only create one of the three IPNs
	db.Exec("INSERT INTO inventory (ipn) VALUES ('CAP-001')")

	body := `{"ids":["CAP-001","MISSING-1","MISSING-2"],"updates":{"location":"B3"}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})

	if data["success"].(float64) != 1 {
		t.Errorf("expected 1 success, got %v", data["success"])
	}
	if data["failed"].(float64) != 2 {
		t.Errorf("expected 2 failed, got %v", data["failed"])
	}

	// Verify errors array contains the missing IDs
	errors := data["errors"].([]interface{})
	if len(errors) != 2 {
		t.Errorf("expected 2 error messages, got %d", len(errors))
	}
}

func TestBulkUpdateTransactionSafety(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO inventory (ipn, location) VALUES ('CAP-001', 'A1'), ('RES-001', 'A2')")

	// Each update should be independent - if one fails, others should still succeed
	body := `{"ids":["CAP-001","RES-001"],"updates":{"location":"B3"}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})

	// Both should succeed
	if data["success"].(float64) != 2 {
		t.Errorf("expected 2 success, got %v", data["success"])
	}
}

func TestBulkUpdateTimestamps(t *testing.T) {
	cleanup := setupBulkUpdateTestDB(t)
	defer cleanup()

	db.Exec("INSERT INTO inventory (ipn, updated_at) VALUES ('CAP-001', '2020-01-01 00:00:00')")

	body := `{"ids":["CAP-001"],"updates":{"location":"B3"}}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/bulk-update", strings.NewReader(body))
	req = withUsername(req, "admin")
	w := httptest.NewRecorder()
	
	handleBulkUpdateInventory(w, req)

	var updatedAt string
	db.QueryRow("SELECT updated_at FROM inventory WHERE ipn='CAP-001'").Scan(&updatedAt)

	// Should be updated to current timestamp
	if updatedAt == "2020-01-01 00:00:00" {
		t.Error("expected updated_at to be updated")
	}
}

// Helper functions
// Note: contextKey type is defined in middleware.go
const ctxUsernameBulk contextKey = "username"

func withUsername(req *http.Request, username string) *http.Request {
	ctx := context.WithValue(req.Context(), ctxUsernameBulk, username)
	return req.WithContext(ctx)
}
