package main

import (
	"net/http"

	"zrp/internal/handlers/engineering"
)

// Type aliases for backward compatibility.
type MarketPricingResult = engineering.MarketPricingResult
type PriceBreak = engineering.PriceBreak
type DistributorClient = engineering.DistributorClient

// Wrapper functions for backward compatibility with tests and other root files.

func getAppSetting(key string) string {
	var val string
	db.QueryRow("SELECT value FROM app_settings WHERE key = ?", key).Scan(&val)
	return val
}

func setAppSetting(key, value string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO app_settings (key, value) VALUES (?, ?)", key, value)
	return err
}

func hasDistributorKeys() bool {
	return getEngineeringHandler().HasDistributorKeys()
}

func maskSetting(s string) string {
	return engineering.MaskSetting(s)
}

func getPartMPN(ipn string) string {
	// Parts are loaded from CSV files on disk
	cats, _, _, _ := loadPartsFromDir()
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

// Keep distributor client constructors accessible for tests.
func newDigikeyClient(clientID, clientSecret string) DistributorClient {
	return engineering.NewDigikeyClient(clientID, clientSecret)
}

func newMouserClient(apiKey string) DistributorClient {
	return engineering.NewMouserClient(apiKey)
}

// Keep helper functions accessible for tests.
func round2(f float64) float64 {
	return engineering.Round2(f)
}

func truncate(s string, maxLen int) string {
	return engineering.Truncate(s, maxLen)
}

func parseMouserPrice(s string) float64 {
	return engineering.ParseMouserPrice(s)
}

func parseMouserAvailability(s string) int {
	return engineering.ParseMouserAvailability(s)
}

func parseMouserLeadTime(s string) int {
	return engineering.ParseMouserLeadTime(s)
}

func getCachedPricing(partIPN string) ([]MarketPricingResult, error) {
	return getEngineeringHandler().GetCachedPricing(partIPN)
}

func cachePricingResult(r MarketPricingResult) error {
	return getEngineeringHandler().CachePricingResult(r)
}

func getDistributorClients() ([]DistributorClient, []string) {
	return getEngineeringHandler().GetDistributorClients()
}

// --- HTTP Handlers ---

func handleGetMarketPricing(w http.ResponseWriter, r *http.Request, partIPN string) {
	getEngineeringHandler().GetMarketPricing(w, r, partIPN)
}

func handleUpdateDigikeySettings(w http.ResponseWriter, r *http.Request) {
	getEngineeringHandler().UpdateDigikeySettings(w, r)
}

func handleUpdateMouserSettings(w http.ResponseWriter, r *http.Request) {
	getEngineeringHandler().UpdateMouserSettings(w, r)
}

func handleGetDistributorSettings(w http.ResponseWriter, r *http.Request) {
	getEngineeringHandler().GetDistributorSettings(w, r)
}
