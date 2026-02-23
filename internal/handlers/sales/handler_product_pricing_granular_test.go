package sales_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

func setupProductPricingTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create product_pricing table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS product_pricing (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_ipn TEXT NOT NULL,
			pricing_tier TEXT NOT NULL DEFAULT 'standard' CHECK(pricing_tier IN ('standard','volume','distributor','oem')),
			min_qty INTEGER DEFAULT 0 CHECK(min_qty >= 0),
			max_qty INTEGER DEFAULT 0 CHECK(max_qty >= 0),
			unit_price REAL NOT NULL DEFAULT 0 CHECK(unit_price >= 0),
			currency TEXT DEFAULT 'USD',
			effective_date TEXT DEFAULT '',
			expiry_date TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create product_pricing table: %v", err)
	}

	// Create cost_analysis table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS cost_analysis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_ipn TEXT NOT NULL UNIQUE,
			bom_cost REAL DEFAULT 0,
			labor_cost REAL DEFAULT 0,
			overhead_cost REAL DEFAULT 0,
			total_cost REAL DEFAULT 0,
			margin_pct REAL DEFAULT 0,
			last_calculated DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create cost_analysis table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			username TEXT,
			action TEXT,
			table_name TEXT,
			record_id TEXT,
			details TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	return testDB
}

func insertTestPricing(t *testing.T, db *sql.DB, ipn, tier string, minQty, maxQty int, unitPrice float64, currency, effectiveDate string) int64 {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(`INSERT INTO product_pricing (product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency, effective_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ipn, tier, minQty, maxQty, unitPrice, currency, effectiveDate, now, now)
	if err != nil {
		t.Fatalf("Failed to insert test pricing: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// extractProductPricing decodes a single ProductPricing from the APIResponse envelope.
func extractProductPricing(t *testing.T, body []byte) models.ProductPricing {
	t.Helper()
	var resp models.APIResponse
	json.Unmarshal(body, &resp)
	b, _ := json.Marshal(resp.Data)
	var p models.ProductPricing
	json.Unmarshal(b, &p)
	return p
}

// extractProductPricingList decodes a slice of ProductPricing from the APIResponse envelope.
func extractProductPricingList(t *testing.T, body []byte) []models.ProductPricing {
	t.Helper()
	var resp models.APIResponse
	json.Unmarshal(body, &resp)
	b, _ := json.Marshal(resp.Data)
	var items []models.ProductPricing
	json.Unmarshal(b, &items)
	return items
}

// extractCostAnalysis decodes a CostAnalysis from the APIResponse envelope.
func extractCostAnalysis(t *testing.T, body []byte) models.CostAnalysis {
	t.Helper()
	var resp models.APIResponse
	json.Unmarshal(body, &resp)
	b, _ := json.Marshal(resp.Data)
	var c models.CostAnalysis
	json.Unmarshal(b, &c)
	return c
}

// extractCostAnalysisList decodes a slice of CostAnalysisWithPricing from the APIResponse envelope.
func extractCostAnalysisList(t *testing.T, body []byte) []models.CostAnalysisWithPricing {
	t.Helper()
	var resp models.APIResponse
	json.Unmarshal(body, &resp)
	b, _ := json.Marshal(resp.Data)
	var items []models.CostAnalysisWithPricing
	json.Unmarshal(b, &items)
	return items
}

func TestHandleListProductPricing(t *testing.T) {
	tests := []struct {
		name          string
		setupData     func(*sql.DB)
		queryParams   string
		expectedCount int
		expectedFirst string
	}{
		{
			name: "empty list",
			setupData: func(db *sql.DB) {
				// No data
			},
			queryParams:   "",
			expectedCount: 0,
		},
		{
			name: "multiple pricing tiers",
			setupData: func(db *sql.DB) {
				insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")
				insertTestPricing(t, db, "PROD-001", "volume", 100, 999, 90.0, "USD", "2024-01-01")
				insertTestPricing(t, db, "PROD-002", "standard", 1, 99, 50.0, "USD", "2024-01-01")
			},
			queryParams:   "",
			expectedCount: 3,
			expectedFirst: "PROD-001",
		},
		{
			name: "filter by product_ipn",
			setupData: func(db *sql.DB) {
				insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")
				insertTestPricing(t, db, "PROD-002", "standard", 1, 99, 50.0, "USD", "2024-01-01")
			},
			queryParams:   "?product_ipn=PROD-001",
			expectedCount: 1,
			expectedFirst: "PROD-001",
		},
		{
			name: "filter by pricing_tier",
			setupData: func(db *sql.DB) {
				insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")
				insertTestPricing(t, db, "PROD-001", "volume", 100, 999, 90.0, "USD", "2024-01-01")
				insertTestPricing(t, db, "PROD-002", "standard", 1, 99, 50.0, "USD", "2024-01-01")
			},
			queryParams:   "?pricing_tier=volume",
			expectedCount: 1,
		},
		{
			name: "multiple currencies",
			setupData: func(db *sql.DB) {
				insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")
				insertTestPricing(t, db, "PROD-002", "standard", 1, 99, 85.0, "EUR", "2024-01-01")
			},
			queryParams:   "",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupProductPricingTestDB(t)
			defer db.Close()
			h := newTestHandler(db)

			tt.setupData(db)

			req := httptest.NewRequest("GET", "/api/pricing"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			h.ListProductPricing(w, req)

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			response := extractProductPricingList(t, w.Body.Bytes())

			if len(response) != tt.expectedCount {
				t.Errorf("Expected %d items, got %d", tt.expectedCount, len(response))
			}

			if tt.expectedCount > 0 && tt.expectedFirst != "" {
				if response[0].ProductIPN != tt.expectedFirst {
					t.Errorf("Expected first item to be %s, got %s", tt.expectedFirst, response[0].ProductIPN)
				}
			}
		})
	}
}

func TestHandleGetProductPricing(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	id := insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")

	req := httptest.NewRequest("GET", "/api/pricing/1", nil)
	w := httptest.NewRecorder()

	h.GetProductPricing(w, req, "1")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	response := extractProductPricing(t, w.Body.Bytes())

	if response.ID != int(id) {
		t.Errorf("Expected ID %d, got %d", id, response.ID)
	}
	if response.ProductIPN != "PROD-001" {
		t.Errorf("Expected ProductIPN PROD-001, got %s", response.ProductIPN)
	}
	if response.UnitPrice != 100.0 {
		t.Errorf("Expected UnitPrice 100.0, got %.2f", response.UnitPrice)
	}
}

func TestHandleGetProductPricing_NotFound(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/pricing/999", nil)
	w := httptest.NewRecorder()

	h.GetProductPricing(w, req, "999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateProductPricing(t *testing.T) {
	tests := []struct {
		name           string
		input          models.ProductPricing
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid pricing",
			input: models.ProductPricing{
				ProductIPN:    "PROD-001",
				PricingTier:   "standard",
				MinQty:        1,
				MaxQty:        99,
				UnitPrice:     100.0,
				Currency:      "USD",
				EffectiveDate: "2024-01-01",
			},
			expectedStatus: 200,
		},
		{
			name: "defaults applied",
			input: models.ProductPricing{
				ProductIPN: "PROD-002",
				UnitPrice:  50.0,
			},
			expectedStatus: 200,
		},
		{
			name: "missing product_ipn",
			input: models.ProductPricing{
				UnitPrice: 100.0,
			},
			expectedStatus: 400,
			expectedError:  "product_ipn required",
		},
		{
			name: "volume tier",
			input: models.ProductPricing{
				ProductIPN:  "PROD-003",
				PricingTier: "volume",
				MinQty:      100,
				MaxQty:      999,
				UnitPrice:   90.0,
			},
			expectedStatus: 200,
		},
		{
			name: "EUR currency",
			input: models.ProductPricing{
				ProductIPN: "PROD-004",
				UnitPrice:  85.0,
				Currency:   "EUR",
			},
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupProductPricingTestDB(t)
			defer db.Close()
			h := newTestHandler(db)

			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest("POST", "/api/pricing", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.CreateProductPricing(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == 200 {
				response := extractProductPricing(t, w.Body.Bytes())

				if response.ID == 0 {
					t.Error("Expected non-zero ID")
				}

				// Verify defaults
				if tt.input.Currency == "" && response.Currency != "USD" {
					t.Errorf("Expected default currency USD, got %s", response.Currency)
				}
				if tt.input.PricingTier == "" && response.PricingTier != "standard" {
					t.Errorf("Expected default tier standard, got %s", response.PricingTier)
				}
			}
		})
	}
}

func TestHandleUpdateProductPricing(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	id := insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")

	// Update unit price
	update := map[string]interface{}{
		"unit_price": 110.0,
	}

	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/pricing/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateProductPricing(w, req, "1")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	response := extractProductPricing(t, w.Body.Bytes())

	if response.UnitPrice != 110.0 {
		t.Errorf("Expected updated price 110.0, got %.2f", response.UnitPrice)
	}
	if response.ID != int(id) {
		t.Errorf("ID should not change, expected %d, got %d", id, response.ID)
	}
}

func TestHandleUpdateProductPricing_NotFound(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	update := map[string]interface{}{
		"unit_price": 110.0,
	}

	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/pricing/999", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.UpdateProductPricing(w, req, "999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleDeleteProductPricing(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")

	req := httptest.NewRequest("DELETE", "/api/pricing/1", nil)
	w := httptest.NewRecorder()

	h.DeleteProductPricing(w, req, "1")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify deletion
	var count int
	db.QueryRow("SELECT COUNT(*) FROM product_pricing WHERE id = 1").Scan(&count)
	if count != 0 {
		t.Error("Pricing should be deleted")
	}
}

func TestHandleDeleteProductPricing_NotFound(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("DELETE", "/api/pricing/999", nil)
	w := httptest.NewRecorder()

	h.DeleteProductPricing(w, req, "999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleListCostAnalysis(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Insert cost analysis
	now := time.Now().UTC().Format(time.RFC3339)
	db.Exec(`INSERT INTO cost_analysis (product_ipn, bom_cost, labor_cost, overhead_cost, total_cost, margin_pct, last_calculated, created_at)
		VALUES ('PROD-001', 50, 25, 25, 100, 0, ?, ?)`, now, now)

	// Insert pricing for margin calculation
	insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 200.0, "USD", "2024-01-01")

	req := httptest.NewRequest("GET", "/api/pricing/analysis", nil)
	w := httptest.NewRecorder()

	h.ListCostAnalysis(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	response := extractCostAnalysisList(t, w.Body.Bytes())

	if len(response) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(response))
	}

	if response[0].ProductIPN != "PROD-001" {
		t.Errorf("Expected PROD-001, got %s", response[0].ProductIPN)
	}
	if response[0].SellingPrice != 200.0 {
		t.Errorf("Expected selling price 200.0, got %.2f", response[0].SellingPrice)
	}
}

func TestHandleCreateCostAnalysis(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Insert pricing for margin calculation
	insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 200.0, "USD", "2024-01-01")

	costAnalysis := models.CostAnalysis{
		ProductIPN:   "PROD-001",
		BOMCost:      50.0,
		LaborCost:    25.0,
		OverheadCost: 25.0,
	}

	body, _ := json.Marshal(costAnalysis)
	req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateCostAnalysis(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	response := extractCostAnalysis(t, w.Body.Bytes())

	if response.TotalCost != 100.0 {
		t.Errorf("Expected total cost 100.0, got %.2f", response.TotalCost)
	}

	// Margin = (200 - 100) / 200 * 100 = 50%
	if response.MarginPct != 50.0 {
		t.Errorf("Expected margin 50%%, got %.2f%%", response.MarginPct)
	}
}

func TestHandleCreateCostAnalysis_MissingIPN(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	costAnalysis := models.CostAnalysis{
		BOMCost:   50.0,
		LaborCost: 25.0,
	}

	body, _ := json.Marshal(costAnalysis)
	req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateCostAnalysis(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleProductPricingHistory(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Insert multiple pricing records for same product with explicit timestamps
	// to ensure deterministic ordering
	now := time.Now().UTC()
	older := now.Add(-1 * time.Hour).Format(time.RFC3339)
	newer := now.Format(time.RFC3339)

	db.Exec(`INSERT INTO product_pricing (product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency, effective_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01", older, older)
	db.Exec(`INSERT INTO product_pricing (product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency, effective_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"PROD-001", "standard", 1, 99, 110.0, "USD", "2024-02-01", newer, newer)
	insertTestPricing(t, db, "PROD-002", "standard", 1, 99, 50.0, "USD", "2024-01-01")

	req := httptest.NewRequest("GET", "/api/pricing/history/PROD-001", nil)
	w := httptest.NewRecorder()

	h.ProductPricingHistory(w, req, "PROD-001")

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	response := extractProductPricingList(t, w.Body.Bytes())

	if len(response) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(response))
	}

	// Should be ordered by created_at DESC (most recent first)
	if response[0].UnitPrice != 110.0 {
		t.Errorf("Expected first price 110.0, got %.2f", response[0].UnitPrice)
	}
	if response[1].UnitPrice != 100.0 {
		t.Errorf("Expected second price 100.0, got %.2f", response[1].UnitPrice)
	}
}

func TestHandleBulkUpdateProductPricing(t *testing.T) {
	tests := []struct {
		name            string
		setupData       func(*sql.DB) []int64
		adjustmentType  string
		adjustmentValue float64
		expectedPrices  []float64
	}{
		{
			name: "percentage increase",
			setupData: func(db *sql.DB) []int64 {
				id1 := insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")
				id2 := insertTestPricing(t, db, "PROD-002", "standard", 1, 99, 200.0, "USD", "2024-01-01")
				return []int64{id1, id2}
			},
			adjustmentType:  "percentage",
			adjustmentValue: 10.0, // 10% increase
			expectedPrices:  []float64{110.0, 220.0},
		},
		{
			name: "percentage decrease",
			setupData: func(db *sql.DB) []int64 {
				id1 := insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")
				id2 := insertTestPricing(t, db, "PROD-002", "standard", 1, 99, 200.0, "USD", "2024-01-01")
				return []int64{id1, id2}
			},
			adjustmentType:  "percentage",
			adjustmentValue: -10.0, // 10% decrease
			expectedPrices:  []float64{90.0, 180.0},
		},
		{
			name: "absolute increase",
			setupData: func(db *sql.DB) []int64 {
				id1 := insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")
				id2 := insertTestPricing(t, db, "PROD-002", "standard", 1, 99, 200.0, "USD", "2024-01-01")
				return []int64{id1, id2}
			},
			adjustmentType:  "absolute",
			adjustmentValue: 50.0,
			expectedPrices:  []float64{150.0, 250.0},
		},
		{
			name: "absolute decrease",
			setupData: func(db *sql.DB) []int64 {
				id1 := insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")
				id2 := insertTestPricing(t, db, "PROD-002", "standard", 1, 99, 200.0, "USD", "2024-01-01")
				return []int64{id1, id2}
			},
			adjustmentType:  "absolute",
			adjustmentValue: -25.0,
			expectedPrices:  []float64{75.0, 175.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupProductPricingTestDB(t)
			defer db.Close()
			h := newTestHandler(db)

			ids := tt.setupData(db)

			bulkUpdate := models.BulkPriceUpdate{
				IDs:             make([]int, len(ids)),
				AdjustmentType:  tt.adjustmentType,
				AdjustmentValue: tt.adjustmentValue,
			}
			for i, id := range ids {
				bulkUpdate.IDs[i] = int(id)
			}

			body, _ := json.Marshal(bulkUpdate)
			req := httptest.NewRequest("POST", "/api/pricing/bulk-update", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.BulkUpdateProductPricing(w, req)

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var apiResp models.APIResponse
			json.Unmarshal(w.Body.Bytes(), &apiResp)
			respMap := apiResp.Data.(map[string]interface{})

			updated, ok := respMap["updated"]
			if !ok || updated == nil {
				t.Fatalf("Response missing 'updated' field: %v", respMap)
			}
			if int(updated.(float64)) != len(ids) {
				t.Errorf("Expected %d updated, got %v", len(ids), updated)
			}

			// Verify prices
			for i, id := range ids {
				var price float64
				db.QueryRow("SELECT unit_price FROM product_pricing WHERE id = ?", id).Scan(&price)
				if price != tt.expectedPrices[i] {
					t.Errorf("Expected price %.2f for ID %d, got %.2f", tt.expectedPrices[i], id, price)
				}
			}
		})
	}
}

func TestHandleBulkUpdateProductPricing_InvalidType(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	id := insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")

	bulkUpdate := models.BulkPriceUpdate{
		IDs:             []int{int(id)},
		AdjustmentType:  "invalid",
		AdjustmentValue: 10.0,
	}

	body, _ := json.Marshal(bulkUpdate)
	req := httptest.NewRequest("POST", "/api/pricing/bulk-update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.BulkUpdateProductPricing(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleBulkUpdateProductPricing_EmptyIDs(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	bulkUpdate := models.BulkPriceUpdate{
		IDs:             []int{},
		AdjustmentType:  "percentage",
		AdjustmentValue: 10.0,
	}

	body, _ := json.Marshal(bulkUpdate)
	req := httptest.NewRequest("POST", "/api/pricing/bulk-update", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.BulkUpdateProductPricing(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Test currency handling
func TestProductPricing_MultipleCurrencies(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	currencies := []string{"USD", "EUR", "GBP", "JPY"}
	for i, curr := range currencies {
		insertTestPricing(t, db, "PROD-001", "standard", 1, 99, float64((i+1)*100), curr, "2024-01-01")
	}

	req := httptest.NewRequest("GET", "/api/pricing?product_ipn=PROD-001", nil)
	w := httptest.NewRecorder()

	h.ListProductPricing(w, req)

	response := extractProductPricingList(t, w.Body.Bytes())

	if len(response) != 4 {
		t.Fatalf("Expected 4 pricing records, got %d", len(response))
	}

	// Verify all currencies present
	foundCurrencies := make(map[string]bool)
	for _, p := range response {
		foundCurrencies[p.Currency] = true
	}

	for _, curr := range currencies {
		if !foundCurrencies[curr] {
			t.Errorf("Currency %s not found in response", curr)
		}
	}
}

// Test pricing tier validation
func TestProductPricing_TierOrdering(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Insert in random order, should be sorted by tier
	insertTestPricing(t, db, "PROD-001", "volume", 100, 999, 90.0, "USD", "2024-01-01")
	insertTestPricing(t, db, "PROD-001", "standard", 1, 99, 100.0, "USD", "2024-01-01")
	insertTestPricing(t, db, "PROD-001", "distributor", 1000, 9999, 80.0, "USD", "2024-01-01")

	req := httptest.NewRequest("GET", "/api/pricing?product_ipn=PROD-001", nil)
	w := httptest.NewRecorder()

	h.ListProductPricing(w, req)

	response := extractProductPricingList(t, w.Body.Bytes())

	if len(response) != 3 {
		t.Fatalf("Expected 3 pricing records, got %d", len(response))
	}

	// Verify sorted by pricing_tier, min_qty
	expectedOrder := []string{"distributor", "standard", "volume"}
	for i, expected := range expectedOrder {
		if response[i].PricingTier != expected {
			t.Errorf("Position %d: expected tier %s, got %s", i, expected, response[i].PricingTier)
		}
	}
}

// Test margin calculation edge cases
func TestCostAnalysis_ZeroSellingPrice(t *testing.T) {
	db := setupProductPricingTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// No pricing = margin should be 0
	costAnalysis := models.CostAnalysis{
		ProductIPN:   "PROD-NO-PRICE",
		BOMCost:      50.0,
		LaborCost:    25.0,
		OverheadCost: 25.0,
	}

	body, _ := json.Marshal(costAnalysis)
	req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.CreateCostAnalysis(w, req)

	response := extractCostAnalysis(t, w.Body.Bytes())

	if response.MarginPct != 0.0 {
		t.Errorf("Expected margin 0%% when no pricing, got %.2f%%", response.MarginPct)
	}
}
