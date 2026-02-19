import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useBarcodeScanner } from "./useBarcodeScanner";

// Mock html5-qrcode
const mockStart = vi.fn();
const mockStop = vi.fn();
const mockClear = vi.fn();
const mockGetState = vi.fn().mockReturnValue(1); // NOT_STARTED

vi.mock("html5-qrcode", () => ({
  Html5Qrcode: vi.fn().mockImplementation(() => ({
    start: mockStart,
    stop: mockStop,
    clear: mockClear,
    getState: mockGetState,
  })),
  Html5QrcodeSupportedFormats: {
    QR_CODE: 0,
    CODE_128: 2,
    CODE_39: 3,
    EAN_13: 7,
  },
}));

describe("useBarcodeScanner", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStart.mockResolvedValue(undefined);
    mockStop.mockResolvedValue(undefined);
    mockGetState.mockReturnValue(1);
  });

  it("initializes with idle status", () => {
    const { result } = renderHook(() => useBarcodeScanner());
    expect(result.current.status).toBe("idle");
    expect(result.current.error).toBeNull();
    expect(result.current.lastScanned).toBeNull();
  });

  it("starts scanning and updates status", async () => {
    mockStart.mockResolvedValue(undefined);
    const { result } = renderHook(() => useBarcodeScanner());

    await act(async () => {
      await result.current.start("test-element");
    });

    expect(result.current.status).toBe("scanning");
    expect(mockStart).toHaveBeenCalledTimes(1);
  });

  it("calls onScan callback when code is decoded", async () => {
    let capturedCallback: ((text: string) => void) | null = null;
    mockStart.mockImplementation((_config: unknown, _prefs: unknown, onSuccess: (text: string) => void) => {
      capturedCallback = onSuccess;
      return Promise.resolve();
    });

    const onScan = vi.fn();
    const { result } = renderHook(() => useBarcodeScanner(onScan));

    await act(async () => {
      await result.current.start("test-element");
    });

    expect(capturedCallback).not.toBeNull();

    act(() => {
      capturedCallback!("TEST-IPN-001");
    });

    expect(onScan).toHaveBeenCalledWith("TEST-IPN-001");
    expect(result.current.lastScanned).toBe("TEST-IPN-001");
  });

  it("stops scanning", async () => {
    mockGetState.mockReturnValue(2); // SCANNING
    const { result } = renderHook(() => useBarcodeScanner());

    await act(async () => {
      await result.current.start("test-element");
    });

    await act(async () => {
      await result.current.stop();
    });

    expect(result.current.status).toBe("idle");
  });

  it("handles start errors gracefully", async () => {
    mockStart.mockRejectedValue(new Error("Camera not found"));
    const { result } = renderHook(() => useBarcodeScanner());

    await act(async () => {
      await result.current.start("test-element");
    });

    expect(result.current.status).toBe("error");
    expect(result.current.error).toBe("Camera not found");
  });
});
