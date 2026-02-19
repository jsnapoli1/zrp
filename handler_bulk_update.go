package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BulkUpdateRequest is the request body for bulk-update endpoints
type BulkUpdateRequest struct {
	IDs     []string          `json:"ids"`
	Updates map[string]string `json:"updates"`
}

// allowedBulkUpdateFields defines which fields can be bulk-updated per resource
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

func handleBulkUpdateInventory(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if len(req.IDs) == 0 {
		jsonErr(w, "ids required", 400)
		return
	}
	if len(req.Updates) == 0 {
		jsonErr(w, "updates required", 400)
		return
	}
	for field := range req.Updates {
		if !allowedInventoryUpdateFields[field] {
			jsonErr(w, "field not allowed for bulk update: "+field, 400)
			return
		}
	}

	resp := BulkResponse{Errors: []string{}}
	user := getUsername(r)
	now := time.Now().Format("2006-01-02 15:04:05")

	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM inventory WHERE ipn=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++
			resp.Errors = append(resp.Errors, id+": not found")
			continue
		}

		// Build dynamic update
		setClauses := "updated_at=?"
		args := []interface{}{now}
		for field, value := range req.Updates {
			setClauses += ", " + field + "=?"
			args = append(args, value)
		}
		args = append(args, id)
		_, err := db.Exec("UPDATE inventory SET "+setClauses+" WHERE ipn=?", args...)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_update", "inventory", id, fmt.Sprintf("Bulk update fields: %v", req.Updates))
		}
	}
	jsonResp(w, resp)
}

func handleBulkUpdateWorkOrders(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if len(req.IDs) == 0 {
		jsonErr(w, "ids required", 400)
		return
	}
	if len(req.Updates) == 0 {
		jsonErr(w, "updates required", 400)
		return
	}
	for field := range req.Updates {
		if !allowedWorkOrderUpdateFields[field] {
			jsonErr(w, "field not allowed for bulk update: "+field, 400)
			return
		}
	}

	// Validate status values
	if s, ok := req.Updates["status"]; ok {
		validStatuses := map[string]bool{"open": true, "in_progress": true, "completed": true, "cancelled": true}
		if !validStatuses[s] {
			jsonErr(w, "invalid status: "+s, 400)
			return
		}
	}
	if p, ok := req.Updates["priority"]; ok {
		validPriorities := map[string]bool{"low": true, "normal": true, "high": true, "urgent": true}
		if !validPriorities[p] {
			jsonErr(w, "invalid priority: "+p, 400)
			return
		}
	}

	resp := BulkResponse{Errors: []string{}}
	user := getUsername(r)

	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM work_orders WHERE id=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++
			resp.Errors = append(resp.Errors, id+": not found")
			continue
		}

		setClauses := ""
		args := []interface{}{}
		for field, value := range req.Updates {
			if setClauses != "" {
				setClauses += ", "
			}
			setClauses += field + "=?"
			args = append(args, value)
		}
		args = append(args, id)
		_, err := db.Exec("UPDATE work_orders SET "+setClauses+" WHERE id=?", args...)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_update", "workorder", id, fmt.Sprintf("Bulk update fields: %v", req.Updates))
		}
	}
	jsonResp(w, resp)
}

func handleBulkUpdateDevices(w http.ResponseWriter, r *http.Request) {
	var req BulkUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if len(req.IDs) == 0 {
		jsonErr(w, "ids required", 400)
		return
	}
	if len(req.Updates) == 0 {
		jsonErr(w, "updates required", 400)
		return
	}
	for field := range req.Updates {
		if !allowedDeviceUpdateFields[field] {
			jsonErr(w, "field not allowed for bulk update: "+field, 400)
			return
		}
	}

	if s, ok := req.Updates["status"]; ok {
		validStatuses := map[string]bool{"active": true, "inactive": true, "decommissioned": true, "rma": true}
		if !validStatuses[s] {
			jsonErr(w, "invalid status: "+s, 400)
			return
		}
	}

	resp := BulkResponse{Errors: []string{}}
	user := getUsername(r)

	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM devices WHERE serial_number=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++
			resp.Errors = append(resp.Errors, id+": not found")
			continue
		}

		setClauses := ""
		args := []interface{}{}
		for field, value := range req.Updates {
			if setClauses != "" {
				setClauses += ", "
			}
			setClauses += field + "=?"
			args = append(args, value)
		}
		args = append(args, id)
		_, err := db.Exec("UPDATE devices SET "+setClauses+" WHERE serial_number=?", args...)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_update", "device", id, fmt.Sprintf("Bulk update fields: %v", req.Updates))
		}
	}
	jsonResp(w, resp)
}
