package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupECOTestDB(t *testing.T) *sql.DB {
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
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','review','approved','implemented','rejected','cancelled')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			affected_ipns TEXT,
			created_by TEXT DEFAULT 'engineer',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_at DATETIME,
			approved_by TEXT,
			ncr_id TEXT DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	// Create eco_revisions table
	_, err = testDB.Exec(`
		CREATE TABLE eco_revisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			eco_id TEXT NOT NULL,
			revision TEXT NOT NULL,
			status TEXT DEFAULT 'created',
			changes_summary TEXT,
			created_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_by TEXT,
			approved_at DATETIME,
			implemented_by TEXT,
			implemented_at DATETIME,
			effectivity_date TEXT,
			notes TEXT,
			FOREIGN KEY (eco_id) REFERENCES ecos(id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create eco_revisions table: %v", err)
	}

	// Create audit_log table (needed for logAudit)
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

	// Create part_changes table (needed for recordChangeJSON)
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

	return testDB
}

func TestHandleListECOs_Empty(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/ecos", nil)
	w := httptest.NewRecorder()

	handleListECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Data []ECO `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Data) != 0 {
		t.Errorf("Expected empty list, got %d items", len(response.Data))
	}
}

func TestHandleListECOs_WithData(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert test ECOs
	_, err := db.Exec(`INSERT INTO ecos (id, title, description, status, priority, created_by) VALUES 
		('ECO-001', 'Test ECO 1', 'Description 1', 'draft', 'normal', 'engineer'),
		('ECO-002', 'Test ECO 2', 'Description 2', 'approved', 'high', 'manager')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/ecos", nil)
	w := httptest.NewRecorder()

	handleListECOs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []ECO
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result[0].Title != "Test ECO 2" {
		t.Errorf("Expected first item to be ECO-002 (ordered by created_at DESC), got %s", result[0].ID)
	}
}

func TestHandleListECOs_FilterByStatus(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title, status) VALUES 
		('ECO-001', 'Draft ECO', 'draft'),
		('ECO-002', 'Approved ECO', 'approved'),
		('ECO-003', 'Another Draft', 'draft')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/ecos?status=draft", nil)
	w := httptest.NewRecorder()

	handleListECOs(w, req)

	var result []ECO
	json.NewDecoder(w.Body).Decode(&result)

	if len(result) != 2 {
		t.Errorf("Expected 2 draft ECOs, got %d", len(result))
	}

	for _, eco := range result {
		if eco.Status != "draft" {
			t.Errorf("Expected all ECOs to have draft status, got %s", eco.Status)
		}
	}
}

func TestHandleGetECO_Success(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title, description, status, priority) VALUES 
		('ECO-001', 'Test ECO', 'Test Description', 'draft', 'high')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/ecos/ECO-001", nil)
	w := httptest.NewRecorder()

	handleGetECO(w, req, "ECO-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["id"] != "ECO-001" {
		t.Errorf("Expected id ECO-001, got %v", result["id"])
	}
	if result["title"] != "Test ECO" {
		t.Errorf("Expected title 'Test ECO', got %v", result["title"])
	}
}

func TestHandleGetECO_NotFound(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/ecos/ECO-999", nil)
	w := httptest.NewRecorder()

	handleGetECO(w, req, "ECO-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateECO_Success(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Initialize sequence counter
	db.Exec("CREATE TABLE IF NOT EXISTS id_sequences (prefix TEXT PRIMARY KEY, next_num INTEGER)")

	reqBody := `{
		"title": "New ECO",
		"description": "Test description",
		"status": "draft",
		"priority": "high",
		"affected_ipns": "IPN-001,IPN-002"
	}`
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result ECO
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.ID == "" {
		t.Error("Expected ID to be generated")
	}
	if result.Title != "New ECO" {
		t.Errorf("Expected title 'New ECO', got %s", result.Title)
	}
	if result.Status != "draft" {
		t.Errorf("Expected status 'draft', got %s", result.Status)
	}
	if result.Priority != "high" {
		t.Errorf("Expected priority 'high', got %s", result.Priority)
	}

	// Verify audit log
	var auditCount int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE entity_type='eco'").Scan(&auditCount)
	if auditCount != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", auditCount)
	}

	// Verify initial revision was created
	var revCount int
	db.QueryRow("SELECT COUNT(*) FROM eco_revisions WHERE eco_id=?", result.ID).Scan(&revCount)
	if revCount != 1 {
		t.Errorf("Expected 1 initial revision, got %d", revCount)
	}
}

func TestHandleCreateECO_MissingTitle(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"description": "Test description",
		"status": "draft"
	}`
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateECO_InvalidStatus(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"title": "Test ECO",
		"status": "invalid_status"
	}`
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateECO_DefaultValues(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec("CREATE TABLE IF NOT EXISTS id_sequences (prefix TEXT PRIMARY KEY, next_num INTEGER)")

	reqBody := `{"title": "Test ECO"}`
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	var result ECO
	json.NewDecoder(w.Body).Decode(&result)

	if result.Status != "draft" {
		t.Errorf("Expected default status 'draft', got %s", result.Status)
	}
	if result.Priority != "normal" {
		t.Errorf("Expected default priority 'normal', got %s", result.Priority)
	}
}

func TestHandleUpdateECO_Success(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO ecos (id, title, description, status, priority) VALUES 
		('ECO-001', 'Original Title', 'Original Description', 'draft', 'normal')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	reqBody := `{
		"title": "Updated Title",
		"description": "Updated Description",
		"status": "review",
		"priority": "high"
	}`
	req := httptest.NewRequest("PUT", "/api/v1/ecos/ECO-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateECO(w, req, "ECO-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify database was updated
	var title, status, priority string
	db.QueryRow("SELECT title, status, priority FROM ecos WHERE id=?", "ECO-001").Scan(&title, &status, &priority)

	if title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %s", title)
	}
	if status != "review" {
		t.Errorf("Expected status 'review', got %s", status)
	}
	if priority != "high" {
		t.Errorf("Expected priority 'high', got %s", priority)
	}
}

func TestHandleUpdateECO_NotFound(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{"title": "Updated Title", "status": "draft", "priority": "normal"}`
	req := httptest.NewRequest("PUT", "/api/v1/ecos/ECO-999", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateECO(w, req, "ECO-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleUpdateECO_ValidationError(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test')`)

	reqBody := `{"title": "", "status": "draft", "priority": "normal"}`
	req := httptest.NewRequest("PUT", "/api/v1/ecos/ECO-001", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleUpdateECO(w, req, "ECO-001")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleApproveECO_Success(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ecos (id, title, status) VALUES ('ECO-001', 'Test ECO', 'review')`)
	db.Exec(`INSERT INTO eco_revisions (eco_id, revision, status) VALUES ('ECO-001', 'A', 'created')`)

	req := httptest.NewRequest("POST", "/api/v1/ecos/ECO-001/approve", nil)
	w := httptest.NewRecorder()

	handleApproveECO(w, req, "ECO-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify status was updated
	var status string
	var approvedBy sql.NullString
	db.QueryRow("SELECT status, approved_by FROM ecos WHERE id=?", "ECO-001").Scan(&status, &approvedBy)

	if status != "approved" {
		t.Errorf("Expected status 'approved', got %s", status)
	}
	if !approvedBy.Valid {
		t.Error("Expected approved_by to be set")
	}

	// Verify revision was updated
	var revStatus string
	db.QueryRow("SELECT status FROM eco_revisions WHERE eco_id=?", "ECO-001").Scan(&revStatus)
	if revStatus != "approved" {
		t.Errorf("Expected revision status 'approved', got %s", revStatus)
	}
}

func TestHandleImplementECO_Success(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ecos (id, title, status) VALUES ('ECO-001', 'Test ECO', 'approved')`)
	db.Exec(`INSERT INTO eco_revisions (eco_id, revision, status) VALUES ('ECO-001', 'A', 'approved')`)

	req := httptest.NewRequest("POST", "/api/v1/ecos/ECO-001/implement", nil)
	w := httptest.NewRecorder()

	handleImplementECO(w, req, "ECO-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify status was updated
	var status string
	db.QueryRow("SELECT status FROM ecos WHERE id=?", "ECO-001").Scan(&status)

	if status != "implemented" {
		t.Errorf("Expected status 'implemented', got %s", status)
	}

	// Verify revision was updated
	var revStatus string
	db.QueryRow("SELECT status FROM eco_revisions WHERE eco_id=?", "ECO-001").Scan(&revStatus)
	if revStatus != "implemented" {
		t.Errorf("Expected revision status 'implemented', got %s", revStatus)
	}
}

func TestHandleListECORevisions_Empty(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test ECO')`)

	req := httptest.NewRequest("GET", "/api/v1/ecos/ECO-001/revisions", nil)
	w := httptest.NewRecorder()

	handleListECORevisions(w, req, "ECO-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []ECORevision
	json.NewDecoder(w.Body).Decode(&result)

	if len(result) != 0 {
		t.Errorf("Expected empty list, got %d items", len(result))
	}
}

func TestHandleListECORevisions_WithData(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test ECO')`)
	db.Exec(`INSERT INTO eco_revisions (eco_id, revision, status, changes_summary) VALUES 
		('ECO-001', 'A', 'created', 'Initial revision'),
		('ECO-001', 'B', 'approved', 'Updated revision')
	`)

	req := httptest.NewRequest("GET", "/api/v1/ecos/ECO-001/revisions", nil)
	w := httptest.NewRecorder()

	handleListECORevisions(w, req, "ECO-001")

	var result []ECORevision
	json.NewDecoder(w.Body).Decode(&result)

	if len(result) != 2 {
		t.Errorf("Expected 2 revisions, got %d", len(result))
	}

	if result[0].Revision != "A" {
		t.Errorf("Expected first revision to be A, got %s", result[0].Revision)
	}
	if result[1].Revision != "B" {
		t.Errorf("Expected second revision to be B, got %s", result[1].Revision)
	}
}

func TestHandleCreateECORevision_Success(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test ECO')`)
	db.Exec(`INSERT INTO eco_revisions (eco_id, revision) VALUES ('ECO-001', 'A')`)

	reqBody := `{
		"changes_summary": "Updated components",
		"effectivity_date": "2024-12-31",
		"notes": "Test notes"
	}`
	req := httptest.NewRequest("POST", "/api/v1/ecos/ECO-001/revisions", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleCreateECORevision(w, req, "ECO-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["revision"] != "B" {
		t.Errorf("Expected revision B, got %v", result["revision"])
	}

	// Verify database entry
	var revision, summary string
	db.QueryRow("SELECT revision, changes_summary FROM eco_revisions WHERE eco_id=? ORDER BY id DESC LIMIT 1", "ECO-001").Scan(&revision, &summary)

	if revision != "B" {
		t.Errorf("Expected revision B in DB, got %s", revision)
	}
	if summary != "Updated components" {
		t.Errorf("Expected summary 'Updated components', got %s", summary)
	}
}

func TestHandleGetECORevision_Success(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test ECO')`)
	db.Exec(`INSERT INTO eco_revisions (eco_id, revision, status, changes_summary) VALUES 
		('ECO-001', 'A', 'created', 'Initial revision')
	`)

	req := httptest.NewRequest("GET", "/api/v1/ecos/ECO-001/revisions/A", nil)
	w := httptest.NewRecorder()

	handleGetECORevision(w, req, "ECO-001", "A")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result ECORevision
	json.NewDecoder(w.Body).Decode(&result)

	if result.Revision != "A" {
		t.Errorf("Expected revision A, got %s", result.Revision)
	}
	if result.ChangesSummary != "Initial revision" {
		t.Errorf("Expected changes summary 'Initial revision', got %s", result.ChangesSummary)
	}
}

func TestHandleGetECORevision_NotFound(t *testing.T) {
	oldDB := db
	db = setupECOTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	db.Exec(`INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test ECO')`)

	req := httptest.NewRequest("GET", "/api/v1/ecos/ECO-001/revisions/Z", nil)
	w := httptest.NewRecorder()

	handleGetECORevision(w, req, "ECO-001", "Z")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
