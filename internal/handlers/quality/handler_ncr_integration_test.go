package quality_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

func setupNCRIntegrationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
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

	// Create capas table
	_, err = testDB.Exec(`
		CREATE TABLE capas (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			type TEXT CHECK(type IN ('corrective','preventive')),
			linked_ncr_id TEXT,
			linked_rma_id TEXT,
			root_cause TEXT,
			action_plan TEXT,
			owner TEXT,
			due_date TEXT,
			status TEXT DEFAULT 'open' CHECK(status IN ('open','in-progress','verification','closed')),
			effectiveness_check TEXT,
			approved_by_qe TEXT,
			approved_by_qe_at DATETIME,
			approved_by_mgr TEXT,
			approved_by_mgr_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create capas table: %v", err)
	}

	// Create ecos table
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT,
			created_by TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			ncr_id TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
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

	// Create part_changes table
	_, err = testDB.Exec(`
		CREATE TABLE part_changes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user TEXT,
			table_name TEXT,
			record_id TEXT,
			operation TEXT,
			old_snapshot TEXT,
			new_snapshot TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create part_changes table: %v", err)
	}

	// Create id_sequences table
	_, err = testDB.Exec(`
		CREATE TABLE id_sequences (
			prefix TEXT PRIMARY KEY,
			next_num INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create id_sequences table: %v", err)
	}

	return testDB
}

func TestHandleCreateCAPAFromNCR_Success(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, ipn, severity, root_cause, corrective_action, created_at) VALUES
		('NCR-001', 'Defect in Assembly', 'Missing screws', 'IPN-100', 'major', 'Manufacturing process error', 'Update work instructions', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	body := `{
		"title": "CAPA for NCR-001",
		"type": "corrective",
		"owner": "john.doe",
		"due_date": "2026-02-01"
	}`

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-001/create-capa", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateCAPAFromNCR(w, req, "NCR-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	resultBytes, _ := json.Marshal(response.Data)
	var result models.CAPA
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal CAPA: %v", err)
	}

	if result.Title != "CAPA for NCR-001" {
		t.Errorf("Expected title 'CAPA for NCR-001', got %s", result.Title)
	}

	if result.LinkedNCRID != "NCR-001" {
		t.Errorf("Expected linked_ncr_id 'NCR-001', got %s", result.LinkedNCRID)
	}

	if result.Type != "corrective" {
		t.Errorf("Expected type 'corrective', got %s", result.Type)
	}

	if result.Owner != "john.doe" {
		t.Errorf("Expected owner 'john.doe', got %s", result.Owner)
	}

	if result.Status != "open" {
		t.Errorf("Expected status 'open', got %s", result.Status)
	}
}

func TestHandleCreateCAPAFromNCR_AutoPopulate(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR with root cause and corrective action
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, severity, root_cause, corrective_action, created_at) VALUES
		('NCR-002', 'Paint Defect', 'Uneven coating', 'minor', 'Spray gun malfunction', 'Replace spray nozzle', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	// Send request without title, root_cause, or action_plan - should auto-populate
	body := `{
		"owner": "jane.smith",
		"due_date": "2026-02-15"
	}`

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-002/create-capa", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateCAPAFromNCR(w, req, "NCR-002")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	resultBytes, _ := json.Marshal(response.Data)
	var result models.CAPA
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal CAPA: %v", err)
	}

	// Check auto-populated title
	expectedTitle := "CAPA for NCR NCR-002: Paint Defect"
	if result.Title != expectedTitle {
		t.Errorf("Expected auto-populated title '%s', got %s", expectedTitle, result.Title)
	}

	// Check auto-populated root cause
	if result.RootCause != "Spray gun malfunction" {
		t.Errorf("Expected root_cause 'Spray gun malfunction', got %s", result.RootCause)
	}

	// Check auto-populated action plan
	if result.ActionPlan != "Replace spray nozzle" {
		t.Errorf("Expected action_plan 'Replace spray nozzle', got %s", result.ActionPlan)
	}

	// Check default type when not provided
	if result.Type != "corrective" {
		t.Errorf("Expected default type 'corrective', got %s", result.Type)
	}
}

func TestHandleCreateCAPAFromNCR_EmptyBody(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, severity, created_at) VALUES
		('NCR-003', 'Test Defect', 'Test description', 'critical', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-003/create-capa", nil)
	w := httptest.NewRecorder()

	h.CreateCAPAFromNCR(w, req, "NCR-003")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	resultBytes, _ := json.Marshal(response.Data)
	var result models.CAPA
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal CAPA: %v", err)
	}

	// Should auto-populate title
	expectedTitle := "CAPA for NCR NCR-003: Test Defect"
	if result.Title != expectedTitle {
		t.Errorf("Expected auto-populated title '%s', got %s", expectedTitle, result.Title)
	}
}

func TestHandleCreateCAPAFromNCR_NCRNotFound(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	body := `{
		"title": "CAPA for missing NCR",
		"owner": "john.doe"
	}`

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-999/create-capa", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateCAPAFromNCR(w, req, "NCR-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateCAPAFromNCR_InvalidJSON(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, severity, created_at) VALUES
		('NCR-004', 'Test Defect', 'minor', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	body := `{invalid json`

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-004/create-capa", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateCAPAFromNCR(w, req, "NCR-004")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateECOFromNCR_Success(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, ipn, severity, corrective_action, created_at) VALUES
		('NCR-001', 'Circuit Board Defect', 'Shorts detected in production', 'IPN-200', 'critical', 'Redesign PCB layout', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	body := `{
		"title": "ECO for NCR-001",
		"description": "Modify PCB design",
		"priority": "critical",
		"affected_ipns": "IPN-200,IPN-201"
	}`

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-001/create-eco", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateECOFromNCR(w, req, "NCR-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response data is not a map: %+v", response)
	}

	if result["title"].(string) != "ECO for NCR-001" {
		t.Errorf("Expected title 'ECO for NCR-001', got %s", result["title"])
	}

	if result["ncr_id"].(string) != "NCR-001" {
		t.Errorf("Expected ncr_id 'NCR-001', got %s", result["ncr_id"])
	}

	if result["status"].(string) != "draft" {
		t.Errorf("Expected status 'draft', got %s", result["status"])
	}

	if result["priority"].(string) != "critical" {
		t.Errorf("Expected priority 'critical', got %s", result["priority"])
	}
}

func TestHandleCreateECOFromNCR_AutoPopulate(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, ipn, severity, corrective_action, created_at) VALUES
		('NCR-002', 'Component Failure', 'Resistor overheating', 'IPN-300', 'major', 'Use higher wattage resistor', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	// Send request without optional fields - should auto-populate
	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-002/create-eco", nil)
	w := httptest.NewRecorder()

	h.CreateECOFromNCR(w, req, "NCR-002")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response data is not a map: %+v", response)
	}

	// Check auto-populated title
	expectedTitle := "[NCR NCR-002] Component Failure \u2014 Corrective Action"
	if result["title"].(string) != expectedTitle {
		t.Errorf("Expected auto-populated title '%s', got %s", expectedTitle, result["title"])
	}

	// Check auto-populated description from corrective action
	if result["description"].(string) != "Use higher wattage resistor" {
		t.Errorf("Expected description from corrective_action, got %s", result["description"])
	}

	// Check auto-populated priority based on severity (major -> high)
	if result["priority"].(string) != "high" {
		t.Errorf("Expected priority 'high' for major severity, got %s", result["priority"])
	}

	// Check auto-populated affected IPNs from NCR
	if result["affected_ipns"].(string) != "IPN-300" {
		t.Errorf("Expected affected_ipns 'IPN-300', got %s", result["affected_ipns"])
	}
}

func TestHandleCreateECOFromNCR_PriorityMapping(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	testCases := []struct {
		name             string
		severity         string
		expectedPriority string
	}{
		{
			name:             "Critical severity maps to critical priority",
			severity:         "critical",
			expectedPriority: "critical",
		},
		{
			name:             "Major severity maps to high priority",
			severity:         "major",
			expectedPriority: "high",
		},
		{
			name:             "Minor severity maps to normal priority",
			severity:         "minor",
			expectedPriority: "normal",
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ncrID := "NCR-" + string(rune(100+i))

			// Insert NCR with specific severity
			_, err := testDB.Exec(`INSERT INTO ncrs (id, title, severity, created_at) VALUES (?, 'Test NCR', ?, '2026-01-01 10:00:00')`,
				ncrID, tc.severity)
			if err != nil {
				t.Fatalf("Failed to insert test NCR: %v", err)
			}

			req := httptest.NewRequest("POST", "/api/v1/ncrs/"+ncrID+"/create-eco", nil)
			w := httptest.NewRecorder()

			h.CreateECOFromNCR(w, req, ncrID)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			data, ok := response["data"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected 'data' field in response, got %T", response["data"])
			}

			priority, ok := data["priority"].(string)
			if !ok {
				t.Fatalf("Expected priority field to be a string, got %T (value: %v)", data["priority"], data["priority"])
			}
			if priority != tc.expectedPriority {
				t.Errorf("Expected priority '%s' for severity '%s', got '%s'",
					tc.expectedPriority, tc.severity, priority)
			}
		})
	}
}

func TestHandleCreateECOFromNCR_DescriptionFallback(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert NCR with description but no corrective_action
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, severity, created_at) VALUES
		('NCR-003', 'Test NCR', 'This is the NCR description', 'minor', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-003/create-eco", nil)
	w := httptest.NewRecorder()

	h.CreateECOFromNCR(w, req, "NCR-003")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'data' field in response, got %T", response["data"])
	}

	// Description should fall back to "Corrective action for: <description>"
	expectedDescription := "Corrective action for: This is the NCR description"
	description, ok := data["description"].(string)
	if !ok {
		t.Fatalf("Expected description field to be a string, got %T (value: %v)", data["description"], data["description"])
	}
	if description != expectedDescription {
		t.Errorf("Expected fallback description '%s', got '%s'", expectedDescription, description)
	}
}

func TestHandleCreateECOFromNCR_NCRNotFound(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	body := `{
		"title": "ECO for missing NCR"
	}`

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-999/create-eco", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateECOFromNCR(w, req, "NCR-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateECOFromNCR_InvalidJSON(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, severity, created_at) VALUES
		('NCR-004', 'Test NCR', 'minor', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	body := `{invalid json`

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-004/create-eco", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateECOFromNCR(w, req, "NCR-004")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateCAPAFromNCR_DataIntegrity(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, root_cause, corrective_action, created_at) VALUES
		('NCR-005', 'Data Test', 'Description', 'Root Cause', 'Corrective Action', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-005/create-capa", nil)
	w := httptest.NewRecorder()

	h.CreateCAPAFromNCR(w, req, "NCR-005")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'data' field in response, got %T", response["data"])
	}

	// Verify CAPA was actually inserted into database
	var dbCapa models.CAPA
	err = testDB.QueryRow(`SELECT id, title, type, linked_ncr_id, root_cause, action_plan, status
		FROM capas WHERE linked_ncr_id='NCR-005'`).Scan(
		&dbCapa.ID, &dbCapa.Title, &dbCapa.Type, &dbCapa.LinkedNCRID,
		&dbCapa.RootCause, &dbCapa.ActionPlan, &dbCapa.Status)

	if err != nil {
		t.Fatalf("Failed to query CAPA from database: %v", err)
	}

	if dbCapa.ID != data["id"].(string) {
		t.Errorf("Database ID doesn't match response ID")
	}

	if dbCapa.LinkedNCRID != "NCR-005" {
		t.Errorf("CAPA not properly linked to NCR-005")
	}
}

func TestHandleCreateECOFromNCR_DataIntegrity(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, ipn, severity, created_at) VALUES
		('NCR-006', 'Data Test', 'Description', 'IPN-400', 'critical', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-006/create-eco", nil)
	w := httptest.NewRecorder()

	h.CreateECOFromNCR(w, req, "NCR-006")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'data' field in response, got %T", response["data"])
	}

	// Verify ECO was actually inserted into database
	var dbECO struct {
		ID           string
		Title        string
		Status       string
		Priority     string
		NCRID        string
		AffectedIPNs string
	}
	err = testDB.QueryRow(`SELECT id, title, status, priority, ncr_id, COALESCE(affected_ipns, '')
		FROM ecos WHERE ncr_id='NCR-006'`).Scan(
		&dbECO.ID, &dbECO.Title, &dbECO.Status, &dbECO.Priority, &dbECO.NCRID, &dbECO.AffectedIPNs)

	if err != nil {
		t.Fatalf("Failed to query ECO from database: %v", err)
	}

	responseID, ok := data["id"].(string)
	if !ok {
		t.Fatalf("Expected 'id' field to be string, got %T", data["id"])
	}

	if dbECO.ID != responseID {
		t.Errorf("Database ID doesn't match response ID")
	}

	if dbECO.NCRID != "NCR-006" {
		t.Errorf("ECO not properly linked to NCR-006")
	}

	if dbECO.Status != "draft" {
		t.Errorf("Expected status 'draft', got %s", dbECO.Status)
	}
}

func TestHandleCreateCAPAFromNCR_AuditLog(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, severity, created_at) VALUES
		('NCR-007', 'Audit Test', 'minor', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-007/create-capa", nil)
	w := httptest.NewRecorder()

	h.CreateCAPAFromNCR(w, req, "NCR-007")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify audit log entry was created
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module='capa' AND action='created'").Scan(&count)

	if count < 1 {
		t.Errorf("Expected at least 1 audit log entry for CAPA creation, got %d", count)
	}
}

func TestHandleCreateECOFromNCR_AuditLog(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, severity, created_at) VALUES
		('NCR-008', 'Audit Test', 'minor', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-008/create-eco", nil)
	w := httptest.NewRecorder()

	h.CreateECOFromNCR(w, req, "NCR-008")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify audit log entry was created
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module='eco' AND action='created'").Scan(&count)

	if count < 1 {
		t.Errorf("Expected at least 1 audit log entry for ECO creation, got %d", count)
	}
}

func TestHandleCreateCAPAFromNCR_StatusPropagation(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Test that CAPA is created with correct initial status
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, severity, status, created_at) VALUES
		('NCR-011', 'Status Test', 'minor', 'investigating', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-011/create-capa", nil)
	w := httptest.NewRecorder()

	h.CreateCAPAFromNCR(w, req, "NCR-011")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'data' field in response, got %T", response["data"])
	}

	// CAPA should always start with status 'open' regardless of NCR status
	status, ok := data["status"].(string)
	if !ok {
		t.Fatalf("Expected 'status' field to be string, got %T", data["status"])
	}
	if status != "open" {
		t.Errorf("Expected CAPA status 'open', got %s", status)
	}
}

func TestHandleCreateECOFromNCR_StatusPropagation(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Test that ECO is created with correct initial status
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, severity, status, created_at) VALUES
		('NCR-012', 'Status Test', 'critical', 'resolved', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-012/create-eco", nil)
	w := httptest.NewRecorder()

	h.CreateECOFromNCR(w, req, "NCR-012")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'data' field in response, got %T", response["data"])
	}

	// ECO should always start with status 'draft' regardless of NCR status
	status, ok := data["status"].(string)
	if !ok {
		t.Fatalf("Expected 'status' field to be string, got %T", data["status"])
	}
	if status != "draft" {
		t.Errorf("Expected ECO status 'draft', got %s", status)
	}
}

func TestHandleCreateCAPAFromNCR_MultipleFromSameNCR(t *testing.T) {
	testDB := setupNCRIntegrationTestDB(t)
	defer testDB.Close()
	h := newTestHandler(testDB)

	// Insert test NCR
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, severity, created_at) VALUES
		('NCR-013', 'Multiple CAPA Test', 'major', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	// Create first CAPA
	req1 := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-013/create-capa", nil)
	w1 := httptest.NewRecorder()
	h.CreateCAPAFromNCR(w1, req1, "NCR-013")

	if w1.Code != 200 {
		t.Errorf("Expected status 200 for first CAPA, got %d", w1.Code)
	}

	// Create second CAPA from same NCR (should be allowed)
	req2 := httptest.NewRequest("POST", "/api/v1/ncrs/NCR-013/create-capa", nil)
	w2 := httptest.NewRecorder()
	h.CreateCAPAFromNCR(w2, req2, "NCR-013")

	if w2.Code != 200 {
		t.Errorf("Expected status 200 for second CAPA, got %d", w2.Code)
	}

	// Verify both CAPAs exist
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM capas WHERE linked_ncr_id='NCR-013'").Scan(&count)

	if count != 2 {
		t.Errorf("Expected 2 CAPAs linked to NCR-013, got %d", count)
	}
}
