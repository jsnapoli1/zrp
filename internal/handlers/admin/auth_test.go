package admin_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zrp/internal/handlers/admin"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func TestHandleLoginSuccess(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	createTestUserLocal(t, db, "testuser", "password123", "user", true)

	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	createTestUserLocal(t, db, "testuser", "password123", "user", true)

	reqBody := `{"username":"nonexistent","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleLoginInvalidPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Use a handler that tracks failed attempts in DB
	h.IncrementFailedLoginAttempts = func(username string) error {
		_, err := db.Exec("UPDATE users SET failed_login_attempts = failed_login_attempts + 1 WHERE username = ?", username)
		return err
	}

	createTestUserLocal(t, db, "testuser", "password123", "user", true)

	reqBody := `{"username":"testuser","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	createTestUserLocal(t, db, "testuser", "password123", "user", false)

	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Code != 403 {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestHandleLoginRateLimiting(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	createTestUserLocal(t, db, "testuser", "password123", "user", true)

	callCount := 0
	h.CheckLoginRateLimit = func(ip string) bool {
		callCount++
		return callCount <= 5
	}

	// Make 5 login attempts
	for i := 0; i < 5; i++ {
		reqBody := `{"username":"testuser","password":"password123"}`
		req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		h.HandleLogin(w, req)

		if w.Code != 200 {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	// 6th attempt should be rate limited
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	h.HandleLogin(w, req)

	if w.Code != 429 {
		t.Errorf("Expected status 429 (rate limited), got %d", w.Code)
	}
}

func TestHandleLoginInvalidJSON(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleLoginCleansExpiredSessions(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "password123", "user", true)

	// Create expired session
	expiredToken := "expired-token-123"
	expiredTime := time.Now().Add(-2 * time.Hour)
	db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		expiredToken, userID, expiredTime.Format("2006-01-02 15:04:05"))

	// Login should clean up expired session
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	h.HandleLogin(w, req)

	// Verify expired session was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", expiredToken).Scan(&count)
	if count != 0 {
		t.Error("Expected expired session to be cleaned up")
	}
}

func TestHandleLogoutSuccess(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "password123", "user", true)
	token := createTestSessionLocal(t, db, userID)

	req := httptest.NewRequest("POST", "/api/v1/logout", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	h.HandleLogout(w, req)

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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("POST", "/api/v1/logout", nil)
	w := httptest.NewRecorder()

	h.HandleLogout(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleMeValidSession(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "password123", "admin", true)
	token := createTestSessionLocal(t, db, userID)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	h.HandleMe(w, req)

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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	w := httptest.NewRecorder()

	h.HandleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleMeInvalidToken(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: "invalid-token"})
	w := httptest.NewRecorder()

	h.HandleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleMeExpiredSession(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "password123", "user", true)
	token := "expired-session-token"
	expires := time.Now().Add(-1 * time.Hour)
	db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	h.HandleMe(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for expired session, got %d", w.Code)
	}
}

func TestHandleMeInactivityTimeout(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "password123", "user", true)
	token := "inactive-session-token"
	expires := time.Now().Add(24 * time.Hour)
	lastActivity := time.Now().Add(-31 * time.Minute).Format("2006-01-02 15:04:05")

	db.Exec("INSERT INTO sessions (token, user_id, expires_at, last_activity) VALUES (?, ?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"), lastActivity)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	h.HandleMe(w, req)

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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "password123", "user", true)
	token := "activity-session-token"
	expires := time.Now().UTC().Add(24 * time.Hour)
	oldActivity := time.Now().UTC().Add(-5 * time.Minute).Format("2006-01-02 15:04:05")

	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at, last_activity) VALUES (?, ?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"), oldActivity)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	h.HandleMe(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify last_activity was updated
	var newActivity string
	db.QueryRow("SELECT last_activity FROM sessions WHERE token = ?", token).Scan(&newActivity)

	if newActivity == oldActivity {
		t.Error("Expected last_activity to be updated")
	}
}

func TestHandleChangePasswordSuccess(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "OldPassword123!", "user", true)

	reqBody := `{"current_password":"OldPassword123!","new_password":"NewPassword123!"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleChangePassword(w, req)

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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "OldPassword123!", "user", true)

	reqBody := `{"current_password":"WrongPassword123!","new_password":"NewPassword123!"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleChangePassword(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestHandleChangePasswordTooShort(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "OldPassword123!", "user", true)

	reqBody := `{"current_password":"OldPassword123!","new_password":"short"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleChangePassword(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleChangePasswordMissingFields(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "password", "user", true)

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
			ctx := context.WithValue(req.Context(), admin.CtxUserID, userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			h.HandleChangePassword(w, req)

			if w.Code != 400 {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleChangePasswordUnauthorized(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{"current_password":"old","new_password":"new"}`
	req := httptest.NewRequest("POST", "/api/v1/change-password", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.HandleChangePassword(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestGetClientIPDirect(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	ip := admin.GetClientIP(req)

	if ip != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got '%s'", ip)
	}
}

func TestGetClientIPXForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	ip := admin.GetClientIP(req)

	if ip != "203.0.113.1" {
		t.Errorf("Expected IP '203.0.113.1', got '%s'", ip)
	}
}

func TestHandleGetCSRFToken(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	// Use a real token generator for this test
	h.GenerateToken = func() string {
		return "csrf-test-token-" + time.Now().Format("20060102150405.000000")
	}

	userID := createTestUserLocal(t, db, "testuser", "password123", "user", true)

	req := httptest.NewRequest("GET", "/api/v1/csrf-token", nil)
	ctx := context.WithValue(req.Context(), admin.CtxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleGetCSRFToken(w, req)

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
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/csrf-token", nil)
	w := httptest.NewRecorder()

	h.HandleGetCSRFToken(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestCSRFTokenCleansExpired(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	h.GenerateToken = func() string {
		return "csrf-new-token-" + time.Now().Format("20060102150405.000000")
	}

	userID := createTestUserLocal(t, db, "testuser", "password123", "user", true)

	// Create expired token
	expiredToken := "expired-csrf-token"
	expiredTime := time.Now().Add(-2 * time.Hour)
	db.Exec("INSERT INTO csrf_tokens (token, user_id, expires_at) VALUES (?, ?, ?)",
		expiredToken, userID, expiredTime.Format("2006-01-02 15:04:05"))

	// Generate new token (should clean up expired)
	req := httptest.NewRequest("GET", "/api/v1/csrf-token", nil)
	ctx := context.WithValue(req.Context(), admin.CtxUserID, userID)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.HandleGetCSRFToken(w, req)

	// Verify expired token was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM csrf_tokens WHERE token = ?", expiredToken).Scan(&count)
	if count != 0 {
		t.Error("Expected expired CSRF token to be cleaned up")
	}
}

func TestLoginCSRFToken(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)
	h.GenerateToken = func() string {
		return "login-csrf-token-" + time.Now().Format("20060102150405.000000")
	}

	createTestUserLocal(t, db, "testuser", "password123", "user", true)

	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

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
