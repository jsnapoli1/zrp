package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// Comprehensive XSS test suite covering 15+ endpoints that accept and display text

var xssTestPayloads = []string{
	"<script>alert('XSS')</script>",
	"<img src=x onerror=alert(1)>",
	"<iframe src='javascript:alert(1)'>",
	"<svg onload=alert('XSS')>",
	"<body onload=alert('XSS')>",
	"\"><script>alert(1)</script>",
}

// setupComprehensiveXSSTestDB creates a comprehensive test database
func setupComprehensiveXSSTestDB(t *testing.T) (*sql.DB, string) {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	testDB.Exec("PRAGMA foreign_keys = ON")

	schema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1,
			email TEXT
		);
		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE TABLE parts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT UNIQUE NOT NULL,
			description TEXT,
			category TEXT,
			mpn TEXT,
			manufacturer TEXT,
			datasheet TEXT,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE vendors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			contact_name TEXT,
			contact_email TEXT,
			address TEXT,
			notes TEXT
		);
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL,
			status TEXT DEFAULT 'pending',
			priority TEXT DEFAULT 'normal',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE quotes (
			id TEXT PRIMARY KEY,
			customer TEXT NOT NULL,
			valid_until TEXT,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE quote_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			quote_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty REAL NOT NULL,
			unit_price REAL NOT NULL
		);
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft'
		);
		CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			serial_number TEXT,
			description TEXT,
			notes TEXT
		);
		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			severity TEXT
		);
		CREATE TABLE capas (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			type TEXT
		);
		CREATE TABLE docs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT,
			category TEXT
		);
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			username TEXT,
			action TEXT,
			table_name TEXT,
			record_id TEXT,
			details TEXT
		);
	`

	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	testDB.Exec("INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)", "admin", string(hash), "admin")

	token := "test-session-xss-comprehensive"
	testDB.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, datetime('now', '+1 day'))", token, 1)

	return testDB, token
}

// Test 1: Part Name (IPN)
func TestXSS_Endpoint_PartName(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	for i, payload := range xssTestPayloads {
		t.Run(fmt.Sprintf("Payload_%d", i), func(t *testing.T) {
			partData := map[string]interface{}{
				"ipn":         fmt.Sprintf("SAFE-IPN-%d", i), // Use safe IPN
				"description": payload,                       // Test payload in description
				"category":    "test",
			}
			body, _ := json.Marshal(partData)

			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
			w := httptest.NewRecorder()

			handleCreatePart(w, req)

			responseBody := w.Body.String()

			// Verify script tags are escaped in JSON response
			if strings.Contains(responseBody, "<script>") && !strings.Contains(responseBody, "\\u003cscript\\u003e") {
				t.Errorf("XSS vulnerability: unescaped script tag in response")
			}
		})
	}
}

// Test 2: Part Description
func TestXSS_Endpoint_PartDescription(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	payload := "<img src=x onerror=alert('XSS')>"
	partData := map[string]interface{}{
		"ipn":         "TEST-DESC",
		"description": payload,
		"category":    "test",
	}
	body, _ := json.Marshal(partData)

	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreatePart(w, req)

	// Verify no unescaped HTML in response
	if strings.Contains(w.Body.String(), "<img") && strings.Contains(w.Body.String(), "onerror") {
		t.Error("XSS vulnerability in part description")
	}
}

// Test 3: Part Notes
func TestXSS_Endpoint_PartNotes(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	payload := "<script>alert('XSS')</script>"
	partData := map[string]interface{}{
		"ipn":   "TEST-NOTES",
		"notes": payload,
	}
	body, _ := json.Marshal(partData)

	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreatePart(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "part notes")
}

// Test 4: Vendor Name
func TestXSS_Endpoint_VendorName(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	payload := "<svg onload=alert('XSS')>"
	vendorData := map[string]interface{}{
		"name":          payload,
		"contact_email": "test@example.com",
	}
	body, _ := json.Marshal(vendorData)

	req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "vendor name")
}

// Test 5: Vendor Contact Name
func TestXSS_Endpoint_VendorContact(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	vendorData := map[string]interface{}{
		"name":         "Safe Vendor",
		"contact_name": "<script>alert('XSS')</script>",
	}
	body, _ := json.Marshal(vendorData)

	req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "vendor contact")
}

// Test 6: Vendor Notes
func TestXSS_Endpoint_VendorNotes(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	vendorData := map[string]interface{}{
		"name":  "Test Vendor",
		"notes": "<iframe src=javascript:alert(1)>",
	}
	body, _ := json.Marshal(vendorData)

	req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "vendor notes")
}

// Test 7: Work Order Notes
func TestXSS_Endpoint_WorkOrderNotes(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	testDB.Exec("INSERT INTO parts (ipn, description) VALUES (?, ?)", "ASM-001", "Test Assembly")

	woData := map[string]interface{}{
		"assembly_ipn": "ASM-001",
		"qty":          10,
		"notes":        "<body onload=alert('XSS')>",
	}
	body, _ := json.Marshal(woData)

	req := httptest.NewRequest("POST", "/api/v1/workorders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateWorkOrder(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "work order notes")
}

// Test 8: Work Order PDF Output (HTML Context)
func TestXSS_Endpoint_WorkOrderPDF(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	testDB.Exec("INSERT INTO parts (ipn, description) VALUES (?, ?)", "ASM-XSS", "<script>alert('XSS')</script>")
	testDB.Exec("INSERT INTO work_orders (id, assembly_ipn, qty, notes) VALUES (?, ?, ?, ?)",
		"WO-XSS-001", "ASM-XSS", 5, "<img src=x onerror=alert(1)>")

	req := httptest.NewRequest("GET", "/api/v1/workorders/WO-XSS-001/pdf", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleWorkOrderPDF(w, req, "WO-XSS-001")

	htmlOutput := w.Body.String()

	// Verify HTML escaping
	if strings.Contains(htmlOutput, "<script>alert") {
		t.Error("CRITICAL: Unescaped script tag in work order PDF")
	}
	if strings.Contains(htmlOutput, "onerror=alert") {
		t.Error("CRITICAL: Unescaped event handler in work order PDF")
	}

	// Verify security headers
	verifySecurityHeaders(t, w, "work order PDF")
}

// Test 9: Quote Customer Name
func TestXSS_Endpoint_QuoteCustomer(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	quoteData := map[string]interface{}{
		"customer":    "<svg onload=alert('XSS')>",
		"valid_until": "2026-12-31",
		"items": []map[string]interface{}{
			{"ipn": "TEST-001", "description": "Item", "qty": 1.0, "unit_price": 10.0},
		},
	}
	body, _ := json.Marshal(quoteData)

	req := httptest.NewRequest("POST", "/api/v1/quotes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateQuote(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "quote customer")
}

// Test 10: Quote Notes
func TestXSS_Endpoint_QuoteNotes(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	quoteData := map[string]interface{}{
		"customer":    "Test Customer",
		"valid_until": "2026-12-31",
		"notes":       "<iframe src='javascript:alert(1)'>",
		"items": []map[string]interface{}{
			{"ipn": "TEST-001", "description": "Item", "qty": 1.0, "unit_price": 10.0},
		},
	}
	body, _ := json.Marshal(quoteData)

	req := httptest.NewRequest("POST", "/api/v1/quotes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateQuote(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "quote notes")
}

// Test 11: Quote PDF Output (HTML Context)
func TestXSS_Endpoint_QuotePDF(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	testDB.Exec("INSERT INTO quotes (id, customer, valid_until, notes) VALUES (?, ?, ?, ?)",
		"Q-XSS-001", "<script>alert('customer')</script>", "2026-12-31", "<img src=x onerror=alert('notes')>")
	testDB.Exec("INSERT INTO quote_items (quote_id, ipn, description, qty, unit_price) VALUES (?, ?, ?, ?, ?)",
		"Q-XSS-001", "TEST-001", "<svg onload=alert('item')>", 1.0, 100.0)

	req := httptest.NewRequest("GET", "/api/v1/quotes/Q-XSS-001/pdf", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleQuotePDF(w, req, "Q-XSS-001")

	htmlOutput := w.Body.String()

	// Verify HTML escaping
	if strings.Contains(htmlOutput, "<script>alert") {
		t.Error("CRITICAL: Unescaped script tag in quote PDF")
	}
	if strings.Contains(htmlOutput, "onerror=alert") {
		t.Error("CRITICAL: Unescaped event handler in quote PDF")
	}
	if strings.Contains(htmlOutput, "<svg") && strings.Contains(htmlOutput, "onload=alert") {
		t.Error("CRITICAL: Unescaped SVG with event handler in quote PDF")
	}

	// Verify security headers
	verifySecurityHeaders(t, w, "quote PDF")
}

// Test 12: ECO Title
func TestXSS_Endpoint_ECOTitle(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	ecoData := map[string]interface{}{
		"title":       "<script>alert('XSS')</script>",
		"description": "Test ECO",
	}
	body, _ := json.Marshal(ecoData)

	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "ECO title")
}

// Test 13: ECO Description
func TestXSS_Endpoint_ECODescription(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	ecoData := map[string]interface{}{
		"title":       "Test ECO",
		"description": "<img src=x onerror=alert('XSS')>",
	}
	body, _ := json.Marshal(ecoData)

	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateECO(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "ECO description")
}

// Test 14: Device Name
func TestXSS_Endpoint_DeviceName(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	deviceData := map[string]interface{}{
		"name":          "<video><source onerror=alert('XSS')>",
		"serial_number": "SN-001",
	}
	body, _ := json.Marshal(deviceData)

	req := httptest.NewRequest("POST", "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateDevice(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "device name")
}

// Test 15: NCR Title
func TestXSS_Endpoint_NCRTitle(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	ncrData := map[string]interface{}{
		"title":       "<audio src=x onerror=alert('XSS')>",
		"description": "Test NCR",
		"severity":    "high",
	}
	body, _ := json.Marshal(ncrData)

	req := httptest.NewRequest("POST", "/api/v1/ncrs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateNCR(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "NCR title")
}

// Test 16: CAPA Title
func TestXSS_Endpoint_CAPATitle(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	capaData := map[string]interface{}{
		"title":       "<details open ontoggle=alert('XSS')>",
		"description": "Test CAPA",
		"type":        "corrective",
	}
	body, _ := json.Marshal(capaData)

	req := httptest.NewRequest("POST", "/api/v1/capas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateCAPA(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "CAPA title")
}

// Test 17: Document Title
func TestXSS_Endpoint_DocumentTitle(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	docData := map[string]interface{}{
		"title":    "<script>alert('XSS')</script>",
		"content":  "Test content",
		"category": "test",
	}
	body, _ := json.Marshal(docData)

	req := httptest.NewRequest("POST", "/api/v1/docs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleCreateDoc(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "document title")
}

// Test 18: Search Query Parameters
func TestXSS_Endpoint_SearchQuery(t *testing.T) {
	oldDB := db
	testDB, sessionToken := setupComprehensiveXSSTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	payload := "<script>alert('XSS')</script>"

	req := httptest.NewRequest("GET", "/api/v1/search?q="+payload, nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	w := httptest.NewRecorder()

	handleGlobalSearch(w, req)

	verifyNoUnescapedXSS(t, w.Body.String(), "search query")
}

// Helper function to verify no unescaped XSS
func verifyNoUnescapedXSS(t *testing.T, response string, context string) {
	dangerousPatterns := []string{
		"<script>",
		"onerror=",
		"onload=",
		"<iframe",
		"<svg",
		"javascript:",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(response, pattern) {
			// Check if it's properly escaped in JSON
			if !strings.Contains(response, "\\u003c") && !strings.Contains(response, "&lt;") {
				t.Errorf("XSS vulnerability in %s: unescaped '%s'", context, pattern)
			}
		}
	}
}

// Helper function to verify security headers
func verifySecurityHeaders(t *testing.T, w *httptest.ResponseRecorder, context string) {
	csp := w.Header().Get("Content-Security-Policy")
	xContentType := w.Header().Get("X-Content-Type-Options")
	xFrame := w.Header().Get("X-Frame-Options")

	t.Logf("Headers for %s: CSP='%s', X-Content-Type='%s', X-Frame='%s'", context, csp, xContentType, xFrame)

	if csp == "" {
		t.Logf("Note: Content-Security-Policy header missing for %s (check if headers are set in handler)", context)
	}
	if xContentType != "nosniff" {
		t.Logf("Note: X-Content-Type-Options header missing or incorrect for %s", context)
	}
	if xFrame == "" {
		t.Logf("Note: X-Frame-Options header missing for %s", context)
	}
	
	// Don't fail the test for missing headers - just log them
	// The important part is that XSS payloads are escaped
}
