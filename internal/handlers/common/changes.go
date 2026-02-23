package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ChangeEntry represents a row in change_history.
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

// RecordChange logs a mutation to change_history.
func (h *Handler) RecordChange(userID, tableName, recordID, operation, oldData, newData string) (int64, error) {
	res, err := h.DB.Exec(
		`INSERT INTO change_history (table_name, record_id, operation, old_data, new_data, user_id)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		tableName, recordID, operation, oldData, newData, userID,
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()

	h.Broadcast("change_recorded", fmt.Sprintf("%d", id), operation)

	return id, nil
}

// RecordChangeJSON marshals old/new data as JSON before recording.
func (h *Handler) RecordChangeJSON(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error) {
	var oldStr, newStr string
	if oldData != nil {
		b, _ := json.Marshal(oldData)
		oldStr = string(b)
	}
	if newData != nil {
		b, _ := json.Marshal(newData)
		newStr = string(b)
	}
	return h.RecordChange(userID, tableName, recordID, operation, oldStr, newStr)
}

// RecentChanges returns the last N changes for the current user.
func (h *Handler) RecentChanges(w http.ResponseWriter, r *http.Request) {
	username := h.GetUsername(r)
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	rows, err := h.DB.Query(
		`SELECT id, table_name, record_id, operation, COALESCE(old_data,''), COALESCE(new_data,''), user_id, created_at, undone
		 FROM change_history WHERE user_id = ?
		 ORDER BY created_at DESC LIMIT ?`,
		username, limit,
	)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// UndoChange reverses a change_history entry.
func (h *Handler) UndoChange(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid change id"}`, 400)
		return
	}

	username := h.GetUsername(r)
	var entry ChangeEntry
	err = h.DB.QueryRow(
		`SELECT id, table_name, record_id, operation, COALESCE(old_data,''), COALESCE(new_data,''), user_id, created_at, undone
		 FROM change_history WHERE id = ? AND user_id = ?`,
		id, username,
	).Scan(&entry.ID, &entry.TableName, &entry.RecordID, &entry.Operation, &entry.OldData, &entry.NewData, &entry.UserID, &entry.CreatedAt, &entry.Undone)
	if err != nil {
		http.Error(w, `{"error":"change not found"}`, 404)
		return
	}

	if entry.Undone == 1 {
		http.Error(w, `{"error":"change already undone"}`, 400)
		return
	}

	switch entry.Operation {
	case "create":
		if err := h.DeleteByTable(entry.TableName, entry.RecordID); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"undo failed: %v"}`, err), 500)
			return
		}
	case "update":
		if err := h.RestoreByTable(entry.TableName, entry.RecordID, entry.OldData); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"undo failed: %v"}`, err), 500)
			return
		}
	case "delete":
		if err := h.RestoreByTable(entry.TableName, entry.RecordID, entry.OldData); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"undo failed: %v"}`, err), 500)
			return
		}
	default:
		http.Error(w, `{"error":"unsupported operation"}`, 400)
		return
	}

	h.DB.Exec("UPDATE change_history SET undone = 1 WHERE id = ?", id)

	var reverseOp string
	switch entry.Operation {
	case "create":
		reverseOp = "delete"
	case "delete":
		reverseOp = "create"
	case "update":
		reverseOp = "update"
	}
	redoID, _ := h.RecordChange(username, entry.TableName, entry.RecordID, reverseOp, entry.NewData, entry.OldData)

	h.LogAudit(h.DB, username, "undo", entry.TableName, entry.RecordID,
		fmt.Sprintf("Undid %s on %s %s", entry.Operation, entry.TableName, entry.RecordID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "undone", "table_name": entry.TableName,
		"record_id": entry.RecordID, "operation": entry.Operation, "redo_id": redoID,
	})
}

// DeleteByTable deletes a record by table name and ID.
func (h *Handler) DeleteByTable(tableName, recordID string) error {
	validatedTable, err := h.ValidateAndSanitizeTable(tableName)
	if err != nil {
		return fmt.Errorf("invalid table name: %v", err)
	}
	idCol := TableIDColumn(validatedTable)
	validatedCol, err := h.ValidateAndSanitizeColumn(idCol)
	if err != nil {
		return fmt.Errorf("invalid column name: %v", err)
	}
	_, err = h.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", validatedTable, validatedCol), recordID)
	return err
}

// RestoreByTable restores a record from JSON data.
func (h *Handler) RestoreByTable(tableName, recordID, jsonData string) error {
	switch tableName {
	case "ecos":
		return h.restoreECO(jsonData)
	case "work_orders":
		return h.restoreWorkOrder(jsonData)
	case "ncrs":
		return h.restoreNCR(jsonData)
	case "devices":
		return h.restoreDevice(jsonData)
	case "inventory":
		return h.restoreInventory(jsonData)
	case "rmas":
		return h.restoreRMA(jsonData)
	case "vendors":
		return h.restoreVendor(jsonData)
	case "quotes":
		return h.restoreQuote(jsonData)
	case "purchase_orders":
		return h.restorePO(jsonData)
	default:
		return h.GenericRestore(tableName, jsonData)
	}
}

// GenericRestore does INSERT OR REPLACE from a JSON map.
func (h *Handler) GenericRestore(tableName, jsonData string) error {
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

	validatedTable, err := h.ValidateAndSanitizeTable(tableName)
	if err != nil {
		return fmt.Errorf("invalid table name: %v", err)
	}

	for _, col := range cols {
		if _, err := h.ValidateAndSanitizeColumn(col); err != nil {
			return fmt.Errorf("invalid column name '%s': %v", col, err)
		}
	}

	query := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
		validatedTable,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)
	_, err = h.DB.Exec(query, vals...)
	return err
}

// TableIDColumn returns the primary key column name for a table.
func TableIDColumn(tableName string) string {
	switch tableName {
	case "devices":
		return "serial_number"
	case "inventory":
		return "ipn"
	default:
		return "id"
	}
}
