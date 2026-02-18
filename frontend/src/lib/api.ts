// API client with TypeScript types for ZRP backend

const API_BASE = '/api/v1';

// Common types
export interface ApiResponse<T> {
  data: T;
  error?: string;
}

export interface Part {
  ipn: string;
  category: string;
  description: string;
  cost?: number;
  price?: number;
  lead_time?: number;
  minimum_stock?: number;
  current_stock?: number;
  location?: string;
  vendor?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ECO {
  id: string;
  title: string;
  description: string;
  reason: string;
  status: string;
  created_by: string;
  created_at: string;
  approved_by?: string;
  approved_at?: string;
  implemented_at?: string;
}

export interface WorkOrder {
  id: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  assigned_to?: string;
  due_date?: string;
  created_at: string;
  updated_at: string;
}

export interface Vendor {
  id: string;
  name: string;
  contact_email?: string;
  contact_phone?: string;
  address?: string;
  status: string;
  created_at: string;
}

export interface PurchaseOrder {
  id: string;
  vendor_id: string;
  status: string;
  total_amount: number;
  created_at: string;
  delivery_date?: string;
}

export interface InventoryItem {
  ipn: string;
  current_stock: number;
  minimum_stock: number;
  location: string;
  last_updated: string;
}

export interface DashboardStats {
  total_parts: number;
  low_stock_alerts: number;
  active_work_orders: number;
  pending_ecos: number;
  total_inventory_value: number;
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
      throw new Error(`API error: ${response.statusText}`);
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
  async getParts(): Promise<Part[]> {
    return this.request('/parts');
  }

  async getPart(ipn: string): Promise<Part> {
    return this.request(`/parts/${ipn}`);
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
  async getECOs(): Promise<ECO[]> {
    return this.request('/ecos');
  }

  async getECO(id: string): Promise<ECO> {
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

  async approveECO(id: string): Promise<void> {
    return this.request(`/ecos/${id}/approve`, {
      method: 'POST',
    });
  }

  async implementECO(id: string): Promise<void> {
    return this.request(`/ecos/${id}/implement`, {
      method: 'POST',
    });
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
  async getInventory(): Promise<InventoryItem[]> {
    return this.request('/inventory');
  }

  async getInventoryItem(ipn: string): Promise<InventoryItem> {
    return this.request(`/inventory/${ipn}`);
  }

  async getInventoryHistory(ipn: string): Promise<any[]> {
    return this.request(`/inventory/${ipn}/history`);
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
}

// Export singleton instance
export const api = new ApiClient();