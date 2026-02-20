package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func setupAuthTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Enable foreign keys
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

	// Create sessions table (match production schema)
	_, err = testDB.Exec(`
		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	return testDB
}

func createTestUser(t *testing.T, db *sql.DB, username, password, role string, active bool) int {
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

func TestHandleLogin_Success(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	createTestUser(t, db, "testuser", "password123", "user", true)

	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	user, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response missing user object")
	}

	if user["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got %v", user["username"])
	}

	if user["role"] != "user" {
		t.Errorf("Expected role 'user', got %v", user["role"])
	}

	// Check session cookie was set
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "zrp_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Session cookie not set")
	}

	if sessionCookie.Value == "" {
		t.Error("Session token is empty")
	}

	if !sessionCookie.HttpOnly {
		t.Error("Session cookie should be HttpOnly")
	}

	// Verify session was created in DB
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", sessionCookie.Value).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query sessions: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session in DB, got %d", count)
	}
}

func TestHandleLogin_InvalidUsername(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	createTestUser(t, db, "testuser", "password123", "user", true)

	reqBody := `{"username":"nonexistent","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["error"] != "Invalid username or password" {
		t.Errorf("Expected error message, got %v", resp["error"])
	}
}

func TestHandleLogin_InvalidPassword(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	createTestUser(t, db, "testuser", "password123", "user", true)

	reqBody := `{"username":"testuser","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleLogin_InactiveUser(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	createTestUser(t, db, "testuser", "password123", "user", false)

	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 403 {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["error"] != "Account deactivated" {
		t.Errorf("Expected 'Account deactivated' error, got %v", resp["error"])
	}
}

func TestHandleLogin_RateLimiting(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	createTestUser(t, db, "testuser", "password123", "user", true)

	// Make 5 login attempts (should all succeed or fail based on credentials)
	for i := 0; i < 5; i++ {
		reqBody := `{"username":"testuser","password":"password123"}`
		req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		handleLogin(w, req)

		if w.Code != 200 {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	// 6th attempt should be rate limited
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 429 {
		t.Errorf("Expected status 429 (rate limited), got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["error"] != "Too many login attempts. Try again in a minute." {
		t.Errorf("Expected rate limit error, got %v", resp["error"])
	}
}

func TestHandleLogin_RateLimitPerIP(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	createTestUser(t, db, "testuser", "password123", "user", true)

	// Make 5 attempts from IP1
	for i := 0; i < 5; i++ {
		reqBody := `{"username":"testuser","password":"password123"}`
		req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handleLogin(w, req)
	}

	// Attempt from IP2 should still work (rate limit is per-IP)
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "192.168.1.2:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 from different IP, got %d", w.Code)
	}
}

func TestHandleLogin_InvalidJSON(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	reqBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleLogout(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create a session
	userID := createTestUser(t, db, "testuser", "password123", "user", true)
	token := generateToken()
	expires := time.Now().Add(24 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Make logout request with session cookie
	req := httptest.NewRequest("POST", "/api/v1/logout", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handleLogout(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify session was deleted from DB
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query sessions: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected session to be deleted, but found %d sessions", count)
	}

	// Verify cookie was cleared
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "zrp_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Expected session cookie to be set (for clearing)")
	}

	if sessionCookie.MaxAge != -1 {
		t.Errorf("Expected MaxAge -1 to clear cookie, got %d", sessionCookie.MaxAge)
	}
}

func TestHandleLogout_NoCookie(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("POST", "/api/v1/logout", nil)
	w := httptest.NewRecorder()

	handleLogout(w, req)

	// Should still succeed even without cookie
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleMe_ValidSession(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create user and session
	userID := createTestUser(t, db, "testuser", "password123", "admin", true)
	token := generateToken()
	expires := time.Now().Add(24 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	user, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response missing user object")
	}

	if user["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got %v", user["username"])
	}

	if user["role"] != "admin" {
		t.Errorf("Expected role 'admin', got %v", user["role"])
	}

	if user["display_name"] != "testuser Display" {
		t.Errorf("Expected display_name 'testuser Display', got %v", user["display_name"])
	}
}

func TestHandleMe_NoCookie(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["code"] != "UNAUTHORIZED" {
		t.Errorf("Expected code 'UNAUTHORIZED', got %v", resp["code"])
	}
}

func TestHandleMe_InvalidToken(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "invalid-token-12345"})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleMe_ExpiredSession(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create user and expired session
	userID := createTestUser(t, db, "testuser", "password123", "user", true)
	token := generateToken()
	expires := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for expired session, got %d", w.Code)
	}
}

func TestHandleChangePassword_Success(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createTestUser(t, db, "testuser", "OldPassword123!", "user", true)

	reqBody := `{"current_password":"OldPassword123!","new_password":"NewPassword123!"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	req = withUserID(req, userID)
	w := httptest.NewRecorder()

	handleChangePassword(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify new password works
	var newHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&newHash)
	if err != nil {
		t.Fatalf("Failed to get password hash: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(newHash), []byte("NewPassword123!")); err != nil {
		t.Error("New password doesn't match")
	}

	// Verify old password no longer works
	if err := bcrypt.CompareHashAndPassword([]byte(newHash), []byte("oldpassword")); err == nil {
		t.Error("Old password still works")
	}
}

func TestHandleChangePassword_WrongCurrentPassword(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createTestUser(t, db, "testuser", "OldPassword123!", "user", true)

	reqBody := `{"current_password":"WrongPassword123!","new_password":"NewPassword123!"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	req = withUserID(req, userID)
	w := httptest.NewRecorder()

	handleChangePassword(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["error"] != "Current password is incorrect" {
		t.Errorf("Expected error about incorrect password, got %v", resp["error"])
	}
}

func TestHandleChangePassword_PasswordTooShort(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createTestUser(t, db, "testuser", "OldPassword123!", "user", true)

	reqBody := `{"current_password":"OldPassword123!","new_password":"short"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	req = withUserID(req, userID)
	w := httptest.NewRecorder()

	handleChangePassword(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	// Check for password length error (actual message from validation)
	errorMsg, ok := resp["error"].(string)
	if !ok || errorMsg != "password must be at least 12 characters" {
		t.Errorf("Expected error about password length (12 chars), got %v", resp["error"])
	}
}

func TestHandleChangePassword_MissingFields(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createTestUser(t, db, "testuser", "oldpassword", "user", true)

	testCases := []struct {
		name     string
		reqBody  string
		expected string
	}{
		{
			name:     "missing new password",
			reqBody:  `{"current_password":"oldpassword"}`,
			expected: "Current and new password required",
		},
		{
			name:     "missing current password",
			reqBody:  `{"new_password":"newpassword"}`,
			expected: "Current and new password required",
		},
		{
			name:     "empty strings",
			reqBody:  `{"current_password":"","new_password":""}`,
			expected: "Current and new password required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(tc.reqBody))
			req = withUserID(req, userID)
			w := httptest.NewRecorder()

			handleChangePassword(w, req)

			if w.Code != 400 {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleChangePassword_Unauthorized(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{"current_password":"oldpassword","new_password":"newpassword123"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	// No user ID in context
	w := httptest.NewRecorder()

	handleChangePassword(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestGenerateToken(t *testing.T) {
	token1 := generateToken()
	token2 := generateToken()

	// Tokens should be non-empty
	if token1 == "" {
		t.Error("Generated token is empty")
	}

	// Tokens should be unique
	if token1 == token2 {
		t.Error("Generated tokens are not unique")
	}

	// Token should be hex-encoded 32 bytes = 64 hex chars
	if len(token1) != 64 {
		t.Errorf("Expected token length 64, got %d", len(token1))
	}
}

func TestGetClientIP_DirectConnection(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	ip := getClientIP(req)

	if ip != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got '%s'", ip)
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	ip := getClientIP(req)

	if ip != "203.0.113.1" {
		t.Errorf("Expected IP '203.0.113.1' from X-Forwarded-For, got '%s'", ip)
	}
}

func TestLoginRateLimitReset(t *testing.T) {
	defer resetLoginRateLimit()

	// Use up rate limit
	for i := 0; i < 5; i++ {
		if !checkLoginRateLimit("192.168.1.1") {
			t.Fatalf("Request %d should not be rate limited", i+1)
		}
	}

	// 6th should fail
	if checkLoginRateLimit("192.168.1.1") {
		t.Error("6th request should be rate limited")
	}

	// Reset
	resetLoginRateLimit()

	// Should work again
	if !checkLoginRateLimit("192.168.1.1") {
		t.Error("After reset, request should succeed")
	}
}

func TestHandleLogin_CleansExpiredSessions(t *testing.T) {
	oldDB := db
	db = setupAuthTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	userID := createTestUser(t, db, "testuser", "password123", "user", true)

	// Create an expired session
	expiredToken := generateToken()
	expiredTime := time.Now().Add(-2 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		expiredToken, userID, expiredTime.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create expired session: %v", err)
	}

	// Verify expired session exists before login
	var countBefore int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", expiredToken).Scan(&countBefore)
	if countBefore != 1 {
		t.Fatalf("Expected expired session to exist before login")
	}

	// Login (should trigger cleanup)
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	handleLogin(w, req)

	if w.Code != 200 {
		t.Fatalf("Login failed: %d", w.Code)
	}

	// Verify expired session was cleaned up
	var countAfter int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", expiredToken).Scan(&countAfter)
	if countAfter != 0 {
		t.Errorf("Expected expired session to be cleaned up, but found %d", countAfter)
	}
}
