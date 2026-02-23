package field

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"zrp/internal/audit"
	"zrp/internal/database"
	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
)

// ListFieldReports handles GET /api/field-reports.
func (h *Handler) ListFieldReports(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id,title,COALESCE(report_type,''),status,COALESCE(priority,''),
		COALESCE(customer_name,''),COALESCE(site_location,''),COALESCE(device_ipn,''),
		COALESCE(device_serial,''),COALESCE(reported_by,''),COALESCE(reported_at,''),
		COALESCE(description,''),COALESCE(root_cause,''),COALESCE(resolution,''),
		resolved_at,COALESCE(ncr_id,''),COALESCE(eco_id,''),created_at,updated_at
		FROM field_reports WHERE 1=1`
	var args []interface{}

	if v := r.URL.Query().Get("status"); v != "" {
		query += " AND status=?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("priority"); v != "" {
		query += " AND priority=?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("report_type"); v != "" {
		query += " AND report_type=?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("from"); v != "" {
		query += " AND created_at >= ?"
		args = append(args, v)
	}
	if v := r.URL.Query().Get("to"); v != "" {
		query += " AND created_at <= ?"
		args = append(args, v)
	}

	query += " ORDER BY created_at DESC"
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var items []models.FieldReport
	for rows.Next() {
		var fr models.FieldReport
		var ra sql.NullString
		rows.Scan(&fr.ID, &fr.Title, &fr.ReportType, &fr.Status, &fr.Priority,
			&fr.CustomerName, &fr.SiteLocation, &fr.DeviceIPN, &fr.DeviceSerial,
			&fr.ReportedBy, &fr.ReportedAt, &fr.Description, &fr.RootCause,
			&fr.Resolution, &ra, &fr.NcrID, &fr.EcoID, &fr.CreatedAt, &fr.UpdatedAt)
		fr.ResolvedAt = database.SP(ra)
		items = append(items, fr)
	}
	if items == nil {
		items = []models.FieldReport{}
	}
	response.JSON(w, items)
}

// GetFieldReport handles GET /api/field-reports/:id.
func (h *Handler) GetFieldReport(w http.ResponseWriter, r *http.Request, id string) {
	var fr models.FieldReport
	var ra sql.NullString
	err := h.DB.QueryRow(`SELECT id,title,COALESCE(report_type,''),status,COALESCE(priority,''),
		COALESCE(customer_name,''),COALESCE(site_location,''),COALESCE(device_ipn,''),
		COALESCE(device_serial,''),COALESCE(reported_by,''),COALESCE(reported_at,''),
		COALESCE(description,''),COALESCE(root_cause,''),COALESCE(resolution,''),
		resolved_at,COALESCE(ncr_id,''),COALESCE(eco_id,''),created_at,updated_at
		FROM field_reports WHERE id=?`, id).
		Scan(&fr.ID, &fr.Title, &fr.ReportType, &fr.Status, &fr.Priority,
			&fr.CustomerName, &fr.SiteLocation, &fr.DeviceIPN, &fr.DeviceSerial,
			&fr.ReportedBy, &fr.ReportedAt, &fr.Description, &fr.RootCause,
			&fr.Resolution, &ra, &fr.NcrID, &fr.EcoID, &fr.CreatedAt, &fr.UpdatedAt)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	fr.ResolvedAt = database.SP(ra)
	response.JSON(w, fr)
}

// CreateFieldReport handles POST /api/field-reports.
func (h *Handler) CreateFieldReport(w http.ResponseWriter, r *http.Request) {
	var fr models.FieldReport
	if err := response.DecodeBody(r, &fr); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "title", fr.Title)
	validation.ValidateMaxLength(ve, "title", fr.Title, 255)
	validation.ValidateMaxLength(ve, "description", fr.Description, 1000)
	validation.ValidateMaxLength(ve, "customer_name", fr.CustomerName, 255)
	validation.ValidateMaxLength(ve, "site_location", fr.SiteLocation, 255)
	validation.ValidateMaxLength(ve, "device_ipn", fr.DeviceIPN, 100)
	validation.ValidateMaxLength(ve, "device_serial", fr.DeviceSerial, 100)
	validation.ValidateMaxLength(ve, "root_cause", fr.RootCause, 1000)
	validation.ValidateMaxLength(ve, "resolution", fr.Resolution, 1000)
	if fr.ReportType != "" {
		validation.ValidateEnum(ve, "report_type", fr.ReportType, validation.ValidFieldReportTypes)
	}
	if fr.Status != "" {
		validation.ValidateEnum(ve, "status", fr.Status, validation.ValidFieldReportStatuses)
	}
	if fr.Priority != "" {
		validation.ValidateEnum(ve, "priority", fr.Priority, validation.ValidFieldReportPriorities)
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	fr.ID = h.NextIDFunc("FR", "field_reports", 3)
	if fr.Status == "" {
		fr.Status = "open"
	}
	if fr.Priority == "" {
		fr.Priority = "medium"
	}
	if fr.ReportType == "" {
		fr.ReportType = "failure"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	fr.CreatedAt = now
	fr.UpdatedAt = now
	if fr.ReportedAt == "" {
		fr.ReportedAt = now
	}

	_, err := h.DB.Exec(`INSERT INTO field_reports (id,title,report_type,status,priority,customer_name,
		site_location,device_ipn,device_serial,reported_by,reported_at,description,
		root_cause,resolution,ncr_id,eco_id,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		fr.ID, fr.Title, fr.ReportType, fr.Status, fr.Priority, fr.CustomerName,
		fr.SiteLocation, fr.DeviceIPN, fr.DeviceSerial, fr.ReportedBy, fr.ReportedAt,
		fr.Description, fr.RootCause, fr.Resolution, fr.NcrID, fr.EcoID, fr.CreatedAt, fr.UpdatedAt)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "created", "field_report", fr.ID, "Created "+fr.ID+": "+fr.Title)
	response.JSON(w, fr)
}

// UpdateFieldReport handles PUT /api/field-reports/:id.
func (h *Handler) UpdateFieldReport(w http.ResponseWriter, r *http.Request, id string) {
	// Check exists
	var existing models.FieldReport
	err := h.DB.QueryRow("SELECT id FROM field_reports WHERE id=?", id).Scan(&existing.ID)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	var body map[string]interface{}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	getString := func(key string) string {
		if v, ok := body[key]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	// Validate string lengths
	ve := &validation.ValidationErrors{}
	validation.ValidateMaxLength(ve, "title", getString("title"), 255)
	validation.ValidateMaxLength(ve, "description", getString("description"), 1000)
	validation.ValidateMaxLength(ve, "customer_name", getString("customer_name"), 255)
	validation.ValidateMaxLength(ve, "site_location", getString("site_location"), 255)
	validation.ValidateMaxLength(ve, "device_ipn", getString("device_ipn"), 100)
	validation.ValidateMaxLength(ve, "device_serial", getString("device_serial"), 100)
	validation.ValidateMaxLength(ve, "root_cause", getString("root_cause"), 1000)
	validation.ValidateMaxLength(ve, "resolution", getString("resolution"), 1000)
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	sets := []string{}
	args := []interface{}{}
	fields := []string{"title", "report_type", "status", "priority", "customer_name",
		"site_location", "device_ipn", "device_serial", "reported_by",
		"description", "root_cause", "resolution", "ncr_id", "eco_id"}
	for _, f := range fields {
		if _, ok := body[f]; ok {
			sets = append(sets, f+"=?")
			args = append(args, getString(f))
		}
	}

	now := time.Now().Format("2006-01-02 15:04:05")

	// Auto-set resolved_at when status becomes resolved
	if status := getString("status"); status == "resolved" {
		sets = append(sets, "resolved_at=?")
		args = append(args, now)
	}

	sets = append(sets, "updated_at=?")
	args = append(args, now)
	args = append(args, id)

	if len(sets) > 0 {
		_, err = h.DB.Exec("UPDATE field_reports SET "+strings.Join(sets, ",")+" WHERE id=?", args...)
		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}
	}

	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "updated", "field_report", id, "Updated "+id)
	h.GetFieldReport(w, r, id)
}

// DeleteFieldReport handles DELETE /api/field-reports/:id.
func (h *Handler) DeleteFieldReport(w http.ResponseWriter, r *http.Request, id string) {
	res, err := h.DB.Exec("DELETE FROM field_reports WHERE id=?", id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		response.Err(w, "not found", 404)
		return
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "deleted", "field_report", id, "Deleted "+id)
	response.JSON(w, map[string]string{"status": "ok"})
}

// FieldReportCreateNCR handles POST /api/field-reports/:id/ncr.
func (h *Handler) FieldReportCreateNCR(w http.ResponseWriter, r *http.Request, id string) {
	var fr models.FieldReport
	var ra sql.NullString
	err := h.DB.QueryRow(`SELECT id,title,COALESCE(report_type,''),status,COALESCE(priority,''),
		COALESCE(customer_name,''),COALESCE(site_location,''),COALESCE(device_ipn,''),
		COALESCE(device_serial,''),COALESCE(reported_by,''),COALESCE(reported_at,''),
		COALESCE(description,''),COALESCE(root_cause,''),COALESCE(resolution,''),
		resolved_at,COALESCE(ncr_id,''),COALESCE(eco_id,''),created_at,updated_at
		FROM field_reports WHERE id=?`, id).
		Scan(&fr.ID, &fr.Title, &fr.ReportType, &fr.Status, &fr.Priority,
			&fr.CustomerName, &fr.SiteLocation, &fr.DeviceIPN, &fr.DeviceSerial,
			&fr.ReportedBy, &fr.ReportedAt, &fr.Description, &fr.RootCause,
			&fr.Resolution, &ra, &fr.NcrID, &fr.EcoID, &fr.CreatedAt, &fr.UpdatedAt)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	// Create NCR from field report data
	ncrID := h.NextIDFunc("NCR", "ncrs", 3)
	now := time.Now().Format("2006-01-02 15:04:05")
	severity := "minor"
	if fr.Priority == "critical" {
		severity = "critical"
	} else if fr.Priority == "high" {
		severity = "major"
	}

	_, err = h.DB.Exec(`INSERT INTO ncrs (id,title,description,ipn,serial_number,defect_type,severity,status,created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		ncrID, fr.Title, fr.Description, fr.DeviceIPN, fr.DeviceSerial, "field_report", severity, "open", now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Link NCR back to field report
	h.DB.Exec("UPDATE field_reports SET ncr_id=?, updated_at=? WHERE id=?", ncrID, now, id)

	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "created", "ncr", ncrID, fmt.Sprintf("Created %s from field report %s", ncrID, id))

	response.JSON(w, map[string]string{"id": ncrID, "title": fr.Title, "status": "open", "severity": severity})
}
