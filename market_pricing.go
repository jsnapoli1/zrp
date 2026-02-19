package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"
)

// MarketPricingResult represents pricing data from a distributor
type MarketPricingResult struct {
	ID            int          `json:"id"`
	PartIPN       string       `json:"part_ipn"`
	MPN           string       `json:"mpn"`
	Distributor   string       `json:"distributor"`
	DistributorPN string       `json:"distributor_pn"`
	Manufacturer  string       `json:"manufacturer"`
	Description   string       `json:"description"`
	StockQty      int          `json:"stock_qty"`
	LeadTimeDays  int          `json:"lead_time_days"`
	Currency      string       `json:"currency"`
	PriceBreaks   []PriceBreak `json:"price_breaks"`
	ProductURL    string       `json:"product_url"`
	DatasheetURL  string       `json:"datasheet_url"`
	FetchedAt     string       `json:"fetched_at"`
}

// PriceBreak represents a quantity-based price tier
type PriceBreak struct {
	Qty       int     `json:"qty"`
	UnitPrice float64 `json:"unit_price"`
}

// DistributorClient is the interface for distributor API integrations
type DistributorClient interface {
	Search(mpn string) ([]MarketPricingResult, error)
	Name() string
}

// --- Digikey Mock Client ---

type digikeyClient struct {
	apiKey    string
	clientID  string
}

func newDigikeyClient(apiKey, clientID string) DistributorClient {
	return &digikeyClient{apiKey: apiKey, clientID: clientID}
}

func (d *digikeyClient) Name() string { return "digikey" }

func (d *digikeyClient) Search(mpn string) ([]MarketPricingResult, error) {
	// Mock implementation - replace with real Digikey API v3 call
	// Real: POST https://api.digikey.com/Search/v3/Products/Keyword
	// Headers: X-DIGIKEY-Client-Id, Authorization: Bearer <token>
	r := rand.New(rand.NewSource(hashString(mpn)))
	basePrice := 0.50 + r.Float64()*20.0
	stock := r.Intn(50000)
	leadTime := 7 + r.Intn(21)

	return []MarketPricingResult{{
		MPN:           mpn,
		Distributor:   "Digikey",
		DistributorPN: fmt.Sprintf("DK-%s-ND", mpn),
		Manufacturer:  mockManufacturer(mpn, r),
		Description:   fmt.Sprintf("%s (Digikey)", mpn),
		StockQty:      stock,
		LeadTimeDays:  leadTime,
		Currency:      "USD",
		PriceBreaks: []PriceBreak{
			{Qty: 1, UnitPrice: round2(basePrice)},
			{Qty: 10, UnitPrice: round2(basePrice * 0.90)},
			{Qty: 100, UnitPrice: round2(basePrice * 0.80)},
			{Qty: 1000, UnitPrice: round2(basePrice * 0.65)},
			{Qty: 5000, UnitPrice: round2(basePrice * 0.55)},
		},
		ProductURL:   fmt.Sprintf("https://www.digikey.com/product-detail/%s", mpn),
		DatasheetURL: fmt.Sprintf("https://www.digikey.com/datasheet/%s.pdf", mpn),
		FetchedAt:    time.Now().UTC().Format(time.RFC3339),
	}}, nil
}

// --- Mouser Mock Client ---

type mouserClient struct {
	apiKey string
}

func newMouserClient(apiKey string) DistributorClient {
	return &mouserClient{apiKey: apiKey}
}

func (m *mouserClient) Name() string { return "mouser" }

func (m *mouserClient) Search(mpn string) ([]MarketPricingResult, error) {
	// Mock implementation - replace with real Mouser API call
	// Real: POST https://api.mouser.com/api/v1/search/partnumber?apiKey=<key>
	r := rand.New(rand.NewSource(hashString(mpn) + 1))
	basePrice := 0.45 + r.Float64()*22.0
	stock := r.Intn(40000)
	leadTime := 5 + r.Intn(28)

	return []MarketPricingResult{{
		MPN:           mpn,
		Distributor:   "Mouser",
		DistributorPN: fmt.Sprintf("MOU-%s", mpn),
		Manufacturer:  mockManufacturer(mpn, r),
		Description:   fmt.Sprintf("%s (Mouser)", mpn),
		StockQty:      stock,
		LeadTimeDays:  leadTime,
		Currency:      "USD",
		PriceBreaks: []PriceBreak{
			{Qty: 1, UnitPrice: round2(basePrice)},
			{Qty: 10, UnitPrice: round2(basePrice * 0.88)},
			{Qty: 100, UnitPrice: round2(basePrice * 0.78)},
			{Qty: 1000, UnitPrice: round2(basePrice * 0.62)},
			{Qty: 2500, UnitPrice: round2(basePrice * 0.52)},
		},
		ProductURL:   fmt.Sprintf("https://www.mouser.com/ProductDetail/%s", mpn),
		DatasheetURL: fmt.Sprintf("https://www.mouser.com/datasheet/%s.pdf", mpn),
		FetchedAt:    time.Now().UTC().Format(time.RFC3339),
	}}, nil
}

// --- Helpers ---

func hashString(s string) int64 {
	var h int64
	for _, c := range s {
		h = h*31 + int64(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

func mockManufacturer(mpn string, r *rand.Rand) string {
	mfgs := []string{"Texas Instruments", "STMicroelectronics", "Murata", "TDK", "Yageo", "Samsung", "Vishay", "ON Semiconductor"}
	return mfgs[r.Intn(len(mfgs))]
}

// --- DB Cache ---

func getCachedPricing(partIPN string) ([]MarketPricingResult, error) {
	cutoff := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	rows, err := db.Query(`SELECT id, part_ipn, mpn, distributor, distributor_pn, manufacturer,
		description, stock_qty, lead_time_days, currency, price_breaks, product_url, datasheet_url, fetched_at
		FROM market_pricing WHERE part_ipn = ? AND fetched_at > ?`, partIPN, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MarketPricingResult
	for rows.Next() {
		var r MarketPricingResult
		var pb string
		err := rows.Scan(&r.ID, &r.PartIPN, &r.MPN, &r.Distributor, &r.DistributorPN,
			&r.Manufacturer, &r.Description, &r.StockQty, &r.LeadTimeDays,
			&r.Currency, &pb, &r.ProductURL, &r.DatasheetURL, &r.FetchedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(pb), &r.PriceBreaks)
		results = append(results, r)
	}
	return results, nil
}

func cachePricingResult(r MarketPricingResult) error {
	pb, _ := json.Marshal(r.PriceBreaks)
	_, err := db.Exec(`INSERT OR REPLACE INTO market_pricing
		(part_ipn, mpn, distributor, distributor_pn, manufacturer, description,
		 stock_qty, lead_time_days, currency, price_breaks, product_url, datasheet_url, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.PartIPN, r.MPN, r.Distributor, r.DistributorPN, r.Manufacturer, r.Description,
		r.StockQty, r.LeadTimeDays, r.Currency, string(pb), r.ProductURL, r.DatasheetURL, r.FetchedAt)
	return err
}

// --- Distributor registry ---

func getDistributorClients() []DistributorClient {
	var clients []DistributorClient

	// Load Digikey config from app_settings
	dkKey := getAppSetting("digikey_api_key")
	dkClient := getAppSetting("digikey_client_id")
	if dkKey != "" || dkClient != "" {
		clients = append(clients, newDigikeyClient(dkKey, dkClient))
	} else {
		// Use mock even without keys for demo
		clients = append(clients, newDigikeyClient("mock", "mock"))
	}

	// Load Mouser config from app_settings
	mouserKey := getAppSetting("mouser_api_key")
	if mouserKey != "" {
		clients = append(clients, newMouserClient(mouserKey))
	} else {
		clients = append(clients, newMouserClient("mock"))
	}

	return clients
}

func getAppSetting(key string) string {
	var val string
	db.QueryRow("SELECT value FROM app_settings WHERE key = ?", key).Scan(&val)
	return val
}

func setAppSetting(key, value string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO app_settings (key, value) VALUES (?, ?)", key, value)
	return err
}

// --- HTTP Handlers ---

func handleGetMarketPricing(w http.ResponseWriter, r *http.Request, partIPN string) {
	// Get part's MPN from fields
	mpn := getPartMPN(partIPN)
	if mpn == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []MarketPricingResult{},
			"error":   "Part has no MPN set",
		})
		return
	}

	forceRefresh := r.URL.Query().Get("refresh") == "true"

	if !forceRefresh {
		cached, err := getCachedPricing(partIPN)
		if err == nil && len(cached) > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{"results": cached, "cached": true})
			return
		}
	}

	// Fetch from all distributors
	clients := getDistributorClients()
	var results []MarketPricingResult
	for _, c := range clients {
		res, err := c.Search(mpn)
		if err != nil {
			continue
		}
		for i := range res {
			res[i].PartIPN = partIPN
			cachePricingResult(res[i])
		}
		results = append(results, res...)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"results": results, "cached": false})
}

func getPartMPN(ipn string) string {
	// Parts are loaded from CSV files on disk
	cats, _, _ := loadPartsFromDir()
	for _, parts := range cats {
		for _, p := range parts {
			if p.IPN == ipn {
				if mpn, ok := p.Fields["mpn"]; ok && mpn != "" {
					return mpn
				}
				if mpn, ok := p.Fields["manufacturer_part_number"]; ok && mpn != "" {
					return mpn
				}
				return ""
			}
		}
	}
	return ""
}

func handleUpdateDigikeySettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		APIKey   string `json:"api_key"`
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400)
		return
	}
	if err := setAppSetting("digikey_api_key", body.APIKey); err != nil {
		http.Error(w, `{"error":"failed to save"}`, 500)
		return
	}
	if err := setAppSetting("digikey_client_id", body.ClientID); err != nil {
		http.Error(w, `{"error":"failed to save"}`, 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleUpdateMouserSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400)
		return
	}
	if err := setAppSetting("mouser_api_key", body.APIKey); err != nil {
		http.Error(w, `{"error":"failed to save"}`, 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGetDistributorSettings(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"digikey": map[string]string{
			"api_key":   maskSetting(getAppSetting("digikey_api_key")),
			"client_id": maskSetting(getAppSetting("digikey_client_id")),
		},
		"mouser": map[string]string{
			"api_key": maskSetting(getAppSetting("mouser_api_key")),
		},
	})
}

func maskSetting(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
