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

func setupSearchTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create ecos table
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	// Create work_orders table
	_, err = testDB.Exec(`
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Create devices table
	_, err = testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			ipn TEXT NOT NULL,
			customer TEXT,
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	// Create ncrs table
	_, err = testDB.Exec(`
		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
	}

	// Create purchase_orders table
	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			status TEXT DEFAULT 'draft',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create purchase_orders table: %v", err)
	}

	// Create quotes table
	_, err = testDB.Exec(`
		CREATE TABLE quotes (
			id TEXT PRIMARY KEY,
			customer TEXT,
			status TEXT DEFAULT 'draft',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create quotes table: %v", err)
	}

	return testDB
}

func insertTestSearchData(t *testing.T, db *sql.DB) {
	// Insert ECOs
	_, err := db.Exec("INSERT INTO ecos (id, title, description, status) VALUES (?, ?, ?, ?)",
		"ECO-001", "Test ECO", "Description for ECO-001", "draft")
	if err != nil {
		t.Fatalf("Failed to insert test ECO: %v", err)
	}

	_, err = db.Exec("INSERT INTO ecos (id, title, description, status) VALUES (?, ?, ?, ?)",
		"ECO-002", "Another ECO", "This contains keyword search_term", "approved")
	if err != nil {
		t.Fatalf("Failed to insert test ECO: %v", err)
	}

	// Insert Work Orders
	_, err = db.Exec("INSERT INTO work_orders (id, assembly_ipn, status) VALUES (?, ?, ?)",
		"WO-001", "ASSY-100", "pending")
	if err != nil {
		t.Fatalf("Failed to insert test WO: %v", err)
	}

	_, err = db.Exec("INSERT INTO work_orders (id, assembly_ipn, status) VALUES (?, ?, ?)",
		"WO-002", "ASSY-200", "in_progress")
	if err != nil {
		t.Fatalf("Failed to insert test WO: %v", err)
	}

	// Insert Devices
	_, err = db.Exec("INSERT INTO devices (serial_number, ipn, customer, status) VALUES (?, ?, ?, ?)",
		"SN12345", "DEV-100", "Acme Corp", "active")
	if err != nil {
		t.Fatalf("Failed to insert test device: %v", err)
	}

	_, err = db.Exec("INSERT INTO devices (serial_number, ipn, customer, status) VALUES (?, ?, ?, ?)",
		"SN67890", "DEV-200", "Beta Inc", "inactive")
	if err != nil {
		t.Fatalf("Failed to insert test device: %v", err)
	}

	// Insert NCRs
	_, err = db.Exec("INSERT INTO ncrs (id, title, status) VALUES (?, ?, ?)",
		"NCR-001", "Defect in part", "open")
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	// Insert POs
	_, err = db.Exec("INSERT INTO purchase_orders (id, status) VALUES (?, ?)",
		"PO-001", "draft")
	if err != nil {
		t.Fatalf("Failed to insert test PO: %v", err)
	}

	// Insert Quotes
	_, err = db.Exec("INSERT INTO quotes (id, customer, status) VALUES (?, ?, ?)",
		"QT-001", "Acme Corp", "sent")
	if err != nil {
		t.Fatalf("Failed to insert test quote: %v", err)
	}
}

func TestHandleGlobalSearch_EmptyQuery(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupSearchTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/search?q=", nil)
	w := httptest.NewRecorder()

	handleGlobalSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify empty arrays returned
	data := response["data"].(map[string]interface{})
	if parts, ok := data["parts"].([]interface{}); !ok || len(parts) != 0 {
		t.Errorf("Expected empty parts array")
	}
	if ecos, ok := data["ecos"].([]interface{}); !ok || len(ecos) != 0 {
		t.Errorf("Expected empty ecos array")
	}
}

func TestHandleGlobalSearch_BasicSearch(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupSearchTestDB(t)
	defer db.Close()

	insertTestSearchData(t, db)

	tests := []struct {
		name          string
		query         string
		expectECOs    int
		expectWOs     int
		expectDevices int
	}{
		{
			name:       "Search by ECO ID",
			query:      "ECO-001",
			expectECOs: 1,
		},
		{
			name:      "Search by Work Order ID",
			query:     "WO-001",
			expectWOs: 1,
		},
		{
			name:          "Search by Device Serial",
			query:         "SN12345",
			expectDevices: 1,
		},
		{
			name:          "Search by Customer",
			query:         "Acme",
			expectDevices: 1,
		},
		{
			name:       "Search by description content",
			query:      "search_term",
			expectECOs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/search?q="+tt.query, nil)
			w := httptest.NewRecorder()

			handleGlobalSearch(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			data := response["data"].(map[string]interface{})

			if tt.expectECOs > 0 {
				ecos := data["ecos"].([]interface{})
				if len(ecos) != tt.expectECOs {
					t.Errorf("Expected %d ECOs, got %d", tt.expectECOs, len(ecos))
				}
			}

			if tt.expectWOs > 0 {
				wos := data["workorders"].([]interface{})
				if len(wos) != tt.expectWOs {
					t.Errorf("Expected %d WOs, got %d", tt.expectWOs, len(wos))
				}
			}

			if tt.expectDevices > 0 {
				devices := data["devices"].([]interface{})
				if len(devices) != tt.expectDevices {
					t.Errorf("Expected %d devices, got %d", tt.expectDevices, len(devices))
				}
			}
		})
	}
}

func TestHandleGlobalSearch_Limit(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupSearchTestDB(t)
	defer db.Close()

	insertTestSearchData(t, db)

	// Insert multiple ECOs
	for i := 3; i <= 25; i++ {
		db.Exec("INSERT INTO ecos (id, title, status) VALUES (?, ?, ?)",
			"ECO-"+fmt.Sprintf("%03d", i), "Test ECO "+fmt.Sprintf("%d", i), "draft")
	}

	tests := []struct {
		name      string
		limit     string
		maxECOs   int
	}{
		{
			name:    "Default limit (20)",
			limit:   "",
			maxECOs: 20,
		},
		{
			name:    "Custom limit (5)",
			limit:   "5",
			maxECOs: 5,
		},
		{
			name:    "Large limit (100)",
			limit:   "100",
			maxECOs: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/search?q=ECO"
			if tt.limit != "" {
				url += "&limit=" + tt.limit
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			handleGlobalSearch(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			data := response["data"].(map[string]interface{})
			ecos := data["ecos"].([]interface{})

			if len(ecos) > tt.maxECOs {
				t.Errorf("Expected max %d ECOs, got %d", tt.maxECOs, len(ecos))
			}
		})
	}
}

func TestHandleGlobalSearch_SQLInjection(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupSearchTestDB(t)
	defer db.Close()

	insertTestSearchData(t, db)

	sqlInjectionAttempts := []string{
		"'; DROP TABLE ecos; --",
		"' OR '1'='1",
		"1' UNION SELECT * FROM users --",
		"<script>alert('xss')</script>",
		"'; DELETE FROM work_orders; --",
	}

	for _, attempt := range sqlInjectionAttempts {
		t.Run("SQL_Injection_"+attempt, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/search?q="+url.QueryEscape(attempt), nil)
			w := httptest.NewRecorder()

			// Should not panic or cause SQL errors
			handleGlobalSearch(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 even with SQL injection attempt, got %d", w.Code)
			}

			// Verify tables still exist
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM ecos").Scan(&count)
			if err != nil {
				t.Errorf("ECOs table damaged by SQL injection: %v", err)
			}
		})
	}
}

func TestHandleGlobalSearch_CaseInsensitive(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupSearchTestDB(t)
	defer db.Close()

	insertTestSearchData(t, db)

	tests := []struct {
		name  string
		query string
	}{
		{"Lowercase", "eco-001"},
		{"Uppercase", "ECO-001"},
		{"Mixed case", "EcO-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/search?q="+tt.query, nil)
			w := httptest.NewRecorder()

			handleGlobalSearch(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			data := response["data"].(map[string]interface{})
			ecos := data["ecos"].([]interface{})

			if len(ecos) != 1 {
				t.Errorf("Case-insensitive search failed for %s", tt.query)
			}
		})
	}
}

func TestHandleGlobalSearch_PartialMatch(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupSearchTestDB(t)
	defer db.Close()

	insertTestSearchData(t, db)

	tests := []struct {
		name          string
		query         string
		expectResults bool
	}{
		{"Partial ECO ID", "ECO", true},
		{"Partial Work Order", "WO", true},
		{"Partial Device Serial", "SN123", true},
		{"Partial Customer", "Acme", true},
		{"No match", "NONEXISTENT123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/search?q="+tt.query, nil)
			w := httptest.NewRecorder()

			handleGlobalSearch(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			meta := response["meta"].(map[string]interface{})
			total := int(meta["total"].(float64))

			if tt.expectResults && total == 0 {
				t.Errorf("Expected results for %s, got none", tt.query)
			}
			if !tt.expectResults && total > 0 {
				t.Errorf("Expected no results for %s, got %d", tt.query, total)
			}
		})
	}
}

func TestHandleGlobalSearch_MetadataResponse(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupSearchTestDB(t)
	defer db.Close()

	insertTestSearchData(t, db)

	req := httptest.NewRequest("GET", "/api/search?q=ECO", nil)
	w := httptest.NewRecorder()

	handleGlobalSearch(w, req)

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check metadata exists
	meta, ok := response["meta"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected meta field in response")
	}

	if _, ok := meta["total"]; !ok {
		t.Error("Expected total in meta")
	}

	if query, ok := meta["query"].(string); !ok || query != "ECO" {
		t.Errorf("Expected query 'ECO' in meta, got %s", query)
	}
}

func TestHandleGlobalSearch_XSSInResults(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupSearchTestDB(t)
	defer db.Close()

	// Insert data with potential XSS
	xssPayload := "<script>alert('xss')</script>"
	db.Exec("INSERT INTO ecos (id, title, description) VALUES (?, ?, ?)",
		"ECO-XSS", xssPayload, "Test")

	req := httptest.NewRequest("GET", "/api/search?q=ECO-XSS", nil)
	w := httptest.NewRecorder()

	handleGlobalSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify JSON encoding escapes HTML
	body := w.Body.String()
	if strings.Contains(body, "<script>") {
		t.Error("Response contains unescaped HTML/script tags - XSS vulnerability!")
	}
}
