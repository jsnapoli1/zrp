package main

import (
	"net/http"

	"zrp/internal/handlers/engineering"
)

var engineeringHandler *engineering.Handler

func getEngineeringHandler() *engineering.Handler {
	if engineeringHandler == nil || engineeringHandler.DB != db {
		engineeringHandler = &engineering.Handler{
			DB:                     db,
			Hub:                    wsHub,
			PartsDir:               partsDir,
			NextIDFunc:             nextID,
			RecordChangeJSON:       recordChangeJSON,
			GetECOSnapshot:         getECOSnapshot,
			GetPartByIPN:           getPartByIPN,
			EmailOnECOApproved:     emailOnECOApproved,
			EmailOnECOImplemented:  emailOnECOImplemented,
			ApplyPartChangesForECO: applyPartChangesForECO,
			GetPartMPN:             getPartMPN,
			GetAppSetting:          getAppSetting,
			SetAppSetting:          setAppSetting,
		}
	}
	return engineeringHandler
}

func handleListECOs(w http.ResponseWriter, r *http.Request) {
	getEngineeringHandler().ListECOs(w, r)
}

func handleGetECO(w http.ResponseWriter, r *http.Request, id string) {
	getEngineeringHandler().GetECO(w, r, id)
}

func handleCreateECO(w http.ResponseWriter, r *http.Request) {
	getEngineeringHandler().CreateECO(w, r)
}

func handleUpdateECO(w http.ResponseWriter, r *http.Request, id string) {
	getEngineeringHandler().UpdateECO(w, r, id)
}

func handleApproveECO(w http.ResponseWriter, r *http.Request, id string) {
	getEngineeringHandler().ApproveECO(w, r, id)
}

func handleImplementECO(w http.ResponseWriter, r *http.Request, id string) {
	getEngineeringHandler().ImplementECO(w, r, id)
}

func handleListECORevisions(w http.ResponseWriter, r *http.Request, ecoID string) {
	getEngineeringHandler().ListECORevisions(w, r, ecoID)
}

func handleCreateECORevision(w http.ResponseWriter, r *http.Request, ecoID string) {
	getEngineeringHandler().CreateECORevision(w, r, ecoID)
}

func handleGetECORevision(w http.ResponseWriter, r *http.Request, ecoID, revLetter string) {
	getEngineeringHandler().GetECORevision(w, r, ecoID, revLetter)
}

// ensureInitialRevision delegates to the engineering handler.
func ensureInitialRevision(ecoID, user, now string) {
	getEngineeringHandler().EnsureInitialRevision(ecoID, user, now)
}
