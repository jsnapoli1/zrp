package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShipmentCRUD(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create shipment
	body := `{"type":"outbound","from_address":"123 Main St","to_address":"456 Oak Ave","carrier":"FedEx","notes":"Test shipment","lines":[{"ipn":"IPN-001","qty":5}]}`
	req := authedRequest("POST", "/api/v1/shipments", body, cookie)
	w := httptest.NewRecorder()
	handleCreateShipment(w, req)
	if w.Code != 200 {
		t.Fatalf("create shipment: %d %s", w.Code, w.Body.String())
	}
	var resp APIResponse
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
	req = authedRequest("GET", "/api/v1/shipments", "", cookie)
	w = httptest.NewRecorder()
	handleListShipments(w, req)
	if w.Code != 200 {
		t.Fatalf("list: %d", w.Code)
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp.Data.([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 shipment, got %d", len(items))
	}

	// Get shipment
	req = authedRequest("GET", "/api/v1/shipments/"+shipID, "", cookie)
	w = httptest.NewRecorder()
	handleGetShipment(w, req, shipID)
	if w.Code != 200 {
		t.Fatalf("get: %d", w.Code)
	}

	// Update shipment
	body = `{"type":"outbound","status":"packed","tracking_number":"","carrier":"FedEx","from_address":"123 Main St","to_address":"456 Oak Ave","notes":"Updated"}`
	req = authedRequest("PUT", "/api/v1/shipments/"+shipID, body, cookie)
	w = httptest.NewRecorder()
	handleUpdateShipment(w, req, shipID)
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
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create
	body := `{"type":"outbound","from_address":"A","to_address":"B"}`
	req := authedRequest("POST", "/api/v1/shipments", body, cookie)
	w := httptest.NewRecorder()
	handleCreateShipment(w, req)
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	shipID := resp.Data.(map[string]interface{})["id"].(string)

	// Ship it
	body = `{"tracking_number":"1Z999","carrier":"UPS"}`
	req = authedRequest("POST", "/api/v1/shipments/"+shipID+"/ship", body, cookie)
	w = httptest.NewRecorder()
	handleShipShipment(w, req, shipID)
	if w.Code != 200 {
		t.Fatalf("ship: %d %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
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
	req = authedRequest("POST", "/api/v1/shipments/"+shipID+"/ship", body, cookie)
	w = httptest.NewRecorder()
	handleShipShipment(w, req, shipID)
	if w.Code != 400 {
		t.Errorf("expected 400 for double ship, got %d", w.Code)
	}
}

func TestDeliverShipment(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create inbound shipment with inventory line
	// First ensure inventory item exists
	db.Exec("INSERT OR IGNORE INTO inventory (ipn, qty_on_hand) VALUES ('TEST-IPN', 10)")

	body := `{"type":"inbound","from_address":"Vendor","to_address":"Warehouse","lines":[{"ipn":"TEST-IPN","qty":5}]}`
	req := authedRequest("POST", "/api/v1/shipments", body, cookie)
	w := httptest.NewRecorder()
	handleCreateShipment(w, req)
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	shipID := resp.Data.(map[string]interface{})["id"].(string)

	// Ship first
	req = authedRequest("POST", "/api/v1/shipments/"+shipID+"/ship", `{"tracking_number":"TR1","carrier":"DHL"}`, cookie)
	w = httptest.NewRecorder()
	handleShipShipment(w, req, shipID)

	// Deliver
	req = authedRequest("POST", "/api/v1/shipments/"+shipID+"/deliver", `{}`, cookie)
	w = httptest.NewRecorder()
	handleDeliverShipment(w, req, shipID)
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
	req = authedRequest("POST", "/api/v1/shipments/"+shipID+"/deliver", `{}`, cookie)
	w = httptest.NewRecorder()
	handleDeliverShipment(w, req, shipID)
	if w.Code != 400 {
		t.Errorf("expected 400 for double deliver, got %d", w.Code)
	}
}

func TestShipmentPackList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"type":"outbound","from_address":"A","to_address":"B","lines":[{"ipn":"IPN-1","qty":2},{"ipn":"IPN-2","serial_number":"SN-100","qty":1}]}`
	req := authedRequest("POST", "/api/v1/shipments", body, cookie)
	w := httptest.NewRecorder()
	handleCreateShipment(w, req)
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	shipID := resp.Data.(map[string]interface{})["id"].(string)

	// Get pack list
	req = authedRequest("GET", "/api/v1/shipments/"+shipID+"/pack-list", "", cookie)
	w = httptest.NewRecorder()
	handleShipmentPackList(w, req, shipID)
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
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("GET", "/api/v1/shipments/NONEXISTENT", "", cookie)
	w := httptest.NewRecorder()
	handleGetShipment(w, req, "NONEXISTENT")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Ship non-existent
	req = authedRequest("POST", "/api/v1/shipments/NONEXISTENT/ship", `{"tracking_number":"X","carrier":"Y"}`, cookie)
	w = httptest.NewRecorder()
	handleShipShipment(w, req, "NONEXISTENT")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Deliver non-existent
	req = authedRequest("POST", "/api/v1/shipments/NONEXISTENT/deliver", `{}`, cookie)
	w = httptest.NewRecorder()
	handleDeliverShipment(w, req, "NONEXISTENT")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Pack list non-existent
	req = authedRequest("GET", "/api/v1/shipments/NONEXISTENT/pack-list", "", cookie)
	w = httptest.NewRecorder()
	handleShipmentPackList(w, req, "NONEXISTENT")
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestShipmentWithWorkOrderAndRMA(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"type":"outbound","from_address":"Factory","to_address":"Customer","lines":[{"ipn":"BOARD-001","qty":1,"work_order_id":"WO-2026-0001","serial_number":"SN-001"},{"ipn":"BOARD-002","qty":1,"rma_id":"RMA-2026-0001"}]}`
	req := authedRequest("POST", "/api/v1/shipments", body, cookie)
	w := httptest.NewRecorder()
	handleCreateShipment(w, req)
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var resp APIResponse
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

// Ensure handleListShipments uses correct HTTP handler signature
func TestShipmentListHTTP(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/shipments", nil)
	w := httptest.NewRecorder()
	handleListShipments(w, req)
	if w.Code != 200 {
		t.Fatalf("list: %d", w.Code)
	}
	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp.Data.([]interface{})
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d", len(items))
	}
}
