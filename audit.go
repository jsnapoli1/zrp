package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func logAudit(db *sql.DB, username, action, module, recordID, summary string) {
	_, err := db.Exec("INSERT INTO audit_log (username, action, module, record_id, summary) VALUES (?, ?, ?, ?, ?)",
		username, action, module, recordID, summary)
	if err != nil {
		fmt.Printf("audit log error: %v\n", err)
	}
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
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Action    string `json:"action"`
	Module    string `json:"module"`
	RecordID  string `json:"record_id"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
}

func handleAuditLog(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	module := r.URL.Query().Get("module")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	user := r.URL.Query().Get("user")
	dateFrom := r.URL.Query().Get("from")
	dateTo := r.URL.Query().Get("to")

	query := "SELECT id, COALESCE(username,'system'), action, module, record_id, COALESCE(summary,''), created_at FROM audit_log"
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
	if dateFrom != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, dateTo+" 23:59:59")
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var items []AuditEntry
	for rows.Next() {
		var e AuditEntry
		rows.Scan(&e.ID, &e.Username, &e.Action, &e.Module, &e.RecordID, &e.Summary, &e.CreatedAt)
		items = append(items, e)
	}
	if items == nil {
		items = []AuditEntry{}
	}
	jsonResp(w, items)
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
