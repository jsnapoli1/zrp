package quality

import (
	"net/http"
	"strconv"
	"time"

	"zrp/internal/audit"
	"zrp/internal/models"
	"zrp/internal/response"
)

// ListTests handles GET /api/v1/tests.
func (h *Handler) ListTests(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id,serial_number,ipn,COALESCE(firmware_version,''),COALESCE(test_type,''),result,COALESCE(measurements,''),COALESCE(notes,''),tested_by,tested_at FROM test_records ORDER BY tested_at DESC")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.TestRecord
	for rows.Next() {
		var t models.TestRecord
		rows.Scan(&t.ID, &t.SerialNumber, &t.IPN, &t.FirmwareVersion, &t.TestType, &t.Result, &t.Measurements, &t.Notes, &t.TestedBy, &t.TestedAt)
		items = append(items, t)
	}
	if items == nil {
		items = []models.TestRecord{}
	}
	response.JSON(w, items)
}

// GetTests handles GET /api/v1/tests/:serial.
func (h *Handler) GetTests(w http.ResponseWriter, r *http.Request, serial string) {
	rows, err := h.DB.Query("SELECT id,serial_number,ipn,COALESCE(firmware_version,''),COALESCE(test_type,''),result,COALESCE(measurements,''),COALESCE(notes,''),tested_by,tested_at FROM test_records WHERE serial_number=? ORDER BY tested_at DESC", serial)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.TestRecord
	for rows.Next() {
		var t models.TestRecord
		rows.Scan(&t.ID, &t.SerialNumber, &t.IPN, &t.FirmwareVersion, &t.TestType, &t.Result, &t.Measurements, &t.Notes, &t.TestedBy, &t.TestedAt)
		items = append(items, t)
	}
	if items == nil {
		items = []models.TestRecord{}
	}
	response.JSON(w, items)
}

// CreateTest handles POST /api/v1/tests.
func (h *Handler) CreateTest(w http.ResponseWriter, r *http.Request) {
	var t models.TestRecord
	if err := response.DecodeBody(r, &t); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	res, err := h.DB.Exec("INSERT INTO test_records (serial_number,ipn,firmware_version,test_type,result,measurements,notes,tested_by,tested_at) VALUES (?,?,?,?,?,?,?,?,?)",
		t.SerialNumber, t.IPN, t.FirmwareVersion, t.TestType, t.Result, t.Measurements, t.Notes, "operator", now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	id, _ := res.LastInsertId()
	t.ID = int(id)
	t.TestedAt = now
	t.TestedBy = "operator"
	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), "created", "test", t.SerialNumber, "Test "+t.Result+" for "+t.SerialNumber)
	response.JSON(w, t)
}

// GetTestByID handles GET /api/v1/tests/:id (numeric or serial fallback).
func (h *Handler) GetTestByID(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		// Not a numeric ID, fall back to serial number lookup
		h.GetTests(w, r, idStr)
		return
	}
	var t models.TestRecord
	err = h.DB.QueryRow("SELECT id,serial_number,ipn,COALESCE(firmware_version,''),COALESCE(test_type,''),result,COALESCE(measurements,''),COALESCE(notes,''),tested_by,tested_at FROM test_records WHERE id=?", id).
		Scan(&t.ID, &t.SerialNumber, &t.IPN, &t.FirmwareVersion, &t.TestType, &t.Result, &t.Measurements, &t.Notes, &t.TestedBy, &t.TestedAt)
	if err != nil {
		// Fall back to serial number lookup
		h.GetTests(w, r, idStr)
		return
	}
	response.JSON(w, t)
}
