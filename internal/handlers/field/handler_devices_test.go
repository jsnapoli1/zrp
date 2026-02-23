package field_test

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"zrp/internal/handlers/field"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

func setupDevicesTestDB(t *testing.T) (*sql.DB, *field.Handler) {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create devices table
	_, err = testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			ipn TEXT NOT NULL,
			firmware_version TEXT,
			customer TEXT,
			location TEXT,
			status TEXT DEFAULT 'active',
			install_date TEXT,
			last_seen TEXT,
			notes TEXT,
			created_at TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	// Create test_records table for device history
	_, err = testDB.Exec(`
		CREATE TABLE test_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			serial_number TEXT NOT NULL,
			ipn TEXT NOT NULL,
			firmware_version TEXT,
			test_type TEXT,
			result TEXT NOT NULL,
			measurements TEXT,
			notes TEXT,
			tested_by TEXT NOT NULL,
			tested_at TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test_records table: %v", err)
	}

	// Create campaign_devices table for device history
	_, err = testDB.Exec(`
		CREATE TABLE campaign_devices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			campaign_id TEXT NOT NULL,
			serial_number TEXT NOT NULL,
			status TEXT NOT NULL,
			updated_at TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create campaign_devices table: %v", err)
	}

	// Create audit_log table for audit tracking
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

	// Create changes table for change tracking
	_, err = testDB.Exec(`
		CREATE TABLE changes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			action TEXT NOT NULL,
			old_value TEXT,
			new_value TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create changes table: %v", err)
	}

	h := &field.Handler{
		DB:  testDB,
		Hub: nil,
		NextIDFunc: func(prefix, table string, digits int) string {
			return prefix + "-0001"
		},
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

func createTestDevice(t *testing.T, db *sql.DB, serial, ipn, fwVersion, customer, location, status, installDate, notes string) {
	t.Helper()
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec(`
		INSERT INTO devices (serial_number, ipn, firmware_version, customer, location, status, install_date, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, serial, ipn, fwVersion, customer, location, status, installDate, notes, now)
	if err != nil {
		t.Fatalf("Failed to create test device: %v", err)
	}
}

func TestHandleListDevices(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	// Create test devices
	createTestDevice(t, testDB, "DEV001", "IPN-100", "v1.0.0", "Acme Corp", "Building A", "active", "2024-01-15", "Test device")
	createTestDevice(t, testDB, "DEV002", "IPN-200", "v1.1.0", "Widget Inc", "Lab 3", "inactive", "2024-02-20", "")

	req := httptest.NewRequest("GET", "/api/v1/devices", nil)
	w := httptest.NewRecorder()

	h.ListDevices(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	devicesJSON, _ := json.Marshal(resp.Data)
	var devices []models.Device
	if err := json.Unmarshal(devicesJSON, &devices); err != nil {
		t.Fatalf("Failed to unmarshal devices: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(devices))
	}

	// Verify first device
	if devices[0].SerialNumber != "DEV001" {
		t.Errorf("Expected serial DEV001, got %s", devices[0].SerialNumber)
	}
	if devices[0].IPN != "IPN-100" {
		t.Errorf("Expected IPN IPN-100, got %s", devices[0].IPN)
	}
	if devices[0].FirmwareVersion != "v1.0.0" {
		t.Errorf("Expected firmware v1.0.0, got %s", devices[0].FirmwareVersion)
	}
	if devices[0].Status != "active" {
		t.Errorf("Expected status active, got %s", devices[0].Status)
	}
}

func TestHandleListDevices_Empty(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("GET", "/api/v1/devices", nil)
	w := httptest.NewRecorder()

	h.ListDevices(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	devicesJSON, _ := json.Marshal(resp.Data)
	var devices []models.Device
	if err := json.Unmarshal(devicesJSON, &devices); err != nil {
		t.Fatalf("Failed to unmarshal devices: %v", err)
	}

	if len(devices) != 0 {
		t.Errorf("Expected 0 devices, got %d", len(devices))
	}
}

func TestHandleGetDevice(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	createTestDevice(t, testDB, "DEV001", "IPN-100", "v1.0.0", "Acme Corp", "Building A", "active", "2024-01-15", "Test device")

	req := httptest.NewRequest("GET", "/api/v1/devices/DEV001", nil)
	w := httptest.NewRecorder()

	h.GetDevice(w, req, "DEV001")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	deviceJSON, _ := json.Marshal(resp.Data)
	var device models.Device
	if err := json.Unmarshal(deviceJSON, &device); err != nil {
		t.Fatalf("Failed to unmarshal device: %v", err)
	}

	if device.SerialNumber != "DEV001" {
		t.Errorf("Expected serial DEV001, got %s", device.SerialNumber)
	}
	if device.Customer != "Acme Corp" {
		t.Errorf("Expected customer Acme Corp, got %s", device.Customer)
	}
	if device.Location != "Building A" {
		t.Errorf("Expected location Building A, got %s", device.Location)
	}
}

func TestHandleGetDevice_NotFound(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("GET", "/api/v1/devices/NONEXISTENT", nil)
	w := httptest.NewRecorder()

	h.GetDevice(w, req, "NONEXISTENT")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleCreateDevice(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	device := models.Device{
		SerialNumber:    "DEV001",
		IPN:             "IPN-100",
		FirmwareVersion: "v1.0.0",
		Customer:        "Acme Corp",
		Location:        "Building A",
		Status:          "active",
		InstallDate:     "2024-01-15",
		Notes:           "Test device",
	}

	body, _ := json.Marshal(device)
	req := httptest.NewRequest("POST", "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateDevice(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	deviceJSON, _ := json.Marshal(resp.Data)
	var created models.Device
	if err := json.Unmarshal(deviceJSON, &created); err != nil {
		t.Fatalf("Failed to unmarshal device: %v", err)
	}

	if created.SerialNumber != "DEV001" {
		t.Errorf("Expected serial DEV001, got %s", created.SerialNumber)
	}

	// Verify device was actually created in DB
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM devices WHERE serial_number = ?", "DEV001").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 device in DB, got %d", count)
	}
}

func TestHandleCreateDevice_DefaultStatus(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	device := models.Device{
		SerialNumber: "DEV001",
		IPN:          "IPN-100",
		// Status not provided - should default to "active"
	}

	body, _ := json.Marshal(device)
	req := httptest.NewRequest("POST", "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateDevice(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	deviceJSON, _ := json.Marshal(resp.Data)
	var created models.Device
	if err := json.Unmarshal(deviceJSON, &created); err != nil {
		t.Fatalf("Failed to unmarshal device: %v", err)
	}

	if created.Status != "active" {
		t.Errorf("Expected default status 'active', got %s", created.Status)
	}
}

func TestHandleCreateDevice_MissingSerialNumber(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	device := models.Device{
		IPN: "IPN-100",
		// Missing SerialNumber
	}

	body, _ := json.Marshal(device)
	req := httptest.NewRequest("POST", "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateDevice(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateDevice_MissingIPN(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	device := models.Device{
		SerialNumber: "DEV001",
		// Missing IPN
	}

	body, _ := json.Marshal(device)
	req := httptest.NewRequest("POST", "/api/v1/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateDevice(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateDevice_InvalidJSON(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("POST", "/api/v1/devices", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateDevice(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleUpdateDevice(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	createTestDevice(t, testDB, "DEV001", "IPN-100", "v1.0.0", "Acme Corp", "Building A", "active", "2024-01-15", "Test device")

	updated := models.Device{
		IPN:             "IPN-200",
		FirmwareVersion: "v2.0.0",
		Customer:        "Widget Inc",
		Location:        "Building B",
		Status:          "inactive",
		InstallDate:     "2024-02-20",
		Notes:           "Updated notes",
	}

	body, _ := json.Marshal(updated)
	req := httptest.NewRequest("PUT", "/api/v1/devices/DEV001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdateDevice(w, req, "DEV001")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	deviceJSON, _ := json.Marshal(resp.Data)
	var device models.Device
	if err := json.Unmarshal(deviceJSON, &device); err != nil {
		t.Fatalf("Failed to unmarshal device: %v", err)
	}

	if device.IPN != "IPN-200" {
		t.Errorf("Expected IPN IPN-200, got %s", device.IPN)
	}
	if device.FirmwareVersion != "v2.0.0" {
		t.Errorf("Expected firmware v2.0.0, got %s", device.FirmwareVersion)
	}
	if device.Customer != "Widget Inc" {
		t.Errorf("Expected customer Widget Inc, got %s", device.Customer)
	}
}

func TestHandleExportDevices(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	createTestDevice(t, testDB, "DEV001", "IPN-100", "v1.0.0", "Acme Corp", "Building A", "active", "2024-01-15", "Test device")
	createTestDevice(t, testDB, "DEV002", "IPN-200", "v1.1.0", "Widget Inc", "Lab 3", "inactive", "2024-02-20", "")

	req := httptest.NewRequest("GET", "/api/v1/devices/export", nil)
	w := httptest.NewRecorder()

	h.ExportDevices(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type text/csv, got %s", contentType)
	}

	contentDisposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "devices.csv") {
		t.Errorf("Expected Content-Disposition to contain devices.csv, got %s", contentDisposition)
	}

	// Parse CSV
	reader := csv.NewReader(w.Body)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) != 3 { // Header + 2 devices
		t.Errorf("Expected 3 CSV records, got %d", len(records))
	}

	// Verify header
	if records[0][0] != "serial_number" {
		t.Errorf("Expected first header to be serial_number, got %s", records[0][0])
	}

	// Verify first device
	if records[1][0] != "DEV001" {
		t.Errorf("Expected first device serial DEV001, got %s", records[1][0])
	}
}

func TestHandleImportDevices(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	// Create CSV content
	csvContent := `serial_number,ipn,firmware_version,customer,location,status,install_date,notes
DEV001,IPN-100,v1.0.0,Acme Corp,Building A,active,2024-01-15,Test device
DEV002,IPN-200,v1.1.0,Widget Inc,Lab 3,inactive,2024-02-20,Another device`

	// Create multipart form request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "devices.csv")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte(csvContent))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/devices/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.ImportDevices(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})
	imported := int(result["imported"].(float64))
	if imported != 2 {
		t.Errorf("Expected 2 devices imported, got %d", imported)
	}

	// Verify devices were created
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 devices in DB, got %d", count)
	}
}

func TestHandleImportDevices_SkipInvalidRows(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	// CSV with some invalid rows (missing serial_number or ipn)
	csvContent := `serial_number,ipn,firmware_version,customer,location,status,install_date,notes
DEV001,IPN-100,v1.0.0,Acme Corp,Building A,active,2024-01-15,Valid device
,IPN-200,v1.1.0,Widget Inc,Lab 3,inactive,2024-02-20,Missing serial
DEV003,,v1.2.0,Test Corp,Lab 5,active,2024-03-01,Missing IPN`

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "devices.csv")
	part.Write([]byte(csvContent))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/devices/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	h.ImportDevices(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})
	imported := int(result["imported"].(float64))
	skipped := int(result["skipped"].(float64))

	if imported != 1 {
		t.Errorf("Expected 1 device imported, got %d", imported)
	}
	if skipped != 2 {
		t.Errorf("Expected 2 devices skipped, got %d", skipped)
	}
}

func TestHandleImportDevices_MissingFile(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	req := httptest.NewRequest("POST", "/api/v1/devices/import", nil)
	w := httptest.NewRecorder()

	h.ImportDevices(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleDeviceHistory(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	createTestDevice(t, testDB, "DEV001", "IPN-100", "v1.0.0", "Acme Corp", "Building A", "active", "2024-01-15", "Test device")

	// Add test record
	now := time.Now().Format("2006-01-02 15:04:05")
	testDB.Exec(`INSERT INTO test_records (serial_number, ipn, firmware_version, test_type, result, measurements, notes, tested_by, tested_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"DEV001", "IPN-100", "v1.0.0", "functional", "pass", "{}", "All tests passed", "testuser", now)

	// Add campaign device
	testDB.Exec(`INSERT INTO campaign_devices (campaign_id, serial_number, status, updated_at)
		VALUES (?, ?, ?, ?)`,
		"CAMP001", "DEV001", "updated", now)

	req := httptest.NewRequest("GET", "/api/v1/devices/DEV001/history", nil)
	w := httptest.NewRecorder()

	h.DeviceHistory(w, req, "DEV001")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})
	tests := result["tests"].([]interface{})
	if len(tests) != 1 {
		t.Errorf("Expected 1 test record, got %d", len(tests))
	}

	campaigns := result["campaigns"].([]interface{})
	if len(campaigns) != 1 {
		t.Errorf("Expected 1 campaign, got %d", len(campaigns))
	}
}

func TestHandleDeviceHistory_NoHistory(t *testing.T) {
	testDB, h := setupDevicesTestDB(t)
	defer testDB.Close()

	createTestDevice(t, testDB, "DEV001", "IPN-100", "v1.0.0", "Acme Corp", "Building A", "active", "2024-01-15", "Test device")

	req := httptest.NewRequest("GET", "/api/v1/devices/DEV001/history", nil)
	w := httptest.NewRecorder()

	h.DeviceHistory(w, req, "DEV001")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result := resp.Data.(map[string]interface{})
	// Should return empty arrays, not nil
	if result["tests"] == nil || result["campaigns"] == nil {
		t.Error("Expected empty arrays for tests and campaigns, got nil")
	}
}
