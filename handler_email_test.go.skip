package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupEmailTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create email_config table
	_, err = testDB.Exec(`
		CREATE TABLE email_config (
			id INTEGER PRIMARY KEY DEFAULT 1,
			smtp_host TEXT,
			smtp_port INTEGER DEFAULT 587,
			smtp_user TEXT,
			smtp_password TEXT,
			from_address TEXT,
			from_name TEXT DEFAULT 'ZRP',
			enabled INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create email_config table: %v", err)
	}

	// Create email_log table
	_, err = testDB.Exec(`
		CREATE TABLE email_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			to_address TEXT NOT NULL,
			recipient TEXT DEFAULT '',
			subject TEXT NOT NULL,
			body TEXT,
			event_type TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'sent',
			error TEXT,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create email_log table: %v", err)
	}

	// Create email_subscriptions table
	_, err = testDB.Exec(`
		CREATE TABLE email_subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			UNIQUE(user_id, event_type)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create email_subscriptions table: %v", err)
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

	// Create notifications table (needed for sendNotificationEmail)
	_, err = testDB.Exec(`
		CREATE TABLE notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			severity TEXT DEFAULT 'info',
			title TEXT NOT NULL,
			message TEXT,
			record_id TEXT,
			module TEXT,
			user_id TEXT DEFAULT '',
			emailed INTEGER DEFAULT 0,
			read_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create notifications table: %v", err)
	}

	return testDB
}

func TestHandleGetEmailConfig_Default(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/email/config", nil)
	w := httptest.NewRecorder()

	handleGetEmailConfig(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var config EmailConfig
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return defaults when no config exists
	if config.ID != 1 {
		t.Errorf("Expected ID 1, got %d", config.ID)
	}
	if config.SMTPPort != 587 {
		t.Errorf("Expected default port 587, got %d", config.SMTPPort)
	}
	if config.FromName != "ZRP" {
		t.Errorf("Expected default from_name 'ZRP', got '%s'", config.FromName)
	}
}

func TestHandleGetEmailConfig_Existing(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert config
	_, err := db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, from_name, enabled) 
		VALUES (1, 'smtp.example.com', 587, 'user@example.com', 'secret123', 'noreply@example.com', 'ZRP System', 1)`)
	if err != nil {
		t.Fatalf("Failed to insert config: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/email/config", nil)
	w := httptest.NewRecorder()

	handleGetEmailConfig(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var config EmailConfig
	if err := json.NewDecoder(w.Body).Decode(&config); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if config.SMTPHost != "smtp.example.com" {
		t.Errorf("Expected smtp_host 'smtp.example.com', got '%s'", config.SMTPHost)
	}
	if config.SMTPUser != "user@example.com" {
		t.Errorf("Expected smtp_user 'user@example.com', got '%s'", config.SMTPUser)
	}
	// Password should be masked
	if config.SMTPPassword != "****" {
		t.Errorf("Password should be masked as '****', got '%s'", config.SMTPPassword)
	}
	if config.Enabled != 1 {
		t.Errorf("Expected enabled 1, got %d", config.Enabled)
	}
}

func TestHandleUpdateEmailConfig_New(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	reqBody := `{
		"smtp_host": "smtp.gmail.com",
		"smtp_port": 587,
		"smtp_user": "test@gmail.com",
		"smtp_password": "newpassword123",
		"from_address": "noreply@test.com",
		"from_name": "Test System",
		"enabled": 1
	}`
	req := httptest.NewRequest("POST", "/api/email/config", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateEmailConfig(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify config was saved
	var config EmailConfig
	err := db.QueryRow("SELECT smtp_host, smtp_port, smtp_user, smtp_password, from_address, from_name, enabled FROM email_config WHERE id = 1").
		Scan(&config.SMTPHost, &config.SMTPPort, &config.SMTPUser, &config.SMTPPassword, &config.FromAddress, &config.FromName, &config.Enabled)
	if err != nil {
		t.Fatalf("Failed to query config: %v", err)
	}

	if config.SMTPHost != "smtp.gmail.com" {
		t.Errorf("Expected smtp_host 'smtp.gmail.com', got '%s'", config.SMTPHost)
	}
	if config.SMTPPassword != "newpassword123" {
		t.Errorf("Password should be stored unmasked in DB, got '%s'", config.SMTPPassword)
	}

	// Response should mask password
	var resp EmailConfig
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.SMTPPassword != "****" {
		t.Errorf("Response password should be masked, got '%s'", resp.SMTPPassword)
	}

	// Verify audit log
	var auditCount int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module = 'email_config' AND action = 'updated'").Scan(&auditCount)
	if auditCount == 0 {
		t.Error("Update should create audit log entry")
	}
}

func TestHandleUpdateEmailConfig_PreservePassword(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert existing config
	_, err := db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, enabled) 
		VALUES (1, 'smtp.old.com', 587, 'old@test.com', 'oldpassword', 'old@test.com', 1)`)
	if err != nil {
		t.Fatalf("Failed to insert config: %v", err)
	}

	// Update with masked password (frontend sends **** when password not changed)
	reqBody := `{
		"smtp_host": "smtp.new.com",
		"smtp_port": 587,
		"smtp_user": "new@test.com",
		"smtp_password": "****",
		"from_address": "new@test.com",
		"enabled": 1
	}`
	req := httptest.NewRequest("POST", "/api/email/config", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateEmailConfig(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify old password was preserved
	var password string
	db.QueryRow("SELECT smtp_password FROM email_config WHERE id = 1").Scan(&password)
	if password != "oldpassword" {
		t.Errorf("Old password should be preserved when **** sent, got '%s'", password)
	}
}

func TestHandleUpdateEmailConfig_DefaultPort(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Send config with port 0 (should default to 587)
	reqBody := `{
		"smtp_host": "smtp.test.com",
		"smtp_port": 0,
		"smtp_user": "test@test.com",
		"smtp_password": "pass123",
		"from_address": "test@test.com",
		"enabled": 1
	}`
	req := httptest.NewRequest("POST", "/api/email/config", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateEmailConfig(w, req)

	// Verify port defaulted to 587
	var port int
	db.QueryRow("SELECT smtp_port FROM email_config WHERE id = 1").Scan(&port)
	if port != 587 {
		t.Errorf("Port should default to 587 when 0 or negative, got %d", port)
	}
}

func TestHandleUpdateEmailConfig_InvalidJSON(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	reqBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/email/config", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateEmailConfig(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandleTestEmail_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert enabled email config
	_, err := db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, enabled) 
		VALUES (1, 'smtp.test.com', 587, 'user@test.com', 'pass123', 'noreply@test.com', 1)`)
	if err != nil {
		t.Fatalf("Failed to insert config: %v", err)
	}

	// Mock SMTP send function
	var capturedAddr, capturedFrom string
	var capturedTo []string
	var capturedMsg []byte
	oldSMTPSendFunc := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		capturedAddr = addr
		capturedFrom = from
		capturedTo = to
		capturedMsg = msg
		return nil
	}
	defer func() { SMTPSendFunc = oldSMTPSendFunc }()

	reqBody := `{"to": "recipient@test.com"}`
	req := httptest.NewRequest("POST", "/api/email/test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleTestEmail(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "sent" {
		t.Errorf("Expected status 'sent', got '%s'", resp["status"])
	}
	if resp["to"] != "recipient@test.com" {
		t.Errorf("Expected to 'recipient@test.com', got '%s'", resp["to"])
	}

	// Verify SMTP was called correctly
	if capturedAddr != "smtp.test.com:587" {
		t.Errorf("Expected SMTP addr 'smtp.test.com:587', got '%s'", capturedAddr)
	}
	if capturedFrom != "noreply@test.com" {
		t.Errorf("Expected from 'noreply@test.com', got '%s'", capturedFrom)
	}
	if len(capturedTo) != 1 || capturedTo[0] != "recipient@test.com" {
		t.Errorf("Expected to ['recipient@test.com'], got %v", capturedTo)
	}
	if !strings.Contains(string(capturedMsg), "ZRP Test Email") {
		t.Error("Message should contain 'ZRP Test Email' in subject")
	}

	// Verify audit log
	var auditCount int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE action = 'test_email'").Scan(&auditCount)
	if auditCount == 0 {
		t.Error("Test email should create audit log entry")
	}
}

func TestHandleTestEmail_TestEmailField(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert enabled email config
	db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, enabled) 
		VALUES (1, 'smtp.test.com', 587, 'user@test.com', 'pass123', 'noreply@test.com', 1)`)

	// Mock SMTP
	oldSMTPSendFunc := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		return nil
	}
	defer func() { SMTPSendFunc = oldSMTPSendFunc }()

	// Test with "test_email" field instead of "to"
	reqBody := `{"test_email": "alternate@test.com"}`
	req := httptest.NewRequest("POST", "/api/email/test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleTestEmail(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["to"] != "alternate@test.com" {
		t.Errorf("Should support 'test_email' field, got '%s'", resp["to"])
	}
}

func TestHandleTestEmail_MissingRecipient(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	reqBody := `{}`
	req := httptest.NewRequest("POST", "/api/email/test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleTestEmail(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 when 'to' missing, got %d", w.Code)
	}
}

func TestHandleTestEmail_SendFailure(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert enabled email config
	db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, enabled) 
		VALUES (1, 'smtp.test.com', 587, 'user@test.com', 'pass123', 'noreply@test.com', 1)`)

	// Mock SMTP to return error
	oldSMTPSendFunc := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		return smtp.ServerNotAvailable("test error")
	}
	defer func() { SMTPSendFunc = oldSMTPSendFunc }()

	reqBody := `{"to": "recipient@test.com"}`
	req := httptest.NewRequest("POST", "/api/email/test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleTestEmail(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500 when send fails, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(strings.ToLower(resp["error"]), "send failed") {
		t.Error("Error should mention 'send failed'")
	}
}

func TestHandleListEmailLog(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert test log entries
	_, err := db.Exec(`INSERT INTO email_log (to_address, subject, body, event_type, status, error) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"user1@test.com", "Test 1", "Body 1", "test_event", "sent", "")
	if err != nil {
		t.Fatalf("Failed to insert log entry: %v", err)
	}

	_, err = db.Exec(`INSERT INTO email_log (to_address, subject, body, event_type, status, error) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"user2@test.com", "Test 2", "Body 2", "eco_approved", "failed", "Connection timeout")
	if err != nil {
		t.Fatalf("Failed to insert log entry: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/email/log", nil)
	w := httptest.NewRecorder()

	handleListEmailLog(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var logs []EmailLogEntry
	if err := json.NewDecoder(w.Body).Decode(&logs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(logs))
	}

	// Should be ordered by sent_at DESC (most recent first)
	if logs[0].Subject != "Test 2" {
		t.Errorf("Expected first log to be 'Test 2', got '%s'", logs[0].Subject)
	}
	if logs[0].Status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", logs[0].Status)
	}
	if logs[0].Error != "Connection timeout" {
		t.Errorf("Expected error 'Connection timeout', got '%s'", logs[0].Error)
	}
}

func TestHandleListEmailLog_Empty(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/email/log", nil)
	w := httptest.NewRecorder()

	handleListEmailLog(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var logs []EmailLogEntry
	if err := json.NewDecoder(w.Body).Decode(&logs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected empty array, got %d logs", len(logs))
	}
}

func TestSendEmail_LogsSuccess(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert enabled email config
	db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, enabled) 
		VALUES (1, 'smtp.test.com', 587, 'user@test.com', 'pass123', 'noreply@test.com', 1)`)

	// Mock SMTP
	oldSMTPSendFunc := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		return nil
	}
	defer func() { SMTPSendFunc = oldSMTPSendFunc }()

	err := sendEmail("recipient@test.com", "Test Subject", "Test Body")
	if err != nil {
		t.Errorf("sendEmail should succeed, got error: %v", err)
	}

	// Verify log entry created
	var status string
	db.QueryRow("SELECT status FROM email_log WHERE to_address = ?", "recipient@test.com").Scan(&status)
	if status != "sent" {
		t.Errorf("Expected log status 'sent', got '%s'", status)
	}
}

func TestSendEmail_LogsFailure(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert enabled email config
	db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, enabled) 
		VALUES (1, 'smtp.test.com', 587, 'user@test.com', 'pass123', 'noreply@test.com', 1)`)

	// Mock SMTP to fail
	oldSMTPSendFunc := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		return smtp.ServerNotAvailable("test error")
	}
	defer func() { SMTPSendFunc = oldSMTPSendFunc }()

	err := sendEmail("recipient@test.com", "Test Subject", "Test Body")
	if err == nil {
		t.Error("sendEmail should return error when SMTP fails")
	}

	// Verify log entry created with failure
	var status, errMsg string
	db.QueryRow("SELECT status, error FROM email_log WHERE to_address = ?", "recipient@test.com").Scan(&status, &errMsg)
	if status != "failed" {
		t.Errorf("Expected log status 'failed', got '%s'", status)
	}
	if !strings.Contains(errMsg, "test error") {
		t.Errorf("Error message should contain 'test error', got '%s'", errMsg)
	}
}

func TestHandleGetEmailSubscriptions_Default(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/email/subscriptions", nil)
	w := httptest.NewRecorder()

	handleGetEmailSubscriptions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var subs map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&subs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// All event types should default to enabled
	for _, eventType := range EmailEventTypes {
		if !subs[eventType] {
			t.Errorf("Event type '%s' should default to enabled", eventType)
		}
	}
}

func TestHandleGetEmailSubscriptions_Custom(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert custom subscription (disable low_stock for testuser)
	_, err := db.Exec(`INSERT INTO email_subscriptions (user_id, event_type, enabled) VALUES (?, ?, ?)`,
		"testuser", "low_stock", 0)
	if err != nil {
		t.Fatalf("Failed to insert subscription: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/email/subscriptions", nil)
	w := httptest.NewRecorder()

	handleGetEmailSubscriptions(w, req)

	var subs map[string]bool
	json.NewDecoder(w.Body).Decode(&subs)

	if subs["low_stock"] {
		t.Error("low_stock should be disabled for testuser")
	}
	if !subs["eco_approved"] {
		t.Error("eco_approved should still be enabled (default)")
	}
}

func TestHandleUpdateEmailSubscriptions(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	reqBody := `{
		"eco_approved": false,
		"low_stock": true,
		"ncr_created": false
	}`
	req := httptest.NewRequest("POST", "/api/email/subscriptions", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleUpdateEmailSubscriptions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify subscriptions were saved
	var enabled int
	db.QueryRow("SELECT enabled FROM email_subscriptions WHERE user_id = 'testuser' AND event_type = 'eco_approved'").Scan(&enabled)
	if enabled != 0 {
		t.Error("eco_approved should be disabled")
	}

	db.QueryRow("SELECT enabled FROM email_subscriptions WHERE user_id = 'testuser' AND event_type = 'low_stock'").Scan(&enabled)
	if enabled != 1 {
		t.Error("low_stock should be enabled")
	}

	// Verify audit log
	var auditCount int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module = 'email_subscriptions' AND action = 'updated'").Scan(&auditCount)
	if auditCount == 0 {
		t.Error("Update should create audit log entry")
	}
}

func TestIsUserSubscribed_Default(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// No subscription record = default to subscribed
	subscribed := isUserSubscribed("testuser", "eco_approved")
	if !subscribed {
		t.Error("User should be subscribed by default")
	}
}

func TestIsUserSubscribed_Disabled(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert disabled subscription
	db.Exec(`INSERT INTO email_subscriptions (user_id, event_type, enabled) VALUES (?, ?, ?)`,
		"testuser", "eco_approved", 0)

	subscribed := isUserSubscribed("testuser", "eco_approved")
	if subscribed {
		t.Error("User should not be subscribed when explicitly disabled")
	}
}

func TestIsUserSubscribed_Enabled(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert enabled subscription
	db.Exec(`INSERT INTO email_subscriptions (user_id, event_type, enabled) VALUES (?, ?, ?)`,
		"testuser", "eco_approved", 1)

	subscribed := isUserSubscribed("testuser", "eco_approved")
	if !subscribed {
		t.Error("User should be subscribed when explicitly enabled")
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"test.user@sub.example.com", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
		{"user@domain", false}, // no TLD
	}

	for _, tt := range tests {
		result := isValidEmail(tt.email)
		if result != tt.valid {
			t.Errorf("isValidEmail(%s) = %v, expected %v", tt.email, result, tt.valid)
		}
	}
}

func TestEmailConfigEnabled(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// No config = disabled
	if emailConfigEnabled() {
		t.Error("Should be disabled when no config exists")
	}

	// Insert disabled config
	db.Exec(`INSERT INTO email_config (id, enabled) VALUES (1, 0)`)
	if emailConfigEnabled() {
		t.Error("Should be disabled when enabled=0")
	}

	// Update to enabled
	db.Exec(`UPDATE email_config SET enabled = 1 WHERE id = 1`)
	if !emailConfigEnabled() {
		t.Error("Should be enabled when enabled=1")
	}
}

func TestSendEmailWithEvent_HTMLInjection(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupEmailTestDB(t)
	defer db.Close()

	// Insert enabled email config
	db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, enabled) 
		VALUES (1, 'smtp.test.com', 587, 'user@test.com', 'pass123', 'noreply@test.com', 1)`)

	var capturedMsg []byte
	oldSMTPSendFunc := SMTPSendFunc
	SMTPSendFunc = func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		capturedMsg = msg
		return nil
	}
	defer func() { SMTPSendFunc = oldSMTPSendFunc }()

	// Try to inject HTML/script
	maliciousBody := "Test <script>alert('xss')</script> body"
	err := sendEmailWithEvent("test@test.com", "Test", maliciousBody, "test_event")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Email is sent as text/plain, so HTML should be treated as literal text
	msgStr := string(capturedMsg)
	if !strings.Contains(msgStr, "Content-Type: text/plain") {
		t.Error("Email should be sent as text/plain to prevent HTML injection")
	}
	if !strings.Contains(msgStr, "<script>") {
		t.Error("Script tags should be present as literal text (not executed)")
	}
}
