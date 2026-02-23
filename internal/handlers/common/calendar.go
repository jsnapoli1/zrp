package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// CalendarEvent represents a calendar event.
type CalendarEvent struct {
	Date  string `json:"date"`
	Type  string `json:"type"`
	ID    string `json:"id"`
	Title string `json:"title"`
	Color string `json:"color"`
}

// Calendar returns calendar events for a given month.
func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil {
			month = m
		}
	}

	startDate := fmt.Sprintf("%04d-%02d-01", year, month)
	endDate := fmt.Sprintf("%04d-%02d-31", year, month)

	var events []CalendarEvent

	rows, err := h.DB.Query(`SELECT id, COALESCE(notes,''),
		CASE WHEN completed_at IS NOT NULL THEN completed_at
		ELSE datetime(created_at, '+30 days') END as due_date,
		assembly_ipn, qty
		FROM work_orders
		WHERE CASE WHEN completed_at IS NOT NULL THEN completed_at
		ELSE datetime(created_at, '+30 days') END BETWEEN ? AND ?`,
		startDate, endDate+" 23:59:59")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, notes, dueDate, assemblyIPN string
			var qty int
			rows.Scan(&id, &notes, &dueDate, &assemblyIPN, &qty)
			if len(dueDate) >= 10 {
				dueDate = dueDate[:10]
			}
			title := fmt.Sprintf("Build %s x%d", assemblyIPN, qty)
			if notes != "" {
				title = notes
			}
			events = append(events, CalendarEvent{Date: dueDate, Type: "workorder", ID: id, Title: title, Color: "blue"})
		}
	}

	rows2, err := h.DB.Query(`SELECT id, COALESCE(notes,''), expected_date FROM purchase_orders WHERE expected_date BETWEEN ? AND ?`, startDate, endDate)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var id, notes, expDate string
			rows2.Scan(&id, &notes, &expDate)
			if len(expDate) >= 10 {
				expDate = expDate[:10]
			}
			title := "PO expected delivery"
			if notes != "" {
				title = notes
			}
			events = append(events, CalendarEvent{Date: expDate, Type: "po", ID: id, Title: title, Color: "green"})
		}
	}

	rows3, err := h.DB.Query(`SELECT id, customer, valid_until FROM quotes WHERE valid_until BETWEEN ? AND ?`, startDate, endDate)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var id, customer, validUntil string
			rows3.Scan(&id, &customer, &validUntil)
			if len(validUntil) >= 10 {
				validUntil = validUntil[:10]
			}
			title := fmt.Sprintf("Quote %s expires", id)
			if customer != "" {
				title = fmt.Sprintf("Quote for %s expires", customer)
			}
			events = append(events, CalendarEvent{Date: validUntil, Type: "quote", ID: id, Title: title, Color: "orange"})
		}
	}

	if events == nil {
		events = []CalendarEvent{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
