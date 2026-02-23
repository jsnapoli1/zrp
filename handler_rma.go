package main

import (
	"net/http"
)

func handleListRMAs(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().ListRMAs(w, r)
}

func handleGetRMA(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().GetRMA(w, r, id)
}

func handleCreateRMA(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().CreateRMA(w, r)
}

func handleUpdateRMA(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().UpdateRMA(w, r, id)
}
