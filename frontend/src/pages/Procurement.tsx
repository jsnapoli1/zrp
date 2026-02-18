import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { 
  ShoppingCart, 
  Plus, 
  FileText,
  Clock,
  Package
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
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
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Textarea } from "../components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import { api, type PurchaseOrder, type Vendor } from "../lib/api";

function Procurement() {
  const [purchaseOrders, setPurchaseOrders] = useState<PurchaseOrder[]>([]);
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [parts, setParts] = useState<{ ipn: string; description?: string }[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  interface POLineForm {
    ipn: string;
    mpn: string;
    manufacturer: string;
    qty_ordered: string;
    unit_price: string;
    notes: string;
  }

  const [poForm, setPoForm] = useState({
    vendor_id: "",
    notes: "",
    expected_date: "",
    lines: [{ ipn: "", mpn: "", manufacturer: "", qty_ordered: "", unit_price: "", notes: "" }] as POLineForm[]
  });

  useEffect(() => {
    fetchPurchaseOrders();
    fetchVendors();
    fetchParts();
  }, []);

  const fetchPurchaseOrders = async () => {
    try {
      setLoading(true);
      const data = await api.getPurchaseOrders();
      setPurchaseOrders(data);
    } catch (error) {
      console.error("Failed to fetch purchase orders:", error);
    } finally {
      setLoading(false);
    }
  };

  const fetchVendors = async () => {
    try {
      const data = await api.getVendors();
      setVendors(data.filter(v => v.status === 'active'));
    } catch (error) {
      console.error("Failed to fetch vendors:", error);
    }
  };

  const fetchParts = async () => {
    try {
      const data = await api.getParts();
      const partsArray = Array.isArray(data) ? data : [];
      setParts(partsArray.map(p => ({ ipn: p.ipn, description: p.description })));
    } catch (error) {
      console.error("Failed to fetch parts:", error);
    }
  };

  const handleCreatePO = async () => {
    try {
      const lines = poForm.lines.filter(line => line.ipn && line.qty_ordered).map(line => ({
        ipn: line.ipn,
        mpn: line.mpn,
        manufacturer: line.manufacturer,
        qty_ordered: parseFloat(line.qty_ordered) || 0,
        unit_price: parseFloat(line.unit_price) || 0,
        qty_received: 0,
        notes: line.notes,
        id: 0, // Will be set by backend
        po_id: "" // Will be set by backend
      }));

      await api.createPurchaseOrder({
        vendor_id: poForm.vendor_id,
        status: "draft",
        notes: poForm.notes || undefined,
        expected_date: poForm.expected_date || undefined,
        lines
      });
      
      setCreateDialogOpen(false);
      resetForm();
      fetchPurchaseOrders();
    } catch (error) {
      console.error("Failed to create purchase order:", error);
    }
  };

  const resetForm = () => {
    setPoForm({
      vendor_id: "",
      notes: "",
      expected_date: "",
      lines: [{ ipn: "", mpn: "", manufacturer: "", qty_ordered: "", unit_price: "", notes: "" }]
    });
  };

  const addLineItem = () => {
    setPoForm(prev => ({
      ...prev,
      lines: [...prev.lines, { ipn: "", mpn: "", manufacturer: "", qty_ordered: "", unit_price: "", notes: "" }]
    }));
  };

  const removeLineItem = (index: number) => {
    if (poForm.lines.length > 1) {
      setPoForm(prev => ({
        ...prev,
        lines: prev.lines.filter((_, i) => i !== index)
      }));
    }
  };

  const updateLineItem = (index: number, field: keyof POLineForm, value: string) => {
    setPoForm(prev => ({
      ...prev,
      lines: prev.lines.map((line, i) => 
        i === index ? { ...line, [field]: value } : line
      )
    }));
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

  const getTotalAmount = (po: PurchaseOrder) => {
    if (!po.lines) return 0;
    return po.lines.reduce((sum, line) => sum + (line.qty_ordered * (line.unit_price || 0)), 0);
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  const getVendorName = (vendorId: string) => {
    const vendor = vendors.find(v => v.id === vendorId);
    return vendor?.name || vendorId;
  };

  const filteredParts = (searchTerm: string) => {
    if (!searchTerm) return [];
    return parts.filter(part => 
      part.ipn.toLowerCase().includes(searchTerm.toLowerCase())
    ).slice(0, 5);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading purchase orders...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Procurement</h1>
          <p className="text-muted-foreground">
            Manage purchase orders and vendor relationships.
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create PO
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Create Purchase Order</DialogTitle>
            </DialogHeader>
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="vendor">Vendor</Label>
                  <Select value={poForm.vendor_id} onValueChange={(value) => setPoForm(prev => ({ ...prev, vendor_id: value }))}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select vendor" />
                    </SelectTrigger>
                    <SelectContent>
                      {vendors.map((vendor) => (
                        <SelectItem key={vendor.id} value={vendor.id}>
                          {vendor.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor="expected_date">Expected Date</Label>
                  <Input
                    type="date"
                    value={poForm.expected_date}
                    onChange={(e) => setPoForm(prev => ({ ...prev, expected_date: e.target.value }))}
                  />
                </div>
              </div>

              <div>
                <Label htmlFor="notes">Notes</Label>
                <Textarea
                  id="notes"
                  value={poForm.notes}
                  onChange={(e) => setPoForm(prev => ({ ...prev, notes: e.target.value }))}
                  placeholder="Optional notes for this PO"
                  rows={2}
                />
              </div>

              <div>
                <div className="flex justify-between items-center mb-4">
                  <Label>Line Items</Label>
                  <Button type="button" variant="outline" size="sm" onClick={addLineItem}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Item
                  </Button>
                </div>
                
                <div className="space-y-4">
                  {poForm.lines.map((line, index) => (
                    <Card key={index}>
                      <CardContent className="pt-4">
                        <div className="grid grid-cols-12 gap-2 items-end">
                          <div className="col-span-3">
                            <Label>IPN</Label>
                            <Input
                              value={line.ipn || ""}
                              onChange={(e) => updateLineItem(index, 'ipn', e.target.value)}
                              placeholder="Internal part number"
                            />
                            {line.ipn && filteredParts(line.ipn).length > 0 && (
                              <div className="mt-1 border rounded-md max-h-32 overflow-y-auto">
                                {filteredParts(line.ipn).map((part) => (
                                  <div
                                    key={part.ipn}
                                    className="p-2 hover:bg-muted cursor-pointer text-sm"
                                    onClick={() => updateLineItem(index, 'ipn', part.ipn)}
                                  >
                                    <div className="font-medium">{part.ipn}</div>
                                    {part.description && (
                                      <div className="text-xs text-muted-foreground">{part.description}</div>
                                    )}
                                  </div>
                                ))}
                              </div>
                            )}
                          </div>
                          <div className="col-span-2">
                            <Label>MPN</Label>
                            <Input
                              value={line.mpn || ""}
                              onChange={(e) => updateLineItem(index, 'mpn', e.target.value)}
                              placeholder="Manufacturer PN"
                            />
                          </div>
                          <div className="col-span-2">
                            <Label>Manufacturer</Label>
                            <Input
                              value={line.manufacturer || ""}
                              onChange={(e) => updateLineItem(index, 'manufacturer', e.target.value)}
                              placeholder="Manufacturer"
                            />
                          </div>
                          <div className="col-span-1">
                            <Label>Qty</Label>
                            <Input
                              type="number"
                              value={line.qty_ordered}
                              onChange={(e) => updateLineItem(index, 'qty_ordered', e.target.value)}
                              placeholder="0"
                            />
                          </div>
                          <div className="col-span-2">
                            <Label>Unit Price</Label>
                            <Input
                              type="number"
                              step="0.01"
                              value={line.unit_price}
                              onChange={(e) => updateLineItem(index, 'unit_price', e.target.value)}
                              placeholder="0.00"
                            />
                          </div>
                          <div className="col-span-1">
                            <Label>Notes</Label>
                            <Input
                              value={line.notes}
                              onChange={(e) => updateLineItem(index, 'notes', e.target.value)}
                              placeholder="Notes"
                            />
                          </div>
                          <div className="col-span-1">
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              onClick={() => removeLineItem(index)}
                              disabled={poForm.lines.length === 1}
                            >
                              Remove
                            </Button>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
                Cancel
              </Button>
              <Button 
                onClick={handleCreatePO}
                disabled={!poForm.vendor_id || !poForm.lines.some(line => line.ipn && line.qty_ordered)}
              >
                Create PO
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Total POs</p>
                <p className="text-2xl font-bold">{purchaseOrders.length}</p>
              </div>
              <ShoppingCart className="h-8 w-8 text-blue-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Draft</p>
                <p className="text-2xl font-bold">
                  {purchaseOrders.filter(po => po.status === 'draft').length}
                </p>
              </div>
              <FileText className="h-8 w-8 text-gray-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Pending</p>
                <p className="text-2xl font-bold text-orange-600">
                  {purchaseOrders.filter(po => ['submitted', 'partial'].includes(po.status)).length}
                </p>
              </div>
              <Clock className="h-8 w-8 text-orange-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Received</p>
                <p className="text-2xl font-bold text-green-600">
                  {purchaseOrders.filter(po => po.status === 'received').length}
                </p>
              </div>
              <Package className="h-8 w-8 text-green-600" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Purchase Orders Table */}
      <Card>
        <CardHeader>
          <CardTitle>Purchase Orders</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>PO Number</TableHead>
                <TableHead>Vendor</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Total</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Expected</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {purchaseOrders.map((po) => (
                <TableRow key={po.id}>
                  <TableCell>
                    <Link 
                      to={`/purchase-orders/${po.id}`}
                      className="font-medium text-blue-600 hover:underline"
                    >
                      {po.id}
                    </Link>
                  </TableCell>
                  <TableCell>{getVendorName(po.vendor_id)}</TableCell>
                  <TableCell>{getStatusBadge(po.status)}</TableCell>
                  <TableCell className="text-right font-mono">
                    ${getTotalAmount(po).toFixed(2)}
                  </TableCell>
                  <TableCell>{formatDate(po.created_at)}</TableCell>
                  <TableCell>{po.expected_date ? formatDate(po.expected_date) : "—"}</TableCell>
                  <TableCell>
                    <Button variant="outline" size="sm" asChild>
                      <Link to={`/purchase-orders/${po.id}`}>
                        View Details
                      </Link>
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
              {purchaseOrders.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                    No purchase orders found
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
export default Procurement;
