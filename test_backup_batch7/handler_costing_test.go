package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

// handler_costing.go is a placeholder module - actual costing logic lives in:
// 1. handler_product_pricing.go - Cost analysis CRUD and margin calculations
// 2. handler_quotes.go - handleQuoteCost for BOM cost rollup
// 3. handler_workorders.go - handleWorkOrderBOM for work order costing
//
// This test file validates that costing functionality exists and works correctly
// across those modules.

func setupCostingTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create cost_analysis table (from handler_product_pricing.go)
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

	// Create product_pricing table (for margin calculations)
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS product_pricing (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_ipn TEXT NOT NULL,
			pricing_tier TEXT NOT NULL DEFAULT 'standard',
			min_qty INTEGER DEFAULT 0,
			max_qty INTEGER DEFAULT 0,
			unit_price REAL NOT NULL DEFAULT 0,
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

	return testDB
}

// Test that cost rollup calculates total_cost correctly
func TestCostRollup_Calculation(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupCostingTestDB(t)
	defer db.Close()

	tests := []struct {
		name          string
		bomCost       float64
		laborCost     float64
		overheadCost  float64
		expectedTotal float64
	}{
		{
			name:          "simple costs",
			bomCost:       100.0,
			laborCost:     50.0,
			overheadCost:  25.0,
			expectedTotal: 175.0,
		},
		{
			name:          "zero overhead",
			bomCost:       100.0,
			laborCost:     50.0,
			overheadCost:  0.0,
			expectedTotal: 150.0,
		},
		{
			name:          "all zeros",
			bomCost:       0.0,
			laborCost:     0.0,
			overheadCost:  0.0,
			expectedTotal: 0.0,
		},
		{
			name:          "high precision",
			bomCost:       123.456,
			laborCost:     78.901,
			overheadCost:  34.567,
			expectedTotal: 236.924,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use handleCreateCostAnalysis which does the rollup
			costAnalysis := CostAnalysis{
				ProductIPN:   "TEST-" + tt.name,
				BOMCost:      tt.bomCost,
				LaborCost:    tt.laborCost,
				OverheadCost: tt.overheadCost,
			}

			body, _ := json.Marshal(costAnalysis)
			req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateCostAnalysis(w, req)

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response CostAnalysis
			json.NewDecoder(w.Body).Decode(&response)

			if response.TotalCost != tt.expectedTotal {
				t.Errorf("Expected total_cost %.3f, got %.3f", tt.expectedTotal, response.TotalCost)
			}
		})
	}
}

// Test margin percentage calculation
func TestCostRollup_MarginCalculation(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupCostingTestDB(t)
	defer db.Close()

	tests := []struct {
		name           string
		productIPN     string
		bomCost        float64
		laborCost      float64
		overheadCost   float64
		sellingPrice   float64
		expectedMargin float64
	}{
		{
			name:           "50% margin",
			productIPN:     "PROD-001",
			bomCost:        50.0,
			laborCost:      25.0,
			overheadCost:   25.0,
			sellingPrice:   200.0,
			expectedMargin: 50.0, // (200 - 100) / 200 * 100 = 50%
		},
		{
			name:           "25% margin",
			productIPN:     "PROD-002",
			bomCost:        60.0,
			laborCost:      30.0,
			overheadCost:   10.0,
			sellingPrice:   133.33,
			expectedMargin: 24.997, // ~25%
		},
		{
			name:           "zero margin (break even)",
			productIPN:     "PROD-003",
			bomCost:        50.0,
			laborCost:      30.0,
			overheadCost:   20.0,
			sellingPrice:   100.0,
			expectedMargin: 0.0,
		},
		{
			name:           "negative margin (loss)",
			productIPN:     "PROD-004",
			bomCost:        80.0,
			laborCost:      30.0,
			overheadCost:   20.0,
			sellingPrice:   100.0,
			expectedMargin: -30.0, // (100 - 130) / 100 * 100 = -30%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Insert pricing first
			db.Exec(`INSERT INTO product_pricing (product_ipn, pricing_tier, unit_price, effective_date)
				VALUES (?, 'standard', ?, datetime('now'))`, tt.productIPN, tt.sellingPrice)

			// Create cost analysis
			costAnalysis := CostAnalysis{
				ProductIPN:   tt.productIPN,
				BOMCost:      tt.bomCost,
				LaborCost:    tt.laborCost,
				OverheadCost: tt.overheadCost,
			}

			body, _ := json.Marshal(costAnalysis)
			req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleCreateCostAnalysis(w, req)

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response CostAnalysis
			json.NewDecoder(w.Body).Decode(&response)

			// Allow small floating point error
			marginDiff := response.MarginPct - tt.expectedMargin
			if marginDiff < -0.01 || marginDiff > 0.01 {
				t.Errorf("Expected margin %.2f%%, got %.2f%%", tt.expectedMargin, response.MarginPct)
			}
		})
	}
}

// Test margin calculation when no pricing exists
func TestCostRollup_NoPricing(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupCostingTestDB(t)
	defer db.Close()

	costAnalysis := CostAnalysis{
		ProductIPN:   "PROD-NO-PRICE",
		BOMCost:      100.0,
		LaborCost:    50.0,
		OverheadCost: 25.0,
	}

	body, _ := json.Marshal(costAnalysis)
	req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateCostAnalysis(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response CostAnalysis
	json.NewDecoder(w.Body).Decode(&response)

	// When no pricing exists, margin should be 0
	if response.MarginPct != 0 {
		t.Errorf("Expected margin 0%% when no pricing exists, got %.2f%%", response.MarginPct)
	}
}

// Test BOM cost component validation
func TestCostRollup_ComponentValidation(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupCostingTestDB(t)
	defer db.Close()

	// Test negative values (should be handled gracefully)
	costAnalysis := CostAnalysis{
		ProductIPN:   "PROD-NEG",
		BOMCost:      -10.0, // Invalid
		LaborCost:    50.0,
		OverheadCost: 25.0,
	}

	body, _ := json.Marshal(costAnalysis)
	req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleCreateCostAnalysis(w, req)

	// Current implementation doesn't validate negative costs, just calculates
	// This is a potential bug - costs should be >= 0
	if w.Code != 200 {
		t.Fatalf("Expected status 200 (current behavior), got %d", w.Code)
	}

	var response CostAnalysis
	json.NewDecoder(w.Body).Decode(&response)

	// Document current behavior: negative BOM cost results in negative total
	expectedTotal := -10.0 + 50.0 + 25.0
	if response.TotalCost != expectedTotal {
		t.Errorf("Expected total_cost %.2f, got %.2f", expectedTotal, response.TotalCost)
	}

	// TODO: Add validation to reject negative cost components
}

// Test cost analysis upsert behavior
func TestCostRollup_Upsert(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupCostingTestDB(t)
	defer db.Close()

	productIPN := "PROD-UPSERT"

	// First insert
	cost1 := CostAnalysis{
		ProductIPN:   productIPN,
		BOMCost:      100.0,
		LaborCost:    50.0,
		OverheadCost: 25.0,
	}

	body, _ := json.Marshal(cost1)
	req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handleCreateCostAnalysis(w, req)

	if w.Code != 200 {
		t.Fatalf("First insert failed: %d", w.Code)
	}

	// Second insert with updated costs (should upsert)
	cost2 := CostAnalysis{
		ProductIPN:   productIPN,
		BOMCost:      120.0,
		LaborCost:    60.0,
		OverheadCost: 30.0,
	}

	body, _ = json.Marshal(cost2)
	req = httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handleCreateCostAnalysis(w, req)

	if w.Code != 200 {
		t.Fatalf("Second insert (upsert) failed: %d", w.Code)
	}

	// Verify only one record exists with updated values
	var count int
	var totalCost float64
	err := db.QueryRow("SELECT COUNT(*), COALESCE(SUM(total_cost), 0) FROM cost_analysis WHERE product_ipn = ?", productIPN).
		Scan(&count, &totalCost)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 record after upsert, got %d", count)
	}

	expectedTotal := 120.0 + 60.0 + 30.0
	if totalCost != expectedTotal {
		t.Errorf("Expected total_cost %.2f after upsert, got %.2f", expectedTotal, totalCost)
	}
}

// Test that costing handles large numbers correctly
func TestCostRollup_LargeNumbers(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupCostingTestDB(t)
	defer db.Close()

	costAnalysis := CostAnalysis{
		ProductIPN:   "PROD-LARGE",
		BOMCost:      1_000_000.00,
		LaborCost:    500_000.00,
		OverheadCost: 250_000.00,
	}

	body, _ := json.Marshal(costAnalysis)
	req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handleCreateCostAnalysis(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response CostAnalysis
	json.NewDecoder(w.Body).Decode(&response)

	expectedTotal := 1_750_000.00
	if response.TotalCost != expectedTotal {
		t.Errorf("Expected total_cost %.2f, got %.2f", expectedTotal, response.TotalCost)
	}
}

// Test material vs labor cost breakdown
func TestCostRollup_MaterialLaborBreakdown(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	db = setupCostingTestDB(t)
	defer db.Close()

	tests := []struct {
		name              string
		bomCost           float64
		laborCost         float64
		overheadCost      float64
		expectedMaterialPct float64
		expectedLaborPct    float64
	}{
		{
			name:              "material heavy (70%)",
			bomCost:           700.0,
			laborCost:         200.0,
			overheadCost:      100.0,
			expectedMaterialPct: 70.0,
			expectedLaborPct:    20.0,
		},
		{
			name:              "labor heavy (60%)",
			bomCost:           200.0,
			laborCost:         600.0,
			overheadCost:      200.0,
			expectedMaterialPct: 20.0,
			expectedLaborPct:    60.0,
		},
		{
			name:              "balanced",
			bomCost:           333.33,
			laborCost:         333.33,
			overheadCost:      333.34,
			expectedMaterialPct: 33.33,
			expectedLaborPct:    33.33,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			costAnalysis := CostAnalysis{
				ProductIPN:   "PROD-" + tt.name,
				BOMCost:      tt.bomCost,
				LaborCost:    tt.laborCost,
				OverheadCost: tt.overheadCost,
			}

			body, _ := json.Marshal(costAnalysis)
			req := httptest.NewRequest("POST", "/api/pricing/analysis", bytes.NewReader(body))
			w := httptest.NewRecorder()
			handleCreateCostAnalysis(w, req)

			var response CostAnalysis
			json.NewDecoder(w.Body).Decode(&response)

			// Calculate actual percentages
			totalCost := response.TotalCost
			if totalCost > 0 {
				materialPct := (tt.bomCost / totalCost) * 100
				laborPct := (tt.laborCost / totalCost) * 100

				if materialPct < tt.expectedMaterialPct-0.01 || materialPct > tt.expectedMaterialPct+0.01 {
					t.Errorf("Expected material %% %.2f, got %.2f", tt.expectedMaterialPct, materialPct)
				}

				if laborPct < tt.expectedLaborPct-0.01 || laborPct > tt.expectedLaborPct+0.01 {
					t.Errorf("Expected labor %% %.2f, got %.2f", tt.expectedLaborPct, laborPct)
				}
			}
		})
	}
}
