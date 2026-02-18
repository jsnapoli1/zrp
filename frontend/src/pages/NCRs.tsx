import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Label } from "../components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { AlertTriangle, Plus } from "lucide-react";
import { api, type NCR } from "../lib/api";

function NCRs() {
  const navigate = useNavigate();
  const [ncrs, setNCRs] = useState<NCR[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formData, setFormData] = useState({
    title: "",
    description: "",
    severity: "minor",
    ipn: "",
  });

  useEffect(() => {
    const fetchNCRs = async () => {
      try {
        const data = await api.getNCRs();
        setNCRs(data);
      } catch (error) {
        console.error("Failed to fetch NCRs:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchNCRs();
  }, []);

  const handleCreateNCR = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const newNCR = await api.createNCR(formData);
      setNCRs([newNCR, ...ncrs]);
      setCreateDialogOpen(false);
      setFormData({ title: "", description: "", severity: "minor", ipn: "" });
    } catch (error) {
      console.error("Failed to create NCR:", error);
    }
  };

  const getSeverityBadgeVariant = (severity: string) => {
    switch (severity) {
      case "critical":
        return "destructive";
      case "major":
        return "secondary";
      case "minor":
      default:
        return "outline";
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case "critical":
        return "bg-red-500";
      case "major":
        return "bg-orange-500";
      case "minor":
      default:
        return "bg-yellow-500";
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading NCRs...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Non-Conformance Reports</h1>
          <p className="text-muted-foreground">
            Track quality issues and corrective actions
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create NCR
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Create New NCR</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleCreateNCR} className="space-y-4">
              <div>
                <Label htmlFor="title">Title *</Label>
                <Input
                  id="title"
                  value={formData.title}
                  onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                  placeholder="Brief description of the non-conformance"
                  required
                />
              </div>
              <div>
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Detailed description of the issue"
                  rows={3}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="severity">Severity *</Label>
                  <Select value={formData.severity} onValueChange={(value) => setFormData({ ...formData, severity: value })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="minor">Minor</SelectItem>
                      <SelectItem value="major">Major</SelectItem>
                      <SelectItem value="critical">Critical</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor="ipn">Affected IPN</Label>
                  <Input
                    id="ipn"
                    value={formData.ipn}
                    onChange={(e) => setFormData({ ...formData, ipn: e.target.value })}
                    placeholder="Part number affected"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit">Create NCR</Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5" />
            NCR Records
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>NCR ID</TableHead>
                <TableHead>Title</TableHead>
                <TableHead>Severity</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Date</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {ncrs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8">
                    <div className="text-muted-foreground">
                      No NCRs found. Create your first NCR to get started.
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                ncrs.map((ncr) => (
                  <TableRow key={ncr.id} className="cursor-pointer hover:bg-muted/50" onClick={() => navigate(`/ncrs/${ncr.id}`)}>
                    <TableCell className="font-medium">{ncr.id}</TableCell>
                    <TableCell>{ncr.title}</TableCell>
                    <TableCell>
                      <Badge variant={getSeverityBadgeVariant(ncr.severity)} className="flex items-center gap-1 w-fit">
                        <div className={`w-2 h-2 rounded-full ${getSeverityColor(ncr.severity)}`} />
                        {ncr.severity.charAt(0).toUpperCase() + ncr.severity.slice(1)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={ncr.status === "closed" || ncr.status === "resolved" ? "default" : "secondary"}>
                        {ncr.status.charAt(0).toUpperCase() + ncr.status.slice(1)}
                      </Badge>
                    </TableCell>
                    <TableCell>{new Date(ncr.created_at).toLocaleDateString()}</TableCell>
                    <TableCell>
                      <Button variant="outline" size="sm" onClick={(e) => { e.stopPropagation(); navigate(`/ncrs/${ncr.id}`); }}>
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
export default NCRs;
