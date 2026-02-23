package procurement_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zrp/internal/handlers/procurement"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

// setupReceivingEcoTestDB creates a full-featured test DB (similar to the root setupTestDB)
// used by the receiving/inspection tests that originated from receiving_eco_test.go.
func setupReceivingEcoTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	schemas := []string{
		`CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			notes TEXT,
			status TEXT DEFAULT 'active',
			lead_time_days INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','sent','confirmed','partial','received','cancelled')),
			notes TEXT,
			created_by TEXT DEFAULT '',
			total REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date DATE,
			received_at DATETIME,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE RESTRICT
		)`,
		`CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			mpn TEXT DEFAULT '',
			manufacturer TEXT DEFAULT '',
			qty_ordered REAL NOT NULL CHECK(qty_ordered > 0),
			qty_received REAL DEFAULT 0 CHECK(qty_received >= 0),
			unit_price REAL DEFAULT 0,
			notes TEXT DEFAULT '',
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE receiving_inspections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			po_line_id INTEGER NOT NULL,
			ipn TEXT NOT NULL,
			qty_received REAL NOT NULL CHECK(qty_received >= 0),
			qty_passed REAL DEFAULT 0 CHECK(qty_passed >= 0),
			qty_failed REAL DEFAULT 0 CHECK(qty_failed >= 0),
			qty_on_hold REAL DEFAULT 0 CHECK(qty_on_hold >= 0),
			inspector TEXT,
			inspected_at DATETIME,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			qty_reserved REAL DEFAULT 0,
			location TEXT,
			description TEXT DEFAULT '',
			mpn TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('receive','issue','adjust','transfer','return','scrap')),
			qty REAL NOT NULL,
			reference TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			ipn TEXT,
			defect_type TEXT DEFAULT 'receiving',
			severity TEXT DEFAULT 'minor',
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, schema := range schemas {
		if _, err := testDB.Exec(schema); err != nil {
			t.Fatalf("Failed to create table: %v\nSchema: %s", err, schema)
		}
	}

	return testDB
}

// insertReceivingInspectionEco inserts vendor, PO, PO line, and receiving inspection for testing.
func insertReceivingInspectionEco(t *testing.T, db *sql.DB, poID, ipn string, qtyReceived float64) int {
	t.Helper()
	// Ensure vendor exists first (required by FK constraint)
	_, err := db.Exec(`INSERT OR IGNORE INTO vendors (id, name, status) VALUES ('V-001', 'Test Vendor', 'active')`)
	if err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}

	// Ensure PO exists
	_, err = db.Exec(`INSERT OR IGNORE INTO purchase_orders (id, vendor_id, status, created_at) VALUES (?, 'V-001', 'confirmed', datetime('now'))`, poID)
	if err != nil {
		t.Fatalf("Failed to create PO: %v", err)
	}

	// Create PO line
	result, err := db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES (?, ?, ?, 1.00)`, poID, ipn, qtyReceived)
	if err != nil {
		t.Fatal("Failed to create PO line:", err)
	}
	poLineID, _ := result.LastInsertId()

	res, err := db.Exec(`INSERT INTO receiving_inspections (po_id, po_line_id, ipn, qty_received, created_at) VALUES (?, ?, ?, ?, datetime('now'))`,
		poID, poLineID, ipn, qtyReceived)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// extractDataJSON extracts the "data" field from an APIResponse JSON body.
func extractDataJSON(body []byte) json.RawMessage {
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	json.Unmarshal(body, &resp)
	return resp.Data
}

// --- Receiving & Inspection Tests (from receiving_eco_test.go) ---

func TestListReceivingAll(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	insertReceivingInspectionEco(t, db, "PO-001", "IPN-001", 100)
	insertReceivingInspectionEco(t, db, "PO-002", "IPN-002", 50)

	req := httptest.NewRequest("GET", "/api/v1/receiving", nil)
	w := httptest.NewRecorder()
	h.ListReceiving(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var items []models.ReceivingInspection
	json.Unmarshal(extractDataJSON(w.Body.Bytes()), &items)
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestListReceivingPending(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	id1 := insertReceivingInspectionEco(t, db, "PO-001", "IPN-001", 100)
	insertReceivingInspectionEco(t, db, "PO-002", "IPN-002", 50)

	db.Exec("UPDATE receiving_inspections SET inspected_at=datetime('now'), qty_passed=100, inspector='tester' WHERE id=?", id1)

	req := httptest.NewRequest("GET", "/api/v1/receiving?status=pending", nil)
	w := httptest.NewRecorder()
	h.ListReceiving(w, req)

	var items []models.ReceivingInspection
	json.Unmarshal(extractDataJSON(w.Body.Bytes()), &items)
	if len(items) != 1 {
		t.Errorf("expected 1 pending, got %d", len(items))
	}
}

func TestListReceivingInspected(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	id1 := insertReceivingInspectionEco(t, db, "PO-001", "IPN-001", 100)
	insertReceivingInspectionEco(t, db, "PO-002", "IPN-002", 50)

	db.Exec("UPDATE receiving_inspections SET inspected_at=datetime('now'), qty_passed=100, inspector='tester' WHERE id=?", id1)

	req := httptest.NewRequest("GET", "/api/v1/receiving?status=inspected", nil)
	w := httptest.NewRecorder()
	h.ListReceiving(w, req)

	var items []models.ReceivingInspection
	json.Unmarshal(extractDataJSON(w.Body.Bytes()), &items)
	if len(items) != 1 {
		t.Errorf("expected 1 inspected, got %d", len(items))
	}
}

func TestInspectPass(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	id := insertReceivingInspectionEco(t, db, "PO-001", "IPN-001", 100)
	idStr := fmt.Sprintf("%d", id)

	body := `{"qty_passed":100,"qty_failed":0,"qty_on_hold":0,"inspector":"alice","notes":"all good"}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/"+idStr+"/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.InspectReceiving(w, req, idStr)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var ri models.ReceivingInspection
	json.Unmarshal(extractDataJSON(w.Body.Bytes()), &ri)
	if ri.QtyPassed != 100 {
		t.Errorf("expected qty_passed=100, got %f", ri.QtyPassed)
	}
	if ri.Inspector != "alice" {
		t.Errorf("expected inspector=alice, got %s", ri.Inspector)
	}
	if ri.InspectedAt == nil {
		t.Error("expected inspected_at to be set")
	}

	// Verify inventory was updated
	var qty float64
	err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn='IPN-001'").Scan(&qty)
	if err != nil {
		t.Fatalf("inventory not found: %v", err)
	}
	if qty != 100 {
		t.Errorf("expected inventory qty=100, got %f", qty)
	}

	// Verify inventory transaction
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn='IPN-001' AND type='receive'").Scan(&txCount)
	if txCount != 1 {
		t.Errorf("expected 1 inventory transaction, got %d", txCount)
	}
}

func TestInspectFail(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	id := insertReceivingInspectionEco(t, db, "PO-001", "IPN-002", 50)
	idStr := fmt.Sprintf("%d", id)

	body := `{"qty_passed":30,"qty_failed":20,"qty_on_hold":0,"inspector":"bob","notes":"cracks found"}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/"+idStr+"/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.InspectReceiving(w, req, idStr)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify NCR was auto-created
	var ncrCount int
	db.QueryRow("SELECT COUNT(*) FROM ncrs WHERE ipn='IPN-002' AND defect_type='receiving'").Scan(&ncrCount)
	if ncrCount != 1 {
		t.Errorf("expected 1 auto-NCR, got %d", ncrCount)
	}

	// Verify passed qty went to inventory
	var qty float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn='IPN-002'").Scan(&qty)
	if qty != 30 {
		t.Errorf("expected inventory qty=30, got %f", qty)
	}
}

func TestInspectHold(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	id := insertReceivingInspectionEco(t, db, "PO-001", "IPN-003", 200)
	idStr := fmt.Sprintf("%d", id)

	body := `{"qty_passed":0,"qty_failed":0,"qty_on_hold":200,"inspector":"charlie"}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/"+idStr+"/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.InspectReceiving(w, req, idStr)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var ri models.ReceivingInspection
	json.Unmarshal(extractDataJSON(w.Body.Bytes()), &ri)
	if ri.QtyOnHold != 200 {
		t.Errorf("expected qty_on_hold=200, got %f", ri.QtyOnHold)
	}

	// No inventory update for hold-only
	var qty float64
	err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn='IPN-003'").Scan(&qty)
	if err != nil {
		return // no inventory record is fine
	}
	if qty != 0 {
		t.Errorf("expected no inventory for hold, got %f", qty)
	}
}

func TestInspectExceedsReceived(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	id := insertReceivingInspectionEco(t, db, "PO-001", "IPN-001", 50)
	idStr := fmt.Sprintf("%d", id)

	body := `{"qty_passed":30,"qty_failed":30,"qty_on_hold":0}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/"+idStr+"/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.InspectReceiving(w, req, idStr)

	if w.Code != 400 {
		t.Errorf("expected 400 for exceeding qty, got %d", w.Code)
	}
}

func TestInspectNotFound(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	body := `{"qty_passed":10,"qty_failed":0,"qty_on_hold":0}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/9999/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.InspectReceiving(w, req, "9999")

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestInspectInvalidID(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	w := httptest.NewRecorder()
	h.InspectReceiving(w, httptest.NewRequest("POST", "/api/v1/receiving/abc/inspect", strings.NewReader(`{}`)), "abc")

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid id, got %d", w.Code)
	}
}

// --- BOM Where-Used Tests ---

func TestWhereUsedEmpty(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()

	tmpDir := t.TempDir()
	h := newTestHandler(db)
	h.PartsDir = tmpDir
	h.LoadPartsFromDir = func() (map[string][]models.Part, map[string][]string, map[string]string, error) {
		return nil, nil, nil, nil
	}

	req := httptest.NewRequest("GET", "/api/v1/parts/IPN-999/where-used", nil)
	w := httptest.NewRecorder()
	h.WhereUsed(w, req, "IPN-999")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []map[string]interface{}
	json.Unmarshal(extractDataJSON(w.Body.Bytes()), &results)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestWhereUsedFindsParentAssembly(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()

	tmpDir := t.TempDir()

	// Create assembly part CSV (this serves as both catalog entry and BOM)
	bomContent := "IPN,description,qty,ref\nPCA-100,Main Board,,\nIPN-001,10k Resistor,4,R1-R4\nIPN-002,100uF Cap,2,C1-C2\n"
	os.WriteFile(filepath.Join(tmpDir, "PCA-100.csv"), []byte(bomContent), 0644)

	h := newTestHandler(db)
	h.PartsDir = tmpDir
	h.LoadPartsFromDir = func() (map[string][]models.Part, map[string][]string, map[string]string, error) {
		cats := map[string][]models.Part{
			"assemblies": {
				{IPN: "PCA-100", Fields: map[string]string{"description": "Main Board"}},
			},
		}
		return cats, nil, nil, nil
	}
	h.GetPartByIPN = func(partsDir, ipn string) (map[string]string, error) {
		if ipn == "PCA-100" {
			return map[string]string{"description": "Main Board"}, nil
		}
		return nil, nil
	}

	req := httptest.NewRequest("GET", "/api/v1/parts/IPN-001/where-used", nil)
	w := httptest.NewRecorder()
	h.WhereUsed(w, req, "IPN-001")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var results []map[string]interface{}
	json.Unmarshal(extractDataJSON(w.Body.Bytes()), &results)
	if len(results) != 1 {
		t.Fatalf("expected 1 where-used entry, got %d; body: %s", len(results), w.Body.String())
	}
	if results[0]["assembly_ipn"] != "PCA-100" {
		t.Errorf("expected assembly_ipn PCA-100, got %v", results[0]["assembly_ipn"])
	}
}

func TestWhereUsedMultipleAssemblies(t *testing.T) {
	db := setupReceivingEcoTestDB(t)
	defer db.Close()
	resetIDCounter()

	tmpDir := t.TempDir()

	// Two assemblies both using IPN-001
	bom1 := "IPN,description,qty,ref\nPCA-100,Board A,,\nIPN-001,Resistor,4,R1-R4\n"
	bom2 := "IPN,description,qty,ref\nASY-200,Assembly B,,\nIPN-001,Resistor,2,R1-R2\nIPN-003,IC,1,U1\n"
	os.WriteFile(filepath.Join(tmpDir, "PCA-100.csv"), []byte(bom1), 0644)
	os.WriteFile(filepath.Join(tmpDir, "ASY-200.csv"), []byte(bom2), 0644)

	h := newTestHandler(db)
	h.PartsDir = tmpDir
	h.LoadPartsFromDir = func() (map[string][]models.Part, map[string][]string, map[string]string, error) {
		cats := map[string][]models.Part{
			"assemblies": {
				{IPN: "PCA-100", Fields: map[string]string{"description": "Board A"}},
				{IPN: "ASY-200", Fields: map[string]string{"description": "Assembly B"}},
			},
		}
		return cats, nil, nil, nil
	}
	h.GetPartByIPN = func(partsDir, ipn string) (map[string]string, error) {
		return nil, nil
	}

	req := httptest.NewRequest("GET", "/api/v1/parts/IPN-001/where-used", nil)
	w := httptest.NewRecorder()
	h.WhereUsed(w, req, "IPN-001")

	var results []map[string]interface{}
	json.Unmarshal(extractDataJSON(w.Body.Bytes()), &results)
	if len(results) != 2 {
		t.Errorf("expected 2 where-used entries, got %d", len(results))
	}
}

// Verify that the procurement.Handler type is used (suppress unused import)
var _ = (*procurement.Handler)(nil)
