package main

import "net/http"

func handleListDocs(w http.ResponseWriter, r *http.Request) {
	getEngineeringHandler().ListDocs(w, r)
}

func handleGetDoc(w http.ResponseWriter, r *http.Request, id string) {
	getEngineeringHandler().GetDoc(w, r, id)
}

func handleCreateDoc(w http.ResponseWriter, r *http.Request) {
	getEngineeringHandler().CreateDoc(w, r)
}

func handleUpdateDoc(w http.ResponseWriter, r *http.Request, id string) {
	getEngineeringHandler().UpdateDoc(w, r, id)
}

func handleApproveDoc(w http.ResponseWriter, r *http.Request, id string) {
	getEngineeringHandler().ApproveDoc(w, r, id)
}
