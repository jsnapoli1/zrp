package main

import "net/http"

func handleBulkUpdateInventory(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkUpdateInventory(w, r)
}

func handleBulkUpdateWorkOrders(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkUpdateWorkOrders(w, r)
}

func handleBulkUpdateDevices(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkUpdateDevices(w, r)
}

func handleBulkUpdateParts(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkUpdateParts(w, r)
}

func handleBulkUpdateECOs(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkUpdateECOs(w, r)
}
