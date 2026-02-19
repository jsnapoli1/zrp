import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ClipboardCheck, Clock, CheckCircle, XCircle, AlertTriangle, ScanLine } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Textarea } from "../components/ui/textarea";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "../components/ui/dialog";
import { api, type ReceivingInspection } from "../lib/api";
import { BarcodeScanner } from "../components/BarcodeScanner";

function Receiving() {
  const [inspections, setInspections] = useState<ReceivingInspection[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>("");
  const [showScanner, setShowScanner] = useState(false);
  const [scanSearch, setScanSearch] = useState("");
  const [inspectDialogOpen, setInspectDialogOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState<ReceivingInspection | null>(null);
  const [inspectForm, setInspectForm] = useState({
    qty_passed: "",
    qty_failed: "",
    qty_on_hold: "",
    inspector: "",
    notes: "",
  });

  useEffect(() => {
    fetchInspections();
  }, [filter]);

  const fetchInspections = async () => {
    try {
      setLoading(true);
      const data = await api.getReceivingInspections(filter || undefined);
      setInspections(data);
    } catch (error) {
      console.error("Failed to fetch inspections:", error);
    } finally {
      setLoading(false);
    }
  };

  const openInspectDialog = (item: ReceivingInspection) => {
    setSelectedItem(item);
    setInspectForm({
      qty_passed: String(item.qty_received),
      qty_failed: "0",
      qty_on_hold: "0",
      inspector: "",
      notes: "",
    });
    setInspectDialogOpen(true);
  };

  const handleInspect = async () => {
    if (!selectedItem) return;
    try {
      await api.inspectReceiving(selectedItem.id, {
        qty_passed: parseFloat(inspectForm.qty_passed) || 0,
        qty_failed: parseFloat(inspectForm.qty_failed) || 0,
        qty_on_hold: parseFloat(inspectForm.qty_on_hold) || 0,
        inspector: inspectForm.inspector || undefined,
        notes: inspectForm.notes || undefined,
      });
      setInspectDialogOpen(false);
      fetchInspections();
    } catch (error) {
      console.error("Failed to inspect:", error);
    }
  };

  const pendingCount = inspections.filter((i) => !i.inspected_at).length;
  const inspectedCount = inspections.filter((i) => !!i.inspected_at).length;
  const failedCount = inspections.filter((i) => i.qty_failed > 0).length;

  const formatDate = (dateStr: string) =>
    new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading inspections...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Receiving & Inspection</h1>
        <p className="text-muted-foreground">
          Inspect received goods before adding to inventory.
        </p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="cursor-pointer" onClick={() => setFilter("pending")}>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Pending Inspection</p>
                <p className="text-2xl font-bold text-orange-600">{pendingCount}</p>
              </div>
              <Clock className="h-8 w-8 text-orange-600" />
            </div>
          </CardContent>
        </Card>
        <Card className="cursor-pointer" onClick={() => setFilter("inspected")}>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Inspected</p>
                <p className="text-2xl font-bold text-green-600">{inspectedCount}</p>
              </div>
              <CheckCircle className="h-8 w-8 text-green-600" />
            </div>
          </CardContent>
        </Card>
        <Card className="cursor-pointer" onClick={() => setFilter("")}>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">With Failures</p>
                <p className="text-2xl font-bold text-red-600">{failedCount}</p>
              </div>
              <XCircle className="h-8 w-8 text-red-600" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Scanner */}
      {showScanner && (
        <Card>
          <CardContent className="pt-4">
            <BarcodeScanner
              onScan={(code) => {
                setScanSearch(code);
                setShowScanner(false);
              }}
            />
          </CardContent>
        </Card>
      )}

      {/* Filter Tabs */}
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setShowScanner(!showScanner)}
        >
          <ScanLine className="h-4 w-4 mr-1" />
          Scan
        </Button>
        {["", "pending", "inspected"].map((f) => (
          <Button
            key={f}
            variant={filter === f ? "default" : "outline"}
            size="sm"
            onClick={() => setFilter(f)}
          >
            {f === "" ? "All" : f.charAt(0).toUpperCase() + f.slice(1)}
          </Button>
        ))}
      </div>

      {/* Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <ClipboardCheck className="h-5 w-5 mr-2" />
            Receiving Inspections
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>PO</TableHead>
                <TableHead>IPN</TableHead>
                <TableHead className="text-right">Received</TableHead>
                <TableHead className="text-right">Passed</TableHead>
                <TableHead className="text-right">Failed</TableHead>
                <TableHead className="text-right">On Hold</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Inspector</TableHead>
                <TableHead>Date</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {inspections.filter((item) => !scanSearch || item.ipn?.toLowerCase().includes(scanSearch.toLowerCase()) || item.po_number?.toLowerCase().includes(scanSearch.toLowerCase())).map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-mono">RI-{item.id}</TableCell>
                  <TableCell>
                    <Link
                      to={`/purchase-orders/${item.po_id}`}
                      className="text-blue-600 hover:underline"
                    >
                      {item.po_id}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Link
                      to={`/parts/${encodeURIComponent(item.ipn)}`}
                      className="font-mono text-blue-600 hover:underline"
                    >
                      {item.ipn}
                    </Link>
                  </TableCell>
                  <TableCell className="text-right">{item.qty_received}</TableCell>
                  <TableCell className="text-right">{item.qty_passed}</TableCell>
                  <TableCell className="text-right">
                    {item.qty_failed > 0 ? (
                      <span className="text-red-600 font-medium">{item.qty_failed}</span>
                    ) : (
                      item.qty_failed
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    {item.qty_on_hold > 0 ? (
                      <span className="text-orange-600 font-medium">{item.qty_on_hold}</span>
                    ) : (
                      item.qty_on_hold
                    )}
                  </TableCell>
                  <TableCell>
                    {item.inspected_at ? (
                      item.qty_failed > 0 ? (
                        <Badge variant="destructive">Failed</Badge>
                      ) : (
                        <Badge variant="default" className="bg-green-600">Passed</Badge>
                      )
                    ) : (
                      <Badge variant="outline">
                        <AlertTriangle className="h-3 w-3 mr-1" />
                        Pending
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>{item.inspector || "—"}</TableCell>
                  <TableCell>{formatDate(item.created_at)}</TableCell>
                  <TableCell>
                    {!item.inspected_at && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => openInspectDialog(item)}
                      >
                        Inspect
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {inspections.length === 0 && (
                <TableRow>
                  <TableCell colSpan={11} className="text-center py-8 text-muted-foreground">
                    No receiving inspections found
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Inspect Dialog */}
      <Dialog open={inspectDialogOpen} onOpenChange={setInspectDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              Inspect RI-{selectedItem?.id} — {selectedItem?.ipn}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="p-3 bg-muted rounded-md text-sm">
              <span className="font-medium">Qty Received:</span>{" "}
              {selectedItem?.qty_received} — PO: {selectedItem?.po_id}
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div>
                <Label>Qty Passed</Label>
                <Input
                  type="number"
                  value={inspectForm.qty_passed}
                  onChange={(e) =>
                    setInspectForm((p) => ({ ...p, qty_passed: e.target.value }))
                  }
                />
              </div>
              <div>
                <Label>Qty Failed</Label>
                <Input
                  type="number"
                  value={inspectForm.qty_failed}
                  onChange={(e) =>
                    setInspectForm((p) => ({ ...p, qty_failed: e.target.value }))
                  }
                />
              </div>
              <div>
                <Label>Qty On Hold</Label>
                <Input
                  type="number"
                  value={inspectForm.qty_on_hold}
                  onChange={(e) =>
                    setInspectForm((p) => ({ ...p, qty_on_hold: e.target.value }))
                  }
                />
              </div>
            </div>
            <div>
              <Label>Inspector</Label>
              <Input
                value={inspectForm.inspector}
                onChange={(e) =>
                  setInspectForm((p) => ({ ...p, inspector: e.target.value }))
                }
                placeholder="Your name (optional)"
              />
            </div>
            <div>
              <Label>Notes</Label>
              <Textarea
                value={inspectForm.notes}
                onChange={(e) =>
                  setInspectForm((p) => ({ ...p, notes: e.target.value }))
                }
                placeholder="Inspection notes..."
                rows={3}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setInspectDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleInspect}>Submit Inspection</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default Receiving;
