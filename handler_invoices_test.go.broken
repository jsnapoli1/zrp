package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test helpers
func extractInvoice(t *testing.T, body []byte) Invoice {
	t.Helper()
	var resp APIResponse
	json.Unmarshal(body, &resp)
	b, _ := json.Marshal(resp.Data)
	var inv Invoice
	json.Unmarshal(b, &inv)
	return inv
}

func extractInvoices(t *testing.T, body []byte) []Invoice {
	t.Helper()
	var resp APIResponse
	json.Unmarshal(body, &resp)
	b, _ := json.Marshal(resp.Data)
	var invs []Invoice
	json.Unmarshal(b, &invs)
	return invs
}

func setupInvoiceTestData(t *testing.T) (salesOrderID, quoteID string) {
	// Create test quote
	quoteID = nextID("Q", "quotes", 3)
	_, err := db.Exec(`INSERT INTO quotes (id, customer, status, created_at) VALUES (?, ?, ?, ?)`,
		quoteID, "Test Customer", "accepted", time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create test quote: %v", err)
	}

	// Add quote lines
	_, err = db.Exec(`INSERT INTO quote_lines (quote_id, ipn, description, qty, unit_price) VALUES (?, ?, ?, ?, ?)`,
		quoteID, "TEST-001", "Test Product 1", 5, 100.0)
	if err != nil {
		t.Fatalf("Failed to create quote line: %v", err)
	}

	_, err = db.Exec(`INSERT INTO quote_lines (quote_id, ipn, description, qty, unit_price) VALUES (?, ?, ?, ?, ?)`,
		quoteID, "TEST-002", "Test Product 2", 2, 250.0)
	if err != nil {
		t.Fatalf("Failed to create quote line: %v", err)
	}

	// Create sales order from quote
	salesOrderID = nextID("SO", "sales_orders", 3)
	_, err = db.Exec(`INSERT INTO sales_orders (id, quote_id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		salesOrderID, quoteID, "Test Customer", "shipped", "testuser", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Add sales order lines with shipped quantities
	_, err = db.Exec(`INSERT INTO sales_order_lines (sales_order_id, ipn, description, qty, qty_shipped, unit_price) VALUES (?, ?, ?, ?, ?, ?)`,
		salesOrderID, "TEST-001", "Test Product 1", 5, 5, 100.0)
	if err != nil {
		t.Fatalf("Failed to create sales order line: %v", err)
	}

	_, err = db.Exec(`INSERT INTO sales_order_lines (sales_order_id, ipn, description, qty, qty_shipped, unit_price) VALUES (?, ?, ?, ?, ?, ?)`,
		salesOrderID, "TEST-002", "Test Product 2", 2, 2, 250.0)
	if err != nil {
		t.Fatalf("Failed to create sales order line: %v", err)
	}

	return salesOrderID, quoteID
}

func TestCreateInvoiceFromSalesOrder(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	salesOrderID, _ := setupInvoiceTestData(t)

	// Test creating invoice from sales order
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/sales-orders/%s/create-invoice", salesOrderID), nil)
	rec := httptest.NewRecorder()

	handleCreateInvoiceFromSalesOrder(rec, req, salesOrderID)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	invoice := extractInvoice(t, rec.Body.Bytes())

	// Verify invoice fields
	if invoice.ID == "" {
		t.Error("Invoice ID should not be empty")
	}
	if invoice.InvoiceNumber == "" {
		t.Error("Invoice number should not be empty")
	}
	if !strings.HasPrefix(invoice.InvoiceNumber, "INV-2") {
		t.Errorf("Invoice number should start with INV-2, got: %s", invoice.InvoiceNumber)
	}
	if invoice.SalesOrderID != salesOrderID {
		t.Errorf("Expected sales_order_id %s, got %s", salesOrderID, invoice.SalesOrderID)
	}
	if invoice.Customer != "Test Customer" {
		t.Errorf("Expected customer 'Test Customer', got %s", invoice.Customer)
	}
	if invoice.Status != "draft" {
		t.Errorf("Expected status 'draft', got %s", invoice.Status)
	}
	if invoice.Total != 1100.0 {
		t.Errorf("Expected total 1100.0 (1000 + 100 tax), got %f", invoice.Total)
	}
	if invoice.Tax != 100.0 { // 10% tax on $1000
		t.Errorf("Expected tax 100.0, got %f", invoice.Tax)
	}

	// Verify invoice lines were created
	if len(invoice.Lines) != 2 {
		t.Errorf("Expected 2 invoice lines, got %d", len(invoice.Lines))
	} else {
		line1 := invoice.Lines[0]
		if line1.IPN != "TEST-001" || line1.Quantity != 5 || line1.UnitPrice != 100.0 || line1.Total != 500.0 {
			t.Errorf("Invoice line 1 incorrect: %+v", line1)
		}

		line2 := invoice.Lines[1]
		if line2.IPN != "TEST-002" || line2.Quantity != 2 || line2.UnitPrice != 250.0 || line2.Total != 500.0 {
			t.Errorf("Invoice line 2 incorrect: %+v", line2)
		}
	}
}

func TestCreateInvoiceFromNonShippedOrder(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	salesOrderID, _ := setupInvoiceTestData(t)

	// Update order status to 'confirmed' (not shipped yet)
	_, err := db.Exec(`UPDATE sales_orders SET status = ? WHERE id = ?`, "confirmed", salesOrderID)
	if err != nil {
		t.Fatalf("Failed to update order status: %v", err)
	}

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/sales-orders/%s/create-invoice", salesOrderID), nil)
	rec := httptest.NewRecorder()

	handleCreateInvoiceFromSalesOrder(rec, req, salesOrderID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "must be shipped") {
		t.Error("Expected error message about order needing to be shipped")
	}
}

func TestCreateInvoiceAlreadyExists(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	salesOrderID, _ := setupInvoiceTestData(t)

	// Create an invoice first
	invoiceID := nextID("INV", "invoices", 6)
	_, err := db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", salesOrderID, "Test Customer", time.Now().Format(time.RFC3339), time.Now().AddDate(0, 0, 30).Format(time.RFC3339), "draft", 1000.0, 100.0, time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create existing invoice: %v", err)
	}

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/sales-orders/%s/create-invoice", salesOrderID), nil)
	rec := httptest.NewRecorder()

	handleCreateInvoiceFromSalesOrder(rec, req, salesOrderID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "already has an invoice") {
		t.Error("Expected error message about existing invoice")
	}
}

func TestListInvoices(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	// Create test sales orders first
	now := time.Now()
	so1ID := fmt.Sprintf("SO-%d-001", now.Year())
	so2ID := fmt.Sprintf("SO-%d-002", now.Year())
	
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		so1ID, "Customer A", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order 1: %v", err)
	}
	
	_, err = db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		so2ID, "Customer B", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order 2: %v", err)
	}

	// Create test invoices
	invoice1ID := fmt.Sprintf("INV-%d-000001", now.Year())
	invoice2ID := fmt.Sprintf("INV-%d-000002", now.Year())

	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoice1ID, "INV-2026-00001", so1ID, "Customer A", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "sent", 1000.0, 100.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice 1: %v", err)
	}

	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoice2ID, "INV-2026-00002", so2ID, "Customer B", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "paid", 2000.0, 200.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice 2: %v", err)
	}

	// Test list all invoices
	req, _ := http.NewRequest("GET", "/api/v1/invoices", nil)
	rec := httptest.NewRecorder()

	handleListInvoices(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	invoices := extractInvoices(t, rec.Body.Bytes())

	if len(invoices) != 2 {
		t.Errorf("Expected 2 invoices, got %d", len(invoices))
	}

	// Test filter by status
	req, _ = http.NewRequest("GET", "/api/v1/invoices?status=sent", nil)
	rec = httptest.NewRecorder()

	handleListInvoices(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	invoices = extractInvoices(t, rec.Body.Bytes())

	if len(invoices) != 1 {
		t.Errorf("Expected 1 invoice with status 'sent', got %d", len(invoices))
	}

	if invoices[0].Status != "sent" {
		t.Errorf("Expected status 'sent', got %s", invoices[0].Status)
	}
}

func TestGetInvoice(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	invoiceID := nextID("INV", "invoices", 6)
	salesOrderID := nextID("SO", "sales_orders", 3)
	now := time.Now()

	// Create sales order first
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		salesOrderID, "Test Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create invoice
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, notes, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", salesOrderID, "Test Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "draft", 1000.0, 100.0, "Test notes", now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Create invoice lines
	_, err = db.Exec(`INSERT INTO invoice_lines (invoice_id, ipn, description, quantity, unit_price, total) VALUES (?, ?, ?, ?, ?, ?)`,
		invoiceID, "TEST-001", "Test Product 1", 5, 100.0, 500.0)
	if err != nil {
		t.Fatalf("Failed to create invoice line: %v", err)
	}

	_, err = db.Exec(`INSERT INTO invoice_lines (invoice_id, ipn, description, quantity, unit_price, total) VALUES (?, ?, ?, ?, ?, ?)`,
		invoiceID, "TEST-002", "Test Product 2", 2, 250.0, 500.0)
	if err != nil {
		t.Fatalf("Failed to create invoice line: %v", err)
	}

	// Test get invoice
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/invoices/%s", invoiceID), nil)
	rec := httptest.NewRecorder()

	handleGetInvoice(rec, req, invoiceID)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	invoice := extractInvoice(t, rec.Body.Bytes())

	if invoice.ID != invoiceID {
		t.Errorf("Expected invoice ID %s, got %s", invoiceID, invoice.ID)
	}
	if invoice.InvoiceNumber != "INV-2026-00001" {
		t.Errorf("Expected invoice number INV-2026-00001, got %s", invoice.InvoiceNumber)
	}
	if len(invoice.Lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(invoice.Lines))
	}
	if invoice.Notes != "Test notes" {
		t.Errorf("Expected notes 'Test notes', got %s", invoice.Notes)
	}
}

func TestSendInvoice(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	invoiceID := nextID("INV", "invoices", 6)
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()

	// Create sales order first
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Test Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create draft invoice
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Test Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "draft", 1000.0, 100.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Test send invoice
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/invoices/%s/send", invoiceID), nil)
	rec := httptest.NewRecorder()

	handleSendInvoice(rec, req, invoiceID)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify status changed
	var status string
	err = db.QueryRow("SELECT status FROM invoices WHERE id = ?", invoiceID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query invoice status: %v", err)
	}

	if status != "sent" {
		t.Errorf("Expected status 'sent', got %s", status)
	}
}

func TestMarkInvoicePaid(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	invoiceID := nextID("INV", "invoices", 6)
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()

	// Create sales order first
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Test Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create sent invoice
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Test Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "sent", 1000.0, 100.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Test mark as paid
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/invoices/%s/mark-paid", invoiceID), nil)
	rec := httptest.NewRecorder()

	handleMarkInvoicePaid(rec, req, invoiceID)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify status and paid_at were set
	var status string
	var paidAt sql.NullString
	err = db.QueryRow("SELECT status, paid_at FROM invoices WHERE id = ?", invoiceID).Scan(&status, &paidAt)
	if err != nil {
		t.Fatalf("Failed to query invoice: %v", err)
	}

	if status != "paid" {
		t.Errorf("Expected status 'paid', got %s", status)
	}

	if !paidAt.Valid {
		t.Error("Expected paid_at to be set")
	}
}

func TestUpdateInvoiceOverdueStatus(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	invoiceID := nextID("INV", "invoices", 6)
	soID := nextID("SO", "sales_orders", 3)
	pastDate := time.Now().AddDate(0, 0, -7) // 7 days ago

	// Create sales order first
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Test Customer", "shipped", "testuser", pastDate.Format(time.RFC3339), pastDate.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create sent invoice that's past due
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Test Customer", pastDate.Format("2006-01-02"), pastDate.Format("2006-01-02"), "sent", 1000.0, 100.0, pastDate.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Run the overdue check function
	updateOverdueInvoices()

	// Verify status changed to overdue
	var status string
	err = db.QueryRow("SELECT status FROM invoices WHERE id = ?", invoiceID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query invoice status: %v", err)
	}

	if status != "overdue" {
		t.Errorf("Expected status 'overdue', got %s", status)
	}
}

func TestGenerateInvoiceNumber(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	// Test first invoice number of the year
	num1 := generateInvoiceNumber()
	expectedPrefix := fmt.Sprintf("INV-%d-", time.Now().Year())
	if !strings.HasPrefix(num1, expectedPrefix) {
		t.Errorf("Expected prefix %s, got %s", expectedPrefix, num1)
	}

	// Create sales order first
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create an invoice with this number
	invoiceID := nextID("INV", "invoices", 6)
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, num1, soID, "Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "draft", 100.0, 10.0, now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Generate next number - should increment
	num2 := generateInvoiceNumber()
	if num2 <= num1 {
		t.Errorf("Expected %s > %s", num2, num1)
	}
}

func TestInvoicePDFGeneration(t *testing.T) {
	cleanup := setupTestDB(t); defer cleanup()

	invoiceID := nextID("INV", "invoices", 6)
	soID := nextID("SO", "sales_orders", 3)
	now := time.Now()

	// Create sales order first
	_, err := db.Exec(`INSERT INTO sales_orders (id, customer, status, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		soID, "Test Customer", "shipped", "testuser", now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create sales order: %v", err)
	}

	// Create invoice with lines
	_, err = db.Exec(`INSERT INTO invoices (id, invoice_number, sales_order_id, customer, issue_date, due_date, status, total, tax, notes, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, "INV-2026-00001", soID, "Test Customer", now.Format("2006-01-02"), now.AddDate(0, 0, 30).Format("2006-01-02"), "paid", 1000.0, 100.0, "Test invoice", now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	_, err = db.Exec(`INSERT INTO invoice_lines (invoice_id, ipn, description, quantity, unit_price, total) VALUES (?, ?, ?, ?, ?, ?)`,
		invoiceID, "TEST-001", "Test Product", 10, 100.0, 1000.0)
	if err != nil {
		t.Fatalf("Failed to create invoice line: %v", err)
	}

	// Test PDF generation
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/invoices/%s/pdf", invoiceID), nil)
	rec := httptest.NewRecorder()

	handleGenerateInvoicePDF(rec, req, invoiceID)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/pdf" {
		t.Errorf("Expected content-type application/pdf, got %s", contentType)
	}

	// Check PDF content starts with PDF header
	body := rec.Body.Bytes()
	if len(body) < 4 || string(body[:4]) != "%PDF" {
		t.Error("Response should start with PDF header")
	}

	// Check for presence of invoice data in PDF (basic text search)
	pdfContent := string(body)
	if !strings.Contains(pdfContent, "INV-2026-00001") {
		t.Error("PDF should contain invoice number")
	}
	if !strings.Contains(pdfContent, "Test Customer") {
		t.Error("PDF should contain customer name")
	}
	if !strings.Contains(pdfContent, "PAID") {
		t.Error("PDF should contain PAID watermark")
	}
}