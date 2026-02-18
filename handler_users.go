package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type UserFull struct {
	ID          int     `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	Active      int     `json:"active"`
	CreatedAt   string  `json:"created_at"`
	LastLogin   *string `json:"last_login"`
}

type CreateUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Active      *int   `json:"active"`
}

type ResetPasswordRequest struct {
	Password string `json:"password"`
}

func getCurrentUser(r *http.Request) *UserFull {
	cookie, err := r.Cookie("zrp_session")
	if err != nil {
		return nil
	}
	var u UserFull
	var lastLogin *string
	err = db.QueryRow(`SELECT u.id, u.username, u.display_name, u.role, u.active, u.created_at, u.last_login
		FROM sessions s JOIN users u ON s.user_id = u.id
		WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP`, cookie.Value).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Active, &u.CreatedAt, &lastLogin)
	if err != nil {
		return nil
	}
	u.LastLogin = lastLogin
	return &u
}

func requireAdmin(w http.ResponseWriter, r *http.Request) *UserFull {
	u := getCurrentUser(r)
	if u == nil {
		jsonErr(w, "Unauthorized", 401)
		return nil
	}
	if u.Role != "admin" {
		jsonErr(w, "Admin access required", 403)
		return nil
	}
	return u
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	rows, err := db.Query(`SELECT id, username, display_name, role, active, created_at, last_login FROM users ORDER BY id`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
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
	jsonResp(w, users)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Invalid request body", 400)
		return
	}
	if req.Username == "" || req.Password == "" {
		jsonErr(w, "Username and password required", 400)
		return
	}
	validRoles := map[string]bool{"admin": true, "user": true, "readonly": true}
	if !validRoles[req.Role] {
		req.Role = "user"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonErr(w, "Failed to hash password", 500)
		return
	}
	result, err := db.Exec(`INSERT INTO users (username, password_hash, display_name, role, active) VALUES (?, ?, ?, ?, 1)`,
		req.Username, string(hash), req.DisplayName, req.Role)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			jsonErr(w, "Username already exists", 409)
			return
		}
		jsonErr(w, err.Error(), 500)
		return
	}
	id, _ := result.LastInsertId()
	w.WriteHeader(201)
	jsonResp(w, map[string]interface{}{"id": id, "username": req.Username, "display_name": req.DisplayName, "role": req.Role})
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request, idStr string) {
	admin := requireAdmin(w, r)
	if admin == nil {
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonErr(w, "Invalid user ID", 400)
		return
	}
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Invalid request body", 400)
		return
	}
	// Admin can't deactivate themselves
	if req.Active != nil && *req.Active == 0 && id == admin.ID {
		jsonErr(w, "Cannot deactivate yourself", 400)
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
	_, err = db.Exec(`UPDATE users SET display_name = ?, role = ?, active = ? WHERE id = ?`,
		req.DisplayName, req.Role, active, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "updated"})
}

func handleResetPassword(w http.ResponseWriter, r *http.Request, idStr string) {
	if requireAdmin(w, r) == nil {
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonErr(w, "Invalid user ID", 400)
		return
	}
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "Invalid request body", 400)
		return
	}
	if req.Password == "" {
		jsonErr(w, "Password required", 400)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonErr(w, "Failed to hash password", 500)
		return
	}
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(hash), id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "password_reset"})
}
