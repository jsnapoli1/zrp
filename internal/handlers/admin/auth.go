package admin

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"zrp/internal/auth"
	"zrp/internal/response"
	"zrp/internal/server"

	"golang.org/x/crypto/bcrypt"
)

// GetClientIP extracts the client IP from the request.
func GetClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

// HandleLogin authenticates a user and creates a session.
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Check rate limit (defense in depth - also enforced at middleware level)
	ip := GetClientIP(r)
	if !h.CheckLoginRateLimit(ip) {
		response.Err(w, "Too many login attempts. Try again in a minute.", 429)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, "Invalid request body", 400)
		return
	}

	// Check if account is locked
	locked, err := h.IsAccountLocked(req.Username)
	if err == nil && locked {
		response.Err(w, "Account temporarily locked due to too many failed login attempts. Try again later.", 403)
		return
	}

	var id int
	var passwordHash, displayName, role string
	var active int
	err = h.DB.QueryRow("SELECT id, password_hash, display_name, role, active FROM users WHERE username = ?", req.Username).
		Scan(&id, &passwordHash, &displayName, &role, &active)
	if err != nil {
		response.Err(w, "Invalid username or password", 401)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		// Increment failed login attempts
		h.IncrementFailedLoginAttempts(req.Username)
		response.Err(w, "Invalid username or password", 401)
		return
	}

	if active == 0 {
		response.Err(w, "Account deactivated", 403)
		return
	}

	// Reset failed login attempts on successful login
	h.ResetFailedLoginAttempts(req.Username)

	// Clean expired sessions
	h.DB.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")

	// Create session with retry
	var token string
	expires := time.Now().Add(24 * time.Hour)
	for i := 0; i < 3; i++ {
		token = h.GenerateToken()
		_, err = h.DB.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
			token, id, expires.Format("2006-01-02 15:04:05"))
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		response.Err(w, "Failed to create session", 500)
		return
	}

	// Update last_login
	h.DB.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?", id)

	http.SetCookie(w, &http.Cookie{
		Name:     "zrp_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})

	// Generate CSRF token for the user
	csrfToken, err := h.GenerateCSRFToken(id)
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

// HandleLogout logs out the user.
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("zrp_session")
	if err == nil {
		h.DB.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
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

// HandleMe returns the current user's info.
func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
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
	err = h.DB.QueryRow(`SELECT u.id, u.username, u.display_name, u.role, COALESCE(s.last_activity, s.created_at)
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
				h.DB.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(401)
				json.NewEncoder(w).Encode(map[string]string{"error": "Session expired due to inactivity", "code": "SESSION_TIMEOUT"})
				return
			}
		}
	}

	// Update last_activity on this request (use UTC for consistency with SQLite)
	h.DB.Exec("UPDATE sessions SET last_activity = ? WHERE token = ?",
		time.Now().UTC().Format("2006-01-02 15:04:05"), cookie.Value)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": UserResponse{ID: id, Username: username, DisplayName: displayName, Role: role},
	})
}

// HandleChangePassword changes the current user's password.
func (h *Handler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(server.CtxUserID).(int)
	if !ok || userID == 0 {
		response.Err(w, "Unauthorized", 401)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, "Invalid request body", 400)
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		response.Err(w, "Current and new password required", 400)
		return
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(req.NewPassword); err != nil {
		response.Err(w, err.Error(), 400)
		return
	}

	var currentHash string
	err := h.DB.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentHash)
	if err != nil {
		response.Err(w, "User not found", 404)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		response.Err(w, "Current password is incorrect", 401)
		return
	}

	// Check password history
	if err := auth.CheckPasswordHistory(h.DB, userID, req.NewPassword); err != nil {
		response.Err(w, err.Error(), 400)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response.Err(w, "Failed to hash password", 500)
		return
	}

	_, err = h.DB.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(newHash), userID)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Add old password to history
	auth.AddPasswordHistory(h.DB, userID, currentHash)

	response.JSON(w, map[string]string{"status": "password_changed"})
}

// GenerateCSRFToken creates a new CSRF token for the given user.
func (h *Handler) GenerateCSRFToken(userID int) (string, error) {
	// Clean up expired tokens first
	h.DB.Exec("DELETE FROM csrf_tokens WHERE expires_at < CURRENT_TIMESTAMP")

	// Clean up old tokens for this user (keep only the most recent 5)
	h.DB.Exec(`DELETE FROM csrf_tokens WHERE user_id = ? AND token NOT IN (
		SELECT token FROM csrf_tokens WHERE user_id = ? ORDER BY created_at DESC LIMIT 5
	)`, userID, userID)

	token := h.GenerateToken()
	expires := time.Now().Add(1 * time.Hour) // CSRF tokens expire in 1 hour

	_, err := h.DB.Exec("INSERT INTO csrf_tokens (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		return "", err
	}

	return token, nil
}

// HandleGetCSRFToken returns a new CSRF token for the authenticated user.
func (h *Handler) HandleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(server.CtxUserID).(int)
	if !ok || userID == 0 {
		response.Err(w, "Unauthorized", 401)
		return
	}

	token, err := h.GenerateCSRFToken(userID)
	if err != nil {
		response.Err(w, "Failed to generate CSRF token", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"csrf_token": token,
		"expires_in": "3600",
	})
}
