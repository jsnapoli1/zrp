package sales_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"zrp/internal/models"
	"zrp/internal/testutil"

	_ "modernc.org/sqlite"
)

func TestShipmentCRUD(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	// Create shipment
	body := `{"type":"outbound","from_address":"123 Main St","to_address":"456 Oak Ave","carrier":"FedEx","notes":"Test shipment","lines":[{"ipn":"IPN-001","qty":5}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/shipments", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreateShipment(w, req)
	if w.Code != 200 {
		t.Fatalf("create shipment: %d %s", w.Code, w.Body.String())
	}
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	shipID := data["id"].(string)
	if !strings.HasPrefix(shipID, "SHP-") {
		t.Errorf("expected SHP- prefix, got %s", shipID)
	}
	if data["type"] != "outbound" {
		t.Errorf("expected outbound, got %v", data["type"])
	}
	if data["status"] != "draft" {
		t.Errorf("expected draft, got %v", data["status"])
	}
	lines := data["lines"].([]interface{})
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	// List shipments
	req = testutil.AuthedRequest("GET", "/api/v1/shipments", nil, cookie)
	w = httptest.NewRecorder()
	h.ListShipments(w, req)
	if w.Code != 200 {
		t.Fatalf("list: %d", w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp.Data.([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 shipment, got %d", len(items))
	}

	// Get shipment
	req = testutil.AuthedRequest("GET", "/api/v1/shipments/"+shipID, nil, cookie)
	w = httptest.NewRecorder()
	h.GetShipment(w, req, shipID)
	if w.Code != 200 {
		t.Fatalf("get: %d", w.Code)
	}

	// Update shipment
	body = `{"type":"outbound","status":"packed","tracking_number":"","carrier":"FedEx","from_address":"123 Main St","to_address":"456 Oak Ave","notes":"Updated"}`
	req = testutil.AuthedRequest("PUT", "/api/v1/shipments/"+shipID, []byte(body), cookie)
	w = httptest.NewRecorder()
	h.UpdateShipment(w, req, shipID)
	if w.Code != 200 {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp.Data.(map[string]interface{})
	if data["status"] != "packed" {
		t.Errorf("expected packed, got %v", data["status"])
	}
}

func TestShipShipment(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	// Create
	body := `{"type":"outbound","from_address":"A","to_address":"B"}`
	req := testutil.AuthedRequest("POST", "/api/v1/shipments", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreateShipment(w, req)
	if w.Code != 200 {
		t.Fatalf("create shipment: %d %s", w.Code, w.Body.String())
	}
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatalf("Expected data in create response, got nil. Response: %+v", resp)
	}
	shipID := resp.Data.(map[string]interface{})["id"].(string)

	// Ship it
	body = `{"tracking_number":"1Z999","carrier":"UPS"}`
	req = testutil.AuthedRequest("POST", "/api/v1/shipments/"+shipID+"/ship", []byte(body), cookie)
	w = httptest.NewRecorder()
	h.ShipShipment(w, req, shipID)
	if w.Code != 200 {
		t.Fatalf("ship: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatalf("Expected data in response, got nil. Response: %+v", resp)
	}
	data := resp.Data.(map[string]interface{})
	if data["status"] != "shipped" {
		t.Errorf("expected shipped, got %v", data["status"])
	}
	if data["tracking_number"] != "1Z999" {
		t.Errorf("expected 1Z999, got %v", data["tracking_number"])
	}
	if data["carrier"] != "UPS" {
		t.Errorf("expected UPS, got %v", data["carrier"])
	}

	// Can't ship again
	req = testutil.AuthedRequest("POST", "/api/v1/shipments/"+shipID+"/ship", []byte(body), cookie)
	w = httptest.NewRecorder()
	h.ShipShipment(w, req, shipID)
	if w.Code != 400 {
		t.Errorf("expected 400 for double ship, got %d", w.Code)
	}
}

func TestDeliverShipment(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	// Create inbound shipment with inventory line
	// First ensure inventory item exists
	db.Exec("INSERT OR IGNORE INTO inventory (ipn, qty_on_hand) VALUES ('TEST-IPN', 10)")

	body := `{"type":"inbound","from_address":"Vendor","to_address":"Warehouse","lines":[{"ipn":"TEST-IPN","qty":5}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/shipments", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreateShipment(w, req)
	if w.Code != 200 {
		t.Fatalf("create shipment: %d %s", w.Code, w.Body.String())
	}
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatalf("Expected data in create response, got nil. Response: %+v", resp)
	}
	shipID := resp.Data.(map[string]interface{})["id"].(string)

	// Ship first
	req = testutil.AuthedRequest("POST", "/api/v1/shipments/"+shipID+"/ship", []byte(`{"tracking_number":"TR1","carrier":"DHL"}`), cookie)
	w = httptest.NewRecorder()
	h.ShipShipment(w, req, shipID)

	// Deliver
	req = testutil.AuthedRequest("POST", "/api/v1/shipments/"+shipID+"/deliver", []byte(`{}`), cookie)
	w = httptest.NewRecorder()
	h.DeliverShipment(w, req, shipID)
	if w.Code != 200 {
		t.Fatalf("deliver: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	if data["status"] != "delivered" {
		t.Errorf("expected delivered, got %v", data["status"])
	}

	// Check inventory was updated
	var qty float64
	db.QueryRow("SELECT qty_on_hand FROM inventory WHERE ipn='TEST-IPN'").Scan(&qty)
	if qty != 15 {
		t.Errorf("expected inventory 15, got %f", qty)
	}

	// Can't deliver again
	req = testutil.AuthedRequest("POST", "/api/v1/shipments/"+shipID+"/deliver", []byte(`{}`), cookie)
	w = httptest.NewRecorder()
	h.DeliverShipment(w, req, shipID)
	if w.Code != 400 {
		t.Errorf("expected 400 for double deliver, got %d", w.Code)
	}
}

func TestShipmentPackList(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	body := `{"type":"outbound","from_address":"A","to_address":"B","lines":[{"ipn":"IPN-1","qty":2},{"ipn":"IPN-2","serial_number":"SN-100","qty":1}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/shipments", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreateShipment(w, req)
	if w.Code != 200 {
		t.Fatalf("create shipment: %d %s", w.Code, w.Body.String())
	}
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatalf("Expected data in create response, got nil. Response: %+v", resp)
	}
	shipID := resp.Data.(map[string]interface{})["id"].(string)

	// Get pack list
	req = testutil.AuthedRequest("GET", "/api/v1/shipments/"+shipID+"/pack-list", nil, cookie)
	w = httptest.NewRecorder()
	h.ShipmentPackList(w, req, shipID)
	if w.Code != 200 {
		t.Fatalf("pack-list: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	pl := resp.Data.(map[string]interface{})
	plLines := pl["lines"].([]interface{})
	if len(plLines) != 2 {
		t.Errorf("expected 2 pack list lines, got %d", len(plLines))
	}
}

func TestShipmentNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	req := testutil.AuthedRequest("GET", "/api/v1/shipments/NONEXISTENT", nil, cookie)
	w := httptest.NewRecorder()
	h.GetShipment(w, req, "NONEXISTENT")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Ship non-existent
	req = testutil.AuthedRequest("POST", "/api/v1/shipments/NONEXISTENT/ship", []byte(`{"tracking_number":"X","carrier":"Y"}`), cookie)
	w = httptest.NewRecorder()
	h.ShipShipment(w, req, "NONEXISTENT")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Deliver non-existent
	req = testutil.AuthedRequest("POST", "/api/v1/shipments/NONEXISTENT/deliver", []byte(`{}`), cookie)
	w = httptest.NewRecorder()
	h.DeliverShipment(w, req, "NONEXISTENT")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Pack list non-existent
	req = testutil.AuthedRequest("GET", "/api/v1/shipments/NONEXISTENT/pack-list", nil, cookie)
	w = httptest.NewRecorder()
	h.ShipmentPackList(w, req, "NONEXISTENT")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestShipmentWithWorkOrderAndRMA(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	cookie := testutil.LoginAdmin(t, db)

	body := `{"type":"outbound","from_address":"Factory","to_address":"Customer","lines":[{"ipn":"BOARD-001","qty":1,"work_order_id":"WO-2026-0001","serial_number":"SN-001"},{"ipn":"BOARD-002","qty":1,"rma_id":"RMA-2026-0001"}]}`
	req := testutil.AuthedRequest("POST", "/api/v1/shipments", []byte(body), cookie)
	w := httptest.NewRecorder()
	h.CreateShipment(w, req)
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]interface{})
	lines := data["lines"].([]interface{})
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	line0 := lines[0].(map[string]interface{})
	if line0["work_order_id"] != "WO-2026-0001" {
		t.Errorf("expected WO link, got %v", line0["work_order_id"])
	}
	line1 := lines[1].(map[string]interface{})
	if line1["rma_id"] != "RMA-2026-0001" {
		t.Errorf("expected RMA link, got %v", line1["rma_id"])
	}
}

// Ensure ListShipments uses correct HTTP handler signature
func TestShipmentListHTTP(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/shipments", nil)
	w := httptest.NewRecorder()
	h.ListShipments(w, req)
	if w.Code != 200 {
		t.Fatalf("list: %d", w.Code)
	}
	var resp models.APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp.Data.([]interface{})
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d", len(items))
	}
}
