import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import { mockVendors, mockPOs } from "../test/mocks";

const mockVendor = {
  ...mockVendors[0],
  contact_phone: "555-1234",
  notes: "Important vendor notes",
};

const mockPOsWithLines = [
  {
    ...mockPOs[0],
    vendor_id: "V-001",
    status: "submitted",
    lines: [{ qty_ordered: 10, unit_price: 5.0 }],
  },
  {
    ...mockPOs[1],
    vendor_id: "V-001",
    status: "partial",
    lines: [{ qty_ordered: 20, unit_price: 3.0 }],
  },
];

const mockGetVendor = vi.fn().mockResolvedValue(mockVendor);
const mockGetPurchaseOrders = vi.fn().mockResolvedValue(mockPOsWithLines);

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "V-001" }),
  };
});

vi.mock("../lib/api", () => ({
  api: {
    getVendor: (...args: any[]) => mockGetVendor(...args),
    getPurchaseOrders: (...args: any[]) => mockGetPurchaseOrders(...args),
  },
}));

import VendorDetail from "./VendorDetail";

beforeEach(() => vi.clearAllMocks());

describe("VendorDetail", () => {
  it("renders loading state", () => {
    render(<VendorDetail />);
    expect(screen.getByText("Loading vendor details...")).toBeInTheDocument();
  });

  it("renders vendor name after loading", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      const elements = screen.getAllByText("Acme Corp");
      expect(elements.length).toBeGreaterThan(0);
    });
  });

  it("shows Back to Vendors link", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("Back to Vendors")).toBeInTheDocument();
    });
    const link = screen.getByText("Back to Vendors").closest("a");
    expect(link).toHaveAttribute("href", "/vendors");
  });

  it("shows vendor status badge", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("ACTIVE")).toBeInTheDocument();
    });
  });

  it("shows contact information section", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("Contact Information")).toBeInTheDocument();
    });
    expect(screen.getByText("Company Name")).toBeInTheDocument();
    expect(screen.getByText("Primary Contact")).toBeInTheDocument();
  });

  it("displays vendor email", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("john@acme.com")).toBeInTheDocument();
    });
  });

  it("displays vendor phone", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("555-1234")).toBeInTheDocument();
    });
  });

  it("displays vendor website", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("https://acme.com")).toBeInTheDocument();
    });
  });

  it("displays vendor notes", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("Important vendor notes")).toBeInTheDocument();
    });
  });

  it("shows lead time", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("14 days")).toBeInTheDocument();
    });
  });

  it("shows summary cards", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("Total POs")).toBeInTheDocument();
    });
    expect(screen.getByText("Active POs")).toBeInTheDocument();
    expect(screen.getByText("Total Value")).toBeInTheDocument();
    const leadTimes = screen.getAllByText("Lead Time");
    expect(leadTimes.length).toBeGreaterThan(0);
  });

  it("calculates total PO value", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      // 10*5 + 20*3 = 110
      expect(screen.getByText("$110")).toBeInTheDocument();
    });
  });

  it("counts active POs (submitted + partial)", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("Active POs")).toBeInTheDocument();
    });
    // Both POs are submitted/partial - active POs card has orange text
    await waitFor(() => {
      const twos = screen.getAllByText("2");
      expect(twos.length).toBeGreaterThanOrEqual(2);
    });
  });

  it("shows Edit Vendor and Create PO buttons", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Vendor")).toBeInTheDocument();
      expect(screen.getByText("Create PO")).toBeInTheDocument();
    });
  });

  it("has tabs for Price Catalog and Purchase Orders", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      const priceCatalogs = screen.getAllByText("Price Catalog");
      expect(priceCatalogs.length).toBeGreaterThan(0);
      const poTabs = screen.getAllByText("Purchase Orders");
      expect(poTabs.length).toBeGreaterThan(0);
    });
  });

  it("shows price catalog table with mock data", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("RES-001")).toBeInTheDocument();
    });
    expect(screen.getByText("CAP-002")).toBeInTheDocument();
    expect(screen.getByText("$0.05")).toBeInTheDocument();
    expect(screen.getByText("$0.08")).toBeInTheDocument();
  });

  it("shows price catalog headers", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN")).toBeInTheDocument();
      expect(screen.getByText("MPN")).toBeInTheDocument();
      expect(screen.getByText("Unit Price")).toBeInTheDocument();
    });
  });

  it("shows vendor not found when vendor is null", async () => {
    mockGetVendor.mockResolvedValueOnce(null);
    render(<VendorDetail />);
    await waitFor(() => {
      expect(screen.getByText("Vendor Not Found")).toBeInTheDocument();
    });
  });

  it("handles API error gracefully", async () => {
    mockGetVendor.mockRejectedValueOnce(new Error("Not found"));
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    render(<VendorDetail />);
    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalled();
    });
    consoleSpy.mockRestore();
  });

  it("calls getVendor with correct id", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(mockGetVendor).toHaveBeenCalledWith("V-001");
    });
  });

  it("calls getPurchaseOrders on mount", async () => {
    render(<VendorDetail />);
    await waitFor(() => {
      expect(mockGetPurchaseOrders).toHaveBeenCalled();
    });
  });
});
