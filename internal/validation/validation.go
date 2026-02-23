package validation

import (
	"database/sql"
	"fmt"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a structured validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors collects multiple field errors.
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

// RequireField checks a required string field is non-empty.
func RequireField(ve *ValidationErrors, field, value string) {
	if strings.TrimSpace(value) == "" {
		ve.Add(field, "is required")
	}
}

// ValidateEnum checks a field is one of allowed values.
func ValidateEnum(ve *ValidationErrors, field, value string, allowed []string) {
	if value == "" {
		return
	}
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	ve.Add(field, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
}

// ValidateDate checks a field is a valid date (YYYY-MM-DD).
func ValidateDate(ve *ValidationErrors, field, value string) {
	if value == "" {
		return
	}
	_, err := time.Parse("2006-01-02", value)
	if err != nil {
		ve.Add(field, "must be a valid date (YYYY-MM-DD)")
	}
}

// ValidatePositiveInt checks a field is > 0.
func ValidatePositiveInt(ve *ValidationErrors, field string, value int) {
	if value <= 0 {
		ve.Add(field, "must be a positive integer")
	}
}

// ValidatePositiveFloat checks a field is > 0.
func ValidatePositiveFloat(ve *ValidationErrors, field string, value float64) {
	if value <= 0 {
		ve.Add(field, "must be a positive number")
	}
}

// ValidateNonNegativeFloat checks a field is >= 0.
func ValidateNonNegativeFloat(ve *ValidationErrors, field string, value float64) {
	if value < 0 {
		ve.Add(field, "must be non-negative")
	}
}

// ValidateFloatRange checks a field is within a specified range.
func ValidateFloatRange(ve *ValidationErrors, field string, value, min, max float64) {
	if value < min || value > max {
		ve.Add(field, fmt.Sprintf("must be between %.2f and %.2f", min, max))
	}
}

// ValidateIntRange checks a field is within a specified range.
func ValidateIntRange(ve *ValidationErrors, field string, value, min, max int) {
	if value < min || value > max {
		ve.Add(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
}

// Maximum value constants to prevent overflow and ensure reasonable limits.
const (
	MaxQuantity     = 1000000.0
	MaxPrice        = 1000000.0
	MaxLeadTimeDays = 730
	MaxPercentage   = 100.0
	MaxWorkOrderQty = 100000
	MaxStringLength = 10000
	MaxTextLength   = 100000
)

// ValidateMaxQuantity checks quantity doesn't exceed reasonable maximum.
func ValidateMaxQuantity(ve *ValidationErrors, field string, value float64) {
	if value > MaxQuantity {
		ve.Add(field, fmt.Sprintf("exceeds maximum allowed quantity of %.0f", MaxQuantity))
	}
}

// ValidateMaxPrice checks price doesn't exceed reasonable maximum.
func ValidateMaxPrice(ve *ValidationErrors, field string, value float64) {
	if value > MaxPrice {
		ve.Add(field, fmt.Sprintf("exceeds maximum allowed price of %.2f", MaxPrice))
	}
}

// ValidatePercentage checks a value is a valid percentage (0-100).
func ValidatePercentage(ve *ValidationErrors, field string, value float64) {
	if value < 0 || value > 100 {
		ve.Add(field, "must be between 0 and 100")
	}
}

// ValidateEmail checks a field is a valid email (if non-empty).
func ValidateEmail(ve *ValidationErrors, field, value string) {
	if value == "" {
		return
	}
	_, err := mail.ParseAddress(value)
	if err != nil {
		ve.Add(field, "must be a valid email address")
	}
}

// ValidateMaxLength checks string doesn't exceed max length.
func ValidateMaxLength(ve *ValidationErrors, field, value string, max int) {
	if len(value) > max {
		ve.Add(field, fmt.Sprintf("must be at most %d characters", max))
	}
}

// ValidateForeignKey checks that a referenced record exists.
// Requires a *sql.DB and uses ValidateAndSanitizeTable from the auth/security package.
func ValidateForeignKey(ve *ValidationErrors, db *sql.DB, field, table, id string, tableValidator func(string) (string, error)) {
	if id == "" {
		return
	}

	validatedTable, err := tableValidator(table)
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

// ValidateForeignKeyCol checks that a referenced record exists using a specific column.
func ValidateForeignKeyCol(ve *ValidationErrors, db *sql.DB, field, table, col, value string, tableValidator func(string) (string, error), colValidator func(string) (string, error)) {
	if value == "" {
		return
	}

	validatedTable, err := tableValidator(table)
	if err != nil {
		ve.Add(field, "invalid table reference")
		return
	}

	validatedCol, err := colValidator(col)
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

// HasReferences checks if a record is referenced by other tables.
func HasReferences(db *sql.DB, table, id string, refs []struct{ Table, Col string }) bool {
	for _, ref := range refs {
		var count int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s=?", ref.Table, ref.Col), id).Scan(&count)
		if count > 0 {
			return true
		}
	}
	return false
}

// IPNPattern matches valid IPN format (letters, numbers, hyphens).
var IPNPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9\-_.]+$`)

// ValidateIPN validates an IPN field.
func ValidateIPN(ve *ValidationErrors, field, value string) {
	if value == "" {
		return
	}
	if !IPNPattern.MatchString(value) {
		ve.Add(field, "must contain only letters, numbers, hyphens, underscores, and dots")
	}
}

// File upload validation constants.
const (
	MaxFileSize       = 100 * 1024 * 1024
	MaxReasonableFile = 10 * 1024 * 1024
	MinFileSize       = 1
)

// DangerousExtensions is the list of blocked file extensions.
var DangerousExtensions = []string{
	".exe", ".bat", ".cmd", ".com", ".scr", ".pif", ".app", ".dmg", ".pkg",
	".sh", ".bash", ".zsh", ".fish", ".csh", ".tcsh",
	".vbs", ".vbe", ".js", ".jse", ".ws", ".wsf", ".wsh",
	".msi", ".msp", ".jar", ".war", ".ear",
	".ps1", ".psm1", ".psd1", ".ps1xml", ".pssc", ".cdxml",
	".reg", ".dll", ".so", ".dylib",
	".apk", ".ipa", ".deb", ".rpm",
}

// AllowedExtensions is the whitelist of safe file extensions.
var AllowedExtensions = []string{
	".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".csv",
	".odt", ".ods", ".odp", ".rtf",
	".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp", ".ico",
	".zip", ".tar", ".gz", ".bz2", ".7z", ".rar",
	".json", ".xml", ".yaml", ".yml", ".toml",
	".dxf", ".dwg", ".step", ".stp", ".iges", ".igs", ".stl",
	".log", ".md", ".markdown",
}

// ValidateFileUpload validates uploaded file size, type, and name.
func ValidateFileUpload(ve *ValidationErrors, filename string, size int64, contentType string) {
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

	if size > MaxReasonableFile {
		ve.Add("file", fmt.Sprintf("is very large (%d MB). Consider splitting or compressing.",
			size/(1024*1024)))
	}

	ValidateFilename(ve, filename)
	ValidateFileExtension(ve, filename)
}

// ValidateFilename checks for path traversal and malicious characters.
func ValidateFilename(ve *ValidationErrors, filename string) {
	if filename == "" {
		ve.Add("filename", "is required")
		return
	}

	if strings.Contains(filename, "..") {
		ve.Add("filename", "contains invalid path traversal sequence (..)")
	}
	if strings.HasPrefix(filename, "/") || strings.HasPrefix(filename, "\\") {
		ve.Add("filename", "cannot be an absolute path")
	}
	if len(filename) >= 2 && filename[1] == ':' {
		ve.Add("filename", "cannot contain drive letters")
	}
	if strings.Contains(filename, "\x00") {
		ve.Add("filename", "contains null bytes")
	}

	dangerousChars := []string{"|", "&", ";", "$", "`", "<", ">", "(", ")", "{", "}", "[", "]", "!", "*", "?"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			ve.Add("filename", fmt.Sprintf("contains dangerous character: %s", char))
		}
	}

	if strings.ContainsAny(filename, "\r\n") {
		ve.Add("filename", "contains line breaks")
	}
}

// ValidateFileExtension checks if file extension is allowed.
func ValidateFileExtension(ve *ValidationErrors, filename string) {
	ext := strings.ToLower(filepath.Ext(filename))

	if ext == "" {
		ve.Add("filename", "must have a file extension")
		return
	}

	for _, dangerous := range DangerousExtensions {
		if ext == dangerous {
			ve.Add("filename", fmt.Sprintf("file type not allowed: %s", ext))
			return
		}
	}

	allowed := false
	for _, safe := range AllowedExtensions {
		if ext == safe {
			allowed = true
			break
		}
	}

	if !allowed {
		ve.Add("filename", fmt.Sprintf("file type not in allowed list: %s (allowed: %s)",
			ext, strings.Join(AllowedExtensions[:10], ", ")+"..."))
	}
}

// SanitizeFilename removes dangerous characters and path components.
func SanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, "\x00", "")

	replacements := map[string]string{
		"..": "_", "/": "_", "\\": "_", "|": "_", "&": "_", ";": "_",
		"$": "_", "`": "_", "<": "_", ">": "_", "(": "", ")": "",
		"{": "", "}": "", "[": "", "]": "", "!": "", "*": "_", "?": "_",
		"\r": "", "\n": "", "\t": "_",
	}
	for old, new := range replacements {
		filename = strings.ReplaceAll(filename, old, new)
	}

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
