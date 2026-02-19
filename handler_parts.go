package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// gitplm CSV format: first row is headers, IPN is derived from filename/category
// Category dirs contain CSV files with parts data

func loadPartsFromDir() (map[string][]Part, map[string][]string, map[string]string, error) {
	categories := make(map[string][]Part)
	schemas := make(map[string][]string)
	titles := make(map[string]string)

	if partsDir == "" {
		return categories, schemas, titles, nil
	}

	entries, err := os.ReadDir(partsDir)
	if err != nil {
		return categories, schemas, titles, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			catDir := filepath.Join(partsDir, entry.Name())
			csvFiles, _ := filepath.Glob(filepath.Join(catDir, "*.csv"))
			catName := strings.ToLower(entry.Name())
			for _, csvFile := range csvFiles {
				parts, cols, title, err := readCSV(csvFile, catName)
				if err != nil {
					continue
				}
				categories[catName] = append(categories[catName], parts...)
				if len(cols) > len(schemas[catName]) {
					schemas[catName] = cols
				}
				if title != "" {
					titles[catName] = title
				}
			}
		} else if strings.HasSuffix(entry.Name(), ".csv") {
			catName := strings.TrimSuffix(entry.Name(), ".csv")
			catName = strings.ToLower(catName)
			parts, cols, title, err := readCSV(filepath.Join(partsDir, entry.Name()), catName)
			if err != nil {
				continue
			}
			categories[catName] = append(categories[catName], parts...)
			schemas[catName] = cols
			if title != "" {
				titles[catName] = title
			}
		}
	}
	return categories, schemas, titles, nil
}

func readCSV(path string, category string) ([]Part, []string, string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, "", err
	}

	// Extract title from comment line if present: # TITLE: <title>
	title := ""
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# TITLE:") {
		title = strings.TrimSpace(strings.TrimPrefix(lines[0], "# TITLE:"))
		// Remove the comment line for CSV parsing
		content = []byte(strings.Join(lines[1:], "\n"))
	}

	r := csv.NewReader(strings.NewReader(string(content)))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, "", err
	}
	if len(records) < 1 {
		return nil, nil, "", fmt.Errorf("empty csv")
	}

	headers := records[0]
	var parts []Part
	for _, row := range records[1:] {
		fields := make(map[string]string)
		ipn := ""
		for i, h := range headers {
			if i < len(row) {
				fields[h] = row[i]
				hl := strings.ToLower(h)
				if hl == "ipn" || hl == "part_number" || hl == "pn" {
					ipn = row[i]
				}
			}
		}
		fields["_category"] = category
		if ipn == "" {
			// Try to derive from filename
			ipn = fields[headers[0]]
		}
		if ipn != "" {
			parts = append(parts, Part{IPN: ipn, Fields: fields})
		}
	}
	return parts, headers, title, nil
}

func handleListParts(w http.ResponseWriter, r *http.Request) {
	cats, _, _, _ := loadPartsFromDir()
	category := r.URL.Query().Get("category")
	q := strings.ToLower(r.URL.Query().Get("q"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 { page = 1 }
	if limit < 1 { limit = 50 }

	var all []Part
	if category != "" {
		all = cats[category]
	} else {
		for _, p := range cats {
			all = append(all, p...)
		}
	}

	// Search filter
	if q != "" {
		var filtered []Part
		for _, p := range all {
			if strings.Contains(strings.ToLower(p.IPN), q) {
				filtered = append(filtered, p)
				continue
			}
			for _, v := range p.Fields {
				if strings.Contains(strings.ToLower(v), q) {
					filtered = append(filtered, p)
					break
				}
			}
		}
		all = filtered
	}

	// Deduplicate by IPN (keep first occurrence)
	seen := make(map[string]bool)
	deduped := make([]Part, 0, len(all))
	for _, p := range all {
		if !seen[p.IPN] {
			seen[p.IPN] = true
			deduped = append(deduped, p)
		}
	}
	all = deduped

	sort.Slice(all, func(i, j int) bool { return all[i].IPN < all[j].IPN })
	total := len(all)
	start := (page - 1) * limit
	if start > total { start = total }
	end := start + limit
	if end > total { end = total }

	if all == nil { all = []Part{} }
	jsonRespMeta(w, all[start:end], total, page, limit)
}

func handleGetPart(w http.ResponseWriter, r *http.Request, ipn string) {
	cats, _, _, _ := loadPartsFromDir()
	for _, parts := range cats {
		for _, p := range parts {
			if p.IPN == ipn {
				jsonResp(w, p)
				return
			}
		}
	}
	jsonErr(w, "part not found", 404)
}

func handleCreatePart(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IPN      string            `json:"ipn"`
		Category string            `json:"category"`
		Fields   map[string]string `json:"fields"`
	}
	if err := decodeBody(r, &body); err != nil {
		jsonErr(w, "invalid request body", 400)
		return
	}
	if body.IPN == "" {
		jsonErr(w, "ipn is required", 400)
		return
	}
	if body.Category == "" {
		jsonErr(w, "category is required", 400)
		return
	}

	// Find the CSV file for this category
	csvPath := findCategoryCSV(body.Category)
	if csvPath == "" {
		jsonErr(w, "category not found", 404)
		return
	}

	// Check IPN uniqueness across all categories
	cats, _, _, _ := loadPartsFromDir()
	for _, parts := range cats {
		for _, p := range parts {
			if p.IPN == body.IPN {
				jsonErr(w, "IPN already exists", 409)
				return
			}
		}
	}

	// Read existing CSV to get headers
	f, err := os.Open(csvPath)
	if err != nil {
		jsonErr(w, "failed to read category CSV", 500)
		return
	}
	csvReader := csv.NewReader(f)
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true
	records, err := csvReader.ReadAll()
	f.Close()
	if err != nil || len(records) < 1 {
		jsonErr(w, "failed to parse category CSV", 500)
		return
	}

	headers := records[0]

	// Build the new row
	row := make([]string, len(headers))
	for i, h := range headers {
		hl := strings.ToLower(h)
		if hl == "ipn" || hl == "part_number" || hl == "pn" {
			row[i] = body.IPN
		} else if v, ok := body.Fields[h]; ok {
			row[i] = v
		} else if v, ok := body.Fields[strings.ToLower(h)]; ok {
			row[i] = v
		}
	}

	// Append to CSV
	records = append(records, row)
	wf, err := os.Create(csvPath)
	if err != nil {
		jsonErr(w, "failed to write CSV", 500)
		return
	}
	csvWriter := csv.NewWriter(wf)
	csvWriter.WriteAll(records)
	wf.Close()

	fields := make(map[string]string)
	for i, h := range headers {
		fields[h] = row[i]
	}
	fields["_category"] = body.Category

	jsonResp(w, Part{IPN: body.IPN, Fields: fields})
}

// findCategoryCSV locates the CSV file for a given category name
func findCategoryCSV(category string) string {
	if partsDir == "" {
		return ""
	}
	// Try direct filename match (e.g., "z-ana" -> "z-ana.csv")
	p := filepath.Join(partsDir, category+".csv")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	// Try case-insensitive
	entries, err := os.ReadDir(partsDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".csv") {
			name := strings.TrimSuffix(e.Name(), ".csv")
			if strings.EqualFold(name, category) {
				return filepath.Join(partsDir, e.Name())
			}
		}
	}
	return ""
}

func handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title  string `json:"title"`
		Prefix string `json:"prefix"`
	}
	if err := decodeBody(r, &body); err != nil || body.Title == "" || body.Prefix == "" {
		jsonErr(w, "title and prefix are required", 400)
		return
	}

	prefix := strings.ToLower(body.Prefix)
	filename := "z-" + prefix + ".csv"
	csvPath := filepath.Join(partsDir, filename)

	// Check if already exists
	if _, err := os.Stat(csvPath); err == nil {
		jsonErr(w, "category with this prefix already exists", 409)
		return
	}

	// Create CSV with title comment on first line, then headers
	f, err := os.Create(csvPath)
	if err != nil {
		jsonErr(w, "failed to create category file", 500)
		return
	}
	// Write title as a special comment line: # TITLE: <title>
	fmt.Fprintf(f, "# TITLE: %s\n", body.Title)
	csvWriter := csv.NewWriter(f)
	csvWriter.Write([]string{"IPN", "description", "manufacturer", "value"})
	csvWriter.Flush()
	f.Close()

	catID := strings.TrimSuffix(filename, ".csv")
	jsonResp(w, Category{ID: catID, Name: body.Title, Count: 0, Columns: []string{"IPN", "description", "manufacturer", "value"}})
}

func handleCheckIPN(w http.ResponseWriter, r *http.Request) {
	ipn := r.URL.Query().Get("ipn")
	if ipn == "" {
		jsonErr(w, "ipn query parameter required", 400)
		return
	}
	cats, _, _, _ := loadPartsFromDir()
	exists := false
	for _, parts := range cats {
		for _, p := range parts {
			if p.IPN == ipn {
				exists = true
				break
			}
		}
		if exists {
			break
		}
	}
	jsonResp(w, map[string]bool{"exists": exists})
}

func handleUpdatePart(w http.ResponseWriter, r *http.Request, ipn string) {
	jsonErr(w, "updating parts via API not yet supported — edit CSVs directly", 501)
}

func handleDeletePart(w http.ResponseWriter, r *http.Request, ipn string) {
	jsonErr(w, "deleting parts via API not yet supported — edit CSVs directly", 501)
}

func handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, schemas, titles, _ := loadPartsFromDir()
	var result []Category
	for name, parts := range cats {
		cols := schemas[name]
		if cols == nil { cols = []string{} }
		displayName := titles[name]
		if displayName == "" {
			displayName = name // Fallback to filename if no title
		}
		result = append(result, Category{ID: name, Name: displayName, Count: len(parts), Columns: cols})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	jsonResp(w, result)
}

func handleAddColumn(w http.ResponseWriter, r *http.Request, catID string) {
	var body struct{ Name string `json:"name"` }
	if err := decodeBody(r, &body); err != nil || body.Name == "" {
		jsonErr(w, "name required", 400)
		return
	}
	// Would need to modify CSV files - stub for now
	jsonResp(w, map[string]string{"status": "column add not yet implemented for CSV backend"})
}

func handleDeleteColumn(w http.ResponseWriter, r *http.Request, catID, colName string) {
	jsonResp(w, map[string]string{"status": "column delete not yet implemented for CSV backend"})
}

// getPartByIPN looks up a single IPN across all CSV categories and returns its fields
func getPartByIPN(pmDir, ipn string) (map[string]string, error) {
	if pmDir == "" {
		return nil, fmt.Errorf("no parts directory configured")
	}
	cats, _, _, err := loadPartsFromDir()
	if err != nil {
		return nil, err
	}
	for _, parts := range cats {
		for _, p := range parts {
			if p.IPN == ipn {
				return p.Fields, nil
			}
		}
	}
	return nil, fmt.Errorf("part not found: %s", ipn)
}

// BOMNode represents a node in the BOM tree
type BOMNode struct {
	IPN         string    `json:"ipn"`
	Description string    `json:"description"`
	Qty         float64   `json:"qty,omitempty"`
	Ref         string    `json:"ref,omitempty"`
	Children    []BOMNode `json:"children"`
}

func handlePartBOM(w http.ResponseWriter, r *http.Request, ipn string) {
	// Only works for assembly IPNs
	upper := strings.ToUpper(ipn)
	if !strings.HasPrefix(upper, "PCA-") && !strings.HasPrefix(upper, "ASY-") {
		jsonErr(w, "BOM only available for assembly IPNs (PCA, ASY prefix)", 400)
		return
	}

	node, err := buildBOMTree(ipn, 0, 5)
	if err != nil {
		jsonErr(w, err.Error(), 404)
		return
	}
	jsonResp(w, node)
}

func buildBOMTree(ipn string, depth, maxDepth int) (*BOMNode, error) {
	if depth > maxDepth {
		return &BOMNode{IPN: ipn, Description: "(max depth reached)", Children: []BOMNode{}}, nil
	}

	// Look up part description
	desc := ""
	fields, _ := getPartByIPN(partsDir, ipn)
	if fields != nil {
		for k, v := range fields {
			if strings.EqualFold(k, "description") || strings.EqualFold(k, "desc") {
				desc = v
				break
			}
		}
	}

	node := &BOMNode{IPN: ipn, Description: desc, Children: []BOMNode{}}

	// Try to find BOM CSV: look for <IPN>.csv in partsDir
	if partsDir == "" {
		return node, nil
	}

	// Search for BOM file: try exact IPN.csv, then in subdirectories
	bomPaths := []string{
		filepath.Join(partsDir, ipn+".csv"),
	}
	// Also check subdirectories
	entries, _ := os.ReadDir(partsDir)
	for _, e := range entries {
		if e.IsDir() {
			bomPaths = append(bomPaths, filepath.Join(partsDir, e.Name(), ipn+".csv"))
		}
	}

	var bomFile string
	for _, p := range bomPaths {
		if _, err := os.Stat(p); err == nil {
			bomFile = p
			break
		}
	}

	if bomFile == "" {
		return node, nil
	}

	f, err := os.Open(bomFile)
	if err != nil {
		return node, nil
	}
	defer f.Close()

	rdr := csv.NewReader(f)
	rdr.LazyQuotes = true
	rdr.TrimLeadingSpace = true
	records, err := rdr.ReadAll()
	if err != nil || len(records) < 2 {
		return node, nil
	}

	headers := records[0]
	ipnIdx, qtyIdx, refIdx, descIdx := -1, -1, -1, -1
	for i, h := range headers {
		hl := strings.ToLower(h)
		switch {
		case hl == "ipn" || hl == "part_number" || hl == "pn":
			ipnIdx = i
		case hl == "qty" || hl == "quantity":
			qtyIdx = i
		case hl == "ref" || hl == "reference" || hl == "designator" || hl == "ref_des":
			refIdx = i
		case hl == "description" || hl == "desc":
			descIdx = i
		}
	}
	if ipnIdx == -1 {
		ipnIdx = 0
	}

	for _, row := range records[1:] {
		if ipnIdx >= len(row) {
			continue
		}
		childIPN := strings.TrimSpace(row[ipnIdx])
		if childIPN == "" {
			continue
		}
		var qty float64 = 1
		if qtyIdx >= 0 && qtyIdx < len(row) {
			if q, err := strconv.ParseFloat(strings.TrimSpace(row[qtyIdx]), 64); err == nil {
				qty = q
			}
		}
		ref := ""
		if refIdx >= 0 && refIdx < len(row) {
			ref = strings.TrimSpace(row[refIdx])
		}
		childDesc := ""
		if descIdx >= 0 && descIdx < len(row) {
			childDesc = strings.TrimSpace(row[descIdx])
		}

		childUpper := strings.ToUpper(childIPN)
		if strings.HasPrefix(childUpper, "PCA-") || strings.HasPrefix(childUpper, "ASY-") {
			// Recursively expand sub-assemblies
			childNode, _ := buildBOMTree(childIPN, depth+1, maxDepth)
			if childNode != nil {
				childNode.Qty = qty
				childNode.Ref = ref
				if childNode.Description == "" {
					childNode.Description = childDesc
				}
				node.Children = append(node.Children, *childNode)
			}
		} else {
			// Leaf part - get description from parts DB if not in BOM
			if childDesc == "" {
				childFields, _ := getPartByIPN(partsDir, childIPN)
				if childFields != nil {
					for k, v := range childFields {
						if strings.EqualFold(k, "description") || strings.EqualFold(k, "desc") {
							childDesc = v
							break
						}
					}
				}
			}
			node.Children = append(node.Children, BOMNode{IPN: childIPN, Description: childDesc, Qty: qty, Ref: ref, Children: []BOMNode{}})
		}
	}

	return node, nil
}

func handlePartCost(w http.ResponseWriter, r *http.Request, ipn string) {
	result := map[string]interface{}{"ipn": ipn}

	// Last unit price from PO lines
	var unitPrice float64
	var poID, lastOrdered string
	err := db.QueryRow(`SELECT pl.unit_price, pl.po_id, po.created_at FROM po_lines pl
		JOIN purchase_orders po ON po.id = pl.po_id
		WHERE pl.ipn=? AND pl.unit_price > 0 ORDER BY po.created_at DESC LIMIT 1`, ipn).Scan(&unitPrice, &poID, &lastOrdered)
	if err == nil {
		result["last_unit_price"] = unitPrice
		result["po_id"] = poID
		result["last_ordered"] = lastOrdered
	}

	// BOM cost for assemblies
	upper := strings.ToUpper(ipn)
	if strings.HasPrefix(upper, "PCA-") || strings.HasPrefix(upper, "ASY-") {
		bomCost := calcBOMCost(ipn, 0, 5)
		result["bom_cost"] = bomCost
	}

	jsonResp(w, result)
}

func calcBOMCost(ipn string, depth, maxDepth int) float64 {
	if depth > maxDepth || partsDir == "" {
		return 0
	}
	// Find BOM file
	bomPaths := []string{filepath.Join(partsDir, ipn+".csv")}
	entries, _ := os.ReadDir(partsDir)
	for _, e := range entries {
		if e.IsDir() {
			bomPaths = append(bomPaths, filepath.Join(partsDir, e.Name(), ipn+".csv"))
		}
	}
	var bomFile string
	for _, p := range bomPaths {
		if _, err := os.Stat(p); err == nil {
			bomFile = p
			break
		}
	}
	if bomFile == "" {
		return 0
	}
	f, err := os.Open(bomFile)
	if err != nil {
		return 0
	}
	defer f.Close()
	rdr := csv.NewReader(f)
	rdr.LazyQuotes = true
	rdr.TrimLeadingSpace = true
	records, err := rdr.ReadAll()
	if err != nil || len(records) < 2 {
		return 0
	}
	headers := records[0]
	ipnIdx, qtyIdx := -1, -1
	for i, h := range headers {
		hl := strings.ToLower(h)
		if hl == "ipn" || hl == "part_number" || hl == "pn" {
			ipnIdx = i
		}
		if hl == "qty" || hl == "quantity" {
			qtyIdx = i
		}
	}
	if ipnIdx == -1 {
		ipnIdx = 0
	}
	var total float64
	for _, row := range records[1:] {
		if ipnIdx >= len(row) {
			continue
		}
		childIPN := strings.TrimSpace(row[ipnIdx])
		if childIPN == "" {
			continue
		}
		var qty float64 = 1
		if qtyIdx >= 0 && qtyIdx < len(row) {
			if q, e := strconv.ParseFloat(strings.TrimSpace(row[qtyIdx]), 64); e == nil {
				qty = q
			}
		}
		childUpper := strings.ToUpper(childIPN)
		if strings.HasPrefix(childUpper, "PCA-") || strings.HasPrefix(childUpper, "ASY-") {
			total += qty * calcBOMCost(childIPN, depth+1, maxDepth)
		} else {
			var price float64
			db.QueryRow("SELECT pl.unit_price FROM po_lines pl JOIN purchase_orders po ON po.id=pl.po_id WHERE pl.ipn=? AND pl.unit_price>0 ORDER BY po.created_at DESC LIMIT 1", childIPN).Scan(&price)
			total += qty * price
		}
	}
	return total
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	d := DashboardData{}
	db.QueryRow("SELECT COUNT(*) FROM ecos WHERE status NOT IN ('implemented','rejected')").Scan(&d.OpenECOs)
	db.QueryRow("SELECT COUNT(*) FROM inventory WHERE qty_on_hand <= reorder_point AND reorder_point > 0").Scan(&d.LowStock)
	db.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE status NOT IN ('received','cancelled')").Scan(&d.OpenPOs)
	db.QueryRow("SELECT COUNT(*) FROM work_orders WHERE status IN ('open','in_progress')").Scan(&d.ActiveWOs)
	db.QueryRow("SELECT COUNT(*) FROM ncrs WHERE status NOT IN ('resolved','closed')").Scan(&d.OpenNCRs)
	db.QueryRow("SELECT COUNT(*) FROM rmas WHERE status NOT IN ('closed')").Scan(&d.OpenRMAs)
	db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&d.TotalDevices)

	// Count parts from CSV
	cats, _, _, _ := loadPartsFromDir()
	for _, p := range cats { d.TotalParts += len(p) }

	json.NewEncoder(w).Encode(d)
}
