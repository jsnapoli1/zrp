package auth

import (
	"database/sql"
	"time"
)

const (
	MaxFailedLoginAttempts = 10
	AccountLockoutDuration = 15 * time.Minute
)

// IncrementFailedLoginAttempts increments the failed login counter.
func IncrementFailedLoginAttempts(db *sql.DB, username string) error {
	_, err := db.Exec(`
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE
		        WHEN failed_login_attempts + 1 >= ? THEN datetime('now', '+15 minutes')
		        ELSE locked_until
		    END
		WHERE username = ?`, MaxFailedLoginAttempts, username)
	return err
}

// ResetFailedLoginAttempts resets the failed login counter after successful login.
func ResetFailedLoginAttempts(db *sql.DB, username string) error {
	_, err := db.Exec(`
		UPDATE users
		SET failed_login_attempts = 0, locked_until = NULL
		WHERE username = ?`, username)
	return err
}

// IsAccountLocked checks if an account is currently locked.
func IsAccountLocked(db *sql.DB, username string) (bool, error) {
	var lockedUntil *string
	err := db.QueryRow("SELECT locked_until FROM users WHERE username = ?", username).Scan(&lockedUntil)
	if err != nil {
		return false, err
	}

	if lockedUntil == nil {
		return false, nil
	}

	var lockTime time.Time
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
	}

	var parseErr error
	for _, format := range formats {
		lockTime, parseErr = time.Parse(format, *lockedUntil)
		if parseErr == nil {
			break
		}
	}

	if parseErr != nil {
		return false, nil
	}

	if time.Now().Before(lockTime) {
		return true, nil
	}

	ResetFailedLoginAttempts(db, username)
	return false, nil
}
