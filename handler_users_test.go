package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func setupUsersTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create users table
	_, err = testDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1,
			last_login TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create sessions table
	_, err = testDB.Exec(`
		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
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

	return testDB
}

func createUsersTestUser(t *testing.T, db *sql.DB, username, password, role string, active bool) int {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	activeInt := 0
	if active {
		activeInt = 1
	}

	result, err := db.Exec(
		"INSERT INTO users (username, password_hash, display_name, role, active) VALUES (?, ?, ?, ?, ?)",
		username, string(hash), username+" Display", role, activeInt,
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

func createUsersTestSession(t *testing.T, db *sql.DB, userID int) string {
	token := "test-session-token-" + time.Now().Format("20060102150405")
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return token
}

func TestHandleListUsers_AsAdmin(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create admin user
	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)

	// Create additional users
	createUsersTestUser(t, db, "user1", "password", "user", true)
	createUsersTestUser(t, db, "user2", "password", "readonly", true)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleListUsers(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// The data field contains the array of users
	usersJSON, _ := json.Marshal(response.Data)
	var result []UserFull
	json.Unmarshal(usersJSON, &result)

	if len(result) != 3 {
		t.Errorf("Expected 3 users, got %d", len(result))
	}
}

func TestHandleListUsers_AsNonAdmin(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create non-admin user
	userID := createUsersTestUser(t, db, "user1", "password", "user", true)
	userToken := createUsersTestSession(t, db, userID)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: userToken})
	w := httptest.NewRecorder()

	handleListUsers(w, req)

	if w.Code != 403 {
		t.Errorf("Expected status 403 for non-admin, got %d", w.Code)
	}
}

func TestHandleListUsers_Unauthorized(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	handleListUsers(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for unauthorized, got %d", w.Code)
	}
}

func TestHandleCreateUser_Success(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)

	reqBody := `{
		"username": "newuser",
		"password": "newpassword",
		"display_name": "New User",
		"role": "user"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleCreateUser(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result, ok := response.Data.(map[string]interface{})
	if !ok || result["id"] == nil {
		t.Error("Expected user ID in response")
	}

	// Verify user was created
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE username=?", "newuser").Scan(&username)
	if err != nil {
		t.Fatalf("Expected user to be created: %v", err)
	}

	if username != "newuser" {
		t.Errorf("Expected username 'newuser', got %s", username)
	}
}

func TestHandleCreateUser_MissingUsername(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)

	reqBody := `{
		"password": "newpassword",
		"role": "user"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleCreateUser(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateUser_MissingPassword(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)

	reqBody := `{
		"username": "newuser",
		"role": "user"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleCreateUser(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateUser_DefaultRole(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)

	reqBody := `{
		"username": "newuser",
		"password": "newpassword"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleCreateUser(w, req)

	// Verify default role is 'user'
	var role string
	db.QueryRow("SELECT role FROM users WHERE username=?", "newuser").Scan(&role)
	if role != "user" {
		t.Errorf("Expected default role 'user', got %s", role)
	}
}

func TestHandleUpdateUser_Success(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)
	targetUserID := createUsersTestUser(t, db, "target", "password", "user", true)

	reqBody := `{
		"display_name": "Updated Name",
		"role": "readonly"
	}`
	req := httptest.NewRequest("PUT", "/api/v1/users/"+strconv.Itoa(targetUserID), bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleUpdateUser(w, req, strconv.Itoa(targetUserID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify updates
	var displayName, role string
	db.QueryRow("SELECT display_name, role FROM users WHERE id=?", targetUserID).Scan(&displayName, &role)

	if displayName != "Updated Name" {
		t.Errorf("Expected display_name 'Updated Name', got %s", displayName)
	}
	if role != "readonly" {
		t.Errorf("Expected role 'readonly', got %s", role)
	}
}

func TestHandleUpdateUser_Deactivate(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)
	targetUserID := createUsersTestUser(t, db, "target", "password", "user", true)

	inactive := 0
	reqBody := `{
		"display_name": "Deactivated User",
		"role": "user",
		"active": 0
	}`
	req := httptest.NewRequest("PUT", "/api/v1/users/"+strconv.Itoa(targetUserID), bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleUpdateUser(w, req, strconv.Itoa(targetUserID))

	// Verify deactivation
	var active int
	db.QueryRow("SELECT active FROM users WHERE id=?", targetUserID).Scan(&active)

	if active != inactive {
		t.Errorf("Expected active=0, got %d", active)
	}
}

func TestHandleUpdateUser_NotFound(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)

	reqBody := `{
		"display_name": "Updated",
		"role": "user"
	}`
	req := httptest.NewRequest("PUT", "/api/v1/users/9999", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleUpdateUser(w, req, "9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleDeleteUser_Success(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)
	targetUserID := createUsersTestUser(t, db, "target", "password", "user", true)

	req := httptest.NewRequest("DELETE", "/api/v1/users/"+strconv.Itoa(targetUserID), nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleDeleteUser(w, req, strconv.Itoa(targetUserID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify user was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE id=?", targetUserID).Scan(&count)
	if count != 0 {
		t.Error("Expected user to be deleted")
	}
}

func TestHandleDeleteUser_NotFound(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)

	req := httptest.NewRequest("DELETE", "/api/v1/users/9999", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleDeleteUser(w, req, "9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleResetPassword_Success(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)
	targetUserID := createUsersTestUser(t, db, "target", "oldpassword", "user", true)

	reqBody := `{
		"password": "newpassword123"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users/"+strconv.Itoa(targetUserID)+"/reset-password", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleResetPassword(w, req, strconv.Itoa(targetUserID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify password was changed
	var passwordHash string
	db.QueryRow("SELECT password_hash FROM users WHERE id=?", targetUserID).Scan(&passwordHash)

	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("newpassword123"))
	if err != nil {
		t.Error("Expected password to be updated")
	}
}

func TestHandleResetPassword_MissingPassword(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)
	targetUserID := createUsersTestUser(t, db, "target", "password", "user", true)

	reqBody := `{}`
	req := httptest.NewRequest("POST", "/api/v1/users/"+strconv.Itoa(targetUserID)+"/reset-password", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleResetPassword(w, req, strconv.Itoa(targetUserID))

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleResetPassword_NotFound(t *testing.T) {
	oldDB := db
	db = setupUsersTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	adminID := createUsersTestUser(t, db, "admin", "password", "admin", true)
	adminToken := createUsersTestSession(t, db, adminID)

	reqBody := `{
		"password": "newpassword"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users/9999/reset-password", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	handleResetPassword(w, req, "9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
