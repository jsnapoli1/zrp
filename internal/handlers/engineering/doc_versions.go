package engineering

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"zrp/internal/audit"
	"zrp/internal/models"
	"zrp/internal/response"
)

// NextRevision increments a revision letter: A->B, B->C, ..., Z->AA.
func NextRevision(rev string) string {
	if rev == "" {
		return "A"
	}
	runes := []rune(rev)
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] < 'Z' {
			runes[i]++
			return string(runes)
		}
		runes[i] = 'A'
	}
	return "A" + string(runes)
}

// SnapshotDocumentVersion saves current document state as a version entry.
func (h *Handler) SnapshotDocumentVersion(docID, changeSummary, createdBy string, ecoID *string) error {
	var d models.Document
	err := h.DB.QueryRow("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents WHERE id=?", docID).
		Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return err
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = h.DB.Exec("INSERT INTO document_versions (document_id, revision, content, file_path, change_summary, status, created_by, created_at, eco_id) VALUES (?,?,?,?,?,?,?,?,?)",
		docID, d.Revision, d.Content, d.FilePath, changeSummary, d.Status, createdBy, now, ecoID)
	return err
}

// ListDocVersions handles GET /api/documents/:id/versions.
func (h *Handler) ListDocVersions(w http.ResponseWriter, r *http.Request, docID string) {
	rows, err := h.DB.Query("SELECT id, document_id, revision, content, file_path, change_summary, status, created_by, created_at, COALESCE(eco_id,'') FROM document_versions WHERE document_id=? ORDER BY id DESC", docID)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var versions []models.DocumentVersion
	for rows.Next() {
		var v models.DocumentVersion
		var ecoID string
		rows.Scan(&v.ID, &v.DocumentID, &v.Revision, &v.Content, &v.FilePath, &v.ChangeSummary, &v.Status, &v.CreatedBy, &v.CreatedAt, &ecoID)
		if ecoID != "" {
			v.ECOID = &ecoID
		}
		versions = append(versions, v)
	}
	if versions == nil {
		versions = []models.DocumentVersion{}
	}
	response.JSON(w, versions)
}

// GetDocVersion handles GET /api/documents/:id/versions/:revision.
func (h *Handler) GetDocVersion(w http.ResponseWriter, r *http.Request, docID, revision string) {
	var v models.DocumentVersion
	var ecoID string
	err := h.DB.QueryRow("SELECT id, document_id, revision, COALESCE(content,''), COALESCE(file_path,''), COALESCE(change_summary,''), COALESCE(status,''), COALESCE(created_by,''), created_at, COALESCE(eco_id,'') FROM document_versions WHERE document_id=? AND revision=? ORDER BY id DESC LIMIT 1", docID, revision).
		Scan(&v.ID, &v.DocumentID, &v.Revision, &v.Content, &v.FilePath, &v.ChangeSummary, &v.Status, &v.CreatedBy, &v.CreatedAt, &ecoID)
	if err != nil {
		response.Err(w, "version not found", 404)
		return
	}
	if ecoID != "" {
		v.ECOID = &ecoID
	}
	response.JSON(w, v)
}

// DiffLine represents a single line in a diff output.
type DiffLine struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// DocDiff handles GET /api/documents/:id/diff.
func (h *Handler) DocDiff(w http.ResponseWriter, r *http.Request, docID string) {
	fromRev := r.URL.Query().Get("from")
	toRev := r.URL.Query().Get("to")
	if fromRev == "" || toRev == "" {
		response.Err(w, "from and to query params required", 400)
		return
	}

	var fromContent, toContent string
	err := h.DB.QueryRow("SELECT content FROM document_versions WHERE document_id=? AND revision=? ORDER BY id DESC LIMIT 1", docID, fromRev).Scan(&fromContent)
	if err != nil {
		response.Err(w, "from revision not found", 404)
		return
	}
	err = h.DB.QueryRow("SELECT content FROM document_versions WHERE document_id=? AND revision=? ORDER BY id DESC LIMIT 1", docID, toRev).Scan(&toContent)
	if err != nil {
		response.Err(w, "to revision not found", 404)
		return
	}

	// Simple line-based diff
	fromLines := strings.Split(fromContent, "\n")
	toLines := strings.Split(toContent, "\n")

	var diff []DiffLine
	// Simple LCS-based diff
	diff = ComputeDiff(fromLines, toLines)

	response.JSON(w, map[string]interface{}{
		"from":  fromRev,
		"to":    toRev,
		"lines": diff,
	})
}

// ComputeDiff computes a simple LCS-based diff between two slices of lines.
func ComputeDiff(from, to []string) []DiffLine {
	// Build LCS table
	m, n := len(from), len(to)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if from[i-1] == to[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else if lcs[i-1][j] >= lcs[i][j-1] {
				lcs[i][j] = lcs[i-1][j]
			} else {
				lcs[i][j] = lcs[i][j-1]
			}
		}
	}

	// Backtrack
	var result []DiffLine
	i, j := m, n
	var stack []DiffLine
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && from[i-1] == to[j-1] {
			stack = append(stack, DiffLine{"same", from[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			stack = append(stack, DiffLine{"added", to[j-1]})
			j--
		} else {
			stack = append(stack, DiffLine{"removed", from[i-1]})
			i--
		}
	}
	for k := len(stack) - 1; k >= 0; k-- {
		result = append(result, stack[k])
	}
	return result
}

// ReleaseDoc handles POST /api/documents/:id/release.
func (h *Handler) ReleaseDoc(w http.ResponseWriter, r *http.Request, docID string) {
	// Snapshot current state, then bump revision and set status to released
	var d models.Document
	err := h.DB.QueryRow("SELECT id,title,COALESCE(category,''),COALESCE(ipn,''),revision,status,COALESCE(content,''),COALESCE(file_path,''),created_by,created_at,updated_at FROM documents WHERE id=?", docID).
		Scan(&d.ID, &d.Title, &d.Category, &d.IPN, &d.Revision, &d.Status, &d.Content, &d.FilePath, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}

	username := audit.GetUsername(h.DB, r)
	// Snapshot current draft
	if err := h.SnapshotDocumentVersion(docID, "Released as revision "+d.Revision, username, nil); err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	// Update document status to released
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = h.DB.Exec("UPDATE documents SET status='released', updated_at=? WHERE id=?", now, docID)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	audit.LogAudit(h.DB, h.Hub, username, "released", "document", docID, fmt.Sprintf("Released %s at revision %s", docID, d.Revision))
	h.GetDoc(w, r, docID)
}

// RevertDoc handles POST /api/documents/:id/revert/:revision.
func (h *Handler) RevertDoc(w http.ResponseWriter, r *http.Request, docID, revision string) {
	var v models.DocumentVersion
	var ecoID string
	err := h.DB.QueryRow("SELECT id, document_id, revision, COALESCE(content,''), COALESCE(file_path,''), COALESCE(change_summary,''), COALESCE(status,''), COALESCE(created_by,''), created_at, COALESCE(eco_id,'') FROM document_versions WHERE document_id=? AND revision=? ORDER BY id DESC LIMIT 1", docID, revision).
		Scan(&v.ID, &v.DocumentID, &v.Revision, &v.Content, &v.FilePath, &v.ChangeSummary, &v.Status, &v.CreatedBy, &v.CreatedAt, &ecoID)
	if err != nil {
		response.Err(w, "version not found", 404)
		return
	}

	username := audit.GetUsername(h.DB, r)
	// Snapshot current state before reverting
	if err := h.SnapshotDocumentVersion(docID, "Before revert to revision "+revision, username, nil); err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = h.DB.Exec("UPDATE documents SET content=?, file_path=?, revision=?, status='draft', updated_at=? WHERE id=?",
		v.Content, v.FilePath, v.Revision, now, docID)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}

	audit.LogAudit(h.DB, h.Hub, username, "reverted", "document", docID, fmt.Sprintf("Reverted %s to revision %s", docID, revision))
	h.GetDoc(w, r, docID)
}

// BumpDocRevisionsForECO bumps revisions on all documents referenced by an ECO.
func (h *Handler) BumpDocRevisionsForECO(ecoID string, username string) error {
	// Find documents linked to this ECO via affected_ipns matching doc IPN
	var affectedIPNs string
	err := h.DB.QueryRow("SELECT COALESCE(affected_ipns,'') FROM ecos WHERE id=?", ecoID).Scan(&affectedIPNs)
	if err != nil || affectedIPNs == "" {
		return nil // no affected IPNs
	}

	// Parse affected_ipns JSON array
	affectedIPNs = strings.TrimSpace(affectedIPNs)
	if !strings.HasPrefix(affectedIPNs, "[") {
		return nil
	}

	// Simple parse: extract quoted strings
	var ipns []string
	for _, part := range strings.Split(affectedIPNs, "\"") {
		p := strings.TrimSpace(part)
		if p != "" && p != "[" && p != "]" && p != "," && !strings.HasPrefix(p, "[") && !strings.HasPrefix(p, "]") && !strings.HasPrefix(p, ",") {
			ipns = append(ipns, p)
		}
	}

	// Find documents matching those IPNs
	for _, ipn := range ipns {
		rows, err := h.DB.Query("SELECT id, revision FROM documents WHERE ipn=?", ipn)
		if err != nil {
			continue
		}
		type docRef struct {
			id  string
			rev string
		}
		var docs []docRef
		for rows.Next() {
			var dr docRef
			rows.Scan(&dr.id, &dr.rev)
			docs = append(docs, dr)
		}
		rows.Close()

		for _, dr := range docs {
			ecoIDStr := ecoID
			h.SnapshotDocumentVersion(dr.id, "Before ECO "+ecoID+" revision bump", username, &ecoIDStr)
			newRev := NextRevision(dr.rev)
			now := time.Now().Format("2006-01-02 15:04:05")
			h.DB.Exec("UPDATE documents SET revision=?, status='draft', updated_at=? WHERE id=?", newRev, now, dr.id)
			// Create version entry for the new revision too
			h.SnapshotDocumentVersion(dr.id, "Revision bumped by ECO "+ecoID, username, &ecoIDStr)
		}
	}
	return nil
}
