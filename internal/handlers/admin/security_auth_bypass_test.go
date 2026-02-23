package admin_test

// Skipped: security_auth_bypass_test.go tests the requireAuth middleware
// defined in the root package (middleware.go). It verifies that unauthenticated,
// expired-session, invalid-token, inactive-user, and SQL-injection requests are
// rejected at the middleware level. It also exercises the full middleware chain
// (requireAuth + requireRBAC) for admin-endpoint access control and tests
// Bearer-token (API key) validation through the middleware. Because these are
// all router-level middleware concerns (not admin handler methods), this test
// belongs in the cross-cutting Batch 11 tests that stay in the root package.
