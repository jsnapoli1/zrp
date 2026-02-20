package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupPartChangesTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create part_changes table
	_, err = testDB.Exec(`
		CREATE TABLE part_changes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			part_ipn TEXT NOT NULL,
			eco_id TEXT DEFAULT '',
			field_name TEXT NOT NULL,
			old_value TEXT DEFAULT '',
			new_value TEXT DEFAULT '',
			status TEXT DEFAULT 'draft',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create part_changes table: %v", err)
	}

	// Create ecos table
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT DEFAULT '[]',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
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

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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

	// Create eco_revisions table (needed for ensureInitialRevision)
	_, err = testDB.Exec(`
		CREATE TABLE eco_revisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			eco_id TEXT NOT NULL,
			revision_num INTEGER DEFAULT 1,
			snapshot TEXT DEFAULT '{}',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create eco_revisions table: %v", err)
	}

	return testDB
}

func setupPartsDir(t *testing.T) string {
	tmpDir := t.TempDir()
	oldPartsDir := partsDir
	partsDir = tmpDir
	t.Cleanup(func() { partsDir = oldPartsDir })
	return tmpDir
}

func createTestPartCSV(t *testing.T, dir, filename, ipn string) {
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create CSV: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"IPN", "Description", "Category", "Cost", "Revision"})
	w.Write([]string{ipn, "Test Part", "Electronics", "10.50", "A"})
	w.Flush()
}

func TestHandleCreatePartChanges_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	tmpDir := setupPartsDir(t)
	createTestPartCSV(t, tmpDir, "parts.csv", "TEST-001")

	changes := struct {
		Changes []struct {
			FieldName string `json:"field_name"`
			OldValue  string `json:"old_value"`
			NewValue  string `json:"new_value"`
		} `json:"changes"`
	}{
		Changes: []struct {
			FieldName string `json:"field_name"`
			OldValue  string `json:"old_value"`
			NewValue  string `json:"new_value"`
		}{
			{FieldName: "Description", OldValue: "Test Part", NewValue: "Updated Part"},
			{FieldName: "Cost", OldValue: "10.50", NewValue: "12.00"},
		},
	}
	body, _ := json.Marshal(changes)

	req := httptest.NewRequest("POST", "/api/parts/TEST-001/changes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreatePartChanges(w, req, "TEST-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []PartChange
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp) != 2 {
		t.Errorf("Expected 2 changes created, got %d", len(resp))
	}

	// Verify in DB
	var count int
	db.QueryRow("SELECT COUNT(*) FROM part_changes WHERE part_ipn = ?", "TEST-001").Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 changes in DB, got %d", count)
	}
}

func TestHandleCreatePartChanges_PartNotFound(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	setupPartsDir(t)

	changes := struct {
		Changes []struct {
			FieldName string `json:"field_name"`
			OldValue  string `json:"old_value"`
			NewValue  string `json:"new_value"`
		} `json:"changes"`
	}{
		Changes: []struct {
			FieldName string `json:"field_name"`
			OldValue  string `json:"old_value"`
			NewValue  string `json:"new_value"`
		}{
			{FieldName: "Description", OldValue: "Old", NewValue: "New"},
		},
	}
	body, _ := json.Marshal(changes)

	req := httptest.NewRequest("POST", "/api/parts/NONEXISTENT/changes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreatePartChanges(w, req, "NONEXISTENT")

	if w.Code != 404 {
		t.Errorf("Expected status 404 for nonexistent part, got %d", w.Code)
	}
}

func TestHandleCreatePartChanges_EmptyChanges(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	tmpDir := setupPartsDir(t)
	createTestPartCSV(t, tmpDir, "parts.csv", "TEST-001")

	body := []byte(`{"changes": []}`)

	req := httptest.NewRequest("POST", "/api/parts/TEST-001/changes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreatePartChanges(w, req, "TEST-001")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for empty changes, got %d", w.Code)
	}
}

func TestHandleListPartChanges_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	// Insert test changes
	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Description", "Old", "New", "draft")
	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Cost", "10", "15", "pending")

	req := httptest.NewRequest("GET", "/api/parts/TEST-001/changes", nil)
	w := httptest.NewRecorder()

	handleListPartChanges(w, req, "TEST-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var changes []PartChange
	if err := json.NewDecoder(w.Body).Decode(&changes); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(changes))
	}
}

func TestHandleListPartChanges_FilterByStatus(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Field1", "Old", "New", "draft")
	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Field2", "Old", "New", "pending")
	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Field3", "Old", "New", "applied")

	req := httptest.NewRequest("GET", "/api/parts/TEST-001/changes?status=draft", nil)
	w := httptest.NewRecorder()

	handleListPartChanges(w, req, "TEST-001")

	var changes []PartChange
	json.NewDecoder(w.Body).Decode(&changes)

	if len(changes) != 1 {
		t.Errorf("Expected 1 draft change, got %d", len(changes))
	}
	if len(changes) > 0 && changes[0].Status != "draft" {
		t.Error("Filtered result should only contain draft changes")
	}
}

func TestHandleDeletePartChange_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	res, _ := db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Description", "Old", "New", "draft")
	id, _ := res.LastInsertId()

	req := httptest.NewRequest("DELETE", "/api/parts/TEST-001/changes/1", nil)
	w := httptest.NewRecorder()

	handleDeletePartChange(w, req, "TEST-001", "1")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM part_changes WHERE id = ?", id).Scan(&count)
	if count != 0 {
		t.Error("Change was not deleted")
	}
}

func TestHandleDeletePartChange_OnlyDraft(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO part_changes (id, part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?, ?)",
		1, "TEST-001", "Description", "Old", "New", "pending")

	req := httptest.NewRequest("DELETE", "/api/parts/TEST-001/changes/1", nil)
	w := httptest.NewRecorder()

	handleDeletePartChange(w, req, "TEST-001", "1")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for non-draft, got %d", w.Code)
	}

	// Verify not deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM part_changes WHERE id = 1").Scan(&count)
	if count != 1 {
		t.Error("Non-draft change should not be deleted")
	}
}

func TestHandleDeletePartChange_NotFound(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("DELETE", "/api/parts/TEST-001/changes/999", nil)
	w := httptest.NewRecorder()

	handleDeletePartChange(w, req, "TEST-001", "999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateECOFromChanges_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	// Insert draft changes
	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Description", "Old Desc", "New Desc", "draft")
	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Cost", "10", "15", "draft")

	body := []byte(`{"title": "Test ECO", "description": "Update part TEST-001", "priority": "high"}`)

	req := httptest.NewRequest("POST", "/api/parts/TEST-001/changes/create-eco", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateECOFromChanges(w, req, "TEST-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	ecoID, ok := resp["eco_id"].(string)
	if !ok || ecoID == "" {
		t.Error("Expected eco_id in response")
	}

	// Verify ECO was created
	var title string
	err := db.QueryRow("SELECT title FROM ecos WHERE id = ?", ecoID).Scan(&title)
	if err != nil {
		t.Fatalf("ECO not found: %v", err)
	}
	if title != "Test ECO" {
		t.Errorf("Expected title 'Test ECO', got %s", title)
	}

	// Verify changes were linked and status updated
	var count int
	db.QueryRow("SELECT COUNT(*) FROM part_changes WHERE eco_id = ? AND status = 'pending'", ecoID).Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 changes linked to ECO, got %d", count)
	}
}

func TestHandleCreateECOFromChanges_NoDraftChanges(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	body := []byte(`{"title": "Test ECO"}`)

	req := httptest.NewRequest("POST", "/api/parts/TEST-001/changes/create-eco", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateECOFromChanges(w, req, "TEST-001")

	if w.Code != 400 {
		t.Errorf("Expected status 400 when no draft changes, got %d", w.Code)
	}
}

func TestHandleListECOPartChanges_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	ecoID := "ECO-001"
	db.Exec("INSERT INTO part_changes (part_ipn, eco_id, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?, ?)",
		"TEST-001", ecoID, "Description", "Old", "New", "pending")
	db.Exec("INSERT INTO part_changes (part_ipn, eco_id, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?, ?)",
		"TEST-002", ecoID, "Cost", "10", "15", "pending")

	req := httptest.NewRequest("GET", "/api/ecos/ECO-001/changes", nil)
	w := httptest.NewRecorder()

	handleListECOPartChanges(w, req, ecoID)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var changes []PartChange
	json.NewDecoder(w.Body).Decode(&changes)

	if len(changes) != 2 {
		t.Errorf("Expected 2 changes for ECO, got %d", len(changes))
	}
}

func TestApplyChangesToCSV_Success(t *testing.T) {
	tmpDir := setupPartsDir(t)
	createTestPartCSV(t, tmpDir, "parts.csv", "TEST-001")

	changes := []partFieldChange{
		{id: 1, field: "Description", newValue: "Updated Description"},
		{id: 2, field: "Cost", newValue: "25.00"},
	}

	err := applyChangesToCSV("TEST-001", changes)
	if err != nil {
		t.Fatalf("applyChangesToCSV failed: %v", err)
	}

	// Verify CSV was updated
	path := filepath.Join(tmpDir, "parts.csv")
	f, _ := os.Open(path)
	defer f.Close()
	r := csv.NewReader(f)
	records, _ := r.ReadAll()

	if len(records) < 2 {
		t.Fatal("CSV has insufficient rows")
	}

	// Check updated values
	if records[1][1] != "Updated Description" {
		t.Errorf("Description not updated, got %s", records[1][1])
	}
	if records[1][3] != "25.00" {
		t.Errorf("Cost not updated, got %s", records[1][3])
	}
}

func TestApplyChangesToCSV_PartNotFound(t *testing.T) {
	setupPartsDir(t)

	changes := []partFieldChange{
		{id: 1, field: "Description", newValue: "New"},
	}

	err := applyChangesToCSV("NONEXISTENT", changes)
	if err == nil {
		t.Error("Expected error for nonexistent part")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got %v", err)
	}
}

func TestApplyChangesToCSV_InvalidField(t *testing.T) {
	tmpDir := setupPartsDir(t)
	createTestPartCSV(t, tmpDir, "parts.csv", "TEST-001")

	changes := []partFieldChange{
		{id: 1, field: "NonexistentField", newValue: "Value"},
	}

	// Should not error, just skip invalid field
	err := applyChangesToCSV("TEST-001", changes)
	if err != nil {
		t.Fatalf("applyChangesToCSV failed: %v", err)
	}

	// Verify CSV unchanged
	path := filepath.Join(tmpDir, "parts.csv")
	f, _ := os.Open(path)
	defer f.Close()
	r := csv.NewReader(f)
	records, _ := r.ReadAll()

	// Should still have original description
	if records[1][1] != "Test Part" {
		t.Error("CSV was modified when it shouldn't be")
	}
}

func TestHandleListAllPartChanges_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Field1", "Old", "New", "draft")
	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-002", "Field2", "Old", "New", "draft")
	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-003", "Field3", "Old", "New", "applied")

	req := httptest.NewRequest("GET", "/api/part-changes", nil)
	w := httptest.NewRecorder()

	handleListAllPartChanges(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var changes []PartChange
	json.NewDecoder(w.Body).Decode(&changes)

	// Default status filter is 'draft'
	if len(changes) != 2 {
		t.Errorf("Expected 2 draft changes, got %d", len(changes))
	}
}

func TestHandleListAllPartChanges_StatusFilter(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?)",
		"TEST-001", "Field1", "Old", "New", "applied")

	req := httptest.NewRequest("GET", "/api/part-changes?status=applied", nil)
	w := httptest.NewRecorder()

	handleListAllPartChanges(w, req)

	var changes []PartChange
	json.NewDecoder(w.Body).Decode(&changes)

	if len(changes) != 1 {
		t.Errorf("Expected 1 applied change, got %d", len(changes))
	}
	if len(changes) > 0 && changes[0].Status != "applied" {
		t.Error("Status filter not working")
	}
}

func TestFindPartInCSV_Success(t *testing.T) {
	tmpDir := setupPartsDir(t)
	createTestPartCSV(t, tmpDir, "parts.csv", "TEST-001")

	csvPath, rowIdx, headers, records, err := findPartInCSV("TEST-001")
	if err != nil {
		t.Fatalf("findPartInCSV failed: %v", err)
	}

	if !strings.Contains(csvPath, "parts.csv") {
		t.Errorf("Wrong CSV path: %s", csvPath)
	}
	if rowIdx != 1 {
		t.Errorf("Expected row index 1, got %d", rowIdx)
	}
	if len(headers) == 0 {
		t.Error("Headers empty")
	}
	if len(records) < 2 {
		t.Error("Not enough records")
	}
}

func TestFindPartInCSV_NotFound(t *testing.T) {
	setupPartsDir(t)

	_, _, _, _, err := findPartInCSV("NONEXISTENT")
	if err == nil {
		t.Error("Expected error for nonexistent part")
	}
}

func TestUpdateBOMReferencesForPartIPN_Success(t *testing.T) {
	tmpDir := setupPartsDir(t)

	// Create assembly directory
	assyDir := filepath.Join(tmpDir, "assemblies")
	os.MkdirAll(assyDir, 0755)

	// Create BOM CSV
	bomPath := filepath.Join(assyDir, "PCA-001.csv")
	f, _ := os.Create(bomPath)
	w := csv.NewWriter(f)
	w.Write([]string{"IPN", "Qty", "RefDes"})
	w.Write([]string{"OLD-IPN", "10", "R1-R10"})
	w.Write([]string{"OTHER-IPN", "5", "C1-C5"})
	w.Flush()
	f.Close()

	err := updateBOMReferencesForPartIPN(tmpDir, "OLD-IPN", "NEW-IPN")
	if err != nil {
		t.Fatalf("updateBOMReferencesForPartIPN failed: %v", err)
	}

	// Verify update
	f2, _ := os.Open(bomPath)
	r := csv.NewReader(f2)
	records, _ := r.ReadAll()
	f2.Close()

	found := false
	for _, rec := range records {
		if len(rec) > 0 && rec[0] == "NEW-IPN" {
			found = true
		}
		if len(rec) > 0 && rec[0] == "OLD-IPN" {
			t.Error("Old IPN still present in BOM")
		}
	}
	if !found {
		t.Error("New IPN not found in BOM")
	}
}

func TestRejectPartChangesForECO(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	ecoID := "ECO-001"
	db.Exec("INSERT INTO part_changes (part_ipn, eco_id, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?, ?)",
		"TEST-001", ecoID, "Field1", "Old", "New", "pending")
	db.Exec("INSERT INTO part_changes (part_ipn, eco_id, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?, ?)",
		"TEST-001", ecoID, "Field2", "Old", "New", "pending")

	rejectPartChangesForECO(ecoID)

	// Verify all changes rejected
	var count int
	db.QueryRow("SELECT COUNT(*) FROM part_changes WHERE eco_id = ? AND status = 'rejected'", ecoID).Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 rejected changes, got %d", count)
	}
}

func TestApplyPartChangesForECO_Integration(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupPartChangesTestDB(t)
	defer db.Close()

	tmpDir := setupPartsDir(t)
	createTestPartCSV(t, tmpDir, "parts.csv", "TEST-001")

	ecoID := "ECO-001"
	db.Exec("INSERT INTO part_changes (part_ipn, eco_id, field_name, old_value, new_value, status) VALUES (?, ?, ?, ?, ?, ?)",
		"TEST-001", ecoID, "Description", "Test Part", "Updated Part", "pending")

	err := applyPartChangesForECO(ecoID)
	if err != nil {
		t.Fatalf("applyPartChangesForECO failed: %v", err)
	}

	// Verify status updated to applied
	var status string
	db.QueryRow("SELECT status FROM part_changes WHERE eco_id = ?", ecoID).Scan(&status)
	if status != "applied" {
		t.Errorf("Expected status 'applied', got %s", status)
	}

	// Verify CSV was updated
	path := filepath.Join(tmpDir, "parts.csv")
	f, _ := os.Open(path)
	defer f.Close()
	r := csv.NewReader(f)
	records, _ := r.ReadAll()

	if records[1][1] != "Updated Part" {
		t.Errorf("CSV not updated, got %s", records[1][1])
	}
}

func TestPartChanges_CaseInsensitiveFields(t *testing.T) {
	tmpDir := setupPartsDir(t)
	createTestPartCSV(t, tmpDir, "parts.csv", "TEST-001")

	// Test lowercase field name
	changes := []partFieldChange{
		{id: 1, field: "description", newValue: "Lowercase Field"},
	}

	err := applyChangesToCSV("TEST-001", changes)
	if err != nil {
		t.Fatalf("applyChangesToCSV failed: %v", err)
	}

	// Verify it worked
	path := filepath.Join(tmpDir, "parts.csv")
	f, _ := os.Open(path)
	defer f.Close()
	r := csv.NewReader(f)
	records, _ := r.ReadAll()

	if records[1][1] != "Lowercase Field" {
		t.Error("Case-insensitive field matching failed")
	}
}
