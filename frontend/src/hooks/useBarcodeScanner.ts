import { useCallback, useEffect, useRef, useState } from "react";
import { Html5Qrcode, Html5QrcodeSupportedFormats } from "html5-qrcode";

export type ScannerStatus = "idle" | "starting" | "scanning" | "error";

export interface UseBarcodeScanner {
  status: ScannerStatus;
  error: string | null;
  lastScanned: string | null;
  start: (elementId: string) => Promise<void>;
  stop: () => Promise<void>;
}

const SUPPORTED_FORMATS = [
  Html5QrcodeSupportedFormats.QR_CODE,
  Html5QrcodeSupportedFormats.CODE_128,
  Html5QrcodeSupportedFormats.CODE_39,
  Html5QrcodeSupportedFormats.EAN_13,
];

export function useBarcodeScanner(onScan?: (value: string) => void): UseBarcodeScanner {
  const [status, setStatus] = useState<ScannerStatus>("idle");
  const [error, setError] = useState<string | null>(null);
  const [lastScanned, setLastScanned] = useState<string | null>(null);
  const scannerRef = useRef<Html5Qrcode | null>(null);
  const onScanRef = useRef(onScan);
  onScanRef.current = onScan;

  const stop = useCallback(async () => {
    if (scannerRef.current) {
      try {
        const state = scannerRef.current.getState();
        // State 2 = SCANNING, 3 = PAUSED
        if (state === 2 || state === 3) {
          await scannerRef.current.stop();
        }
      } catch {
        // ignore stop errors
      }
      scannerRef.current.clear();
      scannerRef.current = null;
    }
    setStatus("idle");
  }, []);

  const start = useCallback(async (elementId: string) => {
    await stop();
    setError(null);
    setStatus("starting");

    try {
      const scanner = new Html5Qrcode(elementId, {
        formatsToSupport: SUPPORTED_FORMATS,
        verbose: false,
      });
      scannerRef.current = scanner;

      await scanner.start(
        { facingMode: "environment" },
        { fps: 10, qrbox: { width: 250, height: 250 } },
        (decodedText) => {
          setLastScanned(decodedText);
          onScanRef.current?.(decodedText);
        },
        () => {
          // scan failure (no code found) — ignore
        }
      );
      setStatus("scanning");
    } catch (err) {
      setStatus("error");
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [stop]);

  useEffect(() => {
    return () => {
      // cleanup on unmount
      if (scannerRef.current) {
        try {
          const state = scannerRef.current.getState();
          if (state === 2 || state === 3) {
            scannerRef.current.stop().catch(() => {});
          }
          scannerRef.current.clear();
        } catch {
          // ignore
        }
      }
    };
  }, []);

  return { status, error, lastScanned, start, stop };
}
