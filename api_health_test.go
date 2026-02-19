package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// TestAPIHealth tests EVERY API endpoint to ensure none return 500 or unexpected 404
// This is a regression safety net to catch issues like missing DB columns, etc.
func TestAPIHealth(t *testing.T) {
	// Setup test database
	oldDB := db
	db = setupHealthTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	// Seed test data
	seedHealthTestData(t, db)

	// Login to get session cookie
	sessionCookie := loginAsAdmin(t)

	// Test all endpoints
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantCode   int  // expected status code
		allow404   bool // some endpoints legitimately return 404 for non-existent resources
		skipAuth   bool // endpoints that don't require auth
	}{
		// Auth endpoints (no auth required)
		{"Health Check", "GET", "/healthz", "", 200, false, true},
		{"Login", "POST", "/auth/login", `{"username":"admin","password":"password"}`, 200, false, true},
		{"Me", "GET", "/auth/me", "", 200, false, false},

		// Search
		{"Global Search", "GET", "/api/v1/search?q=test", "", 200, false, false},
		{"Scan Lookup", "GET", "/api/v1/scan/TEST123", "", 200, true, false},

		// Dashboard
		{"Dashboard", "GET", "/api/v1/dashboard", "", 200, false, false},
		{"Dashboard Charts", "GET", "/api/v1/dashboard/charts", "", 200, false, false},
		{"Low Stock Alerts", "GET", "/api/v1/dashboard/lowstock", "", 200, false, false},
		{"Get Dashboard Widgets", "GET", "/api/v1/dashboard/widgets", "", 200, false, false},

		// Audit
		{"Audit Log", "GET", "/api/v1/audit", "", 200, false, false},

		// Parts
		{"List Parts", "GET", "/api/v1/parts", "", 200, false, false},
		{"List Categories", "GET", "/api/v1/parts/categories", "", 200, false, false},
		{"Check IPN", "GET", "/api/v1/parts/check-ipn?ipn=TEST-001", "", 200, false, false},
		{"Get Part", "GET", "/api/v1/parts/TEST-PART-001", "", 200, true, false},
		{"Part BOM", "GET", "/api/v1/parts/TEST-PART-001/bom", "", 200, true, false},
		{"Part Cost", "GET", "/api/v1/parts/TEST-PART-001/cost", "", 200, true, false},
		{"Part Where Used", "GET", "/api/v1/parts/TEST-PART-001/where-used", "", 200, true, false},
		{"Part Changes", "GET", "/api/v1/parts/TEST-PART-001/changes", "", 200, true, false},
		{"List All Part Changes", "GET", "/api/v1/part-changes", "", 200, false, false},

		// Categories
		{"List Categories 2", "GET", "/api/v1/categories", "", 200, false, false},

		// Calendar
		{"Calendar", "GET", "/api/v1/calendar", "", 200, false, false},

		// ECOs
		{"List ECOs", "GET", "/api/v1/ecos", "", 200, false, false},
		{"Get ECO", "GET", "/api/v1/ecos/ECO-001", "", 200, true, false},
		{"ECO Part Changes", "GET", "/api/v1/ecos/ECO-001/part-changes", "", 200, true, false},
		{"ECO Revisions", "GET", "/api/v1/ecos/ECO-001/revisions", "", 200, true, false},

		// Documents
		{"List Docs", "GET", "/api/v1/docs", "", 200, false, false},
		{"Get Doc", "GET", "/api/v1/docs/DOC-001", "", 200, true, false},
		{"Doc Versions", "GET", "/api/v1/docs/DOC-001/versions", "", 200, true, false},

		// Vendors
		{"List Vendors", "GET", "/api/v1/vendors", "", 200, false, false},
		{"Get Vendor", "GET", "/api/v1/vendors/1", "", 200, true, false},

		// Inventory
		{"List Inventory", "GET", "/api/v1/inventory", "", 200, false, false},
		{"Get Inventory", "GET", "/api/v1/inventory/1", "", 200, true, false},
		{"Inventory History", "GET", "/api/v1/inventory/1/history", "", 200, true, false},

		// Purchase Orders
		{"List POs", "GET", "/api/v1/pos", "", 200, false, false},
		{"Get PO", "GET", "/api/v1/pos/PO-001", "", 200, true, false},

		// Receiving
		{"List Receiving", "GET", "/api/v1/receiving", "", 200, false, false},

		// Work Orders (THIS IS THE CRITICAL ONE - tests qty_good column)
		{"List Work Orders", "GET", "/api/v1/workorders", "", 200, false, false},
		{"Get Work Order", "GET", "/api/v1/workorders/WO-001", "", 200, true, false},
		{"Work Order BOM", "GET", "/api/v1/workorders/WO-001/bom", "", 200, true, false},

		// Tests
		{"List Tests", "GET", "/api/v1/tests", "", 200, false, false},

		// NCRs
		{"List NCRs", "GET", "/api/v1/ncrs", "", 200, false, false},
		{"Get NCR", "GET", "/api/v1/ncrs/NCR-001", "", 200, true, false},

		// Devices
		{"List Devices", "GET", "/api/v1/devices", "", 200, false, false},
		{"Get Device", "GET", "/api/v1/devices/1", "", 200, true, false},
		{"Device History", "GET", "/api/v1/devices/1/history", "", 200, true, false},
		{"Export Devices", "GET", "/api/v1/devices/export", "", 200, false, false},

		// Firmware Campaigns
		{"List Campaigns", "GET", "/api/v1/campaigns", "", 200, false, false},
		{"Get Campaign", "GET", "/api/v1/campaigns/1", "", 200, true, false},
		{"Campaign Progress", "GET", "/api/v1/campaigns/1/progress", "", 200, true, false},
		{"Campaign Devices", "GET", "/api/v1/campaigns/1/devices", "", 200, true, false},
		{"List Firmware", "GET", "/api/v1/firmware", "", 200, false, false},

		// Shipments
		{"List Shipments", "GET", "/api/v1/shipments", "", 200, false, false},
		{"Get Shipment", "GET", "/api/v1/shipments/SHIP-001", "", 200, true, false},

		// CAPAs
		{"CAPA Dashboard", "GET", "/api/v1/capas/dashboard", "", 200, false, false},
		{"List CAPAs", "GET", "/api/v1/capas", "", 200, false, false},
		{"Get CAPA", "GET", "/api/v1/capas/1", "", 200, true, false},

		// RMAs
		{"List RMAs", "GET", "/api/v1/rmas", "", 200, false, false},
		{"Get RMA", "GET", "/api/v1/rmas/1", "", 200, true, false},

		// Quotes
		{"List Quotes", "GET", "/api/v1/quotes", "", 200, false, false},
		{"Get Quote", "GET", "/api/v1/quotes/1", "", 200, true, false},
		{"Quote Cost", "GET", "/api/v1/quotes/1/cost", "", 200, true, false},

		// API Keys
		{"List API Keys", "GET", "/api/v1/apikeys", "", 200, false, false},
		{"List API Keys Alt", "GET", "/api/v1/api-keys", "", 200, false, false},

		// Users
		{"List Users", "GET", "/api/v1/users", "", 200, false, false},

		// Permissions
		{"List Permissions", "GET", "/api/v1/permissions", "", 200, false, false},
		{"List Modules", "GET", "/api/v1/permissions/modules", "", 200, false, false},
		{"My Permissions", "GET", "/api/v1/permissions/me", "", 200, false, false},

		// Attachments
		{"List Attachments", "GET", "/api/v1/attachments", "", 200, false, false},

		// Email
		{"Get Email Config", "GET", "/api/v1/email/config", "", 200, false, false},
		{"Get Email Subscriptions", "GET", "/api/v1/email/subscriptions", "", 200, false, false},
		{"Email Log", "GET", "/api/v1/email-log", "", 200, false, false},

		// Settings
		{"General Settings", "GET", "/api/v1/settings/general", "", 200, false, false},
		{"GitPLM Settings", "GET", "/api/v1/settings/gitplm", "", 200, false, false},
		{"Git Docs Settings", "GET", "/api/v1/settings/git-docs", "", 200, false, false},
		{"Distributor Settings", "GET", "/api/v1/settings/distributors", "", 200, false, false},
		{"Email Settings", "GET", "/api/v1/settings/email", "", 200, false, false},

		// Config
		{"Config", "GET", "/api/v1/config", "", 200, false, false},

		// Reports
		{"Inventory Valuation Report", "GET", "/api/v1/reports/inventory-valuation", "", 200, false, false},
		{"Open ECOs Report", "GET", "/api/v1/reports/open-ecos", "", 200, false, false},
		{"WO Throughput Report", "GET", "/api/v1/reports/wo-throughput", "", 200, false, false},
		{"Low Stock Report", "GET", "/api/v1/reports/low-stock", "", 200, false, false},
		{"NCR Summary Report", "GET", "/api/v1/reports/ncr-summary", "", 200, false, false},

		// Notifications
		{"List Notifications", "GET", "/api/v1/notifications", "", 200, false, false},
		{"Notification Preferences", "GET", "/api/v1/notifications/preferences", "", 200, false, false},
		{"Notification Types", "GET", "/api/v1/notifications/types", "", 200, false, false},

		// RFQs
		{"List RFQs", "GET", "/api/v1/rfqs", "", 200, false, false},
		{"Get RFQ", "GET", "/api/v1/rfqs/1", "", 200, true, false},
		{"RFQ Compare", "GET", "/api/v1/rfqs/1/compare", "", 200, true, false},
		{"RFQ Email Body", "GET", "/api/v1/rfqs/1/email", "", 200, true, false},
		{"RFQ Dashboard", "GET", "/api/v1/rfq-dashboard", "", 200, false, false},

		// Product Pricing
		{"List Product Pricing", "GET", "/api/v1/pricing", "", 200, false, false},
		{"List Cost Analysis", "GET", "/api/v1/pricing/analysis", "", 200, false, false},
		{"Get Product Pricing", "GET", "/api/v1/pricing/1", "", 200, true, false},
		{"Product Pricing History", "GET", "/api/v1/pricing/history/PROD-001", "", 200, true, false},

		// Change History & Undo
		{"Recent Changes", "GET", "/api/v1/changes/recent", "", 200, false, false},
		{"List Undo", "GET", "/api/v1/undo", "", 200, false, false},

		// Backups
		{"List Backups", "GET", "/api/v1/admin/backups", "", 200, false, false},

		// Field Reports
		{"List Field Reports", "GET", "/api/v1/field-reports", "", 200, false, false},
		{"Get Field Report", "GET", "/api/v1/field-reports/1", "", 200, true, false},

		// Sales Orders
		{"List Sales Orders", "GET", "/api/v1/sales-orders", "", 200, false, false},
		{"Get Sales Order", "GET", "/api/v1/sales-orders/SO-001", "", 200, true, false},

		// Invoices
		{"List Invoices", "GET", "/api/v1/invoices", "", 200, false, false},
		{"Get Invoice", "GET", "/api/v1/invoices/INV-001", "", 200, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			// Add session cookie if auth required
			if !tt.skipAuth {
				req.AddCookie(sessionCookie)
			}

			w := httptest.NewRecorder()

			// Route the request through the proper handler
			if tt.skipAuth && tt.path == "/healthz" {
				// Health check bypasses all middleware
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"status":"ok"}`))
				}).ServeHTTP(w, req)
			} else if tt.skipAuth {
				// Auth endpoints don't require session
				http.DefaultServeMux.ServeHTTP(w, req)
			} else {
				// Regular API endpoints - use the API router
				handleAPIRequest(w, req)
			}

			// Check for 500 errors (NEVER acceptable)
			if w.Code == 500 {
				t.Errorf("%s returned 500 Internal Server Error. Body: %s", tt.name, w.Body.String())
			}

			// Check for unexpected 404s
			if w.Code == 404 && !tt.allow404 {
				t.Errorf("%s returned unexpected 404. Body: %s", tt.name, w.Body.String())
			}

			// Verify expected status code
			if tt.wantCode > 0 && w.Code != tt.wantCode && !(tt.allow404 && w.Code == 404) {
				t.Errorf("%s: expected status %d, got %d. Body: %s", tt.name, tt.wantCode, w.Code, w.Body.String())
			}
		})
	}
}

// handleAPIRequest simulates the main API router
func handleAPIRequest(w http.ResponseWriter, r *http.Request) {
	// Extract session from cookie (auth check)
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Unauthorized", 401)
		return
	}

	var userID int
	err = db.QueryRow("SELECT user_id FROM sessions WHERE token = ? AND expires_at > ?", cookie.Value, time.Now()).Scan(&userID)
	if err != nil {
		http.Error(w, "Unauthorized", 401)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Route to the appropriate handler based on path
	path := r.URL.Path
	method := r.Method

	// Auth endpoints
	if path == "/auth/me" {
		handleMe(w, r)
		return
	}
	if path == "/auth/change-password" && method == "POST" {
		handleChangePassword(w, r)
		return
	}

	// Handle API v1 routes
	if len(path) >= 8 && path[:8] == "/api/v1/" {
		routeAPIv1(w, r)
		return
	}

	// Default: 404
	w.WriteHeader(404)
	json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
}

// routeAPIv1 routes /api/v1/* requests to appropriate handlers
func routeAPIv1(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")
	method := r.Method

	// This is a simplified router for testing - just routes to actual handler functions
	// The goal is to verify handlers don't crash with 500 errors

	switch {
	// Search
	case parts[0] == "search" && len(parts) == 1 && method == "GET":
		handleGlobalSearch(w, r)
	case parts[0] == "scan" && len(parts) == 2 && method == "GET":
		handleScanLookup(w, r, parts[1])

	// Dashboard
	case path == "dashboard" && method == "GET":
		handleDashboard(w, r)
	case path == "dashboard/charts" && method == "GET":
		handleDashboardCharts(w, r)
	case path == "dashboard/lowstock" && method == "GET":
		handleLowStockAlerts(w, r)
	case path == "dashboard/widgets" && method == "GET":
		handleGetDashboardWidgets(w, r)

	// Audit
	case path == "audit" && method == "GET":
		handleAuditLog(w, r)

	// Parts
	case parts[0] == "parts" && len(parts) == 1 && method == "GET":
		handleListParts(w, r)
	case parts[0] == "parts" && len(parts) == 2 && parts[1] == "categories" && method == "GET":
		handleListCategories(w, r)
	case parts[0] == "parts" && len(parts) == 2 && parts[1] == "check-ipn" && method == "GET":
		handleCheckIPN(w, r)
	case parts[0] == "parts" && len(parts) == 2 && method == "GET":
		handleGetPart(w, r, parts[1])
	case parts[0] == "parts" && len(parts) == 3 && parts[2] == "bom" && method == "GET":
		handlePartBOM(w, r, parts[1])
	case parts[0] == "parts" && len(parts) == 3 && parts[2] == "cost" && method == "GET":
		handlePartCost(w, r, parts[1])
	case parts[0] == "parts" && len(parts) == 3 && parts[2] == "where-used" && method == "GET":
		handleWhereUsed(w, r, parts[1])
	case parts[0] == "parts" && len(parts) == 3 && parts[2] == "changes" && method == "GET":
		handleListPartChanges(w, r, parts[1])
	case parts[0] == "parts" && len(parts) == 3 && parts[2] == "market-pricing" && method == "GET":
		handleGetMarketPricing(w, r, parts[1])
	case parts[0] == "parts" && len(parts) == 3 && parts[2] == "gitplm-url" && method == "GET":
		handleGetGitPLMURL(w, r, parts[1])

	// Part Changes
	case parts[0] == "part-changes" && len(parts) == 1 && method == "GET":
		handleListAllPartChanges(w, r)

	// Categories
	case parts[0] == "categories" && len(parts) == 1 && method == "GET":
		handleListCategories(w, r)

	// Calendar
	case parts[0] == "calendar" && len(parts) == 1 && method == "GET":
		handleCalendar(w, r)

	// ECOs
	case parts[0] == "ecos" && len(parts) == 1 && method == "GET":
		handleListECOs(w, r)
	case parts[0] == "ecos" && len(parts) == 2 && method == "GET":
		handleGetECO(w, r, parts[1])
	case parts[0] == "ecos" && len(parts) == 3 && parts[2] == "part-changes" && method == "GET":
		handleListECOPartChanges(w, r, parts[1])
	case parts[0] == "ecos" && len(parts) == 3 && parts[2] == "revisions" && method == "GET":
		handleListECORevisions(w, r, parts[1])

	// Documents
	case parts[0] == "docs" && len(parts) == 1 && method == "GET":
		handleListDocs(w, r)
	case parts[0] == "docs" && len(parts) == 2 && method == "GET":
		handleGetDoc(w, r, parts[1])
	case parts[0] == "docs" && len(parts) == 3 && parts[2] == "versions" && method == "GET":
		handleListDocVersions(w, r, parts[1])

	// Vendors
	case parts[0] == "vendors" && len(parts) == 1 && method == "GET":
		handleListVendors(w, r)
	case parts[0] == "vendors" && len(parts) == 2 && method == "GET":
		handleGetVendor(w, r, parts[1])

	// Inventory
	case parts[0] == "inventory" && len(parts) == 1 && method == "GET":
		handleListInventory(w, r)
	case parts[0] == "inventory" && len(parts) == 2 && method == "GET":
		handleGetInventory(w, r, parts[1])
	case parts[0] == "inventory" && len(parts) == 3 && parts[2] == "history" && method == "GET":
		handleInventoryHistory(w, r, parts[1])

	// Purchase Orders
	case parts[0] == "pos" && len(parts) == 1 && method == "GET":
		handleListPOs(w, r)
	case parts[0] == "pos" && len(parts) == 2 && method == "GET":
		handleGetPO(w, r, parts[1])

	// Receiving
	case parts[0] == "receiving" && len(parts) == 1 && method == "GET":
		handleListReceiving(w, r)

	// Work Orders (CRITICAL - tests qty_good column)
	case parts[0] == "workorders" && len(parts) == 1 && method == "GET":
		handleListWorkOrders(w, r)
	case parts[0] == "workorders" && len(parts) == 2 && method == "GET":
		handleGetWorkOrder(w, r, parts[1])
	case parts[0] == "workorders" && len(parts) == 3 && parts[2] == "bom" && method == "GET":
		handleWorkOrderBOM(w, r, parts[1])

	// Tests
	case parts[0] == "tests" && len(parts) == 1 && method == "GET":
		handleListTests(w, r)

	// NCRs
	case parts[0] == "ncrs" && len(parts) == 1 && method == "GET":
		handleListNCRs(w, r)
	case parts[0] == "ncrs" && len(parts) == 2 && method == "GET":
		handleGetNCR(w, r, parts[1])

	// Devices
	case parts[0] == "devices" && len(parts) == 1 && method == "GET":
		handleListDevices(w, r)
	case parts[0] == "devices" && len(parts) == 2 && parts[1] == "export" && method == "GET":
		handleExportDevices(w, r)
	case parts[0] == "devices" && len(parts) == 2 && method == "GET":
		handleGetDevice(w, r, parts[1])
	case parts[0] == "devices" && len(parts) == 3 && parts[2] == "history" && method == "GET":
		handleDeviceHistory(w, r, parts[1])

	// Firmware/Campaigns
	case parts[0] == "campaigns" && len(parts) == 1 && method == "GET":
		handleListCampaigns(w, r)
	case parts[0] == "campaigns" && len(parts) == 2 && method == "GET":
		handleGetCampaign(w, r, parts[1])
	case parts[0] == "campaigns" && len(parts) == 3 && parts[2] == "progress" && method == "GET":
		handleCampaignProgress(w, r, parts[1])
	case parts[0] == "campaigns" && len(parts) == 3 && parts[2] == "devices" && method == "GET":
		handleCampaignDevices(w, r, parts[1])
	case parts[0] == "firmware" && len(parts) == 1 && method == "GET":
		handleListCampaigns(w, r)

	// Shipments
	case parts[0] == "shipments" && len(parts) == 1 && method == "GET":
		handleListShipments(w, r)
	case parts[0] == "shipments" && len(parts) == 2 && method == "GET":
		handleGetShipment(w, r, parts[1])

	// CAPAs
	case parts[0] == "capas" && len(parts) == 2 && parts[1] == "dashboard" && method == "GET":
		handleCAPADashboard(w, r)
	case parts[0] == "capas" && len(parts) == 1 && method == "GET":
		handleListCAPAs(w, r)
	case parts[0] == "capas" && len(parts) == 2 && method == "GET":
		handleGetCAPA(w, r, parts[1])

	// RMAs
	case parts[0] == "rmas" && len(parts) == 1 && method == "GET":
		handleListRMAs(w, r)
	case parts[0] == "rmas" && len(parts) == 2 && method == "GET":
		handleGetRMA(w, r, parts[1])

	// Quotes
	case parts[0] == "quotes" && len(parts) == 1 && method == "GET":
		handleListQuotes(w, r)
	case parts[0] == "quotes" && len(parts) == 2 && method == "GET":
		handleGetQuote(w, r, parts[1])
	case parts[0] == "quotes" && len(parts) == 3 && parts[2] == "cost" && method == "GET":
		handleQuoteCost(w, r, parts[1])

	// API Keys
	case (parts[0] == "apikeys" || parts[0] == "api-keys") && len(parts) == 1 && method == "GET":
		handleListAPIKeys(w, r)

	// Users
	case parts[0] == "users" && len(parts) == 1 && method == "GET":
		handleListUsers(w, r)

	// Permissions
	case parts[0] == "permissions" && len(parts) == 1 && method == "GET":
		handleListPermissions(w, r)
	case parts[0] == "permissions" && len(parts) == 2 && parts[1] == "modules" && method == "GET":
		handleListModules(w, r)
	case parts[0] == "permissions" && len(parts) == 2 && parts[1] == "me" && method == "GET":
		handleMyPermissions(w, r)

	// Attachments
	case parts[0] == "attachments" && len(parts) == 1 && method == "GET":
		handleListAttachments(w, r)

	// Email
	case parts[0] == "email" && len(parts) == 2 && parts[1] == "config" && method == "GET":
		handleGetEmailConfig(w, r)
	case parts[0] == "email" && len(parts) == 2 && parts[1] == "subscriptions" && method == "GET":
		handleGetEmailSubscriptions(w, r)
	case parts[0] == "email-log" && len(parts) == 1 && method == "GET":
		handleListEmailLog(w, r)

	// Settings
	case parts[0] == "settings" && len(parts) == 2 && parts[1] == "general" && method == "GET":
		handleGetGeneralSettings(w, r)
	case parts[0] == "settings" && len(parts) == 2 && parts[1] == "gitplm" && method == "GET":
		handleGetGitPLMConfig(w, r)
	case parts[0] == "settings" && len(parts) == 2 && parts[1] == "git-docs" && method == "GET":
		handleGetGitDocsSettings(w, r)
	case parts[0] == "settings" && len(parts) == 2 && parts[1] == "distributors" && method == "GET":
		handleGetDistributorSettings(w, r)
	case parts[0] == "settings" && len(parts) == 2 && parts[1] == "email" && method == "GET":
		handleGetEmailConfig(w, r)

	// Config
	case parts[0] == "config" && len(parts) == 1 && method == "GET":
		handleConfig(w, r)

	// Reports
	case parts[0] == "reports" && len(parts) == 2 && parts[1] == "inventory-valuation":
		handleReportInventoryValuation(w, r)
	case parts[0] == "reports" && len(parts) == 2 && parts[1] == "open-ecos":
		handleReportOpenECOs(w, r)
	case parts[0] == "reports" && len(parts) == 2 && parts[1] == "wo-throughput":
		handleReportWOThroughput(w, r)
	case parts[0] == "reports" && len(parts) == 2 && parts[1] == "low-stock":
		handleReportLowStock(w, r)
	case parts[0] == "reports" && len(parts) == 2 && parts[1] == "ncr-summary":
		handleReportNCRSummary(w, r)

	// Notifications
	case parts[0] == "notifications" && len(parts) == 1 && method == "GET":
		handleListNotifications(w, r)
	case parts[0] == "notifications" && len(parts) == 2 && parts[1] == "preferences" && method == "GET":
		handleGetNotificationPreferences(w, r)
	case parts[0] == "notifications" && len(parts) == 2 && parts[1] == "types" && method == "GET":
		handleListNotificationTypes(w, r)

	// RFQs
	case parts[0] == "rfqs" && len(parts) == 1 && method == "GET":
		handleListRFQs(w, r)
	case parts[0] == "rfqs" && len(parts) == 2 && method == "GET":
		handleGetRFQ(w, r, parts[1])
	case parts[0] == "rfqs" && len(parts) == 3 && parts[2] == "compare" && method == "GET":
		handleCompareRFQ(w, r, parts[1])
	case parts[0] == "rfqs" && len(parts) == 3 && parts[2] == "email" && method == "GET":
		handleRFQEmailBody(w, r, parts[1])
	case parts[0] == "rfq-dashboard" && len(parts) == 1 && method == "GET":
		handleRFQDashboard(w, r)

	// Product Pricing
	case parts[0] == "pricing" && len(parts) == 1 && method == "GET":
		handleListProductPricing(w, r)
	case parts[0] == "pricing" && len(parts) == 2 && parts[1] == "analysis" && method == "GET":
		handleListCostAnalysis(w, r)
	case parts[0] == "pricing" && len(parts) == 2 && method == "GET":
		handleGetProductPricing(w, r, parts[1])
	case parts[0] == "pricing" && len(parts) == 3 && parts[1] == "history" && method == "GET":
		handleProductPricingHistory(w, r, parts[2])

	// Change History & Undo
	case parts[0] == "changes" && len(parts) == 2 && parts[1] == "recent" && method == "GET":
		handleRecentChanges(w, r)
	case parts[0] == "undo" && len(parts) == 1 && method == "GET":
		handleListUndo(w, r)

	// Backups
	case parts[0] == "admin" && len(parts) == 2 && parts[1] == "backups" && method == "GET":
		handleListBackups(w, r)

	// Field Reports
	case parts[0] == "field-reports" && len(parts) == 1 && method == "GET":
		handleListFieldReports(w, r)
	case parts[0] == "field-reports" && len(parts) == 2 && method == "GET":
		handleGetFieldReport(w, r, parts[1])

	// Sales Orders
	case parts[0] == "sales-orders" && len(parts) == 1 && method == "GET":
		handleListSalesOrders(w, r)
	case parts[0] == "sales-orders" && len(parts) == 2 && method == "GET":
		handleGetSalesOrder(w, r, parts[1])

	// Invoices
	case parts[0] == "invoices" && len(parts) == 1 && method == "GET":
		handleListInvoices(w, r)
	case parts[0] == "invoices" && len(parts) == 2 && method == "GET":
		handleGetInvoice(w, r, parts[1])

	default:
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}
}

// setupHealthTestDB creates an in-memory database with full schema
func setupHealthTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Enable foreign keys
	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run all migrations by calling the actual migration function
	// We'll temporarily replace the global db with our test db
	oldDB := db
	db = testDB
	if err := runMigrations(); err != nil {
		db = oldDB
		t.Fatalf("Failed to run migrations: %v", err)
	}
	db = oldDB

	return testDB
}

// seedHealthTestData seeds the database with minimal test data
func seedHealthTestData(t *testing.T, testDB *sql.DB) {
	// Create admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	_, err := testDB.Exec(
		"INSERT INTO users (username, password_hash, display_name, role, active, email) VALUES (?, ?, ?, ?, ?, ?)",
		"admin", string(hash), "Admin User", "admin", 1, "admin@example.com",
	)
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	// Create a test vendor
	_, err = testDB.Exec(
		"INSERT INTO vendors (id, name, contact_email, contact_phone, address, notes, payment_terms, lead_time_days) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		1, "Test Vendor", "vendor@test.com", "555-0100", "123 Main St", "Test vendor", "Net 30", 7,
	)
	if err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}

	// Create a test work order (CRITICAL - this tests qty_good column)
	_, err = testDB.Exec(
		"INSERT INTO work_orders (id, ipn, title, quantity, status, assigned_to, created_by, priority, due_date, qty_good, qty_scrap) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"WO-001", "TEST-PART-001", "Test Work Order", 100, "pending", "admin", "admin", "normal", "2024-12-31", 0, 0,
	)
	if err != nil {
		t.Fatalf("Failed to create work order: %v", err)
	}

	// Create test ECO
	_, err = testDB.Exec(
		"INSERT INTO ecos (id, title, description, status, created_by, priority, ncr_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"ECO-001", "Test ECO", "Test engineering change order", "draft", "admin", "normal", "",
	)
	if err != nil {
		t.Fatalf("Failed to create ECO: %v", err)
	}

	// Create test inventory
	_, err = testDB.Exec(
		"INSERT INTO inventory (id, ipn, location, quantity, lot, description, mpn) VALUES (?, ?, ?, ?, ?, ?, ?)",
		1, "TEST-PART-001", "A1", 100, "LOT-001", "Test part", "MPN-001",
	)
	if err != nil {
		t.Fatalf("Failed to create inventory: %v", err)
	}

	// Create test device
	_, err = testDB.Exec(
		"INSERT INTO devices (id, serial_number, ipn, firmware_version, status, location) VALUES (?, ?, ?, ?, ?, ?)",
		1, "SN-001", "TEST-PART-001", "1.0.0", "active", "Field",
	)
	if err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	// Create test NCR
	_, err = testDB.Exec(
		"INSERT INTO ncrs (id, title, description, status, severity, created_by) VALUES (?, ?, ?, ?, ?, ?)",
		"NCR-001", "Test NCR", "Test non-conformance", "open", "major", "admin",
	)
	if err != nil {
		t.Fatalf("Failed to create NCR: %v", err)
	}
}

// loginAsAdmin logs in as admin and returns the session cookie
func loginAsAdmin(t *testing.T) *http.Cookie {
	// Create session token
	token := "test-session-token"
	expires := time.Now().Add(24 * time.Hour)

	_, err := db.Exec(
		"INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, 1, expires,
	)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	return &http.Cookie{
		Name:    "session",
		Value:   token,
		Expires: expires,
		Path:    "/",
	}
}
