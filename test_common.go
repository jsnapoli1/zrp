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

// setupTestDB creates a standard in-memory SQLite database for testing
// with foreign keys enabled and common tables created
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create core users table
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
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
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	return testDB
}

// createTestUser creates a test user with the given credentials
func createTestUser(t *testing.T, db *sql.DB, username, password, role string, active bool) int {
	t.Helper()
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

// createTestSessionSimple creates a session token for the given user with default 24h expiry
// Note: Some test files may have their own createTestSession with custom duration parameter
func createTestSessionSimple(t *testing.T, db *sql.DB, userID int) string {
	t.Helper()
	token := "test-session-token-" + time.Now().Format("20060102150405.000000")
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

// loginAdmin creates an admin user and returns their session token
func loginAdmin(t *testing.T, db *sql.DB) string {
	t.Helper()
	adminID := createTestUser(t, db, "admin", "password", "admin", true)
	return createTestSessionSimple(t, db, adminID)
}

// loginUser creates a regular user and returns their session token
func loginUser(t *testing.T, db *sql.DB, username string) string {
	t.Helper()
	userID := createTestUser(t, db, username, "password", "user", true)
	return createTestSessionSimple(t, db, userID)
}

// authedRequest creates an authenticated HTTP request with a session cookie
func authedRequest(method, path string, body []byte, sessionToken string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	
	if sessionToken != "" {
		req.AddCookie(&http.Cookie{Name: "zrp_session", Value: sessionToken})
	}
	
	return req
}

// authedJSONRequest creates an authenticated HTTP request with JSON content type
func authedJSONRequest(method, path string, body interface{}, sessionToken string) *http.Request {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	
	req := authedRequest(method, path, bodyBytes, sessionToken)
	req.Header.Set("Content-Type", "application/json")
	
	return req
}

// decodeAPIResponse decodes an APIResponse from a ResponseRecorder
func decodeAPIResponse(t *testing.T, w *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var response APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode API response: %v", err)
	}
	return response
}

// assertStatus checks that the HTTP status code matches expected
func assertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("Expected status %d, got %d. Body: %s", expected, w.Code, w.Body.String())
	}
}
