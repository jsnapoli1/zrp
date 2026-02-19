package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogAuditEnhanced(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Test basic audit logging
	err := LogAuditEnhanced(db, LogAuditOptions{
		UserID:    1,
		Username:  "testuser",
		Action:    AuditActionCreate,
		Module:    "part",
		RecordID:  "TEST-001",
		Summary:   "Created test part",
		IPAddress: "192.168.1.100",
		UserAgent: "Test/1.0",
	})

	if err != nil {
		t.Fatalf("Failed to log audit: %v", err)
	}

	// Verify log was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE record_id = 'TEST-001'").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 audit log, got %d", count)
	}

	// Verify fields
	var username, action, module, recordID, ipAddr string
	var userID int
	err = db.QueryRow(`SELECT user_id, username, action, module, record_id, 
		COALESCE(ip_address, '') 
		FROM audit_log WHERE record_id = 'TEST-001'`).
		Scan(&userID, &username, &action, &module, &recordID, &ipAddr)

	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}

	if userID != 1 {
		t.Errorf("Expected user_id 1, got %d", userID)
	}
	if username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", username)
	}
	if action != AuditActionCreate {
		t.Errorf("Expected action CREATE, got '%s'", action)
	}
	if ipAddr != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got '%s'", ipAddr)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedIP     string
	}{
		{
			name:       "Direct connection",
			remoteAddr: "192.168.1.100:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:          "X-Forwarded-For",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1, 10.0.0.1",
			expectedIP:    "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.2",
			expectedIP: "203.0.113.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := GetClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestLogDataExport(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	createTestUser(t, db, 1, "admin", "admin@test.com", "admin")
	cookie := createTestSession(t, db, 1, "admin")

	req := httptest.NewRequest("GET", "/api/parts/export?format=csv", nil)
	req.AddCookie(cookie)

	LogDataExport(db, req, "parts", "CSV", 150)

	// Verify export was logged
	var action, summary string
	err := db.QueryRow(`SELECT action, summary FROM audit_log 
		WHERE module = 'parts' AND action = 'EXPORT'`).Scan(&action, &summary)

	if err != nil {
		t.Fatalf("Failed to find export log: %v", err)
	}

	if action != AuditActionExport {
		t.Errorf("Expected EXPORT action, got %s", action)
	}

	if !strings.Contains(summary, "150") || !strings.Contains(summary, "CSV") {
		t.Errorf("Expected summary to contain count and format, got: %s", summary)
	}
}

func TestAuditRetentionPolicy(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Test default retention
	days := GetAuditRetentionDays(db)
	if days != 365 {
		t.Errorf("Expected default retention of 365 days, got %d", days)
	}

	// Test setting retention
	err := SetAuditRetentionDays(db, 730)
	if err != nil {
		t.Fatalf("Failed to set retention days: %v", err)
	}

	days = GetAuditRetentionDays(db)
	if days != 730 {
		t.Errorf("Expected retention of 730 days, got %d", days)
	}
}

func TestCleanupOldAuditLogs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create old and recent audit logs
	oldDate := time.Now().AddDate(0, 0, -400).Format("2006-01-02 15:04:05")
	recentDate := time.Now().Format("2006-01-02 15:04:05")

	db.Exec("INSERT INTO audit_log (username, action, module, record_id, summary, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		"test", "CREATE", "part", "OLD-001", "Old log", oldDate)
	db.Exec("INSERT INTO audit_log (username, action, module, record_id, summary, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		"test", "CREATE", "part", "NEW-001", "Recent log", recentDate)

	// Cleanup logs older than 365 days
	deleted, err := CleanupOldAuditLogs(db, 365)
	if err != nil {
		t.Fatalf("Failed to cleanup logs: %v", err)
	}

	if deleted == 0 {
		t.Error("Expected at least 1 log to be deleted")
	}

	// Verify old log is gone but recent log remains
	var count int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE record_id = 'OLD-001'").Scan(&count)
	if count != 0 {
		t.Error("Old log should have been deleted")
	}

	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE record_id = 'NEW-001'").Scan(&count)
	if count != 1 {
		t.Error("Recent log should still exist")
	}
}

func TestHandleAuditExport(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create test logs
	for i := 0; i < 3; i++ {
		LogAuditEnhanced(db, LogAuditOptions{
			UserID:   1,
			Username: "testuser",
			Action:   AuditActionCreate,
			Module:   "part",
			RecordID: "EXPORT-" + string(rune('A'+i)),
			Summary:  "Export test " + string(rune('A'+i)),
		})
	}

	req := httptest.NewRequest("GET", "/api/audit/export", nil)
	w := httptest.NewRecorder()

	handleAuditExport(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check CSV headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type text/csv, got %s", contentType)
	}

	// Check CSV content
	body := w.Body.String()
	if !strings.Contains(body, "Username") || !strings.Contains(body, "Action") {
		t.Error("CSV should contain headers")
	}
	if !strings.Contains(body, "testuser") {
		t.Error("CSV should contain audit data")
	}
}

func TestHandleAuditRetention(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Test GET
	req := httptest.NewRequest("GET", "/api/audit/retention", nil)
	w := httptest.NewRecorder()
	handleAuditRetention(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var getResp struct {
		RetentionDays int `json:"retention_days"`
	}
	json.NewDecoder(w.Body).Decode(&getResp)
	if getResp.RetentionDays <= 0 {
		t.Error("Expected positive retention days")
	}

	// Test PUT
	reqBody := strings.NewReader(`{"retention_days": 730}`)
	req = httptest.NewRequest("PUT", "/api/audit/retention", reqBody)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handleAuditRetention(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify it was updated
	days := GetAuditRetentionDays(db)
	if days != 730 {
		t.Errorf("Expected retention to be updated to 730, got %d", days)
	}
}
