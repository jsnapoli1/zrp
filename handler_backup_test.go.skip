package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupBackupTest(t *testing.T) (*http.Cookie, func()) {
	t.Helper()
	cleanup := setupTestDB(t)
	// Use a temp backup dir
	oldDir := backupDir
	backupDir = t.TempDir()
	cookie := loginAdmin(t)
	return cookie, func() {
		backupDir = oldDir
		cleanup()
	}
}

func TestCreateBackup(t *testing.T) {
	cookie, cleanup := setupBackupTest(t)
	defer cleanup()

	req := authedRequest("POST", "/api/v1/admin/backup", "", cookie)
	w := httptest.NewRecorder()
	handleCreateBackup(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Response is wrapped in {"data": {"status":"ok","message":"..."}}
	var resp struct {
		Data map[string]string `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", resp.Data)
	}

	// Verify file was created
	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 backup file, got %d", len(entries))
	}
	if !strings.HasPrefix(entries[0].Name(), "zrp-backup-") {
		t.Fatalf("unexpected filename: %s", entries[0].Name())
	}
}

func TestListBackups(t *testing.T) {
	cookie, cleanup := setupBackupTest(t)
	defer cleanup()

	// Create two backups
	handleCreateBackup(httptest.NewRecorder(), authedRequest("POST", "/api/v1/admin/backup", "", cookie))
	// Create another with slightly different name
	os.WriteFile(filepath.Join(backupDir, "zrp-backup-2025-01-01T00-00-00.db"), []byte("fake"), 0644)

	req := authedRequest("GET", "/api/v1/admin/backups", "", cookie)
	w := httptest.NewRecorder()
	handleListBackups(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Data []BackupInfo `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(resp.Data))
	}
	// Should be sorted newest first
	if resp.Data[0].Filename < resp.Data[1].Filename {
		t.Error("backups not sorted newest first")
	}
}

func TestListBackupsEmpty(t *testing.T) {
	cookie, cleanup := setupBackupTest(t)
	defer cleanup()

	// Point to a non-existent directory
	backupDir = filepath.Join(t.TempDir(), "nonexistent")

	req := authedRequest("GET", "/api/v1/admin/backups", "", cookie)
	w := httptest.NewRecorder()
	handleListBackups(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Data []BackupInfo `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data == nil {
		resp.Data = []BackupInfo{}
	}
	if len(resp.Data) != 0 {
		t.Fatalf("expected 0 backups, got %d", len(resp.Data))
	}
}

func TestDeleteBackup(t *testing.T) {
	cookie, cleanup := setupBackupTest(t)
	defer cleanup()

	// Create a backup
	handleCreateBackup(httptest.NewRecorder(), authedRequest("POST", "/api/v1/admin/backup", "", cookie))
	entries, _ := os.ReadDir(backupDir)
	filename := entries[0].Name()

	req := authedRequest("DELETE", "/api/v1/admin/backups/"+filename, "", cookie)
	w := httptest.NewRecorder()
	handleDeleteBackup(w, req, filename)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	entries, _ = os.ReadDir(backupDir)
	if len(entries) != 0 {
		t.Fatalf("expected 0 files after delete, got %d", len(entries))
	}
}

func TestDeleteBackupNotFound(t *testing.T) {
	_, cleanup := setupBackupTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	handleDeleteBackup(w, httptest.NewRequest("DELETE", "/api/v1/admin/backups/nonexistent.db", nil), "nonexistent.db")

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteBackupInvalidFilename(t *testing.T) {
	_, cleanup := setupBackupTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	handleDeleteBackup(w, httptest.NewRequest("DELETE", "/api/v1/admin/backups/../etc/passwd", nil), "../etc/passwd")

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDownloadBackup(t *testing.T) {
	cookie, cleanup := setupBackupTest(t)
	defer cleanup()

	// Create a backup
	handleCreateBackup(httptest.NewRecorder(), authedRequest("POST", "/api/v1/admin/backup", "", cookie))
	entries, _ := os.ReadDir(backupDir)
	filename := entries[0].Name()

	req := authedRequest("GET", "/api/v1/admin/backups/"+filename, "", cookie)
	w := httptest.NewRecorder()
	handleDownloadBackup(w, req, filename)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("expected octet-stream content type, got %s", ct)
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, filename) {
		t.Errorf("Content-Disposition should contain filename, got %s", cd)
	}
	if w.Body.Len() == 0 {
		t.Error("downloaded backup is empty")
	}
}

func TestDownloadBackupNotFound(t *testing.T) {
	_, cleanup := setupBackupTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	handleDownloadBackup(w, httptest.NewRequest("GET", "/api/v1/admin/backups/nope.db", nil), "nope.db")

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDownloadBackupPathTraversal(t *testing.T) {
	_, cleanup := setupBackupTest(t)
	defer cleanup()

	w := httptest.NewRecorder()
	handleDownloadBackup(w, httptest.NewRequest("GET", "/", nil), "../etc/passwd")

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRestoreBackup(t *testing.T) {
	cookie, cleanup := setupBackupTest(t)
	defer cleanup()

	// Set dbFilePath to the test DB so restore can write to it
	oldDBFilePath := dbFilePath
	dbFilePath = fmt.Sprintf("test_%s.db", t.Name())
	defer func() { dbFilePath = oldDBFilePath }()

	// Create a backup first
	handleCreateBackup(httptest.NewRecorder(), authedRequest("POST", "/api/v1/admin/backup", "", cookie))
	entries, _ := os.ReadDir(backupDir)
	filename := entries[0].Name()

	// Wait to ensure pre-restore backup gets a different timestamp
	time.Sleep(1100 * time.Millisecond)

	body := fmt.Sprintf(`{"filename":"%s"}`, filename)
	req := authedRequest("POST", "/api/v1/admin/restore", body, cookie)
	w := httptest.NewRecorder()
	handleRestoreBackup(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should have created a pre-restore backup (now 2 total)
	entries, _ = os.ReadDir(backupDir)
	if len(entries) < 2 {
		t.Errorf("expected pre-restore backup to be created, got %d files", len(entries))
	}
}

func TestRestoreBackupNotFound(t *testing.T) {
	cookie, cleanup := setupBackupTest(t)
	defer cleanup()

	body := `{"filename":"nonexistent.db"}`
	req := authedRequest("POST", "/api/v1/admin/restore", body, cookie)
	w := httptest.NewRecorder()
	handleRestoreBackup(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRestoreBackupMissingFilename(t *testing.T) {
	cookie, cleanup := setupBackupTest(t)
	defer cleanup()

	req := authedRequest("POST", "/api/v1/admin/restore", `{}`, cookie)
	w := httptest.NewRecorder()
	handleRestoreBackup(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRestoreBackupPathTraversal(t *testing.T) {
	cookie, cleanup := setupBackupTest(t)
	defer cleanup()

	body := `{"filename":"../etc/passwd"}`
	req := authedRequest("POST", "/api/v1/admin/restore", body, cookie)
	w := httptest.NewRecorder()
	handleRestoreBackup(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCleanOldBackups(t *testing.T) {
	_, cleanup := setupBackupTest(t)
	defer cleanup()

	oldRetention := backupRetention
	backupRetention = 3
	defer func() { backupRetention = oldRetention }()

	// Create 5 fake backups
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("zrp-backup-2025-01-%02dT00-00-00.db", i+1)
		os.WriteFile(filepath.Join(backupDir, name), []byte("data"), 0644)
	}

	cleanOldBackups()

	entries, _ := os.ReadDir(backupDir)
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "zrp-backup-") {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 backups after cleanup, got %d", count)
	}

	// Should keep the newest 3 (03, 04, 05)
	for _, e := range entries {
		name := e.Name()
		if strings.Contains(name, "01T") || strings.Contains(name, "02T") {
			t.Errorf("old backup %s should have been deleted", name)
		}
	}
}

func TestCleanOldBackupsUnderRetention(t *testing.T) {
	_, cleanup := setupBackupTest(t)
	defer cleanup()

	// Create just 2 backups (under default retention of 7)
	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("zrp-backup-2025-01-%02dT00-00-00.db", i+1)
		os.WriteFile(filepath.Join(backupDir, name), []byte("data"), 0644)
	}

	cleanOldBackups()

	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 backups (none deleted), got %d", len(entries))
	}
}
