package main

import (
	"net/http"

	"zrp/internal/handlers/admin"
)

// Type alias for backward compatibility.
type DashboardWidget = admin.DashboardWidget

func handleGetDashboardWidgets(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().GetDashboardWidgets(w, r)
}

func handleUpdateDashboardWidgets(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().UpdateDashboardWidgets(w, r)
}
