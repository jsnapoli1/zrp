package common

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SearchFilter represents a single search filter.
type SearchFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	AndOr    string      `json:"andOr"`
}

// SavedSearch represents a saved search configuration.
type SavedSearch struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	EntityType string         `json:"entity_type"`
	Filters    []SearchFilter `json:"filters"`
	SortBy     string         `json:"sort_by"`
	SortOrder  string         `json:"sort_order"`
	CreatedBy  string         `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
	IsPublic   bool           `json:"is_public"`
}

// SearchQuery represents a complete search request.
type SearchQuery struct {
	EntityType string         `json:"entity_type"`
	Filters    []SearchFilter `json:"filters"`
	SearchText string         `json:"search_text"`
	SortBy     string         `json:"sort_by"`
	SortOrder  string         `json:"sort_order"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
}

// SearchResult contains search results with metadata.
type SearchResult struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// QuickFilter represents a preset filter.
type QuickFilter struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	EntityType string         `json:"entity_type"`
	Filters    []SearchFilter `json:"filters"`
}

// BuildSearchSQL constructs SQL WHERE clause from filters.
func BuildSearchSQL(filters []SearchFilter, entityType string, searchText string) (string, []interface{}, error) {
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if searchText != "" {
		textFilters := buildTextSearchFilters(entityType, searchText)
		if len(textFilters) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(textFilters, " OR ")+")")
			searchPattern := "%" + searchText + "%"
			for i := 0; i < len(textFilters); i++ {
				args = append(args, searchPattern)
				argIndex++
			}
		}
	}

	for i, filter := range filters {
		clause, filterArgs, err := buildFilterClause(filter, &argIndex)
		if err != nil {
			return "", nil, err
		}

		if clause != "" {
			if i > 0 {
				connector := " AND "
				if strings.ToUpper(filters[i-1].AndOr) == "OR" {
					connector = " OR "
				}
				whereClauses = append(whereClauses, connector+clause)
			} else {
				whereClauses = append(whereClauses, clause)
			}
			args = append(args, filterArgs...)
		}
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauses, "")
	}

	return whereClause, args, nil
}

func buildTextSearchFilters(entityType string, searchText string) []string {
	var fields []string

	switch entityType {
	case "parts":
		fields = []string{
			"ipn LIKE ?",
			"LOWER(json_extract(fields, '$.description')) LIKE LOWER(?)",
			"LOWER(json_extract(fields, '$.vendor')) LIKE LOWER(?)",
			"LOWER(json_extract(fields, '$.mpn')) LIKE LOWER(?)",
		}
	case "workorders":
		fields = []string{"id LIKE ?", "assembly_ipn LIKE ?", "notes LIKE ?"}
	case "ecos":
		fields = []string{"id LIKE ?", "title LIKE ?", "description LIKE ?", "affected_ipns LIKE ?"}
	case "inventory":
		fields = []string{"ipn LIKE ?", "description LIKE ?", "mpn LIKE ?", "location LIKE ?"}
	case "ncrs":
		fields = []string{"id LIKE ?", "title LIKE ?", "description LIKE ?", "ipn LIKE ?", "serial_number LIKE ?"}
	case "devices":
		fields = []string{"serial_number LIKE ?", "ipn LIKE ?", "customer LIKE ?", "location LIKE ?"}
	case "pos":
		fields = []string{"id LIKE ?", "vendor_id LIKE ?", "notes LIKE ?"}
	}

	return fields
}

func quoteField(field string) string {
	return `"` + field + `"`
}

func buildFilterClause(filter SearchFilter, argIndex *int) (string, []interface{}, error) {
	var args []interface{}
	field := quoteField(filter.Field)
	operator := strings.ToLower(filter.Operator)

	valueStr := fmt.Sprintf("%v", filter.Value)

	switch operator {
	case "eq", "=":
		args = append(args, filter.Value)
		*argIndex++
		return fmt.Sprintf("%s = ?", field), args, nil

	case "ne", "!=":
		args = append(args, filter.Value)
		*argIndex++
		return fmt.Sprintf("%s != ?", field), args, nil

	case "contains", "like":
		pattern := valueStr
		if !strings.Contains(pattern, "%") && !strings.Contains(pattern, "*") {
			pattern = "%" + pattern + "%"
		}
		pattern = strings.ReplaceAll(pattern, "*", "%")
		args = append(args, pattern)
		*argIndex++
		return fmt.Sprintf("%s LIKE ?", field), args, nil

	case "startswith":
		pattern := valueStr
		if !strings.HasSuffix(pattern, "%") {
			pattern = pattern + "%"
		}
		args = append(args, pattern)
		*argIndex++
		return fmt.Sprintf("%s LIKE ?", field), args, nil

	case "endswith":
		pattern := valueStr
		if !strings.HasPrefix(pattern, "%") {
			pattern = "%" + pattern
		}
		args = append(args, pattern)
		*argIndex++
		return fmt.Sprintf("%s LIKE ?", field), args, nil

	case "gt", ">":
		args = append(args, filter.Value)
		*argIndex++
		return fmt.Sprintf("%s > ?", field), args, nil

	case "lt", "<":
		args = append(args, filter.Value)
		*argIndex++
		return fmt.Sprintf("%s < ?", field), args, nil

	case "gte", ">=":
		args = append(args, filter.Value)
		*argIndex++
		return fmt.Sprintf("%s >= ?", field), args, nil

	case "lte", "<=":
		args = append(args, filter.Value)
		*argIndex++
		return fmt.Sprintf("%s <= ?", field), args, nil

	case "in":
		values, ok := filter.Value.([]interface{})
		if !ok {
			strVal := fmt.Sprintf("%v", filter.Value)
			parts := strings.Split(strVal, ",")
			values = make([]interface{}, len(parts))
			for i, p := range parts {
				values[i] = strings.TrimSpace(p)
			}
		}
		placeholders := make([]string, len(values))
		for i, v := range values {
			placeholders[i] = "?"
			args = append(args, v)
			*argIndex++
		}
		return fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ",")), args, nil

	case "between":
		values, ok := filter.Value.([]interface{})
		if !ok || len(values) != 2 {
			return "", nil, fmt.Errorf("between operator requires array with 2 values")
		}
		args = append(args, values[0], values[1])
		*argIndex += 2
		return fmt.Sprintf("%s BETWEEN ? AND ?", field), args, nil

	case "isnull":
		return fmt.Sprintf("%s IS NULL", field), args, nil

	case "isnotnull":
		return fmt.Sprintf("%s IS NOT NULL", field), args, nil

	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// GetQuickFilters returns preset quick filters for an entity type.
func GetQuickFilters(entityType string) []QuickFilter {
	filters := make([]QuickFilter, 0)

	switch entityType {
	case "parts":
		filters = append(filters, QuickFilter{ID: "active-parts", Name: "Active Parts", EntityType: "parts",
			Filters: []SearchFilter{{Field: "status", Operator: "eq", Value: "active", AndOr: "AND"}}})
		filters = append(filters, QuickFilter{ID: "obsolete-parts", Name: "Obsolete Parts", EntityType: "parts",
			Filters: []SearchFilter{{Field: "status", Operator: "eq", Value: "obsolete", AndOr: "AND"}}})

	case "workorders":
		filters = append(filters, QuickFilter{ID: "open-wos", Name: "Open Work Orders", EntityType: "workorders",
			Filters: []SearchFilter{{Field: "status", Operator: "in", Value: []interface{}{"open", "in_progress"}, AndOr: "AND"}}})
		filters = append(filters, QuickFilter{ID: "high-priority-wos", Name: "High Priority", EntityType: "workorders",
			Filters: []SearchFilter{
				{Field: "priority", Operator: "eq", Value: "high", AndOr: "AND"},
				{Field: "status", Operator: "ne", Value: "completed", AndOr: "AND"},
			}})
		filters = append(filters, QuickFilter{ID: "overdue-wos", Name: "Overdue", EntityType: "workorders",
			Filters: []SearchFilter{
				{Field: "due_date", Operator: "lt", Value: time.Now().Format("2006-01-02"), AndOr: "AND"},
				{Field: "status", Operator: "ne", Value: "completed", AndOr: "AND"},
			}})

	case "inventory":
		filters = append(filters, QuickFilter{ID: "low-stock", Name: "Low Stock", EntityType: "inventory",
			Filters: []SearchFilter{{Field: "qty_on_hand", Operator: "lte", Value: "reorder_point", AndOr: "AND"}}})
		filters = append(filters, QuickFilter{ID: "out-of-stock", Name: "Out of Stock", EntityType: "inventory",
			Filters: []SearchFilter{{Field: "qty_on_hand", Operator: "lte", Value: 0, AndOr: "AND"}}})

	case "ecos":
		filters = append(filters, QuickFilter{ID: "pending-ecos", Name: "Pending ECOs", EntityType: "ecos",
			Filters: []SearchFilter{{Field: "status", Operator: "in", Value: []interface{}{"draft", "pending"}, AndOr: "AND"}}})
		filters = append(filters, QuickFilter{ID: "approved-ecos", Name: "Approved ECOs", EntityType: "ecos",
			Filters: []SearchFilter{{Field: "status", Operator: "eq", Value: "approved", AndOr: "AND"}}})
		filters = append(filters, QuickFilter{ID: "high-priority-ecos", Name: "High Priority", EntityType: "ecos",
			Filters: []SearchFilter{
				{Field: "priority", Operator: "eq", Value: "high", AndOr: "AND"},
				{Field: "status", Operator: "ne", Value: "rejected", AndOr: "AND"},
			}})

	case "ncrs":
		filters = append(filters, QuickFilter{ID: "open-ncrs", Name: "Open NCRs", EntityType: "ncrs",
			Filters: []SearchFilter{{Field: "status", Operator: "eq", Value: "open", AndOr: "AND"}}})
		filters = append(filters, QuickFilter{ID: "critical-ncrs", Name: "Critical NCRs", EntityType: "ncrs",
			Filters: []SearchFilter{
				{Field: "severity", Operator: "in", Value: []interface{}{"critical", "major"}, AndOr: "AND"},
				{Field: "status", Operator: "ne", Value: "closed", AndOr: "AND"},
			}})
	}

	return filters
}

// InitSearchTables creates database tables for saved searches.
func InitSearchTables(database *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS saved_searches (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			filters TEXT NOT NULL,
			sort_by TEXT DEFAULT '',
			sort_order TEXT DEFAULT 'asc',
			created_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_public BOOLEAN DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_saved_searches_entity ON saved_searches(entity_type);
		CREATE INDEX IF NOT EXISTS idx_saved_searches_user ON saved_searches(created_by);
		CREATE INDEX IF NOT EXISTS idx_saved_searches_public ON saved_searches(is_public);
		CREATE TABLE IF NOT EXISTS search_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			search_text TEXT,
			filters TEXT,
			searched_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_search_history_user ON search_history(user_id);
		CREATE INDEX IF NOT EXISTS idx_search_history_entity ON search_history(entity_type);
	`
	_, err := database.Exec(schema)
	if err != nil {
		return err
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_parts_ipn ON parts(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_parts_category ON parts(category)",
		"CREATE INDEX IF NOT EXISTS idx_workorders_status ON work_orders(status)",
		"CREATE INDEX IF NOT EXISTS idx_workorders_priority ON work_orders(priority)",
		"CREATE INDEX IF NOT EXISTS idx_workorders_assembly_ipn ON work_orders(assembly_ipn)",
		"CREATE INDEX IF NOT EXISTS idx_workorders_due_date ON work_orders(due_date)",
		"CREATE INDEX IF NOT EXISTS idx_workorders_created_at ON work_orders(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_ecos_status ON ecos(status)",
		"CREATE INDEX IF NOT EXISTS idx_ecos_priority ON ecos(priority)",
		"CREATE INDEX IF NOT EXISTS idx_ecos_created_at ON ecos(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_inventory_ipn ON inventory(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_inventory_location ON inventory(location)",
		"CREATE INDEX IF NOT EXISTS idx_inventory_qty_on_hand ON inventory(qty_on_hand)",
		"CREATE INDEX IF NOT EXISTS idx_inventory_reorder_point ON inventory(reorder_point)",
		"CREATE INDEX IF NOT EXISTS idx_ncrs_status ON ncrs(status)",
		"CREATE INDEX IF NOT EXISTS idx_ncrs_severity ON ncrs(severity)",
		"CREATE INDEX IF NOT EXISTS idx_ncrs_ipn ON ncrs(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_ncrs_serial_number ON ncrs(serial_number)",
		"CREATE INDEX IF NOT EXISTS idx_ncrs_created_at ON ncrs(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_devices_serial_number ON devices(serial_number)",
		"CREATE INDEX IF NOT EXISTS idx_devices_ipn ON devices(ipn)",
		"CREATE INDEX IF NOT EXISTS idx_devices_customer ON devices(customer)",
		"CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status)",
		"CREATE INDEX IF NOT EXISTS idx_pos_vendor_id ON purchase_orders(vendor_id)",
		"CREATE INDEX IF NOT EXISTS idx_pos_status ON purchase_orders(status)",
		"CREATE INDEX IF NOT EXISTS idx_pos_created_at ON purchase_orders(created_at)",
	}

	for _, indexSQL := range indexes {
		if _, err := database.Exec(indexSQL); err != nil {
			fmt.Printf("Warning: Failed to create index: %v\n", err)
		}
	}

	return nil
}

// ParseSearchOperators parses search text for advanced operators.
func ParseSearchOperators(searchText string) []SearchFilter {
	filters := make([]SearchFilter, 0)
	operatorPattern := regexp.MustCompile(`(\w+)([:><=!]+)([^\s]+)`)
	matches := operatorPattern.FindAllStringSubmatch(searchText, -1)

	for _, match := range matches {
		if len(match) == 4 {
			field := match[1]
			operator := match[2]
			value := match[3]

			op := "eq"
			switch operator {
			case ":":
				op = "contains"
			case ">":
				op = "gt"
			case "<":
				op = "lt"
			case ">=":
				op = "gte"
			case "<=":
				op = "lte"
			case "!=":
				op = "ne"
			}

			if strings.Contains(value, "*") {
				op = "contains"
			}

			if numVal, err := strconv.ParseFloat(value, 64); err == nil {
				filters = append(filters, SearchFilter{Field: field, Operator: op, Value: numVal, AndOr: "AND"})
			} else {
				filters = append(filters, SearchFilter{Field: field, Operator: op, Value: value, AndOr: "AND"})
			}
		}
	}

	return filters
}
