package main

import (
	"net/http"
)

func handleListTests(w http.ResponseWriter, r *http.Request) {
	getQualityHandler().ListTests(w, r)
}

func handleGetTests(w http.ResponseWriter, r *http.Request, serial string) {
	getQualityHandler().GetTests(w, r, serial)
}

func handleCreateTest(w http.ResponseWriter, r *http.Request) {
	getQualityHandler().CreateTest(w, r)
}

func handleGetTestByID(w http.ResponseWriter, r *http.Request, idStr string) {
	getQualityHandler().GetTestByID(w, r, idStr)
}
