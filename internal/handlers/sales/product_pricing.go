package sales

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"zrp/internal/models"
	"zrp/internal/response"
)

// ListProductPricing handles GET /api/product-pricing.
func (h *Handler) ListProductPricing(w http.ResponseWriter, r *http.Request) {
	ipnFilter := r.URL.Query().Get("product_ipn")
	tierFilter := r.URL.Query().Get("pricing_tier")

	query := `SELECT id, product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency,
		effective_date, COALESCE(expiry_date,''), COALESCE(notes,''), created_at, updated_at
		FROM product_pricing WHERE 1=1`
	var args []interface{}
	if ipnFilter != "" {
		query += " AND product_ipn = ?"
		args = append(args, ipnFilter)
	}
	if tierFilter != "" {
		query += " AND pricing_tier = ?"
		args = append(args, tierFilter)
	}
	query += " ORDER BY product_ipn, pricing_tier, min_qty"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.ProductPricing
	for rows.Next() {
		var p models.ProductPricing
		rows.Scan(&p.ID, &p.ProductIPN, &p.PricingTier, &p.MinQty, &p.MaxQty,
			&p.UnitPrice, &p.Currency, &p.EffectiveDate, &p.ExpiryDate, &p.Notes,
			&p.CreatedAt, &p.UpdatedAt)
		items = append(items, p)
	}
	if items == nil {
		items = []models.ProductPricing{}
	}
	response.JSON(w, items)
}

// GetProductPricing handles GET /api/product-pricing/:id.
func (h *Handler) GetProductPricing(w http.ResponseWriter, r *http.Request, id string) {
	var p models.ProductPricing
	err := h.DB.QueryRow(`SELECT id, product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency,
		effective_date, COALESCE(expiry_date,''), COALESCE(notes,''), created_at, updated_at
		FROM product_pricing WHERE id = ?`, id).Scan(
		&p.ID, &p.ProductIPN, &p.PricingTier, &p.MinQty, &p.MaxQty,
		&p.UnitPrice, &p.Currency, &p.EffectiveDate, &p.ExpiryDate, &p.Notes,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		response.Err(w, "pricing not found", 404)
		return
	}
	response.JSON(w, p)
}

// CreateProductPricing handles POST /api/product-pricing.
func (h *Handler) CreateProductPricing(w http.ResponseWriter, r *http.Request) {
	var p models.ProductPricing
	if err := response.DecodeBody(r, &p); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	if p.ProductIPN == "" {
		response.Err(w, "product_ipn required", 400)
		return
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	if p.PricingTier == "" {
		p.PricingTier = "standard"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := h.DB.Exec(`INSERT INTO product_pricing (product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency, effective_date, expiry_date, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ProductIPN, p.PricingTier, p.MinQty, p.MaxQty, p.UnitPrice, p.Currency,
		p.EffectiveDate, p.ExpiryDate, p.Notes, now, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	id, _ := res.LastInsertId()
	p.ID = int(id)
	p.CreatedAt = now
	p.UpdatedAt = now
	response.JSON(w, p)
}

// UpdateProductPricing handles PUT /api/product-pricing/:id.
func (h *Handler) UpdateProductPricing(w http.ResponseWriter, r *http.Request, id string) {
	// Check exists
	var existing models.ProductPricing
	err := h.DB.QueryRow(`SELECT id, product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency,
		effective_date, COALESCE(expiry_date,''), COALESCE(notes,''), created_at, updated_at
		FROM product_pricing WHERE id = ?`, id).Scan(
		&existing.ID, &existing.ProductIPN, &existing.PricingTier, &existing.MinQty, &existing.MaxQty,
		&existing.UnitPrice, &existing.Currency, &existing.EffectiveDate, &existing.ExpiryDate, &existing.Notes,
		&existing.CreatedAt, &existing.UpdatedAt)
	if err != nil {
		response.Err(w, "pricing not found", 404)
		return
	}

	var update map[string]interface{}
	if err := response.DecodeBody(r, &update); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	if v, ok := update["product_ipn"].(string); ok {
		existing.ProductIPN = v
	}
	if v, ok := update["pricing_tier"].(string); ok {
		existing.PricingTier = v
	}
	if v, ok := update["min_qty"].(float64); ok {
		existing.MinQty = int(v)
	}
	if v, ok := update["max_qty"].(float64); ok {
		existing.MaxQty = int(v)
	}
	if v, ok := update["unit_price"].(float64); ok {
		existing.UnitPrice = v
	}
	if v, ok := update["currency"].(string); ok {
		existing.Currency = v
	}
	if v, ok := update["effective_date"].(string); ok {
		existing.EffectiveDate = v
	}
	if v, ok := update["expiry_date"].(string); ok {
		existing.ExpiryDate = v
	}
	if v, ok := update["notes"].(string); ok {
		existing.Notes = v
	}

	now := time.Now().UTC().Format(time.RFC3339)
	existing.UpdatedAt = now
	_, err = h.DB.Exec(`UPDATE product_pricing SET product_ipn=?, pricing_tier=?, min_qty=?, max_qty=?, unit_price=?, currency=?, effective_date=?, expiry_date=?, notes=?, updated_at=? WHERE id=?`,
		existing.ProductIPN, existing.PricingTier, existing.MinQty, existing.MaxQty,
		existing.UnitPrice, existing.Currency, existing.EffectiveDate, existing.ExpiryDate,
		existing.Notes, now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	response.JSON(w, existing)
}

// DeleteProductPricing handles DELETE /api/product-pricing/:id.
func (h *Handler) DeleteProductPricing(w http.ResponseWriter, r *http.Request, id string) {
	res, err := h.DB.Exec("DELETE FROM product_pricing WHERE id = ?", id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		response.Err(w, "not found", 404)
		return
	}
	response.JSON(w, map[string]string{"status": "deleted"})
}

// ListCostAnalysis handles GET /api/cost-analysis.
func (h *Handler) ListCostAnalysis(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT ca.id, ca.product_ipn, ca.bom_cost, ca.labor_cost, ca.overhead_cost,
		ca.total_cost, ca.margin_pct, ca.last_calculated, ca.created_at,
		COALESCE((SELECT pp.unit_price FROM product_pricing pp WHERE pp.product_ipn = ca.product_ipn AND pp.pricing_tier = 'standard' ORDER BY pp.effective_date DESC LIMIT 1), 0) as selling_price
		FROM cost_analysis ca ORDER BY ca.product_ipn`)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.CostAnalysisWithPricing
	for rows.Next() {
		var c models.CostAnalysisWithPricing
		rows.Scan(&c.ID, &c.ProductIPN, &c.BOMCost, &c.LaborCost, &c.OverheadCost,
			&c.TotalCost, &c.MarginPct, &c.LastCalculated, &c.CreatedAt, &c.SellingPrice)
		items = append(items, c)
	}
	if items == nil {
		items = []models.CostAnalysisWithPricing{}
	}
	response.JSON(w, items)
}

// CreateCostAnalysis handles POST /api/cost-analysis.
func (h *Handler) CreateCostAnalysis(w http.ResponseWriter, r *http.Request) {
	var c models.CostAnalysis
	if err := response.DecodeBody(r, &c); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	if c.ProductIPN == "" {
		response.Err(w, "product_ipn required", 400)
		return
	}
	c.TotalCost = c.BOMCost + c.LaborCost + c.OverheadCost

	// Calculate margin from standard pricing
	var sellingPrice float64
	h.DB.QueryRow(`SELECT unit_price FROM product_pricing WHERE product_ipn = ? AND pricing_tier = 'standard' ORDER BY effective_date DESC LIMIT 1`, c.ProductIPN).Scan(&sellingPrice)
	if sellingPrice > 0 {
		c.MarginPct = ((sellingPrice - c.TotalCost) / sellingPrice) * 100
	}

	now := time.Now().UTC().Format(time.RFC3339)
	c.LastCalculated = now
	c.CreatedAt = now

	// Upsert
	res, err := h.DB.Exec(`INSERT INTO cost_analysis (product_ipn, bom_cost, labor_cost, overhead_cost, total_cost, margin_pct, last_calculated, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(product_ipn) DO UPDATE SET bom_cost=excluded.bom_cost, labor_cost=excluded.labor_cost,
		overhead_cost=excluded.overhead_cost, total_cost=excluded.total_cost, margin_pct=excluded.margin_pct,
		last_calculated=excluded.last_calculated`,
		c.ProductIPN, c.BOMCost, c.LaborCost, c.OverheadCost, c.TotalCost, c.MarginPct, now, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	id, _ := res.LastInsertId()
	c.ID = int(id)
	response.JSON(w, c)
}

// ProductPricingHistory handles GET /api/product-pricing/history/:ipn.
func (h *Handler) ProductPricingHistory(w http.ResponseWriter, r *http.Request, ipn string) {
	rows, err := h.DB.Query(`SELECT id, product_ipn, pricing_tier, min_qty, max_qty, unit_price, currency,
		effective_date, COALESCE(expiry_date,''), COALESCE(notes,''), created_at, updated_at
		FROM product_pricing WHERE product_ipn = ? ORDER BY created_at DESC`, ipn)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.ProductPricing
	for rows.Next() {
		var p models.ProductPricing
		rows.Scan(&p.ID, &p.ProductIPN, &p.PricingTier, &p.MinQty, &p.MaxQty,
			&p.UnitPrice, &p.Currency, &p.EffectiveDate, &p.ExpiryDate, &p.Notes,
			&p.CreatedAt, &p.UpdatedAt)
		items = append(items, p)
	}
	if items == nil {
		items = []models.ProductPricing{}
	}
	response.JSON(w, items)
}

// BulkUpdateProductPricing handles POST /api/product-pricing/bulk-update.
func (h *Handler) BulkUpdateProductPricing(w http.ResponseWriter, r *http.Request) {
	var req models.BulkPriceUpdate
	if err := response.DecodeBody(r, &req); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	if len(req.IDs) == 0 {
		response.Err(w, "ids required", 400)
		return
	}

	updated := 0
	for _, id := range req.IDs {
		var currentPrice float64
		err := h.DB.QueryRow("SELECT unit_price FROM product_pricing WHERE id = ?", id).Scan(&currentPrice)
		if err != nil {
			continue
		}
		var newPrice float64
		switch req.AdjustmentType {
		case "percentage":
			newPrice = currentPrice * (1 + req.AdjustmentValue/100)
		case "absolute":
			newPrice = currentPrice + req.AdjustmentValue
		default:
			response.Err(w, "adjustment_type must be 'percentage' or 'absolute'", 400)
			return
		}
		// Round to 2 decimal places
		newPrice = float64(int(newPrice*100+0.5)) / 100
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = h.DB.Exec("UPDATE product_pricing SET unit_price = ?, updated_at = ? WHERE id = ?", newPrice, now, id)
		if err == nil {
			updated++
		}
	}
	response.JSON(w, map[string]interface{}{
		"updated": updated,
		"total":   len(req.IDs),
	})
}

// ParsePricingID is a helper used by routes.
func ParsePricingID(parts []string, idx int) string {
	if idx < len(parts) {
		return parts[idx]
	}
	return ""
}

// PricingIDStr converts an int to string.
func PricingIDStr(id int) string {
	return strconv.Itoa(id)
}

var _ = fmt.Sprintf // keep import
