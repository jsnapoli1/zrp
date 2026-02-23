package main

import (
	"database/sql"

	"zrp/internal/auth"
)

// Permission module constant aliases.
const (
	ModuleParts        = auth.ModuleParts
	ModuleECOs         = auth.ModuleECOs
	ModuleDocuments    = auth.ModuleDocuments
	ModuleInventory    = auth.ModuleInventory
	ModuleVendors      = auth.ModuleVendors
	ModulePOs          = auth.ModulePOs
	ModuleWorkOrders   = auth.ModuleWorkOrders
	ModuleNCRs         = auth.ModuleNCRs
	ModuleRMAs         = auth.ModuleRMAs
	ModuleQuotes       = auth.ModuleQuotes
	ModulePricing      = auth.ModulePricing
	ModuleDevices      = auth.ModuleDevices
	ModuleFirmware     = auth.ModuleFirmware
	ModuleShipments    = auth.ModuleShipments
	ModuleFieldReports = auth.ModuleFieldReports
	ModuleRFQs         = auth.ModuleRFQs
	ModuleReports      = auth.ModuleReports
	ModuleTesting      = auth.ModuleTesting
	ModuleAdmin        = auth.ModuleAdmin
)

// Permission action constant aliases.
const (
	ActionView    = auth.PermActionView
	ActionCreate  = auth.PermActionCreate
	ActionEdit    = auth.PermActionEdit
	ActionDelete  = auth.PermActionDelete
	ActionApprove = auth.PermActionApprove
)

// Variable aliases.
var AllModules = auth.AllModules
var AllActions = auth.AllActions

// Type alias.
type Permission = auth.PermissionEntry

// Global permission cache.
var permCache = auth.NewPermCache()

func initPermCache() {
	// Reset by creating fresh cache - not needed since NewPermCache is already clean
}

func refreshPermCache() error {
	return permCache.Refresh(db)
}

func HasPermission(role, module, action string) bool {
	return permCache.HasPermission(role, module, action)
}

func GetRolePermissions(role string) []Permission {
	return permCache.GetRolePermissions(role)
}

func initPermissionsTable() error {
	return auth.InitPermissionsTable(db, permCache)
}

func seedDefaultPermissions() error {
	return auth.SeedDefaultPermissions(db)
}

func setRolePermissions(dbConn *sql.DB, role string, perms []Permission) error {
	return auth.SetRolePermissions(dbConn, permCache, role, perms)
}

func mapAPIPathToPermission(apiPath, method string) (module, action string) {
	return auth.MapAPIPathToPermission(apiPath, method)
}
