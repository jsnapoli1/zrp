package main

import (
	"net/http"

	"zrp/internal/handlers/admin"
)

// Type aliases for backward compatibility.
type GeneralSettings = admin.GeneralSettings

func handleGetGeneralSettings(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().GetGeneralSettings(w, r)
}

func handlePutGeneralSettings(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().PutGeneralSettings(w, r)
}
