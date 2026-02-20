package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TestRateLimit_LoginEndpoint tests that the login endpoint has strict rate limiting
// Requirement: 5 requests per minute per IP for login endpoint
func TestRateLimit_LoginEndpoint(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	// Create test user with bcrypt hash
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	db.Exec("INSERT INTO users (username, password_hash, role, active) VALUES (?, ?, ?, ?)",
		"testuser", string(passwordHash), "user", 1)

	// Test with 10 rapid requests - should fail after 5th request
	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	successCount := 0
	rateLimitCount := 0

	for i := 1; i <= 10; i++ {
		body := `{"username":"testuser","password":"testpass123"}`
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.100:12345" // Consistent IP

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusOK || w.Code == http.StatusUnauthorized {
			successCount++
		} else if w.Code == http.StatusTooManyRequests {
			rateLimitCount++
			
			// Verify rate limit headers are present
			if w.Header().Get("X-RateLimit-Limit") == "" {
				t.Errorf("Request %d: Missing X-RateLimit-Limit header", i)
			}
			if w.Header().Get("X-RateLimit-Remaining") == "" {
				t.Errorf("Request %d: Missing X-RateLimit-Remaining header", i)
			}
			if w.Header().Get("X-RateLimit-Reset") == "" {
				t.Errorf("Request %d: Missing X-RateLimit-Reset header", i)
			}
			if w.Header().Get("Retry-After") == "" {
				t.Errorf("Request %d: Missing Retry-After header", i)
			}

			// Verify response body
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Errorf("Request %d: Failed to decode error response: %v", i, err)
			}
			if resp["code"] != "RATE_LIMIT_EXCEEDED" {
				t.Errorf("Request %d: Expected error code RATE_LIMIT_EXCEEDED, got %v", i, resp["code"])
			}
		}
	}

	// Should allow 5 requests, then rate limit the next 5
	if successCount != 5 {
		t.Errorf("Expected 5 successful requests, got %d", successCount)
	}
	if rateLimitCount != 5 {
		t.Errorf("Expected 5 rate-limited requests, got %d", rateLimitCount)
	}
}

// TestRateLimit_429Response tests that rate-limited requests return 429 status code
func TestRateLimit_429Response(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	// Make 6 requests to login endpoint (limit is 5)
	for i := 1; i <= 6; i++ {
		body := `{"username":"test","password":"test"}`
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.200:12345"

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if i <= 5 {
			// First 5 requests should succeed or return auth error (not rate limited)
			if w.Code == http.StatusTooManyRequests {
				t.Errorf("Request %d should not be rate limited, got status %d", i, w.Code)
			}
		} else {
			// 6th request should be rate limited
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d should return 429, got %d", i, w.Code)
			}
		}
	}
}

// TestRateLimit_GlobalAPILimit tests global API rate limit (100 req/min)
func TestRateLimit_GlobalAPILimit(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	// Create admin user for API access
	cookie := loginAdmin(t)

	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	// Test with 105 requests to API endpoint (limit is 100)
	successCount := 0
	rateLimitCount := 0

	for i := 1; i <= 105; i++ {
		req := httptest.NewRequest("GET", "/api/v1/dashboard", nil)
		req.AddCookie(cookie)
		req.RemoteAddr = "192.168.1.50:12345"

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			successCount++
		} else {
			rateLimitCount++
		}
	}

	// Should allow 100 requests, then rate limit the next 5
	if successCount != 100 {
		t.Errorf("Expected 100 successful API requests, got %d", successCount)
	}
	if rateLimitCount != 5 {
		t.Errorf("Expected 5 rate-limited API requests, got %d", rateLimitCount)
	}
}

// TestRateLimit_Headers tests that rate limit headers are present and correct
func TestRateLimit_Headers(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	// Make a request to login endpoint
	body := `{"username":"test","password":"test"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.75:12345"

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Check headers exist
	headers := []string{
		"X-RateLimit-Limit",
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
	}

	for _, header := range headers {
		if w.Header().Get(header) == "" {
			t.Errorf("Missing required rate limit header: %s", header)
		}
	}

	// Verify limit is 5 for login endpoint
	if limit := w.Header().Get("X-RateLimit-Limit"); limit != "5" {
		t.Errorf("Expected X-RateLimit-Limit to be 5, got %s", limit)
	}

	// Verify remaining is 4 after first request
	if remaining := w.Header().Get("X-RateLimit-Remaining"); remaining != "4" {
		t.Errorf("Expected X-RateLimit-Remaining to be 4, got %s", remaining)
	}
}

// TestRateLimit_ResetAfterWindow tests that rate limits reset after the time window
func TestRateLimit_ResetAfterWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping time-dependent test in short mode")
	}

	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	clientIP := "192.168.1.150:12345"

	// Make 5 requests to hit the limit
	for i := 1; i <= 5; i++ {
		body := `{"username":"test","password":"test"}`
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = clientIP

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("Request %d should not be rate limited yet", i)
		}
	}

	// 6th request should be rate limited
	req6 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"username":"test","password":"test"}`))
	req6.Header.Set("Content-Type", "application/json")
	req6.RemoteAddr = clientIP
	w6 := httptest.NewRecorder()
	handler.ServeHTTP(w6, req6)

	if w6.Code != http.StatusTooManyRequests {
		t.Fatalf("6th request should be rate limited, got status %d", w6.Code)
	}

	// Wait for rate limit window to expire (1 minute + buffer)
	t.Log("Waiting 65 seconds for rate limit window to reset...")
	time.Sleep(65 * time.Second)

	// After window expires, request should succeed
	req7 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"username":"test","password":"test"}`))
	req7.Header.Set("Content-Type", "application/json")
	req7.RemoteAddr = clientIP
	w7 := httptest.NewRecorder()
	handler.ServeHTTP(w7, req7)

	if w7.Code == http.StatusTooManyRequests {
		t.Errorf("After rate limit window reset, request should not be rate limited, got status %d", w7.Code)
	}

	// Verify remaining count is reset
	if remaining := w7.Header().Get("X-RateLimit-Remaining"); remaining != "4" {
		t.Errorf("After reset, expected X-RateLimit-Remaining to be 4, got %s", remaining)
	}
}

// TestRateLimit_DifferentIPsIndependent tests that different IPs have independent rate limits
func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	// IP 1: Make 5 requests (hit limit)
	for i := 1; i <= 5; i++ {
		body := `{"username":"test","password":"test"}`
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.168.1.100:12345"

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// IP 1: 6th request should be rate limited
	req1 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"username":"test","password":"test"}`))
	req1.Header.Set("Content-Type", "application/json")
	req1.RemoteAddr = "192.168.1.100:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusTooManyRequests {
		t.Errorf("IP 1 should be rate limited, got status %d", w1.Code)
	}

	// IP 2: First request should NOT be rate limited (independent limit)
	req2 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"username":"test","password":"test"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "192.168.1.200:12345"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code == http.StatusTooManyRequests {
		t.Errorf("IP 2 should not be rate limited (independent from IP 1), got status %d", w2.Code)
	}

	// Verify IP 2 has full quota
	if remaining := w2.Header().Get("X-RateLimit-Remaining"); remaining != "4" {
		t.Errorf("IP 2 should have 4 remaining requests, got %s", remaining)
	}
}

// TestRateLimit_ForwardedForHeader tests that X-Forwarded-For header is respected for rate limiting
func TestRateLimit_ForwardedForHeader(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	// Make 5 requests with X-Forwarded-For header (same IP)
	for i := 1; i <= 5; i++ {
		body := `{"username":"test","password":"test"}`
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "10.0.0.50")
		req.RemoteAddr = "192.168.1.1:12345" // Proxy IP (should be ignored)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// 6th request with same X-Forwarded-For should be rate limited
	req6 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"username":"test","password":"test"}`))
	req6.Header.Set("Content-Type", "application/json")
	req6.Header.Set("X-Forwarded-For", "10.0.0.50")
	req6.RemoteAddr = "192.168.1.1:12345"
	w6 := httptest.NewRecorder()
	handler.ServeHTTP(w6, req6)

	if w6.Code != http.StatusTooManyRequests {
		t.Errorf("Request with same X-Forwarded-For should be rate limited, got status %d", w6.Code)
	}
}

// TestRateLimit_PerEndpointLimits tests that different endpoints have different rate limits
func TestRateLimit_PerEndpointLimits(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	// Create admin user for API access
	cookie := loginAdmin(t)

	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	clientIP := "192.168.1.99:12345"

	// Make 5 login requests (hit login limit of 5)
	for i := 1; i <= 5; i++ {
		body := `{"username":"test","password":"test"}`
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = clientIP

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// 6th login request should be rate limited
	loginReq := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"username":"test","password":"test"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.RemoteAddr = clientIP
	loginW := httptest.NewRecorder()
	handler.ServeHTTP(loginW, loginReq)

	if loginW.Code != http.StatusTooManyRequests {
		t.Errorf("Login endpoint should be rate limited after 5 requests, got status %d", loginW.Code)
	}

	// API endpoint should still work (has separate limit of 100)
	apiReq := httptest.NewRequest("GET", "/api/v1/dashboard", nil)
	apiReq.AddCookie(cookie)
	apiReq.RemoteAddr = clientIP
	apiW := httptest.NewRecorder()
	handler.ServeHTTP(apiW, apiReq)

	if apiW.Code == http.StatusTooManyRequests {
		t.Errorf("API endpoint should not be rate limited (separate limit), got status %d", apiW.Code)
	}
}

// TestRateLimit_StaticAssetsNotLimited tests that static assets are not rate limited
func TestRateLimit_StaticAssetsNotLimited(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	clientIP := "192.168.1.88:12345"

	// Make 150 requests to root path (should not be rate limited)
	for i := 1; i <= 150; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = clientIP

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			t.Errorf("Static asset request %d should not be rate limited, got status %d", i, w.Code)
			break
		}
	}
}

// TestRateLimit_RetryAfterHeader tests that Retry-After header is set correctly
func TestRateLimit_RetryAfterHeader(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	resetRateLimiter()

	mux := setupTestMux()
	handler := securityHeaders(rateLimitMiddleware(gzipMiddleware(logging(requireAuth(requireRBAC(mux))))))

	clientIP := "192.168.1.77:12345"

	// Make 5 requests to hit the limit
	for i := 1; i <= 5; i++ {
		body := `{"username":"test","password":"test"}`
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = clientIP

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// 6th request should include Retry-After header
	req6 := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"username":"test","password":"test"}`))
	req6.Header.Set("Content-Type", "application/json")
	req6.RemoteAddr = clientIP
	w6 := httptest.NewRecorder()
	handler.ServeHTTP(w6, req6)

	if w6.Code != http.StatusTooManyRequests {
		t.Fatalf("Expected 429 status, got %d", w6.Code)
	}

	retryAfter := w6.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-After header is missing")
	}

	// Retry-After should be a positive number (seconds)
	var seconds int
	if _, err := fmt.Sscanf(retryAfter, "%d", &seconds); err != nil {
		t.Errorf("Retry-After header should be a number, got %s", retryAfter)
	}
	if seconds <= 0 || seconds > 60 {
		t.Errorf("Retry-After should be between 1-60 seconds, got %d", seconds)
	}
}

// Helper function to set up test mux
func setupTestMux() *http.ServeMux {
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			handleLogin(w, r)
		} else {
			http.Error(w, "Method not allowed", 405)
		}
	})

	// API routes
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
		path = strings.TrimSuffix(path, "/")

		if path == "dashboard" && r.Method == "GET" {
			handleDashboard(w, r)
		} else {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	})

	// Root handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return mux
}
