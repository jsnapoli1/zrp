package main

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupChangesTestDB(t *testing.T) func() {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create change_history table
	_, err = testDB.Exec(`
		CREATE TABLE change_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			table_name TEXT NOT NULL,
			record_id TEXT NOT NULL,
			operation TEXT NOT NULL,
			old_data TEXT,
			new_data TEXT,
			user_id TEXT NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			undone INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create change_history table: %v", err)
	}

	// Create test tables for restore operations
	_, err = testDB.Exec(`
		CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create vendors table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE parts (
			ipn TEXT PRIMARY KEY,
			title TEXT,
			category TEXT,
			status TEXT DEFAULT 'active',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			action TEXT,
			module TEXT,
			record_id TEXT,
			summary TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	// Initialize wsHub for broadcasting (if needed)
	// wsHub is already initialized globally

	// Save and swap db
	origDB := db
	db = testDB

	return func() {
		db.Close()
		db = origDB
	}
}

func TestRecordChangeCreate(t *testing.T) {
	cleanup := setupChangesTestDB(t)
	defer cleanup()

	newData := `{"id":"V-001","name":"Test Vendor","status":"active"}`
	id, err := recordChange("admin", "vendors", "V-001", "create", "", newData)
	
	if err != nil {
		t.Fatalf("recordChange failed: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero change ID")
	}

	// Verify in database
	var count int
	db.QueryRow("SELECT COUNT(*) FROM change_history WHERE table_name='vendors' AND operation='create'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 change record, got %d", count)
	}
}

// Helper functions
func stringReader(s string) *strings.Reader {
	return strings.NewReader(s)
}
