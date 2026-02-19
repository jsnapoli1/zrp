package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type BulkRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

type BulkResponse struct {
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}

func handleBulkECOs(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400); return
	}
	allowed := map[string]bool{"approve": true, "implement": true, "reject": true, "delete": true}
	if !allowed[req.Action] {
		jsonErr(w, "invalid action: "+req.Action, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := getUsername(r)
	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM ecos WHERE id=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue
		}
		var err error
		switch req.Action {
		case "approve":
			_, err = db.Exec("UPDATE ecos SET status='approved',approved_at=?,approved_by=?,updated_at=? WHERE id=?", now, user, now, id)
		case "implement":
			_, err = db.Exec("UPDATE ecos SET status='implemented',updated_at=? WHERE id=?", now, id)
		case "reject":
			_, err = db.Exec("UPDATE ecos SET status='rejected',updated_at=? WHERE id=?", now, id)
		case "delete":
			createUndoEntry(user, "delete", "eco", id)
			_, err = db.Exec("DELETE FROM ecos WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_"+req.Action, "eco", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	jsonResp(w, resp)
}

func handleBulkWorkOrders(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400); return
	}
	allowed := map[string]bool{"complete": true, "cancel": true, "delete": true}
	if !allowed[req.Action] {
		jsonErr(w, "invalid action: "+req.Action, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := getUsername(r)
	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM work_orders WHERE id=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue
		}
		var err error
		switch req.Action {
		case "complete":
			_, err = db.Exec("UPDATE work_orders SET status='completed',completed_at=? WHERE id=?", now, id)
		case "cancel":
			_, err = db.Exec("UPDATE work_orders SET status='cancelled' WHERE id=?", id)
		case "delete":
			createUndoEntry(user, "delete", "workorder", id)
			_, err = db.Exec("DELETE FROM work_orders WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_"+req.Action, "workorder", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	jsonResp(w, resp)
}

func handleBulkNCRs(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400); return
	}
	allowed := map[string]bool{"close": true, "resolve": true, "delete": true}
	if !allowed[req.Action] {
		jsonErr(w, "invalid action: "+req.Action, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := getUsername(r)
	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM ncrs WHERE id=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue
		}
		var err error
		switch req.Action {
		case "close":
			_, err = db.Exec("UPDATE ncrs SET status='closed',resolved_at=? WHERE id=?", now, id)
		case "resolve":
			_, err = db.Exec("UPDATE ncrs SET status='resolved',resolved_at=? WHERE id=?", now, id)
		case "delete":
			createUndoEntry(user, "delete", "ncr", id)
			_, err = db.Exec("DELETE FROM ncrs WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_"+req.Action, "ncr", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	jsonResp(w, resp)
}

func handleBulkDevices(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400); return
	}
	allowed := map[string]bool{"decommission": true, "delete": true}
	if !allowed[req.Action] {
		jsonErr(w, "invalid action: "+req.Action, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	user := getUsername(r)
	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM devices WHERE serial_number=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue
		}
		var err error
		switch req.Action {
		case "decommission":
			_, err = db.Exec("UPDATE devices SET status='decommissioned' WHERE serial_number=?", id)
		case "delete":
			createUndoEntry(user, "delete", "device", id)
			_, err = db.Exec("DELETE FROM devices WHERE serial_number=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_"+req.Action, "device", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	jsonResp(w, resp)
}

func handleBulkInventory(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400); return
	}
	if req.Action != "delete" {
		jsonErr(w, "invalid action: "+req.Action, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	user := getUsername(r)
	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM inventory WHERE ipn=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue
		}
		createUndoEntry(user, "delete", "inventory", id)
		_, err := db.Exec("DELETE FROM inventory WHERE ipn=?", id)
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_delete", "inventory", id, "Bulk delete: "+id)
		}
	}
	jsonResp(w, resp)
}

func handleBulkRMAs(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400); return
	}
	allowed := map[string]bool{"close": true, "delete": true}
	if !allowed[req.Action] {
		jsonErr(w, "invalid action: "+req.Action, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := getUsername(r)
	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM rmas WHERE id=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue
		}
		var err error
		switch req.Action {
		case "close":
			_, err = db.Exec("UPDATE rmas SET status='closed',resolved_at=? WHERE id=?", now, id)
		case "delete":
			createUndoEntry(user, "delete", "rma", id)
			_, err = db.Exec("DELETE FROM rmas WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_"+req.Action, "rma", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	jsonResp(w, resp)
}

func handleBulkParts(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400); return
	}
	allowed := map[string]bool{"delete": true, "archive": true}
	if !allowed[req.Action] {
		jsonErr(w, "invalid action: "+req.Action, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	user := getUsername(r)
	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM parts WHERE ipn=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue
		}
		var err error
		switch req.Action {
		case "archive":
			_, err = db.Exec("UPDATE parts SET status='archived' WHERE ipn=?", id)
		case "delete":
			createUndoEntry(user, "delete", "part", id)
			_, err = db.Exec("DELETE FROM parts WHERE ipn=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_"+req.Action, "part", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	jsonResp(w, resp)
}

func handleBulkPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	var req BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", 400); return
	}
	allowed := map[string]bool{"approve": true, "cancel": true, "delete": true}
	if !allowed[req.Action] {
		jsonErr(w, "invalid action: "+req.Action, 400); return
	}
	resp := BulkResponse{Errors: []string{}}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := getUsername(r)
	for _, id := range req.IDs {
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE id=?", id).Scan(&exists)
		if exists == 0 {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": not found"); continue
		}
		var err error
		switch req.Action {
		case "approve":
			_, err = db.Exec("UPDATE purchase_orders SET status='approved',approved_at=?,approved_by=? WHERE id=?", now, user, id)
		case "cancel":
			_, err = db.Exec("UPDATE purchase_orders SET status='cancelled' WHERE id=?", id)
		case "delete":
			createUndoEntry(user, "delete", "po", id)
			_, err = db.Exec("DELETE FROM purchase_orders WHERE id=?", id)
		}
		if err != nil {
			resp.Failed++; resp.Errors = append(resp.Errors, id+": "+err.Error())
		} else {
			resp.Success++
			logAudit(db, user, "bulk_"+req.Action, "po", id, fmt.Sprintf("Bulk %s: %s", req.Action, id))
		}
	}
	jsonResp(w, resp)
}

// suppress unused import warnings
var _ = strings.TrimSpace
