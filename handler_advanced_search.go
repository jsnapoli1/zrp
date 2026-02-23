package main

import "net/http"

func handleAdvancedSearch(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().AdvancedSearch(w, r)
}

func handleSaveSavedSearch(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().SaveSavedSearch(w, r)
}

func handleGetSavedSearches(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().GetSavedSearches(w, r)
}

func handleDeleteSavedSearch(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().DeleteSavedSearch(w, r)
}

func handleGetQuickFilters(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().GetQuickFiltersHandler(w, r)
}

func handleGetSearchHistory(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().GetSearchHistory(w, r)
}
