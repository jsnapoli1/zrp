package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBulkUpdateInventoryLocation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ids":["CAP-001-0001","RES-001-0001"],"updates":{"location":"Shelf-B3"}}`
	req := authedRequest("POST", "/api/v1/inventory/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateInventory(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["success"].(float64) != 2 {
		t.Errorf("expected 2 success, got %v", data["success"])
	}

	// Verify
	var loc string
	db.QueryRow("SELECT location FROM inventory WHERE ipn='CAP-001-0001'").Scan(&loc)
	if loc != "Shelf-B3" {
		t.Errorf("expected Shelf-B3, got %s", loc)
	}
}

func TestBulkUpdateInventoryReorderPoint(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ids":["CAP-001-0001"],"updates":{"reorder_point":"50"}}`
	req := authedRequest("POST", "/api/v1/inventory/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateInventory(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var rp float64
	db.QueryRow("SELECT reorder_point FROM inventory WHERE ipn='CAP-001-0001'").Scan(&rp)
	if rp != 50 {
		t.Errorf("expected 50, got %v", rp)
	}
}

func TestBulkUpdateInventoryDisallowedField(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ids":["CAP-001-0001"],"updates":{"qty_on_hand":"999"}}`
	req := authedRequest("POST", "/api/v1/inventory/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateInventory(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for disallowed field, got %d", w.Code)
	}
}

func TestBulkUpdateInventoryEmptyIDs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ids":[],"updates":{"location":"X"}}`
	req := authedRequest("POST", "/api/v1/inventory/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateInventory(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBulkUpdateInventoryNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ids":["NONEXISTENT"],"updates":{"location":"X"}}`
	req := authedRequest("POST", "/api/v1/inventory/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateInventory(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["failed"].(float64) != 1 {
		t.Errorf("expected 1 failed, got %v", data["failed"])
	}
}

func TestBulkUpdateWorkOrdersStatus(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	year := fmt.Sprintf("%d", time.Now().Year())
	body := fmt.Sprintf(`{"ids":["WO-%s-0001","WO-%s-0002"],"updates":{"status":"completed"}}`, year, year)
	req := authedRequest("POST", "/api/v1/workorders/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateWorkOrders(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["success"].(float64) != 2 {
		t.Errorf("expected 2 success, got %v", data["success"])
	}

	var status string
	db.QueryRow("SELECT status FROM work_orders WHERE id=?", fmt.Sprintf("WO-%s-0001", year)).Scan(&status)
	if status != "completed" {
		t.Errorf("expected completed, got %s", status)
	}
}

func TestBulkUpdateWorkOrdersPriority(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	year := fmt.Sprintf("%d", time.Now().Year())
	body := fmt.Sprintf(`{"ids":["WO-%s-0001"],"updates":{"priority":"critical"}}`, year)
	req := authedRequest("POST", "/api/v1/workorders/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateWorkOrders(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var prio string
	db.QueryRow("SELECT priority FROM work_orders WHERE id=?", fmt.Sprintf("WO-%s-0001", year)).Scan(&prio)
	if prio != "critical" {
		t.Errorf("expected critical, got %s", prio)
	}
}

func TestBulkUpdateWorkOrdersInvalidStatus(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	year := fmt.Sprintf("%d", time.Now().Year())
	body := fmt.Sprintf(`{"ids":["WO-%s-0001"],"updates":{"status":"bogus"}}`, year)
	req := authedRequest("POST", "/api/v1/workorders/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateWorkOrders(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBulkUpdateWorkOrdersDisallowedField(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	year := fmt.Sprintf("%d", time.Now().Year())
	body := fmt.Sprintf(`{"ids":["WO-%s-0001"],"updates":{"assembly_ipn":"HACK"}}`, year)
	req := authedRequest("POST", "/api/v1/workorders/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateWorkOrders(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBulkUpdateDevicesStatus(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ids":["SN-001","SN-002"],"updates":{"status":"inactive"}}`
	req := authedRequest("POST", "/api/v1/devices/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateDevices(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["success"].(float64) != 2 {
		t.Errorf("expected 2 success, got %v", data["success"])
	}

	var status string
	db.QueryRow("SELECT status FROM devices WHERE serial_number='SN-001'").Scan(&status)
	if status != "inactive" {
		t.Errorf("expected inactive, got %s", status)
	}
}

func TestBulkUpdateDevicesCustomer(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ids":["SN-001"],"updates":{"customer":"NewCorp","location":"Building C"}}`
	req := authedRequest("POST", "/api/v1/devices/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateDevices(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var cust, loc string
	db.QueryRow("SELECT customer, location FROM devices WHERE serial_number='SN-001'").Scan(&cust, &loc)
	if cust != "NewCorp" {
		t.Errorf("expected NewCorp, got %s", cust)
	}
	if loc != "Building C" {
		t.Errorf("expected Building C, got %s", loc)
	}
}

func TestBulkUpdateDevicesInvalidStatus(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ids":["SN-001"],"updates":{"status":"bogus"}}`
	req := authedRequest("POST", "/api/v1/devices/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateDevices(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBulkUpdateDevicesDisallowedField(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"ids":["SN-001"],"updates":{"serial_number":"HACK"}}`
	req := authedRequest("POST", "/api/v1/devices/bulk-update", body, cookie)
	w := httptest.NewRecorder()
	handleBulkUpdateDevices(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
