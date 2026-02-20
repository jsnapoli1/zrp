package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestRecordChangeAndRecentChanges(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create a vendor (which should auto-record a change)
	body := `{"name":"Test Vendor","status":"active"}`
	req := authedRequest("POST", "/api/v1/vendors", body, cookie)
	w := httptest.NewRecorder()
	handleCreateVendor(w, req)
	if w.Code != 200 {
		t.Fatalf("create vendor failed: %d %s", w.Code, w.Body.String())
	}

	// Check recent changes
	req = authedRequest("GET", "/api/v1/changes/recent", "", cookie)
	w = httptest.NewRecorder()
	handleRecentChanges(w, req)
	if w.Code != 200 {
		t.Fatalf("recent changes failed: %d %s", w.Code, w.Body.String())
	}

	var wrapper struct {
		Data []ChangeEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &wrapper)
	if len(wrapper.Data) == 0 {
		t.Fatal("expected at least one change entry")
	}
	if wrapper.Data[0].Operation != "create" {
		t.Fatalf("expected create operation, got %s", wrapper.Data[0].Operation)
	}
	if wrapper.Data[0].TableName != "vendors" {
		t.Fatalf("expected vendors table, got %s", wrapper.Data[0].TableName)
	}
}

func TestUndoChangeCreate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Create a vendor
	body := `{"name":"Undo Test Vendor","status":"active"}`
	req := authedRequest("POST", "/api/v1/vendors", body, cookie)
	w := httptest.NewRecorder()
	handleCreateVendor(w, req)
	if w.Code != 200 {
		t.Fatalf("create vendor failed: %d %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Data map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	vendorID := fmt.Sprintf("%v", createResp.Data["id"])

	// Get recent changes to find the change ID
	req = authedRequest("GET", "/api/v1/changes/recent", "", cookie)
	w = httptest.NewRecorder()
	handleRecentChanges(w, req)
	var changesResp struct {
		Data []ChangeEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &changesResp)
	if len(changesResp.Data) == 0 {
		t.Fatal("no changes found")
	}
	changeID := changesResp.Data[0].ID

	// Undo the create (should delete the vendor)
	idStr := fmt.Sprintf("%d", changeID)
	req = authedRequest("POST", "/api/v1/changes/"+idStr, "", cookie)
	w = httptest.NewRecorder()
	handleUndoChange(w, req, idStr)
	if w.Code != 200 {
		t.Fatalf("undo failed: %d %s", w.Code, w.Body.String())
	}

	// Verify vendor is deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM vendors WHERE id=?", vendorID).Scan(&count)
	if count != 0 {
		t.Fatal("expected vendor to be deleted after undo")
	}

	// Verify undo response has redo_id
	var undoResp struct {
		Data map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &undoResp)
	if undoResp.Data["redo_id"] == nil {
		t.Fatal("expected redo_id in undo response")
	}
}

func TestUndoChangeUpdate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Delete POs that reference V-001 to avoid FK constraint issues during restore
	db.Exec("DELETE FROM purchase_orders WHERE vendor_id='V-001'")

	// Get original vendor name
	var origName string
	db.QueryRow("SELECT name FROM vendors WHERE id='V-001'").Scan(&origName)

	// Update vendor
	body := `{"name":"Updated Name","status":"active"}`
	req := authedRequest("PUT", "/api/v1/vendors/V-001", body, cookie)
	w := httptest.NewRecorder()
	handleUpdateVendor(w, req, "V-001")
	if w.Code != 200 {
		t.Fatalf("update failed: %d %s", w.Code, w.Body.String())
	}

	// Get the change entry
	req = authedRequest("GET", "/api/v1/changes/recent", "", cookie)
	w = httptest.NewRecorder()
	handleRecentChanges(w, req)
	var changesResp struct {
		Data []ChangeEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &changesResp)
	if len(changesResp.Data) == 0 {
		t.Fatal("no changes found")
	}

	// Undo the update
	changeID := changesResp.Data[0].ID
	idStr := fmt.Sprintf("%d", changeID)
	req = authedRequest("POST", "/api/v1/changes/"+idStr, "", cookie)
	w = httptest.NewRecorder()
	handleUndoChange(w, req, idStr)
	if w.Code != 200 {
		t.Fatalf("undo failed: %d %s", w.Code, w.Body.String())
	}

	// Verify vendor name is restored
	var restoredName string
	db.QueryRow("SELECT name FROM vendors WHERE id='V-001'").Scan(&restoredName)
	if restoredName != origName {
		t.Fatalf("expected name %q, got %q", origName, restoredName)
	}
}

func TestUndoChangeDelete(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Delete POs that reference V-001 first (required by FK constraint)
	db.Exec("DELETE FROM purchase_orders WHERE vendor_id='V-001'")

	// Delete vendor
	req := authedRequest("DELETE", "/api/v1/vendors/V-001", "", cookie)
	w := httptest.NewRecorder()
	handleDeleteVendor(w, req, "V-001")
	if w.Code != 200 {
		t.Fatalf("delete failed: %d %s", w.Code, w.Body.String())
	}

	// Get the change entry
	req = authedRequest("GET", "/api/v1/changes/recent", "", cookie)
	w = httptest.NewRecorder()
	handleRecentChanges(w, req)
	var changesResp struct {
		Data []ChangeEntry `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &changesResp)

	// Find the delete change
	var deleteChangeID int
	for _, c := range changesResp.Data {
		if c.Operation == "delete" && c.TableName == "vendors" {
			deleteChangeID = c.ID
			break
		}
	}
	if deleteChangeID == 0 {
		t.Fatal("no delete change found")
	}

	// Undo the delete
	idStr := fmt.Sprintf("%d", deleteChangeID)
	req = authedRequest("POST", "/api/v1/changes/"+idStr, "", cookie)
	w = httptest.NewRecorder()
	handleUndoChange(w, req, idStr)
	if w.Code != 200 {
		t.Fatalf("undo failed: %d %s", w.Code, w.Body.String())
	}

	// Verify vendor is restored
	var count int
	db.QueryRow("SELECT COUNT(*) FROM vendors WHERE id='V-001'").Scan(&count)
	if count != 1 {
		t.Fatal("expected vendor to be restored")
	}
}

func TestUndoAlreadyUndone(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Insert a change manually and mark as undone
	db.Exec(`INSERT INTO change_history (table_name, record_id, operation, old_data, new_data, user_id, undone)
		VALUES ('vendors', 'V-FAKE', 'create', '', '{}', 'admin', 1)`)

	req := authedRequest("POST", "/api/v1/changes/1", "", cookie)
	w := httptest.NewRecorder()
	handleUndoChange(w, req, "1")
	if w.Code != 400 {
		t.Fatalf("expected 400 for already undone, got %d", w.Code)
	}
}

func TestUndoChangeNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("POST", "/api/v1/changes/9999", "", cookie)
	w := httptest.NewRecorder()
	handleUndoChange(w, req, "9999")
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUndoChangeInvalidID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("POST", "/api/v1/changes/abc", "", cookie)
	w := httptest.NewRecorder()
	handleUndoChange(w, req, "abc")
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRecentChangesEmpty(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("GET", "/api/v1/changes/recent", "", cookie)
	w := httptest.NewRecorder()
	handleRecentChanges(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
