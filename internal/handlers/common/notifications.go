package common

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Notification represents a notification record.
type Notification struct {
	ID        int     `json:"id"`
	Type      string  `json:"type"`
	Severity  string  `json:"severity"`
	Title     string  `json:"title"`
	Message   *string `json:"message"`
	RecordID  *string `json:"record_id"`
	Module    *string `json:"module"`
	ReadAt    *string `json:"read_at"`
	CreatedAt string  `json:"created_at"`
}

// PendingNotif holds a notification candidate before insertion.
type PendingNotif struct {
	NType          string
	Severity       string
	Title          string
	Message        *string
	RecordID       *string
	Module         *string
	DeliveryMethod string
	UserID         int
}

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string { return &s }

// ListNotifications returns notifications, optionally filtered to unread only.
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	unread := r.URL.Query().Get("unread")
	q := `SELECT id, type, severity, title, message, record_id, module, read_at, created_at FROM notifications`
	if unread == "true" {
		q += ` WHERE read_at IS NULL`
	}
	q += ` ORDER BY created_at DESC LIMIT 50`

	rows, err := h.DB.Query(q)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}
	defer rows.Close()

	var notifs []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.Severity, &n.Title, &n.Message, &n.RecordID, &n.Module, &n.ReadAt, &n.CreatedAt); err != nil {
			continue
		}
		notifs = append(notifs, n)
	}
	if notifs == nil {
		notifs = []Notification{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifs)
}

// MarkNotificationRead marks a single notification as read.
func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request, id string) {
	_, err := h.DB.Exec("UPDATE notifications SET read_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "read"})
}

// GenerateNotifications checks for actionable conditions and creates notifications.
func (h *Handler) GenerateNotifications() {
	log.Println("Generating notifications...")
	var pending []PendingNotif

	// Low stock
	func() {
		rows, err := h.DB.Query(`SELECT ipn, qty_on_hand, reorder_point FROM inventory WHERE reorder_point > 0 AND qty_on_hand < reorder_point`)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var ipn string
			var qty, rp float64
			rows.Scan(&ipn, &qty, &rp)
			pending = append(pending, PendingNotif{NType: "low_stock", Severity: "warning", Title: "Low Stock: " + ipn,
				Message: StringPtr(fmt.Sprintf("%.0f on hand, reorder point %.0f", qty, rp)), RecordID: StringPtr(ipn), Module: StringPtr("inventory")})
		}
	}()

	// Overdue work orders
	func() {
		rows, err := h.DB.Query(`SELECT id, assembly_ipn FROM work_orders WHERE status = 'in_progress' AND started_at < datetime('now', '-7 days')`)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, ipn string
			rows.Scan(&id, &ipn)
			pending = append(pending, PendingNotif{NType: "overdue_wo", Severity: "warning", Title: "Overdue WO: " + id,
				Message: StringPtr("In progress for >7 days: " + ipn), RecordID: StringPtr(id), Module: StringPtr("workorders")})
		}
	}()

	// Open NCRs > 14 days
	func() {
		rows, err := h.DB.Query(`SELECT id, title FROM ncrs WHERE status = 'open' AND created_at < datetime('now', '-14 days')`)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, title string
			rows.Scan(&id, &title)
			t := title
			pending = append(pending, PendingNotif{NType: "open_ncr", Severity: "error", Title: "Open NCR >14d: " + id,
				Message: &t, RecordID: StringPtr(id), Module: StringPtr("ncr")})
		}
	}()

	// New RMAs in last hour
	func() {
		rows, err := h.DB.Query(`SELECT id, serial_number, customer FROM rmas WHERE created_at > datetime('now', '-1 hour')`)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, sn string
			var cust *string
			rows.Scan(&id, &sn, &cust)
			msg := "SN: " + sn
			if cust != nil {
				msg += " — " + *cust
			}
			pending = append(pending, PendingNotif{NType: "new_rma", Severity: "info", Title: "New RMA: " + id,
				Message: &msg, RecordID: StringPtr(id), Module: StringPtr("rma")})
		}
	}()

	// Now insert all collected notifications
	for _, p := range pending {
		h.CreateNotificationIfNew(p.NType, p.Severity, p.Title, p.Message, p.RecordID, p.Module)
	}
	log.Printf("Notification check complete: %d candidates", len(pending))
}

// CreateNotificationIfNew inserts a notification if no duplicate exists in the last 24 hours.
func (h *Handler) CreateNotificationIfNew(ntype, severity, title string, message, recordID, module *string) {
	// Dedup: don't create if same type+record exists within 24h
	var count int
	if recordID != nil {
		h.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type = ? AND record_id = ? AND created_at > datetime('now', '-24 hours')`,
			ntype, *recordID).Scan(&count)
	} else {
		h.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE type = ? AND title = ? AND created_at > datetime('now', '-24 hours')`,
			ntype, title).Scan(&count)
	}
	if count > 0 {
		return
	}
	_, err := h.DB.Exec(`INSERT INTO notifications (type, severity, title, message, record_id, module) VALUES (?, ?, ?, ?, ?, ?)`,
		ntype, severity, title, message, recordID, module)
	if err != nil {
		log.Println("Failed to insert notification:", err)
	}
}
