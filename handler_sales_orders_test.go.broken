package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper to extract data from APIResponse wrapper
func extractSalesOrder(t *testing.T, body []byte) SalesOrder {
	t.Helper()
	var resp APIResponse
	json.Unmarshal(body, &resp)
	b, _ := json.Marshal(resp.Data)
	var so SalesOrder
	json.Unmarshal(b, &so)
	return so
}

func extractSalesOrders(t *testing.T, body []byte) []SalesOrder {
	t.Helper()
	var resp APIResponse
	json.Unmarshal(body, &resp)
	b, _ := json.Marshal(resp.Data)
	var orders []SalesOrder
	json.Unmarshal(b, &orders)
	return orders
}

func TestSalesOrderCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create
	body := `{"customer":"Acme Corp","notes":"Test order","lines":[{"ipn":"IPN-001","description":"Widget","qty":10,"unit_price":25.50}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	created := extractSalesOrder(t, w.Body.Bytes())
	if !strings.HasPrefix(created.ID, "SO-") {
		t.Errorf("expected SO- prefix, got %s", created.ID)
	}
	if created.Status != "draft" {
		t.Errorf("expected draft, got %s", created.Status)
	}
	orderID := created.ID

	// List
	req = authedRequest("GET", "/api/v1/sales-orders", "", cookie)
	w = httptest.NewRecorder()
	handleListSalesOrders(w, req)
	if w.Code != 200 {
		t.Fatalf("list: %d", w.Code)
	}
	orders := extractSalesOrders(t, w.Body.Bytes())
	if len(orders) != 1 {
		t.Errorf("expected 1, got %d", len(orders))
	}

	// Get
	req = authedRequest("GET", "/api/v1/sales-orders/"+orderID, "", cookie)
	w = httptest.NewRecorder()
	handleGetSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("get: %d", w.Code)
	}
	fetched := extractSalesOrder(t, w.Body.Bytes())
	if len(fetched.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(fetched.Lines))
	}
	if fetched.Lines[0].UnitPrice != 25.50 {
		t.Errorf("expected 25.50, got %f", fetched.Lines[0].UnitPrice)
	}

	// Update
	body = `{"customer":"Acme Corp Updated","status":"draft","notes":"updated"}`
	req = authedRequest("PUT", "/api/v1/sales-orders/"+orderID, body, cookie)
	w = httptest.NewRecorder()
	handleUpdateSalesOrder(w, req, orderID)
	if w.Code != 200 {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
}

func TestSalesOrderStatusFilter(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	for _, cust := range []string{"Alpha Inc", "Beta LLC"} {
		body := `{"customer":"` + cust + `","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`
		req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
		w := httptest.NewRecorder()
		handleCreateSalesOrder(w, req)
		if w.Code != 200 {
			t.Fatalf("create: %d", w.Code)
		}
	}

	// Filter by status
	req := authedRequest("GET", "/api/v1/sales-orders?status=draft", "", cookie)
	w := httptest.NewRecorder()
	handleListSalesOrders(w, req)
	orders := extractSalesOrders(t, w.Body.Bytes())
	if len(orders) != 2 {
		t.Errorf("expected 2 draft, got %d", len(orders))
	}

	// Filter by customer
	req = authedRequest("GET", "/api/v1/sales-orders?customer=Alpha", "", cookie)
	w = httptest.NewRecorder()
	handleListSalesOrders(w, req)
	orders = extractSalesOrders(t, w.Body.Bytes())
	if len(orders) != 1 {
		t.Errorf("expected 1 for Alpha, got %d", len(orders))
	}
}

func TestConvertQuoteToOrder(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create accepted quote
	body := `{"customer":"Test Customer","status":"accepted","lines":[{"ipn":"IPN-001","description":"Part A","qty":5,"unit_price":100}]}`
	req := authedRequest("POST", "/api/v1/quotes", body, cookie)
	w := httptest.NewRecorder()
	handleCreateQuote(w, req)
	if w.Code != 200 {
		t.Fatalf("create quote: %d %s", w.Code, w.Body.String())
	}
	var qResp APIResponse
	json.Unmarshal(w.Body.Bytes(), &qResp)
	qb, _ := json.Marshal(qResp.Data)
	var q Quote
	json.Unmarshal(qb, &q)

	// Convert
	req = authedRequest("POST", "/api/v1/quotes/"+q.ID+"/convert-to-order", "", cookie)
	w = httptest.NewRecorder()
	handleConvertQuoteToOrder(w, req, q.ID)
	if w.Code != 200 {
		t.Fatalf("convert: %d %s", w.Code, w.Body.String())
	}
	so := extractSalesOrder(t, w.Body.Bytes())
	if so.QuoteID != q.ID {
		t.Errorf("expected quote_id %s, got %s", q.ID, so.QuoteID)
	}
	if so.Customer != "Test Customer" {
		t.Errorf("expected Test Customer, got %s", so.Customer)
	}
	if len(so.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(so.Lines))
	}

	// Duplicate convert -> 409
	req = authedRequest("POST", "/api/v1/quotes/"+q.ID+"/convert-to-order", "", cookie)
	w = httptest.NewRecorder()
	handleConvertQuoteToOrder(w, req, q.ID)
	if w.Code != 409 {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestConvertDraftQuoteFails(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"customer":"Test","status":"draft","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`
	req := authedRequest("POST", "/api/v1/quotes", body, cookie)
	w := httptest.NewRecorder()
	handleCreateQuote(w, req)
	var qResp APIResponse
	json.Unmarshal(w.Body.Bytes(), &qResp)
	qb, _ := json.Marshal(qResp.Data)
	var q Quote
	json.Unmarshal(qb, &q)

	req = authedRequest("POST", "/api/v1/quotes/"+q.ID+"/convert-to-order", "", cookie)
	w = httptest.NewRecorder()
	handleConvertQuoteToOrder(w, req, q.ID)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSalesOrderWorkflow(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Seed inventory
	db.Exec("INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location) VALUES (?,?,?,?)", "WIDGET-01", 100, 0, "A1")

	// Create
	body := `{"customer":"Workflow Corp","lines":[{"ipn":"WIDGET-01","description":"Widget","qty":10,"unit_price":25}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	so := extractSalesOrder(t, w.Body.Bytes())
	id := so.ID

	// Confirm
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/confirm", "", cookie)
	w = httptest.NewRecorder()
	handleConfirmSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("confirm: %d %s", w.Code, w.Body.String())
	}
	so = extractSalesOrder(t, w.Body.Bytes())
	if so.Status != "confirmed" {
		t.Errorf("expected confirmed, got %s", so.Status)
	}

	// Allocate
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/allocate", "", cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("allocate: %d %s", w.Code, w.Body.String())
	}
	so = extractSalesOrder(t, w.Body.Bytes())
	if so.Status != "allocated" {
		t.Errorf("expected allocated, got %s", so.Status)
	}
	var qtyReserved float64
	db.QueryRow("SELECT qty_reserved FROM inventory WHERE ipn='WIDGET-01'").Scan(&qtyReserved)
	if qtyReserved != 10 {
		t.Errorf("expected 10 reserved, got %.0f", qtyReserved)
	}

	// Pick
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/pick", "", cookie)
	w = httptest.NewRecorder()
	handlePickSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("pick: %d %s", w.Code, w.Body.String())
	}
	so = extractSalesOrder(t, w.Body.Bytes())
	if so.Status != "picked" {
		t.Errorf("expected picked, got %s", so.Status)
	}

	// Ship
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/ship", "", cookie)
	w = httptest.NewRecorder()
	handleShipSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("ship: %d %s", w.Code, w.Body.String())
	}
	so = extractSalesOrder(t, w.Body.Bytes())
	if so.Status != "shipped" {
		t.Errorf("expected shipped, got %s", so.Status)
	}
	var qtyOnHand float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn='WIDGET-01'").Scan(&qtyOnHand)
	if qtyOnHand != 90 {
		t.Errorf("expected 90, got %.0f", qtyOnHand)
	}
	if so.ShipmentID == nil {
		t.Error("expected shipment_id")
	}

	// Invoice
	req = authedRequest("POST", "/api/v1/sales-orders/"+id+"/invoice", "", cookie)
	w = httptest.NewRecorder()
	handleInvoiceSalesOrder(w, req, id)
	if w.Code != 200 {
		t.Fatalf("invoice: %d %s", w.Code, w.Body.String())
	}
	so = extractSalesOrder(t, w.Body.Bytes())
	if so.Status != "invoiced" {
		t.Errorf("expected invoiced, got %s", so.Status)
	}
	if so.InvoiceID == nil {
		t.Error("expected invoice_id")
	}
	var invTotal float64
	db.QueryRow("SELECT total FROM invoices WHERE sales_order_id=?", id).Scan(&invTotal)
	if invTotal != 250 {
		t.Errorf("expected 250, got %.2f", invTotal)
	}
}

func TestAllocateInsufficientInventory(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	db.Exec("INSERT INTO inventory (ipn,qty_on_hand,qty_reserved,location) VALUES (?,?,?,?)", "SCARCE-01", 5, 0, "A1")

	body := `{"customer":"Test","lines":[{"ipn":"SCARCE-01","qty":10,"unit_price":10}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	so := extractSalesOrder(t, w.Body.Bytes())

	// Confirm
	req = authedRequest("POST", "/api/v1/sales-orders/"+so.ID+"/confirm", "", cookie)
	w = httptest.NewRecorder()
	handleConfirmSalesOrder(w, req, so.ID)

	// Allocate should fail
	req = authedRequest("POST", "/api/v1/sales-orders/"+so.ID+"/allocate", "", cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, so.ID)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSalesOrderInvalidTransition(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"customer":"Test","lines":[{"ipn":"IPN-001","qty":1,"unit_price":10}]}`
	req := authedRequest("POST", "/api/v1/sales-orders", body, cookie)
	w := httptest.NewRecorder()
	handleCreateSalesOrder(w, req)
	so := extractSalesOrder(t, w.Body.Bytes())

	// Try allocate from draft (needs confirmed)
	req = authedRequest("POST", "/api/v1/sales-orders/"+so.ID+"/allocate", "", cookie)
	w = httptest.NewRecorder()
	handleAllocateSalesOrder(w, req, so.ID)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
