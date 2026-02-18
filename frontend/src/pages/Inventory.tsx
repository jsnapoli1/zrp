import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { 
  Package, 
  AlertTriangle, 
  Plus, 
  MoreHorizontal,
  Trash2
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Checkbox } from "../components/ui/checkbox";
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../components/ui/dropdown-menu";
import { api, type InventoryItem } from "../lib/api";

function Inventory() {
  const [inventory, setInventory] = useState<InventoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [showLowStock, setShowLowStock] = useState(false);
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [receiveDialogOpen, setReceiveDialogOpen] = useState(false);
  const [parts, setParts] = useState<{ ipn: string; description?: string }[]>([]);

  // Quick receive form state
  const [receiveForm, setReceiveForm] = useState({
    ipn: "",
    qty: "",
    reference: "",
    notes: "",
  });

  useEffect(() => {
    fetchInventory();
    fetchParts();
  }, [showLowStock]);

  const fetchInventory = async () => {
    try {
      setLoading(true);
      const data = await api.getInventory(showLowStock);
      setInventory(data);
    } catch (error) {
      console.error("Failed to fetch inventory:", error);
    } finally {
      setLoading(false);
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

  const handleQuickReceive = async () => {
    try {
      await api.createInventoryTransaction({
        ipn: receiveForm.ipn,
        type: "receive",
        qty: parseFloat(receiveForm.qty),
        reference: receiveForm.reference || undefined,
        notes: receiveForm.notes || undefined,
      });
      
      setReceiveDialogOpen(false);
      setReceiveForm({ ipn: "", qty: "", reference: "", notes: "" });
      fetchInventory(); // Refresh list
    } catch (error) {
      console.error("Failed to receive inventory:", error);
    }
  };

  const handleBulkDelete = async () => {
    if (selectedItems.size === 0 || !confirm(`Delete ${selectedItems.size} inventory items?`)) {
      return;
    }
    
    try {
      await api.bulkDeleteInventory(Array.from(selectedItems));
      setSelectedItems(new Set());
      fetchInventory();
    } catch (error) {
      console.error("Failed to delete items:", error);
    }
  };

  const toggleSelectAll = () => {
    if (selectedItems.size === inventory.length) {
      setSelectedItems(new Set());
    } else {
      setSelectedItems(new Set(inventory.map(item => item.ipn)));
    }
  };

  const toggleSelectItem = (ipn: string) => {
    const newSelected = new Set(selectedItems);
    if (newSelected.has(ipn)) {
      newSelected.delete(ipn);
    } else {
      newSelected.add(ipn);
    }
    setSelectedItems(newSelected);
  };

  const getAvailableQty = (item: InventoryItem) => {
    return Math.max(0, item.qty_on_hand - item.qty_reserved);
  };

  const isLowStock = (item: InventoryItem) => {
    return item.reorder_point > 0 && item.qty_on_hand <= item.reorder_point;
  };

  const filteredParts = parts.filter(part => 
    part.ipn.toLowerCase().includes(receiveForm.ipn.toLowerCase())
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading inventory...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Inventory</h1>
          <p className="text-muted-foreground">
            Manage your inventory levels and stock tracking.
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant={showLowStock ? "default" : "outline"}
            onClick={() => setShowLowStock(!showLowStock)}
          >
            <AlertTriangle className="h-4 w-4 mr-2" />
            {showLowStock ? "Show All" : "Low Stock"}
          </Button>
          <Dialog open={receiveDialogOpen} onOpenChange={setReceiveDialogOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Quick Receive
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Quick Receive Inventory</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <Label htmlFor="ipn">IPN</Label>
                  <Input
                    id="ipn"
                    value={receiveForm.ipn}
                    onChange={(e) => setReceiveForm(prev => ({ ...prev, ipn: e.target.value }))}
                    placeholder="Search IPN..."
                  />
                  {receiveForm.ipn && filteredParts.length > 0 && (
                    <div className="mt-2 border rounded-md max-h-40 overflow-y-auto">
                      {filteredParts.slice(0, 5).map((part) => (
                        <div
                          key={part.ipn}
                          className="p-2 hover:bg-muted cursor-pointer"
                          onClick={() => setReceiveForm(prev => ({ ...prev, ipn: part.ipn }))}
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
                <div>
                  <Label htmlFor="qty">Quantity</Label>
                  <Input
                    id="qty"
                    type="number"
                    value={receiveForm.qty}
                    onChange={(e) => setReceiveForm(prev => ({ ...prev, qty: e.target.value }))}
                    placeholder="0"
                  />
                </div>
                <div>
                  <Label htmlFor="reference">Reference</Label>
                  <Input
                    id="reference"
                    value={receiveForm.reference}
                    onChange={(e) => setReceiveForm(prev => ({ ...prev, reference: e.target.value }))}
                    placeholder="PO number, invoice, etc."
                  />
                </div>
                <div>
                  <Label htmlFor="notes">Notes</Label>
                  <Input
                    id="notes"
                    value={receiveForm.notes}
                    onChange={(e) => setReceiveForm(prev => ({ ...prev, notes: e.target.value }))}
                    placeholder="Optional notes"
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setReceiveDialogOpen(false)}>
                  Cancel
                </Button>
                <Button 
                  onClick={handleQuickReceive}
                  disabled={!receiveForm.ipn || !receiveForm.qty}
                >
                  Receive
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Total Items</p>
                <p className="text-2xl font-bold">{inventory.length}</p>
              </div>
              <Package className="h-8 w-8 text-blue-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Low Stock Items</p>
                <p className="text-2xl font-bold text-red-600">
                  {inventory.filter(isLowStock).length}
                </p>
              </div>
              <AlertTriangle className="h-8 w-8 text-red-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Selected</p>
                <p className="text-2xl font-bold">{selectedItems.size}</p>
              </div>
              <Checkbox
                checked={selectedItems.size === inventory.length && inventory.length > 0}
                onCheckedChange={toggleSelectAll}
              />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            {selectedItems.size > 0 ? (
              <Button 
                variant="destructive" 
                className="w-full" 
                onClick={handleBulkDelete}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete Selected
              </Button>
            ) : (
              <div className="text-center text-muted-foreground">
                Select items for bulk actions
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Inventory Table */}
      <Card>
        <CardHeader>
          <CardTitle>Inventory Items</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">
                  <Checkbox
                    checked={selectedItems.size === inventory.length && inventory.length > 0}
                    onCheckedChange={toggleSelectAll}
                  />
                </TableHead>
                <TableHead>IPN</TableHead>
                <TableHead>Description</TableHead>
                <TableHead className="text-right">On Hand</TableHead>
                <TableHead className="text-right">Reserved</TableHead>
                <TableHead className="text-right">Available</TableHead>
                <TableHead>Location</TableHead>
                <TableHead className="text-right">Reorder Point</TableHead>
                <TableHead className="w-10"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {inventory.map((item) => (
                <TableRow 
                  key={item.ipn}
                  className={isLowStock(item) ? "bg-red-50" : ""}
                >
                  <TableCell>
                    <Checkbox
                      checked={selectedItems.has(item.ipn)}
                      onCheckedChange={() => toggleSelectItem(item.ipn)}
                    />
                  </TableCell>
                  <TableCell>
                    <Link 
                      to={`/inventory/${item.ipn}`}
                      className="font-medium text-blue-600 hover:underline"
                    >
                      {item.ipn}
                    </Link>
                  </TableCell>
                  <TableCell>{item.description || "—"}</TableCell>
                  <TableCell className="text-right font-mono">{item.qty_on_hand}</TableCell>
                  <TableCell className="text-right font-mono">{item.qty_reserved}</TableCell>
                  <TableCell className="text-right font-mono">{getAvailableQty(item)}</TableCell>
                  <TableCell>{item.location || "—"}</TableCell>
                  <TableCell className="text-right font-mono">
                    <div className="flex items-center justify-end gap-2">
                      {item.reorder_point}
                      {isLowStock(item) && (
                        <Badge variant="destructive" className="text-xs">
                          <AlertTriangle className="h-3 w-3" />
                        </Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem asChild>
                          <Link to={`/inventory/${item.ipn}`}>View Details</Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem 
                          onClick={() => {
                            setReceiveForm(prev => ({ ...prev, ipn: item.ipn }));
                            setReceiveDialogOpen(true);
                          }}
                        >
                          Quick Receive
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
              {inventory.length === 0 && (
                <TableRow>
                  <TableCell colSpan={9} className="text-center py-8 text-muted-foreground">
                    {showLowStock ? "No low stock items found" : "No inventory items found"}
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
export default Inventory;
