package procurement

import (
	"fmt"
	"net/http"

	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
)

// ListVendors returns all vendors.
func (h *Handler) ListVendors(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		response.Err(w, "database not initialized", 503)
		return
	}
	rows, err := h.DB.Query("SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),status,lead_time_days,created_at FROM vendors ORDER BY name")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.Vendor
	for rows.Next() {
		var v models.Vendor
		rows.Scan(&v.ID, &v.Name, &v.Website, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.LeadTimeDays, &v.CreatedAt)
		items = append(items, v)
	}
	if items == nil {
		items = []models.Vendor{}
	}
	response.JSON(w, items)
}

// GetVendor returns a single vendor by ID.
func (h *Handler) GetVendor(w http.ResponseWriter, r *http.Request, id string) {
	if h.DB == nil {
		response.Err(w, "database not initialized", 503)
		return
	}
	var v models.Vendor
	err := h.DB.QueryRow("SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),status,lead_time_days,created_at FROM vendors WHERE id=?", id).
		Scan(&v.ID, &v.Name, &v.Website, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.LeadTimeDays, &v.CreatedAt)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	response.JSON(w, v)
}

// CreateVendor creates a new vendor.
func (h *Handler) CreateVendor(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil {
		response.Err(w, "database not initialized", 503)
		return
	}
	var v models.Vendor
	if err := response.DecodeBody(r, &v); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "name", v.Name)
	validation.ValidateMaxLength(ve, "name", v.Name, 255)
	validation.ValidateMaxLength(ve, "contact_name", v.ContactName, 255)
	validation.ValidateMaxLength(ve, "notes", v.Notes, 10000)
	validation.ValidateMaxLength(ve, "website", v.Website, 255)
	validation.ValidateMaxLength(ve, "contact_phone", v.ContactPhone, 50)
	validation.ValidateEmail(ve, "contact_email", v.ContactEmail)
	if v.Status != "" {
		validation.ValidateEnum(ve, "status", v.Status, validation.ValidVendorStatuses)
	}
	if v.LeadTimeDays < 0 {
		ve.Add("lead_time_days", "must be non-negative")
	}
	validation.ValidateIntRange(ve, "lead_time_days", v.LeadTimeDays, 0, validation.MaxLeadTimeDays)
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	var maxNum int
	h.DB.QueryRow("SELECT COALESCE(MAX(CAST(SUBSTR(id,3) AS INTEGER)),0) FROM vendors WHERE id LIKE 'V-%'").Scan(&maxNum)
	v.ID = fmt.Sprintf("V-%03d", maxNum+1)
	if v.Status == "" {
		v.Status = "active"
	}
	_, err := h.DB.Exec("INSERT INTO vendors (id,name,website,contact_name,contact_email,contact_phone,notes,status,lead_time_days) VALUES (?,?,?,?,?,?,?,?,?)",
		v.ID, v.Name, v.Website, v.ContactName, v.ContactEmail, v.ContactPhone, v.Notes, v.Status, v.LeadTimeDays)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	username := h.GetUsername(r)
	h.LogAudit(username, "created", "vendor", v.ID, "Created vendor "+v.Name)
	h.RecordChangeJSON(username, "vendors", v.ID, "create", nil, v)
	response.JSON(w, v)
}

// UpdateVendor updates an existing vendor.
func (h *Handler) UpdateVendor(w http.ResponseWriter, r *http.Request, id string) {
	// Snapshot old data before update
	oldSnap, _ := h.GetVendorSnapshot(id)

	var v models.Vendor
	if err := response.DecodeBody(r, &v); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.ValidateMaxLength(ve, "name", v.Name, 255)
	validation.ValidateMaxLength(ve, "contact_name", v.ContactName, 255)
	validation.ValidateMaxLength(ve, "notes", v.Notes, 10000)
	validation.ValidateMaxLength(ve, "website", v.Website, 255)
	validation.ValidateMaxLength(ve, "contact_phone", v.ContactPhone, 50)
	validation.ValidateEmail(ve, "contact_email", v.ContactEmail)
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	_, err := h.DB.Exec("UPDATE vendors SET name=?,website=?,contact_name=?,contact_email=?,contact_phone=?,notes=?,status=?,lead_time_days=? WHERE id=?",
		v.Name, v.Website, v.ContactName, v.ContactEmail, v.ContactPhone, v.Notes, v.Status, v.LeadTimeDays, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	username := h.GetUsername(r)
	h.LogAudit(username, "updated", "vendor", id, "Updated vendor "+v.Name)
	newSnap, _ := h.GetVendorSnapshot(id)
	h.RecordChangeJSON(username, "vendors", id, "update", oldSnap, newSnap)
	h.GetVendor(w, r, id)
}

// DeleteVendor deletes a vendor by ID.
func (h *Handler) DeleteVendor(w http.ResponseWriter, r *http.Request, id string) {
	// Check for referencing records
	var poCount int
	h.DB.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE vendor_id=?", id).Scan(&poCount)
	if poCount > 0 {
		response.Err(w, fmt.Sprintf("cannot delete vendor: %d purchase orders reference it", poCount), 409)
		return
	}
	var rfqCount int
	h.DB.QueryRow("SELECT COUNT(*) FROM rfq_vendors WHERE vendor_id=?", id).Scan(&rfqCount)
	if rfqCount > 0 {
		response.Err(w, fmt.Sprintf("cannot delete vendor: %d RFQs reference it", rfqCount), 409)
		return
	}

	// Snapshot for change_history
	oldSnap, _ := h.GetVendorSnapshot(id)

	username := h.GetUsername(r)
	undoID, _ := h.CreateUndoEntry(username, "delete", "vendor", id)
	_, err := h.DB.Exec("DELETE FROM vendors WHERE id=?", id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	h.LogAudit(username, "deleted", "vendor", id, "Deleted vendor "+id)
	changeID, _ := h.RecordChangeJSON(username, "vendors", id, "delete", oldSnap, nil)
	resp := map[string]interface{}{"deleted": id}
	if undoID > 0 {
		resp["undo_id"] = undoID
	}
	if changeID > 0 {
		resp["change_id"] = changeID
	}
	response.JSON(w, resp)
}
