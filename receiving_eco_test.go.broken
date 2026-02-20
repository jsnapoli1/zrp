package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// extractData extracts the "data" field from an APIResponse JSON body
func extractData(body []byte) json.RawMessage {
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	json.Unmarshal(body, &resp)
	return resp.Data
}

// --- Receiving & Inspection Tests ---

func insertReceivingInspection(t *testing.T, poID, ipn string, qtyReceived float64) int {
	t.Helper()
	// Ensure vendor exists first (required by FK constraint)
	result, err := db.Exec(`INSERT OR IGNORE INTO vendors (id, name, status) VALUES ('V-001', 'Test Vendor', 'active')`)
	if err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}
	
	// Verify vendor exists
	var vendorCount int
	db.QueryRow(`SELECT COUNT(*) FROM vendors WHERE id = 'V-001'`).Scan(&vendorCount)
	if vendorCount == 0 {
		t.Fatal("Vendor was not created")
	}
	
	// Ensure PO exists (status must be one of: draft, sent, confirmed, partial, received, cancelled)
	result, err = db.Exec(`INSERT OR IGNORE INTO purchase_orders (id, vendor_id, status, created_at) VALUES (?, 'V-001', 'confirmed', datetime('now'))`, poID)
	if err != nil {
		t.Fatalf("Failed to create PO: %v", err)
	}
	
	// Verify PO exists
	var poCount int
	db.QueryRow(`SELECT COUNT(*) FROM purchase_orders WHERE id = ?`, poID).Scan(&poCount)
	if poCount == 0 {
		t.Fatalf("PO was not created: %s", poID)
	}
	
	// Create PO line (let it auto-generate the ID)
	result, err = db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES (?, ?, ?, 1.00)`, poID, ipn, qtyReceived)
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

func TestListReceivingAll(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertReceivingInspection(t, "PO-001", "IPN-001", 100)
	insertReceivingInspection(t, "PO-002", "IPN-002", 50)

	req := httptest.NewRequest("GET", "/api/v1/receiving", nil)
	w := httptest.NewRecorder()
	handleListReceiving(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var items []ReceivingInspection
	json.Unmarshal(extractData(w.Body.Bytes()), &items)
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestListReceivingPending(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	id1 := insertReceivingInspection(t, "PO-001", "IPN-001", 100)
	insertReceivingInspection(t, "PO-002", "IPN-002", 50)

	db.Exec("UPDATE receiving_inspections SET inspected_at=datetime('now'), qty_passed=100, inspector='tester' WHERE id=?", id1)

	req := httptest.NewRequest("GET", "/api/v1/receiving?status=pending", nil)
	w := httptest.NewRecorder()
	handleListReceiving(w, req)

	var items []ReceivingInspection
	json.Unmarshal(extractData(w.Body.Bytes()), &items)
	if len(items) != 1 {
		t.Errorf("expected 1 pending, got %d", len(items))
	}
}

func TestListReceivingInspected(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	id1 := insertReceivingInspection(t, "PO-001", "IPN-001", 100)
	insertReceivingInspection(t, "PO-002", "IPN-002", 50)

	db.Exec("UPDATE receiving_inspections SET inspected_at=datetime('now'), qty_passed=100, inspector='tester' WHERE id=?", id1)

	req := httptest.NewRequest("GET", "/api/v1/receiving?status=inspected", nil)
	w := httptest.NewRecorder()
	handleListReceiving(w, req)

	var items []ReceivingInspection
	json.Unmarshal(extractData(w.Body.Bytes()), &items)
	if len(items) != 1 {
		t.Errorf("expected 1 inspected, got %d", len(items))
	}
}

func TestInspectPass(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	id := insertReceivingInspection(t, "PO-001", "IPN-001", 100)
	idStr := fmt.Sprintf("%d", id)

	body := `{"qty_passed":100,"qty_failed":0,"qty_on_hold":0,"inspector":"alice","notes":"all good"}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/"+idStr+"/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleInspectReceiving(w, req, idStr)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var ri ReceivingInspection
	json.Unmarshal(extractData(w.Body.Bytes()), &ri)
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
	cleanup := setupTestDB(t)
	defer cleanup()

	id := insertReceivingInspection(t, "PO-001", "IPN-002", 50)
	idStr := fmt.Sprintf("%d", id)

	body := `{"qty_passed":30,"qty_failed":20,"qty_on_hold":0,"inspector":"bob","notes":"cracks found"}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/"+idStr+"/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleInspectReceiving(w, req, idStr)

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
	cleanup := setupTestDB(t)
	defer cleanup()

	id := insertReceivingInspection(t, "PO-001", "IPN-003", 200)
	idStr := fmt.Sprintf("%d", id)

	body := `{"qty_passed":0,"qty_failed":0,"qty_on_hold":200,"inspector":"charlie"}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/"+idStr+"/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleInspectReceiving(w, req, idStr)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var ri ReceivingInspection
	json.Unmarshal(extractData(w.Body.Bytes()), &ri)
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
	cleanup := setupTestDB(t)
	defer cleanup()

	id := insertReceivingInspection(t, "PO-001", "IPN-001", 50)
	idStr := fmt.Sprintf("%d", id)

	body := `{"qty_passed":30,"qty_failed":30,"qty_on_hold":0}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/"+idStr+"/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleInspectReceiving(w, req, idStr)

	if w.Code != 400 {
		t.Errorf("expected 400 for exceeding qty, got %d", w.Code)
	}
}

func TestInspectNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	body := `{"qty_passed":10,"qty_failed":0,"qty_on_hold":0}`
	req := httptest.NewRequest("POST", "/api/v1/receiving/9999/inspect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleInspectReceiving(w, req, "9999")

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestInspectInvalidID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	w := httptest.NewRecorder()
	handleInspectReceiving(w, httptest.NewRequest("POST", "/api/v1/receiving/abc/inspect", strings.NewReader(`{}`)), "abc")

	if w.Code != 400 {
		t.Errorf("expected 400 for invalid id, got %d", w.Code)
	}
}

// --- ECO Revision Tests ---

func createTestECO(t *testing.T) string {
	t.Helper()
	body := `{"title":"Test ECO","description":"Test description","affected_ipns":"IPN-001"}`
	req := httptest.NewRequest("POST", "/api/v1/ecos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleCreateECO(w, req)
	if w.Code != 200 {
		t.Fatalf("create ECO failed: %d %s", w.Code, w.Body.String())
	}
	var eco struct {
		Data ECO `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &eco)
	if eco.Data.ID == "" {
		t.Fatal("created ECO has empty ID")
	}
	return eco.Data.ID
}

func TestECOCreateHasInitialRevision(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ecoID := createTestECO(t)

	req := httptest.NewRequest("GET", "/api/v1/ecos/"+ecoID+"/revisions", nil)
	w := httptest.NewRecorder()
	handleListECORevisions(w, req, ecoID)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var revs []ECORevision
	json.Unmarshal(extractData(w.Body.Bytes()), &revs)
	if len(revs) != 1 {
		t.Fatalf("expected 1 initial revision, got %d", len(revs))
	}
	if revs[0].Revision != "A" {
		t.Errorf("expected revision A, got %s", revs[0].Revision)
	}
	if revs[0].ChangesSummary != "Initial revision" {
		t.Errorf("expected 'Initial revision', got %s", revs[0].ChangesSummary)
	}
}

func TestCreateECORevision(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ecoID := createTestECO(t)

	body := `{"changes_summary":"Updated BOM","effectivity_date":"2026-03-01","notes":"Critical update"}`
	req := httptest.NewRequest("POST", "/api/v1/ecos/"+ecoID+"/revisions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleCreateECORevision(w, req, ecoID)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(extractData(w.Body.Bytes()), &resp)
	if resp["revision"] != "B" {
		t.Errorf("expected revision B, got %v", resp["revision"])
	}
	if resp["changes_summary"] != "Updated BOM" {
		t.Errorf("expected 'Updated BOM', got %v", resp["changes_summary"])
	}
}

func TestGetSpecificRevision(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ecoID := createTestECO(t)

	req := httptest.NewRequest("GET", "/api/v1/ecos/"+ecoID+"/revisions/A", nil)
	w := httptest.NewRecorder()
	handleGetECORevision(w, req, ecoID, "A")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var rev ECORevision
	json.Unmarshal(extractData(w.Body.Bytes()), &rev)
	if rev.Revision != "A" {
		t.Errorf("expected A, got %s", rev.Revision)
	}
}

func TestGetRevisionNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ecoID := createTestECO(t)

	w := httptest.NewRecorder()
	handleGetECORevision(w, httptest.NewRequest("GET", "/api/v1/ecos/"+ecoID+"/revisions/Z", nil), ecoID, "Z")

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRevisionAutoIncrements(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	ecoID := createTestECO(t)

	for _, summary := range []string{"Rev B changes", "Rev C changes"} {
		body := fmt.Sprintf(`{"changes_summary":"%s"}`, summary)
		req := httptest.NewRequest("POST", "/api/v1/ecos/"+ecoID+"/revisions", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleCreateECORevision(w, req, ecoID)
	}

	// List all
	req := httptest.NewRequest("GET", "/api/v1/ecos/"+ecoID+"/revisions", nil)
	w := httptest.NewRecorder()
	handleListECORevisions(w, req, ecoID)

	var revs []ECORevision
	json.Unmarshal(extractData(w.Body.Bytes()), &revs)
	if len(revs) != 3 {
		t.Errorf("expected 3 revisions (A,B,C), got %d", len(revs))
	}
	if len(revs) >= 3 && revs[2].Revision != "C" {
		t.Errorf("expected last revision C, got %s", revs[2].Revision)
	}
}

func TestApproveECOUpdatesRevision(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	partsDir = t.TempDir()

	ecoID := createTestECO(t)

	req := httptest.NewRequest("POST", "/api/v1/ecos/"+ecoID+"/approve", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleApproveECO(w, req, ecoID)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check revision status
	rw := httptest.NewRecorder()
	handleGetECORevision(rw, httptest.NewRequest("GET", "/api/v1/ecos/"+ecoID+"/revisions/A", nil), ecoID, "A")

	var rev ECORevision
	json.Unmarshal(extractData(rw.Body.Bytes()), &rev)
	if rev.Status != "approved" {
		t.Errorf("expected revision status 'approved', got '%s'", rev.Status)
	}
	if rev.ApprovedBy == nil {
		t.Error("expected approved_by to be set")
	}
}

func TestImplementECOUpdatesRevision(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	partsDir = t.TempDir()

	ecoID := createTestECO(t)

	// Approve first
	req := httptest.NewRequest("POST", "/api/v1/ecos/"+ecoID+"/approve", nil)
	w := httptest.NewRecorder()
	handleApproveECO(w, req, ecoID)

	// Implement
	req = httptest.NewRequest("POST", "/api/v1/ecos/"+ecoID+"/implement", nil)
	w = httptest.NewRecorder()
	handleImplementECO(w, req, ecoID)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check revision status
	rw := httptest.NewRecorder()
	handleGetECORevision(rw, httptest.NewRequest("GET", "/api/v1/ecos/"+ecoID+"/revisions/A", nil), ecoID, "A")

	var rev ECORevision
	json.Unmarshal(extractData(rw.Body.Bytes()), &rev)
	if rev.Status != "implemented" {
		t.Errorf("expected revision status 'implemented', got '%s'", rev.Status)
	}
	if rev.ImplementedBy == nil {
		t.Error("expected implemented_by to be set")
	}
}

// --- BOM Where-Used Tests ---

func TestWhereUsedEmpty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	partsDir = t.TempDir()

	req := httptest.NewRequest("GET", "/api/v1/parts/IPN-999/where-used", nil)
	w := httptest.NewRecorder()
	handleWhereUsed(w, req, "IPN-999")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []map[string]interface{}
	json.Unmarshal(extractData(w.Body.Bytes()), &results)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestWhereUsedFindsParentAssembly(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Create assembly part CSV (this serves as both catalog entry and BOM)
	bomContent := "IPN,description,qty,ref\nPCA-100,Main Board,,\nIPN-001,10k Resistor,4,R1-R4\nIPN-002,100uF Cap,2,C1-C2\n"
	os.WriteFile(filepath.Join(tmpDir, "PCA-100.csv"), []byte(bomContent), 0644)

	req := httptest.NewRequest("GET", "/api/v1/parts/IPN-001/where-used", nil)
	w := httptest.NewRecorder()
	handleWhereUsed(w, req, "IPN-001")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var results []map[string]interface{}
	json.Unmarshal(extractData(w.Body.Bytes()), &results)
	if len(results) != 1 {
		t.Fatalf("expected 1 where-used entry, got %d; body: %s", len(results), w.Body.String())
	}
	if results[0]["assembly_ipn"] != "PCA-100" {
		t.Errorf("expected assembly_ipn PCA-100, got %v", results[0]["assembly_ipn"])
	}
}

func TestWhereUsedMultipleAssemblies(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()
	partsDir = tmpDir

	// Two assemblies both using IPN-001
	bom1 := "IPN,description,qty,ref\nPCA-100,Board A,,\nIPN-001,Resistor,4,R1-R4\n"
	bom2 := "IPN,description,qty,ref\nASY-200,Assembly B,,\nIPN-001,Resistor,2,R1-R2\nIPN-003,IC,1,U1\n"
	os.WriteFile(filepath.Join(tmpDir, "PCA-100.csv"), []byte(bom1), 0644)
	os.WriteFile(filepath.Join(tmpDir, "ASY-200.csv"), []byte(bom2), 0644)

	req := httptest.NewRequest("GET", "/api/v1/parts/IPN-001/where-used", nil)
	w := httptest.NewRecorder()
	handleWhereUsed(w, req, "IPN-001")

	var results []map[string]interface{}
	json.Unmarshal(extractData(w.Body.Bytes()), &results)
	if len(results) != 2 {
		t.Errorf("expected 2 where-used entries, got %d", len(results))
	}
}
