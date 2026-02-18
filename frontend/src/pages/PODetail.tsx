import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { 
  ArrowLeft, 
  Package, 
  FileText,
  Clock,
  CheckCircle,
  Building
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "../components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
} from "../components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import { api, type PurchaseOrder, type Vendor } from "../lib/api";

function PODetail() {
  const { id } = useParams<{ id: string }>();
  const [po, setPO] = useState<PurchaseOrder | null>(null);
  const [vendor, setVendor] = useState<Vendor | null>(null);
  const [loading, setLoading] = useState(true);
  const [receiveDialogOpen, setReceiveDialogOpen] = useState(false);
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);

  const [receiveForm, setReceiveForm] = useState<{ [lineId: number]: string }>({});
  const [newStatus, setNewStatus] = useState("");

  useEffect(() => {
    if (id) {
      fetchPODetail();
    }
  }, [id]);

  const fetchPODetail = async () => {
    if (!id) return;
    
    try {
      setLoading(true);
      const data = await api.getPurchaseOrder(id);
      setPO(data);
      
      // Fetch vendor details
      if (data.vendor_id) {
        try {
          const vendorData = await api.getVendor(data.vendor_id);
          setVendor(vendorData);
        } catch (error) {
          console.error("Failed to fetch vendor:", error);
        }
      }
    } catch (error) {
      console.error("Failed to fetch purchase order:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleReceiveItems = async () => {
    if (!id) return;
    
    try {
      const lines = Object.entries(receiveForm)
        .filter(([_, qty]) => qty && parseFloat(qty) > 0)
        .map(([lineId, qty]) => ({
          id: parseInt(lineId),
          qty: parseFloat(qty)
        }));
      
      if (lines.length === 0) return;
      
      await api.receivePurchaseOrder(id, lines);
      setReceiveDialogOpen(false);
      setReceiveForm({});
      fetchPODetail();
    } catch (error) {
      console.error("Failed to receive items:", error);
    }
  };

  const handleStatusChange = async () => {
    if (!id || !newStatus) return;
    
    try {
      await api.updatePurchaseOrder(id, { status: newStatus });
      setStatusDialogOpen(false);
      setNewStatus("");
      fetchPODetail();
    } catch (error) {
      console.error("Failed to update status:", error);
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const formatDateShort = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  const getStatusBadge = (status: string) => {
    const variants = {
      draft: "secondary",
      submitted: "default",
      partial: "outline",
      received: "default",
      closed: "secondary"
    } as const;

    const colors = {
      draft: "text-gray-700",
      submitted: "text-blue-700",
      partial: "text-orange-700",
      received: "text-green-700",
      closed: "text-gray-700"
    } as const;

    return (
      <Badge variant={variants[status as keyof typeof variants] || "secondary"}>
        <span className={colors[status as keyof typeof colors] || "text-gray-700"}>
          {status.toUpperCase()}
        </span>
      </Badge>
    );
  };

  const getTotalAmount = () => {
    if (!po?.lines) return 0;
    return po.lines.reduce((sum, line) => sum + (line.qty_ordered * (line.unit_price || 0)), 0);
  };

  const getLineTotal = (line: { qty_ordered: number; unit_price?: number }) => {
    return line.qty_ordered * (line.unit_price || 0);
  };

  const getPendingQty = (line: { qty_ordered: number; qty_received: number }) => {
    return Math.max(0, line.qty_ordered - line.qty_received);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "draft":
        return <FileText className="h-5 w-5 text-gray-600" />;
      case "submitted":
        return <Clock className="h-5 w-5 text-blue-600" />;
      case "partial":
        return <Package className="h-5 w-5 text-orange-600" />;
      case "received":
        return <CheckCircle className="h-5 w-5 text-green-600" />;
      case "closed":
        return <CheckCircle className="h-5 w-5 text-gray-600" />;
      default:
        return <FileText className="h-5 w-5 text-gray-600" />;
    }
  };

  const canReceive = () => {
    return po && ['submitted', 'partial'].includes(po.status) && 
           po.lines && po.lines.some(line => getPendingQty(line) > 0);
  };

  const canChangeStatus = () => {
    return po && po.status !== 'closed';
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading purchase order...</p>
        </div>
      </div>
    );
  }

  if (!po) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" asChild>
            <Link to="/procurement">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Procurement
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="p-8 text-center">
            <h3 className="text-lg font-semibold mb-2">Purchase Order Not Found</h3>
            <p className="text-muted-foreground">
              The purchase order "{id}" could not be found.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" asChild>
            <Link to="/procurement">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Procurement
            </Link>
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{po.id}</h1>
            <p className="text-muted-foreground">
              Purchase Order Details
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          {canChangeStatus() && (
            <Dialog open={statusDialogOpen} onOpenChange={setStatusDialogOpen}>
              <DialogTrigger asChild>
                <Button variant="outline">Change Status</Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Change PO Status</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  <div>
                    <Label>Current Status</Label>
                    <div className="mt-1">
                      {getStatusBadge(po.status)}
                    </div>
                  </div>
                  <div>
                    <Label htmlFor="status">New Status</Label>
                    <Select value={newStatus} onValueChange={setNewStatus}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select new status" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="draft">Draft</SelectItem>
                        <SelectItem value="submitted">Submitted</SelectItem>
                        <SelectItem value="received">Received</SelectItem>
                        <SelectItem value="closed">Closed</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setStatusDialogOpen(false)}>
                    Cancel
                  </Button>
                  <Button onClick={handleStatusChange} disabled={!newStatus}>
                    Update Status
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          )}
          
          {canReceive() && (
            <Dialog open={receiveDialogOpen} onOpenChange={setReceiveDialogOpen}>
              <DialogTrigger asChild>
                <Button>
                  <Package className="h-4 w-4 mr-2" />
                  Receive Items
                </Button>
              </DialogTrigger>
              <DialogContent className="max-w-4xl">
                <DialogHeader>
                  <DialogTitle>Receive Items</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>IPN</TableHead>
                        <TableHead>Description</TableHead>
                        <TableHead className="text-right">Ordered</TableHead>
                        <TableHead className="text-right">Received</TableHead>
                        <TableHead className="text-right">Pending</TableHead>
                        <TableHead>Receive Qty</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {po.lines?.filter(line => getPendingQty(line) > 0).map((line) => (
                        <TableRow key={line.id}>
                          <TableCell className="font-medium">{line.ipn}</TableCell>
                          <TableCell>{line.manufacturer && line.mpn ? `${line.manufacturer} ${line.mpn}` : line.mpn || "—"}</TableCell>
                          <TableCell className="text-right">{line.qty_ordered}</TableCell>
                          <TableCell className="text-right">{line.qty_received}</TableCell>
                          <TableCell className="text-right font-semibold">{getPendingQty(line)}</TableCell>
                          <TableCell>
                            <Input
                              type="number"
                              min="0"
                              max={getPendingQty(line)}
                              value={receiveForm[line.id] || ""}
                              onChange={(e) => setReceiveForm(prev => ({ 
                                ...prev, 
                                [line.id]: e.target.value 
                              }))}
                              placeholder="0"
                              className="w-20"
                            />
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setReceiveDialogOpen(false)}>
                    Cancel
                  </Button>
                  <Button 
                    onClick={handleReceiveItems}
                    disabled={!Object.values(receiveForm).some(qty => qty && parseFloat(qty) > 0)}
                  >
                    Receive Items
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          )}
        </div>
      </div>

      {/* PO Header Info */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              {getStatusIcon(po.status)}
              Status & Dates
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Status</p>
              <div className="mt-1">
                {getStatusBadge(po.status)}
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Created</p>
              <p className="text-sm">{formatDate(po.created_at)}</p>
            </div>
            {po.expected_date && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Expected</p>
                <p className="text-sm">{formatDateShort(po.expected_date)}</p>
              </div>
            )}
            {po.received_at && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Received</p>
                <p className="text-sm">{formatDate(po.received_at)}</p>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Building className="h-5 w-5" />
              Vendor Information
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Vendor</p>
              <p className="font-medium">{vendor?.name || po.vendor_id}</p>
            </div>
            {vendor?.contact_email && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Email</p>
                <p className="text-sm">{vendor.contact_email}</p>
              </div>
            )}
            {vendor?.contact_phone && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Phone</p>
                <p className="text-sm">{vendor.contact_phone}</p>
              </div>
            )}
            {vendor?.lead_time_days && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Lead Time</p>
                <p className="text-sm">{vendor.lead_time_days} days</p>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Order Summary</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex justify-between">
              <span className="text-sm font-medium text-muted-foreground">Line Items:</span>
              <span className="text-sm">{po.lines?.length || 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm font-medium text-muted-foreground">Total Ordered:</span>
              <span className="text-sm">{po.lines?.reduce((sum, line) => sum + line.qty_ordered, 0) || 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm font-medium text-muted-foreground">Total Received:</span>
              <span className="text-sm">{po.lines?.reduce((sum, line) => sum + line.qty_received, 0) || 0}</span>
            </div>
            <div className="flex justify-between text-lg font-semibold pt-2 border-t">
              <span>Total Amount:</span>
              <span>${getTotalAmount().toFixed(2)}</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {po.notes && (
        <Card>
          <CardHeader>
            <CardTitle>Notes</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm whitespace-pre-wrap">{po.notes}</p>
          </CardContent>
        </Card>
      )}

      {/* Line Items */}
      <Card>
        <CardHeader>
          <CardTitle>Line Items</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>IPN</TableHead>
                <TableHead>MPN</TableHead>
                <TableHead>Manufacturer</TableHead>
                <TableHead className="text-right">Qty Ordered</TableHead>
                <TableHead className="text-right">Qty Received</TableHead>
                <TableHead className="text-right">Unit Price</TableHead>
                <TableHead className="text-right">Total</TableHead>
                <TableHead>Notes</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {po.lines?.map((line) => (
                <TableRow key={line.id}>
                  <TableCell className="font-medium">{line.ipn}</TableCell>
                  <TableCell>{line.mpn || "—"}</TableCell>
                  <TableCell>{line.manufacturer || "—"}</TableCell>
                  <TableCell className="text-right">{line.qty_ordered}</TableCell>
                  <TableCell className="text-right">
                    <span className={line.qty_received < line.qty_ordered ? "text-orange-600" : "text-green-600"}>
                      {line.qty_received}
                    </span>
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {line.unit_price ? `$${line.unit_price.toFixed(2)}` : "—"}
                  </TableCell>
                  <TableCell className="text-right font-mono font-semibold">
                    ${getLineTotal(line).toFixed(2)}
                  </TableCell>
                  <TableCell>{line.notes || "—"}</TableCell>
                </TableRow>
              ))}
              {(!po.lines || po.lines.length === 0) && (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                    No line items found
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
export default PODetail;
