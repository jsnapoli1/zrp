package main

import (
	"net/http"
)

func handleListSalesOrders(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().ListSalesOrders(w, r)
}

func handleGetSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().GetSalesOrder(w, r, id)
}

func getSalesOrderLines(orderID string) []SalesOrderLine {
	// Keep backward-compatible wrapper for other root-level code that calls this.
	rows, err := db.Query("SELECT id,sales_order_id,ipn,COALESCE(description,''),qty,qty_allocated,qty_picked,qty_shipped,COALESCE(unit_price,0),COALESCE(notes,'') FROM sales_order_lines WHERE sales_order_id=?", orderID)
	if err != nil {
		return []SalesOrderLine{}
	}
	defer rows.Close()
	var lines []SalesOrderLine
	for rows.Next() {
		var l SalesOrderLine
		rows.Scan(&l.ID, &l.SalesOrderID, &l.IPN, &l.Description, &l.Qty, &l.QtyAllocated, &l.QtyPicked, &l.QtyShipped, &l.UnitPrice, &l.Notes)
		lines = append(lines, l)
	}
	if lines == nil {
		lines = []SalesOrderLine{}
	}
	return lines
}

func handleCreateSalesOrder(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().CreateSalesOrder(w, r)
}

func handleUpdateSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().UpdateSalesOrder(w, r, id)
}

func handleConvertQuoteToOrder(w http.ResponseWriter, r *http.Request, quoteID string) {
	getSalesHandler().ConvertQuoteToOrder(w, r, quoteID)
}

func handleConfirmSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().ConfirmSalesOrder(w, r, id)
}

func handleAllocateSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().AllocateSalesOrder(w, r, id)
}

func handlePickSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().PickSalesOrder(w, r, id)
}

func handleShipSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().ShipSalesOrder(w, r, id)
}

func handleInvoiceSalesOrder(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().InvoiceSalesOrder(w, r, id)
}
