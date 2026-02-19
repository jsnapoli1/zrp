package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// ChangeEntry represents a row in change_history
type ChangeEntry struct {
	ID        int    `json:"id"`
	TableName string `json:"table_name"`
	RecordID  string `json:"record_id"`
	Operation string `json:"operation"`
	OldData   string `json:"old_data"`
	NewData   string `json:"new_data"`
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
	Undone    int    `json:"undone"`
}

// recordChange logs a mutation to change_history for undo/redo support.
// oldData/newData should be JSON strings (or empty for create/delete).
func recordChange(userID, tableName, recordID, operation, oldData, newData string) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO change_history (table_name, record_id, operation, old_data, new_data, user_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		tableName, recordID, operation, oldData, newData, userID,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()

	// Broadcast change event for real-time UI
	wsHub.Broadcast(WSEvent{
		Type:   "change_recorded",
		ID:     fmt.Sprintf("%d", id),
		Action: operation,
	})

	return id, nil
}

// recordChangeJSON marshals old/new data as JSON before recording.
func recordChangeJSON(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error) {
	var oldStr, newStr string
	if oldData != nil {
		b, _ := json.Marshal(oldData)
		oldStr = string(b)
	}
	if newData != nil {
		b, _ := json.Marshal(newData)
		newStr = string(b)
	}
	return recordChange(userID, tableName, recordID, operation, oldStr, newStr)
}

// handleRecentChanges returns the last 50 changes for the current user
func handleRecentChanges(w http.ResponseWriter, r *http.Request) {
	username := getUsername(r)
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	rows, err := db.Query(
		`SELECT id, table_name, record_id, operation, COALESCE(old_data,''), COALESCE(new_data,''), user_id, created_at, undone
		 FROM change_history WHERE user_id = ?
		 ORDER BY created_at DESC LIMIT ?`,
		username, limit,
	)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var entries []ChangeEntry
	for rows.Next() {
		var e ChangeEntry
		rows.Scan(&e.ID, &e.TableName, &e.RecordID, &e.Operation, &e.OldData, &e.NewData, &e.UserID, &e.CreatedAt, &e.Undone)
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []ChangeEntry{}
	}
	jsonResp(w, entries)
}

// handleUndoChange reverses a change_history entry
func handleUndoChange(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonErr(w, "invalid change id", 400)
		return
	}

	username := getUsername(r)
	var entry ChangeEntry
	err = db.QueryRow(
		`SELECT id, table_name, record_id, operation, COALESCE(old_data,''), COALESCE(new_data,''), user_id, created_at, undone
		 FROM change_history WHERE id = ? AND user_id = ?`,
		id, username,
	).Scan(&entry.ID, &entry.TableName, &entry.RecordID, &entry.Operation, &entry.OldData, &entry.NewData, &entry.UserID, &entry.CreatedAt, &entry.Undone)
	if err != nil {
		jsonErr(w, "change not found", 404)
		return
	}

	if entry.Undone == 1 {
		jsonErr(w, "change already undone", 400)
		return
	}

	// Perform the reversal based on operation type
	switch entry.Operation {
	case "create":
		// Undo create = delete the record
		if err := deleteByTable(entry.TableName, entry.RecordID); err != nil {
			jsonErr(w, fmt.Sprintf("undo failed: %v", err), 500)
			return
		}
	case "update":
		// Undo update = restore old_data
		if err := restoreByTable(entry.TableName, entry.RecordID, entry.OldData); err != nil {
			jsonErr(w, fmt.Sprintf("undo failed: %v", err), 500)
			return
		}
	case "delete":
		// Undo delete = re-insert from old_data
		if err := restoreByTable(entry.TableName, entry.RecordID, entry.OldData); err != nil {
			jsonErr(w, fmt.Sprintf("undo failed: %v", err), 500)
			return
		}
	default:
		jsonErr(w, "unsupported operation", 400)
		return
	}

	// Mark as undone
	db.Exec("UPDATE change_history SET undone = 1 WHERE id = ?", id)

	// Record the undo itself as a new change (enables redo)
	var reverseOp string
	switch entry.Operation {
	case "create":
		reverseOp = "delete"
	case "delete":
		reverseOp = "create"
	case "update":
		reverseOp = "update"
	}
	// Swap old/new for the undo record
	redoID, _ := recordChange(username, entry.TableName, entry.RecordID, reverseOp, entry.NewData, entry.OldData)

	logAudit(db, username, "undo", entry.TableName, entry.RecordID,
		fmt.Sprintf("Undid %s on %s %s", entry.Operation, entry.TableName, entry.RecordID))

	jsonResp(w, map[string]interface{}{
		"status":     "undone",
		"table_name": entry.TableName,
		"record_id":  entry.RecordID,
		"operation":  entry.Operation,
		"redo_id":    redoID,
	})
}

// deleteByTable deletes a record by table name and ID
func deleteByTable(tableName, recordID string) error {
	idCol := tableIDColumn(tableName)
	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tableName, idCol), recordID)
	return err
}

// restoreByTable restores a record from JSON data using INSERT OR REPLACE
func restoreByTable(tableName, recordID, jsonData string) error {
	// Use entity-specific restore functions where available
	switch tableName {
	case "ecos":
		return restoreECO(jsonData)
	case "work_orders":
		return restoreWorkOrder(jsonData)
	case "ncrs":
		return restoreNCR(jsonData)
	case "devices":
		return restoreDevice(jsonData)
	case "inventory":
		return restoreInventory(jsonData)
	case "rmas":
		return restoreRMA(jsonData)
	case "vendors":
		return restoreVendor(jsonData)
	case "quotes":
		return restoreQuote(jsonData)
	case "purchase_orders":
		return restorePO(jsonData)
	default:
		return genericRestore(tableName, jsonData)
	}
}

// genericRestore does INSERT OR REPLACE from a JSON map for tables without specific restore functions
func genericRestore(tableName, jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return err
	}
	if len(m) == 0 {
		return fmt.Errorf("empty data")
	}

	cols := make([]string, 0, len(m))
	vals := make([]interface{}, 0, len(m))
	placeholders := make([]string, 0, len(m))
	for k, v := range m {
		cols = append(cols, k)
		vals = append(vals, v)
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
		tableName,
		joinStrings(cols, ", "),
		joinStrings(placeholders, ", "),
	)
	_, err := db.Exec(query, vals...)
	return err
}

func joinStrings(s []string, sep string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += sep
		}
		result += v
	}
	return result
}

// tableIDColumn returns the primary key column name for a table
func tableIDColumn(tableName string) string {
	switch tableName {
	case "devices":
		return "serial_number"
	case "inventory":
		return "ipn"
	default:
		return "id"
	}
}
