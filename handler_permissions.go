package main

import (
	"encoding/json"
	"net/http"
)

// GET /api/v1/permissions — list all permissions for all roles (or ?role=X)
func handleListPermissions(w http.ResponseWriter, r *http.Request) {
	roleFilter := r.URL.Query().Get("role")

	rows, err := db.Query("SELECT id, role, module, action FROM role_permissions ORDER BY role, module, action")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Role, &p.Module, &p.Action); err != nil {
			continue
		}
		if roleFilter == "" || p.Role == roleFilter {
			perms = append(perms, p)
		}
	}
	if perms == nil {
		perms = []Permission{}
	}
	jsonResp(w, perms)
}

// GET /api/v1/permissions/modules — list all available modules and actions
func handleListModules(w http.ResponseWriter, r *http.Request) {
	type ModuleInfo struct {
		Module  string   `json:"module"`
		Actions []string `json:"actions"`
	}
	var modules []ModuleInfo
	for _, mod := range AllModules {
		modules = append(modules, ModuleInfo{Module: mod, Actions: AllActions})
	}
	jsonResp(w, modules)
}

// GET /api/v1/permissions/me — get current user's permissions
func handleMyPermissions(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(ctxRole).(string)
	if role == "" || role == "admin" {
		// Bearer token OR admin — return all permissions
		var allPerms []Permission
		for _, mod := range AllModules {
			for _, act := range AllActions {
				allPerms = append(allPerms, Permission{Module: mod, Action: act})
			}
		}
		jsonResp(w, allPerms)
		return
	}
	jsonResp(w, GetRolePermissions(role))
}

// PUT /api/v1/permissions/:role — replace all permissions for a role
func handleSetPermissions(w http.ResponseWriter, r *http.Request, role string) {
	if role == "" {
		jsonErr(w, "Role required", 400)
		return
	}

	var req struct {
		Permissions []struct {
			Module string `json:"module"`
			Action string `json:"action"`
		} `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Invalid request body", 400)
		return
	}

	// Validate
	validModules := make(map[string]bool)
	for _, m := range AllModules {
		validModules[m] = true
	}
	validActions := make(map[string]bool)
	for _, a := range AllActions {
		validActions[a] = true
	}

	var perms []Permission
	for _, p := range req.Permissions {
		if !validModules[p.Module] {
			jsonErr(w, "Invalid module: "+p.Module, 400)
			return
		}
		if !validActions[p.Action] {
			jsonErr(w, "Invalid action: "+p.Action, 400)
			return
		}
		perms = append(perms, Permission{Role: role, Module: p.Module, Action: p.Action})
	}

	if err := setRolePermissions(db, role, perms); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	jsonResp(w, map[string]string{"status": "updated"})
}
