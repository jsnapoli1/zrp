package auth

import (
	"errors"
	"regexp"
)

// ValidTableNames is a whitelist of allowed table names.
var ValidTableNames = map[string]bool{
	"parts":                  true,
	"ecos":                   true,
	"users":                  true,
	"sessions":               true,
	"api_keys":               true,
	"categories":             true,
	"custom_columns":         true,
	"work_orders":            true,
	"purchase_orders":        true,
	"po_lines":               true,
	"receiving":              true,
	"inventory":              true,
	"inventory_transactions": true,
	"ncrs":                   true,
	"capas":                  true,
	"rmas":                   true,
	"devices":                true,
	"campaigns":              true,
	"campaign_devices":       true,
	"shipments":              true,
	"quotes":                 true,
	"docs":                   true,
	"doc_versions":           true,
	"vendors":                true,
	"prices":                 true,
	"email_config":           true,
	"email_log":              true,
	"notifications":          true,
	"notification_prefs":     true,
	"attachments":            true,
	"undo_log":               true,
	"part_changes":           true,
	"changes":                true,
	"permissions":            true,
	"role_permissions":       true,
	"rfqs":                   true,
	"rfq_quotes":             true,
	"product_pricing":        true,
	"cost_analysis":          true,
	"sales_orders":           true,
	"invoices":               true,
	"field_reports":          true,
	"saved_searches":         true,
	"search_history":         true,
	"password_history":       true,
	"password_reset_tokens":  true,
}

// ValidColumnNames is a whitelist of commonly used column names.
var ValidColumnNames = map[string]bool{
	"id":                      true,
	"ipn":                     true,
	"mpn":                     true,
	"description":             true,
	"category":                true,
	"status":                  true,
	"created_at":              true,
	"updated_at":              true,
	"created_by":              true,
	"updated_by":              true,
	"name":                    true,
	"email":                   true,
	"username":                true,
	"role":                    true,
	"active":                  true,
	"title":                   true,
	"content":                 true,
	"quantity":                true,
	"price":                   true,
	"vendor_id":               true,
	"part_id":                 true,
	"eco_id":                  true,
	"user_id":                 true,
	"device_id":               true,
	"campaign_id":             true,
	"shipment_id":             true,
	"quote_id":                true,
	"doc_id":                  true,
	"po_id":                   true,
	"wo_id":                   true,
	"ncr_id":                  true,
	"capa_id":                 true,
	"rma_id":                  true,
	"invoice_id":              true,
	"sales_order_id":          true,
	"field_report_id":         true,
	"revision":                true,
	"approved":                true,
	"approved_by":             true,
	"approved_at":             true,
	"failed_login_attempts":   true,
	"locked_until":            true,
}

// ValidateTableName checks if a table name is in the whitelist.
func ValidateTableName(table string) error {
	if !ValidTableNames[table] {
		return errors.New("invalid table name")
	}
	return nil
}

// ValidateColumnName checks if a column name is in the whitelist.
func ValidateColumnName(column string) error {
	if !ValidColumnNames[column] {
		return errors.New("invalid column name")
	}
	return nil
}

// SanitizeIdentifier ensures an identifier contains only safe characters.
func SanitizeIdentifier(identifier string) (string, error) {
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validPattern.MatchString(identifier) {
		return "", errors.New("invalid identifier format")
	}
	return identifier, nil
}

// ValidateAndSanitizeTable validates and sanitizes a table name.
func ValidateAndSanitizeTable(table string) (string, error) {
	sanitized, err := SanitizeIdentifier(table)
	if err != nil {
		return "", err
	}
	if err := ValidateTableName(sanitized); err != nil {
		return "", err
	}
	return sanitized, nil
}

// ValidateAndSanitizeColumn validates and sanitizes a column name.
func ValidateAndSanitizeColumn(column string) (string, error) {
	sanitized, err := SanitizeIdentifier(column)
	if err != nil {
		return "", err
	}
	if err := ValidateColumnName(sanitized); err != nil {
		return "", err
	}
	return sanitized, nil
}

// ValidatePasswordStrength checks password complexity.
func ValidatePasswordStrength(password string) error {
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}

	var (
		hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString
		hasLower   = regexp.MustCompile(`[a-z]`).MatchString
		hasNumber  = regexp.MustCompile(`[0-9]`).MatchString
		hasSpecial = regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>_\-+=]`).MatchString
	)

	checks := 0
	if hasUpper(password) {
		checks++
	}
	if hasLower(password) {
		checks++
	}
	if hasNumber(password) {
		checks++
	}
	if hasSpecial(password) {
		checks++
	}

	if checks < 3 {
		return errors.New("password must contain at least 3 of: uppercase, lowercase, numbers, special characters")
	}

	return nil
}
