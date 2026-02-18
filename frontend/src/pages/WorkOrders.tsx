import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { 
  Wrench, 
  Plus, 
  Clock,
  Play,
  CheckCircle,
  AlertTriangle,
  Calendar
} from "lucide-react";
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
import { api, type WorkOrder, type Part } from "../lib/api";

function WorkOrders() {
  const [workOrders, setWorkOrders] = useState<WorkOrder[]>([]);
  const [parts, setParts] = useState<Part[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  const [woForm, setWoForm] = useState({
    assembly_ipn: "",
    qty: 1,
    status: "open",
    priority: "medium",
    notes: "",
  });

  useEffect(() => {
    fetchWorkOrders();
    fetchParts();
  }, []);

  const fetchWorkOrders = async () => {
    try {
      setLoading(true);
      const data = await api.getWorkOrders();
      setWorkOrders(data);
    } catch (error) {
      console.error("Failed to fetch work orders:", error);
    } finally {
      setLoading(false);
    }
  };

  const fetchParts = async () => {
    try {
      const data = await api.getParts();
      // Filter to assemblies only (this would typically be based on a category or type field)
      const partsArray = Array.isArray(data) ? data : [];
      setParts(partsArray);
    } catch (error) {
      console.error("Failed to fetch parts:", error);
    }
  };

  const handleCreateWO = async () => {
    try {
      await api.createWorkOrder(woForm);
      setCreateDialogOpen(false);
      resetForm();
      fetchWorkOrders();
    } catch (error) {
      console.error("Failed to create work order:", error);
    }
  };

  const resetForm = () => {
    setWoForm({
      assembly_ipn: "",
      qty: 1,
      status: "open",
      priority: "medium",
      notes: "",
    });
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
        return <Clock className="h-4 w-4 text-gray-600" />;
      case "in_progress":
        return <Play className="h-4 w-4 text-blue-600" />;
      case "completed":
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      case "on_hold":
        return <AlertTriangle className="h-4 w-4 text-orange-600" />;
      default:
        return <Clock className="h-4 w-4 text-gray-600" />;
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  // Removed unused formatDateTime function

  const getDaysOld = (dateStr: string) => {
    const now = new Date();
    const created = new Date(dateStr);
    const diffTime = Math.abs(now.getTime() - created.getTime());
    return Math.ceil(diffTime / (1000 * 60 * 60 * 24));
  };

  const filteredParts = parts.filter(part => 
    part.ipn.toLowerCase().includes(woForm.assembly_ipn.toLowerCase())
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading work orders...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Work Orders</h1>
          <p className="text-muted-foreground">
            Manage production work orders and assembly tracking.
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Work Order
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create Work Order</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div>
                <Label htmlFor="assembly_ipn">Assembly IPN *</Label>
                <Input
                  id="assembly_ipn"
                  value={woForm.assembly_ipn}
                  onChange={(e) => setWoForm(prev => ({ ...prev, assembly_ipn: e.target.value }))}
                  placeholder="Search for assembly..."
                />
                {woForm.assembly_ipn && filteredParts.length > 0 && (
                  <div className="mt-2 border rounded-md max-h-40 overflow-y-auto">
                    {filteredParts.slice(0, 5).map((part) => (
                      <div
                        key={part.ipn}
                        className="p-2 hover:bg-muted cursor-pointer"
                        onClick={() => setWoForm(prev => ({ ...prev, assembly_ipn: part.ipn }))}
                      >
                        <div className="font-medium">{part.ipn}</div>
                        {part.description && (
                          <div className="text-sm text-muted-foreground">{part.description}</div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="qty">Quantity *</Label>
                  <Input
                    id="qty"
                    type="number"
                    min="1"
                    value={woForm.qty}
                    onChange={(e) => setWoForm(prev => ({ ...prev, qty: parseInt(e.target.value) || 1 }))}
                    placeholder="1"
                  />
                </div>
                <div>
                  <Label htmlFor="priority">Priority</Label>
                  <Select value={woForm.priority} onValueChange={(value) => setWoForm(prev => ({ ...prev, priority: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="low">Low</SelectItem>
                      <SelectItem value="medium">Medium</SelectItem>
                      <SelectItem value="high">High</SelectItem>
                      <SelectItem value="critical">Critical</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div>
                <Label htmlFor="notes">Notes</Label>
                <Textarea
                  id="notes"
                  value={woForm.notes}
                  onChange={(e) => setWoForm(prev => ({ ...prev, notes: e.target.value }))}
                  placeholder="Optional work order notes..."
                  rows={3}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreateWO} disabled={!woForm.assembly_ipn}>
                Create Work Order
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Total WOs</p>
                <p className="text-2xl font-bold">{workOrders.length}</p>
              </div>
              <Wrench className="h-8 w-8 text-blue-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Open</p>
                <p className="text-2xl font-bold text-gray-600">
                  {workOrders.filter(wo => wo.status === 'open').length}
                </p>
              </div>
              <Clock className="h-8 w-8 text-gray-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">In Progress</p>
                <p className="text-2xl font-bold text-blue-600">
                  {workOrders.filter(wo => wo.status === 'in_progress').length}
                </p>
              </div>
              <Play className="h-8 w-8 text-blue-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">On Hold</p>
                <p className="text-2xl font-bold text-orange-600">
                  {workOrders.filter(wo => wo.status === 'on_hold').length}
                </p>
              </div>
              <AlertTriangle className="h-8 w-8 text-orange-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Completed</p>
                <p className="text-2xl font-bold text-green-600">
                  {workOrders.filter(wo => wo.status === 'completed').length}
                </p>
              </div>
              <CheckCircle className="h-8 w-8 text-green-600" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Work Orders Table */}
      <Card>
        <CardHeader>
          <CardTitle>Work Orders</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>WO ID</TableHead>
                <TableHead>Assembly</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Priority</TableHead>
                <TableHead className="text-right">Qty</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Age</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {workOrders.map((wo) => (
                <TableRow key={wo.id}>
                  <TableCell>
                    <Link 
                      to={`/work-orders/${wo.id}`}
                      className="font-medium text-blue-600 hover:underline"
                    >
                      {wo.id}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <div>
                      <div className="font-medium">{wo.assembly_ipn}</div>
                      {parts.find(p => p.ipn === wo.assembly_ipn)?.description && (
                        <div className="text-sm text-muted-foreground">
                          {parts.find(p => p.ipn === wo.assembly_ipn)?.description}
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      {getStatusIcon(wo.status)}
                      {getStatusBadge(wo.status)}
                    </div>
                  </TableCell>
                  <TableCell>{getPriorityBadge(wo.priority)}</TableCell>
                  <TableCell className="text-right font-mono">{wo.qty}</TableCell>
                  <TableCell>{formatDate(wo.created_at)}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Calendar className="h-3 w-3 text-muted-foreground" />
                      <span className="text-sm text-muted-foreground">
                        {getDaysOld(wo.created_at)}d
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/work-orders/${wo.id}`}>
                          View Details
                        </Link>
                      </Button>
                      <Button variant="outline" size="sm" asChild>
                        <Link to={`/work-orders/${wo.id}/bom`}>
                          BOM
                        </Link>
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
              {workOrders.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                    No work orders found
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
export default WorkOrders;
