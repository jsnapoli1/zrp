import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockWorkOrders, mockVendors } from "../test/mocks";

const mockWO = { ...mockWorkOrders[0], notes: "Test notes for WO" };
const mockBOM = {
  wo_id: "WO-001",
  assembly_ipn: "IPN-003",
  qty: 10,
  bom: [
    { ipn: "IPN-001", description: "10k Resistor", qty_required: 20, qty_on_hand: 500, shortage: 0, status: "ok" },
    { ipn: "IPN-002", description: "100uF Cap", qty_required: 50, qty_on_hand: 20, shortage: 30, status: "shortage" },
    { ipn: "IPN-004", description: "LED", qty_required: 10, qty_on_hand: 8, shortage: 2, status: "low" },
  ],
};

const mockGetWorkOrder = vi.fn().mockResolvedValue(mockWO);
const mockGetWorkOrderBOM = vi.fn().mockResolvedValue(mockBOM);
const mockGetVendors = vi.fn().mockResolvedValue(mockVendors);
const mockUpdateWorkOrder = vi.fn().mockResolvedValue(mockWO);
const mockGeneratePOFromWorkOrder = vi.fn().mockResolvedValue({ po_id: "PO-003", lines: 1 });

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "WO-001" }),
  };
});

vi.mock("../lib/api", () => ({
  api: {
    getWorkOrder: (...args: any[]) => mockGetWorkOrder(...args),
    getWorkOrderBOM: (...args: any[]) => mockGetWorkOrderBOM(...args),
    getVendors: (...args: any[]) => mockGetVendors(...args),
    updateWorkOrder: (...args: any[]) => mockUpdateWorkOrder(...args),
    generatePOFromWorkOrder: (...args: any[]) => mockGeneratePOFromWorkOrder(...args),
  },
}));

import WorkOrderDetail from "./WorkOrderDetail";

beforeEach(() => vi.clearAllMocks());

describe("WorkOrderDetail", () => {
  it("renders loading state", () => {
    render(<WorkOrderDetail />);
    expect(screen.getByText("Loading work order...")).toBeInTheDocument();
  });

  it("renders work order details after loading", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    expect(screen.getByText("Work Order Details")).toBeInTheDocument();
  });

  it("displays assembly IPN and quantity", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-003")).toBeInTheDocument();
    });
    // Quantity shown as large text in Assembly Information card
    const qtyEl = screen.getByText("10", { selector: ".text-2xl" });
    expect(qtyEl).toBeInTheDocument();
  });

  it("shows status and priority badges", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("OPEN")).toBeInTheDocument();
      expect(screen.getByText("MEDIUM")).toBeInTheDocument();
    });
  });

  it("displays notes when present", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Test notes for WO")).toBeInTheDocument();
    });
  });

  it("shows BOM vs Inventory Comparison table", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("BOM vs Inventory Comparison")).toBeInTheDocument();
    });
  });

  it("renders BOM items with correct data", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("10k Resistor")).toBeInTheDocument();
      expect(screen.getByText("100uF Cap")).toBeInTheDocument();
      expect(screen.getByText("LED")).toBeInTheDocument();
    });
  });

  it("highlights shortage items", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("SHORTAGE")).toBeInTheDocument();
      expect(screen.getByText("OK")).toBeInTheDocument();
      expect(screen.getByText("LOW")).toBeInTheDocument();
    });
  });

  it("shows shortage count in BOM Status card", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("BOM Status")).toBeInTheDocument();
      expect(screen.getByText("Shortages:")).toBeInTheDocument();
    });
  });

  it("shows Change Status button for open work order", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Change Status")).toBeInTheDocument();
    });
  });

  it("opens status change dialog", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Change Status")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Change Status"));
    await waitFor(() => {
      expect(screen.getByText("Change Work Order Status")).toBeInTheDocument();
      expect(screen.getByText("Current Status")).toBeInTheDocument();
      expect(screen.getByText("New Status")).toBeInTheDocument();
    });
  });

  it("shows Generate PO button when shortages exist", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Generate PO")).toBeInTheDocument();
    });
  });

  it("opens generate PO dialog", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Generate PO")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Generate PO"));
    await waitFor(() => {
      expect(screen.getByText("Generate Purchase Order")).toBeInTheDocument();
      expect(screen.getByText(/items have shortages/)).toBeInTheDocument();
      expect(screen.getByText("Select Vendor")).toBeInTheDocument();
    });
  });

  it("does not show Generate PO when no shortages", async () => {
    mockGetWorkOrderBOM.mockResolvedValueOnce({
      wo_id: "WO-001",
      assembly_ipn: "IPN-003",
      qty: 10,
      bom: [
        { ipn: "IPN-001", description: "10k Resistor", qty_required: 20, qty_on_hand: 500, shortage: 0, status: "ok" },
      ],
    });
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    expect(screen.queryByText("Generate PO")).not.toBeInTheDocument();
  });

  it("does not show Change Status for completed work order", async () => {
    mockGetWorkOrder.mockResolvedValueOnce({ ...mockWorkOrders[2] }); // completed
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("WO-003")).toBeInTheDocument();
    });
    expect(screen.queryByText("Change Status")).not.toBeInTheDocument();
  });

  it("shows back to work orders link", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Back to Work Orders")).toBeInTheDocument();
    });
  });

  it("shows not found state when work order is null", async () => {
    mockGetWorkOrder.mockRejectedValueOnce(new Error("Not found"));
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Work Order Not Found")).toBeInTheDocument();
    });
  });

  it("shows BOM empty state when no BOM data", async () => {
    mockGetWorkOrderBOM.mockResolvedValueOnce({ wo_id: "WO-001", assembly_ipn: "IPN-003", qty: 10, bom: [] });
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("No BOM data available for this work order")).toBeInTheDocument();
    });
  });

  it("displays Assembly Information card", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Assembly Information")).toBeInTheDocument();
      expect(screen.getByText("Assembly IPN")).toBeInTheDocument();
      expect(screen.getByText("Quantity")).toBeInTheDocument();
    });
  });

  it("displays Status & Priority card", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Status & Priority")).toBeInTheDocument();
    });
  });

  it("shows BOM table headers", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN")).toBeInTheDocument();
      expect(screen.getByText("Description")).toBeInTheDocument();
      expect(screen.getByText("Required")).toBeInTheDocument();
      expect(screen.getByText("On Hand")).toBeInTheDocument();
      expect(screen.getByText("Shortage")).toBeInTheDocument();
    });
  });

  it("shows shortage quantity in red for shortage items", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("30")).toBeInTheDocument();
    });
  });

  // Form submission tests
  it("selects new status and clicks Update Status, verifies updateWorkOrder called", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Change Status")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Change Status"));
    await waitFor(() => {
      expect(screen.getByText("Change Work Order Status")).toBeInTheDocument();
    });

    // Select new status
    fireEvent.click(screen.getByText("Select new status"));
    await waitFor(() => {
      expect(screen.getByText("In Progress")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("In Progress"));

    // Click Update Status
    fireEvent.click(screen.getByText("Update Status"));

    await waitFor(() => {
      expect(mockUpdateWorkOrder).toHaveBeenCalledWith("WO-001", { status: "in_progress" });
    });
  });

  it("selects vendor and clicks Generate PO, verifies generatePOFromWorkOrder called", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Generate PO")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Generate PO"));
    await waitFor(() => {
      expect(screen.getByText("Generate Purchase Order")).toBeInTheDocument();
    });

    // Select vendor
    fireEvent.click(screen.getByText("Select vendor for PO"));
    await waitFor(() => {
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Acme Corp"));

    // Click Generate PO button in dialog
    const generateButtons = screen.getAllByText("Generate PO");
    const submitButton = generateButtons[generateButtons.length - 1];
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockGeneratePOFromWorkOrder).toHaveBeenCalledWith("WO-001", "V-001");
    });
  });

  it("Update Status button is disabled when no status selected", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Change Status")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Change Status"));
    await waitFor(() => {
      expect(screen.getByText("Update Status")).toBeInTheDocument();
    });

    expect(screen.getByText("Update Status")).toBeDisabled();
  });

  it("status change error path — logs error on reject", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockUpdateWorkOrder.mockRejectedValueOnce(new Error("Server error"));
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Change Status")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Change Status"));
    await waitFor(() => {
      expect(screen.getByText("Select new status")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Select new status"));
    await waitFor(() => {
      expect(screen.getByText("In Progress")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("In Progress"));
    fireEvent.click(screen.getByText("Update Status"));
    await waitFor(() => {
      expect(mockUpdateWorkOrder).toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith("Failed to update status:", expect.any(Error));
    });
    consoleSpy.mockRestore();
  });

  it("generate PO error path — logs error on reject", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockGeneratePOFromWorkOrder.mockRejectedValueOnce(new Error("PO generation failed"));
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Generate PO")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Generate PO"));
    await waitFor(() => {
      expect(screen.getByText("Select vendor for PO")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Select vendor for PO"));
    await waitFor(() => {
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Acme Corp"));
    const generateButtons = screen.getAllByText("Generate PO");
    fireEvent.click(generateButtons[generateButtons.length - 1]);
    await waitFor(() => {
      expect(mockGeneratePOFromWorkOrder).toHaveBeenCalled();
      expect(consoleSpy).toHaveBeenCalledWith("Failed to generate PO:", expect.any(Error));
    });
    consoleSpy.mockRestore();
  });

  it("does not show Change Status for cancelled work order", async () => {
    mockGetWorkOrder.mockResolvedValueOnce({
      id: "WO-010", assembly_ipn: "IPN-001", qty: 1, status: "cancelled", priority: "low", created_at: "2024-01-01",
    });
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("WO-010")).toBeInTheDocument();
    });
    expect(screen.queryByText("Change Status")).not.toBeInTheDocument();
  });

  it("Generate PO button disabled when no vendor selected", async () => {
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Generate PO")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Generate PO"));
    await waitFor(() => {
      expect(screen.getByText("Generate Purchase Order")).toBeInTheDocument();
    });
    // The submit Generate PO button in footer should be disabled
    const generateButtons = screen.getAllByText("Generate PO");
    const submitButton = generateButtons[generateButtons.length - 1];
    expect(submitButton).toBeDisabled();
  });

  it("renders started_at and completed_at when present", async () => {
    mockGetWorkOrder.mockResolvedValueOnce({
      id: "WO-002", assembly_ipn: "IPN-003", qty: 5, status: "completed", priority: "high",
      created_at: "2024-01-10", started_at: "2024-01-11T09:00:00Z", completed_at: "2024-01-15T17:00:00Z",
    });
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Started")).toBeInTheDocument();
      expect(screen.getByText("Completed")).toBeInTheDocument();
    });
  });

  it("does not render Started/Completed labels when dates not set", async () => {
    // Default mockWO has no started_at or completed_at
    render(<WorkOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    expect(screen.queryByText("Started")).not.toBeInTheDocument();
    expect(screen.queryByText("Completed")).not.toBeInTheDocument();
  });
});
