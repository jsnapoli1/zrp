package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupBackupTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create test tables
	_, err = testDB.Exec(`
		CREATE TABLE ecos (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ecos table: %v", err)
	}

	_, err = testDB.Exec(`
		CREATE TABLE inventory (
			ipn TEXT PRIMARY KEY,
			qty_on_hand REAL DEFAULT 0,
			description TEXT DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create inventory table: %v", err)
	}

	return testDB
}

func setupBackupDir(t *testing.T) string {
	tmpDir := t.TempDir()
	oldBackupDir := backupDir
	backupDir = tmpDir
	t.Cleanup(func() { backupDir = oldBackupDir })
	return tmpDir
}

func setupTestDBFile(t *testing.T) string {
	tmpFile := filepath.Join(t.TempDir(), "test.db")
	testDB, err := sql.Open("sqlite", tmpFile)
	if err != nil {
		t.Fatalf("Failed to create test DB file: %v", err)
	}
	testDB.Exec("CREATE TABLE test_data (id INTEGER PRIMARY KEY, value TEXT)")
	testDB.Exec("INSERT INTO test_data (value) VALUES ('test123')")
	testDB.Close()

	oldDBFilePath := dbFilePath
	dbFilePath = tmpFile
	t.Cleanup(func() { dbFilePath = oldDBFilePath })
	return tmpFile
}

func TestPerformBackup_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupBackupTestDB(t)
	defer db.Close()

	tmpDir := setupBackupDir(t)
	tmpFile := setupTestDBFile(t)

	// Insert test data
	db.Exec("INSERT INTO ecos (id, title) VALUES ('ECO-001', 'Test ECO')")

	err := performBackup()
	if err != nil {
		t.Fatalf("performBackup failed: %v", err)
	}

	// Verify backup file was created
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read backup dir: %v", err)
	}

	if len(entries) == 0 {
		t.Error("No backup file created")
	}

	// Verify backup filename format
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "zrp-backup-") && strings.HasSuffix(e.Name(), ".db") {
			found = true
			info, _ := e.Info()
			if info.Size() == 0 {
				t.Error("Backup file is empty")
			}
		}
	}
	if !found {
		t.Error("Backup file with correct format not found")
	}

	// Verify it's a valid database
	backupPath := filepath.Join(tmpDir, entries[0].Name())
	testBackupDB, err := sql.Open("sqlite", backupPath)
	if err != nil {
		t.Fatalf("Failed to open backup file: %v", err)
	}
	defer testBackupDB.Close()

	_ = tmpFile // use tmpFile to avoid unused warning
}

func TestListBackups_Empty(t *testing.T) {
	setupBackupDir(t)

	backups, err := listBackups()
	if err != nil {
		t.Fatalf("listBackups failed: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("Expected 0 backups, got %d", len(backups))
	}
}

func TestListBackups_MultipleFiles(t *testing.T) {
	tmpDir := setupBackupDir(t)

	// Create test backup files
	testFiles := []struct {
		name    string
		content string
	}{
		{"zrp-backup-2024-01-01T10-00-00.db", "backup1"},
		{"zrp-backup-2024-01-02T10-00-00.db", "backup2"},
		{"zrp-backup-2024-01-03T10-00-00.db", "backup3"},
		{"not-a-backup.db", "ignore"},
		{"zrp-backup-incomplete", "ignore"},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tmpDir, tf.name)
		os.WriteFile(path, []byte(tf.content), 0644)
		time.Sleep(10 * time.Millisecond) // Ensure different mod times
	}

	backups, err := listBackups()
	if err != nil {
		t.Fatalf("listBackups failed: %v", err)
	}

	// Should only include valid backup files
	if len(backups) != 3 {
		t.Errorf("Expected 3 backups, got %d", len(backups))
	}

	// Verify sorted newest first
	if len(backups) >= 2 {
		if backups[0].Filename < backups[1].Filename {
			t.Error("Backups not sorted newest first")
		}
	}

	// Verify fields
	for _, b := range backups {
		if b.Filename == "" {
			t.Error("Backup filename is empty")
		}
		if b.Size == 0 {
			t.Error("Backup size is 0")
		}
		if b.CreatedAt == "" {
			t.Error("Backup created_at is empty")
		}
		// Verify RFC3339 format
		_, err := time.Parse(time.RFC3339, b.CreatedAt)
		if err != nil {
			t.Errorf("Invalid created_at format: %v", err)
		}
	}
}

func TestHandleListBackups_Success(t *testing.T) {
	tmpDir := setupBackupDir(t)

	// Create test backup file
	path := filepath.Join(tmpDir, "zrp-backup-2024-01-01T10-00-00.db")
	os.WriteFile(path, []byte("test backup"), 0644)

	req := httptest.NewRequest("GET", "/api/backups", nil)
	w := httptest.NewRecorder()

	handleListBackups(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var backups []BackupInfo
	if err := json.NewDecoder(w.Body).Decode(&backups); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(backups))
	}
}

func TestHandleCreateBackup_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupBackupTestDB(t)
	defer db.Close()

	setupBackupDir(t)
	setupTestDBFile(t)

	req := httptest.NewRequest("POST", "/api/backups", nil)
	w := httptest.NewRecorder()

	handleCreateBackup(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", resp["status"])
	}
}

func TestHandleDeleteBackup_Success(t *testing.T) {
	tmpDir := setupBackupDir(t)

	filename := "zrp-backup-2024-01-01T10-00-00.db"
	path := filepath.Join(tmpDir, filename)
	os.WriteFile(path, []byte("test"), 0644)

	req := httptest.NewRequest("DELETE", "/api/backups/"+filename, nil)
	w := httptest.NewRecorder()

	handleDeleteBackup(w, req, filename)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify file was deleted
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("Backup file was not deleted")
	}
}

func TestHandleDeleteBackup_NotFound(t *testing.T) {
	setupBackupDir(t)

	req := httptest.NewRequest("DELETE", "/api/backups/nonexistent.db", nil)
	w := httptest.NewRecorder()

	handleDeleteBackup(w, req, "nonexistent.db")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleDeleteBackup_InvalidFilename(t *testing.T) {
	setupBackupDir(t)

	tests := []struct {
		name     string
		filename string
	}{
		{"path traversal", "../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"subdirectory", "subdir/file.db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/api/backups/"+tt.filename, nil)
			w := httptest.NewRecorder()

			handleDeleteBackup(w, req, tt.filename)

			if w.Code != 400 {
				t.Errorf("Expected status 400 for %s, got %d", tt.filename, w.Code)
			}
		})
	}
}

func TestHandleDownloadBackup_Success(t *testing.T) {
	tmpDir := setupBackupDir(t)

	filename := "zrp-backup-2024-01-01T10-00-00.db"
	content := "test backup content"
	path := filepath.Join(tmpDir, filename)
	os.WriteFile(path, []byte(content), 0644)

	req := httptest.NewRequest("GET", "/api/backups/"+filename+"/download", nil)
	w := httptest.NewRecorder()

	handleDownloadBackup(w, req, filename)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("Expected Content-Type application/octet-stream, got %s", w.Header().Get("Content-Type"))
	}

	if !strings.Contains(w.Header().Get("Content-Disposition"), filename) {
		t.Errorf("Content-Disposition doesn't contain filename: %s", w.Header().Get("Content-Disposition"))
	}

	if w.Body.String() != content {
		t.Errorf("Expected body %q, got %q", content, w.Body.String())
	}
}

func TestHandleDownloadBackup_NotFound(t *testing.T) {
	setupBackupDir(t)

	req := httptest.NewRequest("GET", "/api/backups/nonexistent.db/download", nil)
	w := httptest.NewRecorder()

	handleDownloadBackup(w, req, "nonexistent.db")

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleDownloadBackup_InvalidFilename(t *testing.T) {
	setupBackupDir(t)

	tests := []string{"../etc/passwd", "/etc/passwd", "subdir/file.db"}

	for _, filename := range tests {
		t.Run(filename, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/backups/"+filename+"/download", nil)
			w := httptest.NewRecorder()

			handleDownloadBackup(w, req, filename)

			if w.Code != 400 {
				t.Errorf("Expected status 400 for %s, got %d", filename, w.Code)
			}
		})
	}
}

func TestHandleRestoreBackup_Success(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()

	tmpDir := setupBackupDir(t)
	tmpDBFile := setupTestDBFile(t)

	// Create a backup file with test data
	backupFile := filepath.Join(tmpDir, "zrp-backup-restore-test.db")
	backupDB, err := sql.Open("sqlite", backupFile)
	if err != nil {
		t.Fatalf("Failed to create backup DB: %v", err)
	}
	backupDB.Exec("CREATE TABLE test_data (id INTEGER PRIMARY KEY, value TEXT)")
	backupDB.Exec("INSERT INTO test_data (value) VALUES ('restored_data')")
	backupDB.Close()

	db = setupBackupTestDB(t)
	defer db.Close()

	body := map[string]string{"filename": "zrp-backup-restore-test.db"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/backups/restore", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleRestoreBackup(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", resp["status"])
	}

	// Verify database was replaced
	restoredData, err := os.ReadFile(tmpDBFile)
	if err != nil {
		t.Fatalf("Failed to read restored DB: %v", err)
	}

	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if !bytes.Equal(restoredData, backupData) {
		t.Error("Database was not properly restored")
	}
}

func TestHandleRestoreBackup_MissingFilename(t *testing.T) {
	setupBackupDir(t)

	req := httptest.NewRequest("POST", "/api/backups/restore", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleRestoreBackup(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleRestoreBackup_InvalidFilename(t *testing.T) {
	setupBackupDir(t)

	tests := []string{"../etc/passwd", "/etc/passwd", "subdir/file.db"}

	for _, filename := range tests {
		t.Run(filename, func(t *testing.T) {
			body := map[string]string{"filename": filename}
			bodyBytes, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/api/backups/restore", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handleRestoreBackup(w, req)

			if w.Code != 400 {
				t.Errorf("Expected status 400 for %s, got %d", filename, w.Code)
			}
		})
	}
}

func TestHandleRestoreBackup_NotFound(t *testing.T) {
	setupBackupDir(t)

	body := map[string]string{"filename": "nonexistent.db"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/backups/restore", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleRestoreBackup(w, req)

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestCleanOldBackups_Retention(t *testing.T) {
	tmpDir := setupBackupDir(t)

	// Set retention to 3
	oldRetention := backupRetention
	backupRetention = 3
	defer func() { backupRetention = oldRetention }()

	// Create 5 backup files with different timestamps
	for i := 1; i <= 5; i++ {
		filename := fmt.Sprintf("zrp-backup-2024-01-%02dT10-00-00.db", i)
		path := filepath.Join(tmpDir, filename)
		os.WriteFile(path, []byte(fmt.Sprintf("backup%d", i)), 0644)
	}

	cleanOldBackups()

	// Should keep only 3 newest (03, 04, 05)
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read backup dir: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 backups after cleanup, got %d", len(entries))
	}

	// Verify oldest were deleted
	for _, e := range entries {
		if strings.Contains(e.Name(), "-01T") || strings.Contains(e.Name(), "-02T") {
			t.Errorf("Old backup not deleted: %s", e.Name())
		}
	}
}

func TestCleanOldBackups_NoCleanupNeeded(t *testing.T) {
	tmpDir := setupBackupDir(t)

	oldRetention := backupRetention
	backupRetention = 5
	defer func() { backupRetention = oldRetention }()

	// Create 3 backup files
	for i := 1; i <= 3; i++ {
		filename := fmt.Sprintf("zrp-backup-2024-01-%02dT10-00-00.db", i)
		path := filepath.Join(tmpDir, filename)
		os.WriteFile(path, []byte(fmt.Sprintf("backup%d", i)), 0644)
	}

	cleanOldBackups()

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read backup dir: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 backups (no cleanup), got %d", len(entries))
	}
}

func TestBackupConcurrency(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupBackupTestDB(t)
	defer db.Close()

	setupBackupDir(t)
	setupTestDBFile(t)

	// Test concurrent backup attempts
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			done <- performBackup()
		}()
	}

	// All should complete without error (mutex protects)
	for i := 0; i < 3; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Concurrent backup %d failed: %v", i, err)
		}
	}
}

func TestBackupWithLargeData(t *testing.T) {
	oldDB := db
	defer func() { db = oldDB }()
	db = setupBackupTestDB(t)
	defer db.Close()

	tmpDir := setupBackupDir(t)
	setupTestDBFile(t)

	// Insert large amount of test data
	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO inventory (ipn, description) VALUES (?, ?)",
			fmt.Sprintf("TEST-%04d", i),
			strings.Repeat("x", 1000))
	}

	err := performBackup()
	if err != nil {
		t.Fatalf("Backup with large data failed: %v", err)
	}

	// Verify backup file size
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) > 0 {
		info, _ := entries[0].Info()
		if info.Size() < 10000 {
			t.Error("Backup file seems too small for large dataset")
		}
	}
}
