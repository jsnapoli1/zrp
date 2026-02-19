package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

func handleListNCRs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),COALESCE(created_by,''),created_at,resolved_at FROM ncrs ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []NCR
	for rows.Next() {
		var n NCR
		var ra sql.NullString
		rows.Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedBy, &n.CreatedAt, &ra)
		n.ResolvedAt = sp(ra)
		items = append(items, n)
	}
	if items == nil { items = []NCR{} }
	jsonResp(w, items)
}

func handleGetNCR(w http.ResponseWriter, r *http.Request, id string) {
	var n NCR
	var ra sql.NullString
	err := db.QueryRow("SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),COALESCE(created_by,''),created_at,resolved_at FROM ncrs WHERE id=?", id).
		Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedBy, &n.CreatedAt, &ra)
	if err != nil { jsonErr(w, "not found", 404); return }
	n.ResolvedAt = sp(ra)
	jsonResp(w, n)
}

func handleCreateNCR(w http.ResponseWriter, r *http.Request) {
	var n NCR
	if err := decodeBody(r, &n); err != nil { jsonErr(w, "invalid body", 400); return }

	ve := &ValidationErrors{}
	requireField(ve, "title", n.Title)
	validateMaxLength(ve, "title", n.Title, 500)
	if n.Severity != "" { validateEnum(ve, "severity", n.Severity, validNCRSeverities) }
	if n.Status != "" { validateEnum(ve, "status", n.Status, validNCRStatuses) }
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	n.ID = nextID("NCR", "ncrs", 3)
	if n.Status == "" { n.Status = "open" }
	if n.Severity == "" { n.Severity = "minor" }
	now := time.Now().Format("2006-01-02 15:04:05")
	username := getUsername(r)
	
	// Add created_by field (Gap 5.2)
	_, err := db.Exec("INSERT INTO ncrs (id,title,description,ipn,serial_number,defect_type,severity,status,created_by,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
		n.ID, n.Title, n.Description, n.IPN, n.SerialNumber, n.DefectType, n.Severity, n.Status, username, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	n.CreatedAt = now
	logAudit(db, username, "created", "ncr", n.ID, "Created "+n.ID+": "+n.Title)
	recordChangeJSON(getUsername(r), "ncrs", n.ID, "create", nil, n)
	go emailOnNCRCreated(n.ID, n.Title)
	jsonResp(w, n)
}

func handleUpdateNCR(w http.ResponseWriter, r *http.Request, id string) {
	oldSnap, _ := getNCRSnapshot(id)
	var body map[string]interface{}
	if err := decodeBody(r, &body); err != nil { jsonErr(w, "invalid body", 400); return }

	// Extract fields
	getString := func(key string) string {
		if v, ok := body[key]; ok && v != nil { return fmt.Sprintf("%v", v) }
		return ""
	}
	getBool := func(key string) bool {
		if v, ok := body[key]; ok {
			if b, ok := v.(bool); ok { return b }
		}
		return false
	}

	title := getString("title")
	description := getString("description")
	ipn := getString("ipn")
	serialNumber := getString("serial_number")
	defectType := getString("defect_type")
	severity := getString("severity")
	status := getString("status")
	rootCause := getString("root_cause")
	correctiveAction := getString("corrective_action")
	createECO := getBool("create_eco")

	now := time.Now().Format("2006-01-02 15:04:05")
	var resolvedAt interface{}
	if status == "resolved" || status == "closed" {
		resolvedAt = now
	}
	_, err := db.Exec("UPDATE ncrs SET title=?,description=?,ipn=?,serial_number=?,defect_type=?,severity=?,status=?,root_cause=?,corrective_action=?,resolved_at=COALESCE(?,resolved_at) WHERE id=?",
		title, description, ipn, serialNumber, defectType, severity, status, rootCause, correctiveAction, resolvedAt, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "ncr", id, "Updated "+id+": "+title)
	newSnap, _ := getNCRSnapshot(id)
	recordChangeJSON(getUsername(r), "ncrs", id, "update", oldSnap, newSnap)

	// Auto-create linked ECO if resolving with corrective action
	var linkedECOID string
	if (status == "resolved" || status == "closed") && correctiveAction != "" && createECO {
		ecoID := nextID("ECO", "ecos", 3)
		ecoTitle := fmt.Sprintf("[NCR %s] %s — Corrective Action", id, title)
		affectedIPNs := ""
		if ipn != "" { affectedIPNs = ipn }
		_, err := db.Exec("INSERT INTO ecos (id,title,description,status,priority,affected_ipns,created_by,created_at,updated_at,ncr_id) VALUES (?,?,?,?,?,?,?,?,?,?)",
			ecoID, ecoTitle, correctiveAction, "draft", "normal", affectedIPNs, getUsername(r), now, now, id)
		if err == nil {
			linkedECOID = ecoID
			logAudit(db, getUsername(r), "created", "eco", ecoID, "Auto-created from NCR "+id)
		}
	}

	// Return NCR with linked ECO info
	var n NCR
	var ra sql.NullString
	err = db.QueryRow("SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),created_at,resolved_at FROM ncrs WHERE id=?", id).
		Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedAt, &ra)
	if err != nil { jsonErr(w, "not found", 404); return }
	n.ResolvedAt = sp(ra)

	if linkedECOID != "" {
		resp := map[string]interface{}{
			"id": n.ID, "title": n.Title, "description": n.Description,
			"ipn": n.IPN, "serial_number": n.SerialNumber, "defect_type": n.DefectType,
			"severity": n.Severity, "status": n.Status, "root_cause": n.RootCause,
			"corrective_action": n.CorrectiveAction, "created_at": n.CreatedAt,
			"resolved_at": n.ResolvedAt, "linked_eco_id": linkedECOID,
		}
		jsonResp(w, resp)
	} else {
		jsonResp(w, n)
	}
}
