package parts_test

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zrp/internal/handlers/parts"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

// setupPartsTestDB creates an in-memory database with tables needed by parts handlers.
func setupPartsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0 CHECK(qty_on_hand >= 0),
			qty_reserved REAL DEFAULT 0 CHECK(qty_reserved >= 0),
			location TEXT,
			reorder_point REAL DEFAULT 0 CHECK(reorder_point >= 0),
			reorder_qty REAL DEFAULT 0 CHECK(reorder_qty >= 0),
			description TEXT DEFAULT '',
			mpn TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create purchase_orders table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			qty_ordered INTEGER DEFAULT 0,
			unit_price REAL DEFAULT 0,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create po_lines table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE market_pricing (
			ipn TEXT,
			qty INTEGER,
			price REAL,
			source TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(ipn, qty)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create market_pricing table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			module TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL DEFAULT '',
			record_id TEXT NOT NULL DEFAULT '',
			user_id INTEGER,
			username TEXT DEFAULT '',
			summary TEXT DEFAULT '',
			changes TEXT DEFAULT '{}',
			ip_address TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			status TEXT DEFAULT 'draft',
			priority TEXT DEFAULT 'normal',
			affected_ipns TEXT DEFAULT '',
			created_by TEXT DEFAULT 'engineer',
			linked_ncr_id TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_at DATETIME,
			approved_by TEXT DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE rmas (
			id TEXT PRIMARY KEY,
			serial_number TEXT NOT NULL,
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create rmas table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL DEFAULT 1,
			status TEXT DEFAULT 'draft',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			ipn TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create app_settings table: %v", err)
	}

	return testDB
}

// newTestHandler creates a parts.Handler suitable for testing with no-op callbacks.
func newTestHandler(db *sql.DB, partsDir string) *parts.Handler {
	h := &parts.Handler{
		DB:       db,
		Hub:      nil,
		PartsDir: partsDir,
		NextID: func(prefix, table string, digits int) string {
			var count int
			db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
			return fmt.Sprintf("%s-%0*d", prefix, digits, count+1)
		},
		EnsureInitialRevision: func(ecoID, user, now string) {},
		SnapshotDocumentVersion: func(docID, changeSummary, createdBy string, ecoID *string) error {
			return nil
		},
		HandleGetDoc:           func(w http.ResponseWriter, r *http.Request, id string) {},
		LogSensitiveDataAccess: func(r *http.Request, dataType, recordID, details string) {},
	}
	h.LoadPartsFromDir = h.LoadPartsFromDirImpl
	h.GetPartByIPN = h.GetPartByIPNImpl
	return h
}

// setupPartsTestEnv creates a temporary directory structure with gitplm CSV files
func setupPartsTestEnv(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create resistors category directory
	resistorsDir := filepath.Join(tmpDir, "resistors")
	if err := os.MkdirAll(resistorsDir, 0755); err != nil {
		t.Fatalf("Failed to create resistors dir: %v", err)
	}

	resistorsCSV := `IPN,description,manufacturer,mpn,value,tolerance,package,datasheet,notes,status
R-0402-10K,10K resistor 0402,Yageo,RC0402FR-0710KL,10K,1%,0402,http://example.com/ds-10k.pdf,Standard part,active
R-0805-1K,1K resistor 0805,Panasonic,ERJ-6ENF1001V,1K,1%,0805,http://example.com/ds-1k.pdf,Preferred,active
R-0603-100R,100R resistor 0603,Vishay,CRCW0603100RFKEA,100,1%,0603,,Low stock,active`

	if err := os.WriteFile(filepath.Join(resistorsDir, "standard.csv"), []byte(resistorsCSV), 0644); err != nil {
		t.Fatalf("Failed to write resistors CSV: %v", err)
	}

	// Create capacitors category directory
	capacitorsDir := filepath.Join(tmpDir, "capacitors")
	if err := os.MkdirAll(capacitorsDir, 0755); err != nil {
		t.Fatalf("Failed to create capacitors dir: %v", err)
	}

	capacitorsCSV := `IPN,description,manufacturer,mpn,capacitance,voltage,package,status
C-0805-10U,10uF capacitor 0805,Murata,GRM21BR61C106KE15L,10uF,16V,0805,active
C-0402-100N,100nF capacitor 0402,Samsung,CL05B104KO5NNNC,100nF,16V,0402,active`

	if err := os.WriteFile(filepath.Join(capacitorsDir, "ceramic.csv"), []byte(capacitorsCSV), 0644); err != nil {
		t.Fatalf("Failed to write capacitors CSV: %v", err)
	}

	// Create standalone CSV file (top-level category)
	icsCSV := `part_number,description,manufacturer,mpn,status
IC-STM32F4,STM32F4 microcontroller,ST,STM32F405RGT6,active
IC-LDO-3V3,3.3V LDO regulator,TI,TLV1117-33,active`

	if err := os.WriteFile(filepath.Join(tmpDir, "ics.csv"), []byte(icsCSV), 0644); err != nil {
		t.Fatalf("Failed to write ICs CSV: %v", err)
	}

	return tmpDir
}

func TestHandleListParts(t *testing.T) {
	partsDir := setupPartsTestEnv(t)
	h := newTestHandler(nil, partsDir)

	tests := []struct {
		name          string
		query         string
		category      string
		page          string
		limit         string
		expectedCount int
		expectedTotal int
		checkIPN      string
	}{
		{
			name:          "List all parts",
			expectedCount: 7,
			expectedTotal: 7,
		},
		{
			name:          "Filter by category - resistors",
			category:      "resistors",
			expectedCount: 3,
			expectedTotal: 3,
			checkIPN:      "R-0402-10K",
		},
		{
			name:          "Filter by category - capacitors",
			category:      "capacitors",
			expectedCount: 2,
			expectedTotal: 2,
			checkIPN:      "C-0805-10U",
		},
		{
			name:          "Filter by category - ics",
			category:      "ics",
			expectedCount: 2,
			expectedTotal: 2,
			checkIPN:      "IC-STM32F4",
		},
		{
			name:          "Search by IPN",
			query:         "R-0402",
			expectedCount: 1,
			expectedTotal: 1,
			checkIPN:      "R-0402-10K",
		},
		{
			name:          "Search by description",
			query:         "microcontroller",
			expectedCount: 1,
			expectedTotal: 1,
			checkIPN:      "IC-STM32F4",
		},
		{
			name:          "Search by manufacturer",
			query:         "yageo",
			expectedCount: 1,
			expectedTotal: 1,
			checkIPN:      "R-0402-10K",
		},
		{
			name:          "Search with no results",
			query:         "nonexistent",
			expectedCount: 0,
			expectedTotal: 0,
		},
		{
			name:          "Pagination - page 1, limit 3",
			page:          "1",
			limit:         "3",
			expectedCount: 3,
			expectedTotal: 7,
		},
		{
			name:          "Pagination - page 2, limit 3",
			page:          "2",
			limit:         "3",
			expectedCount: 3,
			expectedTotal: 7,
		},
		{
			name:          "Pagination - page 3, limit 3",
			page:          "3",
			limit:         "3",
			expectedCount: 1,
			expectedTotal: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts", nil)
			q := req.URL.Query()
			if tt.query != "" {
				q.Set("q", tt.query)
			}
			if tt.category != "" {
				q.Set("category", tt.category)
			}
			if tt.page != "" {
				q.Set("page", tt.page)
			}
			if tt.limit != "" {
				q.Set("limit", tt.limit)
			}
			req.URL.RawQuery = q.Encode()

			rr := httptest.NewRecorder()
			h.ListParts(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
				return
			}

			var resp struct {
				Data []models.Part `json:"data"`
				Meta struct {
					Total int `json:"total"`
					Page  int `json:"page"`
					Limit int `json:"limit"`
				} `json:"meta"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(resp.Data) != tt.expectedCount {
				t.Errorf("Expected %d parts, got %d", tt.expectedCount, len(resp.Data))
			}

			if resp.Meta.Total != tt.expectedTotal {
				t.Errorf("Expected total %d, got %d", tt.expectedTotal, resp.Meta.Total)
			}

			if tt.checkIPN != "" {
				found := false
				for _, p := range resp.Data {
					if p.IPN == tt.checkIPN {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find IPN %s in results", tt.checkIPN)
				}
			}
		})
	}
}

func TestHandleListParts_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()

	// Create duplicate entries in different files
	catDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(catDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	csv1 := `IPN,description
DUP-001,First occurrence
UNIQUE-001,Unique part`

	csv2 := `IPN,description
DUP-001,Duplicate occurrence
UNIQUE-002,Another unique`

	os.WriteFile(filepath.Join(catDir, "file1.csv"), []byte(csv1), 0644)
	os.WriteFile(filepath.Join(catDir, "file2.csv"), []byte(csv2), 0644)

	h := newTestHandler(nil, tmpDir)

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	rr := httptest.NewRecorder()
	h.ListParts(rr, req)

	var resp struct {
		Data []models.Part `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)

	if len(resp.Data) != 3 {
		t.Errorf("Expected 3 deduplicated parts, got %d", len(resp.Data))
	}

	dupCount := 0
	for _, p := range resp.Data {
		if p.IPN == "DUP-001" {
			dupCount++
		}
	}
	if dupCount != 1 {
		t.Errorf("Expected DUP-001 to appear once, appeared %d times", dupCount)
	}
}

func TestHandleGetPart(t *testing.T) {
	partsDir := setupPartsTestEnv(t)
	h := newTestHandler(nil, partsDir)

	tests := []struct {
		name           string
		ipn            string
		expectedStatus int
		checkField     string
		checkValue     string
	}{
		{
			name:           "Get existing resistor",
			ipn:            "R-0402-10K",
			expectedStatus: http.StatusOK,
			checkField:     "description",
			checkValue:     "10K resistor 0402",
		},
		{
			name:           "Get existing capacitor",
			ipn:            "C-0805-10U",
			expectedStatus: http.StatusOK,
			checkField:     "manufacturer",
			checkValue:     "Murata",
		},
		{
			name:           "Get existing IC",
			ipn:            "IC-STM32F4",
			expectedStatus: http.StatusOK,
			checkField:     "mpn",
			checkValue:     "STM32F405RGT6",
		},
		{
			name:           "Get non-existent part",
			ipn:            "NONEXISTENT-001",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts/"+tt.ipn, nil)
			rr := httptest.NewRecorder()
			h.GetPart(rr, req, tt.ipn)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
				return
			}

			if tt.expectedStatus == http.StatusOK {
				var resp struct {
					Data models.Part `json:"data"`
				}
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if resp.Data.IPN != tt.ipn {
					t.Errorf("Expected IPN %s, got %s", tt.ipn, resp.Data.IPN)
				}

				if tt.checkField != "" {
					if val, ok := resp.Data.Fields[tt.checkField]; !ok || val != tt.checkValue {
						t.Errorf("Expected field %s=%s, got %s", tt.checkField, tt.checkValue, val)
					}
				}
			}
		})
	}
}

func TestHandleCreatePart(t *testing.T) {
	tmpDir := t.TempDir()

	// Create resistors category CSV (must be <category>.csv in partsDir root)
	os.WriteFile(filepath.Join(tmpDir, "resistors.csv"), []byte("IPN,description\n"), 0644)

	h := newTestHandler(nil, tmpDir)

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		checkCreated   bool
	}{
		{
			name: "Create valid part",
			body: map[string]interface{}{
				"ipn":      "R-TEST-001",
				"category": "resistors",
				"fields": map[string]string{
					"description":  "Test resistor",
					"manufacturer": "TestCorp",
					"mpn":          "TEST-001",
					"status":       "active",
				},
			},
			expectedStatus: http.StatusOK,
			checkCreated:   true,
		},
		{
			name: "Create part without IPN",
			body: map[string]interface{}{
				"category": "resistors",
				"fields": map[string]string{
					"description": "No IPN",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Create part without category",
			body: map[string]interface{}{
				"ipn": "R-TEST-002",
				"fields": map[string]string{
					"description": "No category",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Create duplicate part",
			body: map[string]interface{}{
				"ipn":      "R-TEST-001",
				"category": "resistors",
				"fields": map[string]string{
					"description": "Duplicate",
				},
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.CreatePart(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
				return
			}

			if tt.checkCreated {
				ipn := tt.body["ipn"].(string)
				req2 := httptest.NewRequest("GET", "/api/v1/parts/"+ipn, nil)
				rr2 := httptest.NewRecorder()
				h.GetPart(rr2, req2, ipn)

				if rr2.Code != http.StatusOK {
					t.Errorf("Created part not found")
				}
			}
		})
	}
}

func TestHandleCheckIPN(t *testing.T) {
	partsDir := setupPartsTestEnv(t)
	h := newTestHandler(nil, partsDir)

	tests := []struct {
		name           string
		ipn            string
		expectedExists bool
	}{
		{
			name:           "Check existing IPN",
			ipn:            "R-0402-10K",
			expectedExists: true,
		},
		{
			name:           "Check non-existent IPN",
			ipn:            "NONEXISTENT-001",
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts/check-ipn?ipn="+tt.ipn, nil)
			rr := httptest.NewRecorder()
			h.CheckIPN(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
				return
			}

			var resp struct {
				Data struct {
					Exists bool `json:"exists"`
				} `json:"data"`
			}
			json.NewDecoder(rr.Body).Decode(&resp)

			if resp.Data.Exists != tt.expectedExists {
				t.Errorf("Expected exists=%v, got %v", tt.expectedExists, resp.Data.Exists)
			}
		})
	}
}

func TestHandleListCategories(t *testing.T) {
	partsDir := setupPartsTestEnv(t)
	h := newTestHandler(nil, partsDir)

	req := httptest.NewRequest("GET", "/api/v1/parts/categories", nil)
	rr := httptest.NewRecorder()
	h.ListCategories(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
		return
	}

	var resp struct {
		Data []models.Category `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)

	if len(resp.Data) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(resp.Data))
	}

	categories := make(map[string]bool)
	for _, cat := range resp.Data {
		categories[cat.Name] = true
	}

	expected := []string{"resistors", "capacitors", "ics"}
	for _, exp := range expected {
		if !categories[exp] {
			t.Errorf("Expected category %s not found", exp)
		}
	}
}

func TestHandleCreateCategory(t *testing.T) {
	tmpDir := t.TempDir()
	h := newTestHandler(nil, tmpDir)

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Create valid category",
			body: map[string]interface{}{
				"prefix": "conn",
				"title":  "Connectors",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Create category without name",
			body: map[string]interface{}{
				"schema": []string{"IPN", "description"},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/api/v1/parts/categories", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.CreateCategory(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				prefix := tt.body["prefix"].(string)
				csvFile := fmt.Sprintf("z-%s.csv", strings.ToLower(prefix))
				csvPath := filepath.Join(tmpDir, csvFile)
				if _, err := os.Stat(csvPath); os.IsNotExist(err) {
					t.Errorf("Category CSV file not created: %s", csvPath)
				}
			}
		})
	}
}

func TestHandlePartBOM(t *testing.T) {
	tmpDir := t.TempDir()

	// Create assembly parts list
	asmCSV := `IPN,description
PCA-001,Test Assembly`
	os.WriteFile(filepath.Join(tmpDir, "assemblies.csv"), []byte(asmCSV), 0644)

	// Create BOM file for PCA-001
	bomCSV := `IPN,qty,description
R-001,2,Resistor 1K
C-001,1,Capacitor 10uF`
	os.WriteFile(filepath.Join(tmpDir, "PCA-001.csv"), []byte(bomCSV), 0644)

	// Create component parts
	os.WriteFile(filepath.Join(tmpDir, "resistors.csv"), []byte("IPN,description\nR-001,Resistor 1K\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "capacitors.csv"), []byte("IPN,description\nC-001,Capacitor 10uF\n"), 0644)

	h := newTestHandler(nil, tmpDir)

	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-001/bom", nil)
	rr := httptest.NewRecorder()
	h.PartBOM(rr, req, "PCA-001")

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		return
	}

	var resp struct {
		Data parts.BOMNode `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)

	if len(resp.Data.Children) != 2 {
		t.Errorf("Expected 2 BOM children, got %d", len(resp.Data.Children))
		return
	}

	for _, child := range resp.Data.Children {
		if child.IPN == "R-001" && child.Qty != 2 {
			t.Errorf("Expected R-001 qty=2, got %f", child.Qty)
		}
		if child.IPN == "C-001" && child.Qty != 1 {
			t.Errorf("Expected C-001 qty=1, got %f", child.Qty)
		}
	}
}

func TestHandlePartBOM_NonAssembly(t *testing.T) {
	partsDir := setupPartsTestEnv(t)
	h := newTestHandler(nil, partsDir)

	req := httptest.NewRequest("GET", "/api/v1/parts/R-0402-10K/bom", nil)
	rr := httptest.NewRecorder()
	h.PartBOM(rr, req, "R-0402-10K")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for non-assembly part, got %d", rr.Code)
	}
}

func TestHandleUpdatePart(t *testing.T) {
	tmpDir := t.TempDir()
	catDir := filepath.Join(tmpDir, "resistors")
	os.MkdirAll(catDir, 0755)
	csvContent := "IPN,description,status\nR-001,Original description,active\n"
	os.WriteFile(filepath.Join(catDir, "test.csv"), []byte(csvContent), 0644)

	h := newTestHandler(nil, tmpDir)

	updateBody := map[string]interface{}{
		"fields": map[string]string{
			"description": "Updated description",
			"status":      "obsolete",
			"new_field":   "new value",
		},
	}

	bodyBytes, _ := json.Marshal(updateBody)
	req := httptest.NewRequest("PUT", "/api/v1/parts/R-001", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdatePart(rr, req, "R-001")

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeletePart(t *testing.T) {
	tmpDir := t.TempDir()
	catDir := filepath.Join(tmpDir, "resistors")
	os.MkdirAll(catDir, 0755)
	csvContent := "IPN,description\nR-001,Part to delete\nR-002,Part to keep\n"
	os.WriteFile(filepath.Join(catDir, "test.csv"), []byte(csvContent), 0644)

	h := newTestHandler(nil, tmpDir)

	req := httptest.NewRequest("DELETE", "/api/v1/parts/R-001", nil)
	rr := httptest.NewRecorder()
	h.DeletePart(rr, req, "R-001")

	if rr.Code != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", rr.Code)
	}
}

func TestHandleAddColumn(t *testing.T) {
	t.Skip("Column addition not yet implemented - stub handler only")
	tmpDir := t.TempDir()
	catDir := filepath.Join(tmpDir, "resistors")
	os.MkdirAll(catDir, 0755)
	csvContent := "IPN,description\nR-001,Test part\n"
	os.WriteFile(filepath.Join(catDir, "test.csv"), []byte(csvContent), 0644)

	h := newTestHandler(nil, tmpDir)

	addColBody := map[string]interface{}{
		"name": "tolerance",
	}

	bodyBytes, _ := json.Marshal(addColBody)
	req := httptest.NewRequest("POST", "/api/v1/parts/categories/resistors/columns", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.AddColumn(rr, req, "resistors")

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
		return
	}

	req2 := httptest.NewRequest("GET", "/api/v1/parts/categories", nil)
	rr2 := httptest.NewRecorder()
	h.ListCategories(rr2, req2)

	var resp struct {
		Data []models.Category `json:"data"`
	}
	json.NewDecoder(rr2.Body).Decode(&resp)

	found := false
	for _, cat := range resp.Data {
		if cat.Name == "resistors" {
			for _, col := range cat.Columns {
				if col == "tolerance" {
					found = true
					break
				}
			}
		}
	}

	if !found {
		t.Errorf("Column 'tolerance' not found in category schema")
	}
}

func TestHandleDeleteColumn(t *testing.T) {
	t.Skip("Column deletion not yet implemented - stub handler only")
	tmpDir := t.TempDir()
	catDir := filepath.Join(tmpDir, "resistors")
	os.MkdirAll(catDir, 0755)
	csvContent := "IPN,description,tolerance\nR-001,Test part,1%\n"
	os.WriteFile(filepath.Join(catDir, "test.csv"), []byte(csvContent), 0644)

	h := newTestHandler(nil, tmpDir)

	req := httptest.NewRequest("DELETE", "/api/v1/parts/categories/resistors/columns/tolerance", nil)
	rr := httptest.NewRecorder()
	h.DeleteColumn(rr, req, "resistors", "tolerance")

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
		return
	}

	req2 := httptest.NewRequest("GET", "/api/v1/parts/categories", nil)
	rr2 := httptest.NewRecorder()
	h.ListCategories(rr2, req2)

	var resp struct {
		Data []models.Category `json:"data"`
	}
	json.NewDecoder(rr2.Body).Decode(&resp)

	for _, cat := range resp.Data {
		if cat.Name == "resistors" {
			for _, col := range cat.Columns {
				if col == "tolerance" {
					t.Errorf("Column 'tolerance' should have been deleted")
				}
			}
		}
	}
}

func TestLoadPartsFromDir_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	h := newTestHandler(nil, tmpDir)

	cats, schemas, titles, err := h.LoadPartsFromDir()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(cats) != 0 {
		t.Errorf("Expected 0 categories, got %d", len(cats))
	}
	if len(schemas) != 0 {
		t.Errorf("Expected 0 schemas, got %d", len(schemas))
	}
	if len(titles) != 0 {
		t.Errorf("Expected 0 titles, got %d", len(titles))
	}
}

func TestLoadPartsFromDir_NilPartsDir(t *testing.T) {
	h := newTestHandler(nil, "")

	cats, schemas, titles, err := h.LoadPartsFromDir()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(cats) != 0 || len(schemas) != 0 || len(titles) != 0 {
		t.Errorf("Expected empty results when partsDir is empty")
	}
}

func TestReadCSV_InvalidFile(t *testing.T) {
	_, _, _, err := parts.ReadCSV("/nonexistent/file.csv", "test")

	if err == nil {
		t.Errorf("Expected error for non-existent file")
	}
}

func TestReadCSV_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.csv")
	os.WriteFile(emptyFile, []byte(""), 0644)

	_, _, _, err := parts.ReadCSV(emptyFile, "test")

	if err == nil {
		t.Errorf("Expected error for empty CSV file")
	}
}

func TestHandleDashboard(t *testing.T) {
	partsDir := setupPartsTestEnv(t)
	testDB := setupPartsTestDB(t)
	defer testDB.Close()

	h := newTestHandler(testDB, partsDir)

	// Insert some low stock items
	var err error
	_, err = testDB.Exec("INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES (?, ?, ?)", "R-0402-10K", 5, 100)
	if err != nil {
		t.Fatalf("Failed to insert low stock item: %v", err)
	}
	_, err = testDB.Exec("INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES (?, ?, ?)", "C-0805-10U", 150, 100)
	if err != nil {
		t.Fatalf("Failed to insert normal stock item: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/dashboard", nil)
	rr := httptest.NewRecorder()
	h.Dashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
		return
	}

	var resp struct {
		Data models.DashboardData `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)

	if resp.Data.TotalParts != 7 {
		t.Errorf("Expected 7 total parts, got %d", resp.Data.TotalParts)
	}

	if resp.Data.LowStock != 1 {
		t.Errorf("Expected 1 low stock item, got %d", resp.Data.LowStock)
	}
}

// Keep csv import used by setupCSVPartsDir in handler_parts_create_test.go
var _ = csv.NewWriter
