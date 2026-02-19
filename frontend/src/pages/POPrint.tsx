import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { ArrowLeft, Printer } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import { Button } from "../components/ui/button";
import { api, type PurchaseOrder, type Vendor } from "../lib/api";
import "../styles/print.css";

const COMPANY_NAME = "ZRP";

function POPrint() {
  const { id } = useParams<{ id: string }>();
  const [po, setPO] = useState<PurchaseOrder | null>(null);
  const [vendor, setVendor] = useState<Vendor | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      api
        .getPurchaseOrder(id)
        .then(async (data) => {
          setPO(data);
          if (data.vendor_id) {
            try {
              const v = await api.getVendor(data.vendor_id);
              setVendor(v);
            } catch {}
          }
        })
        .catch(() => {})
        .finally(() => setLoading(false));
    }
  }, [id]);

  const formatDate = (dateStr: string) =>
    new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });

  const poUrl = `${window.location.origin}/purchase-orders/${id}`;

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <p className="text-muted-foreground">Loading...</p>
      </div>
    );
  }

  if (!po) {
    return (
      <div className="p-8 text-center">
        <p>Purchase Order not found.</p>
        <Button variant="outline" asChild className="mt-4">
          <Link to="/procurement">Back to Procurement</Link>
        </Button>
      </div>
    );
  }

  const totalAmount = po.lines?.reduce(
    (sum, line) => sum + line.qty_ordered * (line.unit_price || 0),
    0
  ) || 0;

  return (
    <div className="max-w-4xl mx-auto">
      {/* Screen-only controls */}
      <div className="flex items-center gap-4 mb-6 print:hidden">
        <Button variant="outline" size="sm" asChild>
          <Link to={`/purchase-orders/${id}`}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to PO
          </Link>
        </Button>
        <Button onClick={() => window.print()}>
          <Printer className="h-4 w-4 mr-2" />
          Print PO
        </Button>
      </div>

      {/* Printable content */}
      <div className="print-page">
        {/* Header */}
        <div className="print-header flex justify-between items-center border-b-2 border-black pb-2 mb-6">
          <div>
            <h1 className="text-2xl font-bold">{COMPANY_NAME}</h1>
            <p className="text-sm text-muted-foreground">Purchase Order</p>
          </div>
          <div className="text-right">
            <QRCodeSVG value={poUrl} size={80} className="qr-code" />
          </div>
        </div>

        {/* PO Info */}
        <div className="grid grid-cols-2 gap-4 mb-6 avoid-break">
          <div className="space-y-2">
            <div>
              <span className="text-sm font-semibold">PO Number:</span>{" "}
              <span className="font-mono font-bold text-lg">{po.id}</span>
            </div>
            <div>
              <span className="text-sm font-semibold">Status:</span>{" "}
              <span className="uppercase">{po.status}</span>
            </div>
            <div>
              <span className="text-sm font-semibold">Created:</span>{" "}
              <span>{formatDate(po.created_at)}</span>
            </div>
            {po.expected_date && (
              <div>
                <span className="text-sm font-semibold">Expected:</span>{" "}
                <span>{formatDate(po.expected_date)}</span>
              </div>
            )}
          </div>
          <div className="space-y-2">
            <h3 className="text-sm font-bold">Vendor</h3>
            <div>
              <span className="font-semibold">{vendor?.name || po.vendor_id}</span>
            </div>
            {vendor?.contact_email && (
              <div className="text-sm">{vendor.contact_email}</div>
            )}
            {vendor?.contact_phone && (
              <div className="text-sm">{vendor.contact_phone}</div>
            )}
          </div>
        </div>

        {/* Notes */}
        {po.notes && (
          <div className="mb-6 avoid-break">
            <h2 className="text-lg font-bold border-b mb-2">Notes</h2>
            <p className="text-sm whitespace-pre-wrap">{po.notes}</p>
          </div>
        )}

        {/* Line Items */}
        {po.lines && po.lines.length > 0 && (
          <div className="mb-6">
            <h2 className="text-lg font-bold border-b mb-2">Line Items</h2>
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="bg-gray-100">
                  <th className="border px-2 py-1 text-left">IPN</th>
                  <th className="border px-2 py-1 text-left">MPN</th>
                  <th className="border px-2 py-1 text-left">Manufacturer</th>
                  <th className="border px-2 py-1 text-right">Qty Ordered</th>
                  <th className="border px-2 py-1 text-right">Qty Received</th>
                  <th className="border px-2 py-1 text-right">Unit Price</th>
                  <th className="border px-2 py-1 text-right">Total</th>
                </tr>
              </thead>
              <tbody>
                {po.lines.map((line) => (
                  <tr key={line.id}>
                    <td className="border px-2 py-1 font-mono">{line.ipn}</td>
                    <td className="border px-2 py-1">{line.mpn || "—"}</td>
                    <td className="border px-2 py-1">{line.manufacturer || "—"}</td>
                    <td className="border px-2 py-1 text-right font-mono">{line.qty_ordered}</td>
                    <td className="border px-2 py-1 text-right font-mono">{line.qty_received}</td>
                    <td className="border px-2 py-1 text-right font-mono">
                      {line.unit_price ? `$${line.unit_price.toFixed(2)}` : "—"}
                    </td>
                    <td className="border px-2 py-1 text-right font-mono font-bold">
                      ${(line.qty_ordered * (line.unit_price || 0)).toFixed(2)}
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr className="font-bold">
                  <td colSpan={6} className="border px-2 py-1 text-right">
                    Total:
                  </td>
                  <td className="border px-2 py-1 text-right font-mono">
                    ${totalAmount.toFixed(2)}
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>
        )}

        {/* Signature */}
        <div className="mt-12 avoid-break">
          <div className="grid grid-cols-2 gap-12">
            <div>
              <p className="text-sm font-semibold mb-8">Authorized By:</p>
              <div className="border-b border-black" />
              <p className="text-xs text-muted-foreground mt-1">Name / Date</p>
            </div>
            <div>
              <p className="text-sm font-semibold mb-8">Received By:</p>
              <div className="border-b border-black" />
              <p className="text-xs text-muted-foreground mt-1">Name / Date</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default POPrint;
