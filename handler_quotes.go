package main

import (
	"net/http"

	"zrp/internal/handlers/sales"
	"zrp/internal/websocket"
)

// salesHandler is the shared sales handler instance.
var salesHandler *sales.Handler

// internalHub is the shared internal websocket Hub used by extracted handlers.
var internalHub *websocket.Hub

// getInternalHub returns the shared internal websocket Hub, lazily creating it.
func getInternalHub() *websocket.Hub {
	if internalHub == nil {
		internalHub = websocket.NewHub()
	}
	return internalHub
}

// getSalesHandler returns the sales handler, lazily initializing if needed (for tests).
func getSalesHandler() *sales.Handler {
	if salesHandler == nil || salesHandler.DB != db {
		salesHandler = &sales.Handler{
			DB:                 db,
			Hub:                getInternalHub(),
			NextID:             nextID,
			RecordChangeJSON:   recordChangeJSON,
			GetQuoteSnapshot:   getQuoteSnapshot,
			GenerateInvoiceNum: generateInvoiceNumber,
			CompanyName:        companyName,
			CompanyEmail:       companyEmail,
		}
	}
	return salesHandler
}

func handleListQuotes(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().ListQuotes(w, r)
}

func handleGetQuote(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().GetQuote(w, r, id)
}

func handleCreateQuote(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().CreateQuote(w, r)
}

func handleUpdateQuote(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().UpdateQuote(w, r, id)
}

func handleQuoteCost(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().QuoteCost(w, r, id)
}

func handleQuotePDF(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().QuotePDF(w, r, id)
}
