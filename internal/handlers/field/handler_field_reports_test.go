package field_test

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"zrp/internal/handlers/field"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

func newTestHandler(db *sql.DB) *field.Handler {
	return &field.Handler{
		DB:  db,
		Hub: nil,
		NextIDFunc: func(prefix, table string, digits int) string {
			var count int
			db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
			return prefix + "-" + strings.Repeat("0", digits-1) + string(rune('1'+count))
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
}

// nextIDFunc returns a NextIDFunc that generates sequential IDs with a counter.
func nextIDFunc(db *sql.DB) func(prefix, table string, digits int) string {
	counter := 0
	return func(prefix, table string, digits int) string {
		counter++
		id := prefix + "-"
		for i := len(intToStr(counter)); i < digits; i++ {
			id += "0"
		}
		id += intToStr(counter)
		return id
	}
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func setupFieldReportsTestDB(t *testing.T) (*sql.DB, *field.Handler) {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create field_reports table
	_, err = testDB.Exec(`
		CREATE TABLE field_reports (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			report_type TEXT DEFAULT 'failure',
			status TEXT DEFAULT 'open',
			priority TEXT DEFAULT 'medium',
			customer_name TEXT,
			site_location TEXT,
			device_ipn TEXT,
			device_serial TEXT,
			reported_by TEXT,
			reported_at TEXT,
			description TEXT,
			root_cause TEXT,
			resolution TEXT,
			resolved_at TEXT,
			ncr_id TEXT,
			eco_id TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create field_reports table: %v", err)
	}

	// Create ncrs table for FK relationship
	_, err = testDB.Exec(`
		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			ipn TEXT,
			serial_number TEXT,
			defect_type TEXT,
			severity TEXT DEFAULT 'minor',
			status TEXT DEFAULT 'open',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
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

	h := &field.Handler{
		DB:  testDB,
		Hub: nil,
		NextIDFunc: nextIDFunc(testDB),
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

func stringReader(s string) *strings.Reader {
	return strings.NewReader(s)
}

func TestFieldReportCRUD(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	// List empty
	req := httptest.NewRequest("GET", "/api/v1/field-reports", nil)
	w := httptest.NewRecorder()
	h.ListFieldReports(w, req)
	if w.Code != 200 {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}
	var listResp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &listResp)
	items := listResp.Data.([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}

	// Create
	body := `{"title":"Motor overheating","report_type":"failure","priority":"high","customer_name":"Acme Corp","site_location":"Plant 3","device_ipn":"MOT-001-0001","device_serial":"SN-12345","reported_by":"John","description":"Motor runs hot after 2 hours"}`
	req = httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(body))
	w = httptest.NewRecorder()
	h.CreateFieldReport(w, req)
	if w.Code != 200 {
		t.Fatalf("create: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var createResp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &createResp)
	frData := createResp.Data.(map[string]interface{})
	frID := frData["id"].(string)
	if frData["status"] != "open" {
		t.Errorf("expected status open, got %v", frData["status"])
	}
	if frData["title"] != "Motor overheating" {
		t.Errorf("expected title Motor overheating, got %v", frData["title"])
	}

	// Get
	req = httptest.NewRequest("GET", "/api/v1/field-reports/"+frID, nil)
	w = httptest.NewRecorder()
	h.GetFieldReport(w, req, frID)
	if w.Code != 200 {
		t.Fatalf("get: expected 200, got %d", w.Code)
	}
	var getResp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &getResp)
	getData := getResp.Data.(map[string]interface{})
	if getData["customer_name"] != "Acme Corp" {
		t.Errorf("expected Acme Corp, got %v", getData["customer_name"])
	}

	// Update
	body = `{"status":"investigating","root_cause":"Bearing failure"}`
	req = httptest.NewRequest("PUT", "/api/v1/field-reports/"+frID, stringReader(body))
	w = httptest.NewRecorder()
	h.UpdateFieldReport(w, req, frID)
	if w.Code != 200 {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updateResp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &updateResp)
	upData := updateResp.Data.(map[string]interface{})
	if upData["status"] != "investigating" {
		t.Errorf("expected investigating, got %v", upData["status"])
	}
	if upData["root_cause"] != "Bearing failure" {
		t.Errorf("expected Bearing failure, got %v", upData["root_cause"])
	}

	// Delete
	req = httptest.NewRequest("DELETE", "/api/v1/field-reports/"+frID, nil)
	w = httptest.NewRecorder()
	h.DeleteFieldReport(w, req, frID)
	if w.Code != 200 {
		t.Fatalf("delete: expected 200, got %d", w.Code)
	}

	// Verify deleted
	req = httptest.NewRequest("GET", "/api/v1/field-reports/"+frID, nil)
	w = httptest.NewRecorder()
	h.GetFieldReport(w, req, frID)
	if w.Code != 404 {
		t.Errorf("get deleted: expected 404, got %d", w.Code)
	}
}

func TestFieldReportGetNotFound(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("GET", "/api/v1/field-reports/FR-9999", nil)
	w := httptest.NewRecorder()
	h.GetFieldReport(w, req, "FR-9999")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFieldReportCreateInvalidBody(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader("not json"))
	w := httptest.NewRecorder()
	h.CreateFieldReport(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFieldReportCreateMissingTitle(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(`{"title":""}`))
	w := httptest.NewRecorder()
	h.CreateFieldReport(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFieldReportValidationMaxLength(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	tests := []struct {
		name  string
		body  string
		valid bool
	}{
		{"title too long", `{"title":"` + string(make([]byte, 256)) + `"}`, false},
		{"description too long", `{"title":"Test","description":"` + string(make([]byte, 1001)) + `"}`, false},
		{"customer_name too long", `{"title":"Test","customer_name":"` + string(make([]byte, 256)) + `"}`, false},
		{"valid lengths", `{"title":"Test","description":"Short desc","customer_name":"Acme"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(tt.body))
			w := httptest.NewRecorder()
			h.CreateFieldReport(w, req)
			if tt.valid && w.Code != 200 {
				t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
			} else if !tt.valid && w.Code == 200 {
				t.Errorf("expected validation error, got 200")
			}
		})
	}
}

func TestFieldReportFilters(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	// Create two reports
	body1 := `{"title":"Failure 1","report_type":"failure","priority":"high","status":"open"}`
	req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(body1))
	w := httptest.NewRecorder()
	h.CreateFieldReport(w, req)

	body2 := `{"title":"Visit 1","report_type":"visit","priority":"low","status":"open"}`
	req = httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(body2))
	w = httptest.NewRecorder()
	h.CreateFieldReport(w, req)

	// Filter by type
	req = httptest.NewRequest("GET", "/api/v1/field-reports?report_type=failure", nil)
	w = httptest.NewRecorder()
	h.ListFieldReports(w, req)
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp.Data.([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 failure, got %d", len(items))
	}

	// Filter by priority
	req = httptest.NewRequest("GET", "/api/v1/field-reports?priority=low", nil)
	w = httptest.NewRecorder()
	h.ListFieldReports(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	items = resp.Data.([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 low priority, got %d", len(items))
	}

	// Filter by status
	req = httptest.NewRequest("GET", "/api/v1/field-reports?status=open", nil)
	w = httptest.NewRecorder()
	h.ListFieldReports(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	items = resp.Data.([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 open reports, got %d", len(items))
	}
}

func TestFieldReportResolve(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	// Create
	body := `{"title":"Test issue","report_type":"other"}`
	req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(body))
	w := httptest.NewRecorder()
	h.CreateFieldReport(w, req)
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	frID := resp.Data.(map[string]interface{})["id"].(string)

	// Resolve
	body = `{"status":"resolved","resolution":"Fixed the widget","root_cause":"Bad solder joint"}`
	req = httptest.NewRequest("PUT", "/api/v1/field-reports/"+frID, stringReader(body))
	w = httptest.NewRecorder()
	h.UpdateFieldReport(w, req, frID)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["status"] != "resolved" {
		t.Errorf("expected resolved, got %v", data["status"])
	}
	if data["resolved_at"] == nil || data["resolved_at"] == "" {
		t.Error("expected resolved_at to be set")
	}
}

func TestFieldReportCreateNCR(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	// Create field report
	body := `{"title":"Solder defect in field","report_type":"failure","priority":"critical","device_ipn":"PCA-001-0001","device_serial":"SN-99","description":"Cold solder joint found"}`
	req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(body))
	w := httptest.NewRecorder()
	h.CreateFieldReport(w, req)
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	frID := resp.Data.(map[string]interface{})["id"].(string)

	// Create NCR from field report
	req = httptest.NewRequest("POST", "/api/v1/field-reports/"+frID+"/create-ncr", nil)
	w = httptest.NewRecorder()
	h.FieldReportCreateNCR(w, req, frID)
	if w.Code != 200 {
		t.Fatalf("create-ncr: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var ncrResp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &ncrResp)
	ncrData := ncrResp.Data.(map[string]interface{})
	ncrID := ncrData["id"].(string)
	if ncrID == "" {
		t.Fatal("expected NCR ID")
	}
	if ncrData["title"] != "Solder defect in field" {
		t.Errorf("expected NCR title from field report, got %v", ncrData["title"])
	}
	if ncrData["severity"] != "critical" {
		t.Errorf("expected critical severity from critical priority, got %v", ncrData["severity"])
	}

	// Verify field report now references the NCR
	req = httptest.NewRequest("GET", "/api/v1/field-reports/"+frID, nil)
	w = httptest.NewRecorder()
	h.GetFieldReport(w, req, frID)
	json.Unmarshal(w.Body.Bytes(), &resp)
	frData := resp.Data.(map[string]interface{})
	if frData["ncr_id"] != ncrID {
		t.Errorf("expected ncr_id %s, got %v", ncrID, frData["ncr_id"])
	}
}

func TestFieldReportCreateNCRNotFound(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("POST", "/api/v1/field-reports/FR-9999/create-ncr", nil)
	w := httptest.NewRecorder()
	h.FieldReportCreateNCR(w, req, "FR-9999")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFieldReportCreateNCRSeverityMapping(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	tests := []struct {
		priority string
		severity string
	}{
		{"critical", "critical"},
		{"high", "major"},
		{"medium", "minor"},
		{"low", "minor"},
	}

	for _, tt := range tests {
		t.Run("priority_"+tt.priority, func(t *testing.T) {
			body := `{"title":"Test","priority":"` + tt.priority + `"}`
			req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(body))
			w := httptest.NewRecorder()
			h.CreateFieldReport(w, req)
			var resp models.APIResponse
			json.Unmarshal(w.Body.Bytes(), &resp)
			frID := resp.Data.(map[string]interface{})["id"].(string)

			req = httptest.NewRequest("POST", "/api/v1/field-reports/"+frID+"/create-ncr", nil)
			w = httptest.NewRecorder()
			h.FieldReportCreateNCR(w, req, frID)

			var ncrResp models.APIResponse
			json.Unmarshal(w.Body.Bytes(), &ncrResp)
			ncrData := ncrResp.Data.(map[string]interface{})
			if ncrData["severity"] != tt.severity {
				t.Errorf("priority %s: expected severity %s, got %v", tt.priority, tt.severity, ncrData["severity"])
			}
		})
	}
}

func TestFieldReportECOLink(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	// Create field report with eco_id
	body := `{"title":"Design flaw","report_type":"failure","eco_id":"ECO-2026-001"}`
	req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(body))
	w := httptest.NewRecorder()
	h.CreateFieldReport(w, req)
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	frData := resp.Data.(map[string]interface{})
	if frData["eco_id"] != "ECO-2026-001" {
		t.Errorf("expected eco_id ECO-2026-001, got %v", frData["eco_id"])
	}
}

func TestFieldReportLocationData(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	body := `{"title":"On-site issue","site_location":"Building A, Floor 2, Room 201","customer_name":"Test Corp"}`
	req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(body))
	w := httptest.NewRecorder()
	h.CreateFieldReport(w, req)
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})

	if data["site_location"] != "Building A, Floor 2, Room 201" {
		t.Errorf("expected detailed location, got %v", data["site_location"])
	}
}

func TestFieldReportDefaults(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	// Create with minimal data
	body := `{"title":"Minimal report"}`
	req := httptest.NewRequest("POST", "/api/v1/field-reports", stringReader(body))
	w := httptest.NewRecorder()
	h.CreateFieldReport(w, req)
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})

	if data["status"] != "open" {
		t.Errorf("expected default status 'open', got %v", data["status"])
	}
	if data["priority"] != "medium" {
		t.Errorf("expected default priority 'medium', got %v", data["priority"])
	}
	if data["report_type"] != "failure" {
		t.Errorf("expected default report_type 'failure', got %v", data["report_type"])
	}
}

func TestFieldReportUpdateNotFound(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	body := `{"status":"resolved"}`
	req := httptest.NewRequest("PUT", "/api/v1/field-reports/FR-9999", stringReader(body))
	w := httptest.NewRecorder()
	h.UpdateFieldReport(w, req, "FR-9999")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFieldReportDeleteNotFound(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("DELETE", "/api/v1/field-reports/FR-9999", nil)
	w := httptest.NewRecorder()
	h.DeleteFieldReport(w, req, "FR-9999")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFieldReportDateRangeFilter(t *testing.T) {
	testDB, h := setupFieldReportsTestDB(t)
	defer testDB.Close()

	// Create reports with different dates
	testDB.Exec(`INSERT INTO field_reports (id, title, created_at) VALUES
		('FR-001', 'Old Report', '2025-01-01 10:00:00'),
		('FR-002', 'Recent Report', '2026-02-15 10:00:00')`)

	// Filter by date range
	req := httptest.NewRequest("GET", "/api/v1/field-reports?from=2026-01-01", nil)
	w := httptest.NewRecorder()
	h.ListFieldReports(w, req)
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp.Data.([]interface{})

	if len(items) != 1 {
		t.Errorf("expected 1 report from 2026, got %d", len(items))
	}
}
