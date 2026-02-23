package procurement

import (
	"database/sql"
	"net/http"

	"zrp/internal/models"
	"zrp/internal/validation"
)

// Handler holds dependencies for procurement handlers.
type Handler struct {
	DB       *sql.DB
	PartsDir string

	NextIDFunc func(prefix, table string, digits int) string

	// RecordChangeJSON records a change history entry. Set by the root package.
	RecordChangeJSON func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error)

	// GetVendorSnapshot returns a snapshot of a vendor row. Set by the root package.
	GetVendorSnapshot func(id string) (map[string]interface{}, error)

	// GetPOSnapshot returns a snapshot of a PO row with lines. Set by the root package.
	GetPOSnapshot func(id string) (map[string]interface{}, error)

	// CreateUndoEntry creates an undo entry. Set by the root package.
	CreateUndoEntry func(username, action, entityType, entityID string) (int64, error)

	// ValidateForeignKey validates that a referenced record exists. Set by the root package.
	ValidateForeignKey func(ve *validation.ValidationErrors, field, table, id string)

	// GetPartByIPN looks up a part by IPN in the parts directory. Set by the root package.
	GetPartByIPN func(partsDir, ipn string) (map[string]string, error)

	// LoadPartsFromDir loads parts from the parts directory. Set by the root package.
	LoadPartsFromDir func() (map[string][]models.Part, map[string][]string, map[string]string, error)

	// EmailOnPOReceived sends email notification when a PO is received. Set by the root package.
	EmailOnPOReceived func(poID string)

	// RecordPriceFromPO records a price history entry from a PO line. Set by the root package.
	RecordPriceFromPO func(poID, ipn string, unitPrice float64, vendorID string)

	// LogAudit logs an audit event. Set by the root package.
	LogAudit func(username, action, module, recordID, summary string)

	// GetUsername extracts the username from the request. Set by the root package.
	GetUsername func(r *http.Request) string
}
