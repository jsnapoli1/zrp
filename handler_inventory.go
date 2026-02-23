package main

import (
	"net/http"

	"zrp/internal/handlers/inventory"
)

// inventoryHandler is the shared inventory handler instance.
var inventoryHandler *inventory.Handler

// getInventoryHandler returns the inventory handler, lazily initializing if needed (for tests).
func getInventoryHandler() *inventory.Handler {
	if inventoryHandler == nil || inventoryHandler.DB != db {
		inventoryHandler = &inventory.Handler{
			DB:              db,
			Hub:             wsHub,
			PartsDir:        partsDir,
			GetPartByIPN:    getPartByIPN,
			EmailOnLowStock: emailOnLowStock,
		}
	}
	return inventoryHandler
}

func handleListInventory(w http.ResponseWriter, r *http.Request) {
	getInventoryHandler().ListInventory(w, r)
}

func handleGetInventory(w http.ResponseWriter, r *http.Request, ipn string) {
	getInventoryHandler().GetInventory(w, r, ipn)
}

func handleInventoryTransact(w http.ResponseWriter, r *http.Request) {
	getInventoryHandler().Transact(w, r)
}

func handleInventoryHistory(w http.ResponseWriter, r *http.Request, ipn string) {
	getInventoryHandler().History(w, r, ipn)
}

func handleBulkDeleteInventory(w http.ResponseWriter, r *http.Request) {
	getInventoryHandler().BulkDelete(w, r)
}
