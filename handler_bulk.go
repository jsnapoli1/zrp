package main

import "net/http"

func handleBulkECOs(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkECOs(w, r)
}

func handleBulkWorkOrders(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkWorkOrders(w, r)
}

func handleBulkNCRs(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkNCRs(w, r)
}

func handleBulkDevices(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkDevices(w, r)
}

func handleBulkInventory(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkInventory(w, r)
}

func handleBulkRMAs(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkRMAs(w, r)
}

func handleBulkParts(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkParts(w, r)
}

func handleBulkPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().BulkPurchaseOrders(w, r)
}
