package admin

import (
	"encoding/json"
	"net/http"

	"zrp/internal/auth"
	"zrp/internal/response"
	"zrp/internal/server"
)

// HandleListPermissions lists all permissions for all roles (or ?role=X).
func (h *Handler) HandleListPermissions(w http.ResponseWriter, r *http.Request) {
	roleFilter := r.URL.Query().Get("role")

	rows, err := h.DB.Query("SELECT id, role, module, action FROM role_permissions ORDER BY role, module, action")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var perms []auth.PermissionEntry
	for rows.Next() {
		var p auth.PermissionEntry
		if err := rows.Scan(&p.ID, &p.Role, &p.Module, &p.Action); err != nil {
			continue
		}
		if roleFilter == "" || p.Role == roleFilter {
			perms = append(perms, p)
		}
	}
	if perms == nil {
		perms = []auth.PermissionEntry{}
	}
	response.JSON(w, perms)
}

// HandleListModules lists all available modules and actions.
func (h *Handler) HandleListModules(w http.ResponseWriter, r *http.Request) {
	type ModuleInfo struct {
		Module  string   `json:"module"`
		Actions []string `json:"actions"`
	}
	var modules []ModuleInfo
	for _, mod := range auth.AllModules {
		modules = append(modules, ModuleInfo{Module: mod, Actions: auth.AllActions})
	}
	response.JSON(w, modules)
}

// HandleMyPermissions returns the current user's permissions.
func (h *Handler) HandleMyPermissions(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(server.CtxRole).(string)
	if role == "" || role == "admin" {
		// Bearer token OR admin — return all permissions
		var allPerms []auth.PermissionEntry
		for _, mod := range auth.AllModules {
			for _, act := range auth.AllActions {
				allPerms = append(allPerms, auth.PermissionEntry{Module: mod, Action: act})
			}
		}
		response.JSON(w, allPerms)
		return
	}
	response.JSON(w, h.GetRolePermissions(role))
}

// HandleSetPermissions replaces all permissions for a role.
func (h *Handler) HandleSetPermissions(w http.ResponseWriter, r *http.Request, role string) {
	// Step 1: Validate role parameter (400 for bad input)
	if role == "" {
		response.Err(w, "Role required", 400)
		return
	}

	// Step 2: Validate request body JSON (400 for bad input)
	var req struct {
		Permissions []struct {
			Module string `json:"module"`
			Action string `json:"action"`
		} `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, "Invalid request body", 400)
		return
	}

	// Step 3: Validate module and action values (400 for bad input)
	validModules := make(map[string]bool)
	for _, m := range auth.AllModules {
		validModules[m] = true
	}
	validActions := make(map[string]bool)
	for _, a := range auth.AllActions {
		validActions[a] = true
	}

	// Use a map to deduplicate permissions
	permMap := make(map[string]bool)
	var perms []auth.PermissionEntry
	for _, p := range req.Permissions {
		if !validModules[p.Module] {
			response.Err(w, "Invalid module: "+p.Module, 400)
			return
		}
		if !validActions[p.Action] {
			response.Err(w, "Invalid action: "+p.Action, 400)
			return
		}
		// Deduplicate using module:action as key
		key := p.Module + ":" + p.Action
		if !permMap[key] {
			permMap[key] = true
			perms = append(perms, auth.PermissionEntry{Role: role, Module: p.Module, Action: p.Action})
		}
	}

	// Step 4: SECURITY - Only admins can modify permissions (403 for unauthorized)
	callerRole, _ := r.Context().Value(server.CtxRole).(string)
	if callerRole != "admin" {
		response.Err(w, "Forbidden: Only admins can modify permissions", 403)
		return
	}

	// Step 5: Perform the operation
	if err := h.SetRolePermissions(h.DB, role, perms); err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	response.JSON(w, map[string]string{"status": "updated"})
}
