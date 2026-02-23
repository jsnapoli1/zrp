package main

import (
	"net/http"

	"zrp/internal/handlers/parts"
)

// Type aliases for backward compatibility with tests and other root-level code.
type GitDocsConfig = parts.GitDocsConfig

func handleGetGitDocsSettings(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().GetGitDocsSettings(w, r)
}

func handlePutGitDocsSettings(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().PutGitDocsSettings(w, r)
}

func handlePushDocToGit(w http.ResponseWriter, r *http.Request, docID string) {
	getPartsHandler().PushDocToGit(w, r, docID)
}

func handleSyncDocFromGit(w http.ResponseWriter, r *http.Request, docID string) {
	getPartsHandler().SyncDocFromGit(w, r, docID)
}

func handleCreateECOPR(w http.ResponseWriter, r *http.Request, ecoID string) {
	getPartsHandler().CreateECOPR(w, r, ecoID)
}
