package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDigikeyClientInterface(t *testing.T) {
	var _ DistributorClient = newDigikeyClient("id", "secret")
}

func TestMouserClientInterface(t *testing.T) {
	var _ DistributorClient = newMouserClient("key")
}

func TestDigikeyClientName(t *testing.T) {
	client := newDigikeyClient("id", "secret")
	if client.Name() != "digikey" {
		t.Fatalf("expected digikey, got %s", client.Name())
	}
}

func TestMouserClientName(t *testing.T) {
	client := newMouserClient("key")
	if client.Name() != "mouser" {
		t.Fatalf("expected mouser, got %s", client.Name())
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

	// Update Digikey settings (now uses client_id + client_secret)
	req := authedRequest("POST", "/api/v1/settings/digikey", `{"client_id":"cid-456","client_secret":"secret-789"}`, cookie)
	w := httptest.NewRecorder()
	handleUpdateDigikeySettings(w, req)
	if w.Code != 200 {
		t.Fatalf("digikey settings: %d %s", w.Code, w.Body.String())
	}

	if getAppSetting("digikey_client_id") != "cid-456" {
		t.Error("digikey client id not stored")
	}
	if getAppSetting("digikey_client_secret") != "secret-789" {
		t.Error("digikey client secret not stored")
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
	if !strings.Contains(resp["digikey"]["client_secret"], "****") {
		t.Error("client secret should be masked")
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

func TestHasDistributorKeys(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// No keys configured
	if hasDistributorKeys() {
		t.Error("expected false with no keys")
	}

	// Only Mouser
	setAppSetting("mouser_api_key", "test-key")
	if !hasDistributorKeys() {
		t.Error("expected true with mouser key")
	}

	// Reset and set Digikey
	setAppSetting("mouser_api_key", "")
	setAppSetting("digikey_client_id", "id")
	setAppSetting("digikey_client_secret", "secret")
	if !hasDistributorKeys() {
		t.Error("expected true with digikey keys")
	}
}

func TestGetDistributorClientsNoKeys(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	clients := getDistributorClients()
	if len(clients) != 0 {
		t.Errorf("expected 0 clients without keys, got %d", len(clients))
	}
}

func TestGetDistributorClientsWithKeys(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	setAppSetting("digikey_client_id", "id")
	setAppSetting("digikey_client_secret", "secret")
	setAppSetting("mouser_api_key", "key")

	clients := getDistributorClients()
	if len(clients) != 2 {
		t.Errorf("expected 2 clients, got %d", len(clients))
	}
}

func TestHandleMarketPricingNotConfigured(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := authedRequest("GET", "/api/v1/parts/IPN-001/market-pricing", "", loginAdmin(t))
	w := httptest.NewRecorder()
	handleGetMarketPricing(w, req, "IPN-001")

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["not_configured"] != true {
		t.Error("expected not_configured=true when no API keys set")
	}
}

func TestMouserParseHelpers(t *testing.T) {
	if parseMouserPrice("$1.23") != 1.23 {
		t.Error("parseMouserPrice failed")
	}
	if parseMouserPrice("1,234.56") != 1234.56 {
		t.Error("parseMouserPrice with commas failed")
	}
	if parseMouserAvailability("1,234 In Stock") != 1234 {
		t.Error("parseMouserAvailability failed")
	}
	if parseMouserAvailability("0") != 0 {
		t.Error("parseMouserAvailability zero failed")
	}
	if parseMouserLeadTime("14 Days") != 14 {
		t.Error("parseMouserLeadTime days failed")
	}
	if parseMouserLeadTime("2 Weeks") != 14 {
		t.Error("parseMouserLeadTime weeks failed")
	}
}
