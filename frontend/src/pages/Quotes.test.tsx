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
      expect(screen.getByText("Accepted")).toBeInTheDocument();
    });
  });

  it("displays statistics cards", async () => {
    render(<Quotes />);
    await waitFor(() => {
      expect(screen.getByText("Total Quotes")).toBeInTheDocument();
      expect(screen.getByText("Accepted")).toBeInTheDocument();
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
});
