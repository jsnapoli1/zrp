package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Attachment struct {
	ID           int    `json:"id"`
	Module       string `json:"module"`
	RecordID     string `json:"record_id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	SizeBytes    int64  `json:"size_bytes"`
	MimeType     string `json:"mime_type"`
	UploadedBy   string `json:"uploaded_by"`
	CreatedAt    string `json:"created_at"`
}

func handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		jsonErr(w, "Failed to parse form", 400)
		return
	}
	module := r.FormValue("module")
	recordID := r.FormValue("record_id")
	if module == "" || recordID == "" {
		jsonErr(w, "module and record_id required", 400)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		jsonErr(w, "File required", 400)
		return
	}
	defer file.Close()

	// Build filename
	ts := time.Now().UnixMilli()
	safeName := strings.ReplaceAll(header.Filename, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	filename := fmt.Sprintf("%s-%s-%d-%s", module, recordID, ts, safeName)

	// Save file
	outPath := filepath.Join("uploads", filename)
	out, err := os.Create(outPath)
	if err != nil {
		jsonErr(w, "Failed to save file", 500)
		return
	}
	defer out.Close()
	written, err := io.Copy(out, file)
	if err != nil {
		jsonErr(w, "Failed to write file", 500)
		return
	}

	// Get uploader
	uploadedBy := "unknown"
	if u := getCurrentUser(r); u != nil {
		uploadedBy = u.Username
	}

	mimeType := header.Header.Get("Content-Type")
	result, err := db.Exec(`INSERT INTO attachments (module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		module, recordID, filename, header.Filename, written, mimeType, uploadedBy)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	id, _ := result.LastInsertId()
	w.WriteHeader(201)
	jsonResp(w, Attachment{
		ID:           int(id),
		Module:       module,
		RecordID:     recordID,
		Filename:     filename,
		OriginalName: header.Filename,
		SizeBytes:    written,
		MimeType:     mimeType,
		UploadedBy:   uploadedBy,
	})
}

func handleListAttachments(w http.ResponseWriter, r *http.Request) {
	module := r.URL.Query().Get("module")
	recordID := r.URL.Query().Get("record_id")
	if module == "" || recordID == "" {
		jsonErr(w, "module and record_id required", 400)
		return
	}
	rows, err := db.Query(`SELECT id, module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by, created_at
		FROM attachments WHERE module = ? AND record_id = ? ORDER BY created_at DESC`, module, recordID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var atts []Attachment
	for rows.Next() {
		var a Attachment
		rows.Scan(&a.ID, &a.Module, &a.RecordID, &a.Filename, &a.OriginalName, &a.SizeBytes, &a.MimeType, &a.UploadedBy, &a.CreatedAt)
		atts = append(atts, a)
	}
	if atts == nil {
		atts = []Attachment{}
	}
	jsonResp(w, atts)
}

func handleServeFile(w http.ResponseWriter, r *http.Request, filename string) {
	path := filepath.Join("uploads", filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Disposition", "inline; filename=\""+filename+"\"")
	http.ServeFile(w, r, path)
}

func handleDeleteAttachment(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonErr(w, "Invalid ID", 400)
		return
	}
	var filename string
	err = db.QueryRow("SELECT filename FROM attachments WHERE id = ?", id).Scan(&filename)
	if err != nil {
		jsonErr(w, "Attachment not found", 404)
		return
	}
	db.Exec("DELETE FROM attachments WHERE id = ?", id)
	os.Remove(filepath.Join("uploads", filename))
	jsonResp(w, map[string]string{"status": "deleted"})
}
