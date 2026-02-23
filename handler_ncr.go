package main

import (
	"database/sql"
	"net/http"

	"zrp/internal/handlers/quality"
	"zrp/internal/models"
)

// qualityHandler is the shared quality handler instance.
var qualityHandler *quality.Handler

// getQualityHandler returns the quality handler, lazily initializing if needed (for tests).
func getQualityHandler() *quality.Handler {
	if qualityHandler == nil || qualityHandler.DB != db {
		qualityHandler = &quality.Handler{
			DB:               db,
			Hub:              wsHub,
			NextIDFunc:       nextID,
			RecordChangeJSON: recordChangeJSON,
			GetNCRSnapshot:   getNCRSnapshot,
			GetCAPASnapshot:  getCAPASnapshot,
			GetUserID:        getUserID,
			GetUserRole:      getUserRole,
			CanApproveCAPA:   canApproveCAPA,
			EmailOnNCRCreated: emailOnNCRCreated,
			EmailOnCAPACreated: func(c models.CAPA) {
				emailOnCAPACreated(CAPA{
					ID: c.ID, Title: c.Title, Type: c.Type,
					LinkedNCRID: c.LinkedNCRID, LinkedRMAID: c.LinkedRMAID,
					RootCause: c.RootCause, ActionPlan: c.ActionPlan,
					Owner: c.Owner, DueDate: c.DueDate, Status: c.Status,
					EffectivenessCheck: c.EffectivenessCheck,
					ApprovedByQE: c.ApprovedByQE, ApprovedByQEAt: c.ApprovedByQEAt,
					ApprovedByMgr: c.ApprovedByMgr, ApprovedByMgrAt: c.ApprovedByMgrAt,
					CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
				})
			},
			EmailOnCAPACreatedWithDB: func(database *sql.DB, c models.CAPA) {
				emailOnCAPACreatedWithDB(database, CAPA{
					ID: c.ID, Title: c.Title, Type: c.Type,
					LinkedNCRID: c.LinkedNCRID, LinkedRMAID: c.LinkedRMAID,
					RootCause: c.RootCause, ActionPlan: c.ActionPlan,
					Owner: c.Owner, DueDate: c.DueDate, Status: c.Status,
					EffectivenessCheck: c.EffectivenessCheck,
					ApprovedByQE: c.ApprovedByQE, ApprovedByQEAt: c.ApprovedByQEAt,
					ApprovedByMgr: c.ApprovedByMgr, ApprovedByMgrAt: c.ApprovedByMgrAt,
					CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
				})
			},
			SendEmail: sendEmail,
		}
	}
	return qualityHandler
}

func handleListNCRs(w http.ResponseWriter, r *http.Request) {
	getQualityHandler().ListNCRs(w, r)
}

func handleGetNCR(w http.ResponseWriter, r *http.Request, id string) {
	getQualityHandler().GetNCR(w, r, id)
}

func handleCreateNCR(w http.ResponseWriter, r *http.Request) {
	getQualityHandler().CreateNCR(w, r)
}

func handleUpdateNCR(w http.ResponseWriter, r *http.Request, id string) {
	getQualityHandler().UpdateNCR(w, r, id)
}
