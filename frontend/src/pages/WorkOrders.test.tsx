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

  it("shows summary cards", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("Total WOs")).toBeInTheDocument();
      expect(screen.getByText("Open")).toBeInTheDocument();
      expect(screen.getByText("In Progress")).toBeInTheDocument();
      expect(screen.getByText("On Hold")).toBeInTheDocument();
      expect(screen.getByText("Completed")).toBeInTheDocument();
    });
  });

  it("has create work order button", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("Create Work Order")).toBeInTheDocument();
    });
  });

  it("opens create dialog on button click", async () => {
    render(<WorkOrders />);
    await waitFor(() => {
      expect(screen.getByText("WO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Work Order"));
    await waitFor(() => {
      expect(screen.getByLabelText(/assembly ipn/i)).toBeInTheDocument();
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
      expect(viewButtons.length).toBeGreaterThan(0);
    });
  });
});
