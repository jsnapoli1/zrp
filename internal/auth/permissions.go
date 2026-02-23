package auth

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Permission modules correspond to major feature areas.
const (
	ModuleParts        = "parts"
	ModuleECOs         = "ecos"
	ModuleDocuments    = "documents"
	ModuleInventory    = "inventory"
	ModuleVendors      = "vendors"
	ModulePOs          = "purchase_orders"
	ModuleWorkOrders   = "work_orders"
	ModuleNCRs         = "ncrs"
	ModuleRMAs         = "rmas"
	ModuleQuotes       = "quotes"
	ModulePricing      = "pricing"
	ModuleDevices      = "devices"
	ModuleFirmware     = "firmware"
	ModuleShipments    = "shipments"
	ModuleFieldReports = "field_reports"
	ModuleRFQs         = "rfqs"
	ModuleReports      = "reports"
	ModuleTesting      = "testing"
	ModuleAdmin        = "admin"
)

// Permission actions.
const (
	PermActionView    = "view"
	PermActionCreate  = "create"
	PermActionEdit    = "edit"
	PermActionDelete  = "delete"
	PermActionApprove = "approve"
)

// AllModules lists every module.
var AllModules = []string{
	ModuleParts, ModuleECOs, ModuleDocuments, ModuleInventory, ModuleVendors,
	ModulePOs, ModuleWorkOrders, ModuleNCRs, ModuleRMAs, ModuleQuotes,
	ModulePricing, ModuleDevices, ModuleFirmware, ModuleShipments,
	ModuleFieldReports, ModuleRFQs, ModuleReports, ModuleTesting, ModuleAdmin,
}

// AllActions lists every action.
var AllActions = []string{PermActionView, PermActionCreate, PermActionEdit, PermActionDelete, PermActionApprove}

// PermissionEntry represents a single permission assignment.
type PermissionEntry struct {
	ID     int    `json:"id"`
	Role   string `json:"role"`
	Module string `json:"module"`
	Action string `json:"action"`
}

// PermCache caches role→permissions for fast middleware lookups.
type PermCache struct {
	sync.RWMutex
	data    map[string]map[string]map[string]bool // role → module → action → true
	updated time.Time
}

// NewPermCache creates a new empty permission cache.
func NewPermCache() *PermCache {
	return &PermCache{
		data: make(map[string]map[string]map[string]bool),
	}
}

// Refresh loads all role_permissions into the in-memory cache.
func (pc *PermCache) Refresh(db *sql.DB) error {
	rows, err := db.Query("SELECT role, module, action FROM role_permissions")
	if err != nil {
		return err
	}
	defer rows.Close()

	data := make(map[string]map[string]map[string]bool)
	for rows.Next() {
		var role, module, action string
		if err := rows.Scan(&role, &module, &action); err != nil {
			continue
		}
		if data[role] == nil {
			data[role] = make(map[string]map[string]bool)
		}
		if data[role][module] == nil {
			data[role][module] = make(map[string]bool)
		}
		data[role][module][action] = true
	}

	pc.Lock()
	pc.data = data
	pc.updated = time.Now()
	pc.Unlock()
	return nil
}

// HasPermission checks whether a role has permission for module+action.
func (pc *PermCache) HasPermission(role, module, action string) bool {
	pc.RLock()
	defer pc.RUnlock()
	if pc.data[role] == nil {
		return false
	}
	if pc.data[role][module] == nil {
		return false
	}
	return pc.data[role][module][action]
}

// GetRolePermissions returns all permissions for a role.
func (pc *PermCache) GetRolePermissions(role string) []PermissionEntry {
	pc.RLock()
	defer pc.RUnlock()
	var perms []PermissionEntry
	if pc.data[role] == nil {
		return perms
	}
	for mod, actions := range pc.data[role] {
		for act := range actions {
			perms = append(perms, PermissionEntry{Role: role, Module: mod, Action: act})
		}
	}
	return perms
}

// InitPermissionsTable creates the role_permissions table and seeds default data.
func InitPermissionsTable(db *sql.DB, pc *PermCache) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS role_permissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		role TEXT NOT NULL,
		module TEXT NOT NULL,
		action TEXT NOT NULL,
		UNIQUE(role, module, action)
	)`)
	if err != nil {
		return fmt.Errorf("create role_permissions table: %w", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM role_permissions").Scan(&count)
	if count == 0 {
		if err := SeedDefaultPermissions(db); err != nil {
			return fmt.Errorf("seed permissions: %w", err)
		}
	}

	return pc.Refresh(db)
}

// SeedDefaultPermissions populates the default role permissions.
func SeedDefaultPermissions(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO role_permissions (role, module, action) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Admin: everything
	for _, mod := range AllModules {
		for _, act := range AllActions {
			if _, err := stmt.Exec("admin", mod, act); err != nil {
				return err
			}
		}
	}

	// User: everything except admin module
	for _, mod := range AllModules {
		if mod == ModuleAdmin {
			continue
		}
		for _, act := range AllActions {
			if _, err := stmt.Exec("user", mod, act); err != nil {
				return err
			}
		}
	}

	// Readonly: view only on all modules
	for _, mod := range AllModules {
		if _, err := stmt.Exec("readonly", mod, PermActionView); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SetRolePermissions replaces all permissions for a role with the given set.
func SetRolePermissions(db *sql.DB, pc *PermCache, role string, perms []PermissionEntry) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM role_permissions WHERE role = ?", role); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO role_permissions (role, module, action) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range perms {
		if _, err := stmt.Exec(role, p.Module, p.Action); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return pc.Refresh(db)
}

// MapAPIPathToPermission maps an API path + method to (module, action).
// Returns empty strings if no permission mapping exists (passthrough).
func MapAPIPathToPermission(apiPath, method string) (module, action string) {
	parts := strings.Split(apiPath, "/")
	if len(parts) == 0 {
		return "", ""
	}

	seg := parts[0]

	switch method {
	case "GET":
		action = PermActionView
	case "POST":
		action = PermActionCreate
	case "PUT", "PATCH":
		action = PermActionEdit
	case "DELETE":
		action = PermActionDelete
	}

	if len(parts) >= 3 {
		switch parts[2] {
		case "approve":
			action = PermActionApprove
		case "implement":
			action = PermActionApprove
		}
	}

	switch seg {
	case "parts", "categories", "part-changes":
		module = ModuleParts
	case "ecos":
		module = ModuleECOs
	case "docs":
		module = ModuleDocuments
	case "inventory":
		module = ModuleInventory
	case "vendors":
		module = ModuleVendors
	case "pos":
		module = ModulePOs
	case "workorders":
		module = ModuleWorkOrders
	case "ncrs":
		module = ModuleNCRs
	case "rmas":
		module = ModuleRMAs
	case "quotes":
		module = ModuleQuotes
	case "pricing":
		module = ModulePricing
	case "devices":
		module = ModuleDevices
	case "campaigns":
		module = ModuleFirmware
	case "shipments":
		module = ModuleShipments
	case "field-reports":
		module = ModuleFieldReports
	case "rfqs", "rfq-dashboard":
		module = ModuleRFQs
	case "reports":
		module = ModuleReports
	case "tests":
		module = ModuleTesting
	case "users", "apikeys", "api-keys", "admin":
		module = ModuleAdmin
	case "email":
		module = ModuleAdmin
	case "settings":
		module = ModuleAdmin
	case "receiving":
		module = ModuleInventory
	case "prices":
		module = ModulePricing
	case "dashboard", "search", "scan", "audit", "calendar",
		"changes", "undo", "notifications", "email-log",
		"config", "attachments", "openapi.json":
		return "", ""
	default:
		return "", ""
	}

	return module, action
}
