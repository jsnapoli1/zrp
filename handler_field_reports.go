package main

import (
	"net/http"
)

func handleListFieldReports(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().ListFieldReports(w, r)
}

func handleGetFieldReport(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().GetFieldReport(w, r, id)
}

func handleCreateFieldReport(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().CreateFieldReport(w, r)
}

func handleUpdateFieldReport(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().UpdateFieldReport(w, r, id)
}

func handleDeleteFieldReport(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().DeleteFieldReport(w, r, id)
}

func handleFieldReportCreateNCR(w http.ResponseWriter, r *http.Request, id string) {
	getFieldHandler().FieldReportCreateNCR(w, r, id)
}
