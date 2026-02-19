import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Label } from "../components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Plus, ShieldCheck } from "lucide-react";
import { api, type CAPA, type CAPADashboard } from "../lib/api";
import { toast } from "sonner";
function CAPAs() {
  const navigate = useNavigate();
  const [capas, setCAPAs] = useState<CAPA[]>([]);
  const [dashboard, setDashboard] = useState<CAPADashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formData, setFormData] = useState({
    title: "",
    type: "corrective",
    root_cause: "",
    action_plan: "",
    owner: "",
    due_date: "",
    linked_ncr_id: "",
    linked_rma_id: "",
  });

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [capaData, dashData] = await Promise.all([
          api.getCAPAs(),
          api.getCAPADashboard(),
        ]);
        setCAPAs(capaData);
        setDashboard(dashData);
      } catch (error) {
        toast.error("Failed to fetch CAPAs"); console.error("Failed to fetch CAPAs:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const newCAPA = await api.createCAPA(formData);
      setCAPAs([newCAPA, ...capas]);
      setCreateDialogOpen(false);
      setFormData({ title: "", type: "corrective", root_cause: "", action_plan: "", owner: "", due_date: "", linked_ncr_id: "", linked_rma_id: "" });
    } catch (error) {
      toast.error("Failed to create CAPA"); console.error("Failed to create CAPA:", error);
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "open": return "secondary";
      case "in-progress": return "default";
      case "verification": return "outline";
      case "closed": return "secondary";
      default: return "outline";
    }
  };

  const getTypeBadge = (type: string) => {
    return type === "corrective" ? "destructive" : "default";
  };

  if (loading) {
    return <div className="flex items-center justify-center min-h-[400px]"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" /></div>;
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ShieldCheck className="h-6 w-6" />
          <h1 className="text-2xl font-bold">CAPAs</h1>
          <Badge variant="outline">{capas.length}</Badge>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="mr-2 h-4 w-4" />New CAPA</Button>
          </DialogTrigger>
          <DialogContent className="max-w-lg">
            <DialogHeader><DialogTitle>Create CAPA</DialogTitle></DialogHeader>
            <form onSubmit={handleCreate} className="space-y-4">
              <div><Label>Title</Label><Input value={formData.title} onChange={(e) => setFormData({ ...formData, title: e.target.value })} required /></div>
              <div><Label>Type</Label>
                <Select value={formData.type} onValueChange={(v) => setFormData({ ...formData, type: v })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="corrective">Corrective</SelectItem>
                    <SelectItem value="preventive">Preventive</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div><Label>Owner</Label><Input value={formData.owner} onChange={(e) => setFormData({ ...formData, owner: e.target.value })} /></div>
              <div><Label>Due Date</Label><Input type="date" value={formData.due_date} onChange={(e) => setFormData({ ...formData, due_date: e.target.value })} /></div>
              <div><Label>Root Cause</Label><Textarea value={formData.root_cause} onChange={(e) => setFormData({ ...formData, root_cause: e.target.value })} /></div>
              <div><Label>Action Plan</Label><Textarea value={formData.action_plan} onChange={(e) => setFormData({ ...formData, action_plan: e.target.value })} /></div>
              <div><Label>Linked NCR ID</Label><Input value={formData.linked_ncr_id} onChange={(e) => setFormData({ ...formData, linked_ncr_id: e.target.value })} placeholder="e.g. NCR-001" /></div>
              <div><Label>Linked RMA ID</Label><Input value={formData.linked_rma_id} onChange={(e) => setFormData({ ...formData, linked_rma_id: e.target.value })} placeholder="e.g. RMA-001" /></div>
              <Button type="submit" className="w-full">Create CAPA</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {/* Dashboard summary */}
      {dashboard && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium">Open CAPAs</CardTitle></CardHeader>
            <CardContent><div className="text-2xl font-bold">{dashboard.total_open}</div></CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium text-red-600">Overdue</CardTitle></CardHeader>
            <CardContent><div className="text-2xl font-bold text-red-600">{dashboard.total_overdue}</div></CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2"><CardTitle className="text-sm font-medium">By Owner</CardTitle></CardHeader>
            <CardContent>
              <div className="space-y-1 text-sm">
                {dashboard.by_owner.map((o) => (
                  <div key={o.owner} className="flex justify-between">
                    <span>{o.owner || "Unassigned"}</span>
                    <span>{o.count} {o.overdue > 0 && <span className="text-red-500">({o.overdue} overdue)</span>}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Title</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Owner</TableHead>
                <TableHead>Due Date</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Linked</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {capas.map((c) => (
                <TableRow key={c.id} className="cursor-pointer hover:bg-muted/50" onClick={() => navigate(`/capas/${c.id}`)}>
                  <TableCell className="font-mono text-sm">{c.id}</TableCell>
                  <TableCell>{c.title}</TableCell>
                  <TableCell><Badge variant={getTypeBadge(c.type)}>{c.type}</Badge></TableCell>
                  <TableCell>{c.owner}</TableCell>
                  <TableCell>{c.due_date}</TableCell>
                  <TableCell><Badge variant={getStatusBadge(c.status)}>{c.status}</Badge></TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {c.linked_ncr_id && <span>NCR: {c.linked_ncr_id} </span>}
                    {c.linked_rma_id && <span>RMA: {c.linked_rma_id}</span>}
                  </TableCell>
                </TableRow>
              ))}
              {capas.length === 0 && (
                <TableRow><TableCell colSpan={7} className="text-center py-8 text-muted-foreground">No CAPAs found</TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

export default CAPAs;
