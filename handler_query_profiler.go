package main

import (
	"net/http"
)

func handleQueryProfilerStats(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().QueryProfilerStats(w, r)
}

func handleQueryProfilerSlowQueries(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().QueryProfilerSlowQueries(w, r)
}

func handleQueryProfilerAllQueries(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().QueryProfilerAllQueries(w, r)
}

func handleQueryProfilerReset(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().QueryProfilerReset(w, r)
}
