package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Audit action constants
const (
	AuditActionCreate        = "CREATE"
	AuditActionUpdate        = "UPDATE"
	AuditActionDelete        = "DELETE"
	AuditActionView          = "VIEW"
	AuditActionViewSensitive = "VIEW_SENSITIVE"
	AuditActionExport        = "EXPORT"
	AuditActionLogin         = "LOGIN"
	AuditActionLogout        = "LOGOUT"
	AuditActionApprove       = "APPROVE"
	AuditActionReject        = "REJECT"
)

// logAudit is the legacy simple audit function - kept for backward compatibility
func logAudit(db *sql.DB, username, action, module, recordID, summary string) {
	_, err := db.Exec("INSERT INTO audit_log (username, action, module, record_id, summary) VALUES (?, ?, ?, ?, ?)",
		username, action, module, recordID, summary)
	if err != nil {
		fmt.Printf("audit log error: %v\n", err)
	}
	// Broadcast WebSocket event for real-time UI updates
	wsHub.Broadcast(WSEvent{
		Type:   module + "_" + action + "d",
		ID:     recordID,
		Action: action,
	})
}

func getUsername(r *http.Request) string {
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

type AuditEntry struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	Action      string `json:"action"`
	Module      string `json:"module"`
	RecordID    string `json:"record_id"`
	Summary     string `json:"summary"`
	BeforeValue string `json:"before_value,omitempty"`
	AfterValue  string `json:"after_value,omitempty"`
	IPAddress   string `json:"ip_address,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func handleAuditLog(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	module := r.URL.Query().Get("module")
	// Support frontend's "entity_type" param as alias for "module"
	if module == "" {
		module = r.URL.Query().Get("entity_type")
	}
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	user := r.URL.Query().Get("user")
	search := r.URL.Query().Get("search")
	dateFrom := r.URL.Query().Get("from")
	dateTo := r.URL.Query().Get("to")

	var args []interface{}
	var conditions []string
	if module != "" {
		conditions = append(conditions, "module = ?")
		args = append(args, module)
	}
	if user != "" {
		conditions = append(conditions, "username = ?")
		args = append(args, user)
	}
	if search != "" {
		conditions = append(conditions, "(summary LIKE ? OR action LIKE ? OR module LIKE ? OR record_id LIKE ?)")
		s := "%" + search + "%"
		args = append(args, s, s, s, s)
	}
	if dateFrom != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, dateTo+" 23:59:59")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM audit_log" + whereClause
	db.QueryRow(countQuery, args...).Scan(&total)

	// Get paginated results
	offset := (page - 1) * limit
	query := `SELECT id, COALESCE(user_id, 0), COALESCE(username,'system'), action, module, record_id, 
		COALESCE(summary,''), COALESCE(before_value,''), COALESCE(after_value,''), 
		COALESCE(ip_address,''), COALESCE(user_agent,''), created_at 
		FROM audit_log` + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)

	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var items []AuditEntry
	for rows.Next() {
		var e AuditEntry
		rows.Scan(&e.ID, &e.UserID, &e.Username, &e.Action, &e.Module, &e.RecordID, 
			&e.Summary, &e.BeforeValue, &e.AfterValue, &e.IPAddress, &e.UserAgent, &e.CreatedAt)
		items = append(items, e)
	}
	if items == nil {
		items = []AuditEntry{}
	}
	// Return in format expected by frontend: { entries: [], total: N }
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": items,
		"total":   total,
	})
}

// handleAuditExport exports audit logs to CSV
func handleAuditExport(w http.ResponseWriter, r *http.Request) {
	module := r.URL.Query().Get("module")
	if module == "" {
		module = r.URL.Query().Get("entity_type")
	}
	user := r.URL.Query().Get("user")
	search := r.URL.Query().Get("search")
	dateFrom := r.URL.Query().Get("from")
	dateTo := r.URL.Query().Get("to")
	action := r.URL.Query().Get("action")

	var args []interface{}
	var conditions []string
	if module != "" {
		conditions = append(conditions, "module = ?")
		args = append(args, module)
	}
	if user != "" {
		conditions = append(conditions, "username = ?")
		args = append(args, user)
	}
	if action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, action)
	}
	if search != "" {
		conditions = append(conditions, "(summary LIKE ? OR action LIKE ? OR module LIKE ? OR record_id LIKE ?)")
		s := "%" + search + "%"
		args = append(args, s, s, s, s)
	}
	if dateFrom != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, dateTo+" 23:59:59")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	query := `SELECT id, COALESCE(username,'system'), action, module, record_id, 
		COALESCE(summary,''), COALESCE(ip_address,''), COALESCE(user_agent,''), created_at 
		FROM audit_log` + whereClause + " ORDER BY created_at DESC LIMIT 10000"

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	// Log the export action
	LogDataExport(db, r, "audit_log", "CSV", 0)

	// Set CSV headers
	filename := fmt.Sprintf("audit_log_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"ID", "Username", "Action", "Module", "Record ID", "Summary", "IP Address", "User Agent", "Timestamp"})

	// Write data
	recordCount := 0
	for rows.Next() {
		var id int
		var username, action, module, recordID, summary, ipAddr, userAgent, createdAt string
		rows.Scan(&id, &username, &action, &module, &recordID, &summary, &ipAddr, &userAgent, &createdAt)
		writer.Write([]string{
			strconv.Itoa(id),
			username,
			action,
			module,
			recordID,
			summary,
			ipAddr,
			userAgent,
			createdAt,
		})
		recordCount++
	}

	// Update the export log with actual count
	LogDataExport(db, r, "audit_log", "CSV", recordCount)
}

// handleAuditRetention manages audit log retention settings
func handleAuditRetention(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		days := GetAuditRetentionDays(db)
		jsonResp(w, map[string]interface{}{
			"retention_days": days,
		})
		return
	}

	if r.Method == "PUT" {
		var req struct {
			RetentionDays int `json:"retention_days"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error(), 400)
			return
		}

		if req.RetentionDays < 30 || req.RetentionDays > 3650 {
			jsonErr(w, "Retention days must be between 30 and 3650 (10 years)", 400)
			return
		}

		if err := SetAuditRetentionDays(db, req.RetentionDays); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}

		LogSimpleAudit(db, r, AuditActionUpdate, "settings", "audit_retention", 
			fmt.Sprintf("Updated audit retention to %d days", req.RetentionDays))

		jsonResp(w, map[string]interface{}{
			"success":        true,
			"retention_days": req.RetentionDays,
		})
		return
	}

	if r.Method == "POST" && r.URL.Path == "/api/audit/cleanup" {
		days := GetAuditRetentionDays(db)
		deleted, err := CleanupOldAuditLogs(db, days)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}

		LogSimpleAudit(db, r, "cleanup", "audit_log", "retention", 
			fmt.Sprintf("Cleaned up %d old audit logs (retention: %d days)", deleted, days))

		jsonResp(w, map[string]interface{}{
			"success": true,
			"deleted": deleted,
			"message": fmt.Sprintf("Deleted %d audit log entries older than %d days", deleted, days),
		})
		return
	}

	http.Error(w, "Method not allowed", 405)
}

func handleDashboardCharts(w http.ResponseWriter, r *http.Request) {
	// ECOs by status
	ecoStatuses := map[string]int{"draft": 0, "review": 0, "approved": 0, "implemented": 0}
	rows, _ := db.Query("SELECT status, COUNT(*) FROM ecos GROUP BY status")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var s string
			var c int
			rows.Scan(&s, &c)
			ecoStatuses[s] = c
		}
	}

	// Work orders by status
	woStatuses := map[string]int{"open": 0, "in_progress": 0, "completed": 0}
	rows2, _ := db.Query("SELECT status, COUNT(*) FROM work_orders GROUP BY status")
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var s string
			var c int
			rows2.Scan(&s, &c)
			woStatuses[s] = c
		}
	}

	// Inventory value - top 10 by qty (use unit_price from po_lines if available, else 1.0)
	type InvValue struct {
		IPN   string  `json:"ipn"`
		Value float64 `json:"value"`
	}
	rows3, _ := db.Query(`SELECT i.ipn, i.qty_on_hand * COALESCE(
		(SELECT pl.unit_price FROM po_lines pl WHERE pl.ipn = i.ipn AND pl.unit_price > 0 ORDER BY pl.id DESC LIMIT 1),
		1.0
	) as value FROM inventory i ORDER BY value DESC LIMIT 10`)
	var invValues []InvValue
	if rows3 != nil {
		defer rows3.Close()
		for rows3.Next() {
			var iv InvValue
			rows3.Scan(&iv.IPN, &iv.Value)
			invValues = append(invValues, iv)
		}
	}
	if invValues == nil {
		invValues = []InvValue{}
	}

	jsonResp(w, map[string]interface{}{
		"ecos_by_status": ecoStatuses,
		"wos_by_status":  woStatuses,
		"inventory_value": invValues,
	})
}

func handleLowStockAlerts(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT ipn, qty_on_hand, reorder_point FROM inventory WHERE qty_on_hand < reorder_point AND reorder_point > 0")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type LowStockItem struct {
		IPN          string  `json:"ipn"`
		QtyOnHand    float64 `json:"qty_on_hand"`
		ReorderPoint float64 `json:"reorder_point"`
	}
	var items []LowStockItem
	for rows.Next() {
		var i LowStockItem
		rows.Scan(&i.IPN, &i.QtyOnHand, &i.ReorderPoint)
		items = append(items, i)
	}
	if items == nil {
		items = []LowStockItem{}
	}
	jsonResp(w, items)
}

// Stub functions for audit enhancements
func LogDataExport(db *sql.DB, r *http.Request, module, format string, recordCount int) {
	// Log data export action
	username := getUsername(r)
	summary := fmt.Sprintf("Exported %d records from %s as %s", recordCount, module, format)
	logAudit(db, username, "export", module, "", summary)
}

func GetAuditRetentionDays(db *sql.DB) int {
	// Default retention: 365 days
	var days int
	err := db.QueryRow("SELECT COALESCE((SELECT value FROM settings WHERE key = 'audit_retention_days'), '365')").Scan(&days)
	if err != nil {
		return 365
	}
	return days
}

func SetAuditRetentionDays(db *sql.DB, days int) error {
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('audit_retention_days', ?)", days)
	return err
}

func LogSimpleAudit(db *sql.DB, r *http.Request, action, module, recordID, summary string) {
	username := getUsername(r)
	logAudit(db, username, action, module, recordID, summary)
}

func CleanupOldAuditLogs(db *sql.DB, retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	result, err := db.Exec("DELETE FROM audit_log WHERE created_at < ?", cutoffDate)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func LogSensitiveDataAccess(db *sql.DB, r *http.Request, dataType, recordID, details string) {
	username := getUsername(r)
	summary := fmt.Sprintf("Accessed sensitive data: %s (%s) - %s", dataType, recordID, details)
	logAudit(db, username, "access", dataType, recordID, summary)
}

// LogAuditOptions contains all options for audit logging
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

// LogAuditEnhanced logs a comprehensive audit entry with all fields
func LogAuditEnhanced(db *sql.DB, opts LogAuditOptions) error {
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
		opts.UserID,
		opts.Username,
		opts.Action,
		opts.Module,
		opts.RecordID,
		opts.Summary,
		beforeJSON,
		afterJSON,
		opts.IPAddress,
		opts.UserAgent,
	)

	if err != nil {
		fmt.Printf("audit log error: %v\n", err)
		return err
	}

	// Broadcast WebSocket event for real-time UI updates
	wsHub.Broadcast(WSEvent{
		Type:   opts.Module + "_" + strings.ToLower(opts.Action),
		ID:     opts.RecordID,
		Action: opts.Action,
	})

	return nil
}

// GetUserContext extracts user information from request
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

// GetClientIP extracts the real client IP from the request (handles proxies)
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// LogUpdateWithDiff logs an update operation with before/after values
func LogUpdateWithDiff(db *sql.DB, r *http.Request, module, recordID string, before, after interface{}) {
	userID, username := GetUserContext(r, db)
	LogAuditEnhanced(db, LogAuditOptions{
		UserID:      userID,
		Username:    username,
		Action:      AuditActionUpdate,
		Module:      module,
		RecordID:    recordID,
		Summary:     fmt.Sprintf("Updated %s %s", module, recordID),
		BeforeValue: before,
		AfterValue:  after,
		IPAddress:   GetClientIP(r),
		UserAgent:   r.UserAgent(),
	})
}
