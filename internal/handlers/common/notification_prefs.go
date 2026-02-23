package common

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// NotificationTypeInfo describes an available notification type.
type NotificationTypeInfo struct {
	Type             string   `json:"type"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Icon             string   `json:"icon"`
	HasThreshold     bool     `json:"has_threshold"`
	ThresholdLabel   *string  `json:"threshold_label,omitempty"`
	ThresholdDefault *float64 `json:"threshold_default,omitempty"`
}

// NotificationPreference represents a user's preference for a notification type.
type NotificationPreference struct {
	ID             int      `json:"id"`
	UserID         int      `json:"user_id"`
	Type           string   `json:"notification_type"`
	Enabled        bool     `json:"enabled"`
	DeliveryMethod string   `json:"delivery_method"`
	ThresholdValue *float64 `json:"threshold_value"`
}

// Float64Ptr returns a pointer to the given float64.
func Float64Ptr(f float64) *float64 { return &f }

// NotificationTypes is the list of supported notification types.
var NotificationTypes = []NotificationTypeInfo{
	{Type: "low_stock", Name: "Low Stock", Description: "When inventory drops below the minimum quantity threshold", Icon: "package", HasThreshold: true, ThresholdLabel: StringPtr("Minimum Qty"), ThresholdDefault: Float64Ptr(10)},
	{Type: "overdue_wo", Name: "Overdue Work Order", Description: "When a work order has been in progress longer than the threshold days", Icon: "clock", HasThreshold: true, ThresholdLabel: StringPtr("Days Overdue"), ThresholdDefault: Float64Ptr(7)},
	{Type: "open_ncr", Name: "Open NCR", Description: "When an NCR has been open longer than 14 days", Icon: "alert-triangle", HasThreshold: false},
	{Type: "eco_approval", Name: "ECO Approval", Description: "When an ECO requires your approval or is approved", Icon: "check-circle", HasThreshold: false},
	{Type: "eco_implemented", Name: "ECO Implemented", Description: "When an ECO is marked as implemented", Icon: "check-square", HasThreshold: false},
	{Type: "po_received", Name: "PO Received", Description: "When a purchase order is received", Icon: "truck", HasThreshold: false},
	{Type: "wo_completed", Name: "Work Order Completed", Description: "When a work order is completed", Icon: "check", HasThreshold: false},
	{Type: "field_report_critical", Name: "Critical Field Report", Description: "When a field report is marked as critical", Icon: "alert-circle", HasThreshold: false},
}

// InitNotificationPrefsTable creates the notification_preferences table if it does not exist.
func (h *Handler) InitNotificationPrefsTable() {
	_, err := h.DB.Exec(`CREATE TABLE IF NOT EXISTS notification_preferences (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		notification_type TEXT NOT NULL,
		enabled INTEGER DEFAULT 1,
		delivery_method TEXT DEFAULT 'in_app',
		threshold_value REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, notification_type)
	)`)
	if err != nil {
		log.Println("Failed to create notification_preferences table:", err)
	}
}

// EnsureDefaultPreferences creates default preferences for a user if they don't exist.
func (h *Handler) EnsureDefaultPreferences(userID int) {
	for _, nt := range NotificationTypes {
		var count int
		h.DB.QueryRow("SELECT COUNT(*) FROM notification_preferences WHERE user_id=? AND notification_type=?", userID, nt.Type).Scan(&count)
		if count == 0 {
			h.DB.Exec("INSERT INTO notification_preferences (user_id, notification_type, enabled, delivery_method, threshold_value) VALUES (?, ?, 1, 'in_app', ?)",
				userID, nt.Type, nt.ThresholdDefault)
		}
	}
}

// GetNotificationPreferences handles GET /api/notification-preferences.
func (h *Handler) GetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(h.CtxUserID).(int)
	if !ok || userID == 0 {
		http.Error(w, `{"error":"unauthorized"}`, 401)
		return
	}
	h.EnsureDefaultPreferences(userID)

	prefs := h.GetNotificationPrefsForUser(userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}

// GetNotificationPrefsForUser returns all notification preferences for a user.
func (h *Handler) GetNotificationPrefsForUser(userID int) []NotificationPreference {
	rows, err := h.DB.Query("SELECT id, user_id, notification_type, enabled, delivery_method, threshold_value FROM notification_preferences WHERE user_id=? ORDER BY notification_type", userID)
	if err != nil {
		return []NotificationPreference{}
	}
	defer rows.Close()

	var prefs []NotificationPreference
	for rows.Next() {
		var p NotificationPreference
		var enabled int
		if err := rows.Scan(&p.ID, &p.UserID, &p.Type, &enabled, &p.DeliveryMethod, &p.ThresholdValue); err != nil {
			continue
		}
		p.Enabled = enabled == 1
		prefs = append(prefs, p)
	}
	if prefs == nil {
		prefs = []NotificationPreference{}
	}
	return prefs
}

// UpdateNotificationPreferences handles PUT /api/notification-preferences.
func (h *Handler) UpdateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(h.CtxUserID).(int)
	if !ok || userID == 0 {
		http.Error(w, `{"error":"unauthorized"}`, 401)
		return
	}
	h.EnsureDefaultPreferences(userID)

	var prefs []NotificationPreference
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400)
		return
	}

	for _, p := range prefs {
		if !IsValidNotificationType(p.Type) {
			continue
		}
		if !IsValidDeliveryMethod(p.DeliveryMethod) {
			p.DeliveryMethod = "in_app"
		}
		enabled := 0
		if p.Enabled {
			enabled = 1
		}
		h.DB.Exec(`INSERT OR REPLACE INTO notification_preferences (user_id, notification_type, enabled, delivery_method, threshold_value, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			userID, p.Type, enabled, p.DeliveryMethod, p.ThresholdValue)
	}

	username := h.GetUsername(r)
	h.LogAudit(h.DB, username, "updated", "notification_preferences", "", "Updated notification preferences")
	h.GetNotificationPreferences(w, r)
}

// UpdateSingleNotificationPreference handles PUT /api/notification-preferences/:type.
func (h *Handler) UpdateSingleNotificationPreference(w http.ResponseWriter, r *http.Request, notifType string) {
	userID, ok := r.Context().Value(h.CtxUserID).(int)
	if !ok || userID == 0 {
		http.Error(w, `{"error":"unauthorized"}`, 401)
		return
	}

	if !IsValidNotificationType(notifType) {
		http.Error(w, `{"error":"invalid notification type"}`, 400)
		return
	}

	h.EnsureDefaultPreferences(userID)

	var p NotificationPreference
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400)
		return
	}

	if !IsValidDeliveryMethod(p.DeliveryMethod) {
		p.DeliveryMethod = "in_app"
	}
	enabled := 0
	if p.Enabled {
		enabled = 1
	}
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO notification_preferences (user_id, notification_type, enabled, delivery_method, threshold_value, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		userID, notifType, enabled, p.DeliveryMethod, p.ThresholdValue)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}

	h.GetNotificationPreferences(w, r)
}

// ListNotificationTypes handles GET /api/notification-types.
func (h *Handler) ListNotificationTypes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NotificationTypes)
}

// IsValidNotificationType checks if a notification type is recognized.
func IsValidNotificationType(t string) bool {
	for _, nt := range NotificationTypes {
		if nt.Type == t {
			return true
		}
	}
	return false
}

// IsValidDeliveryMethod checks if a delivery method is recognized.
func IsValidDeliveryMethod(m string) bool {
	return m == "in_app" || m == "email" || m == "both"
}

// GetUserNotifPref returns the preference for a given user and type.
func (h *Handler) GetUserNotifPref(userID int, notifType string) (enabled bool, deliveryMethod string, threshold *float64) {
	var e int
	err := h.DB.QueryRow("SELECT enabled, delivery_method, threshold_value FROM notification_preferences WHERE user_id=? AND notification_type=?",
		userID, notifType).Scan(&e, &deliveryMethod, &threshold)
	if err != nil {
		return true, "in_app", nil
	}
	return e == 1, deliveryMethod, threshold
}

// GenerateNotificationsFiltered generates notifications respecting per-user preferences.
func (h *Handler) GenerateNotificationsFiltered() {
	log.Println("Generating notifications (filtered)...")

	rows, err := h.DB.Query("SELECT id FROM users WHERE active=1")
	if err != nil {
		log.Println("Failed to get users for notification generation:", err)
		h.GenerateNotifications()
		return
	}
	defer rows.Close()

	var userIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		h.GenerateNotifications()
		return
	}

	for _, uid := range userIDs {
		h.EnsureDefaultPreferences(uid)
		h.GenerateNotificationsForUser(uid)
	}
}

// GenerateNotificationsForUser generates notifications for a specific user based on their preferences.
func (h *Handler) GenerateNotificationsForUser(userID int) {
	var pending []PendingNotif

	// Low stock
	enabled, deliveryMethod, threshold := h.GetUserNotifPref(userID, "low_stock")
	if enabled {
		rows, err := h.DB.Query(`SELECT ipn, qty_on_hand, reorder_point FROM inventory WHERE reorder_point > 0 AND qty_on_hand < reorder_point`)
		if err == nil {
			for rows.Next() {
				var ipn string
				var qty, rp float64
				rows.Scan(&ipn, &qty, &rp)
				if threshold != nil && qty >= *threshold {
					continue
				}
				p := PendingNotif{
					NType:          "low_stock",
					Severity:       "warning",
					Title:          "Low Stock: " + ipn,
					Message:        StringPtr(fmt.Sprintf("%.0f on hand, reorder point %.0f", qty, rp)),
					RecordID:       StringPtr(ipn),
					Module:         StringPtr("inventory"),
					DeliveryMethod: deliveryMethod,
					UserID:         userID,
				}
				pending = append(pending, p)
			}
			rows.Close()
		}
	}

	// Overdue work orders
	enabled, deliveryMethod, threshold = h.GetUserNotifPref(userID, "overdue_wo")
	if enabled {
		days := 7
		if threshold != nil {
			days = int(*threshold)
		}
		q := fmt.Sprintf(`SELECT id, assembly_ipn FROM work_orders WHERE status = 'in_progress' AND started_at < datetime('now', '-%d days')`, days)
		rows, err := h.DB.Query(q)
		if err == nil {
			for rows.Next() {
				var id, ipn string
				rows.Scan(&id, &ipn)
				p := PendingNotif{
					NType:          "overdue_wo",
					Severity:       "warning",
					Title:          "Overdue WO: " + id,
					Message:        StringPtr(fmt.Sprintf("In progress for >%d days: %s", days, ipn)),
					RecordID:       StringPtr(id),
					Module:         StringPtr("workorders"),
					DeliveryMethod: deliveryMethod,
					UserID:         userID,
				}
				pending = append(pending, p)
			}
			rows.Close()
		}
	}

	// Open NCRs > 14 days
	enabled, deliveryMethod, _ = h.GetUserNotifPref(userID, "open_ncr")
	if enabled {
		rows, err := h.DB.Query(`SELECT id, title FROM ncrs WHERE status = 'open' AND created_at < datetime('now', '-14 days')`)
		if err == nil {
			for rows.Next() {
				var id, title string
				rows.Scan(&id, &title)
				t := title
				p := PendingNotif{
					NType:          "open_ncr",
					Severity:       "error",
					Title:          "Open NCR >14d: " + id,
					Message:        &t,
					RecordID:       StringPtr(id),
					Module:         StringPtr("ncr"),
					DeliveryMethod: deliveryMethod,
					UserID:         userID,
				}
				pending = append(pending, p)
			}
			rows.Close()
		}
	}

	for _, p := range pending {
		h.CreateNotificationIfNew(p.NType, p.Severity, p.Title, p.Message, p.RecordID, p.Module)
		if p.DeliveryMethod == "email" || p.DeliveryMethod == "both" {
			if h.EmailConfigEnabled() {
				msg := ""
				if p.Message != nil {
					msg = *p.Message
				}
				go h.SendNotificationEmail(0, p.Title, msg)
			}
		}
	}
}
