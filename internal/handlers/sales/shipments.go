package sales

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"zrp/internal/audit"
	"zrp/internal/database"
	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
)

// ListShipments handles GET /api/shipments.
func (h *Handler) ListShipments(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id,type,status,COALESCE(tracking_number,''),COALESCE(carrier,''),ship_date,delivery_date,COALESCE(from_address,''),COALESCE(to_address,''),COALESCE(notes,''),COALESCE(created_by,''),created_at,updated_at FROM shipments ORDER BY created_at DESC")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.Shipment
	for rows.Next() {
		var s models.Shipment
		var sd, dd sql.NullString
		rows.Scan(&s.ID, &s.Type, &s.Status, &s.TrackingNumber, &s.Carrier, &sd, &dd, &s.FromAddress, &s.ToAddress, &s.Notes, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
		s.ShipDate = database.SP(sd)
		s.DeliveryDate = database.SP(dd)
		items = append(items, s)
	}
	if items == nil {
		items = []models.Shipment{}
	}
	response.JSON(w, items)
}

// GetShipment handles GET /api/shipments/:id.
func (h *Handler) GetShipment(w http.ResponseWriter, r *http.Request, id string) {
	var s models.Shipment
	var sd, dd sql.NullString
	err := h.DB.QueryRow("SELECT id,type,status,COALESCE(tracking_number,''),COALESCE(carrier,''),ship_date,delivery_date,COALESCE(from_address,''),COALESCE(to_address,''),COALESCE(notes,''),COALESCE(created_by,''),created_at,updated_at FROM shipments WHERE id=?", id).
		Scan(&s.ID, &s.Type, &s.Status, &s.TrackingNumber, &s.Carrier, &sd, &dd, &s.FromAddress, &s.ToAddress, &s.Notes, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	s.ShipDate = database.SP(sd)
	s.DeliveryDate = database.SP(dd)
	s.Lines = h.getShipmentLines(id)
	response.JSON(w, s)
}

func (h *Handler) getShipmentLines(shipmentID string) []models.ShipmentLine {
	rows, err := h.DB.Query("SELECT id,shipment_id,COALESCE(ipn,''),COALESCE(serial_number,''),qty,COALESCE(work_order_id,''),COALESCE(rma_id,'') FROM shipment_lines WHERE shipment_id=?", shipmentID)
	if err != nil {
		return []models.ShipmentLine{}
	}
	defer rows.Close()
	var lines []models.ShipmentLine
	for rows.Next() {
		var l models.ShipmentLine
		rows.Scan(&l.ID, &l.ShipmentID, &l.IPN, &l.SerialNumber, &l.Qty, &l.WorkOrderID, &l.RMAID)
		lines = append(lines, l)
	}
	if lines == nil {
		lines = []models.ShipmentLine{}
	}
	return lines
}

// CreateShipment handles POST /api/shipments.
func (h *Handler) CreateShipment(w http.ResponseWriter, r *http.Request) {
	var s models.Shipment
	if err := response.DecodeBody(r, &s); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	if s.Type != "" {
		validation.ValidateEnum(ve, "type", s.Type, validation.ValidShipmentTypes)
	}
	if s.Status != "" {
		validation.ValidateEnum(ve, "status", s.Status, validation.ValidShipmentStatuses)
	}
	for i, line := range s.Lines {
		if line.Qty <= 0 {
			ve.Add(fmt.Sprintf("lines[%d].qty", i), "must be positive")
		}
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	s.ID = h.NextID("SHP", "shipments", 4)
	if s.Type == "" {
		s.Type = "outbound"
	}
	if s.Status == "" {
		s.Status = "draft"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	s.CreatedBy = audit.GetUsername(h.DB, r)
	_, err := h.DB.Exec("INSERT INTO shipments (id,type,status,tracking_number,carrier,from_address,to_address,notes,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
		s.ID, s.Type, s.Status, s.TrackingNumber, s.Carrier, s.FromAddress, s.ToAddress, s.Notes, s.CreatedBy, now, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	s.CreatedAt = now
	s.UpdatedAt = now

	// Insert lines if provided
	for _, line := range s.Lines {
		_, err := h.DB.Exec("INSERT INTO shipment_lines (shipment_id,ipn,serial_number,qty,work_order_id,rma_id) VALUES (?,?,?,?,?,?)",
			s.ID, line.IPN, line.SerialNumber, line.Qty, line.WorkOrderID, line.RMAID)
		if err != nil {
			response.Err(w, err.Error(), 500)
			return
		}
	}

	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), "created", "shipment", s.ID, "Created shipment "+s.ID)
	s.Lines = h.getShipmentLines(s.ID)
	response.JSON(w, s)
}

// UpdateShipment handles PUT /api/shipments/:id.
func (h *Handler) UpdateShipment(w http.ResponseWriter, r *http.Request, id string) {
	var s models.Shipment
	if err := response.DecodeBody(r, &s); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := h.DB.Exec("UPDATE shipments SET type=?,status=?,tracking_number=?,carrier=?,from_address=?,to_address=?,notes=?,updated_at=? WHERE id=?",
		s.Type, s.Status, s.TrackingNumber, s.Carrier, s.FromAddress, s.ToAddress, s.Notes, now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Replace lines if provided
	if s.Lines != nil {
		h.DB.Exec("DELETE FROM shipment_lines WHERE shipment_id=?", id)
		for _, line := range s.Lines {
			h.DB.Exec("INSERT INTO shipment_lines (shipment_id,ipn,serial_number,qty,work_order_id,rma_id) VALUES (?,?,?,?,?,?)",
				id, line.IPN, line.SerialNumber, line.Qty, line.WorkOrderID, line.RMAID)
		}
	}

	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), "updated", "shipment", id, fmt.Sprintf("Updated shipment %s: status=%s", id, s.Status))
	h.GetShipment(w, r, id)
}

// ShipShipment handles POST /api/shipments/:id/ship.
func (h *Handler) ShipShipment(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		TrackingNumber string `json:"tracking_number"`
		Carrier        string `json:"carrier"`
	}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	// Verify shipment exists and is in valid state
	var status string
	err := h.DB.QueryRow("SELECT status FROM shipments WHERE id=?", id).Scan(&status)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	if status == "shipped" || status == "delivered" {
		response.Err(w, "shipment already "+status, 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = h.DB.Exec("UPDATE shipments SET status='shipped',tracking_number=?,carrier=?,ship_date=?,updated_at=? WHERE id=?",
		body.TrackingNumber, body.Carrier, now, now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), "shipped", "shipment", id, fmt.Sprintf("Shipped %s via %s tracking %s", id, body.Carrier, body.TrackingNumber))
	h.GetShipment(w, r, id)
}

// DeliverShipment handles POST /api/shipments/:id/deliver.
func (h *Handler) DeliverShipment(w http.ResponseWriter, r *http.Request, id string) {
	// Verify shipment exists
	var status, shipType string
	err := h.DB.QueryRow("SELECT status, type FROM shipments WHERE id=?", id).Scan(&status, &shipType)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	if status == "delivered" {
		response.Err(w, "already delivered", 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = h.DB.Exec("UPDATE shipments SET status='delivered',delivery_date=?,updated_at=? WHERE id=?", now, now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Update inventory for inbound shipments
	if shipType == "inbound" {
		lines := h.getShipmentLines(id)
		for _, line := range lines {
			if line.IPN != "" && line.Qty > 0 {
				h.DB.Exec("UPDATE inventory SET qty_on_hand = qty_on_hand + ?, updated_at = ? WHERE ipn = ?", line.Qty, now, line.IPN)
				h.DB.Exec("INSERT INTO inventory_transactions (ipn,type,qty,reference,notes,created_at) VALUES (?,'receive',?,?,?,?)",
					line.IPN, line.Qty, "SHP:"+id, "Inbound shipment delivered", now)
			}
		}
	}

	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), "delivered", "shipment", id, "Marked "+id+" as delivered")
	h.GetShipment(w, r, id)
}

// ShipmentPackList handles GET /api/shipments/:id/pack-list.
func (h *Handler) ShipmentPackList(w http.ResponseWriter, r *http.Request, id string) {
	// Verify shipment exists
	var exists int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM shipments WHERE id=?", id).Scan(&exists)
	if err != nil || exists == 0 {
		response.Err(w, "not found", 404)
		return
	}

	lines := h.getShipmentLines(id)

	// Auto-create pack list record
	now := time.Now().Format("2006-01-02 15:04:05")
	res, _ := h.DB.Exec("INSERT INTO pack_lists (shipment_id,created_at) VALUES (?,?)", id, now)
	plID, _ := res.LastInsertId()

	pl := models.PackList{
		ID:         int(plID),
		ShipmentID: id,
		CreatedAt:  now,
		Lines:      lines,
	}
	response.JSON(w, pl)
}
