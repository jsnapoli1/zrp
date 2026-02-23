package validation

// Common enum values - these MUST match DB CHECK constraints in database package.
var (
	ValidECOStatuses           = []string{"draft", "review", "approved", "implemented", "rejected", "cancelled"}
	ValidECOPriorities         = []string{"low", "normal", "high", "critical"}
	ValidPOStatuses            = []string{"draft", "sent", "confirmed", "partial", "received", "cancelled"}
	ValidWOStatuses            = []string{"draft", "open", "in_progress", "completed", "cancelled", "on_hold"}
	ValidWOPriorities          = []string{"low", "normal", "high", "critical"}
	ValidNCRSeverities         = []string{"minor", "major", "critical"}
	ValidNCRStatuses           = []string{"open", "investigating", "resolved", "closed"}
	ValidRMAStatuses           = []string{"open", "received", "diagnosing", "repairing", "resolved", "closed", "scrapped"}
	ValidQuoteStatuses         = []string{"draft", "sent", "accepted", "rejected", "expired", "cancelled"}
	ValidShipmentTypes         = []string{"inbound", "outbound", "transfer"}
	ValidShipmentStatuses      = []string{"draft", "packed", "shipped", "delivered", "cancelled"}
	ValidDeviceStatuses        = []string{"active", "inactive", "rma", "decommissioned", "maintenance"}
	ValidCampaignStatuses      = []string{"draft", "active", "paused", "completed", "cancelled"}
	ValidCampaignDevStatuses   = []string{"pending", "in_progress", "success", "failed", "skipped"}
	ValidDocStatuses           = []string{"draft", "review", "approved", "released", "obsolete"}
	ValidCAPATypes             = []string{"corrective", "preventive"}
	ValidCAPAStatuses          = []string{"open", "in_progress", "pending_review", "closed", "cancelled"}
	ValidVendorStatuses        = []string{"active", "preferred", "inactive", "blocked"}
	ValidInventoryTypes        = []string{"receive", "issue", "adjust", "transfer", "return", "scrap"}
	ValidRFQStatuses           = []string{"draft", "sent", "quoting", "awarded", "cancelled"}
	ValidFieldReportTypes      = []string{"failure", "performance", "safety", "visit", "other"}
	ValidFieldReportStatuses   = []string{"open", "investigating", "resolved", "closed"}
	ValidSalesOrderStatuses    = []string{"draft", "confirmed", "allocated", "picked", "shipped", "invoiced", "closed"}
	ValidInvoiceStatuses       = []string{"draft", "sent", "paid", "overdue", "cancelled"}
	ValidFieldReportPriorities = []string{"low", "medium", "high", "critical"}
)
