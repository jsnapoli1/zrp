package admin_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// Test 1: Session Cookie Security Flags
func TestSessionCookie_SecurityFlags(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	createTestUserLocal(t, db, "testuser", "SecurePass123!", "user", true)

	reqBody := `{"username":"testuser","password":"SecurePass123!"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Code != 200 {
		t.Fatalf("Login failed with status %d: %s", w.Code, w.Body.String())
	}

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

	// Test HttpOnly flag
	if !sessionCookie.HttpOnly {
		t.Error("SECURITY: Session cookie MUST have HttpOnly flag to prevent XSS attacks")
	}

	// Test Secure flag
	if !sessionCookie.Secure {
		t.Error("SECURITY: Session cookie MUST have Secure flag to prevent transmission over HTTP")
	}

	// Test SameSite flag
	if sessionCookie.SameSite != http.SameSiteLaxMode && sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("SECURITY: Session cookie MUST have SameSite flag (Lax or Strict), got %v", sessionCookie.SameSite)
	}

	// Test that cookie has an expiration
	if sessionCookie.Expires.IsZero() {
		t.Error("SECURITY: Session cookie should have an expiration time")
	}

	// Test that expiration is reasonable (not too long)
	maxExpiry := time.Now().Add(48 * time.Hour)
	if sessionCookie.Expires.After(maxExpiry) {
		t.Errorf("SECURITY: Session cookie expiration too long (%v), should be <= 48 hours",
			sessionCookie.Expires.Sub(time.Now()))
	}
}

// Test 2: Session Fixation Attack Prevention
func TestSessionFixation_Prevention(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "SecurePass123!", "user", true)

	// Step 1: Attacker creates a session
	attackerToken := "attacker-token-" + time.Now().Format("20060102150405.000000")
	expires := time.Now().Add(24 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		attackerToken, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create attacker session: %v", err)
	}

	// Step 2: Victim logs in
	reqBody := `{"username":"testuser","password":"SecurePass123!"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: attackerToken})
	w := httptest.NewRecorder()
	h.HandleLogin(w, req)

	if w.Code != 200 {
		t.Fatalf("Login failed: %d", w.Code)
	}

	// Step 3: Verify new session token was generated
	cookies := w.Result().Cookies()
	var newSessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "zrp_session" {
			newSessionCookie = c
			break
		}
	}

	if newSessionCookie == nil {
		t.Fatal("No session cookie returned after login")
	}

	// CRITICAL: New token MUST be different from the old token
	if newSessionCookie.Value == attackerToken {
		t.Error("SECURITY: Session fixation vulnerability! Login MUST generate a new session token, not reuse existing one")
	}
}

// Test 3: Session Timeout After Inactivity
func TestSessionTimeout_InactivityPeriod(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "SecurePass123!", "user", true)

	// Create a session with last_activity set to 31 minutes ago (past 30-min timeout)
	inactiveToken := "inactive-token-" + time.Now().Format("20060102150405.000000")
	expires := time.Now().Add(24 * time.Hour)
	lastActivity := time.Now().Add(-31 * time.Minute)

	_, err := db.Exec(`INSERT INTO sessions (token, user_id, expires_at, last_activity)
		VALUES (?, ?, ?, ?)`,
		inactiveToken, userID, expires.Format("2006-01-02 15:04:05"),
		lastActivity.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create inactive session: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: inactiveToken})
	w := httptest.NewRecorder()

	h.HandleMe(w, req)

	// Should be rejected due to inactivity timeout
	if w.Code == 200 {
		t.Error("SECURITY: Session should be invalidated after 30 minutes of inactivity")
	}
}

// Test 4: Active Session Updates Last Activity
func TestSessionTimeout_ActivityUpdate(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "SecurePass123!", "user", true)

	// Create a session
	activeToken := "active-token-" + time.Now().Format("20060102150405.000000")
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	initialActivity := now.Add(-5 * time.Minute)

	_, err := db.Exec(`INSERT INTO sessions (token, user_id, created_at, expires_at, last_activity)
		VALUES (?, ?, ?, ?, ?)`,
		activeToken, userID, now.Format("2006-01-02 15:04:05"),
		expires.Format("2006-01-02 15:04:05"),
		initialActivity.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: activeToken})
	w := httptest.NewRecorder()

	h.HandleMe(w, req)

	if w.Code != 200 {
		t.Fatalf("handleMe failed with status %d: %s", w.Code, w.Body.String())
	}

	// Check if last_activity was updated
	var lastActivity string
	err = db.QueryRow("SELECT last_activity FROM sessions WHERE token = ?", activeToken).
		Scan(&lastActivity)
	if err != nil {
		t.Fatalf("Failed to query last_activity: %v", err)
	}

	var parsedActivity time.Time
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}
	for _, format := range formats {
		parsedActivity, err = time.Parse(format, lastActivity)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("Failed to parse last_activity '%s': %v", lastActivity, err)
	}

	if parsedActivity.Before(time.Now().UTC().Add(-1 * time.Minute)) {
		t.Error("SECURITY: last_activity should be updated on each authenticated request")
	}
}

// Test 5: Multiple Concurrent Sessions Per User
func TestConcurrentSessions_MultipleAllowed(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	createTestUserLocal(t, db, "testuser", "SecurePass123!", "user", true)

	// Login from first device
	reqBody := `{"username":"testuser","password":"SecurePass123!"}`
	req1 := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req1.RemoteAddr = "192.168.1.100:12345"
	w1 := httptest.NewRecorder()
	h.HandleLogin(w1, req1)

	if w1.Code != 200 {
		t.Fatalf("First login failed: %d", w1.Code)
	}

	var token1 string
	for _, c := range w1.Result().Cookies() {
		if c.Name == "zrp_session" {
			token1 = c.Value
			break
		}
	}

	// Login from second device
	req2 := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req2.RemoteAddr = "192.168.1.101:12345"
	w2 := httptest.NewRecorder()
	h.HandleLogin(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("Second login failed: %d", w2.Code)
	}

	var token2 string
	for _, c := range w2.Result().Cookies() {
		if c.Name == "zrp_session" {
			token2 = c.Value
			break
		}
	}

	// Tokens should be different
	if token1 == token2 {
		t.Error("Different logins should generate different session tokens")
	}

	// Both sessions should be valid
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token IN (?, ?)",
		token1, token2).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query sessions: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 concurrent sessions, found %d", count)
	}

	// Both sessions should work
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token1})
	w := httptest.NewRecorder()
	h.HandleMe(w, req)
	if w.Code != 200 {
		t.Error("First session should still be valid")
	}

	req = httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token2})
	w = httptest.NewRecorder()
	h.HandleMe(w, req)
	if w.Code != 200 {
		t.Error("Second session should be valid")
	}
}

// Test 6: Session Cleanup on Logout
func TestSessionCleanup_OnLogout(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "SecurePass123!", "user", true)

	// Create a session
	token := "logout-test-token-" + time.Now().Format("20060102150405.000000")
	expires := time.Now().Add(24 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&count)
	if count != 1 {
		t.Fatal("Session not created properly")
	}

	// Logout
	req := httptest.NewRequest("POST", "/api/v1/logout", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()
	h.HandleLogout(w, req)

	if w.Code != 200 {
		t.Fatalf("Logout failed: %d", w.Code)
	}

	// Verify session was deleted
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&count)
	if count != 0 {
		t.Error("SECURITY: Session should be deleted from database on logout")
	}

	// Verify cookie was invalidated
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "zrp_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Logout should return a cookie to clear the session")
	}

	if sessionCookie.MaxAge != -1 {
		t.Error("SECURITY: Logout cookie should have MaxAge=-1 to delete it")
	}
}

// Test 7: Expired Sessions Cannot Be Used
func TestExpiredSessions_CannotBeUsed(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "testuser", "SecurePass123!", "user", true)

	// Create an expired session
	expiredToken := "expired-token-" + time.Now().Format("20060102150405.000000")
	expires := time.Now().Add(-1 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		expiredToken, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create expired session: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: expiredToken})
	w := httptest.NewRecorder()

	h.HandleMe(w, req)

	if w.Code != 401 {
		t.Error("SECURITY: Expired sessions MUST be rejected")
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["code"] != "UNAUTHORIZED" {
		t.Errorf("Expected UNAUTHORIZED error, got %v", resp["code"])
	}
}
