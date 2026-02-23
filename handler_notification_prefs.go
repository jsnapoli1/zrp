package main

import "net/http"

func initNotificationPrefsTable() {
	getCommonHandler().InitNotificationPrefsTable()
}

func handleGetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().GetNotificationPreferences(w, r)
}

func handleUpdateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().UpdateNotificationPreferences(w, r)
}

func handleUpdateSingleNotificationPreference(w http.ResponseWriter, r *http.Request, notifType string) {
	getCommonHandler().UpdateSingleNotificationPreference(w, r, notifType)
}

func handleListNotificationTypes(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ListNotificationTypes(w, r)
}

func generateNotificationsFiltered() {
	getCommonHandler().GenerateNotificationsFiltered()
}
