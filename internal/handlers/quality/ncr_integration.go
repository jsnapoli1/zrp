package quality

import (
	"fmt"
	"net/http"
	"time"

	"zrp/internal/audit"
	"zrp/internal/models"
	"zrp/internal/response"
)

// CreateCAPAFromNCR creates a CAPA directly from an NCR (Gap 5.1).
// POST /api/v1/ncrs/{id}/create-capa
func (h *Handler) CreateCAPAFromNCR(w http.ResponseWriter, r *http.Request, ncrID string) {
	// Get the NCR details first
	var ncr models.NCR
	err := h.DB.QueryRow(`SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),
		COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),
		created_at,resolved_at FROM ncrs WHERE id=?`, ncrID).
		Scan(&ncr.ID, &ncr.Title, &ncr.Description, &ncr.IPN, &ncr.SerialNumber,
			&ncr.DefectType, &ncr.Severity, &ncr.Status, &ncr.RootCause, &ncr.CorrectiveAction, &ncr.CreatedAt, &ncr.ResolvedAt)
	if err != nil {
		response.Err(w, "NCR not found", 404)
		return
	}

	// Parse optional request body for CAPA details
	var requestData struct {
		Title      string `json:"title"`
		Type       string `json:"type"`
		RootCause  string `json:"root_cause"`
		ActionPlan string `json:"action_plan"`
		Owner      string `json:"owner"`
		DueDate    string `json:"due_date"`
	}

	if r.ContentLength > 0 {
		if err := response.DecodeBody(r, &requestData); err != nil {
			response.Err(w, "invalid request body", 400)
			return
		}
	}

	// Create CAPA with auto-populated fields
	capaID := h.NextIDFunc("CAPA", "capas", 3)
	now := time.Now().Format("2006-01-02 15:04:05")

	// Auto-populate title if not provided
	title := requestData.Title
	if title == "" {
		title = fmt.Sprintf("CAPA for NCR %s: %s", ncr.ID, ncr.Title)
	}

	// Auto-populate type if not provided
	capaType := requestData.Type
	if capaType == "" {
		capaType = "corrective"
	}

	// Auto-populate root cause from NCR if available
	rootCause := requestData.RootCause
	if rootCause == "" && ncr.RootCause != "" {
		rootCause = ncr.RootCause
	}

	// Auto-populate action plan from NCR corrective action if available
	actionPlan := requestData.ActionPlan
	if actionPlan == "" && ncr.CorrectiveAction != "" {
		actionPlan = ncr.CorrectiveAction
	}

	_, err = h.DB.Exec(`INSERT INTO capas (id,title,type,linked_ncr_id,root_cause,action_plan,owner,due_date,
		status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		capaID, title, capaType, ncrID, rootCause, actionPlan,
		requestData.Owner, requestData.DueDate, "open", now, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Create the response CAPA object
	newCAPA := models.CAPA{
		ID:          capaID,
		Title:       title,
		Type:        capaType,
		LinkedNCRID: ncrID,
		RootCause:   rootCause,
		ActionPlan:  actionPlan,
		Owner:       requestData.Owner,
		DueDate:     requestData.DueDate,
		Status:      "open",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "created", "capa", capaID, fmt.Sprintf("Created CAPA from NCR %s", ncrID))
	h.RecordChangeJSON(username, "capas", capaID, "create", nil, newCAPA)
	if h.EmailOnCAPACreatedWithDB != nil {
		go h.EmailOnCAPACreatedWithDB(h.DB, newCAPA)
	}

	response.JSON(w, newCAPA)
}

// CreateECOFromNCR creates an ECO directly from an NCR (Gap 5.7).
// POST /api/v1/ncrs/{id}/create-eco
func (h *Handler) CreateECOFromNCR(w http.ResponseWriter, r *http.Request, ncrID string) {
	// Get the NCR details first
	var ncr models.NCR
	err := h.DB.QueryRow(`SELECT id,title,COALESCE(description,''),COALESCE(ipn,''),COALESCE(serial_number,''),
		COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),
		created_at,resolved_at FROM ncrs WHERE id=?`, ncrID).
		Scan(&ncr.ID, &ncr.Title, &ncr.Description, &ncr.IPN, &ncr.SerialNumber,
			&ncr.DefectType, &ncr.Severity, &ncr.Status, &ncr.RootCause, &ncr.CorrectiveAction, &ncr.CreatedAt, &ncr.ResolvedAt)
	if err != nil {
		response.Err(w, "NCR not found", 404)
		return
	}

	// Parse optional request body for ECO details
	var requestData struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		Priority     string `json:"priority"`
		AffectedIPNs string `json:"affected_ipns"`
	}

	if r.ContentLength > 0 {
		if err := response.DecodeBody(r, &requestData); err != nil {
			response.Err(w, "invalid request body", 400)
			return
		}
	}

	// Create ECO with auto-populated fields
	ecoID := h.NextIDFunc("ECO", "ecos", 3)
	now := time.Now().Format("2006-01-02 15:04:05")
	username := audit.GetUsername(h.DB, r)

	// Auto-populate title if not provided
	title := requestData.Title
	if title == "" {
		title = fmt.Sprintf("[NCR %s] %s — Corrective Action", ncr.ID, ncr.Title)
	}

	// Auto-populate description from NCR corrective action if available
	description := requestData.Description
	if description == "" && ncr.CorrectiveAction != "" {
		description = ncr.CorrectiveAction
	} else if description == "" && ncr.Description != "" {
		description = fmt.Sprintf("Corrective action for: %s", ncr.Description)
	}

	// Auto-populate priority based on NCR severity
	priority := requestData.Priority
	if priority == "" {
		switch ncr.Severity {
		case "critical":
			priority = "critical"
		case "major":
			priority = "high"
		default:
			priority = "normal"
		}
	}

	// Auto-populate affected IPNs from NCR if available
	affectedIPNs := requestData.AffectedIPNs
	if affectedIPNs == "" && ncr.IPN != "" {
		affectedIPNs = ncr.IPN
	}

	_, err = h.DB.Exec(`INSERT INTO ecos (id,title,description,status,priority,affected_ipns,created_by,
		created_at,updated_at,ncr_id) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		ecoID, title, description, "draft", priority, affectedIPNs, username, now, now, ncrID)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Create the response ECO object
	type ECO struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		Status       string `json:"status"`
		Priority     string `json:"priority"`
		AffectedIPNs string `json:"affected_ipns"`
		CreatedBy    string `json:"created_by"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		NCRID        string `json:"ncr_id"`
	}

	newECO := ECO{
		ID:           ecoID,
		Title:        title,
		Description:  description,
		Status:       "draft",
		Priority:     priority,
		AffectedIPNs: affectedIPNs,
		CreatedBy:    username,
		CreatedAt:    now,
		UpdatedAt:    now,
		NCRID:        ncrID,
	}

	audit.LogAudit(h.DB, h.Hub, username, "created", "eco", ecoID, fmt.Sprintf("Created ECO from NCR %s", ncrID))
	h.RecordChangeJSON(username, "ecos", ecoID, "create", nil, newECO)

	response.JSON(w, newECO)
}
