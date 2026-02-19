package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// --- RFQ Handlers ---

func handleListRFQs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, title, status, created_by, created_at, updated_at, COALESCE(due_date,''), COALESCE(notes,'') FROM rfqs ORDER BY created_at DESC`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []RFQ
	for rows.Next() {
		var rfq RFQ
		rows.Scan(&rfq.ID, &rfq.Title, &rfq.Status, &rfq.CreatedBy, &rfq.CreatedAt, &rfq.UpdatedAt, &rfq.DueDate, &rfq.Notes)
		items = append(items, rfq)
	}
	if items == nil {
		items = []RFQ{}
	}
	jsonResp(w, items)
}

func handleGetRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var rfq RFQ
	err := db.QueryRow(`SELECT id, title, status, created_by, created_at, updated_at, COALESCE(due_date,''), COALESCE(notes,'') FROM rfqs WHERE id=?`, id).
		Scan(&rfq.ID, &rfq.Title, &rfq.Status, &rfq.CreatedBy, &rfq.CreatedAt, &rfq.UpdatedAt, &rfq.DueDate, &rfq.Notes)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	// Load lines
	lineRows, _ := db.Query(`SELECT id, rfq_id, ipn, description, qty, unit FROM rfq_lines WHERE rfq_id=?`, id)
	if lineRows != nil {
		defer lineRows.Close()
		for lineRows.Next() {
			var l RFQLine
			lineRows.Scan(&l.ID, &l.RFQID, &l.IPN, &l.Description, &l.Qty, &l.Unit)
			rfq.Lines = append(rfq.Lines, l)
		}
	}
	if rfq.Lines == nil {
		rfq.Lines = []RFQLine{}
	}

	// Load vendors
	vRows, _ := db.Query(`SELECT rv.id, rv.rfq_id, rv.vendor_id, rv.status, COALESCE(rv.quoted_at,''), COALESCE(rv.notes,''), COALESCE(v.name,'') FROM rfq_vendors rv LEFT JOIN vendors v ON rv.vendor_id=v.id WHERE rv.rfq_id=?`, id)
	if vRows != nil {
		defer vRows.Close()
		for vRows.Next() {
			var v RFQVendor
			vRows.Scan(&v.ID, &v.RFQID, &v.VendorID, &v.Status, &v.QuotedAt, &v.Notes, &v.VendorName)
			rfq.Vendors = append(rfq.Vendors, v)
		}
	}
	if rfq.Vendors == nil {
		rfq.Vendors = []RFQVendor{}
	}

	// Load quotes
	qRows, _ := db.Query(`SELECT id, rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq, COALESCE(notes,'') FROM rfq_quotes WHERE rfq_id=?`, id)
	if qRows != nil {
		defer qRows.Close()
		for qRows.Next() {
			var q RFQQuote
			qRows.Scan(&q.ID, &q.RFQID, &q.RFQVendorID, &q.RFQLineID, &q.UnitPrice, &q.LeadTimeDays, &q.MOQ, &q.Notes)
			rfq.Quotes = append(rfq.Quotes, q)
		}
	}
	if rfq.Quotes == nil {
		rfq.Quotes = []RFQQuote{}
	}

	jsonResp(w, rfq)
}

func handleCreateRFQ(w http.ResponseWriter, r *http.Request) {
	var rfq RFQ
	if err := decodeBody(r, &rfq); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if rfq.Title == "" {
		jsonErr(w, "title required", 400)
		return
	}
	rfq.ID = nextID("RFQ", "rfqs", 4)
	rfq.Status = "draft"
	rfq.CreatedBy = getUser(r)
	now := time.Now().Format(time.RFC3339)
	rfq.CreatedAt = now
	rfq.UpdatedAt = now

	_, err := db.Exec(`INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at, due_date, notes) VALUES (?,?,?,?,?,?,?,?)`,
		rfq.ID, rfq.Title, rfq.Status, rfq.CreatedBy, rfq.CreatedAt, rfq.UpdatedAt, rfq.DueDate, rfq.Notes)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Insert lines
	for i, l := range rfq.Lines {
		res, _ := db.Exec(`INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?,?,?,?,?)`,
			rfq.ID, l.IPN, l.Description, l.Qty, l.Unit)
		if res != nil {
			lid, _ := res.LastInsertId()
			rfq.Lines[i].ID = int(lid)
			rfq.Lines[i].RFQID = rfq.ID
		}
	}

	// Insert vendors
	for i, v := range rfq.Vendors {
		res, _ := db.Exec(`INSERT INTO rfq_vendors (rfq_id, vendor_id, status, notes) VALUES (?,?,?,?)`,
			rfq.ID, v.VendorID, "pending", v.Notes)
		if res != nil {
			vid, _ := res.LastInsertId()
			rfq.Vendors[i].ID = int(vid)
			rfq.Vendors[i].RFQID = rfq.ID
			rfq.Vendors[i].Status = "pending"
		}
	}

	logAudit(db, getUser(r), "create", "rfq", rfq.ID, "Created RFQ: "+rfq.Title)
	w.WriteHeader(201)
	jsonResp(w, rfq)
}

func handleUpdateRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var existing RFQ
	err := db.QueryRow(`SELECT id, status FROM rfqs WHERE id=?`, id).Scan(&existing.ID, &existing.Status)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	var rfq RFQ
	if err := decodeBody(r, &rfq); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	now := time.Now().Format(time.RFC3339)
	_, err = db.Exec(`UPDATE rfqs SET title=?, due_date=?, notes=?, updated_at=? WHERE id=?`,
		rfq.Title, rfq.DueDate, rfq.Notes, now, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Replace lines
	db.Exec(`DELETE FROM rfq_lines WHERE rfq_id=?`, id)
	for _, l := range rfq.Lines {
		db.Exec(`INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?,?,?,?,?)`,
			id, l.IPN, l.Description, l.Qty, l.Unit)
	}

	// Replace vendors
	db.Exec(`DELETE FROM rfq_vendors WHERE rfq_id=?`, id)
	for _, v := range rfq.Vendors {
		db.Exec(`INSERT INTO rfq_vendors (rfq_id, vendor_id, status, notes) VALUES (?,?,?,?)`,
			id, v.VendorID, "pending", v.Notes)
	}

	logAudit(db, getUser(r), "update", "rfq", id, "Updated RFQ")
	handleGetRFQ(w, r, id)
}

func handleDeleteRFQ(w http.ResponseWriter, r *http.Request, id string) {
	res, err := db.Exec(`DELETE FROM rfqs WHERE id=?`, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		jsonErr(w, "not found", 404)
		return
	}
	db.Exec(`DELETE FROM rfq_lines WHERE rfq_id=?`, id)
	db.Exec(`DELETE FROM rfq_vendors WHERE rfq_id=?`, id)
	db.Exec(`DELETE FROM rfq_quotes WHERE rfq_id=?`, id)
	logAudit(db, getUser(r), "delete", "rfq", id, "Deleted RFQ")
	jsonResp(w, map[string]string{"status": "deleted"})
}

func handleSendRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var status string
	err := db.QueryRow(`SELECT status FROM rfqs WHERE id=?`, id).Scan(&status)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	if status != "draft" {
		jsonErr(w, "RFQ must be in draft status to send", 400)
		return
	}

	now := time.Now().Format(time.RFC3339)
	db.Exec(`UPDATE rfqs SET status='sent', updated_at=? WHERE id=?`, now, id)
	db.Exec(`UPDATE rfq_vendors SET status='pending' WHERE rfq_id=?`, id)

	logAudit(db, getUser(r), "send", "rfq", id, "Sent RFQ to vendors")
	handleGetRFQ(w, r, id)
}

func handleAwardRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var status string
	err := db.QueryRow(`SELECT status FROM rfqs WHERE id=?`, id).Scan(&status)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	var body struct {
		VendorID string `json:"vendor_id"`
	}
	if err := decodeBody(r, &body); err != nil || body.VendorID == "" {
		jsonErr(w, "vendor_id required", 400)
		return
	}

	// Find the rfq_vendor entry
	var rfqVendorID int
	err = db.QueryRow(`SELECT id FROM rfq_vendors WHERE rfq_id=? AND vendor_id=?`, id, body.VendorID).Scan(&rfqVendorID)
	if err != nil {
		jsonErr(w, "vendor not in this RFQ", 400)
		return
	}

	now := time.Now().Format(time.RFC3339)
	db.Exec(`UPDATE rfqs SET status='awarded', updated_at=? WHERE id=?`, now, id)

	// Auto-create PO from winning quotes
	poID := nextID("PO", "purchase_orders", 4)
	db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, notes, created_at) VALUES (?,?,?,?,?)`,
		poID, body.VendorID, "draft", "Auto-created from "+id, now)

	// Get quotes for winning vendor and create PO lines
	type poLineData struct {
		ipn       string
		qty       float64
		unitPrice float64
	}
	var poLines []poLineData
	qRows, _ := db.Query(`SELECT rq.rfq_line_id, rq.unit_price, rl.ipn, rl.qty FROM rfq_quotes rq
		JOIN rfq_lines rl ON rq.rfq_line_id=rl.id
		WHERE rq.rfq_id=? AND rq.rfq_vendor_id=?`, id, rfqVendorID)
	if qRows != nil {
		for qRows.Next() {
			var lineID int
			var d poLineData
			qRows.Scan(&lineID, &d.unitPrice, &d.ipn, &d.qty)
			poLines = append(poLines, d)
		}
		qRows.Close()
	}
	for _, d := range poLines {
		db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES (?,?,?,?)`,
			poID, d.ipn, d.qty, d.unitPrice)
	}

	logAudit(db, getUser(r), "award", "rfq", id, "Awarded RFQ to vendor "+body.VendorID+", created "+poID)

	resp := map[string]string{"status": "awarded", "po_id": poID}
	jsonResp(w, resp)
}

func handleCompareRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var rfqID string
	err := db.QueryRow(`SELECT id FROM rfqs WHERE id=?`, id).Scan(&rfqID)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	// Get lines
	lineRows, _ := db.Query(`SELECT id, ipn, description, qty, unit FROM rfq_lines WHERE rfq_id=?`, id)
	var lines []RFQLine
	if lineRows != nil {
		defer lineRows.Close()
		for lineRows.Next() {
			var l RFQLine
			lineRows.Scan(&l.ID, &l.IPN, &l.Description, &l.Qty, &l.Unit)
			l.RFQID = id
			lines = append(lines, l)
		}
	}

	// Get vendors
	vRows, _ := db.Query(`SELECT rv.id, rv.vendor_id, COALESCE(v.name,'') FROM rfq_vendors rv LEFT JOIN vendors v ON rv.vendor_id=v.id WHERE rv.rfq_id=?`, id)
	var vendors []RFQVendor
	if vRows != nil {
		defer vRows.Close()
		for vRows.Next() {
			var v RFQVendor
			vRows.Scan(&v.ID, &v.VendorID, &v.VendorName)
			v.RFQID = id
			vendors = append(vendors, v)
		}
	}

	// Get quotes indexed by line+vendor
	qRows, _ := db.Query(`SELECT rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq, COALESCE(notes,'') FROM rfq_quotes WHERE rfq_id=?`, id)
	type QuoteEntry struct {
		UnitPrice    float64 `json:"unit_price"`
		LeadTimeDays int     `json:"lead_time_days"`
		MOQ          int     `json:"moq"`
		Notes        string  `json:"notes"`
	}
	// map[line_id]map[vendor_id]QuoteEntry
	matrix := make(map[int]map[int]QuoteEntry)
	if qRows != nil {
		defer qRows.Close()
		for qRows.Next() {
			var vendorID, lineID int
			var q QuoteEntry
			qRows.Scan(&vendorID, &lineID, &q.UnitPrice, &q.LeadTimeDays, &q.MOQ, &q.Notes)
			if matrix[lineID] == nil {
				matrix[lineID] = make(map[int]QuoteEntry)
			}
			matrix[lineID][vendorID] = q
		}
	}

	resp := map[string]interface{}{
		"lines":   lines,
		"vendors": vendors,
		"matrix":  matrix,
	}
	if lines == nil {
		resp["lines"] = []RFQLine{}
	}
	if vendors == nil {
		resp["vendors"] = []RFQVendor{}
	}
	jsonResp(w, resp)
}

func handleCreateRFQQuote(w http.ResponseWriter, r *http.Request, rfqID string) {
	var q RFQQuote
	if err := decodeBody(r, &q); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	q.RFQID = rfqID

	res, err := db.Exec(`INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq, notes) VALUES (?,?,?,?,?,?,?)`,
		q.RFQID, q.RFQVendorID, q.RFQLineID, q.UnitPrice, q.LeadTimeDays, q.MOQ, q.Notes)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	qid, _ := res.LastInsertId()
	q.ID = int(qid)

	// Mark vendor as quoted
	now := time.Now().Format(time.RFC3339)
	db.Exec(`UPDATE rfq_vendors SET status='quoted', quoted_at=? WHERE id=?`, now, q.RFQVendorID)

	w.WriteHeader(201)
	jsonResp(w, q)
}

func handleUpdateRFQQuote(w http.ResponseWriter, r *http.Request, rfqID string, quoteID string) {
	var q RFQQuote
	if err := decodeBody(r, &q); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	_, err := db.Exec(`UPDATE rfq_quotes SET unit_price=?, lead_time_days=?, moq=?, notes=? WHERE id=? AND rfq_id=?`,
		q.UnitPrice, q.LeadTimeDays, q.MOQ, q.Notes, quoteID, rfqID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	jsonResp(w, map[string]string{"status": "updated"})
}

func handleCloseRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var status string
	err := db.QueryRow(`SELECT status FROM rfqs WHERE id=?`, id).Scan(&status)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	if status != "awarded" && status != "sent" {
		jsonErr(w, "RFQ must be in awarded or sent status to close", 400)
		return
	}

	now := time.Now().Format(time.RFC3339)
	db.Exec(`UPDATE rfqs SET status='closed', updated_at=? WHERE id=?`, now, id)
	logAudit(db, getUser(r), "close", "rfq", id, "Closed RFQ")
	handleGetRFQ(w, r, id)
}

func handleRFQDashboard(w http.ResponseWriter, r *http.Request) {
	type DashboardRFQ struct {
		ID               string  `json:"id"`
		Title            string  `json:"title"`
		Status           string  `json:"status"`
		DueDate          string  `json:"due_date"`
		VendorCount      int     `json:"vendor_count"`
		ResponseCount    int     `json:"response_count"`
		LineCount        int     `json:"line_count"`
		TotalQuotedValue float64 `json:"total_quoted_value"`
	}
	type Dashboard struct {
		OpenRFQs         int            `json:"open_rfqs"`
		PendingResponses int            `json:"pending_responses"`
		AwardedThisMonth int            `json:"awarded_this_month"`
		RFQs             []DashboardRFQ `json:"rfqs"`
	}

	var dash Dashboard

	// Count open RFQs (draft or sent)
	db.QueryRow(`SELECT COUNT(*) FROM rfqs WHERE status IN ('draft','sent')`).Scan(&dash.OpenRFQs)

	// Count pending vendor responses
	db.QueryRow(`SELECT COUNT(*) FROM rfq_vendors WHERE status='pending'`).Scan(&dash.PendingResponses)

	// Awarded this month
	monthStart := time.Now().Format("2006-01") + "-01"
	db.QueryRow(`SELECT COUNT(*) FROM rfqs WHERE status='awarded' AND updated_at>=?`, monthStart).Scan(&dash.AwardedThisMonth)

	// Active RFQs with stats
	rows, err := db.Query(`SELECT r.id, r.title, r.status, COALESCE(r.due_date,''),
		(SELECT COUNT(*) FROM rfq_vendors WHERE rfq_id=r.id) as vendor_count,
		(SELECT COUNT(*) FROM rfq_vendors WHERE rfq_id=r.id AND status='quoted') as response_count,
		(SELECT COUNT(*) FROM rfq_lines WHERE rfq_id=r.id) as line_count,
		COALESCE((SELECT SUM(rq.unit_price * rl.qty) FROM rfq_quotes rq JOIN rfq_lines rl ON rq.rfq_line_id=rl.id WHERE rq.rfq_id=r.id),0) as total_quoted
		FROM rfqs r WHERE r.status IN ('draft','sent','awarded')
		ORDER BY r.created_at DESC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d DashboardRFQ
			rows.Scan(&d.ID, &d.Title, &d.Status, &d.DueDate, &d.VendorCount, &d.ResponseCount, &d.LineCount, &d.TotalQuotedValue)
			dash.RFQs = append(dash.RFQs, d)
		}
	}
	if dash.RFQs == nil {
		dash.RFQs = []DashboardRFQ{}
	}

	jsonResp(w, dash)
}

func handleRFQEmailBody(w http.ResponseWriter, r *http.Request, id string) {
	var rfq RFQ
	err := db.QueryRow(`SELECT id, title, COALESCE(due_date,''), COALESCE(notes,'') FROM rfqs WHERE id=?`, id).
		Scan(&rfq.ID, &rfq.Title, &rfq.DueDate, &rfq.Notes)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	// Load lines
	lineRows, _ := db.Query(`SELECT ipn, description, qty, unit FROM rfq_lines WHERE rfq_id=?`, id)
	if lineRows != nil {
		defer lineRows.Close()
		for lineRows.Next() {
			var l RFQLine
			lineRows.Scan(&l.IPN, &l.Description, &l.Qty, &l.Unit)
			rfq.Lines = append(rfq.Lines, l)
		}
	}

	// Build email body
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Subject: Request for Quote - %s (%s)\n\n", rfq.Title, rfq.ID))
	sb.WriteString("Dear Vendor,\n\n")
	sb.WriteString(fmt.Sprintf("We are requesting a quote for the following items (RFQ: %s).\n\n", rfq.ID))

	if rfq.DueDate != "" {
		sb.WriteString(fmt.Sprintf("Please respond by: %s\n\n", rfq.DueDate))
	}

	sb.WriteString("Items:\n")
	sb.WriteString(fmt.Sprintf("%-15s %-30s %10s %6s\n", "Part Number", "Description", "Qty", "Unit"))
	sb.WriteString(strings.Repeat("-", 65) + "\n")
	for _, l := range rfq.Lines {
		sb.WriteString(fmt.Sprintf("%-15s %-30s %10.0f %6s\n", l.IPN, l.Description, l.Qty, l.Unit))
	}

	sb.WriteString("\nPlease provide:\n")
	sb.WriteString("- Unit price\n- Lead time\n- Minimum order quantity (MOQ)\n- Any relevant notes or conditions\n\n")

	if rfq.Notes != "" {
		sb.WriteString(fmt.Sprintf("Additional notes: %s\n\n", rfq.Notes))
	}

	sb.WriteString("Thank you for your prompt response.\n")

	jsonResp(w, map[string]string{
		"subject": fmt.Sprintf("Request for Quote - %s (%s)", rfq.Title, rfq.ID),
		"body":    sb.String(),
	})
}

func handleAwardRFQPerLine(w http.ResponseWriter, r *http.Request, id string) {
	var status string
	err := db.QueryRow(`SELECT status FROM rfqs WHERE id=?`, id).Scan(&status)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}

	var body struct {
		Awards []struct {
			LineID   int    `json:"line_id"`
			VendorID string `json:"vendor_id"`
		} `json:"awards"`
	}
	if err := decodeBody(r, &body); err != nil || len(body.Awards) == 0 {
		jsonErr(w, "awards array required", 400)
		return
	}

	now := time.Now().Format(time.RFC3339)

	// Group awards by vendor to create POs
	vendorLines := make(map[string][]int) // vendor_id -> []line_id
	for _, a := range body.Awards {
		vendorLines[a.VendorID] = append(vendorLines[a.VendorID], a.LineID)
	}

	var poIDs []string
	for vendorID, lineIDs := range vendorLines {
		poID := nextID("PO", "purchase_orders", 4)
		db.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, notes, created_at) VALUES (?,?,?,?,?)`,
			poID, vendorID, "draft", "Auto-created from "+id+" (per-line award)", now)

		// Find rfq_vendor_id for this vendor
		var rfqVendorID int
		db.QueryRow(`SELECT id FROM rfq_vendors WHERE rfq_id=? AND vendor_id=?`, id, vendorID).Scan(&rfqVendorID)

		for _, lineID := range lineIDs {
			var ipn string
			var qty float64
			var unitPrice float64
			db.QueryRow(`SELECT rl.ipn, rl.qty, COALESCE(rq.unit_price,0) FROM rfq_lines rl
				LEFT JOIN rfq_quotes rq ON rq.rfq_line_id=rl.id AND rq.rfq_vendor_id=?
				WHERE rl.id=?`, rfqVendorID, lineID).Scan(&ipn, &qty, &unitPrice)
			db.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES (?,?,?,?)`,
				poID, ipn, qty, unitPrice)
		}
		poIDs = append(poIDs, poID)
	}

	db.Exec(`UPDATE rfqs SET status='awarded', updated_at=? WHERE id=?`, now, id)
	logAudit(db, getUser(r), "award_per_line", "rfq", id, fmt.Sprintf("Per-line award, created POs: %v", poIDs))

	jsonResp(w, map[string]interface{}{
		"status": "awarded",
		"po_ids": poIDs,
	})
}

// getUser extracts the username from the request context/session
func getUser(r *http.Request) string {
	return getUsername(r)
}
