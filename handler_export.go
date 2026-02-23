package main

import "net/http"

func handleExportParts(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ExportParts(w, r)
}

func handleExportInventory(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ExportInventory(w, r)
}

func handleExportWorkOrders(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ExportWorkOrders(w, r)
}

func handleExportECOs(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ExportECOs(w, r)
}

func handleExportVendors(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ExportVendors(w, r)
}
