package main

import "net/http"

func handleGlobalSearch(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().GlobalSearch(w, r)
}
