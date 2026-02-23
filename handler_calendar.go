package main

import "net/http"

func handleCalendar(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().Calendar(w, r)
}
