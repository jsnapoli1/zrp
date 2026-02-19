package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// handleExportParts exports parts list to CSV or Excel
func handleExportParts(w http.ResponseWriter, r *http.Request) {
	// This will be updated at the end with actual count
	defer func() {
		// Count is logged after CSV is written
	}()
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

	query := "SELECT ipn,COALESCE(category,''),COALESCE(description,''),COALESCE(mpn,''),COALESCE(manufacturer,''),lifecycle,COALESCE(notes,'') FROM parts WHERE 1=1"
	var args []interface{}

	if search != "" {
		query += " AND (ipn LIKE ? OR description LIKE ? OR mpn LIKE ?)"
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}
	if category != "" {
		query += " AND category=?"
		args = append(args, category)
	}

	query += " ORDER BY ipn"

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	headers := []string{"IPN", "Category", "Description", "MPN", "Manufacturer", "Lifecycle", "Notes"}
	var data [][]string

	for rows.Next() {
		var ipn, category, description, mpn, manufacturer, lifecycle, notes string
		rows.Scan(&ipn, &category, &description, &mpn, &manufacturer, &lifecycle, &notes)
		data = append(data, []string{ipn, category, description, mpn, manufacturer, lifecycle, notes})
	}

	// Log the export
	LogDataExport(db, r, "parts", format, len(data))

	if format == "xlsx" {
		exportExcel(w, "Parts", headers, data)
	} else {
		exportCSV(w, "parts.csv", headers, data)
	}
}

// handleExportInventory exports inventory list to CSV or Excel
func handleExportInventory(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	lowStock := r.URL.Query().Get("low_stock")

	query := "SELECT ipn,qty_on_hand,qty_reserved,COALESCE(location,''),reorder_point,reorder_qty,COALESCE(description,''),COALESCE(mpn,''),updated_at FROM inventory"
	if lowStock == "true" {
		query += " WHERE qty_on_hand <= reorder_point AND reorder_point > 0"
	}
	query += " ORDER BY ipn"

	rows, err := db.Query(query)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	headers := []string{"IPN", "Qty On Hand", "Qty Reserved", "Location", "Reorder Point", "Reorder Qty", "Description", "MPN", "Updated At"}
	var data [][]string

	for rows.Next() {
		var ipn, location, description, mpn, updatedAt string
		var qtyOnHand, qtyReserved, reorderPoint, reorderQty float64
		rows.Scan(&ipn, &qtyOnHand, &qtyReserved, &location, &reorderPoint, &reorderQty, &description, &mpn, &updatedAt)
		data = append(data, []string{
			ipn,
			fmt.Sprintf("%.2f", qtyOnHand),
			fmt.Sprintf("%.2f", qtyReserved),
			location,
			fmt.Sprintf("%.2f", reorderPoint),
			fmt.Sprintf("%.2f", reorderQty),
			description,
			mpn,
			updatedAt,
		})
	}

	// Log the export
	LogDataExport(db, r, "inventory", format, len(data))

	if format == "xlsx" {
		exportExcel(w, "Inventory", headers, data)
	} else {
		exportCSV(w, "inventory.csv", headers, data)
	}
}

// handleExportWorkOrders exports work orders to CSV or Excel
func handleExportWorkOrders(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	status := r.URL.Query().Get("status")
	query := "SELECT id,assembly_ipn,qty,COALESCE(qty_good,0),COALESCE(qty_scrap,0),status,priority,COALESCE(notes,''),created_at,COALESCE(started_at,''),COALESCE(completed_at,'') FROM work_orders"
	var args []interface{}

	if status != "" {
		query += " WHERE status=?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	headers := []string{"ID", "Assembly IPN", "Qty", "Qty Good", "Qty Scrap", "Status", "Priority", "Notes", "Created At", "Started At", "Completed At"}
	var data [][]string

	for rows.Next() {
		var id, assemblyIPN, status, priority, notes, createdAt, startedAt, completedAt string
		var qty, qtyGood, qtyScrap int
		rows.Scan(&id, &assemblyIPN, &qty, &qtyGood, &qtyScrap, &status, &priority, &notes, &createdAt, &startedAt, &completedAt)
		data = append(data, []string{
			id,
			assemblyIPN,
			strconv.Itoa(qty),
			strconv.Itoa(qtyGood),
			strconv.Itoa(qtyScrap),
			status,
			priority,
			notes,
			createdAt,
			startedAt,
			completedAt,
		})
	}

	// Log the export
	LogDataExport(db, r, "work_orders", format, len(data))

	if format == "xlsx" {
		exportExcel(w, "WorkOrders", headers, data)
	} else {
		exportCSV(w, "work_orders.csv", headers, data)
	}
}

// handleExportECOs exports ECOs to CSV or Excel
func handleExportECOs(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	status := r.URL.Query().Get("status")
	query := "SELECT id,title,COALESCE(description,''),COALESCE(status,''),COALESCE(priority,''),COALESCE(affected_ipns,''),COALESCE(created_by,''),COALESCE(created_at,''),COALESCE(updated_at,''),COALESCE(approved_at,''),COALESCE(approved_by,''),COALESCE(ncr_id,'') FROM ecos"
	var args []interface{}

	if status != "" {
		query += " WHERE status=?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	headers := []string{"ID", "Title", "Description", "Status", "Priority", "Affected IPNs", "Created By", "Created At", "Updated At", "Approved At", "Approved By", "NCR ID"}
	var data [][]string

	for rows.Next() {
		var id, title, description, status, priority, affectedIPNs, createdBy, createdAt, updatedAt, approvedAt, approvedBy, ncrID string
		rows.Scan(&id, &title, &description, &status, &priority, &affectedIPNs, &createdBy, &createdAt, &updatedAt, &approvedAt, &approvedBy, &ncrID)
		data = append(data, []string{id, title, description, status, priority, affectedIPNs, createdBy, createdAt, updatedAt, approvedAt, approvedBy, ncrID})
	}

	// Log the export
	LogDataExport(db, r, "ecos", format, len(data))

	if format == "xlsx" {
		exportExcel(w, "ECOs", headers, data)
	} else {
		exportCSV(w, "ecos.csv", headers, data)
	}
}

// handleExportVendors exports vendors to CSV or Excel
func handleExportVendors(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	status := r.URL.Query().Get("status")
	query := "SELECT id,name,COALESCE(website,''),COALESCE(contact_name,''),COALESCE(contact_email,''),COALESCE(contact_phone,''),COALESCE(notes,''),status,lead_time_days,created_at FROM vendors"
	var args []interface{}

	if status != "" {
		query += " WHERE status=?"
		args = append(args, status)
	}

	query += " ORDER BY name"

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	headers := []string{"ID", "Name", "Website", "Contact Name", "Contact Email", "Contact Phone", "Notes", "Status", "Lead Time Days", "Created At"}
	var data [][]string

	for rows.Next() {
		var id, name, website, contactName, contactEmail, contactPhone, notes, status, createdAt string
		var leadTimeDays int
		rows.Scan(&id, &name, &website, &contactName, &contactEmail, &contactPhone, &notes, &status, &leadTimeDays, &createdAt)
		data = append(data, []string{id, name, website, contactName, contactEmail, contactPhone, notes, status, strconv.Itoa(leadTimeDays), createdAt})
	}

	// Log the export
	LogDataExport(db, r, "vendors", format, len(data))

	if format == "xlsx" {
		exportExcel(w, "Vendors", headers, data)
	} else {
		exportCSV(w, "vendors.csv", headers, data)
	}
}

// exportCSV writes data to CSV format
func exportCSV(w http.ResponseWriter, filename string, headers []string, data [][]string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write headers
	if err := writer.Write(headers); err != nil {
		http.Error(w, "Failed to write CSV headers", 500)
		return
	}

	// Write data
	for _, row := range data {
		if err := writer.Write(row); err != nil {
			http.Error(w, "Failed to write CSV row", 500)
			return
		}
	}
}

// exportExcel writes data to Excel format
func exportExcel(w http.ResponseWriter, sheetName string, headers []string, data [][]string) {
	f := excelize.NewFile()
	defer f.Close()

	// Create a new sheet
	index, err := f.NewSheet(sheetName)
	if err != nil {
		http.Error(w, "Failed to create Excel sheet", 500)
		return
	}

	// Set the active sheet
	f.SetActiveSheet(index)

	// Create header style
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#D3D3D3"},
			Pattern: 1,
		},
	})
	if err != nil {
		http.Error(w, "Failed to create header style", 500)
		return
	}

	// Write headers
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Write data
	for rowIdx, row := range data {
		for colIdx, value := range row {
			cell := fmt.Sprintf("%s%d", string(rune('A'+colIdx)), rowIdx+2)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// Auto-fit columns
	for i := range headers {
		col := string(rune('A' + i))
		f.SetColWidth(sheetName, col, col, 15)
	}

	// Delete default Sheet1 if we created a different sheet
	if sheetName != "Sheet1" {
		f.DeleteSheet("Sheet1")
	}

	// Write to response
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.xlsx", strings.ToLower(sheetName)))

	if err := f.Write(w); err != nil {
		http.Error(w, "Failed to write Excel file", 500)
		return
	}
}
