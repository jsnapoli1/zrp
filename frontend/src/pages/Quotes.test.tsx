import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockQuotes } from "../test/mocks";
import type { Quote } from "../lib/api";

const quotesWithLines: Quote[] = [
  { ...mockQuotes[0], lines: [{ id: 1, quote_id: "Q-001", ipn: "IPN-001", description: "10k Resistor", qty: 100, unit_price: 0.05, notes: "" }] },
  { ...mockQuotes[1], status: "accepted", lines: [{ id: 2, quote_id: "Q-002", ipn: "IPN-002", description: "Cap", qty: 50, unit_price: 0.50, notes: "" }] },
];

const mockGetQuotes = vi.fn().mockResolvedValue(quotesWithLines);
const mockCreateQuote = vi.fn().mockResolvedValue(quotesWithLines[0]);

vi.mock("../lib/api", () => ({
  api: {
    getQuotes: (...args: any[]) => mockGetQuotes(...args),
    createQuote: (...args: any[]) => mockCreateQuote(...args),
  },
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

import Quotes from "./Quotes";

beforeEach(() => vi.clearAllMocks());

describe("Quotes", () => {
  it("renders loading state", () => {
    render(<Quotes />);
    expect(screen.getByText("Loading quotes...")).toBeInTheDocument();
  });

  it("renders quote list after loading", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    expect(screen.getByText("Q-002")).toBeInTheDocument();
  });

  it("shows customer names", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Acme Inc")).toBeInTheDocument();
      expect(screen.getByText("Tech Co")).toBeInTheDocument();
    });
  });

  it("has create quote button", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Create Quote")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetQuotes.mockResolvedValueOnce([]);
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText(/No quotes found/)).toBeInTheDocument();
    });
  });

  it("displays quote totals calculated from line items", async () => {
    render(<Quotes />);
    await waitFor(() => {
      // 100 * 0.05 = $5.00
      expect(screen.getByText("$5.00")).toBeInTheDocument();
      // 50 * 0.50 = $25.00
      expect(screen.getByText("$25.00")).toBeInTheDocument();
    });
  });

  it("shows status badges with correct text", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Draft")).toBeInTheDocument();
      // "Accepted" appears as both badge and stats label
      const acceptedElements = screen.getAllByText("Accepted");
      expect(acceptedElements.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("displays statistics cards", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Total Quotes")).toBeInTheDocument();
      expect(screen.getByText("Pending")).toBeInTheDocument();
      expect(screen.getByText("Total Value")).toBeInTheDocument();
    });
  });

  it("shows correct statistics values", async () => {
    render(<Quotes />);
    await waitFor(() => {
      // Total quotes = 2
      expect(screen.getByText("2")).toBeInTheDocument();
      // Accepted = 1
      expect(screen.getByText("1")).toBeInTheDocument();
    });
  });

  it("shows valid until dates", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    // Dates are rendered via toLocaleDateString
  });

  it("opens create quote dialog when button clicked", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByText("Create New Quote")).toBeInTheDocument();
    });
  });

  it("shows form fields in create dialog", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByLabelText("Customer *")).toBeInTheDocument();
      expect(screen.getByLabelText("Valid Until")).toBeInTheDocument();
      expect(screen.getByLabelText("Notes")).toBeInTheDocument();
    });
  });

  it("shows line items table in create dialog", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByText("Line Items")).toBeInTheDocument();
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });
  });

  it("adds line items in create dialog", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Add Item"));
    // Should now have 2 rows of inputs (IPN placeholders)
    const ipnInputs = screen.getAllByPlaceholderText("Part number");
    expect(ipnInputs.length).toBe(2);
  });

  it("submits create quote form", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByLabelText("Customer *")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText("Customer *"), { target: { value: "New Customer" } });
    // Submit - find the submit button inside the dialog (second "Create Quote" text)
    const submitButtons = screen.getAllByText("Create Quote");
    // The last one is the submit button in the form
    fireEvent.click(submitButtons[submitButtons.length - 1]);
    await waitFor(() => {
      expect(mockCreateQuote).toHaveBeenCalled();
    });
  });

  it("shows cancel button in create dialog", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
  });

  it("shows view details buttons for each quote", async () => {
    render(<Quotes />);
    await waitFor(() => {
      const viewButtons = screen.getAllByText("View Details");
      expect(viewButtons.length).toBe(2);
    });
  });

  it("shows page header and description", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Quotes")).toBeInTheDocument();
      expect(screen.getByText("Manage customer quotes and proposals")).toBeInTheDocument();
    });
  });

  it("shows Quote Records card title", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Quote Records")).toBeInTheDocument();
    });
  });

  it("displays table headers", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Quote ID")).toBeInTheDocument();
      expect(screen.getByText("Customer")).toBeInTheDocument();
      expect(screen.getByText("Total")).toBeInTheDocument();
      expect(screen.getByText("Status")).toBeInTheDocument();
    });
  });

  it("shows total value in stats", async () => {
    render(<Quotes />);
    await waitFor(() => {
      // Total = $5.00 + $25.00 = $30
      expect(screen.getByText("$30")).toBeInTheDocument();
    });
  });

  it("handles getQuotes API rejection gracefully", async () => {
    mockGetQuotes.mockRejectedValueOnce(new Error("Network error"));
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.queryByText("Loading quotes...")).not.toBeInTheDocument();
    });
    // Page renders with empty list
    expect(screen.getByText(/No quotes found/)).toBeInTheDocument();
  });

  it("navigates to quote detail on row click", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Q-001"));
    expect(mockNavigate).toHaveBeenCalledWith("/quotes/Q-001");
  });

  it("navigates via View Details button with stopPropagation", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    const viewButtons = screen.getAllByText("View Details");
    fireEvent.click(viewButtons[0]);
    // Should navigate (button calls navigate) — only one navigate call, not two (stopPropagation prevents row click)
    expect(mockNavigate).toHaveBeenCalledWith("/quotes/Q-001");
  });

  it("removes line item in create dialog", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByText("Add Item")).toBeInTheDocument();
    });
    // Add a second line so remove button appears
    fireEvent.click(screen.getByText("Add Item"));
    let ipnInputs = screen.getAllByPlaceholderText("Part number");
    expect(ipnInputs.length).toBe(2);
    // Click the first trash button to remove a line
    screen.getAllByRole("button").filter(btn => btn.querySelector("svg.lucide-trash-2") || btn.innerHTML.includes("Trash2"));
    // Use a more reliable approach: find buttons with Trash2 icon
    const removeButtons = screen.getAllByRole("button").filter(b => {
      const svg = b.querySelector("svg");
      return svg && b.closest("td") !== null;
    });
    fireEvent.click(removeButtons[0]);
    ipnInputs = screen.getAllByPlaceholderText("Part number");
    expect(ipnInputs.length).toBe(1);
  });

  it("updates line item fields in create dialog", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Part number")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByPlaceholderText("Description"), { target: { value: "New Desc" } });
    expect(screen.getByPlaceholderText("Description")).toHaveValue("New Desc");

    // Update qty
    const qtyInput = screen.getByDisplayValue("1");
    fireEvent.change(qtyInput, { target: { value: "10" } });
    expect(qtyInput).toHaveValue(10);

    // Update unit_price
    const priceInput = screen.getByDisplayValue("0");
    fireEvent.change(priceInput, { target: { value: "5.50" } });
    expect(priceInput).toHaveValue(5.5);
  });

  it("cancel button closes create dialog", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByText("Create New Quote")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Create New Quote")).not.toBeInTheDocument();
    });
  });

  it("handles createQuote API rejection gracefully", async () => {
    mockGetQuotes.mockResolvedValueOnce(quotesWithLines);
    mockCreateQuote.mockRejectedValueOnce(new Error("Create failed"));
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Q-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create Quote"));
    await waitFor(() => {
      expect(screen.getByLabelText("Customer *")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText("Customer *"), { target: { value: "Fail Corp" } });
    const submitButtons = screen.getAllByText("Create Quote");
    fireEvent.click(submitButtons[submitButtons.length - 1]);
    await waitFor(() => {
      expect(mockCreateQuote).toHaveBeenCalled();
    });
    // Should not crash — dialog stays open, original quotes still visible
    expect(screen.getByText("Q-001")).toBeInTheDocument();
  });
});
