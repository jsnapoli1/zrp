package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"zrp/internal/models"
	"zrp/internal/websocket"
)

// Action constants.
const (
	ActionCreate        = "CREATE"
	ActionUpdate        = "UPDATE"
	ActionDelete        = "DELETE"
	ActionView          = "VIEW"
	ActionViewSensitive = "VIEW_SENSITIVE"
	ActionExport        = "EXPORT"
	ActionLogin         = "LOGIN"
	ActionLogout        = "LOGOUT"
	ActionApprove       = "APPROVE"
	ActionReject        = "REJECT"
)

// LogAudit is the legacy simple audit function.
func LogAudit(db *sql.DB, hub *websocket.Hub, username, action, module, recordID, summary string) {
	_, err := db.Exec("INSERT INTO audit_log (username, action, module, record_id, summary) VALUES (?, ?, ?, ?, ?)",
		username, action, module, recordID, summary)
	if err != nil {
		fmt.Printf("audit log error: %v\n", err)
	}
	if hub != nil {
		hub.Broadcast(websocket.Event{
			Type:   module + "_" + action + "d",
			ID:     recordID,
			Action: action,
		})
	}
}

// GetUsername extracts the username from a session cookie.
func GetUsername(db *sql.DB, r *http.Request) string {
	cookie, err := r.Cookie("zrp_session")
	if err != nil {
		return "system"
	}
	var username string
	err = db.QueryRow("SELECT u.username FROM users u JOIN sessions s ON u.id = s.user_id WHERE s.token = ?", cookie.Value).Scan(&username)
	if err != nil {
		return "system"
	}
	return username
}

// GetUserContext extracts user information from request.
func GetUserContext(r *http.Request, db *sql.DB) (userID int, username string) {
	cookie, err := r.Cookie("zrp_session")
	if err != nil {
		return 0, "system"
	}
	err = db.QueryRow("SELECT u.id, u.username FROM users u JOIN sessions s ON u.id = s.user_id WHERE s.token = ?", cookie.Value).
		Scan(&userID, &username)
	if err != nil {
		return 0, "system"
	}
	return userID, username
}

// GetClientIP extracts the real client IP from the request (handles proxies).
func GetClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// LogAuditOptions contains all options for audit logging.
type LogAuditOptions struct {
	UserID      int
	Username    string
	Action      string
	Module      string
	RecordID    string
	Summary     string
	BeforeValue interface{}
	AfterValue  interface{}
	IPAddress   string
	UserAgent   string
}

// LogAuditEnhanced logs a comprehensive audit entry with all fields.
func LogAuditEnhanced(db *sql.DB, hub *websocket.Hub, opts LogAuditOptions) error {
	var beforeJSON, afterJSON []byte
	var err error

	if opts.BeforeValue != nil {
		beforeJSON, err = json.Marshal(opts.BeforeValue)
		if err != nil {
			beforeJSON = nil
		}
	}
	if opts.AfterValue != nil {
		afterJSON, err = json.Marshal(opts.AfterValue)
		if err != nil {
			afterJSON = nil
		}
	}

	query := `INSERT INTO audit_log
		(user_id, username, action, module, record_id, summary, before_value, after_value, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(query,
		opts.UserID, opts.Username, opts.Action, opts.Module, opts.RecordID,
		opts.Summary, beforeJSON, afterJSON, opts.IPAddress, opts.UserAgent,
	)
	if err != nil {
		fmt.Printf("audit log error: %v\n", err)
		return err
	}

	if hub != nil {
		hub.Broadcast(websocket.Event{
			Type:   opts.Module + "_" + strings.ToLower(opts.Action),
			ID:     opts.RecordID,
			Action: opts.Action,
		})
	}
	return nil
}

// LogDataExport logs a data export action.
func LogDataExport(db *sql.DB, hub *websocket.Hub, r *http.Request, module, format string, recordCount int) {
	username := GetUsername(db, r)
	_, err := db.Exec(`INSERT INTO data_exports (entity_type, format, record_count, user_id, exported_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`, module, format, recordCount, username)
	if err != nil {
		summary := fmt.Sprintf("Exported %d records from %s as %s", recordCount, module, format)
		LogAudit(db, hub, username, "export", module, "", summary)
	}
}

// GetAuditRetentionDays returns the audit log retention period.
func GetAuditRetentionDays(db *sql.DB) int {
	var days int
	err := db.QueryRow("SELECT COALESCE((SELECT value FROM settings WHERE key = 'audit_retention_days'), '365')").Scan(&days)
	if err != nil {
		return 365
	}
	return days
}

// SetAuditRetentionDays updates the audit log retention period.
func SetAuditRetentionDays(db *sql.DB, days int) error {
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('audit_retention_days', ?)", days)
	return err
}

// LogSimpleAudit logs an audit entry using the request context.
func LogSimpleAudit(db *sql.DB, hub *websocket.Hub, r *http.Request, action, module, recordID, summary string) {
	username := GetUsername(db, r)
	LogAudit(db, hub, username, action, module, recordID, summary)
}

// CleanupOldAuditLogs deletes audit log entries older than retentionDays.
func CleanupOldAuditLogs(db *sql.DB, retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	result, err := db.Exec("DELETE FROM audit_log WHERE created_at < ?", cutoffDate)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// LogSensitiveDataAccess logs access to sensitive data.
func LogSensitiveDataAccess(db *sql.DB, hub *websocket.Hub, r *http.Request, dataType, recordID, details string) {
	username := GetUsername(db, r)
	summary := fmt.Sprintf("Accessed sensitive data: %s (%s) - %s", dataType, recordID, details)
	LogAudit(db, hub, username, "access", dataType, recordID, summary)
}

// LogUpdateWithDiff logs an update operation with before/after values.
func LogUpdateWithDiff(db *sql.DB, hub *websocket.Hub, r *http.Request, module, recordID string, before, after interface{}) {
	userID, username := GetUserContext(r, db)
	LogAuditEnhanced(db, hub, LogAuditOptions{
		UserID:      userID,
		Username:    username,
		Action:      ActionUpdate,
		Module:      module,
		RecordID:    recordID,
		Summary:     fmt.Sprintf("Updated %s %s", module, recordID),
		BeforeValue: before,
		AfterValue:  after,
		IPAddress:   GetClientIP(r),
		UserAgent:   r.UserAgent(),
	})
}

// Re-export AuditEntry type for use by callers (defined in models).
type AuditEntry = models.AuditEntry
