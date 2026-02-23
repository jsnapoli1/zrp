package main

import (
	"net/http"
	"strings"
	"time"

	"zrp/internal/handlers/admin"
)

// Type aliases for backward compatibility.
type APIKey = admin.APIKey
type CreateAPIKeyRequest = admin.CreateAPIKeyRequest

func generateAPIKey() (string, error) {
	return admin.GenerateAPIKey()
}

func hashAPIKey(key string) string {
	return admin.HashAPIKey(key)
}

func handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().ListAPIKeys(w, r)
}

func handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().CreateAPIKey(w, r)
}

func handleDeleteAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	getAdminHandler().DeleteAPIKey(w, r, id)
}

func handleToggleAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	getAdminHandler().ToggleAPIKey(w, r, id)
}

// validateBearerToken checks an Authorization: Bearer token against the DB.
// Kept in root because it's used by requireAuth middleware before adminHandler is initialized.
func validateBearerToken(token string) bool {
	if !strings.HasPrefix(token, "zrp_") {
		return false
	}
	keyHash := admin.HashAPIKey(token)
	var id int
	var enabled int
	var expiresAt *string
	err := db.QueryRow("SELECT id, enabled, expires_at FROM api_keys WHERE key_hash = ?", keyHash).Scan(&id, &enabled, &expiresAt)
	if err != nil || enabled == 0 {
		return false
	}
	if expiresAt != nil && *expiresAt != "" {
		exp, err := time.Parse("2006-01-02T15:04:05Z", *expiresAt)
		if err != nil {
			exp, err = time.Parse("2006-01-02", *expiresAt)
		}
		if err == nil && time.Now().After(exp) {
			return false
		}
	}
	db.Exec("UPDATE api_keys SET last_used = ? WHERE id = ?", time.Now().Format("2006-01-02 15:04:05"), id)
	return true
}
