package field_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zrp/internal/handlers/quality"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

func setupTestingHandlerTestDB(t *testing.T) (*sql.DB, *quality.Handler) {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create test_records table
	_, err = testDB.Exec(`
		CREATE TABLE test_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			serial_number TEXT NOT NULL,
			ipn TEXT NOT NULL,
			firmware_version TEXT,
			test_type TEXT,
			result TEXT NOT NULL,
			measurements TEXT,
			notes TEXT,
			tested_by TEXT NOT NULL,
			tested_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test_records table: %v", err)
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

	// Create users and sessions for authentication
	_, err = testDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT,
			role TEXT DEFAULT 'user'
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	// Insert test user
	_, err = testDB.Exec(`INSERT INTO users (id, username, email, role) VALUES (1, 'testuser', 'test@example.com', 'admin')`)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Insert test session
	_, err = testDB.Exec(`INSERT INTO sessions (token, user_id) VALUES ('test-session-token', 1)`)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	h := &quality.Handler{
		DB:  testDB,
		Hub: nil,
		NextIDFunc: func(prefix, table string, digits int) string {
			return prefix + "-0001"
		},
		RecordChangeJSON: func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error) {
			return 0, nil
		},
		GetNCRSnapshot: func(id string) (map[string]interface{}, error) {
			return nil, nil
		},
		GetCAPASnapshot: func(id string) (map[string]interface{}, error) {
			return nil, nil
		},
		GetUserID: func(r *http.Request) (int, error) {
			return 1, nil
		},
		GetUserRole: func(r *http.Request) string {
			return "admin"
		},
		CanApproveCAPA: func(r *http.Request, approvalType string) bool {
			return true
		},
		EmailOnNCRCreated: func(ncrID, title string) {},
		EmailOnCAPACreated: func(c models.CAPA) {},
		EmailOnCAPACreatedWithDB: func(database *sql.DB, c models.CAPA) {},
		SendEmail: func(to, subject, body string) error {
			return nil
		},
	}

	return testDB, h
}

func TestHandleListTests_Empty(t *testing.T) {
	testDB, h := setupTestingHandlerTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("GET", "/api/tests", nil)
	w := httptest.NewRecorder()

	h.ListTests(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Response has data wrapper
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}

	testsJSON, _ := json.Marshal(resp.Data)
	var tests []models.TestRecord
	if err := json.Unmarshal(testsJSON, &tests); err != nil {
		t.Fatalf("Failed to unmarshal tests: %v", err)
	}

	if len(tests) != 0 {
		t.Errorf("Expected empty array, got %d tests", len(tests))
	}
}

func TestHandleListTests_WithData(t *testing.T) {
	testDB, h := setupTestingHandlerTestDB(t)
	defer testDB.Close()

	// Insert test records with different timestamps to ensure DESC order
	now1 := time.Now().Add(-1 * time.Hour).Format("2006-01-02 15:04:05")
	now2 := time.Now().Format("2006-01-02 15:04:05")
	_, err := testDB.Exec(`INSERT INTO test_records
		(serial_number, ipn, firmware_version, test_type, result, measurements, notes, tested_by, tested_at)
		VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?),
		(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"SN001", "PCB-100", "v1.2.3", "functional", "pass", "{\"voltage\": 5.0}", "All tests passed", "operator1", now1,
		"SN002", "PCB-100", "v1.2.3", "burn-in", "fail", "{\"temp\": 85}", "Overheating detected", "operator2", now2,
	)
	if err != nil {
		t.Fatalf("Failed to insert test records: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/tests", nil)
	w := httptest.NewRecorder()

	h.ListTests(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Response has data wrapper
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}

	testsJSON, _ := json.Marshal(resp.Data)
	var tests []models.TestRecord
	if err := json.Unmarshal(testsJSON, &tests); err != nil {
		t.Fatalf("Failed to unmarshal tests: %v", err)
	}

	if len(tests) != 2 {
		t.Errorf("Expected 2 tests, got %d", len(tests))
	}

	if len(tests) >= 2 {
		// Verify DESC order by tested_at (most recent first)
		if tests[0].SerialNumber != "SN002" {
			t.Errorf("Expected first test serial_number='SN002', got '%s'", tests[0].SerialNumber)
		}
		if tests[1].SerialNumber != "SN001" {
			t.Errorf("Expected second test serial_number='SN001', got '%s'", tests[1].SerialNumber)
		}
	}
}

func TestHandleCreateTest_Success(t *testing.T) {
	testDB, h := setupTestingHandlerTestDB(t)
	defer testDB.Close()

	body := `{
		"serial_number": "SN-TEST-001",
		"ipn": "PCB-500",
		"firmware_version": "v2.0.0",
		"test_type": "functional",
		"result": "pass",
		"measurements": "{\"voltage\": 5.1}",
		"notes": "All measurements within spec"
	}`

	req := httptest.NewRequest("POST", "/api/tests", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	h.CreateTest(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Response has data wrapper
	var resp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}

	testJSON, _ := json.Marshal(resp.Data)
	var test models.TestRecord
	if err := json.Unmarshal(testJSON, &test); err != nil {
		t.Fatalf("Failed to unmarshal test: %v", err)
	}

	if test.ID == 0 {
		t.Error("Expected ID to be set")
	}
	if test.SerialNumber != "SN-TEST-001" {
		t.Errorf("Expected serial_number='SN-TEST-001', got '%s'", test.SerialNumber)
	}
}

func TestHandleCreateTest_InvalidJSON(t *testing.T) {
	testDB, h := setupTestingHandlerTestDB(t)
	defer testDB.Close()

	body := `{invalid json`

	req := httptest.NewRequest("POST", "/api/tests", bytes.NewBufferString(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	h.CreateTest(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}
