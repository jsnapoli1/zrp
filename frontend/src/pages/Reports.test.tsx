import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
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
    render(<Reports />);
    fireEvent.click(screen.getByText("Inventory Valuation"));
    expect(screen.getByText("Generating report...")).toBeInTheDocument();
    
    await waitFor(() => {
      expect(screen.getByText("Electronics")).toBeInTheDocument();
    }, { timeout: 3000 });
    expect(screen.getAllByText("Warehouse A").length).toBeGreaterThan(0);
    expect(screen.getByText("$125,430")).toBeInTheDocument();
  });

  it("generates open ECOs report", async () => {
    render(<Reports />);
    fireEvent.click(screen.getByText("Open ECOs Report"));
    
    await waitFor(() => {
      expect(screen.getByText("Widget Improvement v2.1")).toBeInTheDocument();
    }, { timeout: 3000 });
    expect(screen.getByText("In Review")).toBeInTheDocument();
  });

  it("generates vendor performance report", async () => {
    render(<Reports />);
    fireEvent.click(screen.getByText("Vendor Performance"));
    
    await waitFor(() => {
      expect(screen.getByText("ABC Electronics")).toBeInTheDocument();
    }, { timeout: 3000 });
    expect(screen.getByText("94%")).toBeInTheDocument();
  });

  it("shows CSV and HTML export buttons after report generated", async () => {
    render(<Reports />);
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await waitFor(() => {
      expect(screen.getByText("CSV")).toBeInTheDocument();
      expect(screen.getByText("HTML")).toBeInTheDocument();
    }, { timeout: 3000 });
  });

  it("exports CSV when CSV button clicked", async () => {
    const mockCreateObjectURL = vi.fn().mockReturnValue("blob:test");
    const mockRevokeObjectURL = vi.fn();
    global.URL.createObjectURL = mockCreateObjectURL;
    global.URL.revokeObjectURL = mockRevokeObjectURL;
    
    render(<Reports />);
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await waitFor(() => screen.getByText("CSV"), { timeout: 3000 });
    
    const mockAppendChild = vi.spyOn(document.body, "appendChild").mockImplementation((node) => node as any);
    const mockRemoveChild = vi.spyOn(document.body, "removeChild").mockImplementation((node) => node as any);
    
    fireEvent.click(screen.getByText("CSV"));
    expect(mockCreateObjectURL).toHaveBeenCalled();
    
    mockAppendChild.mockRestore();
    mockRemoveChild.mockRestore();
  });

  it("exports HTML when HTML button clicked", async () => {
    const mockCreateObjectURL = vi.fn().mockReturnValue("blob:test");
    const mockRevokeObjectURL = vi.fn();
    global.URL.createObjectURL = mockCreateObjectURL;
    global.URL.revokeObjectURL = mockRevokeObjectURL;
    
    render(<Reports />);
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await waitFor(() => screen.getByText("HTML"), { timeout: 3000 });
    
    const mockAppendChild = vi.spyOn(document.body, "appendChild").mockImplementation((node) => node as any);
    const mockRemoveChild = vi.spyOn(document.body, "removeChild").mockImplementation((node) => node as any);
    
    fireEvent.click(screen.getByText("HTML"));
    expect(mockCreateObjectURL).toHaveBeenCalled();
    
    mockAppendChild.mockRestore();
    mockRemoveChild.mockRestore();
  });

  it("shows summary total for inventory valuation", async () => {
    render(<Reports />);
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await waitFor(() => {
      expect(screen.getByText("Total: $275,650")).toBeInTheDocument();
    }, { timeout: 3000 });
  });

  it("shows generated timestamp", async () => {
    render(<Reports />);
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    await waitFor(() => {
      expect(screen.getByText(/Generated on/)).toBeInTheDocument();
    }, { timeout: 3000 });
  });

  it("highlights selected report card", async () => {
    render(<Reports />);
    fireEvent.click(screen.getByText("Inventory Valuation"));
    
    // After clicking, the report results section should appear
    await waitFor(() => {
      expect(screen.getByText("Generating report...")).toBeInTheDocument();
    });
  });

  it("generates work order throughput report", async () => {
    render(<Reports />);
    fireEvent.click(screen.getByText("Work Order Throughput"));
    
    await waitFor(() => {
      expect(screen.getByText("January 2024")).toBeInTheDocument();
    }, { timeout: 3000 });
  });
});
