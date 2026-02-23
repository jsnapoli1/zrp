package admin_test

// Skipped: rbac_test.go tests the requireRBAC middleware defined in the root
// package (middleware.go). It verifies that the RBAC middleware correctly
// allows or denies requests based on the user role stored in the request
// context and the requested URL path. Because requireRBAC is a router-level
// middleware (not an admin handler method), this test belongs in the
// cross-cutting Batch 11 tests that stay in the root package.
