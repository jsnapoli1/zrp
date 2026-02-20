package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupGeneralSettingsTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create app_settings table
	_, err = testDB.Exec(`
		CREATE TABLE app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create app_settings table: %v", err)
	}

	return testDB
}

func TestHandleGetGeneralSettings_Defaults(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/settings/general", nil)
	w := httptest.NewRecorder()

	handleGetGeneralSettings(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var settings GeneralSettings
	if err := json.NewDecoder(w.Body).Decode(&settings); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify defaults are applied when DB is empty
	if settings.AppName == "" {
		t.Error("Expected app_name to have a default value")
	}
	if settings.Currency == "" {
		t.Error("Expected currency to have a default value")
	}
	if settings.DateFormat == "" {
		t.Error("Expected date_format to have a default value")
	}
}

func TestHandleGetGeneralSettings_WithStoredValues(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	// Insert custom settings
	db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?)", "general_app_name", "Custom PLM")
	db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?)", "general_company_name", "Acme Corp")
	db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?)", "general_company_address", "123 Main St")
	db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?)", "general_currency", "EUR")
	db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?)", "general_date_format", "DD/MM/YYYY")

	req := httptest.NewRequest("GET", "/api/settings/general", nil)
	w := httptest.NewRecorder()

	handleGetGeneralSettings(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var settings GeneralSettings
	if err := json.NewDecoder(w.Body).Decode(&settings); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if settings.AppName != "Custom PLM" {
		t.Errorf("Expected app_name 'Custom PLM', got %s", settings.AppName)
	}
	if settings.CompanyName != "Acme Corp" {
		t.Errorf("Expected company_name 'Acme Corp', got %s", settings.CompanyName)
	}
	if settings.CompanyAddress != "123 Main St" {
		t.Errorf("Expected company_address '123 Main St', got %s", settings.CompanyAddress)
	}
	if settings.Currency != "EUR" {
		t.Errorf("Expected currency 'EUR', got %s", settings.Currency)
	}
	if settings.DateFormat != "DD/MM/YYYY" {
		t.Errorf("Expected date_format 'DD/MM/YYYY', got %s", settings.DateFormat)
	}
}

func TestHandlePutGeneralSettings_CreateNew(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	newSettings := GeneralSettings{
		AppName:        "New PLM",
		CompanyName:    "Test Inc",
		CompanyAddress: "456 Test Ave",
		Currency:       "GBP",
		DateFormat:     "MM/DD/YYYY",
	}
	body, _ := json.Marshal(newSettings)

	req := httptest.NewRequest("PUT", "/api/settings/general", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlePutGeneralSettings(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp GeneralSettings
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response matches input
	if resp.AppName != newSettings.AppName {
		t.Errorf("Expected app_name %s, got %s", newSettings.AppName, resp.AppName)
	}

	// Verify settings were stored in DB
	var storedAppName string
	err := db.QueryRow("SELECT value FROM app_settings WHERE key = ?", "general_app_name").Scan(&storedAppName)
	if err != nil {
		t.Fatalf("Failed to query stored settings: %v", err)
	}
	if storedAppName != "New PLM" {
		t.Errorf("Expected stored app_name 'New PLM', got %s", storedAppName)
	}
}

func TestHandlePutGeneralSettings_UpdateExisting(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	// Insert initial settings
	db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?)", "general_app_name", "Old Name")
	db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?)", "general_currency", "USD")

	// Update settings
	newSettings := GeneralSettings{
		AppName:        "Updated Name",
		CompanyName:    "Updated Corp",
		CompanyAddress: "Updated Address",
		Currency:       "CAD",
		DateFormat:     "YYYY-MM-DD",
	}
	body, _ := json.Marshal(newSettings)

	req := httptest.NewRequest("PUT", "/api/settings/general", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlePutGeneralSettings(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify updates were applied
	var storedAppName string
	err := db.QueryRow("SELECT value FROM app_settings WHERE key = ?", "general_app_name").Scan(&storedAppName)
	if err != nil {
		t.Fatalf("Failed to query updated settings: %v", err)
	}
	if storedAppName != "Updated Name" {
		t.Errorf("Expected updated app_name 'Updated Name', got %s", storedAppName)
	}

	// Verify count - should still be 5 settings (upsert, not insert new)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM app_settings WHERE key LIKE 'general_%'").Scan(&count)
	if count != 5 {
		t.Errorf("Expected 5 settings after update, got %d", count)
	}
}

func TestHandlePutGeneralSettings_InvalidJSON(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("PUT", "/api/settings/general", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlePutGeneralSettings(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandlePutGeneralSettings_EmptyValues(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	// Test that empty strings are allowed
	emptySettings := GeneralSettings{
		AppName:        "",
		CompanyName:    "",
		CompanyAddress: "",
		Currency:       "",
		DateFormat:     "",
	}
	body, _ := json.Marshal(emptySettings)

	req := httptest.NewRequest("PUT", "/api/settings/general", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlePutGeneralSettings(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for empty values, got %d", w.Code)
	}

	// Verify empty values were stored
	var storedAppName string
	err := db.QueryRow("SELECT value FROM app_settings WHERE key = ?", "general_app_name").Scan(&storedAppName)
	if err != nil {
		t.Fatalf("Failed to query settings: %v", err)
	}
	if storedAppName != "" {
		t.Errorf("Expected empty app_name, got %s", storedAppName)
	}
}

func TestHandlePutGeneralSettings_SpecialCharacters(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	specialSettings := GeneralSettings{
		AppName:        "PLM \"Super\" System",
		CompanyName:    "O'Reilly & Sons",
		CompanyAddress: "123 Main St\nSuite 456\nCity, ST 12345",
		Currency:       "€ EUR",
		DateFormat:     "DD/MM/YYYY",
	}
	body, _ := json.Marshal(specialSettings)

	req := httptest.NewRequest("PUT", "/api/settings/general", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlePutGeneralSettings(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp GeneralSettings
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.AppName != specialSettings.AppName {
		t.Errorf("Special characters not preserved in app_name")
	}
	if resp.CompanyName != specialSettings.CompanyName {
		t.Errorf("Special characters not preserved in company_name")
	}
}

func TestHandlePutGeneralSettings_LongValues(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	longSettings := GeneralSettings{
		AppName:        strings.Repeat("A", 1000),
		CompanyName:    strings.Repeat("B", 1000),
		CompanyAddress: strings.Repeat("C", 5000),
		Currency:       "USD",
		DateFormat:     "YYYY-MM-DD",
	}
	body, _ := json.Marshal(longSettings)

	req := httptest.NewRequest("PUT", "/api/settings/general", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlePutGeneralSettings(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for long values, got %d", w.Code)
	}

	// Verify long values were stored correctly
	var storedAddress string
	err := db.QueryRow("SELECT value FROM app_settings WHERE key = ?", "general_company_address").Scan(&storedAddress)
	if err != nil {
		t.Fatalf("Failed to query settings: %v", err)
	}
	if len(storedAddress) != 5000 {
		t.Errorf("Expected address length 5000, got %d", len(storedAddress))
	}
}

func TestHandleGetGeneralSettings_PartialSettings(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	// Only set some settings, others should use defaults
	db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?)", "general_company_name", "Partial Corp")
	db.Exec("INSERT INTO app_settings (key, value) VALUES (?, ?)", "general_currency", "JPY")

	req := httptest.NewRequest("GET", "/api/settings/general", nil)
	w := httptest.NewRecorder()

	handleGetGeneralSettings(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var settings GeneralSettings
	json.NewDecoder(w.Body).Decode(&settings)

	// Should have stored values
	if settings.CompanyName != "Partial Corp" {
		t.Errorf("Expected company_name 'Partial Corp', got %s", settings.CompanyName)
	}
	if settings.Currency != "JPY" {
		t.Errorf("Expected currency 'JPY', got %s", settings.Currency)
	}

	// Should have defaults for others
	if settings.AppName == "" {
		t.Error("Expected app_name to have a default value")
	}
	if settings.DateFormat == "" {
		t.Error("Expected date_format to have a default value")
	}
}

func TestHandlePutGeneralSettings_Idempotency(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	settings := GeneralSettings{
		AppName:        "Test PLM",
		CompanyName:    "Test Corp",
		CompanyAddress: "Test Address",
		Currency:       "USD",
		DateFormat:     "YYYY-MM-DD",
	}
	body, _ := json.Marshal(settings)

	// First update
	req1 := httptest.NewRequest("PUT", "/api/settings/general", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handlePutGeneralSettings(w1, req1)

	// Second update with same data
	req2 := httptest.NewRequest("PUT", "/api/settings/general", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handlePutGeneralSettings(w2, req2)

	if w1.Code != w2.Code {
		t.Errorf("Idempotency failed: different status codes %d vs %d", w1.Code, w2.Code)
	}

	// Verify no duplicate entries
	var count int
	db.QueryRow("SELECT COUNT(*) FROM app_settings WHERE key LIKE 'general_%'").Scan(&count)
	if count != 5 {
		t.Errorf("Expected 5 unique settings, got %d (duplicates created)", count)
	}
}

func TestGeneralSettingsDefaults_Coverage(t *testing.T) {
	// Verify all defaults are properly defined
	expectedDefaults := map[string]string{
		"app_name":        "ZRP",
		"company_name":    "",
		"company_address": "",
		"currency":        "USD",
		"date_format":     "YYYY-MM-DD",
	}

	for key, expectedVal := range expectedDefaults {
		actualVal, exists := generalSettingsDefaults[key]
		if !exists {
			t.Errorf("Default not defined for key: %s", key)
		}
		if actualVal != expectedVal {
			t.Errorf("Default for %s: expected %q, got %q", key, expectedVal, actualVal)
		}
	}

	// Verify all keys are in the keys list
	for key := range generalSettingsDefaults {
		found := false
		for _, k := range generalSettingsKeys {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Key %s exists in defaults but not in keys list", key)
		}
	}
}

func TestHandlePutGeneralSettings_UnicodeSupport(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupGeneralSettingsTestDB(t)
	defer db.Close()

	unicodeSettings := GeneralSettings{
		AppName:        "製品ライフサイクル管理",
		CompanyName:    "Société Française",
		CompanyAddress: "Straße 123, München",
		Currency:       "¥ JPY",
		DateFormat:     "YYYY年MM月DD日",
	}
	body, _ := json.Marshal(unicodeSettings)

	req := httptest.NewRequest("PUT", "/api/settings/general", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlePutGeneralSettings(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp GeneralSettings
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.AppName != unicodeSettings.AppName {
		t.Errorf("Unicode not preserved in app_name: expected %s, got %s", unicodeSettings.AppName, resp.AppName)
	}
}
