package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

// DEFAULT_TAX_RATE is kept for backward compatibility with tests.
const DEFAULT_TAX_RATE = 0.10

func handleListInvoices(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().ListInvoices(w, r)
}

func handleGetInvoice(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().GetInvoice(w, r, id)
}

func getInvoiceLines(invoiceID string) []InvoiceLine {
	// Keep backward-compatible wrapper for other root-level code that calls this.
	rows, err := db.Query(`SELECT id, invoice_id, ipn, description, quantity, unit_price, total
		FROM invoice_lines WHERE invoice_id = ? ORDER BY id`, invoiceID)
	if err != nil {
		return []InvoiceLine{}
	}
	defer rows.Close()

	var lines []InvoiceLine
	for rows.Next() {
		var line InvoiceLine
		rows.Scan(&line.ID, &line.InvoiceID, &line.IPN, &line.Description,
			&line.Quantity, &line.UnitPrice, &line.Total)
		lines = append(lines, line)
	}
	if lines == nil {
		lines = []InvoiceLine{}
	}
	return lines
}

func handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().CreateInvoice(w, r)
}

func handleUpdateInvoice(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().UpdateInvoice(w, r, id)
}

func handleCreateInvoiceFromSalesOrder(w http.ResponseWriter, r *http.Request, salesOrderID string) {
	getSalesHandler().CreateInvoiceFromSalesOrder(w, r, salesOrderID)
}

func handleSendInvoice(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().SendInvoice(w, r, id)
}

func handleMarkInvoicePaid(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().MarkInvoicePaid(w, r, id)
}

func handleGenerateInvoicePDF(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().GenerateInvoicePDF(w, r, id)
}

func generateInvoiceNumber() string {
	year := time.Now().Year()

	// Get the highest sequence number for this year
	var maxSeq sql.NullInt64
	db.QueryRow(`SELECT MAX(CAST(SUBSTR(invoice_number, LENGTH('INV-' || ? || '-') + 1) AS INTEGER))
		FROM invoices WHERE invoice_number LIKE 'INV-' || ? || '-%'`, year, year).Scan(&maxSeq)

	seq := 1
	if maxSeq.Valid {
		seq = int(maxSeq.Int64) + 1
	}

	return fmt.Sprintf("INV-%d-%05d", year, seq)
}

// updateOverdueInvoices should be called periodically.
func updateOverdueInvoices() {
	getSalesHandler().UpdateOverdueInvoices()
}
