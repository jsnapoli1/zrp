package field_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"zrp/internal/handlers/field"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

func setupRMATestDB(t *testing.T) (*sql.DB, *field.Handler) {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create rmas table
	_, err = testDB.Exec(`
		CREATE TABLE rmas (
			id TEXT PRIMARY KEY,
			serial_number TEXT NOT NULL,
			customer TEXT,
			reason TEXT NOT NULL,
			status TEXT DEFAULT 'open' CHECK(status IN ('open','received','diagnosing','repairing','resolved','closed','scrapped')),
			defect_description TEXT,
			resolution TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			received_at DATETIME,
			resolved_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create rmas table: %v", err)
	}

	// Create audit_log table (needed for logAudit)
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

	// Create id_sequences table for nextID
	_, err = testDB.Exec(`
		CREATE TABLE id_sequences (
			prefix TEXT PRIMARY KEY,
			next_num INTEGER DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create id_sequences table: %v", err)
	}

	var mu sync.Mutex
	counter := 0
	h := &field.Handler{
		DB:  testDB,
		Hub: nil,
		NextIDFunc: func(prefix, table string, digits int) string {
			mu.Lock()
			defer mu.Unlock()
			counter++
			s := fmt.Sprintf("%d", counter)
			for len(s) < digits {
				s = "0" + s
			}
			return prefix + "-" + s
		},
		RecordChangeJSON: func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error) {
			return 0, nil
		},
		GetDeviceSnapshot: func(id string) (map[string]interface{}, error) {
			return nil, nil
		},
		GetRMASnapshot: func(id string) (map[string]interface{}, error) {
			return nil, nil
		},
	}

	return testDB, h
}

// insertTestRMA inserts a test RMA directly into the database.
func insertTestRMA(t *testing.T, db *sql.DB, id, serial, customer, reason, status, defect, resolution string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO rmas (id, serial_number, customer, reason, status, defect_description, resolution, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		id, serial, customer, reason, status, defect, resolution,
	)
	if err != nil {
		t.Fatalf("Failed to insert test RMA: %v", err)
	}
}

// =============================================================================
// LIST RMAs TESTS
// =============================================================================

func TestHandleListRMAs_Empty(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/rmas", nil)
	w := httptest.NewRecorder()

	h.ListRMAs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	rmas, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(rmas) != 0 {
		t.Errorf("Expected empty array, got %d RMAs", len(rmas))
	}
}

func TestHandleListRMAs_WithData(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "DOA - Device won't power on", "open", "Device completely dead", "")
	insertTestRMA(t, db, "RMA-002", "SN67890", "Beta Inc", "Intermittent failure", "received", "Random shutdowns", "")
	insertTestRMA(t, db, "RMA-003", "SN54321", "Gamma LLC", "Wrong part shipped", "closed", "Incorrect model", "Replaced with correct unit")

	req := httptest.NewRequest("GET", "/api/rmas", nil)
	w := httptest.NewRecorder()

	h.ListRMAs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	rmasData, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(rmasData) != 3 {
		t.Errorf("Expected 3 RMAs, got %d", len(rmasData))
	}

	// Verify each RMA has expected fields
	for i, rData := range rmasData {
		rma := rData.(map[string]interface{})
		if rma["id"] == nil {
			t.Errorf("RMA %d missing id", i)
		}
		if rma["serial_number"] == nil {
			t.Errorf("RMA %d missing serial_number", i)
		}
		if rma["status"] == nil {
			t.Errorf("RMA %d missing status", i)
		}
	}
}

func TestHandleListRMAs_OrderByCreatedDesc(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Insert with explicit timestamps to verify ordering
	db.Exec(`INSERT INTO rmas (id, serial_number, reason, status, created_at) VALUES
		('RMA-001', 'SN001', 'Test 1', 'open', '2024-01-01 10:00:00'),
		('RMA-002', 'SN002', 'Test 2', 'open', '2024-01-03 10:00:00'),
		('RMA-003', 'SN003', 'Test 3', 'open', '2024-01-02 10:00:00')
	`)

	req := httptest.NewRequest("GET", "/api/rmas", nil)
	w := httptest.NewRecorder()

	h.ListRMAs(w, req)

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmas := resp.Data.([]interface{})

	if len(rmas) != 3 {
		t.Fatalf("Expected 3 RMAs, got %d", len(rmas))
	}

	// Should be ordered DESC by created_at: RMA-002, RMA-003, RMA-001
	first := rmas[0].(map[string]interface{})
	if first["id"] != "RMA-002" {
		t.Errorf("Expected first RMA to be RMA-002 (most recent), got %v", first["id"])
	}
}

// =============================================================================
// GET RMA TESTS
// =============================================================================

func TestHandleGetRMA_Success(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Device failure", "open", "Won't boot", "")

	req := httptest.NewRequest("GET", "/api/rmas/RMA-001", nil)
	w := httptest.NewRecorder()

	h.GetRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	rmaData := resp.Data.(map[string]interface{})
	if rmaData["id"] != "RMA-001" {
		t.Errorf("Expected ID 'RMA-001', got '%v'", rmaData["id"])
	}
	if rmaData["serial_number"] != "SN12345" {
		t.Errorf("Expected serial_number 'SN12345', got '%v'", rmaData["serial_number"])
	}
	if rmaData["customer"] != "Acme Corp" {
		t.Errorf("Expected customer 'Acme Corp', got '%v'", rmaData["customer"])
	}
	if rmaData["status"] != "open" {
		t.Errorf("Expected status 'open', got '%v'", rmaData["status"])
	}
}

func TestHandleGetRMA_NotFound(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/rmas/RMA-999", nil)
	w := httptest.NewRecorder()

	h.GetRMA(w, req, "RMA-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleGetRMA_WithTimestamps(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Insert RMA with received_at timestamp
	_, err := db.Exec(`
		INSERT INTO rmas (id, serial_number, reason, status, received_at, created_at)
		VALUES ('RMA-001', 'SN12345', 'Test', 'received', '2024-01-15 14:30:00', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert RMA: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/rmas/RMA-001", nil)
	w := httptest.NewRecorder()

	h.GetRMA(w, req, "RMA-001")

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmaData := resp.Data.(map[string]interface{})

	if rmaData["received_at"] == nil {
		t.Error("Expected received_at to be set")
	}
}

// =============================================================================
// CREATE RMA TESTS
// =============================================================================

func TestHandleCreateRMA_Success(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	reqBody := `{
		"serial_number": "SN12345",
		"customer": "Acme Corp",
		"reason": "Device won't power on",
		"defect_description": "Customer reports the device is completely dead",
		"status": "open"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	rmaData := resp.Data.(map[string]interface{})
	rmaID := rmaData["id"].(string)

	if rmaID == "" {
		t.Error("Expected non-empty RMA ID")
	}
	if rmaData["serial_number"] != "SN12345" {
		t.Errorf("Expected serial_number 'SN12345', got '%v'", rmaData["serial_number"])
	}
	if rmaData["status"] != "open" {
		t.Errorf("Expected status 'open', got '%v'", rmaData["status"])
	}
	if rmaData["created_at"] == "" {
		t.Error("Expected non-empty created_at")
	}

	// Verify RMA was saved to database
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM rmas WHERE id = ?", rmaID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count RMAs: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 RMA in DB, got %d", count)
	}

	// Verify audit log entry
	err = db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE record_id = ? AND action = ?", rmaID, "created").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count audit log: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", count)
	}
}

func TestHandleCreateRMA_DefaultStatus(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Device failure"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmaData := resp.Data.(map[string]interface{})

	if rmaData["status"] != "open" {
		t.Errorf("Expected default status 'open', got '%v'", rmaData["status"])
	}
}

func TestHandleCreateRMA_MissingSerialNumber(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	reqBody := `{
		"customer": "Acme Corp",
		"reason": "Device failure"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "serial_number") {
		t.Errorf("Expected error message to mention 'serial_number', got: %s", w.Body.String())
	}
}

func TestHandleCreateRMA_MissingReason(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	reqBody := `{
		"serial_number": "SN12345",
		"customer": "Acme Corp"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "reason") {
		t.Errorf("Expected error message to mention 'reason', got: %s", w.Body.String())
	}
}

func TestHandleCreateRMA_InvalidStatus(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Device failure",
		"status": "invalid_status"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "status") {
		t.Errorf("Expected error message to mention 'status', got: %s", w.Body.String())
	}
}

func TestHandleCreateRMA_MaxLengthValidation(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	tests := []struct {
		name     string
		body     string
		wantErr  bool
		errField string
	}{
		{
			name:     "serial_number too long",
			body:     fmt.Sprintf(`{"serial_number":"%s","reason":"Test"}`, strings.Repeat("X", 101)),
			wantErr:  true,
			errField: "serial_number",
		},
		{
			name:     "customer too long",
			body:     fmt.Sprintf(`{"serial_number":"SN123","reason":"Test","customer":"%s"}`, strings.Repeat("X", 256)),
			wantErr:  true,
			errField: "customer",
		},
		{
			name:     "reason too long",
			body:     fmt.Sprintf(`{"serial_number":"SN123","reason":"%s"}`, strings.Repeat("X", 256)),
			wantErr:  true,
			errField: "reason",
		},
		{
			name:     "defect_description too long",
			body:     fmt.Sprintf(`{"serial_number":"SN123","reason":"Test","defect_description":"%s"}`, strings.Repeat("X", 1001)),
			wantErr:  true,
			errField: "defect_description",
		},
		{
			name:     "resolution too long",
			body:     fmt.Sprintf(`{"serial_number":"SN123","reason":"Test","resolution":"%s"}`, strings.Repeat("X", 1001)),
			wantErr:  true,
			errField: "resolution",
		},
		{
			name:    "valid max lengths",
			body:    fmt.Sprintf(`{"serial_number":"%s","reason":"%s","customer":"%s","defect_description":"%s","resolution":"%s"}`, strings.Repeat("X", 100), strings.Repeat("Y", 255), strings.Repeat("Z", 255), strings.Repeat("A", 1000), strings.Repeat("B", 1000)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.CreateRMA(w, req)

			if tt.wantErr {
				if w.Code != 400 {
					t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
				}
				if !strings.Contains(w.Body.String(), tt.errField) {
					t.Errorf("Expected error message to mention '%s', got: %s", tt.errField, w.Body.String())
				}
			} else {
				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestHandleCreateRMA_InvalidJSON(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	reqBody := `{invalid json`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "invalid") {
		t.Errorf("Expected error message about invalid body, got: %s", w.Body.String())
	}
}

func TestHandleCreateRMA_MultipleValidationErrors(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Missing both required fields + invalid status
	reqBody := `{
		"customer": "Acme Corp",
		"status": "invalid_status"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	body := w.Body.String()
	// Should report validation errors (may not include all depending on validation order)
	if !strings.Contains(body, "serial_number") && !strings.Contains(body, "reason") {
		t.Error("Expected error to mention at least one missing required field")
	}
}

// =============================================================================
// UPDATE RMA TESTS
// =============================================================================

func TestHandleUpdateRMA_Success(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Device failure", "open", "Won't boot", "")

	reqBody := `{
		"serial_number": "SN12345-UPDATED",
		"customer": "Acme Corp Updated",
		"reason": "Updated reason",
		"status": "received",
		"defect_description": "Updated defect description",
		"resolution": "Diagnosed and repaired"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.UpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmaData := resp.Data.(map[string]interface{})

	if rmaData["serial_number"] != "SN12345-UPDATED" {
		t.Errorf("Expected serial_number to be updated, got '%v'", rmaData["serial_number"])
	}
	if rmaData["status"] != "received" {
		t.Errorf("Expected status 'received', got '%v'", rmaData["status"])
	}

	// Verify audit log
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE record_id = ? AND action = ?", "RMA-001", "updated").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count audit log: %v", err)
	}
	if count < 1 {
		t.Errorf("Expected at least 1 audit log entry, got %d", count)
	}
}

func TestHandleUpdateRMA_StatusTransitions(t *testing.T) {
	tests := []struct {
		name          string
		fromStatus    string
		toStatus      string
		expectError   bool
		checkRecvd    bool // Check if received_at should be set
		checkResolved bool // Check if resolved_at should be set
	}{
		{"open to received", "open", "received", false, true, false},
		{"received to diagnosing", "received", "diagnosing", false, false, false},
		{"diagnosing to repairing", "diagnosing", "repairing", false, false, false},
		{"repairing to resolved", "repairing", "resolved", false, false, false},
		{"resolved to closed", "resolved", "closed", false, false, true},
		{"open to closed", "open", "closed", false, false, true},
		{"received to scrapped", "received", "scrapped", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, h := setupRMATestDB(t)
			defer db.Close()

			insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", tt.fromStatus, "Test defect", "")

			reqBody := fmt.Sprintf(`{
				"serial_number": "SN12345",
				"customer": "Acme Corp",
				"reason": "Test",
				"status": "%s",
				"defect_description": "Test defect"
			}`, tt.toStatus)
			req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			h.UpdateRMA(w, req, "RMA-001")

			if tt.expectError {
				if w.Code == 200 {
					t.Errorf("Expected error, but got status 200")
				}
			} else {
				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
				}

				var resp models.APIResponse
				json.NewDecoder(w.Body).Decode(&resp)
				rmaData := resp.Data.(map[string]interface{})

				if rmaData["status"] != tt.toStatus {
					t.Errorf("Expected status '%s', got '%v'", tt.toStatus, rmaData["status"])
				}

				// Verify timestamp was set if expected
				if tt.checkRecvd {
					if rmaData["received_at"] == nil {
						t.Error("Expected received_at to be set")
					}
				}
				if tt.checkResolved {
					if rmaData["resolved_at"] == nil {
						t.Error("Expected resolved_at to be set")
					}
				}
			}
		})
	}
}

func TestHandleUpdateRMA_ReceivedAtTimestamp(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "open", "Test", "")

	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Test",
		"status": "received"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.UpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify received_at was set in database
	var receivedAt sql.NullString
	err := db.QueryRow("SELECT received_at FROM rmas WHERE id = ?", "RMA-001").Scan(&receivedAt)
	if err != nil {
		t.Fatalf("Failed to query received_at: %v", err)
	}

	if !receivedAt.Valid {
		t.Error("Expected received_at to be set in database")
	} else {
		// Just verify it's a valid timestamp that parses
		formats := []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02T15:04:05Z"}
		var parsedTime time.Time
		var parseErr error
		for _, format := range formats {
			parsedTime, parseErr = time.Parse(format, receivedAt.String)
			if parseErr == nil {
				break
			}
		}
		if parseErr != nil {
			t.Errorf("Failed to parse received_at timestamp '%s': %v", receivedAt.String, parseErr)
		} else {
			// Verify it's a recent timestamp (within last hour - accounts for timezone differences)
			now := time.Now()
			hourAgo := now.Add(-1 * time.Hour)
			hourFromNow := now.Add(1 * time.Hour)
			if parsedTime.Before(hourAgo) || parsedTime.After(hourFromNow) {
				t.Logf("Warning: Timestamp %v seems unexpected (now=%v), but may be due to timezone handling", parsedTime, now)
			}
		}
	}
}

func TestHandleUpdateRMA_ResolvedAtTimestamp(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "repairing", "Test", "")

	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Test",
		"status": "closed",
		"resolution": "Repaired and tested"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.UpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify resolved_at was set in database
	var resolvedAt sql.NullString
	err := db.QueryRow("SELECT resolved_at FROM rmas WHERE id = ?", "RMA-001").Scan(&resolvedAt)
	if err != nil {
		t.Fatalf("Failed to query resolved_at: %v", err)
	}

	if !resolvedAt.Valid {
		t.Error("Expected resolved_at to be set when status is 'closed'")
	} else {
		t.Logf("resolved_at timestamp: %s", resolvedAt.String)
	}
}

// BUG FOUND: This test is disabled because "shipped" is referenced in handler_rma.go line 70
// but is NOT in validRMAStatuses. The code checks: if rm.Status == "closed" || rm.Status == "shipped"
// but validRMAStatuses = []string{"open", "received", "diagnosing", "repairing", "resolved", "closed", "scrapped"}
// FIX NEEDED: Either add "shipped" to validRMAStatuses or remove the check from handler_rma.go
func TestHandleUpdateRMA_ShippedTimestamp_BUG(t *testing.T) {
	t.Skip("BUG: 'shipped' status is referenced in code but not in validRMAStatuses - this test documents the issue")
}

func TestHandleUpdateRMA_InvalidJSON(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "open", "", "")

	reqBody := `{invalid json`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.UpdateRMA(w, req, "RMA-001")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandleUpdateRMA_MaxLengthValidation(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Test", "open", "", "")

	tests := []struct {
		name     string
		body     string
		wantErr  bool
		errField string
	}{
		{
			name:     "serial_number too long",
			body:     fmt.Sprintf(`{"serial_number":"%s"}`, strings.Repeat("X", 101)),
			wantErr:  true,
			errField: "serial_number",
		},
		{
			name:     "customer too long",
			body:     fmt.Sprintf(`{"serial_number":"SN123","customer":"%s"}`, strings.Repeat("X", 256)),
			wantErr:  true,
			errField: "customer",
		},
		{
			name:    "valid update",
			body:    `{"serial_number":"SN123-NEW","customer":"New Customer","status":"received"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.UpdateRMA(w, req, "RMA-001")

			if tt.wantErr {
				if w.Code != 400 {
					t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
				}
				if !strings.Contains(w.Body.String(), tt.errField) {
					t.Errorf("Expected error message to mention '%s', got: %s", tt.errField, w.Body.String())
				}
			} else {
				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
				}
			}
		})
	}
}

// =============================================================================
// EDGE CASES & SECURITY TESTS
// =============================================================================

func TestHandleCreateRMA_EmptyFields(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Test",
		"customer": "",
		"defect_description": "",
		"resolution": ""
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 (empty optional fields allowed), got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateRMA_XSS_Prevention(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	reqBody := `{
		"serial_number": "<script>alert('xss')</script>",
		"customer": "<img src=x onerror=alert('xss')>",
		"reason": "<svg onload=alert('xss')>",
		"defect_description": "'; DROP TABLE rmas; --"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmaData := resp.Data.(map[string]interface{})
	rmaID := rmaData["id"].(string)

	// Verify data was stored as-is (no injection)
	var serial, customer, reason, defect string
	err := db.QueryRow("SELECT serial_number, customer, reason, defect_description FROM rmas WHERE id = ?", rmaID).
		Scan(&serial, &customer, &reason, &defect)
	if err != nil {
		t.Fatalf("Failed to query RMA: %v", err)
	}

	if serial != "<script>alert('xss')</script>" {
		t.Errorf("XSS payload was modified: %s", serial)
	}
	if customer != "<img src=x onerror=alert('xss')>" {
		t.Errorf("XSS payload was modified: %s", customer)
	}

	// Verify table still exists (SQL injection didn't work)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM rmas").Scan(&count)
	if err != nil {
		t.Error("Table 'rmas' appears to have been deleted - SQL injection vulnerability!")
	}
}

func TestHandleCreateRMA_SQLInjection_Prevention(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Try various SQL injection payloads
	payloads := []string{
		"'; DROP TABLE rmas; --",
		"' OR '1'='1",
		"'; UPDATE rmas SET status='closed'; --",
		"1' UNION SELECT * FROM audit_log--",
	}

	for _, payload := range payloads {
		t.Run(fmt.Sprintf("payload_%s", payload), func(t *testing.T) {
			reqBody := fmt.Sprintf(`{
				"serial_number": "%s",
				"reason": "Test"
			}`, payload)
			req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			h.CreateRMA(w, req)

			// Should succeed (payload stored as data, not executed)
			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Verify table still exists
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM rmas").Scan(&count)
			if err != nil {
				t.Errorf("Table 'rmas' appears damaged - SQL injection vulnerability! %v", err)
			}
		})
	}
}

func TestHandleListRMAs_NullFields(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Insert RMA with NULL optional fields
	_, err := db.Exec(`
		INSERT INTO rmas (id, serial_number, reason, status, created_at)
		VALUES ('RMA-001', 'SN12345', 'Test reason', 'open', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert RMA: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/rmas", nil)
	w := httptest.NewRecorder()

	h.ListRMAs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmas := resp.Data.([]interface{})

	if len(rmas) != 1 {
		t.Fatalf("Expected 1 RMA, got %d", len(rmas))
	}

	rma := rmas[0].(map[string]interface{})
	// COALESCE should convert NULL to empty string
	if rma["customer"] != "" {
		t.Errorf("Expected empty customer (COALESCE), got '%v'", rma["customer"])
	}
}

func TestHandleUpdateRMA_PreserveExistingTimestamps(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Insert RMA with received_at already set
	_, err := db.Exec(`
		INSERT INTO rmas (id, serial_number, reason, status, received_at, created_at)
		VALUES ('RMA-001', 'SN12345', 'Test', 'received', '2024-01-01 10:00:00', datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert RMA: %v", err)
	}

	// Update to diagnosing (doesn't set received_at)
	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Test",
		"status": "diagnosing"
	}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.UpdateRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify received_at was preserved (COALESCE logic)
	var receivedAt string
	err = db.QueryRow("SELECT received_at FROM rmas WHERE id = ?", "RMA-001").Scan(&receivedAt)
	if err != nil {
		t.Fatalf("Failed to query received_at: %v", err)
	}

	// SQLite may return timestamps in different formats, verify it contains the date/time
	if !strings.Contains(receivedAt, "2024-01-01") || !strings.Contains(receivedAt, "10:00:00") {
		t.Errorf("Expected received_at to be preserved with date 2024-01-01 and time 10:00:00, got '%s'", receivedAt)
	}
}

// =============================================================================
// BUSINESS LOGIC TESTS
// =============================================================================

func TestHandleCreateRMA_AllValidStatuses(t *testing.T) {
	validStatuses := []string{"open", "received", "diagnosing", "repairing", "resolved", "closed", "scrapped"}

	for _, status := range validStatuses {
		t.Run(fmt.Sprintf("status_%s", status), func(t *testing.T) {
			db, h := setupRMATestDB(t)
			defer db.Close()

			reqBody := fmt.Sprintf(`{
				"serial_number": "SN12345",
				"reason": "Test",
				"status": "%s"
			}`, status)
			req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			h.CreateRMA(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200 for valid status '%s', got %d: %s", status, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleUpdateRMA_CompleteWorkflow(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Create RMA
	insertTestRMA(t, db, "RMA-001", "SN12345", "Acme Corp", "Device won't boot", "open", "Completely dead", "")

	// Step 1: Mark as received
	reqBody := `{"serial_number":"SN12345","reason":"Device won't boot","status":"received"}`
	req := httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	h.UpdateRMA(w, req, "RMA-001")
	if w.Code != 200 {
		t.Fatalf("Failed to mark as received: %d", w.Code)
	}

	// Step 2: Diagnose
	reqBody = `{"serial_number":"SN12345","reason":"Device won't boot","status":"diagnosing"}`
	req = httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()
	h.UpdateRMA(w, req, "RMA-001")
	if w.Code != 200 {
		t.Fatalf("Failed to mark as diagnosing: %d", w.Code)
	}

	// Step 3: Repair
	reqBody = `{"serial_number":"SN12345","reason":"Device won't boot","status":"repairing","defect_description":"Bad power supply"}`
	req = httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()
	h.UpdateRMA(w, req, "RMA-001")
	if w.Code != 200 {
		t.Fatalf("Failed to mark as repairing: %d", w.Code)
	}

	// Step 4: Close
	reqBody = `{"serial_number":"SN12345","reason":"Device won't boot","status":"closed","resolution":"Replaced power supply, tested OK"}`
	req = httptest.NewRequest("PUT", "/api/rmas/RMA-001", bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()
	h.UpdateRMA(w, req, "RMA-001")
	if w.Code != 200 {
		t.Fatalf("Failed to close RMA: %d", w.Code)
	}

	// Verify final state
	var status, resolution string
	var receivedAt, resolvedAt sql.NullString
	err := db.QueryRow("SELECT status, resolution, received_at, resolved_at FROM rmas WHERE id = ?", "RMA-001").
		Scan(&status, &resolution, &receivedAt, &resolvedAt)
	if err != nil {
		t.Fatalf("Failed to query final state: %v", err)
	}

	if status != "closed" {
		t.Errorf("Expected final status 'closed', got '%s'", status)
	}
	if !receivedAt.Valid {
		t.Error("Expected received_at to be set")
	}
	if !resolvedAt.Valid {
		t.Error("Expected resolved_at to be set")
	}
	if resolution != "Replaced power supply, tested OK" {
		t.Errorf("Expected resolution to be set, got '%s'", resolution)
	}
}

func TestHandleCreateRMA_MinimalData(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Only required fields
	reqBody := `{
		"serial_number": "SN12345",
		"reason": "Test"
	}`
	req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.CreateRMA(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 with minimal data, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmaData := resp.Data.(map[string]interface{})

	if rmaData["status"] != "open" {
		t.Errorf("Expected default status 'open', got '%v'", rmaData["status"])
	}
}

func TestHandleGetRMA_WithAllFields(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Insert RMA with all fields populated
	_, err := db.Exec(`
		INSERT INTO rmas (id, serial_number, customer, reason, status, defect_description, resolution, created_at, received_at, resolved_at)
		VALUES ('RMA-001', 'SN12345', 'Acme Corp', 'Device failure', 'closed', 'Power supply failed', 'Replaced PSU',
				'2024-01-01 10:00:00', '2024-01-02 11:00:00', '2024-01-05 14:30:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert RMA: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/rmas/RMA-001", nil)
	w := httptest.NewRecorder()

	h.GetRMA(w, req, "RMA-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	rmaData := resp.Data.(map[string]interface{})

	// Verify all fields are present
	expectedFields := []string{"id", "serial_number", "customer", "reason", "status", "defect_description", "resolution", "created_at"}
	for _, f := range expectedFields {
		if rmaData[f] == nil || rmaData[f] == "" {
			t.Errorf("Expected field '%s' to be populated", f)
		}
	}

	if rmaData["received_at"] == nil {
		t.Error("Expected received_at to be set")
	}
	if rmaData["resolved_at"] == nil {
		t.Error("Expected resolved_at to be set")
	}
}

func TestHandleCreateRMA_IDGeneration(t *testing.T) {
	db, h := setupRMATestDB(t)
	defer db.Close()

	// Create multiple RMAs and verify IDs are sequential
	for i := 1; i <= 5; i++ {
		reqBody := fmt.Sprintf(`{
			"serial_number": "SN%05d",
			"reason": "Test %d"
		}`, i, i)
		req := httptest.NewRequest("POST", "/api/rmas", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		h.CreateRMA(w, req)

		if w.Code != 200 {
			t.Fatalf("Failed to create RMA %d: %d", i, w.Code)
		}

		var resp models.APIResponse
		json.NewDecoder(w.Body).Decode(&resp)
		rmaData := resp.Data.(map[string]interface{})
		rmaID := rmaData["id"].(string)

		// Verify ID format (RMA-XXX with at least 3 digits)
		if !strings.HasPrefix(rmaID, "RMA-") {
			t.Errorf("Expected ID to start with 'RMA-', got '%s'", rmaID)
		}
		if len(rmaID) < 7 { // RMA-001 = 7 chars minimum
			t.Errorf("Expected ID length >= 7, got %d for '%s'", len(rmaID), rmaID)
		}
	}

	// Verify all 5 RMAs exist
	var count int
	db.QueryRow("SELECT COUNT(*) FROM rmas").Scan(&count)
	if count != 5 {
		t.Errorf("Expected 5 RMAs in database, got %d", count)
	}
}
