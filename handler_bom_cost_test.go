package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// setupBOMCostTestDB creates an in-memory database with purchase orders and po_lines for cost data
func setupBOMCostTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create purchase_orders table
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

	// Create po_lines table
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

	return testDB
}

// setupBOMTestParts creates test parts directory with components and assemblies
func setupBOMTestParts(t *testing.T, dir string) {
	t.Helper()

	// Create component parts CSV
	f, err := os.Create(filepath.Join(dir, "z-components.csv"))
	if err != nil {
		t.Fatalf("Failed to create components CSV: %v", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.WriteAll([][]string{
		{"IPN", "description", "manufacturer"},
		{"RES-001", "100Ω Resistor", "Vishay"},
		{"RES-002", "1kΩ Resistor", "Yageo"},
		{"CAP-001", "10µF Capacitor", "Murata"},
		{"IC-001", "Op Amp LM358", "TI"},
		{"IC-002", "MCU STM32F4", "ST"},
	})
}

// createBOMFile creates a BOM CSV file for an assembly
func createBOMFile(t *testing.T, dir string, assemblyIPN string, bomLines [][]string) {
	t.Helper()

	f, err := os.Create(filepath.Join(dir, assemblyIPN+".csv"))
	if err != nil {
		t.Fatalf("Failed to create BOM file for %s: %v", assemblyIPN, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.WriteAll(bomLines)
}

// insertPOCost inserts a purchase order line for component pricing
func insertPOCost(t *testing.T, db *sql.DB, poID, ipn string, unitPrice float64) {
	t.Helper()

	// Ensure PO exists
	_, err := db.Exec("INSERT OR IGNORE INTO purchase_orders (id, supplier) VALUES (?, 'Test Supplier')", poID)
	if err != nil {
		t.Fatalf("Failed to insert PO %s: %v", poID, err)
	}

	// Insert PO line
	_, err = db.Exec("INSERT INTO po_lines (po_id, line_num, ipn, qty, unit_price) VALUES (?, 1, ?, 1, ?)",
		poID, ipn, unitPrice)
	if err != nil {
		t.Fatalf("Failed to insert PO line for %s: %v", ipn, err)
	}
}

func TestBOMCost_SimpleBOM(t *testing.T) {
	// Test Case 1: Simple BOM with 3 components
	// Assembly PCA-SIMPLE contains: 2x RES-001 ($0.10 each), 1x CAP-001 ($0.50), 1x IC-001 ($2.00)
	// Expected total: 2*0.10 + 0.50 + 2.00 = $2.70

	oldDB := db
	testDB := setupBOMCostTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	dir := t.TempDir()
	setupBOMTestParts(t, dir)

	oldPartsDir := partsDir
	partsDir = dir
	defer func() { partsDir = oldPartsDir }()

	// Insert component costs
	insertPOCost(t, testDB, "PO-001", "RES-001", 0.10)
	insertPOCost(t, testDB, "PO-002", "CAP-001", 0.50)
	insertPOCost(t, testDB, "PO-003", "IC-001", 2.00)

	// Create simple BOM
	createBOMFile(t, dir, "PCA-SIMPLE", [][]string{
		{"IPN", "qty", "ref", "description"},
		{"RES-001", "2", "R1,R2", "100Ω Resistor"},
		{"CAP-001", "1", "C1", "10µF Capacitor"},
		{"IC-001", "1", "U1", "Op Amp"},
	})

	// Test cost endpoint
	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-SIMPLE/cost", nil)
	w := httptest.NewRecorder()
	handlePartCost(w, req, "PCA-SIMPLE")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	bomCost, ok := apiResp.Data["bom_cost"].(float64)
	if !ok {
		t.Fatalf("bom_cost not found in response: %v", apiResp.Data)
	}

	expected := 2.70
	tolerance := 0.001
	if bomCost < expected-tolerance || bomCost > expected+tolerance {
		t.Errorf("expected BOM cost %.2f, got %.2f", expected, bomCost)
	}
}

func TestBOMCost_NestedBOM(t *testing.T) {
	// Test Case 2: Nested BOM (2 levels deep)
	// PCA-SUB1: 1x RES-001 ($0.10), 1x CAP-001 ($0.50) = $0.60
	// PCA-SUB2: 2x RES-002 ($0.15 each), 1x IC-001 ($2.00) = $2.30
	// ASY-MAIN: 2x PCA-SUB1, 1x PCA-SUB2, 1x IC-002 ($5.00)
	// Expected: 2*0.60 + 2.30 + 5.00 = $8.50

	oldDB := db
	testDB := setupBOMCostTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	dir := t.TempDir()
	setupBOMTestParts(t, dir)

	oldPartsDir := partsDir
	partsDir = dir
	defer func() { partsDir = oldPartsDir }()

	// Insert component costs
	insertPOCost(t, testDB, "PO-001", "RES-001", 0.10)
	insertPOCost(t, testDB, "PO-002", "RES-002", 0.15)
	insertPOCost(t, testDB, "PO-003", "CAP-001", 0.50)
	insertPOCost(t, testDB, "PO-004", "IC-001", 2.00)
	insertPOCost(t, testDB, "PO-005", "IC-002", 5.00)

	// Create sub-assembly BOMs
	createBOMFile(t, dir, "PCA-SUB1", [][]string{
		{"IPN", "qty"},
		{"RES-001", "1"},
		{"CAP-001", "1"},
	})

	createBOMFile(t, dir, "PCA-SUB2", [][]string{
		{"IPN", "qty"},
		{"RES-002", "2"},
		{"IC-001", "1"},
	})

	// Create main assembly BOM
	createBOMFile(t, dir, "ASY-MAIN", [][]string{
		{"IPN", "qty", "ref"},
		{"PCA-SUB1", "2", "A1,A2"},
		{"PCA-SUB2", "1", "A3"},
		{"IC-002", "1", "U1"},
	})

	// Test cost endpoint
	req := httptest.NewRequest("GET", "/api/v1/parts/ASY-MAIN/cost", nil)
	w := httptest.NewRecorder()
	handlePartCost(w, req, "ASY-MAIN")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	bomCost, ok := apiResp.Data["bom_cost"].(float64)
	if !ok {
		t.Fatalf("bom_cost not found in response: %v", apiResp.Data)
	}

	expected := 8.50
	tolerance := 0.001
	if bomCost < expected-tolerance || bomCost > expected+tolerance {
		t.Errorf("expected BOM cost %.2f, got %.2f", expected, bomCost)
	}
}

func TestBOMCost_MissingCostData(t *testing.T) {
	// Test Case 3: Missing cost data for some components
	// PCA-PARTIAL: 1x RES-001 ($0.10), 1x CAP-001 (no cost), 1x IC-001 ($2.00)
	// Expected: Only components with known costs are counted: 0.10 + 2.00 = $2.10

	oldDB := db
	testDB := setupBOMCostTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	dir := t.TempDir()
	setupBOMTestParts(t, dir)

	oldPartsDir := partsDir
	partsDir = dir
	defer func() { partsDir = oldPartsDir }()

	// Insert costs for only some components
	insertPOCost(t, testDB, "PO-001", "RES-001", 0.10)
	insertPOCost(t, testDB, "PO-002", "IC-001", 2.00)
	// CAP-001 has no cost data

	// Create BOM
	createBOMFile(t, dir, "PCA-PARTIAL", [][]string{
		{"IPN", "qty"},
		{"RES-001", "1"},
		{"CAP-001", "1"},
		{"IC-001", "1"},
	})

	// Test cost endpoint
	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-PARTIAL/cost", nil)
	w := httptest.NewRecorder()
	handlePartCost(w, req, "PCA-PARTIAL")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	bomCost, ok := apiResp.Data["bom_cost"].(float64)
	if !ok {
		t.Fatalf("bom_cost not found in response: %v", apiResp.Data)
	}

	expected := 2.10
	tolerance := 0.001
	if bomCost < expected-tolerance || bomCost > expected+tolerance {
		t.Errorf("expected BOM cost %.2f (partial, ignoring missing costs), got %.2f", expected, bomCost)
	}
}

func TestBOMCost_CircularBOM(t *testing.T) {
	// Test Case 4: Circular BOM (assembly containing itself)
	// This should be prevented by max depth limit
	// PCA-CIRCULAR contains itself as a component - should not crash, limited by maxDepth

	oldDB := db
	testDB := setupBOMCostTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	dir := t.TempDir()
	setupBOMTestParts(t, dir)

	oldPartsDir := partsDir
	partsDir = dir
	defer func() { partsDir = oldPartsDir }()

	// Insert component cost
	insertPOCost(t, testDB, "PO-001", "RES-001", 0.10)

	// Create circular BOM (assembly includes itself)
	createBOMFile(t, dir, "PCA-CIRCULAR", [][]string{
		{"IPN", "qty"},
		{"RES-001", "1"},
		{"PCA-CIRCULAR", "1"}, // Self-reference
	})

	// Test cost endpoint - should not crash due to maxDepth limit
	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-CIRCULAR/cost", nil)
	w := httptest.NewRecorder()

	// Should complete without panic
	handlePartCost(w, req, "PCA-CIRCULAR")

	if w.Code != 200 {
		t.Fatalf("expected 200 even with circular BOM (depth-limited), got %d: %s", w.Code, w.Body.String())
	}

	var apiResp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	// Should have some cost (from RES-001) and not crash
	bomCost, ok := apiResp.Data["bom_cost"].(float64)
	if !ok {
		t.Fatalf("bom_cost not found in response: %v", apiResp.Data)
	}

	// The cost should be finite (not infinite due to circular reference)
	if bomCost < 0 || bomCost > 1000 {
		t.Errorf("unexpected BOM cost for circular BOM: %.2f (should be finite)", bomCost)
	}
}

func TestBOMCost_LargeBOM(t *testing.T) {
	// Test Case 5: Large BOM (100+ line items)
	// Create an assembly with 100 components, each costing $0.01
	// Expected: 100 * 0.01 = $1.00

	oldDB := db
	testDB := setupBOMCostTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	dir := t.TempDir()

	oldPartsDir := partsDir
	partsDir = dir
	defer func() { partsDir = oldPartsDir }()

	// Create components CSV with 100 parts
	f, err := os.Create(filepath.Join(dir, "z-bulk.csv"))
	if err != nil {
		t.Fatalf("Failed to create bulk components CSV: %v", err)
	}

	w := csv.NewWriter(f)
	headers := []string{"IPN", "description"}
	w.Write(headers)

	for i := 1; i <= 100; i++ {
		ipn := "COMP-" + pad(i, 3)
		w.Write([]string{ipn, "Component " + pad(i, 3)})
		// Insert cost for each component
		insertPOCost(t, testDB, "PO-BULK", ipn, 0.01)
	}
	w.Flush()
	f.Close()

	// Create large BOM
	bomLines := [][]string{{"IPN", "qty"}}
	for i := 1; i <= 100; i++ {
		ipn := "COMP-" + pad(i, 3)
		bomLines = append(bomLines, []string{ipn, "1"})
	}
	createBOMFile(t, dir, "PCA-LARGE", bomLines)

	// Test cost endpoint
	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-LARGE/cost", nil)
	w2 := httptest.NewRecorder()
	handlePartCost(w2, req, "PCA-LARGE")

	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var apiResp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	bomCost, ok := apiResp.Data["bom_cost"].(float64)
	if !ok {
		t.Fatalf("bom_cost not found in response: %v", apiResp.Data)
	}

	expected := 1.00
	tolerance := 0.001
	if bomCost < expected-tolerance || bomCost > expected+tolerance {
		t.Errorf("expected BOM cost %.2f for 100 components @ $0.01 each, got %.2f", expected, bomCost)
	}
}

func TestBOMCost_EmptyBOM(t *testing.T) {
	// Edge case: Assembly with no BOM file
	// Should return 0 cost

	oldDB := db
	testDB := setupBOMCostTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	dir := t.TempDir()
	setupBOMTestParts(t, dir)

	oldPartsDir := partsDir
	partsDir = dir
	defer func() { partsDir = oldPartsDir }()

	// Don't create BOM file for PCA-EMPTY

	// Test cost endpoint
	req := httptest.NewRequest("GET", "/api/v1/parts/PCA-EMPTY/cost", nil)
	w := httptest.NewRecorder()
	handlePartCost(w, req, "PCA-EMPTY")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	bomCost, ok := apiResp.Data["bom_cost"].(float64)
	if !ok {
		t.Fatalf("bom_cost not found in response: %v", apiResp.Data)
	}

	if bomCost != 0 {
		t.Errorf("expected BOM cost 0 for assembly with no BOM file, got %.2f", bomCost)
	}
}

func TestBOMCost_NonAssemblyPart(t *testing.T) {
	// Edge case: Cost endpoint called for non-assembly part (no PCA- or ASY- prefix)
	// Should not include bom_cost in response

	oldDB := db
	testDB := setupBOMCostTestDB(t)
	defer testDB.Close()
	db = testDB
	defer func() { db = oldDB }()

	dir := t.TempDir()
	setupBOMTestParts(t, dir)

	oldPartsDir := partsDir
	partsDir = dir
	defer func() { partsDir = oldPartsDir }()

	// Insert cost for a regular component
	insertPOCost(t, testDB, "PO-001", "RES-001", 0.10)

	// Test cost endpoint for non-assembly
	req := httptest.NewRequest("GET", "/api/v1/parts/RES-001/cost", nil)
	w := httptest.NewRecorder()
	handlePartCost(w, req, "RES-001")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var apiResp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &apiResp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	// Should have last_unit_price but NOT bom_cost
	if _, exists := apiResp.Data["bom_cost"]; exists {
		t.Errorf("bom_cost should not be present for non-assembly part, got: %v", apiResp.Data)
	}

	if lastPrice, ok := apiResp.Data["last_unit_price"].(float64); !ok || lastPrice != 0.10 {
		t.Errorf("expected last_unit_price 0.10, got %v", apiResp.Data["last_unit_price"])
	}
}

// pad is a helper to zero-pad integers
func pad(n, width int) string {
	return fmt.Sprintf("%0*d", width, n)
}
