package main

import "net/http"

func handleScanLookup(w http.ResponseWriter, r *http.Request, code string) {
	getCommonHandler().ScanLookup(w, r, code)
}
