package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// INVENTORY
// ============================================================

func pageInventoryList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	lowStock := r.URL.Query().Get("low_stock") == "true"
	q := strings.ToLower(r.URL.Query().Get("q"))
	query := "SELECT ipn,qty_on_hand,qty_reserved,COALESCE(location,''),reorder_point,reorder_qty,COALESCE(description,''),COALESCE(mpn,''),updated_at FROM inventory"
	if lowStock {
		query += " WHERE qty_on_hand <= reorder_point AND reorder_point > 0"
	}
	query += " ORDER BY ipn"
	rows, err := db.Query(query)
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var items []InventoryItem
	for rows.Next() {
		var i InventoryItem
		rows.Scan(&i.IPN, &i.QtyOnHand, &i.QtyReserved, &i.Location, &i.ReorderPoint, &i.ReorderQty, &i.Description, &i.MPN, &i.UpdatedAt)
		if q != "" && !strings.Contains(strings.ToLower(i.IPN), q) && !strings.Contains(strings.ToLower(i.Description), q) {
			continue
		}
		items = append(items, i)
	}
	if items == nil { items = []InventoryItem{} }

	data := PageData{Title: "Inventory", ActiveNav: "inventory", User: user, Inventory: items, Query: r.URL.Query().Get("q"), Total: len(items), LowStock: lowStock}
	pf := []string{"templates/inventory/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "inventory-table", data); return }
	render(w, pf, "layout", data)
}

func pageInventoryDetail(w http.ResponseWriter, r *http.Request, ipn string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var i InventoryItem
	err := db.QueryRow("SELECT ipn,qty_on_hand,qty_reserved,COALESCE(location,''),reorder_point,reorder_qty,COALESCE(description,''),COALESCE(mpn,''),updated_at FROM inventory WHERE ipn=?", ipn).
		Scan(&i.IPN, &i.QtyOnHand, &i.QtyReserved, &i.Location, &i.ReorderPoint, &i.ReorderQty, &i.Description, &i.MPN, &i.UpdatedAt)
	if err != nil { http.Error(w, "Not found", 404); return }

	rows, _ := db.Query("SELECT id,ipn,type,qty,COALESCE(reference,''),COALESCE(notes,''),created_at FROM inventory_transactions WHERE ipn=? ORDER BY created_at DESC LIMIT 50", ipn)
	var txns []InventoryTransaction
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t InventoryTransaction
			rows.Scan(&t.ID, &t.IPN, &t.Type, &t.Qty, &t.Reference, &t.Notes, &t.CreatedAt)
			txns = append(txns, t)
		}
	}
	if txns == nil { txns = []InventoryTransaction{} }

	data := PageData{Title: "Inventory: " + ipn, ActiveNav: "inventory", User: user, InventoryItem: i, Transactions: txns}
	render(w, []string{"templates/inventory/detail.html"}, "layout", data)
}

func pageInventoryReceiveForm(w http.ResponseWriter, r *http.Request) {
	renderPartial(w, []string{"templates/inventory/list.html"}, "inventory-receive-form", PageData{})
}

func pageInventoryReceive(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ipn := r.FormValue("ipn")
	qty, _ := strconv.ParseFloat(r.FormValue("qty"), 64)
	ref := r.FormValue("reference")
	notes := r.FormValue("notes")
	now := time.Now().Format("2006-01-02 15:04:05")

	db.Exec("INSERT OR IGNORE INTO inventory (ipn) VALUES (?)", ipn)
	db.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)", ipn, "receive", qty, ref, notes, now)
	db.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+?,updated_at=? WHERE ipn=?", qty, now, ipn)
	logAudit(db, getUsername(r), "receive", "inventory", ipn, fmt.Sprintf("Received %.0f of %s", qty, ipn))

	w.Header().Set("HX-Redirect", "/inventory/"+ipn)
	w.WriteHeader(200)
}

// ============================================================
// PROCUREMENT / PURCHASE ORDERS
// ============================================================

func pageProcurementList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	status := r.URL.Query().Get("status")
	query := "SELECT id,vendor_id,status,COALESCE(notes,''),created_at,COALESCE(expected_date,''),received_at FROM purchase_orders"
	var args []interface{}
	if status != "" {
		query += " WHERE status=?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	rows, err := db.Query(query, args...)
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var pos []PurchaseOrder
	for rows.Next() {
		var p PurchaseOrder
		var ra sql.NullString
		rows.Scan(&p.ID, &p.VendorID, &p.Status, &p.Notes, &p.CreatedAt, &p.ExpectedDate, &ra)
		p.ReceivedAt = sp(ra)
		pos = append(pos, p)
	}
	if pos == nil { pos = []PurchaseOrder{} }

	data := PageData{Title: "Purchase Orders", ActiveNav: "procurement", User: user, POs: pos, Status: status}
	pf := []string{"templates/procurement/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "po-table", data); return }
	render(w, pf, "layout", data)
}

func pageProcurementDetail(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var p PurchaseOrder
	var ra sql.NullString
	err := db.QueryRow("SELECT id,vendor_id,status,COALESCE(notes,''),created_at,COALESCE(expected_date,''),received_at FROM purchase_orders WHERE id=?", id).
		Scan(&p.ID, &p.VendorID, &p.Status, &p.Notes, &p.CreatedAt, &p.ExpectedDate, &ra)
	if err != nil { http.Error(w, "PO not found", 404); return }
	p.ReceivedAt = sp(ra)

	lines, _ := db.Query("SELECT id,po_id,ipn,COALESCE(mpn,''),COALESCE(manufacturer,''),qty_ordered,qty_received,unit_price,COALESCE(notes,'') FROM po_lines WHERE po_id=?", id)
	if lines != nil {
		defer lines.Close()
		for lines.Next() {
			var l POLine
			lines.Scan(&l.ID, &l.POID, &l.IPN, &l.MPN, &l.Manufacturer, &l.QtyOrdered, &l.QtyReceived, &l.UnitPrice, &l.Notes)
			p.Lines = append(p.Lines, l)
		}
	}

	data := PageData{Title: "PO: " + p.ID, ActiveNav: "procurement", User: user, PO: p}
	render(w, []string{"templates/procurement/detail.html"}, "layout", data)
}

func pageProcurementNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	vendors := rpLoadAllVendors()
	data := PageData{Title: "New Purchase Order", ActiveNav: "procurement", User: user, AllVendors: vendors}
	render(w, []string{"templates/procurement/form.html"}, "layout", data)
}

func pageProcurementCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := nextID("PO", "purchase_orders", 4)
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO purchase_orders (id,vendor_id,status,notes,created_at,expected_date) VALUES (?,?,?,?,?,?)",
		id, r.FormValue("vendor_id"), "draft", r.FormValue("notes"), now, r.FormValue("expected_date"))
	logAudit(db, getUsername(r), "created", "po", id, "Created PO "+id)
	w.Header().Set("HX-Redirect", "/procurement/"+id)
	w.WriteHeader(200)
}

// ============================================================
// VENDORS
// ============================================================

func rpLoadAllVendors() []Vendor {
	rows, err := db.Query("SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),COALESCE(status,'active'),COALESCE(lead_time_days,0),created_at FROM vendors ORDER BY name")
	if err != nil { return []Vendor{} }
	defer rows.Close()
	var vendors []Vendor
	for rows.Next() {
		var v Vendor
		rows.Scan(&v.ID, &v.Name, &v.Website, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.LeadTimeDays, &v.CreatedAt)
		vendors = append(vendors, v)
	}
	if vendors == nil { return []Vendor{} }
	return vendors
}

func pageVendorsList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	vendors := rpLoadAllVendors()
	data := PageData{Title: "Vendors", ActiveNav: "vendors", User: user, Vendors: vendors, Total: len(vendors)}
	pf := []string{"templates/vendors/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "vendors-table", data); return }
	render(w, pf, "layout", data)
}

func pageVendorDetail(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var v Vendor
	err := db.QueryRow("SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),COALESCE(status,'active'),COALESCE(lead_time_days,0),created_at FROM vendors WHERE id=?", id).
		Scan(&v.ID, &v.Name, &v.Website, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.LeadTimeDays, &v.CreatedAt)
	if err != nil { http.Error(w, "Vendor not found", 404); return }

	rows, _ := db.Query("SELECT p.id,p.ipn,p.vendor_id,v.name,COALESCE(p.mpn,''),p.unit_price,COALESCE(p.min_qty,1),COALESCE(p.lead_time_days,0),COALESCE(p.valid_until,''),p.created_at FROM vendor_prices p JOIN vendors v ON p.vendor_id=v.id WHERE p.vendor_id=? ORDER BY p.ipn", id)
	var prices []VendorPriceEntry
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var p VendorPriceEntry
			rows.Scan(&p.ID, &p.IPN, &p.VendorID, &p.VendorName, &p.MPN, &p.UnitPrice, &p.MinQty, &p.LeadTimeDays, &p.ValidUntil, &p.CreatedAt)
			prices = append(prices, p)
		}
	}

	data := PageData{Title: "Vendor: " + v.Name, ActiveNav: "vendors", User: user, Vendor: v, VendorPrices: prices}
	render(w, []string{"templates/vendors/detail.html"}, "layout", data)
}

func pageVendorNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	data := PageData{Title: "New Vendor", ActiveNav: "vendors", User: user}
	render(w, []string{"templates/vendors/form.html"}, "layout", data)
}

func pageVendorCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := nextID("V", "vendors", 3)
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO vendors (id,name,website,contact_name,contact_email,contact_phone,notes,status,lead_time_days,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
		id, r.FormValue("name"), r.FormValue("website"), r.FormValue("contact_name"),
		r.FormValue("contact_email"), r.FormValue("contact_phone"), r.FormValue("notes"),
		"active", rpAtoi(r.FormValue("lead_time_days")), now)
	logAudit(db, getUsername(r), "created", "vendor", id, "Created vendor "+id)
	w.Header().Set("HX-Redirect", "/vendors/"+id)
	w.WriteHeader(200)
}

func pageVendorEdit(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	var v Vendor
	err := db.QueryRow("SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),COALESCE(status,'active'),COALESCE(lead_time_days,0),created_at FROM vendors WHERE id=?", id).
		Scan(&v.ID, &v.Name, &v.Website, &v.ContactName, &v.ContactEmail, &v.ContactPhone, &v.Notes, &v.Status, &v.LeadTimeDays, &v.CreatedAt)
	if err != nil { http.Error(w, "Not found", 404); return }
	data := PageData{Title: "Edit Vendor", ActiveNav: "vendors", User: user, Vendor: v}
	render(w, []string{"templates/vendors/form.html"}, "layout", data)
}

func pageVendorUpdate(w http.ResponseWriter, r *http.Request, id string) {
	r.ParseForm()
	db.Exec("UPDATE vendors SET name=?,website=?,contact_name=?,contact_email=?,contact_phone=?,notes=?,lead_time_days=? WHERE id=?",
		r.FormValue("name"), r.FormValue("website"), r.FormValue("contact_name"),
		r.FormValue("contact_email"), r.FormValue("contact_phone"), r.FormValue("notes"),
		rpAtoi(r.FormValue("lead_time_days")), id)
	logAudit(db, getUsername(r), "updated", "vendor", id, "Updated vendor "+id)
	w.Header().Set("HX-Redirect", "/vendors/"+id)
	w.WriteHeader(200)
}

func pageVendorDelete(w http.ResponseWriter, r *http.Request, id string) {
	db.Exec("DELETE FROM vendors WHERE id=?", id)
	logAudit(db, getUsername(r), "deleted", "vendor", id, "Deleted vendor "+id)
	w.Header().Set("HX-Redirect", "/vendors")
	w.WriteHeader(200)
}

// ============================================================
// WORK ORDERS
// ============================================================

func pageWorkOrdersList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	status := r.URL.Query().Get("status")
	query := "SELECT id,assembly_ipn,qty,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders"
	var args []interface{}
	if status != "" { query += " WHERE status=?"; args = append(args, status) }
	query += " ORDER BY created_at DESC"
	rows, err := db.Query(query, args...)
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var wos []WorkOrder
	for rows.Next() {
		var wo WorkOrder
		var sa, ca sql.NullString
		rows.Scan(&wo.ID, &wo.AssemblyIPN, &wo.Qty, &wo.Status, &wo.Priority, &wo.Notes, &wo.CreatedAt, &sa, &ca)
		wo.StartedAt = sp(sa); wo.CompletedAt = sp(ca)
		wos = append(wos, wo)
	}
	if wos == nil { wos = []WorkOrder{} }

	data := PageData{Title: "Work Orders", ActiveNav: "workorders", User: user, WorkOrders: wos, Status: status}
	pf := []string{"templates/workorders/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "wo-table", data); return }
	render(w, pf, "layout", data)
}

func pageWorkOrderDetail(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var wo WorkOrder
	var sa, ca sql.NullString
	err := db.QueryRow("SELECT id,assembly_ipn,qty,status,priority,COALESCE(notes,''),created_at,started_at,completed_at FROM work_orders WHERE id=?", id).
		Scan(&wo.ID, &wo.AssemblyIPN, &wo.Qty, &wo.Status, &wo.Priority, &wo.Notes, &wo.CreatedAt, &sa, &ca)
	if err != nil { http.Error(w, "WO not found", 404); return }
	wo.StartedAt = sp(sa); wo.CompletedAt = sp(ca)

	data := PageData{Title: "Work Order: " + wo.ID, ActiveNav: "workorders", User: user, WorkOrder: wo}
	render(w, []string{"templates/workorders/detail.html"}, "layout", data)
}

func pageWorkOrderNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	cats, _, _ := loadPartsFromDir()
	var assemblies []Part
	for _, parts := range cats {
		for _, p := range parts {
			u := strings.ToUpper(p.IPN)
			if strings.HasPrefix(u, "PCA-") || strings.HasPrefix(u, "ASY-") { assemblies = append(assemblies, p) }
		}
	}
	data := PageData{Title: "New Work Order", ActiveNav: "workorders", User: user, AllAssemblies: assemblies}
	render(w, []string{"templates/workorders/form.html"}, "layout", data)
}

func pageWorkOrderCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := nextID("WO", "work_orders", 4)
	now := time.Now().Format("2006-01-02 15:04:05")
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	priority := r.FormValue("priority")
	if priority == "" { priority = "normal" }
	db.Exec("INSERT INTO work_orders (id,assembly_ipn,qty,status,priority,notes,created_at) VALUES (?,?,?,?,?,?,?)",
		id, r.FormValue("assembly_ipn"), qty, "open", priority, r.FormValue("notes"), now)
	logAudit(db, getUsername(r), "created", "workorder", id, "Created WO "+id)
	w.Header().Set("HX-Redirect", "/workorders/"+id)
	w.WriteHeader(200)
}

// ============================================================
// NCRs
// ============================================================

func pageNCRsList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	status := r.URL.Query().Get("status")
	severity := r.URL.Query().Get("severity")
	query := "SELECT id,title,description,COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),created_at,resolved_at FROM ncrs WHERE 1=1"
	var args []interface{}
	if status != "" { query += " AND status=?"; args = append(args, status) }
	if severity != "" { query += " AND severity=?"; args = append(args, severity) }
	query += " ORDER BY created_at DESC"
	rows, err := db.Query(query, args...)
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var ncrs []NCR
	for rows.Next() {
		var n NCR
		var ra sql.NullString
		rows.Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedAt, &ra)
		n.ResolvedAt = sp(ra)
		ncrs = append(ncrs, n)
	}
	if ncrs == nil { ncrs = []NCR{} }

	data := PageData{Title: "NCRs", ActiveNav: "ncr", User: user, NCRs: ncrs, Status: status, Severity: severity}
	pf := []string{"templates/ncrs/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "ncr-table", data); return }
	render(w, pf, "layout", data)
}

func pageNCRDetail(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var n NCR
	var ra sql.NullString
	err := db.QueryRow("SELECT id,title,description,COALESCE(ipn,''),COALESCE(serial_number,''),COALESCE(defect_type,''),severity,status,COALESCE(root_cause,''),COALESCE(corrective_action,''),created_at,resolved_at FROM ncrs WHERE id=?", id).
		Scan(&n.ID, &n.Title, &n.Description, &n.IPN, &n.SerialNumber, &n.DefectType, &n.Severity, &n.Status, &n.RootCause, &n.CorrectiveAction, &n.CreatedAt, &ra)
	if err != nil { http.Error(w, "NCR not found", 404); return }
	n.ResolvedAt = sp(ra)

	data := PageData{Title: "NCR: " + n.ID, ActiveNav: "ncr", User: user, NCR: n}
	render(w, []string{"templates/ncrs/detail.html"}, "layout", data)
}

func pageNCRNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	data := PageData{Title: "New NCR", ActiveNav: "ncr", User: user}
	render(w, []string{"templates/ncrs/form.html"}, "layout", data)
}

func pageNCRCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := nextID("NCR", "ncrs", 3)
	now := time.Now().Format("2006-01-02 15:04:05")
	sev := r.FormValue("severity")
	if sev == "" { sev = "minor" }
	db.Exec("INSERT INTO ncrs (id,title,description,ipn,serial_number,defect_type,severity,status,root_cause,corrective_action,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
		id, r.FormValue("title"), r.FormValue("description"), r.FormValue("ipn"),
		r.FormValue("serial_number"), r.FormValue("defect_type"), sev, "open",
		r.FormValue("root_cause"), r.FormValue("corrective_action"), now)
	logAudit(db, getUsername(r), "created", "ncr", id, "Created NCR "+id)
	w.Header().Set("HX-Redirect", "/ncr/"+id)
	w.WriteHeader(200)
}

// ============================================================
// TEST RECORDS
// ============================================================

func pageTestingList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	rows, err := db.Query("SELECT id,serial_number,COALESCE(ipn,''),COALESCE(firmware_version,''),test_type,result,COALESCE(measurements,''),COALESCE(notes,''),tested_by,tested_at FROM test_records ORDER BY tested_at DESC")
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var tests []TestRecord
	for rows.Next() {
		var t TestRecord
		rows.Scan(&t.ID, &t.SerialNumber, &t.IPN, &t.FirmwareVersion, &t.TestType, &t.Result, &t.Measurements, &t.Notes, &t.TestedBy, &t.TestedAt)
		tests = append(tests, t)
	}
	if tests == nil { tests = []TestRecord{} }

	data := PageData{Title: "Test Records", ActiveNav: "testing", User: user, Tests: tests}
	pf := []string{"templates/testing/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "testing-table", data); return }
	render(w, pf, "layout", data)
}

func pageTestingNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	data := PageData{Title: "New Test Record", ActiveNav: "testing", User: user}
	render(w, []string{"templates/testing/form.html"}, "layout", data)
}

func pageTestingCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO test_records (serial_number,ipn,firmware_version,test_type,result,measurements,notes,tested_by,tested_at) VALUES (?,?,?,?,?,?,?,?,?)",
		r.FormValue("serial_number"), r.FormValue("ipn"), r.FormValue("firmware_version"),
		r.FormValue("test_type"), r.FormValue("result"), r.FormValue("measurements"),
		r.FormValue("notes"), getUsername(r), now)
	logAudit(db, getUsername(r), "created", "test", r.FormValue("serial_number"), "Created test record")
	w.Header().Set("HX-Redirect", "/testing")
	w.WriteHeader(200)
}

// ============================================================
// RMAs
// ============================================================

func pageRMAsList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	status := r.URL.Query().Get("status")
	query := "SELECT id,serial_number,customer,reason,status,COALESCE(defect_description,''),COALESCE(resolution,''),created_at,received_at,resolved_at FROM rmas"
	var args []interface{}
	if status != "" { query += " WHERE status=?"; args = append(args, status) }
	query += " ORDER BY created_at DESC"
	rows, err := db.Query(query, args...)
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var rmas []RMA
	for rows.Next() {
		var rm RMA
		var reca, resa sql.NullString
		rows.Scan(&rm.ID, &rm.SerialNumber, &rm.Customer, &rm.Reason, &rm.Status, &rm.DefectDescription, &rm.Resolution, &rm.CreatedAt, &reca, &resa)
		rm.ReceivedAt = sp(reca); rm.ResolvedAt = sp(resa)
		rmas = append(rmas, rm)
	}
	if rmas == nil { rmas = []RMA{} }

	data := PageData{Title: "RMAs", ActiveNav: "rma", User: user, RMAs: rmas, Status: status}
	pf := []string{"templates/rmas/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "rma-table", data); return }
	render(w, pf, "layout", data)
}

func pageRMADetail(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var rm RMA
	var reca, resa sql.NullString
	err := db.QueryRow("SELECT id,serial_number,customer,reason,status,COALESCE(defect_description,''),COALESCE(resolution,''),created_at,received_at,resolved_at FROM rmas WHERE id=?", id).
		Scan(&rm.ID, &rm.SerialNumber, &rm.Customer, &rm.Reason, &rm.Status, &rm.DefectDescription, &rm.Resolution, &rm.CreatedAt, &reca, &resa)
	if err != nil { http.Error(w, "RMA not found", 404); return }
	rm.ReceivedAt = sp(reca); rm.ResolvedAt = sp(resa)

	data := PageData{Title: "RMA: " + rm.ID, ActiveNav: "rma", User: user, RMA: rm}
	render(w, []string{"templates/rmas/detail.html"}, "layout", data)
}

func pageRMANew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	data := PageData{Title: "New RMA", ActiveNav: "rma", User: user}
	render(w, []string{"templates/rmas/form.html"}, "layout", data)
}

func pageRMACreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := nextID("RMA", "rmas", 3)
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO rmas (id,serial_number,customer,reason,status,defect_description,created_at) VALUES (?,?,?,?,?,?,?)",
		id, r.FormValue("serial_number"), r.FormValue("customer"), r.FormValue("reason"),
		"open", r.FormValue("defect_description"), now)
	logAudit(db, getUsername(r), "created", "rma", id, "Created RMA "+id)
	w.Header().Set("HX-Redirect", "/rma/"+id)
	w.WriteHeader(200)
}

// ============================================================
// DEVICES
// ============================================================

func pageDevicesList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	q := strings.ToLower(r.URL.Query().Get("q"))
	rows, err := db.Query("SELECT serial_number,ipn,firmware_version,COALESCE(customer,''),COALESCE(location,''),status,COALESCE(install_date,''),last_seen,COALESCE(notes,''),created_at FROM devices ORDER BY serial_number")
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var devices []Device
	for rows.Next() {
		var d Device
		var ls sql.NullString
		rows.Scan(&d.SerialNumber, &d.IPN, &d.FirmwareVersion, &d.Customer, &d.Location, &d.Status, &d.InstallDate, &ls, &d.Notes, &d.CreatedAt)
		d.LastSeen = sp(ls)
		if q != "" && !strings.Contains(strings.ToLower(d.SerialNumber), q) && !strings.Contains(strings.ToLower(d.Customer), q) && !strings.Contains(strings.ToLower(d.IPN), q) { continue }
		devices = append(devices, d)
	}
	if devices == nil { devices = []Device{} }

	data := PageData{Title: "Devices", ActiveNav: "devices", User: user, Devices: devices, Query: r.URL.Query().Get("q"), Total: len(devices)}
	pf := []string{"templates/devices/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "devices-table", data); return }
	render(w, pf, "layout", data)
}

func pageDeviceDetail(w http.ResponseWriter, r *http.Request, sn string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var d Device
	var ls sql.NullString
	err := db.QueryRow("SELECT serial_number,ipn,firmware_version,COALESCE(customer,''),COALESCE(location,''),status,COALESCE(install_date,''),last_seen,COALESCE(notes,''),created_at FROM devices WHERE serial_number=?", sn).
		Scan(&d.SerialNumber, &d.IPN, &d.FirmwareVersion, &d.Customer, &d.Location, &d.Status, &d.InstallDate, &ls, &d.Notes, &d.CreatedAt)
	if err != nil { http.Error(w, "Device not found", 404); return }
	d.LastSeen = sp(ls)

	data := PageData{Title: "Device: " + sn, ActiveNav: "devices", User: user, Device: d}
	render(w, []string{"templates/devices/detail.html"}, "layout", data)
}

func pageDeviceNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	data := PageData{Title: "New Device", ActiveNav: "devices", User: user}
	render(w, []string{"templates/devices/form.html"}, "layout", data)
}

func pageDeviceCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	now := time.Now().Format("2006-01-02 15:04:05")
	sn := r.FormValue("serial_number")
	db.Exec("INSERT INTO devices (serial_number,ipn,firmware_version,customer,location,status,install_date,notes,created_at) VALUES (?,?,?,?,?,?,?,?,?)",
		sn, r.FormValue("ipn"), r.FormValue("firmware_version"), r.FormValue("customer"),
		r.FormValue("location"), "active", r.FormValue("install_date"), r.FormValue("notes"), now)
	logAudit(db, getUsername(r), "created", "device", sn, "Created device "+sn)
	w.Header().Set("HX-Redirect", "/devices/"+sn)
	w.WriteHeader(200)
}

// ============================================================
// FIRMWARE CAMPAIGNS
// ============================================================

func pageFirmwareList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	rows, err := db.Query("SELECT id,name,version,COALESCE(category,''),status,COALESCE(target_filter,''),COALESCE(notes,''),created_at,started_at,completed_at FROM firmware_campaigns ORDER BY created_at DESC")
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var campaigns []FirmwareCampaign
	for rows.Next() {
		var c FirmwareCampaign
		var sa, ca sql.NullString
		rows.Scan(&c.ID, &c.Name, &c.Version, &c.Category, &c.Status, &c.TargetFilter, &c.Notes, &c.CreatedAt, &sa, &ca)
		c.StartedAt = sp(sa); c.CompletedAt = sp(ca)
		campaigns = append(campaigns, c)
	}
	if campaigns == nil { campaigns = []FirmwareCampaign{} }

	data := PageData{Title: "Firmware Campaigns", ActiveNav: "firmware", User: user, Campaigns: campaigns}
	pf := []string{"templates/firmware/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "firmware-table", data); return }
	render(w, pf, "layout", data)
}

func pageFirmwareDetail(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var c FirmwareCampaign
	var sa, ca sql.NullString
	err := db.QueryRow("SELECT id,name,version,COALESCE(category,''),status,COALESCE(target_filter,''),COALESCE(notes,''),created_at,started_at,completed_at FROM firmware_campaigns WHERE id=?", id).
		Scan(&c.ID, &c.Name, &c.Version, &c.Category, &c.Status, &c.TargetFilter, &c.Notes, &c.CreatedAt, &sa, &ca)
	if err != nil { http.Error(w, "Campaign not found", 404); return }
	c.StartedAt = sp(sa); c.CompletedAt = sp(ca)

	drows, _ := db.Query("SELECT campaign_id,serial_number,status,updated_at FROM campaign_devices WHERE campaign_id=?", id)
	var cdevs []CampaignDevice
	if drows != nil {
		defer drows.Close()
		for drows.Next() {
			var cd CampaignDevice
			var ua sql.NullString
			drows.Scan(&cd.CampaignID, &cd.SerialNumber, &cd.Status, &ua)
			cd.UpdatedAt = sp(ua)
			cdevs = append(cdevs, cd)
		}
	}

	data := PageData{Title: "Campaign: " + c.Name, ActiveNav: "firmware", User: user, Campaign: c, CampaignDevices: cdevs}
	render(w, []string{"templates/firmware/detail.html"}, "layout", data)
}

func pageFirmwareNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	data := PageData{Title: "New Campaign", ActiveNav: "firmware", User: user}
	render(w, []string{"templates/firmware/form.html"}, "layout", data)
}

func pageFirmwareCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := nextID("FW", "firmware_campaigns", 3)
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO firmware_campaigns (id,name,version,category,status,target_filter,notes,created_at) VALUES (?,?,?,?,?,?,?,?)",
		id, r.FormValue("name"), r.FormValue("version"), r.FormValue("category"),
		"draft", r.FormValue("target_filter"), r.FormValue("notes"), now)
	logAudit(db, getUsername(r), "created", "campaign", id, "Created campaign "+id)
	w.Header().Set("HX-Redirect", "/firmware/"+id)
	w.WriteHeader(200)
}

// ============================================================
// REPORTS
// ============================================================

func pageReports(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	reports := []ReportCard{
		{Name: "Inventory Valuation", Description: "Current stock value by part", URL: "/api/v1/reports/inventory-valuation", Icon: "📦"},
		{Name: "Open ECOs", Description: "All open engineering change orders", URL: "/api/v1/reports/open-ecos", Icon: "🔄"},
		{Name: "WO Throughput", Description: "Work order completion metrics", URL: "/api/v1/reports/wo-throughput", Icon: "⚙️"},
		{Name: "Low Stock", Description: "Parts below reorder point", URL: "/api/v1/reports/low-stock", Icon: "⚠️"},
		{Name: "NCR Summary", Description: "Non-conformance report analysis", URL: "/api/v1/reports/ncr-summary", Icon: "📋"},
	}

	data := PageData{Title: "Reports", ActiveNav: "reports", User: user, ReportList: reports}
	render(w, []string{"templates/reports/index.html"}, "layout", data)
}

// ============================================================
// AUDIT LOG
// ============================================================

func pageAuditList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	q := r.URL.Query().Get("q")
	module := r.URL.Query().Get("module")
	query := "SELECT id, username, action, module, record_id, summary, created_at FROM audit_log WHERE 1=1"
	var args []interface{}
	if q != "" {
		query += " AND (summary LIKE ? OR record_id LIKE ? OR username LIKE ?)"
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}
	if module != "" { query += " AND module=?"; args = append(args, module) }
	query += " ORDER BY created_at DESC LIMIT 200"

	rows, err := db.Query(query, args...)
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var entries []AuditEntry
	for rows.Next() {
		var a AuditEntry
		rows.Scan(&a.ID, &a.Username, &a.Action, &a.Module, &a.RecordID, &a.Summary, &a.CreatedAt)
		entries = append(entries, a)
	}
	if entries == nil { entries = []AuditEntry{} }

	data := PageData{Title: "Audit Log", ActiveNav: "audit", User: user, AuditEntries: entries, Query: r.URL.Query().Get("q")}
	pf := []string{"templates/audit/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "audit-table", data); return }
	render(w, pf, "layout", data)
}

// ============================================================
// USERS
// ============================================================

func pageUsersList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	rows, err := db.Query("SELECT id,username,display_name,role,active,created_at,last_login FROM users ORDER BY username")
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var users []UserFull
	for rows.Next() {
		var u UserFull
		var ll sql.NullString
		rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Active, &u.CreatedAt, &ll)
		u.LastLogin = sp(ll)
		users = append(users, u)
	}
	if users == nil { users = []UserFull{} }

	data := PageData{Title: "Users", ActiveNav: "users", User: user, UserList: users}
	render(w, []string{"templates/users/list.html"}, "layout", data)
}

func pageUserNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	data := PageData{Title: "New User", ActiveNav: "users", User: user}
	render(w, []string{"templates/users/form.html"}, "layout", data)
}

func pageUserCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")
	displayName := r.FormValue("display_name")
	role := r.FormValue("role")
	if role == "" { role = "user" }
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO users (username,password_hash,display_name,role,active,created_at) VALUES (?,?,?,?,1,?)",
		username, string(hash), displayName, role, now)
	logAudit(db, getUsername(r), "created", "user", username, "Created user "+username)
	w.Header().Set("HX-Redirect", "/users")
	w.WriteHeader(200)
}

// ============================================================
// API KEYS
// ============================================================

func pageAPIKeysList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	rows, err := db.Query("SELECT id,name,SUBSTR(key,1,12) || '...',active,created_at,COALESCE(last_used,'never') FROM api_keys ORDER BY created_at DESC")
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var keys []APIKeyEntry
	for rows.Next() {
		var k APIKeyEntry
		rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.Active, &k.CreatedAt, &k.LastUsed)
		keys = append(keys, k)
	}
	if keys == nil { keys = []APIKeyEntry{} }

	data := PageData{Title: "API Keys", ActiveNav: "apikeys", User: user, APIKeyList: keys}
	render(w, []string{"templates/apikeys/list.html"}, "layout", data)
}

func pageAPIKeyGenerate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	if name == "" { name = "api-key" }
	token := generateToken()
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO api_keys (name,key,active,created_at) VALUES (?,?,1,?)", name, token, now)
	logAudit(db, getUsername(r), "created", "apikey", name, "Generated API key: "+name)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="card mb-4 border-l-4 border-green-500 bg-green-50 p-4">
		<p class="font-semibold text-green-600 mb-2">API Key Generated</p>
		<p class="text-sm mb-2">Copy this key now — it won't be shown again:</p>
		<code class="block bg-gray-100 p-2 rounded text-sm font-mono break-all">%s</code>
	</div>`, token)
}

// ============================================================
// EMAIL SETTINGS
// ============================================================

func pageEmailSettings(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var cfg EmailConfigData
	db.QueryRow("SELECT COALESCE(smtp_host,''),COALESCE(smtp_port,587),COALESCE(smtp_user,''),COALESCE(smtp_password,''),COALESCE(from_address,''),COALESCE(enabled,0) FROM email_config LIMIT 1").
		Scan(&cfg.SMTPHost, &cfg.SMTPPort, &cfg.SMTPUser, &cfg.SMTPPassword, &cfg.FromAddress, &cfg.Enabled)

	data := PageData{Title: "Email Settings", ActiveNav: "email", User: user, EmailCfg: cfg}
	render(w, []string{"templates/email/index.html"}, "layout", data)
}

func pageEmailSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	port, _ := strconv.Atoi(r.FormValue("smtp_port"))
	enabled := 0
	if r.FormValue("enabled") == "on" || r.FormValue("enabled") == "1" { enabled = 1 }
	db.Exec("DELETE FROM email_config")
	db.Exec("INSERT INTO email_config (smtp_host,smtp_port,smtp_user,smtp_password,from_address,enabled) VALUES (?,?,?,?,?,?)",
		r.FormValue("smtp_host"), port, r.FormValue("smtp_user"), r.FormValue("smtp_password"),
		r.FormValue("from_address"), enabled)
	logAudit(db, getUsername(r), "updated", "email", "config", "Updated email settings")
	w.Header().Set("HX-Redirect", "/email")
	w.WriteHeader(200)
}

// ============================================================
// DOCUMENTS
// ============================================================

func pageDocsList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	rows, err := db.Query("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents ORDER BY updated_at DESC")
	if err != nil { http.Error(w, err.Error(), 500); return }
	defer rows.Close()
	var docs []Document
	for rows.Next() {
		var d Document
		rows.Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
		docs = append(docs, d)
	}
	if docs == nil { docs = []Document{} }

	data := PageData{Title: "Documents", ActiveNav: "docs", User: user, Documents: docs}
	pf := []string{"templates/documents/list.html"}
	if isHTMX(r) { renderPartial(w, pf, "docs-table", data); return }
	render(w, pf, "layout", data)
}

func pageDocDetail(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

	var d Document
	err := db.QueryRow("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents WHERE id=?", id).
		Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil { http.Error(w, "Document not found", 404); return }

	data := PageData{Title: "Doc: " + d.Title, ActiveNav: "docs", User: user, Document: d}
	render(w, []string{"templates/documents/detail.html"}, "layout", data)
}

func pageDocNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
	data := PageData{Title: "New Document", ActiveNav: "docs", User: user}
	render(w, []string{"templates/documents/form.html"}, "layout", data)
}

func pageDocCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := nextID("DOC", "documents", 3)
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec("INSERT INTO documents (id,title,category,ipn,revision,status,content,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
		id, r.FormValue("title"), r.FormValue("category"), r.FormValue("ipn"),
		r.FormValue("revision"), "draft", r.FormValue("content"), getUsername(r), now, now)
	logAudit(db, getUsername(r), "created", "document", id, "Created document "+id)
	w.Header().Set("HX-Redirect", "/docs/"+id)
	w.WriteHeader(200)
}

// ============================================================
// GLOBAL SEARCH
// ============================================================

func pageSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	if len(q) < 2 { w.Write([]byte("")); return }

	var results []string

	cats, _, _ := loadPartsFromDir()
	count := 0
	for _, parts := range cats {
		for _, p := range parts {
			if count >= 5 { break }
			if strings.Contains(strings.ToLower(p.IPN), q) {
				results = append(results, fmt.Sprintf(`<a href="/parts/%s" class="block px-3 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm">🔧 %s</a>`, p.IPN, p.IPN))
				count++
			}
		}
	}

	erows, _ := db.Query("SELECT id,title FROM ecos WHERE LOWER(id) LIKE ? OR LOWER(title) LIKE ? LIMIT 3", "%"+q+"%", "%"+q+"%")
	if erows != nil {
		defer erows.Close()
		for erows.Next() {
			var eid, title string
			erows.Scan(&eid, &title)
			if len(title) > 40 { title = title[:40] + "…" }
			results = append(results, fmt.Sprintf(`<a href="/ecos/%s" class="block px-3 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm">🔄 %s: %s</a>`, eid, eid, title))
		}
	}

	drows, _ := db.Query("SELECT serial_number FROM devices WHERE LOWER(serial_number) LIKE ? LIMIT 3", "%"+q+"%")
	if drows != nil {
		defer drows.Close()
		for drows.Next() {
			var sn string
			drows.Scan(&sn)
			results = append(results, fmt.Sprintf(`<a href="/devices/%s" class="block px-3 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm">📱 %s</a>`, sn, sn))
		}
	}

	if len(results) == 0 {
		w.Write([]byte(`<div class="bg-white dark:bg-gray-800 rounded shadow p-3 text-sm text-gray-500">No results</div>`))
		return
	}

	w.Write([]byte(`<div class="bg-white dark:bg-gray-800 rounded shadow">`))
	for _, res := range results {
		w.Write([]byte(res))
	}
	w.Write([]byte(`</div>`))
}

// ============================================================
// HELPERS
// ============================================================

func rpAtoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
