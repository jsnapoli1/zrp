package main

import (
	"net/http"
)

func handleListShipments(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().ListShipments(w, r)
}

func handleGetShipment(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().GetShipment(w, r, id)
}

func getShipmentLines(shipmentID string) []ShipmentLine {
	// Keep backward-compatible wrapper for other root-level code that calls this.
	rows, err := db.Query("SELECT id,shipment_id,COALESCE(ipn,''),COALESCE(serial_number,''),qty,COALESCE(work_order_id,''),COALESCE(rma_id,'') FROM shipment_lines WHERE shipment_id=?", shipmentID)
	if err != nil {
		return []ShipmentLine{}
	}
	defer rows.Close()
	var lines []ShipmentLine
	for rows.Next() {
		var l ShipmentLine
		rows.Scan(&l.ID, &l.ShipmentID, &l.IPN, &l.SerialNumber, &l.Qty, &l.WorkOrderID, &l.RMAID)
		lines = append(lines, l)
	}
	if lines == nil {
		lines = []ShipmentLine{}
	}
	return lines
}

func handleCreateShipment(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().CreateShipment(w, r)
}

func handleUpdateShipment(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().UpdateShipment(w, r, id)
}

func handleShipShipment(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().ShipShipment(w, r, id)
}

func handleDeliverShipment(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().DeliverShipment(w, r, id)
}

func handleShipmentPackList(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().ShipmentPackList(w, r, id)
}
