package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestUndoDeleteVendor(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Delete POs that reference V-001 first (required by FK constraint)
	db.Exec("DELETE FROM purchase_orders WHERE vendor_id='V-001'")

	// Verify vendor V-001 exists (seeded)
	req := authedRequest("GET", "/api/v1/vendors/V-001", "", cookie)
	w := httptest.NewRecorder()
	handleGetVendor(w, req, "V-001")
	if w.Code != 200 {
		t.Fatalf("expected vendor to exist, got %d: %s", w.Code, w.Body.String())
	}

	// Delete vendor
	req = authedRequest("DELETE", "/api/v1/vendors/V-001", "", cookie)
	w = httptest.NewRecorder()
	handleDeleteVendor(w, req, "V-001")
	if w.Code != 200 {
		t.Fatalf("delete failed: %d %s", w.Code, w.Body.String())
	}

	// Check response has undo_id (wrapped in APIResponse.Data)
	var delWrapper struct {
		Data map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &delWrapper)
	undoID, ok := delWrapper.Data["undo_id"]
	if !ok {
		t.Fatalf("expected undo_id in delete response, got: %s", w.Body.String())
	}

	// Verify vendor is gone
	req = authedRequest("GET", "/api/v1/vendors/V-001", "", cookie)
	w = httptest.NewRecorder()
	handleGetVendor(w, req, "V-001")
	if w.Code == 200 {
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["error"] == nil {
			t.Fatal("expected vendor to be deleted")
		}
	}

	// List undo entries
	req = authedRequest("GET", "/api/v1/undo", "", cookie)
	w = httptest.NewRecorder()
	handleListUndo(w, req)
	if w.Code != 200 {
		t.Fatalf("list undo failed: %d %s", w.Code, w.Body.String())
	}
	var undoWrapper struct {
		Data []UndoLogEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &undoWrapper)
	if len(undoWrapper.Data) == 0 {
		t.Fatalf("expected at least one undo entry, got: %s", w.Body.String())
	}
	undoList := undoWrapper.Data

	// Perform undo - undoList[0].ID is valid since we just checked
	_ = undoID // used above to verify existence
	undoIDStr := fmt.Sprintf("%d", undoList[0].ID)
	req = authedRequest("POST", "/api/v1/undo/"+undoIDStr, "", cookie)
	w = httptest.NewRecorder()
	handlePerformUndo(w, req, undoIDStr)
	if w.Code != 200 {
		t.Fatalf("undo failed: %d %s", w.Code, w.Body.String())
	}

	// Verify vendor is restored
	req = authedRequest("GET", "/api/v1/vendors/V-001", "", cookie)
	w = httptest.NewRecorder()
	handleGetVendor(w, req, "V-001")
	if w.Code != 200 {
		t.Fatalf("expected vendor restored, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUndoExpired(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create an undo entry manually with expired time
	db.Exec(`INSERT INTO undo_log (user_id, action, entity_type, entity_id, previous_data, created_at, expires_at)
		VALUES ('admin', 'delete', 'vendor', 'V-FAKE', '{}', '2020-01-01 00:00:00', '2020-01-02 00:00:00')`)

	// Try to undo it
	req := authedRequest("POST", "/api/v1/undo/1", "", cookie)
	w := httptest.NewRecorder()
	handlePerformUndo(w, req, "1")
	if w.Code != 404 {
		t.Fatalf("expected 404 for expired undo, got %d", w.Code)
	}
}

func TestUndoInvalidID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("POST", "/api/v1/undo/abc", "", cookie)
	w := httptest.NewRecorder()
	handlePerformUndo(w, req, "abc")
	if w.Code != 400 {
		t.Fatalf("expected 400 for invalid id, got %d", w.Code)
	}
}

func TestUndoListEmpty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("GET", "/api/v1/undo", "", cookie)
	w := httptest.NewRecorder()
	handleListUndo(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []UndoLogEntry
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

func TestUndoBulkDeleteECO(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Find a seeded ECO
	var ecoID string
	err := db.QueryRow("SELECT id FROM ecos LIMIT 1").Scan(&ecoID)
	if err != nil {
		t.Skip("no ECOs seeded")
	}

	// Bulk delete it
	body := fmt.Sprintf(`{"ids":[%q],"action":"delete"}`, ecoID)
	req := authedRequest("POST", "/api/v1/ecos/bulk", body, cookie)
	w := httptest.NewRecorder()
	handleBulkECOs(w, req)
	if w.Code != 200 {
		t.Fatalf("bulk delete failed: %d %s", w.Code, w.Body.String())
	}

	// Check ECO is deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM ecos WHERE id=?", ecoID).Scan(&count)
	if count != 0 {
		t.Fatal("expected ECO to be deleted")
	}

	// Check undo entry exists
	req = authedRequest("GET", "/api/v1/undo", "", cookie)
	w = httptest.NewRecorder()
	handleListUndo(w, req)
	var listWrapper struct {
		Data []UndoLogEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &listWrapper)
	if len(listWrapper.Data) == 0 {
		t.Fatal("expected undo entry for bulk delete")
	}

	// Undo it
	undoIDStr := fmt.Sprintf("%d", listWrapper.Data[0].ID)
	req = authedRequest("POST", "/api/v1/undo/"+undoIDStr, "", cookie)
	w = httptest.NewRecorder()
	handlePerformUndo(w, req, undoIDStr)
	if w.Code != 200 {
		t.Fatalf("undo failed: %d %s", w.Code, w.Body.String())
	}

	// Verify ECO restored
	db.QueryRow("SELECT COUNT(*) FROM ecos WHERE id=?", ecoID).Scan(&count)
	if count != 1 {
		t.Fatal("expected ECO to be restored")
	}
}

func TestSnapshotEntity(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Test snapshot for unsupported type
	_, err := snapshotEntity("unknown", "id")
	if err == nil {
		t.Fatal("expected error for unsupported entity type")
	}

	// Test snapshot for nonexistent entity
	_, err = snapshotEntity("vendor", "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent entity")
	}
}
