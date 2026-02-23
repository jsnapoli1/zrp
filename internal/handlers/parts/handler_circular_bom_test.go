package parts_test

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zrp/internal/handlers/parts"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

// setupCircularBOMTestDB creates an in-memory database for circular BOM tests
func setupCircularBOMTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			supplier TEXT NOT NULL,
			status TEXT DEFAULT 'draft',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create purchase_orders table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			line_num INTEGER NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty REAL NOT NULL,
			unit_price REAL NOT NULL,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create po_lines table: %v", err)
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

	return testDB
}

// createPartCSV creates a component part CSV file
func createPartCSV(t *testing.T, dir string, ipn string, description string) {
	t.Helper()

	csvPath := filepath.Join(dir, "z-components.csv")

	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		f, err := os.Create(csvPath)
		if err != nil {
			t.Fatalf("Failed to create components CSV: %v", err)
		}
		w := csv.NewWriter(f)
		w.Write([]string{"IPN", "description", "manufacturer"})
		w.Write([]string{ipn, description, "TestMfg"})
		w.Flush()
		f.Close()
	} else {
		f, err := os.OpenFile(csvPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatalf("Failed to open components CSV: %v", err)
		}
		w := csv.NewWriter(f)
		w.Write([]string{ipn, description, "TestMfg"})
		w.Flush()
		f.Close()
	}
}

// createBOMCSV creates a BOM CSV file for an assembly
func createBOMCSV(t *testing.T, dir string, assemblyIPN string, bomLines [][]string) {
	t.Helper()

	f, err := os.Create(filepath.Join(dir, assemblyIPN+".csv"))
	if err != nil {
		t.Fatalf("Failed to create BOM file for %s: %v", assemblyIPN, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.WriteAll(bomLines)
}

// insertComponentCost inserts a purchase order line for component pricing
func insertComponentCost(t *testing.T, db *sql.DB, ipn string, unitPrice float64) {
	t.Helper()

	poID := "PO-" + ipn
	_, err := db.Exec("INSERT OR IGNORE INTO purchase_orders (id, supplier) VALUES (?, 'Test Supplier')", poID)
	if err != nil {
		t.Fatalf("Failed to insert PO %s: %v", poID, err)
	}

	_, err = db.Exec("INSERT INTO po_lines (po_id, line_num, ipn, qty, unit_price) VALUES (?, 1, ?, 1, ?)",
		poID, ipn, unitPrice)
	if err != nil {
		t.Fatalf("Failed to insert PO line for %s: %v", ipn, err)
	}
}

// TestCircularBOM_DirectSelfReference tests a direct circular reference (Part A contains Part A)
func TestCircularBOM_DirectSelfReference(t *testing.T) {
	testDB := setupCircularBOMTestDB(t)
	defer testDB.Close()

	dir := t.TempDir()

	h := newTestHandler(testDB, dir)

	// Create assembly that references itself
	createPartCSV(t, dir, "PCA-SELF", "Self-referencing assembly")
	createPartCSV(t, dir, "RES-001", "100R Resistor")

	insertComponentCost(t, testDB, "RES-001", 0.10)

	createBOMCSV(t, dir, "PCA-SELF", [][]string{
		{"IPN", "qty", "ref", "description"},
		{"RES-001", "1", "R1", "100R Resistor"},
		{"PCA-SELF", "1", "A1", "Self-reference"},
	})

	// Test 1: BOM retrieval should not crash
	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-SELF/bom", nil)
	w := httptest.NewRecorder()
	h.PartBOM(w, req, "PCA-SELF")

	if w.Code != 200 {
		t.Fatalf("BOM endpoint should not crash with circular reference, got %d: %s", w.Code, w.Body.String())
	}

	var bomResp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &bomResp); err != nil {
		t.Fatalf("Failed to parse BOM response: %v", err)
	}

	t.Logf("Direct circular BOM response: %v", bomResp.Data)

	// Test 2: Cost calculation should not hang
	done := make(chan bool)
	go func() {
		req2 := httptest.NewRequest("GET", "/api/v1/parts/PCA-SELF/cost", nil)
		w2 := httptest.NewRecorder()
		h.PartCost(w2, req2, "PCA-SELF")

		if w2.Code != 200 {
			t.Errorf("Cost endpoint should not crash with circular reference, got %d: %s", w2.Code, w2.Body.String())
		}

		var costResp struct {
			Data map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(w2.Body.Bytes(), &costResp); err != nil {
			t.Errorf("Failed to parse cost response: %v", err)
		}

		if bomCost, ok := costResp.Data["bom_cost"].(float64); ok {
			if bomCost < 0 || bomCost > 10000 {
				t.Errorf("BOM cost should be finite for circular reference, got %.2f", bomCost)
			}
			t.Logf("Direct circular BOM cost: %.2f", bomCost)
		}

		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Cost calculation with direct circular reference timed out (hung/infinite loop)")
	}
}

// TestCircularBOM_IndirectReference tests an indirect circular reference (Part A -> Part B -> Part A)
func TestCircularBOM_IndirectReference(t *testing.T) {
	testDB := setupCircularBOMTestDB(t)
	defer testDB.Close()

	dir := t.TempDir()

	h := newTestHandler(testDB, dir)

	createPartCSV(t, dir, "PCA-A", "Assembly A")
	createPartCSV(t, dir, "PCA-B", "Assembly B")
	createPartCSV(t, dir, "RES-001", "100R Resistor")
	createPartCSV(t, dir, "CAP-001", "10uF Capacitor")

	insertComponentCost(t, testDB, "RES-001", 0.10)
	insertComponentCost(t, testDB, "CAP-001", 0.25)

	createBOMCSV(t, dir, "PCA-A", [][]string{
		{"IPN", "qty", "ref"},
		{"RES-001", "1", "R1"},
		{"PCA-B", "1", "A1"},
	})

	createBOMCSV(t, dir, "PCA-B", [][]string{
		{"IPN", "qty", "ref"},
		{"CAP-001", "1", "C1"},
		{"PCA-A", "1", "A1"},
	})

	// Test 1: BOM retrieval should be depth-limited
	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-A/bom", nil)
	w := httptest.NewRecorder()
	h.PartBOM(w, req, "PCA-A")

	if w.Code != 200 {
		t.Fatalf("BOM endpoint should not crash with indirect circular reference, got %d: %s", w.Code, w.Body.String())
	}

	var bomResp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &bomResp); err != nil {
		t.Fatalf("Failed to parse BOM response: %v", err)
	}

	t.Logf("Indirect circular BOM response (should be depth-limited): %+v", bomResp.Data)

	// Test 2: Cost calculation should not hang
	done := make(chan bool)
	go func() {
		req2 := httptest.NewRequest("GET", "/api/v1/parts/PCA-A/cost", nil)
		w2 := httptest.NewRecorder()
		h.PartCost(w2, req2, "PCA-A")

		if w2.Code != 200 {
			t.Errorf("Cost endpoint should handle indirect circular reference, got %d: %s", w2.Code, w2.Body.String())
		}

		var costResp struct {
			Data map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(w2.Body.Bytes(), &costResp); err != nil {
			t.Errorf("Failed to parse cost response: %v", err)
		}

		if bomCost, ok := costResp.Data["bom_cost"].(float64); ok {
			if bomCost < 0 || bomCost > 10000 {
				t.Errorf("BOM cost should be finite for indirect circular reference, got %.2f", bomCost)
			}
			t.Logf("Indirect circular BOM cost: %.2f", bomCost)
		}

		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Cost calculation with indirect circular reference timed out (hung/infinite loop)")
	}
}

// TestCircularBOM_DeepNesting tests a deeply nested BOM (10+ levels)
func TestCircularBOM_DeepNesting(t *testing.T) {
	testDB := setupCircularBOMTestDB(t)
	defer testDB.Close()

	dir := t.TempDir()

	h := newTestHandler(testDB, dir)

	depth := 15
	for i := 0; i < depth; i++ {
		ipn := fmt.Sprintf("PCA-LEVEL-%02d", i)
		createPartCSV(t, dir, ipn, fmt.Sprintf("Assembly Level %d", i))

		if i == depth-1 {
			createPartCSV(t, dir, "RES-BOTTOM", "Bottom resistor")
			insertComponentCost(t, testDB, "RES-BOTTOM", 0.10)
			createBOMCSV(t, dir, ipn, [][]string{
				{"IPN", "qty"},
				{"RES-BOTTOM", "1"},
			})
		} else {
			nextIPN := fmt.Sprintf("PCA-LEVEL-%02d", i+1)
			createBOMCSV(t, dir, ipn, [][]string{
				{"IPN", "qty"},
				{nextIPN, "1"},
			})
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-LEVEL-00/bom", nil)
	w := httptest.NewRecorder()
	h.PartBOM(w, req, "PCA-LEVEL-00")

	if w.Code != 200 {
		t.Fatalf("BOM endpoint should handle deep nesting, got %d: %s", w.Code, w.Body.String())
	}

	var bomResp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &bomResp); err != nil {
		t.Fatalf("Failed to parse BOM response: %v", err)
	}

	bomJSON, _ := json.MarshalIndent(bomResp.Data, "", "  ")
	t.Logf("Deep nested BOM (15 levels requested): %s", string(bomJSON))

	if !strings.Contains(string(bomJSON), "max depth reached") {
		t.Logf("Warning: Deep BOM may not be properly depth-limited")
	}

	// Test cost calculation with timeout
	done := make(chan bool)
	go func() {
		req2 := httptest.NewRequest("GET", "/api/v1/parts/PCA-LEVEL-00/cost", nil)
		w2 := httptest.NewRecorder()
		h.PartCost(w2, req2, "PCA-LEVEL-00")

		if w2.Code != 200 {
			t.Errorf("Cost endpoint should handle deep nesting, got %d: %s", w2.Code, w2.Body.String())
		}

		var costResp struct {
			Data map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(w2.Body.Bytes(), &costResp); err != nil {
			t.Errorf("Failed to parse cost response: %v", err)
		}

		if bomCost, ok := costResp.Data["bom_cost"].(float64); ok {
			t.Logf("Deep nested BOM cost: %.2f", bomCost)
		}

		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Cost calculation with deep nesting timed out")
	}
}

// TestCircularBOM_ComplexCircular tests a complex circular scenario with multiple paths
func TestCircularBOM_ComplexCircular(t *testing.T) {
	testDB := setupCircularBOMTestDB(t)
	defer testDB.Close()

	dir := t.TempDir()

	h := newTestHandler(testDB, dir)

	createPartCSV(t, dir, "ASY-ROOT", "Root Assembly")
	createPartCSV(t, dir, "PCA-A", "Sub-Assembly A")
	createPartCSV(t, dir, "PCA-B", "Sub-Assembly B")
	createPartCSV(t, dir, "PCA-C", "Sub-Assembly C")
	createPartCSV(t, dir, "RES-001", "Resistor")

	insertComponentCost(t, testDB, "RES-001", 0.10)

	createBOMCSV(t, dir, "ASY-ROOT", [][]string{
		{"IPN", "qty"},
		{"PCA-A", "1"},
		{"PCA-B", "1"},
		{"RES-001", "1"},
	})

	createBOMCSV(t, dir, "PCA-A", [][]string{
		{"IPN", "qty"},
		{"PCA-C", "1"},
	})

	createBOMCSV(t, dir, "PCA-B", [][]string{
		{"IPN", "qty"},
		{"PCA-C", "2"},
	})

	createBOMCSV(t, dir, "PCA-C", [][]string{
		{"IPN", "qty"},
		{"ASY-ROOT", "1"},
	})

	req := httptest.NewRequest("GET", "/api/v1/parts/ASY-ROOT/bom", nil)
	w := httptest.NewRecorder()
	h.PartBOM(w, req, "ASY-ROOT")

	if w.Code != 200 {
		t.Fatalf("BOM endpoint should handle complex circular reference, got %d: %s", w.Code, w.Body.String())
	}

	var bomResp models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &bomResp); err != nil {
		t.Fatalf("Failed to parse BOM response: %v", err)
	}

	bomJSON, _ := json.MarshalIndent(bomResp.Data, "", "  ")
	t.Logf("Complex circular BOM: %s", string(bomJSON))

	done := make(chan bool)
	go func() {
		req2 := httptest.NewRequest("GET", "/api/v1/parts/ASY-ROOT/cost", nil)
		w2 := httptest.NewRecorder()
		h.PartCost(w2, req2, "ASY-ROOT")

		if w2.Code != 200 {
			t.Errorf("Cost endpoint should handle complex circular reference, got %d: %s", w2.Code, w2.Body.String())
		}

		var costResp struct {
			Data map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(w2.Body.Bytes(), &costResp); err != nil {
			t.Errorf("Failed to parse cost response: %v", err)
		}

		if bomCost, ok := costResp.Data["bom_cost"].(float64); ok {
			if bomCost < 0 || bomCost > 10000 {
				t.Errorf("BOM cost should be finite for complex circular reference, got %.2f", bomCost)
			}
			t.Logf("Complex circular BOM cost: %.2f", bomCost)
		}

		done <- true
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Cost calculation with complex circular reference timed out")
	}
}

// TestCircularBOM_GracefulTermination verifies all BOM operations terminate gracefully
func TestCircularBOM_GracefulTermination(t *testing.T) {
	testDB := setupCircularBOMTestDB(t)
	defer testDB.Close()

	dir := t.TempDir()

	h := newTestHandler(testDB, dir)

	createPartCSV(t, dir, "PCA-ALPHA", "Assembly Alpha")
	createPartCSV(t, dir, "PCA-BETA", "Assembly Beta")

	createBOMCSV(t, dir, "PCA-ALPHA", [][]string{
		{"IPN", "qty"},
		{"PCA-BETA", "10"},
	})

	createBOMCSV(t, dir, "PCA-BETA", [][]string{
		{"IPN", "qty"},
		{"PCA-ALPHA", "10"},
	})

	testCases := []struct {
		name     string
		endpoint string
		ipn      string
	}{
		{"BOM retrieval ALPHA", "/bom", "PCA-ALPHA"},
		{"BOM retrieval BETA", "/bom", "PCA-BETA"},
		{"Cost calculation ALPHA", "/cost", "PCA-ALPHA"},
		{"Cost calculation BETA", "/cost", "PCA-BETA"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan bool)
			go func() {
				w := httptest.NewRecorder()

				if tc.endpoint == "/bom" {
					req := httptest.NewRequest("GET", "/api/v1/parts/"+tc.ipn+"/bom", nil)
					h.PartBOM(w, req, tc.ipn)
				} else {
					req := httptest.NewRequest("GET", "/api/v1/parts/"+tc.ipn+"/cost", nil)
					h.PartCost(w, req, tc.ipn)
				}

				if w.Code != 200 {
					t.Errorf("%s failed: got %d: %s", tc.name, w.Code, w.Body.String())
				}

				done <- true
			}()

			select {
			case <-done:
				t.Logf("%s completed successfully", tc.name)
			case <-time.After(5 * time.Second):
				t.Fatalf("%s timed out - did not terminate gracefully", tc.name)
			}
		})
	}
}

// TestCircularBOM_RejectOrLimit verifies circular BOMs are either rejected or depth-limited
func TestCircularBOM_RejectOrLimit(t *testing.T) {
	testDB := setupCircularBOMTestDB(t)
	defer testDB.Close()

	dir := t.TempDir()

	h := newTestHandler(testDB, dir)

	createPartCSV(t, dir, "PCA-CIRCLE", "Circular Assembly")
	createPartCSV(t, dir, "RES-001", "Resistor")

	insertComponentCost(t, testDB, "RES-001", 0.10)

	createBOMCSV(t, dir, "PCA-CIRCLE", [][]string{
		{"IPN", "qty"},
		{"RES-001", "1"},
		{"PCA-CIRCLE", "1"},
	})

	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-CIRCLE/bom", nil)
	w := httptest.NewRecorder()
	h.PartBOM(w, req, "PCA-CIRCLE")

	if w.Code == 400 || w.Code == 422 {
		t.Logf("Circular BOM explicitly rejected with status %d: %s", w.Code, w.Body.String())

		body := w.Body.String()
		if !strings.Contains(strings.ToLower(body), "circular") &&
			!strings.Contains(strings.ToLower(body), "cycle") &&
			!strings.Contains(strings.ToLower(body), "loop") {
			t.Logf("Warning: Error message doesn't mention circular/cycle/loop: %s", body)
		}
	} else if w.Code == 200 {
		var bomResp models.APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &bomResp); err != nil {
			t.Fatalf("Failed to parse BOM response: %v", err)
		}

		bomJSON, _ := json.MarshalIndent(bomResp.Data, "", "  ")

		if strings.Contains(string(bomJSON), "max depth") ||
			strings.Contains(string(bomJSON), "depth limit") ||
			strings.Contains(string(bomJSON), "maximum depth") {
			t.Logf("Circular BOM depth-limited (acceptable): %s", string(bomJSON))
		} else {
			t.Errorf("Circular BOM returned 200 but doesn't appear to be depth-limited: %s", string(bomJSON))
		}
	} else {
		t.Errorf("Unexpected status code for circular BOM: %d", w.Code)
	}
}

// Keep these imports used
var (
	_ = parts.BOMNode{}
)
