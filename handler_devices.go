package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func handleListDevices(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT serial_number,ipn,COALESCE(firmware_version,''),COALESCE(customer,''),COALESCE(location,''),status,COALESCE(install_date,''),last_seen,COALESCE(notes,''),created_at FROM devices ORDER BY serial_number")
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []Device
	for rows.Next() {
		var d Device
		var ls sql.NullString
		rows.Scan(&d.SerialNumber, &d.IPN, &d.FirmwareVersion, &d.Customer, &d.Location, &d.Status, &d.InstallDate, &ls, &d.Notes, &d.CreatedAt)
		d.LastSeen = sp(ls)
		items = append(items, d)
	}
	if items == nil { items = []Device{} }
	jsonResp(w, items)
}

func handleGetDevice(w http.ResponseWriter, r *http.Request, serial string) {
	var d Device
	var ls sql.NullString
	err := db.QueryRow("SELECT serial_number,ipn,COALESCE(firmware_version,''),COALESCE(customer,''),COALESCE(location,''),status,COALESCE(install_date,''),last_seen,COALESCE(notes,''),created_at FROM devices WHERE serial_number=?", serial).
		Scan(&d.SerialNumber, &d.IPN, &d.FirmwareVersion, &d.Customer, &d.Location, &d.Status, &d.InstallDate, &ls, &d.Notes, &d.CreatedAt)
	if err != nil { jsonErr(w, "not found", 404); return }
	d.LastSeen = sp(ls)
	jsonResp(w, d)
}

func handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var d Device
	if err := decodeBody(r, &d); err != nil { jsonErr(w, "invalid body", 400); return }
	if d.Status == "" { d.Status = "active" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO devices (serial_number,ipn,firmware_version,customer,location,status,install_date,notes,created_at) VALUES (?,?,?,?,?,?,?,?,?)",
		d.SerialNumber, d.IPN, d.FirmwareVersion, d.Customer, d.Location, d.Status, d.InstallDate, d.Notes, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	d.CreatedAt = now
	logAudit(db, getUsername(r), "created", "device", d.SerialNumber, "Registered device "+d.SerialNumber)
	jsonResp(w, d)
}

func handleUpdateDevice(w http.ResponseWriter, r *http.Request, serial string) {
	var d Device
	if err := decodeBody(r, &d); err != nil { jsonErr(w, "invalid body", 400); return }
	_, err := db.Exec("UPDATE devices SET ipn=?,firmware_version=?,customer=?,location=?,status=?,install_date=?,notes=? WHERE serial_number=?",
		d.IPN, d.FirmwareVersion, d.Customer, d.Location, d.Status, d.InstallDate, d.Notes, serial)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "device", serial, "Updated device "+serial)
	handleGetDevice(w, r, serial)
}

func handleExportDevices(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT serial_number,ipn,COALESCE(firmware_version,''),COALESCE(customer,''),COALESCE(location,''),status,COALESCE(install_date,''),COALESCE(notes,'') FROM devices ORDER BY serial_number")
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=devices.csv")
	cw := csv.NewWriter(w)
	cw.Write([]string{"serial_number", "ipn", "firmware_version", "customer", "location", "status", "install_date", "notes"})
	for rows.Next() {
		var sn, ipn, fw, cust, loc, status, install, notes string
		rows.Scan(&sn, &ipn, &fw, &cust, &loc, &status, &install, &notes)
		cw.Write([]string{sn, ipn, fw, cust, loc, status, install, notes})
	}
	cw.Flush()
}

func handleImportDevices(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		jsonErr(w, "file required", 400)
		return
	}
	defer file.Close()
	cr := csv.NewReader(file)
	headers, err := cr.Read()
	if err != nil {
		jsonErr(w, "invalid CSV", 400)
		return
	}
	// Map header names to indices
	idx := map[string]int{}
	for i, h := range headers {
		idx[strings.TrimSpace(strings.ToLower(h))] = i
	}
	colOf := func(row []string, name string) string {
		if i, ok := idx[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	imported, skipped := 0, 0
	var errors []string
	now := time.Now().Format("2006-01-02 15:04:05")
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("row parse error: %v", err))
			continue
		}
		sn := colOf(row, "serial_number")
		ipn := colOf(row, "ipn")
		if sn == "" || ipn == "" {
			skipped++
			continue
		}
		status := colOf(row, "status")
		if status == "" {
			status = "active"
		}
		_, err = db.Exec(`INSERT INTO devices (serial_number,ipn,firmware_version,customer,location,status,install_date,notes,created_at) VALUES (?,?,?,?,?,?,?,?,?)
			ON CONFLICT(serial_number) DO UPDATE SET ipn=excluded.ipn,firmware_version=excluded.firmware_version,customer=excluded.customer,location=excluded.location,status=excluded.status,install_date=excluded.install_date,notes=excluded.notes`,
			sn, ipn, colOf(row, "firmware_version"), colOf(row, "customer"), colOf(row, "location"), status, colOf(row, "install_date"), colOf(row, "notes"), now)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %s: %v", sn, err))
		} else {
			imported++
		}
	}
	logAudit(db, getUsername(r), "imported", "device", "", fmt.Sprintf("Imported %d devices", imported))
	jsonResp(w, map[string]interface{}{"imported": imported, "skipped": skipped, "errors": errors})
}

func handleDeviceHistory(w http.ResponseWriter, r *http.Request, serial string) {
	// Get test records
	tests := []TestRecord{}
	rows, _ := db.Query("SELECT id,serial_number,ipn,COALESCE(firmware_version,''),COALESCE(test_type,''),result,COALESCE(measurements,''),COALESCE(notes,''),tested_by,tested_at FROM test_records WHERE serial_number=? ORDER BY tested_at DESC", serial)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t TestRecord
			rows.Scan(&t.ID, &t.SerialNumber, &t.IPN, &t.FirmwareVersion, &t.TestType, &t.Result, &t.Measurements, &t.Notes, &t.TestedBy, &t.TestedAt)
			tests = append(tests, t)
		}
	}
	// Get campaign updates
	campaigns := []CampaignDevice{}
	rows2, _ := db.Query("SELECT campaign_id,serial_number,status,updated_at FROM campaign_devices WHERE serial_number=?", serial)
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var cd CampaignDevice
			var ua sql.NullString
			rows2.Scan(&cd.CampaignID, &cd.SerialNumber, &cd.Status, &ua)
			cd.UpdatedAt = sp(ua)
			campaigns = append(campaigns, cd)
		}
	}
	jsonResp(w, map[string]interface{}{"tests": tests, "campaigns": campaigns})
}
