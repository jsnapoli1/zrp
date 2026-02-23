package sales

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"

	"zrp/internal/audit"
	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
)

// ListSalesOrders handles GET /api/sales-orders.
func (h *Handler) ListSalesOrders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	customer := r.URL.Query().Get("customer")

	query := "SELECT id,COALESCE(quote_id,''),customer,status,COALESCE(notes,''),COALESCE(created_by,''),created_at,updated_at FROM sales_orders"
	var conditions []string
	var args []interface{}

	if status != "" {
		conditions = append(conditions, "status=?")
		args = append(args, status)
	}
	if customer != "" {
		conditions = append(conditions, "customer LIKE ?")
		args = append(args, "%"+customer+"%")
	}
	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			query += " AND " + c
		}
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.SalesOrder
	for rows.Next() {
		var o models.SalesOrder
		rows.Scan(&o.ID, &o.QuoteID, &o.Customer, &o.Status, &o.Notes, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt)
		items = append(items, o)
	}
	if items == nil {
		items = []models.SalesOrder{}
	}
	response.JSON(w, items)
}

// GetSalesOrder handles GET /api/sales-orders/:id.
func (h *Handler) GetSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	var o models.SalesOrder
	err := h.DB.QueryRow("SELECT id,COALESCE(quote_id,''),customer,status,COALESCE(notes,''),COALESCE(created_by,''),created_at,updated_at FROM sales_orders WHERE id=?", id).
		Scan(&o.ID, &o.QuoteID, &o.Customer, &o.Status, &o.Notes, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	o.Lines = h.getSalesOrderLines(id)

	// Attach shipment/invoice IDs if they exist
	var shipID sql.NullString
	h.DB.QueryRow("SELECT DISTINCT sl.shipment_id FROM shipment_lines sl WHERE sl.sales_order_id=? LIMIT 1", id).Scan(&shipID)
	if shipID.Valid {
		o.ShipmentID = &shipID.String
	}
	var invID sql.NullString
	h.DB.QueryRow("SELECT id FROM invoices WHERE sales_order_id=? LIMIT 1", id).Scan(&invID)
	if invID.Valid {
		o.InvoiceID = &invID.String
	}

	response.JSON(w, o)
}

func (h *Handler) getSalesOrderLines(orderID string) []models.SalesOrderLine {
	rows, err := h.DB.Query("SELECT id,sales_order_id,ipn,COALESCE(description,''),qty,qty_allocated,qty_picked,qty_shipped,COALESCE(unit_price,0),COALESCE(notes,'') FROM sales_order_lines WHERE sales_order_id=?", orderID)
	if err != nil {
		return []models.SalesOrderLine{}
	}
	defer rows.Close()
	var lines []models.SalesOrderLine
	for rows.Next() {
		var l models.SalesOrderLine
		rows.Scan(&l.ID, &l.SalesOrderID, &l.IPN, &l.Description, &l.Qty, &l.QtyAllocated, &l.QtyPicked, &l.QtyShipped, &l.UnitPrice, &l.Notes)
		lines = append(lines, l)
	}
	if lines == nil {
		lines = []models.SalesOrderLine{}
	}
	return lines
}

// CreateSalesOrder handles POST /api/sales-orders.
func (h *Handler) CreateSalesOrder(w http.ResponseWriter, r *http.Request) {
	var o models.SalesOrder
	if err := response.DecodeBody(r, &o); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "customer", o.Customer)
	if o.Status != "" {
		validation.ValidateEnum(ve, "status", o.Status, validation.ValidSalesOrderStatuses)
	}
	for i, l := range o.Lines {
		if l.Qty <= 0 {
			ve.Add(fmt.Sprintf("lines[%d].qty", i), "must be positive")
		}
		if l.UnitPrice < 0 {
			ve.Add(fmt.Sprintf("lines[%d].unit_price", i), "must be non-negative")
		}
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	o.ID = h.NextID("SO", "sales_orders", 4)
	if o.Status == "" {
		o.Status = "draft"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	o.CreatedBy = audit.GetUsername(h.DB, r)
	_, err := h.DB.Exec("INSERT INTO sales_orders (id,quote_id,customer,status,notes,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)",
		o.ID, o.QuoteID, o.Customer, o.Status, o.Notes, o.CreatedBy, now, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	for _, l := range o.Lines {
		h.DB.Exec("INSERT INTO sales_order_lines (sales_order_id,ipn,description,qty,unit_price,notes) VALUES (?,?,?,?,?,?)",
			o.ID, l.IPN, l.Description, l.Qty, l.UnitPrice, l.Notes)
	}
	o.CreatedAt = now
	o.UpdatedAt = now
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "created", "sales_order", o.ID, "Created "+o.ID+" for "+o.Customer)
	h.RecordChangeJSON(username, "sales_orders", o.ID, "create", nil, o)
	response.JSON(w, o)
}

// UpdateSalesOrder handles PUT /api/sales-orders/:id.
func (h *Handler) UpdateSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	var o models.SalesOrder
	if err := response.DecodeBody(r, &o); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := h.DB.Exec("UPDATE sales_orders SET customer=?,status=?,notes=?,updated_at=? WHERE id=?",
		o.Customer, o.Status, o.Notes, now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), "updated", "sales_order", id, "Updated "+id+": status="+o.Status)
	h.GetSalesOrder(w, r, id)
}

// ConvertQuoteToOrder handles POST /api/quotes/:id/convert.
func (h *Handler) ConvertQuoteToOrder(w http.ResponseWriter, r *http.Request, quoteID string) {
	// Fetch quote
	var q models.Quote
	var aa sql.NullString
	err := h.DB.QueryRow("SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes WHERE id=?", quoteID).
		Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
	if err != nil {
		response.Err(w, "quote not found", 404)
		return
	}
	if q.Status != "accepted" {
		response.Err(w, "quote must be in 'accepted' status to convert", 400)
		return
	}

	// Check if already converted
	var existingID string
	err = h.DB.QueryRow("SELECT id FROM sales_orders WHERE quote_id=?", quoteID).Scan(&existingID)
	if err == nil {
		response.Err(w, fmt.Sprintf("quote already converted to order %s", existingID), 409)
		return
	}

	// Get quote lines
	rows, _ := h.DB.Query("SELECT ipn,COALESCE(description,''),qty,COALESCE(unit_price,0),COALESCE(notes,'') FROM quote_lines WHERE quote_id=?", quoteID)
	var lines []models.QuoteLine
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l models.QuoteLine
			rows.Scan(&l.IPN, &l.Description, &l.Qty, &l.UnitPrice, &l.Notes)
			lines = append(lines, l)
		}
	}

	// Create sales order
	orderID := h.NextID("SO", "sales_orders", 4)
	now := time.Now().Format("2006-01-02 15:04:05")
	username := audit.GetUsername(h.DB, r)
	_, err = h.DB.Exec("INSERT INTO sales_orders (id,quote_id,customer,status,notes,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)",
		orderID, quoteID, q.Customer, "draft", q.Notes, username, now, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	for _, l := range lines {
		h.DB.Exec("INSERT INTO sales_order_lines (sales_order_id,ipn,description,qty,unit_price,notes) VALUES (?,?,?,?,?,?)",
			orderID, l.IPN, l.Description, l.Qty, l.UnitPrice, l.Notes)
	}

	audit.LogAudit(h.DB, h.Hub, username, "created", "sales_order", orderID, fmt.Sprintf("Converted quote %s to order %s", quoteID, orderID))
	h.RecordChangeJSON(username, "sales_orders", orderID, "create", nil, map[string]string{"quote_id": quoteID})
	h.GetSalesOrder(w, r, orderID)
}

// ConfirmSalesOrder handles POST /api/sales-orders/:id/confirm.
func (h *Handler) ConfirmSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	h.transitionSalesOrder(w, r, id, "draft", "confirmed")
}

// AllocateSalesOrder handles POST /api/sales-orders/:id/allocate.
func (h *Handler) AllocateSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	// Check inventory availability
	lines := h.getSalesOrderLines(id)
	for _, l := range lines {
		var qtyOnHand, qtyReserved float64
		err := h.DB.QueryRow("SELECT COALESCE(qty_on_hand,0), COALESCE(qty_reserved,0) FROM inventory WHERE ipn=?", l.IPN).Scan(&qtyOnHand, &qtyReserved)
		if err != nil {
			response.Err(w, fmt.Sprintf("inventory record not found for %s", l.IPN), 400)
			return
		}
		available := qtyOnHand - qtyReserved
		if available < float64(l.Qty) {
			response.Err(w, fmt.Sprintf("insufficient inventory for %s: need %d, available %.0f", l.IPN, l.Qty, available), 400)
			return
		}
	}

	// Reserve inventory
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, l := range lines {
		h.DB.Exec("UPDATE inventory SET qty_reserved = qty_reserved + ?, updated_at = ? WHERE ipn=?", l.Qty, now, l.IPN)
		h.DB.Exec("UPDATE sales_order_lines SET qty_allocated=? WHERE id=?", l.Qty, l.ID)
		h.DB.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
			l.IPN, "adjust", 0, fmt.Sprintf("SO:%s", id), fmt.Sprintf("Reserved %d for %s", l.Qty, id), now)
	}

	h.transitionSalesOrder(w, r, id, "confirmed", "allocated")
}

// PickSalesOrder handles POST /api/sales-orders/:id/pick.
func (h *Handler) PickSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	lines := h.getSalesOrderLines(id)
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, l := range lines {
		h.DB.Exec("UPDATE sales_order_lines SET qty_picked=? WHERE id=?", l.Qty, l.ID)
		_ = now
	}
	h.transitionSalesOrder(w, r, id, "allocated", "picked")
}

// ShipSalesOrder handles POST /api/sales-orders/:id/ship.
func (h *Handler) ShipSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	var o models.SalesOrder
	err := h.DB.QueryRow("SELECT id,COALESCE(quote_id,''),customer,status FROM sales_orders WHERE id=?", id).
		Scan(&o.ID, &o.QuoteID, &o.Customer, &o.Status)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	if o.Status != "picked" {
		response.Err(w, "order must be in 'picked' status to ship", 400)
		return
	}

	lines := h.getSalesOrderLines(id)
	now := time.Now().Format("2006-01-02 15:04:05")
	username := audit.GetUsername(h.DB, r)

	// Create outbound shipment
	shipID := h.NextID("SH", "shipments", 4)
	h.DB.Exec("INSERT INTO shipments (id,type,status,to_address,notes,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)",
		shipID, "outbound", "packed", o.Customer, fmt.Sprintf("Shipment for %s", id), username, now, now)

	for _, l := range lines {
		// Create shipment line
		h.DB.Exec("INSERT INTO shipment_lines (shipment_id,ipn,qty,sales_order_id) VALUES (?,?,?,?)",
			shipID, l.IPN, l.Qty, id)
		// Reduce inventory (issue)
		h.DB.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand - ?, qty_reserved = qty_reserved - ?, updated_at = ? WHERE ipn=?",
			l.Qty, l.Qty, now, l.IPN)
		h.DB.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,?,?,?,?,?)",
			l.IPN, "issue", float64(l.Qty), fmt.Sprintf("SO:%s", id), fmt.Sprintf("Shipped %d for %s", l.Qty, id), now)
		h.DB.Exec("UPDATE sales_order_lines SET qty_shipped=? WHERE id=?", l.Qty, l.ID)
	}

	h.DB.Exec("UPDATE sales_orders SET status='shipped',updated_at=? WHERE id=?", now, id)
	audit.LogAudit(h.DB, h.Hub, username, "shipped", "sales_order", id, fmt.Sprintf("Shipped %s via shipment %s", id, shipID))
	h.GetSalesOrder(w, r, id)
}

// InvoiceSalesOrder handles POST /api/sales-orders/:id/invoice.
func (h *Handler) InvoiceSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	var o models.SalesOrder
	err := h.DB.QueryRow("SELECT id,COALESCE(quote_id,''),customer,status FROM sales_orders WHERE id=?", id).
		Scan(&o.ID, &o.QuoteID, &o.Customer, &o.Status)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	if o.Status != "shipped" {
		response.Err(w, "order must be in 'shipped' status to invoice", 400)
		return
	}

	lines := h.getSalesOrderLines(id)
	var total float64
	for _, l := range lines {
		total += float64(l.Qty) * l.UnitPrice
	}
	total = math.Round(total*100) / 100

	now := time.Now().Format("2006-01-02 15:04:05")
	invID := h.NextID("INV", "invoices", 4)
	issueDate := time.Now().Format("2006-01-02")
	dueDate := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	username := audit.GetUsername(h.DB, r)

	_, err = h.DB.Exec("INSERT INTO invoices (id,invoice_number,sales_order_id,customer,status,total,created_at,issue_date,due_date) VALUES (?,?,?,?,?,?,?,?,?)",
		invID, invID, id, o.Customer, "draft", total, now, issueDate, dueDate)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	h.DB.Exec("UPDATE sales_orders SET status='invoiced',updated_at=? WHERE id=?", now, id)
	audit.LogAudit(h.DB, h.Hub, username, "invoiced", "sales_order", id, fmt.Sprintf("Created invoice %s for %s (%.2f)", invID, id, total))
	h.GetSalesOrder(w, r, id)
}

func (h *Handler) transitionSalesOrder(w http.ResponseWriter, r *http.Request, id, fromStatus, toStatus string) {
	var currentStatus string
	err := h.DB.QueryRow("SELECT status FROM sales_orders WHERE id=?", id).Scan(&currentStatus)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	if currentStatus != fromStatus {
		response.Err(w, fmt.Sprintf("order must be in '%s' status (currently '%s')", fromStatus, currentStatus), 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	h.DB.Exec("UPDATE sales_orders SET status=?,updated_at=? WHERE id=?", toStatus, now, id)
	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), toStatus, "sales_order", id, fmt.Sprintf("Transitioned %s from %s to %s", id, fromStatus, toStatus))
	h.GetSalesOrder(w, r, id)
}
