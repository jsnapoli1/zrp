package auth

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordReused = errors.New("password was recently used, please choose a different password")
	ErrInvalidToken   = errors.New("invalid or expired token")
)

// CheckPasswordHistory verifies a password hasn't been used recently.
func CheckPasswordHistory(db *sql.DB, userID int, newPassword string) error {
	rows, err := db.Query(`
		SELECT password_hash FROM password_history
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 5`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var oldHash string
		if err := rows.Scan(&oldHash); err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(oldHash), []byte(newPassword)) == nil {
			return ErrPasswordReused
		}
	}
	return nil
}

// AddPasswordHistory adds a password to the user's history.
func AddPasswordHistory(db *sql.DB, userID int, passwordHash string) error {
	_, err := db.Exec(
		"INSERT INTO password_history (user_id, password_hash, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)",
		userID, passwordHash)
	return err
}

// GeneratePasswordResetToken creates a reset token valid for 1 hour.
// tokenGenerator should be a function that generates a random token string.
func GeneratePasswordResetToken(db *sql.DB, username string, tokenGenerator func() string) (string, error) {
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err != nil {
		return "", err
	}

	token := tokenGenerator()
	expiresAt := time.Now().Add(1 * time.Hour)

	_, err = db.Exec(
		"INSERT INTO password_reset_tokens (token, user_id, created_at, expires_at, used) VALUES (?, ?, CURRENT_TIMESTAMP, ?, 0)",
		token, userID, expiresAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidatePasswordResetToken checks if a token is valid and not expired.
func ValidatePasswordResetToken(db *sql.DB, token string) (bool, int) {
	var userID int
	var expiresAt string

	err := db.QueryRow(
		"SELECT user_id, expires_at FROM password_reset_tokens WHERE token = ? AND used = 0",
		token).Scan(&userID, &expiresAt)
	if err != nil {
		return false, 0
	}

	expires, err := time.Parse("2006-01-02 15:04:05", expiresAt)
	if err != nil {
		return false, 0
	}

	if time.Now().After(expires) {
		return false, 0
	}

	return true, userID
}

// ResetPasswordWithToken resets a password using a valid reset token.
func ResetPasswordWithToken(db *sql.DB, token, newPassword string) error {
	valid, userID := ValidatePasswordResetToken(db, token)
	if !valid {
		return ErrInvalidToken
	}

	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	if err := CheckPasswordHistory(db, userID, newPassword); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(hash), userID)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE password_reset_tokens SET used = 1 WHERE token = ?", token)
	if err != nil {
		return err
	}

	AddPasswordHistory(db, userID, string(hash))

	return nil
}
