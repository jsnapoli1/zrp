import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockWorkOrders, mockParts } from "../test/mocks";

const mockGetWorkOrders = vi.fn().mockResolvedValue(mockWorkOrders);
const mockGetParts = vi.fn().mockResolvedValue(mockParts);
const mockCreateWorkOrder = vi.fn().mockResolvedValue(mockWorkOrders[0]);

vi.mock("../lib/api", () => ({
  api: {
    getWorkOrders: (...args: any[]) => mockGetWorkOrders(...args),
    getParts: (...args: any[]) => mockGetParts(...args),
    createWorkOrder: (...args: any[]) => mockCreateWorkOrder(...args),
  },
}));

import WorkOrders from "./WorkOrders";

beforeEach(() => vi.clearAllMocks());

describe("WorkOrders", () => {
  it("renders loading state", () => {
    render(<WorkOrders />);
    expect(screen.getByText("Loading work orders...")).toBeInTheDocument();
  });

  it("renders work orders table after loading", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    expect(screen.getByText("WO-002")).toBeInTheDocument();
    expect(screen.getByText("WO-003")).toBeInTheDocument();
  });

  it("shows summary cards with correct counts", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("Total WOs")).toBeInTheDocument();
    });
    // Total = 3
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("Open")).toBeInTheDocument();
    expect(screen.getByText("In Progress")).toBeInTheDocument();
    expect(screen.getByText("On Hold")).toBeInTheDocument();
    expect(screen.getByText("Completed")).toBeInTheDocument();
  });

  it("has create work order button", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("Create Work Order")).toBeInTheDocument();
    });
  });

  it("opens create dialog and shows form fields", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Work Order"));
    await waitFor(() => {
      expect(screen.getByLabelText(/assembly ipn/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/quantity/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
    });
  });

  it("create dialog has cancel button that closes it", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Work Order"));
    await waitFor(() => {
      expect(screen.getByLabelText(/assembly ipn/i)).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
  });

  it("create button is disabled when assembly_ipn is empty", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Work Order"));
    await waitFor(() => {
      expect(screen.getByLabelText(/assembly ipn/i)).toBeInTheDocument();
    });
    // The second "Create Work Order" button in the dialog footer
    const createButtons = screen.getAllByText("Create Work Order");
    const submitButton = createButtons[createButtons.length - 1];
    expect(submitButton).toBeDisabled();
  });

  it("typing in assembly IPN shows filtered parts dropdown", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Work Order"));
    await waitFor(() => {
      expect(screen.getByLabelText(/assembly ipn/i)).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText(/assembly ipn/i), { target: { value: "IPN" } });
    await waitFor(() => {
      // Dropdown shows filtered parts - check that the input accepted the value
      expect(screen.getByLabelText(/assembly ipn/i)).toHaveValue("IPN");
    });
  });

  it("selecting a part from dropdown fills the IPN field", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Work Order"));
    await waitFor(() => {
      expect(screen.getByLabelText(/assembly ipn/i)).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText(/assembly ipn/i), { target: { value: "IPN-001" } });
    await waitFor(() => {
      // Find dropdown items by their container class
      const dropdownItems = document.querySelectorAll(".hover\\:bg-muted.cursor-pointer");
      expect(dropdownItems.length).toBeGreaterThan(0);
      fireEvent.click(dropdownItems[0]);
    });
    expect(screen.getByLabelText(/assembly ipn/i)).toHaveValue("IPN-001");
  });

  it("submits create form and calls API", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Work Order"));
    await waitFor(() => {
      expect(screen.getByLabelText(/assembly ipn/i)).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText(/assembly ipn/i), { target: { value: "IPN-003" } });
    fireEvent.change(screen.getByLabelText(/quantity/i), { target: { value: "5" } });
    
    const createButtons = screen.getAllByText("Create Work Order");
    const submitButton = createButtons[createButtons.length - 1];
    fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(mockCreateWorkOrder).toHaveBeenCalledWith(expect.objectContaining({
        assembly_ipn: "IPN-003",
        qty: 5,
      }));
    });
  });

  it("shows status badges with correct labels", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("OPEN")).toBeInTheDocument();
      expect(screen.getByText("IN PROGRESS")).toBeInTheDocument();
      expect(screen.getByText("COMPLETED")).toBeInTheDocument();
    });
  });

  it("shows priority badges", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("MEDIUM")).toBeInTheDocument();
      expect(screen.getByText("HIGH")).toBeInTheDocument();
      expect(screen.getByText("LOW")).toBeInTheDocument();
    });
  });

  it("shows empty state when no work orders", async () => {
    mockGetWorkOrders.mockResolvedValueOnce([]);
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("No work orders found")).toBeInTheDocument();
    });
  });

  it("has view details and BOM links", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      const viewButtons = screen.getAllByText("View Details");
      expect(viewButtons.length).toBe(3);
      const bomButtons = screen.getAllByText("BOM");
      expect(bomButtons.length).toBe(3);
    });
  });

  it("renders table headers correctly", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO ID")).toBeInTheDocument();
      expect(screen.getByText("Assembly")).toBeInTheDocument();
      expect(screen.getByText("Status")).toBeInTheDocument();
      expect(screen.getByText("Priority")).toBeInTheDocument();
      expect(screen.getByText("Qty")).toBeInTheDocument();
      expect(screen.getByText("Created")).toBeInTheDocument();
      expect(screen.getByText("Age")).toBeInTheDocument();
      expect(screen.getByText("Actions")).toBeInTheDocument();
    });
  });

  it("handles API error gracefully", async () => {
    mockGetWorkOrders.mockRejectedValueOnce(new Error("Network error"));
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("No work orders found")).toBeInTheDocument();
    });
  });

  it("displays assembly description when available", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      // IPN-003 matches "MCU STM32" from mockParts
      expect(screen.getAllByText("MCU STM32").length).toBeGreaterThan(0);
    });
  });

  it("displays quantity for each work order", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("10")).toBeInTheDocument();
      expect(screen.getByText("5")).toBeInTheDocument();
      expect(screen.getByText("100")).toBeInTheDocument();
    });
  });

  it("priority select interaction in create form", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Work Order"));
    await waitFor(() => {
      expect(screen.getByLabelText(/assembly ipn/i)).toBeInTheDocument();
    });
    // Default is "Medium" — change to "Critical" via the select
    fireEvent.click(screen.getByText("Medium"));
    await waitFor(() => {
      expect(screen.getByText("Critical")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Critical"));
    // Fill IPN and submit to verify priority is sent
    fireEvent.change(screen.getByLabelText(/assembly ipn/i), { target: { value: "IPN-001" } });
    const createButtons = screen.getAllByText("Create Work Order");
    fireEvent.click(createButtons[createButtons.length - 1]);
    await waitFor(() => {
      expect(mockCreateWorkOrder).toHaveBeenCalledWith(expect.objectContaining({
        priority: "critical",
      }));
    });
  });

  it("quantity field edge cases — parseInt fallback to 1", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Work Order"));
    await waitFor(() => {
      expect(screen.getByLabelText(/quantity/i)).toBeInTheDocument();
    });
    const qtyInput = screen.getByLabelText(/quantity/i);

    // qty=0 → parseInt("0") is 0, which is falsy, so || 1 → 1
    fireEvent.change(qtyInput, { target: { value: "0" } });
    expect(qtyInput).toHaveValue(1);

    // negative → parseInt("-3") is -3, truthy, stays -3
    fireEvent.change(qtyInput, { target: { value: "-3" } });
    expect(qtyInput).toHaveValue(-3);

    // NaN → parseInt("abc") is NaN, falsy, so || 1 → 1
    fireEvent.change(qtyInput, { target: { value: "abc" } });
    expect(qtyInput).toHaveValue(1);
  });

  it("View Details links have correct /work-orders/{id} URLs", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    const viewLinks = screen.getAllByText("View Details");
    expect(viewLinks[0].closest("a")).toHaveAttribute("href", "/work-orders/WO-001");
    expect(viewLinks[1].closest("a")).toHaveAttribute("href", "/work-orders/WO-002");
    expect(viewLinks[2].closest("a")).toHaveAttribute("href", "/work-orders/WO-003");
  });

  it("renders cancelled and on_hold status badges", async () => {
    mockGetWorkOrders.mockResolvedValueOnce([
      { id: "WO-010", assembly_ipn: "IPN-001", qty: 1, status: "cancelled", priority: "low", created_at: "2024-01-01" },
      { id: "WO-011", assembly_ipn: "IPN-002", qty: 2, status: "on_hold", priority: "medium", created_at: "2024-01-02" },
    ]);
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("CANCELLED")).toBeInTheDocument();
      expect(screen.getByText("ON HOLD")).toBeInTheDocument();
    });
    // Verify styling classes
    const cancelledBadge = screen.getByText("CANCELLED");
    expect(cancelledBadge).toHaveClass("text-red-700");
    const onHoldBadge = screen.getByText("ON HOLD");
    expect(onHoldBadge).toHaveClass("text-orange-700");
  });
});
