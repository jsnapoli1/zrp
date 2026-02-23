package admin_test

import (
	"bytes"
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

func randomTestStr(n int) string {
	return fmt.Sprintf("%d", time.Now().UnixNano())[len(fmt.Sprintf("%d", time.Now().UnixNano()))-n:]
}

// TestPasswordHashing_BCryptUsed verifies passwords are hashed with bcrypt.
func TestPasswordHashing_BCryptUsed(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	username := "hashtest_" + randomTestStr(8)
	password := "SecurePassword123!"

	body := map[string]string{
		"username":     username,
		"password":     password,
		"display_name": "Hash Test",
		"role":         "user",
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != 201 {
		t.Fatalf("Failed to create user: %d - %s", w.Code, w.Body.String())
	}

	var storedHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&storedHash)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	if storedHash == password {
		t.Error("SECURITY FAIL: Password stored in plain text!")
	}

	if !strings.HasPrefix(storedHash, "$2") {
		t.Errorf("Password doesn't appear to be bcrypt: %s", storedHash[:10])
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		t.Errorf("Bcrypt comparison failed: %v", err)
	}
}

// TestPasswordComplexity_WeakPasswordsRejected tests weak password rejection.
func TestPasswordComplexity_WeakPasswordsRejected(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	weakPasswords := []string{
		"123456",
		"password",
		"short",
		"12345678901",
		"abcdefghabcd",
	}

	for _, pwd := range weakPasswords {
		t.Run(pwd, func(t *testing.T) {
			username := "weak_" + randomTestStr(6)
			body := map[string]string{
				"username":     username,
				"password":     pwd,
				"display_name": "Test",
				"role":         "user",
			}
			bodyJSON, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
			w := httptest.NewRecorder()

			h.CreateUser(w, req)

			if w.Code != 400 {
				t.Errorf("Weak password accepted! Status: %d", w.Code)
			}
		})
	}
}

// TestPasswordComplexity_StrongPasswordsAccepted tests strong password acceptance.
func TestPasswordComplexity_StrongPasswordsAccepted(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	strongPasswords := []string{
		"SecurePass123!",
		"MyP@ssw0rd2024",
		"C0mpl3x&Secure",
	}

	for _, pwd := range strongPasswords {
		t.Run(pwd, func(t *testing.T) {
			username := "strong_" + randomTestStr(6)
			body := map[string]string{
				"username":     username,
				"password":     pwd,
				"display_name": "Test",
				"role":         "user",
			}
			bodyJSON, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
			w := httptest.NewRecorder()

			h.CreateUser(w, req)

			if w.Code != 201 {
				t.Errorf("Strong password rejected: %s - %d - %s", pwd, w.Code, w.Body.String())
			}
		})
	}
}

// TestBruteForceProtection_AccountLockout tests account lockout via handler callbacks.
func TestBruteForceProtection_AccountLockout(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	h := newTestHandler(db)

	// Track failed attempts and simulate lockout
	failedAttempts := 0
	h.CheckLoginRateLimit = func(ip string) bool {
		return failedAttempts < 10
	}
	h.IncrementFailedLoginAttempts = func(username string) error {
		failedAttempts++
		return nil
	}
	h.IsAccountLocked = func(username string) (bool, error) {
		return failedAttempts >= 10, nil
	}

	username := "lockout_" + randomTestStr(8)
	password := "CorrectPassword123!"
	createTestUserLocal(t, db, username, password, "user", true)

	// Attempt 10 failed logins
	for i := 0; i < 10; i++ {
		body := map[string]string{
			"username": username,
			"password": "WrongPassword123!",
		}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		h.HandleLogin(w, req)
	}

	// Now try with correct password - should be locked
	body := map[string]string{
		"username": username,
		"password": password,
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Code != 429 && w.Code != 403 {
		t.Errorf("Account NOT locked! Status: %d (expected 429 or 403)", w.Code)
	}
}
