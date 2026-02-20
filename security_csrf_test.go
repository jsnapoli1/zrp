package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupCSRFTestDB(t *testing.T) (*sql.DB, func()) {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Create all required tables
	schema := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			email TEXT,
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			description TEXT,
			category TEXT,
			lifecycle TEXT DEFAULT 'Active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL
		);

		CREATE TABLE csrf_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`

	_, err = testDB.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Seed test data
	_, err = testDB.Exec(`
		INSERT INTO users (id, username, password, email, role) VALUES 
		(1, 'admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin@test.com', 'admin'),
		(2, 'user', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'user@test.com', 'user')
	`)
	if err != nil {
		t.Fatalf("Failed to seed users: %v", err)
	}

	// Create test session
	sessionToken := "test-session-token-123"
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	_, err = testDB.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, 1, ?)",
		sessionToken, expiresAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Seed some test data
	_, err = testDB.Exec("INSERT INTO parts (ipn, description, category) VALUES ('TEST-001', 'Test Part', 'Resistors')")
	if err != nil {
		t.Fatalf("Failed to seed parts: %v", err)
	}

	cleanup := func() {
		testDB.Close()
	}

	return testDB, cleanup
}

// Helper function to create a valid CSRF token using the same logic as production
func createCSRFToken(testDB *sql.DB, userID int) (string, error) {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)
	
	expiresAt := time.Now().UTC().Add(1 * time.Hour)
	_, err := testDB.Exec("INSERT INTO csrf_tokens (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt.Format("2006-01-02 15:04:05"))
	return token, err
}

// Helper to make authenticated request with optional CSRF token
func makeAuthenticatedRequest(method, path string, body []byte, csrfToken string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	// Add session cookie
	req.AddCookie(&http.Cookie{
		Name:  "zrp_session",
		Value: "test-session-token-123",
	})

	// Add CSRF token if provided
	if csrfToken != "" {
		req.Header.Set("X-CSRF-Token", csrfToken)
	}

	return req
}

// Test 1: POST without CSRF token should fail
func TestCSRF_CreatePart_NoToken(t *testing.T) {
	testDB, cleanup := setupCSRFTestDB(t)
	defer cleanup()
	db = testDB

	body := []byte(`{"ipn":"NEW-001","description":"New Part","category":"Resistors"}`)
	req := makeAuthenticatedRequest("POST", "/api/v1/parts", body, "")
	w := httptest.NewRecorder()

	handler := csrfMiddleware(http.HandlerFunc(handleCreatePart))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden without CSRF token, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(strings.ToLower(resp["error"]), "csrf") {
		t.Errorf("Expected CSRF error message, got: %v", resp)
	}
}

// Test 2: GET requests should NOT require CSRF token
func TestCSRF_GetRequests_NoTokenRequired(t *testing.T) {
	testDB, cleanup := setupCSRFTestDB(t)
	defer cleanup()
	db = testDB

	req := makeAuthenticatedRequest("GET", "/api/v1/parts", nil, "")
	w := httptest.NewRecorder()

	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	handler.ServeHTTP(w, req)

	if w.Code == http.StatusForbidden {
		t.Errorf("GET request should not require CSRF token, got 403")
	}
}

// Test 3: Valid CSRF token should work
func TestCSRF_CreatePart_ValidToken(t *testing.T) {
	testDB, cleanup := setupCSRFTestDB(t)
	defer cleanup()
	db = testDB

	csrfToken, err := createCSRFToken(testDB, 1)
	if err != nil {
		t.Fatalf("Failed to create CSRF token: %v", err)
	}

	body := []byte(`{"ipn":"NEW-003","description":"New Part","category":"Resistors"}`)
	req := makeAuthenticatedRequest("POST", "/api/v1/parts", body, csrfToken)
	w := httptest.NewRecorder()

	handler := csrfMiddleware(http.HandlerFunc(handleCreatePart))
	handler.ServeHTTP(w, req)

	if w.Code == http.StatusForbidden {
		t.Errorf("Valid CSRF token should not be rejected, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 4: Invalid CSRF token should fail
func TestCSRF_CreatePart_InvalidToken(t *testing.T) {
	testDB, cleanup := setupCSRFTestDB(t)
	defer cleanup()
	db = testDB

	body := []byte(`{"ipn":"NEW-002","description":"New Part","category":"Resistors"}`)
	req := makeAuthenticatedRequest("POST", "/api/v1/parts", body, "invalid-token-xyz")
	w := httptest.NewRecorder()

	handler := csrfMiddleware(http.HandlerFunc(handleCreatePart))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden with invalid CSRF token, got %d", w.Code)
	}
}

// Test 5: CSRF token tied to user session
func TestCSRF_TokenTiedToUserSession(t *testing.T) {
	testDB, cleanup := setupCSRFTestDB(t)
	defer cleanup()
	db = testDB

	// Create CSRF token for user 1
	csrfToken, err := createCSRFToken(testDB, 1)
	if err != nil {
		t.Fatalf("Failed to create CSRF token: %v", err)
	}

	// Create session for user 2
	sessionToken2 := "test-session-token-456"
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	_, err = testDB.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, 2, ?)",
		sessionToken2, expiresAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create session for user 2: %v", err)
	}

	// Try to use user 1's CSRF token with user 2's session
	body := []byte(`{"ipn":"NEW-004","description":"Test Part","category":"Resistors"}`)
	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.AddCookie(&http.Cookie{
		Name:  "zrp_session",
		Value: sessionToken2, // User 2's session
	})

	w := httptest.NewRecorder()

	handler := csrfMiddleware(http.HandlerFunc(handleCreatePart))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("CSRF token from user 1 should not work with user 2's session, got %d", w.Code)
	}
}

// Test 6: Expired CSRF token should be rejected
func TestCSRF_ExpiredToken(t *testing.T) {
	testDB, cleanup := setupCSRFTestDB(t)
	defer cleanup()
	db = testDB

	// Create expired CSRF token
	b := make([]byte, 32)
	rand.Read(b)
	expiredToken := hex.EncodeToString(b)
	
	expiresAt := time.Now().UTC().Add(-1 * time.Hour) // Expired 1 hour ago
	_, err := testDB.Exec("INSERT INTO csrf_tokens (token, user_id, expires_at) VALUES (?, 1, ?)",
		expiredToken, expiresAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create expired CSRF token: %v", err)
	}

	body := []byte(`{"ipn":"NEW-005","description":"Test Part","category":"Resistors"}`)
	req := makeAuthenticatedRequest("POST", "/api/v1/parts", body, expiredToken)
	w := httptest.NewRecorder()

	handler := csrfMiddleware(http.HandlerFunc(handleCreatePart))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expired CSRF token should be rejected, got %d", w.Code)
	}
}

// Test 7-29: Test various state-changing endpoints
func TestCSRF_StateChangingEndpoints(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{"Delete Part", "DELETE", "/api/v1/parts/TEST-001", nil},
		{"Update Part", "PUT", "/api/v1/parts/TEST-001", []byte(`{"description":"Updated"}`)},
		{"Create Category", "POST", "/api/v1/categories", []byte(`{"name":"New Category"}`)},
		{"Create Vendor", "POST", "/api/v1/vendors", []byte(`{"name":"New Vendor"}`)},
		{"Update Vendor", "PUT", "/api/v1/vendors/1", []byte(`{"name":"Updated Vendor"}`)},
		{"Create Device", "POST", "/api/v1/devices", []byte(`{"serial_number":"SN001","ipn":"TEST-001"}`)},
		{"Update Device", "PUT", "/api/v1/devices/SN001", []byte(`{"status":"Retired"}`)},
		{"Create WorkOrder", "POST", "/api/v1/workorders", []byte(`{"wo_number":"WO-001","ipn":"TEST-001","quantity":10}`)},
		{"Update WorkOrder", "PUT", "/api/v1/workorders/1", []byte(`{"status":"Completed"}`)},
		{"Create NCR", "POST", "/api/v1/ncr", []byte(`{"ncr_number":"NCR-001","description":"Defect"}`)},
		{"Update NCR", "PUT", "/api/v1/ncr/1", []byte(`{"status":"Closed"}`)},
		{"Create CAPA", "POST", "/api/v1/capa", []byte(`{"capa_number":"CAPA-001","description":"Action"}`)},
		{"Update CAPA", "PUT", "/api/v1/capa/1", []byte(`{"status":"Closed"}`)},
		{"Create Procurement", "POST", "/api/v1/procurement", []byte(`{"po_number":"PO-001","vendor_id":1}`)},
		{"Update Procurement", "PUT", "/api/v1/procurement/1", []byte(`{"status":"Submitted"}`)},
		{"Create Invoice", "POST", "/api/v1/invoices", []byte(`{"invoice_number":"INV-001","amount":1000}`)},
		{"Update Invoice", "PUT", "/api/v1/invoices/1", []byte(`{"status":"Paid"}`)},
		{"Create SalesOrder", "POST", "/api/v1/sales-orders", []byte(`{"so_number":"SO-001","customer":"Test"}`)},
		{"Update SalesOrder", "PUT", "/api/v1/sales-orders/1", []byte(`{"status":"Fulfilled"}`)},
		{"Inventory Transact", "POST", "/api/v1/inventory/transact", []byte(`{"ipn":"TEST-001","quantity":10,"type":"add"}`)},
		{"Create PartChange", "POST", "/api/v1/part-changes", []byte(`{"eco_number":"ECO-001","ipn":"TEST-001"}`)},
		{"Update PartChange", "PUT", "/api/v1/part-changes/1", []byte(`{"status":"Approved"}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB, cleanup := setupCSRFTestDB(t)
			defer cleanup()
			db = testDB

			req := makeAuthenticatedRequest(tt.method, tt.path, tt.body, "")
			w := httptest.NewRecorder()

			handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success":true}`))
			}))
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("%s: Expected 403 Forbidden without CSRF token, got %d", tt.name, w.Code)
			}
		})
	}
}
