package main

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"
)

func handleListQuotes(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []Quote
	for rows.Next() {
		var q Quote
		var aa sql.NullString
		rows.Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
		q.AcceptedAt = sp(aa)
		items = append(items, q)
	}
	if items == nil { items = []Quote{} }
	jsonResp(w, items)
}

func handleGetQuote(w http.ResponseWriter, r *http.Request, id string) {
	var q Quote
	var aa sql.NullString
	err := db.QueryRow("SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes WHERE id=?", id).
		Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
	if err != nil { jsonErr(w, "not found", 404); return }
	q.AcceptedAt = sp(aa)

	rows, _ := db.Query("SELECT id,quote_id,ipn,COALESCE(description,''),qty,COALESCE(unit_price,0),COALESCE(notes,'') FROM quote_lines WHERE quote_id=?", id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l QuoteLine
			rows.Scan(&l.ID, &l.QuoteID, &l.IPN, &l.Description, &l.Qty, &l.UnitPrice, &l.Notes)
			q.Lines = append(q.Lines, l)
		}
	}
	if q.Lines == nil { q.Lines = []QuoteLine{} }
	jsonResp(w, q)
}

func handleCreateQuote(w http.ResponseWriter, r *http.Request) {
	var q Quote
	if err := decodeBody(r, &q); err != nil { jsonErr(w, "invalid body", 400); return }
	q.ID = nextID("Q", "quotes", 3)
	if q.Status == "" { q.Status = "draft" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO quotes (id,customer,status,notes,created_at,valid_until) VALUES (?,?,?,?,?,?)",
		q.ID, q.Customer, q.Status, q.Notes, now, q.ValidUntil)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	for _, l := range q.Lines {
		db.Exec("INSERT INTO quote_lines (quote_id,ipn,description,qty,unit_price,notes) VALUES (?,?,?,?,?,?)",
			q.ID, l.IPN, l.Description, l.Qty, l.UnitPrice, l.Notes)
	}
	q.CreatedAt = now
	logAudit(db, getUsername(r), "created", "quote", q.ID, "Created "+q.ID+" for "+q.Customer)
	jsonResp(w, q)
}

func handleUpdateQuote(w http.ResponseWriter, r *http.Request, id string) {
	var q Quote
	if err := decodeBody(r, &q); err != nil { jsonErr(w, "invalid body", 400); return }
	_, err := db.Exec("UPDATE quotes SET customer=?,status=?,notes=?,valid_until=? WHERE id=?",
		q.Customer, q.Status, q.Notes, q.ValidUntil, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "quote", id, "Updated "+id+": status="+q.Status)
	handleGetQuote(w, r, id)
}

func handleQuoteCost(w http.ResponseWriter, r *http.Request, id string) {
	rows, err := db.Query("SELECT ipn,description,qty,COALESCE(unit_price,0) FROM quote_lines WHERE quote_id=?", id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	type MarginLine struct {
		IPN             string   `json:"ipn"`
		Qty             int      `json:"qty"`
		UnitPriceQuoted float64  `json:"unit_price_quoted"`
		BOMCost         *float64 `json:"bom_cost"`
		MarginPerUnit   *float64 `json:"margin_per_unit"`
		MarginPct       *float64 `json:"margin_pct"`
	}
	var lines []MarginLine
	totalQuoted := 0.0
	totalBOM := 0.0
	bomAvailable := false
	for rows.Next() {
		var ipn, desc string
		var qty int
		var unitPrice float64
		rows.Scan(&ipn, &desc, &qty, &unitPrice)
		_ = desc
		ml := MarginLine{IPN: ipn, Qty: qty, UnitPriceQuoted: unitPrice}
		totalQuoted += float64(qty) * unitPrice

		// Look up BOM cost from latest PO line
		var bomCost float64
		err := db.QueryRow(`SELECT pl.unit_price FROM po_lines pl JOIN purchase_orders po ON pl.po_id=po.id WHERE pl.ipn=? ORDER BY po.created_at DESC LIMIT 1`, ipn).Scan(&bomCost)
		if err == nil {
			ml.BOMCost = &bomCost
			margin := unitPrice - bomCost
			ml.MarginPerUnit = &margin
			if unitPrice > 0 {
				pct := math.Round(margin/unitPrice*10000) / 100
				ml.MarginPct = &pct
			}
			totalBOM += bomCost * float64(qty)
			bomAvailable = true
		}
		lines = append(lines, ml)
	}
	if lines == nil { lines = []MarginLine{} }

	result := map[string]interface{}{
		"quote_id":    id,
		"lines":       lines,
		"total_quoted": totalQuoted,
	}
	if bomAvailable {
		totalMargin := totalQuoted - totalBOM
		totalMarginPct := 0.0
		if totalQuoted > 0 {
			totalMarginPct = math.Round(totalMargin/totalQuoted*10000) / 100
		}
		result["total_bom_cost"] = totalBOM
		result["total_margin"] = totalMargin
		result["total_margin_pct"] = totalMarginPct
	}
	jsonResp(w, result)
}

func handleQuotePDF(w http.ResponseWriter, r *http.Request, id string) {
	var q Quote
	var aa sql.NullString
	err := db.QueryRow("SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes WHERE id=?", id).
		Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
	if err != nil {
		http.Error(w, "Quote not found", 404)
		return
	}

	rows, _ := db.Query("SELECT id,quote_id,ipn,COALESCE(description,''),qty,COALESCE(unit_price,0),COALESCE(notes,'') FROM quote_lines WHERE quote_id=?", id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l QuoteLine
			rows.Scan(&l.ID, &l.QuoteID, &l.IPN, &l.Description, &l.Qty, &l.UnitPrice, &l.Notes)
			q.Lines = append(q.Lines, l)
		}
	}

	lineRows := ""
	total := 0.0
	for _, l := range q.Lines {
		lineTotal := float64(l.Qty) * l.UnitPrice
		total += lineTotal
		lineRows += fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td style="text-align:center">%d</td><td style="text-align:right">$%.2f</td><td style="text-align:right">$%.2f</td></tr>`,
			l.IPN, l.Description, l.Qty, l.UnitPrice, lineTotal)
	}
	if lineRows == "" {
		lineRows = `<tr><td colspan="5" style="text-align:center;color:#999">No line items</td></tr>`
	}

	date := q.CreatedAt
	if len(date) > 10 {
		date = date[:10]
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Quote — %s</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: Arial, Helvetica, sans-serif; font-size: 11pt; color: #000; padding: 0.5in; }
  h1 { font-size: 18pt; margin-bottom: 2pt; }
  h2 { font-size: 13pt; margin: 16pt 0 6pt; border-bottom: 2px solid #000; padding-bottom: 3pt; }
  table { width: 100%%; border-collapse: collapse; margin-bottom: 12pt; }
  th, td { border: 1px solid #000; padding: 4pt 6pt; text-align: left; font-size: 10pt; }
  th { background: #eee; font-weight: bold; }
  .header { display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 3px solid #000; padding-bottom: 8pt; margin-bottom: 12pt; }
  .info-grid { display: grid; grid-template-columns: auto 1fr; gap: 4pt 12pt; margin-bottom: 12pt; font-size: 10pt; }
  .info-grid dt { font-weight: bold; }
  .total-row td { font-weight: bold; font-size: 11pt; }
  .footer { margin-top: 24pt; font-size: 9pt; color: #555; border-top: 1px solid #999; padding-top: 8pt; }
  @media print { body { padding: 0; } @page { margin: 0.5in; } }
</style>
</head><body>
<div class="header">
  <div>
    <h1>ZRP — Quote</h1>
    <div style="font-size:10pt;color:#555">` + companyName + `</div>
  </div>
  <div style="text-align:right;font-size:10pt">
    <div><strong>Quote:</strong> %s</div>
    <div><strong>Date:</strong> %s</div>
    <div><strong>Valid Until:</strong> %s</div>
    <div><strong>Status:</strong> %s</div>
  </div>
</div>

<h2>Customer</h2>
<div class="info-grid">
  <dt>Customer:</dt><dd>%s</dd>
</div>

<h2>Line Items</h2>
<table>
  <thead><tr><th>IPN</th><th>Description</th><th style="text-align:center">Qty</th><th style="text-align:right">Unit Price</th><th style="text-align:right">Total</th></tr></thead>
  <tbody>%s
    <tr class="total-row"><td colspan="4" style="text-align:right;border-top:2px solid #000">Subtotal:</td><td style="text-align:right;border-top:2px solid #000">$%.2f</td></tr>
  </tbody>
</table>

%s

<div class="footer">
  <p><strong>Terms:</strong> Net 30. Prices valid through the date shown above.</p>
  <p><strong>Contact:</strong> ` + companyEmail + ` | ` + companyName + `</p>
</div>

<script>window.onload = () => window.print()</script>
</body></html>`,
		q.ID, q.ID, date, q.ValidUntil, q.Status,
		q.Customer, lineRows, total,
		func() string {
			if q.Notes != "" {
				return fmt.Sprintf(`<h2>Notes</h2><p style="font-size:10pt">%s</p>`, q.Notes)
			}
			return ""
		}())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
