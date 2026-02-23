package main

import (
	"net/http"
)

func handleListReceiving(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().ListReceiving(w, r)
}

func handleInspectReceiving(w http.ResponseWriter, r *http.Request, idStr string) {
	getProcurementHandler().InspectReceiving(w, r, idStr)
}

func handleWhereUsed(w http.ResponseWriter, r *http.Request, ipn string) {
	getProcurementHandler().WhereUsed(w, r, ipn)
}
