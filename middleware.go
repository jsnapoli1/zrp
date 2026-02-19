package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const (
	ctxUserID contextKey = "userID"
	ctxRole   contextKey = "role"
)

// gzipResponseWriter wraps http.ResponseWriter to support gzip compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// gzipMiddleware compresses responses when client supports gzip
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only compress if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Don't compress if response is already compressed or is a range request
		if r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // Let gzip calculate correct length

		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Exempt paths: all non-API routes (SPA handles its own auth),
		// plus specific API endpoints that don't require auth
		if !strings.HasPrefix(path, "/api/") ||
			path == "/api/v1/openapi.json" {
			next.ServeHTTP(w, r)
			return
		}

		// Check Bearer token first
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if validateBearerToken(token) {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid API key", "code": "UNAUTHORIZED"})
			return
		}

		// Check session cookie
		cookie, err := r.Cookie("zrp_session")
		if err != nil {
			if !strings.HasPrefix(path, "/api/") {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized", "code": "UNAUTHORIZED"})
			return
		}

		var userID int
		var role string
		var active int
		err = db.QueryRow(`SELECT s.user_id, u.role, u.active FROM sessions s JOIN users u ON s.user_id = u.id
			WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP`, cookie.Value).Scan(&userID, &role, &active)
		if err != nil {
			if !strings.HasPrefix(path, "/api/") {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized", "code": "UNAUTHORIZED"})
			return
		}

		if active == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(map[string]string{"error": "Account deactivated", "code": "FORBIDDEN"})
			return
		}

		// Sliding window: extend session expiry on each authenticated request
		newExpiry := time.Now().Add(24 * time.Hour)
		db.Exec("UPDATE sessions SET expires_at = ? WHERE token = ?",
			newExpiry.Format("2006-01-02 15:04:05"), cookie.Value)

		// Update cookie expiry to match
		http.SetCookie(w, &http.Cookie{
			Name:     "zrp_session",
			Value:    cookie.Value,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  newExpiry,
		})

		// Store user info in context
		ctx := context.WithValue(r.Context(), ctxUserID, userID)
		ctx = context.WithValue(ctx, ctxRole, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isAdminOnly returns true if the API path (after /api/v1/) is restricted to admin role.
// Kept for backward compatibility with tests; new code uses permission-based checks.
func isAdminOnly(apiPath string) bool {
	seg := strings.SplitN(apiPath, "/", 2)[0]
	switch seg {
	case "users", "apikeys", "api-keys":
		return true
	}
	if strings.HasPrefix(apiPath, "email/") || strings.HasPrefix(apiPath, "settings/email") {
		return true
	}
	return false
}

// requireRBAC enforces permission-based access control on /api/v1/ routes.
// It uses the role_permissions table via the permission cache.
func requireRBAC(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, "/api/v1/") {
			next.ServeHTTP(w, r)
			return
		}

		role, _ := r.Context().Value(ctxRole).(string)
		// Bearer tokens without a role context get full access (backward compat)
		if role == "" {
			next.ServeHTTP(w, r)
			return
		}

		method := r.Method
		apiPath := strings.TrimPrefix(path, "/api/v1/")
		apiPath = strings.TrimSuffix(apiPath, "/")

		// Map the API path to a module+action
		module, action := mapAPIPathToPermission(apiPath, method)

		// If no mapping exists, allow (passthrough routes like dashboard, search, etc.)
		if module == "" || action == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check permission
		if !HasPermission(role, module, action) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Permission denied",
				"code":  "FORBIDDEN",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
