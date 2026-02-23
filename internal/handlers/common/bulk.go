package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BulkRequest is the request body for bulk action endpoints.
type BulkRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

// BulkResponse is the response for bulk action endpoints.
type BulkResponse struct {
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}

// BulkECOs handles bulk ECO actions.
func (h *Handler) BulkECOs(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	allowed := map[string]bool{"approve": true, "implement": true, "reject": true, "delete": true}
	if !allowed[req.Action] {
		http.Error(w, `{"error":"invalid action: `+req.Action+`"}`, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := h.GetUsername(r)
	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM ecos WHERE id=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		var err error
		switch req.Action {
		case "approve":
			_, err = h.DB.Exec("UPDATE ecos SET status='approved',approved_at=?,approved_by=?,updated_at=? WHERE id=?", now, user, now, id)
		case "implement":
			_, err = h.DB.Exec("UPDATE ecos SET status='implemented',updated_at=? WHERE id=?", now, id)
		case "reject":
			_, err = h.DB.Exec("UPDATE ecos SET status='rejected',updated_at=? WHERE id=?", now, id)
		case "delete":
			h.CreateUndoEntry(user, "delete", "eco", id)
			_, err = h.DB.Exec("DELETE FROM ecos WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_"+req.Action, "eco", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkWorkOrders handles bulk work order actions.
func (h *Handler) BulkWorkOrders(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	allowed := map[string]bool{"complete": true, "cancel": true, "delete": true}
	if !allowed[req.Action] {
		http.Error(w, `{"error":"invalid action: `+req.Action+`"}`, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := h.GetUsername(r)
	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM work_orders WHERE id=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		var err error
		switch req.Action {
		case "complete":
			_, err = h.DB.Exec("UPDATE work_orders SET status='completed',completed_at=? WHERE id=?", now, id)
		case "cancel":
			_, err = h.DB.Exec("UPDATE work_orders SET status='cancelled' WHERE id=?", id)
		case "delete":
			h.CreateUndoEntry(user, "delete", "workorder", id)
			_, err = h.DB.Exec("DELETE FROM work_orders WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_"+req.Action, "workorder", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkNCRs handles bulk NCR actions.
func (h *Handler) BulkNCRs(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	allowed := map[string]bool{"close": true, "resolve": true, "delete": true}
	if !allowed[req.Action] {
		http.Error(w, `{"error":"invalid action: `+req.Action+`"}`, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := h.GetUsername(r)
	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM ncrs WHERE id=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		var err error
		switch req.Action {
		case "close":
			_, err = h.DB.Exec("UPDATE ncrs SET status='closed',resolved_at=? WHERE id=?", now, id)
		case "resolve":
			_, err = h.DB.Exec("UPDATE ncrs SET status='resolved',resolved_at=? WHERE id=?", now, id)
		case "delete":
			h.CreateUndoEntry(user, "delete", "ncr", id)
			_, err = h.DB.Exec("DELETE FROM ncrs WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_"+req.Action, "ncr", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkDevices handles bulk device actions.
func (h *Handler) BulkDevices(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	allowed := map[string]bool{"decommission": true, "delete": true}
	if !allowed[req.Action] {
		http.Error(w, `{"error":"invalid action: `+req.Action+`"}`, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	user := h.GetUsername(r)
	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM devices WHERE serial_number=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		var err error
		switch req.Action {
		case "decommission":
			_, err = h.DB.Exec("UPDATE devices SET status='decommissioned' WHERE serial_number=?", id)
		case "delete":
			h.CreateUndoEntry(user, "delete", "device", id)
			_, err = h.DB.Exec("DELETE FROM devices WHERE serial_number=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_"+req.Action, "device", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkInventory handles bulk inventory delete.
func (h *Handler) BulkInventory(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	if req.Action != "delete" {
		http.Error(w, `{"error":"invalid action: `+req.Action+`"}`, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	user := h.GetUsername(r)
	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM inventory WHERE ipn=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		h.CreateUndoEntry(user, "delete", "inventory", id)
		_, err := h.DB.Exec("DELETE FROM inventory WHERE ipn=?", id)
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_delete", "inventory", id, "Bulk delete: "+id)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkRMAs handles bulk RMA actions.
func (h *Handler) BulkRMAs(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	allowed := map[string]bool{"close": true, "delete": true}
	if !allowed[req.Action] {
		http.Error(w, `{"error":"invalid action: `+req.Action+`"}`, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := h.GetUsername(r)
	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM rmas WHERE id=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		var err error
		switch req.Action {
		case "close":
			_, err = h.DB.Exec("UPDATE rmas SET status='closed',resolved_at=? WHERE id=?", now, id)
		case "delete":
			h.CreateUndoEntry(user, "delete", "rma", id)
			_, err = h.DB.Exec("DELETE FROM rmas WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_"+req.Action, "rma", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkParts handles bulk part actions.
func (h *Handler) BulkParts(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	allowed := map[string]bool{"delete": true, "archive": true}
	if !allowed[req.Action] {
		http.Error(w, `{"error":"invalid action: `+req.Action+`"}`, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	user := h.GetUsername(r)
	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM parts WHERE ipn=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		var err error
		switch req.Action {
		case "archive":
			_, err = h.DB.Exec("UPDATE parts SET status='archived' WHERE ipn=?", id)
		case "delete":
			h.CreateUndoEntry(user, "delete", "part", id)
			_, err = h.DB.Exec("DELETE FROM parts WHERE ipn=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_"+req.Action, "part", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BulkPurchaseOrders handles bulk PO actions.
func (h *Handler) BulkPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, 400); return
	}
	allowed := map[string]bool{"approve": true, "cancel": true, "delete": true}
	if !allowed[req.Action] {
		http.Error(w, `{"error":"invalid action: `+req.Action+`"}`, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := h.GetUsername(r)
	for _, id := range req.IDs {
		var exists int
		h.DB.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE id=?", id).Scan(&exists)
		if exists == 0 { resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue }
		var err error
		switch req.Action {
		case "approve":
			_, err = h.DB.Exec("UPDATE purchase_orders SET status='approved',approved_at=?,approved_by=? WHERE id=?", now, user, id)
		case "cancel":
			_, err = h.DB.Exec("UPDATE purchase_orders SET status='cancelled' WHERE id=?", id)
		case "delete":
			h.CreateUndoEntry(user, "delete", "po", id)
			_, err = h.DB.Exec("DELETE FROM purchase_orders WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			h.LogAudit(h.DB, user, "bulk_"+req.Action, "po", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
