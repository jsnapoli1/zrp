import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { 
  ArrowLeft, 
  Package, 
  Plus,
  Minus,
  RotateCcw,
  Settings2,
  Calendar
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
import { api, type InventoryItem, type InventoryTransaction } from "../lib/api";

function InventoryDetail() {
  const { ipn } = useParams<{ ipn: string }>();
  const [item, setItem] = useState<InventoryItem | null>(null);
  const [transactions, setTransactions] = useState<InventoryTransaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [transactionDialogOpen, setTransactionDialogOpen] = useState(false);

  const [transactionForm, setTransactionForm] = useState({
    type: "receive",
    qty: "",
    reference: "",
    notes: "",
  });

  useEffect(() => {
    if (ipn) {
      fetchInventoryDetail();
      fetchTransactionHistory();
    }
  }, [ipn]);

  const fetchInventoryDetail = async () => {
    if (!ipn) return;
    
    try {
      setLoading(true);
      const data = await api.getInventoryItem(ipn);
      setItem(data);
    } catch (error) {
      console.error("Failed to fetch inventory item:", error);
    } finally {
      setLoading(false);
    }
  };

  const fetchTransactionHistory = async () => {
    if (!ipn) return;
    
    try {
      const data = await api.getInventoryHistory(ipn);
      setTransactions(data);
    } catch (error) {
      console.error("Failed to fetch transaction history:", error);
    }
  };

  const handleTransaction = async () => {
    if (!ipn) return;
    
    try {
      await api.createInventoryTransaction({
        ipn,
        type: transactionForm.type,
        qty: parseFloat(transactionForm.qty),
        reference: transactionForm.reference || undefined,
        notes: transactionForm.notes || undefined,
      });
      
      setTransactionDialogOpen(false);
      setTransactionForm({ type: "receive", qty: "", reference: "", notes: "" });
      fetchInventoryDetail();
      fetchTransactionHistory();
    } catch (error) {
      console.error("Failed to create transaction:", error);
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

  const getTransactionIcon = (type: string) => {
    switch (type) {
      case "receive":
        return <Plus className="h-4 w-4 text-green-600" />;
      case "issue":
        return <Minus className="h-4 w-4 text-red-600" />;
      case "adjust":
        return <Settings2 className="h-4 w-4 text-blue-600" />;
      case "return":
        return <RotateCcw className="h-4 w-4 text-orange-600" />;
      default:
        return <Package className="h-4 w-4 text-gray-600" />;
    }
  };

  const getTransactionBadgeVariant = (type: string) => {
    switch (type) {
      case "receive":
        return "default";
      case "issue":
        return "destructive";
      case "adjust":
        return "secondary";
      case "return":
        return "outline";
      default:
        return "secondary";
    }
  };

  const getAvailableQty = () => {
    if (!item) return 0;
    return Math.max(0, item.qty_on_hand - item.qty_reserved);
  };

  const isLowStock = () => {
    if (!item) return false;
    return item.reorder_point > 0 && item.qty_on_hand <= item.reorder_point;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading inventory item...</p>
        </div>
      </div>
    );
  }

  if (!item) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" asChild>
            <Link to="/inventory">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Inventory
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="p-8 text-center">
            <h3 className="text-lg font-semibold mb-2">Inventory Item Not Found</h3>
            <p className="text-muted-foreground">
              The inventory item "{ipn}" could not be found.
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
            <Link to="/inventory">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Inventory
            </Link>
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{item.ipn}</h1>
            <p className="text-muted-foreground">
              {item.description || "No description available"}
            </p>
          </div>
        </div>
        <Dialog open={transactionDialogOpen} onOpenChange={setTransactionDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              New Transaction
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create Inventory Transaction</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div>
                <Label htmlFor="type">Transaction Type</Label>
                <Select 
                  value={transactionForm.type} 
                  onValueChange={(value) => setTransactionForm(prev => ({ ...prev, type: value }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="receive">Receive</SelectItem>
                    <SelectItem value="issue">Issue</SelectItem>
                    <SelectItem value="adjust">Adjust</SelectItem>
                    <SelectItem value="return">Return</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="qty">Quantity</Label>
                <Input
                  id="qty"
                  type="number"
                  value={transactionForm.qty}
                  onChange={(e) => setTransactionForm(prev => ({ ...prev, qty: e.target.value }))}
                  placeholder="Enter quantity"
                />
                {transactionForm.type === "adjust" && (
                  <p className="text-xs text-muted-foreground mt-1">
                    For adjustments, enter the new total quantity
                  </p>
                )}
              </div>
              <div>
                <Label htmlFor="reference">Reference</Label>
                <Input
                  id="reference"
                  value={transactionForm.reference}
                  onChange={(e) => setTransactionForm(prev => ({ ...prev, reference: e.target.value }))}
                  placeholder="PO number, work order, etc."
                />
              </div>
              <div>
                <Label htmlFor="notes">Notes</Label>
                <Textarea
                  id="notes"
                  value={transactionForm.notes}
                  onChange={(e) => setTransactionForm(prev => ({ ...prev, notes: e.target.value }))}
                  placeholder="Additional notes (optional)"
                  rows={3}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setTransactionDialogOpen(false)}>
                Cancel
              </Button>
              <Button 
                onClick={handleTransaction}
                disabled={!transactionForm.qty}
              >
                Create Transaction
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Stock Level Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className={isLowStock() ? "border-red-200 bg-red-50" : ""}>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">On Hand</p>
                <p className="text-2xl font-bold">{item.qty_on_hand}</p>
              </div>
              <Package className="h-8 w-8 text-blue-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Reserved</p>
                <p className="text-2xl font-bold">{item.qty_reserved}</p>
              </div>
              <Minus className="h-8 w-8 text-orange-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Available</p>
                <p className="text-2xl font-bold">{getAvailableQty()}</p>
              </div>
              <Settings2 className="h-8 w-8 text-green-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Reorder Point</p>
                <p className="text-2xl font-bold">
                  {item.reorder_point}
                  {isLowStock() && (
                    <Badge variant="destructive" className="ml-2 text-xs">
                      LOW
                    </Badge>
                  )}
                </p>
              </div>
              <RotateCcw className="h-8 w-8 text-gray-600" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Item Details */}
      <Card>
        <CardHeader>
          <CardTitle>Item Details</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-3">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Internal Part Number</p>
                <p className="text-base">{item.ipn}</p>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Description</p>
                <p className="text-base">{item.description || "—"}</p>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Manufacturer Part Number</p>
                <p className="text-base">{item.mpn || "—"}</p>
              </div>
            </div>
            <div className="space-y-3">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Location</p>
                <p className="text-base">{item.location || "—"}</p>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Reorder Quantity</p>
                <p className="text-base">{item.reorder_qty}</p>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Last Updated</p>
                <p className="text-base">{formatDate(item.updated_at)}</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Transaction History */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            Transaction History
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Date</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="text-right">Quantity</TableHead>
                <TableHead>Reference</TableHead>
                <TableHead>Notes</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {transactions.map((transaction) => (
                <TableRow key={transaction.id}>
                  <TableCell>{formatDate(transaction.created_at)}</TableCell>
                  <TableCell>
                    <Badge 
                      variant={getTransactionBadgeVariant(transaction.type)}
                      className="flex items-center gap-1 w-fit"
                    >
                      {getTransactionIcon(transaction.type)}
                      {transaction.type.toUpperCase()}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right font-mono">
                    {transaction.type === "issue" && transaction.qty > 0 ? `-${transaction.qty}` : transaction.qty}
                  </TableCell>
                  <TableCell>{transaction.reference || "—"}</TableCell>
                  <TableCell>{transaction.notes || "—"}</TableCell>
                </TableRow>
              ))}
              {transactions.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                    No transaction history found for this item
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
export default InventoryDetail;
