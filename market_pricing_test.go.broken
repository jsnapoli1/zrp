package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDistributorClientInterface(t *testing.T) {
	var _ DistributorClient = newDigikeyClient("", "")
	var _ DistributorClient = newMouserClient("")
}

func TestDigikeyClientName(t *testing.T) {
	c := newDigikeyClient("cid", "secret")
	if c.Name() != "digikey" {
		t.Errorf("expected digikey, got %s", c.Name())
	}
}

func TestMouserClientName(t *testing.T) {
	c := newMouserClient("key")
	if c.Name() != "mouser" {
		t.Errorf("expected mouser, got %s", c.Name())
	}
}

func TestParseMouserPrice(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"$1.23", 1.23},
		{"1.23", 1.23},
		{"$0.0045", 0.0045},
		{"$1,234.56", 1234.56},
		{"", 0},
		{"  $5.00  ", 5.00},
	}
	for _, tc := range tests {
		got := parseMouserPrice(tc.input)
		if got != tc.expected {
			t.Errorf("parseMouserPrice(%q) = %f, want %f", tc.input, got, tc.expected)
		}
	}
}

func TestParseMouserAvailability(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"15,000 In Stock", 15000},
		{"500 In Stock", 500},
		{"0 In Stock", 0},
		{"", 0},
		{"None", 0},
		{"1234", 1234},
	}
	for _, tc := range tests {
		got := parseMouserAvailability(tc.input)
		if got != tc.expected {
			t.Errorf("parseMouserAvailability(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestParseMouserLeadTime(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"14 Days", 14},
		{"2 Weeks", 14},
		{"4 weeks", 28},
		{"", 0},
		{"unknown", 0},
	}
	for _, tc := range tests {
		got := parseMouserLeadTime(tc.input)
		if got != tc.expected {
			t.Errorf("parseMouserLeadTime(%q) = %d, want %d", tc.input, got, tc.expected)
		}
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
		FetchedAt:   "2020-01-01T00:00:00Z",
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
	req := authedRequest("POST", "/api/v1/settings/digikey",
		`{"client_id":"cid-456","client_secret":"secret-789"}`, cookie)
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

func TestGetDistributorClientsUnconfigured(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	clients, unconfigured := getDistributorClients()
	if len(clients) != 0 {
		t.Error("expected no clients when no keys configured")
	}
	if len(unconfigured) != 2 {
		t.Errorf("expected 2 unconfigured, got %d", len(unconfigured))
	}
}

func TestGetDistributorClientsPartialConfig(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	setAppSetting("mouser_api_key", "test-key")
	clients, unconfigured := getDistributorClients()
	if len(clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(clients))
	}
	if clients[0].Name() != "mouser" {
		t.Errorf("expected mouser client, got %s", clients[0].Name())
	}
	if len(unconfigured) != 1 || unconfigured[0] != "Digikey" {
		t.Errorf("expected Digikey unconfigured, got %v", unconfigured)
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

func TestMarketPricingHandlerNoMPN(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := authedRequest("GET", "/api/v1/parts/IPN-001/market-pricing", "", loginAdmin(t))
	w := httptest.NewRecorder()
	handleGetMarketPricing(w, req, "IPN-001")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "Part has no MPN set" {
		t.Errorf("expected no MPN error, got: %v", resp)
	}
}

func TestRound2(t *testing.T) {
	if round2(1.555) != 1.56 {
		t.Errorf("round2(1.555) = %f", round2(1.555))
	}
	if round2(1.004) != 1.0 {
		t.Errorf("round2(1.004) = %f", round2(1.004))
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Error("should not truncate short string")
	}
	if truncate("hello world", 5) != "hello..." {
		t.Errorf("got %q", truncate("hello world", 5))
	}
}
