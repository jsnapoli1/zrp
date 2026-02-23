package admin_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zrp/internal/handlers/admin"
	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

// newBackupTestHandler returns a Handler wired to a real temp backup directory.
// The returned cleanup function must be deferred.
func newBackupTestHandler(t *testing.T, backupDir string) *admin.Handler {
	t.Helper()
	db := setupAuthTestDB(t)
	t.Cleanup(func() { db.Close() })

	h := newTestHandler(db)

	// Wire real backup behaviour that writes to the temp dir.
	h.PerformBackup = func() error {
		name := fmt.Sprintf("zrp-backup-%s.db",
			time.Now().Format("2006-01-02T15-04-05"))
		return os.WriteFile(filepath.Join(backupDir, name), []byte("backup-data"), 0644)
	}
	h.ListBackups = func() ([]admin.BackupInfo, error) {
		entries, err := os.ReadDir(backupDir)
		if err != nil {
			if os.IsNotExist(err) {
				return []admin.BackupInfo{}, nil
			}
			return nil, err
		}
		var out []admin.BackupInfo
		for i := len(entries) - 1; i >= 0; i-- {
			e := entries[i]
			if strings.HasPrefix(e.Name(), "zrp-backup-") {
				info, _ := e.Info()
				out = append(out, admin.BackupInfo{
					Filename:  e.Name(),
					Size:      info.Size(),
					CreatedAt: info.ModTime().Format("2006-01-02T15:04:05"),
				})
			}
		}
		if out == nil {
			out = []admin.BackupInfo{}
		}
		return out, nil
	}

	return h
}

func TestHandleCreateBackup(t *testing.T) {
	backupDir := t.TempDir()
	h := newBackupTestHandler(t, backupDir)

	req := httptest.NewRequest("POST", "/api/v1/admin/backup", nil)
	w := httptest.NewRecorder()
	h.HandleCreateBackup(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data, _ := json.Marshal(resp.Data)
	var m map[string]string
	json.Unmarshal(data, &m)
	if m["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", m)
	}

	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 backup file, got %d", len(entries))
	}
	if !strings.HasPrefix(entries[0].Name(), "zrp-backup-") {
		t.Fatalf("unexpected filename: %s", entries[0].Name())
	}
}

func TestHandleListBackups(t *testing.T) {
	backupDir := t.TempDir()
	h := newBackupTestHandler(t, backupDir)

	// Create two backup files directly.
	os.WriteFile(filepath.Join(backupDir, "zrp-backup-2025-01-02T00-00-00.db"), []byte("data"), 0644)
	os.WriteFile(filepath.Join(backupDir, "zrp-backup-2025-01-01T00-00-00.db"), []byte("data"), 0644)

	req := httptest.NewRequest("GET", "/api/v1/admin/backups", nil)
	w := httptest.NewRecorder()
	h.HandleListBackups(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data, _ := json.Marshal(resp.Data)
	var backups []admin.BackupInfo
	json.Unmarshal(data, &backups)

	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(backups))
	}
	// Should be sorted newest first (reverse dir order).
	if backups[0].Filename < backups[1].Filename {
		t.Error("backups not sorted newest first")
	}
}

func TestHandleListBackupsEmpty(t *testing.T) {
	backupDir := filepath.Join(t.TempDir(), "nonexistent")
	h := newBackupTestHandler(t, backupDir)

	req := httptest.NewRequest("GET", "/api/v1/admin/backups", nil)
	w := httptest.NewRecorder()
	h.HandleListBackups(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp models.APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data, _ := json.Marshal(resp.Data)
	var backups []admin.BackupInfo
	json.Unmarshal(data, &backups)

	if len(backups) != 0 {
		t.Fatalf("expected 0 backups, got %d", len(backups))
	}
}

func TestHandleDeleteBackup(t *testing.T) {
	// HandleDeleteBackup uses a hard-coded "backups/" prefix, so create that dir.
	backupsDir := filepath.Join(t.TempDir(), "backups")
	os.MkdirAll(backupsDir, 0755)

	// chdir so the handler's relative "backups/<file>" resolves correctly.
	origDir, _ := os.Getwd()
	os.Chdir(filepath.Dir(backupsDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	filename := "zrp-backup-2025-01-01T00-00-00.db"
	os.WriteFile(filepath.Join(backupsDir, filename), []byte("data"), 0644)

	h := newBackupTestHandler(t, backupsDir)

	req := httptest.NewRequest("DELETE", "/api/v1/admin/backups/"+filename, nil)
	w := httptest.NewRecorder()
	h.HandleDeleteBackup(w, req, filename)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	entries, _ := os.ReadDir(backupsDir)
	if len(entries) != 0 {
		t.Fatalf("expected 0 files after delete, got %d", len(entries))
	}
}

func TestHandleDeleteBackupNotFound(t *testing.T) {
	backupsDir := filepath.Join(t.TempDir(), "backups")
	os.MkdirAll(backupsDir, 0755)
	origDir, _ := os.Getwd()
	os.Chdir(filepath.Dir(backupsDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	h := newBackupTestHandler(t, backupsDir)

	w := httptest.NewRecorder()
	h.HandleDeleteBackup(w, httptest.NewRequest("DELETE", "/api/v1/admin/backups/nonexistent.db", nil), "nonexistent.db")

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleDeleteBackupInvalidFilename(t *testing.T) {
	h := newBackupTestHandler(t, t.TempDir())

	w := httptest.NewRecorder()
	h.HandleDeleteBackup(w, httptest.NewRequest("DELETE", "/api/v1/admin/backups/../etc/passwd", nil), "../etc/passwd")

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleDownloadBackup(t *testing.T) {
	backupsDir := filepath.Join(t.TempDir(), "backups")
	os.MkdirAll(backupsDir, 0755)
	origDir, _ := os.Getwd()
	os.Chdir(filepath.Dir(backupsDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	filename := "zrp-backup-2025-01-01T00-00-00.db"
	os.WriteFile(filepath.Join(backupsDir, filename), []byte("backup-content"), 0644)

	h := newBackupTestHandler(t, backupsDir)

	req := httptest.NewRequest("GET", "/api/v1/admin/backups/"+filename, nil)
	w := httptest.NewRecorder()
	h.HandleDownloadBackup(w, req, filename)

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

func TestHandleDownloadBackupNotFound(t *testing.T) {
	backupsDir := filepath.Join(t.TempDir(), "backups")
	os.MkdirAll(backupsDir, 0755)
	origDir, _ := os.Getwd()
	os.Chdir(filepath.Dir(backupsDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	h := newBackupTestHandler(t, backupsDir)

	w := httptest.NewRecorder()
	h.HandleDownloadBackup(w, httptest.NewRequest("GET", "/api/v1/admin/backups/nope.db", nil), "nope.db")

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleDownloadBackupPathTraversal(t *testing.T) {
	h := newBackupTestHandler(t, t.TempDir())

	w := httptest.NewRecorder()
	h.HandleDownloadBackup(w, httptest.NewRequest("GET", "/", nil), "../etc/passwd")

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleRestoreBackupMissingFilename(t *testing.T) {
	h := newBackupTestHandler(t, t.TempDir())

	req := httptest.NewRequest("POST", "/api/v1/admin/restore", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.HandleRestoreBackup(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleRestoreBackupPathTraversal(t *testing.T) {
	h := newBackupTestHandler(t, t.TempDir())

	body := `{"filename":"../etc/passwd"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/restore", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleRestoreBackup(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRestoreBackupNotFound(t *testing.T) {
	backupsDir := filepath.Join(t.TempDir(), "backups")
	os.MkdirAll(backupsDir, 0755)
	origDir, _ := os.Getwd()
	os.Chdir(filepath.Dir(backupsDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	h := newBackupTestHandler(t, backupsDir)

	body := `{"filename":"nonexistent.db"}`
	req := httptest.NewRequest("POST", "/api/v1/admin/restore", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleRestoreBackup(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRestoreBackupSuccess(t *testing.T) {
	backupsDir := filepath.Join(t.TempDir(), "backups")
	os.MkdirAll(backupsDir, 0755)
	origDir, _ := os.Getwd()
	os.Chdir(filepath.Dir(backupsDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	// Create a backup file to restore from.
	filename := "zrp-backup-2025-01-01T00-00-00.db"
	os.WriteFile(filepath.Join(backupsDir, filename), []byte("restored-data"), 0644)

	// Create a temp DB file that the handler will overwrite.
	dbFile := filepath.Join(t.TempDir(), "test.db")
	os.WriteFile(dbFile, []byte("old-data"), 0644)

	db := setupAuthTestDB(t)
	h := newTestHandler(db)
	h.DBFilePath = func() string { return dbFile }
	h.PerformBackup = func() error {
		// Write a pre-restore backup.
		pre := filepath.Join(backupsDir, "zrp-backup-pre-restore.db")
		return os.WriteFile(pre, []byte("pre-restore"), 0644)
	}
	h.InitDB = func(path string) error {
		// Simulate reopening the DB (no-op for test).
		return nil
	}

	body := fmt.Sprintf(`{"filename":"%s"}`, filename)
	req := httptest.NewRequest("POST", "/api/v1/admin/restore", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleRestoreBackup(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the DB file was overwritten with backup data.
	data, _ := os.ReadFile(dbFile)
	if string(data) != "restored-data" {
		t.Errorf("expected db file to contain restored data, got %q", string(data))
	}

	// Verify pre-restore backup was created.
	entries, _ := os.ReadDir(backupsDir)
	found := false
	for _, e := range entries {
		if e.Name() == "zrp-backup-pre-restore.db" {
			found = true
		}
	}
	if !found {
		t.Error("expected pre-restore backup to be created")
	}
}
