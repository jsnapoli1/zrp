package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupUndoTestDB(t *testing.T) *sql.DB {
	testDB, _ := sql.Open("sqlite", ":memory:")
	testDB.Exec("PRAGMA foreign_keys = ON")
	testDB.Exec(`CREATE TABLE undo_log (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL, action TEXT NOT NULL, entity_type TEXT NOT NULL, entity_id TEXT NOT NULL, previous_data TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, expires_at DATETIME NOT NULL)`)
	testDB.Exec(`CREATE TABLE ecos (id TEXT PRIMARY KEY, title TEXT NOT NULL, description TEXT, status TEXT DEFAULT 'draft', priority TEXT DEFAULT 'normal', affected_ipns TEXT DEFAULT '[]', created_by TEXT DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP, approved_at DATETIME, approved_by TEXT)`)
	testDB.Exec(`CREATE TABLE work_orders (id TEXT PRIMARY KEY, assembly_ipn TEXT NOT NULL, qty INTEGER NOT NULL DEFAULT 1, status TEXT DEFAULT 'draft', priority TEXT DEFAULT 'normal', notes TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, started_at DATETIME, completed_at DATETIME)`)
	testDB.Exec(`CREATE TABLE inventory (ipn TEXT PRIMARY KEY, qty_on_hand REAL DEFAULT 0, qty_reserved REAL DEFAULT 0, location TEXT, reorder_point REAL DEFAULT 0, reorder_qty REAL DEFAULT 0, description TEXT DEFAULT '', mpn TEXT DEFAULT '', updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	testDB.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL)`)
	testDB.Exec(`CREATE TABLE sessions (token TEXT PRIMARY KEY, user_id INTEGER NOT NULL)`)
	testDB.Exec(`CREATE TABLE audit_log (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT DEFAULT 'system', action TEXT, module TEXT, record_id TEXT, summary TEXT)`)
	testDB.Exec("INSERT INTO users (id, username) VALUES (1, 'testuser'), (2, 'user1')")
	testDB.Exec("INSERT INTO sessions (token, user_id) VALUES ('test-session-token', 1), ('user1-token', 2)")
	return testDB
}

func TestSnapshotEntity_ECO(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupUndoTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO ecos (id, title, description) VALUES (?, ?, ?)", "ECO-001", "Test ECO", "Test description")

	snapshot, err := snapshotEntity("eco", "ECO-001")
	if err != nil {
		t.Fatalf("snapshotEntity failed: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal([]byte(snapshot), &data)

	if data["id"] != "ECO-001" {
		t.Error("Snapshot missing id")
	}
}

func TestSnapshotEntity_UnsupportedType(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupUndoTestDB(t)
	defer db.Close()

	_, err := snapshotEntity("unsupported_type", "ID-001")
	if err == nil {
		t.Error("Expected error for unsupported entity type")
	}
}

func TestCreateUndoEntry_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupUndoTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO ecos (id, title) VALUES (?, ?)", "ECO-001", "Test")

	id, err := createUndoEntry("testuser", "delete", "eco", "ECO-001")
	if err != nil {
		t.Fatalf("createUndoEntry failed: %v", err)
	}

	if id == 0 {
		t.Error("Expected non-zero undo entry ID")
	}
}

func TestHandleListUndo_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupUndoTestDB(t)
	defer db.Close()

	future := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO undo_log (user_id, action, entity_type, entity_id, previous_data, expires_at) VALUES (?, ?, ?, ?, ?, ?)",
		"testuser", "delete", "eco", "ECO-001", `{"id":"ECO-001"}`, future)

	req := httptest.NewRequest("GET", "/api/undo", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "test-session-token"})
	w := httptest.NewRecorder()

	handleListUndo(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRestoreECO_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupUndoTestDB(t)
	defer db.Close()

	jsonData := `{"id": "ECO-RESTORED","title": "Restored ECO","description": "Test restore","status": "draft","priority": "normal","affected_ipns": "[]","created_by": "testuser","created_at": "2024-01-01 10:00:00","updated_at": "2024-01-01 10:00:00","approved_at": null,"approved_by": null}`

	err := restoreECO(jsonData)
	if err != nil {
		t.Fatalf("restoreECO failed: %v", err)
	}

	var title string
	db.QueryRow("SELECT title FROM ecos WHERE id = ?", "ECO-RESTORED").Scan(&title)

	if title != "Restored ECO" {
		t.Errorf("Expected title 'Restored ECO', got %s", title)
	}
}

func TestGetRowAsMap_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupUndoTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO ecos (id, title) VALUES (?, ?)", "ECO-001", "Test")

	row, err := getRowAsMap("SELECT id, title FROM ecos WHERE id = ?", "ECO-001")
	if err != nil {
		t.Fatalf("getRowAsMap failed: %v", err)
	}

	if row["id"] != "ECO-001" {
		t.Error("Map missing id")
	}
}

func TestUndoEntry_24HourExpiration(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupUndoTestDB(t)
	defer db.Close()

	db.Exec("INSERT INTO ecos (id, title) VALUES (?, ?)", "ECO-001", "Test")

	id, _ := createUndoEntry("testuser", "delete", "eco", "ECO-001")

	var expiresAt string
	db.QueryRow("SELECT expires_at FROM undo_log WHERE id = ?", id).Scan(&expiresAt)

	expires, _ := time.Parse("2006-01-02 15:04:05", expiresAt)
	duration := expires.Sub(time.Now())

	if duration < 23*time.Hour || duration > 25*time.Hour {
		t.Errorf("Expected ~24 hour expiration, got %v", duration)
	}
}
