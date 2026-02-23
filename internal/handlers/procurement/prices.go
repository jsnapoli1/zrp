package procurement

import (
	"net/http"
	"strconv"

	"zrp/internal/models"
	"zrp/internal/response"
)

// ListPrices returns price history for a given IPN.
func (h *Handler) ListPrices(w http.ResponseWriter, r *http.Request, ipn string) {
	rows, err := h.DB.Query(`
		SELECT ph.id, ph.ipn, ph.vendor_id, COALESCE(ph.vendor_name, v.name, ''), ph.unit_price, ph.currency, ph.min_qty, ph.lead_time_days, ph.po_id, ph.recorded_at, ph.notes
		FROM price_history ph
		LEFT JOIN vendors v ON ph.vendor_id = v.id
		WHERE ph.ipn = ?
		ORDER BY ph.recorded_at DESC`, ipn)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.PriceHistory
	for rows.Next() {
		var p models.PriceHistory
		rows.Scan(&p.ID, &p.IPN, &p.VendorID, &p.VendorName, &p.UnitPrice, &p.Currency, &p.MinQty, &p.LeadTimeDays, &p.POID, &p.RecordedAt, &p.Notes)
		items = append(items, p)
	}
	if items == nil {
		items = []models.PriceHistory{}
	}
	response.JSON(w, items)
}

// CreatePrice adds a new price history entry.
func (h *Handler) CreatePrice(w http.ResponseWriter, r *http.Request) {
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
	if err := response.DecodeBody(r, &p); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	if p.IPN == "" || p.UnitPrice <= 0 {
		response.Err(w, "ipn and unit_price > 0 required", 400)
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
		if err := h.DB.QueryRow("SELECT name FROM vendors WHERE id=?", *p.VendorID).Scan(&name); err == nil {
			vendorName = &name
		}
	}

	res, err := h.DB.Exec(`INSERT INTO price_history (ipn, vendor_id, vendor_name, unit_price, currency, min_qty, lead_time_days, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.IPN, p.VendorID, vendorName, p.UnitPrice, p.Currency, p.MinQty, p.LeadTimeDays, p.Notes)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	id, _ := res.LastInsertId()
	h.LogAudit(h.GetUsername(r), "created", "price", strconv.FormatInt(id, 10), "Added price for "+p.IPN)
	response.JSON(w, map[string]interface{}{"id": id, "ipn": p.IPN, "unit_price": p.UnitPrice})
}

// DeletePrice deletes a price history entry.
func (h *Handler) DeletePrice(w http.ResponseWriter, r *http.Request, id string) {
	res, err := h.DB.Exec("DELETE FROM price_history WHERE id=?", id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		response.Err(w, "not found", 404)
		return
	}
	h.LogAudit(h.GetUsername(r), "deleted", "price", id, "Deleted price entry "+id)
	response.JSON(w, map[string]string{"status": "deleted"})
}

// PriceTrend returns price trend data for a given IPN.
func (h *Handler) PriceTrend(w http.ResponseWriter, r *http.Request, ipn string) {
	rows, err := h.DB.Query(`
		SELECT DATE(ph.recorded_at) as d, ph.unit_price, COALESCE(ph.vendor_name, v.name, '')
		FROM price_history ph
		LEFT JOIN vendors v ON ph.vendor_id = v.id
		WHERE ph.ipn = ?
		ORDER BY ph.recorded_at ASC`, ipn)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var points []models.PriceTrendPoint
	for rows.Next() {
		var p models.PriceTrendPoint
		rows.Scan(&p.Date, &p.Price, &p.Vendor)
		points = append(points, p)
	}
	if points == nil {
		points = []models.PriceTrendPoint{}
	}
	response.JSON(w, points)
}
