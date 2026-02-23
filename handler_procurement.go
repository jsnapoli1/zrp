package main

import (
	"net/http"
)

func handleListPOs(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().ListPOs(w, r)
}

func handleGetPO(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().GetPO(w, r, id)
}

func handleCreatePO(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().CreatePO(w, r)
}

func handleUpdatePO(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().UpdatePO(w, r, id)
}

func handleGeneratePOFromWO(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().GeneratePOFromWO(w, r)
}

func handleReceivePO(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().ReceivePO(w, r, id)
}

func handleGeneratePOSuggestions(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().GeneratePOSuggestions(w, r)
}

func handleReviewPOSuggestion(w http.ResponseWriter, r *http.Request, suggestionID int) {
	getProcurementHandler().ReviewPOSuggestion(w, r, suggestionID)
}
