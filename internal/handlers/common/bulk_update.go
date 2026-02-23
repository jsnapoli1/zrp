package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BulkUpdateRequest is the request body for bulk-update endpoints.
type BulkUpdateRequest struct {
	IDs     []string          `json:"ids"`
	Updates map[string]string `json:"updates"`
}

var allowedInventoryUpdateFields = map[string]bool{
	"location": true, "reorder_point": true, "reorder_qty": true,
}

var allowedWorkOrderUpdateFields = map[string]bool{
	"status": true, "priority": true, "due_date": true,
}

var allowedDeviceUpdateFields = map[string]bool{
	"status": true, "customer": true, "location": true,
}

var allowedPartUpdateFields = map[string]bool{
	"category": true, "status": true, "lifecycle": true, "min_stock": true,
}

var allowedECOUpdateFields = map[string]bool{
	"status": true, "priority": true,
}

// BulkUpdateInventory handles bulk inventory field updates.
func (h *Handler) BulkUpdateInventory(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	if len(req.IDs) == 0 { http.Error(w, `{"error":"ids required"}`, 400); return }
	if len(req.Updates) == 0 { http.Error(w, `{"error":"updates required"}`, 400); return }
	for field := range req.Updates {
		if !allowedInventoryUpdateFields[field] {
			http.Error(w, `{"error":"field not allowed for bulk update: `+field+`"}`, 400); return
		}
	}

	resp := BulkResponse{Errors: []string{}}
	user := h.GetUsername(r)
	now := time.Now().Format("2006-01-02 15:04:05")

	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM inventory WHERE ipn=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		setClauses := "updated_at=?"
		args := []interface{}{now}
		for field, value := range req.Updates {
			setClauses += ", " + field + "=?"
			args = append(args, value)
		}
		args = append(args, id)
		_, err := h.DB.Exec("UPDATE inventory SET "+setClauses+" WHERE ipn=?", args...)
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_update", "inventory", id, fmt.Sprintf("Bulk update fields: %v", req.Updates))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkUpdateWorkOrders handles bulk work order field updates.
func (h *Handler) BulkUpdateWorkOrders(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	if len(req.IDs) == 0 { http.Error(w, `{"error":"ids required"}`, 400); return }
	if len(req.Updates) == 0 { http.Error(w, `{"error":"updates required"}`, 400); return }
	for field := range req.Updates {
		if !allowedWorkOrderUpdateFields[field] {
			http.Error(w, `{"error":"field not allowed for bulk update: `+field+`"}`, 400); return
		}
	}
	if s, ok := req.Updates["status"]; ok {
		valid := map[string]bool{"draft": true, "open": true, "in_progress": true, "completed": true, "cancelled": true, "on_hold": true}
		if !valid[s] { http.Error(w, `{"error":"invalid status: `+s+`"}`, 400); return }
	}
	if p, ok := req.Updates["priority"]; ok {
		valid := map[string]bool{"low": true, "normal": true, "high": true, "critical": true}
		if !valid[p] { http.Error(w, `{"error":"invalid priority: `+p+`"}`, 400); return }
	}

	resp := BulkResponse{Errors: []string{}}
	user := h.GetUsername(r)

	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM work_orders WHERE id=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		setClauses := ""
		args := []interface{}{}
		for field, value := range req.Updates {
			if setClauses != "" { setClauses += ", " }
			setClauses += field + "=?"
			args = append(args, value)
		}
		args = append(args, id)
		_, err := h.DB.Exec("UPDATE work_orders SET "+setClauses+" WHERE id=?", args...)
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_update", "workorder", id, fmt.Sprintf("Bulk update fields: %v", req.Updates))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkUpdateDevices handles bulk device field updates.
func (h *Handler) BulkUpdateDevices(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	if len(req.IDs) == 0 { http.Error(w, `{"error":"ids required"}`, 400); return }
	if len(req.Updates) == 0 { http.Error(w, `{"error":"updates required"}`, 400); return }
	for field := range req.Updates {
		if !allowedDeviceUpdateFields[field] {
			http.Error(w, `{"error":"field not allowed for bulk update: `+field+`"}`, 400); return
		}
	}
	if s, ok := req.Updates["status"]; ok {
		valid := map[string]bool{"active": true, "inactive": true, "decommissioned": true, "rma": true}
		if !valid[s] { http.Error(w, `{"error":"invalid status: `+s+`"}`, 400); return }
	}

	resp := BulkResponse{Errors: []string{}}
	user := h.GetUsername(r)

	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM devices WHERE serial_number=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		setClauses := ""
		args := []interface{}{}
		for field, value := range req.Updates {
			if setClauses != "" { setClauses += ", " }
			setClauses += field + "=?"
			args = append(args, value)
		}
		args = append(args, id)
		_, err := h.DB.Exec("UPDATE devices SET "+setClauses+" WHERE serial_number=?", args...)
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_update", "device", id, fmt.Sprintf("Bulk update fields: %v", req.Updates))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkUpdateParts handles bulk part field updates.
func (h *Handler) BulkUpdateParts(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	if len(req.IDs) == 0 { http.Error(w, `{"error":"ids required"}`, 400); return }
	if len(req.Updates) == 0 { http.Error(w, `{"error":"updates required"}`, 400); return }
	for field := range req.Updates {
		if !allowedPartUpdateFields[field] {
			http.Error(w, `{"error":"field not allowed for bulk update: `+field+`"}`, 400); return
		}
	}

	resp := BulkResponse{Errors: []string{}}
	user := h.GetUsername(r)
	now := time.Now().Format("2006-01-02 15:04:05")

	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM parts WHERE ipn=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		setClauses := "updated_at=?"
		args := []interface{}{now}
		for field, value := range req.Updates {
			setClauses += ", " + field + "=?"
			args = append(args, value)
		}
		args = append(args, id)
		_, err := h.DB.Exec("UPDATE parts SET "+setClauses+" WHERE ipn=?", args...)
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_update", "part", id, fmt.Sprintf("Bulk update fields: %v", req.Updates))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkUpdateECOs handles bulk ECO field updates.
func (h *Handler) BulkUpdateECOs(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	if len(req.IDs) == 0 { http.Error(w, `{"error":"ids required"}`, 400); return }
	if len(req.Updates) == 0 { http.Error(w, `{"error":"updates required"}`, 400); return }
	for field := range req.Updates {
		if !allowedECOUpdateFields[field] {
			http.Error(w, `{"error":"field not allowed for bulk update: `+field+`"}`, 400); return
		}
	}
	if s, ok := req.Updates["status"]; ok {
		valid := map[string]bool{"draft": true, "open": true, "approved": true, "implemented": true, "rejected": true}
		if !valid[s] { http.Error(w, `{"error":"invalid status: `+s+`"}`, 400); return }
	}

	resp := BulkResponse{Errors: []string{}}
	user := h.GetUsername(r)
	now := time.Now().Format("2006-01-02 15:04:05")

	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM ecos WHERE id=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		setClauses := "updated_at=?"
		args := []interface{}{now}
		for field, value := range req.Updates {
			setClauses += ", " + field + "=?"
			args = append(args, value)
		}
		args = append(args, id)
		_, err := h.DB.Exec("UPDATE ecos SET "+setClauses+" WHERE id=?", args...)
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_update", "eco", id, fmt.Sprintf("Bulk update fields: %v", req.Updates))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
