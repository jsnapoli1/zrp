package main

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupDocVersionsTestDB(t *testing.T) func() {
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
			status TEXT DEFAULT 'draft',
			content TEXT,
			file_path TEXT,
			created_by TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create documents table: %v", err)
	}

	// Create document_versions table
	_, err = testDB.Exec(`
		CREATE TABLE document_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT NOT NULL,
			revision TEXT NOT NULL,
			content TEXT,
			file_path TEXT,
			change_summary TEXT,
			status TEXT,
			created_by TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			eco_id TEXT,
			FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create document_versions table: %v", err)
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

	// Save and swap db
	origDB := db
	db = testDB

	return func() {
		db.Close()
		db = origDB
	}
}

func createTestDocument(t *testing.T, id, title, category, ipn, revision, status, content string) {
	_, err := db.Exec(`INSERT INTO documents (id, title, category, ipn, revision, status, content, created_by) 
		VALUES (?, ?, ?, ?, ?, ?, ?, 'admin')`, id, title, category, ipn, revision, status, content)
	if err != nil {
		t.Fatalf("Failed to create test document: %v", err)
	}
}

func TestDocVersionListEmpty(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "draft", "Content")

	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/versions", nil)
	w := httptest.NewRecorder()
	handleListDocVersions(w, req, "DOC-001")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	versions := resp.Data.([]interface{})
	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}
}

func TestDocVersionSnapshot(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "draft", "Version A content")

	// Create snapshot
	err := snapshotDocumentVersion("DOC-001", "Initial snapshot", "admin", nil)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	// Verify snapshot exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM document_versions WHERE document_id='DOC-001'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 version, got %d", count)
	}

	// Verify snapshot content
	var content string
	db.QueryRow("SELECT content FROM document_versions WHERE document_id='DOC-001'").Scan(&content)
	if content != "Version A content" {
		t.Errorf("expected 'Version A content', got %s", content)
	}
}

func TestDocVersionList(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "B", "draft", "Version B content")
	
	// Create two versions
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, change_summary, status, created_by) 
		VALUES ('DOC-001', 'A', 'Version A content', 'Initial version', 'released', 'admin')`)
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, change_summary, status, created_by) 
		VALUES ('DOC-001', 'B', 'Version B content', 'Added section 2', 'draft', 'admin')`)

	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/versions", nil)
	w := httptest.NewRecorder()
	handleListDocVersions(w, req, "DOC-001")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	versionsData, _ := json.Marshal(resp.Data)
	var versions []DocumentVersion
	json.Unmarshal(versionsData, &versions)

	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}

	// Should be in descending order (newest first)
	if versions[0].Revision != "B" {
		t.Errorf("expected first version to be B, got %s", versions[0].Revision)
	}
	if versions[1].Revision != "A" {
		t.Errorf("expected second version to be A, got %s", versions[1].Revision)
	}
}

func TestDocVersionGetByRevision(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "released", "Current content")
	
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, change_summary, status, created_by) 
		VALUES ('DOC-001', 'A', 'Version A content', 'Initial', 'released', 'admin')`)

	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/versions/A", nil)
	w := httptest.NewRecorder()
	handleGetDocVersion(w, req, "DOC-001", "A")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	versionData, _ := json.Marshal(resp.Data)
	var version DocumentVersion
	json.Unmarshal(versionData, &version)

	if version.Revision != "A" {
		t.Errorf("expected revision A, got %s", version.Revision)
	}
	if version.Content != "Version A content" {
		t.Errorf("expected 'Version A content', got %s", version.Content)
	}
}

func TestDocVersionGetNotFound(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "draft", "Content")

	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/versions/Z", nil)
	w := httptest.NewRecorder()
	handleGetDocVersion(w, req, "DOC-001", "Z")

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDocDiff(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "B", "draft", "Current")

	db.Exec(`INSERT INTO document_versions (document_id, revision, content, change_summary, status, created_by) 
		VALUES ('DOC-001', 'A', 'Line 1\nLine 2\nLine 3', 'Version A', 'released', 'admin')`)
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, change_summary, status, created_by) 
		VALUES ('DOC-001', 'B', 'Line 1\nLine 2 modified\nLine 3\nLine 4', 'Version B', 'draft', 'admin')`)

	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/diff?from=A&to=B", nil)
	w := httptest.NewRecorder()
	handleDocDiff(w, req, "DOC-001")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	diffData := resp.Data.(map[string]interface{})

	if diffData["from"] != "A" {
		t.Errorf("expected from=A, got %v", diffData["from"])
	}
	if diffData["to"] != "B" {
		t.Errorf("expected to=B, got %v", diffData["to"])
	}

	lines := diffData["lines"].([]interface{})
	if len(lines) == 0 {
		t.Error("expected diff lines")
	}

	// Check for added and removed lines
	hasAdded := false
	hasRemoved := false
	for _, line := range lines {
		lineMap := line.(map[string]interface{})
		if lineMap["type"] == "added" {
			hasAdded = true
		}
		if lineMap["type"] == "removed" {
			hasRemoved = true
		}
	}
	if !hasAdded {
		t.Error("expected at least one added line")
	}
	if !hasRemoved {
		t.Error("expected at least one removed line")
	}
}

func TestDocDiffMissingParams(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "draft", "Content")

	tests := []struct {
		name  string
		query string
	}{
		{"missing from", "?to=B"},
		{"missing to", "?from=A"},
		{"missing both", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/diff"+tt.query, nil)
			w := httptest.NewRecorder()
			handleDocDiff(w, req, "DOC-001")

			if w.Code != 400 {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestDocDiffFromVersionNotFound(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "B", "draft", "Content")
	
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, created_by) 
		VALUES ('DOC-001', 'B', 'Content B', 'admin')`)

	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/diff?from=A&to=B", nil)
	w := httptest.NewRecorder()
	handleDocDiff(w, req, "DOC-001")

	if w.Code != 404 {
		t.Errorf("expected 404 for missing from version, got %d", w.Code)
	}
}

func TestDocDiffToVersionNotFound(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "draft", "Content")
	
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, created_by) 
		VALUES ('DOC-001', 'A', 'Content A', 'admin')`)

	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/diff?from=A&to=Z", nil)
	w := httptest.NewRecorder()
	handleDocDiff(w, req, "DOC-001")

	if w.Code != 404 {
		t.Errorf("expected 404 for missing to version, got %d", w.Code)
	}
}

func TestDocRelease(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Release Test", "spec", "IPN-001", "A", "draft", "Draft content")

	req := httptest.NewRequest("POST", "/api/v1/docs/DOC-001/release", nil)
	w := httptest.NewRecorder()
	handleReleaseDoc(w, req, "DOC-001")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify document status changed to released
	var status string
	db.QueryRow("SELECT status FROM documents WHERE id='DOC-001'").Scan(&status)
	if status != "released" {
		t.Errorf("expected status 'released', got %s", status)
	}

	// Verify version was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM document_versions WHERE document_id='DOC-001'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 version after release, got %d", count)
	}

	// Verify version has correct content
	var versionContent string
	db.QueryRow("SELECT content FROM document_versions WHERE document_id='DOC-001'").Scan(&versionContent)
	if versionContent != "Draft content" {
		t.Errorf("expected version content 'Draft content', got %s", versionContent)
	}
}

func TestDocReleaseNotFound(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/v1/docs/DOC-999/release", nil)
	w := httptest.NewRecorder()
	handleReleaseDoc(w, req, "DOC-999")

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDocRevert(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Revert Test", "spec", "IPN-001", "B", "draft", "Updated content")

	// Create version A
	_, err := db.Exec(`INSERT INTO document_versions (document_id, revision, content, file_path, created_by) 
		VALUES ('DOC-001', 'A', 'Original content', '/docs/original.pdf', 'admin')`)
	if err != nil {
		t.Fatalf("Failed to insert version: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/docs/DOC-001/revert/A", nil)
	w := httptest.NewRecorder()
	handleRevertDoc(w, req, "DOC-001", "A")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify document content was reverted
	var content, filePath, revision, status string
	db.QueryRow("SELECT content, file_path, revision, status FROM documents WHERE id='DOC-001'").
		Scan(&content, &filePath, &revision, &status)

	if content != "Original content" {
		t.Errorf("expected content 'Original content', got %s", content)
	}
	if filePath != "/docs/original.pdf" {
		t.Errorf("expected file_path '/docs/original.pdf', got %s", filePath)
	}
	if revision != "A" {
		t.Errorf("expected revision A after revert, got %s", revision)
	}
	if status != "draft" {
		t.Errorf("expected status 'draft' after revert, got %s", status)
	}

	// Verify snapshot was created before revert (count should be 2: original + before-revert snapshot)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM document_versions WHERE document_id='DOC-001'").Scan(&count)
	if count < 2 {
		t.Errorf("expected at least 2 versions (original + before-revert), got %d", count)
	}
}

func TestDocRevertNotFound(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "draft", "Content")

	req := httptest.NewRequest("POST", "/api/v1/docs/DOC-001/revert/Z", nil)
	w := httptest.NewRecorder()
	handleRevertDoc(w, req, "DOC-001", "Z")

	if w.Code != 404 {
		t.Errorf("expected 404 for missing version, got %d", w.Code)
	}
}

func TestNextRevisionIncrement(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", "A"},
		{"A", "B"},
		{"B", "C"},
		{"Y", "Z"},
		{"Z", "AA"},
		{"AA", "AB"},
		{"AZ", "BA"},
		{"ZZ", "AAA"},
	}

	for _, tt := range tests {
		got := nextRevision(tt.in)
		if got != tt.out {
			t.Errorf("nextRevision(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}

func TestComputeDiffBasic(t *testing.T) {
	from := []string{"a", "b", "c"}
	to := []string{"a", "x", "c", "d"}
	diff := computeDiff(from, to)

	if len(diff) == 0 {
		t.Fatal("expected diff results")
	}

	// Verify diff contains correct operations
	types := make([]string, len(diff))
	for i, d := range diff {
		types[i] = d.Type
	}

	// Should have same, removed, added, same, added
	hasRemoved := false
	hasAdded := false
	for _, t := range types {
		if t == "removed" {
			hasRemoved = true
		}
		if t == "added" {
			hasAdded = true
		}
	}

	if !hasRemoved {
		t.Error("expected at least one removed line in diff")
	}
	if !hasAdded {
		t.Error("expected at least one added line in diff")
	}
}

func TestComputeDiffIdentical(t *testing.T) {
	from := []string{"a", "b", "c"}
	to := []string{"a", "b", "c"}
	diff := computeDiff(from, to)

	// All lines should be "same"
	for _, line := range diff {
		if line.Type != "same" {
			t.Errorf("expected all lines to be 'same', got %s", line.Type)
		}
	}
}

func TestComputeDiffEmpty(t *testing.T) {
	// Empty to non-empty
	diff1 := computeDiff([]string{}, []string{"a", "b"})
	if len(diff1) != 2 {
		t.Errorf("expected 2 added lines, got %d", len(diff1))
	}
	for _, line := range diff1 {
		if line.Type != "added" {
			t.Errorf("expected 'added', got %s", line.Type)
		}
	}

	// Non-empty to empty
	diff2 := computeDiff([]string{"a", "b"}, []string{})
	if len(diff2) != 2 {
		t.Errorf("expected 2 removed lines, got %d", len(diff2))
	}
	for _, line := range diff2 {
		if line.Type != "removed" {
			t.Errorf("expected 'removed', got %s", line.Type)
		}
	}
}

func TestDocVersionWithECO(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "draft", "Content")

	// Create version with ECO reference
	ecoID := "ECO-2026-001"
	err := snapshotDocumentVersion("DOC-001", "Changed by ECO", "admin", &ecoID)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	// Verify ECO ID is stored
	var storedEcoID string
	db.QueryRow("SELECT COALESCE(eco_id, '') FROM document_versions WHERE document_id='DOC-001'").Scan(&storedEcoID)
	if storedEcoID != ecoID {
		t.Errorf("expected eco_id %s, got %s", ecoID, storedEcoID)
	}

	// Verify version list includes ECO ID
	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/versions", nil)
	w := httptest.NewRecorder()
	handleListDocVersions(w, req, "DOC-001")

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	versionsData, _ := json.Marshal(resp.Data)
	var versions []DocumentVersion
	json.Unmarshal(versionsData, &versions)

	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}

	if versions[0].ECOID == nil {
		t.Fatal("expected ECOID to be set")
	}
	if *versions[0].ECOID != ecoID {
		t.Errorf("expected ECOID %s, got %s", ecoID, *versions[0].ECOID)
	}
}

func TestDocVersionChangeSummary(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "draft", "Content")

	summary := "Added section on safety requirements"
	err := snapshotDocumentVersion("DOC-001", summary, "admin", nil)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	// Verify change summary
	var storedSummary string
	db.QueryRow("SELECT change_summary FROM document_versions WHERE document_id='DOC-001'").Scan(&storedSummary)
	if storedSummary != summary {
		t.Errorf("expected summary %q, got %q", summary, storedSummary)
	}
}

func TestDocVersionMultipleRevisions(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "C", "draft", "Content C")

	// Create multiple versions
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, created_by) VALUES 
		('DOC-001', 'A', 'Content A', 'admin')`)
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, created_by) VALUES 
		('DOC-001', 'B', 'Content B', 'admin')`)
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, created_by) VALUES 
		('DOC-001', 'C', 'Content C', 'admin')`)

	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/versions", nil)
	w := httptest.NewRecorder()
	handleListDocVersions(w, req, "DOC-001")

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	versionsData, _ := json.Marshal(resp.Data)
	var versions []DocumentVersion
	json.Unmarshal(versionsData, &versions)

	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}

	// Verify reverse chronological order
	if versions[0].Revision != "C" || versions[1].Revision != "B" || versions[2].Revision != "A" {
		t.Error("versions not in reverse chronological order")
	}
}

func TestDocVersionFilePath(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "A", "draft", "Content")
	
	// Update file path
	db.Exec(`UPDATE documents SET file_path='/docs/test_v1.pdf' WHERE id='DOC-001'`)

	// Create snapshot
	err := snapshotDocumentVersion("DOC-001", "Initial", "admin", nil)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	// Verify file path was captured
	var filePath string
	db.QueryRow("SELECT file_path FROM document_versions WHERE document_id='DOC-001'").Scan(&filePath)
	if filePath != "/docs/test_v1.pdf" {
		t.Errorf("expected file_path '/docs/test_v1.pdf', got %s", filePath)
	}
}

func TestDocDiffLargeChanges(t *testing.T) {
	cleanup := setupDocVersionsTestDB(t)
	defer cleanup()

	createTestDocument(t, "DOC-001", "Test Doc", "spec", "IPN-001", "B", "draft", "Current")

	// Create two versions with many changes
	fromContent := strings.Repeat("Line %d\n", 50)
	toContent := strings.Repeat("Modified Line %d\n", 50)

	db.Exec(`INSERT INTO document_versions (document_id, revision, content, created_by) 
		VALUES ('DOC-001', 'A', ?, 'admin')`, fromContent)
	db.Exec(`INSERT INTO document_versions (document_id, revision, content, created_by) 
		VALUES ('DOC-001', 'B', ?, 'admin')`, toContent)

	req := httptest.NewRequest("GET", "/api/v1/docs/DOC-001/diff?from=A&to=B", nil)
	w := httptest.NewRecorder()
	handleDocDiff(w, req, "DOC-001")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	diffData := resp.Data.(map[string]interface{})
	lines := diffData["lines"].([]interface{})

	// Should have many diff lines
	if len(lines) < 50 {
		t.Errorf("expected at least 50 diff lines, got %d", len(lines))
	}
}
