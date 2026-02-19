import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockParts, mockCategories } from "../test/mocks";

const mockGetParts = vi.fn();
const mockGetCategories = vi.fn();
const mockCreatePart = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getParts: (...args: any[]) => mockGetParts(...args),
    getCategories: (...args: any[]) => mockGetCategories(...args),
    createPart: (...args: any[]) => mockCreatePart(...args),
    deletePart: vi.fn().mockResolvedValue(undefined),
  },
}));

import Parts from "./Parts";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 3, page: 1, limit: 50 } });
  mockGetCategories.mockResolvedValue(mockCategories);
  mockCreatePart.mockResolvedValue(mockParts[0]);
});

// Helper: wait for parts to load
const waitForLoad = () => waitFor(() => expect(screen.getByText("IPN-001")).toBeInTheDocument());

describe("Parts", () => {
  it("renders page title and subtitle", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Parts")).toBeInTheDocument();
    expect(screen.getByText("Manage your parts inventory and specifications")).toBeInTheDocument();
  });

  it("renders parts table after loading", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("IPN-002")).toBeInTheDocument();
    expect(screen.getByText("IPN-003")).toBeInTheDocument();
  });

  it("shows loading skeletons initially", () => {
    mockGetParts.mockReturnValue(new Promise(() => {}));
    render(<Parts />);
    expect(screen.queryByText("IPN-001")).not.toBeInTheDocument();
  });

  it("has search input with placeholder", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByPlaceholderText(/search parts by ipn/i)).toBeInTheDocument();
  });

  it("has add part button", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Add Part")).toBeInTheDocument();
  });

  it("opens create dialog on button click", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByText("Add New Part")).toBeInTheDocument();
      expect(screen.getByText("Create a new part in your inventory system.")).toBeInTheDocument();
    });
  });

  it("shows parts count", async () => {
    render(<Parts />);
    await waitForLoad();
    // Text nodes split: find container with "Parts (3)"
    const el = screen.getByText((_, element) => {
      if (!element || element.tagName === 'H1') return false;
      const text = element.textContent || '';
      return text.includes('Parts (') && text.includes('3') && element.classList?.contains('font-semibold');
    });
    expect(el).toBeInTheDocument();
  });

  it("shows empty state when no parts", async () => {
    mockGetParts.mockResolvedValue({ data: [], meta: { total: 0, page: 1, limit: 50 } });
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText(/no parts available/i)).toBeInTheDocument();
    });
  });

  it("calls getParts and getCategories on mount", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(mockGetParts).toHaveBeenCalled();
    expect(mockGetCategories).toHaveBeenCalled();
  });

  it("handles API error gracefully", async () => {
    mockGetParts.mockRejectedValue(new Error("fail"));
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("Parts")).toBeInTheDocument();
    });
  });

  it("renders Filters card", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Filters")).toBeInTheDocument();
  });

  it("shows table headers", async () => {
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("IPN")).toBeInTheDocument();
    expect(screen.getByText("Category")).toBeInTheDocument();
    expect(screen.getByText("Description")).toBeInTheDocument();
    expect(screen.getByText("Cost")).toBeInTheDocument();
    expect(screen.getByText("Stock")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
  });

  it("create dialog has required form fields", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByLabelText("IPN *")).toBeInTheDocument();
      expect(screen.getByLabelText("Description")).toBeInTheDocument();
      expect(screen.getByLabelText("Cost ($)")).toBeInTheDocument();
      expect(screen.getByLabelText("Price ($)")).toBeInTheDocument();
      expect(screen.getByLabelText("Minimum Stock")).toBeInTheDocument();
      expect(screen.getByLabelText("Current Stock")).toBeInTheDocument();
      expect(screen.getByLabelText("Location")).toBeInTheDocument();
      expect(screen.getByLabelText("Vendor")).toBeInTheDocument();
    });
  });

  it("create button disabled when IPN empty", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByText("Create Part")).toBeDisabled();
    });
  });

  it("create button enabled when IPN filled", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => expect(screen.getByLabelText("IPN *")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("IPN *"), { target: { value: "NEW-001" } });
    expect(screen.getByText("Create Part")).not.toBeDisabled();
  });

  it("submits create form and closes dialog", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => expect(screen.getByLabelText("IPN *")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("IPN *"), { target: { value: "NEW-001" } });
    fireEvent.change(screen.getByLabelText("Description"), { target: { value: "Test part" } });
    fireEvent.click(screen.getByText("Create Part"));
    await waitFor(() => {
      expect(mockCreatePart).toHaveBeenCalledWith(
        expect.objectContaining({ ipn: "NEW-001", description: "Test part" })
      );
    });
  });

  it("cancel button closes create dialog", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => expect(screen.getByText("Add New Part")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Add New Part")).not.toBeInTheDocument();
    });
  });

  it("shows page info", async () => {
    render(<Parts />);
    await waitForLoad();
    // Text may be split across elements
    const el = screen.getByText((_, element) => element?.textContent === 'Page 1 of 1' || false);
    expect(el).toBeInTheDocument();
  });

  it("shows pagination when more than one page", async () => {
    mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 100, page: 1, limit: 50 } });
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Previous")).toBeInTheDocument();
    expect(screen.getByText("Next")).toBeInTheDocument();
    expect(screen.getByText("Previous").closest("button")).toBeDisabled();
    expect(screen.getByText("Next").closest("button")).not.toBeDisabled();
  });

  it("shows showing X to Y of Z text for pagination", async () => {
    mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 100, page: 1, limit: 50 } });
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText(/Showing 1 to 50 of 100 parts/)).toBeInTheDocument();
  });

  it("shows filtered empty state message when search active", async () => {
    render(<Parts />);
    await waitForLoad();
    // Change search, triggering a re-fetch
    mockGetParts.mockResolvedValue({ data: [], meta: { total: 0, page: 1, limit: 50 } });
    const searchInput = screen.getByPlaceholderText(/search parts by ipn/i);
    fireEvent.change(searchInput, { target: { value: "nonexistent" } });
    await waitFor(() => {
      expect(screen.getByText("No parts found matching your criteria")).toBeInTheDocument();
    });
  });

  it("handles category error gracefully", async () => {
    mockGetCategories.mockRejectedValue(new Error("fail"));
    render(<Parts />);
    await waitForLoad();
    expect(screen.getByText("Parts")).toBeInTheDocument();
  });

  it("navigates to part detail on row click", async () => {
    // We can test by checking that clicking a row triggers navigation
    render(<Parts />);
    await waitForLoad();
    // Click on the row containing IPN-001
    const row = screen.getByText("IPN-001").closest("tr");
    fireEvent.click(row!);
    // The component calls navigate(`/parts/${encodeURIComponent(ipn)}`)
    // Since we're using BrowserRouter, check window.location
    await waitFor(() => {
      expect(window.location.pathname).toBe("/parts/IPN-001");
    });
  });

  it("navigates to correct part on different row click", async () => {
    render(<Parts />);
    await waitForLoad();
    const row = screen.getByText("IPN-002").closest("tr");
    fireEvent.click(row!);
    await waitFor(() => {
      expect(window.location.pathname).toBe("/parts/IPN-002");
    });
    // Reset
    window.history.pushState({}, "", "/");
  });

  it("clicking Next changes page and triggers API call", async () => {
    mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 100, page: 1, limit: 50 } });
    render(<Parts />);
    await waitForLoad();
    mockGetParts.mockClear();
    fireEvent.click(screen.getByText("Next"));
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalledWith(
        expect.objectContaining({ page: 2 })
      );
    });
  });

  it("clicking Previous after Next goes back to page 1", async () => {
    mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 150, page: 1, limit: 50 } });
    render(<Parts />);
    await waitForLoad();
    // Go to page 2
    fireEvent.click(screen.getByText("Next"));
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }));
    });
    mockGetParts.mockClear();
    // Go back to page 1
    fireEvent.click(screen.getByText("Previous"));
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalledWith(expect.objectContaining({ page: 1 }));
    });
  });

  it("search calls getParts with query param", async () => {
    render(<Parts />);
    await waitForLoad();
    mockGetParts.mockClear();
    const searchInput = screen.getByPlaceholderText(/search parts by ipn/i);
    fireEvent.change(searchInput, { target: { value: "resistor" } });
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalledWith(
        expect.objectContaining({ q: "resistor", page: 1 })
      );
    });
  });

  it("category filter calls getParts with category param", async () => {
    render(<Parts />);
    await waitForLoad();
    mockGetParts.mockClear();
    // Find the category select trigger (the one inside the Filters card)
    const filtersCard = screen.getByText("Filters").closest("[class*='card']")!;
    const selectTrigger = filtersCard.querySelector('[role="combobox"]')!;
    fireEvent.click(selectTrigger);
    await waitFor(() => expect(screen.getByText("Resistors (50)")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Resistors (50)"));
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalledWith(
        expect.objectContaining({ category: "resistors", page: 1 })
      );
    });
  });

  it("reset button clears search, category, and page", async () => {
    mockGetParts.mockResolvedValue({ data: mockParts, meta: { total: 100, page: 1, limit: 50 } });
    render(<Parts />);
    await waitForLoad();
    // Set search
    const searchInput = screen.getByPlaceholderText(/search parts by ipn/i);
    fireEvent.change(searchInput, { target: { value: "test" } });
    await waitFor(() => expect(mockGetParts).toHaveBeenCalledWith(expect.objectContaining({ q: "test" })));
    // Go to page 2
    fireEvent.click(screen.getByText("Next"));
    await waitFor(() => expect(mockGetParts).toHaveBeenCalledWith(expect.objectContaining({ page: 2 })));
    mockGetParts.mockClear();
    // Find reset button - it's the outline button with RotateCcw icon
    const buttons = screen.getAllByRole("button");
    const resetBtn = buttons.find(b => b.querySelector('svg.lucide-rotate-ccw') || b.querySelector('[class*="rotate"]'));
    if (resetBtn) fireEvent.click(resetBtn);
    else {
      // Fallback: find button with RotateCcw icon by its variant
      const outlineBtns = buttons.filter(b => b.textContent === '' || b.textContent?.trim() === '');
      fireEvent.click(outlineBtns[0]);
    }
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalledWith(
        expect.objectContaining({ page: 1, limit: 50 })
      );
      // Should NOT have q or category params
      const lastCall = mockGetParts.mock.calls[mockGetParts.mock.calls.length - 1][0];
      expect(lastCall.q).toBeUndefined();
      expect(lastCall.category).toBeUndefined();
    });
    // Search input should be cleared
    expect(searchInput).toHaveValue("");
  });

  it("create part error keeps dialog open", async () => {
    mockCreatePart.mockRejectedValue(new Error("Server error"));
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => expect(screen.getByLabelText("IPN *")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("IPN *"), { target: { value: "FAIL-001" } });
    fireEvent.click(screen.getByText("Create Part"));
    await waitFor(() => {
      expect(mockCreatePart).toHaveBeenCalled();
    });
    // Dialog should still be open after error
    expect(screen.getByText("Add New Part")).toBeInTheDocument();
    expect(screen.getByLabelText("IPN *")).toBeInTheDocument();
  });

  it("create part with all fields sends correct payload types", async () => {
    render(<Parts />);
    await waitForLoad();
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => expect(screen.getByLabelText("IPN *")).toBeInTheDocument());
    fireEvent.change(screen.getByLabelText("IPN *"), { target: { value: "FULL-001" } });
    fireEvent.change(screen.getByLabelText("Description"), { target: { value: "Full test part" } });
    fireEvent.change(screen.getByLabelText("Cost ($)"), { target: { value: "12.50" } });
    fireEvent.change(screen.getByLabelText("Price ($)"), { target: { value: "25.99" } });
    fireEvent.change(screen.getByLabelText("Current Stock"), { target: { value: "100" } });
    fireEvent.change(screen.getByLabelText("Location"), { target: { value: "Bin A3" } });
    fireEvent.change(screen.getByLabelText("Vendor"), { target: { value: "Acme Corp" } });
    fireEvent.click(screen.getByText("Create Part"));
    await waitFor(() => {
      expect(mockCreatePart).toHaveBeenCalledWith(
        expect.objectContaining({
          ipn: "FULL-001",
          description: "Full test part",
          cost: 12.50,
          price: 25.99,
          current_stock: 100,
          location: "Bin A3",
          vendor: "Acme Corp",
        })
      );
    });
    // Verify types
    const payload = mockCreatePart.mock.calls[0][0];
    expect(typeof payload.cost).toBe("number");
    expect(typeof payload.price).toBe("number");
    expect(typeof payload.current_stock).toBe("number");
  });

  it("displayParts extracts fields from alternative field names", async () => {
    const altParts: any[] = [
      {
        ipn: "ALT-001",
        created_at: "2024-01-01",
        updated_at: "2024-01-01",
        fields: {
          _category: "Sensors",
          desc: "Temperature sensor",
          qty_on_hand: "42",
          status: "active",
          cost: "3.25",
        },
      },
    ];
    mockGetParts.mockResolvedValue({ data: altParts, meta: { total: 1, page: 1, limit: 50 } });
    render(<Parts />);
    await waitFor(() => expect(screen.getByText("ALT-001")).toBeInTheDocument());
    // _category should display as "Sensors"
    expect(screen.getByText("Sensors")).toBeInTheDocument();
    // desc should display as description
    expect(screen.getByText("Temperature sensor")).toBeInTheDocument();
    // qty_on_hand=42 should show as stock
    expect(screen.getByText("42")).toBeInTheDocument();
    // cost should render as $3.25
    expect(screen.getByText("$3.25")).toBeInTheDocument();
  });
});
