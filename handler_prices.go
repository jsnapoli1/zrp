package main

import (
	"net/http"
	"strconv"
)

type PriceHistory struct {
	ID           int     `json:"id"`
	IPN          string  `json:"ipn"`
	VendorID     *string `json:"vendor_id"`
	VendorName   *string `json:"vendor_name"`
	UnitPrice    float64 `json:"unit_price"`
	Currency     string  `json:"currency"`
	MinQty       int     `json:"min_qty"`
	LeadTimeDays *int    `json:"lead_time_days"`
	POID         *string `json:"po_id"`
	RecordedAt   string  `json:"recorded_at"`
	Notes        *string `json:"notes"`
}

type PriceTrendPoint struct {
	Date   string  `json:"date"`
	Price  float64 `json:"price"`
	Vendor string  `json:"vendor"`
}

func handleListPrices(w http.ResponseWriter, r *http.Request, ipn string) {
	rows, err := db.Query(`
		SELECT ph.id, ph.ipn, ph.vendor_id, COALESCE(ph.vendor_name, v.name, ''), ph.unit_price, ph.currency, ph.min_qty, ph.lead_time_days, ph.po_id, ph.recorded_at, ph.notes
		FROM price_history ph
		LEFT JOIN vendors v ON ph.vendor_id = v.id
		WHERE ph.ipn = ?
		ORDER BY ph.recorded_at DESC`, ipn)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []PriceHistory
	for rows.Next() {
		var p PriceHistory
		rows.Scan(&p.ID, &p.IPN, &p.VendorID, &p.VendorName, &p.UnitPrice, &p.Currency, &p.MinQty, &p.LeadTimeDays, &p.POID, &p.RecordedAt, &p.Notes)
		items = append(items, p)
	}
	if items == nil {
		items = []PriceHistory{}
	}
	jsonResp(w, items)
}

func handleCreatePrice(w http.ResponseWriter, r *http.Request) {
	var p struct {
		IPN          string  `json:"ipn"`
		VendorID     *string `json:"vendor_id"`
		VendorName   *string `json:"vendor_name"`
		UnitPrice    float64 `json:"unit_price"`
		Currency     string  `json:"currency"`
		MinQty       int     `json:"min_qty"`
		LeadTimeDays *int    `json:"lead_time_days"`
		Notes        *string `json:"notes"`
	}
	if err := decodeBody(r, &p); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if p.IPN == "" || p.UnitPrice <= 0 {
		jsonErr(w, "ipn and unit_price > 0 required", 400)
		return
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	if p.MinQty <= 0 {
		p.MinQty = 1
	}

	// Resolve vendor name if vendor_id provided
	var vendorName *string
	if p.VendorName != nil {
		vendorName = p.VendorName
	} else if p.VendorID != nil && *p.VendorID != "" {
		var name string
		if err := db.QueryRow("SELECT name FROM vendors WHERE id=?", *p.VendorID).Scan(&name); err == nil {
			vendorName = &name
		}
	}

	res, err := db.Exec(`INSERT INTO price_history (ipn, vendor_id, vendor_name, unit_price, currency, min_qty, lead_time_days, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.IPN, p.VendorID, vendorName, p.UnitPrice, p.Currency, p.MinQty, p.LeadTimeDays, p.Notes)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	id, _ := res.LastInsertId()
	logAudit(db, getUsername(r), "created", "price", strconv.FormatInt(id, 10), "Added price for "+p.IPN)
	jsonResp(w, map[string]interface{}{"id": id, "ipn": p.IPN, "unit_price": p.UnitPrice})
}

func handleDeletePrice(w http.ResponseWriter, r *http.Request, id string) {
	res, err := db.Exec("DELETE FROM price_history WHERE id=?", id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		jsonErr(w, "not found", 404)
		return
	}
	logAudit(db, getUsername(r), "deleted", "price", id, "Deleted price entry "+id)
	jsonResp(w, map[string]string{"status": "deleted"})
}

func handlePriceTrend(w http.ResponseWriter, r *http.Request, ipn string) {
	rows, err := db.Query(`
		SELECT DATE(ph.recorded_at) as d, ph.unit_price, COALESCE(ph.vendor_name, v.name, '')
		FROM price_history ph
		LEFT JOIN vendors v ON ph.vendor_id = v.id
		WHERE ph.ipn = ?
		ORDER BY ph.recorded_at ASC`, ipn)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var points []PriceTrendPoint
	for rows.Next() {
		var p PriceTrendPoint
		rows.Scan(&p.Date, &p.Price, &p.Vendor)
		points = append(points, p)
	}
	if points == nil {
		points = []PriceTrendPoint{}
	}
	jsonResp(w, points)
}

// recordPriceFromPO records a price history entry when a PO line is received
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
