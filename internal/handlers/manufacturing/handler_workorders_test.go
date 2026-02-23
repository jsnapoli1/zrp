package manufacturing_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zrp/internal/handlers/manufacturing"
	"zrp/internal/models"
	"zrp/internal/testutil"

	_ "modernc.org/sqlite"
)

func setupWorkOrderTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Create tables (simplified for testing)
	tables := []string{
		`CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL DEFAULT 1,
			qty_good INTEGER,
			qty_scrap INTEGER,
			status TEXT NOT NULL DEFAULT 'open',
			priority TEXT NOT NULL DEFAULT 'normal',
			notes TEXT,
			created_at TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT
		)`,
		`CREATE TABLE wo_serials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			wo_id TEXT NOT NULL,
			serial_number TEXT UNIQUE NOT NULL,
			status TEXT NOT NULL DEFAULT 'assigned',
			notes TEXT
		)`,
		`CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL NOT NULL DEFAULT 0,
			qty_reserved REAL NOT NULL DEFAULT 0,
			location TEXT,
			reorder_point REAL DEFAULT 0,
			reorder_qty REAL DEFAULT 0,
			description TEXT,
			mpn TEXT,
			updated_at TEXT
		)`,
		`CREATE TABLE inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			type TEXT NOT NULL,
			qty REAL NOT NULL,
			reference TEXT,
			notes TEXT,
			created_at TEXT NOT NULL
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

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			t.Fatal(err)
		}
	}

	return db
}

func newTestHandler(db *sql.DB) *manufacturing.Handler {
	return &manufacturing.Handler{
		DB:  db,
		Hub: nil,
		NextIDFunc: func(prefix, table string, digits int) string {
			return prefix + "0001"
		},
		RecordChangeJSON: func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error) {
			return 0, nil
		},
		CreateUndoEntry: func(username, action, entityType, entityID string) (int64, error) {
			return 0, nil
		},
		GetWorkOrderSnapshot: func(id string) (map[string]interface{}, error) {
			return nil, nil
		},
	}
}

func TestWorkOrderKit(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test data
	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at) VALUES ('WO001', 'ASY-001', 5, 'open', '2026-01-01 00:00:00')`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES
		('PART-001', 10.0, 0.0),
		('PART-002', 3.0, 0.0),
		('PART-003', 0.0, 0.0)`)
	if err != nil {
		t.Fatal(err)
	}

	// Test kitting materials
	req, err := http.NewRequest("POST", "/api/v1/workorders/WO001/kit", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h.WorkOrderKit(rr, req, "WO001")

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
	}

	var envelope struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	result := envelope.Data

	if result["wo_id"] != "WO001" {
		t.Errorf("Expected wo_id WO001, got %v", result["wo_id"])
	}

	if result["status"] != "kitted" {
		t.Errorf("Expected status kitted, got %v", result["status"])
	}

	// Verify inventory was reserved
	var reserved float64
	err = testDB.QueryRow("SELECT qty_reserved FROM inventory WHERE ipn = 'PART-001'").Scan(&reserved)
	if err != nil {
		t.Fatal(err)
	}
	if reserved != 5.0 {
		t.Errorf("Expected 5.0 reserved for PART-001, got %f", reserved)
	}
}

func TestWorkOrderSerials(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test data
	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at) VALUES ('WO002', 'ASY-002', 2, 'in_progress', '2026-01-01 00:00:00')`)
	if err != nil {
		t.Fatal(err)
	}

	// Test adding a serial number
	serial := models.WOSerial{
		SerialNumber: "TEST001",
		Status:       "building", // Must match wo_serials schema CHECK constraint
		Notes:        "Test serial",
	}

	jsonData, err := json.Marshal(serial)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/api/v1/workorders/WO002/serials", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h.WorkOrderAddSerial(rr, req, "WO002")

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
	}

	// Extract data field from API response
	var apiResp struct {
		Data models.WOSerial `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
		t.Fatal(err)
	}
	result := apiResp.Data

	if result.SerialNumber != "TEST001" {
		t.Errorf("Expected serial TEST001, got %s", result.SerialNumber)
	}

	if result.WOID != "WO002" {
		t.Errorf("Expected wo_id WO002, got %s", result.WOID)
	}

	// Test getting serials
	req2, err := http.NewRequest("GET", "/api/v1/workorders/WO002/serials", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr2 := httptest.NewRecorder()
	h.WorkOrderSerials(rr2, req2, "WO002")

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Extract data field from API response
	var apiResp2 struct {
		Data []models.WOSerial `json:"data"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &apiResp2); err != nil {
		t.Fatal(err)
	}
	serials := apiResp2.Data

	if len(serials) != 1 {
		t.Errorf("Expected 1 serial, got %d", len(serials))
	}

	if serials[0].SerialNumber != "TEST001" {
		t.Errorf("Expected serial TEST001, got %s", serials[0].SerialNumber)
	}
}

func TestWorkOrderStatusTransitions(t *testing.T) {
	tests := []struct {
		from     string
		to       string
		expected bool
	}{
		{"draft", "open", true},
		{"draft", "cancelled", true},
		{"draft", "completed", false}, // Invalid transition
		{"open", "in_progress", true},
		{"open", "on_hold", true},
		{"open", "cancelled", true},
		{"in_progress", "completed", true},
		{"in_progress", "on_hold", true},
		{"completed", "open", false}, // Terminal state
		{"cancelled", "open", false}, // Terminal state
	}

	for _, tt := range tests {
		result := manufacturing.IsValidStatusTransition(tt.from, tt.to)
		if result != tt.expected {
			t.Errorf("IsValidStatusTransition(%s, %s) = %v, want %v", tt.from, tt.to, result, tt.expected)
		}
	}
}

func TestWorkOrderCompletion(t *testing.T) {
	testDB := setupWorkOrderTestDB(t)
	defer testDB.Close()

	// Insert test data
	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at) VALUES ('WO003', 'ASY-003', 2, 'in_progress', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES
		('ASY-003', 0.0, 0.0),
		('PART-004', 10.0, 4.0)`)
	if err != nil {
		t.Fatal(err)
	}

	// Start transaction for testing
	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	// Test work order completion
	err = manufacturing.HandleWorkOrderCompletion(tx, "WO003", "ASY-003", 2, "testuser")
	if err != nil {
		t.Fatalf("HandleWorkOrderCompletion failed: %v", err)
	}

	// Verify finished goods were added
	var assemblyQty float64
	err = tx.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = 'ASY-003'").Scan(&assemblyQty)
	if err != nil {
		t.Fatal(err)
	}
	if assemblyQty != 2.0 {
		t.Errorf("Expected 2.0 finished goods, got %f", assemblyQty)
	}

	// Verify materials were consumed
	var partQtyOnHand, partQtyReserved float64
	err = tx.QueryRow("SELECT qty_on_hand, qty_reserved FROM inventory WHERE ipn = 'PART-004'").Scan(&partQtyOnHand, &partQtyReserved)
	if err != nil {
		t.Fatal(err)
	}
	if partQtyOnHand != 6.0 { // 10 - 4 (reserved consumed) = 6
		t.Errorf("Expected 6.0 remaining on hand, got %f", partQtyOnHand)
	}
	if partQtyReserved != 0.0 {
		t.Errorf("Expected 0.0 reserved after completion, got %f", partQtyReserved)
	}
}

func TestGenerateSerialNumber(t *testing.T) {
	tests := []struct {
		assemblyIPN string
		prefix      string
	}{
		{"ASY-001", "ASY"},
		{"PCA-MAIN-V1.0", "PCA"},
		{"X", "X"},
	}

	for _, tt := range tests {
		serial := manufacturing.GenerateSerialNumber(tt.assemblyIPN)
		if !bytes.HasPrefix([]byte(serial), []byte(tt.prefix)) {
			t.Errorf("GenerateSerialNumber(%s) = %s, expected to start with %s", tt.assemblyIPN, serial, tt.prefix)
		}
		if len(serial) < len(tt.prefix)+12 { // prefix + timestamp
			t.Errorf("GenerateSerialNumber(%s) = %s, too short", tt.assemblyIPN, serial)
		}
	}
}
