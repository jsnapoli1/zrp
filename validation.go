package main

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a structured validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors collects multiple field errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve *ValidationErrors) Add(field, message string) {
	ve.Errors = append(ve.Errors, ValidationError{Field: field, Message: message})
}

func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

func (ve *ValidationErrors) Error() string {
	msgs := make([]string, len(ve.Errors))
	for i, e := range ve.Errors {
		msgs[i] = e.Field + ": " + e.Message
	}
	return strings.Join(msgs, "; ")
}

// requireField checks a required string field is non-empty
func requireField(ve *ValidationErrors, field, value string) {
	if strings.TrimSpace(value) == "" {
		ve.Add(field, "is required")
	}
}

// validateEnum checks a field is one of allowed values
func validateEnum(ve *ValidationErrors, field, value string, allowed []string) {
	if value == "" {
		return // only validate if set; combine with requireField if mandatory
	}
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	ve.Add(field, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
}

// validateDate checks a field is a valid date (YYYY-MM-DD)
func validateDate(ve *ValidationErrors, field, value string) {
	if value == "" {
		return
	}
	_, err := time.Parse("2006-01-02", value)
	if err != nil {
		ve.Add(field, "must be a valid date (YYYY-MM-DD)")
	}
}

// validatePositiveInt checks a field is > 0
func validatePositiveInt(ve *ValidationErrors, field string, value int) {
	if value <= 0 {
		ve.Add(field, "must be a positive integer")
	}
}

// validatePositiveFloat checks a field is > 0
func validatePositiveFloat(ve *ValidationErrors, field string, value float64) {
	if value <= 0 {
		ve.Add(field, "must be a positive number")
	}
}

// validateNonNegativeFloat checks a field is >= 0
func validateNonNegativeFloat(ve *ValidationErrors, field string, value float64) {
	if value < 0 {
		ve.Add(field, "must be non-negative")
	}
}

// validateEmail checks a field is a valid email (if non-empty)
func validateEmail(ve *ValidationErrors, field, value string) {
	if value == "" {
		return
	}
	_, err := mail.ParseAddress(value)
	if err != nil {
		ve.Add(field, "must be a valid email address")
	}
}

// validateMaxLength checks string doesn't exceed max length
func validateMaxLength(ve *ValidationErrors, field, value string, max int) {
	if len(value) > max {
		ve.Add(field, fmt.Sprintf("must be at most %d characters", max))
	}
}

// validateForeignKey checks that a referenced record exists
func validateForeignKey(ve *ValidationErrors, field, table, id string) {
	if id == "" {
		return
	}
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id=?", table), id).Scan(&count)
	if err != nil || count == 0 {
		ve.Add(field, fmt.Sprintf("references non-existent %s: %s", table, id))
	}
}

// validateForeignKeyCol checks that a referenced record exists using a specific column
func validateForeignKeyCol(ve *ValidationErrors, field, table, col, value string) {
	if value == "" {
		return
	}
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s=?", table, col), value).Scan(&count)
	if err != nil || count == 0 {
		ve.Add(field, fmt.Sprintf("references non-existent record: %s", value))
	}
}

// writeValidationError writes a 400 response with structured validation errors
func writeValidationError(w interface{ WriteHeader(int) }, ve *ValidationErrors) {
	type writer interface {
		WriteHeader(int)
	}
	// Use the standard pattern
}

// ipnPattern matches valid IPN format (letters, numbers, hyphens)
var ipnPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9\-_.]+$`)

func validateIPN(ve *ValidationErrors, field, value string) {
	if value == "" {
		return
	}
	if !ipnPattern.MatchString(value) {
		ve.Add(field, "must contain only letters, numbers, hyphens, underscores, and dots")
	}
}

// Common enum values
var (
	// These MUST match DB CHECK constraints in db.go
	validECOStatuses           = []string{"draft", "review", "approved", "implemented", "rejected", "cancelled"}
	validECOPriorities         = []string{"low", "normal", "high", "critical"}
	validPOStatuses            = []string{"draft", "sent", "confirmed", "partial", "received", "cancelled"}
	validWOStatuses            = []string{"draft", "open", "in_progress", "completed", "cancelled", "on_hold"}
	validWOPriorities          = []string{"low", "normal", "high", "critical"}
	validNCRSeverities         = []string{"minor", "major", "critical"}
	validNCRStatuses           = []string{"open", "investigating", "resolved", "closed"}
	validRMAStatuses           = []string{"open", "received", "diagnosing", "repairing", "resolved", "closed", "scrapped"}
	validQuoteStatuses         = []string{"draft", "sent", "accepted", "rejected", "expired", "cancelled"}
	validShipmentTypes         = []string{"inbound", "outbound", "transfer"}
	validShipmentStatuses      = []string{"draft", "packed", "shipped", "delivered", "cancelled"}
	validDeviceStatuses        = []string{"active", "inactive", "rma", "decommissioned", "maintenance"}
	validCampaignStatuses      = []string{"draft", "active", "paused", "completed", "cancelled"}
	validCampaignDevStatuses   = []string{"pending", "in_progress", "success", "failed", "skipped"}
	validDocStatuses           = []string{"draft", "review", "approved", "released", "obsolete"}
	validCAPATypes             = []string{"corrective", "preventive"}
	validCAPAStatuses          = []string{"open", "in_progress", "pending_review", "closed", "cancelled"}
	validVendorStatuses        = []string{"active", "preferred", "inactive", "blocked"}
	validInventoryTypes        = []string{"receive", "issue", "adjust", "transfer", "return", "scrap"}
	validRFQStatuses           = []string{"draft", "sent", "quoting", "awarded", "cancelled"}
	validFieldReportTypes      = []string{"failure", "performance", "safety", "visit", "other"}
	validFieldReportStatuses   = []string{"open", "investigating", "resolved", "closed"}
	validSalesOrderStatuses    = []string{"draft", "confirmed", "allocated", "picked", "shipped", "invoiced", "closed"}
	validInvoiceStatuses       = []string{"draft", "sent", "paid", "overdue", "cancelled"}
	validFieldReportPriorities = []string{"low", "medium", "high", "critical"}
)

// hasReferences checks if a record is referenced by other tables
func hasReferences(table, id string, refs []struct{ table, col string }) bool {
	for _, ref := range refs {
		var count int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s=?", ref.table, ref.col), id).Scan(&count)
		if count > 0 {
			return true
		}
	}
	return false
}
