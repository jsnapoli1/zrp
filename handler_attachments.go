package main

import "net/http"

func handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().UploadAttachment(w, r)
}

func handleListAttachments(w http.ResponseWriter, r *http.Request) {
	getCommonHandler().ListAttachments(w, r)
}

func handleServeFile(w http.ResponseWriter, r *http.Request, filename string) {
	getCommonHandler().ServeFile(w, r, filename)
}

func handleDeleteAttachment(w http.ResponseWriter, r *http.Request, idStr string) {
	getCommonHandler().DeleteAttachment(w, r, idStr)
}

func handleDownloadAttachment(w http.ResponseWriter, r *http.Request, idStr string) {
	getCommonHandler().DownloadAttachment(w, r, idStr)
}
