package main

import (
	"fmt"
	"net/mail"
	"path/filepath"
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

// validateFloatRange checks a field is within a specified range
func validateFloatRange(ve *ValidationErrors, field string, value, min, max float64) {
	if value < min || value > max {
		ve.Add(field, fmt.Sprintf("must be between %.2f and %.2f", min, max))
	}
}

// validateIntRange checks a field is within a specified range
func validateIntRange(ve *ValidationErrors, field string, value, min, max int) {
	if value < min || value > max {
		ve.Add(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
}

// Maximum value constants to prevent overflow and ensure reasonable limits
const (
	MaxQuantity       = 1000000.0  // Maximum quantity for inventory/orders (1 million)
	MaxPrice          = 1000000.0  // Maximum unit price ($1M)
	MaxLeadTimeDays   = 730        // Maximum lead time (2 years)
	MaxPercentage     = 100.0      // Maximum percentage value
	MaxWorkOrderQty   = 100000     // Maximum work order quantity
	MaxStringLength   = 10000      // Maximum string field length
	MaxTextLength     = 100000     // Maximum text field length
)

// validateMaxQuantity checks quantity doesn't exceed reasonable maximum
func validateMaxQuantity(ve *ValidationErrors, field string, value float64) {
	if value > MaxQuantity {
		ve.Add(field, fmt.Sprintf("exceeds maximum allowed quantity of %.0f", MaxQuantity))
	}
}

// validateMaxPrice checks price doesn't exceed reasonable maximum
func validateMaxPrice(ve *ValidationErrors, field string, value float64) {
	if value > MaxPrice {
		ve.Add(field, fmt.Sprintf("exceeds maximum allowed price of %.2f", MaxPrice))
	}
}

// validatePercentage checks a value is a valid percentage (0-100)
func validatePercentage(ve *ValidationErrors, field string, value float64) {
	if value < 0 || value > 100 {
		ve.Add(field, "must be between 0 and 100")
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
	
	// Validate table name to prevent SQL injection
	validatedTable, err := ValidateAndSanitizeTable(table)
	if err != nil {
		ve.Add(field, "invalid table reference")
		return
	}
	
	var count int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id=?", validatedTable), id).Scan(&count)
	if err != nil || count == 0 {
		ve.Add(field, fmt.Sprintf("references non-existent %s: %s", validatedTable, id))
	}
}

// validateForeignKeyCol checks that a referenced record exists using a specific column
func validateForeignKeyCol(ve *ValidationErrors, field, table, col, value string) {
	if value == "" {
		return
	}
	
	// Validate table and column names to prevent SQL injection
	validatedTable, err := ValidateAndSanitizeTable(table)
	if err != nil {
		ve.Add(field, "invalid table reference")
		return
	}
	
	validatedCol, err := ValidateAndSanitizeColumn(col)
	if err != nil {
		ve.Add(field, "invalid column reference")
		return
	}
	
	var count int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s=?", validatedTable, validatedCol), value).Scan(&count)
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

// File upload validation constants
const (
	MaxFileSize         = 100 * 1024 * 1024 // 100MB
	MaxReasonableFile   = 10 * 1024 * 1024  // 10MB for normal uploads
	MinFileSize         = 1                 // 1 byte minimum
)

// Dangerous file extensions that should be blocked
var dangerousExtensions = []string{
	".exe", ".bat", ".cmd", ".com", ".scr", ".pif", ".app", ".dmg", ".pkg",
	".sh", ".bash", ".zsh", ".fish", ".csh", ".tcsh",
	".vbs", ".vbe", ".js", ".jse", ".ws", ".wsf", ".wsh",
	".msi", ".msp", ".jar", ".war", ".ear",
	".ps1", ".psm1", ".psd1", ".ps1xml", ".pssc", ".cdxml",
	".reg", ".dll", ".so", ".dylib",
	".apk", ".ipa", ".deb", ".rpm",
}

// Allowed safe file extensions (whitelist approach)
var allowedExtensions = []string{
	// Documents
	".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".csv",
	".odt", ".ods", ".odp", ".rtf",
	// Images
	".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp", ".ico",
	// Archives
	".zip", ".tar", ".gz", ".bz2", ".7z", ".rar",
	// Data
	".json", ".xml", ".yaml", ".yml", ".toml",
	// CAD/Engineering
	".dxf", ".dwg", ".step", ".stp", ".iges", ".igs", ".stl",
	// Other
	".log", ".md", ".markdown",
}

// validateFileUpload validates uploaded file size, type, and name
func validateFileUpload(ve *ValidationErrors, filename string, size int64, contentType string) {
	// Check file size
	if size == 0 {
		ve.Add("file", "cannot be empty (0 bytes)")
		return
	}
	
	if size < MinFileSize {
		ve.Add("file", "is too small")
		return
	}
	
	if size > MaxFileSize {
		ve.Add("file", fmt.Sprintf("exceeds maximum size of %d MB (got %d MB)", 
			MaxFileSize/(1024*1024), size/(1024*1024)))
		return
	}
	
	// Warn about large files
	if size > MaxReasonableFile {
		ve.Add("file", fmt.Sprintf("is very large (%d MB). Consider splitting or compressing.", 
			size/(1024*1024)))
	}
	
	// Validate filename
	validateFilename(ve, filename)
	
	// Validate extension
	validateFileExtension(ve, filename)
}

// validateFilename checks for path traversal and malicious characters
func validateFilename(ve *ValidationErrors, filename string) {
	if filename == "" {
		ve.Add("filename", "is required")
		return
	}
	
	// Check for path traversal attempts
	if strings.Contains(filename, "..") {
		ve.Add("filename", "contains invalid path traversal sequence (..)")
	}
	
	// Check for absolute paths
	if strings.HasPrefix(filename, "/") || strings.HasPrefix(filename, "\\") {
		ve.Add("filename", "cannot be an absolute path")
	}
	
	// Check for drive letters (Windows)
	if len(filename) >= 2 && filename[1] == ':' {
		ve.Add("filename", "cannot contain drive letters")
	}
	
	// Check for null bytes
	if strings.Contains(filename, "\x00") {
		ve.Add("filename", "contains null bytes")
	}
	
	// Check for control characters and dangerous chars
	dangerousChars := []string{"|", "&", ";", "$", "`", "<", ">", "(", ")", "{", "}", "[", "]", "!", "*", "?"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			ve.Add("filename", fmt.Sprintf("contains dangerous character: %s", char))
		}
	}
	
	// Check for CRLF injection
	if strings.ContainsAny(filename, "\r\n") {
		ve.Add("filename", "contains line breaks")
	}
}

// validateFileExtension checks if file extension is allowed
func validateFileExtension(ve *ValidationErrors, filename string) {
	// Get extension
	ext := strings.ToLower(filepath.Ext(filename))
	
	if ext == "" {
		ve.Add("filename", "must have a file extension")
		return
	}
	
	// Check against dangerous extensions first
	for _, dangerous := range dangerousExtensions {
		if ext == dangerous {
			ve.Add("filename", fmt.Sprintf("file type not allowed: %s", ext))
			return
		}
	}
	
	// Check against allowed extensions (whitelist)
	allowed := false
	for _, safe := range allowedExtensions {
		if ext == safe {
			allowed = true
			break
		}
	}
	
	if !allowed {
		ve.Add("filename", fmt.Sprintf("file type not in allowed list: %s (allowed: %s)", 
			ext, strings.Join(allowedExtensions[:10], ", ")+"..."))
	}
}

// sanitizeFilename removes dangerous characters and path components
func sanitizeFilename(filename string) string {
	// Remove path components
	filename = filepath.Base(filename)
	
	// Remove null bytes
	filename = strings.ReplaceAll(filename, "\x00", "")
	
	// Remove/replace dangerous characters
	replacements := map[string]string{
		"..":  "_",
		"/":   "_",
		"\\":  "_",
		"|":   "_",
		"&":   "_",
		";":   "_",
		"$":   "_",
		"`":   "_",
		"<":   "_",
		">":   "_",
		"(":   "",
		")":   "",
		"{":   "",
		"}":   "",
		"[":   "",
		"]":   "",
		"!":   "",
		"*":   "_",
		"?":   "_",
		"\r":  "",
		"\n":  "",
		"\t":  "_",
	}
	
	for old, new := range replacements {
		filename = strings.ReplaceAll(filename, old, new)
	}
	
	// Limit length
	if len(filename) > 255 {
		ext := filepath.Ext(filename)
		nameWithoutExt := filename[:len(filename)-len(ext)]
		if len(nameWithoutExt) > 200 {
			nameWithoutExt = nameWithoutExt[:200]
		}
		filename = nameWithoutExt + ext
	}
	
	return filename
}
