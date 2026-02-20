package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupScanTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create inventory table
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			location TEXT,
			qty REAL DEFAULT 0,
			description TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create devices table
	_, err = testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			model TEXT,
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	return testDB
}

func insertScanTestData(t *testing.T, testDB *sql.DB, partsDir string) {
	// Insert inventory items
	_, err := testDB.Exec("INSERT INTO inventory (ipn, location, qty, description) VALUES (?, ?, ?, ?)",
		"PART-001", "A1", 100.0, "Resistor 10K")
	if err != nil {
		t.Fatalf("Failed to insert inventory: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO inventory (ipn, location, qty, description) VALUES (?, ?, ?, ?)",
		"PART-002", "B2", 50.0, "Capacitor 100uF")
	if err != nil {
		t.Fatalf("Failed to insert inventory: %v", err)
	}

	// Insert devices
	_, err = testDB.Exec("INSERT INTO devices (serial_number, model, status) VALUES (?, ?, ?)",
		"SN12345", "MODEL-A", "active")
	if err != nil {
		t.Fatalf("Failed to insert device: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO devices (serial_number, model, status) VALUES (?, ?, ?)",
		"SN67890", "MODEL-B", "inactive")
	if err != nil {
		t.Fatalf("Failed to insert device: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO devices (serial_number, model, status) VALUES (?, ?, ?)",
		"SCAN-TEST-001", "TEST-MODEL", "active")
	if err != nil {
		t.Fatalf("Failed to insert test device: %v", err)
	}
}

func TestHandleScanLookup_EmptyCode(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupScanTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/scan?code=", nil)
	w := httptest.NewRecorder()

	handleScanLookup(w, req, "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty code, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "missing code") {
		t.Error("Expected error message about missing code")
	}
}

func TestHandleScanLookup_InventoryMatch(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	// Create temp parts directory
	partsDir = t.TempDir()
	insertScanTestData(t, db, partsDir)

	tests := []struct {
		name           string
		code           string
		expectType     string
		expectID       string
		expectMinCount int
	}{
		{
			name:           "Exact inventory match",
			code:           "PART-001",
			expectType:     "inventory",
			expectID:       "PART-001",
			expectMinCount: 1,
		},
		{
			name:           "Partial inventory match",
			code:           "PART",
			expectType:     "inventory",
			expectMinCount: 2,
		},
		{
			name:           "Case insensitive match",
			code:           "part-001",
			expectType:     "inventory",
			expectMinCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/scan?code="+tt.code, nil)
			w := httptest.NewRecorder()

			handleScanLookup(w, req, tt.code)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			results := response["results"].([]interface{})
			if len(results) < tt.expectMinCount {
				t.Errorf("Expected at least %d results, got %d", tt.expectMinCount, len(results))
			}

			if tt.expectID != "" && len(results) > 0 {
				firstResult := results[0].(map[string]interface{})
				if firstResult["type"] != tt.expectType {
					t.Errorf("Expected type %s, got %s", tt.expectType, firstResult["type"])
				}
				if firstResult["id"] != tt.expectID {
					t.Errorf("Expected id %s, got %s", tt.expectID, firstResult["id"])
				}
			}

			if code, ok := response["code"].(string); !ok || code != tt.code {
				t.Errorf("Expected code %s in response, got %s", tt.code, code)
			}
		})
	}
}

func TestHandleScanLookup_DeviceMatch(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()
	insertScanTestData(t, db, partsDir)

	tests := []struct {
		name       string
		code       string
		expectType string
		expectID   string
	}{
		{
			name:       "Exact device serial match",
			code:       "SN12345",
			expectType: "device",
			expectID:   "SN12345",
		},
		{
			name:       "Partial device serial match",
			code:       "SN123",
			expectType: "device",
			expectID:   "SN12345",
		},
		{
			name:       "Case insensitive device match",
			code:       "sn12345",
			expectType: "device",
			expectID:   "SN12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/scan?code="+tt.code, nil)
			w := httptest.NewRecorder()

			handleScanLookup(w, req, tt.code)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			results := response["results"].([]interface{})
			if len(results) < 1 {
				t.Errorf("Expected at least 1 result, got %d", len(results))
			}

			found := false
			for _, r := range results {
				result := r.(map[string]interface{})
				if result["type"] == tt.expectType && result["id"] == tt.expectID {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected to find device with type %s and id %s", tt.expectType, tt.expectID)
			}
		})
	}
}

func TestHandleScanLookup_NoMatch(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()
	insertScanTestData(t, db, partsDir)

	req := httptest.NewRequest("GET", "/api/scan?code=NONEXISTENT123", nil)
	w := httptest.NewRecorder()

	handleScanLookup(w, req, "NONEXISTENT123")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	results := response["results"].([]interface{})
	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-existent code, got %d", len(results))
	}
}

func TestHandleScanLookup_MultipleMatches(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()
	insertScanTestData(t, db, partsDir)

	// Search with a code that matches both parts and devices
	req := httptest.NewRequest("GET", "/api/scan?code=PART", nil)
	w := httptest.NewRecorder()

	handleScanLookup(w, req, "PART")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	results := response["results"].([]interface{})
	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 'PART', got %d", len(results))
	}

	// Verify we get inventory results
	inventoryCount := 0
	for _, r := range results {
		result := r.(map[string]interface{})
		if result["type"] == "inventory" {
			inventoryCount++
		}
	}

	if inventoryCount < 2 {
		t.Errorf("Expected at least 2 inventory results, got %d", inventoryCount)
	}
}

func TestHandleScanLookup_SQLInjection(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()
	insertScanTestData(t, db, partsDir)

	sqlInjectionAttempts := []string{
		"'; DROP TABLE inventory; --",
		"' OR '1'='1",
		"1' UNION SELECT * FROM devices --",
		"'; DELETE FROM devices; --",
		"SN12345'; UPDATE devices SET status='hacked' WHERE '1'='1",
	}

	for _, attempt := range sqlInjectionAttempts {
		t.Run("SQL_Injection_"+attempt, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/scan?code="+url.QueryEscape(attempt), nil)
			w := httptest.NewRecorder()

			// Should not panic or cause SQL errors
			handleScanLookup(w, req, attempt)

			if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 200 or 400, got %d", w.Code)
			}

			// Verify tables still exist and data is intact
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM inventory").Scan(&count)
			if err != nil {
				t.Errorf("inventory table damaged by SQL injection: %v", err)
			}
			if count != 2 {
				t.Errorf("Expected 2 inventory items, got %d - data may have been modified", count)
			}

			err = db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
			if err != nil {
				t.Errorf("devices table damaged by SQL injection: %v", err)
			}
			if count != 3 {
				t.Errorf("Expected 3 devices, got %d - data may have been modified", count)
			}
		})
	}
}

func TestHandleScanLookup_MalformedBarcodes(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()
	insertScanTestData(t, db, partsDir)

	malformedInputs := []string{
		"", // empty
		strings.Repeat("A", 1000), // very long
		"\x00\x01\x02", // null bytes and control chars
		"../../../etc/passwd", // path traversal
		"<script>alert('xss')</script>", // XSS attempt
		"'\"; rm -rf /; --", // command injection attempt
		"�����", // invalid UTF-8
	}

	for i, input := range malformedInputs {
		t.Run(fmt.Sprintf("Malformed_%d", i), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/scan?code="+input, nil)
			w := httptest.NewRecorder()

			// Should handle gracefully without panicking
			handleScanLookup(w, req, input)

			// Accept either bad request or OK with empty results
			if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 200 or 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleScanLookup_XSSInResults(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()

	// Insert device with XSS payload in model field
	xssPayload := "<script>alert('xss')</script>"
	_, err := db.Exec("INSERT INTO devices (serial_number, model, status) VALUES (?, ?, ?)",
		"XSS-TEST", xssPayload, "active")
	if err != nil {
		t.Fatalf("Failed to insert XSS test device: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/scan?code=XSS-TEST", nil)
	w := httptest.NewRecorder()

	handleScanLookup(w, req, "XSS-TEST")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify JSON encoding escapes HTML
	body := w.Body.String()
	if strings.Contains(body, "<script>") {
		t.Error("Response contains unescaped HTML/script tags - XSS vulnerability!")
	}

	// Should contain escaped version
	if !strings.Contains(body, "\\u003c") && !strings.Contains(body, "&lt;") {
		t.Log("Warning: XSS payload may not be properly escaped")
	}
}

func TestHandleScanLookup_ResultStructure(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()
	insertScanTestData(t, db, partsDir)

	req := httptest.NewRequest("GET", "/api/scan?code=SN12345", nil)
	w := httptest.NewRecorder()

	handleScanLookup(w, req, "SN12345")

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response structure
	if _, ok := response["results"]; !ok {
		t.Error("Response missing 'results' field")
	}

	if _, ok := response["code"]; !ok {
		t.Error("Response missing 'code' field")
	}

	results := response["results"].([]interface{})
	if len(results) > 0 {
		result := results[0].(map[string]interface{})

		// Verify ScanResult structure
		requiredFields := []string{"type", "id", "label", "link"}
		for _, field := range requiredFields {
			if _, ok := result[field]; !ok {
				t.Errorf("Result missing required field '%s'", field)
			}
		}

		// Verify link format
		if link, ok := result["link"].(string); ok {
			if !strings.HasPrefix(link, "/") {
				t.Errorf("Expected link to start with '/', got %s", link)
			}
		}
	}
}

func TestHandleScanLookup_DeduplicationInventory(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()

	// Insert multiple inventory entries with same IPN (different locations)
	db.Exec("INSERT INTO inventory (ipn, location, qty) VALUES (?, ?, ?)",
		"DUP-PART", "A1", 10.0)
	db.Exec("INSERT INTO inventory (ipn, location, qty) VALUES (?, ?, ?)",
		"DUP-PART", "B2", 20.0)

	req := httptest.NewRequest("GET", "/api/scan?code=DUP-PART", nil)
	w := httptest.NewRecorder()

	handleScanLookup(w, req, "DUP-PART")

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	results := response["results"].([]interface{})

	// Count inventory results
	inventoryCount := 0
	for _, r := range results {
		result := r.(map[string]interface{})
		if result["type"] == "inventory" {
			inventoryCount++
		}
	}

	// Should be deduplicated (only 1 result per IPN despite multiple locations)
	if inventoryCount != 1 {
		t.Errorf("Expected 1 deduplicated inventory result, got %d", inventoryCount)
	}
}

func TestHandleScanLookup_ContentTypeJSON(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()

	req := httptest.NewRequest("GET", "/api/scan?code=test", nil)
	w := httptest.NewRecorder()

	handleScanLookup(w, req, "test")

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestHandleScanLookup_EmptyResultsArray(t *testing.T) {
	origDB := db
	origPartsDir := partsDir
	defer func() {
		db = origDB
		partsDir = origPartsDir
	}()

	db = setupScanTestDB(t)
	defer db.Close()

	partsDir = t.TempDir()

	req := httptest.NewRequest("GET", "/api/scan?code=NOEXIST", nil)
	w := httptest.NewRecorder()

	handleScanLookup(w, req, "NOEXIST")

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	results := response["results"].([]interface{})
	if results == nil {
		t.Error("Expected empty array, got nil")
	}

	if len(results) != 0 {
		t.Errorf("Expected empty array, got %d results", len(results))
	}
}
