package main

import "zrp/internal/models"

// Type aliases for backward compatibility during migration.
// These allow all existing handler code and tests to continue using
// the unqualified type names while the actual definitions live in internal/models.

type APIResponse = models.APIResponse
type Meta = models.Meta
type ECO = models.ECO
type ECORevision = models.ECORevision
type Document = models.Document
type Vendor = models.Vendor
type InventoryItem = models.InventoryItem
type InventoryTransaction = models.InventoryTransaction
type PurchaseOrder = models.PurchaseOrder
type POLine = models.POLine
type WorkOrder = models.WorkOrder
type WOSerial = models.WOSerial
type TestRecord = models.TestRecord
type FieldReport = models.FieldReport
type NCR = models.NCR
type Device = models.Device
type FirmwareCampaign = models.FirmwareCampaign
type CampaignDevice = models.CampaignDevice
type RMA = models.RMA
type Quote = models.Quote
type QuoteLine = models.QuoteLine
type DashboardData = models.DashboardData
type Part = models.Part
type Category = models.Category
type Shipment = models.Shipment
type ShipmentLine = models.ShipmentLine
type PackList = models.PackList
type RFQ = models.RFQ
type RFQLine = models.RFQLine
type RFQVendor = models.RFQVendor
type RFQQuote = models.RFQQuote
type DocumentVersion = models.DocumentVersion
type SalesOrder = models.SalesOrder
type SalesOrderLine = models.SalesOrderLine
type Invoice = models.Invoice
type InvoiceLine = models.InvoiceLine
type PriceHistory = models.PriceHistory
type PriceTrendPoint = models.PriceTrendPoint
type ReceivingInspection = models.ReceivingInspection
type ProductPricing = models.ProductPricing
type CostAnalysis = models.CostAnalysis
type CostAnalysisWithPricing = models.CostAnalysisWithPricing
type BulkPriceUpdate = models.BulkPriceUpdate
