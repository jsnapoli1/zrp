import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockVendors } from "../test/mocks";

const mockPO = {
  id: "PO-001",
  vendor_id: "V-001",
  status: "submitted",
  notes: "Urgent order",
  created_at: "2024-01-20T10:00:00Z",
  expected_date: "2024-02-01",
  received_at: undefined,
  lines: [
    { id: 1, po_id: "PO-001", ipn: "IPN-001", mpn: "R10K", manufacturer: "Yageo", qty_ordered: 100, qty_received: 50, unit_price: 0.01, notes: "Batch 1" },
    { id: 2, po_id: "PO-001", ipn: "IPN-002", mpn: "C100U", manufacturer: "Murata", qty_ordered: 50, qty_received: 50, unit_price: 0.10, notes: "" },
  ],
};

const mockGetPurchaseOrder = vi.fn().mockResolvedValue(mockPO);
const mockGetVendor = vi.fn().mockResolvedValue(mockVendors[0]);
const mockUpdatePurchaseOrder = vi.fn().mockResolvedValue(mockPO);
const mockReceivePurchaseOrder = vi.fn().mockResolvedValue(mockPO);

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "PO-001" }),
  };
});

vi.mock("../lib/api", () => ({
  api: {
    getPurchaseOrder: (...args: any[]) => mockGetPurchaseOrder(...args),
    getVendor: (...args: any[]) => mockGetVendor(...args),
    updatePurchaseOrder: (...args: any[]) => mockUpdatePurchaseOrder(...args),
    receivePurchaseOrder: (...args: any[]) => mockReceivePurchaseOrder(...args),
  },
}));

import PODetail from "./PODetail";

beforeEach(() => vi.clearAllMocks());

describe("PODetail", () => {
  it("renders loading state", () => {
    render(<PODetail />);
    expect(screen.getByText("Loading purchase order...")).toBeInTheDocument();
  });

  it("renders PO details after loading", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
      expect(screen.getByText("Purchase Order Details")).toBeInTheDocument();
    });
  });

  it("displays vendor information", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Vendor Information")).toBeInTheDocument();
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
      expect(screen.getByText("john@acme.com")).toBeInTheDocument();
    });
  });

  it("shows status badge", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("SUBMITTED")).toBeInTheDocument();
    });
  });

  it("displays notes", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Urgent order")).toBeInTheDocument();
    });
  });

  it("shows line items table", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Line Items")).toBeInTheDocument();
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
      expect(screen.getByText("IPN-002")).toBeInTheDocument();
      expect(screen.getByText("R10K")).toBeInTheDocument();
      expect(screen.getByText("C100U")).toBeInTheDocument();
      expect(screen.getByText("Yageo")).toBeInTheDocument();
      expect(screen.getByText("Murata")).toBeInTheDocument();
    });
  });

  it("displays line item prices", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("$0.01")).toBeInTheDocument();
      expect(screen.getByText("$0.10")).toBeInTheDocument();
    });
  });

  it("shows order summary card", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Order Summary")).toBeInTheDocument();
      expect(screen.getByText("Line Items:")).toBeInTheDocument();
      expect(screen.getByText("Total Ordered:")).toBeInTheDocument();
      expect(screen.getByText("Total Received:")).toBeInTheDocument();
      expect(screen.getByText("Total Amount:")).toBeInTheDocument();
    });
  });

  it("calculates total amount correctly", async () => {
    render(<PODetail />);
    await waitFor(() => {
      // 100*0.01 + 50*0.10 = 1.00 + 5.00 = $6.00
      expect(screen.getByText("$6.00")).toBeInTheDocument();
    });
  });

  it("shows Receive Items button for submitted PO with pending items", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Receive Items")).toBeInTheDocument();
    });
  });

  it("opens receive items dialog", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Receive Items")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Receive Items"));
    await waitFor(() => {
      // Dialog shows items with pending qty
      expect(screen.getByText("Ordered")).toBeInTheDocument();
      expect(screen.getByText("Received")).toBeInTheDocument();
      expect(screen.getByText("Pending")).toBeInTheDocument();
    });
  });

  it("shows Change Status button for non-closed PO", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Change Status")).toBeInTheDocument();
    });
  });

  it("opens change status dialog", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Change Status")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Change Status"));
    await waitFor(() => {
      expect(screen.getByText("Change PO Status")).toBeInTheDocument();
      expect(screen.getByText("Current Status")).toBeInTheDocument();
    });
  });

  it("does not show Change Status for closed PO", async () => {
    mockGetPurchaseOrder.mockResolvedValueOnce({ ...mockPO, status: "closed" });
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    expect(screen.queryByText("Change Status")).not.toBeInTheDocument();
  });

  it("does not show Receive Items for draft PO", async () => {
    mockGetPurchaseOrder.mockResolvedValueOnce({ ...mockPO, status: "draft" });
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    expect(screen.queryByText("Receive Items")).not.toBeInTheDocument();
  });

  it("does not show Receive Items when all items received", async () => {
    mockGetPurchaseOrder.mockResolvedValueOnce({
      ...mockPO,
      lines: mockPO.lines.map(l => ({ ...l, qty_received: l.qty_ordered })),
    });
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    expect(screen.queryByText("Receive Items")).not.toBeInTheDocument();
  });

  it("shows back to procurement link", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Back to Procurement")).toBeInTheDocument();
    });
  });

  it("shows not found state when PO is null", async () => {
    mockGetPurchaseOrder.mockRejectedValueOnce(new Error("Not found"));
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Purchase Order Not Found")).toBeInTheDocument();
    });
  });

  it("shows empty line items state", async () => {
    mockGetPurchaseOrder.mockResolvedValueOnce({ ...mockPO, lines: [] });
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("No line items found")).toBeInTheDocument();
    });
  });

  it("displays expected date", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Expected")).toBeInTheDocument();
    });
  });

  it("displays Status & Dates card", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Status & Dates")).toBeInTheDocument();
    });
  });

  it("shows lead time from vendor", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Lead Time")).toBeInTheDocument();
      expect(screen.getByText("14 days")).toBeInTheDocument();
    });
  });

  it("shows line item notes", async () => {
    render(<PODetail />);
    await waitFor(() => {
      expect(screen.getByText("Batch 1")).toBeInTheDocument();
    });
  });

  it("shows line total calculations", async () => {
    render(<PODetail />);
    await waitFor(() => {
      // Line 1: 100 * 0.01 = $1.00
      expect(screen.getByText("$1.00")).toBeInTheDocument();
      // Line 2: 50 * 0.10 = $5.00
      expect(screen.getByText("$5.00")).toBeInTheDocument();
    });
  });
});
