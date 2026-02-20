package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Rate limiter for login attempts: max 5 per minute per IP
var loginLimiter = struct {
	sync.Mutex
	attempts map[string][]time.Time
}{attempts: make(map[string][]time.Time)}

func checkLoginRateLimit(ip string) bool {
	loginLimiter.Lock()
	defer loginLimiter.Unlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)

	// Filter to recent attempts
	recent := loginLimiter.attempts[ip][:0]
	for _, t := range loginLimiter.attempts[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	loginLimiter.attempts[ip] = recent

	if len(recent) >= 5 {
		return false
	}

	loginLimiter.attempts[ip] = append(loginLimiter.attempts[ip], now)
	return true
}

func resetLoginRateLimit() {
	loginLimiter.Lock()
	loginLimiter.attempts = make(map[string][]time.Time)
	loginLimiter.Unlock()
}

func getClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	// Rate limiting is now handled by rateLimitMiddleware
	// Old checkLoginRateLimit removed to avoid double rate limiting

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Invalid request body", 400)
		return
	}

	// Check if account is locked
	locked, err := IsAccountLocked(req.Username)
	if err == nil && locked {
		jsonErr(w, "Account temporarily locked due to too many failed login attempts. Try again later.", 403)
		return
	}

	var id int
	var passwordHash, displayName, role string
	var active int
	err = db.QueryRow("SELECT id, password_hash, display_name, role, active FROM users WHERE username = ?", req.Username).
		Scan(&id, &passwordHash, &displayName, &role, &active)
	if err != nil {
		jsonErr(w, "Invalid username or password", 401)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		// Increment failed login attempts
		IncrementFailedLoginAttempts(req.Username)
		jsonErr(w, "Invalid username or password", 401)
		return
	}

	if active == 0 {
		jsonErr(w, "Account deactivated", 403)
		return
	}

	// Reset failed login attempts on successful login
	ResetFailedLoginAttempts(req.Username)

	// Clean expired sessions
	db.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")

	// Create session with retry
	var token string
	expires := time.Now().Add(24 * time.Hour)
	for i := 0; i < 3; i++ {
		token = generateToken()
		_, err = db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
			token, id, expires.Format("2006-01-02 15:04:05"))
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		jsonErr(w, "Failed to create session", 500)
		return
	}

	// Update last_login
	db.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?", id)

	http.SetCookie(w, &http.Cookie{
		Name:     "zrp_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Only transmit over HTTPS
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})

	// Generate CSRF token for the user
	csrfToken, err := generateCSRFToken(id)
	if err != nil {
		// Log error but don't fail login
		csrfToken = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":       UserResponse{ID: id, Username: req.Username, DisplayName: displayName, Role: role},
		"csrf_token": csrfToken,
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("zrp_session")
	if err == nil {
		db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "zrp_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("zrp_session")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized", "code": "UNAUTHORIZED"})
		return
	}

	var id int
	var username, displayName, role string
	var lastActivity string
	err = db.QueryRow(`SELECT u.id, u.username, u.display_name, u.role, COALESCE(s.last_activity, s.created_at)
		FROM sessions s JOIN users u ON s.user_id = u.id 
		WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP`, cookie.Value).
		Scan(&id, &username, &displayName, &role, &lastActivity)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized", "code": "UNAUTHORIZED"})
		return
	}

	// Check for inactivity timeout (30 minutes)
	if lastActivity != "" {
		// Try multiple time formats (SQLite can store in different formats)
		var lastActivityTime time.Time
		var parseErr error
		formats := []string{
			"2006-01-02 15:04:05",
			time.RFC3339,
			"2006-01-02T15:04:05Z",
		}
		for _, format := range formats {
			lastActivityTime, parseErr = time.Parse(format, lastActivity)
			if parseErr == nil {
				break
			}
		}
		
		if parseErr == nil {
			inactivityPeriod := time.Since(lastActivityTime)
			if inactivityPeriod > 30*time.Minute {
				// Session has been inactive too long - invalidate it
				db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(401)
				json.NewEncoder(w).Encode(map[string]string{"error": "Session expired due to inactivity", "code": "SESSION_TIMEOUT"})
				return
			}
		}
	}

	// Update last_activity on this request (use UTC for consistency with SQLite)
	db.Exec("UPDATE sessions SET last_activity = ? WHERE token = ?",
		time.Now().UTC().Format("2006-01-02 15:04:05"), cookie.Value)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": UserResponse{ID: id, Username: username, DisplayName: displayName, Role: role},
	})
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxUserID).(int)
	if !ok || userID == 0 {
		jsonErr(w, "Unauthorized", 401)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Invalid request body", 400)
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		jsonErr(w, "Current and new password required", 400)
		return
	}
	
	// Validate password strength
	if err := ValidatePasswordStrength(req.NewPassword); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	var currentHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentHash)
	if err != nil {
		jsonErr(w, "User not found", 404)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		jsonErr(w, "Current password is incorrect", 401)
		return
	}

	// Check password history
	if err := CheckPasswordHistory(userID, req.NewPassword); err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		jsonErr(w, "Failed to hash password", 500)
		return
	}

	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(newHash), userID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Add old password to history
	AddPasswordHistory(userID, currentHash)

	jsonResp(w, map[string]string{"status": "password_changed"})
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateCSRFToken creates a new CSRF token for the given user
func generateCSRFToken(userID int) (string, error) {
	// Clean up expired tokens first
	db.Exec("DELETE FROM csrf_tokens WHERE expires_at < CURRENT_TIMESTAMP")
	
	// Clean up old tokens for this user (keep only the most recent 5)
	db.Exec(`DELETE FROM csrf_tokens WHERE user_id = ? AND token NOT IN (
		SELECT token FROM csrf_tokens WHERE user_id = ? ORDER BY created_at DESC LIMIT 5
	)`, userID, userID)

	token := generateToken()
	expires := time.Now().Add(1 * time.Hour) // CSRF tokens expire in 1 hour
	
	_, err := db.Exec("INSERT INTO csrf_tokens (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		return "", err
	}
	
	return token, nil
}

// handleGetCSRFToken returns a new CSRF token for the authenticated user
func handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxUserID).(int)
	if !ok || userID == 0 {
		jsonErr(w, "Unauthorized", 401)
		return
	}

	token, err := generateCSRFToken(userID)
	if err != nil {
		jsonErr(w, "Failed to generate CSRF token", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"csrf_token": token,
		"expires_in": "3600", // 1 hour in seconds
	})
}
