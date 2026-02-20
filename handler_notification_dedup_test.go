package main

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// setupNotificationTestDB creates a test database with necessary tables for notification tests
func setupNotificationTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create notifications table
	_, err = testDB.Exec(`CREATE TABLE IF NOT EXISTS notifications (
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
	)`)
	if err != nil {
		t.Fatalf("Failed to create notifications table: %v", err)
	}

	// Create inventory table for low stock tests
	_, err = testDB.Exec(`CREATE TABLE inventory (
		ipn TEXT PRIMARY KEY,
		qty_on_hand REAL DEFAULT 0,
		qty_reserved REAL DEFAULT 0,
		location TEXT,
		reorder_point REAL DEFAULT 0,
		reorder_qty REAL DEFAULT 0,
		description TEXT DEFAULT '',
		mpn TEXT DEFAULT '',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	// Create work_orders table for overdue WO tests
	_, err = testDB.Exec(`CREATE TABLE work_orders (
		id TEXT PRIMARY KEY,
		assembly_ipn TEXT NOT NULL,
		qty INTEGER DEFAULT 1,
		qty_good INTEGER,
		qty_scrap INTEGER,
		status TEXT DEFAULT 'pending',
		priority TEXT DEFAULT 'normal',
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME
	)`)
	if err != nil {
		t.Fatalf("Failed to create work_orders table: %v", err)
	}

	// Create ncrs table for NCR tests
	_, err = testDB.Exec(`CREATE TABLE ncrs (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT,
		ipn TEXT,
		serial_number TEXT,
		defect_type TEXT,
		severity TEXT,
		status TEXT DEFAULT 'open',
		root_cause TEXT,
		corrective_action TEXT,
		created_by TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		resolved_at DATETIME
	)`)
	if err != nil {
		t.Fatalf("Failed to create ncrs table: %v", err)
	}

	// Create notification_preferences table
	_, err = testDB.Exec(`CREATE TABLE IF NOT EXISTS notification_preferences (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		notification_type TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		delivery_method TEXT DEFAULT 'in_app',
		threshold_value REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, notification_type)
	)`)
	if err != nil {
		t.Fatalf("Failed to create notification_preferences table: %v", err)
	}

	// Create users table
	_, err = testDB.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT,
		role TEXT DEFAULT 'user',
		email TEXT,
		active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create audit_log table - CRITICAL: Used by almost every handler
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			username TEXT,
			action TEXT,
			table_name TEXT,
			record_id TEXT,
			details TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	// Insert test user
	_, err = testDB.Exec(`INSERT INTO users (id, username, email, active) VALUES (1, 'testuser', 'test@example.com', 1)`)
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	return testDB
}

// Test 1: Same alert triggered multiple times within short window → only 1 notification sent
func TestNotificationDedup_SameAlertRapidFire_OnlyOneNotification(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert low stock inventory item
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) 
		VALUES ('TEST-001', 2, 10, 'Test Part Low Stock')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Trigger the same low_stock alert 5 times rapidly
	for i := 0; i < 5; i++ {
		createNotificationIfNew("low_stock", "warning", "Low Stock: TEST-001",
			stringPtr("2 on hand, reorder point 10"), stringPtr("TEST-001"), stringPtr("inventory"))
	}

	// Count notifications - should only have 1
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-001'`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query notifications: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 notification after 5 rapid triggers, got %d", count)
	}
}

// Test 2: Different alert types don't interfere with each other
func TestNotificationDedup_DifferentAlertTypes_NoInterference(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create different types of alerts for the same record
	createNotificationIfNew("low_stock", "warning", "Low Stock: PART-001",
		stringPtr("Low stock alert"), stringPtr("PART-001"), stringPtr("inventory"))

	createNotificationIfNew("overdue_wo", "warning", "Overdue WO: PART-001",
		stringPtr("Work order overdue"), stringPtr("PART-001"), stringPtr("workorders"))

	createNotificationIfNew("open_ncr", "error", "Open NCR: PART-001",
		stringPtr("NCR open"), stringPtr("PART-001"), stringPtr("ncr"))

	// Count total notifications - should have 3 (one of each type)
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE record_id='PART-001'`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query notifications: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 notifications (one of each type), got %d", count)
	}

	// Verify each type exists
	var lowStockCount, overdueWOCount, ncrCount int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='PART-001'`).Scan(&lowStockCount)
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='overdue_wo' AND record_id='PART-001'`).Scan(&overdueWOCount)
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='open_ncr' AND record_id='PART-001'`).Scan(&ncrCount)

	if lowStockCount != 1 || overdueWOCount != 1 || ncrCount != 1 {
		t.Errorf("Expected 1 notification of each type, got low_stock=%d, overdue_wo=%d, open_ncr=%d",
			lowStockCount, overdueWOCount, ncrCount)
	}
}

// Test 3: Same alert after cooldown period → new notification sent
func TestNotificationDedup_AfterCooldownPeriod_NewNotificationSent(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create first notification
	createNotificationIfNew("low_stock", "warning", "Low Stock: TEST-002",
		stringPtr("First alert"), stringPtr("TEST-002"), stringPtr("inventory"))

	// Count - should be 1
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-002'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification initially, got %d", count)
	}

	// Manually set the created_at to 25 hours ago (outside the 24-hour window)
	_, err := db.Exec(`UPDATE notifications 
		SET created_at = datetime('now', '-25 hours') 
		WHERE type='low_stock' AND record_id='TEST-002'`)
	if err != nil {
		t.Fatalf("Failed to update notification timestamp: %v", err)
	}

	// Trigger the same alert again (should create new notification after cooldown)
	createNotificationIfNew("low_stock", "warning", "Low Stock: TEST-002",
		stringPtr("Second alert"), stringPtr("TEST-002"), stringPtr("inventory"))

	// Count - should now be 2
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-002'`).Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 notifications after cooldown period, got %d", count)
	}
}

// Test 4: Within cooldown period → no new notification
func TestNotificationDedup_WithinCooldownPeriod_NoNewNotification(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create first notification
	createNotificationIfNew("low_stock", "warning", "Low Stock: TEST-003",
		stringPtr("First alert"), stringPtr("TEST-003"), stringPtr("inventory"))

	// Manually set created_at to 23 hours ago (still within 24-hour window)
	_, err := db.Exec(`UPDATE notifications 
		SET created_at = datetime('now', '-23 hours') 
		WHERE type='low_stock' AND record_id='TEST-003'`)
	if err != nil {
		t.Fatalf("Failed to update notification timestamp: %v", err)
	}

	// Try to create another notification (should be blocked by dedup)
	createNotificationIfNew("low_stock", "warning", "Low Stock: TEST-003",
		stringPtr("Second alert"), stringPtr("TEST-003"), stringPtr("inventory"))

	// Count - should still be 1
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-003'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification within cooldown period, got %d", count)
	}
}

// Test 5: Notifications without record_id (title-based dedup)
func TestNotificationDedup_NoRecordID_TitleBasedDedup(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	title := "System Health Check"

	// Create notification without record_id - first time
	createNotificationIfNew("system_health", "info", title, stringPtr("Health check passed"), nil, stringPtr("system"))

	// Try to create same notification again
	createNotificationIfNew("system_health", "info", title, stringPtr("Health check passed again"), nil, stringPtr("system"))

	// Count - should be 1 (deduped by type + title)
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='system_health' AND title=?`, title).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification with title-based dedup, got %d", count)
	}
}

// Test 6: User notification preferences respected (disabled notification)
func TestNotificationDedup_UserPrefsDisabled_NoNotification(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initNotificationPrefsTable()

	// Insert inventory item that would trigger low_stock
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) 
		VALUES ('TEST-004', 2, 10, 'Test Part')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Set up preferences - disable low_stock notifications for user 1
	ensureDefaultPreferences(1)
	_, err = db.Exec(`UPDATE notification_preferences 
		SET enabled=0 
		WHERE user_id=1 AND notification_type='low_stock'`)
	if err != nil {
		t.Fatalf("Failed to update preferences: %v", err)
	}

	// Generate notifications for user 1
	generateNotificationsForUser(1)

	// Count low_stock notifications - should be 0 (disabled)
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-004'`).Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 notifications when user pref is disabled, got %d", count)
	}
}

// Test 7: User notification preferences respected (enabled notification)
func TestNotificationDedup_UserPrefsEnabled_NotificationCreated(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initNotificationPrefsTable()

	// Insert inventory item that would trigger low_stock
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) 
		VALUES ('TEST-005', 2, 10, 'Test Part')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Set up preferences - ensure low_stock is enabled
	ensureDefaultPreferences(1)
	_, err = db.Exec(`UPDATE notification_preferences 
		SET enabled=1 
		WHERE user_id=1 AND notification_type='low_stock'`)
	if err != nil {
		t.Fatalf("Failed to update preferences: %v", err)
	}

	// Generate notifications for user 1
	generateNotificationsForUser(1)

	// Count low_stock notifications - should be 1
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-005'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification when user pref is enabled, got %d", count)
	}
}

// Test 8: Multiple users, different preferences
func TestNotificationDedup_MultipleUsers_IndependentPreferences(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initNotificationPrefsTable()

	// Create second test user
	_, err := db.Exec(`INSERT INTO users (id, username, email, active) VALUES (2, 'testuser2', 'test2@example.com', 1)`)
	if err != nil {
		t.Fatalf("Failed to insert second test user: %v", err)
	}

	// Insert inventory item
	_, err = db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) 
		VALUES ('TEST-006', 2, 10, 'Test Part')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// User 1: enable low_stock, User 2: disable low_stock
	ensureDefaultPreferences(1)
	ensureDefaultPreferences(2)
	db.Exec(`UPDATE notification_preferences SET enabled=1 WHERE user_id=1 AND notification_type='low_stock'`)
	db.Exec(`UPDATE notification_preferences SET enabled=0 WHERE user_id=2 AND notification_type='low_stock'`)

	// Generate notifications for both users
	generateNotificationsForUser(1)
	generateNotificationsForUser(2)

	// Count low_stock notifications - should be 1 (only from user 1)
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-006'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification (only user 1 enabled), got %d", count)
	}
}

// Test 9: Deduplication with custom threshold
func TestNotificationDedup_CustomThreshold_RespectedInDedup(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initNotificationPrefsTable()

	// Insert inventory item with qty=5
	_, err := db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point, description) 
		VALUES ('TEST-007', 5, 10, 'Test Part')`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Set custom threshold to 3 - qty of 5 is above threshold, should NOT alert
	ensureDefaultPreferences(1)
	_, err = db.Exec(`UPDATE notification_preferences 
		SET threshold_value=3 
		WHERE user_id=1 AND notification_type='low_stock'`)
	if err != nil {
		t.Fatalf("Failed to update threshold: %v", err)
	}

	// Generate notifications
	generateNotificationsForUser(1)

	// Count - should be 0 (qty 5 > threshold 3)
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-007'`).Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 notifications (qty above custom threshold), got %d", count)
	}

	// Now update inventory to qty=2 (below threshold 3)
	db.Exec(`UPDATE inventory SET qty_on_hand=2 WHERE ipn='TEST-007'`)

	// Generate again
	generateNotificationsForUser(1)

	// Count - should be 1 now (qty 2 < threshold 3)
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='TEST-007'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification (qty below custom threshold), got %d", count)
	}
}

// Test 10: Rapid sequential notification creation (stress test)
func TestNotificationDedup_RapidSequential_NoDuplicates(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Rapidly create the same notification 100 times in a tight loop
	iterations := 100
	for i := 0; i < iterations; i++ {
		createNotificationIfNew("low_stock", "warning", "Low Stock: RAPID",
			stringPtr("Rapid test"), stringPtr("RAPID"), stringPtr("inventory"))
	}

	// Count - should be 1 despite many rapid creation attempts
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='low_stock' AND record_id='RAPID'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification despite %d rapid sequential attempts, got %d", iterations, count)
	}
}

// Test 11: Overdue work order deduplication
func TestNotificationDedup_OverdueWorkOrder_Deduplicated(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert overdue work order
	_, err := db.Exec(`INSERT INTO work_orders (id, assembly_ipn, status, started_at) 
		VALUES ('WO-001', 'ASSEMBLY-001', 'in_progress', datetime('now', '-8 days'))`)
	if err != nil {
		t.Fatalf("Failed to insert work order: %v", err)
	}

	// Trigger notification multiple times
	for i := 0; i < 3; i++ {
		createNotificationIfNew("overdue_wo", "warning", "Overdue WO: WO-001",
			stringPtr("In progress for >7 days"), stringPtr("WO-001"), stringPtr("workorders"))
	}

	// Count - should be 1
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='overdue_wo' AND record_id='WO-001'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 overdue_wo notification, got %d", count)
	}
}

// Test 12: NCR notification deduplication
func TestNotificationDedup_OpenNCR_Deduplicated(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert open NCR
	_, err := db.Exec(`INSERT INTO ncrs (id, title, status, created_at) 
		VALUES ('NCR-001', 'Critical Defect', 'open', datetime('now', '-15 days'))`)
	if err != nil {
		t.Fatalf("Failed to insert NCR: %v", err)
	}

	// Trigger notification multiple times
	for i := 0; i < 3; i++ {
		createNotificationIfNew("open_ncr", "error", "Open NCR >14d: NCR-001",
			stringPtr("Critical Defect"), stringPtr("NCR-001"), stringPtr("ncr"))
	}

	// Count - should be 1
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type='open_ncr' AND record_id='NCR-001'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 open_ncr notification, got %d", count)
	}
}

// Test 13: Verify cooldown period is exactly 24 hours
func TestNotificationDedup_CooldownPeriod_Exactly24Hours(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create first notification
	createNotificationIfNew("low_stock", "warning", "Low Stock: COOLDOWN",
		stringPtr("First"), stringPtr("COOLDOWN"), stringPtr("inventory"))

	// Test at 23 hours 59 minutes (should still be blocked)
	db.Exec(`UPDATE notifications 
		SET created_at = datetime('now', '-23 hours', '-59 minutes') 
		WHERE record_id='COOLDOWN'`)

	createNotificationIfNew("low_stock", "warning", "Low Stock: COOLDOWN",
		stringPtr("Second"), stringPtr("COOLDOWN"), stringPtr("inventory"))

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE record_id='COOLDOWN'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification at 23:59, got %d", count)
	}

	// Test at exactly 24 hours 1 second (should create new)
	db.Exec(`UPDATE notifications 
		SET created_at = datetime('now', '-24 hours', '-1 seconds') 
		WHERE record_id='COOLDOWN'`)

	createNotificationIfNew("low_stock", "warning", "Low Stock: COOLDOWN",
		stringPtr("Third"), stringPtr("COOLDOWN"), stringPtr("inventory"))

	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE record_id='COOLDOWN'`).Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 notifications after 24:01, got %d", count)
	}
}

// Test 14: Email delivery method flag
func TestNotificationDedup_EmailDeliveryMethod_FlagSet(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initNotificationPrefsTable()

	// Insert inventory item
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES ('EMAIL-TEST', 2, 10)`)

	// Set delivery method to email
	ensureDefaultPreferences(1)
	db.Exec(`UPDATE notification_preferences SET delivery_method='email' WHERE user_id=1 AND notification_type='low_stock'`)

	// Generate notification
	generateNotificationsForUser(1)

	// Verify notification was created (email sending is tested elsewhere)
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE record_id='EMAIL-TEST'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 notification with email delivery, got %d", count)
	}
}

// Test 15: In-app only delivery method
func TestNotificationDedup_InAppOnlyDelivery_NoEmail(t *testing.T) {
	oldDB := db
	db = setupNotificationTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initNotificationPrefsTable()

	// Insert inventory item
	db.Exec(`INSERT INTO inventory (ipn, qty_on_hand, reorder_point) VALUES ('INAPP-TEST', 2, 10)`)

	// Set delivery method to in_app only
	ensureDefaultPreferences(1)
	db.Exec(`UPDATE notification_preferences SET delivery_method='in_app' WHERE user_id=1 AND notification_type='low_stock'`)

	// Generate notification
	generateNotificationsForUser(1)

	// Verify notification was created
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE record_id='INAPP-TEST'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 in-app notification, got %d", count)
	}

	// Verify emailed flag is 0
	var emailed int
	db.QueryRow(`SELECT emailed FROM notifications WHERE record_id='INAPP-TEST'`).Scan(&emailed)
	if emailed != 0 {
		t.Errorf("Expected emailed flag to be 0 for in-app only, got %d", emailed)
	}
}
