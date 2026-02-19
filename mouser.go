package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// mouserClient implements DistributorClient for Mouser API v2
type mouserClient struct {
	apiKey     string
	httpClient *http.Client
}

func newMouserClient(apiKey string) DistributorClient {
	return &mouserClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *mouserClient) Name() string { return "mouser" }

// Mouser API v2 response types
type mouserSearchResponse struct {
	Errors         []mouserError       `json:"Errors"`
	SearchResults  mouserSearchResults `json:"SearchResults"`
}

type mouserError struct {
	ID             int    `json:"Id"`
	Code           string `json:"Code"`
	Message        string `json:"Message"`
}

type mouserSearchResults struct {
	NumberOfResult int            `json:"NumberOfResult"`
	Parts          []mouserPart   `json:"Parts"`
}

type mouserPart struct {
	MouserPartNumber       string              `json:"MouserPartNumber"`
	ManufacturerPartNumber string              `json:"ManufacturerPartNumber"`
	Manufacturer           string              `json:"Manufacturer"`
	Description            string              `json:"Description"`
	Availability           string              `json:"Availability"`
	FactoryLeadTime        string              `json:"LeadTime"`
	DataSheetUrl           string              `json:"DataSheetUrl"`
	ProductDetailUrl       string              `json:"ProductDetailUrl"`
	PriceBreaks            []mouserPriceBreak  `json:"PriceBreaks"`
}

type mouserPriceBreak struct {
	Quantity int    `json:"Quantity"`
	Price    string `json:"Price"`
	Currency string `json:"Currency"`
}

func (m *mouserClient) Search(mpn string) ([]MarketPricingResult, error) {
	reqBody := map[string]interface{}{
		"SearchByPartRequest": map[string]interface{}{
			"mouserPartNumber": mpn,
			"partSearchOptions": "BeginsWith",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	apiURL := fmt.Sprintf("https://api.mouser.com/api/v2/search/partnumber?apiKey=%s", m.apiKey)
	resp, err := m.httpClient.Post(apiURL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("mouser search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mouser search error %d: %s", resp.StatusCode, string(body))
	}

	var searchResp mouserSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("mouser decode error: %w", err)
	}

	if len(searchResp.Errors) > 0 {
		return nil, fmt.Errorf("mouser API error: %s", searchResp.Errors[0].Message)
	}

	var results []MarketPricingResult
	for _, p := range searchResp.SearchResults.Parts {
		var breaks []PriceBreak
		for _, pb := range p.PriceBreaks {
			price := parseMouserPrice(pb.Price)
			breaks = append(breaks, PriceBreak{
				Qty:       pb.Quantity,
				UnitPrice: round2(price),
			})
		}

		stock := parseMouserAvailability(p.Availability)
		leadDays := parseMouserLeadTime(p.FactoryLeadTime)

		results = append(results, MarketPricingResult{
			MPN:           p.ManufacturerPartNumber,
			Distributor:   "Mouser",
			DistributorPN: p.MouserPartNumber,
			Manufacturer:  p.Manufacturer,
			Description:   p.Description,
			StockQty:      stock,
			LeadTimeDays:  leadDays,
			Currency:      "USD",
			PriceBreaks:   breaks,
			ProductURL:    p.ProductDetailUrl,
			DatasheetURL:  p.DataSheetUrl,
			FetchedAt:     time.Now().UTC().Format(time.RFC3339),
		})
	}

	return results, nil
}

// parseMouserPrice parses "$1.23" or "1.23" to float64
func parseMouserPrice(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseMouserAvailability extracts stock count from strings like "1,234 In Stock"
func parseMouserAvailability(s string) int {
	s = strings.TrimSpace(s)
	// Extract numeric portion
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return 0
	}
	numStr := strings.ReplaceAll(parts[0], ",", "")
	n, _ := strconv.Atoi(numStr)
	return n
}

// parseMouserLeadTime extracts days from strings like "14 Days" or "2 Weeks"
func parseMouserLeadTime(s string) int {
	s = strings.TrimSpace(strings.ToLower(s))
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(parts[0])
	if strings.Contains(parts[1], "week") {
		return n * 7
	}
	return n
}
