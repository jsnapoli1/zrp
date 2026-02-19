import { useCallback, useId, useState } from "react";
import { ScanLine, Camera, CameraOff, CheckCircle } from "lucide-react";
import { Button } from "./ui/button";
import { useBarcodeScanner } from "../hooks/useBarcodeScanner";

export interface BarcodeScannerProps {
  onScan: (value: string) => void;
  /** Auto-stop after first successful scan */
  autoStop?: boolean;
  className?: string;
}

export function BarcodeScanner({ onScan, autoStop = true, className }: BarcodeScannerProps) {
  const elementId = useId().replace(/:/g, "-") + "-scanner";
  const [showSuccess, setShowSuccess] = useState(false);

  const handleScan = useCallback(
    (value: string) => {
      setShowSuccess(true);
      onScan(value);
      if (autoStop) {
        setTimeout(() => scanner.stop(), 300);
      }
      setTimeout(() => setShowSuccess(false), 2000);
    },
    [onScan, autoStop]
  );

  const scanner = useBarcodeScanner(handleScan);

  const handleToggle = () => {
    if (scanner.status === "scanning" || scanner.status === "starting") {
      scanner.stop();
    } else {
      scanner.start(elementId);
    }
  };

  const isActive = scanner.status === "scanning" || scanner.status === "starting";

  return (
    <div className={className} data-testid="barcode-scanner">
      <div className="flex items-center gap-2 mb-2">
        <Button
          variant={isActive ? "destructive" : "default"}
          size="sm"
          onClick={handleToggle}
          data-testid="scanner-toggle"
        >
          {isActive ? (
            <>
              <CameraOff className="h-4 w-4 mr-1" />
              Stop Scanner
            </>
          ) : (
            <>
              <Camera className="h-4 w-4 mr-1" />
              Start Scanner
            </>
          )}
        </Button>

        {showSuccess && (
          <span className="flex items-center gap-1 text-green-600 text-sm" data-testid="scan-success">
            <CheckCircle className="h-4 w-4" />
            Scanned: {scanner.lastScanned}
          </span>
        )}
      </div>

      {scanner.error && (
        <p className="text-red-500 text-sm mb-2" data-testid="scanner-error">
          {scanner.error}
        </p>
      )}

      <div
        id={elementId}
        data-testid="scanner-viewport"
        className="relative overflow-hidden rounded-lg bg-black"
        style={{ minHeight: isActive ? 300 : 0, display: isActive ? "block" : "none" }}
      >
        {isActive && (
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none z-10">
            <ScanLine className="h-12 w-12 text-white/50 animate-pulse" />
          </div>
        )}
      </div>
    </div>
  );
}

export default BarcodeScanner;
