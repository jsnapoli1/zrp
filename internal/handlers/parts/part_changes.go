package parts

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zrp/internal/response"
)

// PartChange represents a pending field change for a part, gated by an ECO.
type PartChange struct {
	ID        int64  `json:"id"`
	PartIPN   string `json:"part_ipn"`
	ECOID     string `json:"eco_id,omitempty"`
	FieldName string `json:"field_name"`
	OldValue  string `json:"old_value"`
	NewValue  string `json:"new_value"`
	Status    string `json:"status"` // draft, pending, applied, rejected
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

// CreatePartChanges handles POST /api/parts/:ipn/changes.
func (h *Handler) CreatePartChanges(w http.ResponseWriter, r *http.Request, ipn string) {
	var body struct {
		Changes []struct {
			FieldName string `json:"field_name"`
			OldValue  string `json:"old_value"`
			NewValue  string `json:"new_value"`
		} `json:"changes"`
	}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	if len(body.Changes) == 0 {
		response.Err(w, "changes required", 400)
		return
	}

	// Verify part exists
	_, err := h.GetPartByIPN(h.PartsDir, ipn)
	if err != nil {
		response.Err(w, "part not found: "+ipn, 404)
		return
	}

	user := h.getUsername(r)
	now := time.Now().Format("2006-01-02 15:04:05")
	var created []PartChange

	for _, c := range body.Changes {
		if c.FieldName == "" {
			continue
		}
		res, err := h.DB.Exec(
			"INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status, created_by, created_at) VALUES (?,?,?,?,?,?,?)",
			ipn, c.FieldName, c.OldValue, c.NewValue, "draft", user, now,
		)
		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}
		id, _ := res.LastInsertId()
		created = append(created, PartChange{
			ID: id, PartIPN: ipn, FieldName: c.FieldName,
			OldValue: c.OldValue, NewValue: c.NewValue,
			Status: "draft", CreatedBy: user, CreatedAt: now,
		})
	}

	h.logAudit(user, "created", "part_changes", ipn, fmt.Sprintf("Created %d pending changes for %s", len(created), ipn))
	response.JSON(w, created)
}

// ListPartChanges handles GET /api/parts/:ipn/changes.
func (h *Handler) ListPartChanges(w http.ResponseWriter, r *http.Request, ipn string) {
	status := r.URL.Query().Get("status")
	query := "SELECT id, part_ipn, COALESCE(eco_id,''), field_name, old_value, new_value, status, created_by, created_at FROM part_changes WHERE part_ipn=?"
	args := []interface{}{ipn}
	if status != "" {
		query += " AND status=?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var items []PartChange
	for rows.Next() {
		var pc PartChange
		rows.Scan(&pc.ID, &pc.PartIPN, &pc.ECOID, &pc.FieldName, &pc.OldValue, &pc.NewValue, &pc.Status, &pc.CreatedBy, &pc.CreatedAt)
		items = append(items, pc)
	}
	if items == nil {
		items = []PartChange{}
	}
	response.JSON(w, items)
}

// DeletePartChange handles DELETE /api/parts/:ipn/changes/:changeID.
func (h *Handler) DeletePartChange(w http.ResponseWriter, r *http.Request, ipn string, changeID string) {
	id, err := strconv.ParseInt(changeID, 10, 64)
	if err != nil {
		response.Err(w, "invalid change id", 400)
		return
	}

	// Only allow deleting draft changes
	var status string
	err = h.DB.QueryRow("SELECT status FROM part_changes WHERE id=? AND part_ipn=?", id, ipn).Scan(&status)
	if err != nil {
		response.Err(w, "change not found", 404)
		return
	}
	if status != "draft" {
		response.Err(w, "can only delete draft changes", 400)
		return
	}

	h.DB.Exec("DELETE FROM part_changes WHERE id=?", id)
	h.logAudit(h.getUsername(r), "deleted", "part_changes", fmt.Sprintf("%s/%d", ipn, id), "Deleted pending change")
	response.JSON(w, map[string]string{"status": "deleted"})
}

// CreateECOFromChanges handles POST /api/parts/:ipn/changes/create-eco.
func (h *Handler) CreateECOFromChanges(w http.ResponseWriter, r *http.Request, ipn string) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	// Get all draft changes for this part
	rows, err := h.DB.Query("SELECT id, field_name, old_value, new_value FROM part_changes WHERE part_ipn=? AND status='draft'", ipn)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var changeIDs []int64
	var summaryParts []string
	for rows.Next() {
		var id int64
		var fn, ov, nv string
		rows.Scan(&id, &fn, &ov, &nv)
		changeIDs = append(changeIDs, id)
		summaryParts = append(summaryParts, fmt.Sprintf("%s: %q -> %q", fn, ov, nv))
	}

	if len(changeIDs) == 0 {
		response.Err(w, "no draft changes found for this part", 400)
		return
	}

	user := h.getUsername(r)
	now := time.Now().Format("2006-01-02 15:04:05")

	if body.Title == "" {
		body.Title = fmt.Sprintf("Part changes for %s", ipn)
	}
	if body.Priority == "" {
		body.Priority = "normal"
	}
	if body.Description == "" {
		body.Description = "Changes:\n" + strings.Join(summaryParts, "\n")
	}

	ecoID := h.NextID("ECO", "ecos", 3)
	ipnsJSON, _ := json.Marshal([]string{ipn})

	_, err = h.DB.Exec("INSERT INTO ecos (id,title,description,status,priority,affected_ipns,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)",
		ecoID, body.Title, body.Description, "draft", body.Priority, string(ipnsJSON), user, now, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	h.EnsureInitialRevision(ecoID, user, now)

	// Link changes to ECO and update status to pending
	for _, cid := range changeIDs {
		h.DB.Exec("UPDATE part_changes SET eco_id=?, status='pending' WHERE id=?", ecoID, cid)
	}

	h.logAudit(user, "created", "eco", ecoID, fmt.Sprintf("Created ECO from %d part changes for %s", len(changeIDs), ipn))
	response.JSON(w, map[string]interface{}{
		"eco_id":        ecoID,
		"changes_count": len(changeIDs),
	})
}

// ListECOPartChanges handles GET /api/ecos/:id/part-changes.
func (h *Handler) ListECOPartChanges(w http.ResponseWriter, r *http.Request, ecoID string) {
	rows, err := h.DB.Query("SELECT id, part_ipn, COALESCE(eco_id,''), field_name, old_value, new_value, status, created_by, created_at FROM part_changes WHERE eco_id=? ORDER BY part_ipn, field_name", ecoID)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var items []PartChange
	for rows.Next() {
		var pc PartChange
		rows.Scan(&pc.ID, &pc.PartIPN, &pc.ECOID, &pc.FieldName, &pc.OldValue, &pc.NewValue, &pc.Status, &pc.CreatedBy, &pc.CreatedAt)
		items = append(items, pc)
	}
	if items == nil {
		items = []PartChange{}
	}
	response.JSON(w, items)
}

// ApplyPartChangesForECO is called when an ECO is implemented. It applies all
// pending part_changes linked to the ECO by updating the CSV files on disk.
func (h *Handler) ApplyPartChangesForECO(ecoID string) error {
	rows, err := h.DB.Query("SELECT id, part_ipn, field_name, new_value FROM part_changes WHERE eco_id=? AND status='pending'", ecoID)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Group changes by part IPN
	changesByIPN := make(map[string][]partFieldChange)
	for rows.Next() {
		var id int64
		var ipn, fn, nv string
		rows.Scan(&id, &ipn, &fn, &nv)
		changesByIPN[ipn] = append(changesByIPN[ipn], partFieldChange{id: id, field: fn, newValue: nv})
	}

	for ipn, changes := range changesByIPN {
		if err := h.applyChangesToCSV(ipn, changes); err != nil {
			// Mark as failed but continue
			for _, c := range changes {
				h.DB.Exec("UPDATE part_changes SET status='rejected' WHERE id=?", c.id)
			}
			continue
		}
		for _, c := range changes {
			h.DB.Exec("UPDATE part_changes SET status='applied' WHERE id=?", c.id)
		}
	}
	return nil
}

// RejectPartChangesForECO marks all pending changes linked to an ECO as rejected.
func (h *Handler) RejectPartChangesForECO(ecoID string) {
	h.DB.Exec("UPDATE part_changes SET status='rejected' WHERE eco_id=? AND status='pending'", ecoID)
}

type partFieldChange struct {
	id       int64
	field    string
	newValue string
}

func (h *Handler) applyChangesToCSV(ipn string, changes []partFieldChange) error {
	if h.PartsDir == "" {
		return fmt.Errorf("no parts directory configured")
	}

	// Find which CSV file contains this IPN
	csvPath, rowIdx, headers, records, err := h.findPartInCSV(ipn)
	if err != nil {
		return err
	}

	// Build header index
	headerIdx := make(map[string]int)
	for i, hdr := range headers {
		headerIdx[hdr] = i
		headerIdx[strings.ToLower(hdr)] = i
	}

	// Apply changes to the row
	row := records[rowIdx]
	for _, c := range changes {
		idx, ok := headerIdx[c.field]
		if !ok {
			idx, ok = headerIdx[strings.ToLower(c.field)]
		}
		if !ok {
			continue // field not found in CSV, skip
		}
		if idx < len(row) {
			row[idx] = c.newValue
		}
	}
	records[rowIdx] = row

	// Write back
	return writePartCSV(csvPath, records)
}

func (h *Handler) findPartInCSV(ipn string) (string, int, []string, [][]string, error) {
	entries, err := os.ReadDir(h.PartsDir)
	if err != nil {
		return "", 0, nil, nil, err
	}

	// Check all CSV files
	var csvPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			catDir := filepath.Join(h.PartsDir, entry.Name())
			csvFiles, _ := filepath.Glob(filepath.Join(catDir, "*.csv"))
			csvPaths = append(csvPaths, csvFiles...)
		} else if strings.HasSuffix(entry.Name(), ".csv") {
			csvPaths = append(csvPaths, filepath.Join(h.PartsDir, entry.Name()))
		}
	}

	for _, csvPath := range csvPaths {
		f, err := os.Open(csvPath)
		if err != nil {
			continue
		}
		rd := csv.NewReader(f)
		rd.LazyQuotes = true
		rd.TrimLeadingSpace = true
		records, err := rd.ReadAll()
		f.Close()
		if err != nil || len(records) < 2 {
			continue
		}

		headers := records[0]
		// Find IPN column
		ipnCol := -1
		for i, hdr := range headers {
			hl := strings.ToLower(hdr)
			if hl == "ipn" || hl == "part_number" || hl == "pn" {
				ipnCol = i
				break
			}
		}
		if ipnCol == -1 {
			ipnCol = 0
		}

		for rowIdx := 1; rowIdx < len(records); rowIdx++ {
			if ipnCol < len(records[rowIdx]) && records[rowIdx][ipnCol] == ipn {
				return csvPath, rowIdx, headers, records, nil
			}
		}
	}

	return "", 0, nil, nil, fmt.Errorf("part %s not found in any CSV", ipn)
}

func writePartCSV(path string, records [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	err = w.WriteAll(records)
	if err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

// ListAllPartChanges handles GET /api/part-changes.
func (h *Handler) ListAllPartChanges(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "draft"
	}
	rows, err := h.DB.Query("SELECT id, part_ipn, COALESCE(eco_id,''), field_name, old_value, new_value, status, created_by, created_at FROM part_changes WHERE status=? ORDER BY created_at DESC", status)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var items []PartChange
	for rows.Next() {
		var pc PartChange
		rows.Scan(&pc.ID, &pc.PartIPN, &pc.ECOID, &pc.FieldName, &pc.OldValue, &pc.NewValue, &pc.Status, &pc.CreatedBy, &pc.CreatedAt)
		items = append(items, pc)
	}
	if items == nil {
		items = []PartChange{}
	}
	response.JSON(w, items)
}

// UpdateBOMReferencesForPartIPN updates all BOM CSV files to reference a new IPN
// when a part IPN has changed.
func (h *Handler) UpdateBOMReferencesForPartIPN(partsDir, oldIPN, newIPN string) error {
	if partsDir == "" {
		return fmt.Errorf("no parts directory configured")
	}

	// Find all CSV files that might contain BOMs
	entries, err := os.ReadDir(partsDir)
	if err != nil {
		return err
	}

	var bomFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check for BOM files in subdirectories
			subDir := filepath.Join(partsDir, entry.Name())
			csvFiles, _ := filepath.Glob(filepath.Join(subDir, "*.csv"))

			// Filter for assembly/BOM files (typically PCA-*, ASY-*, or in assemblies/ dir)
			for _, csvFile := range csvFiles {
				if strings.Contains(entry.Name(), "assembl") ||
					strings.Contains(filepath.Base(csvFile), "PCA-") ||
					strings.Contains(filepath.Base(csvFile), "ASY-") {
					bomFiles = append(bomFiles, csvFile)
				}
			}
		}
	}

	// Update each BOM file
	for _, bomPath := range bomFiles {
		if err := updateIPNInBOMFile(bomPath, oldIPN, newIPN); err != nil {
			return fmt.Errorf("failed to update BOM %s: %w", bomPath, err)
		}
	}

	return nil
}

func updateIPNInBOMFile(bomPath, oldIPN, newIPN string) error {
	f, err := os.Open(bomPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	if len(records) < 2 {
		return nil // Empty BOM, nothing to update
	}

	// Find IPN column
	headers := records[0]
	ipnIdx := -1
	for i, hdr := range headers {
		if strings.EqualFold(hdr, "IPN") || strings.EqualFold(hdr, "Part Number") {
			ipnIdx = i
			break
		}
	}

	if ipnIdx == -1 {
		return nil // No IPN column found
	}

	// Update all matching IPNs
	updated := false
	for i := 1; i < len(records); i++ {
		if ipnIdx < len(records[i]) && records[i][ipnIdx] == oldIPN {
			records[i][ipnIdx] = newIPN
			updated = true
		}
	}

	if !updated {
		return nil // No changes needed
	}

	// Write back to file
	f2, err := os.Create(bomPath)
	if err != nil {
		return err
	}
	defer f2.Close()

	writer := csv.NewWriter(f2)
	defer writer.Flush()

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
