// API client with TypeScript types for ZRP backend

const API_BASE = '/api/v1';

// Common types
export interface ApiResponse<T> {
  data: T;
  meta?: {
    total: number;
    page: number;
    limit: number;
  };
  error?: string;
}

export interface Part {
  ipn: string;
  category?: string;
  description?: string;
  cost?: number;
  price?: number;
  lead_time?: number;
  minimum_stock?: number;
  current_stock?: number;
  location?: string;
  vendor?: string;
  status?: string;
  created_at?: string;
  updated_at?: string;
  fields?: Record<string, string>;
}

export interface Category {
  id: string;
  name: string;
  count: number;
  columns: string[];
}

export interface BOMNode {
  ipn: string;
  description: string;
  qty?: number;
  ref?: string;
  children: BOMNode[];
}

export interface ReceivingInspection {
  id: number;
  po_id: string;
  po_line_id: number;
  ipn: string;
  qty_received: number;
  qty_passed: number;
  qty_failed: number;
  qty_on_hold: number;
  inspector: string;
  inspected_at?: string;
  notes: string;
  created_at: string;
}

export interface WhereUsedEntry {
  assembly_ipn: string;
  description: string;
  qty: number;
  ref: string;
}

export interface PartCost {
  ipn: string;
  last_unit_price?: number;
  po_id?: string;
  last_ordered?: string;
  bom_cost?: number;
}

// Document types for upload
export interface Document {
  id: string;
  title: string;
  category: string;
  ipn: string;
  revision: string;
  status: string;
  content: string;
  file_path: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface Attachment {
  id: string;
  module: string;
  record_id: string;
  filename: string;
  original_name: string;
  size_bytes: number;
  mime_type: string;
  uploaded_by: string;
  created_at: string;
}

export interface ECORevision {
  id: number;
  eco_id: string;
  revision: string;
  status: string;
  changes_summary: string;
  created_by: string;
  created_at: string;
  approved_by?: string;
  approved_at?: string;
  implemented_by?: string;
  implemented_at?: string;
  effectivity_date?: string;
  notes: string;
}

export interface ECO {
  id: string;
  title: string;
  description: string;
  reason: string;
  status: string;
  priority?: string;
  affected_ipns?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  approved_by?: string;
  approved_at?: string;
  implemented_at?: string;
}

export interface WorkOrder {
  id: string;
  assembly_ipn: string;
  qty: number;
  status: string;
  priority: string;
  notes?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface Vendor {
  id: string;
  name: string;
  website?: string;
  contact_name?: string;
  contact_email?: string;
  contact_phone?: string;
  notes?: string;
  status: string;
  lead_time_days: number;
  created_at: string;
}

export interface PurchaseOrder {
  id: string;
  vendor_id: string;
  status: string;
  notes?: string;
  created_at: string;
  expected_date?: string;
  received_at?: string;
  lines?: POLine[];
}

export interface POLine {
  id: number;
  po_id: string;
  ipn: string;
  mpn?: string;
  manufacturer?: string;
  qty_ordered: number;
  qty_received: number;
  unit_price?: number;
  notes?: string;
}

export interface InventoryItem {
  ipn: string;
  qty_on_hand: number;
  qty_reserved: number;
  location?: string;
  reorder_point: number;
  reorder_qty: number;
  description?: string;
  mpn?: string;
  updated_at: string;
}

export interface InventoryTransaction {
  id: number;
  ipn: string;
  type: string;
  qty: number;
  reference?: string;
  notes?: string;
  created_at: string;
}

export interface NCR {
  id: string;
  title: string;
  description: string;
  ipn: string;
  serial_number: string;
  defect_type: string;
  severity: string;
  status: string;
  root_cause: string;
  corrective_action: string;
  created_at: string;
  resolved_at?: string;
}

export interface RMA {
  id: string;
  serial_number: string;
  customer: string;
  reason: string;
  status: string;
  defect_description: string;
  resolution: string;
  created_at: string;
  received_at?: string;
  resolved_at?: string;
}

export interface Shipment {
  id: string;
  type: string;
  status: string;
  tracking_number: string;
  carrier: string;
  ship_date?: string;
  delivery_date?: string;
  from_address: string;
  to_address: string;
  notes: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  lines?: ShipmentLine[];
}

export interface ShipmentLine {
  id: number;
  shipment_id: string;
  ipn: string;
  serial_number: string;
  qty: number;
  work_order_id: string;
  rma_id: string;
}

export interface PackList {
  id: number;
  shipment_id: string;
  created_at: string;
  lines?: ShipmentLine[];
}

export interface TestRecord {
  id: number;
  serial_number: string;
  ipn: string;
  firmware_version: string;
  test_type: string;
  result: string;
  measurements: string;
  notes: string;
  tested_by: string;
  tested_at: string;
}

export interface Device {
  serial_number: string;
  ipn: string;
  firmware_version: string;
  customer: string;
  location: string;
  status: string;
  install_date: string;
  last_seen?: string;
  notes: string;
  created_at: string;
}

export interface FirmwareCampaign {
  id: string;
  name: string;
  version: string;
  category: string;
  status: string;
  target_filter: string;
  notes: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface CampaignDevice {
  campaign_id: string;
  serial_number: string;
  status: string;
  updated_at?: string;
}

// RFQ types
export interface RFQ {
  id: string;
  title: string;
  status: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  due_date: string;
  notes: string;
  lines?: RFQLine[];
  vendors?: RFQVendor[];
  quotes?: RFQQuote[];
}

export interface RFQLine {
  id: number;
  rfq_id: string;
  ipn: string;
  description: string;
  qty: number;
  unit: string;
}

export interface RFQVendor {
  id: number;
  rfq_id: string;
  vendor_id: string;
  vendor_name?: string;
  status: string;
  quoted_at: string;
  notes: string;
}

export interface RFQQuote {
  id: number;
  rfq_id: string;
  rfq_vendor_id: number;
  rfq_line_id: number;
  unit_price: number;
  lead_time_days: number;
  moq: number;
  notes: string;
}

export interface RFQDashboard {
  open_rfqs: number;
  pending_responses: number;
  awarded_this_month: number;
  rfqs: {
    id: string;
    title: string;
    status: string;
    due_date: string;
    vendor_count: number;
    response_count: number;
    line_count: number;
    total_quoted_value: number;
  }[];
}

export interface RFQCompare {
  lines: RFQLine[];
  vendors: RFQVendor[];
  matrix: Record<number, Record<number, { unit_price: number; lead_time_days: number; moq: number; notes: string }>>;
}

export interface Quote {
  id: string;
  customer: string;
  status: string;
  notes: string;
  created_at: string;
  valid_until: string;
  accepted_at?: string;
  lines?: QuoteLine[];
}

export interface QuoteLine {
  id: number;
  quote_id: string;
  ipn: string;
  description: string;
  qty: number;
  unit_price: number;
  notes: string;
}

export interface DashboardStats {
  total_parts: number;
  low_stock_alerts: number;
  active_work_orders: number;
  pending_ecos: number;
  total_inventory_value: number;
}

export interface CalendarEvent {
  date: string;
  type: string;
  id: string;
  title: string;
  color: string;
}

export interface AuditLogEntry {
  id: string;
  timestamp: string;
  user: string;
  action: string;
  entity_type: string;
  entity_id: string;
  details: string;
  ip_address?: string;
}

export interface User {
  id: string;
  username: string;
  email: string;
  role: 'admin' | 'user' | 'readonly';
  status: 'active' | 'inactive';
  last_login?: string;
  created_at: string;
}

export interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  full_key?: string;
  status: 'active' | 'revoked';
  created_at: string;
  last_used?: string;
  created_by: string;
}

export interface UndoEntry {
  id: number;
  user_id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  previous_data: string;
  created_at: string;
  expires_at: string;
}

export interface ChangeEntry {
  id: number;
  table_name: string;
  record_id: string;
  operation: string;
  old_data: string;
  new_data: string;
  user_id: string;
  created_at: string;
  undone: number;
}

export interface EmailConfig {
  enabled: boolean;
  smtp_host: string;
  smtp_port: number;
  smtp_security: 'none' | 'tls' | 'ssl';
  smtp_username: string;
  smtp_password: string;
  from_address: string;
  from_name: string;
}

export interface EmailLogEntry {
  id: number;
  to_address: string;
  subject: string;
  body: string;
  event_type: string;
  status: string;
  error: string;
  sent_at: string;
}

// API client class
class ApiClient {
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${API_BASE}${endpoint}`;
    
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      if (response.status === 401 && !endpoint.includes('/auth/')) {
        // Session expired — redirect to login
        window.location.href = '/login';
        throw new Error('Session expired');
      }
      const body = await response.json().catch(() => ({ error: response.statusText }));
      throw new Error(body.error || `API error: ${response.statusText}`);
    }

    return response.json();
  }

  // Dashboard
  async getDashboard(): Promise<DashboardStats> {
    return this.request('/dashboard');
  }

  async getDashboardCharts(): Promise<any> {
    return this.request('/dashboard/charts');
  }

  async getLowStockAlerts(): Promise<InventoryItem[]> {
    return this.request('/dashboard/lowstock');
  }

  // Global search
  async globalSearch(query: string): Promise<any> {
    return this.request(`/search?q=${encodeURIComponent(query)}`);
  }

  // Parts
  async getParts(params?: { 
    category?: string; 
    q?: string; 
    page?: number; 
    limit?: number; 
  }): Promise<ApiResponse<Part[]>> {
    const searchParams = new URLSearchParams();
    if (params?.category) searchParams.set('category', params.category);
    if (params?.q) searchParams.set('q', params.q);
    if (params?.page) searchParams.set('page', params.page.toString());
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    
    const url = `/parts${searchParams.toString() ? `?${searchParams.toString()}` : ''}`;
    return this.request(url);
  }

  async getPart(ipn: string): Promise<Part> {
    return this.request(`/parts/${ipn}`);
  }

  async getCategories(): Promise<Category[]> {
    return this.request('/parts/categories');
  }

  async getPartBOM(ipn: string): Promise<BOMNode> {
    return this.request(`/parts/${ipn}/bom`);
  }

  async getPartCost(ipn: string): Promise<PartCost> {
    return this.request(`/parts/${ipn}/cost`);
  }

  async createPart(part: Partial<Part>): Promise<Part> {
    return this.request('/parts', {
      method: 'POST',
      body: JSON.stringify(part),
    });
  }

  async updatePart(ipn: string, part: Partial<Part>): Promise<Part> {
    return this.request(`/parts/${ipn}`, {
      method: 'PUT',
      body: JSON.stringify(part),
    });
  }

  async deletePart(ipn: string): Promise<void> {
    return this.request(`/parts/${ipn}`, {
      method: 'DELETE',
    });
  }

  // ECOs
  async getECOs(status?: string): Promise<ECO[]> {
    const url = `/ecos${status ? `?status=${status}` : ''}`;
    return this.request(url);
  }

  async getECO(id: string): Promise<ECO & { affected_parts?: any[] }> {
    return this.request(`/ecos/${id}`);
  }

  async createECO(eco: Partial<ECO>): Promise<ECO> {
    return this.request('/ecos', {
      method: 'POST',
      body: JSON.stringify(eco),
    });
  }

  async updateECO(id: string, eco: Partial<ECO>): Promise<ECO> {
    return this.request(`/ecos/${id}`, {
      method: 'PUT',
      body: JSON.stringify(eco),
    });
  }

  async approveECO(id: string): Promise<ECO> {
    return this.request(`/ecos/${id}/approve`, {
      method: 'POST',
    });
  }

  async implementECO(id: string): Promise<ECO> {
    return this.request(`/ecos/${id}/implement`, {
      method: 'POST',
    });
  }

  async rejectECO(id: string): Promise<ECO> {
    return this.request(`/ecos/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ status: 'rejected' }),
    });
  }

  // ECO Revisions
  async getECORevisions(ecoId: string): Promise<ECORevision[]> {
    return this.request(`/ecos/${ecoId}/revisions`);
  }

  async createECORevision(ecoId: string, data: { changes_summary: string; effectivity_date?: string; notes?: string }): Promise<ECORevision> {
    return this.request(`/ecos/${ecoId}/revisions`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getECORevision(ecoId: string, rev: string): Promise<ECORevision> {
    return this.request(`/ecos/${ecoId}/revisions/${rev}`);
  }

  // Work Orders
  async getWorkOrders(): Promise<WorkOrder[]> {
    return this.request('/workorders');
  }

  async getWorkOrder(id: string): Promise<WorkOrder> {
    return this.request(`/workorders/${id}`);
  }

  async createWorkOrder(workOrder: Partial<WorkOrder>): Promise<WorkOrder> {
    return this.request('/workorders', {
      method: 'POST',
      body: JSON.stringify(workOrder),
    });
  }

  async updateWorkOrder(id: string, workOrder: Partial<WorkOrder>): Promise<WorkOrder> {
    return this.request(`/workorders/${id}`, {
      method: 'PUT',
      body: JSON.stringify(workOrder),
    });
  }

  async getWorkOrderBOM(id: string): Promise<{
    wo_id: string;
    assembly_ipn: string;
    qty: number;
    bom: Array<{
      ipn: string;
      description: string;
      qty_required: number;
      qty_on_hand: number;
      shortage: number;
      status: string;
    }>;
  }> {
    return this.request(`/workorders/${id}/bom`);
  }

  // Vendors
  async getVendors(): Promise<Vendor[]> {
    return this.request('/vendors');
  }

  async getVendor(id: string): Promise<Vendor> {
    return this.request(`/vendors/${id}`);
  }

  async createVendor(vendor: Partial<Vendor>): Promise<Vendor> {
    return this.request('/vendors', {
      method: 'POST',
      body: JSON.stringify(vendor),
    });
  }

  async updateVendor(id: string, vendor: Partial<Vendor>): Promise<Vendor> {
    return this.request(`/vendors/${id}`, {
      method: 'PUT',
      body: JSON.stringify(vendor),
    });
  }

  async deleteVendor(id: string): Promise<void> {
    return this.request(`/vendors/${id}`, {
      method: 'DELETE',
    });
  }

  // Inventory
  async getInventory(lowStock?: boolean): Promise<InventoryItem[]> {
    const query = lowStock ? '?low_stock=true' : '';
    return this.request(`/inventory${query}`);
  }

  async getInventoryItem(ipn: string): Promise<InventoryItem> {
    return this.request(`/inventory/${ipn}`);
  }

  async getInventoryHistory(ipn: string): Promise<InventoryTransaction[]> {
    return this.request(`/inventory/${ipn}/history`);
  }

  async createInventoryTransaction(transaction: {
    ipn: string;
    type: string;
    qty: number;
    reference?: string;
    notes?: string;
  }): Promise<void> {
    return this.request('/inventory/transact', {
      method: 'POST',
      body: JSON.stringify(transaction),
    });
  }

  async bulkDeleteInventory(ipns: string[]): Promise<void> {
    return this.request('/inventory/bulk-delete', {
      method: 'DELETE',
      body: JSON.stringify({ ipns }),
    });
  }

  async bulkUpdateInventory(ids: string[], updates: Record<string, string>): Promise<{ success: number; failed: number; errors: string[] }> {
    return this.request('/inventory/bulk-update', {
      method: 'POST',
      body: JSON.stringify({ ids, updates }),
    });
  }

  async bulkUpdateWorkOrders(ids: string[], updates: Record<string, string>): Promise<{ success: number; failed: number; errors: string[] }> {
    return this.request('/workorders/bulk-update', {
      method: 'POST',
      body: JSON.stringify({ ids, updates }),
    });
  }

  async bulkUpdateDevices(ids: string[], updates: Record<string, string>): Promise<{ success: number; failed: number; errors: string[] }> {
    return this.request('/devices/bulk-update', {
      method: 'POST',
      body: JSON.stringify({ ids, updates }),
    });
  }

  // Purchase Orders
  async getPurchaseOrders(): Promise<PurchaseOrder[]> {
    return this.request('/pos');
  }

  async getPurchaseOrder(id: string): Promise<PurchaseOrder> {
    return this.request(`/pos/${id}`);
  }

  async createPurchaseOrder(po: Partial<PurchaseOrder>): Promise<PurchaseOrder> {
    return this.request('/pos', {
      method: 'POST',
      body: JSON.stringify(po),
    });
  }

  async updatePurchaseOrder(id: string, po: Partial<PurchaseOrder>): Promise<PurchaseOrder> {
    return this.request(`/pos/${id}`, {
      method: 'PUT',
      body: JSON.stringify(po),
    });
  }

  async receivePurchaseOrder(id: string, lines: { id: number; qty: number }[], skipInspection?: boolean): Promise<PurchaseOrder> {
    return this.request(`/pos/${id}/receive`, {
      method: 'POST',
      body: JSON.stringify({ lines, skip_inspection: skipInspection }),
    });
  }

  // Receiving/Inspection
  async getReceivingInspections(status?: string): Promise<ReceivingInspection[]> {
    const url = `/receiving${status ? `?status=${status}` : ''}`;
    return this.request(url);
  }

  async inspectReceiving(id: number, data: { qty_passed: number; qty_failed: number; qty_on_hold: number; inspector?: string; notes?: string }): Promise<ReceivingInspection> {
    return this.request(`/receiving/${id}/inspect`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Where-used
  async getPartWhereUsed(ipn: string): Promise<WhereUsedEntry[]> {
    return this.request(`/parts/${ipn}/where-used`);
  }

  async generatePOFromWorkOrder(woId: string, vendorId: string): Promise<{ po_id: string; lines: number }> {
    return this.request('/pos/generate', {
      method: 'POST',
      body: JSON.stringify({ wo_id: woId, vendor_id: vendorId }),
    });
  }

  // NCRs
  async getNCRs(): Promise<NCR[]> {
    return this.request('/ncrs');
  }

  async getNCR(id: string): Promise<NCR> {
    return this.request(`/ncrs/${id}`);
  }

  async createNCR(ncr: Partial<NCR>): Promise<NCR> {
    return this.request('/ncrs', {
      method: 'POST',
      body: JSON.stringify(ncr),
    });
  }

  async updateNCR(id: string, ncr: Partial<NCR> & { create_eco?: boolean }): Promise<NCR> {
    return this.request(`/ncrs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(ncr),
    });
  }

  // RMAs
  async getRMAs(): Promise<RMA[]> {
    return this.request('/rmas');
  }

  async getRMA(id: string): Promise<RMA> {
    return this.request(`/rmas/${id}`);
  }

  async createRMA(rma: Partial<RMA>): Promise<RMA> {
    return this.request('/rmas', {
      method: 'POST',
      body: JSON.stringify(rma),
    });
  }

  async updateRMA(id: string, rma: Partial<RMA>): Promise<RMA> {
    return this.request(`/rmas/${id}`, {
      method: 'PUT',
      body: JSON.stringify(rma),
    });
  }

  // Shipments
  async getShipments(): Promise<Shipment[]> {
    return this.request('/shipments');
  }

  async getShipment(id: string): Promise<Shipment> {
    return this.request(`/shipments/${id}`);
  }

  async createShipment(shipment: Partial<Shipment>): Promise<Shipment> {
    return this.request('/shipments', {
      method: 'POST',
      body: JSON.stringify(shipment),
    });
  }

  async updateShipment(id: string, shipment: Partial<Shipment>): Promise<Shipment> {
    return this.request(`/shipments/${id}`, {
      method: 'PUT',
      body: JSON.stringify(shipment),
    });
  }

  async shipShipment(id: string, trackingNumber: string, carrier: string): Promise<Shipment> {
    return this.request(`/shipments/${id}/ship`, {
      method: 'POST',
      body: JSON.stringify({ tracking_number: trackingNumber, carrier }),
    });
  }

  async deliverShipment(id: string): Promise<Shipment> {
    return this.request(`/shipments/${id}/deliver`, {
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  async getShipmentPackList(id: string): Promise<PackList> {
    return this.request(`/shipments/${id}/pack-list`);
  }

  // Testing
  async getTestRecords(): Promise<TestRecord[]> {
    return this.request('/testing');
  }

  async getTestRecord(id: number): Promise<TestRecord> {
    return this.request(`/testing/${id}`);
  }

  async createTestRecord(testRecord: Partial<TestRecord>): Promise<TestRecord> {
    return this.request('/testing', {
      method: 'POST',
      body: JSON.stringify(testRecord),
    });
  }

  // Devices
  async getDevices(): Promise<Device[]> {
    return this.request('/devices');
  }

  async getDevice(serialNumber: string): Promise<Device> {
    return this.request(`/devices/${serialNumber}`);
  }

  async createDevice(device: Partial<Device>): Promise<Device> {
    return this.request('/devices', {
      method: 'POST',
      body: JSON.stringify(device),
    });
  }

  async updateDevice(serialNumber: string, device: Partial<Device>): Promise<Device> {
    return this.request(`/devices/${serialNumber}`, {
      method: 'PUT',
      body: JSON.stringify(device),
    });
  }

  async importDevices(file: File): Promise<{ success: number; errors: string[] }> {
    const formData = new FormData();
    formData.append('file', file);
    
    const response = await fetch(`${API_BASE}/devices/import`, {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      throw new Error(`Import failed: ${response.statusText}`);
    }

    return response.json();
  }

  async exportDevices(): Promise<Blob> {
    const response = await fetch(`${API_BASE}/devices/export`, {
      method: 'GET',
    });

    if (!response.ok) {
      throw new Error(`Export failed: ${response.statusText}`);
    }

    return response.blob();
  }

  // Firmware Campaigns
  async getFirmwareCampaigns(): Promise<FirmwareCampaign[]> {
    return this.request('/firmware');
  }

  async getFirmwareCampaign(id: string): Promise<FirmwareCampaign> {
    return this.request(`/firmware/${id}`);
  }

  async createFirmwareCampaign(campaign: Partial<FirmwareCampaign>): Promise<FirmwareCampaign> {
    return this.request('/firmware', {
      method: 'POST',
      body: JSON.stringify(campaign),
    });
  }

  async updateFirmwareCampaign(id: string, campaign: Partial<FirmwareCampaign>): Promise<FirmwareCampaign> {
    return this.request(`/firmware/${id}`, {
      method: 'PUT',
      body: JSON.stringify(campaign),
    });
  }

  async getCampaignDevices(campaignId: string): Promise<CampaignDevice[]> {
    return this.request(`/firmware/${campaignId}/devices`);
  }

  // Quotes
  async getQuotes(): Promise<Quote[]> {
    return this.request('/quotes');
  }

  async getQuote(id: string): Promise<Quote> {
    return this.request(`/quotes/${id}`);
  }

  async createQuote(quote: Partial<Quote>): Promise<Quote> {
    return this.request('/quotes', {
      method: 'POST',
      body: JSON.stringify(quote),
    });
  }

  async updateQuote(id: string, quote: Partial<Quote>): Promise<Quote> {
    return this.request(`/quotes/${id}`, {
      method: 'PUT',
      body: JSON.stringify(quote),
    });
  }

  async exportQuotePDF(id: string): Promise<Blob> {
    const response = await fetch(`${API_BASE}/quotes/${id}/pdf`, {
      method: 'GET',
    });

    if (!response.ok) {
      throw new Error(`PDF export failed: ${response.statusText}`);
    }

    return response.blob();
  }

  // Documents
  async getDocuments(): Promise<(Document & { attachment_count?: number })[]> {
    return this.request('/docs');
  }

  async getDocument(id: string): Promise<Document & { attachments?: Attachment[] }> {
    return this.request(`/docs/${id}`);
  }

  async createDocument(doc: Partial<Document>): Promise<Document> {
    return this.request('/docs', {
      method: 'POST',
      body: JSON.stringify(doc),
    });
  }

  async updateDocument(id: string, doc: Partial<Document>): Promise<Document> {
    return this.request(`/docs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(doc),
    });
  }

  async uploadAttachment(file: File, module: string, recordId: string): Promise<Attachment> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('module', module);
    formData.append('record_id', recordId);
    
    const response = await fetch(`${API_BASE}/attachments`, {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      throw new Error(`Upload failed: ${response.statusText}`);
    }

    return response.json();
  }

  async downloadAttachment(id: string): Promise<Blob> {
    const response = await fetch(`${API_BASE}/attachments/${id}/download`);
    if (!response.ok) {
      throw new Error(`Download failed: ${response.statusText}`);
    }
    return response.blob();
  }

  // Calendar
  async getCalendarEvents(year: number, month: number): Promise<CalendarEvent[]> {
    return this.request(`/calendar?year=${year}&month=${month}`);
  }

  // Audit Log
  async getAuditLogs(params?: {
    search?: string;
    entityType?: string;
    user?: string;
    page?: number;
    limit?: number;
  }): Promise<{ entries: AuditLogEntry[]; total: number }> {
    const queryParams = new URLSearchParams();
    if (params?.search) queryParams.append('search', params.search);
    if (params?.entityType) queryParams.append('entity_type', params.entityType);
    if (params?.user) queryParams.append('user', params.user);
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.limit) queryParams.append('limit', params.limit.toString());

    return this.request(`/audit?${queryParams.toString()}`);
  }

  // Users
  async getUsers(): Promise<User[]> {
    return this.request('/users');
  }

  async getUser(id: string): Promise<User> {
    return this.request(`/users/${id}`);
  }

  async createUser(user: {
    username: string;
    email: string;
    password: string;
    role: string;
  }): Promise<User> {
    return this.request('/users', {
      method: 'POST',
      body: JSON.stringify(user),
    });
  }

  async updateUser(id: string, user: Partial<User>): Promise<User> {
    return this.request(`/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(user),
    });
  }

  async deleteUser(id: string): Promise<void> {
    return this.request(`/users/${id}`, {
      method: 'DELETE',
    });
  }

  // API Keys
  async getAPIKeys(): Promise<APIKey[]> {
    return this.request('/api-keys');
  }

  async createAPIKey(name: string): Promise<APIKey> {
    return this.request('/api-keys', {
      method: 'POST',
      body: JSON.stringify({ name }),
    });
  }

  async revokeAPIKey(id: string): Promise<void> {
    return this.request(`/api-keys/${id}/revoke`, {
      method: 'POST',
    });
  }

  // GitPLM Settings
  async getGitPLMConfig(): Promise<{ base_url: string }> {
    return this.request('/settings/gitplm');
  }

  async updateGitPLMConfig(config: { base_url: string }): Promise<{ base_url: string }> {
    return this.request('/settings/gitplm', {
      method: 'PUT',
      body: JSON.stringify(config),
    });
  }

  async getGitPLMURL(ipn: string): Promise<{ url: string; configured: boolean }> {
    return this.request(`/parts/${encodeURIComponent(ipn)}/gitplm-url`);
  }

  // Email Settings
  async getEmailConfig(): Promise<EmailConfig> {
    return this.request('/email/config');
  }

  async updateEmailConfig(config: EmailConfig): Promise<EmailConfig> {
    return this.request('/email/config', {
      method: 'PUT',
      body: JSON.stringify(config),
    });
  }

  async testEmail(testEmail: string): Promise<{ success: boolean; message: string }> {
    return this.request('/email/test', {
      method: 'POST',
      body: JSON.stringify({ test_email: testEmail }),
    });
  }

  // Email subscriptions
  async getEmailSubscriptions(): Promise<Record<string, boolean>> {
    return this.request('/email/subscriptions');
  }

  async updateEmailSubscriptions(subs: Record<string, boolean>): Promise<Record<string, boolean>> {
    return this.request('/email/subscriptions', {
      method: 'PUT',
      body: JSON.stringify(subs),
    });
  }

  // Email log
  async getEmailLog(): Promise<EmailLogEntry[]> {
    return this.request('/email-log');
  }

  // Undo
  async getUndoList(limit?: number): Promise<UndoEntry[]> {
    const params = limit ? `?limit=${limit}` : '';
    return this.request(`/undo${params}`);
  }

  async performUndo(id: number): Promise<{ status: string; entity_type: string; entity_id: string }> {
    return this.request(`/undo/${id}`, { method: 'POST' });
  }

  // Change History
  async getRecentChanges(limit?: number): Promise<ChangeEntry[]> {
    const params = limit ? `?limit=${limit}` : '';
    return this.request(`/changes/recent${params}`);
  }

  async undoChange(id: number): Promise<{ status: string; table_name: string; record_id: string; operation: string; redo_id: number }> {
    return this.request(`/changes/${id}`, { method: 'POST' });
  }

  // Backups
  async getBackups(): Promise<{ filename: string; size: number; created_at: string }[]> {
    const resp: any = await this.request('/admin/backups');
    return resp.data || resp;
  }

  async createBackup(): Promise<void> {
    return this.request('/admin/backup', { method: 'POST' });
  }

  async deleteBackup(filename: string): Promise<void> {
    return this.request(`/admin/backups/${filename}`, { method: 'DELETE' });
  }

  async restoreBackup(filename: string): Promise<void> {
    return this.request('/admin/restore', {
      method: 'POST',
      body: JSON.stringify({ filename }),
    });
  }

  // Auth
  async login(username: string, password: string): Promise<{ user: { id: number; username: string; display_name: string; role: string } }> {
    const response = await fetch('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({ error: 'Login failed' }));
      throw new Error(body.error || 'Login failed');
    }
    return response.json();
  }

  async logout(): Promise<void> {
    await fetch('/auth/logout', { method: 'POST' });
  }

  async getMe(): Promise<{ user: { id: number; username: string; display_name: string; role: string } } | null> {
    const response = await fetch('/auth/me');
    if (!response.ok) return null;
    return response.json();
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    const response = await fetch('/auth/change-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({ error: 'Failed to change password' }));
      throw new Error(body.error || 'Failed to change password');
    }
  }

  // RFQs
  async getRFQs(): Promise<RFQ[]> {
    return this.request('/rfqs');
  }

  async getRFQ(id: string): Promise<RFQ> {
    return this.request(`/rfqs/${id}`);
  }

  async createRFQ(rfq: Partial<RFQ>): Promise<RFQ> {
    return this.request('/rfqs', {
      method: 'POST',
      body: JSON.stringify(rfq),
    });
  }

  async updateRFQ(id: string, rfq: Partial<RFQ>): Promise<RFQ> {
    return this.request(`/rfqs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(rfq),
    });
  }

  async deleteRFQ(id: string): Promise<void> {
    return this.request(`/rfqs/${id}`, {
      method: 'DELETE',
    });
  }

  async sendRFQ(id: string): Promise<RFQ> {
    return this.request(`/rfqs/${id}/send`, {
      method: 'POST',
    });
  }

  async awardRFQ(id: string, vendorId: string): Promise<{ status: string; po_id: string }> {
    return this.request(`/rfqs/${id}/award`, {
      method: 'POST',
      body: JSON.stringify({ vendor_id: vendorId }),
    });
  }

  async compareRFQ(id: string): Promise<RFQCompare> {
    return this.request(`/rfqs/${id}/compare`);
  }

  async createRFQQuote(rfqId: string, quote: Partial<RFQQuote>): Promise<RFQQuote> {
    return this.request(`/rfqs/${rfqId}/quotes`, {
      method: 'POST',
      body: JSON.stringify(quote),
    });
  }

  async updateRFQQuote(rfqId: string, quoteId: number, quote: Partial<RFQQuote>): Promise<{ status: string }> {
    return this.request(`/rfqs/${rfqId}/quotes/${quoteId}`, {
      method: 'PUT',
      body: JSON.stringify(quote),
    });
  }

  async closeRFQ(id: string): Promise<RFQ> {
    return this.request(`/rfqs/${id}/close`, { method: 'POST' });
  }

  async getRFQEmailBody(id: string): Promise<{ subject: string; body: string }> {
    return this.request(`/rfqs/${id}/email`);
  }

  async awardRFQPerLine(id: string, awards: { line_id: number; vendor_id: string }[]): Promise<{ status: string; po_ids: string[] }> {
    return this.request(`/rfqs/${id}/award-lines`, {
      method: 'POST',
      body: JSON.stringify({ awards }),
    });
  }

  async getRFQDashboard(): Promise<RFQDashboard> {
    return this.request('/rfq-dashboard');
  }

  // Market Pricing
  async getMarketPricing(partIPN: string, refresh = false): Promise<MarketPricingResponse> {
    const qs = refresh ? '?refresh=true' : '';
    return this.request(`/parts/${partIPN}/market-pricing${qs}`);
  }

  async updateDigikeySettings(settings: { client_id: string; client_secret: string }): Promise<{ status: string }> {
    return this.request('/settings/digikey', { method: 'POST', body: JSON.stringify(settings) });
  }

  async updateMouserSettings(settings: { api_key: string }): Promise<{ status: string }> {
    return this.request('/settings/mouser', { method: 'POST', body: JSON.stringify(settings) });
  }

  async getDistributorSettings(): Promise<DistributorSettings> {
    return this.request('/settings/distributors');
  }
}

// Market Pricing types
export interface PriceBreak {
  qty: number;
  unit_price: number;
}

export interface MarketPricingResult {
  id: number;
  part_ipn: string;
  mpn: string;
  distributor: string;
  distributor_pn: string;
  manufacturer: string;
  description: string;
  stock_qty: number;
  lead_time_days: number;
  currency: string;
  price_breaks: PriceBreak[];
  product_url: string;
  datasheet_url: string;
  fetched_at: string;
}

export interface MarketPricingResponse {
  results: MarketPricingResult[];
  cached: boolean;
  error?: string;
  unconfigured?: string[];
  errors?: string[];
}

export interface DistributorSettings {
  digikey: { client_id: string; client_secret: string };
  mouser: { api_key: string };
}

// Export singleton instance
export const api = new ApiClient();