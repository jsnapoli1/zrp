package main

import (
	"net/http"
)

func handleListRFQs(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().ListRFQs(w, r)
}

func handleGetRFQ(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().GetRFQ(w, r, id)
}

func handleCreateRFQ(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().CreateRFQ(w, r)
}

func handleUpdateRFQ(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().UpdateRFQ(w, r, id)
}

func handleDeleteRFQ(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().DeleteRFQ(w, r, id)
}

func handleSendRFQ(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().SendRFQ(w, r, id)
}

func handleAwardRFQ(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().AwardRFQ(w, r, id)
}

func handleCompareRFQ(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().CompareRFQ(w, r, id)
}

func handleCreateRFQQuote(w http.ResponseWriter, r *http.Request, rfqID string) {
	getProcurementHandler().CreateRFQQuote(w, r, rfqID)
}

func handleUpdateRFQQuote(w http.ResponseWriter, r *http.Request, rfqID string, quoteID string) {
	getProcurementHandler().UpdateRFQQuote(w, r, rfqID, quoteID)
}

func handleCloseRFQ(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().CloseRFQ(w, r, id)
}

func handleRFQDashboard(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().RFQDashboard(w, r)
}

func handleRFQEmailBody(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().RFQEmailBody(w, r, id)
}

func handleAwardRFQPerLine(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().AwardRFQPerLine(w, r, id)
}

// getUser extracts the username from the request context/session.
// Kept for backward compatibility with any code that may reference it.
func getUser(r *http.Request) string {
	return getUsername(r)
}
