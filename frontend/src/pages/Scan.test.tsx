import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import Scan from "./Scan";

// Mock BarcodeScanner since it uses html5-qrcode
// Mock needs to export both named export and default for lazy loading
const MockScanner = ({ onScan }: { onScan: (v: string) => void }) => (
  <div data-testid="mock-scanner">
    <button onClick={() => onScan("TEST-001")}>Mock Scan</button>
  </div>
);

vi.mock("../components/BarcodeScanner", () => ({
  BarcodeScanner: MockScanner,
  default: { BarcodeScanner: MockScanner },
}));

// Mock fetch
globalThis.fetch = vi.fn();

describe("Scan page", () => {
  it("renders the scanner page", async () => {
    render(<Scan />);
    expect(screen.getByText("Barcode Scanner")).toBeInTheDocument();
    // Wait for lazy-loaded BarcodeScanner component to render
    await waitFor(() => expect(screen.getByTestId("mock-scanner")).toBeInTheDocument());
  });

  it("calls API on scan", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ results: [] }),
    });

    render(<Scan />);
    // Wait for lazy-loaded component before interacting
    await waitFor(() => expect(screen.getByTestId("mock-scanner")).toBeInTheDocument());
    
    const { fireEvent } = await import("@testing-library/react");
    fireEvent.click(screen.getByText("Mock Scan"));

    expect(globalThis.fetch).toHaveBeenCalledWith("/api/v1/scan/TEST-001");
  });
});
