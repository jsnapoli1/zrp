package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func withUserID(r *http.Request, userID int) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxUserID, userID))
}

func decodeAPIResp(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode API response: %v", err)
	}
	if err := json.Unmarshal(resp.Data, target); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
}

func TestListNotificationTypes(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/notifications/types", nil)
	w := httptest.NewRecorder()
	handleListNotificationTypes(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var types []NotificationTypeInfo
	decodeAPIResp(t, w, &types)
	if len(types) != 8 {
		t.Fatalf("expected 8 notification types, got %d", len(types))
	}

	for _, nt := range types {
		if nt.Type == "low_stock" {
			if !nt.HasThreshold {
				t.Error("low_stock should have threshold")
			}
			if nt.ThresholdDefault == nil || *nt.ThresholdDefault != 10 {
				t.Error("low_stock default threshold should be 10")
			}
		}
	}
}

func TestGetNotificationPreferencesUnauthorized(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/notifications/preferences", nil)
	w := httptest.NewRecorder()
	handleGetNotificationPreferences(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetNotificationPreferencesDefaults(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	initNotificationPrefsTable()

	req := httptest.NewRequest("GET", "/api/v1/notifications/preferences", nil)
	req = withUserID(req, 1) // admin user from seedDB
	w := httptest.NewRecorder()
	handleGetNotificationPreferences(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var prefs []NotificationPreference
	decodeAPIResp(t, w, &prefs)
	if len(prefs) != 8 {
		t.Fatalf("expected 8 default prefs, got %d", len(prefs))
	}

	for _, p := range prefs {
		if !p.Enabled {
			t.Errorf("expected all defaults enabled, %s is disabled", p.Type)
		}
		if p.DeliveryMethod != "in_app" {
			t.Errorf("expected default delivery_method 'in_app', got '%s' for %s", p.DeliveryMethod, p.Type)
		}
	}
}

func TestUpdateNotificationPreferencesBulk(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	initNotificationPrefsTable()

	body := `[{"notification_type":"low_stock","enabled":false,"delivery_method":"email","threshold_value":5},{"notification_type":"overdue_wo","enabled":true,"delivery_method":"both","threshold_value":3}]`
	req := httptest.NewRequest("PUT", "/api/v1/notifications/preferences", strings.NewReader(body))
	req = withUserID(req, 1)
	req.Header.Set("Content-Type", "application/json")
	// Need a session cookie for getUsername
	cookie := loginAdmin(t)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handleUpdateNotificationPreferences(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var prefs []NotificationPreference
	decodeAPIResp(t, w, &prefs)

	for _, p := range prefs {
		if p.Type == "low_stock" {
			if p.Enabled {
				t.Error("low_stock should be disabled")
			}
			if p.DeliveryMethod != "email" {
				t.Errorf("low_stock delivery should be 'email', got '%s'", p.DeliveryMethod)
			}
			if p.ThresholdValue == nil || *p.ThresholdValue != 5 {
				t.Error("low_stock threshold should be 5")
			}
		}
		if p.Type == "overdue_wo" {
			if !p.Enabled {
				t.Error("overdue_wo should be enabled")
			}
			if p.DeliveryMethod != "both" {
				t.Errorf("overdue_wo delivery should be 'both', got '%s'", p.DeliveryMethod)
			}
			if p.ThresholdValue == nil || *p.ThresholdValue != 3 {
				t.Error("overdue_wo threshold should be 3")
			}
		}
	}
}

func TestUpdateSingleNotificationPreference(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	initNotificationPrefsTable()

	body := `{"enabled":false,"delivery_method":"both","threshold_value":null}`
	req := httptest.NewRequest("PUT", "/api/v1/notifications/preferences/open_ncr", strings.NewReader(body))
	req = withUserID(req, 1)
	req.Header.Set("Content-Type", "application/json")
	cookie := loginAdmin(t)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handleUpdateSingleNotificationPreference(w, req, "open_ncr")

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var prefs []NotificationPreference
	decodeAPIResp(t, w, &prefs)

	for _, p := range prefs {
		if p.Type == "open_ncr" {
			if p.Enabled {
				t.Error("open_ncr should be disabled")
			}
			if p.DeliveryMethod != "both" {
				t.Errorf("open_ncr delivery should be 'both', got '%s'", p.DeliveryMethod)
			}
		}
	}
}

func TestUpdateSingleNotificationPreferenceInvalidType(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	initNotificationPrefsTable()

	body := `{"enabled":false,"delivery_method":"in_app"}`
	req := httptest.NewRequest("PUT", "/api/v1/notifications/preferences/invalid_type", strings.NewReader(body))
	req = withUserID(req, 1)
	w := httptest.NewRecorder()
	handleUpdateSingleNotificationPreference(w, req, "invalid_type")

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetUserNotifPrefDefault(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	initNotificationPrefsTable()

	// No preferences set — should return defaults
	enabled, method, threshold := getUserNotifPref(999, "low_stock")
	if !enabled {
		t.Error("default should be enabled")
	}
	if method != "in_app" {
		t.Errorf("default method should be 'in_app', got '%s'", method)
	}
	if threshold != nil {
		t.Error("default threshold should be nil for non-existent user")
	}
}

func TestGetUserNotifPrefCustom(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	initNotificationPrefsTable()

	// Set a custom preference
	db.Exec("INSERT INTO notification_preferences (user_id, notification_type, enabled, delivery_method, threshold_value) VALUES (1, 'low_stock', 0, 'email', 25)")

	enabled, method, threshold := getUserNotifPref(1, "low_stock")
	if enabled {
		t.Error("should be disabled")
	}
	if method != "email" {
		t.Errorf("method should be 'email', got '%s'", method)
	}
	if threshold == nil || *threshold != 25 {
		t.Error("threshold should be 25")
	}
}

func TestGenerateNotificationsFilteredDisabled(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	initNotificationPrefsTable()

	// Insert test inventory data that would trigger low_stock
	db.Exec("INSERT INTO inventory (ipn, description, qty_on_hand, reorder_point) VALUES ('TEST-001', 'Test Part', 2, 10)")

	// Disable low_stock for admin user (id=1)
	ensureDefaultPreferences(1)
	db.Exec("UPDATE notification_preferences SET enabled=0 WHERE user_id=1 AND notification_type='low_stock'")

	// Generate for user 1
	generateNotificationsForUser(1)

	// Should NOT have created a low_stock notification
	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-001'").Scan(&count)
	if count != 0 {
		t.Error("expected no low_stock notification when disabled, got", count)
	}
}

func TestGenerateNotificationsFilteredEnabled(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	initNotificationPrefsTable()

	// Insert test inventory data
	db.Exec("INSERT INTO inventory (ipn, description, qty_on_hand, reorder_point) VALUES ('TEST-002', 'Test Part 2', 2, 10)")

	// Default prefs (all enabled)
	ensureDefaultPreferences(1)

	generateNotificationsForUser(1)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-002'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 low_stock notification, got %d", count)
	}
}

func TestGenerateNotificationsCustomThreshold(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	initNotificationPrefsTable()

	// Part with qty=5, reorder_point=10
	db.Exec("INSERT INTO inventory (ipn, description, qty_on_hand, reorder_point) VALUES ('TEST-003', 'Test Part 3', 5, 10)")

	ensureDefaultPreferences(1)
	// Set custom threshold to 3 — qty 5 is above 3, so should NOT alert
	db.Exec("UPDATE notification_preferences SET threshold_value=3 WHERE user_id=1 AND notification_type='low_stock'")

	generateNotificationsForUser(1)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-003'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 low_stock notification with custom threshold 3 (qty=5), got %d", count)
	}
}

func TestValidDeliveryMethod(t *testing.T) {
	if !isValidDeliveryMethod("in_app") {
		t.Error("in_app should be valid")
	}
	if !isValidDeliveryMethod("email") {
		t.Error("email should be valid")
	}
	if !isValidDeliveryMethod("both") {
		t.Error("both should be valid")
	}
	if isValidDeliveryMethod("sms") {
		t.Error("sms should not be valid")
	}
}

func TestValidNotificationType(t *testing.T) {
	if !isValidNotificationType("low_stock") {
		t.Error("low_stock should be valid")
	}
	if isValidNotificationType("nonexistent") {
		t.Error("nonexistent should not be valid")
	}
}
