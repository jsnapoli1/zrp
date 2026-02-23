package sales

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"zrp/internal/audit"
	"zrp/internal/models"
	"zrp/internal/response"
)

// DefaultTaxRate is the default tax rate (10%).
const DefaultTaxRate = 0.10

// ListInvoices handles GET /api/invoices.
func (h *Handler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	customer := r.URL.Query().Get("customer")
	fromDate := r.URL.Query().Get("from_date")
	toDate := r.URL.Query().Get("to_date")

	query := `SELECT id, invoice_number, sales_order_id, customer, issue_date, due_date, status,
		total, tax, notes, created_at, paid_at FROM invoices`
	var conditions []string
	var args []interface{}

	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	if customer != "" {
		conditions = append(conditions, "customer LIKE ?")
		args = append(args, "%"+customer+"%")
	}
	if fromDate != "" {
		conditions = append(conditions, "issue_date >= ?")
		args = append(args, fromDate)
	}
	if toDate != "" {
		conditions = append(conditions, "issue_date <= ?")
		args = append(args, toDate)
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, c := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += c
		}
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		var paidAt sql.NullString
		err := rows.Scan(&inv.ID, &inv.InvoiceNumber, &inv.SalesOrderID, &inv.Customer,
			&inv.IssueDate, &inv.DueDate, &inv.Status, &inv.Total, &inv.Tax,
			&inv.Notes, &inv.CreatedAt, &paidAt)
		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}
		if paidAt.Valid {
			inv.PaidAt = &paidAt.String
		}
		invoices = append(invoices, inv)
	}

	if invoices == nil {
		invoices = []models.Invoice{}
	}
	response.JSON(w, invoices)
}

// GetInvoice handles GET /api/invoices/:id.
func (h *Handler) GetInvoice(w http.ResponseWriter, r *http.Request, id string) {
	var inv models.Invoice
	var paidAt sql.NullString

	err := h.DB.QueryRow(`SELECT id, invoice_number, sales_order_id, customer, issue_date, due_date,
		status, total, tax, notes, created_at, paid_at FROM invoices WHERE id = ?`, id).
		Scan(&inv.ID, &inv.InvoiceNumber, &inv.SalesOrderID, &inv.Customer,
			&inv.IssueDate, &inv.DueDate, &inv.Status, &inv.Total, &inv.Tax,
			&inv.Notes, &inv.CreatedAt, &paidAt)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Err(w, "invoice not found", 404)
		} else {
			response.Err(w, err.Error(), 500)
		}
		return
	}

	if paidAt.Valid {
		inv.PaidAt = &paidAt.String
	}

	// Load invoice lines
	inv.Lines = h.getInvoiceLines(id)

	response.JSON(w, inv)
}

func (h *Handler) getInvoiceLines(invoiceID string) []models.InvoiceLine {
	rows, err := h.DB.Query(`SELECT id, invoice_id, ipn, description, quantity, unit_price, total
		FROM invoice_lines WHERE invoice_id = ? ORDER BY id`, invoiceID)
	if err != nil {
		return []models.InvoiceLine{}
	}
	defer rows.Close()

	var lines []models.InvoiceLine
	for rows.Next() {
		var line models.InvoiceLine
		rows.Scan(&line.ID, &line.InvoiceID, &line.IPN, &line.Description,
			&line.Quantity, &line.UnitPrice, &line.Total)
		lines = append(lines, line)
	}

	if lines == nil {
		lines = []models.InvoiceLine{}
	}
	return lines
}

// CreateInvoice handles POST /api/invoices.
func (h *Handler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	var inv models.Invoice
	if err := response.DecodeBody(r, &inv); err != nil {
		response.Err(w, "invalid JSON", 400)
		return
	}

	// Validate required fields
	if inv.SalesOrderID == "" || inv.Customer == "" {
		response.Err(w, "sales_order_id and customer are required", 400)
		return
	}

	// Generate ID and invoice number
	inv.ID = h.NextID("INV", "invoices", 6)
	inv.InvoiceNumber = h.GenerateInvoiceNum()

	// Set defaults
	if inv.Status == "" {
		inv.Status = "draft"
	}
	if inv.IssueDate == "" {
		inv.IssueDate = time.Now().Format("2006-01-02")
	}
	if inv.DueDate == "" {
		inv.DueDate = time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	}

	// Calculate totals if lines provided
	if len(inv.Lines) > 0 {
		subtotal := 0.0
		for i := range inv.Lines {
			inv.Lines[i].Total = float64(inv.Lines[i].Quantity) * inv.Lines[i].UnitPrice
			subtotal += inv.Lines[i].Total
		}
		inv.Tax = subtotal * DefaultTaxRate
		inv.Total = subtotal + inv.Tax
	}

	// Insert invoice
	_, err := h.DB.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer,
		issue_date, due_date, status, total, tax, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.InvoiceNumber, inv.SalesOrderID, inv.Customer, inv.IssueDate,
		inv.DueDate, inv.Status, inv.Total, inv.Tax, inv.Notes, time.Now().Format(time.RFC3339))
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Insert lines
	for _, line := range inv.Lines {
		_, err := h.DB.Exec(`INSERT INTO invoice_lines (invoice_id, ipn, description, quantity, unit_price, total)
			VALUES (?, ?, ?, ?, ?, ?)`,
			inv.ID, line.IPN, line.Description, line.Quantity, line.UnitPrice, line.Total)
		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}
	}

	// Return created invoice with lines
	inv.Lines = h.getInvoiceLines(inv.ID)
	inv.CreatedAt = time.Now().Format(time.RFC3339)

	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "create", "invoices", inv.ID, fmt.Sprintf("Created invoice %s for customer %s", inv.InvoiceNumber, inv.Customer))
	response.JSON(w, inv)
}

// UpdateInvoice handles PUT /api/invoices/:id.
func (h *Handler) UpdateInvoice(w http.ResponseWriter, r *http.Request, id string) {
	var inv models.Invoice
	if err := response.DecodeBody(r, &inv); err != nil {
		response.Err(w, "invalid JSON", 400)
		return
	}

	// Check if invoice exists and is editable
	var currentStatus string
	err := h.DB.QueryRow("SELECT status FROM invoices WHERE id = ?", id).Scan(&currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Err(w, "invoice not found", 404)
		} else {
			response.Err(w, err.Error(), 500)
		}
		return
	}

	if currentStatus == "paid" || currentStatus == "cancelled" {
		response.Err(w, "cannot edit paid or cancelled invoices", 400)
		return
	}

	// Calculate totals if lines provided
	if len(inv.Lines) > 0 {
		subtotal := 0.0
		for i := range inv.Lines {
			inv.Lines[i].Total = float64(inv.Lines[i].Quantity) * inv.Lines[i].UnitPrice
			subtotal += inv.Lines[i].Total
		}
		inv.Tax = subtotal * DefaultTaxRate
		inv.Total = subtotal + inv.Tax
	}

	// Update invoice
	_, err = h.DB.Exec(`UPDATE invoices SET customer = ?, issue_date = ?, due_date = ?,
		status = ?, total = ?, tax = ?, notes = ? WHERE id = ?`,
		inv.Customer, inv.IssueDate, inv.DueDate, inv.Status, inv.Total, inv.Tax, inv.Notes, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Update lines - delete existing and insert new
	_, err = h.DB.Exec("DELETE FROM invoice_lines WHERE invoice_id = ?", id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	for _, line := range inv.Lines {
		_, err := h.DB.Exec(`INSERT INTO invoice_lines (invoice_id, ipn, description, quantity, unit_price, total)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, line.IPN, line.Description, line.Quantity, line.UnitPrice, line.Total)
		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}
	}

	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "update", "invoices", id, fmt.Sprintf("Updated invoice %s", inv.InvoiceNumber))

	// Return updated invoice
	h.GetInvoice(w, r, id)
}

// CreateInvoiceFromSalesOrder handles POST /api/sales-orders/:id/create-invoice.
func (h *Handler) CreateInvoiceFromSalesOrder(w http.ResponseWriter, r *http.Request, salesOrderID string) {
	// Verify sales order exists and is shipped
	var order models.SalesOrder
	err := h.DB.QueryRow(`SELECT id, quote_id, customer, status, notes, created_by, created_at, updated_at
		FROM sales_orders WHERE id = ?`, salesOrderID).
		Scan(&order.ID, &order.QuoteID, &order.Customer, &order.Status, &order.Notes,
			&order.CreatedBy, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Err(w, "sales order not found", 404)
		} else {
			response.Err(w, err.Error(), 500)
		}
		return
	}

	if order.Status != "shipped" {
		response.Err(w, "sales order must be shipped before creating invoice", 400)
		return
	}

	// Check if invoice already exists
	var existingInvoiceID string
	err = h.DB.QueryRow("SELECT id FROM invoices WHERE sales_order_id = ?", salesOrderID).Scan(&existingInvoiceID)
	if err == nil {
		response.Err(w, "sales order already has an invoice", 400)
		return
	} else if err != sql.ErrNoRows {
		response.Err(w, err.Error(), 500)
		return
	}

	// Get sales order lines
	lines := h.getSalesOrderLines(salesOrderID)

	// Create invoice
	inv := models.Invoice{
		ID:            h.NextID("INV", "invoices", 6),
		InvoiceNumber: h.GenerateInvoiceNum(),
		SalesOrderID:  salesOrderID,
		Customer:      order.Customer,
		IssueDate:     time.Now().Format("2006-01-02"),
		DueDate:       time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		Status:        "draft",
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	// Convert sales order lines to invoice lines and calculate totals
	subtotal := 0.0
	for _, soLine := range lines {
		invLine := models.InvoiceLine{
			InvoiceID:   inv.ID,
			IPN:         soLine.IPN,
			Description: soLine.Description,
			Quantity:    soLine.QtyShipped,
			UnitPrice:   soLine.UnitPrice,
			Total:       float64(soLine.QtyShipped) * soLine.UnitPrice,
		}
		inv.Lines = append(inv.Lines, invLine)
		subtotal += invLine.Total
	}

	inv.Tax = subtotal * DefaultTaxRate
	inv.Total = subtotal + inv.Tax

	// Insert invoice
	_, err = h.DB.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer,
		issue_date, due_date, status, total, tax, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.InvoiceNumber, inv.SalesOrderID, inv.Customer, inv.IssueDate,
		inv.DueDate, inv.Status, inv.Total, inv.Tax, inv.Notes, inv.CreatedAt)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Insert lines
	for _, line := range inv.Lines {
		_, err := h.DB.Exec(`INSERT INTO invoice_lines (invoice_id, ipn, description, quantity, unit_price, total)
			VALUES (?, ?, ?, ?, ?, ?)`,
			inv.ID, line.IPN, line.Description, line.Quantity, line.UnitPrice, line.Total)
		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}
	}

	// Update sales order status to invoiced
	_, err = h.DB.Exec("UPDATE sales_orders SET status = ? WHERE id = ?", "invoiced", salesOrderID)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "create", "invoices", inv.ID, fmt.Sprintf("Created invoice %s from sales order %s", inv.InvoiceNumber, salesOrderID))
	response.JSON(w, inv)
}

// SendInvoice handles POST /api/invoices/:id/send.
func (h *Handler) SendInvoice(w http.ResponseWriter, r *http.Request, id string) {
	// Check if invoice exists and is in draft status
	var inv models.Invoice
	err := h.DB.QueryRow("SELECT status, customer, invoice_number FROM invoices WHERE id = ?", id).
		Scan(&inv.Status, &inv.Customer, &inv.InvoiceNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Err(w, "invoice not found", 404)
		} else {
			response.Err(w, err.Error(), 500)
		}
		return
	}

	if inv.Status != "draft" {
		response.Err(w, "only draft invoices can be sent", 400)
		return
	}

	// Update status to sent
	_, err = h.DB.Exec("UPDATE invoices SET status = ? WHERE id = ?", "sent", id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "send", "invoices", id, fmt.Sprintf("Sent invoice %s to customer %s", inv.InvoiceNumber, inv.Customer))
	response.JSON(w, map[string]string{"status": "sent", "message": "Invoice sent successfully"})
}

// MarkInvoicePaid handles POST /api/invoices/:id/pay.
func (h *Handler) MarkInvoicePaid(w http.ResponseWriter, r *http.Request, id string) {
	// Check if invoice exists and can be marked as paid
	var inv models.Invoice
	err := h.DB.QueryRow("SELECT status, invoice_number FROM invoices WHERE id = ?", id).
		Scan(&inv.Status, &inv.InvoiceNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Err(w, "invoice not found", 404)
		} else {
			response.Err(w, err.Error(), 500)
		}
		return
	}

	if inv.Status == "cancelled" {
		response.Err(w, "cannot mark cancelled invoice as paid", 400)
		return
	}

	// Update status to paid and set paid_at
	now := time.Now().Format(time.RFC3339)
	_, err = h.DB.Exec("UPDATE invoices SET status = ?, paid_at = ? WHERE id = ?", "paid", now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "pay", "invoices", id, fmt.Sprintf("Marked invoice %s as paid", inv.InvoiceNumber))
	response.JSON(w, map[string]string{"status": "paid", "message": "Invoice marked as paid"})
}

// GenerateInvoicePDF handles GET /api/invoices/:id/pdf.
func (h *Handler) GenerateInvoicePDF(w http.ResponseWriter, r *http.Request, id string) {
	// Get invoice with lines
	var inv models.Invoice
	var paidAt sql.NullString

	err := h.DB.QueryRow(`SELECT id, invoice_number, sales_order_id, customer, issue_date, due_date,
		status, total, tax, notes, created_at, paid_at FROM invoices WHERE id = ?`, id).
		Scan(&inv.ID, &inv.InvoiceNumber, &inv.SalesOrderID, &inv.Customer,
			&inv.IssueDate, &inv.DueDate, &inv.Status, &inv.Total, &inv.Tax,
			&inv.Notes, &inv.CreatedAt, &paidAt)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Err(w, "invoice not found", 404)
		} else {
			response.Err(w, err.Error(), 500)
		}
		return
	}

	if paidAt.Valid {
		inv.PaidAt = &paidAt.String
	}
	inv.Lines = h.getInvoiceLines(id)

	// Generate PDF content
	pdfContent := GenerateInvoicePDFContent(inv)

	// Set headers
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"invoice_%s.pdf\"", inv.InvoiceNumber))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfContent)))

	w.Write(pdfContent)
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "pdf", "invoices", id, fmt.Sprintf("Generated PDF for invoice %s", inv.InvoiceNumber))
}

// UpdateOverdueInvoices updates overdue invoices - should be called periodically.
func (h *Handler) UpdateOverdueInvoices() {
	today := time.Now().Format("2006-01-02")
	_, err := h.DB.Exec(`UPDATE invoices SET status = 'overdue'
		WHERE status = 'sent' AND due_date < ?`, today)
	if err != nil {
		fmt.Printf("Error updating overdue invoices: %v\n", err)
	}
}

// GenerateInvoicePDFContent generates a basic PDF for an invoice.
func GenerateInvoicePDFContent(inv models.Invoice) []byte {
	content := fmt.Sprintf(`%%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj

2 0 obj
<<
/Type /Pages
/Kids [3 0 R]
/Count 1
>>
endobj

3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
/Contents 4 0 R
/Resources <<
/Font <<
/F1 <<
/Type /Font
/Subtype /Type1
/BaseFont /Helvetica
>>
>>
>>
>>
endobj

4 0 obj
<<
/Length 1000
>>
stream
BT
/F1 12 Tf
50 750 Td
(INVOICE) Tj
0 -20 Td
(Invoice Number: %s) Tj
0 -20 Td
(Customer: %s) Tj
0 -20 Td
(Issue Date: %s) Tj
0 -20 Td
(Due Date: %s) Tj
0 -20 Td
(Status: %s) Tj
`, inv.InvoiceNumber, inv.Customer, inv.IssueDate, inv.DueDate, inv.Status)

	// Add lines
	yPos := 650
	for _, line := range inv.Lines {
		content += fmt.Sprintf(`0 %d Td
(%s - Qty: %d @ $%.2f = $%.2f) Tj
`, yPos-650, line.Description, line.Quantity, line.UnitPrice, line.Total)
		yPos -= 20
	}

	content += fmt.Sprintf(`
0 %d Td
(Subtotal: $%.2f) Tj
0 -20 Td
(Tax: $%.2f) Tj
0 -20 Td
(Total: $%.2f) Tj
`, yPos-650-60, inv.Total-inv.Tax, inv.Tax, inv.Total)

	// Add PAID watermark if paid
	if inv.Status == "paid" {
		content += `
/F1 24 Tf
200 400 Td
(PAID) Tj
`
	}

	content += `
ET
endstream
endobj

xref
0 5
0000000000 65535 f
0000000010 00000 n
0000000079 00000 n
0000000136 00000 n
0000000395 00000 n
trailer
<<
/Size 5
/Root 1 0 R
>>
startxref
1450
%%EOF`

	return []byte(content)
}
