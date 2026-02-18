import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ ipn: "IPN-003" }),
    useNavigate: () => mockNavigate,
  };
});

const mockPart = {
  ipn: "IPN-003",
  fields: {
    _category: "ICs",
    description: "MCU STM32",
    manufacturer: "STMicro",
    mpn: "STM32F401",
    cost: "5.00",
    price: "15.00",
    stock: "10",
    location: "Shelf C",
    status: "active",
    datasheet: "https://example.com/datasheet.pdf",
    notes: "Main MCU for product line",
  },
};

const mockBOM = {
  ipn: "IPN-003",
  description: "MCU STM32",
  qty: 1,
  children: [
    { ipn: "IPN-001", description: "10k Resistor", qty: 4, ref: "R1-R4", children: [] },
    { ipn: "IPN-002", description: "100uF Cap", qty: 2, ref: "C1-C2", children: [] },
  ],
};

const mockCost = {
  ipn: "IPN-003",
  last_unit_price: 4.50,
  po_id: "PO-001",
  last_ordered: "2024-01-15",
  bom_cost: 12.50,
};

const mockGetPart = vi.fn().mockResolvedValue(mockPart);
const mockGetPartBOM = vi.fn().mockResolvedValue(mockBOM);
const mockGetPartCost = vi.fn().mockResolvedValue(mockCost);

vi.mock("../lib/api", () => ({
  api: {
    getPart: (...args: any[]) => mockGetPart(...args),
    getPartBOM: (...args: any[]) => mockGetPartBOM(...args),
    getPartCost: (...args: any[]) => mockGetPartCost(...args),
  },
}));

import PartDetail from "./PartDetail";

beforeEach(() => vi.clearAllMocks());

describe("PartDetail", () => {
  it("renders part IPN as heading", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-003")).toBeInTheDocument();
    });
  });

  it("renders part description", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("MCU STM32")).toBeInTheDocument();
    });
  });

  it("renders category and status badges", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("ICs")).toBeInTheDocument();
      expect(screen.getByText("active")).toBeInTheDocument();
    });
  });

  it("renders Part Details card with fields", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Part Details")).toBeInTheDocument();
    });
    expect(screen.getByText("STMicro")).toBeInTheDocument();
    expect(screen.getByText("STM32F401")).toBeInTheDocument();
    expect(screen.getByText("Shelf C")).toBeInTheDocument();
  });

  it("renders manufacturer and MPN labels", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Manufacturer")).toBeInTheDocument();
      expect(screen.getByText("MPN")).toBeInTheDocument();
    });
  });

  it("renders stock value", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Stock")).toBeInTheDocument();
    });
  });

  it("renders notes", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Main MCU for product line")).toBeInTheDocument();
    });
  });

  it("renders datasheet link", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("View Datasheet")).toBeInTheDocument();
    });
    const link = screen.getByText("View Datasheet").closest("a");
    expect(link).toHaveAttribute("href", "https://example.com/datasheet.pdf");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders Cost Information card", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Cost Information")).toBeInTheDocument();
    });
  });

  it("shows unit cost", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("$5.00")).toBeInTheDocument();
    });
  });

  it("shows last purchase price", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("$4.50")).toBeInTheDocument();
      expect(screen.getByText("Last Purchase Price")).toBeInTheDocument();
    });
  });

  it("shows PO reference in cost section", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText(/PO: PO-001/)).toBeInTheDocument();
    });
  });

  it("shows BOM cost rollup", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("$12.50")).toBeInTheDocument();
      expect(screen.getByText("BOM Cost Rollup")).toBeInTheDocument();
    });
  });

  it("has back to parts button", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Back to Parts")).toBeInTheDocument();
    });
  });

  it("navigates back on back button click", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Back to Parts")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Back to Parts"));
    expect(mockNavigate).toHaveBeenCalledWith("/parts");
  });

  it("shows no cost info message when no cost data", async () => {
    mockGetPart.mockResolvedValueOnce({ ipn: "IPN-003", fields: {} });
    mockGetPartCost.mockResolvedValueOnce({ ipn: "IPN-003" });
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("No cost information available")).toBeInTheDocument();
    });
  });

  it("shows Part Not Found for missing part", async () => {
    mockGetPart.mockRejectedValueOnce(new Error("Not found"));
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Part Not Found")).toBeInTheDocument();
    });
    expect(screen.getByText(/could not be found/)).toBeInTheDocument();
  });
});

describe("PartDetail - BOM (assembly IPN)", () => {
  beforeEach(() => {
    // Override useParams to return an assembly IPN
    vi.mocked(require("react-router-dom").useParams).mockReturnValue({ ipn: "PCA-100" });
    mockGetPart.mockResolvedValue({
      ipn: "PCA-100",
      fields: { _category: "Assemblies", description: "Main Board", status: "active" },
    });
    mockGetPartBOM.mockResolvedValue(mockBOM);
    mockGetPartCost.mockResolvedValue({ ipn: "PCA-100", bom_cost: 25.0 });
  });

  it("renders Bill of Materials card for assembly", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Bill of Materials")).toBeInTheDocument();
    });
  });

  it("renders BOM children IPNs", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
      expect(screen.getByText("IPN-002")).toBeInTheDocument();
    });
  });

  it("renders BOM quantities", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Qty: 4")).toBeInTheDocument();
      expect(screen.getByText("Qty: 2")).toBeInTheDocument();
    });
  });

  it("renders BOM reference designators", async () => {
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("R1-R4")).toBeInTheDocument();
      expect(screen.getByText("C1-C2")).toBeInTheDocument();
    });
  });

  it("shows no BOM data message when BOM fetch fails", async () => {
    mockGetPartBOM.mockRejectedValueOnce(new Error("fail"));
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("No BOM data available for this assembly")).toBeInTheDocument();
    });
  });
});
