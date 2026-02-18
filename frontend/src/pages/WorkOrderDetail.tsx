import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { 
  ArrowLeft, 
  Package,
  AlertTriangle,
  CheckCircle,
  Clock,
  Play,
  Settings2,
  ShoppingCart
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
// Tabs removed - not used in this component
import { api, type WorkOrder, type Vendor } from "../lib/api";

interface BOMItem {
  ipn: string;
  description: string;
  qty_required: number;
  qty_on_hand: number;
  shortage: number;
  status: string;
}

function WorkOrderDetail() {
  const { id } = useParams<{ id: string }>();
  const [workOrder, setWorkOrder] = useState<WorkOrder | null>(null);
  const [bomData, setBomData] = useState<{
    wo_id: string;
    assembly_ipn: string;
    qty: number;
    bom: BOMItem[];
  } | null>(null);
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);
  const [generatePODialogOpen, setGeneratePODialogOpen] = useState(false);

  const [newStatus, setNewStatus] = useState("");
  const [selectedVendor, setSelectedVendor] = useState("");

  useEffect(() => {
    if (id) {
      fetchWorkOrderDetail();
      fetchBOMData();
      fetchVendors();
    }
  }, [id]);

  const fetchWorkOrderDetail = async () => {
    if (!id) return;
    
    try {
      setLoading(true);
      const data = await api.getWorkOrder(id);
      setWorkOrder(data);
    } catch (error) {
      console.error("Failed to fetch work order:", error);
    } finally {
      setLoading(false);
    }
  };

  const fetchBOMData = async () => {
    if (!id) return;
    
    try {
      const data = await api.getWorkOrderBOM(id);
      setBomData(data);
    } catch (error) {
      console.error("Failed to fetch BOM data:", error);
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

  const handleStatusChange = async () => {
    if (!id || !newStatus) return;
    
    try {
      await api.updateWorkOrder(id, { status: newStatus });
      setStatusDialogOpen(false);
      setNewStatus("");
      fetchWorkOrderDetail();
    } catch (error) {
      console.error("Failed to update status:", error);
    }
  };

  const handleGeneratePO = async () => {
    if (!id || !selectedVendor) return;
    
    try {
      const result = await api.generatePOFromWorkOrder(id, selectedVendor);
      setGeneratePODialogOpen(false);
      setSelectedVendor("");
      // Navigate to the created PO or show success message
      console.log(`Generated PO ${result.po_id} with ${result.lines} line items`);
    } catch (error) {
      console.error("Failed to generate PO:", error);
    }
  };

  const getStatusBadge = (status: string) => {
    const variants = {
      open: "secondary",
      in_progress: "default",
      completed: "default",
      on_hold: "outline",
      cancelled: "secondary"
    } as const;

    const colors = {
      open: "text-gray-700",
      in_progress: "text-blue-700",
      completed: "text-green-700",
      on_hold: "text-orange-700",
      cancelled: "text-red-700"
    } as const;

    return (
      <Badge variant={variants[status as keyof typeof variants] || "secondary"}>
        <span className={colors[status as keyof typeof colors] || "text-gray-700"}>
          {status.replace('_', ' ').toUpperCase()}
        </span>
      </Badge>
    );
  };

  const getPriorityBadge = (priority: string) => {
    const colors = {
      critical: "bg-red-100 text-red-800 border-red-200",
      high: "bg-orange-100 text-orange-800 border-orange-200",
      medium: "bg-yellow-100 text-yellow-800 border-yellow-200",
      low: "bg-green-100 text-green-800 border-green-200"
    } as const;

    return (
      <Badge 
        variant="outline" 
        className={colors[priority as keyof typeof colors] || colors.medium}
      >
        {priority.toUpperCase()}
      </Badge>
    );
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "open":
        return <Clock className="h-5 w-5 text-gray-600" />;
      case "in_progress":
        return <Play className="h-5 w-5 text-blue-600" />;
      case "completed":
        return <CheckCircle className="h-5 w-5 text-green-600" />;
      case "on_hold":
        return <AlertTriangle className="h-5 w-5 text-orange-600" />;
      default:
        return <Clock className="h-5 w-5 text-gray-600" />;
    }
  };

  const getBOMStatusIcon = (status: string) => {
    switch (status) {
      case "ok":
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      case "low":
        return <AlertTriangle className="h-4 w-4 text-orange-600" />;
      case "shortage":
        return <AlertTriangle className="h-4 w-4 text-red-600" />;
      default:
        return <Package className="h-4 w-4 text-gray-600" />;
    }
  };

  const getBOMRowClass = (status: string) => {
    switch (status) {
      case "shortage":
        return "bg-red-50 border-red-200";
      case "low":
        return "bg-orange-50 border-orange-200";
      default:
        return "";
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

  // Removed unused formatDateShort function

  const canChangeStatus = () => {
    return workOrder && workOrder.status !== 'completed' && workOrder.status !== 'cancelled';
  };

  const hasShortages = () => {
    return bomData && bomData.bom.some(item => item.status === 'shortage');
  };

  const getShortageCount = () => {
    return bomData ? bomData.bom.filter(item => item.status === 'shortage').length : 0;
  };

  const getTotalItems = () => {
    return bomData ? bomData.bom.length : 0;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading work order...</p>
        </div>
      </div>
    );
  }

  if (!workOrder) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" asChild>
            <Link to="/work-orders">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Work Orders
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="p-8 text-center">
            <h3 className="text-lg font-semibold mb-2">Work Order Not Found</h3>
            <p className="text-muted-foreground">
              The work order "{id}" could not be found.
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
            <Link to="/work-orders">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Work Orders
            </Link>
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{workOrder.id}</h1>
            <p className="text-muted-foreground">
              Work Order Details
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          {canChangeStatus() && (
            <Dialog open={statusDialogOpen} onOpenChange={setStatusDialogOpen}>
              <DialogTrigger asChild>
                <Button variant="outline">
                  <Settings2 className="h-4 w-4 mr-2" />
                  Change Status
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Change Work Order Status</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-2">Current Status</p>
                    <div className="flex items-center gap-2">
                      {getStatusIcon(workOrder.status)}
                      {getStatusBadge(workOrder.status)}
                    </div>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-2">New Status</p>
                    <Select value={newStatus} onValueChange={setNewStatus}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select new status" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="open">Open</SelectItem>
                        <SelectItem value="in_progress">In Progress</SelectItem>
                        <SelectItem value="on_hold">On Hold</SelectItem>
                        <SelectItem value="completed">Completed</SelectItem>
                        <SelectItem value="cancelled">Cancelled</SelectItem>
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
          
          {hasShortages() && (
            <Dialog open={generatePODialogOpen} onOpenChange={setGeneratePODialogOpen}>
              <DialogTrigger asChild>
                <Button>
                  <ShoppingCart className="h-4 w-4 mr-2" />
                  Generate PO
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Generate Purchase Order</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  <div>
                    <p className="text-sm text-muted-foreground mb-4">
                      This will create a purchase order for all shortage items in this work order.
                    </p>
                    <div className="bg-orange-50 border border-orange-200 rounded-md p-3">
                      <div className="flex items-center gap-2">
                        <AlertTriangle className="h-4 w-4 text-orange-600" />
                        <span className="text-sm font-medium text-orange-800">
                          {getShortageCount()} items have shortages
                        </span>
                      </div>
                    </div>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-2">Select Vendor</p>
                    <Select value={selectedVendor} onValueChange={setSelectedVendor}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select vendor for PO" />
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
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setGeneratePODialogOpen(false)}>
                    Cancel
                  </Button>
                  <Button onClick={handleGeneratePO} disabled={!selectedVendor}>
                    Generate PO
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          )}
        </div>
      </div>

      {/* Work Order Header Info */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              {getStatusIcon(workOrder.status)}
              Status & Priority
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Status</p>
              <div className="mt-1">
                {getStatusBadge(workOrder.status)}
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Priority</p>
              <div className="mt-1">
                {getPriorityBadge(workOrder.priority)}
              </div>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Created</p>
              <p className="text-sm">{formatDate(workOrder.created_at)}</p>
            </div>
            {workOrder.started_at && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Started</p>
                <p className="text-sm">{formatDate(workOrder.started_at)}</p>
              </div>
            )}
            {workOrder.completed_at && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">Completed</p>
                <p className="text-sm">{formatDate(workOrder.completed_at)}</p>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Package className="h-5 w-5" />
              Assembly Information
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Assembly IPN</p>
              <p className="font-medium">{workOrder.assembly_ipn}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Quantity</p>
              <p className="text-2xl font-bold">{workOrder.qty}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>BOM Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex justify-between">
              <span className="text-sm font-medium text-muted-foreground">Total Items:</span>
              <span className="text-sm">{getTotalItems()}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm font-medium text-muted-foreground">Available:</span>
              <span className="text-sm text-green-600">
                {bomData ? bomData.bom.filter(item => item.status === 'ok').length : 0}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-sm font-medium text-muted-foreground">Low Stock:</span>
              <span className="text-sm text-orange-600">
                {bomData ? bomData.bom.filter(item => item.status === 'low').length : 0}
              </span>
            </div>
            <div className="flex justify-between text-lg font-semibold pt-2 border-t">
              <span className="text-red-600">Shortages:</span>
              <span className="text-red-600">{getShortageCount()}</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {workOrder.notes && (
        <Card>
          <CardHeader>
            <CardTitle>Notes</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm whitespace-pre-wrap">{workOrder.notes}</p>
          </CardContent>
        </Card>
      )}

      {/* BOM vs Inventory Comparison */}
      <Card>
        <CardHeader>
          <CardTitle>BOM vs Inventory Comparison</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Status</TableHead>
                <TableHead>IPN</TableHead>
                <TableHead>Description</TableHead>
                <TableHead className="text-right">Required</TableHead>
                <TableHead className="text-right">On Hand</TableHead>
                <TableHead className="text-right">Shortage</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {bomData?.bom.map((item, index) => (
                <TableRow 
                  key={index}
                  className={getBOMRowClass(item.status)}
                >
                  <TableCell>
                    <div className="flex items-center gap-2">
                      {getBOMStatusIcon(item.status)}
                      <Badge 
                        variant={item.status === 'ok' ? 'default' : item.status === 'low' ? 'outline' : 'destructive'}
                        className="text-xs"
                      >
                        {item.status.toUpperCase()}
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell className="font-medium">{item.ipn}</TableCell>
                  <TableCell>{item.description || "—"}</TableCell>
                  <TableCell className="text-right font-mono">{item.qty_required}</TableCell>
                  <TableCell className="text-right font-mono">{item.qty_on_hand}</TableCell>
                  <TableCell className="text-right font-mono">
                    {item.shortage > 0 ? (
                      <span className="text-red-600 font-semibold">{item.shortage}</span>
                    ) : (
                      "—"
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {(!bomData?.bom || bomData.bom.length === 0) && (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                    No BOM data available for this work order
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
export default WorkOrderDetail;
