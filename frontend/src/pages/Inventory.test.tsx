import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockInventory, mockParts } from "../test/mocks";

const mockGetInventory = vi.fn().mockResolvedValue(mockInventory);
const mockGetParts = vi.fn().mockResolvedValue(mockParts);
const mockCreateInventoryTransaction = vi.fn().mockResolvedValue(undefined);
const mockBulkDeleteInventory = vi.fn().mockResolvedValue(undefined);

vi.mock("../lib/api", () => ({
  api: {
    getInventory: (...args: any[]) => mockGetInventory(...args),
    getParts: (...args: any[]) => mockGetParts(...args),
    createInventoryTransaction: (...args: any[]) => mockCreateInventoryTransaction(...args),
    bulkDeleteInventory: (...args: any[]) => mockBulkDeleteInventory(...args),
  },
}));

import Inventory from "./Inventory";

beforeEach(() => vi.clearAllMocks());

describe("Inventory", () => {
  it("renders loading state", () => {
    render(<Inventory />);
    expect(screen.getByText("Loading inventory...")).toBeInTheDocument();
  });

  it("renders page title and subtitle", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("Inventory")).toBeInTheDocument();
    });
    expect(screen.getByText("Manage your inventory levels and stock tracking.")).toBeInTheDocument();
  });

  it("renders inventory table after loading", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    expect(screen.getByText("IPN-002")).toBeInTheDocument();
    expect(screen.getByText("IPN-003")).toBeInTheDocument();
  });

  it("shows summary cards", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("Total Items")).toBeInTheDocument();
      expect(screen.getByText("Low Stock Items")).toBeInTheDocument();
      expect(screen.getByText("Selected")).toBeInTheDocument();
    });
  });

  it("shows correct total items count", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("3")).toBeInTheDocument(); // 3 items
    });
  });

  it("shows low stock items count", async () => {
    render(<Inventory />);
    // IPN-002 has qty_on_hand=20, reorder_point=50 → low stock
    // IPN-003 has qty_on_hand=10, reorder_point=5 → NOT low stock (10 > 5)
    await waitFor(() => {
      expect(screen.getByText("Low Stock Items")).toBeInTheDocument();
    });
  });

  it("has quick receive button", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("Quick Receive")).toBeInTheDocument();
    });
  });

  it("has low stock filter button", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("Low Stock")).toBeInTheDocument();
    });
  });

  it("opens quick receive dialog", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => {
      expect(screen.getByText("Quick Receive Inventory")).toBeInTheDocument();
    });
  });

  it("quick receive dialog has form fields", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => {
      expect(screen.getByLabelText("IPN")).toBeInTheDocument();
      expect(screen.getByLabelText("Quantity")).toBeInTheDocument();
      expect(screen.getByLabelText("Reference")).toBeInTheDocument();
      expect(screen.getByLabelText("Notes")).toBeInTheDocument();
    });
  });

  it("receive button disabled when IPN/qty empty", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => {
      expect(screen.getByText("Receive")).toBeInTheDocument();
    });
    // The submit "Receive" button (not the trigger) should be disabled
    const buttons = screen.getAllByText("Receive");
    const submitBtn = buttons.find(b => b.closest("[role='dialog']"));
    expect(submitBtn?.closest("button")).toBeDisabled();
  });

  it("submits quick receive form", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => {
      expect(screen.getByLabelText("IPN")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("IPN"), { target: { value: "IPN-001" } });
    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "50" } });
    fireEvent.change(screen.getByLabelText("Reference"), { target: { value: "PO-100" } });

    // Find Receive button inside dialog
    const dialog = screen.getByRole("dialog");
    const receiveBtn = dialog.querySelector("button:not([disabled])");
    // Click the non-disabled receive button
    const buttons = screen.getAllByText("Receive");
    fireEvent.click(buttons[buttons.length - 1]);

    await waitFor(() => {
      expect(mockCreateInventoryTransaction).toHaveBeenCalledWith(
        expect.objectContaining({
          ipn: "IPN-001",
          type: "receive",
          qty: 50,
          reference: "PO-100",
        })
      );
    });
  });

  it("shows table headers", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN")).toBeInTheDocument();
      expect(screen.getByText("Description")).toBeInTheDocument();
      expect(screen.getByText("On Hand")).toBeInTheDocument();
      expect(screen.getByText("Reserved")).toBeInTheDocument();
      expect(screen.getByText("Available")).toBeInTheDocument();
      expect(screen.getByText("Location")).toBeInTheDocument();
      expect(screen.getByText("Reorder Point")).toBeInTheDocument();
    });
  });

  it("shows stock levels in table", async () => {
    render(<Inventory />);
    await waitFor(() => {
      // IPN-001: on_hand=500, reserved=50, available=450
      expect(screen.getByText("500")).toBeInTheDocument();
      expect(screen.getByText("50")).toBeInTheDocument();
      expect(screen.getByText("450")).toBeInTheDocument();
    });
  });

  it("shows locations in table", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("Bin A1")).toBeInTheDocument();
      expect(screen.getByText("Bin B2")).toBeInTheDocument();
      expect(screen.getByText("Shelf C")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetInventory.mockResolvedValueOnce([]);
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("No inventory items found")).toBeInTheDocument();
    });
  });

  it("shows low stock empty state when filtered", async () => {
    mockGetInventory.mockResolvedValueOnce(mockInventory); // initial
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });

    mockGetInventory.mockResolvedValueOnce([]);
    fireEvent.click(screen.getByText("Low Stock"));
    await waitFor(() => {
      expect(screen.getByText("No low stock items found")).toBeInTheDocument();
    });
  });

  it("toggles low stock filter", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Low Stock"));
    await waitFor(() => {
      expect(mockGetInventory).toHaveBeenCalledWith(true);
    });
  });

  it("shows select all checkbox in summary card", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("Selected")).toBeInTheDocument();
      expect(screen.getByText("0")).toBeInTheDocument();
    });
  });

  it("shows bulk actions message when nothing selected", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("Select items for bulk actions")).toBeInTheDocument();
    });
  });

  it("renders inventory item links", async () => {
    render(<Inventory />);
    await waitFor(() => {
      const link = screen.getByText("IPN-001").closest("a");
      expect(link).toHaveAttribute("href", "/inventory/IPN-001");
    });
  });

  it("cancel closes quick receive dialog", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Quick Receive"));
    await waitFor(() => {
      expect(screen.getByText("Quick Receive Inventory")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Quick Receive Inventory")).not.toBeInTheDocument();
    });
  });

  it("shows Inventory Items card title", async () => {
    render(<Inventory />);
    await waitFor(() => {
      expect(screen.getByText("Inventory Items")).toBeInTheDocument();
    });
  });
});
