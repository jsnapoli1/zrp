package main

import (
	"net/http"

	"zrp/internal/handlers/procurement"
	"zrp/internal/models"
	"zrp/internal/validation"
)

var procurementHandler *procurement.Handler

func initProcurementHandler() {
	procurementHandler = getProcurementHandler()
}

func getProcurementHandler() *procurement.Handler {
	if procurementHandler == nil || procurementHandler.DB != db {
		procurementHandler = &procurement.Handler{
			DB:                db,
			PartsDir:          partsDir,
			NextIDFunc:        nextID,
			RecordChangeJSON:  recordChangeJSON,
			GetVendorSnapshot: getVendorSnapshot,
			GetPOSnapshot:     getPOSnapshot,
			CreateUndoEntry:   createUndoEntry,
			ValidateForeignKey: func(ve *validation.ValidationErrors, field, table, id string) {
				if id == "" {
					return
				}
				sanitized, err := ValidateAndSanitizeTable(table)
				if err != nil {
					ve.Add(field, "invalid reference table")
					return
				}
				var count int
				if err := db.QueryRow("SELECT COUNT(*) FROM "+sanitized+" WHERE id=?", id).Scan(&count); err != nil || count == 0 {
					ve.Add(field, "referenced record not found")
				}
			},
			GetPartByIPN:      getPartByIPN,
			LoadPartsFromDir: func() (map[string][]models.Part, map[string][]string, map[string]string, error) {
				cats, schemas, titles, err := loadPartsFromDir()
				result := make(map[string][]models.Part)
				for k, parts := range cats {
					mparts := make([]models.Part, len(parts))
					for i, p := range parts {
						mparts[i] = models.Part{IPN: p.IPN, Fields: p.Fields}
					}
					result[k] = mparts
				}
				return result, schemas, titles, err
			},
			EmailOnPOReceived: emailOnPOReceived,
			RecordPriceFromPO: recordPriceFromPO,
			LogAudit: func(username, action, module, recordID, summary string) {
				logAudit(db, username, action, module, recordID, summary)
			},
			GetUsername: getUsername,
		}
	}
	return procurementHandler
}

func handleListVendors(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().ListVendors(w, r)
}

func handleGetVendor(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().GetVendor(w, r, id)
}

func handleCreateVendor(w http.ResponseWriter, r *http.Request) {
	getProcurementHandler().CreateVendor(w, r)
}

func handleUpdateVendor(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().UpdateVendor(w, r, id)
}

func handleDeleteVendor(w http.ResponseWriter, r *http.Request, id string) {
	getProcurementHandler().DeleteVendor(w, r, id)
}
