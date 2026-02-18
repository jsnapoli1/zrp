package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"os"
	"strings"
	"testing"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	dbFile := fmt.Sprintf("test_%s.db", t.Name())
	os.Remove(dbFile)
	if err := initDB(dbFile); err != nil {
		t.Fatal(err)
	}
	seedDB()
	os.MkdirAll("uploads", 0755)
	return func() { os.Remove(dbFile) }
}

// loginAdmin logs in as admin and returns the session cookie
func loginAdmin(t *testing.T) *http.Cookie {
	t.Helper()
	body := `{"username":"admin","password":"changeme"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleLogin(w, req)
	if w.Code != 200 {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "zrp_session" {
			return c
		}
	}
	t.Fatal("no session cookie")
	return nil
}

func authedRequest(method, path string, body string, cookie *http.Cookie) *http.Request {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return req
}

// --- Auth Tests ---

func TestLoginSuccess(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	if cookie.Value == "" {
		t.Error("empty session token")
	}
}

func TestLoginFailure(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	body := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleLogin(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLogout(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handleLogout(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Session should be invalid now
	req2 := httptest.NewRequest("GET", "/auth/me", nil)
	req2.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	handleMe(w2, req2)
	if w2.Code != 401 {
		t.Errorf("expected 401 after logout, got %d", w2.Code)
	}
}

func TestMe(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handleMe(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	user := resp["user"].(map[string]interface{})
	if user["username"] != "admin" {
		t.Errorf("expected admin, got %v", user["username"])
	}
}

func TestMeUnauthorized(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/auth/me", nil)
	w := httptest.NewRecorder()
	handleMe(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// --- User Tests ---

func TestCreateUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	body := `{"username":"newuser","display_name":"New User","password":"pass123","role":"user"}`
	req := authedRequest("POST", "/api/v1/users", body, cookie)
	w := httptest.NewRecorder()
	handleCreateUser(w, req)
	if w.Code != 201 {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateDuplicateUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	body := `{"username":"admin","display_name":"Dup","password":"pass","role":"user"}`
	req := authedRequest("POST", "/api/v1/users", body, cookie)
	w := httptest.NewRecorder()
	handleCreateUser(w, req)
	if w.Code != 409 {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestUpdateUserRole(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	// Get engineer user ID
	var engineerID int
	db.QueryRow("SELECT id FROM users WHERE username='engineer'").Scan(&engineerID)

	body := `{"display_name":"Engineer Updated","role":"readonly","active":1}`
	req := authedRequest("PUT", fmt.Sprintf("/api/v1/users/%d", engineerID), body, cookie)
	w := httptest.NewRecorder()
	handleUpdateUser(w, req, fmt.Sprintf("%d", engineerID))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify role changed
	var role string
	db.QueryRow("SELECT role FROM users WHERE id=?", engineerID).Scan(&role)
	if role != "readonly" {
		t.Errorf("expected readonly, got %s", role)
	}
}

func TestDeactivateUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	var engineerID int
	db.QueryRow("SELECT id FROM users WHERE username='engineer'").Scan(&engineerID)

	body := `{"display_name":"Engineer","role":"user","active":0}`
	req := authedRequest("PUT", fmt.Sprintf("/api/v1/users/%d", engineerID), body, cookie)
	w := httptest.NewRecorder()
	handleUpdateUser(w, req, fmt.Sprintf("%d", engineerID))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Deactivated user can't login
	loginBody := `{"username":"engineer","password":"changeme"}`
	req2 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handleLogin(w2, req2)
	if w2.Code != 403 {
		t.Errorf("expected 403 for deactivated user, got %d", w2.Code)
	}
}

// --- API Key Tests ---

func TestAPIKeyLifecycle(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)

	// Create key
	body := `{"name":"Test Key"}`
	req := authedRequest("POST", "/api/v1/apikeys", body, cookie)
	w := httptest.NewRecorder()
	handleCreateAPIKey(w, req)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	key := resp["key"].(string)
	if !strings.HasPrefix(key, "zrp_") {
		t.Errorf("key should start with zrp_, got %s", key)
	}

	// Verify Bearer auth works
	if !validateBearerToken(key) {
		t.Error("valid key should authenticate")
	}
}

func TestBearerAuthWorks(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	body := `{"name":"Bearer Test"}`
	req := authedRequest("POST", "/api/v1/apikeys", body, cookie)
	w := httptest.NewRecorder()
	handleCreateAPIKey(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	key := resp["key"].(string)

	if !validateBearerToken(key) {
		t.Error("Bearer token should be valid")
	}
}

func TestRevokedKeyRejected(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)

	// Create key
	body := `{"name":"Revoke Test"}`
	req := authedRequest("POST", "/api/v1/apikeys", body, cookie)
	w := httptest.NewRecorder()
	handleCreateAPIKey(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	key := resp["key"].(string)
	id := fmt.Sprintf("%.0f", resp["id"].(float64))

	// Delete key
	req2 := authedRequest("DELETE", "/api/v1/apikeys/"+id, "", cookie)
	w2 := httptest.NewRecorder()
	handleDeleteAPIKey(w2, req2, id)
	if w2.Code != 200 {
		t.Errorf("expected 200, got %d", w2.Code)
	}

	// Key should no longer work
	if validateBearerToken(key) {
		t.Error("revoked key should not authenticate")
	}
}

func TestDisabledKeyRejected(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)

	body := `{"name":"Disable Test"}`
	req := authedRequest("POST", "/api/v1/apikeys", body, cookie)
	w := httptest.NewRecorder()
	handleCreateAPIKey(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	key := resp["key"].(string)
	id := fmt.Sprintf("%.0f", resp["id"].(float64))

	// Disable key
	req2 := authedRequest("PUT", "/api/v1/apikeys/"+id, `{"enabled":0}`, cookie)
	w2 := httptest.NewRecorder()
	handleToggleAPIKey(w2, req2, id)

	if validateBearerToken(key) {
		t.Error("disabled key should not authenticate")
	}
}

// --- Attachment Tests ---

func TestAttachmentLifecycle(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)

	// Upload
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("module", "eco")
	mw.WriteField("record_id", "ECO-2026-001")
	fw, _ := mw.CreateFormFile("file", "test.txt")
	fw.Write([]byte("hello world"))
	mw.Close()

	req := httptest.NewRequest("POST", "/api/v1/attachments", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handleUploadAttachment(w, req)
	if w.Code != 201 {
		t.Fatalf("upload expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var uploadResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &uploadResp)
	data := uploadResp["data"].(map[string]interface{})
	attID := fmt.Sprintf("%.0f", data["id"].(float64))

	// List
	req2 := httptest.NewRequest("GET", "/api/v1/attachments?module=eco&record_id=ECO-2026-001", nil)
	req2.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	handleListAttachments(w2, req2)
	if w2.Code != 200 {
		t.Errorf("list expected 200, got %d", w2.Code)
	}

	var listResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &listResp)
	items := listResp["data"].([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(items))
	}

	// Delete
	req3 := authedRequest("DELETE", "/api/v1/attachments/"+attID, "", cookie)
	w3 := httptest.NewRecorder()
	handleDeleteAttachment(w3, req3, attID)
	if w3.Code != 200 {
		t.Errorf("delete expected 200, got %d", w3.Code)
	}
}

// --- Bulk Operation Tests ---

func TestBulkApproveECOs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)

	body := `{"ids":["ECO-2026-001"],"action":"approve"}`
	req := authedRequest("POST", "/api/v1/ecos/bulk", body, cookie)
	w := httptest.NewRecorder()
	handleBulkECOs(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["success"].(float64) != 1 {
		t.Errorf("expected 1 success, got %v", data["success"])
	}

	// Verify status
	var status string
	db.QueryRow("SELECT status FROM ecos WHERE id='ECO-2026-001'").Scan(&status)
	if status != "approved" {
		t.Errorf("expected approved, got %s", status)
	}
}

func TestBulkInvalidAction(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)

	body := `{"ids":["ECO-2026-001"],"action":"invalid"}`
	req := authedRequest("POST", "/api/v1/ecos/bulk", body, cookie)
	w := httptest.NewRecorder()
	handleBulkECOs(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Legacy Tests ---

func TestNextID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	id := nextID("ECO", "ecos", 3)
	if !strings.HasPrefix(id, "ECO-2026-") {
		t.Errorf("unexpected id format: %s", id)
	}
	if !strings.HasSuffix(id, "003") {
		t.Errorf("expected suffix 003, got: %s", id)
	}
}

func TestDBMigrations(t *testing.T) {
	os.Remove("test_migrations.db")
	defer os.Remove("test_migrations.db")
	if err := initDB("test_migrations.db"); err != nil {
		t.Fatal("migration failed:", err)
	}
	if err := runMigrations(); err != nil {
		t.Fatal("re-migration failed:", err)
	}
}

func TestSeedDB(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	var count int
	db.QueryRow("SELECT COUNT(*) FROM ecos").Scan(&count)
	if count < 2 {
		t.Errorf("expected at least 2 ecos, got %d", count)
	}
	db.QueryRow("SELECT COUNT(*) FROM vendors").Scan(&count)
	if count < 2 {
		t.Errorf("expected at least 2 vendors, got %d", count)
	}
	db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	if count < 2 {
		t.Errorf("expected at least 2 devices, got %d", count)
	}
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count < 2 {
		t.Errorf("expected at least 2 users, got %d", count)
	}
}

func TestSPHelper(t *testing.T) {
	s := "hello"
	ns := ns(&s)
	if !ns.Valid || ns.String != "hello" {
		t.Error("ns failed")
	}
	result := sp(ns)
	if result == nil || *result != "hello" {
		t.Error("sp failed")
	}
}

// --- Report Tests ---

func TestReportInventoryValuation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("GET", "/api/v1/reports/inventory-valuation", "", cookie)
	w := httptest.NewRecorder()
	handleReportInventoryValuation(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatal("expected data")
	}
}

func TestReportInventoryValuationCSV(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/reports/inventory-valuation?format=csv", nil)
	w := httptest.NewRecorder()
	handleReportInventoryValuation(w, req)
	if w.Header().Get("Content-Type") != "text/csv" {
		t.Errorf("expected text/csv, got %s", w.Header().Get("Content-Type"))
	}
}

func TestReportOpenECOs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/reports/open-ecos", nil)
	w := httptest.NewRecorder()
	handleReportOpenECOs(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReportWOThroughput(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/reports/wo-throughput?days=30", nil)
	w := httptest.NewRecorder()
	handleReportWOThroughput(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatal("expected data")
	}
}

func TestReportWOThroughputInvalidDays(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/reports/wo-throughput?days=45", nil)
	w := httptest.NewRecorder()
	handleReportWOThroughput(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Should default to 30
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	dataMap := resp.Data.(map[string]interface{})
	if int(dataMap["days"].(float64)) != 30 {
		t.Errorf("expected default 30 days, got %v", dataMap["days"])
	}
}

func TestReportLowStock(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/reports/low-stock", nil)
	w := httptest.NewRecorder()
	handleReportLowStock(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReportNCRSummary(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/reports/ncr-summary", nil)
	w := httptest.NewRecorder()
	handleReportNCRSummary(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatal("expected data")
	}
}

// --- Quote Margin Tests ---

func TestQuoteCostMargin(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create a quote with lines
	body := `{"customer":"Test Corp","lines":[{"ipn":"RES-001-0001","description":"Resistor","qty":10,"unit_price":1.50}]}`
	req := authedRequest("POST", "/api/v1/quotes", body, cookie)
	w := httptest.NewRecorder()
	handleCreateQuote(w, req)
	if w.Code != 200 {
		t.Fatalf("create quote failed: %d %s", w.Code, w.Body.String())
	}
	var createResp APIResponse
	json.Unmarshal(w.Body.Bytes(), &createResp)
	qData := createResp.Data.(map[string]interface{})
	qid := qData["id"].(string)

	// Get cost/margin
	req2 := authedRequest("GET", "/api/v1/quotes/"+qid+"/cost", "", cookie)
	w2 := httptest.NewRecorder()
	handleQuoteCost(w2, req2, qid)
	if w2.Code != 200 {
		t.Fatalf("quote cost failed: %d %s", w2.Code, w2.Body.String())
	}
	var costResp APIResponse
	json.Unmarshal(w2.Body.Bytes(), &costResp)
	costData := costResp.Data.(map[string]interface{})
	if costData["quote_id"] != qid {
		t.Errorf("expected quote_id %s, got %v", qid, costData["quote_id"])
	}
	if costData["total_quoted"].(float64) != 15.0 {
		t.Errorf("expected total_quoted 15.0, got %v", costData["total_quoted"])
	}
}

// --- Config Test ---

func TestConfig(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	gitplmUIURL = "http://localhost:8888"

	req := httptest.NewRequest("GET", "/api/v1/config", nil)
	w := httptest.NewRecorder()
	handleConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["gitplm_ui_url"] != "http://localhost:8888" {
		t.Errorf("expected gitplm_ui_url, got %v", data["gitplm_ui_url"])
	}
}

// --- Price History Tests ---

func TestPriceCreateAndList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create price entry
	body := `{"ipn":"CAP-001-0001","vendor_id":"V-001","unit_price":0.05,"min_qty":100,"lead_time_days":3}`
	req := authedRequest("POST", "/api/v1/prices", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePrice(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// List prices
	req2 := authedRequest("GET", "/api/v1/prices/CAP-001-0001", "", cookie)
	w2 := httptest.NewRecorder()
	handleListPrices(w2, req2, "CAP-001-0001")
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var resp APIResponse
	json.Unmarshal(w2.Body.Bytes(), &resp)
	items := resp.Data.([]interface{})
	if len(items) < 1 {
		t.Errorf("expected at least 1 price entry, got %d", len(items))
	}
}

func TestPriceDelete(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ipn":"CAP-001-0001","unit_price":0.03}`
	req := authedRequest("POST", "/api/v1/prices", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePrice(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	id := fmt.Sprintf("%.0f", data["id"].(float64))

	req2 := authedRequest("DELETE", "/api/v1/prices/"+id, "", cookie)
	w2 := httptest.NewRecorder()
	handleDeletePrice(w2, req2, id)
	if w2.Code != 200 {
		t.Errorf("expected 200, got %d", w2.Code)
	}
}

func TestPriceTrend(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Add two prices
	api2 := func(b string) { req := authedRequest("POST", "/api/v1/prices", b, cookie); w := httptest.NewRecorder(); handleCreatePrice(w, req) }
	api2(`{"ipn":"RES-001-0001","unit_price":0.01}`)
	api2(`{"ipn":"RES-001-0001","unit_price":0.02}`)

	req := authedRequest("GET", "/api/v1/prices/RES-001-0001/trend", "", cookie)
	w := httptest.NewRecorder()
	handlePriceTrend(w, req, "RES-001-0001")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	points := resp.Data.([]interface{})
	if len(points) != 2 {
		t.Errorf("expected 2 trend points, got %d", len(points))
	}
}

func TestPriceInvalidCreate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ipn":"","unit_price":0}`
	req := authedRequest("POST", "/api/v1/prices", body, cookie)
	w := httptest.NewRecorder()
	handleCreatePrice(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Email Config Tests ---

func TestEmailConfigGetDefault(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("GET", "/api/v1/email/config", "", cookie)
	w := httptest.NewRecorder()
	handleGetEmailConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["enabled"].(float64) != 0 {
		t.Error("expected disabled by default")
	}
}

func TestEmailConfigUpdate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"smtp_host":"smtp.test.com","smtp_port":465,"smtp_user":"user@test.com","smtp_password":"secret","from_address":"noreply@test.com","from_name":"ZRP Test","enabled":1}`
	req := authedRequest("PUT", "/api/v1/email/config", body, cookie)
	w := httptest.NewRecorder()
	handleUpdateEmailConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["smtp_host"] != "smtp.test.com" {
		t.Errorf("expected smtp.test.com, got %v", data["smtp_host"])
	}
	if data["smtp_password"] != "****" {
		t.Error("password should be masked")
	}
}

func TestEmailTestMissingTo(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{}`
	req := authedRequest("POST", "/api/v1/email/test", body, cookie)
	w := httptest.NewRecorder()
	handleTestEmail(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Dashboard Widget Tests ---

func TestDashboardWidgetsGet(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("GET", "/api/v1/dashboard/widgets", "", cookie)
	w := httptest.NewRecorder()
	handleGetDashboardWidgets(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp.Data.([]interface{})
	if len(items) != 11 {
		t.Errorf("expected 11 default widgets, got %d", len(items))
	}
}

func TestDashboardWidgetsUpdate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `[{"widget_type":"kpi_open_ecos","position":5,"enabled":0},{"widget_type":"kpi_low_stock","position":0,"enabled":1}]`
	req := authedRequest("PUT", "/api/v1/dashboard/widgets", body, cookie)
	w := httptest.NewRecorder()
	handleUpdateDashboardWidgets(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify
	var enabled int
	db.QueryRow("SELECT enabled FROM dashboard_widgets WHERE widget_type='kpi_open_ecos'").Scan(&enabled)
	if enabled != 0 {
		t.Errorf("expected disabled, got %d", enabled)
	}
}

func TestRecordPriceFromPO(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	recordPriceFromPO("PO-2026-0001", "CAP-001-0001", 0.05, "V-001")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM price_history WHERE ipn='CAP-001-0001' AND po_id='PO-2026-0001'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 price record, got %d", count)
	}
}

func TestRecordPriceFromPOZeroPrice(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	recordPriceFromPO("PO-2026-0001", "CAP-001-0001", 0, "V-001")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM price_history WHERE ipn='CAP-001-0001'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 price records for zero price, got %d", count)
	}
}

// --- Email Tests ---

func TestGetEmailConfig(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("GET", "/api/v1/email/config", "", cookie)
	w := httptest.NewRecorder()
	handleGetEmailConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["smtp_port"].(float64) != 587 {
		t.Errorf("expected default port 587, got %v", data["smtp_port"])
	}
}

func TestUpdateEmailConfig(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"smtp_host":"smtp.test.com","smtp_port":465,"smtp_user":"user@test.com","smtp_password":"secret","from_address":"noreply@test.com","from_name":"Test","enabled":1}`
	req := authedRequest("PUT", "/api/v1/email/config", body, cookie)
	w := httptest.NewRecorder()
	handleUpdateEmailConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["smtp_host"] != "smtp.test.com" {
		t.Errorf("expected smtp.test.com, got %v", data["smtp_host"])
	}
	if data["smtp_password"] != "****" {
		t.Errorf("expected masked password, got %v", data["smtp_password"])
	}
}

func TestUpdateEmailConfigMaskedPassword(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Set initial password
	body := `{"smtp_host":"smtp.test.com","smtp_port":587,"smtp_user":"u","smtp_password":"realpass","from_address":"a@b.com","enabled":1}`
	req := authedRequest("PUT", "/api/v1/email/config", body, cookie)
	w := httptest.NewRecorder()
	handleUpdateEmailConfig(w, req)

	// Update with masked password - should keep original
	body2 := `{"smtp_host":"smtp.test.com","smtp_port":587,"smtp_user":"u","smtp_password":"****","from_address":"a@b.com","enabled":1}`
	req2 := authedRequest("PUT", "/api/v1/email/config", body2, cookie)
	w2 := httptest.NewRecorder()
	handleUpdateEmailConfig(w2, req2)

	var actual string
	db.QueryRow("SELECT smtp_password FROM email_config WHERE id=1").Scan(&actual)
	if actual != "realpass" {
		t.Errorf("expected original password preserved, got %q", actual)
	}
}

func TestTestEmailMissingTo(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("POST", "/api/v1/email/test", `{}`, cookie)
	w := httptest.NewRecorder()
	handleTestEmail(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestTestEmailWithMockSMTP(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Configure email
	body := `{"smtp_host":"localhost","smtp_port":2525,"smtp_user":"u","smtp_password":"p","from_address":"test@test.com","from_name":"ZRP","enabled":1}`
	req := authedRequest("PUT", "/api/v1/email/config", body, cookie)
	w := httptest.NewRecorder()
	handleUpdateEmailConfig(w, req)

	// Mock SMTP
	var sentTo string
	origSend := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		sentTo = to[0]
		return nil
	}
	defer func() { SMTPSendFunc = origSend }()

	req2 := authedRequest("POST", "/api/v1/email/test", `{"to":"recipient@test.com"}`, cookie)
	w2 := httptest.NewRecorder()
	handleTestEmail(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	if sentTo != "recipient@test.com" {
		t.Errorf("expected recipient@test.com, got %q", sentTo)
	}

	// Check email log
	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_log WHERE to_address='recipient@test.com'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 email log entry, got %d", count)
	}
}

func TestListEmailLog(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Insert a log entry
	db.Exec("INSERT INTO email_log (to_address, subject, body, status, sent_at) VALUES ('a@b.com', 'Test', 'body', 'sent', '2026-01-01 00:00:00')")

	req := authedRequest("GET", "/api/v1/email-log", "", cookie)
	w := httptest.NewRecorder()
	handleListEmailLog(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	if len(data) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(data))
	}
}

func TestEmailOnECOApproved(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Enable email config
	db.Exec("INSERT OR REPLACE INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, from_name, enabled) VALUES (1, 'localhost', 2525, 'u', 'p', 'admin@test.com', 'ZRP', 1)")

	// Mock SMTP
	var sentSubject string
	origSend := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		sentSubject = string(msg)
		return nil
	}
	defer func() { SMTPSendFunc = origSend }()

	emailOnECOApproved("ECO-2026-001")

	if !strings.Contains(sentSubject, "ECO-2026-001") {
		t.Errorf("expected email about ECO-2026-001")
	}
}

func TestEmailOnLowStock(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Enable email config
	db.Exec("INSERT OR REPLACE INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, from_name, enabled) VALUES (1, 'localhost', 2525, 'u', 'p', 'admin@test.com', 'ZRP', 1)")

	// Insert inventory item below reorder point
	db.Exec("INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES ('TEST-001', 2, 10)")

	var sentMsg string
	origSend := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		sentMsg = string(msg)
		return nil
	}
	defer func() { SMTPSendFunc = origSend }()

	emailOnLowStock("TEST-001")

	if !strings.Contains(sentMsg, "TEST-001") {
		t.Errorf("expected email about low stock TEST-001")
	}
}

func TestEmailOnLowStockAboveThreshold(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	db.Exec("INSERT OR REPLACE INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, from_name, enabled) VALUES (1, 'localhost', 2525, 'u', 'p', 'admin@test.com', 'ZRP', 1)")
	db.Exec("INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES ('TEST-002', 50, 10)")

	called := false
	origSend := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		called = true
		return nil
	}
	defer func() { SMTPSendFunc = origSend }()

	emailOnLowStock("TEST-002")

	if called {
		t.Errorf("should not send email when stock is above reorder point")
	}
}

func TestSettingsEmailAliases(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// GET /api/v1/settings/email should work
	req := authedRequest("GET", "/api/v1/settings/email", "", cookie)
	w := httptest.NewRecorder()
	handleGetEmailConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// --- Helper Tests ---

func TestIpnCategory(t *testing.T) {
	tests := []struct{ ipn, want string }{
		{"RES-001-0001", "RES"},
		{"PCA-002-0003", "PCA"},
		{"CAP-100", "CAP"},
	}
	for _, tt := range tests {
		got := ipnCategory(tt.ipn)
		if got != tt.want {
			t.Errorf("ipnCategory(%q) = %q, want %q", tt.ipn, got, tt.want)
		}
	}
}

