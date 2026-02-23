package admin_test

// Skipped: security_rate_limit_test.go tests the rateLimitMiddleware, the
// global rate limiter, per-endpoint limits, rate-limit headers
// (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, Retry-After),
// and the full middleware chain (securityHeaders, rateLimitMiddleware,
// gzipMiddleware, logging, requireAuth, requireRBAC). All of these are
// router-level middleware concerns, not admin handler methods. This test
// belongs in the cross-cutting Batch 11 tests that stay in the root package.
