package main

import (
	"database/sql"
	"net/http"

	"zrp/internal/handlers/manufacturing"
)

// mfgHandler is the shared manufacturing handler instance.
var mfgHandler *manufacturing.Handler

// getMfgHandler returns the manufacturing handler, lazily initializing if needed (for tests).
func getMfgHandler() *manufacturing.Handler {
	if mfgHandler == nil || mfgHandler.DB != db {
		mfgHandler = &manufacturing.Handler{
			DB:                      db,
			Hub:                     wsHub,
			PartsDir:                partsDir,
			CompanyName:             companyName,
			GetPartByIPN:            getPartByIPN,
			NextIDFunc:              nextID,
			RecordChangeJSON:        recordChangeJSON,
			CreateUndoEntry:         createUndoEntry,
			GetWorkOrderSnapshot:    getWorkOrderSnapshot,
			EmailOnOverdueWorkOrder: emailOnOverdueWorkOrder,
		}
	}
	return mfgHandler
}

func handleListWorkOrders(w http.ResponseWriter, r *http.Request) {
	getMfgHandler().ListWorkOrders(w, r)
}

func handleGetWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
	getMfgHandler().GetWorkOrder(w, r, id)
}

func handleCreateWorkOrder(w http.ResponseWriter, r *http.Request) {
	getMfgHandler().CreateWorkOrder(w, r)
}

func handleUpdateWorkOrder(w http.ResponseWriter, r *http.Request, id string) {
	getMfgHandler().UpdateWorkOrder(w, r, id)
}

func handleWorkOrderBOM(w http.ResponseWriter, r *http.Request, id string) {
	getMfgHandler().WorkOrderBOM(w, r, id)
}

func handleWorkOrderPDF(w http.ResponseWriter, r *http.Request, id string) {
	getMfgHandler().WorkOrderPDF(w, r, id)
}

func handleWorkOrderKit(w http.ResponseWriter, r *http.Request, id string) {
	getMfgHandler().WorkOrderKit(w, r, id)
}

func handleWorkOrderSerials(w http.ResponseWriter, r *http.Request, id string) {
	getMfgHandler().WorkOrderSerials(w, r, id)
}

func handleWorkOrderAddSerial(w http.ResponseWriter, r *http.Request, id string) {
	getMfgHandler().WorkOrderAddSerial(w, r, id)
}

func isValidStatusTransition(from, to string) bool {
	return manufacturing.IsValidStatusTransition(from, to)
}

func generateSerialNumber(assemblyIPN string) string {
	return manufacturing.GenerateSerialNumber(assemblyIPN)
}

func handleWorkOrderCompletion(tx *sql.Tx, woID, assemblyIPN string, qty int, username string) error {
	return manufacturing.HandleWorkOrderCompletion(tx, woID, assemblyIPN, qty, username)
}

func handleWorkOrderCancellation(tx *sql.Tx, woID string) error {
	return manufacturing.HandleWorkOrderCancellation(tx, woID)
}
