package parts

import (
	"database/sql"
	"net/http"

	"zrp/internal/models"
	"zrp/internal/websocket"
)

// Handler holds dependencies for parts handlers.
type Handler struct {
	DB       *sql.DB
	Hub      *websocket.Hub
	PartsDir string

	// NextID generates the next sequential ID for a table. Set by the root package.
	NextID func(prefix, table string, digits int) string

	// EnsureInitialRevision creates the first revision for an ECO. Set by the root package.
	EnsureInitialRevision func(ecoID, user, now string)

	// SnapshotDocumentVersion snapshots a document version before changes. Set by the root package.
	SnapshotDocumentVersion func(docID, changeSummary, createdBy string, ecoID *string) error

	// HandleGetDoc handles the GET /api/documents/:id endpoint. Set by the root package.
	HandleGetDoc func(w http.ResponseWriter, r *http.Request, id string)

	// LoadPartsFromDir loads parts from the CSV-based parts directory.
	LoadPartsFromDir func() (map[string][]models.Part, map[string][]string, map[string]string, error)

	// GetPartByIPN looks up a part by IPN in the parts directory and returns its fields.
	GetPartByIPN func(partsDir, ipn string) (map[string]string, error)

	// LogSensitiveDataAccess logs access to sensitive data. Set by the root package.
	LogSensitiveDataAccess func(r *http.Request, dataType, recordID, details string)
}
