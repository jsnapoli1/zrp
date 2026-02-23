package main

import (
	"zrp/internal/auth"
)

// Variable aliases for backward compatibility.
var ValidTableNames = auth.ValidTableNames
var ValidColumnNames = auth.ValidColumnNames

// Wrapper functions delegating to internal/auth.
func ValidateTableName(table string) error {
	return auth.ValidateTableName(table)
}

func ValidateColumnName(column string) error {
	return auth.ValidateColumnName(column)
}

func SanitizeIdentifier(identifier string) (string, error) {
	return auth.SanitizeIdentifier(identifier)
}

func ValidateAndSanitizeTable(table string) (string, error) {
	return auth.ValidateAndSanitizeTable(table)
}

func ValidateAndSanitizeColumn(column string) (string, error) {
	return auth.ValidateAndSanitizeColumn(column)
}

func ValidatePasswordStrength(password string) error {
	return auth.ValidatePasswordStrength(password)
}

// Error variable aliases.
var ErrPasswordReused = auth.ErrPasswordReused
var ErrInvalidToken = auth.ErrInvalidToken

// Password history wrappers injecting global db.
func CheckPasswordHistory(userID int, newPassword string) error {
	return auth.CheckPasswordHistory(db, userID, newPassword)
}

func AddPasswordHistory(userID int, passwordHash string) error {
	return auth.AddPasswordHistory(db, userID, passwordHash)
}

func GeneratePasswordResetToken(username string) (string, error) {
	return auth.GeneratePasswordResetToken(db, username, generateToken)
}

func ValidatePasswordResetToken(token string) (bool, int) {
	return auth.ValidatePasswordResetToken(db, token)
}

func ResetPasswordWithToken(token, newPassword string) error {
	return auth.ResetPasswordWithToken(db, token, newPassword)
}

// Account lockout constants and wrappers.
const MaxFailedLoginAttempts = auth.MaxFailedLoginAttempts

var AccountLockoutDuration = auth.AccountLockoutDuration

func IncrementFailedLoginAttempts(username string) error {
	return auth.IncrementFailedLoginAttempts(db, username)
}

func ResetFailedLoginAttempts(username string) error {
	return auth.ResetFailedLoginAttempts(db, username)
}

func IsAccountLocked(username string) (bool, error) {
	return auth.IsAccountLocked(db, username)
}
