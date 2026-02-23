package main

import (
	"net/http"

	"zrp/internal/handlers/parts"
)

// Type aliases for backward compatibility with tests and other root-level code.
type PartChange = parts.PartChange

func handleCreatePartChanges(w http.ResponseWriter, r *http.Request, ipn string) {
	getPartsHandler().CreatePartChanges(w, r, ipn)
}

func handleListPartChanges(w http.ResponseWriter, r *http.Request, ipn string) {
	getPartsHandler().ListPartChanges(w, r, ipn)
}

func handleDeletePartChange(w http.ResponseWriter, r *http.Request, ipn string, changeID string) {
	getPartsHandler().DeletePartChange(w, r, ipn, changeID)
}

func handleCreateECOFromChanges(w http.ResponseWriter, r *http.Request, ipn string) {
	getPartsHandler().CreateECOFromChanges(w, r, ipn)
}

func handleListECOPartChanges(w http.ResponseWriter, r *http.Request, ecoID string) {
	getPartsHandler().ListECOPartChanges(w, r, ecoID)
}

func applyPartChangesForECO(ecoID string) error {
	return getPartsHandler().ApplyPartChangesForECO(ecoID)
}

func rejectPartChangesForECO(ecoID string) {
	getPartsHandler().RejectPartChangesForECO(ecoID)
}

func handleListAllPartChanges(w http.ResponseWriter, r *http.Request) {
	getPartsHandler().ListAllPartChanges(w, r)
}

func updateBOMReferencesForPartIPN(pDir, oldIPN, newIPN string) error {
	return getPartsHandler().UpdateBOMReferencesForPartIPN(pDir, oldIPN, newIPN)
}
