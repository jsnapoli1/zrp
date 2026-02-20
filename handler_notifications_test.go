package main

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupNotificationsTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create notifications table
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

	// Create inventory table (for generateNotifications)
	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			reorder_point REAL DEFAULT 0,
			location TEXT DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create work_orders table
	_, err = testDB.Exec(`
		CREATE TABLE work_orders (
			id TEXT PRIMARY KEY,
			assembly_ipn TEXT NOT NULL,
			qty INTEGER NOT NULL DEFAULT 1,
			status TEXT DEFAULT 'draft',
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Create ncrs table
	_, err = testDB.Exec(`
		CREATE TABLE ncrs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
	}

	// Create rmas table
	_, err = testDB.Exec(`
		CREATE TABLE rmas (
			id TEXT PRIMARY KEY,
			serial_number TEXT NOT NULL,
			customer TEXT,
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create rmas table: %v", err)
	}

	return testDB
}

func TestHandleListNotifications_All(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert test notifications
	_, err := db.Exec(`INSERT INTO notifications (type, severity, title, message, record_id, module) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		"low_stock", "warning", "Low Stock: PART-123", "Only 5 remaining", "PART-123", "inventory")
	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}

	_, err = db.Exec(`INSERT INTO notifications (type, severity, title, message, read_at) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		"info", "info", "System Update", "System updated successfully")
	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()

	handleListNotifications(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var notifs []Notification
	if err := json.NewDecoder(w.Body).Decode(&notifs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(notifs) != 2 {
		t.Errorf("Expected 2 notifications, got %d", len(notifs))
	}

	// Should be ordered by created_at DESC
	if notifs[0].Title != "System Update" {
		t.Errorf("Expected first notification to be 'System Update', got '%s'", notifs[0].Title)
	}
}

func TestHandleListNotifications_UnreadOnly(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert unread notification
	_, err := db.Exec(`INSERT INTO notifications (type, severity, title) VALUES (?, ?, ?)`,
		"low_stock", "warning", "Unread Notification")
	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}

	// Insert read notification
	_, err = db.Exec(`INSERT INTO notifications (type, severity, title, read_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		"info", "info", "Read Notification")
	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/notifications?unread=true", nil)
	w := httptest.NewRecorder()

	handleListNotifications(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var notifs []Notification
	if err := json.NewDecoder(w.Body).Decode(&notifs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(notifs) != 1 {
		t.Errorf("Expected 1 unread notification, got %d", len(notifs))
	}

	if notifs[0].Title != "Unread Notification" {
		t.Errorf("Expected 'Unread Notification', got '%s'", notifs[0].Title)
	}

	if notifs[0].ReadAt != nil {
		t.Error("Unread notification should have nil read_at")
	}
}

func TestHandleListNotifications_Empty(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()

	handleListNotifications(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var notifs []Notification
	if err := json.NewDecoder(w.Body).Decode(&notifs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(notifs) != 0 {
		t.Errorf("Expected empty array, got %d notifications", len(notifs))
	}
}

func TestHandleListNotifications_Limit(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert 60 notifications
	for i := 0; i < 60; i++ {
		_, err := db.Exec(`INSERT INTO notifications (type, severity, title) VALUES (?, ?, ?)`,
			"info", "info", "Notification "+string(rune(i+48)))
		if err != nil {
			t.Fatalf("Failed to insert notification: %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()

	handleListNotifications(w, req)

	var notifs []Notification
	json.NewDecoder(w.Body).Decode(&notifs)

	// Should be limited to 50
	if len(notifs) != 50 {
		t.Errorf("Expected 50 notifications (limit), got %d", len(notifs))
	}
}

func TestHandleMarkNotificationRead(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert unread notification
	result, err := db.Exec(`INSERT INTO notifications (type, severity, title) VALUES (?, ?, ?)`,
		"low_stock", "warning", "Test Notification")
	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}
	id, _ := result.LastInsertId()

	req := httptest.NewRequest("PATCH", "/api/notifications/"+string(rune(id+48)), nil)
	w := httptest.NewRecorder()

	handleMarkNotificationRead(w, req, string(rune(id+48)))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "read" {
		t.Errorf("Expected status 'read', got '%s'", resp["status"])
	}

	// Verify read_at was set
	var readAt *string
	db.QueryRow("SELECT read_at FROM notifications WHERE id = ?", id).Scan(&readAt)
	if readAt == nil {
		t.Error("read_at should be set after marking as read")
	}
}

func TestHandleMarkNotificationRead_AlreadyRead(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert already-read notification
	result, err := db.Exec(`INSERT INTO notifications (type, severity, title, read_at) 
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		"info", "info", "Already Read")
	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}
	id, _ := result.LastInsertId()

	req := httptest.NewRequest("PATCH", "/api/notifications/"+string(rune(id+48)), nil)
	w := httptest.NewRecorder()

	handleMarkNotificationRead(w, req, string(rune(id+48)))

	if w.Code != 200 {
		t.Errorf("Expected status 200 even if already read, got %d", w.Code)
	}
}

func TestGenerateNotifications_LowStock(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert inventory with low stock
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES (?, ?, ?)`,
		"PART-001", 5.0, 10.0)
	if err != nil {
		t.Fatalf("Failed to insert inventory: %v", err)
	}

	// Insert inventory with adequate stock
	_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES (?, ?, ?)`,
		"PART-002", 20.0, 10.0)
	if err != nil {
		t.Fatalf("Failed to insert inventory: %v", err)
	}

	generateNotifications()

	// Verify low stock notification was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'low_stock' AND record_id = 'PART-001'").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 low_stock notification for PART-001, got %d", count)
	}

	// Verify no notification for PART-002
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'low_stock' AND record_id = 'PART-002'").Scan(&count)
	if count != 0 {
		t.Error("Should not create notification for adequate stock")
	}

	// Verify notification details
	var severity, title, message string
	db.QueryRow("SELECT severity, title, message FROM notifications WHERE type = 'low_stock' AND record_id = 'PART-001'").
		Scan(&severity, &title, &message)
	if severity != "warning" {
		t.Errorf("Expected severity 'warning', got '%s'", severity)
	}
	if !strings.Contains(title, "PART-001") {
		t.Errorf("Title should contain 'PART-001', got '%s'", title)
	}
	if !strings.Contains(message, "5") || !strings.Contains(message, "10") {
		t.Errorf("Message should contain qty (5) and reorder point (10), got '%s'", message)
	}
}

func TestGenerateNotifications_OverdueWorkOrder(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert overdue work order (started > 7 days ago, still in_progress)
	pastDate := time.Now().Add(-10 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, started_at) VALUES (?, ?, ?, ?, ?)`,
		"WO-001", "ASM-001", 10, "in_progress", pastDate)
	if err != nil {
		t.Fatalf("Failed to insert work order: %v", err)
	}

	// Insert recent work order (not overdue)
	recentDate := time.Now().Add(-2 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, started_at) VALUES (?, ?, ?, ?, ?)`,
		"WO-002", "ASM-002", 5, "in_progress", recentDate)
	if err != nil {
		t.Fatalf("Failed to insert work order: %v", err)
	}

	generateNotifications()

	// Verify overdue notification created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'overdue_wo' AND record_id = 'WO-001'").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 overdue_wo notification for WO-001, got %d", count)
	}

	// Verify no notification for recent work order
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'overdue_wo' AND record_id = 'WO-002'").Scan(&count)
	if count != 0 {
		t.Error("Should not create notification for recent work order")
	}
}

func TestGenerateNotifications_OpenNCR(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert old open NCR (> 14 days)
	pastDate := time.Now().Add(-20 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO ncrs (id, title, status, created_at) VALUES (?, ?, ?, ?)`,
		"NCR-001", "Old Issue", "open", pastDate)
	if err != nil {
		t.Fatalf("Failed to insert NCR: %v", err)
	}

	// Insert recent NCR (< 14 days)
	recentDate := time.Now().Add(-5 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = db.Exec(`INSERT INTO ncrs (id, title, status, created_at) VALUES (?, ?, ?, ?)`,
		"NCR-002", "New Issue", "open", recentDate)
	if err != nil {
		t.Fatalf("Failed to insert NCR: %v", err)
	}

	generateNotifications()

	// Verify notification for old NCR
	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'open_ncr' AND record_id = 'NCR-001'").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 open_ncr notification for NCR-001, got %d", count)
	}

	// Verify severity is error
	var severity string
	db.QueryRow("SELECT severity FROM notifications WHERE type = 'open_ncr' AND record_id = 'NCR-001'").Scan(&severity)
	if severity != "error" {
		t.Errorf("Expected severity 'error' for open NCR, got '%s'", severity)
	}

	// Verify no notification for recent NCR
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'open_ncr' AND record_id = 'NCR-002'").Scan(&count)
	if count != 0 {
		t.Error("Should not create notification for recent NCR")
	}
}

func TestGenerateNotifications_NewRMA(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert recent RMA (< 1 hour)
	recentDate := time.Now().Add(-30 * time.Minute).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO rmas (id, serial_number, customer, created_at) VALUES (?, ?, ?, ?)`,
		"RMA-001", "SN-12345", "Acme Corp", recentDate)
	if err != nil {
		t.Fatalf("Failed to insert RMA: %v", err)
	}

	// Insert old RMA (> 1 hour)
	oldDate := time.Now().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = db.Exec(`INSERT INTO rmas (id, serial_number, customer, created_at) VALUES (?, ?, ?, ?)`,
		"RMA-002", "SN-67890", "Test Inc", oldDate)
	if err != nil {
		t.Fatalf("Failed to insert RMA: %v", err)
	}

	generateNotifications()

	// Verify notification for recent RMA
	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'new_rma' AND record_id = 'RMA-001'").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 new_rma notification for RMA-001, got %d", count)
	}

	// Verify message contains serial and customer
	var message string
	db.QueryRow("SELECT message FROM notifications WHERE type = 'new_rma' AND record_id = 'RMA-001'").Scan(&message)
	if !strings.Contains(message, "SN-12345") {
		t.Errorf("Message should contain serial number, got '%s'", message)
	}
	if !strings.Contains(message, "Acme Corp") {
		t.Errorf("Message should contain customer name, got '%s'", message)
	}

	// Verify no notification for old RMA
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'new_rma' AND record_id = 'RMA-002'").Scan(&count)
	if count != 0 {
		t.Error("Should not create notification for old RMA")
	}
}

func TestCreateNotificationIfNew_NoDuplicates(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	recordID := "PART-123"
	message := "Low stock warning"

	// Create first notification
	createNotificationIfNew("low_stock", "warning", "Low Stock: PART-123", &message, &recordID, stringPtr("inventory"))

	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'low_stock' AND record_id = ?", recordID).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification, got %d", count)
	}

	// Try to create duplicate within 24 hours
	createNotificationIfNew("low_stock", "warning", "Low Stock: PART-123", &message, &recordID, stringPtr("inventory"))

	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'low_stock' AND record_id = ?", recordID).Scan(&count)
	if count != 1 {
		t.Errorf("Should not create duplicate notification, got %d total", count)
	}
}

func TestCreateNotificationIfNew_AllowsAfter24Hours(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	recordID := "PART-456"
	message := "Low stock warning"

	// Insert old notification (> 24 hours ago)
	pastDate := time.Now().Add(-25 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO notifications (type, severity, title, message, record_id, module, created_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"low_stock", "warning", "Low Stock: PART-456", message, recordID, "inventory", pastDate)
	if err != nil {
		t.Fatalf("Failed to insert old notification: %v", err)
	}

	// Create new notification (should be allowed since old one is > 24h)
	createNotificationIfNew("low_stock", "warning", "Low Stock: PART-456", &message, &recordID, stringPtr("inventory"))

	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'low_stock' AND record_id = ?", recordID).Scan(&count)
	if count != 2 {
		t.Errorf("Should allow new notification after 24 hours, got %d total", count)
	}
}

func TestCreateNotificationIfNew_NoRecordID(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	title := "System Maintenance"
	message := "Scheduled maintenance tonight"

	// Create notification without record_id (uses title for dedup)
	createNotificationIfNew("system", "info", title, &message, nil, nil)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'system' AND title = ?", title).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification, got %d", count)
	}

	// Try to create duplicate
	createNotificationIfNew("system", "info", title, &message, nil, nil)

	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'system' AND title = ?", title).Scan(&count)
	if count != 1 {
		t.Errorf("Should not create duplicate notification by title, got %d total", count)
	}
}

func TestNotificationSeverityLevels(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	severities := []string{"info", "warning", "error"}
	for _, sev := range severities {
		_, err := db.Exec(`INSERT INTO notifications (type, severity, title) VALUES (?, ?, ?)`,
			"test", sev, "Test "+sev)
		if err != nil {
			t.Errorf("Failed to insert notification with severity '%s': %v", sev, err)
		}
	}

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()
	handleListNotifications(w, req)

	var notifs []Notification
	json.NewDecoder(w.Body).Decode(&notifs)

	if len(notifs) != 3 {
		t.Errorf("Expected 3 notifications with different severities, got %d", len(notifs))
	}

	// Verify all severities are present
	severityMap := make(map[string]bool)
	for _, n := range notifs {
		severityMap[n.Severity] = true
	}
	for _, sev := range severities {
		if !severityMap[sev] {
			t.Errorf("Severity '%s' not found in results", sev)
		}
	}
}

func TestNotificationTypes(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	types := []string{"low_stock", "overdue_wo", "open_ncr", "new_rma", "info"}
	for _, ntype := range types {
		_, err := db.Exec(`INSERT INTO notifications (type, severity, title) VALUES (?, ?, ?)`,
			ntype, "info", "Test "+ntype)
		if err != nil {
			t.Errorf("Failed to insert notification with type '%s': %v", ntype, err)
		}
	}

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()
	handleListNotifications(w, req)

	var notifs []Notification
	json.NewDecoder(w.Body).Decode(&notifs)

	if len(notifs) != len(types) {
		t.Errorf("Expected %d notifications with different types, got %d", len(types), len(notifs))
	}
}

func TestNotificationModuleField(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	modules := []string{"inventory", "workorders", "ncr", "rma"}
	for i, mod := range modules {
		_, err := db.Exec(`INSERT INTO notifications (type, severity, title, module, record_id) VALUES (?, ?, ?, ?, ?)`,
			"test", "info", "Test", mod, "REC-"+string(rune(i+48)))
		if err != nil {
			t.Errorf("Failed to insert notification with module '%s': %v", mod, err)
		}
	}

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()
	handleListNotifications(w, req)

	var notifs []Notification
	json.NewDecoder(w.Body).Decode(&notifs)

	// Verify module field is populated
	moduleMap := make(map[string]bool)
	for _, n := range notifs {
		if n.Module != nil {
			moduleMap[*n.Module] = true
		}
	}
	if len(moduleMap) != len(modules) {
		t.Errorf("Expected %d different modules, got %d", len(modules), len(moduleMap))
	}
}

func TestNotificationUserIDField(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert notification with user_id
	_, err := db.Exec(`INSERT INTO notifications (type, severity, title, user_id) VALUES (?, ?, ?, ?)`,
		"test", "info", "User-specific notification", "user123")
	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}

	// Insert global notification (no user_id)
	_, err = db.Exec(`INSERT INTO notifications (type, severity, title) VALUES (?, ?, ?)`,
		"test", "info", "Global notification")
	if err != nil {
		t.Fatalf("Failed to insert notification: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()
	handleListNotifications(w, req)

	var notifs []Notification
	json.NewDecoder(w.Body).Decode(&notifs)

	if len(notifs) != 2 {
		t.Errorf("Expected 2 notifications, got %d", len(notifs))
	}
}

func TestGenerateNotifications_MultipleIssues(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert multiple low stock items
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES (?, ?, ?)`, "P1", 2.0, 10.0)
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES (?, ?, ?)`, "P2", 3.0, 15.0)

	// Insert overdue work order
	pastDate := time.Now().Add(-10 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	db.Exec(`INSERT INTO work_orders (id, assembly_ipn, qty, status, started_at) VALUES (?, ?, ?, ?, ?)`,
		"WO-100", "ASM-100", 10, "in_progress", pastDate)

	// Insert old NCR
	db.Exec(`INSERT INTO ncrs (id, title, status, created_at) VALUES (?, ?, ?, ?)`,
		"NCR-100", "Old Issue", "open", pastDate)

	generateNotifications()

	// Verify all notification types created
	var lowStockCount, overdueWOCount, openNCRCount int
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'low_stock'").Scan(&lowStockCount)
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'overdue_wo'").Scan(&overdueWOCount)
	db.QueryRow("SELECT COUNT(*) FROM notifications WHERE type = 'open_ncr'").Scan(&openNCRCount)

	if lowStockCount != 2 {
		t.Errorf("Expected 2 low_stock notifications, got %d", lowStockCount)
	}
	if overdueWOCount != 1 {
		t.Errorf("Expected 1 overdue_wo notification, got %d", overdueWOCount)
	}
	if openNCRCount != 1 {
		t.Errorf("Expected 1 open_ncr notification, got %d", openNCRCount)
	}
}

func TestNotificationPagination(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert 100 notifications
	for i := 0; i < 100; i++ {
		db.Exec(`INSERT INTO notifications (type, severity, title) VALUES (?, ?, ?)`,
			"test", "info", "Notification")
	}

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	w := httptest.NewRecorder()
	handleListNotifications(w, req)

	var notifs []Notification
	json.NewDecoder(w.Body).Decode(&notifs)

	// Should be limited to 50
	if len(notifs) > 50 {
		t.Errorf("Should limit results to 50, got %d", len(notifs))
	}
}

func TestNotificationReadState(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupNotificationsTestDB(t)
	defer db.Close()

	// Insert unread notification
	result, _ := db.Exec(`INSERT INTO notifications (type, severity, title) VALUES (?, ?, ?)`,
		"test", "info", "Test")
	id, _ := result.LastInsertId()

	// Initially read_at should be NULL
	var readAt *string
	db.QueryRow("SELECT read_at FROM notifications WHERE id = ?", id).Scan(&readAt)
	if readAt != nil {
		t.Error("New notification should have NULL read_at")
	}

	// Mark as read
	req := httptest.NewRequest("PATCH", "/api/notifications/"+string(rune(id+48)), nil)
	w := httptest.NewRecorder()
	handleMarkNotificationRead(w, req, string(rune(id+48)))

	// Now read_at should be set
	db.QueryRow("SELECT read_at FROM notifications WHERE id = ?", id).Scan(&readAt)
	if readAt == nil {
		t.Error("Marked notification should have read_at timestamp")
	}

	// Verify it's a valid timestamp
	_, err := time.Parse("2006-01-02 15:04:05", *readAt)
	if err != nil {
		t.Errorf("read_at should be valid timestamp, got '%s'", *readAt)
	}
}
