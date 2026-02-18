import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockPOs, mockVendors, mockParts } from "../test/mocks";

const mockPOsWithLines = [
  { ...mockPOs[0], lines: [
    { id: 1, po_id: "PO-001", ipn: "IPN-001", mpn: "R10K", manufacturer: "Yageo", qty_ordered: 100, qty_received: 0, unit_price: 0.01, notes: "" },
    { id: 2, po_id: "PO-001", ipn: "IPN-002", mpn: "C100U", manufacturer: "Murata", qty_ordered: 50, qty_received: 0, unit_price: 0.10, notes: "" },
  ]},
  { ...mockPOs[1], lines: [] },
];

const mockGetPurchaseOrders = vi.fn().mockResolvedValue(mockPOsWithLines);
const mockGetVendors = vi.fn().mockResolvedValue(mockVendors);
const mockGetParts = vi.fn().mockResolvedValue(mockParts);
const mockCreatePurchaseOrder = vi.fn().mockResolvedValue(mockPOs[0]);

vi.mock("../lib/api", () => ({
  api: {
    getPurchaseOrders: (...args: any[]) => mockGetPurchaseOrders(...args),
    getVendors: (...args: any[]) => mockGetVendors(...args),
    getParts: (...args: any[]) => mockGetParts(...args),
    createPurchaseOrder: (...args: any[]) => mockCreatePurchaseOrder(...args),
  },
}));

import Procurement from "./Procurement";

beforeEach(() => vi.clearAllMocks());

describe("Procurement", () => {
  it("renders loading state", () => {
    render(<Procurement />);
    expect(screen.getByText("Loading purchase orders...")).toBeInTheDocument();
  });

  it("renders PO list after loading", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    expect(screen.getByText("PO-002")).toBeInTheDocument();
  });

  it("shows page title and description", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("Procurement")).toBeInTheDocument();
      expect(screen.getByText("Manage purchase orders and vendor relationships.")).toBeInTheDocument();
    });
  });

  it("has create PO button", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("Create PO")).toBeInTheDocument();
    });
  });

  it("shows summary cards", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("Total POs")).toBeInTheDocument();
      expect(screen.getByText("Draft")).toBeInTheDocument();
      expect(screen.getByText("Pending")).toBeInTheDocument();
      expect(screen.getByText("Received")).toBeInTheDocument();
    });
  });

  it("shows table headers", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO Number")).toBeInTheDocument();
      expect(screen.getByText("Vendor")).toBeInTheDocument();
      expect(screen.getByText("Total")).toBeInTheDocument();
      expect(screen.getByText("Created")).toBeInTheDocument();
      expect(screen.getByText("Expected")).toBeInTheDocument();
    });
  });

  it("shows vendor names in table", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
      expect(screen.getByText("DigiParts")).toBeInTheDocument();
    });
  });

  it("calculates total amount from line items", async () => {
    render(<Procurement />);
    await waitFor(() => {
      // PO-001: 100*0.01 + 50*0.10 = 1.00 + 5.00 = $6.00
      expect(screen.getByText("$6.00")).toBeInTheDocument();
      // PO-002: no lines = $0.00
      expect(screen.getByText("$0.00")).toBeInTheDocument();
    });
  });

  it("shows status badges", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("DRAFT")).toBeInTheDocument();
      expect(screen.getByText("SENT")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetPurchaseOrders.mockResolvedValueOnce([]);
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText(/no purchase orders found/i)).toBeInTheDocument();
    });
  });

  it("opens create PO dialog", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
      expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
    });
  });

  it("create dialog has line items section", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Line Items")).toBeInTheDocument();
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });
  });

  it("can add and remove line items in create dialog", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });
    // Initially 1 line, Remove button disabled
    const removeButtons = screen.getAllByText("Remove");
    expect(removeButtons[0]).toBeDisabled();

    // Add another line
    fireEvent.click(screen.getByText("Add Item"));
    await waitFor(() => {
      const newRemoveButtons = screen.getAllByText("Remove");
      expect(newRemoveButtons.length).toBe(2);
      expect(newRemoveButtons[0]).not.toBeDisabled();
    });
  });

  it("create PO button disabled when vendor not selected", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
    });
    const createButtons = screen.getAllByText("Create PO");
    const submitButton = createButtons[createButtons.length - 1];
    expect(submitButton).toBeDisabled();
  });

  it("cancel button in create dialog works", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create PO"));
    await waitFor(() => {
      expect(screen.getByText("Create Purchase Order")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
  });

  it("has View Details links for each PO", async () => {
    render(<Procurement />);
    await waitFor(() => {
      const viewButtons = screen.getAllByText("View Details");
      expect(viewButtons.length).toBe(2);
    });
  });

  it("calls all APIs on mount", async () => {
    render(<Procurement />);
    await waitFor(() => {
      expect(mockGetPurchaseOrders).toHaveBeenCalled();
      expect(mockGetVendors).toHaveBeenCalled();
      expect(mockGetParts).toHaveBeenCalled();
    });
  });

  it("handles API error gracefully", async () => {
    mockGetPurchaseOrders.mockRejectedValueOnce(new Error("Network error"));
    render(<Procurement />);
    await waitFor(() => {
      expect(screen.getByText(/no purchase orders found/i)).toBeInTheDocument();
    });
  });
});
