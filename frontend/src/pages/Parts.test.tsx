import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockParts, mockCategories } from "../test/mocks";

const mockGetParts = vi.fn().mockResolvedValue({ data: mockParts, meta: { total: 3, page: 1, limit: 50 } });
const mockGetCategories = vi.fn().mockResolvedValue(mockCategories);
const mockCreatePart = vi.fn().mockResolvedValue(mockParts[0]);

vi.mock("../lib/api", () => ({
  api: {
    getParts: (...args: any[]) => mockGetParts(...args),
    getCategories: (...args: any[]) => mockGetCategories(...args),
    createPart: (...args: any[]) => mockCreatePart(...args),
    deletePart: vi.fn().mockResolvedValue(undefined),
  },
}));

import Parts from "./Parts";

beforeEach(() => vi.clearAllMocks());

describe("Parts", () => {
  it("renders page title and subtitle", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("Parts")).toBeInTheDocument();
    });
    expect(screen.getByText("Manage your parts inventory and specifications")).toBeInTheDocument();
  });

  it("renders parts table after loading", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    expect(screen.getByText("IPN-002")).toBeInTheDocument();
    expect(screen.getByText("IPN-003")).toBeInTheDocument();
  });

  it("shows loading skeletons initially", () => {
    mockGetParts.mockReturnValue(new Promise(() => {})); // never resolves
    render(<Parts />);
    // skeletons rendered (no table rows yet)
    expect(screen.queryByText("IPN-001")).not.toBeInTheDocument();
  });

  it("has search input with placeholder", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText(/search parts by ipn/i)).toBeInTheDocument();
    });
  });

  it("has add part button", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("Add Part")).toBeInTheDocument();
    });
  });

  it("opens create dialog on button click", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByText("Add New Part")).toBeInTheDocument();
      expect(screen.getByText("Create a new part in your inventory system.")).toBeInTheDocument();
    });
  });

  it("shows parts count", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText(/Parts \(3\)/)).toBeInTheDocument();
    });
  });

  it("shows empty state when no parts", async () => {
    mockGetParts.mockResolvedValueOnce({ data: [], meta: { total: 0, page: 1, limit: 50 } });
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText(/no parts available/i)).toBeInTheDocument();
    });
  });

  it("calls getParts and getCategories on mount", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalled();
      expect(mockGetCategories).toHaveBeenCalled();
    });
  });

  it("handles API error gracefully", async () => {
    mockGetParts.mockRejectedValueOnce(new Error("fail"));
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("Parts")).toBeInTheDocument();
    });
  });

  // Search/filter tests
  it("debounces search and resets to page 1", async () => {
    vi.useFakeTimers();
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    mockGetParts.mockClear();

    const searchInput = screen.getByPlaceholderText(/search parts by ipn/i);
    fireEvent.change(searchInput, { target: { value: "resistor" } });

    // Not called yet (debounce)
    expect(mockGetParts).not.toHaveBeenCalled();

    // Advance past debounce
    vi.advanceTimersByTime(350);
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalledWith(
        expect.objectContaining({ q: "resistor", page: 1 })
      );
    });
    vi.useRealTimers();
  });

  it("shows filtered empty state message", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });

    // Simulate a search that returns empty
    mockGetParts.mockResolvedValueOnce({ data: [], meta: { total: 0, page: 1, limit: 50 } });

    vi.useFakeTimers();
    const searchInput = screen.getByPlaceholderText(/search parts by ipn/i);
    fireEvent.change(searchInput, { target: { value: "nonexistent" } });
    vi.advanceTimersByTime(350);

    await waitFor(() => {
      expect(screen.getByText("No parts found matching your criteria")).toBeInTheDocument();
    });
    vi.useRealTimers();
  });

  it("renders Filters card with category select", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("Filters")).toBeInTheDocument();
    });
  });

  // Table columns
  it("shows table headers", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN")).toBeInTheDocument();
      expect(screen.getByText("Category")).toBeInTheDocument();
      expect(screen.getByText("Description")).toBeInTheDocument();
      expect(screen.getByText("Cost")).toBeInTheDocument();
      expect(screen.getByText("Stock")).toBeInTheDocument();
      expect(screen.getByText("Status")).toBeInTheDocument();
    });
  });

  // Create part dialog form fields
  it("create dialog has required form fields", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
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
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByText("Create Part")).toBeDisabled();
    });
  });

  it("create button enabled when IPN filled", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByLabelText("IPN *")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("IPN *"), { target: { value: "NEW-001" } });
    expect(screen.getByText("Create Part")).not.toBeDisabled();
  });

  it("submits create form and closes dialog", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByLabelText("IPN *")).toBeInTheDocument();
    });

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
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Add Part"));
    await waitFor(() => {
      expect(screen.getByText("Add New Part")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Add New Part")).not.toBeInTheDocument();
    });
  });

  it("shows page info", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText(/Page 1 of/)).toBeInTheDocument();
    });
  });

  // Pagination with many parts
  it("shows pagination when more than one page", async () => {
    mockGetParts.mockResolvedValueOnce({ data: mockParts, meta: { total: 100, page: 1, limit: 50 } });
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("Previous")).toBeInTheDocument();
      expect(screen.getByText("Next")).toBeInTheDocument();
    });
    // Previous should be disabled on page 1
    expect(screen.getByText("Previous").closest("button")).toBeDisabled();
    expect(screen.getByText("Next").closest("button")).not.toBeDisabled();
  });

  it("shows showing X to Y of Z text for pagination", async () => {
    mockGetParts.mockResolvedValueOnce({ data: mockParts, meta: { total: 100, page: 1, limit: 50 } });
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText(/Showing 1 to 50 of 100 parts/)).toBeInTheDocument();
    });
  });

  it("handles category error gracefully", async () => {
    mockGetCategories.mockRejectedValueOnce(new Error("fail"));
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("Parts")).toBeInTheDocument();
    });
  });
});
