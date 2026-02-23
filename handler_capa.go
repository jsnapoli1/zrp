package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

// CAPA represents a Corrective and Preventive Action
type CAPA struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	Type               string  `json:"type"` // corrective or preventive
	LinkedNCRID        string  `json:"linked_ncr_id"`
	LinkedRMAID        string  `json:"linked_rma_id"`
	RootCause          string  `json:"root_cause"`
	ActionPlan         string  `json:"action_plan"`
	Owner              string  `json:"owner"`
	DueDate            string  `json:"due_date"`
	Status             string  `json:"status"` // open, in-progress, verification, closed
	EffectivenessCheck string  `json:"effectiveness_check"`
	ApprovedByQE       string  `json:"approved_by_qe"`
	ApprovedByQEAt     *string `json:"approved_by_qe_at"`
	ApprovedByMgr      string  `json:"approved_by_mgr"`
	ApprovedByMgrAt    *string `json:"approved_by_mgr_at"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

func handleListCAPAs(w http.ResponseWriter, r *http.Request) {
	getQualityHandler().ListCAPAs(w, r)
}

func handleGetCAPA(w http.ResponseWriter, r *http.Request, id string) {
	getQualityHandler().GetCAPA(w, r, id)
}

func handleCreateCAPA(w http.ResponseWriter, r *http.Request) {
	getQualityHandler().CreateCAPA(w, r)
}

func handleUpdateCAPA(w http.ResponseWriter, r *http.Request, id string) {
	getQualityHandler().UpdateCAPA(w, r, id)
}

func getCAPASnapshot(id string) (map[string]interface{}, error) {
	var c CAPA
	var qeAt, mgrAt sql.NullString
	err := db.QueryRow(`SELECT id,title,type,COALESCE(linked_ncr_id,''),COALESCE(linked_rma_id,''),
		COALESCE(root_cause,''),COALESCE(action_plan,''),COALESCE(owner,''),COALESCE(due_date,''),
		status,COALESCE(effectiveness_check,''),COALESCE(approved_by_qe,''),approved_by_qe_at,
		COALESCE(approved_by_mgr,''),approved_by_mgr_at,created_at,updated_at
		FROM capas WHERE id=?`, id).
		Scan(&c.ID, &c.Title, &c.Type, &c.LinkedNCRID, &c.LinkedRMAID,
			&c.RootCause, &c.ActionPlan, &c.Owner, &c.DueDate,
			&c.Status, &c.EffectivenessCheck, &c.ApprovedByQE, &qeAt,
			&c.ApprovedByMgr, &mgrAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": c.ID, "title": c.Title, "type": c.Type,
		"linked_ncr_id": c.LinkedNCRID, "linked_rma_id": c.LinkedRMAID,
		"root_cause": c.RootCause, "action_plan": c.ActionPlan,
		"owner": c.Owner, "due_date": c.DueDate, "status": c.Status,
		"effectiveness_check": c.EffectivenessCheck,
	}, nil
}

// Dashboard: open CAPAs summary
func handleCAPADashboard(w http.ResponseWriter, r *http.Request) {
	getQualityHandler().CAPADashboard(w, r)
}

// Email notification for new CAPA
func emailOnCAPACreated(c CAPA) {
	emailOnCAPACreatedWithDB(db, c)
}

func emailOnCAPACreatedWithDB(database *sql.DB, c CAPA) {
	if database == nil {
		return
	}
	subject := fmt.Sprintf("New CAPA Created: %s - %s", c.ID, c.Title)
	body := fmt.Sprintf("A new %s CAPA has been created.\n\nID: %s\nTitle: %s\nOwner: %s\nDue Date: %s\nLinked NCR: %s\nLinked RMA: %s",
		c.Type, c.ID, c.Title, c.Owner, c.DueDate, c.LinkedNCRID, c.LinkedRMAID)

	rows, _ := database.Query("SELECT email FROM email_subscriptions WHERE event_type IN ('capa_created','all') AND email != ''")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var email string
			rows.Scan(&email)
			sendEmail(email, subject, body)
		}
	}
}

// Check for overdue CAPAs and send notifications
func checkOverdueCAPAs() {
	now := time.Now().Format("2006-01-02")
	rows, err := db.Query(`SELECT id,title,owner,due_date FROM capas
		WHERE status NOT IN ('closed') AND due_date < ? AND due_date != ''`, now)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, owner, dueDate string
		rows.Scan(&id, &title, &owner, &dueDate)

		subject := fmt.Sprintf("OVERDUE CAPA: %s - %s", id, title)
		body := fmt.Sprintf("CAPA %s is overdue.\n\nTitle: %s\nOwner: %s\nDue Date: %s\n\nPlease take action immediately.",
			id, title, owner, dueDate)

		subRows, _ := db.Query("SELECT email FROM email_subscriptions WHERE event_type IN ('capa_overdue','all') AND email != ''")
		if subRows != nil {
			for subRows.Next() {
				var email string
				subRows.Scan(&email)
				sendEmail(email, subject, body)
			}
			subRows.Close()
		}
	}
}
