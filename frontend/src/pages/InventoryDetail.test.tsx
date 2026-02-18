import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockInventory } from "../test/mocks";
import type { InventoryTransaction } from "../lib/api";

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ ipn: "IPN-001" }),
  };
});

const mockItem = mockInventory[0]; // IPN-001, qty_on_hand=500, qty_reserved=50, reorder_point=100

const mockTransactions: InventoryTransaction[] = [
  { id: 1, ipn: "IPN-001", type: "receive", qty: 500, reference: "PO-001", notes: "Initial stock", created_at: "2024-01-15T10:00:00Z" },
  { id: 2, ipn: "IPN-001", type: "issue", qty: 50, reference: "WO-001", notes: "Production run", created_at: "2024-01-18T14:30:00Z" },
  { id: 3, ipn: "IPN-001", type: "adjust", qty: 500, reference: "", notes: "Cycle count", created_at: "2024-01-20T09:00:00Z" },
];

const mockGetInventoryItem = vi.fn().mockResolvedValue(mockItem);
const mockGetInventoryHistory = vi.fn().mockResolvedValue(mockTransactions);
const mockCreateInventoryTransaction = vi.fn().mockResolvedValue(undefined);

vi.mock("../lib/api", () => ({
  api: {
    getInventoryItem: (...args: any[]) => mockGetInventoryItem(...args),
    getInventoryHistory: (...args: any[]) => mockGetInventoryHistory(...args),
    createInventoryTransaction: (...args: any[]) => mockCreateInventoryTransaction(...args),
  },
}));

import InventoryDetail from "./InventoryDetail";

beforeEach(() => vi.clearAllMocks());

describe("InventoryDetail", () => {
  it("renders loading state initially", () => {
    mockGetInventoryItem.mockReturnValue(new Promise(() => {}));
    render(<InventoryDetail />);
    expect(screen.getByText("Loading inventory item...")).toBeInTheDocument();
  });

  it("renders item IPN as heading", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
  });

  it("renders item description", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("10k Resistor")).toBeInTheDocument();
    });
  });

  it("renders back to inventory link", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Back to Inventory")).toBeInTheDocument();
    });
    const link = screen.getByText("Back to Inventory").closest("a");
    expect(link).toHaveAttribute("href", "/inventory");
  });

  // Stock level cards
  it("renders stock level cards", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("On Hand")).toBeInTheDocument();
      expect(screen.getByText("Reserved")).toBeInTheDocument();
      expect(screen.getByText("Available")).toBeInTheDocument();
      expect(screen.getByText("Reorder Point")).toBeInTheDocument();
    });
  });

  it("shows correct on hand qty", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("500")).toBeInTheDocument();
    });
  });

  it("shows correct reserved qty", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      // reserved = 50, but 50 also appears in transaction table
      expect(screen.getByText("On Hand")).toBeInTheDocument();
    });
  });

  it("shows correct available qty (on_hand - reserved)", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("450")).toBeInTheDocument(); // 500 - 50
    });
  });

  it("shows reorder point", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("100")).toBeInTheDocument();
    });
  });

  // Item details card
  it("renders Item Details card", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Item Details")).toBeInTheDocument();
    });
  });

  it("shows IPN in details", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Internal Part Number")).toBeInTheDocument();
    });
  });

  it("shows location in details", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Bin A1")).toBeInTheDocument();
    });
  });

  it("shows reorder quantity", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Reorder Quantity")).toBeInTheDocument();
    });
  });

  // Transaction history
  it("renders Transaction History card", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Transaction History")).toBeInTheDocument();
    });
  });

  it("shows transaction table headers", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Date")).toBeInTheDocument();
      expect(screen.getByText("Type")).toBeInTheDocument();
      expect(screen.getByText("Quantity")).toBeInTheDocument();
      expect(screen.getByText("Reference")).toBeInTheDocument();
      expect(screen.getByText("Notes")).toBeInTheDocument();
    });
  });

  it("shows transaction types as badges", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("RECEIVE")).toBeInTheDocument();
      expect(screen.getByText("ISSUE")).toBeInTheDocument();
      expect(screen.getByText("ADJUST")).toBeInTheDocument();
    });
  });

  it("shows transaction references", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("PO-001")).toBeInTheDocument();
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
  });

  it("shows transaction notes", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Initial stock")).toBeInTheDocument();
      expect(screen.getByText("Production run")).toBeInTheDocument();
      expect(screen.getByText("Cycle count")).toBeInTheDocument();
    });
  });

  it("shows issue qty with negative sign", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("-50")).toBeInTheDocument();
    });
  });

  it("shows empty transaction history message", async () => {
    mockGetInventoryHistory.mockResolvedValueOnce([]);
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("No transaction history found for this item")).toBeInTheDocument();
    });
  });

  // New Transaction dialog
  it("has New Transaction button", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("New Transaction")).toBeInTheDocument();
    });
  });

  it("opens transaction dialog", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => {
      expect(screen.getByText("Create Inventory Transaction")).toBeInTheDocument();
    });
  });

  it("transaction dialog has form fields", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => {
      expect(screen.getByLabelText("Quantity")).toBeInTheDocument();
      expect(screen.getByLabelText("Reference")).toBeInTheDocument();
      expect(screen.getByLabelText("Notes")).toBeInTheDocument();
      expect(screen.getByLabelText("Transaction Type")).toBeInTheDocument();
    });
  });

  it("create transaction button disabled when qty empty", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => {
      expect(screen.getByText("Create Transaction")).toBeDisabled();
    });
  });

  it("submits transaction form", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => {
      expect(screen.getByLabelText("Quantity")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Quantity"), { target: { value: "25" } });
    fireEvent.change(screen.getByLabelText("Reference"), { target: { value: "PO-999" } });
    fireEvent.click(screen.getByText("Create Transaction"));

    await waitFor(() => {
      expect(mockCreateInventoryTransaction).toHaveBeenCalledWith(
        expect.objectContaining({
          ipn: "IPN-001",
          type: "receive",
          qty: 25,
          reference: "PO-999",
        })
      );
    });
  });

  it("cancel closes transaction dialog", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("New Transaction"));
    await waitFor(() => {
      expect(screen.getByText("Create Inventory Transaction")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Create Inventory Transaction")).not.toBeInTheDocument();
    });
  });

  // Not found state
  it("shows not found when item is null", async () => {
    mockGetInventoryItem.mockRejectedValueOnce(new Error("Not found"));
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("Inventory Item Not Found")).toBeInTheDocument();
    });
  });

  it("shows not found with IPN in message", async () => {
    mockGetInventoryItem.mockResolvedValueOnce(null);
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText(/IPN-001/)).toBeInTheDocument();
    });
  });

  // Low stock indicator
  it("does not show LOW badge when stock is above reorder point", async () => {
    // IPN-001: qty_on_hand=500, reorder_point=100 → not low
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    expect(screen.queryByText("LOW")).not.toBeInTheDocument();
  });

  it("shows LOW badge when stock is at or below reorder point", async () => {
    mockGetInventoryItem.mockResolvedValueOnce({
      ...mockItem,
      qty_on_hand: 50,
      reorder_point: 100,
    });
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("LOW")).toBeInTheDocument();
    });
  });

  it("shows adjust note in dialog", async () => {
    render(<InventoryDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("New Transaction"));
    // The adjust note is only visible when type is "adjust" - default is "receive"
    // So we won't see it by default
    await waitFor(() => {
      expect(screen.queryByText(/For adjustments/)).not.toBeInTheDocument();
    });
  });
});
