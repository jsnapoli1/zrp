package main

import (
	"net/http"

	"zrp/internal/handlers/procurement"
)

var procurementHandler *procurement.Handler

func initProcurementHandler() {
	procurementHandler = getProcurementHandler()
}

func getProcurementHandler() *procurement.Handler {
	if procurementHandler == nil || procurementHandler.DB != db {
		procurementHandler = &procurement.Handler{
			DB:                db,
			Hub:               wsHub,
			NextIDFunc:        nextID,
			RecordChangeJSON:  recordChangeJSON,
			GetVendorSnapshot: getVendorSnapshot,
			CreateUndoEntry:   createUndoEntry,
		}
	}
	return procurementHandler
}

func handleListVendors(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().ListVendors(w, r)
}

func handleGetVendor(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().GetVendor(w, r, id)
}

func handleCreateVendor(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().CreateVendor(w, r)
}

func handleUpdateVendor(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().UpdateVendor(w, r, id)
}

func handleDeleteVendor(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().DeleteVendor(w, r, id)
}
