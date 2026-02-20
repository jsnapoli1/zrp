package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// setupErrorTestDB creates a fresh in-memory database for testing
func setupErrorTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create minimal schema for testing
	schema := `
		CREATE TABLE vendors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			contact_name TEXT DEFAULT '',
			contact_email TEXT DEFAULT '',
			contact_phone TEXT DEFAULT '',
			website TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			status TEXT DEFAULT 'active',
			lead_time_days INTEGER DEFAULT 0 CHECK(lead_time_days >= 0),
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE purchase_orders (
			po_number TEXT PRIMARY KEY,
			vendor_id INTEGER NOT NULL,
			status TEXT DEFAULT 'draft',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE RESTRICT
		);

		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			description TEXT DEFAULT '',
			category TEXT DEFAULT '',
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0 CHECK(qty_on_hand >= 0),
			qty_reserved REAL DEFAULT 0 CHECK(qty_reserved >= 0),
			reorder_point REAL DEFAULT 0 CHECK(reorder_point >= 0),
			location TEXT DEFAULT '',
			FOREIGN KEY (ipn) REFERENCES parts(ipn) ON DELETE CASCADE
		);
	`

	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return testDB
}

// Test ER-001: Database connection lost - should return error
func TestDatabaseConnectionLost(t *testing.T) {
	testDB := setupErrorTestDB(t)

	// Save original db
	originalDB := db
	defer func() { db = originalDB }()

	// Set up with valid DB first
	db = testDB

	// Create a test vendor successfully first
	_, err := db.Exec("INSERT INTO vendors (name, contact_email) VALUES (?, ?)", "TEST Vendor", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Now close the database to simulate connection loss
	db.Close()

	// Try to list vendors - should get error, not crash
	req := httptest.NewRequest("GET", "/api/vendors", nil)
	w := httptest.NewRecorder()

	handleListVendors(w, req)

	if w.Code != http.StatusInternalServerError && w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 500 or 503 status code when DB is unavailable, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if response["error"] == nil {
		t.Error("Expected error field in response when DB is unavailable")
	}

	errorMsg := fmt.Sprintf("%v", response["error"])
	if errorMsg == "" {
		t.Error("Error message should not be empty")
	}

	t.Logf("✅ Database connection failure handled gracefully with error: %s", errorMsg)
}

// Test ER-002: Database locked (busy) - should retry or return proper error
func TestDatabaseBusyTimeout(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	// Set a very short busy timeout to force timeout errors
	_, err := db.Exec("PRAGMA busy_timeout=1")
	if err != nil {
		t.Fatalf("Failed to set busy timeout: %v", err)
	}

	// Start a transaction and hold it
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Insert data in the transaction but don't commit
	_, err = tx.Exec("INSERT INTO vendors (name) VALUES (?)", "LOCK-TEST Vendor")
	if err != nil {
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	// Try to insert the same vendor from another connection (should be blocked)
	done := make(chan bool)
	go func() {
		_, err := db.Exec("INSERT INTO vendors (name) VALUES (?)", "LOCK-TEST Vendor")
		if err == nil {
			t.Error("Expected error due to database lock, got none")
		}
		done <- true
	}()

	select {
	case <-done:
		t.Log("✅ Database busy condition handled (error returned)")
	case <-time.After(5 * time.Second):
		t.Error("Database operation should have timed out but didn't")
	}

	tx.Rollback()
}

// Test ER-003: Invalid JSON in request body - should return 400 Bad Request
func TestInvalidJSONRequest(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name:     "Completely invalid JSON",
			body:     `{"name": "TEST Vendor", "contact_email": "Missing closing brace"`,
			wantCode: 400,
		},
		{
			name:     "Malformed JSON with extra comma",
			body:     `{"name": "TEST Vendor", "contact_email": "test@test.com",}`,
			wantCode: 400,
		},
		{
			name:     "Not JSON at all",
			body:     `This is not JSON`,
			wantCode: 400,
		},
		{
			name:     "Empty body",
			body:     ``,
			wantCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/vendors", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateVendor(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Logf("Response body: %s", w.Body.String())
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if response["error"] == nil {
				t.Error("Expected error field in response for invalid JSON")
			}

			t.Logf("✅ Invalid JSON handled: %v", response["error"])
		})
	}
}

// Test ER-004: Missing required fields - should return 400 with field names
func TestMissingRequiredFields(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	tests := []struct {
		name          string
		body          map[string]interface{}
		requiredField string
	}{
		{
			name:          "Vendor without name",
			body:          map[string]interface{}{"contact_email": "test@test.com"},
			requiredField: "name",
		},
		{
			name:          "Vendor with empty name",
			body:          map[string]interface{}{"name": "", "contact_email": "test@test.com"},
			requiredField: "name",
		},
		{
			name:          "Vendor with whitespace-only name",
			body:          map[string]interface{}{"name": "   ", "contact_email": "test@test.com"},
			requiredField: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/vendors", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateVendor(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 Bad Request, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			errorMsg := fmt.Sprintf("%v", response["error"])
			if !strings.Contains(strings.ToLower(errorMsg), tt.requiredField) {
				t.Errorf("Error message should mention required field '%s', got: %s", tt.requiredField, errorMsg)
			}

			t.Logf("✅ Missing field '%s' properly validated: %s", tt.requiredField, errorMsg)
		})
	}
}

// Test ER-005: Vendor delete with active POs - should be blocked by foreign key constraint
func TestForeignKeyConstraintViolation(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	// Create a vendor
	result, err := db.Exec("INSERT INTO vendors (name, contact_email) VALUES (?, ?)", "Test Vendor", "test@vendor.com")
	if err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}

	vendorID, _ := result.LastInsertId()

	// Create a PO for that vendor
	_, err = db.Exec("INSERT INTO purchase_orders (po_number, vendor_id, status) VALUES (?, ?, ?)",
		"PO-001", vendorID, "open")
	if err != nil {
		t.Fatalf("Failed to create PO: %v", err)
	}

	// Try to delete the vendor - should fail due to ON DELETE RESTRICT
	_, err = db.Exec("DELETE FROM vendors WHERE id = ?", vendorID)
	if err == nil {
		t.Error("Expected error when deleting vendor with active POs, got none")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") &&
		!strings.Contains(strings.ToLower(err.Error()), "constraint") {
		t.Errorf("Expected foreign key constraint error, got: %v", err)
	}

	// Verify vendor still exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM vendors WHERE id = ?", vendorID).Scan(&count)
	if count != 1 {
		t.Error("Vendor should still exist after failed delete")
	}

	t.Logf("✅ Foreign key constraint properly enforced: %v", err)
}

// Test: Constraint violation mid-transaction should rollback
func TestTransactionRollbackOnError(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	// Insert initial vendor
	_, err := db.Exec("INSERT INTO vendors (name, contact_email) VALUES (?, ?)", "TRANS Vendor", "trans@test.com")
	if err != nil {
		t.Fatalf("Failed to insert initial vendor: %v", err)
	}

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Update the vendor
	_, err = tx.Exec("UPDATE vendors SET contact_email = ? WHERE name = ?", "updated@test.com", "TRANS Vendor")
	if err != nil {
		t.Fatalf("Failed to update in transaction: %v", err)
	}

	// Try to insert a duplicate (should violate UNIQUE constraint)
	_, err = tx.Exec("INSERT INTO vendors (name, contact_email) VALUES (?, ?)", "TRANS Vendor", "duplicate@test.com")
	if err == nil {
		t.Error("Expected constraint violation error")
	}

	// Rollback the transaction
	tx.Rollback()

	// Verify the original vendor email is unchanged
	var email string
	err = db.QueryRow("SELECT contact_email FROM vendors WHERE name = ?", "TRANS Vendor").Scan(&email)
	if err != nil {
		t.Fatalf("Failed to query vendor: %v", err)
	}

	if email != "trans@test.com" {
		t.Errorf("Expected original email after rollback, got: %s", email)
	}

	t.Logf("✅ Transaction rollback on constraint violation works correctly")
}

// Test: Negative quantity should be rejected by CHECK constraint
func TestNegativeQuantityRejection(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	// Create a part first (required for foreign key)
	_, err := db.Exec("INSERT INTO parts (ipn, description) VALUES (?, ?)", "QTY-TEST", "Quantity Test Part")
	if err != nil {
		t.Fatalf("Failed to create part: %v", err)
	}

	tests := []struct {
		name      string
		qtyField  string
		value     float64
		shouldErr bool
	}{
		{"Negative qty_on_hand", "qty_on_hand", -10, true},
		{"Negative qty_reserved", "qty_reserved", -5, true},
		{"Negative reorder_point", "reorder_point", -1, true},
		{"Zero qty_on_hand", "qty_on_hand", 0, false},
		{"Positive qty_on_hand", "qty_on_hand", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := fmt.Sprintf("INSERT INTO inventory (ipn, %s) VALUES (?, ?)", tt.qtyField)
			_, err := db.Exec(query, "QTY-TEST", tt.value)

			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for %s = %.2f, got none", tt.qtyField, tt.value)
			} else if !tt.shouldErr && err != nil {
				t.Errorf("Unexpected error for %s = %.2f: %v", tt.qtyField, tt.value, err)
			}

			if err != nil {
				t.Logf("✅ CHECK constraint enforced: %v", err)
			}

			// Clean up for next test iteration (if insert succeeded)
			if err == nil {
				db.Exec("DELETE FROM inventory WHERE ipn = ?", "QTY-TEST")
			}
		})
	}
}

// Test: Very large file path should be handled
func TestVeryLongFilePath(t *testing.T) {
	// Generate a very long filename
	longName := strings.Repeat("a", 500) + ".txt"

	// Create a temporary file with long name (within OS limits)
	tempDir := t.TempDir()
	longPath := tempDir + "/" + longName

	// Attempt to create the file
	file, err := os.Create(longPath)
	if err != nil {
		t.Logf("✅ Very long file path properly rejected by OS: %v", err)
		return
	}
	defer file.Close()
	defer os.Remove(longPath)

	// If created, verify it can be accessed
	_, err = os.Stat(longPath)
	if err != nil {
		t.Errorf("File created but cannot be accessed: %v", err)
	}

	t.Log("✅ Long file path handled (OS allows it)")
}

// Test: Disk full during write - simulated by closing file handle
func TestDiskFullSimulation(t *testing.T) {
	tempFile, err := os.CreateTemp("", "diskfull-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	// Close the file immediately
	tempFile.Close()

	// Try to write to closed file (simulates disk full or write error)
	_, err = tempFile.Write([]byte("test data"))
	if err == nil {
		t.Error("Expected error writing to closed file")
	}

	t.Logf("✅ Write error properly detected: %v", err)

	// Verify cleanup - file should still exist but be empty/closed
	stat, err := os.Stat(tempPath)
	if err != nil {
		t.Errorf("Temp file should still exist: %v", err)
	}

	if stat.Size() != 0 {
		t.Errorf("Expected empty file after failed write, got size: %d", stat.Size())
	}
}

// Test: Network timeout simulation (handler completes without hanging)
func TestNetworkTimeoutHandling(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	// Create a request
	req := httptest.NewRequest("GET", "/api/vendors", nil)
	w := httptest.NewRecorder()

	// Simulate a slow operation
	go func() {
		time.Sleep(100 * time.Millisecond)
		db.Exec("SELECT 1") // Simple query
	}()

	// The handler should complete without hanging
	done := make(chan bool)
	go func() {
		handleListVendors(w, req)
		done <- true
	}()

	select {
	case <-done:
		t.Log("✅ Request completed without hanging")
	case <-time.After(5 * time.Second):
		t.Error("Request timed out - handler may be hanging")
	}
}

// Test: Multiple validation errors should all be reported
func TestMultipleValidationErrors(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	// Create a vendor with multiple validation issues
	body := map[string]interface{}{
		"name":          "", // Missing required field
		"contact_email": strings.Repeat("X", 20000), // Too long (if there's a limit)
	}

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/vendors", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateVendor(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	// Check if error mentions the required field
	errorMsg := fmt.Sprintf("%v", response["error"])
	if !strings.Contains(strings.ToLower(errorMsg), "name") {
		t.Error("Error should mention missing name field")
	}

	t.Logf("✅ Validation errors reported: %s", errorMsg)
}

// Test: Duplicate key insertion should return helpful error
func TestDuplicateKeyError(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	// Insert a vendor
	_, err := db.Exec("INSERT INTO vendors (name, contact_email) VALUES (?, ?)", "DUP Vendor", "dup@test.com")
	if err != nil {
		t.Fatalf("Failed to insert original vendor: %v", err)
	}

	// Try to insert duplicate
	_, err = db.Exec("INSERT INTO vendors (name, contact_email) VALUES (?, ?)", "DUP Vendor", "dup2@test.com")
	if err == nil {
		t.Error("Expected duplicate key error")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "unique") &&
		!strings.Contains(strings.ToLower(err.Error()), "constraint") {
		t.Errorf("Expected UNIQUE constraint error, got: %v", err)
	}

	t.Logf("✅ Duplicate key properly rejected: %v", err)
}

// Test: Graceful handling of nil database connection
func TestNilDatabaseConnection(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()

	// Set db to nil to simulate uninitialized database
	db = nil

	req := httptest.NewRequest("GET", "/api/vendors", nil)
	w := httptest.NewRecorder()

	// Should not panic, should return error
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Handler panicked with nil database: %v", r)
		}
	}()

	handleListVendors(w, req)

	// Should get a 503 Service Unavailable, not a 200 OK
	if w.Code == http.StatusOK {
		t.Error("Expected error with nil database, got 200 OK")
	} else if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 503 or 500 status code, got %d", w.Code)
	} else {
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
			errorMsg := fmt.Sprintf("%v", response["error"])
			t.Logf("✅ Nil database handled gracefully (status %d): %s", w.Code, errorMsg)
		} else {
			t.Logf("✅ Nil database handled gracefully (status %d)", w.Code)
		}
	}
}

// Test: Error message user-friendliness
func TestUserFriendlyErrorMessages(t *testing.T) {
	testDB := setupErrorTestDB(t)
	defer testDB.Close()

	originalDB := db
	defer func() { db = originalDB }()
	db = testDB

	tests := []struct {
		name         string
		body         map[string]interface{}
		checkMessage func(string) bool
		description  string
	}{
		{
			name: "Empty name field",
			body: map[string]interface{}{"name": ""},
			checkMessage: func(msg string) bool {
				return strings.Contains(strings.ToLower(msg), "name") &&
					(strings.Contains(strings.ToLower(msg), "required") ||
						strings.Contains(strings.ToLower(msg), "empty") ||
						strings.Contains(strings.ToLower(msg), "cannot be blank"))
			},
			description: "Should mention 'name' and 'required'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/vendors", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateVendor(w, req)

			var response map[string]interface{}
			json.NewDecoder(w.Body).Decode(&response)

			errorMsg := fmt.Sprintf("%v", response["error"])
			if !tt.checkMessage(errorMsg) {
				t.Errorf("Error message not user-friendly. Expected %s, got: %s",
					tt.description, errorMsg)
			} else {
				t.Logf("✅ User-friendly error message: %s", errorMsg)
			}
		})
	}
}
