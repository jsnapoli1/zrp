package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupNCRTestDB(t *testing.T) *sql.DB {
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

	// Create ecos table (needed for linked ECO creation)
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
			user TEXT,
			action TEXT,
			entity_type TEXT,
			entity_id TEXT,
			details TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
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

func TestHandleListNCRs_Empty(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/ncrs", nil)
	w := httptest.NewRecorder()

	handleListNCRs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []NCR
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty list, got %d items", len(result))
	}
}

func TestHandleListNCRs_WithData(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ncrs (id, title, description, severity, status) VALUES 
		('NCR-001', 'Defect 1', 'Description 1', 'minor', 'open'),
		('NCR-002', 'Defect 2', 'Description 2', 'critical', 'investigating')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/ncrs", nil)
	w := httptest.NewRecorder()

	handleListNCRs(w, req)

	var result []NCR
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	// Should be ordered by created_at DESC
	if result[0].ID != "NCR-002" {
		t.Errorf("Expected NCR-002 first, got %s", result[0].ID)
	}
}

func TestHandleGetNCR_Success(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ncrs (id, title, description, ipn, severity, status) VALUES 
		('NCR-001', 'Test NCR', 'Test Description', 'IPN-001', 'major', 'open')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/ncrs/NCR-001", nil)
	w := httptest.NewRecorder()

	handleGetNCR(w, req, "NCR-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result NCR
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.ID != "NCR-001" {
		t.Errorf("Expected ID NCR-001, got %s", result.ID)
	}
	if result.Title != "Test NCR" {
		t.Errorf("Expected title 'Test NCR', got %s", result.Title)
	}
	if result.Severity != "major" {
		t.Errorf("Expected severity 'major', got %s", result.Severity)
	}
}

func TestHandleGetNCR_NotFound(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/ncrs/NCR-999", nil)
	w := httptest.NewRecorder()

	handleGetNCR(w, req, "NCR-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateNCR_Success(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"title": "New NCR",
		"description": "Test description",
		"ipn": "IPN-001",
		"serial_number": "SN-12345",
		"defect_type": "visual",
		"severity": "critical",
		"status": "open"
	}`
	req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateNCR(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result NCR
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.ID == "" {
		t.Error("Expected ID to be generated")
	}
	if result.Title != "New NCR" {
		t.Errorf("Expected title 'New NCR', got %s", result.Title)
	}
	if result.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got %s", result.Severity)
	}

	// Verify audit log
	var auditCount int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE entity_type='ncr'").Scan(&auditCount)
	if auditCount != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", auditCount)
	}
}

func TestHandleCreateNCR_MissingTitle(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"description": "Test description",
		"severity": "minor"
	}`
	req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateNCR(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateNCR_InvalidSeverity(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"title": "Test NCR",
		"severity": "invalid_severity"
	}`
	req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateNCR(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateNCR_DefaultValues(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{"title": "Test NCR"}`
	req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateNCR(w, req)

	var result NCR
	json.NewDecoder(w.Body).Decode(&result)

	if result.Status != "open" {
		t.Errorf("Expected default status 'open', got %s", result.Status)
	}
	if result.Severity != "minor" {
		t.Errorf("Expected default severity 'minor', got %s", result.Severity)
	}
}

func TestHandleUpdateNCR_Success(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ncrs (id, title, description, severity, status) VALUES 
		('NCR-001', 'Original Title', 'Original Description', 'minor', 'open')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	reqBody := `{
		"title": "Updated Title",
		"description": "Updated Description",
		"severity": "major",
		"status": "investigating",
		"root_cause": "Component failure"
	}`
	req := httptest.NewRequest("PUT", "/api/v1/ncrs/NCR-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateNCR(w, req, "NCR-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify database was updated
	var title, status, rootCause string
	db.QueryRow("SELECT title, status, root_cause FROM ncrs WHERE id=?", "NCR-001").Scan(&title, &status, &rootCause)

	if title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %s", title)
	}
	if status != "investigating" {
		t.Errorf("Expected status 'investigating', got %s", status)
	}
	if rootCause != "Component failure" {
		t.Errorf("Expected root_cause 'Component failure', got %s", rootCause)
	}
}

func TestHandleUpdateNCR_ResolveSetsTimestamp(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ncrs (id, title, status) VALUES ('NCR-001', 'Test NCR', 'investigating')`)

	reqBody := `{
		"title": "Test NCR",
		"severity": "minor",
		"status": "resolved",
		"corrective_action": "Replaced component"
	}`
	req := httptest.NewRequest("PUT", "/api/v1/ncrs/NCR-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateNCR(w, req, "NCR-001")

	// Verify resolved_at was set
	var resolvedAt sql.NullString
	db.QueryRow("SELECT resolved_at FROM ncrs WHERE id=?", "NCR-001").Scan(&resolvedAt)

	if !resolvedAt.Valid {
		t.Error("Expected resolved_at to be set when status is resolved")
	}
}

func TestHandleUpdateNCR_AutoCreateECO(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ncrs (id, title, ipn, status) VALUES 
		('NCR-001', 'Test NCR', 'IPN-001', 'investigating')
	`)

	reqBody := `{
		"title": "Test NCR",
		"ipn": "IPN-001",
		"severity": "major",
		"status": "resolved",
		"corrective_action": "Update assembly process",
		"create_eco": true
	}`
	req := httptest.NewRequest("PUT", "/api/v1/ncrs/NCR-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateNCR(w, req, "NCR-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	// Verify linked ECO was created
	if result["linked_eco_id"] == nil || result["linked_eco_id"] == "" {
		t.Error("Expected linked_eco_id to be set")
	}

	// Verify ECO exists in database
	linkedECOID := result["linked_eco_id"].(string)
	var ecoTitle string
	err := db.QueryRow("SELECT title FROM ecos WHERE id=?", linkedECOID).Scan(&ecoTitle)
	if err != nil {
		t.Fatalf("Expected ECO to exist: %v", err)
	}

	if ecoTitle == "" {
		t.Error("Expected ECO to have a title")
	}

	// Verify ECO is linked to NCR
	var ncrID string
	db.QueryRow("SELECT ncr_id FROM ecos WHERE id=?", linkedECOID).Scan(&ncrID)
	if ncrID != "NCR-001" {
		t.Errorf("Expected ECO to be linked to NCR-001, got %s", ncrID)
	}
}

func TestHandleUpdateNCR_NoECOWithoutCreateFlag(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ncrs (id, title, status) VALUES ('NCR-001', 'Test NCR', 'investigating')`)

	reqBody := `{
		"title": "Test NCR",
		"severity": "major",
		"status": "resolved",
		"corrective_action": "Update process",
		"create_eco": false
	}`
	req := httptest.NewRequest("PUT", "/api/v1/ncrs/NCR-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateNCR(w, req, "NCR-001")

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	// Verify no linked ECO was created
	if result["linked_eco_id"] != nil {
		t.Error("Expected no linked_eco_id when create_eco is false")
	}

	// Verify no ECO exists
	var ecoCount int
	db.QueryRow("SELECT COUNT(*) FROM ecos").Scan(&ecoCount)
	if ecoCount != 0 {
		t.Errorf("Expected 0 ECOs, got %d", ecoCount)
	}
}

func TestHandleUpdateNCR_NoECOWithoutCorrectiveAction(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ncrs (id, title, status) VALUES ('NCR-001', 'Test NCR', 'investigating')`)

	reqBody := `{
		"title": "Test NCR",
		"severity": "major",
		"status": "resolved",
		"create_eco": true
	}`
	req := httptest.NewRequest("PUT", "/api/v1/ncrs/NCR-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateNCR(w, req, "NCR-001")

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	// Verify no linked ECO was created without corrective action
	if result["linked_eco_id"] != nil {
		t.Error("Expected no linked_eco_id when corrective_action is empty")
	}
}

func TestHandleUpdateNCR_AllFields(t *testing.T) {
	oldDB := db
	db = setupNCRTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ncrs (id, title) VALUES ('NCR-001', 'Test')`)

	reqBody := `{
		"title": "Full NCR",
		"description": "Full description",
		"ipn": "IPN-001",
		"serial_number": "SN-123",
		"defect_type": "dimensional",
		"severity": "critical",
		"status": "investigating",
		"root_cause": "Tooling wear",
		"corrective_action": "Replace tooling"
	}`
	req := httptest.NewRequest("PUT", "/api/v1/ncrs/NCR-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateNCR(w, req, "NCR-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify all fields were updated
	var title, description, ipn, serialNumber, defectType, severity, status, rootCause, correctiveAction string
	db.QueryRow("SELECT title, description, ipn, serial_number, defect_type, severity, status, root_cause, corrective_action FROM ncrs WHERE id=?", "NCR-001").
		Scan(&title, &description, &ipn, &serialNumber, &defectType, &severity, &status, &rootCause, &correctiveAction)

	if title != "Full NCR" {
		t.Errorf("Expected title 'Full NCR', got %s", title)
	}
	if ipn != "IPN-001" {
		t.Errorf("Expected ipn 'IPN-001', got %s", ipn)
	}
	if serialNumber != "SN-123" {
		t.Errorf("Expected serial_number 'SN-123', got %s", serialNumber)
	}
	if severity != "critical" {
		t.Errorf("Expected severity 'critical', got %s", severity)
	}
}
