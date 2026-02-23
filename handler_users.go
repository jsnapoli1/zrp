package main

import (
	"net/http"

	"zrp/internal/handlers/admin"
)

// Type aliases for backward compatibility.
type UserFull = admin.UserFull
type CreateUserRequest = admin.CreateUserRequest
type UpdateUserRequest = admin.UpdateUserRequest
type ResetPasswordRequest = admin.ResetPasswordRequest

// adminHandler is the shared admin handler instance.
var adminHandler *admin.Handler

func initAdminHandler() {
	adminHandler = &admin.Handler{
		DB:       db,
		Hub:      wsHub,
		Profiler: profiler,
	}
}

// getAdminHandler returns the admin handler, lazily initializing if needed (for tests).
func getAdminHandler() *admin.Handler {
	if adminHandler == nil || adminHandler.DB != db {
		var p admin.QueryProfiler
		if profiler != nil {
			p = profiler
		}
		adminHandler = &admin.Handler{
			DB:       db,
			Hub:      wsHub,
			Profiler: p,
		}
	}
	return adminHandler
}

func getCurrentUser(r *http.Request) *UserFull {
	return getAdminHandler().GetCurrentUser(r)
}

func requireAdmin(w http.ResponseWriter, r *http.Request) *UserFull {
	return getAdminHandler().RequireAdmin(w, r)
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().ListUsers(w, r)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().CreateUser(w, r)
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request, idStr string) {
	getAdminHandler().UpdateUser(w, r, idStr)
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request, idStr string) {
	getAdminHandler().DeleteUser(w, r, idStr)
}

func handleResetPassword(w http.ResponseWriter, r *http.Request, idStr string) {
	getAdminHandler().ResetPassword(w, r, idStr)
}
