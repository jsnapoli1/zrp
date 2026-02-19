import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent, within } from "../test/test-utils";

const mockGetReceivingInspections = vi.fn();
const mockInspectReceiving = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getReceivingInspections: (...args: any[]) => mockGetReceivingInspections(...args),
    inspectReceiving: (...args: any[]) => mockInspectReceiving(...args),
  },
}));

import Receiving from "./Receiving";

const mockInspections = [
  {
    id: 1,
    po_id: "PO-001",
    po_line_id: 1,
    ipn: "IPN-001",
    qty_received: 100,
    qty_passed: 0,
    qty_failed: 0,
    qty_on_hold: 0,
    inspector: "",
    inspected_at: null,
    notes: "",
    created_at: "2024-01-20 10:00:00",
  },
  {
    id: 2,
    po_id: "PO-002",
    po_line_id: 1,
    ipn: "IPN-002",
    qty_received: 50,
    qty_passed: 50,
    qty_failed: 0,
    qty_on_hold: 0,
    inspector: "alice",
    inspected_at: "2024-01-21 14:00:00",
    notes: "All good",
    created_at: "2024-01-19 09:00:00",
  },
  {
    id: 3,
    po_id: "PO-003",
    po_line_id: 1,
    ipn: "IPN-003",
    qty_received: 200,
    qty_passed: 180,
    qty_failed: 20,
    qty_on_hold: 0,
    inspector: "bob",
    inspected_at: "2024-01-22 11:00:00",
    notes: "Some defects",
    created_at: "2024-01-18 08:00:00",
  },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockGetReceivingInspections.mockResolvedValue(mockInspections);
  mockInspectReceiving.mockResolvedValue({ ...mockInspections[0], inspected_at: "2024-01-23", qty_passed: 100 });
});

const waitForLoad = () =>
  waitFor(() => expect(screen.getByText("Receiving & Inspection")).toBeInTheDocument());

describe("Receiving", () => {
  it("renders page title and subtitle", async () => {
    render(<Receiving />);
    await waitForLoad();
    expect(screen.getByText("Receiving & Inspection")).toBeInTheDocument();
    expect(screen.getByText(/Inspect received goods/)).toBeInTheDocument();
  });

  it("renders loading state initially", () => {
    mockGetReceivingInspections.mockReturnValue(new Promise(() => {}));
    render(<Receiving />);
    expect(screen.getByText("Loading inspections...")).toBeInTheDocument();
  });

  it("renders inspection list with all items", async () => {
    render(<Receiving />);
    await waitForLoad();
    expect(screen.getByText("IPN-001")).toBeInTheDocument();
    expect(screen.getByText("IPN-002")).toBeInTheDocument();
    expect(screen.getByText("IPN-003")).toBeInTheDocument();
  });

  it("renders PO links", async () => {
    render(<Receiving />);
    await waitForLoad();
    expect(screen.getByText("PO-001")).toBeInTheDocument();
    expect(screen.getByText("PO-002")).toBeInTheDocument();
    expect(screen.getByText("PO-003")).toBeInTheDocument();
  });

  // --- Summary Cards ---

  it("renders summary cards", async () => {
    render(<Receiving />);
    await waitForLoad();
    expect(screen.getByText("Pending Inspection")).toBeInTheDocument();
    expect(screen.getByText("With Failures")).toBeInTheDocument();
  });

  it("renders pending count in orange", async () => {
    render(<Receiving />);
    await waitForLoad();
    // Find the orange-colored pending count
    const pendingCard = screen.getByText("Pending Inspection").closest("div")!.parentElement!;
    const count = within(pendingCard).getByText("1");
    expect(count).toHaveClass("text-orange-600");
  });

  it("renders inspected count in green", async () => {
    render(<Receiving />);
    await waitForLoad();
    // Find the Inspected summary card (the one in the card grid, not the filter button)
    const cards = screen.getAllByText("Inspected");
    // The summary card version is inside a card with green styling
    const greenCount = screen.getByText("2");
    expect(greenCount).toHaveClass("text-green-600");
  });

  // --- Filter Tabs ---

  it("renders filter buttons", async () => {
    render(<Receiving />);
    await waitForLoad();
    expect(screen.getByRole("button", { name: "All" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Pending" })).toBeInTheDocument();
  });

  it("calls API with pending filter when clicking Pending button", async () => {
    render(<Receiving />);
    await waitForLoad();
    fireEvent.click(screen.getByRole("button", { name: "Pending" }));
    await waitFor(() => {
      expect(mockGetReceivingInspections).toHaveBeenCalledWith("pending");
    });
  });

  it("calls API with no filter when clicking All button", async () => {
    render(<Receiving />);
    await waitForLoad();
    fireEvent.click(screen.getByRole("button", { name: "Pending" }));
    await waitFor(() => expect(mockGetReceivingInspections).toHaveBeenCalledWith("pending"));
    fireEvent.click(screen.getByRole("button", { name: "All" }));
    await waitFor(() => {
      expect(mockGetReceivingInspections).toHaveBeenCalledWith(undefined);
    });
  });

  // --- Inspect Dialog ---

  it("shows Inspect button only for pending items", async () => {
    render(<Receiving />);
    await waitForLoad();
    const inspectButtons = screen.getAllByRole("button", { name: /^Inspect$/i });
    // Only 1 pending item
    expect(inspectButtons.length).toBe(1);
  });

  it("opens inspect dialog when clicking Inspect button", async () => {
    render(<Receiving />);
    await waitForLoad();
    const inspectButtons = screen.getAllByRole("button", { name: /^Inspect$/i });
    fireEvent.click(inspectButtons[0]);
    await waitFor(() => {
      expect(screen.getByText(/Inspect RI-1/)).toBeInTheDocument();
    });
  });

  it("inspect dialog shows qty received info", async () => {
    render(<Receiving />);
    await waitForLoad();
    fireEvent.click(screen.getAllByRole("button", { name: /^Inspect$/i })[0]);
    await waitFor(() => {
      expect(screen.getByText(/Qty Received/)).toBeInTheDocument();
      expect(screen.getByText(/PO.*PO-001/)).toBeInTheDocument();
    });
  });

  it("submits inspection and calls API", async () => {
    render(<Receiving />);
    await waitForLoad();
    fireEvent.click(screen.getAllByRole("button", { name: /^Inspect$/i })[0]);
    await waitFor(() => {
      expect(screen.getByText("Submit Inspection")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Submit Inspection"));
    await waitFor(() => {
      expect(mockInspectReceiving).toHaveBeenCalledWith(1, expect.objectContaining({
        qty_passed: 100,
      }));
    });
  });

  it("cancel closes inspect dialog", async () => {
    render(<Receiving />);
    await waitForLoad();
    fireEvent.click(screen.getAllByRole("button", { name: /^Inspect$/i })[0]);
    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Submit Inspection")).not.toBeInTheDocument();
    });
  });

  // --- Empty State ---

  it("shows empty state when no inspections", async () => {
    mockGetReceivingInspections.mockResolvedValue([]);
    render(<Receiving />);
    await waitForLoad();
    expect(screen.getByText("No receiving inspections found")).toBeInTheDocument();
  });

  // --- Table Content ---

  it("displays RI-ID format", async () => {
    render(<Receiving />);
    await waitForLoad();
    expect(screen.getByText("RI-1")).toBeInTheDocument();
    expect(screen.getByText("RI-2")).toBeInTheDocument();
    expect(screen.getByText("RI-3")).toBeInTheDocument();
  });

  it("displays inspector name for inspected items", async () => {
    render(<Receiving />);
    await waitForLoad();
    expect(screen.getByText("alice")).toBeInTheDocument();
    expect(screen.getByText("bob")).toBeInTheDocument();
  });
});
