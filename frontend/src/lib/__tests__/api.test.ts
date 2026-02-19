import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

// We need to test the ApiClient class directly, so we import the singleton
// and mock fetch globally
const originalFetch = global.fetch;

let mockFetch: ReturnType<typeof vi.fn>;

beforeEach(() => {
  mockFetch = vi.fn();
  global.fetch = mockFetch;
});

afterEach(() => {
  global.fetch = originalFetch;
  vi.restoreAllMocks();
});

// Import after setup so fetch is available
import { api } from "../api";

describe("ApiClient", () => {
  describe("request() helper (via public methods)", () => {
    it("constructs correct URL with /api/v1 prefix", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ data: [] }),
      });

      await api.getParts();

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/api/v1/parts"),
        expect.any(Object)
      );
    });

    it("sets Content-Type: application/json header", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve([]),
      });

      await api.getECOs();

      const [, options] = mockFetch.mock.calls[0];
      expect(options.headers["Content-Type"]).toBe("application/json");
    });

    it("throws on non-ok response", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        statusText: "Not Found",
        json: () => Promise.reject(new Error("no body")),
      });

      await expect(api.getDashboard()).rejects.toThrow("Not Found");
    });

    it("parses JSON response", async () => {
      const mockData = { total_parts: 5, low_stock_alerts: 1, active_work_orders: 2, pending_ecos: 0, total_inventory_value: 1000 };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockData),
      });

      const result = await api.getDashboard();
      expect(result).toEqual(mockData);
    });

    it("sends POST with JSON body", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ id: "ECO-001" }),
      });

      await api.createECO({ title: "Test", description: "Desc" });

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/ecos");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({ title: "Test", description: "Desc" });
    });

    it("sends PUT with JSON body", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ id: "ECO-001" }),
      });

      await api.updateECO("ECO-001", { status: "approved" });

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/ecos/ECO-001");
      expect(options.method).toBe("PUT");
      expect(JSON.parse(options.body)).toEqual({ status: "approved" });
    });

    it("sends DELETE request", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(undefined),
      });

      await api.deletePart("IPN-001");

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/parts/IPN-001");
      expect(options.method).toBe("DELETE");
    });
  });

  describe("query parameter construction", () => {
    it("getParts() with all params", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ data: [], meta: { total: 0, page: 1, limit: 10 } }),
      });

      await api.getParts({ category: "Resistors", q: "10k", page: 2, limit: 25 });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain("category=Resistors");
      expect(url).toContain("q=10k");
      expect(url).toContain("page=2");
      expect(url).toContain("limit=25");
    });

    it("getParts() with no params omits query string", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ data: [] }),
      });

      await api.getParts();

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toBe("/api/v1/parts");
    });

    it("getParts() with partial params", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ data: [] }),
      });

      await api.getParts({ q: "cap" });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain("q=cap");
      expect(url).not.toContain("category=");
    });

    it("getAuditLogs() constructs query params", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ entries: [], total: 0 }),
      });

      await api.getAuditLogs({ search: "create", entityType: "part", user: "admin", page: 1, limit: 50 });

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain("search=create");
      expect(url).toContain("entity_type=part");
      expect(url).toContain("user=admin");
      expect(url).toContain("page=1");
      expect(url).toContain("limit=50");
    });

    it("getECOs() with status filter", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve([]),
      });

      await api.getECOs("approved");

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toBe("/api/v1/ecos?status=approved");
    });

    it("getECOs() without status filter", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve([]),
      });

      await api.getECOs();

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toBe("/api/v1/ecos");
    });

    it("getInventory() with lowStock flag", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve([]),
      });

      await api.getInventory(true);

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toBe("/api/v1/inventory?low_stock=true");
    });

    it("getCalendarEvents() with year and month", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve([]),
      });

      await api.getCalendarEvents(2024, 3);

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toBe("/api/v1/calendar?year=2024&month=3");
    });

    it("globalSearch() encodes query", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ results: [] }),
      });

      await api.globalSearch("hello world");

      const url = mockFetch.mock.calls[0][0] as string;
      expect(url).toContain("q=hello%20world");
    });
  });

  describe("file upload methods (FormData)", () => {
    it("importDevices() sends FormData without Content-Type header", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: 5, errors: [] }),
      });

      const file = new File(["csv,data"], "devices.csv", { type: "text/csv" });
      await api.importDevices(file);

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/devices/import");
      expect(options.method).toBe("POST");
      expect(options.body).toBeInstanceOf(FormData);
      // Should NOT have Content-Type header (browser sets it with boundary)
      expect(options.headers).toBeUndefined();
    });

    it("importDevices() throws on error response", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: "Bad Request",
      });

      const file = new File(["bad"], "bad.csv");
      await expect(api.importDevices(file)).rejects.toThrow("Import failed: Bad Request");
    });

    it("uploadAttachment() sends FormData with metadata fields", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ id: "ATT-001", filename: "doc.pdf" }),
      });

      const file = new File(["pdf"], "doc.pdf", { type: "application/pdf" });
      await api.uploadAttachment(file, "ecos", "ECO-001");

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/attachments");
      expect(options.method).toBe("POST");
      const formData = options.body as FormData;
      expect(formData.get("module")).toBe("ecos");
      expect(formData.get("record_id")).toBe("ECO-001");
      expect(formData.get("file")).toBeInstanceOf(File);
    });

    it("uploadAttachment() throws on error", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: "Payload Too Large",
      });

      const file = new File(["big"], "big.pdf");
      await expect(api.uploadAttachment(file, "docs", "D-1")).rejects.toThrow("Upload failed: Payload Too Large");
    });
  });

  describe("Blob return methods", () => {
    it("exportDevices() returns a Blob", async () => {
      const mockBlob = new Blob(["csv,data"], { type: "text/csv" });
      mockFetch.mockResolvedValue({
        ok: true,
        blob: () => Promise.resolve(mockBlob),
      });

      const result = await api.exportDevices();

      expect(result).toBeInstanceOf(Blob);
      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/devices/export");
      expect(options.method).toBe("GET");
    });

    it("exportDevices() throws on error", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: "Internal Server Error",
      });

      await expect(api.exportDevices()).rejects.toThrow("Export failed: Internal Server Error");
    });

    it("exportQuotePDF() returns a Blob", async () => {
      const mockBlob = new Blob(["%PDF"], { type: "application/pdf" });
      mockFetch.mockResolvedValue({
        ok: true,
        blob: () => Promise.resolve(mockBlob),
      });

      const result = await api.exportQuotePDF("Q-001");

      expect(result).toBeInstanceOf(Blob);
      expect(mockFetch.mock.calls[0][0]).toBe("/api/v1/quotes/Q-001/pdf");
    });

    it("exportQuotePDF() throws on error", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: "Not Found",
      });

      await expect(api.exportQuotePDF("Q-999")).rejects.toThrow("PDF export failed: Not Found");
    });

    it("downloadAttachment() returns a Blob", async () => {
      const mockBlob = new Blob(["file content"]);
      mockFetch.mockResolvedValue({
        ok: true,
        blob: () => Promise.resolve(mockBlob),
      });

      const result = await api.downloadAttachment("ATT-001");

      expect(result).toBeInstanceOf(Blob);
      expect(mockFetch.mock.calls[0][0]).toBe("/api/v1/attachments/ATT-001/download");
    });

    it("downloadAttachment() throws on error", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        statusText: "Forbidden",
      });

      await expect(api.downloadAttachment("ATT-001")).rejects.toThrow("Download failed: Forbidden");
    });
  });

  describe("specific API methods", () => {
    it("receivePurchaseOrder() sends lines payload", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ id: "PO-001" }),
      });

      await api.receivePurchaseOrder("PO-001", [
        { id: 1, qty: 50 },
        { id: 2, qty: 25 },
      ]);

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/pos/PO-001/receive");
      expect(options.method).toBe("POST");
      expect(JSON.parse(options.body)).toEqual({
        lines: [{ id: 1, qty: 50 }, { id: 2, qty: 25 }],
      });
    });

    it("generatePOFromWorkOrder() sends wo_id and vendor_id", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ po_id: "PO-003", lines: 2 }),
      });

      await api.generatePOFromWorkOrder("WO-001", "V-001");

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/pos/generate");
      expect(JSON.parse(options.body)).toEqual({ wo_id: "WO-001", vendor_id: "V-001" });
    });

    it("approveECO() sends POST to correct endpoint", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ id: "ECO-001", status: "approved" }),
      });

      await api.approveECO("ECO-001");

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/ecos/ECO-001/approve");
      expect(options.method).toBe("POST");
    });

    it("createAPIKey() sends name in body", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ id: "AK-001", name: "Test", full_key: "zrp_test_abc" }),
      });

      await api.createAPIKey("Test");

      expect(JSON.parse(mockFetch.mock.calls[0][1].body)).toEqual({ name: "Test" });
    });

    it("bulkDeleteInventory() sends ipns array", async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(undefined),
      });

      await api.bulkDeleteInventory(["IPN-001", "IPN-002"]);

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toBe("/api/v1/inventory/bulk-delete");
      expect(options.method).toBe("DELETE");
      expect(JSON.parse(options.body)).toEqual({ ipns: ["IPN-001", "IPN-002"] });
    });
  });
});
