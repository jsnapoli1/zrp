package main

import "net/http"

func handleReportInventoryValuation(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ReportInventoryValuation(w, r)
}

func handleReportOpenECOs(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ReportOpenECOs(w, r)
}

func handleReportWOThroughput(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ReportWOThroughput(w, r)
}

func handleReportLowStock(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ReportLowStock(w, r)
}

func handleReportNCRSummary(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ReportNCRSummary(w, r)
}
