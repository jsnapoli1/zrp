package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"zrp/internal/audit"
	"zrp/internal/auth"
	"zrp/internal/response"
	"zrp/internal/server"
	"zrp/internal/validation"
	"zrp/internal/websocket"

	"golang.org/x/crypto/bcrypt"
)

// Handler holds dependencies for admin handlers.
type Handler struct {
	DB       *sql.DB
	Hub      *websocket.Hub
	Profiler QueryProfiler

	// Backup function fields (root-package functions that can't be imported).
	PerformBackup  func() error
	ListBackups    func() ([]BackupInfo, error)
	CleanOldBackups func()
	DBFilePath     func() string
	SetDBFilePath  func(string)
	InitDB         func(string) error

	// Email function fields.
	SendEmail            func(to, subject, body string) error
	SendEmailWithEvent   func(to, subject, body, eventType string) error
	GetEmailConfig       func() (*EmailConfig, error)
	EmailConfigEnabled   func() bool
	IsValidEmail         func(email string) bool
	SendEventEmail       func(to, subject, body, eventType, username string) error
	IsUserSubscribed     func(username, eventType string) bool

	// Auth function fields.
	CheckLoginRateLimit          func(ip string) bool
	IsAccountLocked              func(username string) (bool, error)
	IncrementFailedLoginAttempts func(username string) error
	ResetFailedLoginAttempts     func(username string) error
	GenerateToken                func() string

	// Permission function fields.
	SetRolePermissions func(dbConn *sql.DB, role string, perms []auth.PermissionEntry) error
	GetRolePermissions func(role string) []auth.PermissionEntry
}

// QueryProfiler interface for the profiler dependency.
type QueryProfiler interface {
	GetStatsMap() map[string]interface{}
	GetSlowQueriesAny() interface{}
	GetAllQueriesAny() interface{}
	Reset()
	SlowThresholdString() string
}

// UserFull represents a full user record.
type UserFull struct {
	ID          int     `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	Active      int     `json:"active"`
	CreatedAt   string  `json:"created_at"`
	LastLogin   *string `json:"last_login"`
}

// CreateUserRequest represents a user creation request.
type CreateUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

// UpdateUserRequest represents a user update request.
type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Active      *int   `json:"active"`
}

// ResetPasswordRequest represents a password reset request.
type ResetPasswordRequest struct {
	Password string `json:"password"`
}

// APIKey represents an API key record.
type APIKey struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	KeyPrefix string  `json:"key_prefix"`
	CreatedBy string  `json:"created_by"`
	CreatedAt string  `json:"created_at"`
	LastUsed  *string `json:"last_used"`
	ExpiresAt *string `json:"expires_at"`
	Enabled   int     `json:"enabled"`
}

// CreateAPIKeyRequest represents an API key creation request.
type CreateAPIKeyRequest struct {
	Name      string  `json:"name"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// GeneralSettings represents the general settings.
type GeneralSettings struct {
	AppName        string `json:"app_name"`
	CompanyName    string `json:"company_name"`
	CompanyAddress string `json:"company_address"`
	Currency       string `json:"currency"`
	DateFormat     string `json:"date_format"`
}

var generalSettingsKeys = []string{
	"app_name", "company_name", "company_address", "currency", "date_format",
}

var generalSettingsDefaults = map[string]string{
	"app_name":        "ZRP",
	"company_name":    "",
	"company_address": "",
	"currency":        "USD",
	"date_format":     "YYYY-MM-DD",
}

// DashboardWidget represents a dashboard widget.
type DashboardWidget struct {
	ID         int    `json:"id"`
	UserID     int    `json:"user_id"`
	WidgetType string `json:"widget_type"`
	Position   int    `json:"position"`
	Enabled    int    `json:"enabled"`
}

// BackupInfo represents a backup file entry.
type BackupInfo struct {
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

// EmailConfig represents the email configuration.
type EmailConfig struct {
	ID           int    `json:"id"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	FromAddress  string `json:"from_address"`
	FromName     string `json:"from_name"`
	Enabled      int    `json:"enabled"`
}

// EmailLogEntry represents an email log record.
type EmailLogEntry struct {
	ID        int    `json:"id"`
	To        string `json:"to_address"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	EventType string `json:"event_type"`
	Status    string `json:"status"`
	Error     string `json:"error"`
	SentAt    string `json:"sent_at"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserResponse represents a user response (for auth endpoints).
type UserResponse struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// CtxUserID and CtxRole are re-exported context keys.
var (
	CtxUserID = server.CtxUserID
	CtxRole   = server.CtxRole
)

// EmailEventTypes lists all supported email event types.
var EmailEventTypes = []string{
	"eco_approved",
	"eco_implemented",
	"low_stock",
	"overdue_work_order",
	"po_received",
	"ncr_created",
}

// --- User handlers ---

func (h *Handler) GetCurrentUser(r *http.Request) *UserFull {
	cookie, err := r.Cookie("zrp_session")
	if err != nil {
		return nil
	}
	var u UserFull
	var lastLogin *string
	err = h.DB.QueryRow(`SELECT u.id, u.username, u.display_name, u.role, u.active, u.created_at, u.last_login
		FROM sessions s JOIN users u ON s.user_id = u.id
		WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP`, cookie.Value).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Active, &u.CreatedAt, &lastLogin)
	if err != nil {
		return nil
	}
	u.LastLogin = lastLogin
	return &u
}

func (h *Handler) RequireAdmin(w http.ResponseWriter, r *http.Request) *UserFull {
	u := h.GetCurrentUser(r)
	if u == nil {
		response.Err(w, "Unauthorized", 401)
		return nil
	}
	if u.Role != "admin" {
		response.Err(w, "Admin access required", 403)
		return nil
	}
	return u
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if h.RequireAdmin(w, r) == nil {
		return
	}
	rows, err := h.DB.Query(`SELECT id, username, display_name, role, active, created_at, last_login FROM users ORDER BY id`)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var users []UserFull
	for rows.Next() {
		var u UserFull
		var lastLogin *string
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Active, &u.CreatedAt, &lastLogin); err != nil {
			continue
		}
		u.LastLogin = lastLogin
		users = append(users, u)
	}
	if users == nil {
		users = []UserFull{}
	}
	response.JSON(w, users)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if h.RequireAdmin(w, r) == nil {
		return
	}
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, "Invalid request body", 400)
		return
	}
	if req.Username == "" || req.Password == "" {
		response.Err(w, "Username and password required", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.ValidateMaxLength(ve, "username", req.Username, 100)
	validation.ValidateMaxLength(ve, "display_name", req.DisplayName, 255)
	validation.ValidateMaxLength(ve, "email", req.Email, 255)
	validation.ValidateEmail(ve, "email", req.Email)
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		response.Err(w, err.Error(), 400)
		return
	}

	validRoles := map[string]bool{"admin": true, "user": true, "readonly": true}
	if !validRoles[req.Role] {
		req.Role = "user"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Err(w, "Failed to hash password", 500)
		return
	}
	result, err := h.DB.Exec(`INSERT INTO users (username, password_hash, display_name, email, role, active) VALUES (?, ?, ?, ?, ?, 1)`,
		req.Username, string(hash), req.DisplayName, req.Email, req.Role)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			response.Err(w, "Username already exists", 409)
			return
		}
		response.Err(w, err.Error(), 500)
		return
	}
	id, _ := result.LastInsertId()

	auth.AddPasswordHistory(h.DB, int(id), string(hash))

	w.WriteHeader(201)
	response.JSON(w, map[string]interface{}{"id": id, "username": req.Username, "display_name": req.DisplayName, "role": req.Role})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request, idStr string) {
	admin := h.RequireAdmin(w, r)
	if admin == nil {
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Err(w, "Invalid user ID", 400)
		return
	}
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, "Invalid request body", 400)
		return
	}
	if req.Active != nil && *req.Active == 0 && id == admin.ID {
		response.Err(w, "Cannot deactivate yourself", 400)
		return
	}
	validRoles := map[string]bool{"admin": true, "user": true, "readonly": true}
	if !validRoles[req.Role] {
		req.Role = "user"
	}
	active := 1
	if req.Active != nil {
		active = *req.Active
	}
	result, err := h.DB.Exec(`UPDATE users SET display_name = ?, role = ?, active = ? WHERE id = ?`,
		req.DisplayName, req.Role, active, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.Err(w, "User not found", 404)
		return
	}
	response.JSON(w, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request, idStr string) {
	admin := h.RequireAdmin(w, r)
	if admin == nil {
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Err(w, "Invalid user ID", 400)
		return
	}
	if id == admin.ID {
		response.Err(w, "Cannot delete yourself", 400)
		return
	}
	res, err := h.DB.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		response.Err(w, "User not found", 404)
		return
	}
	h.DB.Exec("DELETE FROM sessions WHERE user_id = ?", id)
	response.JSON(w, map[string]string{"status": "deleted"})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request, idStr string) {
	if h.RequireAdmin(w, r) == nil {
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Err(w, "Invalid user ID", 400)
		return
	}
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, "Invalid request body", 400)
		return
	}
	if req.Password == "" {
		response.Err(w, "Password required", 400)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Err(w, "Failed to hash password", 500)
		return
	}
	result, err := h.DB.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(hash), id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.Err(w, "User not found", 404)
		return
	}
	response.JSON(w, map[string]string{"status": "password_reset"})
}

// --- API Key handlers ---

func GenerateAPIKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "zrp_" + hex.EncodeToString(b), nil
}

func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, name, key_prefix, created_by, created_at, last_used, expires_at, enabled FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		response.Err(w, "Failed to fetch API keys. Please try again.", 500)
		return
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var lastUsed, expiresAt *string
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.CreatedBy, &k.CreatedAt, &lastUsed, &expiresAt, &k.Enabled); err != nil {
			continue
		}
		k.LastUsed = lastUsed
		k.ExpiresAt = expiresAt
		keys = append(keys, k)
	}
	if keys == nil {
		keys = []APIKey{}
	}
	response.JSON(w, keys)
}

func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, "Invalid request body", 400)
		return
	}
	if req.Name == "" {
		response.Err(w, "Name is required", 400)
		return
	}

	key, err := GenerateAPIKey()
	if err != nil {
		response.Err(w, "Failed to generate key", 500)
		return
	}

	keyHash := HashAPIKey(key)
	keyPrefix := key[:12]

	var expiresAt interface{}
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		expiresAt = *req.ExpiresAt
	}

	result, err := h.DB.Exec(`INSERT INTO api_keys (name, key_hash, key_prefix, expires_at) VALUES (?, ?, ?, ?)`,
		req.Name, keyHash, keyPrefix, expiresAt)
	if err != nil {
		response.Err(w, "Failed to create API key. Please try again.", 500)
		return
	}

	id, _ := result.LastInsertId()
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         id,
		"name":       req.Name,
		"key":        key,
		"full_key":   key,
		"key_prefix": keyPrefix,
		"created_by": "admin",
		"created_at": time.Now().Format(time.RFC3339),
		"enabled":    1,
		"status":     "active",
		"message":    "Store this key securely. It will not be shown again.",
	})
}

func (h *Handler) DeleteAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	res, err := h.DB.Exec("DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		response.Err(w, "Failed to delete API key. Please try again.", 500)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		response.Err(w, "API key not found", 404)
		return
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "deleted", "api_key", id, "Revoked API key")
	json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}

func (h *Handler) ToggleAPIKey(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Enabled int `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Err(w, "Invalid body", 400)
		return
	}
	res, err := h.DB.Exec("UPDATE api_keys SET enabled = ? WHERE id = ?", body.Enabled, id)
	if err != nil {
		response.Err(w, "Failed to update API key. Please try again.", 500)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		response.Err(w, "API key not found", 404)
		return
	}
	action := "enabled"
	if body.Enabled == 0 {
		action = "disabled"
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, action, "api_key", id, "API key "+action)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// ValidateBearerToken checks an Authorization: Bearer token against the DB.
func (h *Handler) ValidateBearerToken(token string) bool {
	if !strings.HasPrefix(token, "zrp_") {
		return false
	}
	keyHash := HashAPIKey(token)
	var id int
	var enabled int
	var expiresAt *string
	err := h.DB.QueryRow("SELECT id, enabled, expires_at FROM api_keys WHERE key_hash = ?", keyHash).Scan(&id, &enabled, &expiresAt)
	if err != nil || enabled == 0 {
		return false
	}
	if expiresAt != nil && *expiresAt != "" {
		exp, err := time.Parse("2006-01-02T15:04:05Z", *expiresAt)
		if err != nil {
			exp, err = time.Parse("2006-01-02", *expiresAt)
		}
		if err == nil && time.Now().After(exp) {
			return false
		}
	}
	h.DB.Exec("UPDATE api_keys SET last_used = ? WHERE id = ?", time.Now().Format("2006-01-02 15:04:05"), id)
	return true
}

// --- General Settings handlers ---

func (h *Handler) GetGeneralSettings(w http.ResponseWriter, r *http.Request) {
	s := GeneralSettings{
		AppName:    generalSettingsDefaults["app_name"],
		Currency:   generalSettingsDefaults["currency"],
		DateFormat: generalSettingsDefaults["date_format"],
	}

	for _, key := range generalSettingsKeys {
		var val string
		err := h.DB.QueryRow("SELECT value FROM app_settings WHERE key = ?", "general_"+key).Scan(&val)
		if err != nil {
			continue
		}
		switch key {
		case "app_name":
			s.AppName = val
		case "company_name":
			s.CompanyName = val
		case "company_address":
			s.CompanyAddress = val
		case "currency":
			s.Currency = val
		case "date_format":
			s.DateFormat = val
		}
	}

	response.JSON(w, s)
}

func (h *Handler) PutGeneralSettings(w http.ResponseWriter, r *http.Request) {
	var s GeneralSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	vals := map[string]string{
		"app_name":        s.AppName,
		"company_name":    s.CompanyName,
		"company_address": s.CompanyAddress,
		"currency":        s.Currency,
		"date_format":     s.DateFormat,
	}

	for key, val := range vals {
		_, err := h.DB.Exec(`INSERT INTO app_settings (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, "general_"+key, val)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	response.JSON(w, s)
}

// --- Dashboard Widget handlers ---

func (h *Handler) GetDashboardWidgets(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query("SELECT id, user_id, widget_type, position, enabled FROM dashboard_widgets WHERE user_id=0 ORDER BY position ASC")
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var widgets []DashboardWidget
	for rows.Next() {
		var wg DashboardWidget
		rows.Scan(&wg.ID, &wg.UserID, &wg.WidgetType, &wg.Position, &wg.Enabled)
		widgets = append(widgets, wg)
	}
	if widgets == nil {
		widgets = []DashboardWidget{}
	}
	response.JSON(w, widgets)
}

func (h *Handler) UpdateDashboardWidgets(w http.ResponseWriter, r *http.Request) {
	var updates []struct {
		WidgetType string `json:"widget_type"`
		Position   int    `json:"position"`
		Enabled    int    `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	for _, u := range updates {
		h.DB.Exec("UPDATE dashboard_widgets SET position=?, enabled=? WHERE widget_type=? AND user_id=0",
			u.Position, u.Enabled, u.WidgetType)
	}
	h.GetDashboardWidgets(w, r)
}

// --- Query Profiler handlers ---

func (h *Handler) QueryProfilerStats(w http.ResponseWriter, r *http.Request) {
	if h.Profiler == nil {
		response.Err(w, "Query profiler not initialized", 500)
		return
	}
	stats := h.Profiler.GetStatsMap()
	response.JSON(w, stats)
}

func (h *Handler) QueryProfilerSlowQueries(w http.ResponseWriter, r *http.Request) {
	if h.Profiler == nil {
		response.Err(w, "Query profiler not initialized", 500)
		return
	}
	slow := h.Profiler.GetSlowQueriesAny()
	response.JSON(w, map[string]interface{}{
		"slow_queries": slow,
		"threshold":    h.Profiler.SlowThresholdString(),
	})
}

func (h *Handler) QueryProfilerAllQueries(w http.ResponseWriter, r *http.Request) {
	if h.Profiler == nil {
		response.Err(w, "Query profiler not initialized", 500)
		return
	}
	queries := h.Profiler.GetAllQueriesAny()
	response.JSON(w, map[string]interface{}{
		"queries": queries,
	})
}

func (h *Handler) QueryProfilerReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		response.Err(w, "method not allowed", 405)
		return
	}
	if h.Profiler == nil {
		response.Err(w, "Query profiler not initialized", 500)
		return
	}
	h.Profiler.Reset()
	response.JSON(w, map[string]interface{}{"message": "Profiler reset successfully"})
}
