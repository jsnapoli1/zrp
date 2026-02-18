import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent, act } from "../test/test-utils";
import Reports from "./Reports";

beforeEach(() => vi.clearAllMocks());

describe("Reports", () => {
  it("renders page title and description", () => {
    render(<Reports />);
    expect(screen.getByText("Reports")).toBeInTheDocument();
    expect(screen.getByText("Generate and export reports across all modules.")).toBeInTheDocument();
  });

  it("renders all report cards", () => {
    render(<Reports />);
    expect(screen.getByText("Inventory Valuation")).toBeInTheDocument();
    expect(screen.getByText("Open ECOs Report")).toBeInTheDocument();
    expect(screen.getByText("Work Order Throughput")).toBeInTheDocument();
    expect(screen.getByText("Vendor Performance")).toBeInTheDocument();
    expect(screen.getByText("Low Stock Alert")).toBeInTheDocument();
    expect(screen.getByText("User Activity")).toBeInTheDocument();
    expect(screen.getByText("Purchase Order Summary")).toBeInTheDocument();
    expect(screen.getByText("Part Usage Analysis")).toBeInTheDocument();
  });

  it("shows report descriptions", () => {
    render(<Reports />);
    expect(screen.getByText("Current inventory value by category and location")).toBeInTheDocument();
    expect(screen.getByText("On-time delivery and quality metrics")).toBeInTheDocument();
  });

  it("generates inventory valuation report when card clicked", async () => {
    vi.useFakeTimers();
    render(<Reports />);
    
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    // Should show loading
    expect(screen.getByText("Generating report...")).toBeInTheDocument();
    
    // Advance past the setTimeout
    await act(async () => {
      vi.advanceTimersByTime(1100);
    });
    
    await waitFor(() => {
      expect(screen.getByText("Electronics")).toBeInTheDocument();
    });
    expect(screen.getByText("Warehouse A")).toBeInTheDocument();
    expect(screen.getByText("$125,430")).toBeInTheDocument();
    
    vi.useRealTimers();
  });

  it("generates open ECOs report", async () => {
    vi.useFakeTimers();
    render(<Reports />);
    
    fireEvent.click(screen.getByText("Open ECOs Report"));
    
    await act(async () => {
      vi.advanceTimersByTime(1100);
    });
    
    await waitFor(() => {
      expect(screen.getByText("Widget Improvement v2.1")).toBeInTheDocument();
    });
    expect(screen.getByText("In Review")).toBeInTheDocument();
    
    vi.useRealTimers();
  });

  it("generates vendor performance report", async () => {
    vi.useFakeTimers();
    render(<Reports />);
    
    fireEvent.click(screen.getByText("Vendor Performance"));
    
    await act(async () => {
      vi.advanceTimersByTime(1100);
    });
    
    await waitFor(() => {
      expect(screen.getByText("ABC Electronics")).toBeInTheDocument();
    });
    expect(screen.getByText("94%")).toBeInTheDocument();
    
    vi.useRealTimers();
  });

  it("shows CSV and HTML export buttons after report generated", async () => {
    vi.useFakeTimers();
    render(<Reports />);
    
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await act(async () => {
      vi.advanceTimersByTime(1100);
    });
    
    await waitFor(() => {
      expect(screen.getByText("CSV")).toBeInTheDocument();
      expect(screen.getByText("HTML")).toBeInTheDocument();
    });
    
    vi.useRealTimers();
  });

  it("exports CSV when CSV button clicked", async () => {
    vi.useFakeTimers();
    const mockCreateObjectURL = vi.fn().mockReturnValue("blob:test");
    const mockRevokeObjectURL = vi.fn();
    global.URL.createObjectURL = mockCreateObjectURL;
    global.URL.revokeObjectURL = mockRevokeObjectURL;
    
    render(<Reports />);
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await act(async () => {
      vi.advanceTimersByTime(1100);
    });
    
    await waitFor(() => screen.getByText("CSV"));
    
    // Mock appendChild and removeChild
    const mockAppendChild = vi.spyOn(document.body, "appendChild").mockImplementation((node) => node as any);
    const mockRemoveChild = vi.spyOn(document.body, "removeChild").mockImplementation((node) => node as any);
    
    fireEvent.click(screen.getByText("CSV"));
    
    expect(mockCreateObjectURL).toHaveBeenCalled();
    
    mockAppendChild.mockRestore();
    mockRemoveChild.mockRestore();
    vi.useRealTimers();
  });

  it("exports HTML when HTML button clicked", async () => {
    vi.useFakeTimers();
    const mockCreateObjectURL = vi.fn().mockReturnValue("blob:test");
    const mockRevokeObjectURL = vi.fn();
    global.URL.createObjectURL = mockCreateObjectURL;
    global.URL.revokeObjectURL = mockRevokeObjectURL;
    
    render(<Reports />);
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await act(async () => {
      vi.advanceTimersByTime(1100);
    });
    
    await waitFor(() => screen.getByText("HTML"));
    
    const mockAppendChild = vi.spyOn(document.body, "appendChild").mockImplementation((node) => node as any);
    const mockRemoveChild = vi.spyOn(document.body, "removeChild").mockImplementation((node) => node as any);
    
    fireEvent.click(screen.getByText("HTML"));
    
    expect(mockCreateObjectURL).toHaveBeenCalled();
    
    mockAppendChild.mockRestore();
    mockRemoveChild.mockRestore();
    vi.useRealTimers();
  });

  it("shows summary total for inventory valuation", async () => {
    vi.useFakeTimers();
    render(<Reports />);
    
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await act(async () => {
      vi.advanceTimersByTime(1100);
    });
    
    await waitFor(() => {
      expect(screen.getByText("Total: $275,650")).toBeInTheDocument();
    });
    
    vi.useRealTimers();
  });

  it("shows generated timestamp", async () => {
    vi.useFakeTimers();
    render(<Reports />);
    
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await act(async () => {
      vi.advanceTimersByTime(1100);
    });
    
    await waitFor(() => {
      expect(screen.getByText(/Generated on/)).toBeInTheDocument();
    });
    
    vi.useRealTimers();
  });

  it("highlights selected report card", async () => {
    vi.useFakeTimers();
    render(<Reports />);
    
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    // The card should have ring-2 ring-primary class
    const card = screen.getByText("Inventory Valuation").closest('[class*="cursor-pointer"]');
    expect(card?.className).toContain("ring-2");
    
    vi.useRealTimers();
  });

  it("generates work order throughput report", async () => {
    vi.useFakeTimers();
    render(<Reports />);
    
    fireEvent.click(screen.getByText("Work Order Throughput"));
    
    await act(async () => {
      vi.advanceTimersByTime(1100);
    });
    
    await waitFor(() => {
      expect(screen.getByText("January 2024")).toBeInTheDocument();
    });
    
    vi.useRealTimers();
  });
});
