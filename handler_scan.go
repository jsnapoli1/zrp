package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ScanResult struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Label string `json:"label"`
	Link  string `json:"link"`
}

func handleScanLookup(w http.ResponseWriter, r *http.Request, code string) {
	if code == "" {
		http.Error(w, `{"error":"missing code"}`, http.StatusBadRequest)
		return
	}

	var results []ScanResult
	codeLower := strings.ToLower(code)

	// Search parts by IPN
	cats, _, _ := loadPartsFromDir()
	for _, parts := range cats {
		for _, p := range parts {
			if strings.EqualFold(p.IPN, code) || strings.Contains(strings.ToLower(p.IPN), codeLower) {
				results = append(results, ScanResult{
					Type:  "part",
					ID:    p.IPN,
					Label: fmt.Sprintf("%s - %s", p.IPN, p.Fields["description"]),
					Link:  fmt.Sprintf("/parts/%s", p.IPN),
				})
			}
		}
	}

	// Search inventory
	rows, err := db.Query(`SELECT ipn, location, qty FROM inventory WHERE LOWER(ipn) = LOWER(?) OR LOWER(ipn) LIKE ?`, code, "%"+codeLower+"%")
	if err == nil {
		defer rows.Close()
		seen := map[string]bool{}
		for rows.Next() {
			var ipn, loc string
			var qty float64
			rows.Scan(&ipn, &loc, &qty)
			if !seen[ipn] {
				seen[ipn] = true
				results = append(results, ScanResult{
					Type:  "inventory",
					ID:    ipn,
					Label: fmt.Sprintf("%s (Qty: %.0f, Loc: %s)", ipn, qty, loc),
					Link:  fmt.Sprintf("/inventory/%s", ipn),
				})
			}
		}
	}

	// Search devices by serial number
	devRows, err := db.Query(`SELECT serial_number, model, status FROM devices WHERE LOWER(serial_number) = LOWER(?) OR LOWER(serial_number) LIKE ?`, code, "%"+codeLower+"%")
	if err == nil {
		defer devRows.Close()
		for devRows.Next() {
			var sn, model, status string
			devRows.Scan(&sn, &model, &status)
			results = append(results, ScanResult{
				Type:  "device",
				ID:    sn,
				Label: fmt.Sprintf("%s - %s (%s)", sn, model, status),
				Link:  fmt.Sprintf("/devices/%s", sn),
			})
		}
	}

	if results == nil {
		results = []ScanResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": results, "code": code})
}
