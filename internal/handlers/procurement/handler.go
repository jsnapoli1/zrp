package procurement

import (
	"database/sql"

	"zrp/internal/websocket"
)

// Handler holds dependencies for procurement handlers.
type Handler struct {
	DB         *sql.DB
	Hub        *websocket.Hub
	NextIDFunc func(prefix, table string, digits int) string

	// RecordChangeJSON records a change history entry. Set by the root package.
	RecordChangeJSON func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error)

	// GetVendorSnapshot returns a snapshot of a vendor row. Set by the root package.
	GetVendorSnapshot func(id string) (map[string]interface{}, error)

	// CreateUndoEntry creates an undo entry. Set by the root package.
	CreateUndoEntry func(username, action, entityType, entityID string) (int64, error)
}
