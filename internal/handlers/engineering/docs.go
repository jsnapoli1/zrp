package engineering

import (
	"net/http"
	"time"

	"zrp/internal/audit"
	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
)

// ListDocs handles GET /api/documents.
func (h *Handler) ListDocs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT d.id, d.title, COALESCE(d.category,''), COALESCE(d.ipn,''), d.revision, d.status,
		COALESCE(d.content,''), COALESCE(d.file_path,''), d.created_by, d.created_at, d.updated_at,
		COALESCE(a.cnt, 0)
		FROM documents d
		LEFT JOIN (SELECT record_id, COUNT(*) as cnt FROM attachments WHERE module='document' GROUP BY record_id) a ON a.record_id = d.id
		ORDER BY d.created_at DESC`)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	type DocWithCount struct {
		models.Document
		AttachmentCount int `json:"attachment_count"`
	}
	var items []DocWithCount
	for rows.Next() {
		var d DocWithCount
		rows.Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.AttachmentCount)
		items = append(items, d)
	}
	if items == nil {
		items = []DocWithCount{}
	}
	response.JSON(w, items)
}

// GetDoc handles GET /api/documents/:id.
func (h *Handler) GetDoc(w http.ResponseWriter, r *http.Request, id string) {
	var d models.Document
	err := h.DB.QueryRow("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents WHERE id=?", id).
		Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	// Fetch attachments
	attRows, err := h.DB.Query("SELECT id, module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by, created_at FROM attachments WHERE module='document' AND record_id=? ORDER BY created_at DESC", id)
	var atts []models.Attachment
	if err == nil {
		defer attRows.Close()
		for attRows.Next() {
			var a models.Attachment
			attRows.Scan(&a.ID, &a.Module, &a.RecordID, &a.Filename, &a.OriginalName, &a.SizeBytes, &a.MimeType, &a.UploadedBy, &a.CreatedAt)
			atts = append(atts, a)
		}
	}
	if atts == nil {
		atts = []models.Attachment{}
	}

	type DocWithAttachments struct {
		models.Document
		Attachments []models.Attachment `json:"attachments"`
	}
	response.JSON(w, DocWithAttachments{Document: d, Attachments: atts})
}

// CreateDoc handles POST /api/documents.
func (h *Handler) CreateDoc(w http.ResponseWriter, r *http.Request) {
	var d models.Document
	if err := response.DecodeBody(r, &d); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "title", d.Title)
	if d.Status != "" {
		validation.ValidateEnum(ve, "status", d.Status, validation.ValidDocStatuses)
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	d.ID = h.NextIDFunc("DOC", "documents", 3)
	if d.Revision == "" {
		d.Revision = "A"
	}
	if d.Status == "" {
		d.Status = "draft"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := h.DB.Exec("INSERT INTO documents (id,title,category,ipn,revision,status,content,file_path,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
		d.ID, d.Title, d.Category, d.IPN, d.Revision, d.Status, d.Content, d.FilePath, "engineer", now, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	d.CreatedAt = now
	d.UpdatedAt = now
	d.CreatedBy = "engineer"
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "created", "document", d.ID, "Created "+d.ID+": "+d.Title)
	response.JSON(w, d)
}

// UpdateDoc handles PUT /api/documents/:id.
func (h *Handler) UpdateDoc(w http.ResponseWriter, r *http.Request, id string) {
	var d models.Document
	if err := response.DecodeBody(r, &d); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	// Get current document to preserve status if not provided
	var currentStatus string
	if err := h.DB.QueryRow("SELECT status FROM documents WHERE id=?", id).Scan(&currentStatus); err != nil {
		response.Err(w, "document not found", 404)
		return
	}

	// Snapshot current state before updating
	username := audit.GetUsername(h.DB, r)
	h.SnapshotDocumentVersion(id, "Before update", username, nil)

	// Use current status if not provided in update
	if d.Status == "" {
		d.Status = currentStatus
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := h.DB.Exec("UPDATE documents SET title=?,category=?,ipn=?,revision=?,status=?,content=?,file_path=?,updated_at=? WHERE id=?",
		d.Title, d.Category, d.IPN, d.Revision, d.Status, d.Content, d.FilePath, now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), "updated", "document", id, "Updated "+id+": "+d.Title)
	h.GetDoc(w, r, id)
}

// ApproveDoc handles POST /api/documents/:id/approve.
func (h *Handler) ApproveDoc(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := h.DB.Exec("UPDATE documents SET status='approved',updated_at=? WHERE id=?", now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	audit.LogAudit(h.DB, h.Hub, audit.GetUsername(h.DB, r), "approved", "document", id, "Approved document "+id)
	h.GetDoc(w, r, id)
}
