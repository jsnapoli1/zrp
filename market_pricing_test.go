package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDigikeyClientSearch(t *testing.T) {
	client := newDigikeyClient("test-key", "test-client")
	if client.Name() != "digikey" {
		t.Fatalf("expected digikey, got %s", client.Name())
	}
	results, err := client.Search("STM32F401")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	r := results[0]
	if r.MPN != "STM32F401" {
		t.Errorf("expected MPN STM32F401, got %s", r.MPN)
	}
	if r.Distributor != "Digikey" {
		t.Errorf("expected Digikey, got %s", r.Distributor)
	}
	if len(r.PriceBreaks) == 0 {
		t.Error("expected price breaks")
	}
	// Price breaks should be descending
	for i := 1; i < len(r.PriceBreaks); i++ {
		if r.PriceBreaks[i].UnitPrice >= r.PriceBreaks[i-1].UnitPrice {
			t.Errorf("price breaks not descending at index %d", i)
		}
	}
}

func TestMouserClientSearch(t *testing.T) {
	client := newMouserClient("test-key")
	if client.Name() != "mouser" {
		t.Fatalf("expected mouser, got %s", client.Name())
	}
	results, err := client.Search("STM32F401")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	r := results[0]
	if r.Distributor != "Mouser" {
		t.Errorf("expected Mouser, got %s", r.Distributor)
	}
}

func TestMockResultsDeterministic(t *testing.T) {
	client := newDigikeyClient("k", "c")
	r1, _ := client.Search("ABC123")
	r2, _ := client.Search("ABC123")
	if r1[0].StockQty != r2[0].StockQty {
		t.Error("mock results should be deterministic for same MPN")
	}
	if r1[0].PriceBreaks[0].UnitPrice != r2[0].PriceBreaks[0].UnitPrice {
		t.Error("mock prices should be deterministic")
	}
}

func TestMarketPricingCache(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	r := MarketPricingResult{
		PartIPN:      "IPN-001",
		MPN:          "TEST123",
		Distributor:  "Digikey",
		StockQty:     1000,
		LeadTimeDays: 14,
		Currency:     "USD",
		PriceBreaks:  []PriceBreak{{Qty: 1, UnitPrice: 1.50}, {Qty: 10, UnitPrice: 1.20}},
		FetchedAt:    "2099-01-01T00:00:00Z",
	}
	if err := cachePricingResult(r); err != nil {
		t.Fatal(err)
	}

	cached, err := getCachedPricing("IPN-001")
	if err != nil {
		t.Fatal(err)
	}
	if len(cached) != 1 {
		t.Fatalf("expected 1 cached result, got %d", len(cached))
	}
	if cached[0].StockQty != 1000 {
		t.Errorf("expected stock 1000, got %d", cached[0].StockQty)
	}
	if len(cached[0].PriceBreaks) != 2 {
		t.Errorf("expected 2 price breaks, got %d", len(cached[0].PriceBreaks))
	}
}

func TestMarketPricingCacheExpiry(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	r := MarketPricingResult{
		PartIPN:     "IPN-001",
		MPN:         "TEST123",
		Distributor: "Digikey",
		PriceBreaks: []PriceBreak{{Qty: 1, UnitPrice: 1.50}},
		FetchedAt:   "2020-01-01T00:00:00Z", // expired
	}
	cachePricingResult(r)

	cached, _ := getCachedPricing("IPN-001")
	if len(cached) != 0 {
		t.Error("expired cache should not be returned")
	}
}

func TestDistributorSettingsHandlers(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Update Digikey settings
	req := authedRequest("POST", "/api/v1/settings/digikey", `{"api_key":"dk-123","client_id":"cid-456"}`, cookie)
	w := httptest.NewRecorder()
	handleUpdateDigikeySettings(w, req)
	if w.Code != 200 {
		t.Fatalf("digikey settings: %d %s", w.Code, w.Body.String())
	}

	// Verify stored
	if getAppSetting("digikey_api_key") != "dk-123" {
		t.Error("digikey api key not stored")
	}
	if getAppSetting("digikey_client_id") != "cid-456" {
		t.Error("digikey client id not stored")
	}

	// Update Mouser settings
	req = authedRequest("POST", "/api/v1/settings/mouser", `{"api_key":"mou-789"}`, cookie)
	w = httptest.NewRecorder()
	handleUpdateMouserSettings(w, req)
	if w.Code != 200 {
		t.Fatalf("mouser settings: %d %s", w.Code, w.Body.String())
	}
	if getAppSetting("mouser_api_key") != "mou-789" {
		t.Error("mouser api key not stored")
	}

	// Get distributor settings (should be masked)
	req = authedRequest("GET", "/api/v1/settings/distributors", "", cookie)
	w = httptest.NewRecorder()
	handleGetDistributorSettings(w, req)
	var resp map[string]map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp["digikey"]["api_key"], "****") {
		t.Error("api key should be masked")
	}
}

func TestMaskSetting(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"", ""},
		{"short", "****"},
		{"abcdefghij", "abcd****ghij"},
	}
	for _, tc := range tests {
		got := maskSetting(tc.input)
		if got != tc.expected {
			t.Errorf("maskSetting(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestDistributorClientInterface(t *testing.T) {
	// Verify both clients satisfy the interface
	var _ DistributorClient = newDigikeyClient("", "")
	var _ DistributorClient = newMouserClient("")
}
