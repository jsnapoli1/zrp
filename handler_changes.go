package main

import (
	"net/http"
)

func recordChange(userID, tableName, recordID, operation, oldData, newData string) (int64, error) {
	return getCommonHandler().RecordChange(userID, tableName, recordID, operation, oldData, newData)
}

func recordChangeJSON(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error) {
	return getCommonHandler().RecordChangeJSON(userID, tableName, recordID, operation, oldData, newData)
}

func handleRecentChanges(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().RecentChanges(w, r)
}

func handleUndoChange(w http.ResponseWriter, r *http.Request, idStr string) {
	getCommonHandler().UndoChange(w, r, idStr)
}
