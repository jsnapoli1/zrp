package main

import (
	"database/sql"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"
)

func handleListWorkOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,assembly_ipn,qty,qty_good,qty_scrap,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders ORDER BY created_at DESC")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []WorkOrder
	for rows.Next() {
		var wo WorkOrder
		var sa, ca sql.NullString
		var qtyGood, qtyScrap sql.NullInt64
		rows.Scan(&wo.ID, &wo.AssemblyIPN, &wo.Qty, &qtyGood, &qtyScrap, &wo.Status, &wo.Priority, &wo.Notes, &wo.CreatedAt, &sa, &ca)
		wo.StartedAt = sp(sa); wo.CompletedAt = sp(ca)
		if qtyGood.Valid { good := int(qtyGood.Int64); wo.QtyGood = &good }
		if qtyScrap.Valid { scrap := int(qtyScrap.Int64); wo.QtyScrap = &scrap }
		items = append(items, wo)
	}
	if items == nil { items = []WorkOrder{} }
	jsonResp(w, items)
}

func handleGetWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
	var wo WorkOrder
	var sa, ca sql.NullString
	var qtyGood, qtyScrap sql.NullInt64
	err := db.QueryRow("SELECT id,assembly_ipn,qty,qty_good,qty_scrap,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders WHERE id=?", id).
		Scan(&wo.ID, &wo.AssemblyIPN, &wo.Qty, &qtyGood, &qtyScrap, &wo.Status, &wo.Priority, &wo.Notes, &wo.CreatedAt, &sa, &ca)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	wo.StartedAt = sp(sa); wo.CompletedAt = sp(ca)
	if qtyGood.Valid { good := int(qtyGood.Int64); wo.QtyGood = &good }
	if qtyScrap.Valid { scrap := int(qtyScrap.Int64); wo.QtyScrap = &scrap }
	jsonResp(w, wo)
}

func handleCreateWorkOrder(w http.ResponseWriter, r *http.Request) {
	var wo WorkOrder
	if err := decodeBody(r, &wo); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	requireField(ve, "assembly_ipn", wo.AssemblyIPN)
	validateMaxLength(ve, "assembly_ipn", wo.AssemblyIPN, 100)
	validateMaxLength(ve, "notes", wo.Notes, 10000)
	if wo.Status != "" { validateEnum(ve, "status", wo.Status, validWOStatuses) }
	if wo.Priority != "" { validateEnum(ve, "priority", wo.Priority, validWOPriorities) }
	if wo.Qty < 0 { ve.Add("qty", "must be non-negative") }
	validateIntRange(ve, "qty", wo.Qty, 1, MaxWorkOrderQty)
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	wo.ID = nextID("WO", "work_orders", 4)
	if wo.Status == "" { wo.Status = "open" }
	if wo.Priority == "" { wo.Priority = "normal" }
	if wo.Qty == 0 { wo.Qty = 1 }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO work_orders (id,assembly_ipn,qty,qty_good,qty_scrap,status,priority,notes,created_at) VALUES (?,?,?,?,?,?,?,?,?)",
		wo.ID, wo.AssemblyIPN, wo.Qty, wo.QtyGood, wo.QtyScrap, wo.Status, wo.Priority, wo.Notes, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	wo.CreatedAt = now
	logAudit(db, getUsername(r), "created", "workorder", wo.ID, "Created WO "+wo.ID+" for "+wo.AssemblyIPN)
	recordChangeJSON(getUsername(r), "work_orders", wo.ID, "create", nil, wo)
	jsonResp(w, wo)
}

func handleUpdateWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
	oldSnap, _ := getWorkOrderSnapshot(id)
	var wo WorkOrder
	if err := decodeBody(r, &wo); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	// Get current work order state for validation
	var currentWO WorkOrder
	var sa, ca sql.NullString
	var qtyGood, qtyScrap sql.NullInt64
	err := db.QueryRow("SELECT id,assembly_ipn,qty,qty_good,qty_scrap,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders WHERE id=?", id).
		Scan(&currentWO.ID, &currentWO.AssemblyIPN, &currentWO.Qty, &qtyGood, &qtyScrap, &currentWO.Status, &currentWO.Priority, &currentWO.Notes, &currentWO.CreatedAt, &sa, &ca)
	if err != nil {
		jsonErr(w, "work order not found", 404)
		return
	}
	
	if qtyGood.Valid { good := int(qtyGood.Int64); currentWO.QtyGood = &good }
	if qtyScrap.Valid { scrap := int(qtyScrap.Int64); currentWO.QtyScrap = &scrap }

	// Status transition validation
	ve := &ValidationErrors{}
	validateMaxLength(ve, "assembly_ipn", wo.AssemblyIPN, 100)
	validateMaxLength(ve, "notes", wo.Notes, 10000)
	if wo.Status != "" {
		validateEnum(ve, "status", wo.Status, validWOStatuses)
		// Enforce status state machine transitions
		if !isValidStatusTransition(currentWO.Status, wo.Status) {
			ve.Add("status", fmt.Sprintf("invalid transition from %s to %s", currentWO.Status, wo.Status))
		}
	}
	if wo.Priority != "" { validateEnum(ve, "priority", wo.Priority, validWOPriorities) }
	if wo.Qty < 0 { ve.Add("qty", "must be non-negative") }
	if wo.QtyGood != nil && *wo.QtyGood < 0 { ve.Add("qty_good", "must be non-negative") }
	if wo.QtyScrap != nil && *wo.QtyScrap < 0 { ve.Add("qty_scrap", "must be non-negative") }
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	now := time.Now().Format("2006-01-02 15:04:05")
	
	// Start transaction for atomic updates
	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	// Update work order
	_, err = tx.Exec("UPDATE work_orders SET assembly_ipn=?,qty=?,qty_good=?,qty_scrap=?,status=?,priority=?,notes=?,started_at=CASE WHEN ?='in_progress' AND started_at IS NULL THEN ? ELSE started_at END,completed_at=CASE WHEN ?='completed' THEN ? ELSE completed_at END WHERE id=?",
		wo.AssemblyIPN, wo.Qty, wo.QtyGood, wo.QtyScrap, wo.Status, wo.Priority, wo.Notes, wo.Status, now, wo.Status, now, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Handle inventory integration on completion
	if wo.Status == "completed" && currentWO.Status != "completed" {
		err = handleWorkOrderCompletion(tx, id, wo.AssemblyIPN, wo.Qty, getUsername(r))
		if err != nil {
			jsonErr(w, "failed to update inventory on completion: "+err.Error(), 500)
			return
		}
	}

	// Handle inventory reservation release on cancellation
	if wo.Status == "cancelled" && currentWO.Status != "cancelled" {
		err = handleWorkOrderCancellation(tx, id)
		if err != nil {
			jsonErr(w, "failed to release inventory on cancellation: "+err.Error(), 500)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "updated", "workorder", id, "Updated WO "+id+": status="+wo.Status)
	newSnap, _ := getWorkOrderSnapshot(id)
	recordChangeJSON(getUsername(r), "work_orders", id, "update", oldSnap, newSnap)
	go emailOnOverdueWorkOrder(id)
	handleGetWorkOrder(w, r, id)
}

func isValidStatusTransition(from, to string) bool {
	// Define valid state machine transitions
	validTransitions := map[string][]string{
		"draft":       {"open", "cancelled"},
		"open":        {"in_progress", "on_hold", "cancelled"},
		"in_progress": {"completed", "on_hold", "cancelled"},
		"on_hold":     {"in_progress", "open", "cancelled"},
		"completed":   {}, // Terminal state
		"cancelled":   {}, // Terminal state
	}
	
	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}
	
	for _, state := range allowedStates {
		if state == to {
			return true
		}
	}
	return false
}

func handleWorkOrderCompletion(tx *sql.Tx, woID, assemblyIPN string, qty int, username string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	
	// 1. Add finished goods to inventory
	// First ensure inventory record exists for the assembly
	_, err := tx.Exec("INSERT OR IGNORE INTO inventory (ipn, description) VALUES (?, ?)", 
		assemblyIPN, "Assembled "+assemblyIPN)
	if err != nil {
		return fmt.Errorf("failed to create inventory record: %w", err)
	}
	
	// Add finished goods quantity
	_, err = tx.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand + ?, updated_at = ? WHERE ipn = ?",
		qty, now, assemblyIPN)
	if err != nil {
		return fmt.Errorf("failed to update finished goods inventory: %w", err)
	}
	
	// Log finished goods transaction
	_, err = tx.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
		assemblyIPN, "receive", qty, woID, "WO "+woID+" completion", now)
	if err != nil {
		return fmt.Errorf("failed to log finished goods transaction: %w", err)
	}
	
	// 2. Deduct consumed materials based on BOM (simplified - using all inventory items for now)
	rows, err := tx.Query("SELECT ipn, qty_reserved FROM inventory WHERE qty_reserved > 0")
	if err != nil {
		return fmt.Errorf("failed to query reserved materials: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var ipn string
		var reserved float64
		if err := rows.Scan(&ipn, &reserved); err != nil {
			continue
		}
		
		// Consume the reserved quantity
		// TODO: Track which inventory is reserved for which WO to avoid consuming
		// inventory reserved for other WOs
		consumed := reserved
		
		// Deduct from on_hand and release reservation
		_, err = tx.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand - ?, qty_reserved = qty_reserved - ?, updated_at = ? WHERE ipn = ?",
			consumed, reserved, now, ipn)
		if err != nil {
			return fmt.Errorf("failed to consume material %s: %w", ipn, err)
		}
		
		// Log material consumption
		_, err = tx.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
			ipn, "issue", consumed, woID, "WO "+woID+" material consumption", now)
		if err != nil {
			return fmt.Errorf("failed to log material consumption: %w", err)
		}
	}
	
	return nil
}

func handleWorkOrderCancellation(tx *sql.Tx, woID string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	
	// Release all reserved inventory for this work order
	// (Simplified implementation: releases ALL reserved inventory)
	// TODO: Track which inventory items are reserved for which WO
	rows, err := tx.Query("SELECT ipn, qty_reserved FROM inventory WHERE qty_reserved > 0")
	if err != nil {
		return fmt.Errorf("failed to query reserved materials: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var ipn string
		var reserved float64
		if err := rows.Scan(&ipn, &reserved); err != nil {
			continue
		}
		
		// Release the reservation (don't consume inventory)
		_, err = tx.Exec("UPDATE inventory SET qty_reserved = qty_reserved - ?, updated_at = ? WHERE ipn = ?",
			reserved, now, ipn)
		if err != nil {
			return fmt.Errorf("failed to release reservation for %s: %w", ipn, err)
		}
		
		// Log the reservation release
		_, err = tx.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
			ipn, "return", reserved, woID, "WO "+woID+" cancelled - reservation released", now)
		if err != nil {
			return fmt.Errorf("failed to log reservation release: %w", err)
		}
	}
	
	return nil
}

func handleWorkOrderBOM(w http.ResponseWriter, r *http.Request, id string) {
	var assemblyIPN string
	var qty int
	err := db.QueryRow("SELECT assembly_ipn,qty FROM work_orders WHERE id=?", id).Scan(&assemblyIPN, &qty)
	if err != nil { jsonErr(w, "not found", 404); return }

	type BOMLine struct {
		IPN         string  `json:"ipn"`
		Description string  `json:"description"`
		QtyRequired float64 `json:"qty_required"`
		QtyOnHand   float64 `json:"qty_on_hand"`
		Shortage    float64 `json:"shortage"`
		Status      string  `json:"status"`
	}
	
	// PERFORMANCE NOTE: This loads all inventory instead of actual BOM from CSV.
	// TODO: Use buildBOMTree() from handler_parts.go to get real BOM, then query only those IPNs
	// Current optimization: Filter to only parts with stock or reserved quantities
	rows, _ := db.Query(`SELECT ipn, qty_on_hand FROM inventory 
		WHERE qty_on_hand > 0 OR qty_reserved > 0 
		ORDER BY ipn LIMIT 1000`)
	var bom []BOMLine
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var bl BOMLine
			rows.Scan(&bl.IPN, &bl.QtyOnHand)
			bl.QtyRequired = float64(qty)
			bl.Shortage = bl.QtyRequired - bl.QtyOnHand
			if bl.Shortage < 0 { bl.Shortage = 0 }
			if bl.QtyOnHand >= bl.QtyRequired {
				bl.Status = "ok"
			} else if bl.QtyOnHand > 0 {
				bl.Status = "low"
			} else {
				bl.Status = "shortage"
			}
			fields, err := getPartByIPN(partsDir, bl.IPN)
			if err == nil {
				for k, v := range fields {
					if strings.EqualFold(k, "description") || strings.EqualFold(k, "desc") {
						bl.Description = v
						break
					}
				}
			}
			bom = append(bom, bl)
		}
	}
	if bom == nil { bom = []BOMLine{} }
	jsonResp(w, map[string]interface{}{"wo_id": id, "assembly_ipn": assemblyIPN, "qty": qty, "bom": bom})
}

func handleWorkOrderPDF(w http.ResponseWriter, r *http.Request, id string) {
	var wo WorkOrder
	var sa, ca sql.NullString
	err := db.QueryRow("SELECT id,assembly_ipn,qty,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders WHERE id=?", id).
		Scan(&wo.ID, &wo.AssemblyIPN, &wo.Qty, &wo.Status, &wo.Priority, &wo.Notes, &wo.CreatedAt, &sa, &ca)
	if err != nil {
		http.Error(w, "Work order not found", 404)
		return
	}
	wo.StartedAt = sp(sa)
	wo.CompletedAt = sp(ca)

	type BOMLine struct {
		IPN          string
		Description  string
		MPN          string
		Manufacturer string
		QtyRequired  float64
		QtyOnHand    float64
		RefDes       string
	}
	
	// PERFORMANCE NOTE: This loads all inventory instead of actual BOM from CSV.
	// TODO: Use buildBOMTree() from handler_parts.go to get real BOM, then query only those IPNs
	// Current optimization: Filter to only parts with stock or reserved quantities
	rows, _ := db.Query(`SELECT ipn, qty_on_hand FROM inventory 
		WHERE qty_on_hand > 0 OR qty_reserved > 0 
		ORDER BY ipn LIMIT 1000`)
	var bom []BOMLine
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var bl BOMLine
			rows.Scan(&bl.IPN, &bl.QtyOnHand)
			bl.QtyRequired = float64(wo.Qty)
			fields, ferr := getPartByIPN(partsDir, bl.IPN)
			if ferr == nil {
				for k, v := range fields {
					kl := strings.ToLower(k)
					if kl == "description" || kl == "desc" {
						bl.Description = v
					} else if kl == "mpn" {
						bl.MPN = v
					} else if kl == "manufacturer" || kl == "mfr" {
						bl.Manufacturer = v
					} else if kl == "reference" || kl == "refdes" || kl == "ref_des" {
						bl.RefDes = v
					}
				}
			}
			bom = append(bom, bl)
		}
	}

	assemblyDesc := ""
	if fields, ferr := getPartByIPN(partsDir, wo.AssemblyIPN); ferr == nil {
		for k, v := range fields {
			if strings.EqualFold(k, "description") || strings.EqualFold(k, "desc") {
				assemblyDesc = v
				break
			}
		}
	}

	bomRows := ""
	for _, bl := range bom {
		bomRows += fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td style="text-align:center">%.0f</td><td>%s</td></tr>`,
			html.EscapeString(bl.IPN), html.EscapeString(bl.Description), html.EscapeString(bl.MPN), html.EscapeString(bl.Manufacturer), bl.QtyRequired, html.EscapeString(bl.RefDes))
	}
	if bomRows == "" {
		bomRows = `<tr><td colspan="6" style="text-align:center;color:#999">No BOM data</td></tr>`
	}

	date := wo.CreatedAt
	if len(date) > 10 {
		date = date[:10]
	}

	htmlOutput := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Work Order Traveler — %s</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: Arial, Helvetica, sans-serif; font-size: 11pt; color: #000; padding: 0.5in; }
  h1 { font-size: 18pt; margin-bottom: 2pt; }
  h2 { font-size: 13pt; margin: 16pt 0 6pt; border-bottom: 2px solid #000; padding-bottom: 3pt; }
  table { width: 100%%; border-collapse: collapse; margin-bottom: 12pt; }
  th, td { border: 1px solid #000; padding: 4pt 6pt; text-align: left; font-size: 10pt; }
  th { background: #eee; font-weight: bold; }
  .header { display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 3px solid #000; padding-bottom: 8pt; margin-bottom: 12pt; }
  .header-left { }
  .header-right { text-align: right; font-size: 10pt; }
  .info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 4pt 20pt; margin-bottom: 12pt; font-size: 10pt; }
  .info-grid dt { font-weight: bold; }
  .signoff td { height: 40pt; vertical-align: bottom; }
  .signoff td.label-cell { width: 120pt; font-weight: bold; }
  @media print { body { padding: 0; } @page { margin: 0.5in; } }
</style>
</head><body>
<div class="header">
  <div class="header-left">
    <h1>ZRP — Work Order Traveler</h1>
    <div style="font-size:10pt;color:#555">`+companyName+`</div>
  </div>
  <div class="header-right">
    <div><strong>WO:</strong> %s</div>
    <div><strong>Date:</strong> %s</div>
    <div><strong>Status:</strong> %s</div>
    <div><strong>Priority:</strong> %s</div>
  </div>
</div>

<h2>Assembly Information</h2>
<div class="info-grid">
  <dt>Assembly IPN:</dt><dd>%s</dd>
  <dt>Description:</dt><dd>%s</dd>
  <dt>Quantity:</dt><dd>%d</dd>
  <dt>Notes:</dt><dd>%s</dd>
</div>

<h2>Bill of Materials</h2>
<table>
  <thead><tr><th>IPN</th><th>Description</th><th>MPN</th><th>Manufacturer</th><th>Qty Req</th><th>Ref Des</th></tr></thead>
  <tbody>%s</tbody>
</table>

<h2>Sign-Off</h2>
<table class="signoff">
  <thead><tr><th style="width:120pt">Step</th><th>Name</th><th style="width:100pt">Date</th><th>Signature</th></tr></thead>
  <tbody>
    <tr><td class="label-cell">Kitted by</td><td></td><td></td><td></td></tr>
    <tr><td class="label-cell">Built by</td><td></td><td></td><td></td></tr>
    <tr><td class="label-cell">Tested by</td><td></td><td></td><td></td></tr>
    <tr><td class="label-cell">QA Approved by</td><td></td><td></td><td></td></tr>
  </tbody>
</table>

<script>window.onload = () => window.print()</script>
</body></html>`,
		html.EscapeString(wo.ID), html.EscapeString(wo.ID), html.EscapeString(date), html.EscapeString(wo.Status), html.EscapeString(wo.Priority),
		html.EscapeString(wo.AssemblyIPN), html.EscapeString(assemblyDesc), wo.Qty, html.EscapeString(wo.Notes), bomRows)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Write([]byte(htmlOutput))
}

func handleWorkOrderKit(w http.ResponseWriter, r *http.Request, id string) {
	// Check if work order exists
	var assemblyIPN string
	var qty int
	var status string
	err := db.QueryRow("SELECT assembly_ipn,qty,status FROM work_orders WHERE id=?", id).Scan(&assemblyIPN, &qty, &status)
	if err != nil {
		jsonErr(w, "work order not found", 404)
		return
	}

	// Don't allow kitting if already completed or cancelled
	if status == "completed" || status == "cancelled" {
		jsonErr(w, "cannot kit materials for completed or cancelled work order", 400)
		return
	}

	// Get BOM requirements (simplified - using inventory as BOM for now)
	type KitResult struct {
		IPN         string  `json:"ipn"`
		Required    float64 `json:"required"`
		OnHand      float64 `json:"on_hand"`
		Reserved    float64 `json:"reserved"`
		Kitted      float64 `json:"kitted"`
		Status      string  `json:"status"`
	}

	// First, read all inventory data (close cursor before starting transaction)
	rows, err := db.Query("SELECT ipn, qty_on_hand, qty_reserved FROM inventory")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	
	type inventorySnapshot struct {
		ipn      string
		onHand   float64
		reserved float64
	}
	var snapshots []inventorySnapshot
	
	for rows.Next() {
		var snap inventorySnapshot
		err := rows.Scan(&snap.ipn, &snap.onHand, &snap.reserved)
		if err != nil {
			continue
		}
		snapshots = append(snapshots, snap)
	}
	rows.Close() // Close the query before starting transaction

	var kitResults []KitResult
	
	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	// Now process each inventory item within the transaction
	for _, snap := range snapshots {
		var result KitResult
		result.IPN = snap.ipn
		result.OnHand = snap.onHand
		result.Reserved = snap.reserved
		result.Required = float64(qty) // Simple 1:1 ratio for now
		available := result.OnHand - result.Reserved
		
		if available >= result.Required {
			// Reserve the materials
			result.Kitted = result.Required
			result.Status = "kitted"
			_, err = tx.Exec("UPDATE inventory SET qty_reserved = qty_reserved + ? WHERE ipn = ?", 
				result.Required, result.IPN)
			if err != nil {
				result.Status = "error"
				result.Kitted = 0
			}
		} else if available > 0 {
			// Partial kit
			result.Kitted = available
			result.Status = "partial"
			_, err = tx.Exec("UPDATE inventory SET qty_reserved = qty_reserved + ? WHERE ipn = ?", 
				available, result.IPN)
			if err != nil {
				result.Status = "error"
				result.Kitted = 0
			}
		} else {
			result.Status = "shortage"
		}
		
		kitResults = append(kitResults, result)
	}

	// Allow kitting to succeed even with partial/shortage - just report the status
	// In a real system, this would check BOM requirements, not all inventory
	// Always commit what we can kit
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = tx.Exec("UPDATE work_orders SET status = CASE WHEN status = 'open' THEN 'in_progress' ELSE status END WHERE id = ?", id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	
	if err = tx.Commit(); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	
	// Log the kitting activity (after commit to avoid deadlock)
	logAudit(db, getUsername(r), "kitted", "workorder", id, "Kitted materials for WO "+id)
	
	jsonResp(w, map[string]interface{}{
		"wo_id": id,
		"status": "kitted",
		"items": kitResults,
		"kitted_at": now,
	})
}

func handleWorkOrderSerials(w http.ResponseWriter, r *http.Request, id string) {
	// Verify work order exists
	var exists int
	err := db.QueryRow("SELECT 1 FROM work_orders WHERE id=?", id).Scan(&exists)
	if err != nil {
		jsonErr(w, "work order not found", 404)
		return
	}

	rows, err := db.Query("SELECT id,wo_id,serial_number,status,COALESCE(notes,'') FROM wo_serials WHERE wo_id=? ORDER BY serial_number", id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var serials []WOSerial
	for rows.Next() {
		var serial WOSerial
		err := rows.Scan(&serial.ID, &serial.WOID, &serial.SerialNumber, &serial.Status, &serial.Notes)
		if err != nil {
			continue
		}
		serials = append(serials, serial)
	}
	
	if serials == nil {
		serials = []WOSerial{}
	}
	
	jsonResp(w, serials)
}

func handleWorkOrderAddSerial(w http.ResponseWriter, r *http.Request, id string) {
	// Verify work order exists
	var assemblyIPN string
	var woStatus string
	err := db.QueryRow("SELECT assembly_ipn,status FROM work_orders WHERE id=?", id).Scan(&assemblyIPN, &woStatus)
	if err != nil {
		jsonErr(w, "work order not found", 404)
		return
	}

	// Don't allow adding serials to completed or cancelled WOs
	if woStatus == "completed" || woStatus == "cancelled" {
		jsonErr(w, "cannot add serials to completed or cancelled work order", 400)
		return
	}

	var serial WOSerial
	if err := decodeBody(r, &serial); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	if serial.SerialNumber == "" {
		// Auto-generate serial number if not provided
		serial.SerialNumber = generateSerialNumber(assemblyIPN)
	}
	
	// Check for duplicate serial number
	var count int
	db.QueryRow("SELECT COUNT(*) FROM wo_serials WHERE serial_number=?", serial.SerialNumber).Scan(&count)
	if count > 0 {
		ve.Add("serial_number", "serial number already exists")
	}
	
	if ve.HasErrors() {
		jsonErr(w, ve.Error(), 400)
		return
	}

	serial.WOID = id
	if serial.Status == "" {
		serial.Status = "building" // Default per wo_serials schema CHECK constraint
	}

	_, err = db.Exec("INSERT INTO wo_serials (wo_id,serial_number,status,notes) VALUES (?,?,?,?)",
		serial.WOID, serial.SerialNumber, serial.Status, serial.Notes)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "created", "serial", serial.SerialNumber, "Added serial "+serial.SerialNumber+" to WO "+id)
	
	// Get the created serial with ID
	err = db.QueryRow("SELECT id,wo_id,serial_number,status,COALESCE(notes,'') FROM wo_serials WHERE wo_id=? AND serial_number=?", 
		id, serial.SerialNumber).Scan(&serial.ID, &serial.WOID, &serial.SerialNumber, &serial.Status, &serial.Notes)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	
	jsonResp(w, serial)
}

func generateSerialNumber(assemblyIPN string) string {
	// Simple serial number generation: IPN prefix + timestamp
	prefix := strings.Split(assemblyIPN, "-")[0]
	if len(prefix) > 3 {
		prefix = prefix[:3]
	}
	timestamp := time.Now().Format("060102150405") // YYMMDDHHMMSS
	return fmt.Sprintf("%s%s", strings.ToUpper(prefix), timestamp)
}
