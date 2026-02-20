package main

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTransactionTestDB creates a test database with all necessary tables
func setupTransactionTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create all necessary tables with proper constraints
	tables := []string{
		`CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0 CHECK(qty_on_hand >= 0),
			qty_reserved REAL DEFAULT 0 CHECK(qty_reserved >= 0),
			location TEXT,
			reorder_point REAL DEFAULT 0 CHECK(reorder_point >= 0),
			reorder_qty REAL DEFAULT 0 CHECK(reorder_qty >= 0),
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
		`CREATE TABLE vendors (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			website TEXT,
			contact_name TEXT,
			contact_email TEXT,
			contact_phone TEXT,
			address TEXT DEFAULT '',
			payment_terms TEXT DEFAULT '',
			notes TEXT,
			status TEXT DEFAULT 'active' CHECK(status IN ('active','preferred','inactive','blocked')),
			lead_time_days INTEGER DEFAULT 0 CHECK(lead_time_days >= 0),
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
		`CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL CHECK(qty > 0),
			qty_good INTEGER,
			qty_scrap INTEGER,
			status TEXT DEFAULT 'open' CHECK(status IN ('open','in_progress','complete','cancelled')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','urgent')),
			notes TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME
		)`,
		`CREATE TABLE bom (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			assembly_ipn TEXT NOT NULL,
			component_ipn TEXT NOT NULL,
			qty REAL NOT NULL CHECK(qty > 0),
			reference_designator TEXT DEFAULT '',
			notes TEXT DEFAULT ''
		)`,
		`CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','review','approved','implemented','rejected','cancelled')),
			priority TEXT DEFAULT 'normal' CHECK(priority IN ('low','normal','high','critical')),
			affected_ipns TEXT,
			created_by TEXT DEFAULT 'engineer',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			approved_at DATETIME,
			approved_by TEXT
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

	for _, table := range tables {
		if _, err := testDB.Exec(table); err != nil {
			t.Fatalf("Failed to create table: %v\nSQL: %s", err, table)
		}
	}

	return testDB
}

// TestPOReceiptRollback tests that PO receipt is atomic - if any step fails, everything rolls back
func TestPOReceiptRollback(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	// Setup: Create vendor, PO, and PO lines
	_, err := testDB.Exec("INSERT INTO vendors (id, name) VALUES ('V001', 'Test Vendor')")
	if err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO purchase_orders (id, vendor_id, status) VALUES ('PO001', 'V001', 'confirmed')")
	if err != nil {
		t.Fatalf("Failed to create PO: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES ('PO001', 'PART-001', 100, 10.50)")
	if err != nil {
		t.Fatalf("Failed to create PO line: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES ('PO001', 'PART-002', 50, 5.25)")
	if err != nil {
		t.Fatalf("Failed to create PO line: %v", err)
	}

	// Test: Simulate PO receipt with transaction that should rollback
	// We'll force a failure by violating a CHECK constraint on the second item
	t.Run("PartialReceiptFailure_ShouldRollback", func(t *testing.T) {
		// Perform the failing transaction
		func() {
			tx, err := testDB.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}
			defer tx.Rollback()

			// Receipt line 1 - should succeed
			_, err = tx.Exec("UPDATE po_lines SET qty_received = qty_received + ? WHERE id = ?", 100.0, 1)
			if err != nil {
				t.Fatalf("First update failed unexpectedly: %v", err)
			}

			// Create inventory record
			_, err = tx.Exec("INSERT OR IGNORE INTO inventory (ipn) VALUES (?)", "PART-001")
			if err != nil {
				t.Fatalf("Inventory insert failed: %v", err)
			}

			// Update inventory
			_, err = tx.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand + ? WHERE ipn = ?", 100.0, "PART-001")
			if err != nil {
				t.Fatalf("Inventory update failed: %v", err)
			}

			// Receipt line 2 - this will fail due to negative quantity (violates CHECK constraint)
			_, err = tx.Exec("UPDATE po_lines SET qty_received = ? WHERE id = ?", -10.0, 2)
			if err == nil {
				t.Fatal("Expected negative quantity to fail CHECK constraint, but it succeeded")
			}

			// Transaction should rollback automatically on error
			// Don't commit - let defer rollback happen
		}()

		// Verify nothing was committed (after the transaction closure completes and rollback happens)
		var qtyReceived float64
		err = testDB.QueryRow("SELECT qty_received FROM po_lines WHERE id = ?", 1).Scan(&qtyReceived)
		if err != nil {
			t.Fatalf("Failed to query po_lines: %v", err)
		}
		if qtyReceived != 0 {
			t.Errorf("Expected qty_received = 0 after rollback, got %f", qtyReceived)
		}

		var qtyOnHand float64
		err = testDB.QueryRow("SELECT COALESCE(qty_on_hand, 0) FROM inventory WHERE ipn = ?", "PART-001").Scan(&qtyOnHand)
		if err != nil && err != sql.ErrNoRows {
			t.Fatalf("Failed to query inventory: %v", err)
		}
		if qtyOnHand != 0 {
			t.Errorf("Expected inventory qty_on_hand = 0 after rollback, got %f", qtyOnHand)
		}
	})

	// Test: Successful receipt with proper transaction handling
	t.Run("SuccessfulReceipt_ShouldCommit", func(t *testing.T) {
		tx, err := testDB.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Receipt both lines successfully
		_, err = tx.Exec("UPDATE po_lines SET qty_received = qty_received + ? WHERE id = ?", 100.0, 1)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to update po_line 1: %v", err)
		}

		_, err = tx.Exec("INSERT OR IGNORE INTO inventory (ipn) VALUES (?)", "PART-001")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert inventory: %v", err)
		}

		_, err = tx.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand + ? WHERE ipn = ?", 100.0, "PART-001")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to update inventory: %v", err)
		}

		_, err = tx.Exec("INSERT INTO inventory_transactions (ipn, type, qty, reference) VALUES (?, 'receive', ?, ?)",
			"PART-001", 100.0, "PO001")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert transaction: %v", err)
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Verify data was committed
		var qtyReceived float64
		err = testDB.QueryRow("SELECT qty_received FROM po_lines WHERE id = ?", 1).Scan(&qtyReceived)
		if err != nil {
			t.Fatalf("Failed to query po_lines: %v", err)
		}
		if qtyReceived != 100.0 {
			t.Errorf("Expected qty_received = 100 after commit, got %f", qtyReceived)
		}

		var qtyOnHand float64
		err = testDB.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "PART-001").Scan(&qtyOnHand)
		if err != nil {
			t.Fatalf("Failed to query inventory: %v", err)
		}
		if qtyOnHand != 100.0 {
			t.Errorf("Expected inventory qty_on_hand = 100 after commit, got %f", qtyOnHand)
		}

		var txCount int
		err = testDB.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE reference = 'PO001'").Scan(&txCount)
		if err != nil {
			t.Fatalf("Failed to count transactions: %v", err)
		}
		if txCount != 1 {
			t.Errorf("Expected 1 inventory transaction, got %d", txCount)
		}
	})
}

// TestWorkOrderCompletionRollback tests that work order completion is atomic
func TestWorkOrderCompletionRollback(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	// Setup: Create work order and inventory items
	_, err := testDB.Exec("INSERT INTO work_orders (id, assembly_ipn, qty, status) VALUES ('WO001', 'ASM-100', 10, 'in_progress')")
	if err != nil {
		t.Fatalf("Failed to create work order: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES ('COMP-001', 100, 50)")
	if err != nil {
		t.Fatalf("Failed to create inventory: %v", err)
	}

	t.Run("InventoryUpdateFailure_ShouldNotChangeWOStatus", func(t *testing.T) {
		// Perform the failing transaction
		func() {
			tx, err := testDB.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}
			defer tx.Rollback()

			// Step 1: Update work order status to complete
			_, err = tx.Exec("UPDATE work_orders SET status = 'complete', completed_at = datetime('now') WHERE id = ?", "WO001")
			if err != nil {
				t.Fatalf("Failed to update WO status: %v", err)
			}

			// Step 2: Add finished goods to inventory
			_, err = tx.Exec("INSERT OR IGNORE INTO inventory (ipn) VALUES (?)", "ASM-100")
			if err != nil {
				t.Fatalf("Failed to insert finished goods: %v", err)
			}

			// Step 3: Try to update inventory with invalid value (negative qty violates CHECK)
			_, err = tx.Exec("UPDATE inventory SET qty_on_hand = ? WHERE ipn = ?", -10.0, "ASM-100")
			if err == nil {
				t.Fatal("Expected negative inventory to fail CHECK constraint")
			}

			// Don't commit - let rollback happen
		}()

		// Verify work order status was not changed (after rollback completes)
		var status string
		err = testDB.QueryRow("SELECT status FROM work_orders WHERE id = ?", "WO001").Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query work order: %v", err)
		}
		if status != "in_progress" {
			t.Errorf("Expected WO status to remain 'in_progress' after rollback, got '%s'", status)
		}

		// Verify finished goods were not added
		var count int
		err = testDB.QueryRow("SELECT COUNT(*) FROM inventory WHERE ipn = ?", "ASM-100").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count inventory: %v", err)
		}
		if count > 0 {
			t.Errorf("Expected no ASM-100 inventory after rollback, but found %d records", count)
		}
	})

	t.Run("SuccessfulCompletion_ShouldCommitAllChanges", func(t *testing.T) {
		tx, err := testDB.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Complete work order with proper transaction handling
		_, err = tx.Exec("UPDATE work_orders SET status = 'complete', completed_at = datetime('now'), qty_good = ? WHERE id = ?", 10, "WO001")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to update WO: %v", err)
		}

		// Add finished goods
		_, err = tx.Exec("INSERT OR IGNORE INTO inventory (ipn, description) VALUES (?, ?)", "ASM-100", "Assembled ASM-100")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert finished goods: %v", err)
		}

		_, err = tx.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand + ? WHERE ipn = ?", 10.0, "ASM-100")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to update finished goods qty: %v", err)
		}

		// Log transaction
		_, err = tx.Exec("INSERT INTO inventory_transactions (ipn, type, qty, reference, notes) VALUES (?, 'receive', ?, ?, ?)",
			"ASM-100", 10.0, "WO001", "WO completion")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to log transaction: %v", err)
		}

		// Release reserved components
		_, err = tx.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand - ?, qty_reserved = qty_reserved - ? WHERE ipn = ?",
			50.0, 50.0, "COMP-001")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to release components: %v", err)
		}

		if err = tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Verify all changes were committed
		var status string
		var qtyGood sql.NullInt64
		err = testDB.QueryRow("SELECT status, qty_good FROM work_orders WHERE id = ?", "WO001").Scan(&status, &qtyGood)
		if err != nil {
			t.Fatalf("Failed to query WO: %v", err)
		}
		if status != "complete" {
			t.Errorf("Expected WO status 'complete', got '%s'", status)
		}
		if !qtyGood.Valid || qtyGood.Int64 != 10 {
			t.Errorf("Expected qty_good = 10, got %v", qtyGood)
		}

		var asmQty float64
		err = testDB.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "ASM-100").Scan(&asmQty)
		if err != nil {
			t.Fatalf("Failed to query finished goods: %v", err)
		}
		if asmQty != 10.0 {
			t.Errorf("Expected finished goods qty = 10, got %f", asmQty)
		}

		var compQty, compReserved float64
		err = testDB.QueryRow("SELECT qty_on_hand, qty_reserved FROM inventory WHERE ipn = ?", "COMP-001").Scan(&compQty, &compReserved)
		if err != nil {
			t.Fatalf("Failed to query components: %v", err)
		}
		if compQty != 50.0 {
			t.Errorf("Expected component qty_on_hand = 50, got %f", compQty)
		}
		if compReserved != 0 {
			t.Errorf("Expected component qty_reserved = 0, got %f", compReserved)
		}
	})
}

// TestECOImplementationRollback tests that ECO implementation is atomic
func TestECOImplementationRollback(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	// Setup: Create ECO and BOM
	_, err := testDB.Exec("INSERT INTO ecos (id, title, status, affected_ipns) VALUES ('ECO001', 'Test ECO', 'approved', 'PART-100')")
	if err != nil {
		t.Fatalf("Failed to create ECO: %v", err)
	}

	_, err = testDB.Exec("INSERT INTO bom (assembly_ipn, component_ipn, qty) VALUES ('ASM-100', 'COMP-OLD', 5)")
	if err != nil {
		t.Fatalf("Failed to create BOM: %v", err)
	}

	t.Run("BOMUpdateFailure_ShouldNotImplementECO", func(t *testing.T) {
		// Perform the failing transaction
		func() {
			tx, err := testDB.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}
			defer tx.Rollback()

			// Step 1: Update ECO status
			_, err = tx.Exec("UPDATE ecos SET status = 'implemented', updated_at = datetime('now') WHERE id = ?", "ECO001")
			if err != nil {
				t.Fatalf("Failed to update ECO: %v", err)
			}

			// Step 2: Try to update BOM with invalid data (negative qty violates CHECK)
			_, err = tx.Exec("UPDATE bom SET qty = ? WHERE assembly_ipn = ?", -5.0, "ASM-100")
			if err == nil {
				t.Fatal("Expected negative BOM qty to fail CHECK constraint")
			}

			// Don't commit - let rollback happen
		}()

		// Verify ECO was not implemented (after rollback completes)
		var status string
		err = testDB.QueryRow("SELECT status FROM ecos WHERE id = ?", "ECO001").Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query ECO: %v", err)
		}
		if status != "approved" {
			t.Errorf("Expected ECO status to remain 'approved' after rollback, got '%s'", status)
		}

		// Verify BOM was not changed
		var bomQty float64
		err = testDB.QueryRow("SELECT qty FROM bom WHERE assembly_ipn = ?", "ASM-100").Scan(&bomQty)
		if err != nil {
			t.Fatalf("Failed to query BOM: %v", err)
		}
		if bomQty != 5.0 {
			t.Errorf("Expected BOM qty to remain 5 after rollback, got %f", bomQty)
		}
	})

	t.Run("SuccessfulECOImplementation_ShouldCommitAll", func(t *testing.T) {
		tx, err := testDB.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Update ECO status
		_, err = tx.Exec("UPDATE ecos SET status = 'implemented', updated_at = datetime('now') WHERE id = ?", "ECO001")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to update ECO: %v", err)
		}

		// Update BOM - replace component
		_, err = tx.Exec("DELETE FROM bom WHERE assembly_ipn = ? AND component_ipn = ?", "ASM-100", "COMP-OLD")
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to delete old BOM line: %v", err)
		}

		_, err = tx.Exec("INSERT INTO bom (assembly_ipn, component_ipn, qty) VALUES (?, ?, ?)", "ASM-100", "COMP-NEW", 3)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert new BOM line: %v", err)
		}

		if err = tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Verify changes were committed
		var status string
		err = testDB.QueryRow("SELECT status FROM ecos WHERE id = ?", "ECO001").Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query ECO: %v", err)
		}
		if status != "implemented" {
			t.Errorf("Expected ECO status 'implemented', got '%s'", status)
		}

		var count int
		err = testDB.QueryRow("SELECT COUNT(*) FROM bom WHERE assembly_ipn = ? AND component_ipn = ?", "ASM-100", "COMP-OLD").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count old BOM: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected old BOM line to be deleted, found %d", count)
		}

		var newQty float64
		err = testDB.QueryRow("SELECT qty FROM bom WHERE assembly_ipn = ? AND component_ipn = ?", "ASM-100", "COMP-NEW").Scan(&newQty)
		if err != nil {
			t.Fatalf("Failed to query new BOM: %v", err)
		}
		if newQty != 3.0 {
			t.Errorf("Expected new BOM qty = 3, got %f", newQty)
		}
	})
}

// TestMultiTableOperationRollback tests complex multi-table operations
func TestMultiTableOperationRollback(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	t.Run("ForeignKeyViolation_ShouldRollback", func(t *testing.T) {
		// Perform the failing transaction
		func() {
			tx, err := testDB.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}
			defer tx.Rollback()

			// Try to create PO line for non-existent PO (violates foreign key)
			_, err = tx.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered) VALUES (?, ?, ?)", "PO999", "PART-001", 100)
			if err == nil {
				t.Fatal("Expected foreign key violation, but insert succeeded")
			}
		}()

		// Verify no data was inserted (after rollback completes)
		var count int
		err := testDB.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id = ?", "PO999").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count po_lines: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected no po_lines after FK violation, found %d", count)
		}
	})

	t.Run("ConstraintViolation_MidTransaction_ShouldRollback", func(t *testing.T) {
		// Create vendor first
		_, err := testDB.Exec("INSERT INTO vendors (id, name) VALUES ('V100', 'Test Vendor')")
		if err != nil {
			t.Fatalf("Failed to create vendor: %v", err)
		}

		// Perform the failing transaction
		func() {
			tx, err := testDB.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}
			defer tx.Rollback()

			// Create PO
			_, err = tx.Exec("INSERT INTO purchase_orders (id, vendor_id, status) VALUES ('PO100', 'V100', 'draft')")
			if err != nil {
				t.Fatalf("Failed to create PO: %v", err)
			}

			// Create first PO line - should succeed
			_, err = tx.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered) VALUES ('PO100', 'PART-A', 10)")
			if err != nil {
				t.Fatalf("Failed to create first PO line: %v", err)
			}

			// Create second PO line with invalid qty (violates CHECK qty_ordered > 0)
			_, err = tx.Exec("INSERT INTO po_lines (po_id, ipn, qty_ordered) VALUES ('PO100', 'PART-B', 0)")
			if err == nil {
				t.Fatal("Expected CHECK constraint violation for qty_ordered = 0")
			}

			// Don't commit - let rollback happen
		}()

		// Verify entire transaction was rolled back (after rollback completes)
		var poCount int
		err = testDB.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE id = ?", "PO100").Scan(&poCount)
		if err != nil {
			t.Fatalf("Failed to count POs: %v", err)
		}
		if poCount != 0 {
			t.Errorf("Expected PO to be rolled back, found %d", poCount)
		}

		var lineCount int
		err = testDB.QueryRow("SELECT COUNT(*) FROM po_lines WHERE po_id = ?", "PO100").Scan(&lineCount)
		if err != nil {
			t.Fatalf("Failed to count PO lines: %v", err)
		}
		if lineCount != 0 {
			t.Errorf("Expected all PO lines to be rolled back, found %d", lineCount)
		}
	})
}

// TestDataIntegrityUnderFailure tests that database maintains integrity even when operations fail
func TestDataIntegrityUnderFailure(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	// Setup initial data
	_, err := testDB.Exec("INSERT INTO inventory (ipn, qty_on_hand, qty_reserved) VALUES ('PART-001', 100, 0)")
	if err != nil {
		t.Fatalf("Failed to create inventory: %v", err)
	}

	t.Run("RepeatedFailures_ShouldNotCorruptData", func(t *testing.T) {
		// Attempt multiple failing transactions - data should remain unchanged
		for i := 0; i < 5; i++ {
			tx, err := testDB.Begin()
			if err != nil {
				t.Fatalf("Failed to begin transaction: %v", err)
			}

			// Try to set invalid negative inventory
			_, err = tx.Exec("UPDATE inventory SET qty_on_hand = ? WHERE ipn = ?", -50.0, "PART-001")
			if err == nil {
				tx.Rollback()
				t.Fatal("Expected CHECK constraint violation")
			}
			tx.Rollback()
		}

		// Verify original data is intact
		var qty float64
		err := testDB.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "PART-001").Scan(&qty)
		if err != nil {
			t.Fatalf("Failed to query inventory: %v", err)
		}
		if qty != 100.0 {
			t.Errorf("Expected qty_on_hand = 100 after failed transactions, got %f", qty)
		}
	})

	t.Run("PartialCommit_NotPossible", func(t *testing.T) {
		tx, err := testDB.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		// Make valid update
		_, err = tx.Exec("UPDATE inventory SET qty_reserved = ? WHERE ipn = ?", 20.0, "PART-001")
		if err != nil {
			t.Fatalf("Failed to update: %v", err)
		}

		// Try to make invalid update
		_, err = tx.Exec("UPDATE inventory SET qty_on_hand = ? WHERE ipn = ?", -10.0, "PART-001")
		if err == nil {
			t.Fatal("Expected CHECK constraint violation")
		}

		// Rollback
		tx.Rollback()

		// Verify even the valid update was rolled back
		var reserved float64
		err = testDB.QueryRow("SELECT qty_reserved FROM inventory WHERE ipn = ?", "PART-001").Scan(&reserved)
		if err != nil {
			t.Fatalf("Failed to query inventory: %v", err)
		}
		if reserved != 0 {
			t.Errorf("Expected qty_reserved = 0 after rollback, got %f", reserved)
		}
	})
}

// TestConcurrentTransactionIsolation tests that concurrent transactions don't interfere with rollback
func TestConcurrentTransactionIsolation(t *testing.T) {
	testDB := setupTransactionTestDB(t)
	defer testDB.Close()

	// Setup
	_, err := testDB.Exec("INSERT INTO inventory (ipn, qty_on_hand) VALUES ('PART-001', 100)")
	if err != nil {
		t.Fatalf("Failed to create inventory: %v", err)
	}

	t.Run("IsolatedRollback_DoesNotAffectOtherTransactions", func(t *testing.T) {
		// Transaction 1: Will fail and rollback
		func() {
			tx1, err := testDB.Begin()
			if err != nil {
				t.Fatalf("Failed to begin tx1: %v", err)
			}
			defer tx1.Rollback()

			_, err = tx1.Exec("UPDATE inventory SET qty_on_hand = ? WHERE ipn = ?", -50.0, "PART-001")
			if err == nil {
				t.Fatal("Expected CHECK constraint violation in tx1")
			}
		}()

		// Transaction 2: Should succeed independently
		tx2, err := testDB.Begin()
		if err != nil {
			t.Fatalf("Failed to begin tx2: %v", err)
		}

		_, err = tx2.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand + ? WHERE ipn = ?", 50.0, "PART-001")
		if err != nil {
			tx2.Rollback()
			t.Fatalf("tx2 update failed: %v", err)
		}

		if err = tx2.Commit(); err != nil {
			t.Fatalf("tx2 commit failed: %v", err)
		}

		// Verify tx2 changes committed despite tx1 failure
		var qty float64
		err = testDB.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", "PART-001").Scan(&qty)
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}
		if qty != 150.0 {
			t.Errorf("Expected qty = 150 (tx2 committed), got %f", qty)
		}
	})
}
