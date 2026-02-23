package field

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

// ListCampaigns handles GET /api/firmware/campaigns.
func (h *Handler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id,name,version,category,status,COALESCE(target_filter,''),COALESCE(notes,''),created_at,started_at,completed_at FROM firmware_campaigns ORDER BY created_at DESC")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.FirmwareCampaign
	for rows.Next() {
		var f models.FirmwareCampaign
		var sa, ca sql.NullString
		rows.Scan(&f.ID, &f.Name, &f.Version, &f.Category, &f.Status, &f.TargetFilter, &f.Notes, &f.CreatedAt, &sa, &ca)
		f.StartedAt = database.SP(sa)
		f.CompletedAt = database.SP(ca)
		items = append(items, f)
	}
	if items == nil {
		items = []models.FirmwareCampaign{}
	}
	response.JSON(w, items)
}

// GetCampaign handles GET /api/firmware/campaigns/:id.
func (h *Handler) GetCampaign(w http.ResponseWriter, r *http.Request, id string) {
	var f models.FirmwareCampaign
	var sa, ca sql.NullString
	err := h.DB.QueryRow("SELECT id,name,version,category,status,COALESCE(target_filter,''),COALESCE(notes,''),created_at,started_at,completed_at FROM firmware_campaigns WHERE id=?", id).
		Scan(&f.ID, &f.Name, &f.Version, &f.Category, &f.Status, &f.TargetFilter, &f.Notes, &f.CreatedAt, &sa, &ca)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	f.StartedAt = database.SP(sa)
	f.CompletedAt = database.SP(ca)
	response.JSON(w, f)
}

// CreateCampaign handles POST /api/firmware/campaigns.
func (h *Handler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	var f models.FirmwareCampaign
	if err := response.DecodeBody(r, &f); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "name", f.Name)
	validation.RequireField(ve, "version", f.Version)
	if f.Status != "" {
		validation.ValidateEnum(ve, "status", f.Status, validation.ValidCampaignStatuses)
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	f.ID = h.NextIDFunc("FW", "firmware_campaigns", 3)
	if f.Status == "" {
		f.Status = "draft"
	}
	if f.Category == "" {
		f.Category = "public"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := h.DB.Exec("INSERT INTO firmware_campaigns (id,name,version,category,status,target_filter,notes,created_at) VALUES (?,?,?,?,?,?,?,?)",
		f.ID, f.Name, f.Version, f.Category, f.Status, f.TargetFilter, f.Notes, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	f.CreatedAt = now
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "created", "firmware", f.ID, "Created campaign "+f.ID+": "+f.Name)
	response.JSON(w, f)
}

// UpdateCampaign handles PUT /api/firmware/campaigns/:id.
func (h *Handler) UpdateCampaign(w http.ResponseWriter, r *http.Request, id string) {
	var f models.FirmwareCampaign
	if err := response.DecodeBody(r, &f); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	_, err := h.DB.Exec("UPDATE firmware_campaigns SET name=?,version=?,category=?,status=?,target_filter=?,notes=? WHERE id=?",
		f.Name, f.Version, f.Category, f.Status, f.TargetFilter, f.Notes, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "updated", "firmware", id, "Updated campaign "+id)
	h.GetCampaign(w, r, id)
}

// LaunchCampaign handles POST /api/firmware/campaigns/:id/launch.
func (h *Handler) LaunchCampaign(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	// Get all active devices and add them to campaign
	rows, err := h.DB.Query("SELECT serial_number FROM devices WHERE status='active'")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Collect all serial numbers first
	var serialNumbers []string
	for rows.Next() {
		var sn string
		rows.Scan(&sn)
		serialNumbers = append(serialNumbers, sn)
	}
	rows.Close()

	// Now insert them into campaign_devices
	count := 0
	for _, sn := range serialNumbers {
		_, insertErr := h.DB.Exec("INSERT OR IGNORE INTO campaign_devices (campaign_id,serial_number,status) VALUES (?,?,?)", id, sn, "pending")
		if insertErr != nil {
			fmt.Printf("Error inserting device %s into campaign: %v\n", sn, insertErr)
		} else {
			count++
		}
	}

	h.DB.Exec("UPDATE firmware_campaigns SET status='active',started_at=? WHERE id=?", now, id)
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "launched", "firmware", id, fmt.Sprintf("Launched campaign %s to %d devices", id, count))
	response.JSON(w, map[string]interface{}{"launched": true, "devices_added": count})
}

// CampaignProgress handles GET /api/firmware/campaigns/:id/progress.
func (h *Handler) CampaignProgress(w http.ResponseWriter, r *http.Request, id string) {
	var pending, sent, updated, failed int
	h.DB.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='pending'", id).Scan(&pending)
	h.DB.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='sent'", id).Scan(&sent)
	h.DB.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='updated'", id).Scan(&updated)
	h.DB.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='failed'", id).Scan(&failed)
	total := pending + sent + updated + failed
	response.JSON(w, map[string]int{"total": total, "pending": pending, "sent": sent, "updated": updated, "failed": failed})
}

// CampaignStream handles GET /api/firmware/campaigns/:id/stream (SSE).
func (h *Handler) CampaignStream(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	for {
		var pending, sent, updated, failed int
		h.DB.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='pending'", id).Scan(&pending)
		h.DB.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='sent'", id).Scan(&sent)
		h.DB.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='updated'", id).Scan(&updated)
		h.DB.QueryRow("SELECT COUNT(*) FROM campaign_devices WHERE campaign_id=? AND status='failed'", id).Scan(&failed)
		total := pending + sent + updated + failed
		pct := 0
		if total > 0 {
			pct = (updated + failed) * 100 / total
		}
		fmt.Fprintf(w, "data: {\"pending\":%d,\"sent\":%d,\"updated\":%d,\"failed\":%d,\"total\":%d,\"pct\":%d}\n\n", pending, sent, updated, failed, total, pct)
		flusher.Flush()
		if total > 0 && (updated+failed) >= total {
			break
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

// MarkCampaignDevice handles PUT /api/firmware/campaigns/:id/devices/:serial.
func (h *Handler) MarkCampaignDevice(w http.ResponseWriter, r *http.Request, campaignID, serial string) {
	var body struct {
		Status string `json:"status"`
	}
	if err := response.DecodeBody(r, &body); err != nil || (body.Status != "updated" && body.Status != "failed") {
		response.Err(w, "status must be 'updated' or 'failed'", 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := h.DB.Exec("UPDATE campaign_devices SET status=?,updated_at=? WHERE campaign_id=? AND serial_number=?", body.Status, now, campaignID, serial)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		response.Err(w, "device not found in campaign", 404)
		return
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "marked_"+body.Status, "firmware", campaignID, fmt.Sprintf("Marked %s as %s in campaign %s", serial, body.Status, campaignID))
	response.JSON(w, map[string]string{"status": "ok"})
}

// CampaignDevices handles GET /api/firmware/campaigns/:id/devices.
func (h *Handler) CampaignDevices(w http.ResponseWriter, r *http.Request, id string) {
	rows, err := h.DB.Query("SELECT campaign_id,serial_number,status,updated_at FROM campaign_devices WHERE campaign_id=?", id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.CampaignDevice
	for rows.Next() {
		var cd models.CampaignDevice
		var ua sql.NullString
		rows.Scan(&cd.CampaignID, &cd.SerialNumber, &cd.Status, &ua)
		cd.UpdatedAt = database.SP(ua)
		items = append(items, cd)
	}
	if items == nil {
		items = []models.CampaignDevice{}
	}
	response.JSON(w, items)
}
