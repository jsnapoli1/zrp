package main

import (
	"fmt"
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
	backupDir       = "backups"
	backupMu        sync.Mutex
	backupRetention = 7
)

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
	getAdminHandler().HandleCreateBackup(w, r)
}

func handleListBackups(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleListBackups(w, r)
}

func handleDeleteBackup(w http.ResponseWriter, r *http.Request, filename string) {
	getAdminHandler().HandleDeleteBackup(w, r, filename)
}

func handleDownloadBackup(w http.ResponseWriter, r *http.Request, filename string) {
	getAdminHandler().HandleDownloadBackup(w, r, filename)
}

func handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleRestoreBackup(w, r)
}
