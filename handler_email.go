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
	Status    string `json:"status"`
	Error     string `json:"error"`
	SentAt    string `json:"sent_at"`
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
	rows, err := db.Query("SELECT id, to_address, subject, COALESCE(body,''), status, COALESCE(error,''), sent_at FROM email_log ORDER BY sent_at DESC LIMIT 100")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []EmailLogEntry
	for rows.Next() {
		var e EmailLogEntry
		rows.Scan(&e.ID, &e.To, &e.Subject, &e.Body, &e.Status, &e.Error, &e.SentAt)
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

func sendEmail(to, subject, body string) error {
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
	db.Exec("INSERT INTO email_log (to_address, subject, body, status, error, sent_at) VALUES (?, ?, ?, ?, ?, ?)",
		to, subject, body, status, errStr, time.Now().Format("2006-01-02 15:04:05"))

	return sendErr
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
	return strings.Contains(email, "@") && strings.Contains(email, ".")
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
	if err := sendEmail(userEmail, subject, body); err != nil {
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
	if err := sendEmail(c.FromAddress, subject, body); err != nil {
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
	if err := sendEmail(c.FromAddress, subject, body); err != nil {
		log.Printf("Failed to send overdue WO email: %v", err)
	}
}
