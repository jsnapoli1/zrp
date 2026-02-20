package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
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

// securityHeaders adds security headers to all responses
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")
		
		// Prevent MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		
		// Enable XSS protection (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self' data:; " +
			"connect-src 'self'"
		w.Header().Set("Content-Security-Policy", csp)
		
		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Permissions Policy (disable unnecessary features)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		// HSTS (if using HTTPS)
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		next.ServeHTTP(w, r)
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
		var lastActivity string
		err = db.QueryRow(`SELECT s.user_id, u.role, u.active, COALESCE(s.last_activity, s.created_at) FROM sessions s JOIN users u ON s.user_id = u.id
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
					// Session has been inactive too long - invalidate it
					db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
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

		// Sliding window: extend session expiry on each authenticated request
		newExpiry := time.Now().UTC().Add(24 * time.Hour)
		db.Exec("UPDATE sessions SET expires_at = ?, last_activity = ? WHERE token = ?",
			newExpiry.Format("2006-01-02 15:04:05"), time.Now().UTC().Format("2006-01-02 15:04:05"), cookie.Value)

		// Update cookie expiry to match
		http.SetCookie(w, &http.Cookie{
			Name:     "zrp_session",
			Value:    cookie.Value,
			Path:     "/",
			HttpOnly: true,
			Secure:   true, // Only transmit over HTTPS
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

// Rate limiting structures
type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
}

var globalRateLimiter = &rateLimiter{
	requests: make(map[string][]time.Time),
}

// resetRateLimiter clears all rate limit state (for testing)
func resetRateLimiter() {
	globalRateLimiter.mu.Lock()
	globalRateLimiter.requests = make(map[string][]time.Time)
	globalRateLimiter.mu.Unlock()
}

// cleanupOldRequests removes requests older than the window
func (rl *rateLimiter) cleanupOldRequests(key string, window time.Duration) {
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

// checkRateLimit checks if the request should be rate limited
func (rl *rateLimiter) checkRateLimit(key string, limit int, window time.Duration) (bool, int, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// Clean up old requests
	rl.cleanupOldRequests(key, window)
	
	// Get current request count
	requests := rl.requests[key]
	currentCount := len(requests)
	
	// Calculate reset time (oldest request + window)
	var resetTime time.Time
	if len(requests) > 0 {
		resetTime = requests[0].Add(window)
	} else {
		resetTime = now.Add(window)
	}
	
	// Check if limit exceeded
	if currentCount >= limit {
		return true, 0, resetTime
	}
	
	// Add current request
	rl.requests[key] = append(requests, now)
	remaining := limit - currentCount - 1
	
	return false, remaining, resetTime
}

// rateLimitMiddleware implements rate limiting per IP address
// Global limit: 100 requests per minute per IP
// Login endpoint: 5 requests per minute per IP
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP (handle X-Forwarded-For for proxies)
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = strings.Split(forwarded, ",")[0]
		}
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			clientIP = realIP
		}
		
		// Strip port from IP
		if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
			clientIP = clientIP[:idx]
		}
		
		path := r.URL.Path
		
		// Per-endpoint rate limits (stricter for sensitive endpoints)
		var limit int
		var window time.Duration
		var limitKey string
		
		// Login endpoint: 5 requests per minute
		if path == "/auth/login" {
			limit = 5
			window = time.Minute
			limitKey = "login:" + clientIP
		} else if strings.HasPrefix(path, "/api/") {
			// API endpoints: 100 requests per minute per IP
			limit = 100
			window = time.Minute
			limitKey = "api:" + clientIP
		} else {
			// Static assets and other routes: no rate limit
			next.ServeHTTP(w, r)
			return
		}
		
		// Check rate limit
		exceeded, remaining, resetTime := globalRateLimiter.checkRateLimit(limitKey, limit, window)
		
		// Set rate limit headers
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

// csrfMiddleware protects against Cross-Site Request Forgery (CSRF) attacks
// Requires X-CSRF-Token header on all state-changing operations (POST, PUT, DELETE)
func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only protect state-changing methods
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF for non-API routes
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF for login/logout (no existing session)
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF for API key authentication (Bearer tokens)
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		// Get CSRF token from header
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

		// Get user ID from session context
		// If not authenticated, requireAuth will have already handled it
		userID, ok := r.Context().Value(ctxUserID).(int)
		if !ok {
			// Try to get from session cookie directly for test scenarios
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
			
			// Get user ID from session
			err = db.QueryRow("SELECT user_id FROM sessions WHERE token = ? AND expires_at > CURRENT_TIMESTAMP", 
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

		// Validate CSRF token
		var tokenUserID int
		var expiresAt string
		err := db.QueryRow(`
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

		// Verify token belongs to the current user
		if tokenUserID != userID {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "CSRF token does not match user session",
				"code":  "CSRF_TOKEN_MISMATCH",
			})
			return
		}

		// CSRF token is valid, proceed with request
		next.ServeHTTP(w, r)
	})
}
