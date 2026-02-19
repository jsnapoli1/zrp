import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockSalesOrders } from "../test/mocks";

const mockOrder = { ...mockSalesOrders[0] };
const mockGetSalesOrder = vi.fn().mockResolvedValue(mockOrder);
const mockConfirmSalesOrder = vi.fn().mockResolvedValue({ ...mockOrder, status: "confirmed" });
const mockAllocateSalesOrder = vi.fn().mockResolvedValue({ ...mockOrder, status: "allocated" });

vi.mock("../lib/api", () => ({
  api: {
    getSalesOrder: (...args: any[]) => mockGetSalesOrder(...args),
    confirmSalesOrder: (...args: any[]) => mockConfirmSalesOrder(...args),
    allocateSalesOrder: (...args: any[]) => mockAllocateSalesOrder(...args),
    pickSalesOrder: vi.fn(),
    shipSalesOrder: vi.fn(),
    invoiceSalesOrder: vi.fn(),
  },
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ id: "SO-0001" }),
  };
});

import SalesOrderDetail from "./SalesOrderDetail";

beforeEach(() => vi.clearAllMocks());

describe("SalesOrderDetail", () => {
  it("renders order details", async () => {
    render(<SalesOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("SO-0001")).toBeInTheDocument();
    });
    expect(screen.getAllByText("Acme Inc").length).toBeGreaterThan(0);
  });

  it("shows order lines", async () => {
    render(<SalesOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    expect(screen.getByText("Widget")).toBeInTheDocument();
    expect(screen.getByText("$25.50")).toBeInTheDocument();
  });

  it("shows status progression steps", async () => {
    render(<SalesOrderDetail />);
    await waitFor(() => {
      expect(screen.getAllByText(/draft/i).length).toBeGreaterThan(0);
    });
    expect(screen.getAllByText(/confirmed/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/allocated/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/shipped/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/invoiced/i).length).toBeGreaterThan(0);
  });

  it("shows next action button for draft", async () => {
    render(<SalesOrderDetail />);
    await waitFor(() => {
      expect(screen.getByTestId("next-action")).toBeInTheDocument();
    });
    expect(screen.getByText("Confirm Order")).toBeInTheDocument();
  });

  it("calls confirm on action click", async () => {
    render(<SalesOrderDetail />);
    await waitFor(() => {
      expect(screen.getByTestId("next-action")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("next-action"));
    await waitFor(() => {
      expect(mockConfirmSalesOrder).toHaveBeenCalledWith("SO-0001");
    });
  });

  it("shows quote link", async () => {
    render(<SalesOrderDetail />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
  });

  it("shows line total", async () => {
    render(<SalesOrderDetail />);
    await waitFor(() => {
      // Line total: 10 * 25.50 = 255.00
      expect(screen.getAllByText("$255.00").length).toBeGreaterThanOrEqual(1);
    });
  });
});
