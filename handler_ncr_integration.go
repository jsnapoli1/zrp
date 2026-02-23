package main

import (
	"net/http"
)

func handleCreateCAPAFromNCR(w http.ResponseWriter, r *http.Request, ncrID string) {
	getQualityHandler().CreateCAPAFromNCR(w, r, ncrID)
}

func handleCreateECOFromNCR(w http.ResponseWriter, r *http.Request, ncrID string) {
	getQualityHandler().CreateECOFromNCR(w, r, ncrID)
}
