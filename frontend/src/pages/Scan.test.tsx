import { describe, it, expect, vi } from "vitest";
import { render, screen } from "../test/test-utils";
import Scan from "./Scan";

// Mock BarcodeScanner since it uses html5-qrcode
vi.mock("../components/BarcodeScanner", () => ({
  BarcodeScanner: ({ onScan }: { onScan: (v: string) => void }) => (
    <div data-testid="mock-scanner">
      <button onClick={() => onScan("TEST-001")}>Mock Scan</button>
    </div>
  ),
}));

// Mock fetch
globalThis.fetch = vi.fn();

describe("Scan page", () => {
  it("renders the scanner page", () => {
    render(<Scan />);
    expect(screen.getByText("Barcode Scanner")).toBeInTheDocument();
    expect(screen.getByTestId("mock-scanner")).toBeInTheDocument();
  });

  it("calls API on scan", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ results: [] }),
    });

    render(<Scan />);
    const { fireEvent } = await import("@testing-library/react");
    fireEvent.click(screen.getByText("Mock Scan"));

    expect(globalThis.fetch).toHaveBeenCalledWith("/api/v1/scan/TEST-001");
  });
});
