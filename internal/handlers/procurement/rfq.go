package procurement

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"zrp/internal/models"
	"zrp/internal/response"
)

// ListRFQs returns all RFQs.
func (h *Handler) ListRFQs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, title, status, created_by, created_at, updated_at, COALESCE(due_date,''), COALESCE(notes,'') FROM rfqs ORDER BY created_at DESC`)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.RFQ
	for rows.Next() {
		var rfq models.RFQ
		rows.Scan(&rfq.ID, &rfq.Title, &rfq.Status, &rfq.CreatedBy, &rfq.CreatedAt, &rfq.UpdatedAt, &rfq.DueDate, &rfq.Notes)
		items = append(items, rfq)
	}
	if items == nil {
		items = []models.RFQ{}
	}
	response.JSON(w, items)
}

// GetRFQ returns a single RFQ with lines, vendors, and quotes.
func (h *Handler) GetRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var rfq models.RFQ
	err := h.DB.QueryRow(`SELECT id, title, status, created_by, created_at, updated_at, COALESCE(due_date,''), COALESCE(notes,'') FROM rfqs WHERE id=?`, id).
		Scan(&rfq.ID, &rfq.Title, &rfq.Status, &rfq.CreatedBy, &rfq.CreatedAt, &rfq.UpdatedAt, &rfq.DueDate, &rfq.Notes)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	// Load lines
	lineRows, _ := h.DB.Query(`SELECT id, rfq_id, ipn, description, qty, unit FROM rfq_lines WHERE rfq_id=?`, id)
	if lineRows != nil {
		defer lineRows.Close()
		for lineRows.Next() {
			var l models.RFQLine
			lineRows.Scan(&l.ID, &l.RFQID, &l.IPN, &l.Description, &l.Qty, &l.Unit)
			rfq.Lines = append(rfq.Lines, l)
		}
	}
	if rfq.Lines == nil {
		rfq.Lines = []models.RFQLine{}
	}

	// Load vendors
	vRows, _ := h.DB.Query(`SELECT rv.id, rv.rfq_id, rv.vendor_id, rv.status, COALESCE(rv.quoted_at,''), COALESCE(rv.notes,''), COALESCE(v.name,'') FROM rfq_vendors rv LEFT JOIN vendors v ON rv.vendor_id=v.id WHERE rv.rfq_id=?`, id)
	if vRows != nil {
		defer vRows.Close()
		for vRows.Next() {
			var v models.RFQVendor
			vRows.Scan(&v.ID, &v.RFQID, &v.VendorID, &v.Status, &v.QuotedAt, &v.Notes, &v.VendorName)
			rfq.Vendors = append(rfq.Vendors, v)
		}
	}
	if rfq.Vendors == nil {
		rfq.Vendors = []models.RFQVendor{}
	}

	// Load quotes
	qRows, _ := h.DB.Query(`SELECT id, rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq, COALESCE(notes,'') FROM rfq_quotes WHERE rfq_id=?`, id)
	if qRows != nil {
		defer qRows.Close()
		for qRows.Next() {
			var q models.RFQQuote
			qRows.Scan(&q.ID, &q.RFQID, &q.RFQVendorID, &q.RFQLineID, &q.UnitPrice, &q.LeadTimeDays, &q.MOQ, &q.Notes)
			rfq.Quotes = append(rfq.Quotes, q)
		}
	}
	if rfq.Quotes == nil {
		rfq.Quotes = []models.RFQQuote{}
	}

	response.JSON(w, rfq)
}

// CreateRFQ creates a new RFQ.
func (h *Handler) CreateRFQ(w http.ResponseWriter, r *http.Request) {
	var rfq models.RFQ
	if err := response.DecodeBody(r, &rfq); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	if rfq.Title == "" {
		response.Err(w, "title required", 400)
		return
	}
	rfq.ID = h.NextIDFunc("RFQ", "rfqs", 4)
	rfq.Status = "draft"
	rfq.CreatedBy = h.GetUsername(r)
	now := time.Now().Format(time.RFC3339)
	rfq.CreatedAt = now
	rfq.UpdatedAt = now

	_, err := h.DB.Exec(`INSERT INTO rfqs (id, title, status, created_by, created_at, updated_at, due_date, notes) VALUES (?,?,?,?,?,?,?,?)`,
		rfq.ID, rfq.Title, rfq.Status, rfq.CreatedBy, rfq.CreatedAt, rfq.UpdatedAt, rfq.DueDate, rfq.Notes)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Insert lines
	for i, l := range rfq.Lines {
		res, _ := h.DB.Exec(`INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?,?,?,?,?)`,
			rfq.ID, l.IPN, l.Description, l.Qty, l.Unit)
		if res != nil {
			lid, _ := res.LastInsertId()
			rfq.Lines[i].ID = int(lid)
			rfq.Lines[i].RFQID = rfq.ID
		}
	}

	// Insert vendors
	for i, v := range rfq.Vendors {
		res, _ := h.DB.Exec(`INSERT INTO rfq_vendors (rfq_id, vendor_id, status, notes) VALUES (?,?,?,?)`,
			rfq.ID, v.VendorID, "pending", v.Notes)
		if res != nil {
			vid, _ := res.LastInsertId()
			rfq.Vendors[i].ID = int(vid)
			rfq.Vendors[i].RFQID = rfq.ID
			rfq.Vendors[i].Status = "pending"
		}
	}

	h.LogAudit(rfq.CreatedBy, "create", "rfq", rfq.ID, "Created RFQ: "+rfq.Title)
	w.WriteHeader(201)
	response.JSON(w, rfq)
}

// UpdateRFQ updates an existing RFQ.
func (h *Handler) UpdateRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var existing models.RFQ
	err := h.DB.QueryRow(`SELECT id, status FROM rfqs WHERE id=?`, id).Scan(&existing.ID, &existing.Status)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	var rfq models.RFQ
	if err := response.DecodeBody(r, &rfq); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	now := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec(`UPDATE rfqs SET title=?, due_date=?, notes=?, updated_at=? WHERE id=?`,
		rfq.Title, rfq.DueDate, rfq.Notes, now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Replace lines
	h.DB.Exec(`DELETE FROM rfq_lines WHERE rfq_id=?`, id)
	for _, l := range rfq.Lines {
		h.DB.Exec(`INSERT INTO rfq_lines (rfq_id, ipn, description, qty, unit) VALUES (?,?,?,?,?)`,
			id, l.IPN, l.Description, l.Qty, l.Unit)
	}

	// Replace vendors
	h.DB.Exec(`DELETE FROM rfq_vendors WHERE rfq_id=?`, id)
	for _, v := range rfq.Vendors {
		h.DB.Exec(`INSERT INTO rfq_vendors (rfq_id, vendor_id, status, notes) VALUES (?,?,?,?)`,
			id, v.VendorID, "pending", v.Notes)
	}

	h.LogAudit(h.GetUsername(r), "update", "rfq", id, "Updated RFQ")
	h.GetRFQ(w, r, id)
}

// DeleteRFQ deletes an RFQ and its associated records.
func (h *Handler) DeleteRFQ(w http.ResponseWriter, r *http.Request, id string) {
	res, err := h.DB.Exec(`DELETE FROM rfqs WHERE id=?`, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		response.Err(w, "not found", 404)
		return
	}
	h.DB.Exec(`DELETE FROM rfq_lines WHERE rfq_id=?`, id)
	h.DB.Exec(`DELETE FROM rfq_vendors WHERE rfq_id=?`, id)
	h.DB.Exec(`DELETE FROM rfq_quotes WHERE rfq_id=?`, id)
	h.LogAudit(h.GetUsername(r), "delete", "rfq", id, "Deleted RFQ")
	response.JSON(w, map[string]string{"status": "deleted"})
}

// SendRFQ transitions an RFQ from draft to sent status.
func (h *Handler) SendRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var status string
	err := h.DB.QueryRow(`SELECT status FROM rfqs WHERE id=?`, id).Scan(&status)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	if status != "draft" {
		response.Err(w, "RFQ must be in draft status to send", 400)
		return
	}

	now := time.Now().Format(time.RFC3339)
	h.DB.Exec(`UPDATE rfqs SET status='sent', updated_at=? WHERE id=?`, now, id)
	h.DB.Exec(`UPDATE rfq_vendors SET status='pending' WHERE rfq_id=?`, id)

	h.LogAudit(h.GetUsername(r), "send", "rfq", id, "Sent RFQ to vendors")
	h.GetRFQ(w, r, id)
}

// AwardRFQ awards an RFQ to a vendor and auto-creates a PO.
func (h *Handler) AwardRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var status string
	err := h.DB.QueryRow(`SELECT status FROM rfqs WHERE id=?`, id).Scan(&status)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	var body struct {
		VendorID string `json:"vendor_id"`
	}
	if err := response.DecodeBody(r, &body); err != nil || body.VendorID == "" {
		response.Err(w, "vendor_id required", 400)
		return
	}

	// Find the rfq_vendor entry
	var rfqVendorID int
	err = h.DB.QueryRow(`SELECT id FROM rfq_vendors WHERE rfq_id=? AND vendor_id=?`, id, body.VendorID).Scan(&rfqVendorID)
	if err != nil {
		response.Err(w, "vendor not in this RFQ", 400)
		return
	}

	now := time.Now().Format(time.RFC3339)
	h.DB.Exec(`UPDATE rfqs SET status='awarded', updated_at=? WHERE id=?`, now, id)

	// Auto-create PO from winning quotes
	poID := h.NextIDFunc("PO", "purchase_orders", 4)
	h.DB.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, notes, created_at) VALUES (?,?,?,?,?)`,
		poID, body.VendorID, "draft", "Auto-created from "+id, now)

	// Get quotes for winning vendor and create PO lines
	type poLineData struct {
		ipn       string
		qty       float64
		unitPrice float64
	}
	var poLines []poLineData
	qRows, _ := h.DB.Query(`SELECT rq.rfq_line_id, rq.unit_price, rl.ipn, rl.qty FROM rfq_quotes rq
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
		h.DB.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES (?,?,?,?)`,
			poID, d.ipn, d.qty, d.unitPrice)
	}

	username := h.GetUsername(r)
	h.LogAudit(username, "award", "rfq", id, "Awarded RFQ to vendor "+body.VendorID+", created "+poID)

	resp := map[string]string{"status": "awarded", "po_id": poID}
	response.JSON(w, resp)
}

// CompareRFQ returns a comparison matrix of vendor quotes for an RFQ.
func (h *Handler) CompareRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var rfqID string
	err := h.DB.QueryRow(`SELECT id FROM rfqs WHERE id=?`, id).Scan(&rfqID)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	// Get lines
	lineRows, _ := h.DB.Query(`SELECT id, ipn, description, qty, unit FROM rfq_lines WHERE rfq_id=?`, id)
	var lines []models.RFQLine
	if lineRows != nil {
		defer lineRows.Close()
		for lineRows.Next() {
			var l models.RFQLine
			lineRows.Scan(&l.ID, &l.IPN, &l.Description, &l.Qty, &l.Unit)
			l.RFQID = id
			lines = append(lines, l)
		}
	}

	// Get vendors
	vRows, _ := h.DB.Query(`SELECT rv.id, rv.vendor_id, COALESCE(v.name,'') FROM rfq_vendors rv LEFT JOIN vendors v ON rv.vendor_id=v.id WHERE rv.rfq_id=?`, id)
	var vendors []models.RFQVendor
	if vRows != nil {
		defer vRows.Close()
		for vRows.Next() {
			var v models.RFQVendor
			vRows.Scan(&v.ID, &v.VendorID, &v.VendorName)
			v.RFQID = id
			vendors = append(vendors, v)
		}
	}

	// Get quotes indexed by line+vendor
	qRows, _ := h.DB.Query(`SELECT rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq, COALESCE(notes,'') FROM rfq_quotes WHERE rfq_id=?`, id)
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
		resp["lines"] = []models.RFQLine{}
	}
	if vendors == nil {
		resp["vendors"] = []models.RFQVendor{}
	}
	response.JSON(w, resp)
}

// CreateRFQQuote adds a quote to an RFQ.
func (h *Handler) CreateRFQQuote(w http.ResponseWriter, r *http.Request, rfqID string) {
	var q models.RFQQuote
	if err := response.DecodeBody(r, &q); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	q.RFQID = rfqID

	res, err := h.DB.Exec(`INSERT INTO rfq_quotes (rfq_id, rfq_vendor_id, rfq_line_id, unit_price, lead_time_days, moq, notes) VALUES (?,?,?,?,?,?,?)`,
		q.RFQID, q.RFQVendorID, q.RFQLineID, q.UnitPrice, q.LeadTimeDays, q.MOQ, q.Notes)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	qid, _ := res.LastInsertId()
	q.ID = int(qid)

	// Mark vendor as quoted
	now := time.Now().Format(time.RFC3339)
	h.DB.Exec(`UPDATE rfq_vendors SET status='quoted', quoted_at=? WHERE id=?`, now, q.RFQVendorID)

	w.WriteHeader(201)
	response.JSON(w, q)
}

// UpdateRFQQuote updates an existing quote on an RFQ.
func (h *Handler) UpdateRFQQuote(w http.ResponseWriter, r *http.Request, rfqID string, quoteID string) {
	var q models.RFQQuote
	if err := response.DecodeBody(r, &q); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	_, err := h.DB.Exec(`UPDATE rfq_quotes SET unit_price=?, lead_time_days=?, moq=?, notes=? WHERE id=? AND rfq_id=?`,
		q.UnitPrice, q.LeadTimeDays, q.MOQ, q.Notes, quoteID, rfqID)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	response.JSON(w, map[string]string{"status": "updated"})
}

// CloseRFQ transitions an RFQ to closed status.
func (h *Handler) CloseRFQ(w http.ResponseWriter, r *http.Request, id string) {
	var status string
	err := h.DB.QueryRow(`SELECT status FROM rfqs WHERE id=?`, id).Scan(&status)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	if status != "awarded" && status != "sent" {
		response.Err(w, "RFQ must be in awarded or sent status to close", 400)
		return
	}

	now := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec(`UPDATE rfqs SET status='closed', updated_at=? WHERE id=?`, now, id)
	if err != nil {
		response.Err(w, "failed to update RFQ: "+err.Error(), 500)
		return
	}
	h.LogAudit(h.GetUsername(r), "close", "rfq", id, "Closed RFQ")
	h.GetRFQ(w, r, id)
}

// RFQDashboard returns RFQ dashboard statistics.
func (h *Handler) RFQDashboard(w http.ResponseWriter, r *http.Request) {
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
	h.DB.QueryRow(`SELECT COUNT(*) FROM rfqs WHERE status IN ('draft','sent')`).Scan(&dash.OpenRFQs)

	// Count pending vendor responses
	h.DB.QueryRow(`SELECT COUNT(*) FROM rfq_vendors WHERE status='pending'`).Scan(&dash.PendingResponses)

	// Awarded this month
	monthStart := time.Now().Format("2006-01") + "-01"
	h.DB.QueryRow(`SELECT COUNT(*) FROM rfqs WHERE status='awarded' AND updated_at>=?`, monthStart).Scan(&dash.AwardedThisMonth)

	// Active RFQs with stats
	rows, err := h.DB.Query(`SELECT r.id, r.title, r.status, COALESCE(r.due_date,''),
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

	response.JSON(w, dash)
}

// RFQEmailBody generates an email body for an RFQ.
func (h *Handler) RFQEmailBody(w http.ResponseWriter, r *http.Request, id string) {
	var rfq models.RFQ
	err := h.DB.QueryRow(`SELECT id, title, COALESCE(due_date,''), COALESCE(notes,'') FROM rfqs WHERE id=?`, id).
		Scan(&rfq.ID, &rfq.Title, &rfq.DueDate, &rfq.Notes)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	// Load lines
	lineRows, _ := h.DB.Query(`SELECT ipn, description, qty, unit FROM rfq_lines WHERE rfq_id=?`, id)
	if lineRows != nil {
		defer lineRows.Close()
		for lineRows.Next() {
			var l models.RFQLine
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

	response.JSON(w, map[string]string{
		"subject": fmt.Sprintf("Request for Quote - %s (%s)", rfq.Title, rfq.ID),
		"body":    sb.String(),
	})
}

// AwardRFQPerLine awards an RFQ on a per-line basis to multiple vendors.
func (h *Handler) AwardRFQPerLine(w http.ResponseWriter, r *http.Request, id string) {
	var status string
	err := h.DB.QueryRow(`SELECT status FROM rfqs WHERE id=?`, id).Scan(&status)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	var body struct {
		Awards []struct {
			LineID   int    `json:"line_id"`
			VendorID string `json:"vendor_id"`
		} `json:"awards"`
	}
	if err := response.DecodeBody(r, &body); err != nil || len(body.Awards) == 0 {
		response.Err(w, "awards array required", 400)
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
		poID := h.NextIDFunc("PO", "purchase_orders", 4)
		h.DB.Exec(`INSERT INTO purchase_orders (id, vendor_id, status, notes, created_at) VALUES (?,?,?,?,?)`,
			poID, vendorID, "draft", "Auto-created from "+id+" (per-line award)", now)

		// Find rfq_vendor_id for this vendor
		var rfqVendorID int
		h.DB.QueryRow(`SELECT id FROM rfq_vendors WHERE rfq_id=? AND vendor_id=?`, id, vendorID).Scan(&rfqVendorID)

		for _, lineID := range lineIDs {
			var ipn string
			var qty float64
			var unitPrice float64
			h.DB.QueryRow(`SELECT rl.ipn, rl.qty, COALESCE(rq.unit_price,0) FROM rfq_lines rl
				LEFT JOIN rfq_quotes rq ON rq.rfq_line_id=rl.id AND rq.rfq_vendor_id=?
				WHERE rl.id=?`, rfqVendorID, lineID).Scan(&ipn, &qty, &unitPrice)
			h.DB.Exec(`INSERT INTO po_lines (po_id, ipn, qty_ordered, unit_price) VALUES (?,?,?,?)`,
				poID, ipn, qty, unitPrice)
		}
		poIDs = append(poIDs, poID)
	}

	h.DB.Exec(`UPDATE rfqs SET status='awarded', updated_at=? WHERE id=?`, now, id)
	username := h.GetUsername(r)
	h.LogAudit(username, "award_per_line", "rfq", id, fmt.Sprintf("Per-line award, created POs: %v", poIDs))

	response.JSON(w, map[string]interface{}{
		"status": "awarded",
		"po_ids": poIDs,
	})
}
