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
	rows, err := db.Query(`SELECT id,title,type,COALESCE(linked_ncr_id,''),COALESCE(linked_rma_id,''),
		COALESCE(root_cause,''),COALESCE(action_plan,''),COALESCE(owner,''),COALESCE(due_date,''),
		status,COALESCE(effectiveness_check,''),COALESCE(approved_by_qe,''),approved_by_qe_at,
		COALESCE(approved_by_mgr,''),approved_by_mgr_at,created_at,updated_at
		FROM capas ORDER BY created_at DESC`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []CAPA
	for rows.Next() {
		var c CAPA
		var qeAt, mgrAt sql.NullString
		rows.Scan(&c.ID, &c.Title, &c.Type, &c.LinkedNCRID, &c.LinkedRMAID,
			&c.RootCause, &c.ActionPlan, &c.Owner, &c.DueDate,
			&c.Status, &c.EffectivenessCheck, &c.ApprovedByQE, &qeAt,
			&c.ApprovedByMgr, &mgrAt, &c.CreatedAt, &c.UpdatedAt)
		c.ApprovedByQEAt = sp(qeAt)
		c.ApprovedByMgrAt = sp(mgrAt)
		items = append(items, c)
	}
	if items == nil {
		items = []CAPA{}
	}
	jsonResp(w, items)
}

func handleGetCAPA(w http.ResponseWriter, r *http.Request, id string) {
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
		jsonErr(w, "not found", 404)
		return
	}
	c.ApprovedByQEAt = sp(qeAt)
	c.ApprovedByMgrAt = sp(mgrAt)
	jsonResp(w, c)
}

func handleCreateCAPA(w http.ResponseWriter, r *http.Request) {
	var c CAPA
	if err := decodeBody(r, &c); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	requireField(ve, "title", c.Title)
	validateMaxLength(ve, "title", c.Title, 255)
	validateMaxLength(ve, "root_cause", c.RootCause, 1000)
	validateMaxLength(ve, "action_plan", c.ActionPlan, 1000)
	validateMaxLength(ve, "owner", c.Owner, 255)
	validateMaxLength(ve, "effectiveness_check", c.EffectivenessCheck, 1000)
	if c.Type != "" { validateEnum(ve, "type", c.Type, validCAPATypes) }
	if c.Status != "" { validateEnum(ve, "status", c.Status, validCAPAStatuses) }
	validateDate(ve, "due_date", c.DueDate)
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	c.ID = nextID("CAPA", "capas", 3)
	if c.Status == "" {
		c.Status = "open"
	}
	if c.Type == "" {
		c.Type = "corrective"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT INTO capas (id,title,type,linked_ncr_id,linked_rma_id,root_cause,action_plan,owner,due_date,status,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.Title, c.Type, c.LinkedNCRID, c.LinkedRMAID, c.RootCause, c.ActionPlan, c.Owner, c.DueDate, c.Status, now, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	c.CreatedAt = now
	c.UpdatedAt = now
	logAudit(db, getUsername(r), "created", "capa", c.ID, "Created "+c.ID+": "+c.Title)
	recordChangeJSON(getUsername(r), "capas", c.ID, "create", nil, c)
	go emailOnCAPACreated(c)
	jsonResp(w, c)
}

func handleUpdateCAPA(w http.ResponseWriter, r *http.Request, id string) {
	oldSnap, _ := getCAPASnapshot(id)
	username := getUsername(r)
	_ = getUserRole(r)
	userID, err := getUserID(r)
	if err != nil {
		jsonErr(w, "authentication required", 401)
		return
	}

	var body map[string]interface{}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	getString := func(key string) string {
		if v, ok := body[key]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	title := getString("title")
	capaType := getString("type")
	linkedNCRID := getString("linked_ncr_id")
	linkedRMAID := getString("linked_rma_id")
	rootCause := getString("root_cause")
	actionPlan := getString("action_plan")
	owner := getString("owner")
	dueDate := getString("due_date")
	status := getString("status")
	effectivenessCheck := getString("effectiveness_check")
	
	// Handle approvals with proper RBAC (Gap 5.4)
	approvedByQE := getString("approved_by_qe")
	approvedByMgr := getString("approved_by_mgr")
	
	// Validate string lengths
	ve := &ValidationErrors{}
	validateMaxLength(ve, "title", title, 255)
	validateMaxLength(ve, "root_cause", rootCause, 1000)
	validateMaxLength(ve, "action_plan", actionPlan, 1000)
	validateMaxLength(ve, "owner", owner, 255)
	validateMaxLength(ve, "effectiveness_check", effectivenessCheck, 1000)
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }
	
	now := time.Now().Format("2006-01-02 15:04:05")

	// Get current CAPA state
	var currentCAPA CAPA
	var qeAt, mgrAt sql.NullString
	err = db.QueryRow(`SELECT id,title,type,COALESCE(linked_ncr_id,''),COALESCE(linked_rma_id,''),
		COALESCE(root_cause,''),COALESCE(action_plan,''),COALESCE(owner,''),COALESCE(due_date,''),
		status,COALESCE(effectiveness_check,''),COALESCE(approved_by_qe,''),approved_by_qe_at,
		COALESCE(approved_by_mgr,''),approved_by_mgr_at,created_at,updated_at
		FROM capas WHERE id=?`, id).
		Scan(&currentCAPA.ID, &currentCAPA.Title, &currentCAPA.Type, &currentCAPA.LinkedNCRID, &currentCAPA.LinkedRMAID,
			&currentCAPA.RootCause, &currentCAPA.ActionPlan, &currentCAPA.Owner, &currentCAPA.DueDate,
			&currentCAPA.Status, &currentCAPA.EffectivenessCheck, &currentCAPA.ApprovedByQE, &qeAt,
			&currentCAPA.ApprovedByMgr, &mgrAt, &currentCAPA.CreatedAt, &currentCAPA.UpdatedAt)
	if err != nil {
		jsonErr(w, "CAPA not found", 404)
		return
	}

	// Preserve current values if not provided in update
	if title == "" { title = currentCAPA.Title }
	if capaType == "" { capaType = currentCAPA.Type }
	if linkedNCRID == "" { linkedNCRID = currentCAPA.LinkedNCRID }
	if linkedRMAID == "" { linkedRMAID = currentCAPA.LinkedRMAID }
	if rootCause == "" { rootCause = currentCAPA.RootCause }
	if actionPlan == "" { actionPlan = currentCAPA.ActionPlan }
	if owner == "" { owner = currentCAPA.Owner }
	if dueDate == "" { dueDate = currentCAPA.DueDate }
	if effectivenessCheck == "" { effectivenessCheck = currentCAPA.EffectivenessCheck }

	// Validate status transitions (after merging current values)
	if status == "closed" && effectivenessCheck == "" {
		jsonErr(w, "effectiveness check required before closing", 400)
		return
	}
	if status == "closed" && (approvedByQE == "" || approvedByMgr == "") {
		// Check current approvals too
		checkApprovedByQE := approvedByQE
		checkApprovedByMgr := approvedByMgr
		if checkApprovedByQE == "" { checkApprovedByQE = currentCAPA.ApprovedByQE }
		if checkApprovedByMgr == "" { checkApprovedByMgr = currentCAPA.ApprovedByMgr }
		if checkApprovedByQE == "" || checkApprovedByMgr == "" {
			jsonErr(w, "QE and Manager approval required before closing", 400)
			return
		}
	}

	// Handle approval actions with security (Gap 5.4)
	var newQEAt, newMgrAt interface{}
	newApprovedByQE := currentCAPA.ApprovedByQE
	newApprovedByMgr := currentCAPA.ApprovedByMgr
	
	// QE approval security check
	if approvedByQE != "" && approvedByQE != currentCAPA.ApprovedByQE {
		if !canApproveCAPA(r, "qe") {
			jsonErr(w, "insufficient permissions: only QE role can approve as QE", 403)
			return
		}
		newApprovedByQE = fmt.Sprintf("%d", userID) // Store actual user ID
		newQEAt = now
	}
	
	// Manager approval security check
	if approvedByMgr != "" && approvedByMgr != currentCAPA.ApprovedByMgr {
		if !canApproveCAPA(r, "manager") {
			jsonErr(w, "insufficient permissions: only manager role can approve as manager", 403)
			return
		}
		newApprovedByMgr = fmt.Sprintf("%d", userID) // Store actual user ID
		newMgrAt = now
	}

	// Auto-advance status when both approvals are received (Gap 5.5)
	newStatus := status
	if status == "" {
		newStatus = currentCAPA.Status // Keep current status if not specified
	}
	
	// Check if we should auto-advance to pending_review
	if (newApprovedByQE != "" && newApprovedByMgr != "") && 
	   (currentCAPA.Status == "open" || currentCAPA.Status == "in_progress") &&
	   newStatus != "pending_review" && newStatus != "closed" {
		newStatus = "pending_review"
		logAudit(db, username, "auto-advanced", "capa", id, "Auto-advanced to pending_review status after both approvals received")
	}

	_, err = db.Exec(`UPDATE capas SET title=?,type=?,linked_ncr_id=?,linked_rma_id=?,root_cause=?,action_plan=?,
		owner=?,due_date=?,status=?,effectiveness_check=?,approved_by_qe=?,
		approved_by_qe_at=COALESCE(?,approved_by_qe_at),approved_by_mgr=?,
		approved_by_mgr_at=COALESCE(?,approved_by_mgr_at),updated_at=? WHERE id=?`,
		title, capaType, linkedNCRID, linkedRMAID, rootCause, actionPlan,
		owner, dueDate, newStatus, effectivenessCheck, newApprovedByQE,
		newQEAt, newApprovedByMgr, newMgrAt, now, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, username, "updated", "capa", id, "Updated "+id+": status="+newStatus)
	newSnap, _ := getCAPASnapshot(id)
	recordChangeJSON(username, "capas", id, "update", oldSnap, newSnap)

	handleGetCAPA(w, r, id)
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
	type OwnerSummary struct {
		Owner   string `json:"owner"`
		Count   int    `json:"count"`
		Overdue int    `json:"overdue"`
	}

	now := time.Now().Format("2006-01-02")
	rows, err := db.Query(`SELECT COALESCE(owner,'unassigned'), COUNT(*),
		SUM(CASE WHEN due_date < ? AND due_date != '' THEN 1 ELSE 0 END)
		FROM capas WHERE status NOT IN ('closed')
		GROUP BY owner ORDER BY COUNT(*) DESC`, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var summaries []OwnerSummary
	for rows.Next() {
		var s OwnerSummary
		rows.Scan(&s.Owner, &s.Count, &s.Overdue)
		summaries = append(summaries, s)
	}
	if summaries == nil {
		summaries = []OwnerSummary{}
	}

	// Also get total counts
	var totalOpen, totalOverdue int
	db.QueryRow("SELECT COUNT(*) FROM capas WHERE status NOT IN ('closed')").Scan(&totalOpen)
	db.QueryRow("SELECT COUNT(*) FROM capas WHERE status NOT IN ('closed') AND due_date < ? AND due_date != ''", now).Scan(&totalOverdue)

	jsonResp(w, map[string]interface{}{
		"total_open":    totalOpen,
		"total_overdue": totalOverdue,
		"by_owner":      summaries,
	})
}

// Email notification for new CAPA
func emailOnCAPACreated(c CAPA) {
	subject := fmt.Sprintf("New CAPA Created: %s - %s", c.ID, c.Title)
	body := fmt.Sprintf("A new %s CAPA has been created.\n\nID: %s\nTitle: %s\nOwner: %s\nDue Date: %s\nLinked NCR: %s\nLinked RMA: %s",
		c.Type, c.ID, c.Title, c.Owner, c.DueDate, c.LinkedNCRID, c.LinkedRMAID)

	rows, _ := db.Query("SELECT email FROM email_subscriptions WHERE event_type IN ('capa_created','all') AND email != ''")
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
