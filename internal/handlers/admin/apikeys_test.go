package admin_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"zrp/internal/handlers/admin"

	_ "modernc.org/sqlite"
)

func TestHandleListAPIKeys(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Insert test API keys with explicit timestamps to ensure order
	_, err := db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, created_by, enabled, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"Production Key", "hash1", "zrp_abc12345", "admin", 1, "2024-01-01T10:00:00Z")
	if err != nil {
		t.Fatalf("Failed to insert test key: %v", err)
	}

	_, err = db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, created_by, enabled, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"Test Key", "hash2", "zrp_def67890", "testuser", 0, "2025-12-31T23:59:59Z", "2024-01-02T10:00:00Z")
	if err != nil {
		t.Fatalf("Failed to insert test key: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/keys", nil)
	w := httptest.NewRecorder()

	h.ListAPIKeys(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []admin.APIKey `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Data) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(resp.Data))
	}

	// Verify keys are returned in descending order by created_at
	if resp.Data[0].Name != "Test Key" {
		t.Errorf("Expected first key to be 'Test Key', got '%s'", resp.Data[0].Name)
	}
}

func TestHandleListAPIKeys_Empty(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/keys", nil)
	w := httptest.NewRecorder()

	h.ListAPIKeys(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []admin.APIKey `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Data) != 0 {
		t.Errorf("Expected empty array, got %d keys", len(resp.Data))
	}
}

func TestHandleCreateAPIKey_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{"name": "New Production Key"}`
	req := httptest.NewRequest("POST", "/api/keys", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response contains the full key
	key, ok := resp["key"].(string)
	if !ok || key == "" {
		t.Error("Response should contain 'key' field with the generated key")
	}

	// Verify key format (zrp_ prefix + 32 hex chars = 36 total)
	if !strings.HasPrefix(key, "zrp_") {
		t.Error("Key should start with 'zrp_' prefix")
	}
	if len(key) != 36 {
		t.Errorf("Key should be 36 characters (zrp_ + 32 hex), got %d", len(key))
	}

	// Verify key_prefix is correct
	expectedPrefix := key[:12]
	if resp["key_prefix"] != expectedPrefix {
		t.Errorf("key_prefix should be '%s', got '%s'", expectedPrefix, resp["key_prefix"])
	}

	// Verify message about storing securely
	if !strings.Contains(resp["message"].(string), "Store this key securely") {
		t.Error("Response should warn about storing key securely")
	}

	// Verify key is stored in DB
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE name = ?", "New Production Key").Scan(&count)
	if err != nil || count != 1 {
		t.Error("Key should be stored in database")
	}

	// Verify key hash is stored, not plain key
	var keyHash string
	err = db.QueryRow("SELECT key_hash FROM api_keys WHERE name = ?", "New Production Key").Scan(&keyHash)
	if err != nil {
		t.Fatalf("Failed to retrieve key_hash: %v", err)
	}
	if keyHash == key {
		t.Error("Plain key should not be stored in database, only hash")
	}
	if len(keyHash) != 64 {
		t.Errorf("Hash should be 64 hex chars (SHA256), got %d", len(keyHash))
	}
}

func TestHandleCreateAPIKey_WithExpiration(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	expiresAt := "2026-12-31T23:59:59Z"
	reqBody := `{"name": "Expiring Key", "expires_at": "` + expiresAt + `"}`
	req := httptest.NewRequest("POST", "/api/keys", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Verify expiration is stored
	var storedExpiry string
	err := db.QueryRow("SELECT expires_at FROM api_keys WHERE name = ?", "Expiring Key").Scan(&storedExpiry)
	if err != nil {
		t.Fatalf("Failed to retrieve expires_at: %v", err)
	}
	if storedExpiry != expiresAt {
		t.Errorf("Expected expires_at '%s', got '%s'", expiresAt, storedExpiry)
	}
}

func TestHandleCreateAPIKey_EmptyName(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{"name": ""}`
	req := httptest.NewRequest("POST", "/api/keys", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for empty name, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(strings.ToLower(resp["error"].(string)), "name") {
		t.Error("Error message should mention 'name'")
	}
}

func TestHandleCreateAPIKey_InvalidJSON(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/api/keys", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandleDeleteAPIKey_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Insert test key
	result, err := db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix) VALUES (?, ?, ?)`,
		"Key to Delete", "hash123", "zrp_test1234")
	if err != nil {
		t.Fatalf("Failed to insert test key: %v", err)
	}
	id, _ := result.LastInsertId()

	req := httptest.NewRequest("DELETE", "/api/keys/"+fmt.Sprintf("%d", id), nil)
	w := httptest.NewRecorder()

	h.DeleteAPIKey(w, req, fmt.Sprintf("%d", id))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "revoked" {
		t.Errorf("Expected status 'revoked', got '%s'", resp["status"])
	}

	// Verify key is deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE id = ?", id).Scan(&count)
	if count != 0 {
		t.Error("Key should be deleted from database")
	}

	// Verify audit log entry
	var auditCount int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module = 'api_key' AND action = 'deleted'").Scan(&auditCount)
	if auditCount == 0 {
		t.Error("Delete should create audit log entry")
	}
}

func TestHandleDeleteAPIKey_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("DELETE", "/api/keys/999", nil)
	w := httptest.NewRecorder()

	h.DeleteAPIKey(w, req, "999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleToggleAPIKey_Enable(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Insert disabled key
	result, err := db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, enabled) VALUES (?, ?, ?, ?)`,
		"Disabled Key", "hash456", "zrp_test5678", 0)
	if err != nil {
		t.Fatalf("Failed to insert test key: %v", err)
	}
	id, _ := result.LastInsertId()
	idStr := fmt.Sprintf("%d", id)

	reqBody := `{"enabled": 1}`
	req := httptest.NewRequest("PATCH", "/api/keys/"+idStr, bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ToggleAPIKey(w, req, idStr)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify key is enabled
	var enabled int
	db.QueryRow("SELECT enabled FROM api_keys WHERE id = ?", id).Scan(&enabled)
	if enabled != 1 {
		t.Error("Key should be enabled")
	}

	// Verify audit log
	var auditCount int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module = 'api_key' AND action = 'enabled'").Scan(&auditCount)
	if auditCount == 0 {
		t.Error("Enable should create audit log entry")
	}
}

func TestHandleToggleAPIKey_Disable(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Insert enabled key
	result, err := db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, enabled) VALUES (?, ?, ?, ?)`,
		"Enabled Key", "hash789", "zrp_test9012", 1)
	if err != nil {
		t.Fatalf("Failed to insert test key: %v", err)
	}
	id, _ := result.LastInsertId()
	idStr := fmt.Sprintf("%d", id)

	reqBody := `{"enabled": 0}`
	req := httptest.NewRequest("PATCH", "/api/keys/"+idStr, bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ToggleAPIKey(w, req, idStr)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify key is disabled
	var enabled int
	db.QueryRow("SELECT enabled FROM api_keys WHERE id = ?", id).Scan(&enabled)
	if enabled != 0 {
		t.Error("Key should be disabled")
	}

	// Verify audit log
	var auditCount int
	db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE module = 'api_key' AND action = 'disabled'").Scan(&auditCount)
	if auditCount == 0 {
		t.Error("Disable should create audit log entry")
	}
}

func TestHandleToggleAPIKey_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{"enabled": 1}`
	req := httptest.NewRequest("PATCH", "/api/keys/999", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ToggleAPIKey(w, req, "999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleToggleAPIKey_InvalidJSON(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{invalid}`
	req := httptest.NewRequest("PATCH", "/api/keys/1", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ToggleAPIKey(w, req, "1")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestValidateBearerToken_Valid(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Generate a key and store its hash
	key := "zrp_0123456789abcdef0123456789abcdef"
	keyHash := admin.HashAPIKey(key)
	_, err := db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, enabled) VALUES (?, ?, ?, ?)`,
		"Valid Key", keyHash, key[:12], 1)
	if err != nil {
		t.Fatalf("Failed to insert test key: %v", err)
	}

	// Validate the key
	valid := h.ValidateBearerToken(key)
	if !valid {
		t.Error("Valid key should be accepted")
	}

	// Verify last_used was updated
	var lastUsed *string
	db.QueryRow("SELECT last_used FROM api_keys WHERE key_hash = ?", keyHash).Scan(&lastUsed)
	if lastUsed == nil {
		t.Error("last_used should be updated after successful validation")
	}
}

func TestValidateBearerToken_Disabled(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Generate a key and store its hash as disabled
	key := "zrp_fedcba9876543210fedcba9876543210"
	keyHash := admin.HashAPIKey(key)
	_, err := db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, enabled) VALUES (?, ?, ?, ?)`,
		"Disabled Key", keyHash, key[:12], 0)
	if err != nil {
		t.Fatalf("Failed to insert test key: %v", err)
	}

	valid := h.ValidateBearerToken(key)
	if valid {
		t.Error("Disabled key should be rejected")
	}
}

func TestValidateBearerToken_Expired(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Generate a key with past expiration
	key := "zrp_aabbccddeeff00112233445566778899"
	keyHash := admin.HashAPIKey(key)
	pastDate := time.Now().Add(-24 * time.Hour).Format("2006-01-02T15:04:05Z")
	_, err := db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, enabled, expires_at) VALUES (?, ?, ?, ?, ?)`,
		"Expired Key", keyHash, key[:12], 1, pastDate)
	if err != nil {
		t.Fatalf("Failed to insert test key: %v", err)
	}

	valid := h.ValidateBearerToken(key)
	if valid {
		t.Error("Expired key should be rejected")
	}
}

func TestValidateBearerToken_InvalidPrefix(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	key := "invalid_prefix_0123456789abcdef01234"
	valid := h.ValidateBearerToken(key)
	if valid {
		t.Error("Key without zrp_ prefix should be rejected")
	}
}

func TestValidateBearerToken_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	key := "zrp_99999999999999999999999999999999"
	valid := h.ValidateBearerToken(key)
	if valid {
		t.Error("Non-existent key should be rejected")
	}
}

func TestGenerateAPIKey_Randomness(t *testing.T) {
	// Generate multiple keys and verify they're all different
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := admin.GenerateAPIKey()
		if err != nil {
			t.Fatalf("Failed to generate key: %v", err)
		}
		if keys[key] {
			t.Error("Duplicate key generated - randomness failure")
		}
		keys[key] = true

		// Verify format
		if !strings.HasPrefix(key, "zrp_") {
			t.Error("Generated key should start with zrp_")
		}
		if len(key) != 36 {
			t.Errorf("Generated key should be 36 chars, got %d", len(key))
		}
	}
}

func TestHashAPIKey_Consistency(t *testing.T) {
	key := "zrp_testkey123456789abcdef01234567"
	hash1 := admin.HashAPIKey(key)
	hash2 := admin.HashAPIKey(key)

	if hash1 != hash2 {
		t.Error("Same key should produce same hash")
	}

	if len(hash1) != 64 {
		t.Errorf("SHA256 hash should be 64 hex chars, got %d", len(hash1))
	}
}

func TestHashAPIKey_DifferentKeys(t *testing.T) {
	key1 := "zrp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	key2 := "zrp_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	hash1 := admin.HashAPIKey(key1)
	hash2 := admin.HashAPIKey(key2)

	if hash1 == hash2 {
		t.Error("Different keys should produce different hashes")
	}
}

func TestAPIKeyPrefix_Storage(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{"name": "Prefix Test Key"}`
	req := httptest.NewRequest("POST", "/api/keys", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateAPIKey(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	fullKey := resp["key"].(string)
	expectedPrefix := fullKey[:12]

	// Verify prefix in DB matches first 12 chars of key
	var storedPrefix string
	db.QueryRow("SELECT key_prefix FROM api_keys WHERE name = ?", "Prefix Test Key").Scan(&storedPrefix)
	if storedPrefix != expectedPrefix {
		t.Errorf("Stored prefix '%s' doesn't match expected '%s'", storedPrefix, expectedPrefix)
	}
}

func TestValidateBearerToken_FutureExpiration(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Generate a key with future expiration
	key := "zrp_future9999999999999999999999999"
	keyHash := admin.HashAPIKey(key)
	futureDate := time.Now().Add(365 * 24 * time.Hour).Format("2006-01-02T15:04:05Z")
	_, err := db.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, enabled, expires_at) VALUES (?, ?, ?, ?, ?)`,
		"Future Key", keyHash, key[:12], 1, futureDate)
	if err != nil {
		t.Fatalf("Failed to insert test key: %v", err)
	}

	valid := h.ValidateBearerToken(key)
	if !valid {
		t.Error("Key with future expiration should be valid")
	}
}
