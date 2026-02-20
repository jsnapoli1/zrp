package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func unmarshalResp[T any](body []byte) (T, error) {
	var wrapper struct {
		Data T `json:"data"`
	}
	err := json.Unmarshal(body, &wrapper)
	return wrapper.Data, err
}

func TestCAPACRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// List (empty)
	w := httptest.NewRecorder()
	req := authedRequest("GET", "/api/v1/capas", "", cookie)
	handleListCAPAs(w, req)
	if w.Code != 200 {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}
	list, _ := unmarshalResp[[]CAPA](w.Body.Bytes())
	if len(list) != 0 {
		t.Fatalf("expected 0, got %d", len(list))
	}

	// Create
	body := `{"title":"Fix solder defect","type":"corrective","root_cause":"Insufficient flux","action_plan":"Update profile","owner":"engineer1","due_date":"2026-03-01","linked_ncr_id":"NCR-001"}`
	w = httptest.NewRecorder()
	req = authedRequest("POST", "/api/v1/capas", body, cookie)
	handleCreateCAPA(w, req)
	if w.Code != 200 {
		t.Fatalf("create: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	created, _ := unmarshalResp[CAPA](w.Body.Bytes())
	if created.ID == "" {
		t.Fatalf("expected CAPA ID, response: %s", w.Body.String())
	}
	if created.Title != "Fix solder defect" {
		t.Fatalf("expected 'Fix solder defect', got '%s'", created.Title)
	}
	if created.Status != "open" {
		t.Fatalf("expected 'open', got '%s'", created.Status)
	}
	if created.Type != "corrective" {
		t.Fatalf("expected 'corrective', got '%s'", created.Type)
	}

	// Get
	w = httptest.NewRecorder()
	req = authedRequest("GET", "/api/v1/capas/"+created.ID, "", cookie)
	handleGetCAPA(w, req, created.ID)
	if w.Code != 200 {
		t.Fatalf("get: expected 200, got %d", w.Code)
	}
	fetched, _ := unmarshalResp[CAPA](w.Body.Bytes())
	if fetched.LinkedNCRID != "NCR-001" {
		t.Fatalf("expected 'NCR-001', got '%s'", fetched.LinkedNCRID)
	}

	// Update
	updateBody := `{"title":"Fix solder defect","type":"corrective","status":"in_progress","root_cause":"Insufficient flux","action_plan":"Update profile","owner":"engineer1","due_date":"2026-03-01"}`
	w = httptest.NewRecorder()
	req = authedRequest("PUT", "/api/v1/capas/"+created.ID, updateBody, cookie)
	handleUpdateCAPA(w, req, created.ID)
	if w.Code != 200 {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	updated, _ := unmarshalResp[CAPA](w.Body.Bytes())
	if updated.Status != "in_progress" {
		t.Fatalf("expected 'in_progress', got '%s'", updated.Status)
	}

	// List (should have 1)
	w = httptest.NewRecorder()
	req = authedRequest("GET", "/api/v1/capas", "", cookie)
	handleListCAPAs(w, req)
	list, _ = unmarshalResp[[]CAPA](w.Body.Bytes())
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
}

func TestCAPACloseRequiresEffectivenessAndApproval(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create
	w := httptest.NewRecorder()
	req := authedRequest("POST", "/api/v1/capas", `{"title":"Test close","type":"corrective","owner":"eng"}`, cookie)
	handleCreateCAPA(w, req)
	c, _ := unmarshalResp[CAPA](w.Body.Bytes())

	// Close without effectiveness
	w = httptest.NewRecorder()
	req = authedRequest("PUT", "/api/v1/capas/"+c.ID, `{"title":"Test close","type":"corrective","owner":"eng","status":"closed"}`, cookie)
	handleUpdateCAPA(w, req, c.ID)
	if w.Code != 400 {
		t.Fatalf("expected 400 (no effectiveness), got %d: %s", w.Code, w.Body.String())
	}

	// Close with effectiveness but no approvals
	w = httptest.NewRecorder()
	req = authedRequest("PUT", "/api/v1/capas/"+c.ID, `{"title":"Test close","type":"corrective","owner":"eng","status":"closed","effectiveness_check":"Verified"}`, cookie)
	handleUpdateCAPA(w, req, c.ID)
	if w.Code != 400 {
		t.Fatalf("expected 400 (no approvals), got %d: %s", w.Code, w.Body.String())
	}

	// Close with all requirements
	w = httptest.NewRecorder()
	req = authedRequest("PUT", "/api/v1/capas/"+c.ID, `{"title":"Test close","type":"corrective","owner":"eng","status":"closed","effectiveness_check":"Verified OK","approved_by_qe":"QE Approved","approved_by_mgr":"Manager Approved"}`, cookie)
	handleUpdateCAPA(w, req, c.ID)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	closed, _ := unmarshalResp[CAPA](w.Body.Bytes())
	if closed.Status != "closed" {
		t.Fatalf("expected 'closed', got '%s'", closed.Status)
	}
	if closed.ApprovedByQEAt == nil {
		t.Fatal("expected approved_by_qe_at set")
	}
	if closed.ApprovedByMgrAt == nil {
		t.Fatal("expected approved_by_mgr_at set")
	}
}

func TestCAPADashboard(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	pastDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	futureDate := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

	for i, dd := range []string{pastDate, pastDate, futureDate} {
		w := httptest.NewRecorder()
		body := fmt.Sprintf(`{"title":"CAPA %d","type":"corrective","owner":"eng1","due_date":"%s"}`, i, dd)
		req := httptest.NewRequest("POST", "/api/v1/capas", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		handleCreateCAPA(w, req)
		if w.Code != 200 {
			t.Fatalf("create %d: %d %s", i, w.Code, w.Body.String())
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/capas/dashboard", nil)
	handleCAPADashboard(w, req)
	if w.Code != 200 {
		t.Fatalf("dashboard: expected 200, got %d", w.Code)
	}
	dash, _ := unmarshalResp[map[string]interface{}](w.Body.Bytes())
	if int(dash["total_open"].(float64)) != 3 {
		t.Fatalf("expected 3 open, got %v", dash["total_open"])
	}
	if int(dash["total_overdue"].(float64)) != 2 {
		t.Fatalf("expected 2 overdue, got %v", dash["total_overdue"])
	}
}

func TestCAPAGetNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/capas/CAPA-999", nil)
	handleGetCAPA(w, req, "CAPA-999")
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCAPAPreventiveType(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/capas", strings.NewReader(`{"title":"Prevent recurrence","type":"preventive","linked_rma_id":"RMA-001"}`))
	req.Header.Set("Content-Type", "application/json")
	handleCreateCAPA(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	c, _ := unmarshalResp[CAPA](w.Body.Bytes())
	if c.Type != "preventive" {
		t.Fatalf("expected 'preventive', got '%s'", c.Type)
	}
	if c.LinkedRMAID != "RMA-001" {
		t.Fatalf("expected 'RMA-001', got '%s'", c.LinkedRMAID)
	}
}

func TestCAPADefaultType(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/capas", strings.NewReader(`{"title":"Default type test"}`))
	req.Header.Set("Content-Type", "application/json")
	handleCreateCAPA(w, req)
	c, _ := unmarshalResp[CAPA](w.Body.Bytes())
	if c.Type != "corrective" {
		t.Fatalf("expected default 'corrective', got '%s'", c.Type)
	}
}
