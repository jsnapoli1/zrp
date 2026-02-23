package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// UndoLogEntry represents a stored undo action.
type UndoLogEntry struct {
	ID           int    `json:"id"`
	UserID       string `json:"user_id"`
	Action       string `json:"action"`
	EntityType   string `json:"entity_type"`
	EntityID     string `json:"entity_id"`
	PreviousData string `json:"previous_data"`
	CreatedAt    string `json:"created_at"`
	ExpiresAt    string `json:"expires_at"`
}

// SnapshotEntity captures the current state of an entity before destructive action.
func (h *Handler) SnapshotEntity(entityType, entityID string) (string, error) {
	var data interface{}
	var err error

	switch entityType {
	case "eco":
		data, err = h.GetRowAsMap("SELECT * FROM ecos WHERE id=?", entityID)
	case "workorder":
		data, err = h.GetRowAsMap("SELECT * FROM work_orders WHERE id=?", entityID)
	case "ncr":
		data, err = h.GetRowAsMap("SELECT * FROM ncrs WHERE id=?", entityID)
	case "device":
		data, err = h.GetRowAsMap("SELECT * FROM devices WHERE serial_number=?", entityID)
	case "inventory":
		data, err = h.GetRowAsMap("SELECT * FROM inventory WHERE ipn=?", entityID)
	case "rma":
		data, err = h.GetRowAsMap("SELECT * FROM rmas WHERE id=?", entityID)
	case "vendor":
		data, err = h.GetRowAsMap("SELECT * FROM vendors WHERE id=?", entityID)
	case "quote":
		data, err = h.getQuoteSnapshot(entityID)
	case "po":
		data, err = h.getPOSnapshot(entityID)
	default:
		return "", fmt.Errorf("unsupported entity type: %s", entityType)
	}

	if err != nil {
		return "", err
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CreateUndoEntry snapshots an entity and inserts into undo_log.
func (h *Handler) CreateUndoEntry(username, action, entityType, entityID string) (int64, error) {
	snapshot, err := h.SnapshotEntity(entityType, entityID)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	res, err := h.DB.Exec(
		`INSERT INTO undo_log (user_id, action, entity_type, entity_id, previous_data, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		username, action, entityType, entityID, snapshot,
		now.Format("2006-01-02 15:04:05"),
		expires.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// PerformUndo restores an entity from a snapshot.
func (h *Handler) PerformUndo(entry UndoLogEntry) error {
	switch entry.EntityType {
	case "eco":
		return h.restoreECO(entry.PreviousData)
	case "workorder":
		return h.restoreWorkOrder(entry.PreviousData)
	case "ncr":
		return h.restoreNCR(entry.PreviousData)
	case "device":
		return h.restoreDevice(entry.PreviousData)
	case "inventory":
		return h.restoreInventory(entry.PreviousData)
	case "rma":
		return h.restoreRMA(entry.PreviousData)
	case "vendor":
		return h.restoreVendor(entry.PreviousData)
	case "quote":
		return h.restoreQuote(entry.PreviousData)
	case "po":
		return h.restorePO(entry.PreviousData)
	default:
		return fmt.Errorf("unsupported entity type: %s", entry.EntityType)
	}
}

// ListUndo returns recent undo entries for the current user.
func (h *Handler) ListUndo(w http.ResponseWriter, r *http.Request) {
	username := h.GetUsername(r)
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	rows, err := h.DB.Query(
		`SELECT id, user_id, action, entity_type, entity_id, previous_data, created_at, expires_at
		 FROM undo_log WHERE user_id = ? AND expires_at > CURRENT_TIMESTAMP
		 ORDER BY created_at DESC LIMIT ?`,
		username, limit,
	)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}
	defer rows.Close()

	var entries []UndoLogEntry
	for rows.Next() {
		var e UndoLogEntry
		rows.Scan(&e.ID, &e.UserID, &e.Action, &e.EntityType, &e.EntityID, &e.PreviousData, &e.CreatedAt, &e.ExpiresAt)
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []UndoLogEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// HandlePerformUndo handles the undo HTTP request.
func (h *Handler) HandlePerformUndo(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid undo id"}`, 400)
		return
	}

	username := h.GetUsername(r)
	var entry UndoLogEntry
	err = h.DB.QueryRow(
		`SELECT id, user_id, action, entity_type, entity_id, previous_data, created_at, expires_at
		 FROM undo_log WHERE id = ? AND user_id = ? AND expires_at > CURRENT_TIMESTAMP`,
		id, username,
	).Scan(&entry.ID, &entry.UserID, &entry.Action, &entry.EntityType, &entry.EntityID, &entry.PreviousData, &entry.CreatedAt, &entry.ExpiresAt)
	if err != nil {
		http.Error(w, `{"error":"undo entry not found or expired"}`, 404)
		return
	}

	if err := h.PerformUndo(entry); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"undo failed: %v"}`, err), 500)
		return
	}

	h.DB.Exec("DELETE FROM undo_log WHERE id = ?", id)

	h.LogAudit(h.DB, username, "undo", entry.EntityType, entry.EntityID,
		fmt.Sprintf("Undid %s on %s %s", entry.Action, entry.EntityType, entry.EntityID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "restored", "entity_type": entry.EntityType, "entity_id": entry.EntityID,
	})
}

// CleanExpiredUndo removes expired undo entries periodically.
func (h *Handler) CleanExpiredUndo() {
	for {
		time.Sleep(1 * time.Hour)
		h.DB.Exec("DELETE FROM undo_log WHERE expires_at < CURRENT_TIMESTAMP")
	}
}

// GetRowAsMap runs a query expecting one row and returns it as a map.
func (h *Handler) GetRowAsMap(query string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, fmt.Errorf("entity not found")
	}

	values := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range values {
		ptrs[i] = &values[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for i, col := range cols {
		v := values[i]
		if b, ok := v.([]byte); ok {
			result[col] = string(b)
		} else {
			result[col] = v
		}
	}
	return result, nil
}

// GetRowsAsMapSlice runs a query and returns all rows as a slice of maps.
func (h *Handler) GetRowsAsMapSlice(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			v := values[i]
			if b, ok := v.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = v
			}
		}
		results = append(results, row)
	}
	return results, nil
}

func (h *Handler) getQuoteSnapshot(id string) (map[string]interface{}, error) {
	row, err := h.GetRowAsMap("SELECT * FROM quotes WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	lines, err := h.GetRowsAsMapSlice("SELECT * FROM quote_lines WHERE quote_id=?", id)
	if err == nil {
		row["_lines"] = lines
	}
	return row, nil
}

func (h *Handler) getPOSnapshot(id string) (map[string]interface{}, error) {
	row, err := h.GetRowAsMap("SELECT * FROM purchase_orders WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	lines, err := h.GetRowsAsMapSlice("SELECT * FROM po_lines WHERE po_id=?", id)
	if err == nil {
		row["_lines"] = lines
	}
	return row, nil
}

// --- Restore functions ---

func (h *Handler) restoreECO(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil { return err }
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO ecos (id, title, description, status, priority, affected_ipns, created_by, created_at, updated_at, approved_at, approved_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["title"], m["description"], m["status"], m["priority"], m["affected_ipns"], m["created_by"], m["created_at"], m["updated_at"], m["approved_at"], m["approved_by"])
	return err
}

func (h *Handler) restoreWorkOrder(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil { return err }
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO work_orders (id, assembly_ipn, qty, status, priority, notes, created_at, started_at, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["assembly_ipn"], m["qty"], m["status"], m["priority"], m["notes"], m["created_at"], m["started_at"], m["completed_at"])
	return err
}

func (h *Handler) restoreNCR(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil { return err }
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO ncrs (id, title, description, ipn, serial_number, defect_type, severity, status, root_cause, corrective_action, created_at, resolved_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["title"], m["description"], m["ipn"], m["serial_number"], m["defect_type"], m["severity"], m["status"], m["root_cause"], m["corrective_action"], m["created_at"], m["resolved_at"])
	return err
}

func (h *Handler) restoreDevice(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil { return err }
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO devices (serial_number, ipn, firmware_version, customer, location, status, install_date, last_seen, notes, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["serial_number"], m["ipn"], m["firmware_version"], m["customer"], m["location"], m["status"], m["install_date"], m["last_seen"], m["notes"], m["created_at"])
	return err
}

func (h *Handler) restoreInventory(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil { return err }
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO inventory (ipn, qty_on_hand, qty_reserved, location, reorder_point, reorder_qty, description, mpn, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["ipn"], m["qty_on_hand"], m["qty_reserved"], m["location"], m["reorder_point"], m["reorder_qty"], m["description"], m["mpn"], m["updated_at"])
	return err
}

func (h *Handler) restoreRMA(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil { return err }
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO rmas (id, serial_number, customer, reason, status, defect_description, resolution, created_at, received_at, resolved_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["serial_number"], m["customer"], m["reason"], m["status"], m["defect_description"], m["resolution"], m["created_at"], m["received_at"], m["resolved_at"])
	return err
}

func (h *Handler) restoreVendor(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil { return err }
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO vendors (id, name, website, contact_name, contact_email, contact_phone, notes, status, lead_time_days, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["name"], m["website"], m["contact_name"], m["contact_email"], m["contact_phone"], m["notes"], m["status"], m["lead_time_days"], m["created_at"])
	return err
}

func (h *Handler) restoreQuote(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil { return err }
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO quotes (id, customer, status, notes, created_at, valid_until, accepted_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["customer"], m["status"], m["notes"], m["created_at"], m["valid_until"], m["accepted_at"])
	if err != nil { return err }
	if linesRaw, ok := m["_lines"]; ok {
		if lines, ok := linesRaw.([]interface{}); ok {
			for _, lineRaw := range lines {
				if line, ok := lineRaw.(map[string]interface{}); ok {
					h.DB.Exec(`INSERT OR REPLACE INTO quote_lines (id, quote_id, ipn, description, qty, unit_price, notes) VALUES (?, ?, ?, ?, ?, ?, ?)`,
						line["id"], line["quote_id"], line["ipn"], line["description"], line["qty"], line["unit_price"], line["notes"])
				}
			}
		}
	}
	return nil
}

func (h *Handler) restorePO(jsonData string) error {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil { return err }
	_, err := h.DB.Exec(`INSERT OR REPLACE INTO purchase_orders (id, vendor_id, status, notes, created_at, expected_date, received_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m["id"], m["vendor_id"], m["status"], m["notes"], m["created_at"], m["expected_date"], m["received_at"])
	if err != nil { return err }
	if linesRaw, ok := m["_lines"]; ok {
		if lines, ok := linesRaw.([]interface{}); ok {
			for _, lineRaw := range lines {
				if line, ok := lineRaw.(map[string]interface{}); ok {
					h.DB.Exec(`INSERT OR REPLACE INTO po_lines (id, po_id, ipn, mpn, manufacturer, qty_ordered, qty_received, unit_price, notes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
						line["id"], line["po_id"], line["ipn"], line["mpn"], line["manufacturer"], line["qty_ordered"], line["qty_received"], line["unit_price"], line["notes"])
				}
			}
		}
	}
	return nil
}

// Exported restore methods for use by compatibility wrappers in root package.

func (h *Handler) RestoreECO(jsonData string) error         { return h.restoreECO(jsonData) }
func (h *Handler) RestoreWorkOrder(jsonData string) error    { return h.restoreWorkOrder(jsonData) }
func (h *Handler) RestoreNCR(jsonData string) error          { return h.restoreNCR(jsonData) }
func (h *Handler) RestoreDevice(jsonData string) error       { return h.restoreDevice(jsonData) }
func (h *Handler) RestoreInventory(jsonData string) error    { return h.restoreInventory(jsonData) }
func (h *Handler) RestoreRMA(jsonData string) error          { return h.restoreRMA(jsonData) }
func (h *Handler) RestoreVendor(jsonData string) error       { return h.restoreVendor(jsonData) }
func (h *Handler) RestoreQuote(jsonData string) error        { return h.restoreQuote(jsonData) }
func (h *Handler) RestorePO(jsonData string) error           { return h.restorePO(jsonData) }
