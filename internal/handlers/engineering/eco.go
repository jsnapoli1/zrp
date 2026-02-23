package engineering

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"zrp/internal/audit"
	"zrp/internal/database"
	"zrp/internal/models"
	"zrp/internal/response"
	"zrp/internal/validation"
)

// ListECOs handles GET /api/ecos.
func (h *Handler) ListECOs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	query := "SELECT id,title,COALESCE(description,''),COALESCE(status,''),COALESCE(priority,''),COALESCE(affected_ipns,''),COALESCE(created_by,''),COALESCE(created_at,''),COALESCE(updated_at,''),approved_at,approved_by,COALESCE(ncr_id,'') FROM ecos"
	var args []interface{}
	if status != "" {
		query += " WHERE status=?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	rows, err := h.DB.Query(query, args...)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.ECO
	for rows.Next() {
		var e models.ECO
		var aa, ab sql.NullString
		rows.Scan(&e.ID, &e.Title, &e.Description, &e.Status, &e.Priority, &e.AffectedIPNs, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &aa, &ab, &e.NcrID)
		e.ApprovedAt = database.SP(aa)
		e.ApprovedBy = database.SP(ab)
		items = append(items, e)
	}
	if items == nil {
		items = []models.ECO{}
	}
	response.JSON(w, items)
}

// GetECO handles GET /api/ecos/:id.
func (h *Handler) GetECO(w http.ResponseWriter, r *http.Request, id string) {
	var e models.ECO
	var aa, ab sql.NullString
	err := h.DB.QueryRow("SELECT id,title,COALESCE(description,''),COALESCE(status,''),COALESCE(priority,''),COALESCE(affected_ipns,''),COALESCE(created_by,''),COALESCE(created_at,''),COALESCE(updated_at,''),approved_at,approved_by,COALESCE(ncr_id,'') FROM ecos WHERE id=?", id).
		Scan(&e.ID, &e.Title, &e.Description, &e.Status, &e.Priority, &e.AffectedIPNs, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &aa, &ab, &e.NcrID)
	if err != nil {
		response.Err(w, "not found", 404)
		return
	}
	e.ApprovedAt = database.SP(aa)
	e.ApprovedBy = database.SP(ab)

	// Enrich with affected parts details
	var affectedParts []map[string]string
	var ipns []string
	// Try JSON array first, then comma-separated
	if strings.HasPrefix(strings.TrimSpace(e.AffectedIPNs), "[") {
		json.Unmarshal([]byte(e.AffectedIPNs), &ipns)
	} else if e.AffectedIPNs != "" {
		for _, s := range strings.Split(e.AffectedIPNs, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				ipns = append(ipns, s)
			}
		}
	}
	for _, ipn := range ipns {
		if h.GetPartByIPN != nil {
			fields, err := h.GetPartByIPN(h.PartsDir, ipn)
			if err == nil {
				part := make(map[string]string)
				part["ipn"] = ipn
				for k, v := range fields {
					part[strings.ToLower(k)] = v
				}
				affectedParts = append(affectedParts, part)
			} else {
				affectedParts = append(affectedParts, map[string]string{"ipn": ipn, "error": "not found"})
			}
		} else {
			affectedParts = append(affectedParts, map[string]string{"ipn": ipn, "error": "not found"})
		}
	}
	if affectedParts == nil {
		affectedParts = []map[string]string{}
	}

	// Build enriched response
	resp := map[string]interface{}{
		"id": e.ID, "title": e.Title, "description": e.Description,
		"status": e.Status, "priority": e.Priority, "affected_ipns": e.AffectedIPNs,
		"affected_parts": affectedParts, "created_by": e.CreatedBy,
		"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
		"approved_at": e.ApprovedAt, "approved_by": e.ApprovedBy,
		"ncr_id": e.NcrID,
	}
	response.JSON(w, resp)
}

// CreateECO handles POST /api/ecos.
func (h *Handler) CreateECO(w http.ResponseWriter, r *http.Request) {
	var e models.ECO
	if err := response.DecodeBody(r, &e); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	// Validation
	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "title", e.Title)
	validation.ValidateMaxLength(ve, "title", e.Title, 255)
	validation.ValidateMaxLength(ve, "description", e.Description, 1000)
	validation.ValidateMaxLength(ve, "affected_ipns", e.AffectedIPNs, 1000)
	if e.Status != "" {
		validation.ValidateEnum(ve, "status", e.Status, validation.ValidECOStatuses)
	}
	if e.Priority != "" {
		validation.ValidateEnum(ve, "priority", e.Priority, validation.ValidECOPriorities)
	}
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	e.ID = h.NextIDFunc("ECO", "ecos", 3)
	if e.Status == "" {
		e.Status = "draft"
	}
	if e.Priority == "" {
		e.Priority = "normal"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := h.DB.Exec("INSERT INTO ecos (id,title,description,status,priority,affected_ipns,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)",
		e.ID, e.Title, e.Description, e.Status, e.Priority, e.AffectedIPNs, "engineer", now, now)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	e.CreatedAt = now
	e.UpdatedAt = now
	e.CreatedBy = "engineer"
	h.EnsureInitialRevision(e.ID, "engineer", now)
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "created", "eco", e.ID, "Created "+e.ID+": "+e.Title)
	h.RecordChangeJSON(username, "ecos", e.ID, "create", nil, e)
	response.JSON(w, e)
}

// UpdateECO handles PUT /api/ecos/:id.
func (h *Handler) UpdateECO(w http.ResponseWriter, r *http.Request, id string) {
	// Verify exists
	var exists int
	h.DB.QueryRow("SELECT COUNT(*) FROM ecos WHERE id=?", id).Scan(&exists)
	if exists == 0 {
		response.Err(w, "not found", 404)
		return
	}

	oldSnap, _ := h.GetECOSnapshot(id)
	var e models.ECO
	if err := response.DecodeBody(r, &e); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}

	ve := &validation.ValidationErrors{}
	validation.RequireField(ve, "title", e.Title)
	validation.ValidateMaxLength(ve, "title", e.Title, 255)
	validation.ValidateMaxLength(ve, "description", e.Description, 1000)
	validation.ValidateMaxLength(ve, "affected_ipns", e.AffectedIPNs, 1000)
	validation.ValidateEnum(ve, "status", e.Status, validation.ValidECOStatuses)
	validation.ValidateEnum(ve, "priority", e.Priority, validation.ValidECOPriorities)
	if ve.HasErrors() {
		response.Err(w, ve.Error(), 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := h.DB.Exec("UPDATE ecos SET title=?,description=?,status=?,priority=?,affected_ipns=?,updated_at=? WHERE id=?",
		e.Title, e.Description, e.Status, e.Priority, e.AffectedIPNs, now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	username := audit.GetUsername(h.DB, r)
	audit.LogAudit(h.DB, h.Hub, username, "updated", "eco", id, "Updated "+id+": "+e.Title)
	newSnap, _ := h.GetECOSnapshot(id)
	h.RecordChangeJSON(username, "ecos", id, "update", oldSnap, newSnap)
	h.GetECO(w, r, id)
}

// ApproveECO handles POST /api/ecos/:id/approve.
func (h *Handler) ApproveECO(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	user := audit.GetUsername(h.DB, r)
	_, err := h.DB.Exec("UPDATE ecos SET status='approved',approved_at=?,approved_by=?,updated_at=? WHERE id=?", now, user, now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	// Record approval in latest revision
	h.updateRevisionApproval(id, user, now)
	audit.LogAudit(h.DB, h.Hub, user, "approved", "eco", id, "Approved "+id)
	if h.EmailOnECOApproved != nil {
		go h.EmailOnECOApproved(id)
	}
	h.GetECO(w, r, id)
}

// ImplementECO handles POST /api/ecos/:id/implement.
func (h *Handler) ImplementECO(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	user := audit.GetUsername(h.DB, r)
	_, err := h.DB.Exec("UPDATE ecos SET status='implemented',updated_at=? WHERE id=?", now, id)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	// Record implementation in latest revision
	h.updateRevisionImplementation(id, user, now)
	// Apply any linked part changes
	if h.ApplyPartChangesForECO != nil {
		h.ApplyPartChangesForECO(id)
	}
	audit.LogAudit(h.DB, h.Hub, user, "implemented", "eco", id, "Implemented "+id)
	if h.EmailOnECOImplemented != nil {
		go h.EmailOnECOImplemented(id)
	}
	h.GetECO(w, r, id)
}

// --- ECO Revision Handlers ---

func (h *Handler) nextRevisionLetter(ecoID string) string {
	var last string
	err := h.DB.QueryRow("SELECT revision FROM eco_revisions WHERE eco_id=? ORDER BY id DESC LIMIT 1", ecoID).Scan(&last)
	if err != nil || last == "" {
		return "A"
	}
	return string(rune(last[0] + 1))
}

func (h *Handler) updateRevisionApproval(ecoID, user, now string) {
	h.DB.Exec("UPDATE eco_revisions SET approved_by=?, approved_at=?, status='approved' WHERE eco_id=? AND id=(SELECT MAX(id) FROM eco_revisions WHERE eco_id=?)", user, now, ecoID, ecoID)
}

func (h *Handler) updateRevisionImplementation(ecoID, user, now string) {
	h.DB.Exec("UPDATE eco_revisions SET implemented_by=?, implemented_at=?, status='implemented' WHERE eco_id=? AND id=(SELECT MAX(id) FROM eco_revisions WHERE eco_id=?)", user, now, ecoID, ecoID)
}

// EnsureInitialRevision creates the initial "A" revision for an ECO if none exists.
func (h *Handler) EnsureInitialRevision(ecoID, user, now string) {
	var count int
	h.DB.QueryRow("SELECT COUNT(*) FROM eco_revisions WHERE eco_id=?", ecoID).Scan(&count)
	if count == 0 {
		h.DB.Exec("INSERT INTO eco_revisions (eco_id, revision, status, changes_summary, created_by, created_at) VALUES (?,?,?,?,?,?)",
			ecoID, "A", "created", "Initial revision", user, now)
	}
}

// ListECORevisions handles GET /api/ecos/:id/revisions.
func (h *Handler) ListECORevisions(w http.ResponseWriter, r *http.Request, ecoID string) {
	rows, err := h.DB.Query("SELECT id,eco_id,revision,status,COALESCE(changes_summary,''),COALESCE(created_by,''),created_at,approved_by,approved_at,implemented_by,implemented_at,effectivity_date,COALESCE(notes,'') FROM eco_revisions WHERE eco_id=? ORDER BY id ASC", ecoID)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []models.ECORevision
	for rows.Next() {
		var rev models.ECORevision
		var ab, aa, ib, ia, ed sql.NullString
		rows.Scan(&rev.ID, &rev.ECOID, &rev.Revision, &rev.Status, &rev.ChangesSummary, &rev.CreatedBy, &rev.CreatedAt, &ab, &aa, &ib, &ia, &ed, &rev.Notes)
		rev.ApprovedBy = database.SP(ab)
		rev.ApprovedAt = database.SP(aa)
		rev.ImplementedBy = database.SP(ib)
		rev.ImplementedAt = database.SP(ia)
		rev.EffectivityDate = database.SP(ed)
		items = append(items, rev)
	}
	if items == nil {
		items = []models.ECORevision{}
	}
	response.JSON(w, items)
}

// CreateECORevision handles POST /api/ecos/:id/revisions.
func (h *Handler) CreateECORevision(w http.ResponseWriter, r *http.Request, ecoID string) {
	var body struct {
		ChangesSummary  string `json:"changes_summary"`
		EffectivityDate string `json:"effectivity_date"`
		Notes           string `json:"notes"`
	}
	if err := response.DecodeBody(r, &body); err != nil {
		response.Err(w, "invalid body", 400)
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	user := audit.GetUsername(h.DB, r)
	rev := h.nextRevisionLetter(ecoID)
	var ed *string
	if body.EffectivityDate != "" {
		ed = &body.EffectivityDate
	}
	res, err := h.DB.Exec("INSERT INTO eco_revisions (eco_id, revision, status, changes_summary, created_by, created_at, effectivity_date, notes) VALUES (?,?,?,?,?,?,?,?)",
		ecoID, rev, "created", body.ChangesSummary, user, now, ed, body.Notes)
	if err != nil {
		response.Err(w, err.Error(), 500)
		return
	}
	id, _ := res.LastInsertId()
	audit.LogAudit(h.DB, h.Hub, user, "created", "eco_revision", ecoID, "Created revision "+rev+" for "+ecoID)
	response.JSON(w, map[string]interface{}{"id": id, "eco_id": ecoID, "revision": rev, "status": "created", "changes_summary": body.ChangesSummary, "created_by": user, "created_at": now, "effectivity_date": ed, "notes": body.Notes})
}

// GetECORevision handles GET /api/ecos/:id/revisions/:rev.
func (h *Handler) GetECORevision(w http.ResponseWriter, r *http.Request, ecoID, revLetter string) {
	var rev models.ECORevision
	var ab, aa, ib, ia, ed sql.NullString
	err := h.DB.QueryRow("SELECT id,eco_id,revision,status,COALESCE(changes_summary,''),COALESCE(created_by,''),created_at,approved_by,approved_at,implemented_by,implemented_at,effectivity_date,COALESCE(notes,'') FROM eco_revisions WHERE eco_id=? AND revision=?", ecoID, revLetter).
		Scan(&rev.ID, &rev.ECOID, &rev.Revision, &rev.Status, &rev.ChangesSummary, &rev.CreatedBy, &rev.CreatedAt, &ab, &aa, &ib, &ia, &ed, &rev.Notes)
	if err != nil {
		response.Err(w, "revision not found", 404)
		return
	}
	rev.ApprovedBy = database.SP(ab)
	rev.ApprovedAt = database.SP(aa)
	rev.ImplementedBy = database.SP(ib)
	rev.ImplementedAt = database.SP(ia)
	rev.EffectivityDate = database.SP(ed)
	response.JSON(w, rev)
}
