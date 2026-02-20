package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func setupTestPartsDirForChanges(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	// Create a category dir with a CSV
	catDir := filepath.Join(dir, "resistors")
	os.MkdirAll(catDir, 0755)
	f, _ := os.Create(filepath.Join(catDir, "resistors.csv"))
	w := csv.NewWriter(f)
	w.WriteAll([][]string{
		{"IPN", "description", "manufacturer", "value"},
		{"RES-001", "10k Resistor", "Yageo", "10k"},
		{"RES-002", "100k Resistor", "Yageo", "100k"},
	})
	w.Flush()
	f.Close()

	oldPartsDir := partsDir
	partsDir = dir
	return func() {
		partsDir = oldPartsDir
	}
}

func TestCreatePartChanges(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cleanupParts := setupTestPartsDirForChanges(t)
	defer cleanupParts()
	cookie := loginAdmin(t)

	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"10k 0402 Resistor"},{"field_name":"value","old_value":"10k","new_value":"10k 0402"}]}`
	req := authedRequest("POST", "/api/v1/parts/RES-001/changes", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePartChanges(w, req, "RES-001")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var changes []PartChange
	json.Unmarshal(data, &changes)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	if changes[0].Status != "draft" {
		t.Errorf("expected draft status, got %s", changes[0].Status)
	}
	if changes[0].FieldName != "description" {
		t.Errorf("expected description field, got %s", changes[0].FieldName)
	}
}

func TestCreatePartChangesPartNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cleanupParts := setupTestPartsDirForChanges(t)
	defer cleanupParts()
	cookie := loginAdmin(t)

	body := `{"changes":[{"field_name":"desc","old_value":"x","new_value":"y"}]}`
	req := authedRequest("POST", "/api/v1/parts/NONEXIST/changes", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePartChanges(w, req, "NONEXIST")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreatePartChangesEmpty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"changes":[]}`
	req := authedRequest("POST", "/api/v1/parts/RES-001/changes", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePartChanges(w, req, "RES-001")
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListPartChanges(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cleanupParts := setupTestPartsDirForChanges(t)
	defer cleanupParts()
	cookie := loginAdmin(t)

	// Create changes first
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"Updated"}]}`
	req := authedRequest("POST", "/api/v1/parts/RES-001/changes", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePartChanges(w, req, "RES-001")

	// List them
	req2 := authedRequest("GET", "/api/v1/parts/RES-001/changes", "", cookie)
	w2 := httptest.NewRecorder()
	handleListPartChanges(w2, req2, "RES-001")
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var resp APIResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var changes []PartChange
	json.Unmarshal(data, &changes)
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}
}

func TestDeletePartChange(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cleanupParts := setupTestPartsDirForChanges(t)
	defer cleanupParts()
	cookie := loginAdmin(t)

	// Create
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"Updated"}]}`
	req := authedRequest("POST", "/api/v1/parts/RES-001/changes", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePartChanges(w, req, "RES-001")
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var changes []PartChange
	json.Unmarshal(data, &changes)
	id := fmt.Sprintf("%d", changes[0].ID)

	// Delete
	req2 := authedRequest("DELETE", "/api/v1/parts/RES-001/changes/"+id, "", cookie)
	w2 := httptest.NewRecorder()
	handleDeletePartChange(w2, req2, "RES-001", id)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Verify gone
	var count int
	db.QueryRow("SELECT COUNT(*) FROM part_changes WHERE part_ipn='RES-001'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 changes after delete, got %d", count)
	}
}

func TestDeletePartChangeNotDraft(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cleanupParts := setupTestPartsDirForChanges(t)
	defer cleanupParts()
	cookie := loginAdmin(t)

	// Create and manually set to pending
	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status, created_by, created_at) VALUES ('RES-001','desc','old','new','pending','admin','2026-01-01')")
	var id int64
	db.QueryRow("SELECT id FROM part_changes WHERE part_ipn='RES-001'").Scan(&id)

	req := authedRequest("DELETE", fmt.Sprintf("/api/v1/parts/RES-001/changes/%d", id), "", cookie)
	w := httptest.NewRecorder()
	handleDeletePartChange(w, req, "RES-001", fmt.Sprintf("%d", id))
	if w.Code != 400 {
		t.Errorf("expected 400 for non-draft delete, got %d", w.Code)
	}
}

func TestCreateECOFromChanges(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cleanupParts := setupTestPartsDirForChanges(t)
	defer cleanupParts()
	cookie := loginAdmin(t)

	// Create draft changes
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"Updated Resistor"}]}`
	req := authedRequest("POST", "/api/v1/parts/RES-001/changes", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePartChanges(w, req, "RES-001")

	// Create ECO from changes
	body2 := `{"title":"Update RES-001 description","priority":"high"}`
	req2 := authedRequest("POST", "/api/v1/parts/RES-001/changes/create-eco", body2, cookie)
	w2 := httptest.NewRecorder()
	handleCreateECOFromChanges(w2, req2, "RES-001")
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp APIResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	ecoID := data["eco_id"].(string)
	if ecoID == "" {
		t.Fatal("expected eco_id")
	}

	// Verify changes are now pending with eco_id set
	var status, linkedECO string
	db.QueryRow("SELECT status, eco_id FROM part_changes WHERE part_ipn='RES-001'").Scan(&status, &linkedECO)
	if status != "pending" {
		t.Errorf("expected pending status, got %s", status)
	}
	if linkedECO != ecoID {
		t.Errorf("expected eco_id %s, got %s", ecoID, linkedECO)
	}
}

func TestCreateECOFromChangesNoDrafts(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("POST", "/api/v1/parts/RES-001/changes/create-eco", `{}`, cookie)
	w := httptest.NewRecorder()
	handleCreateECOFromChanges(w, req, "RES-001")
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestApplyPartChangesOnECOImplement(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cleanupParts := setupTestPartsDirForChanges(t)
	defer cleanupParts()
	cookie := loginAdmin(t)

	// Create draft changes
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"10k 0402 Resistor"}]}`
	req := authedRequest("POST", "/api/v1/parts/RES-001/changes", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePartChanges(w, req, "RES-001")

	// Create ECO
	body2 := `{"title":"Update RES-001"}`
	req2 := authedRequest("POST", "/api/v1/parts/RES-001/changes/create-eco", body2, cookie)
	w2 := httptest.NewRecorder()
	handleCreateECOFromChanges(w2, req2, "RES-001")
	var resp APIResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)
	ecoID := resp.Data.(map[string]interface{})["eco_id"].(string)

	// Approve ECO
	req3 := authedRequest("POST", fmt.Sprintf("/api/v1/ecos/%s/approve", ecoID), "", cookie)
	w3 := httptest.NewRecorder()
	handleApproveECO(w3, req3, ecoID)
	if w3.Code != 200 {
		t.Fatalf("approve failed: %d %s", w3.Code, w3.Body.String())
	}

	// Implement ECO - should apply CSV changes
	req4 := authedRequest("POST", fmt.Sprintf("/api/v1/ecos/%s/implement", ecoID), "", cookie)
	w4 := httptest.NewRecorder()
	handleImplementECO(w4, req4, ecoID)
	if w4.Code != 200 {
		t.Fatalf("implement failed: %d %s", w4.Code, w4.Body.String())
	}

	// Verify CSV was updated
	fields, err := getPartByIPN(partsDir, "RES-001")
	if err != nil {
		t.Fatal(err)
	}
	if fields["description"] != "10k 0402 Resistor" {
		t.Errorf("expected '10k 0402 Resistor', got %q", fields["description"])
	}

	// Verify change status is applied
	var status string
	db.QueryRow("SELECT status FROM part_changes WHERE part_ipn='RES-001'").Scan(&status)
	if status != "applied" {
		t.Errorf("expected applied status, got %s", status)
	}
}

func TestListECOPartChanges(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cleanupParts := setupTestPartsDirForChanges(t)
	defer cleanupParts()
	cookie := loginAdmin(t)

	// Create changes and ECO
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"Updated"}]}`
	req := authedRequest("POST", "/api/v1/parts/RES-001/changes", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePartChanges(w, req, "RES-001")

	body2 := `{"title":"Test ECO"}`
	req2 := authedRequest("POST", "/api/v1/parts/RES-001/changes/create-eco", body2, cookie)
	w2 := httptest.NewRecorder()
	handleCreateECOFromChanges(w2, req2, "RES-001")
	var resp APIResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)
	ecoID := resp.Data.(map[string]interface{})["eco_id"].(string)

	// List ECO part changes
	req3 := authedRequest("GET", fmt.Sprintf("/api/v1/ecos/%s/part-changes", ecoID), "", cookie)
	w3 := httptest.NewRecorder()
	handleListECOPartChanges(w3, req3, ecoID)
	if w3.Code != 200 {
		t.Fatalf("expected 200, got %d", w3.Code)
	}

	var resp2 APIResponse
	json.Unmarshal(w3.Body.Bytes(), &resp2)
	data, _ := json.Marshal(resp2.Data)
	var changes []PartChange
	json.Unmarshal(data, &changes)
	if len(changes) != 1 {
		t.Errorf("expected 1 change linked to ECO, got %d", len(changes))
	}
}

func TestListAllPartChanges(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	db.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status, created_by, created_at) VALUES ('RES-001','desc','old','new','draft','admin','2026-01-01')")

	req := authedRequest("GET", "/api/v1/part-changes", "", cookie)
	w := httptest.NewRecorder()
	handleListAllPartChanges(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var changes []PartChange
	json.Unmarshal(data, &changes)
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}
}
