package main

import (
	"database/sql"
	"net/http"

	"zrp/internal/auth"
	"zrp/internal/handlers/admin"
)

// Type aliases for backward compatibility.
type UserFull = admin.UserFull
type CreateUserRequest = admin.CreateUserRequest
type UpdateUserRequest = admin.UpdateUserRequest
type ResetPasswordRequest = admin.ResetPasswordRequest
type BackupInfo = admin.BackupInfo
type EmailConfig = admin.EmailConfig
type EmailLogEntry = admin.EmailLogEntry
type LoginRequest = admin.LoginRequest
type UserResponse = admin.UserResponse

// adminHandler is the shared admin handler instance.
var adminHandler *admin.Handler

func initAdminHandler() {
	adminHandler = buildAdminHandler()
}

func buildAdminHandler() *admin.Handler {
	var p admin.QueryProfiler
	if profiler != nil {
		p = profiler
	}
	return &admin.Handler{
		DB:       db,
		Hub:      wsHub,
		Profiler: p,

		// Backup function fields.
		PerformBackup:   performBackup,
		ListBackups:     listBackups,
		CleanOldBackups: cleanOldBackups,
		DBFilePath:      func() string { return dbFilePath },
		SetDBFilePath:   func(s string) { dbFilePath = s },
		InitDB:          initDB,

		// Email function fields.
		SendEmail:          sendEmail,
		SendEmailWithEvent: sendEmailWithEvent,
		GetEmailConfig:     getEmailConfig,
		EmailConfigEnabled: emailConfigEnabled,
		IsValidEmail:       isValidEmail,
		SendEventEmail:     sendEventEmail,
		IsUserSubscribed:   isUserSubscribed,

		// Auth function fields.
		CheckLoginRateLimit:          checkLoginRateLimit,
		IsAccountLocked:              IsAccountLocked,
		IncrementFailedLoginAttempts: IncrementFailedLoginAttempts,
		ResetFailedLoginAttempts:     ResetFailedLoginAttempts,
		GenerateToken:                generateToken,

		// Permission function fields.
		SetRolePermissions: func(dbConn *sql.DB, role string, perms []auth.PermissionEntry) error {
			return setRolePermissions(dbConn, role, perms)
		},
		GetRolePermissions: GetRolePermissions,
	}
}

// getAdminHandler returns the admin handler, lazily initializing if needed (for tests).
func getAdminHandler() *admin.Handler {
	if adminHandler == nil || adminHandler.DB != db {
		adminHandler = buildAdminHandler()
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
