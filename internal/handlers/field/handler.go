package field

import (
	"database/sql"

	"zrp/internal/websocket"
)

// Handler holds dependencies for field service handlers.
type Handler struct {
	DB  *sql.DB
	Hub *websocket.Hub

	// NextIDFunc generates the next sequential ID for a table.
	NextIDFunc func(prefix, table string, digits int) string

	// RecordChangeJSON records a change history entry.
	RecordChangeJSON func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error)

	// GetDeviceSnapshot returns a snapshot of a device row.
	GetDeviceSnapshot func(id string) (map[string]interface{}, error)

	// GetRMASnapshot returns a snapshot of an RMA row.
	GetRMASnapshot func(id string) (map[string]interface{}, error)
}
