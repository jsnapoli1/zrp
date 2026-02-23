package admin

import (
	"net/http"

	"zrp/internal/audit"
	"zrp/internal/response"
)

// HandleGetEmailConfig returns the email configuration.
func (h *Handler) HandleGetEmailConfig(w http.ResponseWriter, r *http.Request) {
	var c EmailConfig
	err := h.DB.QueryRow("SELECT id, COALESCE(smtp_host,''), COALESCE(smtp_port,587), COALESCE(smtp_user,''), COALESCE(smtp_password,''), COALESCE(from_address,''), COALESCE(from_name,'ZRP'), enabled FROM email_config WHERE id=1").
		Scan(&c.ID, &c.SMTPHost, &c.SMTPPort, &c.SMTPUser, &c.SMTPPassword, &c.FromAddress, &c.FromName, &c.Enabled)
	if err != nil {
		response.JSON(w, EmailConfig{ID: 1, SMTPPort: 587, FromName: "ZRP"})
		return
	}
	if c.SMTPPassword != "" {
		c.SMTPPassword = "****"
	}
	response.JSON(w, c)
}

// HandleUpdateEmailConfig updates the email configuration.
func (h *Handler) HandleUpdateEmailConfig(w http.ResponseWriter, r *http.Request) {
	var c EmailConfig
	if err := response.DecodeBody(r, &c); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	if c.SMTPPassword == "****" {
		var existing string
		h.DB.QueryRow("SELECT COALESCE(smtp_password,'') FROM email_config WHERE id=1").Scan(&existing)
		c.SMTPPassword = existing
	}

	if c.SMTPPort <= 0 {
		c.SMTPPort = 587
	}

	_, err := h.DB.Exec(`INSERT OR REPLACE INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, from_name, enabled)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?)`,
		c.SMTPHost, c.SMTPPort, c.SMTPUser, c.SMTPPassword, c.FromAddress, c.FromName, c.Enabled)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "updated", "email_config", "1", "Updated email configuration")
	c.ID = 1
	if c.SMTPPassword != "" {
		c.SMTPPassword = "****"
	}
	response.JSON(w, c)
}

// HandleTestEmail sends a test email.
func (h *Handler) HandleTestEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To        string `json:"to"`
		TestEmail string `json:"test_email"`
	}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid request body", 400)
		return
	}
	// Support both "to" and "test_email" field names
	if body.To == "" {
		body.To = body.TestEmail
	}
	if body.To == "" {
		response.Err(w, "to address required", 400)
		return
	}

	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "test_email", "email_config", "1", "Test email to "+body.To)

	if err := h.SendEmail(body.To, "ZRP Test Email", "This is a test email from ZRP. If you received this, email notifications are configured correctly."); err != nil {
		response.Err(w, "send failed: "+err.Error(), 500)
		return
	}
	response.JSON(w, map[string]string{"status": "sent", "to": body.To})
}

// HandleListEmailLog returns the email log.
func (h *Handler) HandleListEmailLog(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, to_address, subject, COALESCE(body,''), COALESCE(event_type,''), status, COALESCE(error,''), sent_at FROM email_log ORDER BY sent_at DESC LIMIT 100")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []EmailLogEntry
	for rows.Next() {
		var e EmailLogEntry
		rows.Scan(&e.ID, &e.To, &e.Subject, &e.Body, &e.EventType, &e.Status, &e.Error, &e.SentAt)
		items = append(items, e)
	}
	if items == nil {
		items = []EmailLogEntry{}
	}
	response.JSON(w, items)
}

// HandleGetEmailSubscriptions returns the user's email subscriptions.
func (h *Handler) HandleGetEmailSubscriptions(w http.ResponseWriter, r *http.Request) {
	username := audit.GetUsername(h.DB, r)
	subs := make(map[string]bool)
	// Default all event types to enabled
	for _, et := range EmailEventTypes {
		subs[et] = true
	}
	rows, err := h.DB.Query("SELECT event_type, enabled FROM email_subscriptions WHERE user_id=?", username)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var eventType string
			var enabled int
			rows.Scan(&eventType, &enabled)
			subs[eventType] = enabled == 1
		}
	}
	response.JSON(w, subs)
}

// HandleUpdateEmailSubscriptions updates the user's email subscriptions.
func (h *Handler) HandleUpdateEmailSubscriptions(w http.ResponseWriter, r *http.Request) {
	username := audit.GetUsername(h.DB, r)
	var body map[string]bool
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	for eventType, enabled := range body {
		enabledInt := 0
		if enabled {
			enabledInt = 1
		}
		h.DB.Exec("INSERT OR REPLACE INTO email_subscriptions (user_id, event_type, enabled) VALUES (?, ?, ?)",
			username, eventType, enabledInt)
	}
	audit.LogAudit(h.DB, h.Hub, username, "updated", "email_subscriptions", username, "Updated email subscriptions")
	h.HandleGetEmailSubscriptions(w, r)
}
