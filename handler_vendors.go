package main

import (
	"fmt"
	"net/http"
)

func handleListVendors(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		jsonErr(w, "database not initialized", 503)
		return
	}
	rows, err := db.Query("SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),status,lead_time_days,created_at FROM vendors ORDER BY name")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []Vendor
	for rows.Next() {
		var v Vendor
		rows.Scan(&v.ID, &v.Name, &v.Website, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.LeadTimeDays, &v.CreatedAt)
		items = append(items, v)
	}
	if items == nil { items = []Vendor{} }
	jsonResp(w, items)
}

func handleGetVendor(w http.ResponseWriter, r *http.Request, id string) {
	if db == nil {
		jsonErr(w, "database not initialized", 503)
		return
	}
	var v Vendor
	err := db.QueryRow("SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),status,lead_time_days,created_at FROM vendors WHERE id=?", id).
		Scan(&v.ID, &v.Name, &v.Website, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.LeadTimeDays, &v.CreatedAt)
	if err != nil {
		jsonErr(w, "not found", 404)
		return
	}
	jsonResp(w, v)
}

func handleCreateVendor(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		jsonErr(w, "database not initialized", 503)
		return
	}
	var v Vendor
	if err := decodeBody(r, &v); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	requireField(ve, "name", v.Name)
	validateMaxLength(ve, "name", v.Name, 255)
	validateMaxLength(ve, "contact_name", v.ContactName, 255)
	validateMaxLength(ve, "notes", v.Notes, 10000)
	validateMaxLength(ve, "website", v.Website, 255)
	validateMaxLength(ve, "contact_phone", v.ContactPhone, 50)
	validateEmail(ve, "contact_email", v.ContactEmail)
	if v.Status != "" { validateEnum(ve, "status", v.Status, validVendorStatuses) }
	if v.LeadTimeDays < 0 { ve.Add("lead_time_days", "must be non-negative") }
	validateIntRange(ve, "lead_time_days", v.LeadTimeDays, 0, MaxLeadTimeDays)
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	var maxNum int
	db.QueryRow("SELECT COALESCE(MAX(CAST(SUBSTR(id,3) AS INTEGER)),0) FROM vendors WHERE id LIKE 'V-%'").Scan(&maxNum)
	v.ID = fmt.Sprintf("V-%03d", maxNum+1)
	if v.Status == "" { v.Status = "active" }
	_, err := db.Exec("INSERT INTO vendors (id,name,website,contact_name,contact_email,contact_phone,notes,status,lead_time_days) VALUES (?,?,?,?,?,?,?,?,?)",
		v.ID, v.Name, v.Website, v.ContactName, v.ContactEmail, v.ContactPhone, v.Notes, v.Status, v.LeadTimeDays)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	logAudit(db, getUsername(r), "created", "vendor", v.ID, "Created vendor "+v.Name)
	recordChangeJSON(getUsername(r), "vendors", v.ID, "create", nil, v)
	jsonResp(w, v)
}

func handleUpdateVendor(w http.ResponseWriter, r *http.Request, id string) {
	// Snapshot old data before update
	oldSnap, _ := getVendorSnapshot(id)

	var v Vendor
	if err := decodeBody(r, &v); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	
	ve := &ValidationErrors{}
	validateMaxLength(ve, "name", v.Name, 255)
	validateMaxLength(ve, "contact_name", v.ContactName, 255)
	validateMaxLength(ve, "notes", v.Notes, 10000)
	validateMaxLength(ve, "website", v.Website, 255)
	validateMaxLength(ve, "contact_phone", v.ContactPhone, 50)
	validateEmail(ve, "contact_email", v.ContactEmail)
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }
	
	_, err := db.Exec("UPDATE vendors SET name=?,website=?,contact_name=?,contact_email=?,contact_phone=?,notes=?,status=?,lead_time_days=? WHERE id=?",
		v.Name, v.Website, v.ContactName, v.ContactEmail, v.ContactPhone, v.Notes, v.Status, v.LeadTimeDays, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	logAudit(db, getUsername(r), "updated", "vendor", id, "Updated vendor "+v.Name)
	newSnap, _ := getVendorSnapshot(id)
	recordChangeJSON(getUsername(r), "vendors", id, "update", oldSnap, newSnap)
	handleGetVendor(w, r, id)
}

func handleDeleteVendor(w http.ResponseWriter, r *http.Request, id string) {
	// Check for referencing records
	var poCount int
	db.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE vendor_id=?", id).Scan(&poCount)
	if poCount > 0 {
		jsonErr(w, fmt.Sprintf("cannot delete vendor: %d purchase orders reference it", poCount), 409)
		return
	}
	var rfqCount int
	db.QueryRow("SELECT COUNT(*) FROM rfq_vendors WHERE vendor_id=?", id).Scan(&rfqCount)
	if rfqCount > 0 {
		jsonErr(w, fmt.Sprintf("cannot delete vendor: %d RFQs reference it", rfqCount), 409)
		return
	}

	// Snapshot for change_history
	oldSnap, _ := getVendorSnapshot(id)

	undoID, _ := createUndoEntry(getUsername(r), "delete", "vendor", id)
	_, err := db.Exec("DELETE FROM vendors WHERE id=?", id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	logAudit(db, getUsername(r), "deleted", "vendor", id, "Deleted vendor "+id)
	changeID, _ := recordChangeJSON(getUsername(r), "vendors", id, "delete", oldSnap, nil)
	resp := map[string]interface{}{"deleted": id}
	if undoID > 0 {
		resp["undo_id"] = undoID
	}
	if changeID > 0 {
		resp["change_id"] = changeID
	}
	jsonResp(w, resp)
}
