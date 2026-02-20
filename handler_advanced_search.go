package main

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

// handleAdvancedSearch handles advanced search requests with filters
func handleAdvancedSearch(w http.ResponseWriter, r *http.Request) {
	var query SearchQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set defaults
	if query.Limit == 0 {
		query.Limit = 50
	}
	if query.SortOrder == "" {
		query.SortOrder = "asc"
	}

	// Parse search operators if search text contains them
	if query.SearchText != "" {
		parsedFilters := ParseSearchOperators(query.SearchText)
		query.Filters = append(query.Filters, parsedFilters...)
		// Remove operator patterns from search text for multi-field search
		query.SearchText = strings.TrimSpace(strings.Split(query.SearchText, ":")[0])
	}

	result, err := executeAdvancedSearch(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log search to history
	go logSearchHistory(getUserFromRequest(r), query)

	jsonResp(w, result)
}

// executeAdvancedSearch performs the search based on entity type
func executeAdvancedSearch(query SearchQuery) (*SearchResult, error) {
	switch query.EntityType {
	case "parts":
		return searchParts(query)
	case "workorders":
		return searchWorkOrders(query)
	case "ecos":
		return searchECOs(query)
	case "inventory":
		return searchInventory(query)
	case "ncrs":
		return searchNCRs(query)
	case "devices":
		return searchDevices(query)
	case "pos":
		return searchPOs(query)
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", query.EntityType)
	}
}

// searchParts searches parts with advanced filters
func searchParts(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "parts", query.SearchText)
	if err != nil {
		return nil, err
	}

	// Build ORDER BY clause
	orderBy := ""
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	} else {
		orderBy = " ORDER BY ipn ASC"
	}

	// Count total
	countSQL := "SELECT COUNT(*) FROM parts_view" + whereClause
	var total int
	err = db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get data
	dataSQL := "SELECT ipn, category, fields FROM parts_view" + whereClause + orderBy + 
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)
	
	rows, err := db.Query(dataSQL, args...)
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
		
		part := map[string]interface{}{
			"ipn":      ipn,
			"category": category,
			"fields":   fields,
		}
		parts = append(parts, part)
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit

	return &SearchResult{
		Data:       parts,
		Total:      total,
		Page:       page,
		PageSize:   query.Limit,
		TotalPages: totalPages,
	}, nil
}

// searchWorkOrders searches work orders with advanced filters
func searchWorkOrders(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "workorders", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := ""
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	} else {
		orderBy = " ORDER BY created_at DESC"
	}

	// Count total
	countSQL := "SELECT COUNT(*) FROM work_orders" + whereClause
	var total int
	err = db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get data
	dataSQL := `SELECT id, assembly_ipn, qty, status, "priority", notes, ` +
		`created_at, started_at, completed_at, due_date, qty_good, qty_scrap ` +
		`FROM work_orders` + whereClause + orderBy + 
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)
	
	rows, err := db.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wos []map[string]interface{}
	for rows.Next() {
		var id, assemblyIPN, status, priority string
		var notes, dueDate sql.NullString
		var qty, qtyGood, qtyScrap int
		var createdAt sql.NullTime
		var startedAt, completedAt sql.NullTime
		
		if err := rows.Scan(&id, &assemblyIPN, &qty, &status, &priority, &notes,
			&createdAt, &startedAt, &completedAt, &dueDate, &qtyGood, &qtyScrap); err != nil {
			continue
		}
		
		wo := map[string]interface{}{
			"id":           id,
			"assembly_ipn": assemblyIPN,
			"qty":          qty,
			"status":       status,
			"priority":     priority,
			"notes":        notes.String,
			"created_at":   createdAt.Time,
			"started_at":   startedAt.Time,
			"completed_at": completedAt.Time,
			"due_date":     dueDate.String,
			"qty_good":     qtyGood,
			"qty_scrap":    qtyScrap,
		}
		wos = append(wos, wo)
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit

	return &SearchResult{
		Data:       wos,
		Total:      total,
		Page:       page,
		PageSize:   query.Limit,
		TotalPages: totalPages,
	}, nil
}

// searchECOs searches ECOs with advanced filters
func searchECOs(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "ecos", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := ""
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	} else {
		orderBy = " ORDER BY created_at DESC"
	}

	// Count total
	countSQL := "SELECT COUNT(*) FROM ecos" + whereClause
	var total int
	err = db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get data
	dataSQL := `SELECT id, title, description, status, priority, affected_ipns,
		created_by, created_at, updated_at, approved_at, approved_by
		FROM ecos` + whereClause + orderBy + 
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)
	
	rows, err := db.Query(dataSQL, args...)
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
		
		eco := map[string]interface{}{
			"id":            id,
			"title":         title,
			"description":   description,
			"status":        status,
			"priority":      priority,
			"affected_ipns": affectedIPNs,
			"created_by":    createdBy,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
			"approved_at":   approvedAt,
			"approved_by":   approvedBy,
		}
		ecos = append(ecos, eco)
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit

	return &SearchResult{
		Data:       ecos,
		Total:      total,
		Page:       page,
		PageSize:   query.Limit,
		TotalPages: totalPages,
	}, nil
}

// searchInventory searches inventory with advanced filters
func searchInventory(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "inventory", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := ""
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	} else {
		orderBy = " ORDER BY ipn ASC"
	}

	// Count total
	countSQL := "SELECT COUNT(*) FROM inventory" + whereClause
	var total int
	err = db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get data
	dataSQL := `SELECT ipn, qty_on_hand, qty_reserved, location, reorder_point, 
		reorder_qty, description, mpn, updated_at
		FROM inventory` + whereClause + orderBy + 
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)
	
	rows, err := db.Query(dataSQL, args...)
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
		
		item := map[string]interface{}{
			"ipn":           ipn,
			"qty_on_hand":   qtyOnHand,
			"qty_reserved":  qtyReserved,
			"location":      location,
			"reorder_point": reorderPoint,
			"reorder_qty":   reorderQty,
			"description":   description,
			"mpn":           mpn,
			"updated_at":    updatedAt,
		}
		items = append(items, item)
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit

	return &SearchResult{
		Data:       items,
		Total:      total,
		Page:       page,
		PageSize:   query.Limit,
		TotalPages: totalPages,
	}, nil
}

// searchNCRs searches NCRs with advanced filters
func searchNCRs(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "ncrs", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := ""
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	} else {
		orderBy = " ORDER BY created_at DESC"
	}

	// Count total
	countSQL := "SELECT COUNT(*) FROM ncrs" + whereClause
	var total int
	err = db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get data
	dataSQL := `SELECT id, title, description, ipn, serial_number, defect_type,
		severity, status, root_cause, corrective_action, created_at, resolved_at, created_by
		FROM ncrs` + whereClause + orderBy + 
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)
	
	rows, err := db.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ncrs []map[string]interface{}
	for rows.Next() {
		var id, title, description, ipn, serialNum, defectType, severity, status string
		var rootCause, correctiveAction, createdBy string
		var createdAt, resolvedAt *time.Time
		
		if err := rows.Scan(&id, &title, &description, &ipn, &serialNum, &defectType,
			&severity, &status, &rootCause, &correctiveAction, &createdAt, &resolvedAt, &createdBy); err != nil {
			continue
		}
		
		ncr := map[string]interface{}{
			"id":                 id,
			"title":              title,
			"description":        description,
			"ipn":                ipn,
			"serial_number":      serialNum,
			"defect_type":        defectType,
			"severity":           severity,
			"status":             status,
			"root_cause":         rootCause,
			"corrective_action":  correctiveAction,
			"created_at":         createdAt,
			"resolved_at":        resolvedAt,
			"created_by":         createdBy,
		}
		ncrs = append(ncrs, ncr)
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit

	return &SearchResult{
		Data:       ncrs,
		Total:      total,
		Page:       page,
		PageSize:   query.Limit,
		TotalPages: totalPages,
	}, nil
}

// searchDevices searches devices with advanced filters
func searchDevices(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "devices", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := ""
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	} else {
		orderBy = " ORDER BY created_at DESC"
	}

	// Count total
	countSQL := "SELECT COUNT(*) FROM devices" + whereClause
	var total int
	err = db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get data
	dataSQL := `SELECT serial_number, ipn, firmware_version, customer, location,
		status, install_date, last_seen, notes, created_at
		FROM devices` + whereClause + orderBy + 
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)
	
	rows, err := db.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []map[string]interface{}
	for rows.Next() {
		var serialNum, ipn, fwVersion, customer, location, status, installDate, notes string
		var lastSeen, createdAt *time.Time
		
		if err := rows.Scan(&serialNum, &ipn, &fwVersion, &customer, &location,
			&status, &installDate, &lastSeen, &notes, &createdAt); err != nil {
			continue
		}
		
		device := map[string]interface{}{
			"serial_number":    serialNum,
			"ipn":              ipn,
			"firmware_version": fwVersion,
			"customer":         customer,
			"location":         location,
			"status":           status,
			"install_date":     installDate,
			"last_seen":        lastSeen,
			"notes":            notes,
			"created_at":       createdAt,
		}
		devices = append(devices, device)
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit

	return &SearchResult{
		Data:       devices,
		Total:      total,
		Page:       page,
		PageSize:   query.Limit,
		TotalPages: totalPages,
	}, nil
}

// searchPOs searches purchase orders with advanced filters
func searchPOs(query SearchQuery) (*SearchResult, error) {
	whereClause, args, err := BuildSearchSQL(query.Filters, "pos", query.SearchText)
	if err != nil {
		return nil, err
	}

	orderBy := ""
	if query.SortBy != "" {
		orderBy = fmt.Sprintf(" ORDER BY %s %s", query.SortBy, strings.ToUpper(query.SortOrder))
	} else {
		orderBy = " ORDER BY created_at DESC"
	}

	// Count total
	countSQL := "SELECT COUNT(*) FROM purchase_orders" + whereClause
	var total int
	err = db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get data
	dataSQL := `SELECT id, vendor_id, status, notes, created_at, expected_date, 
		received_at, created_by, total
		FROM purchase_orders` + whereClause + orderBy + 
		fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)
	
	rows, err := db.Query(dataSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pos []map[string]interface{}
	for rows.Next() {
		var id, vendorID, status, notes, expectedDate, createdBy string
		var total float64
		var createdAt, receivedAt *time.Time
		
		if err := rows.Scan(&id, &vendorID, &status, &notes, &createdAt, &expectedDate,
			&receivedAt, &createdBy, &total); err != nil {
			continue
		}
		
		po := map[string]interface{}{
			"id":            id,
			"vendor_id":     vendorID,
			"status":        status,
			"notes":         notes,
			"created_at":    createdAt,
			"expected_date": expectedDate,
			"received_at":   receivedAt,
			"created_by":    createdBy,
			"total":         total,
		}
		pos = append(pos, po)
	}

	page := query.Offset/query.Limit + 1
	totalPages := (total + query.Limit - 1) / query.Limit

	return &SearchResult{
		Data:       pos,
		Total:      total,
		Page:       page,
		PageSize:   query.Limit,
		TotalPages: totalPages,
	}, nil
}

// Saved Search Handlers

func handleSaveSavedSearch(w http.ResponseWriter, r *http.Request) {
	var savedSearch SavedSearch
	if err := json.NewDecoder(r.Body).Decode(&savedSearch); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	savedSearch.ID = uuid.New().String()
	savedSearch.CreatedBy = getUserFromRequest(r)
	savedSearch.CreatedAt = time.Now()

	filtersJSON, _ := json.Marshal(savedSearch.Filters)

	_, err := db.Exec(`INSERT INTO saved_searches 
		(id, name, entity_type, filters, sort_by, sort_order, created_by, is_public)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		savedSearch.ID, savedSearch.Name, savedSearch.EntityType, string(filtersJSON),
		savedSearch.SortBy, savedSearch.SortOrder, savedSearch.CreatedBy, savedSearch.IsPublic)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp(w, savedSearch)
}

func handleGetSavedSearches(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entity_type")
	user := getUserFromRequest(r)

	query := `SELECT id, name, entity_type, filters, sort_by, sort_order, created_by, created_at, is_public
		FROM saved_searches
		WHERE (created_by = ? OR is_public = 1)`
	
	args := []interface{}{user}
	if entityType != "" {
		query += " AND entity_type = ?"
		args = append(args, entityType)
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
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

	jsonResp(w, searches)
}

func handleDeleteSavedSearch(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	user := getUserFromRequest(r)

	// Only allow delete if user owns it
	result, err := db.Exec("DELETE FROM saved_searches WHERE id = ? AND created_by = ?", id, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Not found or permission denied", http.StatusNotFound)
		return
	}

	jsonResp(w, map[string]interface{}{"success": true})
}

func handleGetQuickFilters(w http.ResponseWriter, r *http.Request) {
	entityType := r.URL.Query().Get("entity_type")
	filters := GetQuickFilters(entityType)
	jsonResp(w, filters)
}

func handleGetSearchHistory(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r)
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

	rows, err := db.Query(query, args...)
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

	jsonResp(w, history)
}

func logSearchHistory(user string, query SearchQuery) {
	filtersJSON, _ := json.Marshal(query.Filters)
	db.Exec("INSERT INTO search_history (user_id, entity_type, search_text, filters) VALUES (?, ?, ?, ?)",
		user, query.EntityType, query.SearchText, string(filtersJSON))
}

func getUserFromRequest(r *http.Request) string {
	// Extract from session or token
	// For now, return a default user
	user := r.Header.Get("X-User-ID")
	if user == "" {
		user = "system"
	}
	return user
}
