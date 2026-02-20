package main

import (
	"encoding/json"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"
)

func enableEmailConfig(t *testing.T) {
	t.Helper()
	db.Exec("INSERT OR REPLACE INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, from_name, enabled) VALUES (1, 'smtp.test.com', 587, 'user@test.com', 'pass', 'from@test.com', 'ZRP', 1)")
}

func mockSMTP() func() {
	orig := SMTPSendFunc
	SMTPSendFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return nil
	}
	return func() { SMTPSendFunc = orig }
}

// --- Email Subscription Tests ---

func TestGetEmailSubscriptions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	req := authedRequest("GET", "/api/v1/email/subscriptions", "", cookie)
	w := httptest.NewRecorder()
	handleGetEmailSubscriptions(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data map[string]bool `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	// All should default to true
	for _, et := range EmailEventTypes {
		if !resp.Data[et] {
			t.Errorf("expected %s to be true by default", et)
		}
	}
}

func TestUpdateEmailSubscriptions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	body := `{"eco_approved": false, "ncr_created": true}`
	req := authedRequest("PUT", "/api/v1/email/subscriptions", body, cookie)
	w := httptest.NewRecorder()
	handleUpdateEmailSubscriptions(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data map[string]bool `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data["eco_approved"] != false {
		t.Error("expected eco_approved to be false")
	}
	if resp.Data["ncr_created"] != true {
		t.Error("expected ncr_created to be true")
	}
}

func TestIsUserSubscribed_Default(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	if !isUserSubscribed("admin", "eco_approved") {
		t.Error("expected default subscription to be true")
	}
}

func TestIsUserSubscribed_Disabled(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	db.Exec("INSERT INTO email_subscriptions (user_id, event_type, enabled) VALUES ('admin', 'eco_approved', 0)")
	if isUserSubscribed("admin", "eco_approved") {
		t.Error("expected subscription to be disabled")
	}
}

// --- Email Trigger Tests ---

func TestEmailOnECOApproved_WithLog(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	restore := mockSMTP()
	defer restore()
	enableEmailConfig(t)

	db.Exec("INSERT INTO ecos (id, title, description, status, priority, created_by, created_at, updated_at) VALUES ('ECO-001', 'Test ECO', 'desc', 'approved', 'normal', 'admin', '2024-01-01', '2024-01-01')")
	db.Exec("UPDATE users SET email='admin@test.com' WHERE username='admin'")

	emailOnECOApproved("ECO-001")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_log WHERE event_type='eco_approved'").Scan(&count)
	if count == 0 {
		t.Error("expected email log entry for eco_approved")
	}
}

func TestEmailOnECOApproved_UnsubscribedUser(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	restore := mockSMTP()
	defer restore()
	enableEmailConfig(t)

	db.Exec("INSERT INTO ecos (id, title, description, status, priority, created_by, created_at, updated_at) VALUES ('ECO-002', 'Test ECO', 'desc', 'approved', 'normal', 'admin', '2024-01-01', '2024-01-01')")
	db.Exec("UPDATE users SET email='admin@test.com' WHERE username='admin'")
	db.Exec("INSERT INTO email_subscriptions (user_id, event_type, enabled) VALUES ('admin', 'eco_approved', 0)")

	emailOnECOApproved("ECO-002")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_log WHERE event_type='eco_approved' AND to_address='admin@test.com'").Scan(&count)
	if count != 0 {
		t.Error("expected no email when user is unsubscribed")
	}
}

func TestEmailOnECOImplemented(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	restore := mockSMTP()
	defer restore()
	enableEmailConfig(t)

	db.Exec("INSERT INTO ecos (id, title, description, status, priority, affected_ipns, created_by, created_at, updated_at) VALUES ('ECO-003', 'Impl ECO', 'desc', 'implemented', 'high', 'IPN-001,IPN-002', 'admin', '2024-01-01', '2024-01-01')")

	emailOnECOImplemented("ECO-003")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_log WHERE event_type='eco_implemented'").Scan(&count)
	if count == 0 {
		t.Error("expected email log entry for eco_implemented")
	}
}

func TestEmailOnLowStock_WithLog(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	restore := mockSMTP()
	defer restore()
	enableEmailConfig(t)

	// Use seed data IPN or insert one
	db.Exec("INSERT OR REPLACE INTO inventory (ipn, qty_on_hand, reorder_point, reorder_qty) VALUES ('TEST-001', 5, 10, 20)")

	emailOnLowStock("TEST-001")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_log WHERE event_type='low_stock'").Scan(&count)
	if count == 0 {
		t.Error("expected email log entry for low_stock")
	}
}

func TestEmailOnOverdueWorkOrder(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	restore := mockSMTP()
	defer restore()
	enableEmailConfig(t)

	db.Exec("INSERT INTO work_orders (id, assembly_ipn, qty, status, priority, due_date, created_at) VALUES ('WO-TEST', 'IPN-001', 1, 'open', 'high', '2020-01-01', '2020-01-01')")

	emailOnOverdueWorkOrder("WO-TEST")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_log WHERE event_type='overdue_work_order'").Scan(&count)
	if count == 0 {
		t.Error("expected email log entry for overdue_work_order")
	}
}

func TestEmailOnPOReceived(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	restore := mockSMTP()
	defer restore()
	enableEmailConfig(t)

	db.Exec("INSERT INTO purchase_orders (id, vendor_id, status, created_at, created_by) VALUES ('PO-TEST', 'V-001', 'received', '2024-01-01', 'admin')")
	db.Exec("UPDATE users SET email='admin@test.com' WHERE username='admin'")

	emailOnPOReceived("PO-TEST")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_log WHERE event_type='po_received'").Scan(&count)
	if count == 0 {
		t.Error("expected email log entry for po_received")
	}
}

func TestEmailOnNCRCreated(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	restore := mockSMTP()
	defer restore()
	enableEmailConfig(t)

	emailOnNCRCreated("NCR-TEST", "Test Defect")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_log WHERE event_type='ncr_created'").Scan(&count)
	if count == 0 {
		t.Error("expected email log entry for ncr_created")
	}
}

func TestEmailLogIncludesEventType(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	restore := mockSMTP()
	defer restore()
	enableEmailConfig(t)

	sendEmailWithEvent("test@test.com", "Test", "Body", "eco_approved")

	cookie := loginAdmin(t)
	req := authedRequest("GET", "/api/v1/email-log", "", cookie)
	w := httptest.NewRecorder()
	handleListEmailLog(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "eco_approved") {
		t.Error("expected email log to contain event_type")
	}
}

func TestSendEventEmail_SkipsUnsubscribed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	restore := mockSMTP()
	defer restore()
	enableEmailConfig(t)

	db.Exec("INSERT INTO email_subscriptions (user_id, event_type, enabled) VALUES ('testuser', 'eco_approved', 0)")

	err := sendEventEmail("test@test.com", "Test", "Body", "eco_approved", "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM email_log").Scan(&count)
	if count != 0 {
		t.Error("expected no email sent for unsubscribed user")
	}
}

func TestEmailSubscriptionsAPI_Roundtrip(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Update
	body := `{"eco_approved": false, "low_stock": false, "ncr_created": true}`
	req := authedRequest("PUT", "/api/v1/email/subscriptions", body, cookie)
	w := httptest.NewRecorder()
	handleUpdateEmailSubscriptions(w, req)
	if w.Code != 200 {
		t.Fatalf("PUT expected 200, got %d", w.Code)
	}

	// Read back
	req2 := authedRequest("GET", "/api/v1/email/subscriptions", "", cookie)
	w2 := httptest.NewRecorder()
	handleGetEmailSubscriptions(w2, req2)
	var resp2 struct {
		Data map[string]bool `json:"data"`
	}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	subs := resp2.Data
	if subs["eco_approved"] != false {
		t.Error("eco_approved should be false")
	}
	if subs["low_stock"] != false {
		t.Error("low_stock should be false")
	}
	if subs["ncr_created"] != true {
		t.Error("ncr_created should be true")
	}
	// Untouched ones should default true
	if subs["po_received"] != true {
		t.Error("po_received should default to true")
	}
}
