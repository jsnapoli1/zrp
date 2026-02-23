package sales

import (
	"database/sql"
	"net/http"

	"zrp/internal/websocket"
)

// NextIDFunc generates a sequential ID with the given prefix and table.
type NextIDFunc func(prefix, table string, digits int) string

// RecordChangeJSONFunc records a change history entry.
type RecordChangeJSONFunc func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error)

// GetQuoteSnapshotFunc returns a snapshot of a quote row.
type GetQuoteSnapshotFunc func(id string) (map[string]interface{}, error)

// GenerateInvoiceNumberFunc generates a unique invoice number.
type GenerateInvoiceNumberFunc func() string

// GetSalesOrderLinesFunc retrieves sales order lines (used by invoices from SO).
type GetSalesOrderLinesFunc func(orderID string) []SalesOrderLineCompat

// SalesOrderLineCompat is a minimal interface for sales order lines passed from root package.
type SalesOrderLineCompat struct {
	ID           int
	SalesOrderID string
	IPN          string
	Description  string
	Qty          int
	QtyAllocated int
	QtyPicked    int
	QtyShipped   int
	UnitPrice    float64
	Notes        string
}

// HandleGetQuoteFunc allows invoking the root-level handleGetQuote for chaining.
type HandleGetQuoteFunc func(w http.ResponseWriter, r *http.Request, id string)

// HandleGetSalesOrderFunc allows invoking the root-level handleGetSalesOrder for chaining.
type HandleGetSalesOrderFunc func(w http.ResponseWriter, r *http.Request, id string)

// HandleGetShipmentFunc allows invoking the root-level handleGetShipment for chaining.
type HandleGetShipmentFunc func(w http.ResponseWriter, r *http.Request, id string)

// Handler holds dependencies for sales handlers.
type Handler struct {
	DB  *sql.DB
	Hub *websocket.Hub

	// Function fields set by the root package.
	NextID              NextIDFunc
	RecordChangeJSON    RecordChangeJSONFunc
	GetQuoteSnapshot    GetQuoteSnapshotFunc
	GenerateInvoiceNum  GenerateInvoiceNumberFunc
	CompanyName         string
	CompanyEmail        string
}
