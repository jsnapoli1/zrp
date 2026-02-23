package engineering

import (
	"database/sql"

	"zrp/internal/websocket"
)

// PartLookupFunc looks up a part by IPN and returns its fields.
type PartLookupFunc func(partsDir, ipn string) (map[string]string, error)

// GetPartMPNFunc returns the MPN for a given IPN.
type GetPartMPNFunc func(ipn string) string

// GetAppSettingFunc reads a setting value by key.
type GetAppSettingFunc func(key string) string

// SetAppSettingFunc writes a setting value by key.
type SetAppSettingFunc func(key, value string) error

// Handler holds dependencies for engineering handlers.
type Handler struct {
	DB       *sql.DB
	Hub      *websocket.Hub
	PartsDir string

	// NextIDFunc generates the next sequential ID for a table.
	NextIDFunc func(prefix, table string, digits int) string

	// RecordChangeJSON records a change history entry.
	RecordChangeJSON func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error)

	// GetECOSnapshot returns a snapshot of an ECO row.
	GetECOSnapshot func(id string) (map[string]interface{}, error)

	// GetPartByIPN looks up a part by IPN in the parts directory.
	GetPartByIPN PartLookupFunc

	// EmailOnECOApproved sends notification when an ECO is approved.
	EmailOnECOApproved func(id string)

	// EmailOnECOImplemented sends notification when an ECO is implemented.
	EmailOnECOImplemented func(id string)

	// ApplyPartChangesForECO applies linked part changes for an ECO.
	ApplyPartChangesForECO func(id string) error

	// GetPartMPN returns the MPN for a given IPN.
	GetPartMPN GetPartMPNFunc

	// GetAppSetting reads a setting value by key.
	GetAppSetting GetAppSettingFunc

	// SetAppSetting writes a setting value by key.
	SetAppSetting SetAppSettingFunc
}
