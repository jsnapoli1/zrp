package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestFieldReportCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// List empty
	req := authedRequest("GET", "/api/v1/field-reports", "", nil)
	w := httptest.NewRecorder()
	handleListFieldReports(w, req)
	if w.Code != 200 {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}
	var listResp APIResponse
	json.Unmarshal(w.Body.Bytes(), &listResp)
	items := listResp.Data.([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}

	// Create
	body := `{"title":"Motor overheating","report_type":"failure","priority":"high","customer_name":"Acme Corp","site_location":"Plant 3","device_ipn":"MOT-001-0001","device_serial":"SN-12345","reported_by":"John","description":"Motor runs hot after 2 hours"}`
	req = authedRequest("POST", "/api/v1/field-reports", body, nil)
	w = httptest.NewRecorder()
	handleCreateFieldReport(w, req)
	if w.Code != 200 {
		t.Fatalf("create: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var createResp APIResponse
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
	req = authedRequest("GET", "/api/v1/field-reports/"+frID, "", nil)
	w = httptest.NewRecorder()
	handleGetFieldReport(w, req, frID)
	if w.Code != 200 {
		t.Fatalf("get: expected 200, got %d", w.Code)
	}
	var getResp APIResponse
	json.Unmarshal(w.Body.Bytes(), &getResp)
	getData := getResp.Data.(map[string]interface{})
	if getData["customer_name"] != "Acme Corp" {
		t.Errorf("expected Acme Corp, got %v", getData["customer_name"])
	}

	// Update
	body = `{"status":"investigating","root_cause":"Bearing failure"}`
	req = authedRequest("PUT", "/api/v1/field-reports/"+frID, body, nil)
	w = httptest.NewRecorder()
	handleUpdateFieldReport(w, req, frID)
	if w.Code != 200 {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var updateResp APIResponse
	json.Unmarshal(w.Body.Bytes(), &updateResp)
	upData := updateResp.Data.(map[string]interface{})
	if upData["status"] != "investigating" {
		t.Errorf("expected investigating, got %v", upData["status"])
	}
	if upData["root_cause"] != "Bearing failure" {
		t.Errorf("expected Bearing failure, got %v", upData["root_cause"])
	}

	// Delete
	req = authedRequest("DELETE", "/api/v1/field-reports/"+frID, "", nil)
	w = httptest.NewRecorder()
	handleDeleteFieldReport(w, req, frID)
	if w.Code != 200 {
		t.Fatalf("delete: expected 200, got %d", w.Code)
	}

	// Verify deleted
	req = authedRequest("GET", "/api/v1/field-reports/"+frID, "", nil)
	w = httptest.NewRecorder()
	handleGetFieldReport(w, req, frID)
	if w.Code != 404 {
		t.Errorf("get deleted: expected 404, got %d", w.Code)
	}
}

func TestFieldReportGetNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := authedRequest("GET", "/api/v1/field-reports/FR-9999", "", nil)
	w := httptest.NewRecorder()
	handleGetFieldReport(w, req, "FR-9999")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFieldReportCreateInvalidBody(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := authedRequest("POST", "/api/v1/field-reports", "not json", nil)
	w := httptest.NewRecorder()
	handleCreateFieldReport(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFieldReportCreateMissingTitle(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := authedRequest("POST", "/api/v1/field-reports", `{"title":""}`, nil)
	w := httptest.NewRecorder()
	handleCreateFieldReport(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFieldReportFilters(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create two reports
	body1 := `{"title":"Failure 1","report_type":"failure","priority":"high","status":"open"}`
	req := authedRequest("POST", "/api/v1/field-reports", body1, nil)
	w := httptest.NewRecorder()
	handleCreateFieldReport(w, req)

	body2 := `{"title":"Visit 1","report_type":"visit","priority":"low","status":"open"}`
	req = authedRequest("POST", "/api/v1/field-reports", body2, nil)
	w = httptest.NewRecorder()
	handleCreateFieldReport(w, req)

	// Filter by type
	req = authedRequest("GET", "/api/v1/field-reports?report_type=failure", "", nil)
	w = httptest.NewRecorder()
	handleListFieldReports(w, req)
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp.Data.([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 failure, got %d", len(items))
	}

	// Filter by priority
	req = authedRequest("GET", "/api/v1/field-reports?priority=low", "", nil)
	w = httptest.NewRecorder()
	handleListFieldReports(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	items = resp.Data.([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 low priority, got %d", len(items))
	}
}

func TestFieldReportResolve(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create
	body := `{"title":"Test issue","report_type":"other"}`
	req := authedRequest("POST", "/api/v1/field-reports", body, nil)
	w := httptest.NewRecorder()
	handleCreateFieldReport(w, req)
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	frID := resp.Data.(map[string]interface{})["id"].(string)

	// Resolve
	body = `{"status":"resolved","resolution":"Fixed the widget","root_cause":"Bad solder joint"}`
	req = authedRequest("PUT", "/api/v1/field-reports/"+frID, body, nil)
	w = httptest.NewRecorder()
	handleUpdateFieldReport(w, req, frID)
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
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create field report
	body := `{"title":"Solder defect in field","report_type":"failure","priority":"critical","device_ipn":"PCA-001-0001","device_serial":"SN-99","description":"Cold solder joint found"}`
	req := authedRequest("POST", "/api/v1/field-reports", body, nil)
	w := httptest.NewRecorder()
	handleCreateFieldReport(w, req)
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	frID := resp.Data.(map[string]interface{})["id"].(string)

	// Create NCR from field report
	req = authedRequest("POST", "/api/v1/field-reports/"+frID+"/create-ncr", "", nil)
	w = httptest.NewRecorder()
	handleFieldReportCreateNCR(w, req, frID)
	if w.Code != 200 {
		t.Fatalf("create-ncr: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var ncrResp APIResponse
	json.Unmarshal(w.Body.Bytes(), &ncrResp)
	ncrData := ncrResp.Data.(map[string]interface{})
	ncrID := ncrData["id"].(string)
	if ncrID == "" {
		t.Fatal("expected NCR ID")
	}
	if ncrData["title"] != "Solder defect in field" {
		t.Errorf("expected NCR title from field report, got %v", ncrData["title"])
	}

	// Verify field report now references the NCR
	req = authedRequest("GET", "/api/v1/field-reports/"+frID, "", nil)
	w = httptest.NewRecorder()
	handleGetFieldReport(w, req, frID)
	json.Unmarshal(w.Body.Bytes(), &resp)
	frData := resp.Data.(map[string]interface{})
	if frData["ncr_id"] != ncrID {
		t.Errorf("expected ncr_id %s, got %v", ncrID, frData["ncr_id"])
	}
}

func TestFieldReportCreateNCRNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := authedRequest("POST", "/api/v1/field-reports/FR-9999/create-ncr", "", nil)
	w := httptest.NewRecorder()
	handleFieldReportCreateNCR(w, req, "FR-9999")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFieldReportECOLink(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create field report with eco_id
	body := `{"title":"Design flaw","report_type":"failure","eco_id":"ECO-2026-001"}`
	req := authedRequest("POST", "/api/v1/field-reports", body, nil)
	w := httptest.NewRecorder()
	handleCreateFieldReport(w, req)
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	frData := resp.Data.(map[string]interface{})
	if frData["eco_id"] != "ECO-2026-001" {
		t.Errorf("expected eco_id ECO-2026-001, got %v", frData["eco_id"])
	}
}
