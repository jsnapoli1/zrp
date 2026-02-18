package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

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

		// Exempt paths
		if path == "/" || path == "/index.html" ||
			strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/auth/") ||
			strings.HasPrefix(path, "/files/") ||
			path == "/api/v1/openapi.json" ||
			strings.HasPrefix(path, "/docs") {
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

		// Readonly role enforcement
		if role == "readonly" && strings.HasPrefix(path, "/api/v1/") {
			method := r.Method
			if method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(403)
				json.NewEncoder(w).Encode(map[string]string{"error": "Read-only access", "code": "FORBIDDEN"})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
