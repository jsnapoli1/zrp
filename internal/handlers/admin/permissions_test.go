package admin_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"zrp/internal/auth"
	"zrp/internal/handlers/admin"

	_ "modernc.org/sqlite"
)

// seedPermissionsLocal seeds the default 3-role permission structure.
func seedPermissionsLocal(t *testing.T, db *sql.DB) {
	t.Helper()
	// Admin: everything
	for _, mod := range auth.AllModules {
		for _, act := range auth.AllActions {
			_, err := db.Exec("INSERT OR IGNORE INTO role_permissions (role, module, action) VALUES (?, ?, ?)",
				"admin", mod, act)
			if err != nil {
				t.Fatalf("Failed to insert admin permission: %v", err)
			}
		}
	}

	// User: everything except admin module
	for _, mod := range auth.AllModules {
		if mod == auth.ModuleAdmin {
			continue
		}
		for _, act := range auth.AllActions {
			_, err := db.Exec("INSERT OR IGNORE INTO role_permissions (role, module, action) VALUES (?, ?, ?)",
				"user", mod, act)
			if err != nil {
				t.Fatalf("Failed to insert user permission: %v", err)
			}
		}
	}

	// Readonly: view only
	for _, mod := range auth.AllModules {
		_, err := db.Exec("INSERT OR IGNORE INTO role_permissions (role, module, action) VALUES (?, ?, ?)",
			"readonly", mod, auth.PermActionView)
		if err != nil {
			t.Fatalf("Failed to insert readonly permission: %v", err)
		}
	}
}

// =============================================================================
// TEST: HandleListPermissions
// =============================================================================

func TestHandleListPermissions_EmptyDatabase(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	w := httptest.NewRecorder()

	h.HandleListPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []auth.PermissionEntry `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Data) != 0 {
		t.Errorf("Expected empty array, got %d permissions", len(resp.Data))
	}
}

func TestHandleListPermissions_AllRoles(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	seedPermissionsLocal(t, db)

	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	w := httptest.NewRecorder()

	h.HandleListPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []auth.PermissionEntry `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// admin: 19 modules * 5 actions = 95
	// user: 18 modules * 5 actions = 90
	// readonly: 19 modules * 1 action = 19
	// Total: 204
	expectedCount := 204
	if len(resp.Data) != expectedCount {
		t.Errorf("Expected %d permissions, got %d", expectedCount, len(resp.Data))
	}
}

func TestHandleListPermissions_FilterByRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	seedPermissionsLocal(t, db)

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

			h.HandleListPermissions(w, req)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var resp struct {
				Data []auth.PermissionEntry `json:"data"`
			}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(resp.Data) != tt.expectedCount {
				t.Errorf("Expected %d permissions for role %s, got %d", tt.expectedCount, tt.role, len(resp.Data))
			}

			for _, p := range resp.Data {
				if p.Role != tt.role {
					t.Errorf("Expected role %s, got %s", tt.role, p.Role)
				}
			}
		})
	}
}

func TestHandleListPermissions_OrderedByRoleModuleAction(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('user', 'parts', 'edit')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('admin', 'ecos', 'view')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('readonly', 'parts', 'view')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('admin', 'ecos', 'create')")

	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	w := httptest.NewRecorder()

	h.HandleListPermissions(w, req)

	var resp struct {
		Data []auth.PermissionEntry `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

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
// TEST: HandleListModules
// =============================================================================

func TestHandleListModules_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/permissions/modules", nil)
	w := httptest.NewRecorder()

	h.HandleListModules(w, req)

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

	expectedModuleCount := len(auth.AllModules)
	if len(modules) != expectedModuleCount {
		t.Errorf("Expected %d modules, got %d", expectedModuleCount, len(modules))
	}

	expectedActionCount := len(auth.AllActions)
	for _, mod := range modules {
		if len(mod.Actions) != expectedActionCount {
			t.Errorf("Module %s: expected %d actions, got %d", mod.Module, expectedActionCount, len(mod.Actions))
		}
	}
}

func TestHandleListModules_ContainsExpectedModules(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/permissions/modules", nil)
	w := httptest.NewRecorder()

	h.HandleListModules(w, req)

	var resp struct {
		Data []struct {
			Module  string   `json:"module"`
			Actions []string `json:"actions"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	modules := resp.Data

	criticalModules := []string{auth.ModuleParts, auth.ModuleAdmin, auth.ModuleECOs, auth.ModuleInventory}
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
// TEST: HandleMyPermissions
// =============================================================================

func TestHandleMyPermissions_AdminRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	ctx := context.WithValue(req.Context(), admin.CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleMyPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []auth.PermissionEntry `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	expectedCount := len(auth.AllModules) * len(auth.AllActions)
	if len(resp.Data) != expectedCount {
		t.Errorf("Expected %d permissions for admin, got %d", expectedCount, len(resp.Data))
	}
}

func TestHandleMyPermissions_BearerToken_NoRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Bearer token auth with no role in context
	req := httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	w := httptest.NewRecorder()

	h.HandleMyPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []auth.PermissionEntry `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Bearer tokens get all permissions
	expectedCount := len(auth.AllModules) * len(auth.AllActions)
	if len(resp.Data) != expectedCount {
		t.Errorf("Expected %d permissions for bearer token, got %d", expectedCount, len(resp.Data))
	}
}

func TestHandleMyPermissions_UserRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	seedPermissionsLocal(t, db)

	// Override GetRolePermissions to actually read from DB
	h := newTestHandler(db)
	h.GetRolePermissions = func(role string) []auth.PermissionEntry {
		rows, err := db.Query("SELECT id, role, module, action FROM role_permissions WHERE role = ?", role)
		if err != nil {
			return nil
		}
		defer rows.Close()
		var perms []auth.PermissionEntry
		for rows.Next() {
			var p auth.PermissionEntry
			if err := rows.Scan(&p.ID, &p.Role, &p.Module, &p.Action); err != nil {
				continue
			}
			perms = append(perms, p)
		}
		return perms
	}

	req := httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	ctx := context.WithValue(req.Context(), admin.CtxRole, "user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleMyPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []auth.PermissionEntry `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// User should have permissions on all modules except admin
	expectedCount := (len(auth.AllModules) - 1) * len(auth.AllActions)
	if len(resp.Data) != expectedCount {
		t.Errorf("Expected %d permissions for user, got %d", expectedCount, len(resp.Data))
	}

	// Verify no admin module permissions
	for _, p := range resp.Data {
		if p.Module == auth.ModuleAdmin {
			t.Error("User should not have admin module permissions")
		}
	}
}

func TestHandleMyPermissions_ReadonlyRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	seedPermissionsLocal(t, db)

	h := newTestHandler(db)
	h.GetRolePermissions = func(role string) []auth.PermissionEntry {
		rows, err := db.Query("SELECT id, role, module, action FROM role_permissions WHERE role = ?", role)
		if err != nil {
			return nil
		}
		defer rows.Close()
		var perms []auth.PermissionEntry
		for rows.Next() {
			var p auth.PermissionEntry
			if err := rows.Scan(&p.ID, &p.Role, &p.Module, &p.Action); err != nil {
				continue
			}
			perms = append(perms, p)
		}
		return perms
	}

	req := httptest.NewRequest("GET", "/api/v1/permissions/me", nil)
	ctx := context.WithValue(req.Context(), admin.CtxRole, "readonly")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleMyPermissions(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []auth.PermissionEntry `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	expectedCount := len(auth.AllModules)
	if len(resp.Data) != expectedCount {
		t.Errorf("Expected %d permissions for readonly, got %d", expectedCount, len(resp.Data))
	}

	for _, p := range resp.Data {
		if p.Action != auth.PermActionView {
			t.Errorf("Readonly should only have view action, got %s", p.Action)
		}
	}
}

// =============================================================================
// TEST: HandleSetPermissions
// =============================================================================

func TestHandleSetPermissions_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "view"},
			{"module": "parts", "action": "create"},
			{"module": "inventory", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/supervisor", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "supervisor")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Data.Status != "updated" {
		t.Errorf("Expected status 'updated', got '%s'", resp.Data.Status)
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
}

func TestHandleSetPermissions_ReplacesExisting(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	// Insert initial permissions
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('manager', 'parts', 'view')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('manager', 'parts', 'edit')")
	db.Exec("INSERT INTO role_permissions (role, module, action) VALUES ('manager', 'inventory', 'view')")

	reqBody := `{
		"permissions": [
			{"module": "ecos", "action": "view"},
			{"module": "ecos", "action": "approve"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/manager", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "manager")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "manager").Scan(&count)
	if count != 2 {
		t.Errorf("Expected 2 permissions after replacement, got %d", count)
	}

	// Verify old permissions don't exist
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ? AND module = ?",
		"manager", "parts").Scan(&exists)
	if exists != 0 {
		t.Error("Expected old parts permissions to be deleted")
	}
}

func TestHandleSetPermissions_EmptyPermissions(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	seedPermissionsLocal(t, db)

	reqBody := `{"permissions": []}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/user", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "user")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "user").Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 permissions after setting empty array, got %d", count)
	}
}

func TestHandleSetPermissions_MissingRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{"permissions": [{"module": "parts", "action": "view"}]}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "")

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Role required") {
		t.Errorf("Expected 'Role required' error, got: %s", w.Body.String())
	}
}

func TestHandleSetPermissions_InvalidJSON(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{invalid json`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "test")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid request body") {
		t.Errorf("Expected 'Invalid request body' error, got: %s", w.Body.String())
	}
}

func TestHandleSetPermissions_InvalidModule(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{
		"permissions": [
			{"module": "invalid_module", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "test")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid module, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid module") {
		t.Errorf("Expected 'Invalid module' error, got: %s", w.Body.String())
	}
}

func TestHandleSetPermissions_InvalidAction(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "invalid_action"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "test")

	if w.Code != 400 {
		t.Errorf("Expected status 400 for invalid action, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Invalid action") {
		t.Errorf("Expected 'Invalid action' error, got: %s", w.Body.String())
	}
}

func TestHandleSetPermissions_NonAdminForbidden(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/user", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxRole, "user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "user")

	if w.Code != 403 {
		t.Errorf("Expected status 403 for non-admin, got %d", w.Code)
	}
}

func TestHandleSetPermissions_ReadonlyCannotModify(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "create"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/readonly", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxRole, "readonly")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "readonly")

	if w.Code != 403 {
		t.Errorf("Expected status 403 for readonly user, got %d", w.Code)
	}
}

func TestHandleSetPermissions_DuplicatePermissions(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	reqBody := `{
		"permissions": [
			{"module": "parts", "action": "view"},
			{"module": "parts", "action": "view"},
			{"module": "parts", "action": "view"}
		]
	}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "test")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Due to deduplication, should only save one
	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "test").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 permission (duplicates ignored), got %d", count)
	}
}

func TestHandleSetPermissions_DoesNotAffectOtherRoles(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	seedPermissionsLocal(t, db)

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
	ctx := context.WithValue(req.Context(), admin.CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "user")

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

func TestHandleSetPermissions_AllValidModulesAndActions(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	var permsArray []string
	for _, mod := range auth.AllModules {
		for _, act := range auth.AllActions {
			permsArray = append(permsArray, `{"module":"`+mod+`","action":"`+act+`"}`)
		}
	}
	reqBody := `{"permissions":[` + strings.Join(permsArray, ",") + `]}`

	req := httptest.NewRequest("PUT", "/api/v1/permissions/superadmin", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "superadmin")

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions WHERE role = ?", "superadmin").Scan(&count)
	expectedCount := len(auth.AllModules) * len(auth.AllActions)
	if count != expectedCount {
		t.Errorf("Expected %d permissions, got %d", expectedCount, count)
	}
}

func TestHandleListPermissions_DatabaseError(t *testing.T) {
	db := setupAuthTestDB(t)
	h := newTestHandler(db)

	// Close database to trigger error
	db.Close()

	req := httptest.NewRequest("GET", "/api/v1/permissions", nil)
	w := httptest.NewRecorder()

	h.HandleListPermissions(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500 for database error, got %d", w.Code)
	}
}

func TestHandleSetPermissions_DatabaseError(t *testing.T) {
	db := setupAuthTestDB(t)
	h := newTestHandler(db)

	// Override SetRolePermissions to actually use the DB
	h.SetRolePermissions = func(dbConn *sql.DB, role string, perms []auth.PermissionEntry) error {
		_, err := dbConn.Exec("DELETE FROM role_permissions WHERE role = ?", role)
		return err
	}

	// Close database before operation
	db.Close()

	reqBody := `{"permissions": [{"module": "parts", "action": "view"}]}`
	req := httptest.NewRequest("PUT", "/api/v1/permissions/test", bytes.NewBufferString(reqBody))
	ctx := context.WithValue(req.Context(), admin.CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.HandleSetPermissions(w, req, "test")

	if w.Code != 500 {
		t.Errorf("Expected status 500 for database error, got %d", w.Code)
	}
}
