package admin_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"zrp/internal/models"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func TestHandleListUsers_AsAdmin(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	createTestUserLocal(t, db, "user1", "password", "user", true)
	createTestUserLocal(t, db, "user2", "password", "readonly", true)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	usersJSON, _ := json.Marshal(response.Data)
	var result []map[string]interface{}
	json.Unmarshal(usersJSON, &result)

	if len(result) != 3 {
		t.Errorf("Expected 3 users, got %d", len(result))
	}
}

func TestHandleListUsers_AsNonAdmin(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	userID := createTestUserLocal(t, db, "user1", "password", "user", true)
	userToken := createTestSessionLocal(t, db, userID)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: userToken})
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	if w.Code != 403 {
		t.Errorf("Expected status 403 for non-admin, got %d", w.Code)
	}
}

func TestHandleListUsers_Unauthorized(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	if w.Code != 401 {
		t.Errorf("Expected status 401 for unauthorized, got %d", w.Code)
	}
}

func TestHandleCreateUser_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	reqBody := `{
		"username": "newuser",
		"password": "SecurePass123!",
		"display_name": "New User",
		"role": "user"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response models.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	result, ok := response.Data.(map[string]interface{})
	if !ok || result["id"] == nil {
		t.Error("Expected user ID in response")
	}

	// Verify user was created
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE username=?", "newuser").Scan(&username)
	if err != nil {
		t.Fatalf("Expected user to be created: %v", err)
	}

	if username != "newuser" {
		t.Errorf("Expected username 'newuser', got %s", username)
	}
}

func TestHandleCreateUser_MissingUsername(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	reqBody := `{
		"password": "SecurePass123!",
		"role": "user"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateUser_MissingPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	reqBody := `{
		"username": "newuser",
		"role": "user"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleCreateUser_DefaultRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	reqBody := `{
		"username": "newuser",
		"password": "SecurePass123!"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.CreateUser(w, req)

	// Verify default role is 'user'
	var role string
	db.QueryRow("SELECT role FROM users WHERE username=?", "newuser").Scan(&role)
	if role != "user" {
		t.Errorf("Expected default role 'user', got %s", role)
	}
}

func TestHandleUpdateUser_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)
	targetUserID := createTestUserLocal(t, db, "target", "password", "user", true)

	reqBody := `{
		"display_name": "Updated Name",
		"role": "readonly"
	}`
	req := httptest.NewRequest("PUT", "/api/v1/users/"+strconv.Itoa(targetUserID), bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.UpdateUser(w, req, strconv.Itoa(targetUserID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify updates
	var displayName, role string
	db.QueryRow("SELECT display_name, role FROM users WHERE id=?", targetUserID).Scan(&displayName, &role)

	if displayName != "Updated Name" {
		t.Errorf("Expected display_name 'Updated Name', got %s", displayName)
	}
	if role != "readonly" {
		t.Errorf("Expected role 'readonly', got %s", role)
	}
}

func TestHandleUpdateUser_Deactivate(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)
	targetUserID := createTestUserLocal(t, db, "target", "password", "user", true)

	reqBody := `{
		"display_name": "Deactivated User",
		"role": "user",
		"active": 0
	}`
	req := httptest.NewRequest("PUT", "/api/v1/users/"+strconv.Itoa(targetUserID), bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.UpdateUser(w, req, strconv.Itoa(targetUserID))

	// Verify deactivation
	var active int
	db.QueryRow("SELECT active FROM users WHERE id=?", targetUserID).Scan(&active)

	if active != 0 {
		t.Errorf("Expected active=0, got %d", active)
	}
}

func TestHandleUpdateUser_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	reqBody := `{
		"display_name": "Updated",
		"role": "user"
	}`
	req := httptest.NewRequest("PUT", "/api/v1/users/9999", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.UpdateUser(w, req, "9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleDeleteUser_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)
	targetUserID := createTestUserLocal(t, db, "target", "password", "user", true)

	req := httptest.NewRequest("DELETE", "/api/v1/users/"+strconv.Itoa(targetUserID), nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.DeleteUser(w, req, strconv.Itoa(targetUserID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify user was deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE id=?", targetUserID).Scan(&count)
	if count != 0 {
		t.Error("Expected user to be deleted")
	}
}

func TestHandleDeleteUser_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	req := httptest.NewRequest("DELETE", "/api/v1/users/9999", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.DeleteUser(w, req, "9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleResetPassword_Success(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)
	targetUserID := createTestUserLocal(t, db, "target", "oldpassword", "user", true)

	reqBody := `{
		"password": "newpassword123"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users/"+strconv.Itoa(targetUserID)+"/reset-password", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.ResetPassword(w, req, strconv.Itoa(targetUserID))

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify password was changed
	var passwordHash string
	db.QueryRow("SELECT password_hash FROM users WHERE id=?", targetUserID).Scan(&passwordHash)

	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("newpassword123"))
	if err != nil {
		t.Error("Expected password to be updated")
	}
}

func TestHandleResetPassword_MissingPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)
	targetUserID := createTestUserLocal(t, db, "target", "password", "user", true)

	reqBody := `{}`
	req := httptest.NewRequest("POST", "/api/v1/users/"+strconv.Itoa(targetUserID)+"/reset-password", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.ResetPassword(w, req, strconv.Itoa(targetUserID))

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleResetPassword_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	adminID := createTestUserLocal(t, db, "admin", "password", "admin", true)
	adminToken := createTestSessionLocal(t, db, adminID)

	reqBody := `{
		"password": "newpassword"
	}`
	req := httptest.NewRequest("POST", "/api/v1/users/9999/reset-password", bytes.NewBufferString(reqBody))
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: adminToken})
	w := httptest.NewRecorder()

	h.ResetPassword(w, req, "9999")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}
