package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// SMTPSendFunc is the function used to send emails. Override in tests.
var SMTPSendFunc = smtp.SendMail

type EmailConfig struct {
	ID           int    `json:"id"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	FromAddress  string `json:"from_address"`
	FromName     string `json:"from_name"`
	Enabled      int    `json:"enabled"`
}

type EmailLogEntry struct {
	ID        int    `json:"id"`
	To        string `json:"to_address"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	EventType string `json:"event_type"`
	Status    string `json:"status"`
	Error     string `json:"error"`
	SentAt    string `json:"sent_at"`
}

type EmailSubscription struct {
	ID        int    `json:"id"`
	UserID    string `json:"user_id"`
	EventType string `json:"event_type"`
	Enabled   int    `json:"enabled"`
}

// All supported email event types
var EmailEventTypes = []string{
	"eco_approved",
	"eco_implemented",
	"low_stock",
	"overdue_work_order",
	"po_received",
	"ncr_created",
}

func handleGetEmailConfig(w http.ResponseWriter, r *http.Request) {
	var c EmailConfig
	err := db.QueryRow("SELECT id, COALESCE(smtp_host,''), COALESCE(smtp_port,587), COALESCE(smtp_user,''), COALESCE(smtp_password,''), COALESCE(from_address,''), COALESCE(from_name,'ZRP'), enabled FROM email_config WHERE id=1").
		Scan(&c.ID, &c.SMTPHost, &c.SMTPPort, &c.SMTPUser, &c.SMTPPassword, &c.FromAddress, &c.FromName, &c.Enabled)
	if err != nil {
		jsonResp(w, EmailConfig{ID: 1, SMTPPort: 587, FromName: "ZRP"})
		return
	}
	if c.SMTPPassword != "" {
		c.SMTPPassword = "****"
	}
	jsonResp(w, c)
}

func handleUpdateEmailConfig(w http.ResponseWriter, r *http.Request) {
	var c EmailConfig
	if err := decodeBody(r, &c); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	if c.SMTPPassword == "****" {
		var existing string
		db.QueryRow("SELECT COALESCE(smtp_password,'') FROM email_config WHERE id=1").Scan(&existing)
		c.SMTPPassword = existing
	}

	if c.SMTPPort <= 0 {
		c.SMTPPort = 587
	}

	_, err := db.Exec(`INSERT OR REPLACE INTO email_config (id, smtp_host, smtp_port, smtp_user, smtp_password, from_address, from_name, enabled)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?)`,
		c.SMTPHost, c.SMTPPort, c.SMTPUser, c.SMTPPassword, c.FromAddress, c.FromName, c.Enabled)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	logAudit(db, getUsername(r), "updated", "email_config", "1", "Updated email configuration")
	c.ID = 1
	if c.SMTPPassword != "" {
		c.SMTPPassword = "****"
	}
	jsonResp(w, c)
}

func handleTestEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To        string `json:"to"`
		TestEmail string `json:"test_email"`
	}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid request body", 400)
		return
	}
	// Support both "to" and "test_email" field names
	if body.To == "" {
		body.To = body.TestEmail
	}
	if body.To == "" {
		jsonErr(w, "to address required", 400)
		return
	}

	logAudit(db, getUsername(r), "test_email", "email_config", "1", "Test email to "+body.To)

	if err := sendEmail(body.To, "ZRP Test Email", "This is a test email from ZRP. If you received this, email notifications are configured correctly."); err != nil {
		jsonErr(w, "send failed: "+err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "sent", "to": body.To})
}

func handleListEmailLog(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, to_address, subject, COALESCE(body,''), COALESCE(event_type,''), status, COALESCE(error,''), sent_at FROM email_log ORDER BY sent_at DESC LIMIT 100")
	if err != nil {
		jsonErr(w, err.Error(), 500)
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
	jsonResp(w, items)
}

func getEmailConfig() (*EmailConfig, error) {
	var c EmailConfig
	err := db.QueryRow("SELECT id, COALESCE(smtp_host,''), COALESCE(smtp_port,587), COALESCE(smtp_user,''), COALESCE(smtp_password,''), COALESCE(from_address,''), COALESCE(from_name,'ZRP'), enabled FROM email_config WHERE id=1").
		Scan(&c.ID, &c.SMTPHost, &c.SMTPPort, &c.SMTPUser, &c.SMTPPassword, &c.FromAddress, &c.FromName, &c.Enabled)
	if err != nil {
		return nil, err
	}
	if c.Enabled == 0 || c.SMTPHost == "" {
		return nil, fmt.Errorf("email not configured or disabled")
	}
	return &c, nil
}

func sendEmailWithEvent(to, subject, body, eventType string) error {
	c, err := getEmailConfig()
	if err != nil {
		return err
	}

	from := c.FromAddress
	if from == "" {
		from = c.SMTPUser
	}

	msg := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		c.FromName, from, to, subject, body)

	addr := fmt.Sprintf("%s:%d", c.SMTPHost, c.SMTPPort)
	var auth smtp.Auth
	if c.SMTPUser != "" {
		auth = smtp.PlainAuth("", c.SMTPUser, c.SMTPPassword, c.SMTPHost)
	}

	sendErr := SMTPSendFunc(addr, auth, from, []string{to}, []byte(msg))

	// Log the email
	status := "sent"
	errStr := ""
	if sendErr != nil {
		status = "failed"
		errStr = sendErr.Error()
	}
	db.Exec("INSERT INTO email_log (to_address, subject, body, event_type, status, error, sent_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		to, subject, body, eventType, status, errStr, time.Now().Format("2006-01-02 15:04:05"))

	return sendErr
}

func sendEmail(to, subject, body string) error {
	return sendEmailWithEvent(to, subject, body, "")
}

// sendNotificationEmail sends an email for a notification if email is configured
func sendNotificationEmail(notifID int, title, message string) {
	c, err := getEmailConfig()
	if err != nil {
		return
	}

	var emailed int
	db.QueryRow("SELECT COALESCE(emailed, 0) FROM notifications WHERE id=?", notifID).Scan(&emailed)
	if emailed == 1 {
		return
	}

	body := fmt.Sprintf("Notification: %s\n\n%s\n\n— ZRP", title, message)
	recipient := c.FromAddress
	if recipient == "" {
		return
	}

	if err := sendEmail(recipient, "ZRP: "+title, body); err != nil {
		log.Printf("Failed to send notification email: %v", err)
		return
	}

	db.Exec("UPDATE notifications SET emailed=1 WHERE id=?", notifID)
}

// emailNotificationsForRecent checks recent unread notifications and sends emails
func emailNotificationsForRecent() {
	rows, err := db.Query("SELECT id, title, COALESCE(message,'') FROM notifications WHERE COALESCE(emailed,0)=0 AND created_at > datetime('now', '-10 minutes')")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var title, message string
		rows.Scan(&id, &title, &message)
		sendNotificationEmail(id, title, message)
	}
}

func emailConfigEnabled() bool {
	var enabled int
	err := db.QueryRow("SELECT enabled FROM email_config WHERE id=1").Scan(&enabled)
	return err == nil && enabled == 1
}

func isValidEmail(email string) bool {
	if !strings.Contains(email, "@") {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	local, domain := parts[0], parts[1]
	if local == "" || domain == "" {
		return false
	}
	return strings.Contains(domain, ".")
}

// --- Subscription management ---

func handleGetEmailSubscriptions(w http.ResponseWriter, r *http.Request) {
	username := getUsername(r)
	subs := make(map[string]bool)
	// Default all event types to enabled
	for _, et := range EmailEventTypes {
		subs[et] = true
	}
	rows, err := db.Query("SELECT event_type, enabled FROM email_subscriptions WHERE user_id=?", username)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var eventType string
			var enabled int
			rows.Scan(&eventType, &enabled)
			subs[eventType] = enabled == 1
		}
	}
	jsonResp(w, subs)
}

func handleUpdateEmailSubscriptions(w http.ResponseWriter, r *http.Request) {
	username := getUsername(r)
	var body map[string]bool
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	for eventType, enabled := range body {
		enabledInt := 0
		if enabled {
			enabledInt = 1
		}
		db.Exec("INSERT OR REPLACE INTO email_subscriptions (user_id, event_type, enabled) VALUES (?, ?, ?)",
			username, eventType, enabledInt)
	}
	logAudit(db, username, "updated", "email_subscriptions", username, "Updated email subscriptions")
	handleGetEmailSubscriptions(w, r)
}

// isUserSubscribed checks if a user has opted out of a specific event type.
// Default is subscribed (true) unless explicitly disabled.
func isUserSubscribed(username, eventType string) bool {
	var enabled int
	err := db.QueryRow("SELECT enabled FROM email_subscriptions WHERE user_id=? AND event_type=?", username, eventType).Scan(&enabled)
	if err != nil {
		return true // default: subscribed
	}
	return enabled == 1
}

// sendEventEmail sends an email for a specific event type, checking subscription.
func sendEventEmail(to, subject, body, eventType, username string) error {
	if username != "" && !isUserSubscribed(username, eventType) {
		return nil
	}
	return sendEmailWithEvent(to, subject, body, eventType)
}

// --- Email triggers for key events ---

// emailOnECOApproved sends email to the ECO creator when an ECO is approved.
func emailOnECOApproved(ecoID string) {
	if !emailConfigEnabled() {
		return
	}
	var title, createdBy string
	err := db.QueryRow("SELECT title, created_by FROM ecos WHERE id=?", ecoID).Scan(&title, &createdBy)
	if err != nil {
		return
	}
	// Look up user email (from_address as fallback)
	var userEmail string
	db.QueryRow("SELECT COALESCE(email,'') FROM users WHERE username=?", createdBy).Scan(&userEmail)
	if userEmail == "" {
		// fallback to admin from_address
		c, err := getEmailConfig()
		if err != nil {
			return
		}
		userEmail = c.FromAddress
	}
	if userEmail == "" || !isValidEmail(userEmail) {
		return
	}
	subject := fmt.Sprintf("ECO %s Approved", ecoID)
	body := fmt.Sprintf("Your Engineering Change Order %s (%s) has been approved.\n\n— ZRP", ecoID, title)
	if err := sendEventEmail(userEmail, subject, body, "eco_approved", createdBy); err != nil {
		log.Printf("Failed to send ECO approval email: %v", err)
	}
}

// emailOnLowStock sends email to admin when an inventory item drops below reorder point.
func emailOnLowStock(ipn string) {
	if !emailConfigEnabled() {
		return
	}
	var qtyOnHand, reorderPoint int
	err := db.QueryRow("SELECT qty_on_hand, reorder_point FROM inventory WHERE ipn=?", ipn).Scan(&qtyOnHand, &reorderPoint)
	if err != nil || reorderPoint <= 0 || qtyOnHand > reorderPoint {
		return
	}
	c, err := getEmailConfig()
	if err != nil || c.FromAddress == "" {
		return
	}
	subject := fmt.Sprintf("Low Stock Alert: %s", ipn)
	body := fmt.Sprintf("Inventory item %s has dropped below its reorder point.\n\nCurrent qty: %d\nReorder point: %d\n\n— ZRP", ipn, qtyOnHand, reorderPoint)
	if err := sendEventEmail(c.FromAddress, subject, body, "low_stock", ""); err != nil {
		log.Printf("Failed to send low stock email: %v", err)
	}
}

// emailOnOverdueWorkOrder sends email to admin when a WO is past due and not closed/completed.
func emailOnOverdueWorkOrder(woID string) {
	if !emailConfigEnabled() {
		return
	}
	var status string
	var dueDate *string
	err := db.QueryRow("SELECT status, due_date FROM work_orders WHERE id=?", woID).Scan(&status, &dueDate)
	if err != nil || dueDate == nil || *dueDate == "" {
		return
	}
	if status == "closed" || status == "completed" {
		return
	}
	due, err := time.Parse("2006-01-02", *dueDate)
	if err != nil {
		return
	}
	if time.Now().Before(due) {
		return
	}
	c, err := getEmailConfig()
	if err != nil || c.FromAddress == "" {
		return
	}
	subject := fmt.Sprintf("Overdue Work Order: %s", woID)
	body := fmt.Sprintf("Work Order %s is past its due date (%s) and has status '%s'.\n\n— ZRP", woID, *dueDate, status)
	if err := sendEventEmail(c.FromAddress, subject, body, "overdue_work_order", ""); err != nil {
		log.Printf("Failed to send overdue WO email: %v", err)
	}
}

// emailOnECOImplemented sends email to all affected part owners when an ECO is implemented.
func emailOnECOImplemented(ecoID string) {
	if !emailConfigEnabled() {
		return
	}
	var title, affectedIPNs string
	err := db.QueryRow("SELECT title, COALESCE(affected_ipns,'') FROM ecos WHERE id=?", ecoID).Scan(&title, &affectedIPNs)
	if err != nil {
		return
	}
	c, err := getEmailConfig()
	if err != nil || c.FromAddress == "" {
		return
	}
	subject := fmt.Sprintf("ECO %s Implemented", ecoID)
	body := fmt.Sprintf("Engineering Change Order %s (%s) has been implemented.\n\nAffected parts: %s\n\n— ZRP", ecoID, title, affectedIPNs)
	if err := sendEventEmail(c.FromAddress, subject, body, "eco_implemented", ""); err != nil {
		log.Printf("Failed to send ECO implemented email: %v", err)
	}
}

// emailOnPOReceived sends email to the PO creator when a PO is received.
func emailOnPOReceived(poID string) {
	if !emailConfigEnabled() {
		return
	}
	var createdBy string
	db.QueryRow("SELECT COALESCE(created_by,'') FROM purchase_orders WHERE id=?", poID).Scan(&createdBy)
	var userEmail string
	if createdBy != "" {
		db.QueryRow("SELECT COALESCE(email,'') FROM users WHERE username=?", createdBy).Scan(&userEmail)
	}
	if userEmail == "" || !isValidEmail(userEmail) {
		c, err := getEmailConfig()
		if err != nil || c.FromAddress == "" {
			return
		}
		userEmail = c.FromAddress
		createdBy = ""
	}
	subject := fmt.Sprintf("PO %s Received", poID)
	body := fmt.Sprintf("Purchase Order %s has been received.\n\n— ZRP", poID)
	if err := sendEventEmail(userEmail, subject, body, "po_received", createdBy); err != nil {
		log.Printf("Failed to send PO received email: %v", err)
	}
}

// emailOnNCRCreated sends email to quality team when an NCR is created.
func emailOnNCRCreated(ncrID, title string) {
	if !emailConfigEnabled() {
		return
	}
	c, err := getEmailConfig()
	if err != nil || c.FromAddress == "" {
		return
	}
	subject := fmt.Sprintf("NCR Created: %s", ncrID)
	body := fmt.Sprintf("A new Non-Conformance Report has been created.\n\nNCR: %s\nTitle: %s\n\n— ZRP", ncrID, title)
	if err := sendEventEmail(c.FromAddress, subject, body, "ncr_created", ""); err != nil {
		log.Printf("Failed to send NCR created email: %v", err)
	}
}
