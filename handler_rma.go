package main

import (
	"database/sql"
	"net/http"
	"time"
)

func handleListRMAs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,serial_number,COALESCE(customer,''),COALESCE(reason,''),status,COALESCE(defect_description,''),COALESCE(resolution,''),created_at,received_at,resolved_at FROM rmas ORDER BY created_at DESC")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []RMA
	for rows.Next() {
		var rm RMA
		var ra, resa sql.NullString
		rows.Scan(&rm.ID, &rm.SerialNumber, &rm.Customer, &rm.Reason, &rm.Status, &rm.DefectDescription, &rm.Resolution, &rm.CreatedAt, &ra, &resa)
		rm.ReceivedAt = sp(ra); rm.ResolvedAt = sp(resa)
		items = append(items, rm)
	}
	if items == nil { items = []RMA{} }
	jsonResp(w, items)
}

func handleGetRMA(w http.ResponseWriter, r *http.Request, id string) {
	var rm RMA
	var ra, resa sql.NullString
	err := db.QueryRow("SELECT id,serial_number,COALESCE(customer,''),COALESCE(reason,''),status,COALESCE(defect_description,''),COALESCE(resolution,''),created_at,received_at,resolved_at FROM rmas WHERE id=?", id).
		Scan(&rm.ID, &rm.SerialNumber, &rm.Customer, &rm.Reason, &rm.Status, &rm.DefectDescription, &rm.Resolution, &rm.CreatedAt, &ra, &resa)
	if err != nil { jsonErr(w, "not found", 404); return }
	rm.ReceivedAt = sp(ra); rm.ResolvedAt = sp(resa)
	jsonResp(w, rm)
}

func handleCreateRMA(w http.ResponseWriter, r *http.Request) {
	var rm RMA
	if err := decodeBody(r, &rm); err != nil { jsonErr(w, "invalid body", 400); return }
	rm.ID = nextID("RMA", "rmas", 3)
	if rm.Status == "" { rm.Status = "open" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO rmas (id,serial_number,customer,reason,status,defect_description,created_at) VALUES (?,?,?,?,?,?,?)",
		rm.ID, rm.SerialNumber, rm.Customer, rm.Reason, rm.Status, rm.DefectDescription, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	rm.CreatedAt = now
	logAudit(db, getUsername(r), "created", "rma", rm.ID, "Created "+rm.ID+": "+rm.Reason)
	recordChangeJSON(getUsername(r), "rmas", rm.ID, "create", nil, rm)
	jsonResp(w, rm)
}

func handleUpdateRMA(w http.ResponseWriter, r *http.Request, id string) {
	oldSnap, _ := getRMASnapshot(id)
	var rm RMA
	if err := decodeBody(r, &rm); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	var receivedAt, resolvedAt interface{}
	if rm.Status == "received" { receivedAt = now }
	if rm.Status == "closed" || rm.Status == "shipped" { resolvedAt = now }
	_, err := db.Exec("UPDATE rmas SET serial_number=?,customer=?,reason=?,status=?,defect_description=?,resolution=?,received_at=COALESCE(?,received_at),resolved_at=COALESCE(?,resolved_at) WHERE id=?",
		rm.SerialNumber, rm.Customer, rm.Reason, rm.Status, rm.DefectDescription, rm.Resolution, receivedAt, resolvedAt, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "rma", id, "Updated "+id+": status="+rm.Status)
	newSnap, _ := getRMASnapshot(id)
	recordChangeJSON(getUsername(r), "rmas", id, "update", oldSnap, newSnap)
	handleGetRMA(w, r, id)
}
