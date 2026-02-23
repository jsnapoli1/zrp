package common

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AdvancedSearch handles advanced search requests with filters.
func (h *Handler) AdvancedSearch(w http.ResponseWriter, r *http.Request) {
	var query SearchQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if query.Limit == 0 {
		query.Limit = 50
	}
	if query.SortOrder == "" {
		query.SortOrder = "asc"
	}

	if query.SearchText != "" {
		parsedFilters := ParseSearchOperators(query.SearchText)
		query.Filters = append(query.Filters, parsedFilters...)
		query.SearchText = strings.TrimSpace(strings.Split(query.SearchText, ":")[0])
	}

	result, err := h.executeAdvancedSearch(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.DB != nil {
		filtersJSON, _ := json.Marshal(query.Filters)
		h.DB.Exec("INSERT INTO search_history (user_id, entity_type, search_text, filters) VALUES (?, ?, ?, ?)",
			h.getUserFromRequest(r), query.EntityType, query.SearchText, string(filtersJSON))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) executeAdvancedSearch(query SearchQuery) (*SearchResult, error) {
	switch query.EntityType {
	case "parts":
		return h.searchParts(query)
	case "workorders":
		return h.searchWorkOrders(query)
	case "ecos":
		return h.searchECOs(query)
	case "inventory":
		return h.searchInventory(query)
	case "ncrs":
		return h.searchNCRs(query)
	case "devices":
		return h.searchDevices(query)
	case "pos":
		return h.searchPOs(query)
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", query.EntityType)
	}
}

func (h *Handler) searchParts(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "parts", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := " ORDER BY ipn ASC"
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	}

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM parts_view"+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	dataSQL := "SELECT ipn, category, fields FROM parts_view" + whereClause + orderBy +
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)

	rows, err := h.DB.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []map[string]interface{}
	for rows.Next() {
		var ipn, category, fieldsJSON string
		if err := rows.Scan(&ipn, &category, &fieldsJSON); err != nil {
			continue
		}
		var fields map[string]interface{}
		json.Unmarshal([]byte(fieldsJSON), &fields)
		parts = append(parts, map[string]interface{}{"ipn": ipn, "category": category, "fields": fields})
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit
	return &SearchResult{Data: parts, Total: total, Page: page, PageSize: query.Limit, TotalPages: totalPages}, nil
}

func (h *Handler) searchWorkOrders(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "workorders", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := " ORDER BY created_at DESC"
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	}

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM work_orders"+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	dataSQL := `SELECT id, assembly_ipn, qty, status, "priority", notes, ` +
		`created_at, started_at, completed_at, due_date, qty_good, qty_scrap ` +
		`FROM work_orders` + whereClause + orderBy +
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)

	rows, err := h.DB.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wos []map[string]interface{}
	for rows.Next() {
		var id, assemblyIPN, status, priority string
		var notes, dueDate sql.NullString
		var qty, qtyGood, qtyScrap int
		var createdAt, startedAt, completedAt sql.NullTime

		if err := rows.Scan(&id, &assemblyIPN, &qty, &status, &priority, &notes,
			&createdAt, &startedAt, &completedAt, &dueDate, &qtyGood, &qtyScrap); err != nil {
			continue
		}

		wos = append(wos, map[string]interface{}{
			"id": id, "assembly_ipn": assemblyIPN, "qty": qty, "status": status,
			"priority": priority, "notes": notes.String, "created_at": createdAt.Time,
			"started_at": startedAt.Time, "completed_at": completedAt.Time,
			"due_date": dueDate.String, "qty_good": qtyGood, "qty_scrap": qtyScrap,
		})
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit
	return &SearchResult{Data: wos, Total: total, Page: page, PageSize: query.Limit, TotalPages: totalPages}, nil
}

func (h *Handler) searchECOs(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "ecos", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := " ORDER BY created_at DESC"
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	}

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM ecos"+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	dataSQL := `SELECT id, title, description, status, priority, affected_ipns,
		created_by, created_at, updated_at, approved_at, approved_by
		FROM ecos` + whereClause + orderBy +
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)

	rows, err := h.DB.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ecos []map[string]interface{}
	for rows.Next() {
		var id, title, description, status, priority, affectedIPNs, createdBy, approvedBy string
		var createdAt, updatedAt, approvedAt *time.Time

		if err := rows.Scan(&id, &title, &description, &status, &priority, &affectedIPNs,
			&createdBy, &createdAt, &updatedAt, &approvedAt, &approvedBy); err != nil {
			continue
		}

		ecos = append(ecos, map[string]interface{}{
			"id": id, "title": title, "description": description, "status": status,
			"priority": priority, "affected_ipns": affectedIPNs, "created_by": createdBy,
			"created_at": createdAt, "updated_at": updatedAt, "approved_at": approvedAt,
			"approved_by": approvedBy,
		})
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit
	return &SearchResult{Data: ecos, Total: total, Page: page, PageSize: query.Limit, TotalPages: totalPages}, nil
}

func (h *Handler) searchInventory(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "inventory", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := " ORDER BY ipn ASC"
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	}

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM inventory"+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	dataSQL := `SELECT ipn, qty_on_hand, qty_reserved, location, reorder_point,
		reorder_qty, description, mpn, updated_at
		FROM inventory` + whereClause + orderBy +
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)

	rows, err := h.DB.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var ipn, location, description, mpn string
		var qtyOnHand, qtyReserved, reorderPoint, reorderQty float64
		var updatedAt *time.Time

		if err := rows.Scan(&ipn, &qtyOnHand, &qtyReserved, &location, &reorderPoint,
			&reorderQty, &description, &mpn, &updatedAt); err != nil {
			continue
		}

		items = append(items, map[string]interface{}{
			"ipn": ipn, "qty_on_hand": qtyOnHand, "qty_reserved": qtyReserved,
			"location": location, "reorder_point": reorderPoint, "reorder_qty": reorderQty,
			"description": description, "mpn": mpn, "updated_at": updatedAt,
		})
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit
	return &SearchResult{Data: items, Total: total, Page: page, PageSize: query.Limit, TotalPages: totalPages}, nil
}

func (h *Handler) searchNCRs(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "ncrs", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := " ORDER BY created_at DESC"
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	}

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM ncrs"+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	dataSQL := `SELECT id, title, description, ipn, serial_number, defect_type,
		severity, status, root_cause, corrective_action, created_at, resolved_at, created_by
		FROM ncrs` + whereClause + orderBy +
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)

	rows, err := h.DB.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ncrs []map[string]interface{}
	for rows.Next() {
		var id, title, severity, status string
		var description, ipn, serialNum, defectType sql.NullString
		var rootCause, correctiveAction, createdBy sql.NullString
		var createdAt, resolvedAt sql.NullTime

		if err := rows.Scan(&id, &title, &description, &ipn, &serialNum, &defectType,
			&severity, &status, &rootCause, &correctiveAction, &createdAt, &resolvedAt, &createdBy); err != nil {
			continue
		}

		ncrs = append(ncrs, map[string]interface{}{
			"id": id, "title": title, "description": description.String,
			"ipn": ipn.String, "serial_number": serialNum.String,
			"defect_type": defectType.String, "severity": severity, "status": status,
			"root_cause": rootCause.String, "corrective_action": correctiveAction.String,
			"created_at": createdAt.Time, "resolved_at": resolvedAt.Time, "created_by": createdBy.String,
		})
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit
	return &SearchResult{Data: ncrs, Total: total, Page: page, PageSize: query.Limit, TotalPages: totalPages}, nil
}

func (h *Handler) searchDevices(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "devices", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := " ORDER BY created_at DESC"
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	}

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM devices"+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	dataSQL := `SELECT serial_number, ipn, firmware_version, customer, location,
		status, install_date, last_seen, notes, created_at
		FROM devices` + whereClause + orderBy +
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)

	rows, err := h.DB.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []map[string]interface{}
	for rows.Next() {
		var serialNum, ipn, status string
		var fwVersion, customer, location, installDate, notes sql.NullString
		var lastSeen, createdAt sql.NullTime

		if err := rows.Scan(&serialNum, &ipn, &fwVersion, &customer, &location,
			&status, &installDate, &lastSeen, &notes, &createdAt); err != nil {
			continue
		}

		devices = append(devices, map[string]interface{}{
			"serial_number": serialNum, "ipn": ipn, "firmware_version": fwVersion.String,
			"customer": customer.String, "location": location.String, "status": status,
			"install_date": installDate.String, "last_seen": lastSeen.Time,
			"notes": notes.String, "created_at": createdAt.Time,
		})
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit
	return &SearchResult{Data: devices, Total: total, Page: page, PageSize: query.Limit, TotalPages: totalPages}, nil
}

func (h *Handler) searchPOs(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "pos", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := " ORDER BY created_at DESC"
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	}

	var total int
	err = h.DB.QueryRow("SELECT COUNT(*) FROM purchase_orders"+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	dataSQL := `SELECT id, vendor_id, status, notes, created_at, expected_date,
		received_at, created_by, total
		FROM purchase_orders` + whereClause + orderBy +
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)

	rows, err := h.DB.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pos []map[string]interface{}
	for rows.Next() {
		var id, status string
		var vendorID, notes, expectedDate, createdBy sql.NullString
		var total sql.NullFloat64
		var createdAt, receivedAt sql.NullTime

		if err := rows.Scan(&id, &vendorID, &status, &notes, &createdAt, &expectedDate,
			&receivedAt, &createdBy, &total); err != nil {
			continue
		}

		pos = append(pos, map[string]interface{}{
			"id": id, "vendor_id": vendorID.String, "status": status,
			"notes": notes.String, "created_at": createdAt.Time,
			"expected_date": expectedDate.String, "received_at": receivedAt.Time,
			"created_by": createdBy.String, "total": total.Float64,
		})
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit
	return &SearchResult{Data: pos, Total: total, Page: page, PageSize: query.Limit, TotalPages: totalPages}, nil
}

// SaveSavedSearch saves a search configuration.
func (h *Handler) SaveSavedSearch(w http.ResponseWriter, r *http.Request) {
	var savedSearch SavedSearch
	if err := json.NewDecoder(r.Body).Decode(&savedSearch); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	savedSearch.ID = uuid.New().String()
	savedSearch.CreatedBy = h.getUserFromRequest(r)
	savedSearch.CreatedAt = time.Now()

	filtersJSON, _ := json.Marshal(savedSearch.Filters)

	_, err := h.DB.Exec(`INSERT INTO saved_searches
		(id, name, entity_type, filters, sort_by, sort_order, created_by, is_public)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		savedSearch.ID, savedSearch.Name, savedSearch.EntityType, string(filtersJSON),
		savedSearch.SortBy, savedSearch.SortOrder, savedSearch.CreatedBy, savedSearch.IsPublic)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(savedSearch)
}

// GetSavedSearches returns saved searches for the current user.
func (h *Handler) GetSavedSearches(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entity_type")
	user := h.getUserFromRequest(r)

	query := `SELECT id, name, entity_type, filters, sort_by, sort_order, created_by, created_at, is_public
		FROM saved_searches WHERE (created_by = ? OR is_public = 1)`
	args := []interface{}{user}
	if entityType != "" {
		query += " AND entity_type = ?"
		args = append(args, entityType)
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var searches []SavedSearch
	for rows.Next() {
		var s SavedSearch
		var filtersJSON string
		var sortBy, sortOrder, createdBy *string
		var isPublic int
		if err := rows.Scan(&s.ID, &s.Name, &s.EntityType, &filtersJSON, &sortBy,
			&sortOrder, &createdBy, &s.CreatedAt, &isPublic); err != nil {
			continue
		}
		if sortBy != nil {
			s.SortBy = *sortBy
		}
		if sortOrder != nil {
			s.SortOrder = *sortOrder
		}
		if createdBy != nil {
			s.CreatedBy = *createdBy
		}
		s.IsPublic = isPublic == 1
		json.Unmarshal([]byte(filtersJSON), &s.Filters)
		searches = append(searches, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searches)
}

// DeleteSavedSearch deletes a saved search.
func (h *Handler) DeleteSavedSearch(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	user := h.getUserFromRequest(r)

	result, err := h.DB.Exec("DELETE FROM saved_searches WHERE id = ? AND created_by = ?", id, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Not found or permission denied", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// GetQuickFiltersHandler returns quick filter options for an entity type.
func (h *Handler) GetQuickFiltersHandler(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entity_type")
	filters := GetQuickFilters(entityType)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filters)
}

// GetSearchHistory returns recent search history.
func (h *Handler) GetSearchHistory(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromRequest(r)
	entityType := r.URL.Query().Get("entity_type")
	limit := 10

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}

	query := "SELECT entity_type, search_text, filters, searched_at FROM search_history WHERE user_id = ?"
	args := []interface{}{user}

	if entityType != "" {
		query += " AND entity_type = ?"
		args = append(args, entityType)
	}

	query += " ORDER BY searched_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type historyEntry struct {
		EntityType string    `json:"entity_type"`
		SearchText string    `json:"search_text"`
		Filters    string    `json:"filters"`
		SearchedAt time.Time `json:"searched_at"`
	}

	var history []historyEntry
	for rows.Next() {
		var entry historyEntry
		if err := rows.Scan(&entry.EntityType, &entry.SearchText, &entry.Filters, &entry.SearchedAt); err != nil {
			continue
		}
		history = append(history, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *Handler) getUserFromRequest(r *http.Request) string {
	user := r.Header.Get("X-User-ID")
	if user == "" {
		user = "system"
	}
	return user
}
