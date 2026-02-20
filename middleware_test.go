package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func setupMiddlewareTestDB(t *testing.T) *sql.DB {
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

func createMiddlewareTestUser(t *testing.T, db *sql.DB, username, password, role string, active bool) int {
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

func createTestSession(t *testing.T, db *sql.DB, userID int, expiresIn time.Duration) string {
	token := "test-session-token-" + time.Now().Format("20060102150405")
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

func TestLoggingMiddleware_CORS(t *testing.T) {
	handler := logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS header to be set")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected CORS methods header to be set")
	}
}

func TestLoggingMiddleware_OPTIONS(t *testing.T) {
	handler := logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called for OPTIONS")
	}))

	req := httptest.NewRequest("OPTIONS", "/api/v1/parts", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
	}
}

func TestRequireAuth_NoAuth(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireAuth_ValidSession(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createMiddlewareTestUser(t, db, "testuser", "password123", "user", true)
	token := createTestSession(t, db, userID, 24*time.Hour)

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 with valid session, got %d", w.Code)
	}
}

func TestRequireAuth_ExpiredSession(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createMiddlewareTestUser(t, db, "testuser", "password123", "user", true)
	token := createTestSession(t, db, userID, -1*time.Hour) // Expired

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for expired session, got %d", w.Code)
	}
}

func TestRequireAuth_InactiveUser(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createMiddlewareTestUser(t, db, "testuser", "password123", "user", false) // Inactive
	token := createTestSession(t, db, userID, 24*time.Hour)

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("Expected status 403 for inactive user, got %d", w.Code)
	}
}

func TestRequireAuth_ValidBearerToken(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create API key in the correct format
	apiKey := "zrp_test_api_key_12345"
	keyHash := hashAPIKey(apiKey)
	db.Exec("INSERT INTO api_keys (key_hash, key_prefix, name, enabled) VALUES (?, ?, ?, ?)", keyHash, "zrp_test", "Test Key", 1)

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200 with valid API key, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestRequireAuth_InvalidBearerToken(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.Header.Set("Authorization", "Bearer invalid-api-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for invalid API key, got %d", w.Code)
	}
}

func TestRequireAuth_OpenAPIExempted(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/openapi.json", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected OpenAPI endpoint to be exempted, got %d", w.Code)
	}
}

func TestRequireAuth_NonAPIExempted(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected non-API routes to be exempted, got %d", w.Code)
	}
}

func TestRequireAuth_SessionExtension(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createMiddlewareTestUser(t, db, "testuser", "password123", "user", true)
	token := createTestSession(t, db, userID, 24*time.Hour)

	// Get initial expiry
	var initialExpiry string
	db.QueryRow("SELECT expires_at FROM sessions WHERE token=?", token).Scan(&initialExpiry)

	time.Sleep(1 * time.Second)

	handler := requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Get updated expiry
	var updatedExpiry string
	db.QueryRow("SELECT expires_at FROM sessions WHERE token=?", token).Scan(&updatedExpiry)

	if updatedExpiry == initialExpiry {
		t.Error("Expected session expiry to be updated (sliding window)")
	}
}

func TestRequireRBAC_NoRole(t *testing.T) {
	oldDB := db
	db = setupMiddlewareTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	handler := requireRBAC(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/parts", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// No role in context should allow access (bearer token passthrough)
	if w.Code != 200 {
		t.Errorf("Expected status 200 for no role (passthrough), got %d", w.Code)
	}
}

func TestRequireRBAC_NonAPIExempted(t *testing.T) {
	handler := requireRBAC(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected non-API routes to be exempted, got %d", w.Code)
	}
}

func TestIsAdminOnly_Users(t *testing.T) {
	if !isAdminOnly("users") {
		t.Error("Expected 'users' to be admin-only")
	}
}

func TestIsAdminOnly_APIKeys(t *testing.T) {
	if !isAdminOnly("apikeys") {
		t.Error("Expected 'apikeys' to be admin-only")
	}
	if !isAdminOnly("api-keys") {
		t.Error("Expected 'api-keys' to be admin-only")
	}
}

func TestIsAdminOnly_Email(t *testing.T) {
	if !isAdminOnly("email/subscriptions") {
		t.Error("Expected 'email/subscriptions' to be admin-only")
	}
}

func TestIsAdminOnly_NotAdminPath(t *testing.T) {
	if isAdminOnly("parts") {
		t.Error("Expected 'parts' to not be admin-only")
	}
	if isAdminOnly("ecos") {
		t.Error("Expected 'ecos' to not be admin-only")
	}
}
