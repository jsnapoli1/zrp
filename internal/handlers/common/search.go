package common

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// GlobalSearch handles global search across all entity types.
func (h *Handler) GlobalSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 20
	}
	if q == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"parts": []interface{}{}, "ecos": []interface{}{}, "workorders": []interface{}{},
				"devices": []interface{}{}, "ncrs": []interface{}{}, "pos": []interface{}{}, "quotes": []interface{}{},
			},
			"meta": map[string]interface{}{"total": 0, "query": ""},
		})
		return
	}

	total := 0

	// Parts
	cats, _, _, _ := h.LoadPartsFromDir()
	var matchedParts []map[string]string
	for _, parts := range cats {
		for _, p := range parts {
			if len(matchedParts) >= limit {
				break
			}
			if strings.Contains(strings.ToLower(p.IPN), q) {
				matchedParts = append(matchedParts, p.Fields)
				continue
			}
			for _, v := range p.Fields {
				if strings.Contains(strings.ToLower(v), q) {
					matchedParts = append(matchedParts, p.Fields)
					break
				}
			}
		}
	}
	if matchedParts == nil {
		matchedParts = []map[string]string{}
	}
	total += len(matchedParts)

	// ECOs
	type ecoResult struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	var ecos []ecoResult
	ecoRows, _ := h.DB.Query("SELECT id,title,COALESCE(description,''),status FROM ecos")
	if ecoRows != nil {
		defer ecoRows.Close()
		for ecoRows.Next() {
			var id, title, desc, status string
			ecoRows.Scan(&id, &title, &desc, &status)
			if strings.Contains(strings.ToLower(id), q) || strings.Contains(strings.ToLower(title), q) || strings.Contains(strings.ToLower(desc), q) {
				ecos = append(ecos, ecoResult{id, title, status})
				if len(ecos) >= limit {
					break
				}
			}
		}
	}
	if ecos == nil {
		ecos = []ecoResult{}
	}
	total += len(ecos)

	// Work Orders
	type woResult struct {
		ID          string `json:"id"`
		AssemblyIPN string `json:"assembly_ipn"`
		Status      string `json:"status"`
	}
	var wos []woResult
	woRows, _ := h.DB.Query("SELECT id,assembly_ipn,status FROM work_orders")
	if woRows != nil {
		defer woRows.Close()
		for woRows.Next() {
			var id, aipn, status string
			woRows.Scan(&id, &aipn, &status)
			if strings.Contains(strings.ToLower(id), q) || strings.Contains(strings.ToLower(aipn), q) {
				wos = append(wos, woResult{id, aipn, status})
				if len(wos) >= limit {
					break
				}
			}
		}
	}
	if wos == nil {
		wos = []woResult{}
	}
	total += len(wos)

	// Devices
	type devResult struct {
		SerialNumber string `json:"serial_number"`
		IPN          string `json:"ipn"`
		Customer     string `json:"customer"`
		Status       string `json:"status"`
	}
	var devs []devResult
	devRows, _ := h.DB.Query("SELECT serial_number,ipn,COALESCE(customer,''),status FROM devices")
	if devRows != nil {
		defer devRows.Close()
		for devRows.Next() {
			var sn, ipn, cust, status string
			devRows.Scan(&sn, &ipn, &cust, &status)
			if strings.Contains(strings.ToLower(sn), q) || strings.Contains(strings.ToLower(cust), q) || strings.Contains(strings.ToLower(ipn), q) {
				devs = append(devs, devResult{sn, ipn, cust, status})
				if len(devs) >= limit {
					break
				}
			}
		}
	}
	if devs == nil {
		devs = []devResult{}
	}
	total += len(devs)

	// NCRs
	type ncrResult struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	var ncrs []ncrResult
	ncrRows, _ := h.DB.Query("SELECT id,title,status FROM ncrs")
	if ncrRows != nil {
		defer ncrRows.Close()
		for ncrRows.Next() {
			var id, title, status string
			ncrRows.Scan(&id, &title, &status)
			if strings.Contains(strings.ToLower(id), q) || strings.Contains(strings.ToLower(title), q) {
				ncrs = append(ncrs, ncrResult{id, title, status})
				if len(ncrs) >= limit {
					break
				}
			}
		}
	}
	if ncrs == nil {
		ncrs = []ncrResult{}
	}
	total += len(ncrs)

	// Purchase Orders
	type poResult struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	var pos []poResult
	poRows, _ := h.DB.Query("SELECT id,status FROM purchase_orders")
	if poRows != nil {
		defer poRows.Close()
		for poRows.Next() {
			var id, status string
			poRows.Scan(&id, &status)
			if strings.Contains(strings.ToLower(id), q) {
				pos = append(pos, poResult{id, status})
				if len(pos) >= limit {
					break
				}
			}
		}
	}
	if pos == nil {
		pos = []poResult{}
	}
	total += len(pos)

	// Quotes
	type quoteResult struct {
		ID       string `json:"id"`
		Customer string `json:"customer"`
		Status   string `json:"status"`
	}
	var quotes []quoteResult
	qRows, _ := h.DB.Query("SELECT id,COALESCE(customer,''),status FROM quotes")
	if qRows != nil {
		defer qRows.Close()
		for qRows.Next() {
			var id, cust, status string
			qRows.Scan(&id, &cust, &status)
			if strings.Contains(strings.ToLower(id), q) || strings.Contains(strings.ToLower(cust), q) {
				quotes = append(quotes, quoteResult{id, cust, status})
				if len(quotes) >= limit {
					break
				}
			}
		}
	}
	if quotes == nil {
		quotes = []quoteResult{}
	}
	total += len(quotes)

	result := map[string]interface{}{
		"parts": matchedParts, "ecos": ecos, "workorders": wos,
		"devices": devs, "ncrs": ncrs, "pos": pos, "quotes": quotes,
	}
	meta := map[string]interface{}{"total": total, "query": r.URL.Query().Get("q")}
	json.NewEncoder(w).Encode(map[string]interface{}{"data": result, "meta": meta})
}
