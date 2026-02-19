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
    status: "active",
  },
};

const mockMarketPricing = {
  results: [
    {
      id: 1,
      part_ipn: "IPN-003",
      mpn: "STM32F401",
      distributor: "Digikey",
      distributor_pn: "DK-STM32F401-ND",
      manufacturer: "STMicro",
      description: "STM32F401 (Digikey)",
      stock_qty: 15000,
      lead_time_days: 14,
      currency: "USD",
      price_breaks: [
        { qty: 1, unit_price: 5.50 },
        { qty: 10, unit_price: 4.95 },
        { qty: 100, unit_price: 4.40 },
      ],
      product_url: "https://www.digikey.com/product-detail/STM32F401",
      datasheet_url: "https://www.digikey.com/datasheet/STM32F401.pdf",
      fetched_at: "2026-01-01T00:00:00Z",
    },
    {
      id: 2,
      part_ipn: "IPN-003",
      mpn: "STM32F401",
      distributor: "Mouser",
      distributor_pn: "MOU-STM32F401",
      manufacturer: "STMicro",
      description: "STM32F401 (Mouser)",
      stock_qty: 8000,
      lead_time_days: 10,
      currency: "USD",
      price_breaks: [
        { qty: 1, unit_price: 5.80 },
        { qty: 10, unit_price: 5.10 },
      ],
      product_url: "https://www.mouser.com/ProductDetail/STM32F401",
      datasheet_url: "",
      fetched_at: "2026-01-01T00:00:00Z",
    },
  ],
  cached: false,
};

const mockGetPart = vi.fn();
const mockGetPartBOM = vi.fn();
const mockGetPartCost = vi.fn();
const mockGetPartWhereUsed = vi.fn();
const mockGetMarketPricing = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getPart: (...args: unknown[]) => mockGetPart(...args),
    getPartBOM: (...args: unknown[]) => mockGetPartBOM(...args),
    getPartCost: (...args: unknown[]) => mockGetPartCost(...args),
    getPartWhereUsed: (...args: unknown[]) => mockGetPartWhereUsed(...args),
    getMarketPricing: (...args: unknown[]) => mockGetMarketPricing(...args),
    getGitPLMConfig: () => Promise.resolve({ base_url: "" }),
  },
}));

// Must import after mocks
const { default: PartDetail } = await import("./PartDetail");

describe("Market Pricing on PartDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetPart.mockResolvedValue(mockPart);
    mockGetPartBOM.mockRejectedValue(new Error("no bom"));
    mockGetPartCost.mockResolvedValue({ ipn: "IPN-003", last_unit_price: 0, bom_cost: 0 });
    mockGetPartWhereUsed.mockResolvedValue([]);
    mockGetMarketPricing.mockResolvedValue(mockMarketPricing);
  });

  it("renders market pricing section with distributor data", async () => {
    render(<PartDetail />);

    await waitFor(() => {
      expect(screen.getByText("Market Pricing")).toBeInTheDocument();
    });

    expect(screen.getByText("Digikey")).toBeInTheDocument();
    expect(screen.getByText("Mouser")).toBeInTheDocument();
    expect(screen.getByText("DK-STM32F401-ND")).toBeInTheDocument();
    expect(screen.getByText("15,000")).toBeInTheDocument();
    expect(screen.getByText("$5.5000")).toBeInTheDocument();
  });

  it("shows Refresh button", async () => {
    render(<PartDetail />);

    await waitFor(() => {
      expect(screen.getByTestId("refresh-market-pricing")).toBeInTheDocument();
    });
  });

  it("calls API with refresh=true when Refresh clicked", async () => {
    render(<PartDetail />);

    await waitFor(() => {
      expect(screen.getByText("Market Pricing")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("refresh-market-pricing"));

    await waitFor(() => {
      expect(mockGetMarketPricing).toHaveBeenCalledWith("IPN-003", true);
    });
  });

  it("shows cached badge when results are cached", async () => {
    mockGetMarketPricing.mockResolvedValue({ ...mockMarketPricing, cached: true });
    render(<PartDetail />);

    await waitFor(() => {
      expect(screen.getByText("Cached")).toBeInTheDocument();
    });
  });

  it("shows links to distributor pages", async () => {
    render(<PartDetail />);

    await waitFor(() => {
      expect(screen.getByText("View on Digikey →")).toBeInTheDocument();
      expect(screen.getByText("View on Mouser →")).toBeInTheDocument();
    });
  });
});

describe("DistributorSettings page", () => {
  const mockGetDistributorSettings = vi.fn();
  const mockUpdateDigikeySettings = vi.fn();
  const mockUpdateMouserSettings = vi.fn();

  vi.mock("../lib/api", () => ({
    api: {
      getPart: (...args: unknown[]) => mockGetPart(...args),
      getPartBOM: (...args: unknown[]) => mockGetPartBOM(...args),
      getPartCost: (...args: unknown[]) => mockGetPartCost(...args),
      getPartWhereUsed: (...args: unknown[]) => mockGetPartWhereUsed(...args),
      getMarketPricing: (...args: unknown[]) => mockGetMarketPricing(...args),
      getDistributorSettings: (...args: unknown[]) => mockGetDistributorSettings(...args),
      updateDigikeySettings: (...args: unknown[]) => mockUpdateDigikeySettings(...args),
      updateMouserSettings: (...args: unknown[]) => mockUpdateMouserSettings(...args),
    },
  }));

  it("loads and displays settings page", async () => {
    mockGetDistributorSettings.mockResolvedValue({
      digikey: { api_key: "dk-1****key1", client_id: "cid-****cid1" },
      mouser: { api_key: "mou-****ey89" },
    });

    const { default: DistributorSettings } = await import("./DistributorSettings");
    render(<DistributorSettings />);

    await waitFor(() => {
      expect(screen.getByText("Distributor API Settings")).toBeInTheDocument();
      expect(screen.getByText("Digikey")).toBeInTheDocument();
      expect(screen.getByText("Mouser")).toBeInTheDocument();
    });
  });
});
