import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Label } from "../components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { RotateCcw, Plus } from "lucide-react";
import { api, type RMA } from "../lib/api";

function RMAs() {
  const navigate = useNavigate();
  const [rmas, setRMAs] = useState<RMA[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formData, setFormData] = useState({
    serial_number: "",
    customer: "",
    reason: "",
    defect_description: "",
  });

  useEffect(() => {
    const fetchRMAs = async () => {
      try {
        const data = await api.getRMAs();
        setRMAs(data);
      } catch (error) {
        console.error("Failed to fetch RMAs:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchRMAs();
  }, []);

  const handleCreateRMA = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const newRMA = await api.createRMA(formData);
      setRMAs([newRMA, ...rmas]);
      setCreateDialogOpen(false);
      setFormData({ serial_number: "", customer: "", reason: "", defect_description: "" });
    } catch (error) {
      console.error("Failed to create RMA:", error);
    }
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case "closed":
      case "shipped":
        return "default";
      case "received":
        return "secondary";
      case "open":
      default:
        return "outline";
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading RMAs...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Return Merchandise Authorization</h1>
          <p className="text-muted-foreground">
            Manage device returns and warranty claims
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create RMA
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Create New RMA</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleCreateRMA} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="serial_number">Device Serial Number *</Label>
                  <Input
                    id="serial_number"
                    value={formData.serial_number}
                    onChange={(e) => setFormData({ ...formData, serial_number: e.target.value })}
                    placeholder="Device serial number"
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="customer">Customer *</Label>
                  <Input
                    id="customer"
                    value={formData.customer}
                    onChange={(e) => setFormData({ ...formData, customer: e.target.value })}
                    placeholder="Customer name"
                    required
                  />
                </div>
              </div>
              <div>
                <Label htmlFor="reason">Reason for Return *</Label>
                <Input
                  id="reason"
                  value={formData.reason}
                  onChange={(e) => setFormData({ ...formData, reason: e.target.value })}
                  placeholder="Brief reason for return"
                  required
                />
              </div>
              <div>
                <Label htmlFor="defect_description">Defect Description</Label>
                <Textarea
                  id="defect_description"
                  value={formData.defect_description}
                  onChange={(e) => setFormData({ ...formData, defect_description: e.target.value })}
                  placeholder="Detailed description of the defect or issue"
                  rows={3}
                />
              </div>
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit">Create RMA</Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <RotateCcw className="h-5 w-5" />
            RMA Records
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>RMA ID</TableHead>
                <TableHead>Customer</TableHead>
                <TableHead>Device S/N</TableHead>
                <TableHead>Reason</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rmas.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8">
                    <div className="text-muted-foreground">
                      No RMAs found. Create your first RMA to get started.
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                rmas.map((rma) => (
                  <TableRow key={rma.id} className="cursor-pointer hover:bg-muted/50" onClick={() => navigate(`/rmas/${rma.id}`)}>
                    <TableCell className="font-medium">{rma.id}</TableCell>
                    <TableCell>{rma.customer}</TableCell>
                    <TableCell className="font-mono text-sm">{rma.serial_number}</TableCell>
                    <TableCell>{rma.reason}</TableCell>
                    <TableCell>
                      <Badge variant={getStatusBadgeVariant(rma.status)}>
                        {rma.status.charAt(0).toUpperCase() + rma.status.slice(1)}
                      </Badge>
                    </TableCell>
                    <TableCell>{new Date(rma.created_at).toLocaleDateString()}</TableCell>
                    <TableCell>
                      <Button variant="outline" size="sm" onClick={(e) => { e.stopPropagation(); navigate(`/rmas/${rma.id}`); }}>
                        View Details
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
export default RMAs;
