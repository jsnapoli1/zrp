package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var partsDir string
var gitplmUIURL string
var companyName string
var companyEmail string
var dbFilePath string

func main() {
	pmDir := flag.String("pmDir", "", "Path to gitplm parts database directory")
	port := flag.Int("port", 9000, "HTTP port")
	dbPath := flag.String("db", "zrp.db", "SQLite database path")
	gitplmUI := flag.String("gitplm-ui", "http://localhost:8888", "gitplm-ui base URL")
	flag.Parse()

	partsDir = *pmDir
	gitplmUIURL = *gitplmUI
	dbFilePath = *dbPath

	companyName = os.Getenv("ZRP_COMPANY_NAME")
	if companyName == "" {
		companyName = "Your Company"
	}
	companyEmail = os.Getenv("ZRP_COMPANY_EMAIL")
	if companyEmail == "" {
		companyEmail = "admin@example.com"
	}

	if err := initDB(*dbPath); err != nil {
		log.Fatal("DB init failed:", err)
	}
	seedDB()
	initNotificationPrefsTable()
	if err := initPermissionsTable(); err != nil {
		log.Fatal("Permissions init failed:", err)
	}

	// Start auto-backup scheduler (default 2am, override with ZRP_BACKUP_TIME=HH:MM)
	startAutoBackup(os.Getenv("ZRP_BACKUP_TIME"))

	// Start undo log cleanup goroutine
	go cleanExpiredUndo()

	// Start background notification generator
	go func() {
		time.Sleep(5 * time.Second)
		generateNotificationsFiltered()
		emailNotificationsForRecent()
		for {
			time.Sleep(5 * time.Minute)
			generateNotificationsFiltered()
			emailNotificationsForRecent()
		}
	}()

	mux := http.NewServeMux()

	// WebSocket endpoint (authenticated via session cookie in requireAuth middleware)
	mux.HandleFunc("/api/v1/ws", handleWebSocket)

	// File serving (uploaded attachments)
	mux.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		filename := strings.TrimPrefix(r.URL.Path, "/files/")
		if filename == "" {
			http.NotFound(w, r)
			return
		}
		handleServeFile(w, r, filename)
	})

	// Auth routes
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			handleLogin(w, r)
		} else {
			http.Error(w, "Method not allowed", 405)
		}
	})
	mux.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			handleLogout(w, r)
		} else {
			http.Error(w, "Method not allowed", 405)
		}
	})
	mux.HandleFunc("/auth/me", func(w http.ResponseWriter, r *http.Request) {
		handleMe(w, r)
	})
	mux.HandleFunc("/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			handleChangePassword(w, r)
		} else {
			http.Error(w, "Method not allowed", 405)
		}
	})

	// API routes
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
		path = strings.TrimSuffix(path, "/")
		parts := strings.Split(path, "/")

		switch {
		// Global Search
		case parts[0] == "search" && len(parts) == 1 && r.Method == "GET":
			handleGlobalSearch(w, r)

		// Barcode/QR scan lookup
		case parts[0] == "scan" && len(parts) == 2 && r.Method == "GET":
			handleScanLookup(w, r, parts[1])

		// Dashboard
		case path == "dashboard" && r.Method == "GET":
			handleDashboard(w, r)
		case path == "dashboard/charts" && r.Method == "GET":
			handleDashboardCharts(w, r)
		case path == "dashboard/lowstock" && r.Method == "GET":
			handleLowStockAlerts(w, r)
		case path == "dashboard/widgets" && r.Method == "PUT":
			handleUpdateDashboardWidgets(w, r)
		case path == "dashboard/widgets" && r.Method == "GET":
			handleGetDashboardWidgets(w, r)

		// Audit
		case path == "audit" || (parts[0] == "audit" && len(parts) == 1):
			if r.Method == "GET" {
				handleAuditLog(w, r)
			}

		// Parts
		case parts[0] == "parts" && len(parts) == 1 && r.Method == "GET":
			handleListParts(w, r)
		case parts[0] == "parts" && len(parts) == 1 && r.Method == "POST":
			handleCreatePart(w, r)
		case parts[0] == "parts" && len(parts) == 2 && parts[1] == "categories" && r.Method == "GET":
			handleListCategories(w, r)
		case parts[0] == "parts" && len(parts) == 2 && parts[1] == "check-ipn" && r.Method == "GET":
			handleCheckIPN(w, r)
		case parts[0] == "parts" && len(parts) == 2 && r.Method == "GET":
			handleGetPart(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 3 && parts[2] == "bom" && r.Method == "GET":
			handlePartBOM(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 3 && parts[2] == "cost" && r.Method == "GET":
			handlePartCost(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 3 && parts[2] == "where-used" && r.Method == "GET":
			handleWhereUsed(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 3 && parts[2] == "changes" && r.Method == "POST":
			handleCreatePartChanges(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 3 && parts[2] == "changes" && r.Method == "GET":
			handleListPartChanges(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 4 && parts[2] == "changes" && parts[3] == "create-eco" && r.Method == "POST":
			handleCreateECOFromChanges(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 4 && parts[2] == "changes" && r.Method == "DELETE":
			handleDeletePartChange(w, r, parts[1], parts[3])
		case parts[0] == "parts" && len(parts) == 2 && r.Method == "PUT":
			handleUpdatePart(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 2 && r.Method == "DELETE":
			handleDeletePart(w, r, parts[1])

		// Part Changes (all)
		case parts[0] == "part-changes" && len(parts) == 1 && r.Method == "GET":
			handleListAllPartChanges(w, r)

		// Categories
		case parts[0] == "categories" && len(parts) == 1 && r.Method == "POST":
			handleCreateCategory(w, r)
		case parts[0] == "categories" && len(parts) == 1 && r.Method == "GET":
			handleListCategories(w, r)
		case parts[0] == "categories" && len(parts) == 3 && parts[2] == "columns" && r.Method == "POST":
			handleAddColumn(w, r, parts[1])
		case parts[0] == "categories" && len(parts) == 4 && parts[2] == "columns" && r.Method == "DELETE":
			handleDeleteColumn(w, r, parts[1], parts[3])

		// Calendar
		case parts[0] == "calendar" && len(parts) == 1 && r.Method == "GET":
			handleCalendar(w, r)

		// ECOs
		case parts[0] == "ecos" && len(parts) == 2 && parts[1] == "bulk" && r.Method == "POST":
			handleBulkECOs(w, r)
		case parts[0] == "ecos" && len(parts) == 1 && r.Method == "GET":
			handleListECOs(w, r)
		case parts[0] == "ecos" && len(parts) == 1 && r.Method == "POST":
			handleCreateECO(w, r)
		case parts[0] == "ecos" && len(parts) == 2 && r.Method == "GET":
			handleGetECO(w, r, parts[1])
		case parts[0] == "ecos" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateECO(w, r, parts[1])
		case parts[0] == "ecos" && len(parts) == 3 && parts[2] == "approve" && r.Method == "POST":
			handleApproveECO(w, r, parts[1])
		case parts[0] == "ecos" && len(parts) == 3 && parts[2] == "implement" && r.Method == "POST":
			handleImplementECO(w, r, parts[1])
		case parts[0] == "ecos" && len(parts) == 3 && parts[2] == "part-changes" && r.Method == "GET":
			handleListECOPartChanges(w, r, parts[1])
		case parts[0] == "ecos" && len(parts) == 3 && parts[2] == "revisions" && r.Method == "GET":
			handleListECORevisions(w, r, parts[1])
		case parts[0] == "ecos" && len(parts) == 3 && parts[2] == "revisions" && r.Method == "POST":
			handleCreateECORevision(w, r, parts[1])
		case parts[0] == "ecos" && len(parts) == 4 && parts[2] == "revisions" && r.Method == "GET":
			handleGetECORevision(w, r, parts[1], parts[3])

		// Documents
		case parts[0] == "docs" && len(parts) == 1 && r.Method == "GET":
			handleListDocs(w, r)
		case parts[0] == "docs" && len(parts) == 1 && r.Method == "POST":
			handleCreateDoc(w, r)
		case parts[0] == "docs" && len(parts) == 2 && r.Method == "GET":
			handleGetDoc(w, r, parts[1])
		case parts[0] == "docs" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateDoc(w, r, parts[1])
		case parts[0] == "docs" && len(parts) == 3 && parts[2] == "approve" && r.Method == "POST":
			handleApproveDoc(w, r, parts[1])
		case parts[0] == "docs" && len(parts) == 3 && parts[2] == "versions" && r.Method == "GET":
			handleListDocVersions(w, r, parts[1])
		case parts[0] == "docs" && len(parts) == 4 && parts[2] == "versions" && r.Method == "GET":
			handleGetDocVersion(w, r, parts[1], parts[3])
		case parts[0] == "docs" && len(parts) == 3 && parts[2] == "diff" && r.Method == "GET":
			handleDocDiff(w, r, parts[1])
		case parts[0] == "docs" && len(parts) == 3 && parts[2] == "release" && r.Method == "POST":
			handleReleaseDoc(w, r, parts[1])
		case parts[0] == "docs" && len(parts) == 4 && parts[2] == "revert" && r.Method == "POST":
			handleRevertDoc(w, r, parts[1], parts[3])
		case parts[0] == "docs" && len(parts) == 3 && parts[2] == "push" && r.Method == "POST":
			handlePushDocToGit(w, r, parts[1])
		case parts[0] == "docs" && len(parts) == 3 && parts[2] == "sync" && r.Method == "POST":
			handleSyncDocFromGit(w, r, parts[1])

		// Vendors
		case parts[0] == "vendors" && len(parts) == 1 && r.Method == "GET":
			handleListVendors(w, r)
		case parts[0] == "vendors" && len(parts) == 1 && r.Method == "POST":
			handleCreateVendor(w, r)
		case parts[0] == "vendors" && len(parts) == 2 && r.Method == "GET":
			handleGetVendor(w, r, parts[1])
		case parts[0] == "vendors" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateVendor(w, r, parts[1])
		case parts[0] == "vendors" && len(parts) == 2 && r.Method == "DELETE":
			handleDeleteVendor(w, r, parts[1])

		// Inventory
		case parts[0] == "inventory" && len(parts) == 2 && parts[1] == "bulk" && r.Method == "POST":
			handleBulkInventory(w, r)
		case parts[0] == "inventory" && len(parts) == 2 && parts[1] == "bulk-delete" && r.Method == "DELETE":
			handleBulkDeleteInventory(w, r)
		case parts[0] == "inventory" && len(parts) == 2 && parts[1] == "bulk-update" && r.Method == "POST":
			handleBulkUpdateInventory(w, r)
		case parts[0] == "inventory" && len(parts) == 1 && r.Method == "GET":
			handleListInventory(w, r)
		case parts[0] == "inventory" && len(parts) == 2 && parts[1] == "transact" && r.Method == "POST":
			handleInventoryTransact(w, r)
		case parts[0] == "inventory" && len(parts) == 2 && r.Method == "GET":
			handleGetInventory(w, r, parts[1])
		case parts[0] == "inventory" && len(parts) == 3 && parts[2] == "history" && r.Method == "GET":
			handleInventoryHistory(w, r, parts[1])

		// Purchase Orders
		case parts[0] == "pos" && len(parts) == 1 && r.Method == "GET":
			handleListPOs(w, r)
		case parts[0] == "pos" && len(parts) == 1 && r.Method == "POST":
			handleCreatePO(w, r)
		case parts[0] == "pos" && len(parts) == 2 && (parts[1] == "generate-from-wo" || parts[1] == "generate") && r.Method == "POST":
			handleGeneratePOFromWO(w, r)
		case parts[0] == "pos" && len(parts) == 2 && r.Method == "GET":
			handleGetPO(w, r, parts[1])
		case parts[0] == "pos" && len(parts) == 2 && r.Method == "PUT":
			handleUpdatePO(w, r, parts[1])
		case parts[0] == "pos" && len(parts) == 3 && parts[2] == "receive" && r.Method == "POST":
			handleReceivePO(w, r, parts[1])

		// Receiving/Inspection
		case parts[0] == "receiving" && len(parts) == 1 && r.Method == "GET":
			handleListReceiving(w, r)
		case parts[0] == "receiving" && len(parts) == 3 && parts[2] == "inspect" && r.Method == "POST":
			handleInspectReceiving(w, r, parts[1])

		// Work Orders
		case parts[0] == "workorders" && len(parts) == 2 && parts[1] == "bulk" && r.Method == "POST":
			handleBulkWorkOrders(w, r)
		case parts[0] == "workorders" && len(parts) == 2 && parts[1] == "bulk-update" && r.Method == "POST":
			handleBulkUpdateWorkOrders(w, r)
		case parts[0] == "workorders" && len(parts) == 1 && r.Method == "GET":
			handleListWorkOrders(w, r)
		case parts[0] == "workorders" && len(parts) == 1 && r.Method == "POST":
			handleCreateWorkOrder(w, r)
		case parts[0] == "workorders" && len(parts) == 2 && r.Method == "GET":
			handleGetWorkOrder(w, r, parts[1])
		case parts[0] == "workorders" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateWorkOrder(w, r, parts[1])
		case parts[0] == "workorders" && len(parts) == 3 && parts[2] == "pdf" && r.Method == "GET":
			handleWorkOrderPDF(w, r, parts[1])
		case parts[0] == "workorders" && len(parts) == 3 && parts[2] == "bom" && r.Method == "GET":
			handleWorkOrderBOM(w, r, parts[1])

		// Tests
		case parts[0] == "tests" && len(parts) == 1 && r.Method == "GET":
			handleListTests(w, r)
		case parts[0] == "tests" && len(parts) == 1 && r.Method == "POST":
			handleCreateTest(w, r)
		case parts[0] == "tests" && len(parts) == 2 && r.Method == "GET":
			handleGetTestByID(w, r, parts[1])

		// NCRs
		case parts[0] == "ncrs" && len(parts) == 2 && parts[1] == "bulk" && r.Method == "POST":
			handleBulkNCRs(w, r)
		case parts[0] == "ncrs" && len(parts) == 1 && r.Method == "GET":
			handleListNCRs(w, r)
		case parts[0] == "ncrs" && len(parts) == 1 && r.Method == "POST":
			handleCreateNCR(w, r)
		case parts[0] == "ncrs" && len(parts) == 2 && r.Method == "GET":
			handleGetNCR(w, r, parts[1])
		case parts[0] == "ncrs" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateNCR(w, r, parts[1])
		case parts[0] == "ncrs" && len(parts) == 3 && parts[2] == "create-capa" && r.Method == "POST":
			handleCreateCAPAFromNCR(w, r, parts[1])
		case parts[0] == "ncrs" && len(parts) == 3 && parts[2] == "create-eco" && r.Method == "POST":
			handleCreateECOFromNCR(w, r, parts[1])

		// Devices
		case parts[0] == "devices" && len(parts) == 2 && parts[1] == "bulk" && r.Method == "POST":
			handleBulkDevices(w, r)
		case parts[0] == "devices" && len(parts) == 2 && parts[1] == "bulk-update" && r.Method == "POST":
			handleBulkUpdateDevices(w, r)
		case parts[0] == "devices" && len(parts) == 2 && parts[1] == "export" && r.Method == "GET":
			handleExportDevices(w, r)
		case parts[0] == "devices" && len(parts) == 2 && parts[1] == "import" && r.Method == "POST":
			handleImportDevices(w, r)
		case parts[0] == "devices" && len(parts) == 1 && r.Method == "GET":
			handleListDevices(w, r)
		case parts[0] == "devices" && len(parts) == 1 && r.Method == "POST":
			handleCreateDevice(w, r)
		case parts[0] == "devices" && len(parts) == 2 && r.Method == "GET":
			handleGetDevice(w, r, parts[1])
		case parts[0] == "devices" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateDevice(w, r, parts[1])
		case parts[0] == "devices" && len(parts) == 3 && parts[2] == "history" && r.Method == "GET":
			handleDeviceHistory(w, r, parts[1])

		// Firmware Campaigns
		case parts[0] == "campaigns" && len(parts) == 1 && r.Method == "GET":
			handleListCampaigns(w, r)
		case parts[0] == "campaigns" && len(parts) == 1 && r.Method == "POST":
			handleCreateCampaign(w, r)
		case parts[0] == "campaigns" && len(parts) == 2 && r.Method == "GET":
			handleGetCampaign(w, r, parts[1])
		case parts[0] == "campaigns" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateCampaign(w, r, parts[1])
		case parts[0] == "campaigns" && len(parts) == 3 && parts[2] == "launch" && r.Method == "POST":
			handleLaunchCampaign(w, r, parts[1])
		case parts[0] == "campaigns" && len(parts) == 3 && parts[2] == "progress" && r.Method == "GET":
			handleCampaignProgress(w, r, parts[1])
		case parts[0] == "campaigns" && len(parts) == 3 && parts[2] == "stream" && r.Method == "GET":
			handleCampaignStream(w, r, parts[1])
		case parts[0] == "campaigns" && len(parts) == 3 && parts[2] == "devices" && r.Method == "GET":
			handleCampaignDevices(w, r, parts[1])
		case parts[0] == "campaigns" && len(parts) == 5 && parts[2] == "devices" && parts[4] == "mark" && r.Method == "POST":
			handleMarkCampaignDevice(w, r, parts[1], parts[3])

		// Firmware (aliases for campaigns)
		case parts[0] == "firmware" && len(parts) == 1 && r.Method == "GET":
			handleListCampaigns(w, r)
		case parts[0] == "firmware" && len(parts) == 1 && r.Method == "POST":
			handleCreateCampaign(w, r)
		case parts[0] == "firmware" && len(parts) == 2 && r.Method == "GET":
			handleGetCampaign(w, r, parts[1])
		case parts[0] == "firmware" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateCampaign(w, r, parts[1])
		case parts[0] == "firmware" && len(parts) == 3 && parts[2] == "devices" && r.Method == "GET":
			handleCampaignDevices(w, r, parts[1])

		// Shipments
		case parts[0] == "shipments" && len(parts) == 1 && r.Method == "GET":
			handleListShipments(w, r)
		case parts[0] == "shipments" && len(parts) == 1 && r.Method == "POST":
			handleCreateShipment(w, r)
		case parts[0] == "shipments" && len(parts) == 2 && r.Method == "GET":
			handleGetShipment(w, r, parts[1])
		case parts[0] == "shipments" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateShipment(w, r, parts[1])
		case parts[0] == "shipments" && len(parts) == 3 && parts[2] == "ship" && r.Method == "POST":
			handleShipShipment(w, r, parts[1])
		case parts[0] == "shipments" && len(parts) == 3 && parts[2] == "deliver" && r.Method == "POST":
			handleDeliverShipment(w, r, parts[1])
		case parts[0] == "shipments" && len(parts) == 3 && parts[2] == "pack-list" && r.Method == "GET":
			handleShipmentPackList(w, r, parts[1])

		// CAPAs
		case parts[0] == "capas" && len(parts) == 2 && parts[1] == "dashboard" && r.Method == "GET":
			handleCAPADashboard(w, r)
		case parts[0] == "capas" && len(parts) == 1 && r.Method == "GET":
			handleListCAPAs(w, r)
		case parts[0] == "capas" && len(parts) == 1 && r.Method == "POST":
			handleCreateCAPA(w, r)
		case parts[0] == "capas" && len(parts) == 2 && r.Method == "GET":
			handleGetCAPA(w, r, parts[1])
		case parts[0] == "capas" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateCAPA(w, r, parts[1])

		// RMAs
		case parts[0] == "rmas" && len(parts) == 2 && parts[1] == "bulk" && r.Method == "POST":
			handleBulkRMAs(w, r)
		case parts[0] == "rmas" && len(parts) == 1 && r.Method == "GET":
			handleListRMAs(w, r)
		case parts[0] == "rmas" && len(parts) == 1 && r.Method == "POST":
			handleCreateRMA(w, r)
		case parts[0] == "rmas" && len(parts) == 2 && r.Method == "GET":
			handleGetRMA(w, r, parts[1])
		case parts[0] == "rmas" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateRMA(w, r, parts[1])

		// Quotes
		case parts[0] == "quotes" && len(parts) == 1 && r.Method == "GET":
			handleListQuotes(w, r)
		case parts[0] == "quotes" && len(parts) == 1 && r.Method == "POST":
			handleCreateQuote(w, r)
		case parts[0] == "quotes" && len(parts) == 2 && r.Method == "GET":
			handleGetQuote(w, r, parts[1])
		case parts[0] == "quotes" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateQuote(w, r, parts[1])
		case parts[0] == "quotes" && len(parts) == 3 && parts[2] == "pdf" && r.Method == "GET":
			handleQuotePDF(w, r, parts[1])
		case parts[0] == "quotes" && len(parts) == 3 && parts[2] == "cost" && r.Method == "GET":
			handleQuoteCost(w, r, parts[1])

		// API Keys (supports both "apikeys" and "api-keys" paths)
		case (parts[0] == "apikeys" || parts[0] == "api-keys") && len(parts) == 1 && r.Method == "GET":
			handleListAPIKeys(w, r)
		case (parts[0] == "apikeys" || parts[0] == "api-keys") && len(parts) == 1 && r.Method == "POST":
			handleCreateAPIKey(w, r)
		case (parts[0] == "apikeys" || parts[0] == "api-keys") && len(parts) == 2 && r.Method == "DELETE":
			handleDeleteAPIKey(w, r, parts[1])
		case (parts[0] == "apikeys" || parts[0] == "api-keys") && len(parts) == 2 && r.Method == "PUT":
			handleToggleAPIKey(w, r, parts[1])
		case (parts[0] == "apikeys" || parts[0] == "api-keys") && len(parts) == 3 && parts[2] == "revoke" && r.Method == "POST":
			handleDeleteAPIKey(w, r, parts[1])

		// Users
		case parts[0] == "users" && len(parts) == 1 && r.Method == "GET":
			handleListUsers(w, r)
		case parts[0] == "users" && len(parts) == 1 && r.Method == "POST":
			handleCreateUser(w, r)
		case parts[0] == "users" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateUser(w, r, parts[1])
		case parts[0] == "users" && len(parts) == 2 && r.Method == "DELETE":
			handleDeleteUser(w, r, parts[1])
		case parts[0] == "users" && len(parts) == 3 && parts[2] == "password" && r.Method == "PUT":
			handleResetPassword(w, r, parts[1])

		// Permissions
		case parts[0] == "permissions" && len(parts) == 1 && r.Method == "GET":
			handleListPermissions(w, r)
		case parts[0] == "permissions" && len(parts) == 2 && parts[1] == "modules" && r.Method == "GET":
			handleListModules(w, r)
		case parts[0] == "permissions" && len(parts) == 2 && parts[1] == "me" && r.Method == "GET":
			handleMyPermissions(w, r)
		case parts[0] == "permissions" && len(parts) == 2 && parts[1] != "modules" && parts[1] != "me" && r.Method == "PUT":
			handleSetPermissions(w, r, parts[1])

		// Attachments
		case parts[0] == "attachments" && len(parts) == 1 && r.Method == "POST":
			handleUploadAttachment(w, r)
		case parts[0] == "attachments" && len(parts) == 1 && r.Method == "GET":
			handleListAttachments(w, r)
		case parts[0] == "attachments" && len(parts) == 3 && parts[2] == "download" && r.Method == "GET":
			handleDownloadAttachment(w, r, parts[1])
		case parts[0] == "attachments" && len(parts) == 2 && r.Method == "DELETE":
			handleDeleteAttachment(w, r, parts[1])

		// Prices
		case parts[0] == "prices" && len(parts) == 2 && parts[1] != "" && r.Method == "GET":
			handleListPrices(w, r, parts[1])
		case parts[0] == "prices" && len(parts) == 3 && parts[2] == "trend" && r.Method == "GET":
			handlePriceTrend(w, r, parts[1])
		case parts[0] == "prices" && len(parts) == 1 && r.Method == "POST":
			handleCreatePrice(w, r)
		case parts[0] == "prices" && len(parts) == 2 && r.Method == "DELETE":
			handleDeletePrice(w, r, parts[1])

		// Email
		case parts[0] == "email" && len(parts) == 2 && parts[1] == "config" && r.Method == "GET":
			handleGetEmailConfig(w, r)
		case parts[0] == "email" && len(parts) == 2 && parts[1] == "config" && r.Method == "PUT":
			handleUpdateEmailConfig(w, r)
		case parts[0] == "email" && len(parts) == 2 && parts[1] == "test" && r.Method == "POST":
			handleTestEmail(w, r)
		case parts[0] == "email" && len(parts) == 2 && parts[1] == "subscriptions" && r.Method == "GET":
			handleGetEmailSubscriptions(w, r)
		case parts[0] == "email" && len(parts) == 2 && parts[1] == "subscriptions" && r.Method == "PUT":
			handleUpdateEmailSubscriptions(w, r)
		case parts[0] == "email-log" && len(parts) == 1 && r.Method == "GET":
			handleListEmailLog(w, r)

		// Settings/General
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "general" && r.Method == "GET":
			handleGetGeneralSettings(w, r)
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "general" && r.Method == "PUT":
			handlePutGeneralSettings(w, r)

		// Settings/GitPLM
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "gitplm" && r.Method == "GET":
			handleGetGitPLMConfig(w, r)
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "gitplm" && r.Method == "PUT":
			handleUpdateGitPLMConfig(w, r)

		// Settings/Git Docs
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "git-docs" && r.Method == "GET":
			handleGetGitDocsSettings(w, r)
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "git-docs" && r.Method == "PUT":
			handlePutGitDocsSettings(w, r)

		// ECO PR
		case parts[0] == "ecos" && len(parts) == 3 && parts[2] == "create-pr" && r.Method == "POST":
			handleCreateECOPR(w, r, parts[1])

		// Parts gitplm-url
		case parts[0] == "parts" && len(parts) == 3 && parts[2] == "gitplm-url" && r.Method == "GET":
			handleGetGitPLMURL(w, r, parts[1])

		// Market Pricing
		case parts[0] == "parts" && len(parts) == 3 && parts[2] == "market-pricing" && r.Method == "GET":
			handleGetMarketPricing(w, r, parts[1])

		// Distributor Settings
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "digikey" && r.Method == "POST":
			handleUpdateDigikeySettings(w, r)
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "mouser" && r.Method == "POST":
			handleUpdateMouserSettings(w, r)
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "distributors" && r.Method == "GET":
			handleGetDistributorSettings(w, r)

		// Settings/Email aliases
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "email" && r.Method == "GET":
			handleGetEmailConfig(w, r)
		case parts[0] == "settings" && len(parts) == 2 && parts[1] == "email" && r.Method == "PUT":
			handleUpdateEmailConfig(w, r)
		case parts[0] == "settings" && len(parts) == 3 && parts[1] == "email" && parts[2] == "test" && r.Method == "POST":
			handleTestEmail(w, r)

		// Config
		case parts[0] == "config" && len(parts) == 1 && r.Method == "GET":
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
		case parts[0] == "notifications" && len(parts) == 1 && r.Method == "GET":
			handleListNotifications(w, r)
		case parts[0] == "notifications" && len(parts) == 3 && parts[2] == "read" && r.Method == "POST":
			handleMarkNotificationRead(w, r, parts[1])
		case parts[0] == "notifications" && len(parts) == 2 && parts[1] == "preferences" && r.Method == "GET":
			handleGetNotificationPreferences(w, r)
		case parts[0] == "notifications" && len(parts) == 2 && parts[1] == "preferences" && r.Method == "PUT":
			handleUpdateNotificationPreferences(w, r)
		case parts[0] == "notifications" && len(parts) == 3 && parts[1] == "preferences" && r.Method == "PUT":
			handleUpdateSingleNotificationPreference(w, r, parts[2])
		case parts[0] == "notifications" && len(parts) == 2 && parts[1] == "types" && r.Method == "GET":
			handleListNotificationTypes(w, r)

		// RFQs
		case parts[0] == "rfqs" && len(parts) == 1 && r.Method == "GET":
			handleListRFQs(w, r)
		case parts[0] == "rfqs" && len(parts) == 1 && r.Method == "POST":
			handleCreateRFQ(w, r)
		case parts[0] == "rfqs" && len(parts) == 2 && r.Method == "GET":
			handleGetRFQ(w, r, parts[1])
		case parts[0] == "rfqs" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateRFQ(w, r, parts[1])
		case parts[0] == "rfqs" && len(parts) == 2 && r.Method == "DELETE":
			handleDeleteRFQ(w, r, parts[1])
		case parts[0] == "rfqs" && len(parts) == 3 && parts[2] == "send" && r.Method == "POST":
			handleSendRFQ(w, r, parts[1])
		case parts[0] == "rfqs" && len(parts) == 3 && parts[2] == "award" && r.Method == "POST":
			handleAwardRFQ(w, r, parts[1])
		case parts[0] == "rfqs" && len(parts) == 3 && parts[2] == "compare" && r.Method == "GET":
			handleCompareRFQ(w, r, parts[1])
		case parts[0] == "rfqs" && len(parts) == 3 && parts[2] == "quotes" && r.Method == "POST":
			handleCreateRFQQuote(w, r, parts[1])
		case parts[0] == "rfqs" && len(parts) == 4 && parts[2] == "quotes" && r.Method == "PUT":
			handleUpdateRFQQuote(w, r, parts[1], parts[3])
		case parts[0] == "rfqs" && len(parts) == 3 && parts[2] == "close" && r.Method == "POST":
			handleCloseRFQ(w, r, parts[1])
		case parts[0] == "rfqs" && len(parts) == 3 && parts[2] == "email" && r.Method == "GET":
			handleRFQEmailBody(w, r, parts[1])
		case parts[0] == "rfqs" && len(parts) == 3 && parts[2] == "award-lines" && r.Method == "POST":
			handleAwardRFQPerLine(w, r, parts[1])
		case parts[0] == "rfq-dashboard" && len(parts) == 1 && r.Method == "GET":
			handleRFQDashboard(w, r)

		// Product Pricing
		case parts[0] == "pricing" && len(parts) == 1 && r.Method == "GET":
			handleListProductPricing(w, r)
		case parts[0] == "pricing" && len(parts) == 1 && r.Method == "POST":
			handleCreateProductPricing(w, r)
		case parts[0] == "pricing" && len(parts) == 2 && parts[1] == "analysis" && r.Method == "GET":
			handleListCostAnalysis(w, r)
		case parts[0] == "pricing" && len(parts) == 2 && parts[1] == "analysis" && r.Method == "POST":
			handleCreateCostAnalysis(w, r)
		case parts[0] == "pricing" && len(parts) == 2 && parts[1] == "bulk-update" && r.Method == "POST":
			handleBulkUpdateProductPricing(w, r)
		case parts[0] == "pricing" && len(parts) == 3 && parts[1] == "history" && r.Method == "GET":
			handleProductPricingHistory(w, r, parts[2])
		case parts[0] == "pricing" && len(parts) == 2 && r.Method == "GET":
			handleGetProductPricing(w, r, parts[1])
		case parts[0] == "pricing" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateProductPricing(w, r, parts[1])
		case parts[0] == "pricing" && len(parts) == 2 && r.Method == "DELETE":
			handleDeleteProductPricing(w, r, parts[1])

		// Change History & Undo
		case parts[0] == "changes" && len(parts) == 2 && parts[1] == "recent" && r.Method == "GET":
			handleRecentChanges(w, r)
		case parts[0] == "changes" && len(parts) == 2 && r.Method == "POST":
			handleUndoChange(w, r, parts[1])

		// Undo (legacy)
		case parts[0] == "undo" && len(parts) == 1 && r.Method == "GET":
			handleListUndo(w, r)
		case parts[0] == "undo" && len(parts) == 2 && r.Method == "POST":
			handlePerformUndo(w, r, parts[1])

		// Backups
		case parts[0] == "admin" && len(parts) == 2 && parts[1] == "backup" && r.Method == "POST":
			handleCreateBackup(w, r)
		case parts[0] == "admin" && len(parts) == 2 && parts[1] == "backups" && r.Method == "GET":
			handleListBackups(w, r)
		case parts[0] == "admin" && len(parts) == 3 && parts[1] == "backups" && r.Method == "GET":
			handleDownloadBackup(w, r, parts[2])
		case parts[0] == "admin" && len(parts) == 3 && parts[1] == "backups" && r.Method == "DELETE":
			handleDeleteBackup(w, r, parts[2])
		case parts[0] == "admin" && len(parts) == 2 && parts[1] == "restore" && r.Method == "POST":
			handleRestoreBackup(w, r)

		// Field Reports
		case parts[0] == "field-reports" && len(parts) == 1 && r.Method == "GET":
			handleListFieldReports(w, r)
		case parts[0] == "field-reports" && len(parts) == 1 && r.Method == "POST":
			handleCreateFieldReport(w, r)
		case parts[0] == "field-reports" && len(parts) == 2 && r.Method == "GET":
			handleGetFieldReport(w, r, parts[1])
		case parts[0] == "field-reports" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateFieldReport(w, r, parts[1])
		case parts[0] == "field-reports" && len(parts) == 2 && r.Method == "DELETE":
			handleDeleteFieldReport(w, r, parts[1])
		case parts[0] == "field-reports" && len(parts) == 3 && parts[2] == "create-ncr" && r.Method == "POST":
			handleFieldReportCreateNCR(w, r, parts[1])

		default:
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	})

	// Serve React frontend (SPA with fallback to index.html)
	frontendDir := "frontend/dist"
	frontendFS := http.Dir(frontendDir)
	fileServer := http.FileServer(frontendFS)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly (JS, CSS, images, etc.)
		path := r.URL.Path
		if path != "/" {
			// Check if file exists in frontend/dist
			if f, err := fs.Stat(os.DirFS(frontendDir), strings.TrimPrefix(path, "/")); err == nil && !f.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// SPA fallback: serve index.html for all unmatched routes
		http.ServeFile(w, r, frontendDir+"/index.html")
	})

	// Top-level mux: health check bypasses auth
	root := http.NewServeMux()
	root.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	root.Handle("/", logging(requireAuth(requireRBAC(mux))))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("ZRP server starting on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, root))
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"gitplm_ui_url": gitplmUIURL})
}

func jsonResp(w http.ResponseWriter, data interface{}) {
	json.NewEncoder(w).Encode(APIResponse{Data: data})
}

func jsonRespMeta(w http.ResponseWriter, data interface{}, total, page, limit int) {
	json.NewEncoder(w).Encode(APIResponse{Data: data, Meta: &Meta{Total: total, Page: page, Limit: limit}})
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func decodeBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
