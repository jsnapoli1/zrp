import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

let mockIPN = "IPN-003";
const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ ipn: mockIPN }),
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
  ipn: "PCA-100",
  description: "Main Board",
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

const mockGetPart = vi.fn();
const mockGetPartBOM = vi.fn();
const mockGetPartCost = vi.fn();
const mockGetPartWhereUsed = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getPart: (...args: any[]) => mockGetPart(...args),
    getPartBOM: (...args: any[]) => mockGetPartBOM(...args),
    getPartCost: (...args: any[]) => mockGetPartCost(...args),
    getPartWhereUsed: (...args: any[]) => mockGetPartWhereUsed(...args),
  },
}));

import PartDetail from "./PartDetail";

beforeEach(() => {
  vi.clearAllMocks();
  mockIPN = "IPN-003";
  mockGetPart.mockResolvedValue(mockPart);
  mockGetPartBOM.mockResolvedValue(mockBOM);
  mockGetPartCost.mockResolvedValue(mockCost);
  mockGetPartWhereUsed.mockResolvedValue([]);
});

const waitForLoad = () => waitFor(() => expect(screen.getByText("Part Details")).toBeInTheDocument());

describe("PartDetail", () => {
  it("renders part IPN as heading", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("IPN-003");
  });

  it("renders part description", async () => {
    render(<PartDetail />);
    await waitForLoad();
    // Description appears in multiple places; just check it exists
    expect(screen.getAllByText("MCU STM32").length).toBeGreaterThan(0);
  });

  it("renders category and status badges", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getAllByText("ICs").length).toBeGreaterThan(0);
    expect(screen.getAllByText("active").length).toBeGreaterThan(0);
  });

  it("renders Part Details card with fields", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText("Part Details")).toBeInTheDocument();
    expect(screen.getByText("STMicro")).toBeInTheDocument();
    expect(screen.getByText("STM32F401")).toBeInTheDocument();
    expect(screen.getByText("Shelf C")).toBeInTheDocument();
  });

  it("renders manufacturer and MPN labels", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText("Manufacturer")).toBeInTheDocument();
    expect(screen.getByText("MPN")).toBeInTheDocument();
  });

  it("renders stock value", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText("Stock")).toBeInTheDocument();
  });

  it("renders notes", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText("Main MCU for product line")).toBeInTheDocument();
  });

  it("renders datasheet link", async () => {
    render(<PartDetail />);
    await waitForLoad();
    const link = screen.getByText("View Datasheet").closest("a");
    expect(link).toHaveAttribute("href", "https://example.com/datasheet.pdf");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders Cost Information card", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText("Cost Information")).toBeInTheDocument();
  });

  it("shows unit cost", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText("$5.00")).toBeInTheDocument();
  });

  it("shows last purchase price", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText("$4.50")).toBeInTheDocument();
    expect(screen.getByText("Last Purchase Price")).toBeInTheDocument();
  });

  it("shows PO reference in cost section", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText(/PO: PO-001/)).toBeInTheDocument();
  });

  it("shows BOM cost rollup", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText("$12.50")).toBeInTheDocument();
    expect(screen.getByText("BOM Cost Rollup")).toBeInTheDocument();
  });

  it("has back to parts button", async () => {
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.getByText("Back to Parts")).toBeInTheDocument();
  });

  it("navigates back on back button click", async () => {
    render(<PartDetail />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Back to Parts"));
    expect(mockNavigate).toHaveBeenCalledWith("/parts");
  });

  it("shows no cost info message when no cost data", async () => {
    mockGetPart.mockResolvedValue({ ipn: "IPN-003", fields: {} });
    mockGetPartCost.mockResolvedValue({ ipn: "IPN-003" });
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("No cost information available")).toBeInTheDocument();
    });
  });

  it("shows Part Not Found for missing part", async () => {
    mockGetPart.mockRejectedValue(new Error("Not found"));
    render(<PartDetail />);
    await waitFor(() => {
      expect(screen.getByText("Part Not Found")).toBeInTheDocument();
    });
    expect(screen.getByText(/could not be found/)).toBeInTheDocument();
  });
});

describe("PartDetail - BOM tree expand/collapse", () => {
  beforeEach(() => {
    mockIPN = "PCA-100";
    mockGetPart.mockResolvedValue({
      ipn: "PCA-100",
      fields: { _category: "Assemblies", description: "Main Board", status: "active" },
    });
    mockGetPartBOM.mockResolvedValue({
      ipn: "PCA-100",
      description: "Main Board",
      qty: 1,
      children: [
        {
          ipn: "SUB-001",
          description: "Sub Assembly",
          qty: 1,
          children: [
            { ipn: "IPN-DEEP", description: "Deep Part", qty: 3, children: [] },
          ],
        },
      ],
    });
    mockGetPartCost.mockResolvedValue({ ipn: "PCA-100", bom_cost: 25.0 });
  });

  it("collapses and expands BOM children on toggle click", async () => {
    render(<PartDetail />);
    await waitFor(() => expect(screen.getByText("SUB-001")).toBeInTheDocument());
    // Level 0 and 1 auto-expand (level < 2), so IPN-DEEP should be visible
    expect(screen.getByText("IPN-DEEP")).toBeInTheDocument();

    // Click the SUB-001 row to collapse it
    fireEvent.click(screen.getByText("Sub Assembly"));
    expect(screen.queryByText("IPN-DEEP")).not.toBeInTheDocument();

    // Click again to expand
    fireEvent.click(screen.getByText("Sub Assembly"));
    expect(screen.getByText("IPN-DEEP")).toBeInTheDocument();
  });
});

describe("PartDetail - cost fetch error", () => {
  it("handles cost API rejection gracefully without crashing", async () => {
    mockGetPart.mockResolvedValue({ ipn: "IPN-003", fields: { description: "Test" } });
    mockGetPartCost.mockRejectedValue(new Error("cost fail"));
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    render(<PartDetail />);
    await waitFor(() => expect(screen.getByText("Cost Information")).toBeInTheDocument());
    // No cost data from part fields or API → shows fallback
    expect(screen.getByText("No cost information available")).toBeInTheDocument();
    consoleSpy.mockRestore();
  });
});

describe("PartDetail - IPN URL decoding with special characters", () => {
  it("decodes special characters in IPN", async () => {
    mockIPN = "IPN%2F003%20%26%20004";
    mockGetPart.mockResolvedValue({
      ipn: "IPN/003 & 004",
      fields: { description: "Special part", status: "active" },
    });
    mockGetPartCost.mockResolvedValue({ ipn: "IPN/003 & 004" });
    render(<PartDetail />);
    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("IPN/003 & 004"));
    expect(mockGetPart).toHaveBeenCalledWith("IPN/003 & 004");
  });
});

describe("PartDetail - bom_cost=0 hides BOM Cost Rollup", () => {
  it("does not show BOM Cost Rollup when bom_cost is 0", async () => {
    mockGetPartCost.mockResolvedValue({ ipn: "IPN-003", last_unit_price: 4.50, bom_cost: 0 });
    render(<PartDetail />);
    await waitForLoad();
    expect(screen.queryByText("BOM Cost Rollup")).not.toBeInTheDocument();
  });
});

describe("PartDetail - no description", () => {
  it("shows 'No description available' when part has no description", async () => {
    mockGetPart.mockResolvedValue({
      ipn: "IPN-003",
      fields: { _category: "ICs", status: "active" },
    });
    render(<PartDetail />);
    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("IPN-003"));
    expect(screen.getByText("No description available")).toBeInTheDocument();
  });
});

describe("PartDetail - ASY prefix triggers BOM fetch", () => {
  beforeEach(() => {
    mockIPN = "ASY-200";
    mockGetPart.mockResolvedValue({
      ipn: "ASY-200",
      fields: { _category: "Assemblies", description: "Assembly Unit", status: "active" },
    });
    mockGetPartBOM.mockResolvedValue(mockBOM);
    mockGetPartCost.mockResolvedValue({ ipn: "ASY-200", bom_cost: 10.0 });
  });

  it("fetches BOM for ASY- prefixed IPN", async () => {
    render(<PartDetail />);
    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("ASY-200"));
    expect(mockGetPartBOM).toHaveBeenCalledWith("ASY-200");
    await waitFor(() => expect(screen.getByText("Bill of Materials")).toBeInTheDocument());
  });
});

describe("PartDetail - non-assembly does NOT show BOM", () => {
  beforeEach(() => {
    mockIPN = "RES-100";
    mockGetPart.mockResolvedValue({
      ipn: "RES-100",
      fields: { _category: "Resistors", description: "1k Resistor", status: "active" },
    });
    mockGetPartCost.mockResolvedValue({ ipn: "RES-100" });
  });

  it("does not show BOM section for non-assembly IPN", async () => {
    render(<PartDetail />);
    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("RES-100"));
    expect(screen.queryByText("Bill of Materials")).not.toBeInTheDocument();
    expect(mockGetPartBOM).not.toHaveBeenCalled();
  });
});

describe("PartDetail - BOM (assembly IPN)", () => {
  beforeEach(() => {
    mockIPN = "PCA-100";
    mockGetPart.mockResolvedValue({
      ipn: "PCA-100",
      fields: { _category: "Assemblies", description: "Main Board", status: "active" },
    });
    mockGetPartBOM.mockResolvedValue(mockBOM);
    mockGetPartCost.mockResolvedValue({ ipn: "PCA-100", bom_cost: 25.0 });
  });

  const waitForAssembly = () => waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("PCA-100"));

  it("renders Bill of Materials card for assembly", async () => {
    render(<PartDetail />);
    await waitForAssembly();
    await waitFor(() => {
      expect(screen.getByText("Bill of Materials")).toBeInTheDocument();
    });
  });

  it("renders BOM children IPNs", async () => {
    render(<PartDetail />);
    await waitForAssembly();
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
      expect(screen.getByText("IPN-002")).toBeInTheDocument();
    });
  });

  it("renders BOM quantities", async () => {
    render(<PartDetail />);
    await waitForAssembly();
    await waitFor(() => {
      expect(screen.getByText("Qty: 4")).toBeInTheDocument();
      expect(screen.getByText("Qty: 2")).toBeInTheDocument();
    });
  });

  it("renders BOM reference designators", async () => {
    render(<PartDetail />);
    await waitForAssembly();
    await waitFor(() => {
      expect(screen.getByText("R1-R4")).toBeInTheDocument();
      expect(screen.getByText("C1-C2")).toBeInTheDocument();
    });
  });

  it("clicking BOM part navigates to that part", async () => {
    render(<PartDetail />);
    await waitForAssembly();
    await waitFor(() => expect(screen.getByText("IPN-001")).toBeInTheDocument());
    // Click on IPN-001 in the BOM tree
    fireEvent.click(screen.getByText("IPN-001"));
    expect(mockNavigate).toHaveBeenCalledWith("/parts/IPN-001");
  });

  it("shows no BOM data message when BOM fetch fails", async () => {
    mockGetPartBOM.mockRejectedValue(new Error("fail"));
    render(<PartDetail />);
    await waitForAssembly();
    await waitFor(() => {
      expect(screen.getByText("No BOM data available for this assembly")).toBeInTheDocument();
    });
  });

  // --- Where-Used Tests ---

  it("renders where-used table with parent assemblies", async () => {
    mockGetPartWhereUsed.mockResolvedValue([
      { assembly_ipn: "PCA-100", description: "Main Board", qty: 4, ref: "R1-R4" },
      { assembly_ipn: "ASY-200", description: "Power Supply", qty: 2, ref: "R5-R6" },
    ]);
    render(<PartDetail />);
    await waitForLoad();
    await waitFor(() => {
      expect(screen.getAllByText("PCA-100").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("Main Board").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("ASY-200").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("Power Supply").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("renders where-used qty and ref designators", async () => {
    mockGetPartWhereUsed.mockResolvedValue([
      { assembly_ipn: "PCA-100", description: "Main Board", qty: 4, ref: "R1-R4" },
    ]);
    render(<PartDetail />);
    await waitForLoad();
    await waitFor(() => {
      expect(screen.getAllByText("4").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("R1-R4").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows empty state when no where-used entries", async () => {
    mockGetPartWhereUsed.mockResolvedValue([]);
    render(<PartDetail />);
    await waitForLoad();
    await waitFor(() => {
      expect(screen.getByText("Where Used")).toBeInTheDocument();
    });
    // Should show the "not used in any assemblies" message
    expect(screen.getByText(/not used in any/i)).toBeInTheDocument();
  });

  it("where-used assembly links navigate to part detail", async () => {
    mockGetPartWhereUsed.mockResolvedValue([
      { assembly_ipn: "PCA-100", description: "Main Board", qty: 4, ref: "R1-R4" },
    ]);
    render(<PartDetail />);
    await waitForLoad();
    await waitFor(() => {
      const links = screen.getAllByText("PCA-100");
      const linkEl = links.find(el => el.closest("a"));
      expect(linkEl?.closest("a")).toHaveAttribute("href", "/parts/PCA-100");
    });
  });
});
