package main

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
)

// PartChange represents a pending field change for a part, gated by an ECO
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

func handleCreatePartChanges(w http.ResponseWriter, r *http.Request, ipn string) {
	var body struct {
		Changes []struct {
			FieldName string `json:"field_name"`
			OldValue  string `json:"old_value"`
			NewValue  string `json:"new_value"`
		} `json:"changes"`
	}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}
	if len(body.Changes) == 0 {
		jsonErr(w, "changes required", 400)
		return
	}

	// Verify part exists
	_, err := getPartByIPN(partsDir, ipn)
	if err != nil {
		jsonErr(w, "part not found: "+ipn, 404)
		return
	}

	user := getUsername(r)
	now := time.Now().Format("2006-01-02 15:04:05")
	var created []PartChange

	for _, c := range body.Changes {
		if c.FieldName == "" {
			continue
		}
		res, err := db.Exec(
			"INSERT INTO part_changes (part_ipn, field_name, old_value, new_value, status, created_by, created_at) VALUES (?,?,?,?,?,?,?)",
			ipn, c.FieldName, c.OldValue, c.NewValue, "draft", user, now,
		)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		id, _ := res.LastInsertId()
		created = append(created, PartChange{
			ID: id, PartIPN: ipn, FieldName: c.FieldName,
			OldValue: c.OldValue, NewValue: c.NewValue,
			Status: "draft", CreatedBy: user, CreatedAt: now,
		})
	}

	logAudit(db, user, "created", "part_changes", ipn, fmt.Sprintf("Created %d pending changes for %s", len(created), ipn))
	jsonResp(w, created)
}

func handleListPartChanges(w http.ResponseWriter, r *http.Request, ipn string) {
	status := r.URL.Query().Get("status")
	query := "SELECT id, part_ipn, COALESCE(eco_id,''), field_name, old_value, new_value, status, created_by, created_at FROM part_changes WHERE part_ipn=?"
	args := []interface{}{ipn}
	if status != "" {
		query += " AND status=?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500)
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
	jsonResp(w, items)
}

func handleDeletePartChange(w http.ResponseWriter, r *http.Request, ipn string, changeID string) {
	id, err := strconv.ParseInt(changeID, 10, 64)
	if err != nil {
		jsonErr(w, "invalid change id", 400)
		return
	}

	// Only allow deleting draft changes
	var status string
	err = db.QueryRow("SELECT status FROM part_changes WHERE id=? AND part_ipn=?", id, ipn).Scan(&status)
	if err != nil {
		jsonErr(w, "change not found", 404)
		return
	}
	if status != "draft" {
		jsonErr(w, "can only delete draft changes", 400)
		return
	}

	db.Exec("DELETE FROM part_changes WHERE id=?", id)
	logAudit(db, getUsername(r), "deleted", "part_changes", fmt.Sprintf("%s/%d", ipn, id), "Deleted pending change")
	jsonResp(w, map[string]string{"status": "deleted"})
}

func handleCreateECOFromChanges(w http.ResponseWriter, r *http.Request, ipn string) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
	}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	// Get all draft changes for this part
	rows, err := db.Query("SELECT id, field_name, old_value, new_value FROM part_changes WHERE part_ipn=? AND status='draft'", ipn)
	if err != nil {
		jsonErr(w, err.Error(), 500)
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
		summaryParts = append(summaryParts, fmt.Sprintf("%s: %q → %q", fn, ov, nv))
	}

	if len(changeIDs) == 0 {
		jsonErr(w, "no draft changes found for this part", 400)
		return
	}

	user := getUsername(r)
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

	ecoID := nextID("ECO", "ecos", 3)
	ipnsJSON, _ := json.Marshal([]string{ipn})

	_, err = db.Exec("INSERT INTO ecos (id,title,description,status,priority,affected_ipns,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)",
		ecoID, body.Title, body.Description, "draft", body.Priority, string(ipnsJSON), user, now, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	ensureInitialRevision(ecoID, user, now)

	// Link changes to ECO and update status to pending
	for _, cid := range changeIDs {
		db.Exec("UPDATE part_changes SET eco_id=?, status='pending' WHERE id=?", ecoID, cid)
	}

	logAudit(db, user, "created", "eco", ecoID, fmt.Sprintf("Created ECO from %d part changes for %s", len(changeIDs), ipn))
	jsonResp(w, map[string]interface{}{
		"eco_id":        ecoID,
		"changes_count": len(changeIDs),
	})
}

// handleListECOPartChanges returns all part changes linked to an ECO
func handleListECOPartChanges(w http.ResponseWriter, r *http.Request, ecoID string) {
	rows, err := db.Query("SELECT id, part_ipn, COALESCE(eco_id,''), field_name, old_value, new_value, status, created_by, created_at FROM part_changes WHERE eco_id=? ORDER BY part_ipn, field_name", ecoID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
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
	jsonResp(w, items)
}

// applyPartChangesForECO is called when an ECO is implemented. It applies all
// pending part_changes linked to the ECO by updating the CSV files on disk.
func applyPartChangesForECO(ecoID string) error {
	rows, err := db.Query("SELECT id, part_ipn, field_name, new_value FROM part_changes WHERE eco_id=? AND status='pending'", ecoID)
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
		if err := applyChangesToCSV(ipn, changes); err != nil {
			// Mark as failed but continue
			for _, c := range changes {
				db.Exec("UPDATE part_changes SET status='rejected' WHERE id=?", c.id)
			}
			continue
		}
		for _, c := range changes {
			db.Exec("UPDATE part_changes SET status='applied' WHERE id=?", c.id)
		}
	}
	return nil
}

// rejectPartChangesForECO marks all pending changes linked to an ECO as rejected
func rejectPartChangesForECO(ecoID string) {
	db.Exec("UPDATE part_changes SET status='rejected' WHERE eco_id=? AND status='pending'", ecoID)
}

type partFieldChange struct {
	id       int64
	field    string
	newValue string
}

func applyChangesToCSV(ipn string, changes []partFieldChange) error {
	if partsDir == "" {
		return fmt.Errorf("no parts directory configured")
	}

	// Find which CSV file contains this IPN
	csvPath, rowIdx, headers, records, err := findPartInCSV(ipn)
	if err != nil {
		return err
	}

	// Build header index
	headerIdx := make(map[string]int)
	for i, h := range headers {
		headerIdx[h] = i
		headerIdx[strings.ToLower(h)] = i
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

func findPartInCSV(ipn string) (string, int, []string, [][]string, error) {
	entries, err := os.ReadDir(partsDir)
	if err != nil {
		return "", 0, nil, nil, err
	}

	// Check all CSV files
	var csvPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			catDir := filepath.Join(partsDir, entry.Name())
			csvFiles, _ := filepath.Glob(filepath.Join(catDir, "*.csv"))
			csvPaths = append(csvPaths, csvFiles...)
		} else if strings.HasSuffix(entry.Name(), ".csv") {
			csvPaths = append(csvPaths, filepath.Join(partsDir, entry.Name()))
		}
	}

	for _, csvPath := range csvPaths {
		f, err := os.Open(csvPath)
		if err != nil {
			continue
		}
		r := csv.NewReader(f)
		r.LazyQuotes = true
		r.TrimLeadingSpace = true
		records, err := r.ReadAll()
		f.Close()
		if err != nil || len(records) < 2 {
			continue
		}

		headers := records[0]
		// Find IPN column
		ipnCol := -1
		for i, h := range headers {
			hl := strings.ToLower(h)
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

// handleListAllPartChanges returns all pending part changes across all parts (for dashboard/indicators)
func handleListAllPartChanges(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "draft"
	}
	rows, err := db.Query("SELECT id, part_ipn, COALESCE(eco_id,''), field_name, old_value, new_value, status, created_by, created_at FROM part_changes WHERE status=? ORDER BY created_at DESC", status)
	if err != nil {
		jsonErr(w, err.Error(), 500)
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
	jsonResp(w, items)
}

// updateBOMReferencesForPartIPN updates all BOM CSV files to reference a new IPN
// when a part IPN has changed. This is a cascade operation to maintain BOM integrity.
// Note: In typical PLM workflows, part revisions are stored as fields within the part
// record, so BOMs reference the IPN and automatically get the latest revision.
// This function is for edge cases where the IPN itself needs to change.
func updateBOMReferencesForPartIPN(partsDir, oldIPN, newIPN string) error {
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

// updateIPNInBOMFile updates a single BOM CSV file to replace oldIPN with newIPN
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
	for i, h := range headers {
		if strings.EqualFold(h, "IPN") || strings.EqualFold(h, "Part Number") {
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
