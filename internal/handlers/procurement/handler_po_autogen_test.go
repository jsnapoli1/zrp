package procurement_test

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupPOAutogenTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", "file:test_po_autogen.db?mode=memory&cache=shared")
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
			status TEXT DEFAULT 'active'
		)`,
		`CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft',
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date TEXT,
			received_at DATETIME,
			created_by TEXT,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id)
		)`,
		`CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			mpn TEXT,
			manufacturer TEXT,
			qty_ordered REAL NOT NULL CHECK(qty_ordered > 0),
			qty_received REAL DEFAULT 0,
			unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT,
			qty INTEGER,
			status TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			qty_reserved REAL DEFAULT 0,
			location TEXT,
			description TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE bom_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_ipn TEXT NOT NULL,
			child_ipn TEXT NOT NULL,
			qty_per REAL NOT NULL,
			reference_designators TEXT,
			notes TEXT
		)`,
		`CREATE TABLE part_vendors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			vendor_id TEXT NOT NULL,
			mpn TEXT,
			manufacturer TEXT,
			unit_price REAL DEFAULT 0,
			lead_time_days INTEGER DEFAULT 0,
			moq INTEGER DEFAULT 1,
			is_preferred BOOLEAN DEFAULT 0,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id)
		)`,
		`CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE part_changes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user TEXT,
			table_name TEXT,
			record_id TEXT,
			operation TEXT,
			old_snapshot TEXT,
			new_snapshot TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE id_sequences (
			prefix TEXT PRIMARY KEY,
			next_num INTEGER
		)`,
		`CREATE TABLE po_suggestions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			wo_id TEXT NOT NULL,
			vendor_id TEXT NOT NULL,
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'rejected')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			reviewed_by TEXT,
			reviewed_at DATETIME,
			po_id TEXT,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id),
			FOREIGN KEY (wo_id) REFERENCES work_orders(id)
		)`,
		`CREATE TABLE po_suggestion_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			suggestion_id INTEGER NOT NULL,
			ipn TEXT NOT NULL,
			mpn TEXT,
			manufacturer TEXT,
			qty_needed REAL NOT NULL,
			estimated_unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (suggestion_id) REFERENCES po_suggestions(id) ON DELETE CASCADE
		)`,
	}

	for _, schema := range schemas {
		if _, err := testDB.Exec(schema); err != nil {
			t.Fatalf("Failed to create table: %v\nSchema: %s", err, schema)
		}
	}

	return testDB
}

// TestPOAutogen_BOMShortageDetection tests that shortages are correctly identified from BOM
func TestPOAutogen_BOMShortageDetection(t *testing.T) {
	db := setupPOAutogenTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	// Setup: Assembly that requires 100 units of a component, but only 30 in stock
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status) VALUES
		('WO-001', 'ASSY-PCB-001', 10, 'draft')
	`)

	// BOM: Assembly requires 10 resistors per unit
	db.Exec(`INSERT INTO bom_items (parent_ipn, child_ipn, qty_per) VALUES
		('ASSY-PCB-001', 'RES-10K-0805', 10.0)
	`)

	// Inventory: Only 30 resistors in stock (need 100 total: 10 WO qty * 10 per assembly)
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES
		('RES-10K-0805', 30.0)
	`)

	// Setup vendor and pricing
	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'DigiKey')`)
	db.Exec(`INSERT INTO part_vendors (ipn, vendor_id, mpn, manufacturer, unit_price, is_preferred) VALUES
		('RES-10K-0805', 'VEN-001', 'RC0805FR-0710KL', 'Yageo', 0.05, 1)
	`)

	// Generate PO suggestion from BOM shortage
	reqBody := `{"wo_id": "WO-001", "suggest_only": true}`
	req := httptest.NewRequest("POST", "/api/v1/pos/generate", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	h.GeneratePOSuggestions(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify suggestion was created
	var suggestionCount int
	db.QueryRow("SELECT COUNT(*) FROM po_suggestions WHERE wo_id = 'WO-001'").Scan(&suggestionCount)
	if suggestionCount != 1 {
		t.Errorf("Expected 1 PO suggestion, got %d", suggestionCount)
	}

	// Verify shortage quantity is correct (need 100, have 30, shortage = 70)
	var qtyNeeded float64
	db.QueryRow(`
		SELECT qty_needed FROM po_suggestion_lines psl
		JOIN po_suggestions ps ON ps.id = psl.suggestion_id
		WHERE ps.wo_id = 'WO-001' AND psl.ipn = 'RES-10K-0805'
	`).Scan(&qtyNeeded)

	expectedShortage := 70.0 // (10 WO qty * 10 per assembly) - 30 in stock
	if qtyNeeded != expectedShortage {
		t.Errorf("Expected shortage of %.0f, got %.0f", expectedShortage, qtyNeeded)
	}
}

// TestPOAutogen_MultiVendorSplitting tests that shortages are split by vendor
func TestPOAutogen_MultiVendorSplitting(t *testing.T) {
	db := setupPOAutogenTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	// Setup work order
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status) VALUES
		('WO-002', 'ASSY-PCB-002', 10, 'draft')
	`)

	// BOM with multiple components from different vendors
	db.Exec(`INSERT INTO bom_items (parent_ipn, child_ipn, qty_per) VALUES
		('ASSY-PCB-002', 'RES-10K-0805', 5.0),
		('ASSY-PCB-002', 'CAP-100N-0805', 8.0),
		('ASSY-PCB-002', 'IC-MCU-001', 1.0)
	`)

	// All components have shortages
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES
		('RES-10K-0805', 0.0),
		('CAP-100N-0805', 0.0),
		('IC-MCU-001', 0.0)
	`)

	// Setup vendors: resistors & capacitors from DigiKey, ICs from Mouser
	db.Exec(`INSERT INTO vendors (id, name) VALUES
		('VEN-DK', 'DigiKey'),
		('VEN-MS', 'Mouser')
	`)

	db.Exec(`INSERT INTO part_vendors (ipn, vendor_id, mpn, unit_price, is_preferred) VALUES
		('RES-10K-0805', 'VEN-DK', 'RC0805FR-0710KL', 0.05, 1),
		('CAP-100N-0805', 'VEN-DK', 'CL21B104KBCNNNC', 0.08, 1),
		('IC-MCU-001', 'VEN-MS', 'STM32F103C8T6', 3.50, 1)
	`)

	// Generate PO suggestions
	reqBody := `{"wo_id": "WO-002", "suggest_only": true}`
	req := httptest.NewRequest("POST", "/api/v1/pos/generate", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	h.GeneratePOSuggestions(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should create 2 suggestions: one for DigiKey (2 parts), one for Mouser (1 part)
	var suggestionCount int
	db.QueryRow("SELECT COUNT(*) FROM po_suggestions WHERE wo_id = 'WO-002'").Scan(&suggestionCount)
	if suggestionCount != 2 {
		t.Errorf("Expected 2 PO suggestions (split by vendor), got %d", suggestionCount)
	}

	// Verify DigiKey suggestion has 2 lines
	var dkLineCount int
	db.QueryRow(`
		SELECT COUNT(*) FROM po_suggestion_lines psl
		JOIN po_suggestions ps ON ps.id = psl.suggestion_id
		WHERE ps.wo_id = 'WO-002' AND ps.vendor_id = 'VEN-DK'
	`).Scan(&dkLineCount)
	if dkLineCount != 2 {
		t.Errorf("Expected 2 lines for DigiKey suggestion, got %d", dkLineCount)
	}

	// Verify Mouser suggestion has 1 line
	var msLineCount int
	db.QueryRow(`
		SELECT COUNT(*) FROM po_suggestion_lines psl
		JOIN po_suggestions ps ON ps.id = psl.suggestion_id
		WHERE ps.wo_id = 'WO-002' AND ps.vendor_id = 'VEN-MS'
	`).Scan(&msLineCount)
	if msLineCount != 1 {
		t.Errorf("Expected 1 line for Mouser suggestion, got %d", msLineCount)
	}
}

// TestPOAutogen_SuggestedPOIncludesCorrectDetails tests vendor, quantity, and pricing
func TestPOAutogen_SuggestedPOIncludesCorrectDetails(t *testing.T) {
	db := setupPOAutogenTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	// Setup
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status) VALUES
		('WO-003', 'ASSY-PCB-003', 5, 'draft')
	`)

	db.Exec(`INSERT INTO bom_items (parent_ipn, child_ipn, qty_per) VALUES
		('ASSY-PCB-003', 'RES-1K-0805', 4.0)
	`)

	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES
		('RES-1K-0805', 5.0)
	`)

	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'DigiKey')`)

	db.Exec(`INSERT INTO part_vendors (ipn, vendor_id, mpn, manufacturer, unit_price, is_preferred) VALUES
		('RES-1K-0805', 'VEN-001', 'RC0805FR-071KL', 'Yageo', 0.03, 1)
	`)

	// Generate suggestion
	reqBody := `{"wo_id": "WO-003", "suggest_only": true}`
	req := httptest.NewRequest("POST", "/api/v1/pos/generate", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	h.GeneratePOSuggestions(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify suggestion details
	var vendorID, mpn, manufacturer string
	var qtyNeeded, unitPrice float64
	err := db.QueryRow(`
		SELECT ps.vendor_id, psl.mpn, psl.manufacturer, psl.qty_needed, psl.estimated_unit_price
		FROM po_suggestions ps
		JOIN po_suggestion_lines psl ON psl.suggestion_id = ps.id
		WHERE ps.wo_id = 'WO-003' AND psl.ipn = 'RES-1K-0805'
	`).Scan(&vendorID, &mpn, &manufacturer, &qtyNeeded, &unitPrice)

	if err != nil {
		t.Fatalf("Failed to query suggestion details: %v", err)
	}

	// Need 20 (5 WO qty * 4 per assembly), have 5, shortage = 15
	expectedQty := 15.0
	if qtyNeeded != expectedQty {
		t.Errorf("Expected qty_needed = %.0f, got %.0f", expectedQty, qtyNeeded)
	}

	if vendorID != "VEN-001" {
		t.Errorf("Expected vendor VEN-001, got %s", vendorID)
	}

	if mpn != "RC0805FR-071KL" {
		t.Errorf("Expected MPN RC0805FR-071KL, got %s", mpn)
	}

	if manufacturer != "Yageo" {
		t.Errorf("Expected manufacturer Yageo, got %s", manufacturer)
	}

	if unitPrice != 0.03 {
		t.Errorf("Expected unit price 0.03, got %.2f", unitPrice)
	}
}

// TestPOAutogen_ApproveRejectWorkflow tests user can approve or reject suggestions
func TestPOAutogen_ApproveRejectWorkflow(t *testing.T) {
	db := setupPOAutogenTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	// Setup
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty) VALUES ('WO-004', 'ASSY-001', 10)`)
	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'DigiKey')`)
	db.Exec(`INSERT INTO bom_items (parent_ipn, child_ipn, qty_per) VALUES ('ASSY-001', 'RES-001', 2.0)`)
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('RES-001', 0.0)`)
	db.Exec(`INSERT INTO part_vendors (ipn, vendor_id, unit_price, is_preferred) VALUES ('RES-001', 'VEN-001', 0.10, 1)`)

	// Generate suggestion
	db.Exec(`INSERT INTO po_suggestions (wo_id, vendor_id, status) VALUES ('WO-004', 'VEN-001', 'pending')`)
	var suggestionID int
	db.QueryRow("SELECT id FROM po_suggestions WHERE wo_id = 'WO-004'").Scan(&suggestionID)
	db.Exec(`INSERT INTO po_suggestion_lines (suggestion_id, ipn, qty_needed, estimated_unit_price) VALUES (?, 'RES-001', 20.0, 0.10)`, suggestionID)

	// Test 1: Reject suggestion
	reqBody := `{"status": "rejected", "reason": "Too expensive"}`
	req := httptest.NewRequest("POST", "/api/v1/pos/suggestions/review", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	h.ReviewPOSuggestion(w, req, suggestionID)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for reject, got %d: %s", w.Code, w.Body.String())
	}

	var status string
	db.QueryRow("SELECT status FROM po_suggestions WHERE id = ?", suggestionID).Scan(&status)
	if status != "rejected" {
		t.Errorf("Expected status 'rejected', got %s", status)
	}

	// Test 2: Create new suggestion and approve it
	db.Exec(`INSERT INTO po_suggestions (wo_id, vendor_id, status) VALUES ('WO-004', 'VEN-001', 'pending')`)
	var suggestionID2 int
	db.QueryRow("SELECT id FROM po_suggestions WHERE wo_id = 'WO-004' AND status = 'pending'").Scan(&suggestionID2)
	db.Exec(`INSERT INTO po_suggestion_lines (suggestion_id, ipn, qty_needed, estimated_unit_price) VALUES (?, 'RES-001', 20.0, 0.10)`, suggestionID2)

	reqBody2 := `{"status": "approved"}`
	req2 := httptest.NewRequest("POST", "/api/v1/pos/suggestions/review", strings.NewReader(reqBody2))
	w2 := httptest.NewRecorder()

	h.ReviewPOSuggestion(w2, req2, suggestionID2)

	if w2.Code != 200 {
		t.Errorf("Expected status 200 for approve, got %d: %s", w2.Code, w2.Body.String())
	}

	db.QueryRow("SELECT status FROM po_suggestions WHERE id = ?", suggestionID2).Scan(&status)
	if status != "approved" {
		t.Errorf("Expected status 'approved', got %s", status)
	}
}

// TestPOAutogen_ApprovedPOCreated tests that approved suggestions create actual POs
func TestPOAutogen_ApprovedPOCreated(t *testing.T) {
	db := setupPOAutogenTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	// Setup
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty) VALUES ('WO-005', 'ASSY-002', 10)`)
	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-DK', 'DigiKey')`)
	db.Exec(`INSERT INTO bom_items (parent_ipn, child_ipn, qty_per) VALUES
		('ASSY-002', 'RES-002', 3.0),
		('ASSY-002', 'CAP-002', 5.0)
	`)
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES
		('RES-002', 0.0),
		('CAP-002', 10.0)
	`)
	db.Exec(`INSERT INTO part_vendors (ipn, vendor_id, mpn, unit_price, is_preferred) VALUES
		('RES-002', 'VEN-DK', 'RES-MPN-002', 0.05, 1),
		('CAP-002', 'VEN-DK', 'CAP-MPN-002', 0.12, 1)
	`)

	// Create suggestion
	db.Exec(`INSERT INTO po_suggestions (wo_id, vendor_id, status) VALUES ('WO-005', 'VEN-DK', 'pending')`)
	var suggestionID int
	db.QueryRow("SELECT id FROM po_suggestions WHERE wo_id = 'WO-005'").Scan(&suggestionID)

	// Need 30 RES-002 (10 * 3), have 0, shortage = 30
	// Need 50 CAP-002 (10 * 5), have 10, shortage = 40
	db.Exec(`INSERT INTO po_suggestion_lines (suggestion_id, ipn, mpn, qty_needed, estimated_unit_price) VALUES
		(?, 'RES-002', 'RES-MPN-002', 30.0, 0.05),
		(?, 'CAP-002', 'CAP-MPN-002', 40.0, 0.12)
	`, suggestionID, suggestionID)

	// Approve and create PO
	reqBody := `{"status": "approved", "create_po": true}`
	req := httptest.NewRequest("POST", "/api/v1/pos/suggestions/review", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	h.ReviewPOSuggestion(w, req, suggestionID)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify PO was created
	var poID string
	err := db.QueryRow("SELECT po_id FROM po_suggestions WHERE id = ?", suggestionID).Scan(&poID)
	if err != nil || poID == "" {
		t.Fatalf("Expected PO to be created and linked to suggestion, got error: %v, po_id: %s", err, poID)
	}

	// Verify PO has correct vendor
	var vendorID string
	db.QueryRow("SELECT vendor_id FROM purchase_orders WHERE id = ?", poID).Scan(&vendorID)
	if vendorID != "VEN-DK" {
		t.Errorf("Expected PO vendor VEN-DK, got %s", vendorID)
	}

	// Verify PO has correct lines (2 lines)
	var lineCount int
	db.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id = ?", poID).Scan(&lineCount)
	if lineCount != 2 {
		t.Errorf("Expected 2 PO lines, got %d", lineCount)
	}

	// Verify line quantities and prices
	type POLineDetail struct {
		IPN        string
		QtyOrdered float64
		UnitPrice  float64
		MPN        string
	}
	rows, _ := db.Query("SELECT ipn, qty_ordered, unit_price, mpn FROM po_lines WHERE po_id = ? ORDER BY ipn", poID)
	defer rows.Close()

	var lines []POLineDetail
	for rows.Next() {
		var l POLineDetail
		rows.Scan(&l.IPN, &l.QtyOrdered, &l.UnitPrice, &l.MPN)
		lines = append(lines, l)
	}

	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}

	// Verify first line (CAP-002)
	if lines[0].IPN != "CAP-002" || lines[0].QtyOrdered != 40.0 || lines[0].UnitPrice != 0.12 || lines[0].MPN != "CAP-MPN-002" {
		t.Errorf("CAP-002 line incorrect: IPN=%s, Qty=%.0f, Price=%.2f, MPN=%s",
			lines[0].IPN, lines[0].QtyOrdered, lines[0].UnitPrice, lines[0].MPN)
	}

	// Verify second line (RES-002)
	if lines[1].IPN != "RES-002" || lines[1].QtyOrdered != 30.0 || lines[1].UnitPrice != 0.05 || lines[1].MPN != "RES-MPN-002" {
		t.Errorf("RES-002 line incorrect: IPN=%s, Qty=%.0f, Price=%.2f, MPN=%s",
			lines[1].IPN, lines[1].QtyOrdered, lines[1].UnitPrice, lines[1].MPN)
	}

	// Verify PO status is draft
	var poStatus string
	db.QueryRow("SELECT status FROM purchase_orders WHERE id = ?", poID).Scan(&poStatus)
	if poStatus != "draft" {
		t.Errorf("Expected PO status 'draft', got %s", poStatus)
	}
}

// TestPOAutogen_NoSuggestionsWhenNoShortage tests that no suggestions are created when inventory is sufficient
func TestPOAutogen_NoSuggestionsWhenNoShortage(t *testing.T) {
	db := setupPOAutogenTestDB(t)
	defer db.Close()
	resetIDCounter()
	h := newTestHandler(db)

	// Setup: Sufficient inventory
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty) VALUES ('WO-006', 'ASSY-003', 5)`)
	db.Exec(`INSERT INTO bom_items (parent_ipn, child_ipn, qty_per) VALUES ('ASSY-003', 'RES-003', 2.0)`)
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('RES-003', 100.0)`) // Need only 10, have 100
	db.Exec(`INSERT INTO vendors (id, name) VALUES ('VEN-001', 'DigiKey')`)
	db.Exec(`INSERT INTO part_vendors (ipn, vendor_id, is_preferred) VALUES ('RES-003', 'VEN-001', 1)`)

	// Try to generate suggestions
	reqBody := `{"wo_id": "WO-006", "suggest_only": true}`
	req := httptest.NewRequest("POST", "/api/v1/pos/generate", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	h.GeneratePOSuggestions(w, req)

	// Should return 200 but with no suggestions created
	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var suggestionCount int
	db.QueryRow("SELECT COUNT(*) FROM po_suggestions WHERE wo_id = 'WO-006'").Scan(&suggestionCount)
	if suggestionCount != 0 {
		t.Errorf("Expected 0 suggestions when no shortage, got %d", suggestionCount)
	}

	// Check response message
	var response struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&response)
	if response.Data.Message != "No shortages found" {
		t.Errorf("Expected 'No shortages found' message, got: %s", response.Data.Message)
	}
}
