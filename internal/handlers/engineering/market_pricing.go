package engineering

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

)

// MarketPricingResult represents pricing data from a distributor.
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

// PriceBreak represents a quantity-based price tier.
type PriceBreak struct {
	Qty       int     `json:"qty"`
	UnitPrice float64 `json:"unit_price"`
}

// DistributorClient is the interface for distributor API integrations.
// Implement this interface to add new distributor backends.
type DistributorClient interface {
	Search(mpn string) ([]MarketPricingResult, error)
	Name() string
}

// distributorHTTPClient is the shared HTTP client with sensible timeouts for distributor APIs.
var distributorHTTPClient = &http.Client{Timeout: 15 * time.Second}

// --- Digikey v4 Product Search API Client ---

type digikeyClient struct {
	clientID     string
	clientSecret string
}

// NewDigikeyClient creates a new Digikey API client.
func NewDigikeyClient(clientID, clientSecret string) DistributorClient {
	return &digikeyClient{clientID: clientID, clientSecret: clientSecret}
}

func (d *digikeyClient) Name() string { return "digikey" }

func (d *digikeyClient) getToken() (string, error) {
	body := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=client_credentials",
		d.clientID, d.clientSecret)
	req, err := http.NewRequest("POST", "https://api.digikey.com/v1/oauth2/token",
		strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := distributorHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("digikey oauth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("digikey oauth error %d: %s", resp.StatusCode, string(b))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("digikey token decode error: %w", err)
	}
	return tokenResp.AccessToken, nil
}

func (d *digikeyClient) Search(mpn string) ([]MarketPricingResult, error) {
	token, err := d.getToken()
	if err != nil {
		return nil, fmt.Errorf("digikey auth failed: %w", err)
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"Keywords": mpn,
		"Limit":    10,
		"Offset":   0,
		"FilterOptionsRequest": map[string]interface{}{
			"ManufacturerFilter": []interface{}{},
			"MinimumQuantity":    0,
			"ParameterFilterRequest": map[string]interface{}{
				"CategoryFilter":   nil,
				"FitFilters":       []interface{}{},
				"ParameterFilters": []interface{}{},
			},
		},
		"ExcludeMarketPlaceProducts": false,
	})

	req, err := http.NewRequest("POST", "https://api.digikey.com/products/v4/search/keyword",
		bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-DIGIKEY-Client-Id", d.clientID)

	resp, err := distributorHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("digikey search request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("digikey rate limited — retry later")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("digikey search error %d: %s", resp.StatusCode, Truncate(string(respBody), 500))
	}

	var dkResp struct {
		Products []struct {
			DigiKeyPartNumber      string `json:"DigiKeyPartNumber"`
			ManufacturerPartNumber string `json:"ManufacturerPartNumber"`
			Manufacturer           struct {
				Name string `json:"Name"`
			} `json:"Manufacturer"`
			ProductDescription    string `json:"ProductDescription"`
			QuantityAvailable     int    `json:"QuantityAvailable"`
			ManufacturerLeadWeeks string `json:"ManufacturerLeadWeeks"`
			ProductUrl            string `json:"ProductUrl"`
			DatasheetUrl          string `json:"DatasheetUrl"`
			StandardPricing       []struct {
				BreakQuantity int     `json:"BreakQuantity"`
				UnitPrice     float64 `json:"UnitPrice"`
			} `json:"StandardPricing"`
		} `json:"Products"`
		ProductsCount int `json:"ProductsCount"`
	}
	if err := json.Unmarshal(respBody, &dkResp); err != nil {
		return nil, fmt.Errorf("digikey response parse error: %w", err)
	}

	var results []MarketPricingResult
	for _, p := range dkResp.Products {
		var pbs []PriceBreak
		for _, sp := range p.StandardPricing {
			pbs = append(pbs, PriceBreak{
				Qty:       sp.BreakQuantity,
				UnitPrice: Round2(sp.UnitPrice),
			})
		}

		leadDays := 0
		if weeks, err := strconv.Atoi(p.ManufacturerLeadWeeks); err == nil {
			leadDays = weeks * 7
		}

		results = append(results, MarketPricingResult{
			MPN:           p.ManufacturerPartNumber,
			Distributor:   "Digikey",
			DistributorPN: p.DigiKeyPartNumber,
			Manufacturer:  p.Manufacturer.Name,
			Description:   p.ProductDescription,
			StockQty:      p.QuantityAvailable,
			LeadTimeDays:  leadDays,
			Currency:      "USD",
			PriceBreaks:   pbs,
			ProductURL:    p.ProductUrl,
			DatasheetURL:  p.DatasheetUrl,
			FetchedAt:     time.Now().UTC().Format(time.RFC3339),
		})
	}
	return results, nil
}

// --- Mouser API v2 Client ---

type mouserClient struct {
	apiKey string
}

// NewMouserClient creates a new Mouser API client.
func NewMouserClient(apiKey string) DistributorClient {
	return &mouserClient{apiKey: apiKey}
}

func (m *mouserClient) Name() string { return "mouser" }

func (m *mouserClient) Search(mpn string) ([]MarketPricingResult, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"SearchByPartRequest": map[string]interface{}{
			"mouserPartNumber":  mpn,
			"partSearchOptions": "",
		},
	})

	url := fmt.Sprintf("https://api.mouser.com/api/v2/search/partnumber?apiKey=%s", m.apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := distributorHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mouser search request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("mouser rate limited — retry later")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mouser search error %d: %s", resp.StatusCode, Truncate(string(respBody), 500))
	}

	var mouserResp struct {
		Errors []struct {
			Id      int    `json:"Id"`
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Errors"`
		SearchResults struct {
			NumberOfResult int `json:"NumberOfResult"`
			Parts          []struct {
				MouserPartNumber       string `json:"MouserPartNumber"`
				ManufacturerPartNumber string `json:"ManufacturerPartNumber"`
				Manufacturer           string `json:"Manufacturer"`
				Description            string `json:"Description"`
				Availability           string `json:"Availability"`
				LeadTime               string `json:"LeadTime"`
				ProductDetailUrl       string `json:"ProductDetailUrl"`
				DataSheetUrl           string `json:"DataSheetUrl"`
				PriceBreaks            []struct {
					Quantity int    `json:"Quantity"`
					Price    string `json:"Price"`
					Currency string `json:"Currency"`
				} `json:"PriceBreaks"`
			} `json:"Parts"`
		} `json:"SearchResults"`
	}
	if err := json.Unmarshal(respBody, &mouserResp); err != nil {
		return nil, fmt.Errorf("mouser response parse error: %w", err)
	}

	if len(mouserResp.Errors) > 0 {
		return nil, fmt.Errorf("mouser API error: %s", mouserResp.Errors[0].Message)
	}

	var results []MarketPricingResult
	for _, p := range mouserResp.SearchResults.Parts {
		var pbs []PriceBreak
		for _, pb := range p.PriceBreaks {
			price := ParseMouserPrice(pb.Price)
			if price > 0 {
				pbs = append(pbs, PriceBreak{
					Qty:       pb.Quantity,
					UnitPrice: Round2(price),
				})
			}
		}

		stock := ParseMouserAvailability(p.Availability)
		leadDays := ParseMouserLeadTime(p.LeadTime)

		results = append(results, MarketPricingResult{
			MPN:           p.ManufacturerPartNumber,
			Distributor:   "Mouser",
			DistributorPN: p.MouserPartNumber,
			Manufacturer:  p.Manufacturer,
			Description:   p.Description,
			StockQty:      stock,
			LeadTimeDays:  leadDays,
			Currency:      "USD",
			PriceBreaks:   pbs,
			ProductURL:    p.ProductDetailUrl,
			DatasheetURL:  p.DataSheetUrl,
			FetchedAt:     time.Now().UTC().Format(time.RFC3339),
		})
	}
	return results, nil
}

// ParseMouserPrice handles Mouser's price format like "$1.23" or "1.23".
func ParseMouserPrice(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.Replace(s, "$", "", 1)
	s = strings.Replace(s, ",", "", -1)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// ParseMouserAvailability parses "15,000 In Stock" -> 15000.
func ParseMouserAvailability(s string) int {
	s = strings.TrimSpace(s)
	s = strings.Split(s, " ")[0]
	s = strings.Replace(s, ",", "", -1)
	n, _ := strconv.Atoi(s)
	return n
}

// ParseMouserLeadTime parses "14 Days" or "2 Weeks" -> days.
func ParseMouserLeadTime(s string) int {
	s = strings.TrimSpace(strings.ToLower(s))
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	if strings.Contains(parts[1], "week") {
		return n * 7
	}
	return n // assume days
}

// --- Helpers ---

// Round2 rounds a float to 2 decimal places.
func Round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// Truncate truncates a string to maxLen characters, adding "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// --- DB Cache ---

// GetCachedPricing returns cached pricing results for a part IPN.
func (h *Handler) GetCachedPricing(partIPN string) ([]MarketPricingResult, error) {
	cutoff := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	rows, err := h.DB.Query(`SELECT id, part_ipn, mpn, distributor, distributor_pn, manufacturer,
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

// CachePricingResult caches a market pricing result.
func (h *Handler) CachePricingResult(r MarketPricingResult) error {
	pb, _ := json.Marshal(r.PriceBreaks)
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO market_pricing
		(part_ipn, mpn, distributor, distributor_pn, manufacturer, description,
		 stock_qty, lead_time_days, currency, price_breaks, product_url, datasheet_url, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.PartIPN, r.MPN, r.Distributor, r.DistributorPN, r.Manufacturer, r.Description,
		r.StockQty, r.LeadTimeDays, r.Currency, string(pb), r.ProductURL, r.DatasheetURL, r.FetchedAt)
	return err
}

// --- Distributor registry ---

// GetDistributorClients returns configured distributor clients and unconfigured names.
func (h *Handler) GetDistributorClients() ([]DistributorClient, []string) {
	var clients []DistributorClient
	var unconfigured []string

	// Load Digikey config from app_settings
	dkClientID := h.GetAppSetting("digikey_client_id")
	dkClientSecret := h.GetAppSetting("digikey_client_secret")
	if dkClientID != "" && dkClientSecret != "" {
		clients = append(clients, NewDigikeyClient(dkClientID, dkClientSecret))
	} else {
		unconfigured = append(unconfigured, "Digikey")
	}

	// Load Mouser config from app_settings
	mouserKey := h.GetAppSetting("mouser_api_key")
	if mouserKey != "" {
		clients = append(clients, NewMouserClient(mouserKey))
	} else {
		unconfigured = append(unconfigured, "Mouser")
	}

	return clients, unconfigured
}

// HasDistributorKeys returns true if at least one distributor API is configured.
func (h *Handler) HasDistributorKeys() bool {
	dkID := h.GetAppSetting("digikey_client_id")
	dkSecret := h.GetAppSetting("digikey_client_secret")
	mouserKey := h.GetAppSetting("mouser_api_key")
	return (dkID != "" && dkSecret != "") || mouserKey != ""
}

// MaskSetting masks a setting value for display.
func MaskSetting(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// --- HTTP Handlers ---

// GetMarketPricing handles GET /api/parts/:ipn/market-pricing.
func (h *Handler) GetMarketPricing(w http.ResponseWriter, r *http.Request, partIPN string) {
	// Get part's MPN from fields
	mpn := h.GetPartMPN(partIPN)
	if mpn == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []MarketPricingResult{},
			"error":   "Part has no MPN set",
		})
		return
	}

	forceRefresh := r.URL.Query().Get("refresh") == "true"

	if !forceRefresh {
		cached, err := h.GetCachedPricing(partIPN)
		if err == nil && len(cached) > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{"results": cached, "cached": true})
			return
		}
	}

	clients, unconfigured := h.GetDistributorClients()

	if len(clients) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results":        []MarketPricingResult{},
			"cached":         false,
			"not_configured": true,
			"error":          "No distributor API keys configured. Go to Settings > Distributor API Settings to add your Digikey and/or Mouser API credentials.",
			"unconfigured":   unconfigured,
		})
		return
	}

	// Fetch from all configured distributors
	var results []MarketPricingResult
	var errors []string
	for _, c := range clients {
		res, err := c.Search(mpn)
		if err != nil {
			log.Printf("market pricing: %s search for %q failed: %v", c.Name(), mpn, err)
			errors = append(errors, fmt.Sprintf("%s: %v", c.Name(), err))
			continue
		}
		for i := range res {
			res[i].PartIPN = partIPN
			h.CachePricingResult(res[i])
		}
		results = append(results, res...)
	}

	resp := map[string]interface{}{"results": results, "cached": false}
	if len(unconfigured) > 0 {
		resp["unconfigured"] = unconfigured
	}
	if len(errors) > 0 {
		resp["errors"] = errors
	}
	json.NewEncoder(w).Encode(resp)
}

// UpdateDigikeySettings handles PUT /api/settings/digikey.
func (h *Handler) UpdateDigikeySettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400)
		return
	}
	if err := h.SetAppSetting("digikey_client_id", body.ClientID); err != nil {
		http.Error(w, `{"error":"failed to save"}`, 500)
		return
	}
	if err := h.SetAppSetting("digikey_client_secret", body.ClientSecret); err != nil {
		http.Error(w, `{"error":"failed to save"}`, 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// UpdateMouserSettings handles PUT /api/settings/mouser.
func (h *Handler) UpdateMouserSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400)
		return
	}
	if err := h.SetAppSetting("mouser_api_key", body.APIKey); err != nil {
		http.Error(w, `{"error":"failed to save"}`, 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetDistributorSettings handles GET /api/settings/distributors.
func (h *Handler) GetDistributorSettings(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"digikey": map[string]string{
			"client_id":     MaskSetting(h.GetAppSetting("digikey_client_id")),
			"client_secret": MaskSetting(h.GetAppSetting("digikey_client_secret")),
		},
		"mouser": map[string]string{
			"api_key": MaskSetting(h.GetAppSetting("mouser_api_key")),
		},
	})
}
