package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupShipmentsTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create shipments table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS shipments (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL DEFAULT 'outbound' CHECK(type IN ('inbound','outbound','transfer')),
			status TEXT DEFAULT 'draft' CHECK(status IN ('draft','packed','shipped','delivered','cancelled')),
			tracking_number TEXT DEFAULT '',
			carrier TEXT DEFAULT '',
			ship_date DATETIME,
			delivery_date DATETIME,
			from_address TEXT DEFAULT '',
			to_address TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			created_by TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create shipments table: %v", err)
	}

	// Create shipment_lines table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS shipment_lines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			shipment_id TEXT NOT NULL,
			ipn TEXT DEFAULT '',
			serial_number TEXT DEFAULT '',
			qty INTEGER DEFAULT 1 CHECK(qty > 0),
			work_order_id TEXT DEFAULT '',
			rma_id TEXT DEFAULT '',
			FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create shipment_lines table: %v", err)
	}

	// Create pack_lists table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS pack_lists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			shipment_id TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create pack_lists table: %v", err)
	}

	// Create inventory table (for inbound shipment testing)
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			qty_reserved REAL DEFAULT 0,
			location TEXT,
			reorder_point REAL DEFAULT 0,
			reorder_qty REAL DEFAULT 0,
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
		CREATE TABLE IF NOT EXISTS inventory_transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ipn TEXT NOT NULL,
			type TEXT NOT NULL,
			qty REAL NOT NULL,
			reference TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory_transactions table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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

	// Create sequences table for nextID
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS sequences (
			prefix TEXT PRIMARY KEY,
			current INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sequences table: %v", err)
	}

	return testDB
}

func insertTestShipment(t *testing.T, db *sql.DB, id, shipType, status, trackingNumber, carrier string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO shipments (id, type, status, tracking_number, carrier, from_address, to_address, notes, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'Origin', 'Destination', 'Test notes', 'testuser', ?, ?)`,
		id, shipType, status, trackingNumber, carrier, now, now)
	if err != nil {
		t.Fatalf("Failed to insert test shipment: %v", err)
	}
}

func insertTestShipmentLine(t *testing.T, db *sql.DB, shipmentID, ipn string, qty int) {
	_, err := db.Exec(`INSERT INTO shipment_lines (shipment_id, ipn, qty)
		VALUES (?, ?, ?)`, shipmentID, ipn, qty)
	if err != nil {
		t.Fatalf("Failed to insert test shipment line: %v", err)
	}
}

func TestHandleListShipments(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	tests := []struct {
		name          string
		setupData     func(*sql.DB)
		expectedCount int
	}{
		{
			name: "empty list",
			setupData: func(db *sql.DB) {
				// No data
			},
			expectedCount: 0,
		},
		{
			name: "multiple shipments",
			setupData: func(db *sql.DB) {
				insertTestShipment(t, db, "SHP-0001", "outbound", "draft", "", "")
				insertTestShipment(t, db, "SHP-0002", "inbound", "shipped", "TRACK123", "FedEx")
				insertTestShipment(t, db, "SHP-0003", "transfer", "delivered", "TRACK456", "UPS")
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db = setupShipmentsTestDB(t)
			defer db.Close()

			tt.setupData(db)

			req := httptest.NewRequest("GET", "/api/shipments", nil)
			w := httptest.NewRecorder()

			handleListShipments(w, req)

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response []Shipment
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(response) != tt.expectedCount {
				t.Errorf("Expected %d shipments, got %d", tt.expectedCount, len(response))
			}
		})
	}
}

func TestHandleGetShipment(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	insertTestShipment(t, db, "SHP-0001", "outbound", "draft", "", "")
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-001", 5)
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-002", 10)

	req := httptest.NewRequest("GET", "/api/shipments/SHP-0001", nil)
	w := httptest.NewRecorder()

	handleGetShipment(w, req, "SHP-0001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response Shipment
	json.NewDecoder(w.Body).Decode(&response)

	if response.ID != "SHP-0001" {
		t.Errorf("Expected ID SHP-0001, got %s", response.ID)
	}
	if response.Type != "outbound" {
		t.Errorf("Expected type outbound, got %s", response.Type)
	}
	if len(response.Lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(response.Lines))
	}
}

func TestHandleGetShipment_NotFound(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/shipments/SHP-9999", nil)
	w := httptest.NewRecorder()

	handleGetShipment(w, req, "SHP-9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateShipment(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	tests := []struct {
		name           string
		input          Shipment
		expectedStatus int
		expectedType   string
		expectedStatus2 string
	}{
		{
			name: "valid outbound shipment",
			input: Shipment{
				Type:        "outbound",
				Status:      "draft",
				FromAddress: "Warehouse A",
				ToAddress:   "Customer X",
				Lines: []ShipmentLine{
					{IPN: "PROD-001", Qty: 5},
					{IPN: "PROD-002", Qty: 10},
				},
			},
			expectedStatus:  200,
			expectedType:    "outbound",
			expectedStatus2: "draft",
		},
		{
			name: "defaults applied",
			input: Shipment{
				FromAddress: "Warehouse A",
				ToAddress:   "Customer X",
			},
			expectedStatus:  200,
			expectedType:    "outbound",
			expectedStatus2: "draft",
		},
		{
			name: "inbound shipment",
			input: Shipment{
				Type:        "inbound",
				Status:      "draft",
				FromAddress: "Vendor Y",
				ToAddress:   "Warehouse A",
			},
			expectedStatus:  200,
			expectedType:    "inbound",
			expectedStatus2: "draft",
		},
		{
			name: "with tracking info",
			input: Shipment{
				Type:           "outbound",
				TrackingNumber: "TRACK123",
				Carrier:        "FedEx",
			},
			expectedStatus:  200,
			expectedType:    "outbound",
			expectedStatus2: "draft",
		},
		{
			name: "invalid type",
			input: Shipment{
				Type: "invalid",
			},
			expectedStatus: 400,
		},
		{
			name: "invalid status",
			input: Shipment{
				Status: "invalid",
			},
			expectedStatus: 400,
		},
		{
			name: "invalid qty - zero",
			input: Shipment{
				Lines: []ShipmentLine{
					{IPN: "PROD-001", Qty: 0},
				},
			},
			expectedStatus: 400,
		},
		{
			name: "invalid qty - negative",
			input: Shipment{
				Lines: []ShipmentLine{
					{IPN: "PROD-001", Qty: -5},
				},
			},
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db = setupShipmentsTestDB(t)
			defer db.Close()

			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest("POST", "/api/shipments", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateShipment(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus == 200 {
				var response Shipment
				json.NewDecoder(w.Body).Decode(&response)

				if response.ID == "" {
					t.Error("Expected non-empty ID")
				}
				if !strings.HasPrefix(response.ID, "SHP-") {
					t.Errorf("Expected ID to start with SHP-, got %s", response.ID)
				}
				if response.Type != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, response.Type)
				}
				if response.Status != tt.expectedStatus2 {
					t.Errorf("Expected status %s, got %s", tt.expectedStatus2, response.Status)
				}

				// Verify lines were created
				if tt.input.Lines != nil {
					var count int
					db.QueryRow("SELECT COUNT(*) FROM shipment_lines WHERE shipment_id = ?", response.ID).Scan(&count)
					if count != len(tt.input.Lines) {
						t.Errorf("Expected %d lines, got %d", len(tt.input.Lines), count)
					}
				}
			}
		})
	}
}

func TestHandleUpdateShipment(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	insertTestShipment(t, db, "SHP-0001", "outbound", "draft", "", "")
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-001", 5)

	// Update shipment
	update := Shipment{
		Type:           "outbound",
		Status:         "packed",
		TrackingNumber: "TRACK123",
		Carrier:        "FedEx",
		Lines: []ShipmentLine{
			{IPN: "PROD-001", Qty: 10}, // Updated qty
			{IPN: "PROD-002", Qty: 5},  // New line
		},
	}

	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/shipments/SHP-0001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateShipment(w, req, "SHP-0001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response Shipment
	json.NewDecoder(w.Body).Decode(&response)

	if response.Status != "packed" {
		t.Errorf("Expected status packed, got %s", response.Status)
	}
	if response.TrackingNumber != "TRACK123" {
		t.Errorf("Expected tracking TRACK123, got %s", response.TrackingNumber)
	}
	if len(response.Lines) != 2 {
		t.Errorf("Expected 2 lines after update, got %d", len(response.Lines))
	}
}

func TestHandleShipShipment(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	tests := []struct {
		name           string
		initialStatus  string
		input          map[string]string
		expectedStatus int
	}{
		{
			name:          "ship from draft",
			initialStatus: "draft",
			input: map[string]string{
				"tracking_number": "TRACK123",
				"carrier":         "FedEx",
			},
			expectedStatus: 200,
		},
		{
			name:          "ship from packed",
			initialStatus: "packed",
			input: map[string]string{
				"tracking_number": "TRACK456",
				"carrier":         "UPS",
			},
			expectedStatus: 200,
		},
		{
			name:          "already shipped",
			initialStatus: "shipped",
			input: map[string]string{
				"tracking_number": "TRACK789",
				"carrier":         "DHL",
			},
			expectedStatus: 400,
		},
		{
			name:          "already delivered",
			initialStatus: "delivered",
			input: map[string]string{
				"tracking_number": "TRACK999",
				"carrier":         "USPS",
			},
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db = setupShipmentsTestDB(t)
			defer db.Close()

			insertTestShipment(t, db, "SHP-0001", "outbound", tt.initialStatus, "", "")

			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest("POST", "/api/shipments/SHP-0001/ship", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleShipShipment(w, req, "SHP-0001")

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus == 200 {
				var response Shipment
				json.NewDecoder(w.Body).Decode(&response)

				if response.Status != "shipped" {
					t.Errorf("Expected status shipped, got %s", response.Status)
				}
				if response.TrackingNumber != tt.input["tracking_number"] {
					t.Errorf("Expected tracking %s, got %s", tt.input["tracking_number"], response.TrackingNumber)
				}
				if response.Carrier != tt.input["carrier"] {
					t.Errorf("Expected carrier %s, got %s", tt.input["carrier"], response.Carrier)
				}
				if response.ShipDate == nil || *response.ShipDate == "" {
					t.Error("Expected ship_date to be set")
				}
			}
		})
	}
}

func TestHandleShipShipment_NotFound(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	input := map[string]string{
		"tracking_number": "TRACK123",
		"carrier":         "FedEx",
	}

	body, _ := json.Marshal(input)
	req := httptest.NewRequest("POST", "/api/shipments/SHP-9999/ship", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleShipShipment(w, req, "SHP-9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleDeliverShipment(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	tests := []struct {
		name           string
		shipType       string
		initialStatus  string
		expectedStatus int
	}{
		{
			name:           "deliver outbound",
			shipType:       "outbound",
			initialStatus:  "shipped",
			expectedStatus: 200,
		},
		{
			name:           "deliver inbound",
			shipType:       "inbound",
			initialStatus:  "shipped",
			expectedStatus: 200,
		},
		{
			name:           "already delivered",
			shipType:       "outbound",
			initialStatus:  "delivered",
			expectedStatus: 400,
		},
		{
			name:           "deliver from draft (allowed)",
			shipType:       "outbound",
			initialStatus:  "draft",
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db = setupShipmentsTestDB(t)
			defer db.Close()

			insertTestShipment(t, db, "SHP-0001", tt.shipType, tt.initialStatus, "TRACK123", "FedEx")

			req := httptest.NewRequest("POST", "/api/shipments/SHP-0001/deliver", nil)
			w := httptest.NewRecorder()

			handleDeliverShipment(w, req, "SHP-0001")

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus == 200 {
				var response Shipment
				json.NewDecoder(w.Body).Decode(&response)

				if response.Status != "delivered" {
					t.Errorf("Expected status delivered, got %s", response.Status)
				}
				if response.DeliveryDate == nil || *response.DeliveryDate == "" {
					t.Error("Expected delivery_date to be set")
				}
			}
		})
	}
}

func TestHandleDeliverShipment_InboundInventoryUpdate(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	// Setup inventory
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO inventory (ipn, qty_on_hand, updated_at) VALUES ('PROD-001', 100, ?)", now)
	db.Exec("INSERT INTO inventory (ipn, qty_on_hand, updated_at) VALUES ('PROD-002', 50, ?)", now)

	// Create inbound shipment
	insertTestShipment(t, db, "SHP-0001", "inbound", "shipped", "TRACK123", "FedEx")
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-001", 10)
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-002", 20)

	req := httptest.NewRequest("POST", "/api/shipments/SHP-0001/deliver", nil)
	w := httptest.NewRecorder()

	handleDeliverShipment(w, req, "SHP-0001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify inventory was updated
	var qty1, qty2 float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = 'PROD-001'").Scan(&qty1)
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = 'PROD-002'").Scan(&qty2)

	if qty1 != 110 {
		t.Errorf("Expected PROD-001 qty 110, got %.0f", qty1)
	}
	if qty2 != 70 {
		t.Errorf("Expected PROD-002 qty 70, got %.0f", qty2)
	}

	// Verify transactions were created
	var txCount int
	db.QueryRow("SELECT COUNT(*) FROM inventory_transactions WHERE reference LIKE 'SHP:%'").Scan(&txCount)
	if txCount != 2 {
		t.Errorf("Expected 2 inventory transactions, got %d", txCount)
	}
}

func TestHandleDeliverShipment_OutboundNoInventoryChange(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	// Setup inventory
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO inventory (ipn, qty_on_hand, updated_at) VALUES ('PROD-001', 100, ?)", now)

	// Create outbound shipment
	insertTestShipment(t, db, "SHP-0001", "outbound", "shipped", "TRACK123", "FedEx")
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-001", 10)

	req := httptest.NewRequest("POST", "/api/shipments/SHP-0001/deliver", nil)
	w := httptest.NewRecorder()

	handleDeliverShipment(w, req, "SHP-0001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify inventory was NOT updated for outbound
	var qty float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = 'PROD-001'").Scan(&qty)

	if qty != 100 {
		t.Errorf("Expected PROD-001 qty unchanged at 100, got %.0f", qty)
	}
}

func TestHandleShipmentPackList(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	insertTestShipment(t, db, "SHP-0001", "outbound", "packed", "", "")
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-001", 5)
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-002", 10)

	req := httptest.NewRequest("GET", "/api/shipments/SHP-0001/pack-list", nil)
	w := httptest.NewRecorder()

	handleShipmentPackList(w, req, "SHP-0001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response PackList
	json.NewDecoder(w.Body).Decode(&response)

	if response.ShipmentID != "SHP-0001" {
		t.Errorf("Expected shipment_id SHP-0001, got %s", response.ShipmentID)
	}
	if len(response.Lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(response.Lines))
	}

	// Verify pack list was created in database
	var count int
	db.QueryRow("SELECT COUNT(*) FROM pack_lists WHERE shipment_id = 'SHP-0001'").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 pack list record, got %d", count)
	}
}

func TestHandleShipmentPackList_NotFound(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/shipments/SHP-9999/pack-list", nil)
	w := httptest.NewRecorder()

	handleShipmentPackList(w, req, "SHP-9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test shipment type validation
func TestShipment_TypeValidation(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	validTypes := []string{"inbound", "outbound", "transfer"}

	for _, shipType := range validTypes {
		t.Run("valid_type_"+shipType, func(t *testing.T) {
			db = setupShipmentsTestDB(t)
			defer db.Close()

			shipment := Shipment{
				Type: shipType,
			}

			body, _ := json.Marshal(shipment)
			req := httptest.NewRequest("POST", "/api/shipments", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handleCreateShipment(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200 for type %s, got %d", shipType, w.Code)
			}
		})
	}
}

// Test shipment status validation
func TestShipment_StatusValidation(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	validStatuses := []string{"draft", "packed", "shipped", "delivered", "cancelled"}

	for _, status := range validStatuses {
		t.Run("valid_status_"+status, func(t *testing.T) {
			db = setupShipmentsTestDB(t)
			defer db.Close()

			shipment := Shipment{
				Status: status,
			}

			body, _ := json.Marshal(shipment)
			req := httptest.NewRequest("POST", "/api/shipments", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handleCreateShipment(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200 for status %s, got %d", status, w.Code)
			}
		})
	}
}

// Test carrier data handling
func TestShipment_CarrierData(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	carriers := []string{"FedEx", "UPS", "USPS", "DHL", "OnTrac", "Custom Carrier"}

	for _, carrier := range carriers {
		t.Run("carrier_"+carrier, func(t *testing.T) {
			db = setupShipmentsTestDB(t)
			defer db.Close()

			insertTestShipment(t, db, "SHP-0001", "outbound", "draft", "", "")

			input := map[string]string{
				"tracking_number": "TRACK123",
				"carrier":         carrier,
			}

			body, _ := json.Marshal(input)
			req := httptest.NewRequest("POST", "/api/shipments/SHP-0001/ship", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handleShipShipment(w, req, "SHP-0001")

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d", w.Code)
			}

			var response Shipment
			json.NewDecoder(w.Body).Decode(&response)

			if response.Carrier != carrier {
				t.Errorf("Expected carrier %s, got %s", carrier, response.Carrier)
			}
		})
	}
}

// Test serial number tracking
func TestShipment_SerialNumberTracking(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	shipment := Shipment{
		Type:   "outbound",
		Status: "draft",
		Lines: []ShipmentLine{
			{IPN: "DEVICE-001", SerialNumber: "SN12345", Qty: 1},
			{IPN: "DEVICE-002", SerialNumber: "SN67890", Qty: 1},
		},
	}

	body, _ := json.Marshal(shipment)
	req := httptest.NewRequest("POST", "/api/shipments", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateShipment(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response Shipment
	json.NewDecoder(w.Body).Decode(&response)

	if len(response.Lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(response.Lines))
	}

	for i, line := range response.Lines {
		if line.SerialNumber == "" {
			t.Errorf("Line %d: expected serial number to be set", i)
		}
	}
}

// Test work order and RMA linkage
func TestShipment_WorkOrderRMALinks(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	shipment := Shipment{
		Type:   "outbound",
		Status: "draft",
		Lines: []ShipmentLine{
			{IPN: "PROD-001", Qty: 5, WorkOrderID: "WO-001"},
			{IPN: "PROD-002", Qty: 3, RMAID: "RMA-001"},
		},
	}

	body, _ := json.Marshal(shipment)
	req := httptest.NewRequest("POST", "/api/shipments", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateShipment(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response Shipment
	json.NewDecoder(w.Body).Decode(&response)

	// Verify work order link
	hasWO := false
	hasRMA := false
	for _, line := range response.Lines {
		if line.WorkOrderID == "WO-001" {
			hasWO = true
		}
		if line.RMAID == "RMA-001" {
			hasRMA = true
		}
	}

	if !hasWO {
		t.Error("Expected work order link to be preserved")
	}
	if !hasRMA {
		t.Error("Expected RMA link to be preserved")
	}
}

// Test audit logging
func TestShipment_AuditLogging(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	shipment := Shipment{
		Type:   "outbound",
		Status: "draft",
	}

	body, _ := json.Marshal(shipment)
	req := httptest.NewRequest("POST", "/api/shipments", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handleCreateShipment(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify audit log entry
	var count int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module = 'shipment' AND action = 'created'").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", count)
	}
}

// Test foreign key cascade delete
func TestShipment_CascadeDelete(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	insertTestShipment(t, db, "SHP-0001", "outbound", "draft", "", "")
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-001", 5)
	insertTestShipmentLine(t, db, "SHP-0001", "PROD-002", 10)

	// Delete shipment
	db.Exec("DELETE FROM shipments WHERE id = 'SHP-0001'")

	// Verify lines were cascade deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM shipment_lines WHERE shipment_id = 'SHP-0001'").Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 lines after cascade delete, got %d", count)
	}
}

// Test concurrent shipment operations
func TestShipment_ConcurrentOperations(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupShipmentsTestDB(t)
	defer db.Close()

	insertTestShipment(t, db, "SHP-0001", "outbound", "draft", "", "")

	// Try to ship the same shipment concurrently
	done := make(chan bool)
	successCount := 0

	for i := 0; i < 5; i++ {
		go func(idx int) {
			input := map[string]string{
				"tracking_number": "TRACK" + string(rune('0'+idx)),
				"carrier":         "FedEx",
			}

			body, _ := json.Marshal(input)
			req := httptest.NewRequest("POST", "/api/shipments/SHP-0001/ship", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handleShipShipment(w, req, "SHP-0001")

			if w.Code == 200 {
				successCount++
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// All should succeed or fail gracefully (no crashes)
	// The last one wins due to UPSERT behavior
	var status string
	db.QueryRow("SELECT status FROM shipments WHERE id = 'SHP-0001'").Scan(&status)
	if status != "shipped" {
		t.Errorf("Expected final status shipped, got %s", status)
	}
}
