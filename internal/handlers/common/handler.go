package common

import (
	"database/sql"
	"net/http"
)

// Handler holds dependencies for common/shared handlers.
type Handler struct {
	DB  *sql.DB

	// LoadPartsFromDir loads parts from the parts directory.
	LoadPartsFromDir func() (map[string][]Part, map[string][]string, map[string]string, error)

	// GetCurrentUser returns the current authenticated user info.
	GetCurrentUser func(r *http.Request) *UserInfo

	// ValidateFileUpload validates a file upload.
	ValidateFileUpload func(ve *ValidationErrors, filename string, size int64, contentType string)

	// SanitizeFilename sanitizes a filename.
	SanitizeFilename func(filename string) string

	// EmailConfigEnabled checks if email is configured and enabled.
	EmailConfigEnabled func() bool

	// SendNotificationEmail sends a notification email.
	SendNotificationEmail func(userID int, subject, body string)

	// LogAudit logs an audit event.
	LogAudit func(db *sql.DB, username, action, module, recordID, summary string)

	// GetUsername extracts the username from the request.
	GetUsername func(r *http.Request) string

	// LogDataExport logs a data export event.
	LogDataExport func(db *sql.DB, r *http.Request, module, format string, recordCount int)

	// ValidateAndSanitizeTable validates a table name for SQL safety.
	ValidateAndSanitizeTable func(table string) (string, error)

	// ValidateAndSanitizeColumn validates a column name for SQL safety.
	ValidateAndSanitizeColumn func(column string) (string, error)

	// Broadcast sends a WebSocket event.
	Broadcast func(evtType string, id interface{}, action string)

	// CtxUserID is the context key for extracting user ID.
	CtxUserID interface{}
}

// Part represents a part with IPN and field data.
type Part struct {
	IPN    string
	Fields map[string]string
}

// UserInfo contains basic user information for use in handlers.
type UserInfo struct {
	ID       int
	Username string
}

// ValidationErrors collects field validation errors.
type ValidationErrors struct {
	Errors []ValidationError
}

// ValidationError is a single field validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Add adds a validation error for a field.
func (ve *ValidationErrors) Add(field, message string) {
	ve.Errors = append(ve.Errors, ValidationError{Field: field, Message: message})
}

// HasErrors returns true if there are any errors.
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

// Error returns a string representation of all errors.
func (ve *ValidationErrors) Error() string {
	msg := ""
	for _, e := range ve.Errors {
		if msg != "" {
			msg += "; "
		}
		msg += e.Field + ": " + e.Message
	}
	return msg
}
