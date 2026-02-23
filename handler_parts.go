package main

import (
	"net/http"

	"zrp/internal/handlers/parts"
)

// partsHandler is the shared parts handler instance.
var partsHandler *parts.Handler

// getPartsHandler returns the parts handler, lazily initializing if needed (for tests).
func getPartsHandler() *parts.Handler {
	if partsHandler == nil || partsHandler.DB != db {
		partsHandler = &parts.Handler{
			DB:                      db,
			Hub:                     wsHub,
			PartsDir:                partsDir,
			NextID:                  nextID,
			EnsureInitialRevision:   ensureInitialRevision,
			SnapshotDocumentVersion: snapshotDocumentVersion,
			HandleGetDoc:            handleGetDoc,
			LogSensitiveDataAccess: func(r *http.Request, dataType, recordID, details string) {
				LogSensitiveDataAccess(db, r, dataType, recordID, details)
			},
		}
		partsHandler.LoadPartsFromDir = partsHandler.LoadPartsFromDirImpl
		partsHandler.GetPartByIPN = partsHandler.GetPartByIPNImpl
	}
	return partsHandler
}

// Type aliases for backward compatibility with tests and other root-level code.
type BOMNode = parts.BOMNode

func loadPartsFromDir() (map[string][]Part, map[string][]string, map[string]string, error) {
	return getPartsHandler().LoadPartsFromDir()
}

func getPartByIPN(pmDir, ipn string) (map[string]string, error) {
	return getPartsHandler().GetPartByIPN(pmDir, ipn)
}

func handleListParts(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().ListParts(w, r)
}

func handleGetPart(w http.ResponseWriter, r *http.Request, ipn string) {
	getPartsHandler().GetPart(w, r, ipn)
}

func handleCreatePart(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().CreatePart(w, r)
}

func handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().CreateCategory(w, r)
}

func handleCheckIPN(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().CheckIPN(w, r)
}

func handleUpdatePart(w http.ResponseWriter, r *http.Request, ipn string) {
	getPartsHandler().UpdatePart(w, r, ipn)
}

func handleDeletePart(w http.ResponseWriter, r *http.Request, ipn string) {
	getPartsHandler().DeletePart(w, r, ipn)
}

func handleListCategories(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().ListCategories(w, r)
}

func handleAddColumn(w http.ResponseWriter, r *http.Request, catID string) {
	getPartsHandler().AddColumn(w, r, catID)
}

func handleDeleteColumn(w http.ResponseWriter, r *http.Request, catID, colName string) {
	getPartsHandler().DeleteColumn(w, r, catID, colName)
}

func handlePartBOM(w http.ResponseWriter, r *http.Request, ipn string) {
	getPartsHandler().PartBOM(w, r, ipn)
}

func handlePartCost(w http.ResponseWriter, r *http.Request, ipn string) {
	getPartsHandler().PartCost(w, r, ipn)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().Dashboard(w, r)
}

func readCSV(path string, category string) ([]Part, []string, string, error) {
	return parts.ReadCSV(path, category)
}

func findCategoryCSV(category string) string {
	return getPartsHandler().FindCategoryCSV(category)
}
