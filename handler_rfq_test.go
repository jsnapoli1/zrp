package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
)

// parseRFQ extracts an RFQ from the {"data": ...} wrapper
func parseRFQ(body []byte) RFQ {
	var resp struct{ Data RFQ }
	json.Unmarshal(body, &resp)
	return resp.Data
}

func parseRFQList(body []byte) []RFQ {
	var resp struct{ Data []RFQ }
	json.Unmarshal(body, &resp)
	return resp.Data
}

func TestRFQCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create RFQ
	body := `{"title":"Test RFQ","due_date":"2026-03-01","notes":"test notes","lines":[{"ipn":"IPN-001","description":"10k Resistor","qty":100,"unit":"ea"}],"vendors":[{"vendor_id":"V-001"}]}`
	req := authedRequest("POST", "/api/v1/rfqs", body, cookie)
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)
	if w.Code != 201 {
		t.Fatalf("create RFQ: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	created := parseRFQ(w.Body.Bytes())
	if created.ID == "" {
		t.Fatal("expected RFQ ID")
	}
	if created.Status != "draft" {
		t.Errorf("expected draft status, got %s", created.Status)
	}
	if len(created.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(created.Lines))
	}
	if created.Lines[0].IPN != "IPN-001" {
		t.Errorf("expected IPN-001, got %s", created.Lines[0].IPN)
	}

	rfqID := created.ID

	// List RFQs
	req = authedRequest("GET", "/api/v1/rfqs", "", cookie)
	w = httptest.NewRecorder()
	handleListRFQs(w, req)
	if w.Code != 200 {
		t.Fatalf("list RFQs: expected 200, got %d", w.Code)
	}
	list := parseRFQList(w.Body.Bytes())
	if len(list) < 1 {
		t.Fatal("expected at least 1 RFQ")
	}

	// Get RFQ
	req = authedRequest("GET", "/api/v1/rfqs/"+rfqID, "", cookie)
	w = httptest.NewRecorder()
	handleGetRFQ(w, req, rfqID)
	if w.Code != 200 {
		t.Fatalf("get RFQ: expected 200, got %d", w.Code)
	}
	fetched := parseRFQ(w.Body.Bytes())
	if fetched.Title != "Test RFQ" {
		t.Errorf("expected title 'Test RFQ', got '%s'", fetched.Title)
	}
	if len(fetched.Lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(fetched.Lines))
	}
	if len(fetched.Vendors) != 1 {
		t.Errorf("expected 1 vendor, got %d", len(fetched.Vendors))
	}

	// Update RFQ
	updateBody := `{"title":"Updated RFQ","due_date":"2026-04-01","notes":"updated","lines":[{"ipn":"IPN-001","description":"10k Resistor","qty":200,"unit":"ea"},{"ipn":"IPN-002","description":"Cap","qty":50,"unit":"ea"}],"vendors":[{"vendor_id":"V-001"}]}`
	req = authedRequest("PUT", "/api/v1/rfqs/"+rfqID, updateBody, cookie)
	w = httptest.NewRecorder()
	handleUpdateRFQ(w, req, rfqID)
	if w.Code != 200 {
		t.Fatalf("update RFQ: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	updated := parseRFQ(w.Body.Bytes())
	if updated.Title != "Updated RFQ" {
		t.Errorf("expected 'Updated RFQ', got '%s'", updated.Title)
	}
	if len(updated.Lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(updated.Lines))
	}

	// Delete RFQ
	req = authedRequest("DELETE", "/api/v1/rfqs/"+rfqID, "", cookie)
	w = httptest.NewRecorder()
	handleDeleteRFQ(w, req, rfqID)
	if w.Code != 200 {
		t.Fatalf("delete RFQ: expected 200, got %d", w.Code)
	}

	// Verify deleted
	req = authedRequest("GET", "/api/v1/rfqs/"+rfqID, "", cookie)
	w = httptest.NewRecorder()
	handleGetRFQ(w, req, rfqID)
	if w.Code != 404 {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestRFQCreateValidation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("POST", "/api/v1/rfqs", `{"title":""}`, cookie)
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for empty title, got %d", w.Code)
	}
}

func TestRFQSendWorkflow(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"title":"Send Test","lines":[{"ipn":"IPN-001","description":"Resistor","qty":100,"unit":"ea"}],"vendors":[{"vendor_id":"V-001"}]}`
	req := authedRequest("POST", "/api/v1/rfqs", body, cookie)
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)
	rfq := parseRFQ(w.Body.Bytes())

	// Send
	req = authedRequest("POST", "/api/v1/rfqs/"+rfq.ID+"/send", "", cookie)
	w = httptest.NewRecorder()
	handleSendRFQ(w, req, rfq.ID)
	if w.Code != 200 {
		t.Fatalf("send RFQ: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	sent := parseRFQ(w.Body.Bytes())
	if sent.Status != "sent" {
		t.Errorf("expected sent status, got %s", sent.Status)
	}

	// Can't send again
	req = authedRequest("POST", "/api/v1/rfqs/"+rfq.ID+"/send", "", cookie)
	w = httptest.NewRecorder()
	handleSendRFQ(w, req, rfq.ID)
	if w.Code != 400 {
		t.Errorf("expected 400 for double send, got %d", w.Code)
	}
}

func TestRFQQuotesAndCompare(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	db.Exec(`INSERT INTO vendors (id, name, status) VALUES ('V-TEST', 'Test Vendor', 'active')`)

	body := `{"title":"Quote Test","lines":[{"ipn":"IPN-001","description":"Resistor","qty":100,"unit":"ea"}],"vendors":[{"vendor_id":"V-001"},{"vendor_id":"V-TEST"}]}`
	req := authedRequest("POST", "/api/v1/rfqs", body, cookie)
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)
	rfq := parseRFQ(w.Body.Bytes())
	rfqID := rfq.ID

	// Get to find IDs
	req = authedRequest("GET", "/api/v1/rfqs/"+rfqID, "", cookie)
	w = httptest.NewRecorder()
	handleGetRFQ(w, req, rfqID)
	rfq = parseRFQ(w.Body.Bytes())

	if len(rfq.Lines) == 0 {
		t.Fatal("expected lines")
	}
	if len(rfq.Vendors) < 2 {
		t.Fatal("expected 2 vendors")
	}

	lineID := rfq.Lines[0].ID
	vendor1ID := rfq.Vendors[0].ID
	vendor2ID := rfq.Vendors[1].ID

	// Add quotes
	qBody := fmt.Sprintf(`{"rfq_vendor_id":%d,"rfq_line_id":%d,"unit_price":0.05,"lead_time_days":14,"moq":100}`, vendor1ID, lineID)
	req = authedRequest("POST", "/api/v1/rfqs/"+rfqID+"/quotes", qBody, cookie)
	w = httptest.NewRecorder()
	handleCreateRFQQuote(w, req, rfqID)
	if w.Code != 201 {
		t.Fatalf("create quote: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	qBody = fmt.Sprintf(`{"rfq_vendor_id":%d,"rfq_line_id":%d,"unit_price":0.03,"lead_time_days":21,"moq":500}`, vendor2ID, lineID)
	req = authedRequest("POST", "/api/v1/rfqs/"+rfqID+"/quotes", qBody, cookie)
	w = httptest.NewRecorder()
	handleCreateRFQQuote(w, req, rfqID)
	if w.Code != 201 {
		t.Fatalf("create quote 2: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Compare
	req = authedRequest("GET", "/api/v1/rfqs/"+rfqID+"/compare", "", cookie)
	w = httptest.NewRecorder()
	handleCompareRFQ(w, req, rfqID)
	if w.Code != 200 {
		t.Fatalf("compare: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data map[string]interface{}
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data["lines"] == nil {
		t.Error("expected lines in compare response")
	}
	if resp.Data["vendors"] == nil {
		t.Error("expected vendors in compare response")
	}
	if resp.Data["matrix"] == nil {
		t.Error("expected matrix in compare response")
	}
}

func TestRFQAward(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"title":"Award Test","lines":[{"ipn":"IPN-001","description":"Resistor","qty":100,"unit":"ea"}],"vendors":[{"vendor_id":"V-001"}]}`
	req := authedRequest("POST", "/api/v1/rfqs", body, cookie)
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)
	rfq := parseRFQ(w.Body.Bytes())
	rfqID := rfq.ID

	// Get IDs
	req = authedRequest("GET", "/api/v1/rfqs/"+rfqID, "", cookie)
	w = httptest.NewRecorder()
	handleGetRFQ(w, req, rfqID)
	rfq = parseRFQ(w.Body.Bytes())

	lineID := rfq.Lines[0].ID
	vendorID := rfq.Vendors[0].ID

	// Add quote
	qBody := fmt.Sprintf(`{"rfq_vendor_id":%d,"rfq_line_id":%d,"unit_price":0.05,"lead_time_days":14,"moq":100}`, vendorID, lineID)
	req = authedRequest("POST", "/api/v1/rfqs/"+rfqID+"/quotes", qBody, cookie)
	w = httptest.NewRecorder()
	handleCreateRFQQuote(w, req, rfqID)

	// Award
	req = authedRequest("POST", "/api/v1/rfqs/"+rfqID+"/award", `{"vendor_id":"V-001"}`, cookie)
	w = httptest.NewRecorder()
	handleAwardRFQ(w, req, rfqID)
	if w.Code != 200 {
		t.Fatalf("award: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var awardResp struct {
		Data map[string]string
	}
	json.Unmarshal(w.Body.Bytes(), &awardResp)
	if awardResp.Data["status"] != "awarded" {
		t.Errorf("expected awarded, got %s", awardResp.Data["status"])
	}
	if awardResp.Data["po_id"] == "" {
		t.Error("expected po_id")
	}

	// Verify PO was created
	var poStatus string
	err := db.QueryRow(`SELECT status FROM purchase_orders WHERE id=?`, awardResp.Data["po_id"]).Scan(&poStatus)
	if err != nil {
		t.Fatalf("PO not created: %v", err)
	}

	// Verify PO lines
	var lineCount int
	db.QueryRow(`SELECT COUNT(*) FROM po_lines WHERE po_id=?`, awardResp.Data["po_id"]).Scan(&lineCount)
	if lineCount != 1 {
		t.Errorf("expected 1 PO line, got %d", lineCount)
	}

	// Verify RFQ status
	var rfqStatus string
	db.QueryRow(`SELECT status FROM rfqs WHERE id=?`, rfqID).Scan(&rfqStatus)
	if rfqStatus != "awarded" {
		t.Errorf("expected awarded, got %s", rfqStatus)
	}
}

func TestRFQAwardValidation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Non-existent
	req := authedRequest("POST", "/api/v1/rfqs/FAKE/award", `{"vendor_id":"V-001"}`, cookie)
	w := httptest.NewRecorder()
	handleAwardRFQ(w, req, "FAKE")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Missing vendor_id
	req = authedRequest("POST", "/api/v1/rfqs", `{"title":"Val Test","vendors":[{"vendor_id":"V-001"}]}`, cookie)
	w = httptest.NewRecorder()
	handleCreateRFQ(w, req)
	rfq := parseRFQ(w.Body.Bytes())

	req = authedRequest("POST", "/api/v1/rfqs/"+rfq.ID+"/award", `{}`, cookie)
	w = httptest.NewRecorder()
	handleAwardRFQ(w, req, rfq.ID)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRFQCloseWorkflow(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create and send
	body := `{"title":"Close Test","lines":[{"ipn":"IPN-001","description":"Resistor","qty":100,"unit":"ea"}],"vendors":[{"vendor_id":"V-001"}]}`
	req := authedRequest("POST", "/api/v1/rfqs", body, cookie)
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)
	rfq := parseRFQ(w.Body.Bytes())

	// Can't close draft
	req = authedRequest("POST", "/api/v1/rfqs/"+rfq.ID+"/close", "", cookie)
	w = httptest.NewRecorder()
	handleCloseRFQ(w, req, rfq.ID)
	if w.Code != 400 {
		t.Errorf("expected 400 for closing draft, got %d", w.Code)
	}

	// Send it
	req = authedRequest("POST", "/api/v1/rfqs/"+rfq.ID+"/send", "", cookie)
	w = httptest.NewRecorder()
	handleSendRFQ(w, req, rfq.ID)

	// Close sent RFQ
	req = authedRequest("POST", "/api/v1/rfqs/"+rfq.ID+"/close", "", cookie)
	w = httptest.NewRecorder()
	handleCloseRFQ(w, req, rfq.ID)
	if w.Code != 200 {
		t.Fatalf("close: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	closed := parseRFQ(w.Body.Bytes())
	if closed.Status != "closed" {
		t.Errorf("expected closed status, got %s", closed.Status)
	}
}

func TestRFQDashboard(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create a couple RFQs
	body := `{"title":"Dash Test 1","lines":[{"ipn":"IPN-001","description":"Resistor","qty":100,"unit":"ea"}],"vendors":[{"vendor_id":"V-001"}]}`
	req := authedRequest("POST", "/api/v1/rfqs", body, cookie)
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)

	body = `{"title":"Dash Test 2","vendors":[{"vendor_id":"V-001"}]}`
	req = authedRequest("POST", "/api/v1/rfqs", body, cookie)
	w = httptest.NewRecorder()
	handleCreateRFQ(w, req)

	// Get dashboard
	req = authedRequest("GET", "/api/v1/rfq-dashboard", "", cookie)
	w = httptest.NewRecorder()
	handleRFQDashboard(w, req)
	if w.Code != 200 {
		t.Fatalf("dashboard: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			OpenRFQs         int `json:"open_rfqs"`
			PendingResponses int `json:"pending_responses"`
			RFQs             []struct {
				ID string `json:"id"`
			} `json:"rfqs"`
		}
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.OpenRFQs < 2 {
		t.Errorf("expected at least 2 open RFQs, got %d", resp.Data.OpenRFQs)
	}
	if len(resp.Data.RFQs) < 2 {
		t.Errorf("expected at least 2 RFQs in list, got %d", len(resp.Data.RFQs))
	}
}

func TestRFQEmailBody(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"title":"Email Test","due_date":"2026-03-01","notes":"urgent","lines":[{"ipn":"IPN-001","description":"10k Resistor","qty":1000,"unit":"ea"}]}`
	req := authedRequest("POST", "/api/v1/rfqs", body, cookie)
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)
	rfq := parseRFQ(w.Body.Bytes())

	req = authedRequest("GET", "/api/v1/rfqs/"+rfq.ID+"/email", "", cookie)
	w = httptest.NewRecorder()
	handleRFQEmailBody(w, req, rfq.ID)
	if w.Code != 200 {
		t.Fatalf("email: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Subject string `json:"subject"`
			Body    string `json:"body"`
		}
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Subject == "" {
		t.Error("expected non-empty subject")
	}
	if resp.Data.Body == "" {
		t.Error("expected non-empty body")
	}
	if !contains(resp.Data.Body, "IPN-001") {
		t.Error("expected body to contain IPN-001")
	}
	if !contains(resp.Data.Body, "2026-03-01") {
		t.Error("expected body to contain due date")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}
func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestRFQAwardPerLine(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	db.Exec(`INSERT INTO vendors (id, name, status) VALUES ('V-PL1', 'Vendor PL1', 'active')`)
	db.Exec(`INSERT INTO vendors (id, name, status) VALUES ('V-PL2', 'Vendor PL2', 'active')`)

	body := `{"title":"Per-Line Award","lines":[{"ipn":"IPN-A","description":"Part A","qty":100,"unit":"ea"},{"ipn":"IPN-B","description":"Part B","qty":200,"unit":"ea"}],"vendors":[{"vendor_id":"V-PL1"},{"vendor_id":"V-PL2"}]}`
	req := authedRequest("POST", "/api/v1/rfqs", body, cookie)
	w := httptest.NewRecorder()
	handleCreateRFQ(w, req)
	rfq := parseRFQ(w.Body.Bytes())

	// Get full details
	req = authedRequest("GET", "/api/v1/rfqs/"+rfq.ID, "", cookie)
	w = httptest.NewRecorder()
	handleGetRFQ(w, req, rfq.ID)
	rfq = parseRFQ(w.Body.Bytes())

	line1ID := rfq.Lines[0].ID
	line2ID := rfq.Lines[1].ID
	vendor1RFQID := rfq.Vendors[0].ID
	vendor2RFQID := rfq.Vendors[1].ID

	// Add quotes from both vendors for both lines
	qBody := fmt.Sprintf(`{"rfq_vendor_id":%d,"rfq_line_id":%d,"unit_price":0.05,"lead_time_days":14,"moq":100}`, vendor1RFQID, line1ID)
	req = authedRequest("POST", "/api/v1/rfqs/"+rfq.ID+"/quotes", qBody, cookie)
	w = httptest.NewRecorder()
	handleCreateRFQQuote(w, req, rfq.ID)

	qBody = fmt.Sprintf(`{"rfq_vendor_id":%d,"rfq_line_id":%d,"unit_price":0.03,"lead_time_days":21,"moq":500}`, vendor2RFQID, line2ID)
	req = authedRequest("POST", "/api/v1/rfqs/"+rfq.ID+"/quotes", qBody, cookie)
	w = httptest.NewRecorder()
	handleCreateRFQQuote(w, req, rfq.ID)

	// Award line 1 to V-PL1, line 2 to V-PL2
	awardBody := fmt.Sprintf(`{"awards":[{"line_id":%d,"vendor_id":"V-PL1"},{"line_id":%d,"vendor_id":"V-PL2"}]}`, line1ID, line2ID)
	req = authedRequest("POST", "/api/v1/rfqs/"+rfq.ID+"/award-lines", awardBody, cookie)
	w = httptest.NewRecorder()
	handleAwardRFQPerLine(w, req, rfq.ID)
	if w.Code != 200 {
		t.Fatalf("award-lines: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Status string   `json:"status"`
			POIDs  []string `json:"po_ids"`
		}
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Status != "awarded" {
		t.Errorf("expected awarded, got %s", resp.Data.Status)
	}
	if len(resp.Data.POIDs) != 2 {
		t.Errorf("expected 2 POs, got %d", len(resp.Data.POIDs))
	}

	// Verify RFQ status
	var rfqStatus string
	db.QueryRow(`SELECT status FROM rfqs WHERE id=?`, rfq.ID).Scan(&rfqStatus)
	if rfqStatus != "awarded" {
		t.Errorf("expected awarded, got %s", rfqStatus)
	}
}

func TestRFQNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := authedRequest("GET", "/api/v1/rfqs/NONEXISTENT", "", nil)
	w := httptest.NewRecorder()
	handleGetRFQ(w, req, "NONEXISTENT")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
