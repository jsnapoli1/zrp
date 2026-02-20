package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

// setupNumericTestDB creates an in-memory database for numeric validation testing
func setupNumericTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create all necessary tables with CHECK constraints matching production
	schema := `
		CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			notes TEXT,
			status TEXT DEFAULT 'active' CHECK(status IN ('active','preferred','inactive','blocked')),
			lead_time_days INTEGER DEFAULT 0 CHECK(lead_time_days >= 0),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

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
		);

		CREATE TABLE inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('receive','issue','adjust','transfer','return','scrap')),
			qty REAL NOT NULL,
			reference TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE purchase_orders (
			id TEXT PRIMARY KEY,
			vendor_id TEXT,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','sent','confirmed','partial','received','cancelled')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expected_date TEXT,
			received_at DATETIME,
			created_by TEXT,
			FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE RESTRICT
		);

		CREATE TABLE po_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			po_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			mpn TEXT,
			manufacturer TEXT,
			qty_ordered REAL NOT NULL CHECK(qty_ordered > 0),
			qty_received REAL DEFAULT 0 CHECK(qty_received >= 0),
			unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (po_id) REFERENCES purchase_orders(id) ON DELETE CASCADE
		);

		CREATE TABLE quotes (
			id TEXT PRIMARY KEY,
			customer TEXT NOT NULL,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','sent','accepted','rejected','expired','cancelled')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			valid_until TEXT,
			accepted_at DATETIME
		);

		CREATE TABLE quote_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			quote_id TEXT NOT NULL,
			ipn TEXT NOT NULL,
			description TEXT,
			qty INTEGER NOT NULL CHECK(qty > 0),
			unit_price REAL DEFAULT 0,
			notes TEXT,
			FOREIGN KEY (quote_id) REFERENCES quotes(id) ON DELETE CASCADE
		);

		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL CHECK(qty > 0),
			qty_good INTEGER,
			qty_scrap INTEGER,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','open','in_progress','completed','cancelled','on_hold')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME
		);

		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`

	// Execute schema statements
	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return testDB
}

// ============================================================================
// PHASE 1: NUMERIC OVERFLOW TESTS (NO-001 to NO-006)
// ============================================================================

// TestInventoryQuantityOverflow tests very large and overflow values for inventory quantities
func TestInventoryQuantityOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name        string
		qty         float64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Normal quantity",
			qty:         100.0,
			expectError: false,
		},
		{
			name:        "Very large but valid quantity",
			qty:         999999999.0,
			expectError: false,
		},
		{
			name:        "Maximum safe float64 (should accept or handle gracefully)",
			qty:         1.7e308, // Near max float64
			expectError: false,   // SQLite REAL type should handle this
		},
		{
			name:        "Negative quantity (violates CHECK constraint)",
			qty:         -100.0,
			expectError: true,
			errorMsg:    "CHECK constraint failed",
		},
		{
			name:        "Zero quantity (valid - empty stock)",
			qty:         0.0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create inventory transaction via API
			txn := InventoryTransaction{
				IPN:  "TEST-001",
				Type: "adjust",
				Qty:  tt.qty,
			}

			body, _ := json.Marshal(txn)
			req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleInventoryTransact(w, req)

			if tt.expectError {
				if w.Code == 200 {
					t.Errorf("Expected error for qty=%v, but got success", tt.qty)
				}
			} else {
				if w.Code != 200 {
					t.Errorf("Expected success for qty=%v, but got status %d: %s", tt.qty, w.Code, w.Body.String())
				}
			}
		})
	}
}

// TestPOLineQuantityOverflow tests qty_ordered boundaries for purchase order lines
func TestPOLineQuantityOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create a vendor first
	_, err := db.Exec("INSERT INTO vendors (id, name) VALUES ('V-001', 'Test Vendor')")
	if err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}

	tests := []struct {
		name        string
		qtyOrdered  float64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Normal quantity",
			qtyOrdered:  100.0,
			expectError: false,
		},
		{
			name:        "Very large quantity (1 million)",
			qtyOrdered:  1000000.0,
			expectError: true, // Should be rejected by max validation (once added)
			errorMsg:    "exceeds maximum",
		},
		{
			name:        "Zero quantity (violates CHECK qty_ordered > 0)",
			qtyOrdered:  0.0,
			expectError: true,
			errorMsg:    "must be positive",
		},
		{
			name:        "Negative quantity",
			qtyOrdered:  -50.0,
			expectError: true,
			errorMsg:    "must be positive",
		},
		{
			name:        "Extremely large quantity (overflow risk)",
			qtyOrdered:  1e100,
			expectError: true,
			errorMsg:    "exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po := PurchaseOrder{
				VendorID: "V-001",
				Status:   "draft",
				Lines: []POLine{
					{
						IPN:        "PART-001",
						QtyOrdered: tt.qtyOrdered,
						UnitPrice:  10.0,
					},
				},
			}

			body, _ := json.Marshal(po)
			req := httptest.NewRequest("POST", "/api/v1/procurement/pos", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreatePO(w, req)

			if tt.expectError {
				if w.Code == 200 {
					t.Errorf("Expected error for qty_ordered=%v, but got success", tt.qtyOrdered)
				}
				// Note: Some tests will fail until we add max validation
			} else {
				if w.Code != 200 {
					t.Errorf("Expected success for qty_ordered=%v, but got status %d: %s", tt.qtyOrdered, w.Code, w.Body.String())
				}
			}
		})
	}
}

// TestPriceFieldOverflow tests unit_price boundaries for extreme values
func TestPriceFieldOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	_, err := db.Exec("INSERT INTO vendors (id, name) VALUES ('V-001', 'Test Vendor')")
	if err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}

	tests := []struct {
		name        string
		unitPrice   float64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Normal price",
			unitPrice:   99.99,
			expectError: false,
		},
		{
			name:        "Zero price (valid - free item)",
			unitPrice:   0.0,
			expectError: false,
		},
		{
			name:        "Very high price",
			unitPrice:   999999.99,
			expectError: false,
		},
		{
			name:        "Negative price (should be rejected)",
			unitPrice:   -50.00,
			expectError: true,
			errorMsg:    "must be non-negative",
		},
		{
			name:        "Extremely large price (overflow risk)",
			unitPrice:   1e100,
			expectError: true,
			errorMsg:    "exceeds maximum",
		},
		{
			name:        "Price with extreme precision",
			unitPrice:   0.123456789012345,
			expectError: false, // Should accept but may lose precision
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			po := PurchaseOrder{
				VendorID: "V-001",
				Status:   "draft",
				Lines: []POLine{
					{
						IPN:        "PART-001",
						QtyOrdered: 10.0,
						UnitPrice:  tt.unitPrice,
					},
				},
			}

			body, _ := json.Marshal(po)
			req := httptest.NewRequest("POST", "/api/v1/procurement/pos", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreatePO(w, req)

			if tt.expectError {
				if w.Code == 200 {
					t.Errorf("Expected error for unit_price=%v, but got success", tt.unitPrice)
				}
			} else {
				if w.Code != 200 {
					t.Errorf("Expected success for unit_price=%v, but got status %d: %s", tt.unitPrice, w.Code, w.Body.String())
				}
			}
		})
	}
}

// TestLeadTimeDaysOverflow tests lead_time_days boundaries for vendors
func TestLeadTimeDaysOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name         string
		leadTimeDays int
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "Normal lead time (7 days)",
			leadTimeDays: 7,
			expectError:  false,
		},
		{
			name:         "Zero lead time (same day)",
			leadTimeDays: 0,
			expectError:  false,
		},
		{
			name:         "Long but reasonable (90 days)",
			leadTimeDays: 90,
			expectError:  false,
		},
		{
			name:         "Very long lead time (1 year = 365 days)",
			leadTimeDays: 365,
			expectError:  false,
		},
		{
			name:         "Unreasonably long (10000 days = 27 years)",
			leadTimeDays: 10000,
			expectError:  true, // Should be rejected by max validation
			errorMsg:     "exceeds maximum",
		},
		{
			name:         "Negative lead time (invalid)",
			leadTimeDays: -5,
			expectError:  true,
			errorMsg:     "must be non-negative",
		},
		{
			name:         "INT_MAX overflow risk",
			leadTimeDays: 2147483647, // Max int32
			expectError:  true,
			errorMsg:     "exceeds maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vendor := Vendor{
				Name:         "Test Vendor",
				LeadTimeDays: tt.leadTimeDays,
			}

			body, _ := json.Marshal(vendor)
			req := httptest.NewRequest("POST", "/api/v1/vendors", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateVendor(w, req)

			if tt.expectError {
				if w.Code == 200 {
					t.Errorf("Expected error for lead_time_days=%d, but got success", tt.leadTimeDays)
				}
			} else {
				if w.Code != 200 {
					t.Errorf("Expected success for lead_time_days=%d, but got status %d: %s", tt.leadTimeDays, w.Code, w.Body.String())
				}
			}
		})
	}
}

// TestQuoteLineQuantityOverflow tests qty boundaries for quote lines
func TestQuoteLineQuantityOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name        string
		qty         int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Normal quantity",
			qty:         100,
			expectError: false,
		},
		{
			name:        "Large quantity (100k)",
			qty:         100000,
			expectError: false,
		},
		{
			name:        "Very large quantity (1 million)",
			qty:         1000000,
			expectError: true,
			errorMsg:    "exceeds maximum",
		},
		{
			name:        "Zero quantity (violates CHECK qty > 0)",
			qty:         0,
			expectError: true,
			errorMsg:    "must be positive",
		},
		{
			name:        "Negative quantity",
			qty:         -100,
			expectError: true,
			errorMsg:    "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quote := Quote{
				Customer: "Test Customer",
				Lines: []QuoteLine{
					{
						IPN:       "PART-001",
						Qty:       tt.qty,
						UnitPrice: 50.0,
					},
				},
			}

			body, _ := json.Marshal(quote)
			req := httptest.NewRequest("POST", "/api/v1/quotes", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateQuote(w, req)

			if tt.expectError {
				if w.Code == 200 {
					t.Errorf("Expected error for qty=%d, but got success", tt.qty)
				}
			} else {
				if w.Code != 200 {
					t.Errorf("Expected success for qty=%d, but got status %d: %s", tt.qty, w.Code, w.Body.String())
				}
			}
		})
	}
}

// ============================================================================
// PHASE 1: FLOATING POINT PRECISION TESTS (NO-007 to NO-012)
// ============================================================================

// TestFloatingPointPrecision tests precision issues with REAL types
func TestFloatingPointPrecision(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name      string
		qty       float64
		checkFunc func(t *testing.T, stored float64)
	}{
		{
			name: "Repeating decimal (0.33333...)",
			qty:  0.33333333333333,
			checkFunc: func(t *testing.T, stored float64) {
				// Should be stored with some precision loss
				if math.Abs(stored-0.33333333333333) > 1e-10 {
					t.Errorf("Precision loss too large: expected ~0.333, got %v", stored)
				}
			},
		},
		{
			name: "Very precise decimal",
			qty:  99.999999999,
			checkFunc: func(t *testing.T, stored float64) {
				// Check stored value is close
				if math.Abs(stored-99.999999999) > 1e-6 {
					t.Errorf("Unexpected precision loss: got %v", stored)
				}
			},
		},
		{
			name: "Small difference test (1.0000000001 vs 1.0)",
			qty:  1.0000000001,
			checkFunc: func(t *testing.T, stored float64) {
				if stored == 1.0 {
					// Precision lost to rounding - this is expected for SQLite REAL
					t.Logf("Small precision difference was rounded: %v -> %v", 1.0000000001, stored)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Insert directly to database
			_, err := db.Exec("INSERT INTO inventory (ipn, qty_on_hand) VALUES (?, ?)", tt.name, tt.qty)
			if err != nil {
				t.Fatalf("Failed to insert: %v", err)
			}

			// Retrieve and check
			var stored float64
			err = db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", tt.name).Scan(&stored)
			if err != nil {
				t.Fatalf("Failed to retrieve: %v", err)
			}

			tt.checkFunc(t, stored)
		})
	}
}

// TestPriceCalculationAccuracy tests accumulation errors in price calculations
func TestPriceCalculationAccuracy(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create vendor
	_, err := db.Exec("INSERT INTO vendors (id, name) VALUES ('V-001', 'Test Vendor')")
	if err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}

	// Create PO with ID
	_, err = db.Exec("INSERT INTO purchase_orders (id, vendor_id) VALUES ('PO-0001', 'V-001')")
	if err != nil {
		t.Fatalf("Failed to create PO: %v", err)
	}

	// Test: Add many small-value line items and check total
	numLines := 1000
	unitPrice := 0.01

	for i := 0; i < numLines; i++ {
		_, err = db.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES (?, ?, ?, ?)",
			"PO-0001", fmt.Sprintf("PART-%04d", i), 1.0, unitPrice)
		if err != nil {
			t.Fatalf("Failed to insert line %d: %v", i, err)
		}
	}

	// Calculate total
	var total float64
	err = db.QueryRow("SELECT SUM(qty_ordered * unit_price) FROM po_lines WHERE po_id=?", "PO-0001").Scan(&total)
	if err != nil {
		t.Fatalf("Failed to calculate total: %v", err)
	}

	expected := float64(numLines) * unitPrice
	diff := math.Abs(total - expected)

	t.Logf("Expected total: %v, Calculated total: %v, Difference: %v", expected, total, diff)

	// Allow for small floating point errors
	if diff > 0.01 {
		t.Errorf("Accumulation error too large: expected %v, got %v (diff: %v)", expected, total, diff)
	}
}

// TestWorkOrderQuantityOverflow tests work order qty boundaries
func TestWorkOrderQuantityOverflow(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name        string
		qty         int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Normal quantity",
			qty:         100,
			expectError: false,
		},
		{
			name:        "Large batch (10k)",
			qty:         10000,
			expectError: false,
		},
		{
			name:        "Very large batch (100k)",
			qty:         100000,
			expectError: true,
			errorMsg:    "exceeds maximum",
		},
		{
			name:        "Zero quantity (violates CHECK qty > 0)",
			qty:         0,
			expectError: true,
			errorMsg:    "CHECK constraint failed",
		},
		{
			name:        "Negative quantity",
			qty:         -50,
			expectError: true,
			errorMsg:    "CHECK constraint failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec("INSERT INTO work_orders (id, assembly_ipn, qty) VALUES (?, ?, ?)",
				fmt.Sprintf("WO-%04d", tt.qty), "ASSY-001", tt.qty)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for qty=%d, but insert succeeded", tt.qty)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success for qty=%d, but got error: %v", tt.qty, err)
				}
			}
		})
	}
}

// ============================================================================
// PERCENTAGE FIELD TESTS
// ============================================================================

// TestPercentageFieldValidation tests percentage bounds (0-100%)
// Note: Current schema doesn't have explicit percentage fields, but this
// demonstrates how to validate them when added
func TestPercentageFieldValidation(t *testing.T) {
	// This is a placeholder test for when percentage fields are added
	// (e.g., discount_percent, tax_rate, yield_rate, etc.)

	tests := []struct {
		name        string
		percentage  float64
		expectError bool
	}{
		{"0 percent", 0.0, false},
		{"50 percent", 50.0, false},
		{"100 percent", 100.0, false},
		{"Negative percent", -10.0, true},
		{"Over 100 percent", 150.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Example validation logic that should be added to handlers
			isValid := tt.percentage >= 0 && tt.percentage <= 100
			if isValid == tt.expectError {
				t.Errorf("Validation mismatch for %v%%: expected error=%v, got valid=%v",
					tt.percentage, tt.expectError, isValid)
			}
		})
	}
}

// ============================================================================
// BOUNDARY VALUE TESTS (BC-011 to BC-023 from test plan)
// ============================================================================

// TestInventoryZeroValues tests that zero values are handled correctly
func TestInventoryZeroValues(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Test zero qty_on_hand (should be accepted)
	_, err := db.Exec("INSERT INTO inventory (ipn, qty_on_hand) VALUES ('ZERO-001', 0)")
	if err != nil {
		t.Errorf("Zero qty_on_hand should be accepted: %v", err)
	}

	// Test zero reorder_point (should be accepted - disables reordering)
	_, err = db.Exec("INSERT INTO inventory (ipn, reorder_point) VALUES ('ZERO-002', 0)")
	if err != nil {
		t.Errorf("Zero reorder_point should be accepted: %v", err)
	}

	// Test zero qty_reserved (should be accepted)
	_, err = db.Exec("INSERT INTO inventory (ipn, qty_reserved) VALUES ('ZERO-003', 0)")
	if err != nil {
		t.Errorf("Zero qty_reserved should be accepted: %v", err)
	}
}

// TestNegativeValueRejection tests that negative values are properly rejected
func TestNegativeValueRejection(t *testing.T) {
	oldDB := db
	db = setupNumericTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	tests := []struct {
		name  string
		query string
		args  []interface{}
	}{
		{
			name:  "Negative qty_on_hand",
			query: "INSERT INTO inventory (ipn, qty_on_hand) VALUES (?, ?)",
			args:  []interface{}{"NEG-001", -10.0},
		},
		{
			name:  "Negative qty_reserved",
			query: "INSERT INTO inventory (ipn, qty_reserved) VALUES (?, ?)",
			args:  []interface{}{"NEG-002", -5.0},
		},
		{
			name:  "Negative reorder_point",
			query: "INSERT INTO inventory (ipn, reorder_point) VALUES (?, ?)",
			args:  []interface{}{"NEG-003", -1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec(tt.query, tt.args...)
			if err == nil {
				t.Errorf("Expected CHECK constraint error for %s, but insert succeeded", tt.name)
			}
		})
	}
}
