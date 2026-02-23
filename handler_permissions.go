package main

import (
	"net/http"
)

// GET /api/v1/permissions — list all permissions for all roles (or ?role=X)
func handleListPermissions(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleListPermissions(w, r)
}

// GET /api/v1/permissions/modules — list all available modules and actions
func handleListModules(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleListModules(w, r)
}

// GET /api/v1/permissions/me — get current user's permissions
func handleMyPermissions(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleMyPermissions(w, r)
}

// PUT /api/v1/permissions/:role — replace all permissions for a role
func handleSetPermissions(w http.ResponseWriter, r *http.Request, role string) {
	getAdminHandler().HandleSetPermissions(w, r, role)
}
