package main

import (
	"net/http"
)

func handleListPrices(w http.ResponseWriter, r *http.Request, ipn string) {
	getProcurementHandler().ListPrices(w, r, ipn)
}

func handleCreatePrice(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().CreatePrice(w, r)
}

func handleDeletePrice(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().DeletePrice(w, r, id)
}

func handlePriceTrend(w http.ResponseWriter, r *http.Request, ipn string) {
	getProcurementHandler().PriceTrend(w, r, ipn)
}

// recordPriceFromPO records a price history entry when a PO line is received.
// This is kept in the root package because it uses the global db and is passed
// as a function field to the procurement handler.
func recordPriceFromPO(poID, ipn string, unitPrice float64, vendorID string) {
	if unitPrice <= 0 {
		return
	}
	var vendorName string
	if vendorID != "" {
		db.QueryRow("SELECT name FROM vendors WHERE id=?", vendorID).Scan(&vendorName)
	}
	db.Exec(`INSERT INTO price_history (ipn, vendor_id, vendor_name, unit_price, po_id) VALUES (?, ?, ?, ?, ?)`,
		ipn, vendorID, vendorName, unitPrice, poID)
}
