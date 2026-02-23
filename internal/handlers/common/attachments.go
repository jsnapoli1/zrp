package common

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

// Attachment represents a file attachment.
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

// UploadAttachment handles file upload.
func (h *Handler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	maxUploadSize := int64(100 << 20)
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+1024)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			http.Error(w, `{"error":"File too large. Maximum size is 100MB."}`, 413)
		} else {
			http.Error(w, `{"error":"Failed to parse form"}`, 400)
		}
		return
	}

	module := r.FormValue("module")
	recordID := r.FormValue("record_id")
	if module == "" || recordID == "" {
		http.Error(w, `{"error":"module and record_id required"}`, 400)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"File required"}`, 400)
		return
	}
	defer file.Close()

	fileSize := header.Size

	ve := &ValidationErrors{}
	h.ValidateFileUpload(ve, header.Filename, fileSize, header.Header.Get("Content-Type"))
	if ve.HasErrors() {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, ve.Error()), 400)
		return
	}

	safeName := h.SanitizeFilename(header.Filename)

	ts := time.Now().UnixMilli()
	filename := fmt.Sprintf("%s-%s-%d-%s", module, recordID, ts, safeName)

	outPath := filepath.Join("uploads", filename)
	out, err := os.Create(outPath)
	if err != nil {
		http.Error(w, `{"error":"Failed to save file"}`, 500)
		return
	}
	defer out.Close()
	written, err := io.Copy(out, file)
	if err != nil {
		http.Error(w, `{"error":"Failed to write file"}`, 500)
		return
	}

	uploadedBy := "unknown"
	if u := h.GetCurrentUser(r); u != nil {
		uploadedBy = u.Username
	}

	mimeType := header.Header.Get("Content-Type")
	result, err := h.DB.Exec(`INSERT INTO attachments (module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		module, recordID, filename, header.Filename, written, mimeType, uploadedBy)
	if err != nil {
		http.Error(w, `{"error":"Failed to save attachment. Please try again."}`, 500)
		return
	}

	id, _ := result.LastInsertId()
	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id":%d,"module":"%s","record_id":"%s","filename":"%s","original_name":"%s","size_bytes":%d,"mime_type":"%s","uploaded_by":"%s"}`,
		id, module, recordID, filename, header.Filename, written, mimeType, uploadedBy)
}

// ListAttachments lists attachments for a module/record.
func (h *Handler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	module := r.URL.Query().Get("module")
	recordID := r.URL.Query().Get("record_id")
	if module == "" || recordID == "" {
		http.Error(w, `{"error":"module and record_id required"}`, 400)
		return
	}
	rows, err := h.DB.Query(`SELECT id, module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by, created_at
		FROM attachments WHERE module = ? AND record_id = ? ORDER BY created_at DESC`, module, recordID)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch attachments. Please try again."}`, 500)
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
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "[")
	for i, a := range atts {
		if i > 0 {
			fmt.Fprintf(w, ",")
		}
		fmt.Fprintf(w, `{"id":%d,"module":"%s","record_id":"%s","filename":"%s","original_name":"%s","size_bytes":%d,"mime_type":"%s","uploaded_by":"%s","created_at":"%s"}`,
			a.ID, a.Module, a.RecordID, a.Filename, a.OriginalName, a.SizeBytes, a.MimeType, a.UploadedBy, a.CreatedAt)
	}
	fmt.Fprintf(w, "]")
}

// ServeFile serves a file by filename.
func (h *Handler) ServeFile(w http.ResponseWriter, r *http.Request, filename string) {
	path := filepath.Join("uploads", filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Disposition", "inline; filename=\""+filename+"\"")
	http.ServeFile(w, r, path)
}

// DeleteAttachment deletes an attachment.
func (h *Handler) DeleteAttachment(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"Invalid ID"}`, 400)
		return
	}
	var filename string
	err = h.DB.QueryRow("SELECT filename FROM attachments WHERE id = ?", id).Scan(&filename)
	if err != nil {
		http.Error(w, `{"error":"Attachment not found"}`, 404)
		return
	}

	res, err := h.DB.Exec("DELETE FROM attachments WHERE id = ?", id)
	if err != nil {
		http.Error(w, `{"error":"Failed to delete attachment. Please try again."}`, 500)
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		http.Error(w, `{"error":"Attachment not found"}`, 404)
		return
	}

	filePath := filepath.Join("uploads", filename)
	if err := os.Remove(filePath); err != nil {
		h.LogAudit(h.DB, h.GetUsername(r), "delete_file_failed", "attachment", idStr, "Failed to remove file: "+filename)
	}

	h.LogAudit(h.DB, h.GetUsername(r), "deleted", "attachment", idStr, "Deleted attachment: "+filename)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"deleted"}`)
}

// DownloadAttachment downloads an attachment by ID.
func (h *Handler) DownloadAttachment(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"Invalid ID"}`, 400)
		return
	}
	var filename, originalName, mimeType string
	err = h.DB.QueryRow("SELECT filename, original_name, mime_type FROM attachments WHERE id = ?", id).Scan(&filename, &originalName, &mimeType)
	if err != nil {
		http.Error(w, `{"error":"Attachment not found"}`, 404)
		return
	}
	filePath := filepath.Join("uploads", filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, `{"error":"File not found on disk"}`, 404)
		return
	}
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", originalName))
	http.ServeFile(w, r, filePath)
}
