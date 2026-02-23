package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"zrp/internal/testutil"
)

// Wrapper functions delegating to internal/testutil.

func setupTestDB(t *testing.T) *sql.DB {
	return testutil.SetupTestDB(t)
}

func createTestUser(t *testing.T, db *sql.DB, username, password, role string, active bool) int {
	return testutil.CreateTestUser(t, db, username, password, role, active)
}

func createTestSessionSimple(t *testing.T, db *sql.DB, userID int) string {
	return testutil.CreateTestSessionSimple(t, db, userID)
}

func loginAdmin(t *testing.T, db *sql.DB) string {
	return testutil.LoginAdmin(t, db)
}

func loginUser(t *testing.T, db *sql.DB, username string) string {
	return testutil.LoginUser(t, db, username)
}

func authedRequest(method, path string, body []byte, sessionToken string) *http.Request {
	return testutil.AuthedRequest(method, path, body, sessionToken)
}

func authedJSONRequest(method, path string, body interface{}, sessionToken string) *http.Request {
	return testutil.AuthedJSONRequest(method, path, body, sessionToken)
}

func decodeAPIResponse(t *testing.T, w *httptest.ResponseRecorder) APIResponse {
	return testutil.DecodeAPIResponse(t, w)
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	testutil.AssertStatus(t, w, expected)
}
