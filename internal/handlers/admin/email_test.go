package admin_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"zrp/internal/handlers/admin"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

func TestHandleGetEmailConfig_Default(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/email/config", nil)
	w := httptest.NewRecorder()

	h.HandleGetEmailConfig(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	configData, _ := json.Marshal(resp.Data)
	var config admin.EmailConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	_, err := db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, from_name, enabled)
		VALUES (1, 'smtp.example.com', 587, 'user@example.com', 'secret123', 'noreply@example.com', 'ZRP System', 1)`)
	if err != nil {
		t.Fatalf("Failed to insert config: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/email/config", nil)
	w := httptest.NewRecorder()

	h.HandleGetEmailConfig(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	configData, _ := json.Marshal(resp.Data)
	var config admin.EmailConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

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

	h.HandleUpdateEmailConfig(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify config was saved
	var config admin.EmailConfig
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
	var apiResp models.APIResponse
	json.NewDecoder(w.Body).Decode(&apiResp)
	respData, _ := json.Marshal(apiResp.Data)
	var resp admin.EmailConfig
	json.Unmarshal(respData, &resp)
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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	_, err := db.Exec(`INSERT INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, enabled)
		VALUES (1, 'smtp.old.com', 587, 'old@test.com', 'oldpassword', 'old@test.com', 1)`)
	if err != nil {
		t.Fatalf("Failed to insert config: %v", err)
	}

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

	h.HandleUpdateEmailConfig(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var password string
	db.QueryRow("SELECT smtp_password FROM email_config WHERE id = 1").Scan(&password)
	if password != "oldpassword" {
		t.Errorf("Old password should be preserved when **** sent, got '%s'", password)
	}
}

func TestHandleUpdateEmailConfig_DefaultPort(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

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

	h.HandleUpdateEmailConfig(w, req)

	var port int
	db.QueryRow("SELECT smtp_port FROM email_config WHERE id = 1").Scan(&port)
	if port != 587 {
		t.Errorf("Port should default to 587 when 0 or negative, got %d", port)
	}
}

func TestHandleUpdateEmailConfig_InvalidJSON(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/email/config", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleUpdateEmailConfig(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandleTestEmail_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	h := newTestHandler(db)
	// Override SendEmail to succeed
	h.SendEmail = func(to, subject, body string) error {
		return nil
	}

	reqBody := `{"to": "recipient@test.com"}`
	req := httptest.NewRequest("POST", "/api/email/test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTestEmail(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var apiResp models.APIResponse
	json.NewDecoder(w.Body).Decode(&apiResp)
	respData, _ := json.Marshal(apiResp.Data)
	var resp map[string]string
	json.Unmarshal(respData, &resp)
	if resp["status"] != "sent" {
		t.Errorf("Expected status 'sent', got '%s'", resp["status"])
	}
	if resp["to"] != "recipient@test.com" {
		t.Errorf("Expected to 'recipient@test.com', got '%s'", resp["to"])
	}

	// Verify audit log
	var auditCount int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE action = 'test_email'").Scan(&auditCount)
	if auditCount == 0 {
		t.Error("Test email should create audit log entry")
	}
}

func TestHandleTestEmail_TestEmailField(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	h := newTestHandler(db)
	h.SendEmail = func(to, subject, body string) error {
		return nil
	}

	reqBody := `{"test_email": "alternate@test.com"}`
	req := httptest.NewRequest("POST", "/api/email/test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTestEmail(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var apiResp models.APIResponse
	json.NewDecoder(w.Body).Decode(&apiResp)
	respData, _ := json.Marshal(apiResp.Data)
	var resp map[string]string
	json.Unmarshal(respData, &resp)
	if resp["to"] != "alternate@test.com" {
		t.Errorf("Should support 'test_email' field, got '%s'", resp["to"])
	}
}

func TestHandleTestEmail_MissingRecipient(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{}`
	req := httptest.NewRequest("POST", "/api/email/test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTestEmail(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 when 'to' missing, got %d", w.Code)
	}
}

func TestHandleTestEmail_SendFailure(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	h := newTestHandler(db)
	h.SendEmail = func(to, subject, body string) error {
		return errors.New("test error")
	}

	reqBody := `{"to": "recipient@test.com"}`
	req := httptest.NewRequest("POST", "/api/email/test", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleTestEmail(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500 when send fails, got %d", w.Code)
	}
}

func TestHandleListEmailLog(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	_, err := db.Exec(`INSERT INTO email_log (to_address, subject, body, event_type, status, error, sent_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"user1@test.com", "Test 1", "Body 1", "test_event", "sent", "", "2026-01-01 10:00:00")
	if err != nil {
		t.Fatalf("Failed to insert log entry: %v", err)
	}

	_, err = db.Exec(`INSERT INTO email_log (to_address, subject, body, event_type, status, error, sent_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"user2@test.com", "Test 2", "Body 2", "eco_approved", "failed", "Connection timeout", "2026-01-01 11:00:00")
	if err != nil {
		t.Fatalf("Failed to insert log entry: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/email/log", nil)
	w := httptest.NewRecorder()

	h.HandleListEmailLog(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	logsData, _ := json.Marshal(resp.Data)
	var logs []admin.EmailLogEntry
	if err := json.Unmarshal(logsData, &logs); err != nil {
		t.Fatalf("Failed to unmarshal logs: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(logs))
	}

	// Should be ordered by sent_at DESC
	if logs[0].Subject != "Test 2" {
		t.Errorf("Expected first log to be 'Test 2', got '%s'", logs[0].Subject)
	}
}

func TestHandleListEmailLog_Empty(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/email/log", nil)
	w := httptest.NewRecorder()

	h.HandleListEmailLog(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	logsData, _ := json.Marshal(resp.Data)
	var logs []admin.EmailLogEntry
	if err := json.Unmarshal(logsData, &logs); err != nil {
		t.Fatalf("Failed to unmarshal logs: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected empty array, got %d logs", len(logs))
	}
}

func TestHandleGetEmailSubscriptions_Default(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/email/subscriptions", nil)
	w := httptest.NewRecorder()

	h.HandleGetEmailSubscriptions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	subsData, _ := json.Marshal(resp.Data)
	var subs map[string]bool
	if err := json.Unmarshal(subsData, &subs); err != nil {
		t.Fatalf("Failed to unmarshal subscriptions: %v", err)
	}

	// All event types should default to enabled
	for _, eventType := range admin.EmailEventTypes {
		if !subs[eventType] {
			t.Errorf("Event type '%s' should default to enabled", eventType)
		}
	}
}

func TestHandleGetEmailSubscriptions_Custom(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Create user and session
	cookie := loginUserLocal(t, db, "testuser")

	// Insert custom subscription (disable low_stock for testuser)
	_, err := db.Exec(`INSERT INTO email_subscriptions (user_id, event_type, enabled) VALUES (?, ?, ?)`,
		"testuser", "low_stock", 0)
	if err != nil {
		t.Fatalf("Failed to insert subscription: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/email/subscriptions", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: cookie})
	w := httptest.NewRecorder()

	h.HandleGetEmailSubscriptions(w, req)

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	subsData, _ := json.Marshal(resp.Data)
	var subs map[string]bool
	json.Unmarshal(subsData, &subs)

	if subs["low_stock"] {
		t.Error("low_stock should be disabled for testuser")
	}
	if !subs["eco_approved"] {
		t.Error("eco_approved should still be enabled (default)")
	}
}

func TestHandleUpdateEmailSubscriptions(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	cookie := loginUserLocal(t, db, "testuser")

	reqBody := `{
		"eco_approved": false,
		"low_stock": true,
		"ncr_created": false
	}`
	req := httptest.NewRequest("POST", "/api/email/subscriptions", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: cookie})
	w := httptest.NewRecorder()

	h.HandleUpdateEmailSubscriptions(w, req)

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
