package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

func setupConcurrencyTestDB(t *testing.T) *sql.DB {
	// Use shared cache mode with WAL to properly test concurrency
	testDB, err := sql.Open("sqlite", "file::memory:?mode=memory&cache=shared&_journal_mode=WAL&_busy_timeout=10000&_foreign_keys=1")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Configure connection pool like production
	testDB.SetMaxOpenConns(10)
	testDB.SetMaxIdleConns(5)
	testDB.SetConnMaxLifetime(0)

	// Explicitly enable WAL mode
	if _, err := testDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("Failed to enable WAL mode: %v", err)
	}

	// Set busy timeout
	if _, err := testDB.Exec("PRAGMA busy_timeout=30000"); err != nil {
		t.Fatalf("Failed to set busy_timeout: %v", err)
	}

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Drop tables if they exist (for shared cache mode)
	testDB.Exec("DROP TABLE IF EXISTS inventory")
	testDB.Exec("DROP TABLE IF EXISTS inventory_transactions")
	testDB.Exec("DROP TABLE IF EXISTS audit_log")

	// Create inventory table
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

	// Create inventory_transactions table
	_, err = testDB.Exec(`
		CREATE TABLE inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			type TEXT NOT NULL,
			qty REAL NOT NULL,
			reference TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory_transactions table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	return testDB
}

// TestConcurrentInventoryUpdates_TwoGoroutines tests two goroutines updating the same part simultaneously
func TestConcurrentInventoryUpdates_TwoGoroutines(t *testing.T) {
	oldDB := db
	oldPartsDir := partsDir
	db = setupConcurrencyTestDB(t)
	partsDir = "" // Disable parts enrichment for concurrency tests
	defer func() { 
		db.Close() 
		db = oldDB 
		partsDir = oldPartsDir
	}()

	// Create initial inventory with qty=100
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('IPN-CONCURRENT-1', 100)`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Add 25
	go func() {
		defer wg.Done()
		reqBody := `{
			"ipn": "IPN-CONCURRENT-1",
			"type": "receive",
			"qty": 25,
			"reference": "PO-1"
		}`
		req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()
		handleInventoryTransact(w, req)

		if w.Code != 200 {
			t.Errorf("Goroutine 1: Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	}()

	// Goroutine 2: Add 35
	go func() {
		defer wg.Done()
		reqBody := `{
			"ipn": "IPN-CONCURRENT-1",
			"type": "receive",
			"qty": 35,
			"reference": "PO-2"
		}`
		req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()
		handleInventoryTransact(w, req)

		if w.Code != 200 {
			t.Errorf("Goroutine 2: Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	}()

	wg.Wait()

	// Verify final quantity is accurate (100 + 25 + 35 = 160)
	var finalQty float64
	err = db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-CONCURRENT-1").Scan(&finalQty)
	if err != nil {
		t.Fatalf("Failed to query final quantity: %v", err)
	}

	expectedQty := 160.0
	if finalQty != expectedQty {
		t.Errorf("Expected final quantity %f, got %f (potential race condition - lost update)", expectedQty, finalQty)
	}

	// Verify both transactions were recorded
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn=?", "IPN-CONCURRENT-1").Scan(&txCount)
	if txCount != 2 {
		t.Errorf("Expected 2 transactions, got %d", txCount)
	}
}

// TestConcurrentInventoryUpdates_TenGoroutines tests 10 concurrent updates to verify proper serialization
func TestConcurrentInventoryUpdates_TenGoroutines(t *testing.T) {
	oldDB := db
	oldPartsDir := partsDir
	db = setupConcurrencyTestDB(t)
	partsDir = ""
	defer func() { 
		db.Close() 
		db = oldDB 
		partsDir = oldPartsDir
	}()

	// Create initial inventory with qty=100
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('IPN-CONCURRENT-10', 100)`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	qtyPerUpdate := 10.0

	wg.Add(numGoroutines)

	// Launch 10 goroutines, each adding +10
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			reqBody := `{
				"ipn": "IPN-CONCURRENT-10",
				"type": "receive",
				"qty": 10,
				"reference": "PO-` + string(rune('A'+idx)) + `"
			}`
			req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handleInventoryTransact(w, req)

			if w.Code != 200 {
				t.Errorf("Goroutine %d: Expected status 200, got %d: %s", idx, w.Code, w.Body.String())
			}
		}(i)
	}

	wg.Wait()

	// Verify final quantity is accurate (100 + 10*10 = 200)
	var finalQty float64
	err = db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-CONCURRENT-10").Scan(&finalQty)
	if err != nil {
		t.Fatalf("Failed to query final quantity: %v", err)
	}

	expectedQty := 100.0 + (qtyPerUpdate * float64(numGoroutines))
	if finalQty != expectedQty {
		t.Errorf("Expected final quantity %f, got %f (RACE CONDITION DETECTED - lost updates)", expectedQty, finalQty)
		t.Logf("Lost %f units due to race condition", expectedQty-finalQty)
	}

	// Verify all transactions were recorded
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn=?", "IPN-CONCURRENT-10").Scan(&txCount)
	if txCount != numGoroutines {
		t.Errorf("Expected %d transactions, got %d (some transactions were lost)", numGoroutines, txCount)
	}
}

// TestConcurrentInventoryUpdates_DifferentParts tests that concurrent updates to different parts should not block each other
func TestConcurrentInventoryUpdates_DifferentParts(t *testing.T) {
	oldDB := db
	oldPartsDir := partsDir
	db = setupConcurrencyTestDB(t)
	partsDir = ""
	defer func() { 
		db.Close() 
		db = oldDB 
		partsDir = oldPartsDir
	}()

	// Create two different parts
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES 
		('IPN-PART-A', 100),
		('IPN-PART-B', 200)
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Update Part A
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			reqBody := `{
				"ipn": "IPN-PART-A",
				"type": "receive",
				"qty": 10,
				"reference": "PO-A"
			}`
			req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handleInventoryTransact(w, req)

			if w.Code != 200 {
				t.Errorf("Part A update failed: %d: %s", w.Code, w.Body.String())
			}
		}
	}()

	// Goroutine 2: Update Part B
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			reqBody := `{
				"ipn": "IPN-PART-B",
				"type": "receive",
				"qty": 20,
				"reference": "PO-B"
			}`
			req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handleInventoryTransact(w, req)

			if w.Code != 200 {
				t.Errorf("Part B update failed: %d: %s", w.Code, w.Body.String())
			}
		}
	}()

	wg.Wait()

	// Verify both parts have correct final quantities
	var qtyA, qtyB float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-PART-A").Scan(&qtyA)
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-PART-B").Scan(&qtyB)

	expectedA := 100.0 + (10.0 * 5)
	expectedB := 200.0 + (20.0 * 5)

	if qtyA != expectedA {
		t.Errorf("Part A: Expected %f, got %f", expectedA, qtyA)
	}

	if qtyB != expectedB {
		t.Errorf("Part B: Expected %f, got %f", expectedB, qtyB)
	}

	// Verify transactions were recorded for both parts
	var txCountA, txCountB int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn=?", "IPN-PART-A").Scan(&txCountA)
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn=?", "IPN-PART-B").Scan(&txCountB)

	if txCountA != 5 {
		t.Errorf("Part A: Expected 5 transactions, got %d", txCountA)
	}
	if txCountB != 5 {
		t.Errorf("Part B: Expected 5 transactions, got %d", txCountB)
	}
}

// TestConcurrentInventoryUpdates_MixedOperations tests concurrent receive, issue, and adjust operations
func TestConcurrentInventoryUpdates_MixedOperations(t *testing.T) {
	oldDB := db
	oldPartsDir := partsDir
	db = setupConcurrencyTestDB(t)
	partsDir = ""
	defer func() { 
		db.Close() 
		db = oldDB 
		partsDir = oldPartsDir
	}()

	// Create initial inventory with qty=1000 (large enough to handle concurrent issues)
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('IPN-MIXED', 1000)`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(15)

	// 5 goroutines receiving +20 each
	for i := 0; i < 5; i++ {
		go func(idx int) {
			defer wg.Done()
			reqBody := `{
				"ipn": "IPN-MIXED",
				"type": "receive",
				"qty": 20,
				"reference": "RCV"
			}`
			req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handleInventoryTransact(w, req)
		}(i)
	}

	// 5 goroutines issuing -10 each
	for i := 0; i < 5; i++ {
		go func(idx int) {
			defer wg.Done()
			reqBody := `{
				"ipn": "IPN-MIXED",
				"type": "issue",
				"qty": 10,
				"reference": "ISS"
			}`
			req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handleInventoryTransact(w, req)
		}(i)
	}

	// 5 goroutines returning +5 each
	for i := 0; i < 5; i++ {
		go func(idx int) {
			defer wg.Done()
			reqBody := `{
				"ipn": "IPN-MIXED",
				"type": "return",
				"qty": 5,
				"reference": "RET"
			}`
			req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handleInventoryTransact(w, req)
		}(i)
	}

	wg.Wait()

	// Verify final quantity: 1000 + (5*20) - (5*10) + (5*5) = 1000 + 100 - 50 + 25 = 1075
	var finalQty float64
	err = db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-MIXED").Scan(&finalQty)
	if err != nil {
		t.Fatalf("Failed to query final quantity: %v", err)
	}

	expectedQty := 1075.0
	if finalQty != expectedQty {
		t.Errorf("Expected final quantity %f, got %f (RACE CONDITION in mixed operations)", expectedQty, finalQty)
	}

	// Verify all 15 transactions were recorded
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE ipn=?", "IPN-MIXED").Scan(&txCount)
	if txCount != 15 {
		t.Errorf("Expected 15 transactions, got %d", txCount)
	}
}

// TestConcurrentInventoryRead_WhileUpdating tests that reads are consistent during concurrent updates
func TestConcurrentInventoryRead_WhileUpdating(t *testing.T) {
	oldDB := db
	oldPartsDir := partsDir
	db = setupConcurrencyTestDB(t)
	partsDir = ""
	defer func() { 
		db.Close() 
		db = oldDB 
		partsDir = oldPartsDir
	}()

	// Create initial inventory
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand) VALUES ('IPN-READ-TEST', 500)`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var wg sync.WaitGroup
	readErrors := make(chan error, 20)
	negativeQtyDetected := false
	mu := &sync.Mutex{}

	// 10 goroutines writing
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			reqBody := `{
				"ipn": "IPN-READ-TEST",
				"type": "receive",
				"qty": 10,
				"reference": "WR"
			}`
			req := httptest.NewRequest("POST", "/api/v1/inventory/transact", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()
			handleInventoryTransact(w, req)
		}(i)
	}

	// 10 goroutines reading
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/api/v1/inventory/IPN-READ-TEST", nil)
			w := httptest.NewRecorder()
			handleGetInventory(w, req, "IPN-READ-TEST")

			if w.Code == 200 {
				var resp APIResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err == nil {
					dataBytes, _ := json.Marshal(resp.Data)
					var item InventoryItem
					if err := json.Unmarshal(dataBytes, &item); err == nil {
						if item.QtyOnHand < 0 {
							mu.Lock()
							negativeQtyDetected = true
							mu.Unlock()
							t.Errorf("Read goroutine %d: Detected negative quantity %f", idx, item.QtyOnHand)
						}
					}
				}
			} else {
				readErrors <- err
			}
		}(i)
	}

	wg.Wait()
	close(readErrors)

	if negativeQtyDetected {
		t.Error("Negative quantity detected during concurrent reads (data integrity issue)")
	}

	// Verify final state
	var finalQty float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn=?", "IPN-READ-TEST").Scan(&finalQty)
	expectedQty := 600.0 // 500 + (10 * 10)
	if finalQty != expectedQty {
		t.Errorf("Expected final quantity %f, got %f", expectedQty, finalQty)
	}
}
