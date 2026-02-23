package main

// This file provides compatibility wrappers for types and functions that were
// moved to internal/handlers/common but are still referenced by other root
// handler files and tests.

import (
	"encoding/csv"
	"net/http"

	"zrp/internal/handlers/common"
)

// Type aliases for types now in the common package.
type Attachment = common.Attachment
type BulkRequest = common.BulkRequest
type BulkResponse = common.BulkResponse
type Notification = common.Notification
type CalendarEvent = common.CalendarEvent
type ScanResult = common.ScanResult
type ChangeEntry = common.ChangeEntry
type UndoLogEntry = common.UndoLogEntry
type NotificationTypeInfo = common.NotificationTypeInfo
type NotificationPreference = common.NotificationPreference

type InvValuationReport = common.InvValuationReport
type InvValuationGroup = common.InvValuationGroup
type InvValuationItem = common.InvValuationItem
type OpenECOItem = common.OpenECOItem
type WOThroughputReport = common.WOThroughputReport
type LowStockItem = common.LowStockItem
type NCRSummaryReport = common.NCRSummaryReport

// notificationTypes is an alias for the common NotificationTypes variable.
var notificationTypes = common.NotificationTypes

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string { return &s }

// float64Ptr returns a pointer to the given float64.
func float64Ptr(f float64) *float64 { return &f }

// --- Snapshot function wrappers ---

func snapshotEntity(entityType, entityID string) (string, error) {
	return getCommonHandler().SnapshotEntity(entityType, entityID)
}

func getECOSnapshot(id string) (map[string]interface{}, error) {
	return getCommonHandler().GetRowAsMap("SELECT * FROM ecos WHERE id=?", id)
}

func getWorkOrderSnapshot(id string) (map[string]interface{}, error) {
	return getCommonHandler().GetRowAsMap("SELECT * FROM work_orders WHERE id=?", id)
}

func getNCRSnapshot(id string) (map[string]interface{}, error) {
	return getCommonHandler().GetRowAsMap("SELECT * FROM ncrs WHERE id=?", id)
}

func getDeviceSnapshot(id string) (map[string]interface{}, error) {
	return getCommonHandler().GetRowAsMap("SELECT * FROM devices WHERE serial_number=?", id)
}

func getInventorySnapshot(id string) (map[string]interface{}, error) {
	return getCommonHandler().GetRowAsMap("SELECT * FROM inventory WHERE ipn=?", id)
}

func getRMASnapshot(id string) (map[string]interface{}, error) {
	return getCommonHandler().GetRowAsMap("SELECT * FROM rmas WHERE id=?", id)
}

func getVendorSnapshot(id string) (map[string]interface{}, error) {
	return getCommonHandler().GetRowAsMap("SELECT * FROM vendors WHERE id=?", id)
}

func getQuoteSnapshot(id string) (map[string]interface{}, error) {
	row, err := getCommonHandler().GetRowAsMap("SELECT * FROM quotes WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	lines, err := getCommonHandler().GetRowsAsMapSlice("SELECT * FROM quote_lines WHERE quote_id=?", id)
	if err == nil {
		row["_lines"] = lines
	}
	return row, nil
}

func getPOSnapshot(id string) (map[string]interface{}, error) {
	row, err := getCommonHandler().GetRowAsMap("SELECT * FROM purchase_orders WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	lines, err := getCommonHandler().GetRowsAsMapSlice("SELECT * FROM po_lines WHERE po_id=?", id)
	if err == nil {
		row["_lines"] = lines
	}
	return row, nil
}

func getRowAsMap(query string, args ...interface{}) (map[string]interface{}, error) {
	return getCommonHandler().GetRowAsMap(query, args...)
}

func getRowsAsMapSlice(query string, args ...interface{}) ([]map[string]interface{}, error) {
	return getCommonHandler().GetRowsAsMapSlice(query, args...)
}

// --- Restore function wrappers ---

func restoreECO(jsonData string) error {
	return getCommonHandler().RestoreECO(jsonData)
}

func restoreWorkOrder(jsonData string) error {
	return getCommonHandler().RestoreWorkOrder(jsonData)
}

func restoreNCR(jsonData string) error {
	return getCommonHandler().RestoreNCR(jsonData)
}

func restoreDevice(jsonData string) error {
	return getCommonHandler().RestoreDevice(jsonData)
}

func restoreInventory(jsonData string) error {
	return getCommonHandler().RestoreInventory(jsonData)
}

func restoreRMA(jsonData string) error {
	return getCommonHandler().RestoreRMA(jsonData)
}

func restoreVendor(jsonData string) error {
	return getCommonHandler().RestoreVendor(jsonData)
}

func restoreQuote(jsonData string) error {
	return getCommonHandler().RestoreQuote(jsonData)
}

func restorePO(jsonData string) error {
	return getCommonHandler().RestorePO(jsonData)
}

func performUndo(entry common.UndoLogEntry) error {
	return getCommonHandler().PerformUndo(entry)
}

func deleteByTable(tableName, recordID string) error {
	return getCommonHandler().DeleteByTable(tableName, recordID)
}

func restoreByTable(tableName, recordID, jsonData string) error {
	return getCommonHandler().RestoreByTable(tableName, recordID, jsonData)
}

func genericRestore(tableName, jsonData string) error {
	return getCommonHandler().GenericRestore(tableName, jsonData)
}

func tableIDColumn(tableName string) string {
	return common.TableIDColumn(tableName)
}

// --- Notification function wrappers ---

func createNotificationIfNew(ntype, severity, title string, message, recordID, module *string) {
	getCommonHandler().CreateNotificationIfNew(ntype, severity, title, message, recordID, module)
}

// --- Notification pref wrappers ---

func ensureDefaultPreferences(userID int) {
	getCommonHandler().EnsureDefaultPreferences(userID)
}

func generateNotificationsForUser(userID int) {
	getCommonHandler().GenerateNotificationsForUser(userID)
}

func getUserNotifPref(userID int, notifType string) (enabled bool, deliveryMethod string, threshold *float64) {
	return getCommonHandler().GetUserNotifPref(userID, notifType)
}

func isValidDeliveryMethod(m string) bool {
	return common.IsValidDeliveryMethod(m)
}

func isValidNotificationType(t string) bool {
	return common.IsValidNotificationType(t)
}

func getNotificationPrefsForUser(userID int) []NotificationPreference {
	return getCommonHandler().GetNotificationPrefsForUser(userID)
}

// --- Export function wrappers ---

func exportCSV(w http.ResponseWriter, filename string, headers []string, data [][]string) {
	common.ExportCSV(w, filename, headers, data)
}

func exportExcel(w http.ResponseWriter, sheetName string, headers []string, data [][]string) {
	common.ExportExcel(w, sheetName, headers, data)
}

// --- Search function wrappers ---

func getUserFromRequest(r *http.Request) string {
	user := r.Header.Get("X-User-ID")
	if user == "" {
		user = "system"
	}
	return user
}

// --- Report helpers ---

func ipnCategory(ipn string) string {
	return common.IPNCategory(ipn)
}

func splitIPN(ipn string) []string {
	return common.SplitIPN(ipn)
}

func writeCSV(w http.ResponseWriter, name string, headers []string, writeFn func(*csv.Writer)) {
	common.WriteCSV(w, name, headers, writeFn)
}
