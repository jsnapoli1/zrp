package server

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"zrp/internal/auth"
)

// GzipResponseWriter wraps http.ResponseWriter to support gzip compression.
type GzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w GzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// GzipMiddleware compresses responses when client supports gzip.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")

		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := GzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}

// LoggingMiddleware logs request method, path, and duration. Also sets CORS headers.
func LoggingMiddleware(next http.Handler) http.Handler {
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

// SecurityHeaders adds security headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self' data:; " +
			"connect-src 'self'"
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAuth returns an auth middleware that checks session cookies or Bearer tokens.
// bearerValidator is a function that validates a Bearer token (from api_keys).
func RequireAuth(dbConn *sql.DB, bearerValidator func(string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			if !strings.HasPrefix(path, "/api/") ||
				path == "/api/v1/openapi.json" {
				next.ServeHTTP(w, r)
				return
			}

			// Check Bearer token first
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if bearerValidator(token) {
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
			var lastActivity string
			err = dbConn.QueryRow(`SELECT s.user_id, u.role, u.active, COALESCE(s.last_activity, s.created_at) FROM sessions s JOIN users u ON s.user_id = u.id
				WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP`, cookie.Value).Scan(&userID, &role, &active, &lastActivity)
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

			// Check for inactivity timeout (30 minutes)
			if lastActivity != "" {
				lastActivityTime, err := time.Parse("2006-01-02 15:04:05", lastActivity)
				if err == nil {
					inactivityPeriod := time.Since(lastActivityTime)
					if inactivityPeriod > 30*time.Minute {
						dbConn.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
						if !strings.HasPrefix(path, "/api/") {
							http.Redirect(w, r, "/login", http.StatusSeeOther)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(401)
						json.NewEncoder(w).Encode(map[string]string{"error": "Session expired due to inactivity", "code": "SESSION_TIMEOUT"})
						return
					}
				}
			}

			if active == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(403)
				json.NewEncoder(w).Encode(map[string]string{"error": "Account deactivated", "code": "FORBIDDEN"})
				return
			}

			// Sliding window: extend session expiry
			newExpiry := time.Now().UTC().Add(24 * time.Hour)
			dbConn.Exec("UPDATE sessions SET expires_at = ?, last_activity = ? WHERE token = ?",
				newExpiry.Format("2006-01-02 15:04:05"), time.Now().UTC().Format("2006-01-02 15:04:05"), cookie.Value)

			http.SetCookie(w, &http.Cookie{
				Name:     "zrp_session",
				Value:    cookie.Value,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				Expires:  newExpiry,
			})

			ctx := context.WithValue(r.Context(), CtxUserID, userID)
			ctx = context.WithValue(ctx, CtxRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IsAdminOnly returns true if the API path is restricted to admin role.
func IsAdminOnly(apiPath string) bool {
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

// RequireRBAC enforces permission-based access control on /api/v1/ routes.
func RequireRBAC(pc *auth.PermCache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if !strings.HasPrefix(path, "/api/v1/") {
				next.ServeHTTP(w, r)
				return
			}

			role, _ := r.Context().Value(CtxRole).(string)
			if role == "" {
				next.ServeHTTP(w, r)
				return
			}

			method := r.Method
			apiPath := strings.TrimPrefix(path, "/api/v1/")
			apiPath = strings.TrimSuffix(apiPath, "/")

			module, action := auth.MapAPIPathToPermission(apiPath, method)

			if module == "" || action == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !pc.HasPermission(role, module, action) {
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
}

// RateLimiter tracks request rates per key.
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
	}
}

// Reset clears all rate limit state (for testing).
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	rl.requests = make(map[string][]time.Time)
	rl.mu.Unlock()
}

func (rl *RateLimiter) cleanupOldRequests(key string, window time.Duration) {
	now := time.Now()
	cutoff := now.Add(-window)

	requests := rl.requests[key]
	validRequests := make([]time.Time, 0)

	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	if len(validRequests) > 0 {
		rl.requests[key] = validRequests
	} else {
		delete(rl.requests, key)
	}
}

// CheckRateLimit checks if the request should be rate limited.
func (rl *RateLimiter) CheckRateLimit(key string, limit int, window time.Duration) (bool, int, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.cleanupOldRequests(key, window)

	requests := rl.requests[key]
	currentCount := len(requests)

	var resetTime time.Time
	if len(requests) > 0 {
		resetTime = requests[0].Add(window)
	} else {
		resetTime = now.Add(window)
	}

	if currentCount >= limit {
		return true, 0, resetTime
	}

	rl.requests[key] = append(requests, now)
	remaining := limit - currentCount - 1

	return false, remaining, resetTime
}

// RateLimitMiddleware implements rate limiting per IP address.
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = strings.Split(forwarded, ",")[0]
			}
			if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
				clientIP = realIP
			}

			if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
				clientIP = clientIP[:idx]
			}

			path := r.URL.Path

			var limit int
			var window time.Duration
			var limitKey string

			if path == "/auth/login" {
				limit = 5
				window = time.Minute
				limitKey = "login:" + clientIP
			} else if strings.HasPrefix(path, "/api/") {
				limit = 100
				window = time.Minute
				limitKey = "api:" + clientIP
			} else {
				next.ServeHTTP(w, r)
				return
			}

			exceeded, remaining, resetTime := rl.CheckRateLimit(limitKey, limit, window)

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			if exceeded {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      "Rate limit exceeded",
					"code":       "RATE_LIMIT_EXCEEDED",
					"retryAfter": int(time.Until(resetTime).Seconds()),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CSRFMiddleware protects against CSRF attacks.
func CSRFMiddleware(dbConn *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			if strings.HasPrefix(r.URL.Path, "/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}

			csrfToken := r.Header.Get("X-CSRF-Token")
			if csrfToken == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "CSRF token required",
					"code":  "CSRF_TOKEN_MISSING",
				})
				return
			}

			userID, ok := r.Context().Value(CtxUserID).(int)
			if !ok {
				cookie, err := r.Cookie("zrp_session")
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Unauthorized",
						"code":  "UNAUTHORIZED",
					})
					return
				}

				err = dbConn.QueryRow("SELECT user_id FROM sessions WHERE token = ? AND expires_at > CURRENT_TIMESTAMP",
					cookie.Value).Scan(&userID)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Unauthorized",
						"code":  "UNAUTHORIZED",
					})
					return
				}
			}

			var tokenUserID int
			var expiresAt string
			err := dbConn.QueryRow(`
				SELECT user_id, expires_at
				FROM csrf_tokens
				WHERE token = ? AND expires_at > CURRENT_TIMESTAMP`,
				csrfToken,
			).Scan(&tokenUserID, &expiresAt)

			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid or expired CSRF token",
					"code":  "CSRF_TOKEN_INVALID",
				})
				return
			}

			if tokenUserID != userID {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "CSRF token does not match user session",
					"code":  "CSRF_TOKEN_MISMATCH",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
