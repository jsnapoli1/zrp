package common

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"
)

// --- Inventory Valuation ---

// InvValuationItem represents a single inventory valuation line item.
type InvValuationItem struct {
	IPN       string  `json:"ipn"`
	Desc      string  `json:"description"`
	Category  string  `json:"category"`
	QtyOnHand float64 `json:"qty_on_hand"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
	PORef     string  `json:"po_ref"`
}

// InvValuationGroup groups valuation items by category.
type InvValuationGroup struct {
	Category string             `json:"category"`
	Items    []InvValuationItem `json:"items"`
	Subtotal float64            `json:"subtotal"`
}

// InvValuationReport is the full inventory valuation report.
type InvValuationReport struct {
	Groups     []InvValuationGroup `json:"groups"`
	GrandTotal float64             `json:"grand_total"`
}

// ReportInventoryValuation handles the inventory valuation report endpoint.
func (h *Handler) ReportInventoryValuation(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`
		SELECT i.ipn, COALESCE(i.description,''), COALESCE(i.mpn,''), i.qty_on_hand,
			COALESCE((SELECT pl.unit_price FROM po_lines pl JOIN purchase_orders po ON pl.po_id=po.id
				WHERE pl.ipn=i.ipn ORDER BY po.created_at DESC LIMIT 1), 0) as unit_price,
			COALESCE((SELECT pl.po_id FROM po_lines pl JOIN purchase_orders po ON pl.po_id=po.id
				WHERE pl.ipn=i.ipn ORDER BY po.created_at DESC LIMIT 1), '') as po_ref
		FROM inventory i ORDER BY i.ipn`)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}
	defer rows.Close()

	catMap := map[string][]InvValuationItem{}
	var catOrder []string
	for rows.Next() {
		var item InvValuationItem
		var mpn string
		rows.Scan(&item.IPN, &item.Desc, &mpn, &item.QtyOnHand, &item.UnitPrice, &item.PORef)
		item.Subtotal = item.QtyOnHand * item.UnitPrice
		// Derive category from IPN prefix
		item.Category = IPNCategory(item.IPN)
		if _, ok := catMap[item.Category]; !ok {
			catOrder = append(catOrder, item.Category)
		}
		catMap[item.Category] = append(catMap[item.Category], item)
	}

	report := InvValuationReport{}
	for _, cat := range catOrder {
		items := catMap[cat]
		grp := InvValuationGroup{Category: cat, Items: items}
		for _, it := range items {
			grp.Subtotal += it.Subtotal
		}
		report.GrandTotal += grp.Subtotal
		report.Groups = append(report.Groups, grp)
	}

	if r.URL.Query().Get("format") == "csv" {
		WriteCSV(w, "inventory-valuation", []string{"IPN", "Description", "Category", "Qty On Hand", "Unit Price", "Subtotal", "PO Ref"}, func(cw *csv.Writer) {
			for _, g := range report.Groups {
				for _, it := range g.Items {
					cw.Write([]string{it.IPN, it.Desc, it.Category, fmt.Sprintf("%.2f", it.QtyOnHand), fmt.Sprintf("%.4f", it.UnitPrice), fmt.Sprintf("%.2f", it.Subtotal), it.PORef})
				}
			}
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// --- Open ECOs by Priority ---

// OpenECOItem represents an open ECO in the report.
type OpenECOItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Priority  string `json:"priority"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	AgeDays   int    `json:"age_days"`
}

// ReportOpenECOs handles the open ECOs report endpoint.
func (h *Handler) ReportOpenECOs(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT id, title, status, priority, created_by, created_at FROM ecos WHERE status IN ('draft','review') ORDER BY
		CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'normal' THEN 2 WHEN 'low' THEN 3 ELSE 4 END, created_at`)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}
	defer rows.Close()
	var items []OpenECOItem
	now := time.Now()
	for rows.Next() {
		var e OpenECOItem
		rows.Scan(&e.ID, &e.Title, &e.Status, &e.Priority, &e.CreatedBy, &e.CreatedAt)
		if t, err := time.Parse("2006-01-02 15:04:05", e.CreatedAt); err == nil {
			e.AgeDays = int(now.Sub(t).Hours() / 24)
		}
		items = append(items, e)
	}
	if items == nil {
		items = []OpenECOItem{}
	}

	if r.URL.Query().Get("format") == "csv" {
		WriteCSV(w, "open-ecos", []string{"ID", "Title", "Status", "Priority", "Created By", "Created At", "Age (Days)"}, func(cw *csv.Writer) {
			for _, e := range items {
				cw.Write([]string{e.ID, e.Title, e.Status, e.Priority, e.CreatedBy, e.CreatedAt, strconv.Itoa(e.AgeDays)})
			}
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// --- WO Throughput ---

// WOThroughputReport is the work order throughput report.
type WOThroughputReport struct {
	Days             int            `json:"days"`
	CountByStatus    map[string]int `json:"count_by_status"`
	TotalCompleted   int            `json:"total_completed"`
	AvgCycleTimeDays float64        `json:"avg_cycle_time_days"`
}

// ReportWOThroughput handles the work order throughput report endpoint.
func (h *Handler) ReportWOThroughput(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && (v == 30 || v == 60 || v == 90) {
			days = v
		}
	}
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")

	rows, err := h.DB.Query(`SELECT status, started_at, completed_at FROM work_orders WHERE completed_at IS NOT NULL AND completed_at >= ?`, since)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}
	defer rows.Close()

	report := WOThroughputReport{Days: days, CountByStatus: map[string]int{}}
	var totalCycleHours float64
	cycleCount := 0
	for rows.Next() {
		var status string
		var startedAt, completedAt *string
		rows.Scan(&status, &startedAt, &completedAt)
		report.CountByStatus[status]++
		report.TotalCompleted++
		if startedAt != nil && completedAt != nil {
			if st, err1 := time.Parse("2006-01-02 15:04:05", *startedAt); err1 == nil {
				if ct, err2 := time.Parse("2006-01-02 15:04:05", *completedAt); err2 == nil {
					totalCycleHours += ct.Sub(st).Hours()
					cycleCount++
				}
			}
		}
	}
	if cycleCount > 0 {
		report.AvgCycleTimeDays = math.Round(totalCycleHours/float64(cycleCount)/24*100) / 100
	}

	if r.URL.Query().Get("format") == "csv" {
		WriteCSV(w, "wo-throughput", []string{"Days", "Total Completed", "Avg Cycle Time (Days)"}, func(cw *csv.Writer) {
			cw.Write([]string{strconv.Itoa(report.Days), strconv.Itoa(report.TotalCompleted), fmt.Sprintf("%.2f", report.AvgCycleTimeDays)})
			cw.Write([]string{})
			cw.Write([]string{"Status", "Count"})
			for s, c := range report.CountByStatus {
				cw.Write([]string{s, strconv.Itoa(c)})
			}
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// --- Low Stock ---

// LowStockItem represents a low-stock inventory item.
type LowStockItem struct {
	IPN            string  `json:"ipn"`
	Description    string  `json:"description"`
	QtyOnHand      float64 `json:"qty_on_hand"`
	ReorderPoint   float64 `json:"reorder_point"`
	ReorderQty     float64 `json:"reorder_qty"`
	SuggestedOrder float64 `json:"suggested_order"`
}

// ReportLowStock handles the low stock report endpoint.
func (h *Handler) ReportLowStock(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(`SELECT ipn, COALESCE(description,''), qty_on_hand, reorder_point, reorder_qty FROM inventory WHERE qty_on_hand < reorder_point AND reorder_point > 0 ORDER BY (reorder_point - qty_on_hand) DESC`)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}
	defer rows.Close()
	var items []LowStockItem
	for rows.Next() {
		var it LowStockItem
		rows.Scan(&it.IPN, &it.Description, &it.QtyOnHand, &it.ReorderPoint, &it.ReorderQty)
		it.SuggestedOrder = it.ReorderQty
		if it.SuggestedOrder == 0 {
			it.SuggestedOrder = it.ReorderPoint - it.QtyOnHand
		}
		items = append(items, it)
	}
	if items == nil {
		items = []LowStockItem{}
	}

	if r.URL.Query().Get("format") == "csv" {
		WriteCSV(w, "low-stock", []string{"IPN", "Description", "Qty On Hand", "Reorder Point", "Reorder Qty", "Suggested Order"}, func(cw *csv.Writer) {
			for _, it := range items {
				cw.Write([]string{it.IPN, it.Description, fmt.Sprintf("%.2f", it.QtyOnHand), fmt.Sprintf("%.2f", it.ReorderPoint), fmt.Sprintf("%.2f", it.ReorderQty), fmt.Sprintf("%.2f", it.SuggestedOrder)})
			}
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// --- NCR Summary ---

// NCRSummaryReport is the NCR summary report.
type NCRSummaryReport struct {
	BySeverity     map[string]int `json:"by_severity"`
	ByDefectType   map[string]int `json:"by_defect_type"`
	TotalOpen      int            `json:"total_open"`
	AvgResolveDays float64        `json:"avg_resolve_days"`
}

// ReportNCRSummary handles the NCR summary report endpoint.
func (h *Handler) ReportNCRSummary(w http.ResponseWriter, r *http.Request) {
	report := NCRSummaryReport{BySeverity: map[string]int{}, ByDefectType: map[string]int{}}

	// Open NCRs
	rows, err := h.DB.Query(`SELECT COALESCE(severity,'unknown'), COALESCE(defect_type,'unknown') FROM ncrs WHERE status NOT IN ('closed','resolved')`)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, 500)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var sev, dt string
		rows.Scan(&sev, &dt)
		report.BySeverity[sev]++
		report.ByDefectType[dt]++
		report.TotalOpen++
	}

	// Avg resolve time from resolved NCRs
	rrows, err := h.DB.Query(`SELECT created_at, resolved_at FROM ncrs WHERE resolved_at IS NOT NULL`)
	if err == nil {
		defer rrows.Close()
		var totalHours float64
		count := 0
		for rrows.Next() {
			var ca string
			var ra *string
			rrows.Scan(&ca, &ra)
			if ra != nil {
				if ct, e1 := time.Parse("2006-01-02 15:04:05", ca); e1 == nil {
					if rt, e2 := time.Parse("2006-01-02 15:04:05", *ra); e2 == nil {
						totalHours += rt.Sub(ct).Hours()
						count++
					}
				}
			}
		}
		if count > 0 {
			report.AvgResolveDays = math.Round(totalHours/float64(count)/24*100) / 100
		}
	}

	if r.URL.Query().Get("format") == "csv" {
		WriteCSV(w, "ncr-summary", []string{"Metric", "Value"}, func(cw *csv.Writer) {
			cw.Write([]string{"Total Open", strconv.Itoa(report.TotalOpen)})
			cw.Write([]string{"Avg Resolve Days", fmt.Sprintf("%.2f", report.AvgResolveDays)})
			cw.Write([]string{})
			cw.Write([]string{"Severity", "Count"})
			for k, v := range report.BySeverity {
				cw.Write([]string{k, strconv.Itoa(v)})
			}
			cw.Write([]string{})
			cw.Write([]string{"Defect Type", "Count"})
			for k, v := range report.ByDefectType {
				cw.Write([]string{k, strconv.Itoa(v)})
			}
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// --- Helpers ---

// IPNCategory extracts the category prefix from an IPN.
func IPNCategory(ipn string) string {
	parts := SplitIPN(ipn)
	if len(parts) > 0 {
		return parts[0]
	}
	return "Other"
}

// SplitIPN splits an IPN by hyphens.
func SplitIPN(ipn string) []string {
	var result []string
	current := ""
	for _, c := range ipn {
		if c == '-' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// WriteCSV writes CSV data with the given headers and content function.
func WriteCSV(w http.ResponseWriter, name string, headers []string, writeFn func(*csv.Writer)) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", name))
	cw := csv.NewWriter(w)
	cw.Write(headers)
	writeFn(cw)
	cw.Flush()
}
