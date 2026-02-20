package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// TestLoadLargeDatasets is the main load testing suite for large datasets
func TestLoadLargeDatasets(t *testing.T) {
	t.Run("Search Performance with 10k+ Parts", testSearchPerformance10k)
	t.Run("Pagination with Large Result Sets", testPaginationLargeResults)
	t.Run("Work Order with 1000+ Serial Numbers", testLargeWorkOrder)
	t.Run("CSV Export 10k Records", testCSVExport10k)
	t.Run("Dashboard with Large Data Volumes", testDashboardLargeData)
}

// testSearchPerformance10k tests search/filter performance with 10,000+ parts
// Target: response time < 500ms
func testSearchPerformance10k(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	const numParts = 10000
	t.Logf("Inserting %d parts for search testing...", numParts)

	// Batch insert 10,000 parts
	start := time.Now()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Create parts table first
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS parts (
		ipn TEXT PRIMARY KEY,
		category TEXT DEFAULT '',
		description TEXT DEFAULT '',
		mpn TEXT DEFAULT '',
		manufacturer TEXT DEFAULT '',
		lifecycle TEXT DEFAULT 'active',
		notes TEXT DEFAULT '',
		bom_cost REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO parts (ipn, category, description, mpn, manufacturer, lifecycle) 
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	categories := []string{"Resistors", "Capacitors", "ICs", "Connectors", "Transistors"}
	manufacturers := []string{"TI", "Analog", "NXP", "Infineon", "ST"}

	for i := 0; i < numParts; i++ {
		ipn := fmt.Sprintf("PART-%06d", i)
		category := categories[i%len(categories)]
		desc := fmt.Sprintf("Test component %d for load testing", i)
		mpn := fmt.Sprintf("MPN-%06d", i)
		manufacturer := manufacturers[i%len(manufacturers)]
		lifecycle := "active"

		_, err = stmt.Exec(ipn, category, desc, mpn, manufacturer, lifecycle)
		if err != nil {
			t.Fatalf("Failed to insert part %d: %v", i, err)
		}
	}

	if err = tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	insertTime := time.Since(start)
	t.Logf("✓ Inserted %d parts in %v (%.0f parts/sec)",
		numParts, insertTime, float64(numParts)/insertTime.Seconds())

	// Create indexes for performance
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_parts_ipn ON parts(ipn)`)
	if err != nil {
		t.Logf("Warning: Failed to create index on ipn: %v", err)
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_parts_category ON parts(category)`)
	if err != nil {
		t.Logf("Warning: Failed to create index on category: %v", err)
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_parts_description ON parts(description)`)
	if err != nil {
		t.Logf("Warning: Failed to create index on description: %v", err)
	}

	// Test various search scenarios
	searchTests := []struct {
		name        string
		searchTerm  string
		category    string
		maxDuration time.Duration
	}{
		{"Exact IPN Match", "PART-005000", "", 100 * time.Millisecond},
		{"IPN Prefix Search", "PART-00", "", 500 * time.Millisecond},
		{"Description Search", "component 5000", "", 500 * time.Millisecond},
		{"Category Filter", "", "Resistors", 500 * time.Millisecond},
		{"Combined Search", "Test", "ICs", 500 * time.Millisecond},
		{"Wildcard Search", "MPN-00%", "", 500 * time.Millisecond},
	}

	for _, tt := range searchTests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			query := "SELECT ipn, category, description, mpn, manufacturer FROM parts WHERE 1=1"
			var args []interface{}

			if tt.searchTerm != "" {
				query += " AND (ipn LIKE ? OR description LIKE ? OR mpn LIKE ?)"
				searchPattern := "%" + tt.searchTerm + "%"
				args = append(args, searchPattern, searchPattern, searchPattern)
			}

			if tt.category != "" {
				query += " AND category = ?"
				args = append(args, tt.category)
			}

			query += " LIMIT 100"

			rows, err := db.Query(query, args...)
			if err != nil {
				t.Fatalf("Search query failed: %v", err)
			}

			count := 0
			for rows.Next() {
				count++
				var ipn, category, desc, mpn, mfr string
				rows.Scan(&ipn, &category, &desc, &mpn, &mfr)
			}
			rows.Close()

			elapsed := time.Since(start)

			if elapsed > tt.maxDuration {
				t.Errorf("❌ Search took %v (target: <%v)", elapsed, tt.maxDuration)
			} else {
				t.Logf("✓ Search completed in %v, returned %d results (target: <%v)",
					elapsed, count, tt.maxDuration)
			}
		})
	}
}

// testPaginationLargeResults tests pagination with large result sets
// Target: smooth scrolling, consistent performance
func testPaginationLargeResults(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	const numParts = 10000
	const pageSize = 50
	const numPages = 20 // Test first 20 pages (1000 records)

	t.Logf("Setting up %d parts for pagination testing...", numParts)

	// Reuse parts from inventory table
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO inventory (ipn, qty_on_hand, description, mpn) VALUES (?, ?, ?, ?)`)
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for i := 0; i < numParts; i++ {
		ipn := fmt.Sprintf("INV-%06d", i)
		desc := fmt.Sprintf("Inventory item %d", i)
		mpn := fmt.Sprintf("MPN-%06d", i)
		qty := float64(i % 1000)
		_, err = stmt.Exec(ipn, qty, desc, mpn)
		if err != nil {
			t.Fatalf("Failed to insert inventory item %d: %v", i, err)
		}
	}

	if err = tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	t.Logf("✓ Inserted %d inventory items", numParts)

	// Test pagination performance
	var totalTime time.Duration
	maxPageTime := 100 * time.Millisecond // Each page should load in <100ms

	for page := 0; page < numPages; page++ {
		offset := page * pageSize
		start := time.Now()

		rows, err := db.Query(`
			SELECT ipn, qty_on_hand, description, mpn 
			FROM inventory 
			ORDER BY ipn 
			LIMIT ? OFFSET ?`, pageSize, offset)

		if err != nil {
			t.Fatalf("Pagination query failed at page %d: %v", page, err)
		}

		count := 0
		for rows.Next() {
			count++
			var ipn, desc, mpn string
			var qty float64
			rows.Scan(&ipn, &qty, &desc, &mpn)
		}
		rows.Close()

		elapsed := time.Since(start)
		totalTime += elapsed

		if count != pageSize {
			t.Errorf("Page %d returned %d items, expected %d", page, count, pageSize)
		}

		if elapsed > maxPageTime {
			t.Errorf("❌ Page %d took %v (target: <%v)", page, elapsed, maxPageTime)
		} else if page%5 == 0 {
			t.Logf("✓ Page %d loaded in %v", page, elapsed)
		}
	}

	avgTime := totalTime / time.Duration(numPages)
	t.Logf("✓ Pagination test complete: %d pages, avg %v per page, total %v",
		numPages, avgTime, totalTime)

	if avgTime > maxPageTime {
		t.Errorf("❌ Average page load time %v exceeds target %v", avgTime, maxPageTime)
	}
}

// testLargeWorkOrder tests work order with 1000+ serial numbers (simulating large BOM)
// Target: loads without timeout
func testLargeWorkOrder(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	const numSerials = 1000
	woID := "WO-LARGE-001"
	assemblyIPN := "ASSEMBLY-001"

	t.Logf("Creating work order with %d serial numbers...", numSerials)

	// Create the work order
	start := time.Now()
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, priority) 
		VALUES (?, ?, ?, 'in_progress', 'high')`,
		woID, assemblyIPN, numSerials)
	if err != nil {
		t.Fatalf("Failed to create work order: %v", err)
	}

	// Batch insert serial numbers
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO wo_serials (wo_id, serial_number, status) VALUES (?, ?, 'building')`)
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for i := 0; i < numSerials; i++ {
		serial := fmt.Sprintf("SN-%s-%06d", woID, i)
		_, err = stmt.Exec(woID, serial)
		if err != nil {
			t.Fatalf("Failed to insert serial %d: %v", i, err)
		}
	}

	if err = tx.Commit(); err != nil {
		t.Fatalf("Failed to commit serials: %v", err)
	}

	insertTime := time.Since(start)
	t.Logf("✓ Created work order with %d serials in %v", numSerials, insertTime)

	// Test querying the work order with all serials
	queryStart := time.Now()
	rows, err := db.Query(`
		SELECT wo.id, wo.assembly_ipn, wo.qty, wo.status,
		       GROUP_CONCAT(ws.serial_number) as serials
		FROM work_orders wo
		LEFT JOIN wo_serials ws ON wo.id = ws.wo_id
		WHERE wo.id = ?
		GROUP BY wo.id`, woID)
	if err != nil {
		t.Fatalf("Failed to query work order: %v", err)
	}

	if rows.Next() {
		var id, ipn, status, serials string
		var qty int
		rows.Scan(&id, &ipn, &qty, &status, &serials)
		queryTime := time.Since(queryStart)

		if queryTime > 2*time.Second {
			t.Errorf("❌ Query took %v (target: <2s for reasonable performance)", queryTime)
		} else {
			t.Logf("✓ Retrieved work order with %d serials in %v", numSerials, queryTime)
		}
	}
	rows.Close()

	// Test pagination of serials
	const pageSize = 100
	paginationStart := time.Now()
	rows, err = db.Query(`
		SELECT serial_number, status 
		FROM wo_serials 
		WHERE wo_id = ? 
		ORDER BY serial_number 
		LIMIT ?`, woID, pageSize)
	if err != nil {
		t.Fatalf("Failed to paginate serials: %v", err)
	}

	count := 0
	for rows.Next() {
		count++
		var serial, status string
		rows.Scan(&serial, &status)
	}
	rows.Close()

	paginationTime := time.Since(paginationStart)
	if paginationTime > 200*time.Millisecond {
		t.Errorf("❌ Pagination took %v (target: <200ms)", paginationTime)
	} else {
		t.Logf("✓ Paginated first %d serials in %v", count, paginationTime)
	}
}

// testCSVExport10k tests CSV export of 10,000 records
// Target: completes in reasonable time (<5 seconds)
func testCSVExport10k(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	const numParts = 10000
	t.Logf("Setting up %d parts for CSV export test...", numParts)

	// Insert test data
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Ensure parts table exists
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS parts (
		ipn TEXT PRIMARY KEY,
		category TEXT DEFAULT '',
		description TEXT DEFAULT '',
		mpn TEXT DEFAULT '',
		manufacturer TEXT DEFAULT '',
		lifecycle TEXT DEFAULT 'active',
		notes TEXT DEFAULT ''
	)`)
	if err != nil {
		t.Fatalf("Failed to create parts table: %v", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO parts (ipn, category, description, mpn, manufacturer, lifecycle, notes) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for i := 0; i < numParts; i++ {
		ipn := fmt.Sprintf("EXPORT-%06d", i)
		category := "Test Category"
		desc := fmt.Sprintf("Export test part %d with some description text", i)
		mpn := fmt.Sprintf("MPN-EXPORT-%06d", i)
		mfr := "Test Manufacturer"
		lifecycle := "active"
		notes := "Test notes for export"

		_, err = stmt.Exec(ipn, category, desc, mpn, mfr, lifecycle, notes)
		if err != nil {
			t.Fatalf("Failed to insert part %d: %v", i, err)
		}
	}

	if err = tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	t.Logf("✓ Inserted %d parts for export", numParts)

	// Simulate the export handler
	w := httptest.NewRecorder()

	start := time.Now()

	// Export logic (simplified version of handleExportParts)
	rows, err := db.Query(`
		SELECT ipn, COALESCE(category,''), COALESCE(description,''), 
		       COALESCE(mpn,''), COALESCE(manufacturer,''), 
		       lifecycle, COALESCE(notes,'')
		FROM parts 
		ORDER BY ipn`)
	if err != nil {
		t.Fatalf("Export query failed: %v", err)
	}
	defer rows.Close()

	csvWriter := csv.NewWriter(w)
	csvWriter.Write([]string{"IPN", "Category", "Description", "MPN", "Manufacturer", "Lifecycle", "Notes"})

	exportCount := 0
	for rows.Next() {
		var ipn, category, desc, mpn, mfr, lifecycle, notes string
		if err := rows.Scan(&ipn, &category, &desc, &mpn, &mfr, &lifecycle, &notes); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		csvWriter.Write([]string{ipn, category, desc, mpn, mfr, lifecycle, notes})
		exportCount++
	}
	csvWriter.Flush()

	exportTime := time.Since(start)

	if exportCount != numParts {
		t.Errorf("Exported %d parts, expected %d", exportCount, numParts)
	}

	maxExportTime := 5 * time.Second
	if exportTime > maxExportTime {
		t.Errorf("❌ CSV export took %v (target: <%v)", exportTime, maxExportTime)
	} else {
		t.Logf("✓ Exported %d parts to CSV in %v (%.0f parts/sec)",
			exportCount, exportTime, float64(exportCount)/exportTime.Seconds())
	}

	// Verify CSV content size
	csvBytes := w.Body.Bytes()
	t.Logf("✓ CSV size: %.2f MB", float64(len(csvBytes))/(1024*1024))
}

// testDashboardLargeData tests dashboard performance with large data volumes
// Target: renders quickly (<1 second for aggregations)
func testDashboardLargeData(t *testing.T) {
	cleanup := freshTestDB(t)
	defer cleanup()

	const numInventory = 10000
	const numPOs = 1000
	const numWOs = 500

	t.Logf("Setting up large dataset for dashboard testing...")

	// Insert inventory items
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO inventory (ipn, qty_on_hand, description, location) VALUES (?, ?, ?, ?)`)
	if err != nil {
		t.Fatalf("Failed to prepare inventory statement: %v", err)
	}

	locations := []string{"A1", "A2", "B1", "B2", "C1"}
	for i := 0; i < numInventory; i++ {
		ipn := fmt.Sprintf("DASH-INV-%06d", i)
		qty := float64(i % 500)
		desc := fmt.Sprintf("Dashboard test item %d", i)
		location := locations[i%len(locations)]
		_, err = stmt.Exec(ipn, qty, desc, location)
		if err != nil {
			t.Fatalf("Failed to insert inventory %d: %v", i, err)
		}
	}
	stmt.Close()

	// Insert vendors for POs
	_, err = tx.Exec(`INSERT INTO vendors (id, name) VALUES ('VENDOR-001', 'Test Vendor')`)
	if err != nil {
		t.Logf("Vendor insert warning: %v", err)
	}

	// Insert purchase orders
	stmt, err = tx.Prepare(`INSERT INTO purchase_orders (id, vendor_id, status, total) VALUES (?, 'VENDOR-001', ?, ?)`)
	if err != nil {
		t.Fatalf("Failed to prepare PO statement: %v", err)
	}

	statuses := []string{"draft", "sent", "partial", "received"}
	for i := 0; i < numPOs; i++ {
		poID := fmt.Sprintf("PO-DASH-%06d", i)
		status := statuses[i%len(statuses)]
		total := float64(i*100 + 1000)
		_, err = stmt.Exec(poID, status, total)
		if err != nil {
			t.Fatalf("Failed to insert PO %d: %v", i, err)
		}
	}
	stmt.Close()

	// Insert work orders
	stmt, err = tx.Prepare(`INSERT INTO work_orders (id, assembly_ipn, qty, status) VALUES (?, ?, ?, ?)`)
	if err != nil {
		t.Fatalf("Failed to prepare WO statement: %v", err)
	}

	woStatuses := []string{"open", "in_progress", "completed"}
	for i := 0; i < numWOs; i++ {
		woID := fmt.Sprintf("WO-DASH-%06d", i)
		assemblyIPN := fmt.Sprintf("ASSEMBLY-%03d", i%100)
		qty := i%50 + 1
		status := woStatuses[i%len(woStatuses)]
		_, err = stmt.Exec(woID, assemblyIPN, qty, status)
		if err != nil {
			t.Fatalf("Failed to insert WO %d: %v", i, err)
		}
	}
	stmt.Close()

	if err = tx.Commit(); err != nil {
		t.Fatalf("Failed to commit dashboard data: %v", err)
	}

	t.Logf("✓ Inserted dashboard test data: %d inventory, %d POs, %d WOs",
		numInventory, numPOs, numWOs)

	// Test dashboard aggregation queries
	dashboardTests := []struct {
		name        string
		query       string
		maxDuration time.Duration
	}{
		{
			"Total Inventory Value",
			"SELECT COUNT(*), SUM(qty_on_hand) FROM inventory",
			500 * time.Millisecond,
		},
		{
			"Low Stock Count",
			"SELECT COUNT(*) FROM inventory WHERE qty_on_hand < 10",
			300 * time.Millisecond,
		},
		{
			"PO Status Summary",
			"SELECT status, COUNT(*), SUM(total) FROM purchase_orders GROUP BY status",
			300 * time.Millisecond,
		},
		{
			"WO Status Summary",
			"SELECT status, COUNT(*), SUM(qty) FROM work_orders GROUP BY status",
			300 * time.Millisecond,
		},
		{
			"Inventory by Location",
			"SELECT location, COUNT(*), SUM(qty_on_hand) FROM inventory GROUP BY location",
			400 * time.Millisecond,
		},
	}

	for _, tt := range dashboardTests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			rows, err := db.Query(tt.query)
			if err != nil {
				t.Fatalf("Dashboard query failed: %v", err)
			}

			// Process results
			for rows.Next() {
				// Scan into interface{} to handle variable column counts
				cols, _ := rows.Columns()
				vals := make([]interface{}, len(cols))
				valPtrs := make([]interface{}, len(cols))
				for i := range vals {
					valPtrs[i] = &vals[i]
				}
				rows.Scan(valPtrs...)
			}
			rows.Close()

			elapsed := time.Since(start)

			if elapsed > tt.maxDuration {
				t.Errorf("❌ Dashboard query '%s' took %v (target: <%v)",
					tt.name, elapsed, tt.maxDuration)
			} else {
				t.Logf("✓ Dashboard query '%s' completed in %v", tt.name, elapsed)
			}
		})
	}

	// Test complex dashboard API endpoint simulation
	t.Run("Dashboard API Response", func(t *testing.T) {
		start := time.Now()

		// Simulate fetching multiple metrics in one request
		var inventoryCount, lowStockCount int
		var totalInventoryValue float64

		db.QueryRow("SELECT COUNT(*), SUM(qty_on_hand) FROM inventory").Scan(&inventoryCount, &totalInventoryValue)
		db.QueryRow("SELECT COUNT(*) FROM inventory WHERE qty_on_hand < 10").Scan(&lowStockCount)

		var poCount int
		var poTotal float64
		db.QueryRow("SELECT COUNT(*), SUM(total) FROM purchase_orders WHERE status != 'received'").Scan(&poCount, &poTotal)

		var woCount int
		db.QueryRow("SELECT COUNT(*) FROM work_orders WHERE status != 'completed'").Scan(&woCount)

		// Build response
		response := map[string]interface{}{
			"inventory": map[string]interface{}{
				"total_items":  inventoryCount,
				"total_value":  totalInventoryValue,
				"low_stock":    lowStockCount,
			},
			"purchase_orders": map[string]interface{}{
				"open_count": poCount,
				"open_value": poTotal,
			},
			"work_orders": map[string]interface{}{
				"open_count": woCount,
			},
		}

		_, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal dashboard response: %v", err)
		}

		elapsed := time.Since(start)
		maxDashboardTime := 1 * time.Second

		if elapsed > maxDashboardTime {
			t.Errorf("❌ Dashboard API response took %v (target: <%v)", elapsed, maxDashboardTime)
		} else {
			t.Logf("✓ Dashboard API response generated in %v", elapsed)
		}
	})
}
