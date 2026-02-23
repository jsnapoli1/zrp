package field_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"zrp/internal/handlers/field"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

func setupFirmwareTestDB(t *testing.T) (*sql.DB, *field.Handler) {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create firmware_campaigns table
	_, err = testDB.Exec(`
		CREATE TABLE firmware_campaigns (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			category TEXT DEFAULT 'public',
			status TEXT DEFAULT 'draft',
			target_filter TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create firmware_campaigns table: %v", err)
	}

	// Create campaign_devices table
	_, err = testDB.Exec(`
		CREATE TABLE campaign_devices (
			campaign_id TEXT NOT NULL,
			serial_number TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			updated_at DATETIME,
			PRIMARY KEY (campaign_id, serial_number)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create campaign_devices table: %v", err)
	}

	// Create devices table (for launch testing)
	_, err = testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			ipn TEXT NOT NULL,
			firmware_version TEXT,
			customer TEXT,
			location TEXT,
			status TEXT DEFAULT 'active',
			install_date TEXT,
			last_seen DATETIME,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	h := &field.Handler{
		DB:  testDB,
		Hub: nil,
		NextIDFunc: nextIDFunc(testDB),
		RecordChangeJSON: func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error) {
			return 0, nil
		},
		GetDeviceSnapshot: func(id string) (map[string]interface{}, error) {
			return nil, nil
		},
		GetRMASnapshot: func(id string) (map[string]interface{}, error) {
			return nil, nil
		},
	}

	return testDB, h
}

func insertTestCampaign(t *testing.T, db *sql.DB, id, name, version, category, status string) {
	t.Helper()
	_, err := db.Exec(
		"INSERT INTO firmware_campaigns (id, name, version, category, status, created_at) VALUES (?, ?, ?, ?, ?, datetime('now'))",
		id, name, version, category, status,
	)
	if err != nil {
		t.Fatalf("Failed to insert test campaign: %v", err)
	}
}

func insertTestFirmwareDevice(t *testing.T, db *sql.DB, serial, ipn, status string) {
	t.Helper()
	_, err := db.Exec(
		"INSERT INTO devices (serial_number, ipn, status, created_at) VALUES (?, ?, ?, datetime('now'))",
		serial, ipn, status,
	)
	if err != nil {
		t.Fatalf("Failed to insert test device: %v", err)
	}
}

func insertCampaignDevice(t *testing.T, db *sql.DB, campaignID, serial, status string) {
	t.Helper()
	_, err := db.Exec(
		"INSERT INTO campaign_devices (campaign_id, serial_number, status) VALUES (?, ?, ?)",
		campaignID, serial, status,
	)
	if err != nil {
		t.Fatalf("Failed to insert campaign device: %v", err)
	}
}

// Test handleListCampaigns - Empty
func TestHandleListCampaigns_Empty(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("GET", "/api/firmware/campaigns", nil)
	w := httptest.NewRecorder()

	h.ListCampaigns(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	campaigns, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(campaigns) != 0 {
		t.Errorf("Expected empty array, got %d campaigns", len(campaigns))
	}
}

// Test handleListCampaigns - With Data
func TestHandleListCampaigns_WithData(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Campaign 1", "v1.0.0", "public", "draft")
	insertTestCampaign(t, testDB, "FW-002", "Campaign 2", "v1.1.0", "public", "active")
	insertTestCampaign(t, testDB, "FW-003", "Campaign 3", "v2.0.0", "beta", "completed")

	req := httptest.NewRequest("GET", "/api/firmware/campaigns", nil)
	w := httptest.NewRecorder()

	h.ListCampaigns(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	campaignsData, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(campaignsData) != 3 {
		t.Errorf("Expected 3 campaigns, got %d", len(campaignsData))
	}
}

// Test handleGetCampaign - Success
func TestHandleGetCampaign_Success(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Test Campaign", "v1.0.0", "public", "draft")

	req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-001", nil)
	w := httptest.NewRecorder()

	h.GetCampaign(w, req, "FW-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	campaign := resp.Data.(map[string]interface{})
	if campaign["id"] != "FW-001" {
		t.Errorf("Expected ID FW-001, got %v", campaign["id"])
	}
	if campaign["name"] != "Test Campaign" {
		t.Errorf("Expected name 'Test Campaign', got %v", campaign["name"])
	}
	if campaign["version"] != "v1.0.0" {
		t.Errorf("Expected version 'v1.0.0', got %v", campaign["version"])
	}
}

// Test handleGetCampaign - Not Found
func TestHandleGetCampaign_NotFound(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-999", nil)
	w := httptest.NewRecorder()

	h.GetCampaign(w, req, "FW-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test handleCreateCampaign - Success
func TestHandleCreateCampaign_Success(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	campaign := map[string]interface{}{
		"name":    "New Campaign",
		"version": "v2.0.0",
		"notes":   "Test notes",
	}

	body, _ := json.Marshal(campaign)
	req := httptest.NewRequest("POST", "/api/firmware/campaigns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateCampaign(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	created := resp.Data.(map[string]interface{})
	if created["name"] != "New Campaign" {
		t.Errorf("Expected name 'New Campaign', got %v", created["name"])
	}
	if created["status"] != "draft" {
		t.Errorf("Expected default status 'draft', got %v", created["status"])
	}
	if created["category"] != "public" {
		t.Errorf("Expected default category 'public', got %v", created["category"])
	}

	// Verify it was actually inserted
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM firmware_campaigns WHERE name=?", "New Campaign").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 campaign in DB, got %d", count)
	}
}

// Test handleCreateCampaign - Validation errors
func TestHandleCreateCampaign_ValidationErrors(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	tests := []struct {
		name       string
		campaign   map[string]interface{}
		expectCode int
	}{
		{
			name:       "Missing name",
			campaign:   map[string]interface{}{"version": "v1.0.0"},
			expectCode: 400,
		},
		{
			name:       "Missing version",
			campaign:   map[string]interface{}{"name": "Test"},
			expectCode: 400,
		},
		{
			name:       "Invalid body",
			campaign:   nil,
			expectCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.campaign != nil {
				body, _ = json.Marshal(tt.campaign)
			} else {
				body = []byte("invalid json")
			}

			req := httptest.NewRequest("POST", "/api/firmware/campaigns", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.CreateCampaign(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

// Test handleUpdateCampaign - Success
func TestHandleUpdateCampaign_Success(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Original", "v1.0.0", "public", "draft")

	update := map[string]interface{}{
		"name":    "Updated Campaign",
		"version": "v1.1.0",
		"status":  "active",
		"notes":   "Updated notes",
	}

	body, _ := json.Marshal(update)
	req := httptest.NewRequest("PUT", "/api/firmware/campaigns/FW-001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateCampaign(w, req, "FW-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify the update
	var name, version string
	testDB.QueryRow("SELECT name, version FROM firmware_campaigns WHERE id=?", "FW-001").Scan(&name, &version)
	if name != "Updated Campaign" {
		t.Errorf("Expected name 'Updated Campaign', got %s", name)
	}
	if version != "v1.1.0" {
		t.Errorf("Expected version 'v1.1.0', got %s", version)
	}
}

// Test handleLaunchCampaign - Success
func TestHandleLaunchCampaign_Success(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Launch Test", "v1.0.0", "public", "draft")
	insertTestFirmwareDevice(t, testDB, "SN-001", "IPN-001", "active")
	insertTestFirmwareDevice(t, testDB, "SN-002", "IPN-001", "active")
	insertTestFirmwareDevice(t, testDB, "SN-003", "IPN-002", "inactive")

	req := httptest.NewRequest("POST", "/api/firmware/campaigns/FW-001/launch", nil)
	w := httptest.NewRecorder()

	h.LaunchCampaign(w, req, "FW-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})
	devicesAdded := int(result["devices_added"].(float64))

	// Should add 2 active devices (SN-001, SN-002)
	if devicesAdded != 2 {
		t.Errorf("Expected 2 devices added, got %d", devicesAdded)
	}

	// Verify campaign status changed to active
	var status string
	testDB.QueryRow("SELECT status FROM firmware_campaigns WHERE id=?", "FW-001").Scan(&status)
	if status != "active" {
		t.Errorf("Expected status 'active', got %s", status)
	}

	// Verify devices were added to campaign_devices
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=?", "FW-001").Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 campaign devices, got %d", count)
	}
}

// Test handleCampaignProgress
func TestHandleCampaignProgress(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Progress Test", "v1.0.0", "public", "active")
	insertCampaignDevice(t, testDB, "FW-001", "SN-001", "pending")
	insertCampaignDevice(t, testDB, "FW-001", "SN-002", "sent")
	insertCampaignDevice(t, testDB, "FW-001", "SN-003", "updated")
	insertCampaignDevice(t, testDB, "FW-001", "SN-004", "failed")

	req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-001/progress", nil)
	w := httptest.NewRecorder()

	h.CampaignProgress(w, req, "FW-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	progress := resp.Data.(map[string]interface{})
	if int(progress["total"].(float64)) != 4 {
		t.Errorf("Expected total 4, got %v", progress["total"])
	}
	if int(progress["pending"].(float64)) != 1 {
		t.Errorf("Expected pending 1, got %v", progress["pending"])
	}
	if int(progress["sent"].(float64)) != 1 {
		t.Errorf("Expected sent 1, got %v", progress["sent"])
	}
	if int(progress["updated"].(float64)) != 1 {
		t.Errorf("Expected updated 1, got %v", progress["updated"])
	}
	if int(progress["failed"].(float64)) != 1 {
		t.Errorf("Expected failed 1, got %v", progress["failed"])
	}
}

// Test handleMarkCampaignDevice - Success
func TestHandleMarkCampaignDevice_Success(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Mark Test", "v1.0.0", "public", "active")
	insertCampaignDevice(t, testDB, "FW-001", "SN-001", "pending")

	tests := []struct {
		name   string
		status string
	}{
		{"Mark as updated", "updated"},
		{"Mark as failed", "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset device status
			testDB.Exec("UPDATE campaign_devices SET status='pending' WHERE campaign_id=? AND serial_number=?", "FW-001", "SN-001")

			body, _ := json.Marshal(map[string]string{"status": tt.status})
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/firmware/campaigns/FW-001/devices/SN-001"), bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.MarkCampaignDevice(w, req, "FW-001", "SN-001")

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Verify status was updated
			var status string
			testDB.QueryRow("SELECT status FROM campaign_devices WHERE campaign_id=? AND serial_number=?", "FW-001", "SN-001").Scan(&status)
			if status != tt.status {
				t.Errorf("Expected status '%s', got %s", tt.status, status)
			}
		})
	}
}

// Test handleMarkCampaignDevice - Validation errors
func TestHandleMarkCampaignDevice_ValidationErrors(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Validation Test", "v1.0.0", "public", "active")
	insertCampaignDevice(t, testDB, "FW-001", "SN-001", "pending")

	tests := []struct {
		name       string
		body       map[string]string
		expectCode int
	}{
		{
			name:       "Invalid status",
			body:       map[string]string{"status": "invalid"},
			expectCode: 400,
		},
		{
			name:       "Missing status",
			body:       map[string]string{},
			expectCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("PUT", "/api/firmware/campaigns/FW-001/devices/SN-001", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.MarkCampaignDevice(w, req, "FW-001", "SN-001")

			if w.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, w.Code)
			}
		})
	}
}

// Test handleMarkCampaignDevice - Device not found
func TestHandleMarkCampaignDevice_NotFound(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Not Found Test", "v1.0.0", "public", "active")

	body, _ := json.Marshal(map[string]string{"status": "updated"})
	req := httptest.NewRequest("PUT", "/api/firmware/campaigns/FW-001/devices/SN-999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.MarkCampaignDevice(w, req, "FW-001", "SN-999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test handleCampaignDevices
func TestHandleCampaignDevices(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Devices Test", "v1.0.0", "public", "active")
	insertCampaignDevice(t, testDB, "FW-001", "SN-001", "pending")
	insertCampaignDevice(t, testDB, "FW-001", "SN-002", "updated")
	insertCampaignDevice(t, testDB, "FW-001", "SN-003", "failed")

	req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-001/devices", nil)
	w := httptest.NewRecorder()

	h.CampaignDevices(w, req, "FW-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	devices, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(devices) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(devices))
	}
}

// Test handleCampaignDevices - Empty
func TestHandleCampaignDevices_Empty(t *testing.T) {
	testDB, h := setupFirmwareTestDB(t)
	defer testDB.Close()

	insertTestCampaign(t, testDB, "FW-001", "Empty Devices Test", "v1.0.0", "public", "draft")

	req := httptest.NewRequest("GET", "/api/firmware/campaigns/FW-001/devices", nil)
	w := httptest.NewRecorder()

	h.CampaignDevices(w, req, "FW-001")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	devices, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}

	if len(devices) != 0 {
		t.Errorf("Expected empty array, got %d devices", len(devices))
	}
}
