import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "../test/test-utils";
import { BarcodeScanner } from "./BarcodeScanner";

// Mock the hook
const mockStart = vi.fn();
const mockStop = vi.fn();

vi.mock("../hooks/useBarcodeScanner", () => ({
  useBarcodeScanner: (onScan?: (v: string) => void) => {
    // Store callback so tests can trigger it
    (globalThis as any).__scanCallback = onScan;
    return {
      status: (globalThis as any).__scannerStatus || "idle",
      error: (globalThis as any).__scannerError || null,
      lastScanned: (globalThis as any).__lastScanned || null,
      start: mockStart,
      stop: mockStop,
    };
  },
}));

describe("BarcodeScanner", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (globalThis as any).__scannerStatus = "idle";
    (globalThis as any).__scannerError = null;
    (globalThis as any).__lastScanned = null;
  });

  it("renders with start button", () => {
    render(<BarcodeScanner onScan={vi.fn()} />);
    expect(screen.getByTestId("scanner-toggle")).toHaveTextContent("Start Scanner");
  });

  it("calls start when toggle clicked", () => {
    render(<BarcodeScanner onScan={vi.fn()} />);
    fireEvent.click(screen.getByTestId("scanner-toggle"));
    expect(mockStart).toHaveBeenCalled();
  });

  it("shows stop button when scanning", () => {
    (globalThis as any).__scannerStatus = "scanning";
    render(<BarcodeScanner onScan={vi.fn()} />);
    expect(screen.getByTestId("scanner-toggle")).toHaveTextContent("Stop Scanner");
  });

  it("displays error message", () => {
    (globalThis as any).__scannerError = "Camera denied";
    render(<BarcodeScanner onScan={vi.fn()} />);
    expect(screen.getByTestId("scanner-error")).toHaveTextContent("Camera denied");
  });

  it("hides viewport when idle", () => {
    render(<BarcodeScanner onScan={vi.fn()} />);
    const viewport = screen.getByTestId("scanner-viewport");
    expect(viewport).toHaveStyle({ display: "none" });
  });

  it("shows viewport when scanning", () => {
    (globalThis as any).__scannerStatus = "scanning";
    render(<BarcodeScanner onScan={vi.fn()} />);
    const viewport = screen.getByTestId("scanner-viewport");
    expect(viewport).toHaveStyle({ display: "block" });
  });
});
