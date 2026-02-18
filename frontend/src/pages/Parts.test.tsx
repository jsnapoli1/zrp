import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockParts, mockCategories } from "../test/mocks";

const mockGetParts = vi.fn().mockResolvedValue({ data: mockParts, meta: { total: 3, page: 1, limit: 50 } });
const mockGetCategories = vi.fn().mockResolvedValue(mockCategories);
const mockCreatePart = vi.fn().mockResolvedValue(mockParts[0]);
const mockDeletePart = vi.fn().mockResolvedValue(undefined);

vi.mock("../lib/api", () => ({
  api: {
    getParts: (...args: any[]) => mockGetParts(...args),
    getCategories: (...args: any[]) => mockGetCategories(...args),
    createPart: (...args: any[]) => mockCreatePart(...args),
    deletePart: (...args: any[]) => mockDeletePart(...args),
  },
}));

import Parts from "./Parts";

beforeEach(() => vi.clearAllMocks());

describe("Parts", () => {
  it("renders loading skeleton initially", () => {
    render(<Parts />);
    expect(screen.getByText("Parts Library")).toBeInTheDocument();
  });

  it("renders parts table after loading", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    expect(screen.getByText("IPN-002")).toBeInTheDocument();
    expect(screen.getByText("IPN-003")).toBeInTheDocument();
  });

  it("displays category filters", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("Resistors")).toBeInTheDocument();
    });
  });

  it("has search input", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
    });
  });

  it("has create part button", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText(/add part|create part|new part/i)).toBeInTheDocument();
    });
  });

  it("opens create dialog on button click", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    const addBtn = screen.getByText(/add part|create part|new part/i);
    fireEvent.click(addBtn);
    await waitFor(() => {
      expect(screen.getByText(/create new part|add new part/i)).toBeInTheDocument();
    });
  });

  it("shows empty state when no parts", async () => {
    mockGetParts.mockResolvedValueOnce({ data: [], meta: { total: 0, page: 1, limit: 50 } });
    render(<Parts />);
    await waitFor(() => {
      expect(screen.getByText(/no parts found/i)).toBeInTheDocument();
    });
  });

  it("handles API error", async () => {
    mockGetParts.mockRejectedValueOnce(new Error("fail"));
    render(<Parts />);
    // Should not crash
    await waitFor(() => {
      expect(screen.getByText("Parts Library")).toBeInTheDocument();
    });
  });

  it("calls getParts and getCategories on mount", async () => {
    render(<Parts />);
    await waitFor(() => {
      expect(mockGetParts).toHaveBeenCalled();
      expect(mockGetCategories).toHaveBeenCalled();
    });
  });
});
