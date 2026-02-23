package quality

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"zrp/internal/audit"
	"zrp/internal/database"
	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
)

// ListNCRs handles GET /api/v1/ncrs.
func (h *Handler) ListNCRs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),COALESCE(created_by,''),created_at,resolved_at FROM ncrs ORDER BY created_at DESC")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.NCR
	for rows.Next() {
		var n models.NCR
		var ra sql.NullString
		rows.Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedBy, &n.CreatedAt, &ra)
		n.ResolvedAt = database.SP(ra)
		items = append(items, n)
	}
	if items == nil {
		items = []models.NCR{}
	}
	response.JSON(w, items)
}

// GetNCR handles GET /api/v1/ncrs/:id.
func (h *Handler) GetNCR(w http.ResponseWriter, r *http.Request, id string) {
	var n models.NCR
	var ra sql.NullString
	err := h.DB.QueryRow("SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),COALESCE(created_by,''),created_at,resolved_at FROM ncrs WHERE id=?", id).
		Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedBy, &n.CreatedAt, &ra)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	n.ResolvedAt = database.SP(ra)
	response.JSON(w, n)
}

// CreateNCR handles POST /api/v1/ncrs.
func (h *Handler) CreateNCR(w http.ResponseWriter, r *http.Request) {
	var n models.NCR
	if err := response.DecodeBody(r, &n); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "title", n.Title)
	validation.ValidateMaxLength(ve, "title", n.Title, 255)
	validation.ValidateMaxLength(ve, "description", n.Description, 1000)
	validation.ValidateMaxLength(ve, "ipn", n.IPN, 100)
	validation.ValidateMaxLength(ve, "serial_number", n.SerialNumber, 100)
	validation.ValidateMaxLength(ve, "defect_type", n.DefectType, 255)
	validation.ValidateMaxLength(ve, "root_cause", n.RootCause, 1000)
	validation.ValidateMaxLength(ve, "corrective_action", n.CorrectiveAction, 1000)
	if n.Severity != "" {
		validation.ValidateEnum(ve, "severity", n.Severity, validation.ValidNCRSeverities)
	}
	if n.Status != "" {
		validation.ValidateEnum(ve, "status", n.Status, validation.ValidNCRStatuses)
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	n.ID = h.NextIDFunc("NCR", "ncrs", 3)
	if n.Status == "" {
		n.Status = "open"
	}
	if n.Severity == "" {
		n.Severity = "minor"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	username := audit.GetUsername(h.DB, r)

	// Add created_by field (Gap 5.2)
	_, err := h.DB.Exec("INSERT INTO ncrs (id,title,description,ipn,serial_number,defect_type,severity,status,created_by,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
		n.ID, n.Title, n.Description, n.IPN, n.SerialNumber, n.DefectType, n.Severity, n.Status, username, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	n.CreatedAt = now
	audit.LogAudit(h.DB, h.Hub, username, "created", "ncr", n.ID, "Created "+n.ID+": "+n.Title)
	h.RecordChangeJSON(audit.GetUsername(h.DB, r), "ncrs", n.ID, "create", nil, n)
	if h.EmailOnNCRCreated != nil {
		go h.EmailOnNCRCreated(n.ID, n.Title)
	}
	response.JSON(w, n)
}

// UpdateNCR handles PUT /api/v1/ncrs/:id.
func (h *Handler) UpdateNCR(w http.ResponseWriter, r *http.Request, id string) {
	oldSnap, _ := h.GetNCRSnapshot(id)
	var body map[string]interface{}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	// Extract fields
	getString := func(key string) string {
		if v, ok := body[key]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	getBool := func(key string) bool {
		if v, ok := body[key]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
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

	ve := &validation.ValidationErrors{}
	validation.ValidateMaxLength(ve, "title", title, 255)
	validation.ValidateMaxLength(ve, "description", description, 1000)
	validation.ValidateMaxLength(ve, "ipn", ipn, 100)
	validation.ValidateMaxLength(ve, "serial_number", serialNumber, 100)
	validation.ValidateMaxLength(ve, "defect_type", defectType, 255)
	validation.ValidateMaxLength(ve, "root_cause", rootCause, 1000)
	validation.ValidateMaxLength(ve, "corrective_action", correctiveAction, 1000)
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	var resolvedAt interface{}
	if status == "resolved" || status == "closed" {
		resolvedAt = now
	}
	_, err := h.DB.Exec("UPDATE ncrs SET title=?,description=?,ipn=?,serial_number=?,defect_type=?,severity=?,status=?,root_cause=?,corrective_action=?,resolved_at=COALESCE(?,resolved_at) WHERE id=?",
		title, description, ipn, serialNumber, defectType, severity, status, rootCause, correctiveAction, resolvedAt, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "updated", "ncr", id, "Updated "+id+": "+title)
	newSnap, _ := h.GetNCRSnapshot(id)
	h.RecordChangeJSON(username, "ncrs", id, "update", oldSnap, newSnap)

	// Auto-create linked ECO if resolving with corrective action
	var linkedECOID string
	if (status == "resolved" || status == "closed") && correctiveAction != "" && createECO {
		ecoID := h.NextIDFunc("ECO", "ecos", 3)
		ecoTitle := fmt.Sprintf("[NCR %s] %s — Corrective Action", id, title)
		affectedIPNs := ""
		if ipn != "" {
			affectedIPNs = ipn
		}
		_, err := h.DB.Exec("INSERT INTO ecos (id,title,description,status,priority,affected_ipns,created_by,created_at,updated_at,ncr_id) VALUES (?,?,?,?,?,?,?,?,?,?)",
			ecoID, ecoTitle, correctiveAction, "draft", "normal", affectedIPNs, username, now, now, id)
		if err == nil {
			linkedECOID = ecoID
			audit.LogAudit(h.DB, h.Hub, username, "created", "eco", ecoID, "Auto-created from NCR "+id)
		}
	}

	// Return NCR with linked ECO info
	var n models.NCR
	var ra sql.NullString
	err = h.DB.QueryRow("SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),created_at,resolved_at FROM ncrs WHERE id=?", id).
		Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedAt, &ra)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	n.ResolvedAt = database.SP(ra)

	if linkedECOID != "" {
		resp := map[string]interface{}{
			"id": n.ID, "title": n.Title, "description": n.Description,
			"ipn": n.IPN, "serial_number": n.SerialNumber, "defect_type": n.DefectType,
			"severity": n.Severity, "status": n.Status, "root_cause": n.RootCause,
			"corrective_action": n.CorrectiveAction, "created_at": n.CreatedAt,
			"resolved_at": n.ResolvedAt, "linked_eco_id": linkedECOID,
		}
		response.JSON(w, resp)
	} else {
		response.JSON(w, n)
	}
}
