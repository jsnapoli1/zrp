import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { ArrowLeft, Printer } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import { Button } from "../components/ui/button";
import { api, type WorkOrder } from "../lib/api";
import "../styles/print.css";

interface BOMItem {
  ipn: string;
  description: string;
  qty_required: number;
  qty_on_hand: number;
  shortage: number;
  status: string;
}

const COMPANY_NAME = "ZRP";

function WorkOrderPrint() {
  const { id } = useParams<{ id: string }>();
  const [workOrder, setWorkOrder] = useState<WorkOrder | null>(null);
  const [bomData, setBomData] = useState<{
    wo_id: string;
    assembly_ipn: string;
    qty: number;
    bom: BOMItem[];
  } | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      Promise.all([
        api.getWorkOrder(id).catch(() => null),
        api.getWorkOrderBOM(id).catch(() => null),
      ]).then(([wo, bom]) => {
        setWorkOrder(wo);
        setBomData(bom);
        setLoading(false);
      });
    }
  }, [id]);

  const formatDate = (dateStr: string) =>
    new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });

  const woUrl = `${window.location.origin}/work-orders/${id}`;

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <p className="text-muted-foreground">Loading...</p>
      </div>
    );
  }

  if (!workOrder) {
    return (
      <div className="p-8 text-center">
        <p>Work Order not found.</p>
        <Button variant="outline" asChild className="mt-4">
          <Link to="/work-orders">Back to Work Orders</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto">
      {/* Screen-only controls */}
      <div className="flex items-center gap-4 mb-6 print:hidden">
        <Button variant="outline" size="sm" asChild>
          <Link to={`/work-orders/${id}`}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Work Order
          </Link>
        </Button>
        <Button onClick={() => window.print()}>
          <Printer className="h-4 w-4 mr-2" />
          Print Traveler
        </Button>
      </div>

      {/* Printable content */}
      <div className="print-page">
        {/* Header */}
        <div className="print-header flex justify-between items-center border-b-2 border-black pb-2 mb-6">
          <div>
            <h1 className="text-2xl font-bold">{COMPANY_NAME}</h1>
            <p className="text-sm text-muted-foreground">Work Order Traveler</p>
          </div>
          <div className="text-right">
            <QRCodeSVG value={woUrl} size={80} className="qr-code" />
          </div>
        </div>

        {/* WO Info Grid */}
        <div className="grid grid-cols-2 gap-4 mb-6 avoid-break">
          <div className="space-y-2">
            <div>
              <span className="text-sm font-semibold">WO Number:</span>{" "}
              <span className="font-mono font-bold text-lg">{workOrder.id}</span>
            </div>
            <div>
              <span className="text-sm font-semibold">Assembly IPN:</span>{" "}
              <span className="font-mono">{workOrder.assembly_ipn}</span>
            </div>
            <div>
              <span className="text-sm font-semibold">Quantity:</span>{" "}
              <span className="font-bold">{workOrder.qty}</span>
            </div>
          </div>
          <div className="space-y-2">
            <div>
              <span className="text-sm font-semibold">Status:</span>{" "}
              <span className="uppercase">{workOrder.status.replace("_", " ")}</span>
            </div>
            <div>
              <span className="text-sm font-semibold">Priority:</span>{" "}
              <span className="uppercase">{workOrder.priority}</span>
            </div>
            <div>
              <span className="text-sm font-semibold">Created:</span>{" "}
              <span>{formatDate(workOrder.created_at)}</span>
            </div>
            {workOrder.started_at && (
              <div>
                <span className="text-sm font-semibold">Started:</span>{" "}
                <span>{formatDate(workOrder.started_at)}</span>
              </div>
            )}
          </div>
        </div>

        {/* Notes */}
        {workOrder.notes && (
          <div className="mb-6 avoid-break">
            <h2 className="text-lg font-bold border-b mb-2">Notes</h2>
            <p className="text-sm whitespace-pre-wrap">{workOrder.notes}</p>
          </div>
        )}

        {/* BOM Table */}
        {bomData && bomData.bom.length > 0 && (
          <div className="mb-6">
            <h2 className="text-lg font-bold border-b mb-2">Bill of Materials</h2>
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="bg-gray-100">
                  <th className="border px-2 py-1 text-left">IPN</th>
                  <th className="border px-2 py-1 text-left">Description</th>
                  <th className="border px-2 py-1 text-right">Qty Required</th>
                  <th className="border px-2 py-1 text-right">On Hand</th>
                  <th className="border px-2 py-1 text-right">Shortage</th>
                  <th className="border px-2 py-1 text-center">Status</th>
                </tr>
              </thead>
              <tbody>
                {bomData.bom.map((item, idx) => (
                  <tr key={idx} className={item.status === "shortage" ? "bg-red-50" : ""}>
                    <td className="border px-2 py-1 font-mono">{item.ipn}</td>
                    <td className="border px-2 py-1">{item.description || "—"}</td>
                    <td className="border px-2 py-1 text-right font-mono">{item.qty_required}</td>
                    <td className="border px-2 py-1 text-right font-mono">{item.qty_on_hand}</td>
                    <td className="border px-2 py-1 text-right font-mono font-bold">
                      {item.shortage > 0 ? item.shortage : "—"}
                    </td>
                    <td className="border px-2 py-1 text-center uppercase text-xs">
                      {item.status}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Sign-off area */}
        <div className="mt-12 avoid-break">
          <h2 className="text-lg font-bold border-b mb-4">Sign-Off</h2>
          <div className="grid grid-cols-3 gap-8">
            <div>
              <p className="text-sm font-semibold mb-8">Prepared By:</p>
              <div className="border-b border-black" />
              <p className="text-xs text-muted-foreground mt-1">Name / Date</p>
            </div>
            <div>
              <p className="text-sm font-semibold mb-8">QC Inspection:</p>
              <div className="border-b border-black" />
              <p className="text-xs text-muted-foreground mt-1">Name / Date</p>
            </div>
            <div>
              <p className="text-sm font-semibold mb-8">Approved By:</p>
              <div className="border-b border-black" />
              <p className="text-xs text-muted-foreground mt-1">Name / Date</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default WorkOrderPrint;
