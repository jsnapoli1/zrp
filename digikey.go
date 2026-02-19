package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// digikeyClient implements DistributorClient for Digikey API v4
type digikeyClient struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func newDigikeyClient(clientID, clientSecret string) DistributorClient {
	return &digikeyClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (d *digikeyClient) Name() string { return "digikey" }

// authenticate obtains an OAuth2 client_credentials token from Digikey
func (d *digikeyClient) authenticate() (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.accessToken != "" && time.Now().Before(d.tokenExpiry) {
		return d.accessToken, nil
	}

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {d.clientID},
		"client_secret": {d.clientSecret},
	}

	resp, err := d.httpClient.Post(
		"https://api.digikey.com/v1/oauth2/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("digikey oauth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("digikey oauth error %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("digikey oauth decode error: %w", err)
	}

	d.accessToken = tokenResp.AccessToken
	d.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	return d.accessToken, nil
}

// Digikey API v4 response types
type digikeySearchResponse struct {
	Products          []digikeyProduct `json:"Products"`
	ProductsCount     int              `json:"ProductsCount"`
	ExactManufacturer bool             `json:"ExactManufacturerProducts"`
}

type digikeyProduct struct {
	DigiKeyPartNumber      string                  `json:"DigiKeyPartNumber"`
	ManufacturerPartNumber string                  `json:"ManufacturerPartNumber"`
	Manufacturer           digikeyManufacturer     `json:"Manufacturer"`
	ProductDescription     string                  `json:"ProductDescription"`
	QuantityAvailable      int                     `json:"QuantityAvailable"`
	ManufacturerLeadWeeks  string                  `json:"ManufacturerLeadWeeks"`
	UnitPrice              float64                 `json:"UnitPrice"`
	ProductURL             string                  `json:"ProductUrl"`
	DatasheetURL           string                  `json:"PrimaryDatasheet"`
	StandardPricing        []digikeyPriceBreak     `json:"StandardPricing"`
}

type digikeyManufacturer struct {
	Name string `json:"Name"`
}

type digikeyPriceBreak struct {
	BreakQuantity int     `json:"BreakQuantity"`
	UnitPrice     float64 `json:"UnitPrice"`
}

func (d *digikeyClient) Search(mpn string) ([]MarketPricingResult, error) {
	token, err := d.authenticate()
	if err != nil {
		return nil, err
	}

	reqBody := map[string]interface{}{
		"Keywords":         mpn,
		"RecordCount":      10,
		"RecordStartPosition": 0,
		"ExcludeMarketPlaceProducts": true,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.digikey.com/products/v4/search/keyword", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-DIGIKEY-Client-Id", d.clientID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DIGIKEY-Locale-Site", "US")
	req.Header.Set("X-DIGIKEY-Locale-Language", "en")
	req.Header.Set("X-DIGIKEY-Locale-Currency", "USD")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("digikey search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("digikey search error %d: %s", resp.StatusCode, string(body))
	}

	var searchResp digikeySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("digikey decode error: %w", err)
	}

	var results []MarketPricingResult
	for _, p := range searchResp.Products {
		var breaks []PriceBreak
		for _, pb := range p.StandardPricing {
			breaks = append(breaks, PriceBreak{
				Qty:       pb.BreakQuantity,
				UnitPrice: round2(pb.UnitPrice),
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
			PriceBreaks:   breaks,
			ProductURL:    p.ProductURL,
			DatasheetURL:  p.DatasheetURL,
			FetchedAt:     time.Now().UTC().Format(time.RFC3339),
		})
	}

	return results, nil
}
