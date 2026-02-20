package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	backupDir     = "backups"
	backupMu      sync.Mutex
	backupRetention = 7
)

type BackupInfo struct {
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

func startAutoBackup(backupTime string) {
	hour, min := 2, 0
	if backupTime != "" {
		fmt.Sscanf(backupTime, "%d:%d", &hour, &min)
	}

	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())
			if next.Before(now) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))

			log.Println("Running scheduled backup...")
			if err := performBackup(); err != nil {
				log.Printf("Auto-backup failed: %v", err)
			} else {
				log.Println("Auto-backup completed")
				cleanOldBackups()
			}
		}
	}()
}

func performBackup() error {
	backupMu.Lock()
	defer backupMu.Unlock()

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	ts := time.Now().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("zrp-backup-%s.db", ts)
	destPath := filepath.Join(backupDir, filename)

	// If file exists, add a counter suffix
	counter := 1
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		filename = fmt.Sprintf("zrp-backup-%s-%d.db", ts, counter)
		destPath = filepath.Join(backupDir, filename)
		counter++
	}

	_, err := db.Exec(fmt.Sprintf(`VACUUM INTO '%s'`, destPath))
	if err != nil {
		return fmt.Errorf("vacuum into: %w", err)
	}

	return nil
}

func cleanOldBackups() {
	backups, err := listBackups()
	if err != nil {
		log.Printf("Failed to list backups for cleanup: %v", err)
		return
	}

	if len(backups) <= backupRetention {
		return
	}

	// Sort oldest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Filename < backups[j].Filename
	})

	for i := 0; i < len(backups)-backupRetention; i++ {
		path := filepath.Join(backupDir, backups[i].Filename)
		if err := os.Remove(path); err != nil {
			log.Printf("Failed to remove old backup %s: %v", backups[i].Filename, err)
		} else {
			log.Printf("Removed old backup: %s", backups[i].Filename)
		}
	}
}

func listBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, err
	}

	var backups []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "zrp-backup-") || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Filename:  e.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Filename > backups[j].Filename
	})

	return backups, nil
}

func handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	if err := performBackup(); err != nil {
		jsonErr(w, fmt.Sprintf("Backup failed: %v", err), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok", "message": "Backup created"})
}

func handleListBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := listBackups()
	if err != nil {
		jsonErr(w, fmt.Sprintf("Failed to list backups: %v", err), 500)
		return
	}
	jsonResp(w, backups)
}

func handleDeleteBackup(w http.ResponseWriter, r *http.Request, filename string) {
	if strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		jsonErr(w, "Invalid filename", 400)
		return
	}

	path := filepath.Join(backupDir, filename)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			jsonErr(w, "Backup not found", 404)
		} else {
			jsonErr(w, fmt.Sprintf("Failed to delete: %v", err), 500)
		}
		return
	}
	jsonResp(w, map[string]string{"status": "ok"})
}

func handleDownloadBackup(w http.ResponseWriter, r *http.Request, filename string) {
	if strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		http.Error(w, "Invalid filename", 400)
		return
	}

	path := filepath.Join(backupDir, filename)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Failed to open backup", 500)
		}
		return
	}
	defer f.Close()

	info, _ := f.Stat()
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	io.Copy(w, f)
}

func handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := decodeBody(r, &req); err != nil || req.Filename == "" {
		jsonErr(w, "filename is required", 400)
		return
	}

	if strings.Contains(req.Filename, "/") || strings.Contains(req.Filename, "..") {
		jsonErr(w, "Invalid filename", 400)
		return
	}

	backupPath := filepath.Join(backupDir, req.Filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		jsonErr(w, "Backup not found", 404)
		return
	}

	// Create a pre-restore backup first
	if err := performBackup(); err != nil {
		log.Printf("Warning: pre-restore backup failed: %v", err)
	}

	// Read backup file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		jsonErr(w, fmt.Sprintf("Failed to read backup: %v", err), 500)
		return
	}

	// Write over current database file
	if err := os.WriteFile(dbFilePath, data, 0644); err != nil {
		jsonErr(w, fmt.Sprintf("Failed to restore: %v", err), 500)
		return
	}

	// Reopen database connection
	db.Close()
	if err := initDB(dbFilePath); err != nil {
		jsonErr(w, fmt.Sprintf("Failed to reopen DB after restore: %v", err), 500)
		return
	}

	jsonResp(w, map[string]string{"status": "ok", "message": "Database restored from " + req.Filename})
}
