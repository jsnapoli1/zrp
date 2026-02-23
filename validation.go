package main

import (
	"fmt"

	"zrp/internal/validation"
)

// Type aliases for backward compatibility.
type ValidationError = validation.ValidationError
type ValidationErrors = validation.ValidationErrors

// Wrapper functions delegating to internal/validation.
func requireField(ve *ValidationErrors, field, value string)                   { validation.RequireField(ve, field, value) }
func validateEnum(ve *ValidationErrors, field, value string, allowed []string) { validation.ValidateEnum(ve, field, value, allowed) }
func validateDate(ve *ValidationErrors, field, value string)                   { validation.ValidateDate(ve, field, value) }
func validatePositiveInt(ve *ValidationErrors, field string, value int)         { validation.ValidatePositiveInt(ve, field, value) }
func validatePositiveFloat(ve *ValidationErrors, field string, value float64)   { validation.ValidatePositiveFloat(ve, field, value) }
func validateNonNegativeFloat(ve *ValidationErrors, field string, value float64) { validation.ValidateNonNegativeFloat(ve, field, value) }
func validateFloatRange(ve *ValidationErrors, field string, value, min, max float64) { validation.ValidateFloatRange(ve, field, value, min, max) }
func validateIntRange(ve *ValidationErrors, field string, value, min, max int) { validation.ValidateIntRange(ve, field, value, min, max) }
func validateMaxQuantity(ve *ValidationErrors, field string, value float64)    { validation.ValidateMaxQuantity(ve, field, value) }
func validateMaxPrice(ve *ValidationErrors, field string, value float64)       { validation.ValidateMaxPrice(ve, field, value) }
func validatePercentage(ve *ValidationErrors, field string, value float64)     { validation.ValidatePercentage(ve, field, value) }
func validateEmail(ve *ValidationErrors, field, value string)                  { validation.ValidateEmail(ve, field, value) }
func validateMaxLength(ve *ValidationErrors, field, value string, max int)     { validation.ValidateMaxLength(ve, field, value, max) }
func validateIPN(ve *ValidationErrors, field, value string)                    { validation.ValidateIPN(ve, field, value) }

// validateForeignKey delegates to validation package, injecting the global db.
func validateForeignKey(ve *ValidationErrors, field, table, id string) {
	validation.ValidateForeignKey(ve, db, field, table, id, ValidateAndSanitizeTable)
}

// validateForeignKeyCol delegates to validation package, injecting the global db.
func validateForeignKeyCol(ve *ValidationErrors, field, table, col, value string) {
	validation.ValidateForeignKeyCol(ve, db, field, table, col, value, ValidateAndSanitizeTable, ValidateAndSanitizeColumn)
}

func validateFileUpload(ve *ValidationErrors, filename string, size int64, contentType string) { validation.ValidateFileUpload(ve, filename, size, contentType) }
func validateFilename(ve *ValidationErrors, filename string)                                    { validation.ValidateFilename(ve, filename) }
func validateFileExtension(ve *ValidationErrors, filename string)                               { validation.ValidateFileExtension(ve, filename) }
func sanitizeFilename(filename string) string                                                   { return validation.SanitizeFilename(filename) }

// writeValidationError writes a 400 response with structured validation errors.
func writeValidationError(w interface{ WriteHeader(int) }, ve *ValidationErrors) {
	// Stub - kept for backward compatibility
}

// Constant aliases.
const (
	MaxQuantity       = validation.MaxQuantity
	MaxPrice          = validation.MaxPrice
	MaxLeadTimeDays   = validation.MaxLeadTimeDays
	MaxPercentage     = validation.MaxPercentage
	MaxWorkOrderQty   = validation.MaxWorkOrderQty
	MaxStringLength   = validation.MaxStringLength
	MaxTextLength     = validation.MaxTextLength
	MaxFileSize       = validation.MaxFileSize
	MaxReasonableFile = validation.MaxReasonableFile
	MinFileSize       = validation.MinFileSize
)

// ipnPattern alias.
var ipnPattern = validation.IPNPattern

// Enum aliases.
var (
	validECOStatuses           = validation.ValidECOStatuses
	validECOPriorities         = validation.ValidECOPriorities
	validPOStatuses            = validation.ValidPOStatuses
	validWOStatuses            = validation.ValidWOStatuses
	validWOPriorities          = validation.ValidWOPriorities
	validNCRSeverities         = validation.ValidNCRSeverities
	validNCRStatuses           = validation.ValidNCRStatuses
	validRMAStatuses           = validation.ValidRMAStatuses
	validQuoteStatuses         = validation.ValidQuoteStatuses
	validShipmentTypes         = validation.ValidShipmentTypes
	validShipmentStatuses      = validation.ValidShipmentStatuses
	validDeviceStatuses        = validation.ValidDeviceStatuses
	validCampaignStatuses      = validation.ValidCampaignStatuses
	validCampaignDevStatuses   = validation.ValidCampaignDevStatuses
	validDocStatuses           = validation.ValidDocStatuses
	validCAPATypes             = validation.ValidCAPATypes
	validCAPAStatuses          = validation.ValidCAPAStatuses
	validVendorStatuses        = validation.ValidVendorStatuses
	validInventoryTypes        = validation.ValidInventoryTypes
	validRFQStatuses           = validation.ValidRFQStatuses
	validFieldReportTypes      = validation.ValidFieldReportTypes
	validFieldReportStatuses   = validation.ValidFieldReportStatuses
	validSalesOrderStatuses    = validation.ValidSalesOrderStatuses
	validInvoiceStatuses       = validation.ValidInvoiceStatuses
	validFieldReportPriorities = validation.ValidFieldReportPriorities
)

// Dangerous/allowed extension aliases.
var dangerousExtensions = validation.DangerousExtensions
var allowedExtensions = validation.AllowedExtensions

// hasReferences delegates to validation package, injecting the global db.
func hasReferences(table, id string, refs []struct{ table, col string }) bool {
	// Convert to exported struct type
	converted := make([]struct{ Table, Col string }, len(refs))
	for i, ref := range refs {
		converted[i] = struct{ Table, Col string }{Table: ref.table, Col: ref.col}
	}
	return validation.HasReferences(db, table, id, converted)
}

// Ensure fmt is used (referenced in error messages above).
var _ = fmt.Sprintf
