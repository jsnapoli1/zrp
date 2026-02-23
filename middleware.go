package main

import (
	"net/http"

	"zrp/internal/server"
)

// Context key type and constants - aliases for backward compatibility.
type contextKey = server.ContextKey

const (
	ctxUserID   = server.CtxUserID
	ctxUsername = server.CtxUsername
	ctxRole     = server.CtxRole
)

// Type alias for gzipResponseWriter.
type gzipResponseWriter = server.GzipResponseWriter

// Global rate limiter instance.
var globalRateLimiter = server.NewRateLimiter()

func resetRateLimiter() {
	globalRateLimiter.Reset()
}

// Middleware wrapper functions delegating to internal/server.
func gzipMiddleware(next http.Handler) http.Handler {
	return server.GzipMiddleware(next)
}

func logging(next http.Handler) http.Handler {
	return server.LoggingMiddleware(next)
}

func securityHeaders(next http.Handler) http.Handler {
	return server.SecurityHeaders(next)
}

func requireAuth(next http.Handler) http.Handler {
	return server.RequireAuth(db, validateBearerToken)(next)
}

func isAdminOnly(apiPath string) bool {
	return server.IsAdminOnly(apiPath)
}

func requireRBAC(next http.Handler) http.Handler {
	return server.RequireRBAC(permCache)(next)
}

func rateLimitMiddleware(next http.Handler) http.Handler {
	return server.RateLimitMiddleware(globalRateLimiter)(next)
}

func csrfMiddleware(next http.Handler) http.Handler {
	return server.CSRFMiddleware(db)(next)
}
