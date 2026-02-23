package admin

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"zrp/internal/response"
)

// HandleCreateBackup creates a new backup.
func (h *Handler) HandleCreateBackup(w http.ResponseWriter, r *http.Request) {
	if err := h.PerformBackup(); err != nil {
		response.Err(w, fmt.Sprintf("Backup failed: %v", err), 500)
		return
	}
	response.JSON(w, map[string]string{"status": "ok", "message": "Backup created"})
}

// HandleListBackups lists all backups.
func (h *Handler) HandleListBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := h.ListBackups()
	if err != nil {
		response.Err(w, fmt.Sprintf("Failed to list backups: %v", err), 500)
		return
	}
	response.JSON(w, backups)
}

// HandleDeleteBackup deletes a backup by filename.
func (h *Handler) HandleDeleteBackup(w http.ResponseWriter, r *http.Request, filename string) {
	if strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		response.Err(w, "Invalid filename", 400)
		return
	}

	path := fmt.Sprintf("backups/%s", filename)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			response.Err(w, "Backup not found", 404)
		} else {
			response.Err(w, fmt.Sprintf("Failed to delete: %v", err), 500)
		}
		return
	}
	response.JSON(w, map[string]string{"status": "ok"})
}

// HandleDownloadBackup downloads a backup file.
func (h *Handler) HandleDownloadBackup(w http.ResponseWriter, r *http.Request, filename string) {
	if strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		http.Error(w, "Invalid filename", 400)
		return
	}

	path := fmt.Sprintf("backups/%s", filename)
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
	// Use io.Copy equivalent
	http.ServeContent(w, r, filename, info.ModTime(), f)
}

// HandleRestoreBackup restores the database from a backup.
func (h *Handler) HandleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := response.DecodeBody(r, &req); err != nil || req.Filename == "" {
		response.Err(w, "filename is required", 400)
		return
	}

	if strings.Contains(req.Filename, "/") || strings.Contains(req.Filename, "..") {
		response.Err(w, "Invalid filename", 400)
		return
	}

	backupPath := fmt.Sprintf("backups/%s", req.Filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		response.Err(w, "Backup not found", 404)
		return
	}

	// Create a pre-restore backup first
	if h.PerformBackup != nil {
		h.PerformBackup()
	}

	// Read backup file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		response.Err(w, fmt.Sprintf("Failed to read backup: %v", err), 500)
		return
	}

	// Write over current database file
	dbFilePath := h.DBFilePath()
	if err := os.WriteFile(dbFilePath, data, 0644); err != nil {
		response.Err(w, fmt.Sprintf("Failed to restore: %v", err), 500)
		return
	}

	// Reopen database connection
	h.DB.Close()
	if err := h.InitDB(dbFilePath); err != nil {
		response.Err(w, fmt.Sprintf("Failed to reopen DB after restore: %v", err), 500)
		return
	}

	response.JSON(w, map[string]string{"status": "ok", "message": "Database restored from " + req.Filename})
}
