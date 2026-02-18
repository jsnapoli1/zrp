package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func handleListWorkOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,assembly_ipn,qty,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []WorkOrder
	for rows.Next() {
		var wo WorkOrder
		var sa, ca sql.NullString
		rows.Scan(&wo.ID, &wo.AssemblyIPN, &wo.Qty, &wo.Status, &wo.Priority, &wo.Notes, &wo.CreatedAt, &sa, &ca)
		wo.StartedAt = sp(sa); wo.CompletedAt = sp(ca)
		items = append(items, wo)
	}
	if items == nil { items = []WorkOrder{} }
	jsonResp(w, items)
}

func handleGetWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
	var wo WorkOrder
	var sa, ca sql.NullString
	err := db.QueryRow("SELECT id,assembly_ipn,qty,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders WHERE id=?", id).
		Scan(&wo.ID, &wo.AssemblyIPN, &wo.Qty, &wo.Status, &wo.Priority, &wo.Notes, &wo.CreatedAt, &sa, &ca)
	if err != nil { jsonErr(w, "not found", 404); return }
	wo.StartedAt = sp(sa); wo.CompletedAt = sp(ca)
	jsonResp(w, wo)
}

func handleCreateWorkOrder(w http.ResponseWriter, r *http.Request) {
	var wo WorkOrder
	if err := decodeBody(r, &wo); err != nil { jsonErr(w, "invalid body", 400); return }
	wo.ID = nextID("WO", "work_orders", 4)
	if wo.Status == "" { wo.Status = "open" }
	if wo.Priority == "" { wo.Priority = "normal" }
	if wo.Qty == 0 { wo.Qty = 1 }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO work_orders (id,assembly_ipn,qty,status,priority,notes,created_at) VALUES (?,?,?,?,?,?,?)",
		wo.ID, wo.AssemblyIPN, wo.Qty, wo.Status, wo.Priority, wo.Notes, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	wo.CreatedAt = now
	logAudit(db, getUsername(r), "created", "workorder", wo.ID, "Created WO "+wo.ID+" for "+wo.AssemblyIPN)
	jsonResp(w, wo)
}

func handleUpdateWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
	var wo WorkOrder
	if err := decodeBody(r, &wo); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE work_orders SET assembly_ipn=?,qty=?,status=?,priority=?,notes=?,started_at=CASE WHEN ?='in_progress' AND started_at IS NULL THEN ? ELSE started_at END,completed_at=CASE WHEN ?='completed' THEN ? ELSE completed_at END WHERE id=?",
		wo.AssemblyIPN, wo.Qty, wo.Status, wo.Priority, wo.Notes, wo.Status, now, wo.Status, now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "workorder", id, "Updated WO "+id+": status="+wo.Status)
	go emailOnOverdueWorkOrder(id)
	handleGetWorkOrder(w, r, id)
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
	// Get inventory items and enrich with part descriptions
	rows, _ := db.Query("SELECT ipn, qty_on_hand FROM inventory")
	var bom []BOMLine
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var bl BOMLine
			rows.Scan(&bl.IPN, &bl.QtyOnHand)
			bl.QtyRequired = float64(qty) // simplified: 1 per unit × WO qty
			bl.Shortage = bl.QtyRequired - bl.QtyOnHand
			if bl.Shortage < 0 { bl.Shortage = 0 }
			if bl.QtyOnHand >= bl.QtyRequired {
				bl.Status = "ok"
			} else if bl.QtyOnHand > 0 {
				bl.Status = "low"
			} else {
				bl.Status = "shortage"
			}
			// Try to get description from parts DB
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

	// Get BOM data
	type BOMLine struct {
		IPN          string
		Description  string
		MPN          string
		Manufacturer string
		QtyRequired  float64
		QtyOnHand    float64
		RefDes       string
	}
	rows, _ := db.Query("SELECT ipn, qty_on_hand FROM inventory")
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

	// Get assembly description
	assemblyDesc := ""
	if fields, ferr := getPartByIPN(partsDir, wo.AssemblyIPN); ferr == nil {
		for k, v := range fields {
			if strings.EqualFold(k, "description") || strings.EqualFold(k, "desc") {
				assemblyDesc = v
				break
			}
		}
	}

	// Build BOM rows HTML
	bomRows := ""
	for _, bl := range bom {
		bomRows += fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td style="text-align:center">%.0f</td><td>%s</td></tr>`,
			bl.IPN, bl.Description, bl.MPN, bl.Manufacturer, bl.QtyRequired, bl.RefDes)
	}
	if bomRows == "" {
		bomRows = `<tr><td colspan="6" style="text-align:center;color:#999">No BOM data</td></tr>`
	}

	date := wo.CreatedAt
	if len(date) > 10 {
		date = date[:10]
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
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
    <div style="font-size:10pt;color:#555">` + companyName + `</div>
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
		wo.ID, wo.ID, date, wo.Status, wo.Priority,
		wo.AssemblyIPN, assemblyDesc, wo.Qty, wo.Notes, bomRows)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
