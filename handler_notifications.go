package main

import "net/http"

func handleListNotifications(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ListNotifications(w, r)
}

func handleMarkNotificationRead(w http.ResponseWriter, r *http.Request, id string) {
	getCommonHandler().MarkNotificationRead(w, r, id)
}

func generateNotifications() {
	getCommonHandler().GenerateNotifications()
}
