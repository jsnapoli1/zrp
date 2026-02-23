package models

import "time"

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	Data interface{} `json:"data"`
	Meta *Meta       `json:"meta,omitempty"`
}

// Meta contains pagination metadata.
type Meta struct {
	Total int `json:"total,omitempty"`
	Page  int `json:"page,omitempty"`
	Limit int `json:"limit,omitempty"`
}

type ECO struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Status       string  `json:"status"`
	Priority     string  `json:"priority"`
	AffectedIPNs string  `json:"affected_ipns"`
	CreatedBy    string  `json:"created_by"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	ApprovedAt   *string `json:"approved_at"`
	ApprovedBy   *string `json:"approved_by"`
	NcrID        string  `json:"ncr_id,omitempty"`
}

type ECORevision struct {
	ID              int     `json:"id"`
	ECOID           string  `json:"eco_id"`
	Revision        string  `json:"revision"`
	Status          string  `json:"status"`
	ChangesSummary  string  `json:"changes_summary"`
	CreatedBy       string  `json:"created_by"`
	CreatedAt       string  `json:"created_at"`
	ApprovedBy      *string `json:"approved_by"`
	ApprovedAt      *string `json:"approved_at"`
	ImplementedBy   *string `json:"implemented_by"`
	ImplementedAt   *string `json:"implemented_at"`
	EffectivityDate *string `json:"effectivity_date"`
	Notes           string  `json:"notes"`
}

type Document struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Category  string `json:"category"`
	IPN       string `json:"ipn"`
	Revision  string `json:"revision"`
	Status    string `json:"status"`
	Content   string `json:"content"`
	FilePath  string `json:"file_path"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Vendor struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Website      string `json:"website"`
	ContactName  string `json:"contact_name"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	Notes        string `json:"notes"`
	Status       string `json:"status"`
	LeadTimeDays int    `json:"lead_time_days"`
	CreatedAt    string `json:"created_at"`
}

type InventoryItem struct {
	IPN          string  `json:"ipn"`
	QtyOnHand    float64 `json:"qty_on_hand"`
	QtyReserved  float64 `json:"qty_reserved"`
	Location     string  `json:"location"`
	ReorderPoint float64 `json:"reorder_point"`
	ReorderQty   float64 `json:"reorder_qty"`
	Description  string  `json:"description"`
	MPN          string  `json:"mpn"`
	UpdatedAt    string  `json:"updated_at"`
}

type InventoryTransaction struct {
	ID        int     `json:"id"`
	IPN       string  `json:"ipn"`
	Type      string  `json:"type"`
	Qty       float64 `json:"qty"`
	Reference string  `json:"reference"`
	Notes     string  `json:"notes"`
	CreatedAt string  `json:"created_at"`
}

type PurchaseOrder struct {
	ID           string   `json:"id"`
	VendorID     string   `json:"vendor_id"`
	Status       string   `json:"status"`
	Notes        string   `json:"notes"`
	CreatedAt    string   `json:"created_at"`
	ExpectedDate string   `json:"expected_date"`
	ReceivedAt   *string  `json:"received_at"`
	Lines        []POLine `json:"lines,omitempty"`
}

type POLine struct {
	ID           int     `json:"id"`
	POID         string  `json:"po_id"`
	IPN          string  `json:"ipn"`
	MPN          string  `json:"mpn"`
	Manufacturer string  `json:"manufacturer"`
	QtyOrdered   float64 `json:"qty_ordered"`
	QtyReceived  float64 `json:"qty_received"`
	UnitPrice    float64 `json:"unit_price"`
	Notes        string  `json:"notes"`
}

type WorkOrder struct {
	ID          string  `json:"id"`
	AssemblyIPN string  `json:"assembly_ipn"`
	Qty         int     `json:"qty"`
	QtyGood     *int    `json:"qty_good"`
	QtyScrap    *int    `json:"qty_scrap"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	Notes       string  `json:"notes"`
	CreatedAt   string  `json:"created_at"`
	StartedAt   *string `json:"started_at"`
	CompletedAt *string `json:"completed_at"`
}

type WOSerial struct {
	ID           int    `json:"id"`
	WOID         string `json:"wo_id"`
	SerialNumber string `json:"serial_number"`
	Status       string `json:"status"`
	Notes        string `json:"notes"`
}

type TestRecord struct {
	ID              int    `json:"id"`
	SerialNumber    string `json:"serial_number"`
	IPN             string `json:"ipn"`
	FirmwareVersion string `json:"firmware_version"`
	TestType        string `json:"test_type"`
	Result          string `json:"result"`
	Measurements    string `json:"measurements"`
	Notes           string `json:"notes"`
	TestedBy        string `json:"tested_by"`
	TestedAt        string `json:"tested_at"`
}

type FieldReport struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	ReportType   string  `json:"report_type"`
	Status       string  `json:"status"`
	Priority     string  `json:"priority"`
	CustomerName string  `json:"customer_name"`
	SiteLocation string  `json:"site_location"`
	DeviceIPN    string  `json:"device_ipn"`
	DeviceSerial string  `json:"device_serial"`
	ReportedBy   string  `json:"reported_by"`
	ReportedAt   string  `json:"reported_at"`
	Description  string  `json:"description"`
	RootCause    string  `json:"root_cause"`
	Resolution   string  `json:"resolution"`
	ResolvedAt   *string `json:"resolved_at"`
	NcrID        string  `json:"ncr_id,omitempty"`
	EcoID        string  `json:"eco_id,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type NCR struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	IPN              string  `json:"ipn"`
	SerialNumber     string  `json:"serial_number"`
	DefectType       string  `json:"defect_type"`
	Severity         string  `json:"severity"`
	Status           string  `json:"status"`
	RootCause        string  `json:"root_cause"`
	CorrectiveAction string  `json:"corrective_action"`
	CreatedBy        string  `json:"created_by"`
	CreatedAt        string  `json:"created_at"`
	ResolvedAt       *string `json:"resolved_at"`
}

type Device struct {
	SerialNumber    string  `json:"serial_number"`
	IPN             string  `json:"ipn"`
	FirmwareVersion string  `json:"firmware_version"`
	Customer        string  `json:"customer"`
	Location        string  `json:"location"`
	Status          string  `json:"status"`
	InstallDate     string  `json:"install_date"`
	LastSeen        *string `json:"last_seen"`
	Notes           string  `json:"notes"`
	CreatedAt       string  `json:"created_at"`
}

type FirmwareCampaign struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Version      string  `json:"version"`
	Category     string  `json:"category"`
	Status       string  `json:"status"`
	TargetFilter string  `json:"target_filter"`
	Notes        string  `json:"notes"`
	CreatedAt    string  `json:"created_at"`
	StartedAt    *string `json:"started_at"`
	CompletedAt  *string `json:"completed_at"`
}

type CampaignDevice struct {
	CampaignID   string  `json:"campaign_id"`
	SerialNumber string  `json:"serial_number"`
	Status       string  `json:"status"`
	UpdatedAt    *string `json:"updated_at"`
}

type RMA struct {
	ID                string  `json:"id"`
	SerialNumber      string  `json:"serial_number"`
	Customer          string  `json:"customer"`
	Reason            string  `json:"reason"`
	Status            string  `json:"status"`
	DefectDescription string  `json:"defect_description"`
	Resolution        string  `json:"resolution"`
	CreatedAt         string  `json:"created_at"`
	ReceivedAt        *string `json:"received_at"`
	ResolvedAt        *string `json:"resolved_at"`
}

type Quote struct {
	ID         string      `json:"id"`
	Customer   string      `json:"customer"`
	Status     string      `json:"status"`
	Notes      string      `json:"notes"`
	CreatedAt  string      `json:"created_at"`
	ValidUntil string      `json:"valid_until"`
	AcceptedAt *string     `json:"accepted_at"`
	Lines      []QuoteLine `json:"lines,omitempty"`
}

type QuoteLine struct {
	ID          int     `json:"id"`
	QuoteID     string  `json:"quote_id"`
	IPN         string  `json:"ipn"`
	Description string  `json:"description"`
	Qty         int     `json:"qty"`
	UnitPrice   float64 `json:"unit_price"`
	Notes       string  `json:"notes"`
}

type DashboardData struct {
	OpenECOs     int `json:"open_ecos"`
	LowStock     int `json:"low_stock"`
	OpenPOs      int `json:"open_pos"`
	ActiveWOs    int `json:"active_wos"`
	OpenNCRs     int `json:"open_ncrs"`
	OpenRMAs     int `json:"open_rmas"`
	TotalParts   int `json:"total_parts"`
	TotalDevices int `json:"total_devices"`
}

// Part represents a gitplm part from CSV.
type Part struct {
	IPN    string            `json:"ipn"`
	Fields map[string]string `json:"fields"`
}

type Category struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Count   int      `json:"count"`
	Columns []string `json:"columns"`
}

type Shipment struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	TrackingNumber string         `json:"tracking_number"`
	Carrier        string         `json:"carrier"`
	ShipDate       *string        `json:"ship_date"`
	DeliveryDate   *string        `json:"delivery_date"`
	FromAddress    string         `json:"from_address"`
	ToAddress      string         `json:"to_address"`
	Notes          string         `json:"notes"`
	CreatedBy      string         `json:"created_by"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	Lines          []ShipmentLine `json:"lines,omitempty"`
}

type ShipmentLine struct {
	ID           int    `json:"id"`
	ShipmentID   string `json:"shipment_id"`
	IPN          string `json:"ipn"`
	SerialNumber string `json:"serial_number"`
	Qty          int    `json:"qty"`
	WorkOrderID  string `json:"work_order_id"`
	RMAID        string `json:"rma_id"`
}

type PackList struct {
	ID         int            `json:"id"`
	ShipmentID string         `json:"shipment_id"`
	CreatedAt  string         `json:"created_at"`
	Lines      []ShipmentLine `json:"lines,omitempty"`
}

type RFQ struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Status    string      `json:"status"`
	CreatedBy string      `json:"created_by"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
	DueDate   string      `json:"due_date"`
	Notes     string      `json:"notes"`
	Lines     []RFQLine   `json:"lines,omitempty"`
	Vendors   []RFQVendor `json:"vendors,omitempty"`
	Quotes    []RFQQuote  `json:"quotes,omitempty"`
}

type RFQLine struct {
	ID          int     `json:"id"`
	RFQID       string  `json:"rfq_id"`
	IPN         string  `json:"ipn"`
	Description string  `json:"description"`
	Qty         float64 `json:"qty"`
	Unit        string  `json:"unit"`
}

type RFQVendor struct {
	ID         int    `json:"id"`
	RFQID      string `json:"rfq_id"`
	VendorID   string `json:"vendor_id"`
	VendorName string `json:"vendor_name,omitempty"`
	Status     string `json:"status"`
	QuotedAt   string `json:"quoted_at"`
	Notes      string `json:"notes"`
}

type RFQQuote struct {
	ID           int     `json:"id"`
	RFQID        string  `json:"rfq_id"`
	RFQVendorID  int     `json:"rfq_vendor_id"`
	RFQLineID    int     `json:"rfq_line_id"`
	UnitPrice    float64 `json:"unit_price"`
	LeadTimeDays int     `json:"lead_time_days"`
	MOQ          int     `json:"moq"`
	Notes        string  `json:"notes"`
}

// DocumentVersion represents a snapshot of a document at a specific revision.
type DocumentVersion struct {
	ID            int     `json:"id"`
	DocumentID    string  `json:"document_id"`
	Revision      string  `json:"revision"`
	Content       string  `json:"content"`
	FilePath      string  `json:"file_path"`
	ChangeSummary string  `json:"change_summary"`
	Status        string  `json:"status"`
	CreatedBy     string  `json:"created_by"`
	CreatedAt     string  `json:"created_at"`
	ECOID         *string `json:"eco_id"`
}

type SalesOrder struct {
	ID         string           `json:"id"`
	QuoteID    string           `json:"quote_id"`
	Customer   string           `json:"customer"`
	Status     string           `json:"status"`
	Notes      string           `json:"notes"`
	CreatedBy  string           `json:"created_by"`
	CreatedAt  string           `json:"created_at"`
	UpdatedAt  string           `json:"updated_at"`
	ShipmentID *string          `json:"shipment_id,omitempty"`
	InvoiceID  *string          `json:"invoice_id,omitempty"`
	Lines      []SalesOrderLine `json:"lines,omitempty"`
}

type SalesOrderLine struct {
	ID           int     `json:"id"`
	SalesOrderID string  `json:"sales_order_id"`
	IPN          string  `json:"ipn"`
	Description  string  `json:"description"`
	Qty          int     `json:"qty"`
	QtyAllocated int     `json:"qty_allocated"`
	QtyPicked    int     `json:"qty_picked"`
	QtyShipped   int     `json:"qty_shipped"`
	UnitPrice    float64 `json:"unit_price"`
	Notes        string  `json:"notes"`
}

type Invoice struct {
	ID            string        `json:"id"`
	InvoiceNumber string        `json:"invoice_number"`
	SalesOrderID  string        `json:"sales_order_id"`
	Customer      string        `json:"customer"`
	IssueDate     string        `json:"issue_date"`
	DueDate       string        `json:"due_date"`
	Status        string        `json:"status"`
	Total         float64       `json:"total"`
	Tax           float64       `json:"tax"`
	Notes         string        `json:"notes"`
	CreatedAt     string        `json:"created_at"`
	PaidAt        *string       `json:"paid_at,omitempty"`
	Lines         []InvoiceLine `json:"lines,omitempty"`
}

type InvoiceLine struct {
	ID          int     `json:"id"`
	InvoiceID   string  `json:"invoice_id"`
	IPN         string  `json:"ipn"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Total       float64 `json:"total"`
}

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	Action      string `json:"action"`
	Module      string `json:"module"`
	RecordID    string `json:"record_id"`
	Summary     string `json:"summary"`
	BeforeValue string `json:"before_value,omitempty"`
	AfterValue  string `json:"after_value,omitempty"`
	IPAddress   string `json:"ip_address,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// Permission represents a single permission assignment.
type Permission struct {
	ID     int    `json:"id"`
	Role   string `json:"role"`
	Module string `json:"module"`
	Action string `json:"action"`
}

// ProductPricing represents a product pricing tier entry.
type ProductPricing struct {
	ID            int     `json:"id"`
	ProductIPN    string  `json:"product_ipn"`
	PricingTier   string  `json:"pricing_tier"`
	MinQty        int     `json:"min_qty"`
	MaxQty        int     `json:"max_qty"`
	UnitPrice     float64 `json:"unit_price"`
	Currency      string  `json:"currency"`
	EffectiveDate string  `json:"effective_date"`
	ExpiryDate    string  `json:"expiry_date"`
	Notes         string  `json:"notes"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// CostAnalysis represents a product cost breakdown.
type CostAnalysis struct {
	ID             int     `json:"id"`
	ProductIPN     string  `json:"product_ipn"`
	BOMCost        float64 `json:"bom_cost"`
	LaborCost      float64 `json:"labor_cost"`
	OverheadCost   float64 `json:"overhead_cost"`
	TotalCost      float64 `json:"total_cost"`
	MarginPct      float64 `json:"margin_pct"`
	LastCalculated string  `json:"last_calculated"`
	CreatedAt      string  `json:"created_at"`
}

// CAPA represents a Corrective/Preventive Action record.
type CAPA struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	Type               string  `json:"type"`
	LinkedNCRID        string  `json:"linked_ncr_id"`
	LinkedRMAID        string  `json:"linked_rma_id"`
	RootCause          string  `json:"root_cause"`
	ActionPlan         string  `json:"action_plan"`
	Owner              string  `json:"owner"`
	DueDate            string  `json:"due_date"`
	Status             string  `json:"status"`
	EffectivenessCheck string  `json:"effectiveness_check"`
	ApprovedByQE       string  `json:"approved_by_qe"`
	ApprovedByQEAt     *string `json:"approved_by_qe_at"`
	ApprovedByMgr      string  `json:"approved_by_mgr"`
	ApprovedByMgrAt    *string `json:"approved_by_mgr_at"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

var _ = time.Now // keep time imported
