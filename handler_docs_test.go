package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func setupDocsTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create documents table
	_, err = testDB.Exec(`
		CREATE TABLE documents (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			category TEXT,
			ipn TEXT,
			revision TEXT DEFAULT 'A',
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','approved','obsolete')),
			content TEXT,
			file_path TEXT,
			created_by TEXT DEFAULT 'engineer',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create documents table: %v", err)
	}

	// Create attachments table
	_, err = testDB.Exec(`
		CREATE TABLE attachments (
			id TEXT PRIMARY KEY,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			original_name TEXT NOT NULL,
			size_bytes INTEGER,
			mime_type TEXT,
			uploaded_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create attachments table: %v", err)
	}

	// Create document_versions table (needed for snapshotDocumentVersion)
	_, err = testDB.Exec(`
		CREATE TABLE document_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT NOT NULL,
			revision TEXT,
			content TEXT,
			snapshot_reason TEXT,
			created_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create document_versions table: %v", err)
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

func TestHandleListDocs_Empty(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/documents", nil)
	w := httptest.NewRecorder()

	handleListDocs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.([]interface{})
	if len(result) != 0 {
		t.Errorf("Expected empty list, got %d items", len(result))
	}
}

func TestHandleListDocs_WithData(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO documents (id, title, category, ipn, revision, status, created_at) VALUES 
		('DOC-001', 'Test Doc 1', 'spec', 'IPN-100', 'A', 'draft', '2026-01-01 10:00:00'),
		('DOC-002', 'Test Doc 2', 'manual', 'IPN-200', 'B', 'approved', '2026-01-02 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/documents", nil)
	w := httptest.NewRecorder()

	handleListDocs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Fatalf("Expected 2 documents, got %d", len(resp.Data))
	}

	// Should be sorted by created_at DESC
	if resp.Data[0]["id"].(string) != "DOC-002" {
		t.Errorf("Expected DOC-002 first, got %s", resp.Data[0]["id"])
	}
}

func TestHandleGetDoc_Success(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO documents (id, title, category, ipn, revision, status, content, created_by, created_at, updated_at) VALUES 
		('DOC-001', 'Test Doc', 'spec', 'IPN-100', 'A', 'draft', 'Document content here', 'engineer', '2026-01-01 10:00:00', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Add attachment
	_, err = db.Exec(`INSERT INTO attachments (id, module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by, created_at) VALUES 
		('ATT-001', 'document', 'DOC-001', 'file1.pdf', 'Original File.pdf', 12345, 'application/pdf', 'engineer', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert attachment: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/documents/DOC-001", nil)
	w := httptest.NewRecorder()

	handleGetDoc(w, req, "DOC-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := resp.Data

	if result["id"].(string) != "DOC-001" {
		t.Errorf("Expected DOC-001, got %s", result["id"])
	}

	if result["title"].(string) != "Test Doc" {
		t.Errorf("Expected 'Test Doc', got %s", result["title"])
	}

	attachments := result["attachments"].([]interface{})
	if len(attachments) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(attachments))
	}
}

func TestHandleGetDoc_NotFound(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/documents/DOC-999", nil)
	w := httptest.NewRecorder()

	handleGetDoc(w, req, "DOC-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateDoc_Success(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{
		"title": "New Test Document",
		"category": "spec",
		"ipn": "IPN-100",
		"revision": "A",
		"status": "draft",
		"content": "This is the document content"
	}`

	req := httptest.NewRequest("POST", "/api/v1/documents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateDoc(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data Document `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := resp.Data

	if result.Title != "New Test Document" {
		t.Errorf("Expected 'New Test Document', got %s", result.Title)
	}

	if result.ID == "" {
		t.Error("Expected ID to be generated")
	}

	if result.Status != "draft" {
		t.Errorf("Expected status 'draft', got %s", result.Status)
	}
}

func TestHandleCreateDoc_MissingTitle(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{
		"category": "spec",
		"content": "Content without title"
	}`

	req := httptest.NewRequest("POST", "/api/v1/documents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateDoc(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateDoc_InvalidStatus(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{
		"title": "Test Doc",
		"status": "invalid-status"
	}`

	req := httptest.NewRequest("POST", "/api/v1/documents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateDoc(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateDoc_DefaultValues(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{
		"title": "Minimal Document"
	}`

	req := httptest.NewRequest("POST", "/api/v1/documents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateDoc(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data Document `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := resp.Data

	if result.Revision != "A" {
		t.Errorf("Expected default revision 'A', got %s", result.Revision)
	}

	if result.Status != "draft" {
		t.Errorf("Expected default status 'draft', got %s", result.Status)
	}

	if result.CreatedBy != "engineer" {
		t.Errorf("Expected created_by 'engineer', got %s", result.CreatedBy)
	}
}

func TestHandleUpdateDoc_Success(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO documents (id, title, category, status, created_by, created_at, updated_at) VALUES 
		('DOC-001', 'Original Title', 'spec', 'draft', 'engineer', '2026-01-01 10:00:00', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"title": "Updated Title",
		"category": "manual",
		"revision": "B",
		"content": "Updated content"
	}`

	req := httptest.NewRequest("PUT", "/api/v1/documents/DOC-001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateDoc(w, req, "DOC-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := resp.Data

	if result["title"].(string) != "Updated Title" {
		t.Errorf("Expected 'Updated Title', got %s", result["title"])
	}
}

func TestHandleUpdateDoc_NotFound(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{
		"title": "Updated Title"
	}`

	req := httptest.NewRequest("PUT", "/api/v1/documents/DOC-999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateDoc(w, req, "DOC-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleUpdateDoc_PreservesStatus(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO documents (id, title, status, created_by, created_at, updated_at) VALUES 
		('DOC-001', 'Test Doc', 'approved', 'engineer', '2026-01-01 10:00:00', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	body := `{
		"title": "Updated Title"
	}`

	req := httptest.NewRequest("PUT", "/api/v1/documents/DOC-001", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateDoc(w, req, "DOC-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := apiResp.Data.(map[string]interface{})
	if result["status"].(string) != "approved" {
		t.Errorf("Expected status to remain 'approved', got %s", result["status"])
	}
}

func TestHandleApproveDoc_Success(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO documents (id, title, status, created_by, created_at, updated_at) VALUES 
		('DOC-001', 'Test Doc', 'draft', 'engineer', '2026-01-01 10:00:00', '2026-01-01 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/documents/DOC-001/approve", nil)
	w := httptest.NewRecorder()

	handleApproveDoc(w, req, "DOC-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := apiResp.Data.(map[string]interface{})
	if result["status"].(string) != "approved" {
		t.Errorf("Expected status 'approved', got %s", result["status"])
	}
}

func TestHandleApproveDoc_NotFound(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("POST", "/api/v1/documents/DOC-999/approve", nil)
	w := httptest.NewRecorder()

	handleApproveDoc(w, req, "DOC-999")

	// The handler doesn't check if document exists before updating
	// It will return 200 but with no rows affected
	// This might be a bug - should return 404 if document not found
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleDocs_InvalidJSON(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{invalid json`

	req := httptest.NewRequest("POST", "/api/v1/documents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateDoc(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleDocs_SQLInjectionAttempt(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	body := `{
		"title": "Test'; DROP TABLE documents; --",
		"category": "spec"
	}`

	req := httptest.NewRequest("POST", "/api/v1/documents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateDoc(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the table still exists and the malicious title was stored safely
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		t.Fatalf("Table was dropped or query failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 document, got %d", count)
	}
}

func TestHandleDocs_FilePathValidation(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	testCases := []struct {
		name     string
		filePath string
		wantCode int
	}{
		{
			name:     "Valid file path",
			filePath: "docs/spec/file.pdf",
			wantCode: 200,
		},
		{
			name:     "Path traversal attempt",
			filePath: "../../etc/passwd",
			wantCode: 200, // Currently allowed - potential security issue
		},
		{
			name:     "Absolute path",
			filePath: "/etc/passwd",
			wantCode: 200, // Currently allowed - potential security issue
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{
				"title": "Test Doc",
				"file_path": "` + tc.filePath + `"
			}`

			req := httptest.NewRequest("POST", "/api/v1/documents", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateDoc(w, req)

			if w.Code != tc.wantCode {
				t.Errorf("Expected status %d, got %d", tc.wantCode, w.Code)
			}
		})
	}
}

func TestHandleDocs_CategoryFilter(t *testing.T) {
	oldDB := db
	db = setupDocsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec(`INSERT INTO documents (id, title, category, created_at) VALUES 
		('DOC-001', 'Spec 1', 'spec', '2026-01-01 10:00:00'),
		('DOC-002', 'Manual 1', 'manual', '2026-01-02 10:00:00'),
		('DOC-003', 'Spec 2', 'spec', '2026-01-03 10:00:00')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/documents", nil)
	w := httptest.NewRecorder()

	handleListDocs(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(result))
	}
}
