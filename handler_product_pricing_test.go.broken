package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestProductPricingCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create pricing entry
	body := `{"product_ipn":"IPN-001","pricing_tier":"standard","min_qty":1,"max_qty":100,"unit_price":10.50,"currency":"USD","effective_date":"2024-01-01","notes":"Base price"}`
	req := authedRequest("POST", "/api/v1/pricing", body, loginAdmin(t))
	w := httptest.NewRecorder()
	handleCreateProductPricing(w, req)
	if w.Code != 200 {
		t.Fatalf("create pricing: %d %s", w.Code, w.Body.String())
	}
	var createResp struct {
		Data ProductPricing `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&createResp)
	if createResp.Data.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if createResp.Data.ProductIPN != "IPN-001" {
		t.Errorf("expected IPN-001, got %s", createResp.Data.ProductIPN)
	}
	if createResp.Data.UnitPrice != 10.50 {
		t.Errorf("expected 10.50, got %f", createResp.Data.UnitPrice)
	}
	id := createResp.Data.ID

	// Create a second tier
	body2 := `{"product_ipn":"IPN-001","pricing_tier":"volume","min_qty":100,"max_qty":1000,"unit_price":8.00,"currency":"USD","effective_date":"2024-01-01"}`
	req2 := authedRequest("POST", "/api/v1/pricing", body2, loginAdmin(t))
	w2 := httptest.NewRecorder()
	handleCreateProductPricing(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("create pricing 2: %d %s", w2.Code, w2.Body.String())
	}

	// List all pricing
	req3 := authedRequest("GET", "/api/v1/pricing", "", loginAdmin(t))
	w3 := httptest.NewRecorder()
	handleListProductPricing(w3, req3)
	if w3.Code != 200 {
		t.Fatalf("list pricing: %d %s", w3.Code, w3.Body.String())
	}
	var listResp struct {
		Data []ProductPricing `json:"data"`
	}
	json.NewDecoder(w3.Body).Decode(&listResp)
	if len(listResp.Data) < 2 {
		t.Fatalf("expected >=2 pricing entries, got %d", len(listResp.Data))
	}

	// Get single pricing
	req4 := authedRequest("GET", "/api/v1/pricing/1", "", loginAdmin(t))
	w4 := httptest.NewRecorder()
	handleGetProductPricing(w4, req4, "1")
	if w4.Code != 200 {
		t.Fatalf("get pricing: %d %s", w4.Code, w4.Body.String())
	}

	// Update pricing
	updateBody := `{"unit_price":11.00,"notes":"Updated price"}`
	req5 := authedRequest("PUT", "/api/v1/pricing/1", updateBody, loginAdmin(t))
	w5 := httptest.NewRecorder()
	handleUpdateProductPricing(w5, req5, "1")
	if w5.Code != 200 {
		t.Fatalf("update pricing: %d %s", w5.Code, w5.Body.String())
	}
	var updResp struct {
		Data ProductPricing `json:"data"`
	}
	json.NewDecoder(w5.Body).Decode(&updResp)
	if updResp.Data.UnitPrice != 11.00 {
		t.Errorf("expected 11.00 after update, got %f", updResp.Data.UnitPrice)
	}

	// Delete pricing
	req6 := authedRequest("DELETE", "/api/v1/pricing/1", "", loginAdmin(t))
	w6 := httptest.NewRecorder()
	handleDeleteProductPricing(w6, req6, "1")
	if w6.Code != 200 {
		t.Fatalf("delete pricing: %d %s", w6.Code, w6.Body.String())
	}

	// Verify deleted
	req7 := authedRequest("GET", "/api/v1/pricing/1", "", loginAdmin(t))
	w7 := httptest.NewRecorder()
	handleGetProductPricing(w7, req7, "1")
	if w7.Code != 404 {
		t.Errorf("expected 404 after delete, got %d", w7.Code)
	}

	_ = id
}

func TestProductPricingAnalysis(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create pricing
	body := `{"product_ipn":"IPN-001","pricing_tier":"standard","min_qty":1,"max_qty":100,"unit_price":15.00,"currency":"USD","effective_date":"2024-01-01"}`
	req := authedRequest("POST", "/api/v1/pricing", body, cookie)
	w := httptest.NewRecorder()
	handleCreateProductPricing(w, req)
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	// Create cost analysis
	costBody := `{"product_ipn":"IPN-001","bom_cost":5.00,"labor_cost":2.00,"overhead_cost":1.00}`
	req2 := authedRequest("POST", "/api/v1/pricing/analysis", costBody, cookie)
	w2 := httptest.NewRecorder()
	handleCreateCostAnalysis(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("create cost analysis: %d %s", w2.Code, w2.Body.String())
	}
	var caResp struct {
		Data CostAnalysis `json:"data"`
	}
	json.NewDecoder(w2.Body).Decode(&caResp)
	if caResp.Data.TotalCost != 8.00 {
		t.Errorf("expected total 8.00, got %f", caResp.Data.TotalCost)
	}

	// Get analysis list
	req3 := authedRequest("GET", "/api/v1/pricing/analysis", "", cookie)
	w3 := httptest.NewRecorder()
	handleListCostAnalysis(w3, req3)
	if w3.Code != 200 {
		t.Fatalf("list analysis: %d %s", w3.Code, w3.Body.String())
	}
	var analysisResp struct {
		Data []CostAnalysisWithPricing `json:"data"`
	}
	json.NewDecoder(w3.Body).Decode(&analysisResp)
	if len(analysisResp.Data) < 1 {
		t.Fatalf("expected >=1 analysis entries, got %d", len(analysisResp.Data))
	}
	entry := analysisResp.Data[0]
	if entry.MarginPct == 0 {
		t.Error("expected non-zero margin")
	}
}

func TestProductPricingHistory(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create two pricing entries for same IPN
	for _, price := range []string{"10.00", "12.00"} {
		body := `{"product_ipn":"IPN-001","pricing_tier":"standard","min_qty":1,"max_qty":100,"unit_price":` + price + `,"currency":"USD","effective_date":"2024-01-01"}`
		req := authedRequest("POST", "/api/v1/pricing", body, cookie)
		w := httptest.NewRecorder()
		handleCreateProductPricing(w, req)
		if w.Code != 200 {
			t.Fatalf("create: %d %s", w.Code, w.Body.String())
		}
	}

	// Get history
	req := authedRequest("GET", "/api/v1/pricing/history/IPN-001", "", cookie)
	w := httptest.NewRecorder()
	handleProductPricingHistory(w, req, "IPN-001")
	if w.Code != 200 {
		t.Fatalf("history: %d %s", w.Code, w.Body.String())
	}
	var histResp struct {
		Data []ProductPricing `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&histResp)
	if len(histResp.Data) < 2 {
		t.Fatalf("expected >=2 history entries, got %d", len(histResp.Data))
	}
}

func TestProductPricingBulkUpdate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create two entries
	for _, ipn := range []string{"IPN-001", "IPN-002"} {
		body := `{"product_ipn":"` + ipn + `","pricing_tier":"standard","min_qty":1,"max_qty":100,"unit_price":10.00,"currency":"USD","effective_date":"2024-01-01"}`
		req := authedRequest("POST", "/api/v1/pricing", body, cookie)
		w := httptest.NewRecorder()
		handleCreateProductPricing(w, req)
		if w.Code != 200 {
			t.Fatalf("create: %d", w.Code)
		}
	}

	// Bulk update - 10% increase
	bulkBody := `{"ids":[1,2],"adjustment_type":"percentage","adjustment_value":10}`
	req := authedRequest("POST", "/api/v1/pricing/bulk-update", bulkBody, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateProductPricing(w, req)
	if w.Code != 200 {
		t.Fatalf("bulk update: %d %s", w.Code, w.Body.String())
	}

	// Verify prices updated
	req2 := authedRequest("GET", "/api/v1/pricing/1", "", cookie)
	w2 := httptest.NewRecorder()
	handleGetProductPricing(w2, req2, "1")
	var resp struct {
		Data ProductPricing `json:"data"`
	}
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Data.UnitPrice != 11.00 {
		t.Errorf("expected 11.00 after 10%% increase, got %f", resp.Data.UnitPrice)
	}
}
