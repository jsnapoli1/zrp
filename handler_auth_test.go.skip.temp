package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func setupAuthTestDB(t *testing.T) func() {
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
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			failed_login_attempts INTEGER DEFAULT 0,
			account_locked_until TIMESTAMP
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	// Create csrf_tokens table
	_, err = testDB.Exec(`
		CREATE TABLE csrf_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create csrf_tokens table: %v", err)
	}

	// Create password_history table
	_, err = testDB.Exec(`
		CREATE TABLE password_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create password_history table: %v", err)
	}

	// Save and swap db
	origDB := db
	db = testDB

	return func() {
		db.Close()
		db = origDB
		resetLoginRateLimit()
	}
}

func createTestUser(t *testing.T, username, password, role string, active bool) int {
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

func TestHandleLoginSuccess(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	createTestUser(t, "testuser", "password123", "user", true)

	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	user, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing user object")
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
	if !sessionCookie.HttpOnly {
		t.Error("Session cookie should be HttpOnly")
	}
	if !sessionCookie.Secure {
		t.Error("Session cookie should be Secure")
	}

	// Verify session in DB
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", sessionCookie.Value).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 session in DB, got %d", count)
	}

	// Verify last_login was updated
	var lastLogin sql.NullString
	db.QueryRow("SELECT last_login FROM users WHERE username='testuser'").Scan(&lastLogin)
	if !lastLogin.Valid {
		t.Error("Expected last_login to be set")
	}
}

func TestHandleLoginInvalidUsername(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	createTestUser(t, "testuser", "password123", "user", true)

	reqBody := `{"username":"nonexistent","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleLoginInvalidPassword(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	createTestUser(t, "testuser", "password123", "user", true)

	reqBody := `{"username":"testuser","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Verify failed attempt was incremented
	var attempts int
	db.QueryRow("SELECT failed_login_attempts FROM users WHERE username='testuser'").Scan(&attempts)
	if attempts != 1 {
		t.Errorf("Expected 1 failed attempt, got %d", attempts)
	}
}

func TestHandleLoginInactiveUser(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	createTestUser(t, "testuser", "password123", "user", false)

	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 403 {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestHandleLoginRateLimiting(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	createTestUser(t, "testuser", "password123", "user", true)

	// Make 5 login attempts
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
}

func TestHandleLoginRateLimitPerIP(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	createTestUser(t, "testuser", "password123", "user", true)

	// Make 5 attempts from IP1
	for i := 0; i < 5; i++ {
		reqBody := `{"username":"testuser","password":"password123"}`
		req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handleLogin(w, req)
	}

	// Attempt from IP2 should still work
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "192.168.1.2:12345"
	w := httptest.NewRecorder()
	handleLogin(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 from different IP, got %d", w.Code)
	}
}

func TestHandleLoginInvalidJSON(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	reqBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleLoginCleansExpiredSessions(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "password123", "user", true)

	// Create expired session
	expiredToken := generateToken()
	expiredTime := time.Now().Add(-2 * time.Hour)
	db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		expiredToken, userID, expiredTime.Format("2006-01-02 15:04:05"))

	// Login should clean up expired session
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	handleLogin(w, req)

	// Verify expired session was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", expiredToken).Scan(&count)
	if count != 0 {
		t.Error("Expected expired session to be cleaned up")
	}
}

func TestHandleLogoutSuccess(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "password123", "user", true)
	token := generateToken()
	expires := time.Now().Add(24 * time.Hour)
	db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))

	req := httptest.NewRequest("POST", "/api/v1/logout", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handleLogout(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify session was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&count)
	if count != 0 {
		t.Error("Expected session to be deleted")
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
		t.Fatal("Expected session cookie to be set for clearing")
	}
	if sessionCookie.MaxAge != -1 {
		t.Errorf("Expected MaxAge -1, got %d", sessionCookie.MaxAge)
	}
}

func TestHandleLogoutNoCookie(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/v1/logout", nil)
	w := httptest.NewRecorder()

	handleLogout(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleMeValidSession(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "password123", "admin", true)
	token := generateToken()
	expires := time.Now().Add(24 * time.Hour)
	db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	user, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing user object")
	}

	if user["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got %v", user["username"])
	}
	if user["role"] != "admin" {
		t.Errorf("Expected role 'admin', got %v", user["role"])
	}
}

func TestHandleMeNoCookie(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleMeInvalidToken(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "invalid-token"})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleMeExpiredSession(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "password123", "user", true)
	token := generateToken()
	expires := time.Now().Add(-1 * time.Hour)
	db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for expired session, got %d", w.Code)
	}
}

func TestHandleMeInactivityTimeout(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "password123", "user", true)
	token := generateToken()
	expires := time.Now().Add(24 * time.Hour)
	lastActivity := time.Now().Add(-31 * time.Minute).Format("2006-01-02 15:04:05")
	
	db.Exec("INSERT INTO sessions (token, user_id, expires_at, last_activity) VALUES (?, ?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"), lastActivity)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for inactive session, got %d", w.Code)
	}

	// Verify session was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&count)
	if count != 0 {
		t.Error("Expected inactive session to be deleted")
	}
}

func TestHandleMeUpdatesLastActivity(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "password123", "user", true)
	token := generateToken()
	expires := time.Now().Add(24 * time.Hour)
	oldActivity := time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05")
	
	db.Exec("INSERT INTO sessions (token, user_id, expires_at, last_activity) VALUES (?, ?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"), oldActivity)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Verify last_activity was updated
	var newActivity string
	db.QueryRow("SELECT last_activity FROM sessions WHERE token = ?", token).Scan(&newActivity)
	
	if newActivity == oldActivity {
		t.Error("Expected last_activity to be updated")
	}
}

func TestHandleChangePasswordSuccess(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "OldPassword123!", "user", true)

	reqBody := `{"current_password":"OldPassword123!","new_password":"NewPassword123!"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleChangePassword(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify new password works
	var newHash string
	db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&newHash)
	
	if err := bcrypt.CompareHashAndPassword([]byte(newHash), []byte("NewPassword123!")); err != nil {
		t.Error("New password doesn't match")
	}

	// Verify old password no longer works
	if err := bcrypt.CompareHashAndPassword([]byte(newHash), []byte("OldPassword123!")); err == nil {
		t.Error("Old password still works")
	}
}

func TestHandleChangePasswordWrongCurrent(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "OldPassword123!", "user", true)

	reqBody := `{"current_password":"WrongPassword123!","new_password":"NewPassword123!"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleChangePassword(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleChangePasswordTooShort(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "OldPassword123!", "user", true)

	reqBody := `{"current_password":"OldPassword123!","new_password":"short"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleChangePassword(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleChangePasswordMissingFields(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "password", "user", true)

	tests := []struct {
		name    string
		reqBody string
	}{
		{"missing new password", `{"current_password":"password"}`},
		{"missing current password", `{"new_password":"newpassword"}`},
		{"empty strings", `{"current_password":"","new_password":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(tt.reqBody))
			ctx := context.WithValue(req.Context(), ctxUserID, userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handleChangePassword(w, req)

			if w.Code != 400 {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleChangePasswordUnauthorized(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	reqBody := `{"current_password":"old","new_password":"new"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleChangePassword(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestGenerateToken(t *testing.T) {
	token1 := generateToken()
	token2 := generateToken()

	if token1 == "" {
		t.Error("Generated token is empty")
	}
	if token1 == token2 {
		t.Error("Generated tokens are not unique")
	}
	if len(token1) != 64 {
		t.Errorf("Expected token length 64, got %d", len(token1))
	}
}

func TestGetClientIPDirect(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	ip := getClientIP(req)

	if ip != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got '%s'", ip)
	}
}

func TestGetClientIPXForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	ip := getClientIP(req)

	if ip != "203.0.113.1" {
		t.Errorf("Expected IP '203.0.113.1', got '%s'", ip)
	}
}

func TestHandleGetCSRFToken(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "password123", "user", true)

	req := httptest.NewRequest("GET", "/api/v1/csrf-token", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleGetCSRFToken(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	csrfToken, ok := resp["csrf_token"].(string)
	if !ok || csrfToken == "" {
		t.Fatal("Expected CSRF token in response")
	}

	// Verify token in database
	var count int
	db.QueryRow("SELECT COUNT(*) FROM csrf_tokens WHERE token = ?", csrfToken).Scan(&count)
	if count != 1 {
		t.Error("Expected CSRF token to be stored in database")
	}
}

func TestHandleGetCSRFTokenUnauthorized(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/csrf-token", nil)
	w := httptest.NewRecorder()

	handleGetCSRFToken(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCSRFTokenCleansExpired(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	userID := createTestUser(t, "testuser", "password123", "user", true)

	// Create expired token
	expiredToken := generateToken()
	expiredTime := time.Now().Add(-2 * time.Hour)
	db.Exec("INSERT INTO csrf_tokens (token, user_id, expires_at) VALUES (?, ?, ?)",
		expiredToken, userID, expiredTime.Format("2006-01-02 15:04:05"))

	// Generate new token (should clean up expired)
	req := httptest.NewRequest("GET", "/api/v1/csrf-token", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handleGetCSRFToken(w, req)

	// Verify expired token was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM csrf_tokens WHERE token = ?", expiredToken).Scan(&count)
	if count != 0 {
		t.Error("Expected expired CSRF token to be cleaned up")
	}
}

func TestLoginCSRFToken(t *testing.T) {
	cleanup := setupAuthTestDB(t)
	defer cleanup()

	createTestUser(t, "testuser", "password123", "user", true)

	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	// CSRF token should be included in login response
	csrfToken, ok := resp["csrf_token"].(string)
	if !ok {
		t.Fatal("Expected csrf_token in login response")
	}
	if csrfToken == "" {
		t.Error("CSRF token should not be empty")
	}
}

// Helper functions

func withUsername(req *http.Request, username string) *http.Request {
	ctx := context.WithValue(req.Context(), ctxUsername, username)
	return req.WithContext(ctx)
}

func withUserID(req *http.Request, userID int) *http.Request {
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	return req.WithContext(ctx)
}

func stringReader(s string) *strings.Reader {
	return strings.NewReader(s)
}
