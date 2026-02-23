package main

import (
	"net/http"
)

func handleListProductPricing(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().ListProductPricing(w, r)
}

func handleGetProductPricing(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().GetProductPricing(w, r, id)
}

func handleCreateProductPricing(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().CreateProductPricing(w, r)
}

func handleUpdateProductPricing(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().UpdateProductPricing(w, r, id)
}

func handleDeleteProductPricing(w http.ResponseWriter, r *http.Request, id string) {
	getSalesHandler().DeleteProductPricing(w, r, id)
}

func handleListCostAnalysis(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().ListCostAnalysis(w, r)
}

func handleCreateCostAnalysis(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().CreateCostAnalysis(w, r)
}

func handleProductPricingHistory(w http.ResponseWriter, r *http.Request, ipn string) {
	getSalesHandler().ProductPricingHistory(w, r, ipn)
}

func handleBulkUpdateProductPricing(w http.ResponseWriter, r *http.Request) {
	getSalesHandler().BulkUpdateProductPricing(w, r)
}

