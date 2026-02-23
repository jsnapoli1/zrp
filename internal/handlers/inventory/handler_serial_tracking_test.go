package inventory_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"zrp/internal/handlers/manufacturing"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

// setupSerialTestDB creates an in-memory test database with required tables
func setupSerialTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	// Enable foreign key constraints
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}

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
			due_date TEXT DEFAULT '',
			created_at TEXT NOT NULL,
			started_at TEXT,
			completed_at TEXT
		)`,
		`CREATE TABLE wo_serials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			wo_id TEXT NOT NULL,
			serial_number TEXT NOT NULL UNIQUE,
			status TEXT DEFAULT 'building' CHECK(status IN ('building','testing','complete','failed','scrapped')),
			notes TEXT,
			FOREIGN KEY (wo_id) REFERENCES work_orders(id) ON DELETE CASCADE
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
		`CREATE TABLE bom (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_ipn TEXT NOT NULL,
			child_ipn TEXT NOT NULL,
			quantity REAL NOT NULL DEFAULT 1,
			reference_designator TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			UNIQUE(parent_ipn, child_ipn)
		)`,
		`CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			category TEXT DEFAULT '',
			description TEXT DEFAULT '',
			mpn TEXT DEFAULT '',
			manufacturer TEXT DEFAULT '',
			lifecycle TEXT DEFAULT 'active',
			status TEXT DEFAULT 'active',
			notes TEXT DEFAULT '',
			fields TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, table := range tables {
		if _, err := testDB.Exec(table); err != nil {
			t.Fatal(err)
		}
	}

	return testDB
}

func newSerialTestMfgHandler(db *sql.DB) *manufacturing.Handler {
	return &manufacturing.Handler{
		DB:           db,
		Hub:          nil,
		PartsDir:     "",
		CompanyName:  "Test",
		GetPartByIPN: nil,
		NextIDFunc: func(prefix, table string, digits int) string {
			return prefix + "-999"
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
		EmailOnOverdueWorkOrder: nil,
	}
}

// TestSerialNumberAutoGeneration tests that serial numbers are automatically generated
// when not provided and follow the expected format
func TestSerialNumberAutoGeneration(t *testing.T) {
	testDB := setupSerialTestDB(t)
	defer testDB.Close()
	mh := newSerialTestMfgHandler(testDB)

	// Create a work order
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
		VALUES ('WO001', 'ASY-MAIN-V1', 5, 'in_progress', ?)`, now)
	if err != nil {
		t.Fatal(err)
	}

	// Test auto-generation by sending empty serial number
	serial := models.WOSerial{
		Status: "building",
		Notes:  "Auto-generated serial test",
	}

	jsonData, err := json.Marshal(serial)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/api/v1/workorders/WO001/serials", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mh.WorkOrderAddSerial(rr, req, "WO001")

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
			status, http.StatusOK, rr.Body.String())
	}

	var apiResp struct {
		Data models.WOSerial `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
		t.Fatal(err)
	}
	result := apiResp.Data

	// Verify serial number was auto-generated
	if result.SerialNumber == "" {
		t.Error("Serial number was not auto-generated")
	}

	// Verify serial format starts with assembly prefix
	expectedPrefix := "ASY"
	if !strings.HasPrefix(result.SerialNumber, expectedPrefix) {
		t.Errorf("Serial number %s does not start with expected prefix %s",
			result.SerialNumber, expectedPrefix)
	}

	// Verify serial has timestamp component (should be at least 15 chars: prefix + timestamp)
	if len(result.SerialNumber) < 15 {
		t.Errorf("Serial number %s is too short, expected at least 15 chars", result.SerialNumber)
	}

	// Verify serial links back to work order
	if result.WOID != "WO001" {
		t.Errorf("Serial wo_id = %s, want WO001", result.WOID)
	}
}

// TestSerialNumberFormat tests that serial numbers follow the expected pattern
func TestSerialNumberFormat(t *testing.T) {
	tests := []struct {
		assemblyIPN    string
		expectedPrefix string
	}{
		{"ASY-001", "ASY"},
		{"PCA-MAIN-V1.0", "PCA"},
		{"X-TEST", "X"},
		{"BOARD-123", "BOA"}, // Should truncate to 3 chars
	}

	for _, tt := range tests {
		serial := manufacturing.GenerateSerialNumber(tt.assemblyIPN)

		if !strings.HasPrefix(serial, tt.expectedPrefix) {
			t.Errorf("GenerateSerialNumber(%s) = %s, expected prefix %s",
				tt.assemblyIPN, serial, tt.expectedPrefix)
		}

		// Verify format: prefix + YYMMDDHHMMSS (at least 12 digits for timestamp)
		if len(serial) < len(tt.expectedPrefix)+12 {
			t.Errorf("GenerateSerialNumber(%s) = %s, too short", tt.assemblyIPN, serial)
		}
	}
}

// TestSerialTraceability tests that we can find all serials produced from a work order
func TestSerialTraceability(t *testing.T) {
	testDB := setupSerialTestDB(t)
	defer testDB.Close()
	mh := newSerialTestMfgHandler(testDB)

	now := time.Now().Format("2006-01-02 15:04:05")

	// Create work order
	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
		VALUES ('WO002', 'ASY-TEST-V2', 3, 'in_progress', ?)`, now)
	if err != nil {
		t.Fatal(err)
	}

	// Add multiple serial numbers to the work order
	serialNumbers := []string{"TEST-SN-001", "TEST-SN-002", "TEST-SN-003"}
	for _, sn := range serialNumbers {
		serial := models.WOSerial{
			SerialNumber: sn,
			Status:       "building",
			Notes:        "Test serial for traceability",
		}

		jsonData, _ := json.Marshal(serial)
		req, _ := http.NewRequest("POST", "/api/v1/workorders/WO002/serials", bytes.NewBuffer(jsonData))
		rr := httptest.NewRecorder()
		mh.WorkOrderAddSerial(rr, req, "WO002")

		if rr.Code != http.StatusOK {
			t.Fatalf("Failed to add serial %s: %s", sn, rr.Body.String())
		}
	}

	// Test forward traceability: get all serials for a work order
	req, err := http.NewRequest("GET", "/api/v1/workorders/WO002/serials", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mh.WorkOrderSerials(rr, req, "WO002")

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	var apiResp struct {
		Data []models.WOSerial `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &apiResp); err != nil {
		t.Fatal(err)
	}
	serials := apiResp.Data

	// Verify we got all 3 serials
	if len(serials) != 3 {
		t.Errorf("Expected 3 serials, got %d", len(serials))
	}

	// Verify all serials link back to WO002
	for _, serial := range serials {
		if serial.WOID != "WO002" {
			t.Errorf("Serial %s has wo_id = %s, want WO002", serial.SerialNumber, serial.WOID)
		}
	}

	// Verify serial numbers match what we created
	foundSerials := make(map[string]bool)
	for _, serial := range serials {
		foundSerials[serial.SerialNumber] = true
	}

	for _, expectedSN := range serialNumbers {
		if !foundSerials[expectedSN] {
			t.Errorf("Serial number %s not found in results", expectedSN)
		}
	}
}

// TestReverseSerialTraceability tests that we can find the work order that produced a serial
func TestReverseSerialTraceability(t *testing.T) {
	testDB := setupSerialTestDB(t)
	defer testDB.Close()

	now := time.Now().Format("2006-01-02 15:04:05")

	// Create multiple work orders
	workOrders := []struct {
		id          string
		assemblyIPN string
		serials     []string
	}{
		{"WO003", "ASY-A", []string{"SN-A-001", "SN-A-002"}},
		{"WO004", "ASY-B", []string{"SN-B-001", "SN-B-002"}},
		{"WO005", "ASY-C", []string{"SN-C-001"}},
	}

	for _, wo := range workOrders {
		_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
			VALUES (?, ?, ?, 'in_progress', ?)`, wo.id, wo.assemblyIPN, len(wo.serials), now)
		if err != nil {
			t.Fatal(err)
		}

		for _, sn := range wo.serials {
			_, err := testDB.Exec(`INSERT INTO wo_serials (wo_id, serial_number, status)
				VALUES (?, ?, 'building')`, wo.id, sn)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Test reverse traceability: find work order for each serial
	testCases := []struct {
		serialNumber    string
		expectedWOID    string
		expectedAssyIPN string
	}{
		{"SN-A-001", "WO003", "ASY-A"},
		{"SN-B-002", "WO004", "ASY-B"},
		{"SN-C-001", "WO005", "ASY-C"},
	}

	for _, tc := range testCases {
		// Query to get work order from serial number
		var woID, assyIPN string
		err := testDB.QueryRow(`
			SELECT wo.id, wo.assembly_ipn
			FROM wo_serials ws
			JOIN work_orders wo ON ws.wo_id = wo.id
			WHERE ws.serial_number = ?`, tc.serialNumber).Scan(&woID, &assyIPN)

		if err != nil {
			t.Errorf("Failed to find work order for serial %s: %v", tc.serialNumber, err)
			continue
		}

		if woID != tc.expectedWOID {
			t.Errorf("Serial %s linked to WO %s, expected %s", tc.serialNumber, woID, tc.expectedWOID)
		}

		if assyIPN != tc.expectedAssyIPN {
			t.Errorf("Serial %s linked to assembly %s, expected %s",
				tc.serialNumber, assyIPN, tc.expectedAssyIPN)
		}
	}
}

// TestDuplicateSerialNumberRejection tests that duplicate serial numbers are rejected
func TestDuplicateSerialNumberRejection(t *testing.T) {
	testDB := setupSerialTestDB(t)
	defer testDB.Close()
	mh := newSerialTestMfgHandler(testDB)

	now := time.Now().Format("2006-01-02 15:04:05")

	// Create two work orders
	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
		VALUES ('WO006', 'ASY-DUP', 2, 'in_progress', ?)`, now)
	if err != nil {
		t.Fatal(err)
	}

	_, err = testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
		VALUES ('WO007', 'ASY-DUP2', 2, 'in_progress', ?)`, now)
	if err != nil {
		t.Fatal(err)
	}

	// Add a serial to the first work order
	serial1 := models.WOSerial{
		SerialNumber: "DUPLICATE-SN-001",
		Status:       "building",
	}

	jsonData1, _ := json.Marshal(serial1)
	req1, _ := http.NewRequest("POST", "/api/v1/workorders/WO006/serials", bytes.NewBuffer(jsonData1))
	rr1 := httptest.NewRecorder()
	mh.WorkOrderAddSerial(rr1, req1, "WO006")

	if rr1.Code != http.StatusOK {
		t.Fatalf("Failed to add first serial: %s", rr1.Body.String())
	}

	// Try to add the same serial to another work order (should fail)
	serial2 := models.WOSerial{
		SerialNumber: "DUPLICATE-SN-001",
		Status:       "building",
	}

	jsonData2, _ := json.Marshal(serial2)
	req2, _ := http.NewRequest("POST", "/api/v1/workorders/WO007/serials", bytes.NewBuffer(jsonData2))
	rr2 := httptest.NewRecorder()
	mh.WorkOrderAddSerial(rr2, req2, "WO007")

	if rr2.Code == http.StatusOK {
		t.Error("Expected duplicate serial to be rejected, but it was accepted")
	}

	if rr2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for duplicate serial, got %d", rr2.Code)
	}

	// Verify error message mentions duplicate
	body := rr2.Body.String()
	if !strings.Contains(strings.ToLower(body), "serial") &&
		!strings.Contains(strings.ToLower(body), "exists") {
		t.Errorf("Error message should mention serial already exists: %s", body)
	}
}

// TestSerialStatusTransitions tests serial status workflow
func TestSerialStatusTransitions(t *testing.T) {
	testDB := setupSerialTestDB(t)
	defer testDB.Close()

	now := time.Now().Format("2006-01-02 15:04:05")

	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
		VALUES ('WO008', 'ASY-STATUS', 1, 'in_progress', ?)`, now)
	if err != nil {
		t.Fatal(err)
	}

	// Create serial in building status
	_, err = testDB.Exec(`INSERT INTO wo_serials (wo_id, serial_number, status)
		VALUES ('WO008', 'SN-STATUS-001', 'building')`)
	if err != nil {
		t.Fatal(err)
	}

	// Test valid status transitions
	validTransitions := []string{"testing", "complete"}

	for _, newStatus := range validTransitions {
		_, err := testDB.Exec(`UPDATE wo_serials SET status = ? WHERE serial_number = 'SN-STATUS-001'`, newStatus)
		if err != nil {
			t.Errorf("Failed to transition to status %s: %v", newStatus, err)
		}

		var currentStatus string
		testDB.QueryRow(`SELECT status FROM wo_serials WHERE serial_number = 'SN-STATUS-001'`).Scan(&currentStatus)
		if currentStatus != newStatus {
			t.Errorf("Status = %s, want %s", currentStatus, newStatus)
		}
	}

	// Test invalid status (should be rejected by CHECK constraint)
	_, err = testDB.Exec(`UPDATE wo_serials SET status = 'invalid_status' WHERE serial_number = 'SN-STATUS-001'`)
	if err == nil {
		t.Error("Expected invalid status to be rejected by CHECK constraint")
	}
}

// TestSerialWorkOrderCascadeDelete tests that serials are deleted when work order is deleted
func TestSerialWorkOrderCascadeDelete(t *testing.T) {
	testDB := setupSerialTestDB(t)
	defer testDB.Close()

	now := time.Now().Format("2006-01-02 15:04:05")

	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
		VALUES ('WO009', 'ASY-CASCADE', 2, 'in_progress', ?)`, now)
	if err != nil {
		t.Fatal(err)
	}

	// Add serials
	_, err = testDB.Exec(`INSERT INTO wo_serials (wo_id, serial_number, status)
		VALUES ('WO009', 'SN-CASCADE-001', 'building'), ('WO009', 'SN-CASCADE-002', 'building')`)
	if err != nil {
		t.Fatal(err)
	}

	// Verify serials exist
	var count int
	testDB.QueryRow(`SELECT COUNT(*) FROM wo_serials WHERE wo_id = 'WO009'`).Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 serials before delete, got %d", count)
	}

	// Delete work order
	_, err = testDB.Exec(`DELETE FROM work_orders WHERE id = 'WO009'`)
	if err != nil {
		t.Fatal(err)
	}

	// Verify serials were cascade deleted
	testDB.QueryRow(`SELECT COUNT(*) FROM wo_serials WHERE wo_id = 'WO009'`).Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 serials after cascade delete, got %d", count)
	}
}

// TestSerialNumberUniqueness tests that serial numbers are globally unique
func TestSerialNumberUniqueness(t *testing.T) {
	testDB := setupSerialTestDB(t)
	defer testDB.Close()

	now := time.Now().Format("2006-01-02 15:04:05")

	// Create two work orders
	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
		VALUES ('WO010', 'ASY-UNQ1', 1, 'in_progress', ?), ('WO011', 'ASY-UNQ2', 1, 'in_progress', ?)`,
		now, now)
	if err != nil {
		t.Fatal(err)
	}

	// Add serial to first work order
	_, err = testDB.Exec(`INSERT INTO wo_serials (wo_id, serial_number, status)
		VALUES ('WO010', 'UNIQUE-SN-001', 'building')`)
	if err != nil {
		t.Fatal(err)
	}

	// Try to add same serial to second work order (should fail due to UNIQUE constraint)
	_, err = testDB.Exec(`INSERT INTO wo_serials (wo_id, serial_number, status)
		VALUES ('WO011', 'UNIQUE-SN-001', 'building')`)
	if err == nil {
		t.Error("Expected UNIQUE constraint violation, but insert succeeded")
	}

	// Verify error is about uniqueness
	if !strings.Contains(err.Error(), "UNIQUE") && !strings.Contains(err.Error(), "unique") {
		t.Errorf("Expected UNIQUE constraint error, got: %v", err)
	}
}

// TestWorkOrderCompletionWithSerials tests that work orders track qty_good and serials
func TestWorkOrderCompletionWithSerials(t *testing.T) {
	testDB := setupSerialTestDB(t)
	defer testDB.Close()
	mh := newSerialTestMfgHandler(testDB)

	now := time.Now().Format("2006-01-02 15:04:05")

	// Create work order with qty=3
	_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
		VALUES ('WO012', 'ASY-COMPLETE', 3, 'in_progress', ?)`, now)
	if err != nil {
		t.Fatal(err)
	}

	// Add 3 serials with unique serial numbers
	for i := 1; i <= 3; i++ {
		sn := models.WOSerial{
			SerialNumber: "ASY-COMP-" + now + "-" + string(rune('0'+i)),
			Status:       "building",
		}

		jsonData, _ := json.Marshal(sn)
		req, _ := http.NewRequest("POST", "/api/v1/workorders/WO012/serials", bytes.NewBuffer(jsonData))
		rr := httptest.NewRecorder()
		mh.WorkOrderAddSerial(rr, req, "WO012")

		if rr.Code != http.StatusOK {
			t.Fatalf("Failed to add serial %d: %s", i, rr.Body.String())
		}
	}

	// Verify we have 3 serials
	var serialCount int
	testDB.QueryRow(`SELECT COUNT(*) FROM wo_serials WHERE wo_id = 'WO012'`).Scan(&serialCount)
	if serialCount != 3 {
		t.Errorf("Expected 3 serials, got %d", serialCount)
	}

	// Mark 2 as complete, 1 as failed
	// Get serial IDs
	rows, err := testDB.Query(`SELECT id FROM wo_serials WHERE wo_id = 'WO012' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	var serialIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		serialIDs = append(serialIDs, id)
	}
	rows.Close()

	if len(serialIDs) < 3 {
		t.Fatalf("Expected at least 3 serials, got %d", len(serialIDs))
	}

	// Mark first 2 as complete
	testDB.Exec(`UPDATE wo_serials SET status = 'complete' WHERE id = ?`, serialIDs[0])
	testDB.Exec(`UPDATE wo_serials SET status = 'complete' WHERE id = ?`, serialIDs[1])
	// Mark third as failed
	testDB.Exec(`UPDATE wo_serials SET status = 'failed' WHERE id = ?`, serialIDs[2])

	// Count good vs failed
	var goodCount, failedCount int
	testDB.QueryRow(`SELECT COUNT(*) FROM wo_serials WHERE wo_id = 'WO012' AND status = 'complete'`).Scan(&goodCount)
	testDB.QueryRow(`SELECT COUNT(*) FROM wo_serials WHERE wo_id = 'WO012' AND status = 'failed'`).Scan(&failedCount)

	if goodCount != 2 {
		t.Errorf("Expected 2 good serials, got %d", goodCount)
	}
	if failedCount != 1 {
		t.Errorf("Expected 1 failed serial, got %d", failedCount)
	}

	// Update work order with actual quantities
	_, err = testDB.Exec(`UPDATE work_orders SET qty_good = ?, qty_scrap = ?, status = 'complete',
		completed_at = ? WHERE id = 'WO012'`, goodCount, failedCount, now)
	if err != nil {
		t.Fatal(err)
	}

	// Verify work order reflects serial counts
	var qtyGood, qtyScrap sql.NullInt64
	err = testDB.QueryRow(`SELECT qty_good, qty_scrap FROM work_orders WHERE id = 'WO012'`).Scan(&qtyGood, &qtyScrap)
	if err != nil {
		t.Fatal(err)
	}

	if !qtyGood.Valid || qtyGood.Int64 != 2 {
		t.Errorf("Work order qty_good = %v, want 2", qtyGood)
	}
	if !qtyScrap.Valid || qtyScrap.Int64 != 1 {
		t.Errorf("Work order qty_scrap = %v, want 1", qtyScrap)
	}
}

// TestSerialSearchAndLookup tests various ways to query and find serials
func TestSerialSearchAndLookup(t *testing.T) {
	testDB := setupSerialTestDB(t)
	defer testDB.Close()

	now := time.Now().Format("2006-01-02 15:04:05")

	// Create multiple work orders with serials
	testData := []struct {
		woID    string
		assyIPN string
		serials []string
	}{
		{"WO013", "ASY-SEARCH-A", []string{"SEARCH-A-001", "SEARCH-A-002"}},
		{"WO014", "ASY-SEARCH-B", []string{"SEARCH-B-001"}},
		{"WO015", "ASY-SEARCH-A", []string{"SEARCH-A-003"}},
	}

	for _, td := range testData {
		_, err := testDB.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, created_at)
			VALUES (?, ?, ?, 'in_progress', ?)`, td.woID, td.assyIPN, len(td.serials), now)
		if err != nil {
			t.Fatal(err)
		}

		for _, sn := range td.serials {
			_, err := testDB.Exec(`INSERT INTO wo_serials (wo_id, serial_number, status)
				VALUES (?, ?, 'building')`, td.woID, sn)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Test 1: Find all serials for a specific assembly IPN across work orders
	rows, err := testDB.Query(`
		SELECT ws.serial_number, ws.wo_id
		FROM wo_serials ws
		JOIN work_orders wo ON ws.wo_id = wo.id
		WHERE wo.assembly_ipn = ?
		ORDER BY ws.serial_number`, "ASY-SEARCH-A")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var serialsForAssyA []string
	for rows.Next() {
		var sn, woID string
		rows.Scan(&sn, &woID)
		serialsForAssyA = append(serialsForAssyA, sn)
	}

	// Should find 3 serials for ASY-SEARCH-A (from WO013 and WO015)
	if len(serialsForAssyA) != 3 {
		t.Errorf("Expected 3 serials for ASY-SEARCH-A, got %d", len(serialsForAssyA))
	}

	// Test 2: Find serial by partial match
	var foundSerial string
	err = testDB.QueryRow(`SELECT serial_number FROM wo_serials WHERE serial_number LIKE ?`,
		"%SEARCH-B%").Scan(&foundSerial)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(foundSerial, "SEARCH-B") {
		t.Errorf("Expected serial to contain SEARCH-B, got %s", foundSerial)
	}

	// Test 3: Count serials by status
	var buildingCount int
	err = testDB.QueryRow(`SELECT COUNT(*) FROM wo_serials WHERE status = 'building'`).Scan(&buildingCount)
	if err != nil {
		t.Fatal(err)
	}

	if buildingCount != 4 {
		t.Errorf("Expected 4 serials in building status, got %d", buildingCount)
	}
}
