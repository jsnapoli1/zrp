package main

import (
	"fmt"
	"net/http"
)

func handleListVendors(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),status,lead_time_days,created_at FROM vendors ORDER BY name")
	if err != nil { 
		if isHTMX(r) {
			http.Error(w, err.Error(), 500)
		} else {
			jsonErr(w, err.Error(), 500)
		}
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
	
	if isHTMX(r) {
		// Return just the table for HTMX updates
		data := PageData{Vendors: items}
		renderPartial(w, []string{"templates/vendors/list.html"}, "vendors-table", data)
	} else {
		// Full page render
		data := PageData{
			Title:     "Vendors",
			ActiveNav: "vendors", 
			Vendors:   items,
			User:      getCurrentUser(r),
		}
		render(w, []string{"templates/vendors/list.html"}, "layout", data)
	}
}

func handleGetVendor(w http.ResponseWriter, r *http.Request, id string) {
	var v Vendor
	err := db.QueryRow("SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),status,lead_time_days,created_at FROM vendors WHERE id=?", id).
		Scan(&v.ID, &v.Name, &v.Website, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.LeadTimeDays, &v.CreatedAt)
	if err != nil { 
		if isHTMX(r) {
			http.Error(w, "Vendor not found", 404)
		} else {
			jsonErr(w, "not found", 404)
		}
		return 
	}
	
	if isHTMX(r) {
		data := PageData{Vendor: v}
		renderPartial(w, []string{"templates/vendors/form.html"}, "vendors-edit-form", data)
	} else {
		jsonResp(w, v)
	}
}

func handleCreateVendor(w http.ResponseWriter, r *http.Request) {
	var v Vendor
	if err := decodeBody(r, &v); err != nil { 
		if isHTMX(r) {
			http.Error(w, "Invalid form data", 400)
		} else {
			jsonErr(w, "invalid body", 400)
		}
		return 
	}
	// Auto-generate ID
	var maxNum int
	db.QueryRow("SELECT COALESCE(MAX(CAST(SUBSTR(id,3) AS INTEGER)),0) FROM vendors WHERE id LIKE 'V-%'").Scan(&maxNum)
	v.ID = fmt.Sprintf("V-%03d", maxNum+1)
	if v.Status == "" { v.Status = "active" }
	_, err := db.Exec("INSERT INTO vendors (id,name,website,contact_name,contact_email,contact_phone,notes,status,lead_time_days) VALUES (?,?,?,?,?,?,?,?,?)",
		v.ID, v.Name, v.Website, v.ContactName, v.ContactEmail, v.ContactPhone, v.Notes, v.Status, v.LeadTimeDays)
	if err != nil { 
		if isHTMX(r) {
			http.Error(w, err.Error(), 500)
		} else {
			jsonErr(w, err.Error(), 500)
		}
		return 
	}
	logAudit(db, getUsername(r), "created", "vendor", v.ID, "Created vendor "+v.Name)
	
	if isHTMX(r) {
		data := PageData{Vendor: v}
		renderPartial(w, []string{"templates/vendors/form.html"}, "vendors-create-success", data)
	} else {
		jsonResp(w, v)
	}
}

func handleUpdateVendor(w http.ResponseWriter, r *http.Request, id string) {
	var v Vendor
	if err := decodeBody(r, &v); err != nil { 
		if isHTMX(r) {
			http.Error(w, "Invalid form data", 400)
		} else {
			jsonErr(w, "invalid body", 400)
		}
		return 
	}
	_, err := db.Exec("UPDATE vendors SET name=?,website=?,contact_name=?,contact_email=?,contact_phone=?,notes=?,status=?,lead_time_days=? WHERE id=?",
		v.Name, v.Website, v.ContactName, v.ContactEmail, v.ContactPhone, v.Notes, v.Status, v.LeadTimeDays, id)
	if err != nil { 
		if isHTMX(r) {
			http.Error(w, err.Error(), 500)
		} else {
			jsonErr(w, err.Error(), 500)
		}
		return 
	}
	logAudit(db, getUsername(r), "updated", "vendor", id, "Updated vendor "+v.Name)
	
	if isHTMX(r) {
		v.ID = id // Ensure ID is set for the success message
		data := PageData{Vendor: v}
		renderPartial(w, []string{"templates/vendors/form.html"}, "vendors-update-success", data)
	} else {
		handleGetVendor(w, r, id)
	}
}

func handleDeleteVendor(w http.ResponseWriter, r *http.Request, id string) {
	_, err := db.Exec("DELETE FROM vendors WHERE id=?", id)
	if err != nil { 
		if isHTMX(r) {
			http.Error(w, err.Error(), 500)
		} else {
			jsonErr(w, err.Error(), 500)
		}
		return 
	}
	logAudit(db, getUsername(r), "deleted", "vendor", id, "Deleted vendor "+id)
	
	if isHTMX(r) {
		renderPartial(w, []string{"templates/vendors/form.html"}, "vendors-delete-success", PageData{})
	} else {
		jsonResp(w, map[string]string{"deleted": id})
	}
}

func handleVendorsNewForm(w http.ResponseWriter, r *http.Request) {
	if !isHTMX(r) {
		http.Error(w, "Direct access not supported", 404)
		return
	}
	
	renderPartial(w, []string{"templates/vendors/form.html"}, "vendors-new-form", PageData{})
}
