package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// --- WebSocket Endpoint Tests ---

func TestWebSocketUpgrade(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Start test server with just the ws handler
	srv := httptest.NewServer(http.HandlerFunc(handleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Should be registered in hub
	wsHub.mu.RLock()
	count := len(wsHub.clients)
	wsHub.mu.RUnlock()
	if count < 1 {
		t.Errorf("expected at least 1 client in hub, got %d", count)
	}
}

func TestWebSocketBroadcastFormat(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(handleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer conn.Close()

	// Give registration a moment
	time.Sleep(50 * time.Millisecond)

	// Broadcast an event
	wsHub.Broadcast(WSEvent{Type: "eco_updated", ID: 42, Action: "update"})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	var evt WSEvent
	if err := json.Unmarshal(msg, &evt); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if evt.Type != "eco_updated" {
		t.Errorf("expected type eco_updated, got %s", evt.Type)
	}
	if evt.Action != "update" {
		t.Errorf("expected action update, got %s", evt.Action)
	}
}

func TestWebSocketAuthRequiredViaMiddleware(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create server with auth middleware wrapping ws handler
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/ws", handleWebSocket)
	srv := httptest.NewServer(requireAuth(mux))
	defer srv.Close()

	// Try connecting without auth cookie — should get 401 redirect or rejection
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected connection to fail without auth")
	}
	if resp != nil && resp.StatusCode != 401 {
		// The middleware returns 401 for /api/ paths
		t.Logf("got status %d (expected 401 for unauthenticated ws)", resp.StatusCode)
	}
}

// --- WebSocket Hub Tests ---

func TestHubClientRegistration(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(handleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial 1 failed: %v", err)
	}
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial 2 failed: %v", err)
	}
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	wsHub.mu.RLock()
	count := len(wsHub.clients)
	wsHub.mu.RUnlock()
	if count < 2 {
		t.Errorf("expected at least 2 clients, got %d", count)
	}
}

func TestHubBroadcastToMultipleClients(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(handleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	conn1, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	defer conn1.Close()
	conn2, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	wsHub.Broadcast(WSEvent{Type: "part_created", ID: "IPN-001", Action: "create"})

	for i, conn := range []*websocket.Conn{conn1, conn2} {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("client %d: read failed: %v", i, err)
		}
		var evt WSEvent
		json.Unmarshal(msg, &evt)
		if evt.Type != "part_created" {
			t.Errorf("client %d: expected part_created, got %s", i, evt.Type)
		}
	}
}

func TestHubDisconnectCleanup(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(handleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	conn, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	time.Sleep(50 * time.Millisecond)

	wsHub.mu.RLock()
	before := len(wsHub.clients)
	wsHub.mu.RUnlock()

	conn.Close()
	// Give the read loop time to detect closure and unregister
	time.Sleep(200 * time.Millisecond)

	wsHub.mu.RLock()
	after := len(wsHub.clients)
	wsHub.mu.RUnlock()

	if after >= before {
		t.Errorf("expected client count to decrease after disconnect: before=%d after=%d", before, after)
	}
}

// --- Auth: bcrypt password verification ---

func TestBcryptPasswordVerification(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resetLoginRateLimit()

	// Correct password
	body := `{"username":"admin","password":"changeme"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleLogin(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 for correct password, got %d", w.Code)
	}

	// Wrong password
	body2 := `{"username":"admin","password":"wrongpassword"}`
	req2 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handleLogin(w2, req2)
	if w2.Code != 401 {
		t.Errorf("expected 401 for wrong password, got %d", w2.Code)
	}
}

// --- Auth: session expiry ---

func TestSessionExpiry(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)

	// Manually expire the session
	db.Exec("UPDATE sessions SET expires_at = '2000-01-01 00:00:00' WHERE token = ?", cookie.Value)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handleMe(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401 for expired session, got %d", w.Code)
	}
}

// --- Auth: sliding window ---

func TestSessionSlidingWindow(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)

	// Set the session expiry to something clearly in the past-ish (but still valid)
	// so that when the middleware extends it, the new value is clearly different
	shortExpiry := time.Now().Add(1 * time.Hour).Format("2006-01-02 15:04:05")
	db.Exec("UPDATE sessions SET expires_at = ? WHERE token = ?", shortExpiry, cookie.Value)

	// Make an authenticated request through the middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := requireAuth(mux)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check that expiry was extended to ~24h from now (much later than the 1h we set)
	var newExpiry string
	db.QueryRow("SELECT expires_at FROM sessions WHERE token = ?", cookie.Value).Scan(&newExpiry)

	if newExpiry <= shortExpiry {
		t.Errorf("session expiry should have been extended: orig=%s new=%s", shortExpiry, newExpiry)
	}
}

// --- Auth: logout invalidates session ---

func TestLogoutInvalidatesSession(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)

	// Verify session works
	req := httptest.NewRequest("GET", "/auth/me", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	handleMe(w, req)
	if w.Code != 200 {
		t.Fatalf("session should be valid before logout, got %d", w.Code)
	}

	// Logout
	req2 := httptest.NewRequest("POST", "/auth/logout", nil)
	req2.AddCookie(cookie)
	w2 := httptest.NewRecorder()
	handleLogout(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("logout should return 200, got %d", w2.Code)
	}

	// Session should now be invalid
	req3 := httptest.NewRequest("GET", "/auth/me", nil)
	req3.AddCookie(cookie)
	w3 := httptest.NewRecorder()
	handleMe(w3, req3)
	if w3.Code != 401 {
		t.Errorf("expected 401 after logout, got %d", w3.Code)
	}

	// Verify session deleted from DB
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", cookie.Value).Scan(&count)
	if count != 0 {
		t.Errorf("session should be deleted from DB, found %d", count)
	}
}

// --- Auth: change password ---

func TestChangePasswordRequiresCurrentPassword(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	var userID int
	db.QueryRow("SELECT user_id FROM sessions WHERE token = ?", cookie.Value).Scan(&userID)

	// Wrong current password
	body := `{"current_password":"WrongPassword123!","new_password":"NewPassword123!"}`
	req := httptest.NewRequest("POST", "/auth/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), ctxUserID, userID))
	w := httptest.NewRecorder()
	handleChangePassword(w, req)
	if w.Code != 401 {
		t.Errorf("expected 401 for wrong current password, got %d", w.Code)
	}
}

func TestChangePasswordMinLength(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	var userID int
	db.QueryRow("SELECT user_id FROM sessions WHERE token = ?", cookie.Value).Scan(&userID)

	// Too short new password
	body := `{"current_password":"changeme","new_password":"short"}`
	req := httptest.NewRequest("POST", "/auth/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), ctxUserID, userID))
	w := httptest.NewRecorder()
	handleChangePassword(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400 for short password, got %d", w.Code)
	}
}

func TestChangePasswordSuccess(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	cookie := loginAdmin(t)
	var userID int
	db.QueryRow("SELECT user_id FROM sessions WHERE token = ?", cookie.Value).Scan(&userID)

	body := `{"current_password":"changeme","new_password":"NewPassword123!"}`
	req := httptest.NewRequest("POST", "/auth/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), ctxUserID, userID))
	w := httptest.NewRecorder()
	handleChangePassword(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Old password should no longer work
	resetLoginRateLimit()
	body2 := `{"username":"admin","password":"changeme"}`
	req2 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handleLogin(w2, req2)
	if w2.Code != 401 {
		t.Errorf("old password should fail, got %d", w2.Code)
	}

	// New password should work
	body3 := `{"username":"admin","password":"NewPassword123!"}`
	req3 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	handleLogin(w3, req3)
	if w3.Code != 200 {
		t.Errorf("new password should work, got %d", w3.Code)
	}
}

// --- Auth: rate limiting ---

func TestLoginRateLimiting(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resetLoginRateLimit()

	// Make 5 failed attempts
	for i := 0; i < 5; i++ {
		body := `{"username":"admin","password":"wrong"}`
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handleLogin(w, req)
		if w.Code == 429 {
			t.Fatalf("got 429 too early on attempt %d", i+1)
		}
	}

	// 6th attempt should be rate limited
	body := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleLogin(w, req)
	if w.Code != 429 {
		t.Errorf("expected 429 after 5 attempts, got %d", w.Code)
	}

	// Even correct credentials should be blocked
	body2 := `{"username":"admin","password":"changeme"}`
	req2 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handleLogin(w2, req2)
	if w2.Code != 429 {
		t.Errorf("expected 429 for correct creds when rate limited, got %d", w2.Code)
	}
}

// --- broadcast helper test ---

func TestBroadcastHelper(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	srv := httptest.NewServer(http.HandlerFunc(handleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	broadcast("eco", "update", 1)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	var evt WSEvent
	json.Unmarshal(msg, &evt)
	if evt.Type != "eco_updated" {
		t.Errorf("expected eco_updated, got %s", evt.Type)
	}
	if evt.Action != "update" {
		t.Errorf("expected action update, got %s", evt.Action)
	}
}
