package main

import (
	"database/sql"
	"net/http"

	"zrp/internal/handlers/common"
)

// commonHandler is the shared common handler instance.
var commonHandler *common.Handler

// getCommonHandler returns the common handler, lazily initializing if needed.
func getCommonHandler() *common.Handler {
	if commonHandler == nil || commonHandler.DB != db {
		commonHandler = &common.Handler{
			DB: db,
			LoadPartsFromDir: func() (map[string][]common.Part, map[string][]string, map[string]string, error) {
				cats, catOrder, catDesc, err := loadPartsFromDir()
				// Convert []Part to []common.Part
				result := make(map[string][]common.Part)
				for k, parts := range cats {
					cparts := make([]common.Part, len(parts))
					for i, p := range parts {
						cparts[i] = common.Part{IPN: p.IPN, Fields: p.Fields}
					}
					result[k] = cparts
				}
				return result, catOrder, catDesc, err
			},
			GetCurrentUser: func(r *http.Request) *common.UserInfo {
				u := getCurrentUser(r)
				if u == nil {
					return nil
				}
				return &common.UserInfo{ID: u.ID, Username: u.Username}
			},
			ValidateFileUpload: func(ve *common.ValidationErrors, filename string, size int64, contentType string) {
				// Bridge common.ValidationErrors to root ValidationErrors
				rootVE := &ValidationErrors{}
				validateFileUpload(rootVE, filename, size, contentType)
				for _, e := range rootVE.Errors {
					ve.Add(e.Field, e.Message)
				}
			},
			SanitizeFilename: sanitizeFilename,
			EmailConfigEnabled: emailConfigEnabled,
			SendNotificationEmail: func(notifID int, title, message string) {
				sendNotificationEmail(notifID, title, message)
			},
			LogAudit: func(database *sql.DB, username, action, module, recordID, summary string) {
				logAudit(database, username, action, module, recordID, summary)
			},
			GetUsername:            getUsername,
			LogDataExport: func(database *sql.DB, r *http.Request, module, format string, recordCount int) {
				LogDataExport(database, r, module, format, recordCount)
			},
			ValidateAndSanitizeTable:  ValidateAndSanitizeTable,
			ValidateAndSanitizeColumn: ValidateAndSanitizeColumn,
			Broadcast: func(evtType string, id interface{}, action string) {
				wsHub.Broadcast(WSEvent{
					Type:   evtType,
					ID:     id,
					Action: action,
				})
			},
			CtxUserID: ctxUserID,
		}
	}
	return commonHandler
}
