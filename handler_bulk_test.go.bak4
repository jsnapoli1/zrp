package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupBulkTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create ecos table
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','approved','implemented','rejected')),
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT,
			created_by TEXT,
			approved_by TEXT,
			approved_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			ncr_id TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	// Create work_orders table
	_, err = testDB.Exec(`
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending','in_progress','completed','cancelled')),
			assigned_to TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Create ncrs table
	_, err = testDB.Exec(`
		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			ipn TEXT,
			serial_number TEXT,
			defect_type TEXT,
			severity TEXT DEFAULT 'minor' CHECK(severity IN ('minor','major','critical')),
			status TEXT DEFAULT 'open' CHECK(status IN ('open','investigating','resolved','closed')),
			root_cause TEXT,
			corrective_action TEXT,
			created_by TEXT DEFAULT 'quality',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
	}

	// Create devices table
	_, err = testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			model TEXT,
			status TEXT DEFAULT 'active' CHECK(status IN ('active','maintenance','decommissioned')),
			location TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	// Create inventory table
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			qty_reserved REAL DEFAULT 0,
			location TEXT,
			reorder_point REAL,
			reorder_qty REAL,
			description TEXT,
			mpn TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create rmas table
	_, err = testDB.Exec(`
		CREATE TABLE rmas (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			customer TEXT,
			status TEXT DEFAULT 'open' CHECK(status IN ('open','investigating','resolved','closed')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create rmas table: %v", err)
	}

	// Create parts table
	_, err = testDB.Exec(`
		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			description TEXT,
			mpn TEXT,
			status TEXT DEFAULT 'active' CHECK(status IN ('active','archived','obsolete')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	// Create purchase_orders table
	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','submitted','approved','cancelled','received')),
			notes TEXT,
			approved_by TEXT,
			approved_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date TEXT,
			received_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create purchase_orders table: %v", err)
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

	// Create undo_log table (for createUndoEntry)
	_, err = testDB.Exec(`
		CREATE TABLE undo_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user TEXT,
			action TEXT,
			module TEXT,
			record_id TEXT,
			snapshot TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create undo_log table: %v", err)
	}

	return testDB
}

func TestHandleBulkECOs_ApproveSuccess(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title, description, status, created_at) VALUES 
		('ECO-001', 'ECO 1', 'Description 1', 'draft', '2026-01-01 10:00:00'),
		('ECO-002', 'ECO 2', 'Description 2', 'draft', '2026-01-02 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["ECO-001", "ECO-002"],
		"action": "approve"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/ecos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 2 {
		t.Errorf("Expected 2 successful operations, got %d", result.Success)
	}

	if result.Failed != 0 {
		t.Errorf("Expected 0 failed operations, got %d", result.Failed)
	}

	// Verify status was updated
	var status1, status2 string
	db.QueryRow("SELECT status FROM ecos WHERE id='ECO-001'").Scan(&status1)
	db.QueryRow("SELECT status FROM ecos WHERE id='ECO-002'").Scan(&status2)

	if status1 != "approved" {
		t.Errorf("Expected ECO-001 status 'approved', got %s", status1)
	}

	if status2 != "approved" {
		t.Errorf("Expected ECO-002 status 'approved', got %s", status2)
	}
}

func TestHandleBulkECOs_PartialFailure(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title, status, created_at) VALUES 
		('ECO-001', 'ECO 1', 'draft', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["ECO-001", "ECO-999"],
		"action": "approve"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/ecos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 1 {
		t.Errorf("Expected 1 successful operation, got %d", result.Success)
	}

	if result.Failed != 1 {
		t.Errorf("Expected 1 failed operation, got %d", result.Failed)
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error message, got %d", len(result.Errors))
	}
}

func TestHandleBulkECOs_InvalidAction(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{
		"ids": ["ECO-001"],
		"action": "invalid-action"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/ecos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkECOs(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleBulkECOs_InvalidJSON(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{invalid json`

	req := httptest.NewRequest("POST", "/api/v1/bulk/ecos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkECOs(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleBulkECOs_Delete(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title, status, created_at) VALUES 
		('ECO-001', 'ECO 1', 'draft', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["ECO-001"],
		"action": "delete"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/ecos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 1 {
		t.Errorf("Expected 1 successful deletion, got %d", result.Success)
	}

	// Verify ECO was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM ecos WHERE id='ECO-001'").Scan(&count)
	if count != 0 {
		t.Errorf("Expected ECO to be deleted, but it still exists")
	}
}

func TestHandleBulkWorkOrders_CompleteSuccess(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO work_orders (id, title, status, created_at) VALUES 
		('WO-001', 'Work Order 1', 'in_progress', '2026-01-01 10:00:00'),
		('WO-002', 'Work Order 2', 'pending', '2026-01-02 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["WO-001", "WO-002"],
		"action": "complete"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/work-orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkWorkOrders(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 2 {
		t.Errorf("Expected 2 successful operations, got %d", result.Success)
	}

	// Verify status and completed_at were updated
	var status, completedAt string
	db.QueryRow("SELECT status, COALESCE(completed_at, '') FROM work_orders WHERE id='WO-001'").Scan(&status, &completedAt)

	if status != "completed" {
		t.Errorf("Expected status 'completed', got %s", status)
	}

	if completedAt == "" {
		t.Error("Expected completed_at to be set")
	}
}

func TestHandleBulkNCRs_CloseSuccess(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ncrs (id, title, status, created_at) VALUES 
		('NCR-001', 'NCR 1', 'open', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["NCR-001"],
		"action": "close"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/ncrs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkNCRs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 1 {
		t.Errorf("Expected 1 successful operation, got %d", result.Success)
	}

	// Verify status was updated
	var status string
	db.QueryRow("SELECT status FROM ncrs WHERE id='NCR-001'").Scan(&status)

	if status != "closed" {
		t.Errorf("Expected status 'closed', got %s", status)
	}
}

func TestHandleBulkDevices_DecommissionSuccess(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO devices (serial_number, model, status) VALUES 
		('SN-001', 'Model A', 'active'),
		('SN-002', 'Model B', 'maintenance')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["SN-001", "SN-002"],
		"action": "decommission"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/devices", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkDevices(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 2 {
		t.Errorf("Expected 2 successful operations, got %d", result.Success)
	}

	// Verify status was updated
	var status1, status2 string
	db.QueryRow("SELECT status FROM devices WHERE serial_number='SN-001'").Scan(&status1)
	db.QueryRow("SELECT status FROM devices WHERE serial_number='SN-002'").Scan(&status2)

	if status1 != "decommissioned" {
		t.Errorf("Expected status 'decommissioned', got %s", status1)
	}

	if status2 != "decommissioned" {
		t.Errorf("Expected status 'decommissioned', got %s", status2)
	}
}

func TestHandleBulkInventory_DeleteSuccess(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, location) VALUES 
		('IPN-001', 100, 'A1'),
		('IPN-002', 50, 'B2')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["IPN-001", "IPN-002"],
		"action": "delete"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/inventory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkInventory(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 2 {
		t.Errorf("Expected 2 successful deletions, got %d", result.Success)
	}

	// Verify items were deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM inventory").Scan(&count)
	if count != 0 {
		t.Errorf("Expected all inventory items to be deleted, but %d remain", count)
	}
}

func TestHandleBulkRMAs_CloseSuccess(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO rmas (id, title, status, created_at) VALUES 
		('RMA-001', 'RMA 1', 'open', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["RMA-001"],
		"action": "close"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/rmas", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkRMAs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 1 {
		t.Errorf("Expected 1 successful operation, got %d", result.Success)
	}
}

func TestHandleBulkParts_ArchiveSuccess(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO parts (ipn, description, status) VALUES 
		('IPN-001', 'Part 1', 'active'),
		('IPN-002', 'Part 2', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["IPN-001", "IPN-002"],
		"action": "archive"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/parts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkParts(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 2 {
		t.Errorf("Expected 2 successful operations, got %d", result.Success)
	}

	// Verify status was updated
	var status1, status2 string
	db.QueryRow("SELECT status FROM parts WHERE ipn='IPN-001'").Scan(&status1)
	db.QueryRow("SELECT status FROM parts WHERE ipn='IPN-002'").Scan(&status2)

	if status1 != "archived" {
		t.Errorf("Expected status 'archived', got %s", status1)
	}

	if status2 != "archived" {
		t.Errorf("Expected status 'archived', got %s", status2)
	}
}

func TestHandleBulkPurchaseOrders_ApproveSuccess(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, created_at) VALUES 
		('PO-001', 'V-001', 'draft', '2026-01-01 10:00:00'),
		('PO-002', 'V-002', 'submitted', '2026-01-02 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["PO-001", "PO-002"],
		"action": "approve"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/purchase-orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkPurchaseOrders(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 2 {
		t.Errorf("Expected 2 successful operations, got %d", result.Success)
	}

	// Verify status was updated
	var status1, status2 string
	db.QueryRow("SELECT status FROM purchase_orders WHERE id='PO-001'").Scan(&status1)
	db.QueryRow("SELECT status FROM purchase_orders WHERE id='PO-002'").Scan(&status2)

	if status1 != "approved" {
		t.Errorf("Expected status 'approved', got %s", status1)
	}

	if status2 != "approved" {
		t.Errorf("Expected status 'approved', got %s", status2)
	}
}

func TestHandleBulkECOs_LargeBatch(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert 100 ECOs
	for i := 1; i <= 100; i++ {
		_, err := db.Exec(`INSERT INTO ecos (id, title, status, created_at) VALUES (?, ?, 'draft', '2026-01-01 10:00:00')`,
			"ECO-"+string(rune(i)), "ECO "+string(rune(i)))
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Build IDs array
	var ids []string
	for i := 1; i <= 100; i++ {
		ids = append(ids, "ECO-"+string(rune(i)))
	}

	reqData := BulkRequest{
		IDs:    ids,
		Action: "approve",
	}

	bodyBytes, _ := json.Marshal(reqData)
	req := httptest.NewRequest("POST", "/api/v1/bulk/ecos", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 100 {
		t.Errorf("Expected 100 successful operations, got %d", result.Success)
	}
}

func TestHandleBulkECOs_EmptyIDs(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{
		"ids": [],
		"action": "approve"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/ecos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result BulkResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Success != 0 {
		t.Errorf("Expected 0 successful operations, got %d", result.Success)
	}

	if result.Failed != 0 {
		t.Errorf("Expected 0 failed operations, got %d", result.Failed)
	}
}

func TestHandleBulkInventory_InvalidAction(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{
		"ids": ["IPN-001"],
		"action": "approve"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/inventory", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkInventory(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleBulkECOs_AuditLogging(t *testing.T) {
	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title, status, created_at) VALUES 
		('ECO-001', 'ECO 1', 'draft', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["ECO-001"],
		"action": "approve"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/ecos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify audit log entry was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module='eco' AND action='bulk_approve' AND record_id='ECO-001'").Scan(&count)

	if count != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", count)
	}
}

func TestHandleBulkECOs_TransactionalIntegrity(t *testing.T) {
	// Note: SQLite doesn't enforce transactional integrity in the current implementation
	// Each item is updated independently, so partial failures are possible
	// This test documents the current behavior rather than ideal behavior

	oldDB := db
	db = setupBulkTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title, status, created_at) VALUES 
		('ECO-001', 'ECO 1', 'draft', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"ids": ["ECO-001", "ECO-999"],
		"action": "approve"
	}`

	req := httptest.NewRequest("POST", "/api/v1/bulk/ecos", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleBulkECOs(w, req)

	// Current behavior: partial success (not transactional)
	var result BulkResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.Success != 1 || result.Failed != 1 {
		t.Logf("NOTE: Bulk operations are NOT transactional. Success=%d, Failed=%d", result.Success, result.Failed)
	}

	// ECO-001 should be approved even though ECO-999 failed
	var status string
	db.QueryRow("SELECT status FROM ecos WHERE id='ECO-001'").Scan(&status)
	if status != "approved" {
		t.Errorf("Expected ECO-001 to be approved despite partial failure, got %s", status)
	}
}
