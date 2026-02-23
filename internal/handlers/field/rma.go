package field

import (
	"database/sql"
	"net/http"
	"time"

	"zrp/internal/audit"
	"zrp/internal/database"
	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
)

// ListRMAs handles GET /api/rmas.
func (h *Handler) ListRMAs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id,serial_number,COALESCE(customer,''),COALESCE(reason,''),status,COALESCE(defect_description,''),COALESCE(resolution,''),created_at,received_at,resolved_at FROM rmas ORDER BY created_at DESC")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.RMA
	for rows.Next() {
		var rm models.RMA
		var ra, resa sql.NullString
		rows.Scan(&rm.ID, &rm.SerialNumber, &rm.Customer, &rm.Reason, &rm.Status, &rm.DefectDescription, &rm.Resolution, &rm.CreatedAt, &ra, &resa)
		rm.ReceivedAt = database.SP(ra)
		rm.ResolvedAt = database.SP(resa)
		items = append(items, rm)
	}
	if items == nil {
		items = []models.RMA{}
	}
	response.JSON(w, items)
}

// GetRMA handles GET /api/rmas/:id.
func (h *Handler) GetRMA(w http.ResponseWriter, r *http.Request, id string) {
	var rm models.RMA
	var ra, resa sql.NullString
	err := h.DB.QueryRow("SELECT id,serial_number,COALESCE(customer,''),COALESCE(reason,''),status,COALESCE(defect_description,''),COALESCE(resolution,''),created_at,received_at,resolved_at FROM rmas WHERE id=?", id).
		Scan(&rm.ID, &rm.SerialNumber, &rm.Customer, &rm.Reason, &rm.Status, &rm.DefectDescription, &rm.Resolution, &rm.CreatedAt, &ra, &resa)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	rm.ReceivedAt = database.SP(ra)
	rm.ResolvedAt = database.SP(resa)
	response.JSON(w, rm)
}

// CreateRMA handles POST /api/rmas.
func (h *Handler) CreateRMA(w http.ResponseWriter, r *http.Request) {
	var rm models.RMA
	if err := response.DecodeBody(r, &rm); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "serial_number", rm.SerialNumber)
	validation.RequireField(ve, "reason", rm.Reason)
	validation.ValidateMaxLength(ve, "serial_number", rm.SerialNumber, 100)
	validation.ValidateMaxLength(ve, "customer", rm.Customer, 255)
	validation.ValidateMaxLength(ve, "reason", rm.Reason, 255)
	validation.ValidateMaxLength(ve, "defect_description", rm.DefectDescription, 1000)
	validation.ValidateMaxLength(ve, "resolution", rm.Resolution, 1000)
	if rm.Status != "" {
		validation.ValidateEnum(ve, "status", rm.Status, validation.ValidRMAStatuses)
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	rm.ID = h.NextIDFunc("RMA", "rmas", 3)
	if rm.Status == "" {
		rm.Status = "open"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := h.DB.Exec("INSERT INTO rmas (id,serial_number,customer,reason,status,defect_description,created_at) VALUES (?,?,?,?,?,?,?)",
		rm.ID, rm.SerialNumber, rm.Customer, rm.Reason, rm.Status, rm.DefectDescription, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	rm.CreatedAt = now
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "created", "rma", rm.ID, "Created "+rm.ID+": "+rm.Reason)
	h.RecordChangeJSON(username, "rmas", rm.ID, "create", nil, rm)
	response.JSON(w, rm)
}

// UpdateRMA handles PUT /api/rmas/:id.
func (h *Handler) UpdateRMA(w http.ResponseWriter, r *http.Request, id string) {
	oldSnap, _ := h.GetRMASnapshot(id)
	var rm models.RMA
	if err := response.DecodeBody(r, &rm); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.ValidateMaxLength(ve, "serial_number", rm.SerialNumber, 100)
	validation.ValidateMaxLength(ve, "customer", rm.Customer, 255)
	validation.ValidateMaxLength(ve, "reason", rm.Reason, 255)
	validation.ValidateMaxLength(ve, "defect_description", rm.DefectDescription, 1000)
	validation.ValidateMaxLength(ve, "resolution", rm.Resolution, 1000)
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	var receivedAt, resolvedAt interface{}
	if rm.Status == "received" {
		receivedAt = now
	}
	if rm.Status == "closed" || rm.Status == "shipped" {
		resolvedAt = now
	}
	_, err := h.DB.Exec("UPDATE rmas SET serial_number=?,customer=?,reason=?,status=?,defect_description=?,resolution=?,received_at=COALESCE(?,received_at),resolved_at=COALESCE(?,resolved_at) WHERE id=?",
		rm.SerialNumber, rm.Customer, rm.Reason, rm.Status, rm.DefectDescription, rm.Resolution, receivedAt, resolvedAt, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "updated", "rma", id, "Updated "+id+": status="+rm.Status)
	newSnap, _ := h.GetRMASnapshot(id)
	h.RecordChangeJSON(username, "rmas", id, "update", oldSnap, newSnap)
	h.GetRMA(w, r, id)
}
