import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockQuotes, mockParts } from "../test/mocks";
import type { Quote, QuoteLine } from "../lib/api";

const mockLines: QuoteLine[] = [
  { id: 1, quote_id: "Q-001", ipn: "IPN-001", description: "10k Resistor", qty: 100, unit_price: 0.05, notes: "" },
  { id: 2, quote_id: "Q-001", ipn: "IPN-002", description: "100uF Cap", qty: 50, unit_price: 0.50, notes: "" },
];

const mockQuoteWithLines: Quote = {
  ...mockQuotes[0],
  status: "draft",
  notes: "Test notes",
  lines: mockLines,
};

const mockGetQuote = vi.fn().mockResolvedValue(mockQuoteWithLines);
const mockGetParts = vi.fn().mockResolvedValue({ data: mockParts, meta: { total: 3, page: 1, limit: 50 } });
const mockUpdateQuote = vi.fn().mockResolvedValue(mockQuoteWithLines);
const mockExportQuotePDF = vi.fn().mockResolvedValue(new Blob(["pdf"], { type: "application/pdf" }));

vi.mock("../lib/api", () => ({
  api: {
    getQuote: (...args: any[]) => mockGetQuote(...args),
    getParts: (...args: any[]) => mockGetParts(...args),
    updateQuote: (...args: any[]) => mockUpdateQuote(...args),
    exportQuotePDF: (...args: any[]) => mockExportQuotePDF(...args),
  },
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "Q-001" }),
  };
});

import QuoteDetail from "./QuoteDetail";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetQuote.mockResolvedValue(mockQuoteWithLines);
  mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 3, page: 1, limit: 50 } });
});

describe("QuoteDetail", () => {
  it("renders loading state", () => {
    render(<QuoteDetail />);
    expect(screen.getByText("Loading quote...")).toBeInTheDocument();
  });

  it("renders quote details after loading", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    // Acme Inc appears in header subtitle and customer field
    const acmeElements = screen.getAllByText("Acme Inc");
    expect(acmeElements.length).toBeGreaterThanOrEqual(1);
  });

  it("shows quote status badge", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Draft")).toBeInTheDocument();
    });
  });

  it("shows quote notes", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Test notes")).toBeInTheDocument();
    });
  });

  it("displays line items table", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Line Items")).toBeInTheDocument();
      expect(screen.getByText("10k Resistor")).toBeInTheDocument();
      expect(screen.getByText("100uF Cap")).toBeInTheDocument();
    });
  });

  it("shows line item quantities", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("100")).toBeInTheDocument();
      expect(screen.getByText("50")).toBeInTheDocument();
    });
  });

  it("shows unit prices", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("$0.05")).toBeInTheDocument();
      expect(screen.getByText("$0.50")).toBeInTheDocument();
    });
  });

  it("calculates line totals", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      // 100 * 0.05 = $5.00
      expect(screen.getByText("$5.00")).toBeInTheDocument();
      // 50 * 0.50 = $25.00
      expect(screen.getByText("$25.00")).toBeInTheDocument();
    });
  });

  it("shows quote summary with totals", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Quote Summary")).toBeInTheDocument();
    });
    expect(screen.getByText("Total Quoted")).toBeInTheDocument();
    expect(screen.getByText("Total Cost")).toBeInTheDocument();
    // "Margin" appears in both summary and line item column header
    const marginElements = screen.getAllByText("Margin");
    expect(marginElements.length).toBeGreaterThanOrEqual(1);
  });

  it("calculates total quoted in summary", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      // Total = $5.00 + $25.00 = $30.00
      expect(screen.getByText("$30.00")).toBeInTheDocument();
    });
  });

  it("calculates margin from BOM cost", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      // Cost: IPN-001 cost=0.01*100 + IPN-002 cost=0.10*50 = 1.00 + 5.00 = $6.00
      // Margin: $30.00 - $6.00 = $24.00
      expect(screen.getByText("$24.00")).toBeInTheDocument();
    });
  });

  it("shows margin percentage", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      // (24/30)*100 = 80.0%
      expect(screen.getByText("(80.0% margin)")).toBeInTheDocument();
    });
  });

  it("shows unit cost from parts data", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      // IPN-001 cost = 0.01
      expect(screen.getByText("$0.01")).toBeInTheDocument();
      // IPN-002 cost = 0.10
      expect(screen.getByText("$0.10")).toBeInTheDocument();
    });
  });

  it("shows per-line margin", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      // Line 1 margin: $5.00 - $1.00 = $4.00
      expect(screen.getByText("$4.00")).toBeInTheDocument();
    });
  });

  it("has Export PDF button", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Export PDF")).toBeInTheDocument();
    });
  });

  it("has Download PDF button in sidebar", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Download PDF")).toBeInTheDocument();
    });
  });

  it("calls exportQuotePDF when Export PDF clicked", async () => {
    // Mock URL methods
    window.URL.createObjectURL = vi.fn().mockReturnValue("blob:url");
    window.URL.revokeObjectURL = vi.fn();

    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Export PDF")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Export PDF"));
    await waitFor(() => {
      expect(mockExportQuotePDF).toHaveBeenCalledWith("Q-001");
    });
  });

  it("has Edit Quote button", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
  });

  it("enters edit mode when Edit Quote clicked", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit Quote"));
    await waitFor(() => {
      expect(screen.getByText("Save Changes")).toBeInTheDocument();
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
  });

  it("shows status dropdown in edit mode", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit Quote"));
    await waitFor(() => {
      // Status select has options
      expect(screen.getByRole("combobox")).toBeInTheDocument();
    });
  });

  it("shows Add Item button in edit mode", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit Quote"));
    await waitFor(() => {
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });
  });

  it("saves changes when Save Changes clicked", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit Quote"));
    await waitFor(() => {
      expect(screen.getByText("Save Changes")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Save Changes"));
    await waitFor(() => {
      expect(mockUpdateQuote).toHaveBeenCalledWith("Q-001", expect.any(Object));
    });
  });

  it("cancels edit mode", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit Quote"));
    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
  });

  it("shows Quote Not Found when quote is null", async () => {
    mockGetQuote.mockResolvedValueOnce(null);
    // Need to handle the getParts call too
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Quote Not Found")).toBeInTheDocument();
    });
  });

  it("shows Back to Quotes button", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Back to Quotes")).toBeInTheDocument();
    });
  });

  it("shows Timeline card", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Timeline")).toBeInTheDocument();
      expect(screen.getByText("Created")).toBeInTheDocument();
    });
  });

  it("shows Actions card", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Actions")).toBeInTheDocument();
    });
  });

  it("shows Quote Details card title", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Quote Details")).toBeInTheDocument();
    });
  });

  it("shows table headers for line items", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN")).toBeInTheDocument();
      expect(screen.getByText("Description")).toBeInTheDocument();
      expect(screen.getByText("Qty")).toBeInTheDocument();
      expect(screen.getByText("Unit Cost")).toBeInTheDocument();
      expect(screen.getByText("Unit Price")).toBeInTheDocument();
      expect(screen.getByText("Line Total")).toBeInTheDocument();
    });
  });

  it("displays IPN values in line items", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
      expect(screen.getByText("IPN-002")).toBeInTheDocument();
    });
  });

  it("shows valid until or not specified", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      // valid_until is set on mockQuotes[0]
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
  });

  it("shows accepted_at as dash when not set", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Accepted At")).toBeInTheDocument();
      expect(screen.getByText("—")).toBeInTheDocument();
    });
  });

  it("handles getQuote API rejection gracefully", async () => {
    mockGetQuote.mockRejectedValueOnce(new Error("Network error"));
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Quote Not Found")).toBeInTheDocument();
    });
  });

  it("handles handleSave API rejection gracefully", async () => {
    mockUpdateQuote.mockRejectedValueOnce(new Error("Save failed"));
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit Quote"));
    await waitFor(() => {
      expect(screen.getByText("Save Changes")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Save Changes"));
    await waitFor(() => {
      expect(mockUpdateQuote).toHaveBeenCalled();
    });
    // Should not crash — still in edit mode or page intact
    expect(screen.getByText("Q-001")).toBeInTheDocument();
  });

  it("adds line item in edit mode", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit Quote"));
    await waitFor(() => {
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });
    // Currently 2 lines in edit mode
    const ipnInputsBefore = screen.getAllByDisplayValue(/IPN-00/);
    expect(ipnInputsBefore.length).toBe(2);
    fireEvent.click(screen.getByText("Add Item"));
    // Should now have 3 rows with IPN inputs (2 existing + 1 empty)
    const allIpnInputs = screen.getAllByRole("textbox").filter(
      (input: HTMLElement) => (input as HTMLInputElement).className.includes("w-28")
    );
    expect(allIpnInputs.length).toBe(3);
  });

  it("removes line item in edit mode", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit Quote"));
    await waitFor(() => {
      expect(screen.getByText("Save Changes")).toBeInTheDocument();
    });
    // Find remove buttons (in table cells with Trash2 icon)
    const removeButtons = screen.getAllByRole("button").filter(b => {
      const svg = b.querySelector("svg");
      return svg && b.closest("td") !== null;
    });
    expect(removeButtons.length).toBe(2); // both lines have remove since length > 1
    fireEvent.click(removeButtons[0]);
    // Should now have 1 line
    const remainingRemoveButtons = screen.getAllByRole("button").filter(b => {
      const svg = b.querySelector("svg");
      return svg && b.closest("td") !== null;
    });
    // With only 1 line, remove button is hidden
    expect(remainingRemoveButtons.length).toBe(0);
  });

  it("updates line item fields in edit mode", async () => {
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit Quote"));
    await waitFor(() => {
      expect(screen.getByText("Save Changes")).toBeInTheDocument();
    });
    // Update description of first line
    const descInputs = screen.getAllByDisplayValue("10k Resistor");
    fireEvent.change(descInputs[0], { target: { value: "Updated Resistor" } });
    expect(screen.getByDisplayValue("Updated Resistor")).toBeInTheDocument();
  });

  it("shows 0% margin for zero-price line without division error", async () => {
    const zeroLine: QuoteLine = { id: 3, quote_id: "Q-001", ipn: "IPN-001", description: "Free item", qty: 10, unit_price: 0, notes: "" };
    mockGetQuote.mockResolvedValueOnce({ ...mockQuoteWithLines, lines: [zeroLine] });
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Free item")).toBeInTheDocument();
    });
    // Line total is $0.00, margin % should be 0.0% not NaN or Infinity
    expect(screen.getByText("(0.0%)")).toBeInTheDocument();
    // No NaN anywhere
    expect(screen.queryByText(/NaN/)).not.toBeInTheDocument();
  });

  it("shows negative margin in red when selling below cost", async () => {
    // IPN-003 cost is 5.00, selling at 2.00 → negative margin
    const lossLine: QuoteLine = { id: 4, quote_id: "Q-001", ipn: "IPN-003", description: "MCU below cost", qty: 1, unit_price: 2.00, notes: "" };
    mockGetQuote.mockResolvedValueOnce({ ...mockQuoteWithLines, lines: [lossLine] });
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("MCU below cost")).toBeInTheDocument();
    });
    // Line margin: $2.00 - $5.00 = -$3.00 — shown with red styling
    const negativeMarginEls = screen.getAllByText("$-3.00");
    expect(negativeMarginEls.length).toBeGreaterThanOrEqual(1);
    // At least one should have red styling
    const hasRed = negativeMarginEls.some(el => el.closest("div")?.className.includes("text-red-600"));
    expect(hasRed).toBe(true);
  });

  it("getPartCost returns 0 for unknown IPN", async () => {
    const unknownLine: QuoteLine = { id: 5, quote_id: "Q-001", ipn: "UNKNOWN-999", description: "Unknown part", qty: 5, unit_price: 10.00, notes: "" };
    mockGetQuote.mockResolvedValueOnce({ ...mockQuoteWithLines, lines: [unknownLine] });
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Unknown part")).toBeInTheDocument();
    });
    // Unit cost column should show $0.00 for unknown IPN
    // Total cost in summary should also be $0.00
    const zeroCosts = screen.getAllByText("$0.00");
    expect(zeroCosts.length).toBeGreaterThanOrEqual(1);
  });

  it("handles handleExportPDF API rejection gracefully", async () => {
    mockExportQuotePDF.mockRejectedValueOnce(new Error("PDF export failed"));
    render(<QuoteDetail />);
    await waitFor(() => {
      expect(screen.getByText("Export PDF")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Export PDF"));
    await waitFor(() => {
      expect(mockExportQuotePDF).toHaveBeenCalledWith("Q-001");
    });
    // Should not crash
    expect(screen.getByText("Q-001")).toBeInTheDocument();
  });
});
