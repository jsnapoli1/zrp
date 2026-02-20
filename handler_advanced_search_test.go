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
	"time"

	_ "modernc.org/sqlite"
)

func setupAdvancedSearchTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create parts_view (simplified)
	_, err = testDB.Exec(`
		CREATE TABLE parts_view (
			ipn TEXT PRIMARY KEY,
			category TEXT,
			fields TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create parts_view: %v", err)
	}

	// Create work_orders table
	_, err = testDB.Exec(`
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER DEFAULT 1,
			status TEXT DEFAULT 'pending',
			priority TEXT DEFAULT 'normal',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME,
			due_date TEXT,
			qty_good INTEGER DEFAULT 0,
			qty_scrap INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Create ecos table
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT,
			created_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME,
			approved_at DATETIME,
			approved_by TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	// Create inventory table
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			qty_reserved REAL DEFAULT 0,
			location TEXT,
			reorder_point REAL DEFAULT 0,
			reorder_qty REAL DEFAULT 0,
			description TEXT,
			mpn TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create ncrs table
	_, err = testDB.Exec(`
		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			ipn TEXT,
			serial_number TEXT,
			defect_type TEXT,
			severity TEXT,
			status TEXT DEFAULT 'open',
			root_cause TEXT,
			corrective_action TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME,
			created_by TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
	}

	// Create devices table
	_, err = testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			ipn TEXT NOT NULL,
			firmware_version TEXT,
			customer TEXT,
			location TEXT,
			status TEXT DEFAULT 'active',
			install_date TEXT,
			last_seen DATETIME,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	// Create purchase_orders table
	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date TEXT,
			received_at DATETIME,
			created_by TEXT,
			total REAL DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create purchase_orders table: %v", err)
	}

	// Create saved_searches table
	_, err = testDB.Exec(`
		CREATE TABLE saved_searches (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			filters TEXT,
			sort_by TEXT,
			sort_order TEXT,
			created_by TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_public INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create saved_searches table: %v", err)
	}

	// Create search_history table
	_, err = testDB.Exec(`
		CREATE TABLE search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			search_text TEXT,
			filters TEXT,
			searched_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create search_history table: %v", err)
	}

	return testDB
}

func insertAdvancedSearchTestData(t *testing.T, db *sql.DB) {
	// Insert work orders (include all fields to avoid NULLs)
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, notes, created_at, due_date, qty_good, qty_scrap) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"WO-001", "ASSY-100", 10, "pending", "high", "", time.Now(), "", 0, 0)
	if err != nil {
		t.Fatalf("Failed to insert test WO: %v", err)
	}

	_, err = db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, notes, created_at, due_date, qty_good, qty_scrap) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"WO-002", "ASSY-200", 20, "in_progress", "normal", "", time.Now().Add(-24*time.Hour), "", 0, 0)
	if err != nil {
		t.Fatalf("Failed to insert test WO: %v", err)
	}

	_, err = db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, notes, created_at, due_date, qty_good, qty_scrap) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"WO-003", "ASSY-300", 5, "completed", "low", "", time.Now().Add(-48*time.Hour), "", 0, 0)
	if err != nil {
		t.Fatalf("Failed to insert test WO: %v", err)
	}

	// Insert ECOs
	_, err = db.Exec(`INSERT INTO ecos (id, title, description, status, priority, created_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"ECO-001", "Critical Update", "Security fix", "approved", "high", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test ECO: %v", err)
	}

	_, err = db.Exec(`INSERT INTO ecos (id, title, description, status, priority, created_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"ECO-002", "Minor Enhancement", "UI improvement", "draft", "normal", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test ECO: %v", err)
	}

	// Insert inventory
	_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved, location, reorder_point) 
		VALUES (?, ?, ?, ?, ?)`,
		"PART-001", 100.0, 10.0, "A1", 20.0)
	if err != nil {
		t.Fatalf("Failed to insert test inventory: %v", err)
	}

	_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, qty_reserved, location, reorder_point) 
		VALUES (?, ?, ?, ?, ?)`,
		"PART-002", 5.0, 0.0, "B2", 50.0)
	if err != nil {
		t.Fatalf("Failed to insert test inventory: %v", err)
	}

	// Insert devices
	_, err = db.Exec(`INSERT INTO devices (serial_number, ipn, customer, status, created_at) 
		VALUES (?, ?, ?, ?, ?)`,
		"SN-001", "DEV-100", "Acme Corp", "active", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test device: %v", err)
	}

	// Insert NCRs
	_, err = db.Exec(`INSERT INTO ncrs (id, title, status, severity, created_at) 
		VALUES (?, ?, ?, ?, ?)`,
		"NCR-001", "Quality Issue", "open", "high", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test NCR: %v", err)
	}

	// Insert POs
	_, err = db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, created_at) 
		VALUES (?, ?, ?, ?)`,
		"PO-001", "VENDOR-100", "draft", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test PO: %v", err)
	}
}

func TestHandleAdvancedSearch_InvalidJSON(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() { 
		db.Close()
		db = origDB 
	}()

	req := httptest.NewRequest("POST", "/api/advanced-search", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handleAdvancedSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleAdvancedSearch_WorkOrders(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() {
		time.Sleep(200 * time.Millisecond) // Let goroutines complete
		db.Close()
		db = origDB 
	}()

	insertAdvancedSearchTestData(t, db)

	// Verify data was inserted
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM work_orders").Scan(&count); err != nil {
		t.Fatalf("Failed to verify test data: %v", err)
	}
	if count != 3 {
		t.Fatalf("Expected 3 work orders in test DB, got %d", count)
	}

	tests := []struct {
		name           string
		query          SearchQuery
		expectedCount  int
		expectedStatus string
	}{
		{
			name: "Search all work orders",
			query: SearchQuery{
				EntityType: "workorders",
				Limit:      50,
				SortOrder:  "asc",
			},
			expectedCount: 3,
		},
		{
			name: "Filter by status",
			query: SearchQuery{
				EntityType: "workorders",
				Filters: []SearchFilter{
					{Field: "status", Operator: "eq", Value: "pending"},
				},
				Limit:     50,
				SortOrder: "asc",
			},
			expectedCount:  1,
			expectedStatus: "pending",
		},
		{
			name: "Filter by priority",
			query: SearchQuery{
				EntityType: "workorders",
				Filters: []SearchFilter{
					{Field: "priority", Operator: "eq", Value: "high"},
				},
				Limit:     50,
				SortOrder: "asc",
			},
			expectedCount: 1,
		},
		{
			name: "Multiple filters with AND",
			query: SearchQuery{
				EntityType: "workorders",
				Filters: []SearchFilter{
					{Field: "status", Operator: "eq", Value: "pending", AndOr: "AND"},
					{Field: "priority", Operator: "eq", Value: "high"},
				},
				Limit:     50,
				SortOrder: "asc",
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.query)
			req := httptest.NewRequest("POST", "/api/advanced-search", bytes.NewBuffer(body))
			req.Header.Set("X-User-ID", "test-user")
			w := httptest.NewRecorder()

			handleAdvancedSearch(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var apiResp APIResponse
			if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Extract SearchResult from APIResponse.Data
			resultBytes, _ := json.Marshal(apiResp.Data)
			var result SearchResult
			if err := json.Unmarshal(resultBytes, &result); err != nil {
				t.Fatalf("Failed to unmarshal search result: %v", err)
			}

			if result.Total != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, result.Total)
			}

			if tt.expectedStatus != "" && result.Total > 0 {
				data := result.Data.([]interface{})
				wo := data[0].(map[string]interface{})
				if wo["status"] != tt.expectedStatus {
					t.Errorf("Expected status %s, got %s", tt.expectedStatus, wo["status"])
				}
			}
		})
	}
}

func TestHandleAdvancedSearch_ECOs(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() {
		time.Sleep(200 * time.Millisecond) // Let goroutines complete
		db.Close()
		db = origDB 
	}()

	insertAdvancedSearchTestData(t, db)

	query := SearchQuery{
		EntityType: "ecos",
		Filters: []SearchFilter{
			{Field: "status", Operator: "eq", Value: "approved"},
		},
		Limit:     50,
		SortOrder: "desc",
		SortBy:    "created_at",
	}

	body, _ := json.Marshal(query)
	req := httptest.NewRequest("POST", "/api/advanced-search", bytes.NewBuffer(body))
	req.Header.Set("X-User-ID", "test-user")
	w := httptest.NewRecorder()

	handleAdvancedSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	resultBytes, _ := json.Marshal(apiResp.Data)
	var result SearchResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal search result: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("Expected 1 approved ECO, got %d", result.Total)
	}
}

func TestHandleAdvancedSearch_Inventory(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() {
		time.Sleep(200 * time.Millisecond) // Let goroutines complete
		db.Close()
		db = origDB 
	}()

	insertAdvancedSearchTestData(t, db)

	// Search for low stock items (qty_on_hand <= reorder_point)
	query := SearchQuery{
		EntityType: "inventory",
		SearchText: "PART",
		Limit:      50,
		SortOrder:  "asc",
	}

	body, _ := json.Marshal(query)
	req := httptest.NewRequest("POST", "/api/advanced-search", bytes.NewBuffer(body))
	req.Header.Set("X-User-ID", "test-user")
	w := httptest.NewRecorder()

	handleAdvancedSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	resultBytes, _ := json.Marshal(apiResp.Data)
	var result SearchResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal search result: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("Expected 2 inventory items, got %d", result.Total)
	}
}

func TestHandleAdvancedSearch_Pagination(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() {
		time.Sleep(200 * time.Millisecond) // Let goroutines complete
		db.Close()
		db = origDB 
	}()

	insertAdvancedSearchTestData(t, db)

	// Add more work orders
	for i := 4; i <= 10; i++ {
		db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status) VALUES (?, ?, ?, ?)`,
			fmt.Sprintf("WO-%03d", i), "ASSY-100", 10, "pending")
	}

	tests := []struct {
		name         string
		limit        int
		offset       int
		expectedPage int
	}{
		{"First page", 5, 0, 1},
		{"Second page", 5, 5, 2},
		{"Third page", 3, 6, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := SearchQuery{
				EntityType: "workorders",
				Limit:      tt.limit,
				Offset:     tt.offset,
				SortOrder:  "asc",
			}

			body, _ := json.Marshal(query)
			req := httptest.NewRequest("POST", "/api/advanced-search", bytes.NewBuffer(body))
			req.Header.Set("X-User-ID", "test-user")
			w := httptest.NewRecorder()

			handleAdvancedSearch(w, req)

			var apiResp APIResponse
			if err := json.NewDecoder(w.Body).Decode(&apiResp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			resultBytes, _ := json.Marshal(apiResp.Data)
			var result SearchResult
			if err := json.Unmarshal(resultBytes, &result); err != nil {
				t.Fatalf("Failed to unmarshal search result: %v", err)
			}

			if result.Page != tt.expectedPage {
				t.Errorf("Expected page %d, got %d", tt.expectedPage, result.Page)
			}

			data := result.Data.([]interface{})
			if len(data) > tt.limit {
				t.Errorf("Expected max %d items, got %d", tt.limit, len(data))
			}
		})
	}
}

func TestHandleAdvancedSearch_UnsupportedEntity(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() { 
		db.Close()
		db = origDB 
	}()

	query := SearchQuery{
		EntityType: "unsupported_entity",
		Limit:      50,
	}

	body, _ := json.Marshal(query)
	req := httptest.NewRequest("POST", "/api/advanced-search", bytes.NewBuffer(body))
	req.Header.Set("X-User-ID", "test-user")
	w := httptest.NewRecorder()

	handleAdvancedSearch(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "unsupported entity type") {
		t.Error("Expected error message about unsupported entity type")
	}
}

func TestHandleAdvancedSearch_SQLInjection(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() { 
		db.Close()
		db = origDB 
	}()

	insertAdvancedSearchTestData(t, db)

	// Attempt SQL injection via search text
	query := SearchQuery{
		EntityType: "workorders",
		SearchText: "'; DROP TABLE work_orders; --",
		Limit:      50,
	}

	body, _ := json.Marshal(query)
	req := httptest.NewRequest("POST", "/api/advanced-search", bytes.NewBuffer(body))
	req.Header.Set("X-User-ID", "test-user")
	w := httptest.NewRecorder()

	handleAdvancedSearch(w, req)

	// Should not cause errors
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify table still exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM work_orders").Scan(&count)
	if err != nil {
		t.Errorf("work_orders table damaged by SQL injection: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 work orders, got %d - data may have been modified", count)
	}
}

func TestHandleSaveSavedSearch(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() { 
		db.Close()
		db = origDB 
	}()

	savedSearch := SavedSearch{
		Name:       "My Pending Work Orders",
		EntityType: "workorders",
		Filters: []SearchFilter{
			{Field: "status", Operator: "eq", Value: "pending"},
		},
		SortBy:    "created_at",
		SortOrder: "desc",
		IsPublic:  false,
	}

	body, _ := json.Marshal(savedSearch)
	req := httptest.NewRequest("POST", "/api/saved-searches", bytes.NewBuffer(body))
	req.Header.Set("X-User-ID", "test-user")
	w := httptest.NewRecorder()

	handleSaveSavedSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data SavedSearch `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data

	if result.ID == "" {
		t.Error("Expected ID to be generated")
	}

	if result.CreatedBy != "test-user" {
		t.Errorf("Expected created_by to be 'test-user', got %s", result.CreatedBy)
	}

	// Verify saved in database
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM saved_searches WHERE id = ?", result.ID).Scan(&count)
	if err != nil || count != 1 {
		t.Error("Saved search not found in database")
	}
}

func TestHandleGetSavedSearches(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() { 
		db.Close()
		db = origDB 
	}()

	// Insert test saved searches
	filters := `[{"field":"status","operator":"eq","value":"pending"}]`
	_, err := db.Exec(`INSERT INTO saved_searches (id, name, entity_type, filters, created_by, is_public) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"search-1", "My Search", "workorders", filters, "test-user", 0)
	if err != nil {
		t.Fatalf("Failed to insert search-1: %v", err)
	}

	_, err = db.Exec(`INSERT INTO saved_searches (id, name, entity_type, filters, created_by, is_public) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"search-2", "Public Search", "ecos", filters, "other-user", 1)
	if err != nil {
		t.Fatalf("Failed to insert search-2: %v", err)
	}

	_, err = db.Exec(`INSERT INTO saved_searches (id, name, entity_type, filters, created_by, is_public) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"search-3", "Private Search", "workorders", filters, "other-user", 0)
	if err != nil {
		t.Fatalf("Failed to insert search-3: %v", err)
	}

	tests := []struct {
		name          string
		entityType    string
		user          string
		expectedCount int
	}{
		{
			name:          "Get all searches for user",
			user:          "test-user",
			expectedCount: 2, // Own search + public search
		},
		{
			name:          "Filter by entity type",
			entityType:    "workorders",
			user:          "test-user",
			expectedCount: 1, // Only own workorders search
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/saved-searches"
			if tt.entityType != "" {
				url += "?entity_type=" + tt.entityType
			}

			req := httptest.NewRequest("GET", url, nil)
			req.Header.Set("X-User-ID", tt.user)
			w := httptest.NewRecorder()

			handleGetSavedSearches(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var resp struct {
				Data []SavedSearch `json:"data"`
			}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(resp.Data) != tt.expectedCount {
				t.Errorf("Expected %d searches, got %d", tt.expectedCount, len(resp.Data))
			}
		})
	}
}

func TestHandleDeleteSavedSearch(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() { 
		db.Close()
		db = origDB 
	}()

	filters := `[{"field":"status","operator":"eq","value":"pending"}]`
	db.Exec(`INSERT INTO saved_searches (id, name, entity_type, filters, created_by) 
		VALUES (?, ?, ?, ?, ?)`,
		"search-1", "My Search", "workorders", filters, "test-user")

	tests := []struct {
		name           string
		searchID       string
		user           string
		expectedStatus int
	}{
		{
			name:           "Delete own search",
			searchID:       "search-1",
			user:           "test-user",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Cannot delete other's search",
			searchID:       "search-1",
			user:           "other-user",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/api/saved-searches?id="+tt.searchID, nil)
			req.Header.Set("X-User-ID", tt.user)
			w := httptest.NewRecorder()

			handleDeleteSavedSearch(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandleGetQuickFilters(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/quick-filters?entity_type=workorders", nil)
	w := httptest.NewRecorder()

	handleGetQuickFilters(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []QuickFilter `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return at least some quick filters
	if len(resp.Data) == 0 {
		t.Error("Expected at least some quick filters")
	}
}

func TestHandleGetSearchHistory(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() { 
		db.Close()
		db = origDB 
	}()

	// Insert search history
	filters := `[{"field":"status","operator":"eq","value":"pending"}]`
	db.Exec(`INSERT INTO search_history (user_id, entity_type, search_text, filters) 
		VALUES (?, ?, ?, ?)`,
		"test-user", "workorders", "WO-001", filters)

	db.Exec(`INSERT INTO search_history (user_id, entity_type, search_text, filters) 
		VALUES (?, ?, ?, ?)`,
		"test-user", "ecos", "ECO-001", filters)

	tests := []struct {
		name          string
		entityType    string
		limit         string
		expectedCount int
	}{
		{
			name:          "Get all history",
			expectedCount: 2,
		},
		{
			name:          "Filter by entity type",
			entityType:    "workorders",
			expectedCount: 1,
		},
		{
			name:          "Limit results",
			limit:         "1",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/search-history?"
			if tt.entityType != "" {
				url += "entity_type=" + tt.entityType + "&"
			}
			if tt.limit != "" {
				url += "limit=" + tt.limit
			}

			req := httptest.NewRequest("GET", url, nil)
			req.Header.Set("X-User-ID", "test-user")
			w := httptest.NewRecorder()

			handleGetSearchHistory(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var resp struct {
				Data []interface{} `json:"data"`
			}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(resp.Data) != tt.expectedCount {
				t.Errorf("Expected %d history entries, got %d", tt.expectedCount, len(resp.Data))
			}
		})
	}
}

func TestHandleAdvancedSearch_DevicesAndNCRsAndPOs(t *testing.T) {
	origDB := db
	db = setupAdvancedSearchTestDB(t)
	defer func() { 
		db.Close()
		db = origDB 
	}()

	insertAdvancedSearchTestData(t, db)

	tests := []struct {
		name          string
		entityType    string
		expectedCount int
	}{
		{"Search devices", "devices", 1},
		{"Search NCRs", "ncrs", 1},
		{"Search POs", "pos", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := SearchQuery{
				EntityType: tt.entityType,
				Limit:      50,
				SortOrder:  "asc",
			}

			body, _ := json.Marshal(query)
			req := httptest.NewRequest("POST", "/api/advanced-search", bytes.NewBuffer(body))
			req.Header.Set("X-User-ID", "test-user")
			w := httptest.NewRecorder()

			handleAdvancedSearch(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var result SearchResult
			if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if result.Total != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, result.Total)
			}
		})
	}
}
