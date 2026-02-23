package main

import (
	"net/http"
)

func handleListUndo(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ListUndo(w, r)
}

func handlePerformUndo(w http.ResponseWriter, r *http.Request, idStr string) {
	getCommonHandler().HandlePerformUndo(w, r, idStr)
}

func cleanExpiredUndo() {
	getCommonHandler().CleanExpiredUndo()
}

// createUndoEntry is a thin wrapper used by bulk handlers.
func createUndoEntry(username, action, entityType, entityID string) (int64, error) {
	return getCommonHandler().CreateUndoEntry(username, action, entityType, entityID)
}
