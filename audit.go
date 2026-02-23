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

	"zrp/internal/audit"
)

// Audit action constant aliases for backward compatibility.
const (
	AuditActionCreate        = audit.ActionCreate
	AuditActionUpdate        = audit.ActionUpdate
	AuditActionDelete        = audit.ActionDelete
	AuditActionView          = audit.ActionView
	AuditActionViewSensitive = audit.ActionViewSensitive
	AuditActionExport        = audit.ActionExport
	AuditActionLogin         = audit.ActionLogin
	AuditActionLogout        = audit.ActionLogout
	AuditActionApprove       = audit.ActionApprove
	AuditActionReject        = audit.ActionReject
)

// Type aliases for backward compatibility.
type AuditEntry = audit.AuditEntry
type LogAuditOptions = audit.LogAuditOptions

// Wrapper functions delegating to internal/audit, injecting global db and wsHub.
func logAudit(db *sql.DB, username, action, module, recordID, summary string) {
	audit.LogAudit(db, wsHub, username, action, module, recordID, summary)
}

func getUsername(r *http.Request) string {
	return audit.GetUsername(db, r)
}

func GetUserContext(r *http.Request, dbConn *sql.DB) (int, string) {
	return audit.GetUserContext(r, dbConn)
}

func GetClientIP(r *http.Request) string {
	return audit.GetClientIP(r)
}

func LogAuditEnhanced(dbConn *sql.DB, opts LogAuditOptions) error {
	return audit.LogAuditEnhanced(dbConn, wsHub, opts)
}

func LogDataExport(dbConn *sql.DB, r *http.Request, module, format string, recordCount int) {
	audit.LogDataExport(dbConn, wsHub, r, module, format, recordCount)
}

func GetAuditRetentionDays(dbConn *sql.DB) int {
	return audit.GetAuditRetentionDays(dbConn)
}

func SetAuditRetentionDays(dbConn *sql.DB, days int) error {
	return audit.SetAuditRetentionDays(dbConn, days)
}

func LogSimpleAudit(dbConn *sql.DB, r *http.Request, action, module, recordID, summary string) {
	audit.LogSimpleAudit(dbConn, wsHub, r, action, module, recordID, summary)
}

func CleanupOldAuditLogs(dbConn *sql.DB, retentionDays int) (int64, error) {
	return audit.CleanupOldAuditLogs(dbConn, retentionDays)
}

func LogSensitiveDataAccess(dbConn *sql.DB, r *http.Request, dataType, recordID, details string) {
	audit.LogSensitiveDataAccess(dbConn, wsHub, r, dataType, recordID, details)
}

func LogUpdateWithDiff(dbConn *sql.DB, r *http.Request, module, recordID string, before, after interface{}) {
	audit.LogUpdateWithDiff(dbConn, wsHub, r, module, recordID, before, after)
}

// Handler functions stay in root (they use global db, wsHub, jsonResp, jsonErr).

func handleAuditLog(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	module := r.URL.Query().Get("module")
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

	var total int
	countQuery := "SELECT COUNT(*) FROM audit_log" + whereClause
	db.QueryRow(countQuery, args...).Scan(&total)

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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": items,
		"total":   total,
	})
}

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

	LogDataExport(db, r, "audit_log", "CSV", 0)

	filename := fmt.Sprintf("audit_log_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"ID", "Username", "Action", "Module", "Record ID", "Summary", "IP Address", "User Agent", "Timestamp"})

	recordCount := 0
	for rows.Next() {
		var id int
		var username, action, module, recordID, summary, ipAddr, userAgent, createdAt string
		rows.Scan(&id, &username, &action, &module, &recordID, &summary, &ipAddr, &userAgent, &createdAt)
		writer.Write([]string{
			strconv.Itoa(id), username, action, module, recordID, summary, ipAddr, userAgent, createdAt,
		})
		recordCount++
	}

	LogDataExport(db, r, "audit_log", "CSV", recordCount)
}

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
		"ecos_by_status":  ecoStatuses,
		"wos_by_status":   woStatuses,
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
