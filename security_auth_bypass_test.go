package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// setupSecurityTestDB creates a test database with users, sessions, and permissions
func setupSecurityTestDB(t *testing.T) *sql.DB {
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
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1
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

	// Create api_keys table
	_, err = testDB.Exec(`
		CREATE TABLE api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			created_by TEXT DEFAULT 'admin',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_used DATETIME,
			expires_at DATETIME,
			enabled INTEGER DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create api_keys table: %v", err)
	}

	// Create role_permissions table
	_, err = testDB.Exec(`
		CREATE TABLE role_permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			role TEXT NOT NULL,
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			granted INTEGER DEFAULT 1,
			UNIQUE(role, module, action)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create role_permissions table: %v", err)
	}

	return testDB
}

// createSecurityTestUser creates a test user with specified role and active status
func createSecurityTestUser(t *testing.T, db *sql.DB, username, password, role string, active bool) int {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	activeInt := 0
	if active {
		activeInt = 1
	}

	result, err := db.Exec(
		"INSERT INTO users (username, password_hash, role, active) VALUES (?, ?, ?, ?)",
		username, string(hash), role, activeInt,
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// createSecurityTestSession creates a session token for a user
func createSecurityTestSession(t *testing.T, db *sql.DB, userID int, expiresIn time.Duration) string {
	token := "test-session-" + time.Now().Format("20060102150405.000000")
	expiresAt := time.Now().Add(expiresIn)

	_, err := db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return token
}

// TestAuthBypass_NoAuthProvided tests that endpoints reject requests with no authentication
func TestAuthBypass_NoAuthProvided(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Protected API endpoints that should require authentication
	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/parts"},
		{"POST", "/api/v1/parts"},
		{"GET", "/api/v1/parts/123"},
		{"PUT", "/api/v1/parts/123"},
		{"DELETE", "/api/v1/parts/123"},
		{"GET", "/api/v1/ecos"},
		{"POST", "/api/v1/ecos"},
		{"GET", "/api/v1/ecos/1"},
		{"PUT", "/api/v1/ecos/1"},
		{"GET", "/api/v1/vendors"},
		{"POST", "/api/v1/vendors"},
		{"GET", "/api/v1/inventory"},
		{"POST", "/api/v1/inventory/transact"},
		{"GET", "/api/v1/pos"},
		{"POST", "/api/v1/pos"},
		{"GET", "/api/v1/workorders"},
		{"POST", "/api/v1/workorders"},
		{"GET", "/api/v1/docs"},
		{"POST", "/api/v1/docs"},
		{"GET", "/api/v1/ncrs"},
		{"POST", "/api/v1/ncrs"},
		{"GET", "/api/v1/capas"},
		{"POST", "/api/v1/capas"},
		{"GET", "/api/v1/rmas"},
		{"POST", "/api/v1/rmas"},
		{"GET", "/api/v1/quotes"},
		{"POST", "/api/v1/quotes"},
		{"GET", "/api/v1/devices"},
		{"POST", "/api/v1/devices"},
	}

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))

	for _, endpoint := range protectedEndpoints {
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != 401 {
				t.Errorf("Expected 401 Unauthorized for %s %s without auth, got %d", 
					endpoint.method, endpoint.path, w.Code)
			}

			var response map[string]string
			if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
				if response["code"] != "UNAUTHORIZED" {
					t.Errorf("Expected error code UNAUTHORIZED, got %s", response["code"])
				}
			}
		})
	}
}

// TestAuthBypass_ExpiredSession tests that expired sessions are rejected
func TestAuthBypass_ExpiredSession(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createSecurityTestUser(t, db, "testuser", "password123", "user", true)
	expiredToken := createSecurityTestSession(t, db, userID, -1*time.Hour) // Expired 1 hour ago

	endpoints := []string{
		"/api/v1/parts",
		"/api/v1/ecos",
		"/api/v1/vendors",
		"/api/v1/inventory",
		"/api/v1/workorders",
	}

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for _, path := range endpoints {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			req.AddCookie(&http.Cookie{Name: "zrp_session", Value: expiredToken})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != 401 {
				t.Errorf("Expected 401 for expired session on %s, got %d", path, w.Code)
			}
		})
	}
}

// TestAuthBypass_InvalidSession tests that invalid/forged session tokens are rejected
func TestAuthBypass_InvalidSession(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	invalidTokens := []string{
		"invalid-token-12345",
		"forged-session-token",
		"'; DROP TABLE sessions; --",
		"../../../etc/passwd",
		"<script>alert('xss')</script>",
		"",
		strings.Repeat("a", 1000), // Very long token
	}

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for _, token := range invalidTokens {
		t.Run("Token: "+token[:min(len(token), 30)], func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts", nil)
			req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != 401 {
				t.Errorf("Expected 401 for invalid token, got %d", w.Code)
			}
		})
	}
}

// TestAuthBypass_InactiveUser tests that sessions from inactive users are rejected
func TestAuthBypass_InactiveUser(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create inactive user
	userID := createSecurityTestUser(t, db, "inactive_user", "password123", "user", false)
	token := createSecurityTestSession(t, db, userID, 24*time.Hour)

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("Expected 403 Forbidden for inactive user, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
		if !strings.Contains(response["error"], "deactivated") {
			t.Errorf("Expected error about deactivated account, got %s", response["error"])
		}
	}
}

// TestAuthBypass_CrossUserSession tests that users can't use another user's session
func TestAuthBypass_CrossUserSession(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create two users
	user1ID := createSecurityTestUser(t, db, "user1", "password123", "user", true)
	user2ID := createSecurityTestUser(t, db, "user2", "password123", "user", true)
	
	// Create sessions for both
	token1 := createSecurityTestSession(t, db, user1ID, 24*time.Hour)
	_ = createSecurityTestSession(t, db, user2ID, 24*time.Hour)

	// This test verifies that the session token is properly validated
	// In a real scenario, you would test that user2 can't steal user1's session cookie
	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(ctxUserID)
		if userID != user1ID {
			t.Errorf("Session context has wrong user ID")
		}
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token1})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 with valid session, got %d", w.Code)
	}
}

// TestAuthBypass_AdminEndpoints tests that admin-only endpoints reject non-admin users
func TestAuthBypass_AdminEndpoints(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Seed default permissions for admin and user roles
	if err := seedDefaultPermissions(); err != nil {
		t.Fatalf("Failed to seed permissions: %v", err)
	}
	if err := refreshPermCache(); err != nil {
		t.Fatalf("Failed to refresh permission cache: %v", err)
	}

	// Create regular user and admin user
	regularUserID := createSecurityTestUser(t, db, "regular_user", "password123", "user", true)
	adminUserID := createSecurityTestUser(t, db, "admin_user", "password123", "admin", true)
	
	regularToken := createSecurityTestSession(t, db, regularUserID, 24*time.Hour)
	adminToken := createSecurityTestSession(t, db, adminUserID, 24*time.Hour)

	// Admin-only endpoints
	adminEndpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/users"},
		{"POST", "/api/v1/users"},
		{"PUT", "/api/v1/users/1"},
		{"DELETE", "/api/v1/users/1"},
		{"PUT", "/api/v1/users/1/password"},
		{"GET", "/api/v1/apikeys"},
		{"POST", "/api/v1/apikeys"},
		{"DELETE", "/api/v1/apikeys/1"},
		{"GET", "/api/v1/api-keys"},
		{"POST", "/api/v1/api-keys"},
		{"GET", "/api/v1/email/config"},
		{"PUT", "/api/v1/email/config"},
	}

	// Build the full middleware chain
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	
	handler := requireAuth(requireRBAC(mux))

	for _, endpoint := range adminEndpoints {
		t.Run("RegularUser_"+endpoint.method+"_"+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			req.AddCookie(&http.Cookie{Name: "zrp_session", Value: regularToken})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// For admin-only endpoints, regular users should get 403 Forbidden
			// The RBAC middleware checks permissions, and if the user doesn't have
			// permission for the module/action, they get 403
			if w.Code != 403 && w.Code != 401 {
				t.Errorf("Expected 403/401 for regular user accessing %s %s, got %d. Body: %s", 
					endpoint.method, endpoint.path, w.Code, w.Body.String())
			}
		})

		t.Run("Admin_"+endpoint.method+"_"+endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Admin should have access (200) or might get other errors for invalid requests,
			// but should NOT get 403 Forbidden
			if w.Code == 403 {
				t.Errorf("Admin user should have access to %s %s, got 403", endpoint.method, endpoint.path)
			}
		})
	}
}

// TestAuthBypass_InvalidBearerToken tests that invalid API keys are rejected
func TestAuthBypass_InvalidBearerToken(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	invalidAPIKeys := []string{
		"invalid-api-key",
		"Bearer token",
		"zrp_invalid_key",
		"",
		strings.Repeat("a", 1000),
	}

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for _, key := range invalidAPIKeys {
		t.Run("Key: "+key[:min(len(key), 30)], func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts", nil)
			req.Header.Set("Authorization", "Bearer "+key)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != 401 {
				t.Errorf("Expected 401 for invalid API key, got %d", w.Code)
			}
		})
	}
}

// TestAuthBypass_ValidBearerToken tests that valid API keys are accepted
func TestAuthBypass_ValidBearerToken(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create a valid API key
	apiKey := "zrp_test_secure_api_key_12345"
	keyHash := hashAPIKey(apiKey)
	_, err := db.Exec(
		"INSERT INTO api_keys (key_hash, key_prefix, name, enabled) VALUES (?, ?, ?, ?)",
		keyHash, "zrp_test", "Test Key", 1,
	)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 for valid API key, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestAuthBypass_DisabledAPIKey tests that disabled API keys are rejected
func TestAuthBypass_DisabledAPIKey(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create a disabled API key
	apiKey := "zrp_test_disabled_key_99999"
	keyHash := hashAPIKey(apiKey)
	_, err := db.Exec(
		"INSERT INTO api_keys (key_hash, key_prefix, name, enabled) VALUES (?, ?, ?, ?)",
		keyHash, "zrp_test", "Disabled Key", 0, // enabled = 0
	)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401 for disabled API key, got %d", w.Code)
	}
}

// TestAuthBypass_OpenAPIExemption tests that OpenAPI endpoint is publicly accessible
func TestAuthBypass_OpenAPIExemption(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"openapi": "3.0.0"})
	}))

	req := httptest.NewRequest("GET", "/api/v1/openapi.json", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected OpenAPI endpoint to be publicly accessible, got %d", w.Code)
	}
}

// TestAuthBypass_SessionSlidingWindow tests that sessions are extended on use
func TestAuthBypass_SessionSlidingWindow(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createSecurityTestUser(t, db, "testuser", "password123", "user", true)
	token := createSecurityTestSession(t, db, userID, 24*time.Hour)

	// Get initial expiry timestamp
	var initialExpiry string
	err := db.QueryRow("SELECT expires_at FROM sessions WHERE token=?", token).Scan(&initialExpiry)
	if err != nil {
		t.Fatalf("Failed to get initial expiry: %v", err)
	}

	// Try multiple time formats since SQLite can store timestamps in different formats
	initialTime, err := parseFlexibleTime(initialExpiry)
	if err != nil {
		t.Fatalf("Failed to parse initial expiry '%s': %v", initialExpiry, err)
	}

	// Wait to ensure we're in a different second
	time.Sleep(1500 * time.Millisecond)

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Get updated expiry
	var updatedExpiry string
	err = db.QueryRow("SELECT expires_at FROM sessions WHERE token=?", token).Scan(&updatedExpiry)
	if err != nil {
		t.Fatalf("Failed to get updated expiry: %v", err)
	}

	updatedTime, err := parseFlexibleTime(updatedExpiry)
	if err != nil {
		t.Fatalf("Failed to parse updated expiry '%s': %v", updatedExpiry, err)
	}

	// The middleware should extend the session to 24h from now
	// Since we slept ~1.5s, the updated time should be ~1.5s after the initial time
	// (both are set to 24h from their respective creation times)
	timeDiff := updatedTime.Sub(initialTime).Seconds()
	if timeDiff < 1 {
		t.Errorf("Expected session expiry to be updated by at least 1 second (sliding window), got %.2f seconds", timeDiff)
	}
}

// parseFlexibleTime tries multiple time formats commonly used in SQLite
func parseFlexibleTime(timeStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.999999999Z07:00",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time '%s' with any known format", timeStr)
}

// TestAuthBypass_SQLInjectionAttempts tests that SQL injection attempts in auth are handled safely
func TestAuthBypass_SQLInjectionAttempts(t *testing.T) {
	oldDB := db
	db = setupSecurityTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	sqlInjectionTokens := []string{
		"' OR '1'='1",
		"'; DROP TABLE sessions; --",
		"admin'--",
		"' UNION SELECT * FROM users--",
		"1' OR 1=1--",
	}

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for _, token := range sqlInjectionTokens {
		t.Run("SQLi: "+token, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts", nil)
			req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should be rejected as invalid session
			if w.Code != 401 {
				t.Errorf("Expected 401 for SQL injection attempt, got %d", w.Code)
			}

			// Verify sessions table still exists
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
			if err != nil {
				t.Errorf("Sessions table was affected by SQL injection attempt: %v", err)
			}
		})
	}
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
