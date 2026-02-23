package inventory

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"zrp/internal/audit"
	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
	"zrp/internal/websocket"
)

// PartLookupFunc looks up a part by IPN in the parts directory and returns its fields.
type PartLookupFunc func(partsDir, ipn string) (map[string]string, error)

// EmailOnLowStockFunc sends an email notification when inventory is low.
type EmailOnLowStockFunc func(ipn string)

// Handler holds dependencies for inventory handlers.
type Handler struct {
	DB               *sql.DB
	Hub              *websocket.Hub
	PartsDir         string
	GetPartByIPN     PartLookupFunc
	EmailOnLowStock  EmailOnLowStockFunc
}

// ListInventory handles GET /api/inventory.
func (h *Handler) ListInventory(w http.ResponseWriter, r *http.Request) {
	lowStock := r.URL.Query().Get("low_stock")
	query := "SELECT ipn,qty_on_hand,qty_reserved,COALESCE(location,''),reorder_point,reorder_qty,COALESCE(description,''),COALESCE(mpn,''),updated_at FROM inventory"
	if lowStock == "true" {
		query += " WHERE qty_on_hand <= reorder_point AND reorder_point > 0"
	}
	query += " ORDER BY ipn"
	rows, err := h.DB.Query(query)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.InventoryItem
	for rows.Next() {
		var i models.InventoryItem
		rows.Scan(&i.IPN, &i.QtyOnHand, &i.QtyReserved, &i.Location, &i.ReorderPoint, &i.ReorderQty, &i.Description, &i.MPN, &i.UpdatedAt)
		items = append(items, i)
	}
	if items == nil {
		items = []models.InventoryItem{}
	}
	response.JSON(w, items)
}

// GetInventory handles GET /api/inventory/:ipn.
func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request, ipn string) {
	var i models.InventoryItem
	err := h.DB.QueryRow("SELECT ipn,qty_on_hand,qty_reserved,COALESCE(location,''),reorder_point,reorder_qty,COALESCE(description,''),COALESCE(mpn,''),updated_at FROM inventory WHERE ipn=?", ipn).
		Scan(&i.IPN, &i.QtyOnHand, &i.QtyReserved, &i.Location, &i.ReorderPoint, &i.ReorderQty, &i.Description, &i.MPN, &i.UpdatedAt)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	response.JSON(w, i)
}

// Transact handles POST /api/inventory/transact.
func (h *Handler) Transact(w http.ResponseWriter, r *http.Request) {
	var t models.InventoryTransaction
	if err := response.DecodeBody(r, &t); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "ipn", t.IPN)
	validation.RequireField(ve, "type", t.Type)
	validation.ValidateEnum(ve, "type", t.Type, validation.ValidInventoryTypes)
	if t.Type != "adjust" && t.Qty <= 0 {
		ve.Add("qty", "must be positive")
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")

	// Ensure inventory record exists, enriching with parts DB data
	var desc, mpn string
	if h.GetPartByIPN != nil {
		fields, err2 := h.GetPartByIPN(h.PartsDir, t.IPN)
		if err2 == nil {
			for k, v := range fields {
				kl := strings.ToLower(k)
				if kl == "description" || kl == "desc" {
					desc = v
				}
				if kl == "mpn" {
					mpn = v
				}
			}
		}
	}

	// Begin transaction to ensure atomicity
	tx, err := h.DB.Begin()
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer tx.Rollback() // Rollback if not committed

	// Ensure inventory record exists
	_, err = tx.Exec("INSERT OR IGNORE INTO inventory (ipn, description, mpn) VALUES (?, ?, ?)", t.IPN, desc, mpn)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Insert transaction
	_, err = tx.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
		t.IPN, t.Type, t.Qty, t.Reference, t.Notes, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Update inventory quantity
	switch t.Type {
	case "receive", "return":
		_, err = tx.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand+?,updated_at=? WHERE ipn=?", t.Qty, now, t.IPN)
	case "issue":
		_, err = tx.Exec("UPDATE inventory SET qty_on_hand=qty_on_hand-?,updated_at=? WHERE ipn=?", t.Qty, now, t.IPN)
	case "adjust":
		_, err = tx.Exec("UPDATE inventory SET qty_on_hand=?,updated_at=? WHERE ipn=?", t.Qty, now, t.IPN)
	}
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), t.Type, "inventory", t.IPN, "Inventory "+t.Type+": "+t.IPN)

	// Check low stock in background
	if h.EmailOnLowStock != nil {
		currentDB := h.DB
		ipnCopy := t.IPN
		emailFn := h.EmailOnLowStock
		go func() {
			if currentDB == nil {
				return
			}
			// Check email config enabled
			var enabled int
			if err := currentDB.QueryRow("SELECT enabled FROM email_config WHERE id=1").Scan(&enabled); err != nil || enabled != 1 {
				return
			}
			// Check inventory levels
			var qtyOnHand, reorderPoint int
			if err := currentDB.QueryRow("SELECT qty_on_hand, reorder_point FROM inventory WHERE ipn=?", ipnCopy).Scan(&qtyOnHand, &reorderPoint); err != nil || reorderPoint <= 0 || qtyOnHand > reorderPoint {
				return
			}
			emailFn(ipnCopy)
		}()
	}

	response.JSON(w, map[string]string{"status": "ok"})
}

// History handles GET /api/inventory/:ipn/history.
func (h *Handler) History(w http.ResponseWriter, r *http.Request, ipn string) {
	rows, err := h.DB.Query("SELECT id,ipn,type,qty,COALESCE(reference,''),COALESCE(notes,''),created_at FROM inventory_transactions WHERE ipn=? ORDER BY created_at DESC", ipn)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.InventoryTransaction
	for rows.Next() {
		var t models.InventoryTransaction
		rows.Scan(&t.ID, &t.IPN, &t.Type, &t.Qty, &t.Reference, &t.Notes, &t.CreatedAt)
		items = append(items, t)
	}
	if items == nil {
		items = []models.InventoryTransaction{}
	}
	response.JSON(w, items)
}

// BulkDelete handles POST /api/inventory/bulk-delete.
func (h *Handler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IPNs []string `json:"ipns"`
	}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	if len(body.IPNs) == 0 {
		response.Err(w, "ipns required", 400)
		return
	}
	deleted := 0
	for _, ipn := range body.IPNs {
		res, err := h.DB.Exec("DELETE FROM inventory WHERE ipn=?", ipn)
		if err != nil {
			continue
		}
		n, _ := res.RowsAffected()
		deleted += int(n)
	}
	response.JSON(w, map[string]int{"deleted": deleted})
}
