package parts_test

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"zrp/internal/handlers/parts"
	"zrp/internal/models"
	"zrp/internal/testutil"
)

func setupTestPartsDirForChanges(t *testing.T) string {
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

	return dir
}

func TestCreatePartChanges(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	partsDir := setupTestPartsDirForChanges(t)
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, partsDir)

	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"10k 0402 Resistor"},{"field_name":"value","old_value":"10k","new_value":"10k 0402"}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreatePartChanges(w, req, "RES-001")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var changes []parts.PartChange
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
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	partsDir := setupTestPartsDirForChanges(t)
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, partsDir)

	body := `{"changes":[{"field_name":"desc","old_value":"x","new_value":"y"}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/parts/NONEXIST/changes", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreatePartChanges(w, req, "NONEXIST")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreatePartChangesEmpty(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, "")

	body := `{"changes":[]}`
	req := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreatePartChanges(w, req, "RES-001")
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListPartChanges(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	partsDir := setupTestPartsDirForChanges(t)
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, partsDir)

	// Create changes first
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"Updated"}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreatePartChanges(w, req, "RES-001")

	// List them
	req2 := testutil.AuthedRequest("GET", "/api/v1/parts/RES-001/changes", nil, cookie)
	w2 := httptest.NewRecorder()
	h.ListPartChanges(w2, req2, "RES-001")
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	var resp models.APIResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var changes []parts.PartChange
	json.Unmarshal(data, &changes)
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}
}

func TestDeletePartChange(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	partsDir := setupTestPartsDirForChanges(t)
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, partsDir)

	// Create
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"Updated"}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreatePartChanges(w, req, "RES-001")

	if w.Code != 200 {
		t.Fatalf("Failed to create part change, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var changes []parts.PartChange
	json.Unmarshal(data, &changes)

	if len(changes) == 0 {
		t.Fatalf("No part changes returned from create")
	}
	id := fmt.Sprintf("%d", changes[0].ID)

	// Delete
	req2 := testutil.AuthedRequest("DELETE", "/api/v1/parts/RES-001/changes/"+id, nil, cookie)
	w2 := httptest.NewRecorder()
	h.DeletePartChange(w2, req2, "RES-001", id)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	// Verify gone
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM part_changes WHERE part_ipn='RES-001'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 changes after delete, got %d", count)
	}
}

func TestDeletePartChangeNotDraft(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	partsDir := setupTestPartsDirForChanges(t)
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, partsDir)

	// Create and manually set to pending
	testDB.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status, created_by, created_at) VALUES ('RES-001','desc','old','new','pending','admin','2026-01-01')")
	var id int64
	testDB.QueryRow("SELECT id FROM part_changes WHERE part_ipn='RES-001'").Scan(&id)

	req := testutil.AuthedRequest("DELETE", fmt.Sprintf("/api/v1/parts/RES-001/changes/%d", id), nil, cookie)
	w := httptest.NewRecorder()
	h.DeletePartChange(w, req, "RES-001", fmt.Sprintf("%d", id))
	if w.Code != 400 {
		t.Errorf("expected 400 for non-draft delete, got %d", w.Code)
	}
}

func TestCreateECOFromChanges(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	partsDir := setupTestPartsDirForChanges(t)
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, partsDir)

	// Create draft changes
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"Updated Resistor"}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreatePartChanges(w, req, "RES-001")

	// Create ECO from changes
	body2 := `{"title":"Update RES-001 description","priority":"high"}`
	req2 := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes/create-eco", []byte(body2), cookie)
	w2 := httptest.NewRecorder()
	h.CreateECOFromChanges(w2, req2, "RES-001")
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp models.APIResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected resp.Data to be map[string]interface{}, got %T", resp.Data)
	}
	ecoIDVal, ok := dataMap["eco_id"]
	if !ok {
		t.Fatal("expected eco_id field in response")
	}
	ecoID, ok := ecoIDVal.(string)
	if !ok {
		t.Fatalf("Expected eco_id to be string, got %T", ecoIDVal)
	}
	if ecoID == "" {
		t.Fatal("expected non-empty eco_id")
	}

	// Verify changes are now pending with eco_id set
	var status, linkedECO string
	testDB.QueryRow("SELECT status, eco_id FROM part_changes WHERE part_ipn='RES-001'").Scan(&status, &linkedECO)
	if status != "pending" {
		t.Errorf("expected pending status, got %s", status)
	}
	if linkedECO != ecoID {
		t.Errorf("expected eco_id %s, got %s", ecoID, linkedECO)
	}
}

func TestCreateECOFromChangesNoDrafts(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, "")

	req := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes/create-eco", []byte(`{}`), cookie)
	w := httptest.NewRecorder()
	h.CreateECOFromChanges(w, req, "RES-001")
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestApplyPartChangesOnECOImplement(t *testing.T) {
	// This test requires handleApproveECO and handleImplementECO which are not part of the parts handler.
	// We test the ApplyPartChangesForECO method directly instead.
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	partsDir := setupTestPartsDirForChanges(t)
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, partsDir)

	// Create draft changes
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"10k 0402 Resistor"}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreatePartChanges(w, req, "RES-001")

	// Create ECO
	body2 := `{"title":"Update RES-001"}`
	req2 := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes/create-eco", []byte(body2), cookie)
	w2 := httptest.NewRecorder()
	h.CreateECOFromChanges(w2, req2, "RES-001")

	if w2.Code != 200 {
		t.Fatalf("Failed to create ECO from changes, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp models.APIResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected resp.Data to be map[string]interface{}, got %T", resp.Data)
	}
	ecoIDVal, ok := dataMap["eco_id"]
	if !ok {
		t.Fatalf("Expected 'eco_id' field in response data")
	}
	ecoID, ok := ecoIDVal.(string)
	if !ok {
		t.Fatalf("Expected 'eco_id' to be string, got %T", ecoIDVal)
	}

	// Instead of calling handleApproveECO/handleImplementECO (not in this package),
	// directly call ApplyPartChangesForECO
	err := h.ApplyPartChangesForECO(ecoID)
	if err != nil {
		t.Fatalf("ApplyPartChangesForECO failed: %v", err)
	}

	// Verify CSV was updated
	fields, err := h.GetPartByIPN(partsDir, "RES-001")
	if err != nil {
		t.Fatal(err)
	}
	if fields["description"] != "10k 0402 Resistor" {
		t.Errorf("expected '10k 0402 Resistor', got %q", fields["description"])
	}

	// Verify change status is applied
	var status string
	testDB.QueryRow("SELECT status FROM part_changes WHERE part_ipn='RES-001'").Scan(&status)
	if status != "applied" {
		t.Errorf("expected applied status, got %s", status)
	}
}

func TestListECOPartChanges(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	partsDir := setupTestPartsDirForChanges(t)
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, partsDir)

	// Create changes and ECO
	body := `{"changes":[{"field_name":"description","old_value":"10k Resistor","new_value":"Updated"}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreatePartChanges(w, req, "RES-001")

	body2 := `{"title":"Test ECO"}`
	req2 := testutil.AuthedRequest("POST", "/api/v1/parts/RES-001/changes/create-eco", []byte(body2), cookie)
	w2 := httptest.NewRecorder()
	h.CreateECOFromChanges(w2, req2, "RES-001")

	if w2.Code != 200 {
		t.Fatalf("Failed to create ECO from changes, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp models.APIResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected resp.Data to be map[string]interface{}, got %T", resp.Data)
	}
	ecoIDVal, ok := dataMap["eco_id"]
	if !ok {
		t.Fatalf("Expected 'eco_id' field in response data")
	}
	ecoID, ok := ecoIDVal.(string)
	if !ok {
		t.Fatalf("Expected 'eco_id' to be string, got %T", ecoIDVal)
	}

	// List ECO part changes
	req3 := testutil.AuthedRequest("GET", fmt.Sprintf("/api/v1/ecos/%s/part-changes", ecoID), nil, cookie)
	w3 := httptest.NewRecorder()
	h.ListECOPartChanges(w3, req3, ecoID)
	if w3.Code != 200 {
		t.Fatalf("expected 200, got %d", w3.Code)
	}

	var resp2 models.APIResponse
	json.Unmarshal(w3.Body.Bytes(), &resp2)
	data, _ := json.Marshal(resp2.Data)
	var changes []parts.PartChange
	json.Unmarshal(data, &changes)
	if len(changes) != 1 {
		t.Errorf("expected 1 change linked to ECO, got %d", len(changes))
	}
}

func TestListAllPartChanges(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()
	cookie := testutil.LoginAdmin(t, testDB)

	h := newTestHandler(testDB, "")

	testDB.Exec("INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status, created_by, created_at) VALUES ('RES-001','desc','old','new','draft','admin','2026-01-01')")

	req := testutil.AuthedRequest("GET", "/api/v1/part-changes", nil, cookie)
	w := httptest.NewRecorder()
	h.ListAllPartChanges(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var changes []parts.PartChange
	json.Unmarshal(data, &changes)
	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}
}
