package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// setupPermissionsTestDB creates an in-memory test database with all required tables
func setupPermissionsTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create role_permissions table
	_, err = testDB.Exec(`
		CREATE TABLE role_permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			role TEXT NOT NULL,
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			UNIQUE(role, module, action)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create role_permissions table: %v", err)
	}

	// Create users table (needed for context validation)
	_, err = testDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			email TEXT,
			role TEXT DEFAULT 'user' CHECK(role IN ('admin','user','readonly')),
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	return testDB
}

// seedDefaultPermissionsForTest seeds the default 3-role permission structure
func seedDefaultPermissionsForTest(t *testing.T, db *sql.DB) {
	stmt, err := db.Prepare("INSERT OR IGNORE INTO role_permissions (role, module, action) VALUES (?, ?, ?)")
	if err != nil {
		t.Fatalf("Failed to prepare permissions insert: %v", err)
	}
	defer stmt.Close()

	// Admin: everything
	for _, mod := range AllModules {
		for _, act := range AllActions {
			if _, err := stmt.Exec("admin", mod, act); err != nil {
				t.Fatalf("Failed to insert admin permission: %v", err)
			}
		}
	}

	// User: everything except admin module
	for _, mod := range AllModules {
		if mod == ModuleAdmin {
			continue
		}
		for _, act := range AllActions {
			if _, err := stmt.Exec("user", mod, act); err != nil {
				t.Fatalf("Failed to insert user permission: %v", err)
			}
		}
	}

	// Readonly: view only
	for _, mod := range AllModules {
		if _, err := stmt.Exec("readonly", mod, ActionView); err != nil {
			t.Fatalf("Failed to insert readonly permission: %v", err)
		}
	}
}

// insertTestUser creates a test user
func insertTestUser(t *testing.T, db *sql.DB, username, role string) int {
	result, err := db.Exec(
		"INSERT INTO users (username, password_hash, role, active) VALUES (?, ?, ?, 1)",
		username, "dummy_hash", role,
	)
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

// =============================================================================
// TEST: handleListPermissions - GET /api/v1/permissions
// =============================================================================

func TestHandleListPermissions_EmptyDatabase(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	w := httptest.NewRecorder()

	handleListPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []Permission `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Data) != 0 {
		t.Errorf("Expected empty array, got %d permissions", len(resp.Data))
	}
}

func TestHandleListPermissions_AllRoles(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)

	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	w := httptest.NewRecorder()

	handleListPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []Permission `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Calculate expected count:
	// admin: 19 modules * 5 actions = 95
	// user: 18 modules (all except admin) * 5 actions = 90
	// readonly: 19 modules * 1 action (view) = 19
	// Total: 95 + 90 + 19 = 204
	expectedCount := 204
	if len(resp.Data) != expectedCount {
		t.Errorf("Expected %d permissions, got %d", expectedCount, len(resp.Data))
	}

	// Verify structure of first permission
	if len(resp.Data) > 0 {
		p := resp.Data[0]
		if p.Role == "" {
			t.Error("Permission missing role")
		}
		if p.Module == "" {
			t.Error("Permission missing module")
		}
		if p.Action == "" {
			t.Error("Permission missing action")
		}
	}
}

func TestHandleListPermissions_FilterByRole(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)

	tests := []struct {
		role          string
		expectedCount int
	}{
		{"admin", 95},    // 19 modules * 5 actions
		{"user", 90},     // 18 modules * 5 actions
		{"readonly", 19}, // 19 modules * 1 action
		{"invalid", 0},   // Non-existent role
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/permissions?role="+tt.role, nil)
			w := httptest.NewRecorder()

			handleListPermissions(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var resp struct {
				Data []Permission `json:"data"`
			}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(resp.Data) != tt.expectedCount {
				t.Errorf("Expected %d permissions for role %s, got %d", tt.expectedCount, tt.role, len(resp.Data))
			}

			// Verify all returned permissions have the correct role
			for _, p := range resp.Data {
				if p.Role != tt.role {
					t.Errorf("Expected role %s, got %s", tt.role, p.Role)
				}
			}
		})
	}
}

func TestHandleListPermissions_OrderedByRoleModuleAction(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Insert permissions in random order
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('user', 'parts', 'edit')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('admin', 'ecos', 'view')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('readonly', 'parts', 'view')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('admin', 'ecos', 'create')")

	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	w := httptest.NewRecorder()

	handleListPermissions(w, req)

	var resp struct {
		Data []Permission `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	// Verify they're ordered: admin < readonly < user (alphabetically)
	// Within same role: ecos < parts (alphabetically)
	// Within same module: create < edit < view (alphabetically)
	if len(resp.Data) != 4 {
		t.Fatalf("Expected 4 permissions, got %d", len(resp.Data))
	}

	// First should be admin, ecos, create
	if resp.Data[0].Role != "admin" || resp.Data[0].Module != "ecos" || resp.Data[0].Action != "create" {
		t.Errorf("Expected first to be admin/ecos/create, got %s/%s/%s", resp.Data[0].Role, resp.Data[0].Module, resp.Data[0].Action)
	}

	// Second should be admin, ecos, view
	if resp.Data[1].Role != "admin" || resp.Data[1].Module != "ecos" || resp.Data[1].Action != "view" {
		t.Errorf("Expected second to be admin/ecos/view, got %s/%s/%s", resp.Data[1].Role, resp.Data[1].Module, resp.Data[1].Action)
	}
}

// =============================================================================
// TEST: handleListModules - GET /api/v1/permissions/modules
// =============================================================================

func TestHandleListModules_Success(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/permissions/modules", nil)
	w := httptest.NewRecorder()

	handleListModules(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []struct {
			Module  string   `json:"module"`
			Actions []string `json:"actions"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	modules := resp.Data

	// Should return all modules
	expectedModuleCount := len(AllModules)
	if len(modules) != expectedModuleCount {
		t.Errorf("Expected %d modules, got %d", expectedModuleCount, len(modules))
	}

	// Each module should have all actions
	expectedActionCount := len(AllActions)
	for _, mod := range modules {
		if len(mod.Actions) != expectedActionCount {
			t.Errorf("Module %s: expected %d actions, got %d", mod.Module, expectedActionCount, len(mod.Actions))
		}

		// Verify all actions are present
		for _, expectedAction := range AllActions {
			found := false
			for _, action := range mod.Actions {
				if action == expectedAction {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Module %s: missing action %s", mod.Module, expectedAction)
			}
		}
	}
}

func TestHandleListModules_ContainsExpectedModules(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	req := httptest.NewRequest("GET", "/api/v1/permissions/modules", nil)
	w := httptest.NewRecorder()

	handleListModules(w, req)

	var resp struct {
		Data []struct {
			Module  string   `json:"module"`
			Actions []string `json:"actions"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	modules := resp.Data

	// Check that critical modules are present
	criticalModules := []string{ModuleParts, ModuleAdmin, ModuleECOs, ModuleInventory}
	for _, criticalMod := range criticalModules {
		found := false
		for _, mod := range modules {
			if mod.Module == criticalMod {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Critical module %s not found in response", criticalMod)
		}
	}
}

// =============================================================================
// TEST: handleMyPermissions - GET /api/v1/permissions/me
// =============================================================================

func TestHandleMyPermissions_AdminRole(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()
	refreshPermCache()

	req := httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	ctx := context.WithValue(req.Context(), ctxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleMyPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var perms []Permission
	if err := json.NewDecoder(w.Body).Decode(&perms); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Admin should have all permissions
	expectedCount := len(AllModules) * len(AllActions)
	if len(perms) != expectedCount {
		t.Errorf("Expected %d permissions for admin, got %d", expectedCount, len(perms))
	}
}

func TestHandleMyPermissions_UserRole(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()
	refreshPermCache()

	req := httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	ctx := context.WithValue(req.Context(), ctxRole, "user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleMyPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var perms []Permission
	if err := json.NewDecoder(w.Body).Decode(&perms); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// User should have permissions on all modules except admin
	expectedCount := (len(AllModules) - 1) * len(AllActions)
	if len(perms) != expectedCount {
		t.Errorf("Expected %d permissions for user, got %d", expectedCount, len(perms))
	}

	// Verify no admin module permissions
	for _, p := range perms {
		if p.Module == ModuleAdmin {
			t.Error("User should not have admin module permissions")
		}
	}
}

func TestHandleMyPermissions_ReadonlyRole(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()
	refreshPermCache()

	req := httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	ctx := context.WithValue(req.Context(), ctxRole, "readonly")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleMyPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var perms []Permission
	if err := json.NewDecoder(w.Body).Decode(&perms); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Readonly should only have view permissions
	expectedCount := len(AllModules)
	if len(perms) != expectedCount {
		t.Errorf("Expected %d permissions for readonly, got %d", expectedCount, len(perms))
	}

	// Verify all are view actions
	for _, p := range perms {
		if p.Action != ActionView {
			t.Errorf("Readonly should only have view action, got %s", p.Action)
		}
	}
}

func TestHandleMyPermissions_BearerToken_NoRole(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()
	refreshPermCache()

	// Bearer token auth with no role in context
	req := httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	w := httptest.NewRecorder()

	handleMyPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var perms []Permission
	if err := json.NewDecoder(w.Body).Decode(&perms); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Bearer tokens get all permissions
	expectedCount := len(AllModules) * len(AllActions)
	if len(perms) != expectedCount {
		t.Errorf("Expected %d permissions for bearer token, got %d", expectedCount, len(perms))
	}
}

func TestHandleMyPermissions_CustomRole(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Create custom role with limited permissions
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('custom', 'parts', 'view')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('custom', 'parts', 'create')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('custom', 'inventory', 'view')")

	initPermCache()
	refreshPermCache()

	req := httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	ctx := context.WithValue(req.Context(), ctxRole, "custom")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleMyPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var perms []Permission
	if err := json.NewDecoder(w.Body).Decode(&perms); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(perms) != 3 {
		t.Errorf("Expected 3 permissions for custom role, got %d", len(perms))
	}
}

// =============================================================================
// TEST: handleSetPermissions - PUT /api/v1/permissions/:role
// =============================================================================

func TestHandleSetPermissions_Success(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()
	refreshPermCache()

	// Set new permissions for a custom role
	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "view"},
			{"module": "parts", "action": "create"},
			{"module": "inventory", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/supervisor", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "supervisor")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "updated" {
		t.Errorf("Expected status 'updated', got '%s'", resp["status"])
	}

	// Verify permissions were saved
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "supervisor").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count permissions: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 permissions saved, got %d", count)
	}

	// Verify specific permissions
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ? AND module = ? AND action = ?",
		"supervisor", "parts", "view").Scan(&exists)
	if exists != 1 {
		t.Error("Expected parts:view permission to exist")
	}
}

func TestHandleSetPermissions_ReplacesExisting(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initPermCache()

	// Insert initial permissions
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('manager', 'parts', 'view')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('manager', 'parts', 'edit')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('manager', 'inventory', 'view')")

	// Replace with new permissions
	reqBody := `{
		"permissions": [
			{"module": "ecos", "action": "view"},
			{"module": "ecos", "action": "approve"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/manager", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "manager")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify old permissions are gone
	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "manager").Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 permissions after replacement, got %d", count)
	}

	// Verify new permissions exist
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ? AND module = ?",
		"manager", "ecos").Scan(&exists)
	if exists != 2 {
		t.Error("Expected 2 ecos permissions after replacement")
	}

	// Verify old permissions don't exist
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ? AND module = ?",
		"manager", "parts").Scan(&exists)
	if exists != 0 {
		t.Error("Expected old parts permissions to be deleted")
	}
}

func TestHandleSetPermissions_EmptyPermissions(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()

	// Set empty permissions (effectively revoke all)
	reqBody := `{"permissions": []}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/user", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "user")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify all permissions removed
	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "user").Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 permissions after setting empty array, got %d", count)
	}
}

func TestHandleSetPermissions_MissingRole(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{"permissions": [{"module": "parts", "action": "view"}]}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Role required") {
		t.Errorf("Expected 'Role required' error, got: %s", w.Body.String())
	}
}

func TestHandleSetPermissions_InvalidJSON(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{invalid json`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "test")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid request body") {
		t.Errorf("Expected 'Invalid request body' error, got: %s", w.Body.String())
	}
}

func TestHandleSetPermissions_InvalidModule(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"permissions": [
			{"module": "invalid_module", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "test")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid module, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid module") {
		t.Errorf("Expected 'Invalid module' error, got: %s", w.Body.String())
	}
}

func TestHandleSetPermissions_InvalidAction(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "invalid_action"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "test")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid action, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid action") {
		t.Errorf("Expected 'Invalid action' error, got: %s", w.Body.String())
	}
}

func TestHandleSetPermissions_MultipleInvalidFields(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	reqBody := `{
		"permissions": [
			{"module": "invalid_module", "action": "invalid_action"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "test")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Should fail on first validation error (module checked first)
	if !strings.Contains(w.Body.String(), "Invalid module") {
		t.Errorf("Expected 'Invalid module' error, got: %s", w.Body.String())
	}
}

func TestHandleSetPermissions_AllValidModulesAndActions(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initPermCache()

	// Build permissions array with all valid modules and actions
	var permsArray []string
	for _, mod := range AllModules {
		for _, act := range AllActions {
			permsArray = append(permsArray, `{"module":"`+mod+`","action":"`+act+`"}`)
		}
	}
	reqBody := `{"permissions":[` + strings.Join(permsArray, ",") + `]}`

	req := httptest.NewRequest("PUT", "/api/v1/permissions/superadmin", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "superadmin")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify all permissions saved
	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "superadmin").Scan(&count)
	expectedCount := len(AllModules) * len(AllActions)
	if count != expectedCount {
		t.Errorf("Expected %d permissions, got %d", expectedCount, count)
	}
}

// =============================================================================
// SECURITY TESTS - CRITICAL
// =============================================================================

func TestHandleSetPermissions_CannotEscalateOwnPrivileges(t *testing.T) {
	// SECURITY TEST: User cannot grant themselves admin privileges
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	userID := insertTestUser(t, db, "testuser", "user")
	initPermCache()

	// Simulate user trying to grant themselves admin permissions
	reqBody := `{
		"permissions": [
			{"module": "admin", "action": "view"},
			{"module": "admin", "action": "create"},
			{"module": "admin", "action": "edit"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/user", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRole, "user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "user")

	// ⚠️ VULNERABILITY FOUND: Handler does NOT check if user has permission to modify permissions
	// This endpoint should be admin-only, but there's no RBAC check in handler_permissions.go
	// The middleware check happens at the routing level, but the handler itself doesn't validate

	// For now, document the vulnerability:
	t.Logf("⚠️ SECURITY ISSUE: handleSetPermissions does not validate caller permissions")
	t.Logf("Expected: Only admin role can modify permissions")
	t.Logf("Actual: Any authenticated user can call this endpoint if middleware allows")
	t.Logf("Fix needed: Add role check in handler OR ensure middleware enforces admin-only access")

	// The request succeeds (if middleware allows), which is the vulnerability
	if w.Code == 200 {
		t.Error("⚠️ PRIVILEGE ESCALATION VULNERABILITY: User was able to modify their own role permissions!")
		t.Error("This endpoint MUST be protected by admin-only middleware or handler-level role check")
	}
}

func TestHandleSetPermissions_ReadonlyCannotModify(t *testing.T) {
	// SECURITY TEST: Readonly users cannot modify permissions
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	readonlyID := insertTestUser(t, db, "readonly_user", "readonly")
	initPermCache()

	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "create"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/readonly", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), ctxUserID, readonlyID)
	ctx = context.WithValue(ctx, ctxRole, "readonly")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "readonly")

	// ⚠️ Same vulnerability as above - no role check in handler
	if w.Code == 200 {
		t.Error("⚠️ PRIVILEGE ESCALATION VULNERABILITY: Readonly user was able to modify permissions!")
	}
}

func TestHandleSetPermissions_SQLInjection_Prevention(t *testing.T) {
	// SECURITY TEST: SQL injection in role parameter
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()

	// Try SQL injection in role parameter
	sqlInjectionRole := "admin'; DROP TABLE role_permissions; --"

	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/"+sqlInjectionRole, bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, sqlInjectionRole)

	// Should succeed (role is treated as data, not SQL)
	if w.Code != 200 {
		t.Logf("Request failed (expected for malformed role), status: %d", w.Code)
	}

	// Verify table still exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM role_permissions").Scan(&count)
	if err != nil {
		t.Error("⚠️ SQL INJECTION VULNERABILITY: Table was dropped or damaged!")
		t.Fatalf("Table check failed: %v", err)
	}

	t.Logf("✓ SQL injection prevented - table still exists with %d rows", count)
}

func TestHandleSetPermissions_SQLInjection_ModuleAction(t *testing.T) {
	// SECURITY TEST: SQL injection in module/action fields
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initPermCache()

	payloads := []struct {
		name   string
		module string
		action string
	}{
		{"module injection", "parts'; DROP TABLE role_permissions; --", "view"},
		{"action injection", "parts", "view'; DELETE FROM role_permissions; --"},
		{"union injection", "parts' UNION SELECT * FROM users--", "view"},
	}

	for _, tt := range payloads {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := `{
				"permissions": [
					{"module": "` + tt.module + `", "action": "` + tt.action + `"}
				]
			}`
			req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleSetPermissions(w, req, "test")

			// Should fail validation (invalid module/action)
			if w.Code == 200 {
				t.Logf("Warning: SQL injection payload passed validation (should be rejected as invalid module/action)")
			}

			// Verify table integrity
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM role_permissions").Scan(&count)
			if err != nil {
				t.Error("⚠️ SQL INJECTION VULNERABILITY: Table was damaged!")
				t.Fatalf("Table check failed: %v", err)
			}
		})
	}

	t.Log("✓ SQL injection in module/action fields prevented")
}

func TestHandleSetPermissions_DuplicatePermissions(t *testing.T) {
	// Edge case: duplicate permissions in request
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initPermCache()

	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "view"},
			{"module": "parts", "action": "view"},
			{"module": "parts", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "test")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Due to UNIQUE constraint, should only save one
	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "test").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 permission (duplicates ignored), got %d", count)
	}
}

func TestHandleSetPermissions_CacheRefresh(t *testing.T) {
	// Verify permission cache is refreshed after update
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()
	refreshPermCache()

	// Initially, custom role has no permissions
	if HasPermission("custom", "parts", "view") {
		t.Error("Custom role should not have permissions initially")
	}

	// Set permissions
	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/custom", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "custom")

	if w.Code != 200 {
		t.Fatalf("Failed to set permissions: %d", w.Code)
	}

	// Cache should be refreshed - check if permission is now available
	if !HasPermission("custom", "parts", "view") {
		t.Error("⚠️ BUG: Permission cache not refreshed after setRolePermissions")
		t.Error("Expected HasPermission to return true after setting permissions")
	}
}

func TestHandleSetPermissions_DoesNotAffectOtherRoles(t *testing.T) {
	// Verify updating one role doesn't affect other roles
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()
	refreshPermCache()

	// Count admin permissions before
	var adminCountBefore int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "admin").Scan(&adminCountBefore)

	// Update user role
	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/user", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "user")

	if w.Code != 200 {
		t.Fatalf("Failed to set permissions: %d", w.Code)
	}

	// Count admin permissions after
	var adminCountAfter int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "admin").Scan(&adminCountAfter)

	if adminCountBefore != adminCountAfter {
		t.Errorf("Admin permissions changed! Before: %d, After: %d", adminCountBefore, adminCountAfter)
	}

	// Verify user permissions changed
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "user").Scan(&userCount)
	if userCount != 1 {
		t.Errorf("Expected user to have 1 permission, got %d", userCount)
	}
}

// =============================================================================
// PERMISSION HIERARCHY TESTS
// =============================================================================

func TestHandleSetPermissions_NoCircularDependencies(t *testing.T) {
	// The current system has no role hierarchy, so no circular dependencies possible
	// This test documents that fact
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initPermCache()

	// Set permissions for role A and role B independently
	reqBody := `{"permissions": [{"module": "parts", "action": "view"}]}`

	req := httptest.NewRequest("PUT", "/api/v1/permissions/roleA", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	handleSetPermissions(w, req, "roleA")

	req = httptest.NewRequest("PUT", "/api/v1/permissions/roleB", bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()
	handleSetPermissions(w, req, "roleB")

	t.Log("✓ No role hierarchy implemented - circular dependencies not possible")
}

func TestHandleSetPermissions_DefaultRoles_CanBeModified(t *testing.T) {
	// Verify that even default roles (admin, user, readonly) can be modified
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	seedDefaultPermissionsForTest(t, db)
	initPermCache()

	// Modify admin role (dangerous but allowed)
	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/admin", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "admin")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify admin now only has 1 permission (risky!)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "admin").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 permission for modified admin role, got %d", count)
	}

	t.Log("⚠️ DESIGN ISSUE: Default roles can be modified, potentially locking out all admins")
	t.Log("Consider: Protect default roles OR implement a 'system admin' that cannot be modified")
}

// =============================================================================
// EDGE CASES
// =============================================================================

func TestHandleSetPermissions_ExtremelyLongRoleName(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initPermCache()

	longRole := strings.Repeat("x", 1000)

	reqBody := `{"permissions": [{"module": "parts", "action": "view"}]}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/"+longRole, bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, longRole)

	// Should succeed (no length validation)
	if w.Code != 200 {
		t.Logf("Long role name rejected: %d", w.Code)
	} else {
		t.Log("✓ Long role names accepted (no length limit)")
	}
}

func TestHandleSetPermissions_SpecialCharactersInRole(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	initPermCache()

	specialRoles := []string{
		"role-with-dashes",
		"role_with_underscores",
		"role.with.dots",
		"role with spaces",
		"role@with#special$chars",
		"role'with'quotes",
	}

	for _, role := range specialRoles {
		t.Run(role, func(t *testing.T) {
			reqBody := `{"permissions": [{"module": "parts", "action": "view"}]}`
			req := httptest.NewRequest("PUT", "/api/v1/permissions/"+role, bytes.NewBufferString(reqBody))
			w := httptest.NewRecorder()

			handleSetPermissions(w, req, role)

			if w.Code == 200 {
				t.Logf("✓ Role '%s' accepted", role)
			} else {
				t.Logf("Role '%s' rejected: %d", role, w.Code)
			}
		})
	}
}

func TestHandleListPermissions_DatabaseError(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Close database to trigger error
	db.Close()

	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	w := httptest.NewRecorder()

	handleListPermissions(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500 for database error, got %d", w.Code)
	}
}

func TestHandleSetPermissions_DatabaseError(t *testing.T) {
	oldDB := db
	db = setupPermissionsTestDB(t)

	// Close database before operation
	db.Close()

	reqBody := `{"permissions": [{"module": "parts", "action": "view"}]}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handleSetPermissions(w, req, "test")

	if w.Code != 500 {
		t.Errorf("Expected status 500 for database error, got %d", w.Code)
	}

	db = oldDB
}
