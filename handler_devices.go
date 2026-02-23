package main

import (
	"net/http"

	"zrp/internal/handlers/field"
)

// fieldHandler is the shared field handler instance.
var fieldHandler *field.Handler

// getFieldHandler returns the field handler, lazily initializing if needed (for tests).
func getFieldHandler() *field.Handler {
	if fieldHandler == nil || fieldHandler.DB != db {
		fieldHandler = &field.Handler{
			DB:                db,
			Hub:               wsHub,
			NextIDFunc:        nextID,
			RecordChangeJSON:  recordChangeJSON,
			GetDeviceSnapshot: getDeviceSnapshot,
			GetRMASnapshot:    getRMASnapshot,
		}
	}
	return fieldHandler
}

func handleListDevices(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().ListDevices(w, r)
}

func handleGetDevice(w http.ResponseWriter, r *http.Request, serial string) {
	getFieldHandler().GetDevice(w, r, serial)
}

func handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().CreateDevice(w, r)
}

func handleUpdateDevice(w http.ResponseWriter, r *http.Request, serial string) {
	getFieldHandler().UpdateDevice(w, r, serial)
}

func handleExportDevices(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().ExportDevices(w, r)
}

func handleImportDevices(w http.ResponseWriter, r *http.Request) {
	getFieldHandler().ImportDevices(w, r)
}

func handleDeviceHistory(w http.ResponseWriter, r *http.Request, serial string) {
	getFieldHandler().DeviceHistory(w, r, serial)
}
