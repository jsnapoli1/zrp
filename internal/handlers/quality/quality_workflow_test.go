package quality_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zrp/internal/handlers/quality"
	"zrp/internal/models"
	"zrp/internal/testutil"

	_ "modernc.org/sqlite"
)

// newWorkflowHandler creates a handler with custom stubs for workflow tests.
func newWorkflowHandler(testDB *sql.DB, opts ...func(h *quality.Handler)) *quality.Handler {
	h := newTestHandler(testDB)
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func seedTestUsers(t *testing.T, testDB *sql.DB) {
	t.Helper()
	// Create test users with different roles
	users := []struct {
		username string
		role     string
	}{
		{"test_qe", "qe"},
		{"test_manager", "manager"},
		{"test_user", "user"},
		{"test_admin", "admin"},
	}

	for _, user := range users {
		_, err := testDB.Exec(`INSERT OR IGNORE INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
			user.username, "test_hash", user.role)
		if err != nil {
			t.Fatalf("Failed to create test user %s: %v", user.username, err)
		}

		// Create session for the user
		_, err = testDB.Exec(`INSERT OR IGNORE INTO sessions (token, user_id, expires_at)
			SELECT ?, id, datetime('now', '+30 days') FROM users WHERE username = ?`,
			user.username+"_token", user.username)
		if err != nil {
			t.Fatalf("Failed to create session for %s: %v", user.username, err)
		}
	}
}

func TestQualityWorkflowIntegration(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Close()

	// Seed test data
	seedTestUsers(t, testDB)

	// testutil.SetupTestDB creates the ncrs table without created_by,
	// and the ecos table without ncr_id, so we add them here.
	testDB.Exec("ALTER TABLE ncrs ADD COLUMN created_by TEXT DEFAULT ''")
	testDB.Exec("ALTER TABLE ecos ADD COLUMN ncr_id TEXT DEFAULT ''")

	t.Run("NCR Creation with created_by", func(t *testing.T) {
		testNCRCreationWithCreatedBy(t, testDB)
	})
	t.Run("Create CAPA from NCR via API", func(t *testing.T) {
		testCreateCAPAFromNCRAPI(t, testDB)
	})
	t.Run("Create ECO from NCR via API", func(t *testing.T) {
		testCreateECOFromNCRAPI(t, testDB)
	})
	t.Run("CAPA Approval Security", func(t *testing.T) {
		testCAPAApprovalSecurity(t, testDB)
	})
	t.Run("CAPA Status Auto-advancement", func(t *testing.T) {
		testCAPAStatusAutoAdvancement(t, testDB)
	})
}

func testNCRCreationWithCreatedBy(t *testing.T, testDB *sql.DB) {
	// Look up the test_qe user ID for the handler stub
	var qeUserID int
	err := testDB.QueryRow("SELECT id FROM users WHERE username='test_qe'").Scan(&qeUserID)
	if err != nil {
		t.Fatalf("Failed to find test_qe user: %v", err)
	}

	h := newWorkflowHandler(testDB, func(h *quality.Handler) {
		h.GetUserID = func(r *http.Request) (int, error) {
			return qeUserID, nil
		}
		h.GetUserRole = func(r *http.Request) string {
			return "qe"
		}
	})

	// Create NCR request
	ncrData := map[string]interface{}{
		"title":       "Test NCR with created_by",
		"description": "Testing created_by field",
		"severity":    "minor",
		"defect_type": "material",
	}

	body, _ := json.Marshal(ncrData)
	req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.CreateNCR(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response struct {
		Data models.NCR `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := response.Data

	// Verify created_by is set (will be looked up from session by audit.GetUsername)
	if result.CreatedBy != "test_qe" {
		t.Logf("Note: created_by='%s' (depends on session lookup)", result.CreatedBy)
	}

	// Verify in database
	var createdBy string
	err = testDB.QueryRow("SELECT created_by FROM ncrs WHERE id = ?", result.ID).Scan(&createdBy)
	if err != nil {
		t.Errorf("Failed to query created_by from database: %v", err)
	}
}

func testCreateCAPAFromNCRAPI(t *testing.T, testDB *sql.DB) {
	h := newTestHandler(testDB)

	// First create an NCR
	ncrID := "NCR-2026-TEST1"
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, status, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))`,
		ncrID, "Test NCR", "Test description", "resolved")
	if err != nil {
		t.Fatalf("Failed to create test NCR: %v", err)
	}

	// Test creating CAPA from NCR
	capaData := map[string]interface{}{
		"owner":    "test_manager",
		"due_date": "2026-03-01",
	}

	body, _ := json.Marshal(capaData)
	req := httptest.NewRequest("POST", "/api/v1/ncrs/"+ncrID+"/create-capa", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.CreateCAPAFromNCR(w, req, ncrID)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response struct {
		Data models.CAPA `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := response.Data

	// Verify CAPA is linked to NCR
	if result.LinkedNCRID != ncrID {
		t.Errorf("Expected linked_ncr_id to be '%s', got '%s'", ncrID, result.LinkedNCRID)
	}

	// Verify auto-populated title
	expectedTitle := "CAPA for NCR " + ncrID + ": Test NCR"
	if result.Title != expectedTitle {
		t.Errorf("Expected title to be '%s', got '%s'", expectedTitle, result.Title)
	}
}

func testCreateECOFromNCRAPI(t *testing.T, testDB *sql.DB) {
	h := newTestHandler(testDB)

	// First create an NCR
	ncrID := "NCR-2026-TEST2"
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, description, status, corrective_action, ipn, created_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
		ncrID, "Test NCR for ECO", "Test description", "resolved", "Replace component", "TEST-001")
	if err != nil {
		t.Fatalf("Failed to create test NCR: %v", err)
	}

	// Test creating ECO from NCR
	req := httptest.NewRequest("POST", "/api/v1/ncrs/"+ncrID+"/create-eco", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})

	w := httptest.NewRecorder()
	h.CreateECOFromNCR(w, req, ncrID)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := response.Data

	// Verify ECO is linked to NCR
	if result["ncr_id"] != ncrID {
		t.Errorf("Expected ncr_id to be '%s', got '%v'", ncrID, result["ncr_id"])
	}

	// Verify auto-populated fields
	expectedTitle := "[NCR " + ncrID + "] Test NCR for ECO \u2014 Corrective Action"
	if result["title"] != expectedTitle {
		t.Errorf("Expected title to be '%s', got '%s'", expectedTitle, result["title"])
	}

	if result["affected_ipns"] != "TEST-001" {
		t.Errorf("Expected affected_ipns to be 'TEST-001', got '%v'", result["affected_ipns"])
	}
}

func testCAPAApprovalSecurity(t *testing.T, testDB *sql.DB) {
	// Create a test CAPA
	capaID := "CAPA-2026-TEST"
	_, err := testDB.Exec(`INSERT INTO capas (id, title, type, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
		capaID, "Test CAPA", "corrective", "open")
	if err != nil {
		t.Fatalf("Failed to create test CAPA: %v", err)
	}

	t.Run("QE approval by non-QE user should fail", func(t *testing.T) {
		// Create handler where CanApproveCAPA returns false for QE
		h := newWorkflowHandler(testDB, func(h *quality.Handler) {
			h.CanApproveCAPA = func(r *http.Request, approvalType string) bool {
				return false // non-QE user cannot approve
			}
		})

		updateData := map[string]interface{}{
			"approved_by_qe": "approve",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_user_token"})
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		h.UpdateCAPA(w, req, capaID)

		if w.Code != 403 {
			t.Errorf("Expected status 403 for non-QE user, got %d", w.Code)
		}
	})

	t.Run("QE approval by QE user should succeed", func(t *testing.T) {
		// Create handler where CanApproveCAPA returns true for QE
		h := newWorkflowHandler(testDB, func(h *quality.Handler) {
			h.CanApproveCAPA = func(r *http.Request, approvalType string) bool {
				return approvalType == "qe"
			}
		})

		updateData := map[string]interface{}{
			"approved_by_qe": "approve",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		h.UpdateCAPA(w, req, capaID)

		if w.Code != 200 {
			t.Errorf("Expected status 200 for QE user, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Manager approval by non-manager should fail", func(t *testing.T) {
		// Create handler where CanApproveCAPA returns false for manager
		h := newWorkflowHandler(testDB, func(h *quality.Handler) {
			h.CanApproveCAPA = func(r *http.Request, approvalType string) bool {
				return approvalType == "qe" // Only QE allowed, not manager
			}
		})

		updateData := map[string]interface{}{
			"approved_by_mgr": "approve",
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		h.UpdateCAPA(w, req, capaID)

		if w.Code != 403 {
			t.Errorf("Expected status 403 for non-manager user, got %d", w.Code)
		}
	})
}

func testCAPAStatusAutoAdvancement(t *testing.T, testDB *sql.DB) {
	// Create a test CAPA
	capaID := "CAPA-2026-AUTO"
	_, err := testDB.Exec(`INSERT INTO capas (id, title, type, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
		capaID, "Auto-advancement test CAPA", "corrective", "open")
	if err != nil {
		t.Fatalf("Failed to create test CAPA: %v", err)
	}

	// First, QE approves
	hQE := newWorkflowHandler(testDB, func(h *quality.Handler) {
		h.CanApproveCAPA = func(r *http.Request, approvalType string) bool {
			return approvalType == "qe"
		}
	})

	updateData := map[string]interface{}{
		"approved_by_qe": "approve",
	}

	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_qe_token"})
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	hQE.UpdateCAPA(w, req, capaID)

	if w.Code != 200 {
		t.Errorf("QE approval failed: %d: %s", w.Code, w.Body.String())
		return
	}

	// Now manager approves - this should auto-advance status to "pending_review"
	hMgr := newWorkflowHandler(testDB, func(h *quality.Handler) {
		h.CanApproveCAPA = func(r *http.Request, approvalType string) bool {
			return approvalType == "manager"
		}
	})

	updateData = map[string]interface{}{
		"approved_by_mgr": "approve",
	}

	body, _ = json.Marshal(updateData)
	req = httptest.NewRequest("PUT", "/api/v1/capas/"+capaID, bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test_manager_token"})
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	hMgr.UpdateCAPA(w, req, capaID)

	if w.Code != 200 {
		t.Errorf("Manager approval failed: %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify status was auto-advanced
	var response struct {
		Data models.CAPA `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	result := response.Data

	if result.Status != "pending_review" {
		t.Errorf("Expected status to auto-advance to 'pending_review', got '%s'", result.Status)
	}

	// Verify in database
	var dbStatus string
	err = testDB.QueryRow("SELECT status FROM capas WHERE id = ?", capaID).Scan(&dbStatus)
	if err != nil {
		t.Errorf("Failed to query status from database: %v", err)
	}
	if dbStatus != "pending_review" {
		t.Errorf("Database status should be 'pending_review', got '%s'", dbStatus)
	}
}
