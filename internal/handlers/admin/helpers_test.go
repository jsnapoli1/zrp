package admin_test

import (
	"database/sql"
	"testing"
	"time"

	"zrp/internal/auth"
	"zrp/internal/handlers/admin"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// newTestHandler creates a Handler with the given DB and no-op/stub callbacks.
func newTestHandler(db *sql.DB) *admin.Handler {
	return &admin.Handler{
		DB: db,

		// Backup stubs
		PerformBackup:   func() error { return nil },
		ListBackups:     func() ([]admin.BackupInfo, error) { return nil, nil },
		CleanOldBackups: func() {},
		DBFilePath:      func() string { return "" },
		SetDBFilePath:   func(string) {},
		InitDB:          func(string) error { return nil },

		// Email stubs
		SendEmail:          func(to, subject, body string) error { return nil },
		SendEmailWithEvent: func(to, subject, body, eventType string) error { return nil },
		GetEmailConfig:     func() (*admin.EmailConfig, error) { return nil, nil },
		EmailConfigEnabled: func() bool { return false },
		IsValidEmail:       func(email string) bool { return true },
		SendEventEmail:     func(to, subject, body, eventType, username string) error { return nil },
		IsUserSubscribed:   func(username, eventType string) bool { return true },

		// Auth stubs
		CheckLoginRateLimit:          func(ip string) bool { return true },
		IsAccountLocked:              func(username string) (bool, error) { return false, nil },
		IncrementFailedLoginAttempts: func(username string) error { return nil },
		ResetFailedLoginAttempts:     func(username string) error { return nil },
		GenerateToken: func() string {
			// Use crypto/rand for a realistic but predictable-length token
			b := make([]byte, 32)
			for i := range b {
				b[i] = byte('a' + (i % 26))
			}
			return "testtoken_" + time.Now().Format("20060102150405.000000000")
		},

		// Permission stubs
		SetRolePermissions: func(dbConn *sql.DB, role string, perms []auth.PermissionEntry) error {
			// Delete existing permissions for this role
			_, _ = dbConn.Exec("DELETE FROM role_permissions WHERE role = ?", role)
			// Insert new permissions
			for _, p := range perms {
				_, _ = dbConn.Exec("INSERT INTO role_permissions (role, module, action) VALUES (?, ?, ?)",
					p.Role, p.Module, p.Action)
			}
			return nil
		},
		GetRolePermissions: func(role string) []auth.PermissionEntry {
			return nil
		},
	}
}

// setupAuthTestDB creates an in-memory SQLite database with tables needed for auth tests.
func setupAuthTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	tables := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			email TEXT DEFAULT '',
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1,
			last_login TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			failed_login_attempts INTEGER DEFAULT 0,
			locked_until TIMESTAMP
		)`,
		`CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE csrf_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE password_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			record_id TEXT NOT NULL,
			user_id INTEGER,
			username TEXT DEFAULT '',
			summary TEXT DEFAULT '',
			changes TEXT DEFAULT '{}',
			ip_address TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			created_by TEXT DEFAULT 'admin',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_used DATETIME,
			expires_at DATETIME,
			enabled INTEGER DEFAULT 1
		)`,
		`CREATE TABLE role_permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			role TEXT NOT NULL,
			module TEXT NOT NULL,
			action TEXT NOT NULL,
			UNIQUE(role, module, action)
		)`,
		`CREATE TABLE email_config (
			id INTEGER PRIMARY KEY DEFAULT 1,
			smtp_host TEXT,
			smtp_port INTEGER DEFAULT 587,
			smtp_user TEXT,
			smtp_password TEXT,
			from_address TEXT,
			from_name TEXT DEFAULT 'ZRP',
			enabled INTEGER DEFAULT 0
		)`,
		`CREATE TABLE email_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			to_address TEXT NOT NULL,
			recipient TEXT DEFAULT '',
			subject TEXT NOT NULL,
			body TEXT,
			event_type TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'sent',
			error TEXT,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE email_subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			UNIQUE(user_id, event_type)
		)`,
		`CREATE TABLE notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			severity TEXT DEFAULT 'info',
			title TEXT NOT NULL,
			message TEXT,
			record_id TEXT,
			module TEXT,
			user_id TEXT DEFAULT '',
			emailed INTEGER DEFAULT 0,
			read_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE password_reset_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			used INTEGER DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, ddl := range tables {
		if _, err := testDB.Exec(ddl); err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}

	return testDB
}

// createTestUserLocal creates a test user and returns the user ID.
func createTestUserLocal(t *testing.T, db *sql.DB, username, password, role string, active bool) int {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	activeInt := 0
	if active {
		activeInt = 1
	}

	result, err := db.Exec(
		"INSERT INTO users (username, password_hash, display_name, role, active) VALUES (?, ?, ?, ?, ?)",
		username, string(hash), username+" Display", role, activeInt,
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// createTestSessionLocal creates a session for a user and returns the token.
func createTestSessionLocal(t *testing.T, db *sql.DB, userID int) string {
	t.Helper()
	token := "test-session-" + time.Now().Format("20060102150405.000000")
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err := db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return token
}

// loginAdminLocal creates an admin user, creates a session, and returns the token.
func loginAdminLocal(t *testing.T, db *sql.DB) string {
	t.Helper()
	adminID := createTestUserLocal(t, db, "admin", "changeme", "admin", true)
	return createTestSessionLocal(t, db, adminID)
}

// loginUserLocal creates a regular user and returns their session token.
func loginUserLocal(t *testing.T, db *sql.DB, username string) string {
	t.Helper()
	userID := createTestUserLocal(t, db, username, "password", "user", true)
	return createTestSessionLocal(t, db, userID)
}
