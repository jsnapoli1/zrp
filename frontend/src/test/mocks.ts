import { vi } from "vitest";
import type {
  Part, Category, ECO, WorkOrder, Vendor, PurchaseOrder,
  InventoryItem, NCR, RMA, Shipment, PackList, TestRecord, Device, FirmwareCampaign,
  Quote, DashboardStats, CalendarEvent, AuditLogEntry, User,
  APIKey, EmailConfig, Document, RFQ,
} from "../lib/api";

export const mockDashboardStats: DashboardStats & Record<string, number> = {
  total_parts: 150,
  low_stock_alerts: 5,
  active_work_orders: 12,
  pending_ecos: 3,
  total_inventory_value: 250000,
  open_ecos: 3,
  open_pos: 7,
  open_ncrs: 2,
  total_devices: 45,
  open_rmas: 1,
};

export const mockParts: Part[] = [
  { ipn: "IPN-001", category: "Resistors", description: "10k Resistor", cost: 0.01, price: 0.05, status: "active", created_at: "2024-01-01", updated_at: "2024-01-01" },
  { ipn: "IPN-002", category: "Capacitors", description: "100uF Cap", cost: 0.10, price: 0.50, status: "active", created_at: "2024-01-02", updated_at: "2024-01-02" },
  { ipn: "IPN-003", category: "ICs", description: "MCU STM32", cost: 5.00, price: 15.00, status: "active", created_at: "2024-01-03", updated_at: "2024-01-03" },
];

export const mockCategories: Category[] = [
  { id: "resistors", name: "Resistors", count: 50, columns: ["resistance", "tolerance"] },
  { id: "capacitors", name: "Capacitors", count: 30, columns: ["capacitance", "voltage"] },
  { id: "ics", name: "ICs", count: 20, columns: ["package", "manufacturer"] },
];

export const mockECOs: ECO[] = [
  { id: "ECO-001", title: "Update resistor spec", description: "Change tolerance", reason: "Quality", status: "draft", created_by: "admin", created_at: "2024-01-10", updated_at: "2024-01-10" },
  { id: "ECO-002", title: "Replace MCU", description: "New MCU revision", reason: "EOL", status: "approved", created_by: "admin", created_at: "2024-01-12", updated_at: "2024-01-15", approved_by: "manager", approved_at: "2024-01-15" },
  { id: "ECO-003", title: "BOM update", description: "Update BOM costs", reason: "Cost reduction", status: "open", created_by: "admin", created_at: "2024-01-20", updated_at: "2024-01-20" },
];

export const mockWorkOrders: WorkOrder[] = [
  { id: "WO-001", assembly_ipn: "IPN-003", qty: 10, status: "open", priority: "medium", created_at: "2024-01-15" },
  { id: "WO-002", assembly_ipn: "IPN-003", qty: 5, status: "in_progress", priority: "high", created_at: "2024-01-10", started_at: "2024-01-11" },
  { id: "WO-003", assembly_ipn: "IPN-001", qty: 100, status: "completed", priority: "low", created_at: "2024-01-01", completed_at: "2024-01-05" },
];

export const mockVendors: Vendor[] = [
  { id: "V-001", name: "Acme Corp", website: "https://acme.com", contact_name: "John", contact_email: "john@acme.com", status: "active", lead_time_days: 14, created_at: "2024-01-01" },
  { id: "V-002", name: "DigiParts", website: "https://digiparts.com", contact_name: "Jane", contact_email: "jane@digi.com", status: "active", lead_time_days: 7, created_at: "2024-01-02" },
];

export const mockPOs: PurchaseOrder[] = [
  { id: "PO-001", vendor_id: "V-001", status: "draft", created_at: "2024-01-20", lines: [] },
  { id: "PO-002", vendor_id: "V-002", status: "sent", created_at: "2024-01-18", expected_date: "2024-02-01", lines: [] },
];

export const mockInventory: InventoryItem[] = [
  { ipn: "IPN-001", qty_on_hand: 500, qty_reserved: 50, location: "Bin A1", reorder_point: 100, reorder_qty: 500, description: "10k Resistor", updated_at: "2024-01-20" },
  { ipn: "IPN-002", qty_on_hand: 20, qty_reserved: 5, location: "Bin B2", reorder_point: 50, reorder_qty: 200, description: "100uF Cap", updated_at: "2024-01-19" },
  { ipn: "IPN-003", qty_on_hand: 10, qty_reserved: 0, location: "Shelf C", reorder_point: 5, reorder_qty: 20, description: "MCU STM32", updated_at: "2024-01-21" },
];

export const mockNCRs: NCR[] = [
  { id: "NCR-001", title: "Defective resistor batch", description: "Out of tolerance", ipn: "IPN-001", serial_number: "SN-100", defect_type: "tolerance", severity: "major", status: "open", root_cause: "", corrective_action: "", created_at: "2024-01-18" },
  { id: "NCR-002", title: "Cracked capacitor", description: "Physical damage", ipn: "IPN-002", serial_number: "SN-200", defect_type: "physical", severity: "critical", status: "resolved", root_cause: "Shipping damage", corrective_action: "New packaging", created_at: "2024-01-10", resolved_at: "2024-01-15" },
];

export const mockRMAs: RMA[] = [
  { id: "RMA-001", serial_number: "SN-500", customer: "Acme Inc", reason: "Device not working", status: "received", defect_description: "No power", resolution: "", created_at: "2024-01-22" },
  { id: "RMA-002", serial_number: "SN-600", customer: "Tech Co", reason: "Wrong firmware", status: "resolved", defect_description: "Incorrect FW version", resolution: "Reflashed", created_at: "2024-01-15", resolved_at: "2024-01-20" },
];

export const mockShipments: Shipment[] = [
  { id: "SHP-2024-0001", type: "outbound", status: "draft", tracking_number: "", carrier: "FedEx", from_address: "123 Main St", to_address: "456 Oak Ave", notes: "Test shipment", created_by: "admin", created_at: "2024-01-25", updated_at: "2024-01-25", lines: [{ id: 1, shipment_id: "SHP-2024-0001", ipn: "IPN-001", serial_number: "", qty: 5, work_order_id: "", rma_id: "" }] },
  { id: "SHP-2024-0002", type: "inbound", status: "shipped", tracking_number: "1Z999", carrier: "UPS", from_address: "Vendor HQ", to_address: "Our Warehouse", notes: "", created_by: "admin", created_at: "2024-01-20", updated_at: "2024-01-22", ship_date: "2024-01-22", lines: [] },
];

export const mockPackList: PackList = {
  id: 1, shipment_id: "SHP-2024-0001", created_at: "2024-01-25",
  lines: [{ id: 1, shipment_id: "SHP-2024-0001", ipn: "IPN-001", serial_number: "", qty: 5, work_order_id: "", rma_id: "" }],
};

export const mockTestRecords: TestRecord[] = [
  { id: 1, serial_number: "SN-100", ipn: "IPN-003", firmware_version: "1.0.0", test_type: "functional", result: "pass", measurements: "{}", notes: "", tested_by: "tech1", tested_at: "2024-01-20" },
  { id: 2, serial_number: "SN-101", ipn: "IPN-003", firmware_version: "1.0.0", test_type: "burn-in", result: "fail", measurements: "{}", notes: "Failed at 4hr mark", tested_by: "tech2", tested_at: "2024-01-21" },
];

export const mockDevices: Device[] = [
  { serial_number: "SN-100", ipn: "IPN-003", firmware_version: "1.0.0", customer: "Acme", location: "Building A", status: "active", install_date: "2024-01-01", notes: "", created_at: "2024-01-01" },
  { serial_number: "SN-101", ipn: "IPN-003", firmware_version: "0.9.0", customer: "Tech Co", location: "Floor 2", status: "active", install_date: "2023-12-15", notes: "", created_at: "2023-12-15" },
];

export const mockFirmwareCampaigns: FirmwareCampaign[] = [
  { id: "FW-001", name: "Update to v1.1", version: "1.1.0", category: "ICs", status: "active", target_filter: "", notes: "", created_at: "2024-01-25", started_at: "2024-01-26" },
  { id: "FW-002", name: "Security patch", version: "1.0.1", category: "ICs", status: "completed", target_filter: "", notes: "", created_at: "2024-01-10", completed_at: "2024-01-20" },
];

export const mockQuotes: Quote[] = [
  { id: "Q-001", customer: "Acme Inc", status: "draft", notes: "", created_at: "2024-01-25", valid_until: "2024-02-25" },
  { id: "Q-002", customer: "Tech Co", status: "sent", notes: "Urgent", created_at: "2024-01-20", valid_until: "2024-02-20" },
];

export const mockCalendarEvents: CalendarEvent[] = [
  { date: "2024-01-15", type: "eco", id: "ECO-001", title: "Update resistor spec", color: "blue" },
  { date: "2024-01-20", type: "po", id: "PO-001", title: "PO-001 Created", color: "green" },
];

export const mockAuditLogs: AuditLogEntry[] = [
  { id: "AL-001", timestamp: "2024-01-25T10:00:00Z", user: "admin", action: "create", entity_type: "part", entity_id: "IPN-001", details: "Created part" },
  { id: "AL-002", timestamp: "2024-01-25T11:00:00Z", user: "admin", action: "update", entity_type: "eco", entity_id: "ECO-001", details: "Updated status" },
];

export const mockUsers: User[] = [
  { id: "U-001", username: "admin", email: "admin@zrp.com", role: "admin", status: "active", created_at: "2024-01-01" },
  { id: "U-002", username: "user1", email: "user1@zrp.com", role: "user", status: "active", created_at: "2024-01-05" },
];

export const mockAPIKeys: APIKey[] = [
  { id: "AK-001", name: "Production API", key_prefix: "zrp_prod_", status: "active", created_at: "2024-01-01", created_by: "admin" },
  { id: "AK-002", name: "Test API", key_prefix: "zrp_test_", status: "revoked", created_at: "2024-01-10", created_by: "admin" },
];

export const mockEmailConfig: EmailConfig = {
  enabled: true,
  smtp_host: "smtp.example.com",
  smtp_port: 587,
  smtp_security: "tls",
  smtp_username: "zrp@example.com",
  smtp_password: "secret",
  from_address: "zrp@example.com",
  from_name: "ZRP System",
};

export const mockDocuments: Document[] = [
  { id: "DOC-001", title: "Assembly Guide", category: "procedure", ipn: "IPN-003", revision: "A", status: "released", content: "How to assemble...", file_path: "", created_by: "admin", created_at: "2024-01-05", updated_at: "2024-01-05" },
  { id: "DOC-002", title: "Test Procedure", category: "test", ipn: "IPN-003", revision: "B", status: "draft", content: "Testing steps...", file_path: "", created_by: "admin", created_at: "2024-01-10", updated_at: "2024-01-15" },
];

export const mockRFQs: RFQ[] = [
  { id: "RFQ-2026-0001", title: "Resistor Bulk Quote", status: "draft", created_by: "admin", created_at: "2026-01-15", updated_at: "2026-01-15", due_date: "2026-02-01", notes: "Need pricing for Q2" },
  { id: "RFQ-2026-0002", title: "MCU Sourcing", status: "sent", created_by: "admin", created_at: "2026-01-20", updated_at: "2026-01-21", due_date: "2026-02-15", notes: "" },
];

// Create a mock api object
export function createMockApi() {
  return {
    getDashboard: vi.fn().mockResolvedValue(mockDashboardStats),
    getDashboardCharts: vi.fn().mockResolvedValue({ charts: [] }),
    getLowStockAlerts: vi.fn().mockResolvedValue(mockInventory.filter(i => i.qty_on_hand <= i.reorder_point)),
    globalSearch: vi.fn().mockResolvedValue({ results: [] }),
    getParts: vi.fn().mockResolvedValue({ data: mockParts, meta: { total: 3, page: 1, limit: 50 } }),
    getPart: vi.fn().mockImplementation((ipn: string) => Promise.resolve(mockParts.find(p => p.ipn === ipn) || mockParts[0])),
    getCategories: vi.fn().mockResolvedValue(mockCategories),
    getPartBOM: vi.fn().mockResolvedValue({ ipn: "IPN-003", description: "MCU", qty: 1, children: [] }),
    getPartCost: vi.fn().mockResolvedValue({ ipn: "IPN-003", bom_cost: 5.0 }),
    createPart: vi.fn().mockResolvedValue(mockParts[0]),
    updatePart: vi.fn().mockResolvedValue(mockParts[0]),
    deletePart: vi.fn().mockResolvedValue(undefined),
    getECOs: vi.fn().mockResolvedValue(mockECOs),
    getECO: vi.fn().mockResolvedValue({ ...mockECOs[0], affected_parts: [] }),
    createECO: vi.fn().mockResolvedValue(mockECOs[0]),
    updateECO: vi.fn().mockResolvedValue(mockECOs[0]),
    approveECO: vi.fn().mockResolvedValue({ ...mockECOs[0], status: "approved" }),
    implementECO: vi.fn().mockResolvedValue({ ...mockECOs[0], status: "implemented" }),
    rejectECO: vi.fn().mockResolvedValue({ ...mockECOs[0], status: "rejected" }),
    getWorkOrders: vi.fn().mockResolvedValue(mockWorkOrders),
    getWorkOrder: vi.fn().mockResolvedValue(mockWorkOrders[0]),
    createWorkOrder: vi.fn().mockResolvedValue(mockWorkOrders[0]),
    updateWorkOrder: vi.fn().mockResolvedValue(mockWorkOrders[0]),
    getWorkOrderBOM: vi.fn().mockResolvedValue({ wo_id: "WO-001", assembly_ipn: "IPN-003", qty: 10, bom: [] }),
    getVendors: vi.fn().mockResolvedValue(mockVendors),
    getVendor: vi.fn().mockResolvedValue(mockVendors[0]),
    createVendor: vi.fn().mockResolvedValue(mockVendors[0]),
    updateVendor: vi.fn().mockResolvedValue(mockVendors[0]),
    deleteVendor: vi.fn().mockResolvedValue(undefined),
    getInventory: vi.fn().mockResolvedValue(mockInventory),
    getInventoryItem: vi.fn().mockResolvedValue(mockInventory[0]),
    getInventoryHistory: vi.fn().mockResolvedValue([]),
    createInventoryTransaction: vi.fn().mockResolvedValue(undefined),
    bulkDeleteInventory: vi.fn().mockResolvedValue(undefined),
    getPurchaseOrders: vi.fn().mockResolvedValue(mockPOs),
    getPurchaseOrder: vi.fn().mockResolvedValue(mockPOs[0]),
    createPurchaseOrder: vi.fn().mockResolvedValue(mockPOs[0]),
    updatePurchaseOrder: vi.fn().mockResolvedValue(mockPOs[0]),
    receivePurchaseOrder: vi.fn().mockResolvedValue(mockPOs[0]),
    generatePOFromWorkOrder: vi.fn().mockResolvedValue({ po_id: "PO-003", lines: 3 }),
    getNCRs: vi.fn().mockResolvedValue(mockNCRs),
    getNCR: vi.fn().mockResolvedValue(mockNCRs[0]),
    createNCR: vi.fn().mockResolvedValue(mockNCRs[0]),
    updateNCR: vi.fn().mockResolvedValue(mockNCRs[0]),
    getRMAs: vi.fn().mockResolvedValue(mockRMAs),
    getRMA: vi.fn().mockResolvedValue(mockRMAs[0]),
    createRMA: vi.fn().mockResolvedValue(mockRMAs[0]),
    updateRMA: vi.fn().mockResolvedValue(mockRMAs[0]),
    getShipments: vi.fn().mockResolvedValue(mockShipments),
    getShipment: vi.fn().mockResolvedValue(mockShipments[0]),
    createShipment: vi.fn().mockResolvedValue(mockShipments[0]),
    updateShipment: vi.fn().mockResolvedValue(mockShipments[0]),
    shipShipment: vi.fn().mockResolvedValue({ ...mockShipments[0], status: "shipped" }),
    deliverShipment: vi.fn().mockResolvedValue({ ...mockShipments[0], status: "delivered" }),
    getShipmentPackList: vi.fn().mockResolvedValue(mockPackList),
    getTestRecords: vi.fn().mockResolvedValue(mockTestRecords),
    getTestRecord: vi.fn().mockResolvedValue(mockTestRecords[0]),
    createTestRecord: vi.fn().mockResolvedValue(mockTestRecords[0]),
    getDevices: vi.fn().mockResolvedValue(mockDevices),
    getDevice: vi.fn().mockResolvedValue(mockDevices[0]),
    createDevice: vi.fn().mockResolvedValue(mockDevices[0]),
    updateDevice: vi.fn().mockResolvedValue(mockDevices[0]),
    importDevices: vi.fn().mockResolvedValue({ success: 5, errors: [] }),
    exportDevices: vi.fn().mockResolvedValue(new Blob()),
    getFirmwareCampaigns: vi.fn().mockResolvedValue(mockFirmwareCampaigns),
    getFirmwareCampaign: vi.fn().mockResolvedValue(mockFirmwareCampaigns[0]),
    createFirmwareCampaign: vi.fn().mockResolvedValue(mockFirmwareCampaigns[0]),
    updateFirmwareCampaign: vi.fn().mockResolvedValue(mockFirmwareCampaigns[0]),
    getCampaignDevices: vi.fn().mockResolvedValue([]),
    getQuotes: vi.fn().mockResolvedValue(mockQuotes),
    getQuote: vi.fn().mockResolvedValue({ ...mockQuotes[0], lines: [] }),
    createQuote: vi.fn().mockResolvedValue(mockQuotes[0]),
    updateQuote: vi.fn().mockResolvedValue(mockQuotes[0]),
    exportQuotePDF: vi.fn().mockResolvedValue(new Blob()),
    getDocuments: vi.fn().mockResolvedValue(mockDocuments),
    getDocument: vi.fn().mockResolvedValue({ ...mockDocuments[0], attachments: [] }),
    createDocument: vi.fn().mockResolvedValue(mockDocuments[0]),
    updateDocument: vi.fn().mockResolvedValue(mockDocuments[0]),
    uploadAttachment: vi.fn().mockResolvedValue({ id: "ATT-001", filename: "test.pdf" }),
    downloadAttachment: vi.fn().mockResolvedValue(new Blob()),
    getCalendarEvents: vi.fn().mockResolvedValue(mockCalendarEvents),
    getAuditLogs: vi.fn().mockResolvedValue({ entries: mockAuditLogs, total: 2 }),
    getUsers: vi.fn().mockResolvedValue(mockUsers),
    getUser: vi.fn().mockResolvedValue(mockUsers[0]),
    createUser: vi.fn().mockResolvedValue(mockUsers[0]),
    updateUser: vi.fn().mockResolvedValue(mockUsers[0]),
    deleteUser: vi.fn().mockResolvedValue(undefined),
    getAPIKeys: vi.fn().mockResolvedValue(mockAPIKeys),
    createAPIKey: vi.fn().mockResolvedValue({ ...mockAPIKeys[0], full_key: "zrp_prod_abc123" }),
    revokeAPIKey: vi.fn().mockResolvedValue(undefined),
    getEmailConfig: vi.fn().mockResolvedValue(mockEmailConfig),
    updateEmailConfig: vi.fn().mockResolvedValue(mockEmailConfig),
    testEmail: vi.fn().mockResolvedValue({ success: true, message: "Email sent" }),
    getRFQs: vi.fn().mockResolvedValue(mockRFQs),
    getRFQ: vi.fn().mockResolvedValue(mockRFQs[0]),
    createRFQ: vi.fn().mockResolvedValue(mockRFQs[0]),
    updateRFQ: vi.fn().mockResolvedValue(mockRFQs[0]),
    deleteRFQ: vi.fn().mockResolvedValue(undefined),
    sendRFQ: vi.fn().mockResolvedValue({ ...mockRFQs[0], status: "sent" }),
    awardRFQ: vi.fn().mockResolvedValue({ status: "awarded", po_id: "PO-2026-0001" }),
    compareRFQ: vi.fn().mockResolvedValue({ lines: [], vendors: [], matrix: {} }),
    createRFQQuote: vi.fn().mockResolvedValue({ id: 1, rfq_id: "RFQ-2026-0001", rfq_vendor_id: 1, rfq_line_id: 1, unit_price: 0.05, lead_time_days: 14, moq: 100, notes: "" }),
    updateRFQQuote: vi.fn().mockResolvedValue({ status: "updated" }),
    closeRFQ: vi.fn().mockResolvedValue({ ...mockRFQs[0], status: "closed" }),
    getRFQEmailBody: vi.fn().mockResolvedValue({ subject: "RFQ Email", body: "Dear Vendor..." }),
    awardRFQPerLine: vi.fn().mockResolvedValue({ status: "awarded", po_ids: ["PO-2026-0001"] }),
    getRFQDashboard: vi.fn().mockResolvedValue({ open_rfqs: 2, pending_responses: 3, awarded_this_month: 1, rfqs: [] }),
  };
}
