package main

import (
	"encoding/json"
	"flag"
	"fmt"
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

func main() {
	pmDir := flag.String("pmDir", "", "Path to gitplm parts database directory")
	port := flag.Int("port", 9000, "HTTP port")
	dbPath := flag.String("db", "zrp.db", "SQLite database path")
	gitplmUI := flag.String("gitplm-ui", "http://localhost:8888", "gitplm-ui base URL")
	flag.Parse()

	partsDir = *pmDir
	gitplmUIURL = *gitplmUI

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

	// Start background notification generator (run once after short delay, then every 5 min)
	go func() {
		time.Sleep(5 * time.Second)
		generateNotifications()
		emailNotificationsForRecent()
		for {
			time.Sleep(5 * time.Minute)
			generateNotifications()
			emailNotificationsForRecent()
		}
	}()

	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFile(w, r, "static/index.html")
			return
		}
		// Try static file first
		http.ServeFile(w, r, "static/index.html")
	})

	// File serving
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

	// API routes - using a simple router
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
		path = strings.TrimSuffix(path, "/")
		parts := strings.Split(path, "/")

		switch {
		// Global Search
		case parts[0] == "search" && len(parts) == 1 && r.Method == "GET":
			handleGlobalSearch(w, r)

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
		case parts[0] == "parts" && len(parts) == 2 && r.Method == "GET":
			handleGetPart(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 3 && parts[2] == "bom" && r.Method == "GET":
			handlePartBOM(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 3 && parts[2] == "cost" && r.Method == "GET":
			handlePartCost(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 2 && r.Method == "PUT":
			handleUpdatePart(w, r, parts[1])
		case parts[0] == "parts" && len(parts) == 2 && r.Method == "DELETE":
			handleDeletePart(w, r, parts[1])

		// Categories
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
		case parts[0] == "pos" && len(parts) == 2 && parts[1] == "generate-from-wo" && r.Method == "POST":
			handleGeneratePOFromWO(w, r)
		case parts[0] == "pos" && len(parts) == 2 && r.Method == "GET":
			handleGetPO(w, r, parts[1])
		case parts[0] == "pos" && len(parts) == 2 && r.Method == "PUT":
			handleUpdatePO(w, r, parts[1])
		case parts[0] == "pos" && len(parts) == 3 && parts[2] == "receive" && r.Method == "POST":
			handleReceivePO(w, r, parts[1])

		// Work Orders
		case parts[0] == "workorders" && len(parts) == 2 && parts[1] == "bulk" && r.Method == "POST":
			handleBulkWorkOrders(w, r)
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
			handleGetTests(w, r, parts[1])

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

		// Devices
		case parts[0] == "devices" && len(parts) == 2 && parts[1] == "bulk" && r.Method == "POST":
			handleBulkDevices(w, r)
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

		// API Keys
		case parts[0] == "apikeys" && len(parts) == 1 && r.Method == "GET":
			handleListAPIKeys(w, r)
		case parts[0] == "apikeys" && len(parts) == 1 && r.Method == "POST":
			handleCreateAPIKey(w, r)
		case parts[0] == "apikeys" && len(parts) == 2 && r.Method == "DELETE":
			handleDeleteAPIKey(w, r, parts[1])
		case parts[0] == "apikeys" && len(parts) == 2 && r.Method == "PUT":
			handleToggleAPIKey(w, r, parts[1])

		// Users
		case parts[0] == "users" && len(parts) == 1 && r.Method == "GET":
			handleListUsers(w, r)
		case parts[0] == "users" && len(parts) == 1 && r.Method == "POST":
			handleCreateUser(w, r)
		case parts[0] == "users" && len(parts) == 2 && r.Method == "PUT":
			handleUpdateUser(w, r, parts[1])
		case parts[0] == "users" && len(parts) == 3 && parts[2] == "password" && r.Method == "PUT":
			handleResetPassword(w, r, parts[1])

		// Attachments
		case parts[0] == "attachments" && len(parts) == 1 && r.Method == "POST":
			handleUploadAttachment(w, r)
		case parts[0] == "attachments" && len(parts) == 1 && r.Method == "GET":
			handleListAttachments(w, r)
		case parts[0] == "attachments" && len(parts) == 2 && r.Method == "DELETE":
			handleDeleteAttachment(w, r, parts[1])

		// Prices
		case parts[0] == "prices" && len(parts) == 2 && parts[1] != "" && r.Method == "GET":
			// Check if it's a numeric ID (delete) or IPN (list)
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
		case parts[0] == "email-log" && len(parts) == 1 && r.Method == "GET":
			handleListEmailLog(w, r)

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

		default:
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("ZRP server starting on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, logging(requireAuth(mux))))
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
