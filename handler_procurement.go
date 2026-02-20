package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func handleListPOs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,COALESCE(vendor_id,''),status,COALESCE(notes,''),created_at,COALESCE(expected_date,''),received_at FROM purchase_orders ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []PurchaseOrder
	for rows.Next() {
		var p PurchaseOrder
		var ra sql.NullString
		rows.Scan(&p.ID, &p.VendorID, &p.Status, &p.Notes, &p.CreatedAt, &p.ExpectedDate, &ra)
		p.ReceivedAt = sp(ra)
		items = append(items, p)
	}
	if items == nil { items = []PurchaseOrder{} }
	jsonResp(w, items)
}

func handleGetPO(w http.ResponseWriter, r *http.Request, id string) {
	var p PurchaseOrder
	var ra sql.NullString
	err := db.QueryRow("SELECT id,COALESCE(vendor_id,''),status,COALESCE(notes,''),created_at,COALESCE(expected_date,''),received_at FROM purchase_orders WHERE id=?", id).
		Scan(&p.ID, &p.VendorID, &p.Status, &p.Notes, &p.CreatedAt, &p.ExpectedDate, &ra)
	if err != nil { jsonErr(w, "not found", 404); return }
	p.ReceivedAt = sp(ra)

	// Load lines
	rows, _ := db.Query("SELECT id,po_id,ipn,COALESCE(mpn,''),COALESCE(manufacturer,''),qty_ordered,qty_received,COALESCE(unit_price,0),COALESCE(notes,'') FROM po_lines WHERE po_id=?", id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l POLine
			rows.Scan(&l.ID, &l.POID, &l.IPN, &l.MPN, &l.Manufacturer, &l.QtyOrdered, &l.QtyReceived, &l.UnitPrice, &l.Notes)
			p.Lines = append(p.Lines, l)
		}
	}
	if p.Lines == nil { p.Lines = []POLine{} }
	jsonResp(w, p)
}

func handleCreatePO(w http.ResponseWriter, r *http.Request) {
	var p PurchaseOrder
	if err := decodeBody(r, &p); err != nil { jsonErr(w, "invalid body", 400); return }

	ve := &ValidationErrors{}
	if p.VendorID != "" { validateForeignKey(ve, "vendor_id", "vendors", p.VendorID) }
	if p.Status != "" { validateEnum(ve, "status", p.Status, validPOStatuses) }
	validateDate(ve, "expected_date", p.ExpectedDate)
	for i, l := range p.Lines {
		if l.QtyOrdered <= 0 { ve.Add(fmt.Sprintf("lines[%d].qty_ordered", i), "must be positive") }
		validateMaxQuantity(ve, fmt.Sprintf("lines[%d].qty_ordered", i), l.QtyOrdered)
		if l.UnitPrice < 0 { ve.Add(fmt.Sprintf("lines[%d].unit_price", i), "must be non-negative") }
		validateMaxPrice(ve, fmt.Sprintf("lines[%d].unit_price", i), l.UnitPrice)
	}
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	p.ID = nextID("PO", "purchase_orders", 4)
	if p.Status == "" { p.Status = "draft" }
	now := time.Now().Format("2006-01-02 15:04:05")
	createdBy := getUsername(r)
	_, err := db.Exec("INSERT INTO purchase_orders (id,vendor_id,status,notes,created_at,expected_date,created_by) VALUES (?,?,?,?,?,?,?)",
		p.ID, p.VendorID, p.Status, p.Notes, now, p.ExpectedDate, createdBy)
	if err != nil { jsonErr(w, err.Error(), 500); return }

	for _, l := range p.Lines {
		db.Exec("INSERT INTO po_lines (po_id,ipn,mpn,manufacturer,qty_ordered,unit_price,notes) VALUES (?,?,?,?,?,?,?)",
			p.ID, l.IPN, l.MPN, l.Manufacturer, l.QtyOrdered, l.UnitPrice, l.Notes)
	}
	p.CreatedAt = now
	logAudit(db, getUsername(r), "created", "po", p.ID, "Created PO "+p.ID)
	recordChangeJSON(getUsername(r), "purchase_orders", p.ID, "create", nil, p)
	jsonResp(w, p)
}

func handleUpdatePO(w http.ResponseWriter, r *http.Request, id string) {
	oldSnap, _ := getPOSnapshot(id)
	var p PurchaseOrder
	if err := decodeBody(r, &p); err != nil { jsonErr(w, "invalid body", 400); return }
	_, err := db.Exec("UPDATE purchase_orders SET vendor_id=?,status=?,notes=?,expected_date=? WHERE id=?",
		p.VendorID, p.Status, p.Notes, p.ExpectedDate, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "po", id, "Updated PO "+id)
	newSnap, _ := getPOSnapshot(id)
	recordChangeJSON(getUsername(r), "purchase_orders", id, "update", oldSnap, newSnap)
	handleGetPO(w, r, id)
}

func handleGeneratePOFromWO(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WOID     string `json:"wo_id"`
		VendorID string `json:"vendor_id"`
	}
	if err := decodeBody(r, &body); err != nil || body.WOID == "" {
		jsonErr(w, "wo_id required", 400)
		return
	}

	// Get WO details
	var assemblyIPN string
	var qty int
	err := db.QueryRow("SELECT assembly_ipn, qty FROM work_orders WHERE id=?", body.WOID).Scan(&assemblyIPN, &qty)
	if err != nil {
		jsonErr(w, "work order not found", 404)
		return
	}

	// Get BOM shortages (same logic as handleWorkOrderBOM)
	rows, err := db.Query("SELECT ipn, qty_on_hand FROM inventory")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var lines []POLine
	for rows.Next() {
		var ipn string
		var onHand float64
		rows.Scan(&ipn, &onHand)
		qtyRequired := float64(qty)
		shortage := qtyRequired - onHand
		if shortage > 0 {
			var mpn, manufacturer string
			fields, ferr := getPartByIPN(partsDir, ipn)
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
			lines = append(lines, POLine{IPN: ipn, MPN: mpn, Manufacturer: manufacturer, QtyOrdered: shortage})
		}
	}

	if len(lines) == 0 {
		jsonErr(w, "no shortages found for this work order", 400)
		return
	}

	// Create PO
	poID := nextID("PO", "purchase_orders", 4)
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.Exec("INSERT INTO purchase_orders (id, vendor_id, status, notes, created_at) VALUES (?, ?, 'draft', ?, ?)",
		poID, body.VendorID, "Auto-generated from "+body.WOID, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	for _, l := range lines {
		db.Exec("INSERT INTO po_lines (po_id, ipn, mpn, manufacturer, qty_ordered) VALUES (?, ?, ?, ?, ?)",
			poID, l.IPN, l.MPN, l.Manufacturer, l.QtyOrdered)
	}

	logAudit(db, getUsername(r), "created", "po", poID, "Auto-generated PO from WO "+body.WOID)
	jsonResp(w, map[string]interface{}{"po_id": poID, "lines": len(lines)})
}

func handleReceivePO(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Lines []struct {
			ID  int     `json:"id"`
			Qty float64 `json:"qty"`
		} `json:"lines"`
		SkipInspection bool `json:"skip_inspection"`
	}
	if err := decodeBody(r, &body); err != nil { jsonErr(w, "invalid body", 400); return }
	// Get vendor_id for price recording
	var poVendorID string
	db.QueryRow("SELECT COALESCE(vendor_id,'') FROM purchase_orders WHERE id=?", id).Scan(&poVendorID)

	now := time.Now().Format("2006-01-02 15:04:05")
	for _, l := range body.Lines {
		db.Exec("UPDATE po_lines SET qty_received=qty_received+? WHERE id=?", l.Qty, l.ID)
		var ipn string
		var unitPrice float64
		db.QueryRow("SELECT ipn, COALESCE(unit_price,0) FROM po_lines WHERE id=?", l.ID).Scan(&ipn, &unitPrice)
		// Record price history
		if ipn != "" && unitPrice > 0 {
			recordPriceFromPO(id, ipn, unitPrice, poVendorID)
		}

		if ipn != "" {
			if body.SkipInspection {
				// Legacy behavior: directly update inventory
				db.Exec("INSERT OR IGNORE INTO inventory (ipn) VALUES (?)", ipn)
				db.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+?,updated_at=? WHERE ipn=?", l.Qty, now, ipn)
				db.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,created_at) VALUES (?,?,?,?,?)", ipn, "receive", l.Qty, id, now)
			} else {
				// Create receiving inspection record (inventory updated after inspection)
				db.Exec(`INSERT INTO receiving_inspections (po_id,po_line_id,ipn,qty_received,created_at) VALUES (?,?,?,?,?)`,
					id, l.ID, ipn, l.Qty, now)
			}
		}
	}
	// Check if all received
	var totalOrdered, totalReceived float64
	db.QueryRow("SELECT COALESCE(SUM(qty_ordered),0),COALESCE(SUM(qty_received),0) FROM po_lines WHERE po_id=?", id).Scan(&totalOrdered, &totalReceived)
	if totalReceived >= totalOrdered {
		db.Exec("UPDATE purchase_orders SET status='received',received_at=? WHERE id=?", now, id)
	} else {
		db.Exec("UPDATE purchase_orders SET status='partial' WHERE id=?", id)
	}
	logAudit(db, getUsername(r), "received", "po", id, "Received items on PO "+id)
	go emailOnPOReceived(id)
	handleGetPO(w, r, id)
}

// handleGeneratePOSuggestions analyzes BOM shortages and creates PO suggestions (not actual POs)
func handleGeneratePOSuggestions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WOID        string `json:"wo_id"`
		SuggestOnly bool   `json:"suggest_only"`
	}
	if err := decodeBody(r, &body); err != nil || body.WOID == "" {
		jsonErr(w, "wo_id required", 400)
		return
	}

	// Get work order details
	var assemblyIPN string
	var woQty int
	err := db.QueryRow("SELECT assembly_ipn, qty FROM work_orders WHERE id=?", body.WOID).Scan(&assemblyIPN, &woQty)
	if err != nil {
		jsonErr(w, "work order not found", 404)
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

	bomRows, err := db.Query("SELECT child_ipn, qty_per FROM bom_items WHERE parent_ipn = ?", assemblyIPN)
	if err != nil {
		jsonErr(w, err.Error(), 500)
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
		err := db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn = ?", req.IPN).Scan(&onHand)
		if err == nil && onHand.Valid {
			req.OnHand = onHand.Float64
		}

		req.Shortage = req.Required - req.OnHand
		if req.Shortage > 0 {
			requirements = append(requirements, req)
		}
	}

	if len(requirements) == 0 {
		jsonResp(w, map[string]interface{}{"message": "No shortages found", "suggestions": []interface{}{}})
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
		err := db.QueryRow(`
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
	createdBy := getUsername(r)
	var suggestionIDs []int

	for vendorID, group := range vendorGroups {
		notes := fmt.Sprintf("Auto-generated from %s (BOM shortage analysis)", body.WOID)
		result, err := db.Exec(`
			INSERT INTO po_suggestions (wo_id, vendor_id, status, notes, created_at)
			VALUES (?, ?, 'pending', ?, ?)
		`, body.WOID, vendorID, notes, now)

		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}

		suggestionID, _ := result.LastInsertId()
		suggestionIDs = append(suggestionIDs, int(suggestionID))

		// Add lines to suggestion
		for _, item := range group.Items {
			db.Exec(`
				INSERT INTO po_suggestion_lines 
				(suggestion_id, ipn, mpn, manufacturer, qty_needed, estimated_unit_price)
				VALUES (?, ?, ?, ?, ?, ?)
			`, suggestionID, item.IPN, item.MPN, item.Manufacturer, item.Shortage, item.UnitPrice)
		}
	}

	logAudit(db, createdBy, "created", "po_suggestion", body.WOID, fmt.Sprintf("Generated %d PO suggestions for %s", len(suggestionIDs), body.WOID))

	jsonResp(w, map[string]interface{}{
		"message":        fmt.Sprintf("Created %d PO suggestion(s)", len(suggestionIDs)),
		"suggestion_ids": suggestionIDs,
		"wo_id":          body.WOID,
	})
}

// handleReviewPOSuggestion approves or rejects a PO suggestion, optionally creating the PO
func handleReviewPOSuggestion(w http.ResponseWriter, r *http.Request, suggestionID int) {
	var body struct {
		Status   string `json:"status"`   // "approved" or "rejected"
		Reason   string `json:"reason"`   // Optional reason for rejection
		CreatePO bool   `json:"create_po"` // If true, create PO immediately upon approval
	}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	if body.Status != "approved" && body.Status != "rejected" {
		jsonErr(w, "status must be 'approved' or 'rejected'", 400)
		return
	}

	// Verify suggestion exists
	var woID, vendorID, currentStatus string
	err := db.QueryRow("SELECT wo_id, vendor_id, status FROM po_suggestions WHERE id = ?", suggestionID).
		Scan(&woID, &vendorID, &currentStatus)
	if err != nil {
		jsonErr(w, "suggestion not found", 404)
		return
	}

	if currentStatus != "pending" {
		jsonErr(w, fmt.Sprintf("suggestion already %s", currentStatus), 400)
		return
	}

	// Update suggestion status
	now := time.Now().Format("2006-01-02 15:04:05")
	reviewedBy := getUsername(r)
	notes := body.Reason

	_, err = db.Exec(`
		UPDATE po_suggestions 
		SET status = ?, reviewed_by = ?, reviewed_at = ?, notes = COALESCE(notes || '\nReview: ' || ?, notes)
		WHERE id = ?
	`, body.Status, reviewedBy, now, notes, suggestionID)

	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, reviewedBy, body.Status, "po_suggestion", fmt.Sprintf("%d", suggestionID), 
		fmt.Sprintf("%s PO suggestion #%d for WO %s", strings.Title(body.Status), suggestionID, woID))

	var poID string

	// If approved and create_po is true, create the actual PO
	if body.Status == "approved" && body.CreatePO {
		poID = nextID("PO", "purchase_orders", 4)
		
		// Create PO header
		_, err = db.Exec(`
			INSERT INTO purchase_orders (id, vendor_id, status, notes, created_at, created_by)
			VALUES (?, ?, 'draft', ?, ?, ?)
		`, poID, vendorID, fmt.Sprintf("Created from suggestion #%d for WO %s", suggestionID, woID), now, reviewedBy)

		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}

		// Copy suggestion lines to PO lines
		rows, err := db.Query(`
			SELECT ipn, COALESCE(mpn, ''), COALESCE(manufacturer, ''), qty_needed, estimated_unit_price, COALESCE(notes, '')
			FROM po_suggestion_lines
			WHERE suggestion_id = ?
		`, suggestionID)

		if err != nil {
			jsonErr(w, err.Error(), 500)
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

			_, err = db.Exec(`
				INSERT INTO po_lines (po_id, ipn, mpn, manufacturer, qty_ordered, unit_price, notes)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, poID, ipn, mpn, manufacturer, qtyNeeded, unitPrice, notes)

			if err != nil {
				jsonErr(w, err.Error(), 500)
				return
			}
		}

		// Link PO back to suggestion
		db.Exec("UPDATE po_suggestions SET po_id = ? WHERE id = ?", poID, suggestionID)

		logAudit(db, reviewedBy, "created", "po", poID, fmt.Sprintf("Created PO %s from approved suggestion #%d", poID, suggestionID))
		recordChangeJSON(reviewedBy, "purchase_orders", poID, "create", nil, map[string]interface{}{
			"id":        poID,
			"vendor_id": vendorID,
			"status":    "draft",
			"source":    fmt.Sprintf("suggestion_%d", suggestionID),
		})
	}

	jsonResp(w, map[string]interface{}{
		"suggestion_id": suggestionID,
		"status":        body.Status,
		"po_id":         poID,
		"message":       fmt.Sprintf("Suggestion %s", body.Status),
	})
}
