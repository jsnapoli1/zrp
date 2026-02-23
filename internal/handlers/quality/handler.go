package quality

import (
	"database/sql"
	"net/http"

	"zrp/internal/models"
	"zrp/internal/websocket"
)

// Handler holds dependencies for quality handlers.
type Handler struct {
	DB  *sql.DB
	Hub *websocket.Hub

	// NextIDFunc generates sequential IDs (e.g. "NCR-001").
	NextIDFunc func(prefix, table string, digits int) string

	// RecordChangeJSON records a change history entry.
	RecordChangeJSON func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error)

	// GetNCRSnapshot returns a snapshot of an NCR row.
	GetNCRSnapshot func(id string) (map[string]interface{}, error)

	// GetCAPASnapshot returns a snapshot of a CAPA row.
	GetCAPASnapshot func(id string) (map[string]interface{}, error)

	// GetUserID returns the authenticated user's ID from the request.
	GetUserID func(r *http.Request) (int, error)

	// GetUserRole returns the authenticated user's role from the request.
	GetUserRole func(r *http.Request) string

	// CanApproveCAPA checks if the request user can approve a CAPA.
	CanApproveCAPA func(r *http.Request, approvalType string) bool

	// EmailOnNCRCreated sends email notification when an NCR is created.
	EmailOnNCRCreated func(ncrID, title string)

	// EmailOnCAPACreated sends email notification when a CAPA is created.
	EmailOnCAPACreated func(c models.CAPA)

	// EmailOnCAPACreatedWithDB sends email notification when a CAPA is created (with DB).
	EmailOnCAPACreatedWithDB func(database *sql.DB, c models.CAPA)

	// SendEmail sends an email.
	SendEmail func(to, subject, body string) error
}
