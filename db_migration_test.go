package main

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// TestDatabaseMigrations verifies that all migrations run successfully
// and that the schema is consistent with what the handlers expect
func TestDatabaseMigrations(t *testing.T) {
	// Create a fresh in-memory database
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer testDB.Close()

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run all migrations
	oldDB := db
	db = testDB
	err = runMigrations()
	db = oldDB

	if err != nil {
		t.Fatalf("Migrations failed: %v", err)
	}

	// Verify critical tables exist
	requiredTables := []string{
		"users", "sessions", "audit_log",
		"ecos", "documents", "vendors", "inventory",
		"inventory_transactions", "purchase_orders", "po_lines",
		"work_orders", "wo_serials", "test_records",
		"ncrs", "devices", "firmware_campaigns", "campaign_devices",
		"rmas", "quotes", "quote_lines",
		"change_history", "undo_log", "api_keys",
		"attachments", "notifications", "price_history",
		"email_config", "email_log", "email_subscriptions",
		"eco_revisions", "dashboard_widgets",
		"receiving_inspections", "rfqs", "rfq_vendors",
		"rfq_lines", "rfq_quotes", "product_pricing",
		"cost_analysis", "app_settings", "document_versions",
		"market_pricing", "capas", "part_changes",
		"sales_orders", "sales_order_lines",
		"invoices", "invoice_lines",
		"shipments", "shipment_lines", "pack_lists",
		"field_reports",
	}

	for _, table := range requiredTables {
		exists, err := tableExists(testDB, table)
		if err != nil {
			t.Errorf("Error checking table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("Required table %s does not exist", table)
		}
	}

	// Verify critical columns that have caused bugs in the past
	criticalColumns := []struct {
		table  string
		column string
	}{
		{"work_orders", "qty_good"},   // This was the bug we hit!
		{"work_orders", "qty_scrap"},  // Also added in same migration
		{"inventory", "description"},
		{"inventory", "mpn"},
		{"users", "active"},
		{"users", "email"},
		{"ecos", "ncr_id"},
		{"notifications", "emailed"},
		{"purchase_orders", "created_by"},
		{"shipment_lines", "sales_order_id"},
		{"invoices", "invoice_number"},
		{"invoices", "issue_date"},
		{"invoices", "tax"},
		{"invoices", "notes"},
		{"invoices", "total"}, // Renamed from total_amount
		{"ncrs", "created_by"},
		{"email_log", "event_type"},
	}

	for _, col := range criticalColumns {
		exists, err := columnExists(testDB, col.table, col.column)
		if err != nil {
			t.Errorf("Error checking column %s.%s: %v", col.table, col.column, err)
		}
		if !exists {
			t.Errorf("Critical column %s.%s does not exist (missing migration?)", col.table, col.column)
		}
	}

	t.Logf("✓ All %d required tables exist", len(requiredTables))
	t.Logf("✓ All %d critical columns exist", len(criticalColumns))
}

// TestHandlerQueryConsistency verifies that columns referenced in handler code
// actually exist in the database schema (catches the qty_good bug pattern)
func TestHandlerQueryConsistency(t *testing.T) {
	// Create a fresh in-memory database
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer testDB.Close()

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run all migrations
	oldDB := db
	db = testDB
	err = runMigrations()
	db = oldDB
	if err != nil {
		t.Fatalf("Migrations failed: %v", err)
	}

	// Known queries that handlers execute - verify they don't reference missing columns
	// This is where the qty_good bug would have been caught!
	testQueries := []struct {
		name  string
		query string
	}{
		{
			"List Work Orders - with qty_good",
			"SELECT id, assembly_ipn, qty, qty_good, qty_scrap, status, priority FROM work_orders LIMIT 1",
		},
		{
			"Get Work Order - with qty_good",
			"SELECT id, assembly_ipn, qty, qty_good, qty_scrap FROM work_orders WHERE id = 'WO-TEST'",
		},
		{
			"List Inventory - with description and mpn",
			"SELECT ipn, qty_on_hand, qty_reserved, location, description, mpn FROM inventory LIMIT 1",
		},
		{
			"List Users - with active and email",
			"SELECT id, username, display_name, role, active, email FROM users LIMIT 1",
		},
		{
			"List ECOs - with ncr_id",
			"SELECT id, title, description, status, created_by, ncr_id FROM ecos LIMIT 1",
		},
		{
			"List Notifications - with emailed",
			"SELECT id, user_id, type, message, read, emailed FROM notifications LIMIT 1",
		},
		{
			"List POs - with created_by",
			"SELECT id, vendor_id, status, total, created_by FROM purchase_orders LIMIT 1",
		},
		{
			"List Invoices - with new columns",
			"SELECT id, sales_order_id, total, status, invoice_number, issue_date, tax, notes FROM invoices LIMIT 1",
		},
		{
			"List NCRs - with created_by",
			"SELECT id, title, description, status, severity, created_by FROM ncrs LIMIT 1",
		},
		{
			"Shipment Lines - with sales_order_id",
			"SELECT id, shipment_id, ipn, qty, sales_order_id FROM shipment_lines LIMIT 1",
		},
		{
			"Email Log - with event_type",
			"SELECT id, recipient, subject, event_type FROM email_log LIMIT 1",
		},
	}

	passCount := 0
	for _, test := range testQueries {
		// Try to execute the query - if a column doesn't exist, this will fail
		_, err := testDB.Query(test.query)
		if err != nil {
			if strings.Contains(err.Error(), "no such column") {
				t.Errorf("[%s] FAILED - references missing column: %v", test.name, err)
			} else if strings.Contains(err.Error(), "no such table") {
				t.Errorf("[%s] FAILED - table doesn't exist: %v", test.name, err)
			} else {
				// Other errors (like no rows) are fine - we're just checking schema
				passCount++
			}
		} else {
			passCount++
		}
	}

	t.Logf("✓ %d/%d handler queries verified", passCount, len(testQueries))
	if passCount != len(testQueries) {
		t.Fatalf("Some queries failed - see errors above")
	}
}

// TestMigrationIdempotency verifies that running migrations multiple times is safe
func TestMigrationIdempotency(t *testing.T) {
	// Create a fresh in-memory database
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer testDB.Close()

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	oldDB := db
	db = testDB

	// Run migrations three times
	for i := 1; i <= 3; i++ {
		err = runMigrations()
		if err != nil {
			db = oldDB
			t.Fatalf("Migrations failed on run %d: %v", i, err)
		}
	}

	db = oldDB

	// Verify the database is still healthy
	exists, err := tableExists(testDB, "work_orders")
	if err != nil || !exists {
		t.Error("work_orders table missing after multiple migration runs")
	}

	exists, err = columnExists(testDB, "work_orders", "qty_good")
	if err != nil || !exists {
		t.Error("qty_good column missing after multiple migration runs")
	}

	t.Log("✓ Migrations are idempotent")
}

// tableExists checks if a table exists in the database
func tableExists(db *sql.DB, tableName string) (bool, error) {
	var name string
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
	err := db.QueryRow(query, tableName).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// columnExists checks if a column exists in a table
func columnExists(db *sql.DB, tableName, columnName string) (bool, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}

		err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk)
		if err != nil {
			return false, err
		}

		if strings.EqualFold(name, columnName) {
			return true, nil
		}
	}

	return false, nil
}

// TestForeignKeyConstraints verifies that foreign key constraints are properly set up
func TestForeignKeyConstraints(t *testing.T) {
	// Create a fresh in-memory database
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}
	defer testDB.Close()

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	err = testDB.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("Foreign keys are not enabled")
	}

	// Run migrations
	oldDB := db
	db = testDB
	err = runMigrations()
	db = oldDB

	if err != nil {
		t.Fatalf("Migrations failed: %v", err)
	}

	// Verify some critical foreign keys exist
	checkFK := func(table string, expectedFK bool) {
		query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", table)
		rows, err := testDB.Query(query)
		if err != nil {
			t.Errorf("Failed to check foreign keys for %s: %v", table, err)
			return
		}
		defer rows.Close()

		hasFKs := rows.Next()
		if expectedFK && !hasFKs {
			t.Errorf("Table %s should have foreign keys but doesn't", table)
		}
	}

	// Tables that should have foreign keys
	checkFK("sessions", true)      // user_id -> users
	checkFK("po_lines", true)      // po_id -> purchase_orders
	checkFK("wo_serials", true)    // wo_id -> work_orders
	checkFK("campaign_devices", true) // campaign_id -> firmware_campaigns

	t.Log("✓ Foreign key constraints verified")
}
