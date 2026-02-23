package procurement

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"zrp/internal/database"
	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
)

// ListPOs returns all purchase orders.
func (h *Handler) ListPOs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id,COALESCE(vendor_id,''),status,COALESCE(notes,''),created_at,COALESCE(expected_date,''),received_at FROM purchase_orders ORDER BY created_at DESC")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.PurchaseOrder
	for rows.Next() {
		var p models.PurchaseOrder
		var ra sql.NullString
		rows.Scan(&p.ID, &p.VendorID, &p.Status, &p.Notes, &p.CreatedAt, &p.ExpectedDate, &ra)
		p.ReceivedAt = database.SP(ra)
		items = append(items, p)
	}
	if items == nil {
		items = []models.PurchaseOrder{}
	}
	response.JSON(w, items)
}

// GetPO returns a single purchase order with lines.
func (h *Handler) GetPO(w http.ResponseWriter, r *http.Request, id string) {
	var p models.PurchaseOrder
	var ra sql.NullString
	err := h.DB.QueryRow("SELECT id,COALESCE(vendor_id,''),status,COALESCE(notes,''),created_at,COALESCE(expected_date,''),received_at FROM purchase_orders WHERE id=?", id).
		Scan(&p.ID, &p.VendorID, &p.Status, &p.Notes, &p.CreatedAt, &p.ExpectedDate, &ra)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	p.ReceivedAt = database.SP(ra)

	// Load lines
	rows, _ := h.DB.Query("SELECT id,po_id,ipn,COALESCE(mpn,''),COALESCE(manufacturer,''),qty_ordered,qty_received,COALESCE(unit_price,0),COALESCE(notes,'') FROM po_lines WHERE po_id=?", id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l models.POLine
			rows.Scan(&l.ID, &l.POID, &l.IPN, &l.MPN, &l.Manufacturer, &l.QtyOrdered, &l.QtyReceived, &l.UnitPrice, &l.Notes)
			p.Lines = append(p.Lines, l)
		}
	}
	if p.Lines == nil {
		p.Lines = []models.POLine{}
	}
	response.JSON(w, p)
}

// CreatePO creates a new purchase order.
func (h *Handler) CreatePO(w http.ResponseWriter, r *http.Request) {
	var p models.PurchaseOrder
	if err := response.DecodeBody(r, &p); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	if p.VendorID != "" {
		h.ValidateForeignKey(ve, "vendor_id", "vendors", p.VendorID)
	}
	if p.Status != "" {
		validation.ValidateEnum(ve, "status", p.Status, validation.ValidPOStatuses)
	}
	validation.ValidateDate(ve, "expected_date", p.ExpectedDate)
	for i, l := range p.Lines {
		if l.QtyOrdered <= 0 {
			ve.Add(fmt.Sprintf("lines[%d].qty_ordered", i), "must be positive")
		}
		validation.ValidateMaxQuantity(ve, fmt.Sprintf("lines[%d].qty_ordered", i), l.QtyOrdered)
		if l.UnitPrice < 0 {
			ve.Add(fmt.Sprintf("lines[%d].unit_price", i), "must be non-negative")
		}
		validation.ValidateMaxPrice(ve, fmt.Sprintf("lines[%d].unit_price", i), l.UnitPrice)
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	p.ID = h.NextIDFunc("PO", "purchase_orders", 4)
	if p.Status == "" {
		p.Status = "draft"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	createdBy := h.GetUsername(r)
	_, err := h.DB.Exec("INSERT INTO purchase_orders (id,vendor_id,status,notes,created_at,expected_date,created_by) VALUES (?,?,?,?,?,?,?)",
		p.ID, p.VendorID, p.Status, p.Notes, now, p.ExpectedDate, createdBy)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	for _, l := range p.Lines {
		h.DB.Exec("INSERT INTO po_lines (po_id,ipn,mpn,manufacturer,qty_ordered,unit_price,notes) VALUES (?,?,?,?,?,?,?)",
			p.ID, l.IPN, l.MPN, l.Manufacturer, l.QtyOrdered, l.UnitPrice, l.Notes)
	}
	p.CreatedAt = now
	h.LogAudit(h.GetUsername(r), "created", "po", p.ID, "Created PO "+p.ID)
	h.RecordChangeJSON(h.GetUsername(r), "purchase_orders", p.ID, "create", nil, p)
	response.JSON(w, p)
}

// UpdatePO updates an existing purchase order.
func (h *Handler) UpdatePO(w http.ResponseWriter, r *http.Request, id string) {
	oldSnap, _ := h.GetPOSnapshot(id)
	var p models.PurchaseOrder
	if err := response.DecodeBody(r, &p); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	_, err := h.DB.Exec("UPDATE purchase_orders SET vendor_id=?,status=?,notes=?,expected_date=? WHERE id=?",
		p.VendorID, p.Status, p.Notes, p.ExpectedDate, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	username := h.GetUsername(r)
	h.LogAudit(username, "updated", "po", id, "Updated PO "+id)
	newSnap, _ := h.GetPOSnapshot(id)
	h.RecordChangeJSON(username, "purchase_orders", id, "update", oldSnap, newSnap)
	h.GetPO(w, r, id)
}

// GeneratePOFromWO generates a PO from a work order's BOM shortages.
func (h *Handler) GeneratePOFromWO(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WOID     string `json:"wo_id"`
		VendorID string `json:"vendor_id"`
	}
	if err := response.DecodeBody(r, &body); err != nil || body.WOID == "" {
		response.Err(w, "wo_id required", 400)
		return
	}

	// Get WO details
	var assemblyIPN string
	var qty int
	err := h.DB.QueryRow("SELECT assembly_ipn, qty FROM work_orders WHERE id=?", body.WOID).Scan(&assemblyIPN, &qty)
	if err != nil {
		response.Err(w, "work order not found", 404)
		return
	}

	// Get BOM shortages
	rows, err := h.DB.Query("SELECT ipn, qty_on_hand FROM inventory")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var lines []models.POLine
	for rows.Next() {
		var ipn string
		var onHand float64
		rows.Scan(&ipn, &onHand)
		qtyRequired := float64(qty)
		shortage := qtyRequired - onHand
		if shortage > 0 {
			var mpn, manufacturer string
			if h.GetPartByIPN != nil {
				fields, ferr := h.GetPartByIPN(h.PartsDir, ipn)
				if ferr == nil {
					for k, v := range fields {
						kl := strings.ToLower(k)
						if kl == "mpn" {
							mpn = v
						}
						if kl == "manufacturer" {
							manufacturer = v
						}
					}
				}
			}
			lines = append(lines, models.POLine{IPN: ipn, MPN: mpn, Manufacturer: manufacturer, QtyOrdered: shortage})
		}
	}

	if len(lines) == 0 {
		response.Err(w, "no shortages found for this work order", 400)
		return
	}

	// Create PO
	poID := h.NextIDFunc("PO", "purchase_orders", 4)
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = h.DB.Exec("INSERT INTO purchase_orders (id, vendor_id, status, notes, created_at) VALUES (?, ?, 'draft', ?, ?)",
		poID, body.VendorID, "Auto-generated from "+body.WOID, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	for _, l := range lines {
		h.DB.Exec("INSERT INTO po_lines (po_id, ipn, mpn, manufacturer, qty_ordered) VALUES (?, ?, ?, ?, ?)",
			poID, l.IPN, l.MPN, l.Manufacturer, l.QtyOrdered)
	}

	h.LogAudit(h.GetUsername(r), "created", "po", poID, "Auto-generated PO from WO "+body.WOID)
	response.JSON(w, map[string]interface{}{"po_id": poID, "lines": len(lines)})
}

// ReceivePO handles receiving items on a purchase order.
func (h *Handler) ReceivePO(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Lines []struct {
			ID  int     `json:"id"`
			Qty float64 `json:"qty"`
		} `json:"lines"`
		SkipInspection bool `json:"skip_inspection"`
	}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	// Get vendor_id for price recording
	var poVendorID string
	h.DB.QueryRow("SELECT COALESCE(vendor_id,'') FROM purchase_orders WHERE id=?", id).Scan(&poVendorID)

	now := time.Now().Format("2006-01-02 15:04:05")
	for _, l := range body.Lines {
		h.DB.Exec("UPDATE po_lines SET qty_received=qty_received+? WHERE id=?", l.Qty, l.ID)
		var ipn string
		var unitPrice float64
		h.DB.QueryRow("SELECT ipn, COALESCE(unit_price,0) FROM po_lines WHERE id=?", l.ID).Scan(&ipn, &unitPrice)
		// Record price history
		if ipn != "" && unitPrice > 0 {
			h.RecordPriceFromPO(id, ipn, unitPrice, poVendorID)
		}

		if ipn != "" {
			if body.SkipInspection {
				// Legacy behavior: directly update inventory
				h.DB.Exec("INSERT OR IGNORE INTO inventory (ipn) VALUES (?)", ipn)
				h.DB.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+?,updated_at=? WHERE ipn=?", l.Qty, now, ipn)
				h.DB.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,created_at) VALUES (?,?,?,?,?)", ipn, "receive", l.Qty, id, now)
			} else {
				// Create receiving inspection record (inventory updated after inspection)
				h.DB.Exec(`INSERT INTO receiving_inspections (po_id,po_line_id,ipn,qty_received,created_at) VALUES (?,?,?,?,?)`,
					id, l.ID, ipn, l.Qty, now)
			}
		}
	}
	// Check if all received
	var totalOrdered, totalReceived float64
	h.DB.QueryRow("SELECT COALESCE(SUM(qty_ordered),0),COALESCE(SUM(qty_received),0) FROM po_lines WHERE po_id=?", id).Scan(&totalOrdered, &totalReceived)
	if totalReceived >= totalOrdered {
		h.DB.Exec("UPDATE purchase_orders SET status='received',received_at=? WHERE id=?", now, id)
	} else {
		h.DB.Exec("UPDATE purchase_orders SET status='partial' WHERE id=?", id)
	}
	h.LogAudit(h.GetUsername(r), "received", "po", id, "Received items on PO "+id)
	if h.EmailOnPOReceived != nil {
		go h.EmailOnPOReceived(id)
	}
	h.GetPO(w, r, id)
}

// GeneratePOSuggestions analyzes BOM shortages and creates PO suggestions.
func (h *Handler) GeneratePOSuggestions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WOID        string `json:"wo_id"`
		SuggestOnly bool   `json:"suggest_only"`
	}
	if err := response.DecodeBody(r, &body); err != nil || body.WOID == "" {
		response.Err(w, "wo_id required", 400)
		return
	}

	// Get work order details
	var assemblyIPN string
	var woQty int
	err := h.DB.QueryRow("SELECT assembly_ipn, qty FROM work_orders WHERE id=?", body.WOID).Scan(&assemblyIPN, &woQty)
	if err != nil {
		response.Err(w, "work order not found", 404)
		return
	}

	// Get BOM for the assembly
	type BOMRequirement struct {
		IPN      string
		QtyPer   float64
		Required float64
		OnHand   float64
		Shortage float64
	}

	bomRows, err := h.DB.Query("SELECT child_ipn, qty_per FROM bom_items WHERE parent_ipn = ?", assemblyIPN)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer bomRows.Close()

	var requirements []BOMRequirement
	for bomRows.Next() {
		var req BOMRequirement
		bomRows.Scan(&req.IPN, &req.QtyPer)
		req.Required = req.QtyPer * float64(woQty)

		// Get current inventory
		req.OnHand = 0.0
		var onHand sql.NullFloat64
		err := h.DB.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", req.IPN).Scan(&onHand)
		if err == nil && onHand.Valid {
			req.OnHand = onHand.Float64
		}

		req.Shortage = req.Required - req.OnHand
		if req.Shortage > 0 {
			requirements = append(requirements, req)
		}
	}

	if len(requirements) == 0 {
		response.JSON(w, map[string]interface{}{"message": "No shortages found", "suggestions": []interface{}{}})
		return
	}

	// Group shortages by preferred vendor
	type VendorGroup struct {
		VendorID string
		Items    []struct {
			IPN          string
			MPN          string
			Manufacturer string
			Shortage     float64
			UnitPrice    float64
		}
	}

	vendorGroups := make(map[string]*VendorGroup)

	for _, req := range requirements {
		// Get preferred vendor for this part
		var vendorID, mpn, manufacturer string
		var unitPrice float64
		err := h.DB.QueryRow(`
			SELECT vendor_id, COALESCE(mpn, ''), COALESCE(manufacturer, ''), COALESCE(unit_price, 0)
			FROM part_vendors
			WHERE ipn = ? AND is_preferred = 1
			ORDER BY unit_price ASC
			LIMIT 1
		`, req.IPN).Scan(&vendorID, &mpn, &manufacturer, &unitPrice)

		if err != nil {
			// No preferred vendor found, skip or use default
			continue
		}

		if vendorGroups[vendorID] == nil {
			vendorGroups[vendorID] = &VendorGroup{VendorID: vendorID}
		}

		vendorGroups[vendorID].Items = append(vendorGroups[vendorID].Items, struct {
			IPN          string
			MPN          string
			Manufacturer string
			Shortage     float64
			UnitPrice    float64
		}{
			IPN:          req.IPN,
			MPN:          mpn,
			Manufacturer: manufacturer,
			Shortage:     req.Shortage,
			UnitPrice:    unitPrice,
		})
	}

	// Create PO suggestions for each vendor
	now := time.Now().Format("2006-01-02 15:04:05")
	createdBy := h.GetUsername(r)
	var suggestionIDs []int

	for vendorID, group := range vendorGroups {
		notes := fmt.Sprintf("Auto-generated from %s (BOM shortage analysis)", body.WOID)
		result, err := h.DB.Exec(`
			INSERT INTO po_suggestions (wo_id, vendor_id, status, notes, created_at)
			VALUES (?, ?, 'pending', ?, ?)
		`, body.WOID, vendorID, notes, now)

		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}

		suggestionID, _ := result.LastInsertId()
		suggestionIDs = append(suggestionIDs, int(suggestionID))

		// Add lines to suggestion
		for _, item := range group.Items {
			h.DB.Exec(`
				INSERT INTO po_suggestion_lines
				(suggestion_id, ipn, mpn, manufacturer, qty_needed, estimated_unit_price)
				VALUES (?, ?, ?, ?, ?, ?)
			`, suggestionID, item.IPN, item.MPN, item.Manufacturer, item.Shortage, item.UnitPrice)
		}
	}

	h.LogAudit(createdBy, "created", "po_suggestion", body.WOID, fmt.Sprintf("Generated %d PO suggestions for %s", len(suggestionIDs), body.WOID))

	response.JSON(w, map[string]interface{}{
		"message":        fmt.Sprintf("Created %d PO suggestion(s)", len(suggestionIDs)),
		"suggestion_ids": suggestionIDs,
		"wo_id":          body.WOID,
	})
}

// ReviewPOSuggestion approves or rejects a PO suggestion, optionally creating the PO.
func (h *Handler) ReviewPOSuggestion(w http.ResponseWriter, r *http.Request, suggestionID int) {
	var body struct {
		Status   string `json:"status"`
		Reason   string `json:"reason"`
		CreatePO bool   `json:"create_po"`
	}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	if body.Status != "approved" && body.Status != "rejected" {
		response.Err(w, "status must be 'approved' or 'rejected'", 400)
		return
	}

	// Verify suggestion exists
	var woID, vendorID, currentStatus string
	err := h.DB.QueryRow("SELECT wo_id, vendor_id, status FROM po_suggestions WHERE id = ?", suggestionID).
		Scan(&woID, &vendorID, &currentStatus)
	if err != nil {
		response.Err(w, "suggestion not found", 404)
		return
	}

	if currentStatus != "pending" {
		response.Err(w, fmt.Sprintf("suggestion already %s", currentStatus), 400)
		return
	}

	// Update suggestion status
	now := time.Now().Format("2006-01-02 15:04:05")
	reviewedBy := h.GetUsername(r)
	notes := body.Reason

	_, err = h.DB.Exec(`
		UPDATE po_suggestions
		SET status = ?, reviewed_by = ?, reviewed_at = ?, notes = COALESCE(notes || '\nReview: ' || ?, notes)
		WHERE id = ?
	`, body.Status, reviewedBy, now, notes, suggestionID)

	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	h.LogAudit(reviewedBy, body.Status, "po_suggestion", fmt.Sprintf("%d", suggestionID),
		fmt.Sprintf("%s PO suggestion #%d for WO %s", strings.Title(body.Status), suggestionID, woID))

	var poID string

	// If approved and create_po is true, create the actual PO
	if body.Status == "approved" && body.CreatePO {
		poID = h.NextIDFunc("PO", "purchase_orders", 4)

		// Create PO header
		_, err = h.DB.Exec(`
			INSERT INTO purchase_orders (id, vendor_id, status, notes, created_at, created_by)
			VALUES (?, ?, 'draft', ?, ?, ?)
		`, poID, vendorID, fmt.Sprintf("Created from suggestion #%d for WO %s", suggestionID, woID), now, reviewedBy)

		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}

		// Copy suggestion lines to PO lines
		rows, err := h.DB.Query(`
			SELECT ipn, COALESCE(mpn, ''), COALESCE(manufacturer, ''), qty_needed, estimated_unit_price, COALESCE(notes, '')
			FROM po_suggestion_lines
			WHERE suggestion_id = ?
		`, suggestionID)

		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var ipn, mpn, manufacturer, notes string
			var qtyNeeded, unitPrice float64
			rows.Scan(&ipn, &mpn, &manufacturer, &qtyNeeded, &unitPrice, &notes)

			// Skip lines with zero or negative quantity
			if qtyNeeded <= 0 {
				continue
			}

			_, err = h.DB.Exec(`
				INSERT INTO po_lines (po_id, ipn, mpn, manufacturer, qty_ordered, unit_price, notes)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, poID, ipn, mpn, manufacturer, qtyNeeded, unitPrice, notes)

			if err != nil {
				response.Err(w, err.Error(), 500)
				return
			}
		}

		// Link PO back to suggestion
		h.DB.Exec("UPDATE po_suggestions SET po_id = ? WHERE id = ?", poID, suggestionID)

		h.LogAudit(reviewedBy, "created", "po", poID, fmt.Sprintf("Created PO %s from approved suggestion #%d", poID, suggestionID))
		h.RecordChangeJSON(reviewedBy, "purchase_orders", poID, "create", nil, map[string]interface{}{
			"id":        poID,
			"vendor_id": vendorID,
			"status":    "draft",
			"source":    fmt.Sprintf("suggestion_%d", suggestionID),
		})
	}

	response.JSON(w, map[string]interface{}{
		"suggestion_id": suggestionID,
		"status":        body.Status,
		"po_id":         poID,
		"message":       fmt.Sprintf("Suggestion %s", body.Status),
	})
}
