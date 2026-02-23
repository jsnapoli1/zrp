package procurement

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zrp/internal/database"
	"zrp/internal/models"
	"zrp/internal/response"
)

// ListReceiving returns receiving inspection records.
func (h *Handler) ListReceiving(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	query := `SELECT ri.id, ri.po_id, ri.po_line_id, ri.ipn, ri.qty_received, ri.qty_passed, ri.qty_failed, ri.qty_on_hold,
		COALESCE(ri.inspector,''), ri.inspected_at, COALESCE(ri.notes,''), ri.created_at
		FROM receiving_inspections ri`

	switch status {
	case "pending":
		query += " WHERE ri.inspected_at IS NULL"
	case "inspected":
		query += " WHERE ri.inspected_at IS NOT NULL"
	}
	query += " ORDER BY ri.created_at DESC"

	rows, err := h.DB.Query(query)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var items []models.ReceivingInspection
	for rows.Next() {
		var ri models.ReceivingInspection
		var ia sql.NullString
		rows.Scan(&ri.ID, &ri.POID, &ri.POLineID, &ri.IPN, &ri.QtyReceived, &ri.QtyPassed, &ri.QtyFailed, &ri.QtyOnHold,
			&ri.Inspector, &ia, &ri.Notes, &ri.CreatedAt)
		ri.InspectedAt = database.SP(ia)
		items = append(items, ri)
	}
	if items == nil {
		items = []models.ReceivingInspection{}
	}
	response.JSON(w, items)
}

// InspectReceiving performs a receiving inspection on a record.
func (h *Handler) InspectReceiving(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Err(w, "invalid id", 400)
		return
	}

	var body struct {
		QtyPassed float64 `json:"qty_passed"`
		QtyFailed float64 `json:"qty_failed"`
		QtyOnHold float64 `json:"qty_on_hold"`
		Inspector string  `json:"inspector"`
		Notes     string  `json:"notes"`
	}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	// Verify inspection record exists and has NOT been inspected yet
	var ri models.ReceivingInspection
	var ia sql.NullString
	err = h.DB.QueryRow(`SELECT id, po_id, po_line_id, ipn, qty_received, qty_passed, qty_failed, qty_on_hold,
		COALESCE(inspector,''), inspected_at, COALESCE(notes,''), created_at
		FROM receiving_inspections WHERE id=? AND inspected_at IS NULL`, id).
		Scan(&ri.ID, &ri.POID, &ri.POLineID, &ri.IPN, &ri.QtyReceived, &ri.QtyPassed, &ri.QtyFailed, &ri.QtyOnHold,
			&ri.Inspector, &ia, &ri.Notes, &ri.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Err(w, "inspection record not found or already completed", 404)
		} else {
			response.Err(w, "inspection record not found", 404)
		}
		return
	}

	// Validate totals
	total := body.QtyPassed + body.QtyFailed + body.QtyOnHold
	if total > ri.QtyReceived {
		response.Err(w, fmt.Sprintf("inspection quantities (%.0f) exceed received quantity (%.0f)", total, ri.QtyReceived), 400)
		return
	}

	inspector := body.Inspector
	if inspector == "" {
		inspector = h.GetUsername(r)
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = h.DB.Exec(`UPDATE receiving_inspections SET qty_passed=?, qty_failed=?, qty_on_hold=?, inspector=?, inspected_at=?, notes=? WHERE id=?`,
		body.QtyPassed, body.QtyFailed, body.QtyOnHold, inspector, now, body.Notes, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// If items passed, update inventory
	if body.QtyPassed > 0 {
		h.DB.Exec("INSERT OR IGNORE INTO inventory (ipn) VALUES (?)", ri.IPN)
		h.DB.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+?,updated_at=? WHERE ipn=?", body.QtyPassed, now, ri.IPN)
		h.DB.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
			ri.IPN, "receive", body.QtyPassed, ri.POID, fmt.Sprintf("Inspection passed (RI-%d)", id), now)
	}

	// If items failed, auto-create NCR
	if body.QtyFailed > 0 {
		ncrID := h.NextIDFunc("NCR", "ncrs", 3)
		ncrTitle := fmt.Sprintf("Receiving inspection failure: %s (PO %s)", ri.IPN, ri.POID)
		ncrDesc := fmt.Sprintf("%.0f units failed receiving inspection.\nInspector: %s\nNotes: %s", body.QtyFailed, inspector, body.Notes)
		h.DB.Exec(`INSERT INTO ncrs (id,title,description,ipn,defect_type,severity,status,created_at) VALUES (?,?,?,?,?,?,?,?)`,
			ncrID, ncrTitle, ncrDesc, ri.IPN, "receiving", "minor", "open", now)
		h.LogAudit(inspector, "created", "ncr", ncrID, "Auto-created from receiving inspection failure")
	}

	h.LogAudit(inspector, "inspected", "receiving", fmt.Sprintf("%d", id),
		fmt.Sprintf("Inspected RI-%d: %.0f passed, %.0f failed, %.0f on-hold", id, body.QtyPassed, body.QtyFailed, body.QtyOnHold))

	// Return updated record
	var updated models.ReceivingInspection
	var uia sql.NullString
	h.DB.QueryRow(`SELECT id, po_id, po_line_id, ipn, qty_received, qty_passed, qty_failed, qty_on_hold,
		COALESCE(inspector,''), inspected_at, COALESCE(notes,''), created_at
		FROM receiving_inspections WHERE id=?`, id).
		Scan(&updated.ID, &updated.POID, &updated.POLineID, &updated.IPN, &updated.QtyReceived, &updated.QtyPassed, &updated.QtyFailed, &updated.QtyOnHold,
			&updated.Inspector, &uia, &updated.Notes, &updated.CreatedAt)
	updated.InspectedAt = database.SP(uia)
	response.JSON(w, updated)
}

// WhereUsed returns all assemblies that contain a given IPN in their BOM.
func (h *Handler) WhereUsed(w http.ResponseWriter, r *http.Request, ipn string) {
	type WhereUsedEntry struct {
		AssemblyIPN string  `json:"assembly_ipn"`
		Description string  `json:"description"`
		Qty         float64 `json:"qty"`
		Ref         string  `json:"ref"`
	}

	cats, _, _, _ := h.LoadPartsFromDir()

	// Collect all assembly IPNs (PCA- or ASY- prefixed parts)
	var assemblyIPNs []string
	for _, parts := range cats {
		for _, p := range parts {
			upper := strings.ToUpper(p.IPN)
			if strings.HasPrefix(upper, "PCA-") || strings.HasPrefix(upper, "ASY-") {
				assemblyIPNs = append(assemblyIPNs, p.IPN)
			}
		}
	}

	var results []WhereUsedEntry

	for _, asmIPN := range assemblyIPNs {
		// Find BOM file for this assembly
		bomPaths := []string{filepath.Join(h.PartsDir, asmIPN+".csv")}
		entries, _ := os.ReadDir(h.PartsDir)
		for _, e := range entries {
			if e.IsDir() {
				bomPaths = append(bomPaths, filepath.Join(h.PartsDir, e.Name(), asmIPN+".csv"))
			}
		}

		var bomFile string
		for _, p := range bomPaths {
			if _, err := os.Stat(p); err == nil {
				bomFile = p
				break
			}
		}
		if bomFile == "" {
			continue
		}

		f, err := os.Open(bomFile)
		if err != nil {
			continue
		}

		rdr := csv.NewReader(f)
		rdr.LazyQuotes = true
		rdr.TrimLeadingSpace = true
		records, err := rdr.ReadAll()
		f.Close()
		if err != nil || len(records) < 2 {
			continue
		}

		headers := records[0]
		ipnIdx, qtyIdx, refIdx := -1, -1, -1
		for i, hdr := range headers {
			hl := strings.ToLower(hdr)
			switch {
			case hl == "ipn" || hl == "part_number" || hl == "pn":
				ipnIdx = i
			case hl == "qty" || hl == "quantity":
				qtyIdx = i
			case hl == "ref" || hl == "reference" || hl == "designator" || hl == "ref_des":
				refIdx = i
			}
		}
		if ipnIdx == -1 {
			ipnIdx = 0
		}

		for _, row := range records[1:] {
			if ipnIdx >= len(row) {
				continue
			}
			childIPN := strings.TrimSpace(row[ipnIdx])
			if !strings.EqualFold(childIPN, ipn) {
				continue
			}

			var qty float64 = 1
			if qtyIdx >= 0 && qtyIdx < len(row) {
				if q, err := strconv.ParseFloat(strings.TrimSpace(row[qtyIdx]), 64); err == nil {
					qty = q
				}
			}
			ref := ""
			if refIdx >= 0 && refIdx < len(row) {
				ref = strings.TrimSpace(row[refIdx])
			}

			// Get assembly description
			desc := ""
			if h.GetPartByIPN != nil {
				fields, _ := h.GetPartByIPN(h.PartsDir, asmIPN)
				if fields != nil {
					for k, v := range fields {
						if strings.EqualFold(k, "description") || strings.EqualFold(k, "desc") {
							desc = v
							break
						}
					}
				}
			}

			results = append(results, WhereUsedEntry{
				AssemblyIPN: asmIPN,
				Description: desc,
				Qty:         qty,
				Ref:         ref,
			})
			break // Found in this assembly, move to next
		}
	}

	if results == nil {
		results = []WhereUsedEntry{}
	}
	response.JSON(w, results)
}
