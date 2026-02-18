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
import { Progress } from "../components/ui/progress";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Cpu, Plus, Play, Pause } from "lucide-react";
import { api, type FirmwareCampaign } from "../lib/api";

function Firmware() {
  const navigate = useNavigate();
  const [campaigns, setCampaigns] = useState<FirmwareCampaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    version: "",
    category: "",
    target_filter: "",
    notes: "",
  });

  useEffect(() => {
    const fetchCampaigns = async () => {
      try {
        const data = await api.getFirmwareCampaigns();
        setCampaigns(data);
      } catch (error) {
        console.error("Failed to fetch firmware campaigns:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchCampaigns();
  }, []);

  const handleCreateCampaign = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const newCampaign = await api.createFirmwareCampaign(formData);
      setCampaigns([newCampaign, ...campaigns]);
      setCreateDialogOpen(false);
      setFormData({
        name: "",
        version: "",
        category: "",
        target_filter: "",
        notes: "",
      });
    } catch (error) {
      console.error("Failed to create firmware campaign:", error);
    }
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case "completed":
        return "default";
      case "running":
        return "secondary";
      case "paused":
        return "outline";
      case "draft":
        return "outline";
      case "failed":
        return "destructive";
      default:
        return "outline";
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "completed":
        return "text-green-600";
      case "running":
        return "text-blue-600";
      case "paused":
        return "text-yellow-600";
      case "draft":
        return "text-gray-600";
      case "failed":
        return "text-red-600";
      default:
        return "text-gray-600";
    }
  };

  // Mock progress calculation (in real app, this would come from API)
  const getProgress = (campaign: FirmwareCampaign) => {
    if (campaign.status === "completed") return 100;
    if (campaign.status === "draft") return 0;
    if (campaign.status === "failed") return 30; // Example failed at 30%
    if (campaign.status === "running") return 65; // Example progress
    if (campaign.status === "paused") return 45; // Example paused at 45%
    return 0;
  };

  const categories = [
    "Security Update",
    "Feature Update", 
    "Bug Fix",
    "Performance Update",
    "Critical Patch",
    "Beta Release",
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading firmware campaigns...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Firmware Management</h1>
          <p className="text-muted-foreground">
            Manage firmware update campaigns across device fleet
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Campaign
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Create Firmware Campaign</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleCreateCampaign} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="name">Campaign Name *</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    placeholder="e.g., Security Update Q1 2024"
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="version">Target Version *</Label>
                  <Input
                    id="version"
                    value={formData.version}
                    onChange={(e) => setFormData({ ...formData, version: e.target.value })}
                    placeholder="e.g., v2.1.5"
                    required
                  />
                </div>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="category">Category</Label>
                  <Select value={formData.category} onValueChange={(value) => setFormData({ ...formData, category: value })}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select category" />
                    </SelectTrigger>
                    <SelectContent>
                      {categories.map((category) => (
                        <SelectItem key={category} value={category}>
                          {category}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor="target_filter">Target Filter</Label>
                  <Input
                    id="target_filter"
                    value={formData.target_filter}
                    onChange={(e) => setFormData({ ...formData, target_filter: e.target.value })}
                    placeholder="e.g., ipn:DEV-001 OR customer:ACME"
                  />
                </div>
              </div>

              <div>
                <Label htmlFor="notes">Notes</Label>
                <Textarea
                  id="notes"
                  value={formData.notes}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  placeholder="Campaign description and release notes"
                  rows={3}
                />
              </div>

              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit">Create Campaign</Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Cpu className="h-5 w-5" />
            Firmware Campaigns
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Campaign ID</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Version</TableHead>
                <TableHead>Progress</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {campaigns.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8">
                    <div className="text-muted-foreground">
                      No firmware campaigns found. Create your first campaign to get started.
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                campaigns.map((campaign) => {
                  const progress = getProgress(campaign);
                  
                  return (
                    <TableRow key={campaign.id} className="cursor-pointer hover:bg-muted/50" onClick={() => navigate(`/firmware/${campaign.id}`)}>
                      <TableCell className="font-medium">{campaign.id}</TableCell>
                      <TableCell>{campaign.name}</TableCell>
                      <TableCell className="font-mono">{campaign.version}</TableCell>
                      <TableCell className="w-32">
                        <div className="flex items-center gap-2">
                          <Progress value={progress} className="flex-1" />
                          <span className="text-sm text-muted-foreground w-10">{progress}%</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={getStatusBadgeVariant(campaign.status)} className={getStatusColor(campaign.status)}>
                          {campaign.status.charAt(0).toUpperCase() + campaign.status.slice(1)}
                        </Badge>
                      </TableCell>
                      <TableCell>{new Date(campaign.created_at).toLocaleDateString()}</TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          {campaign.status === "running" ? (
                            <Button variant="outline" size="sm">
                              <Pause className="h-3 w-3" />
                            </Button>
                          ) : campaign.status === "paused" || campaign.status === "draft" ? (
                            <Button variant="outline" size="sm">
                              <Play className="h-3 w-3" />
                            </Button>
                          ) : null}
                          <Button 
                            variant="outline" 
                            size="sm" 
                            onClick={(e) => { 
                              e.stopPropagation(); 
                              navigate(`/firmware/${campaign.id}`); 
                            }}
                          >
                            View
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Campaign Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <Cpu className="h-8 w-8 text-blue-600" />
            </div>
            <div className="text-3xl font-bold text-blue-600 text-center">
              {campaigns.length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Total Campaigns
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <Play className="h-8 w-8 text-green-600" />
            </div>
            <div className="text-3xl font-bold text-green-600 text-center">
              {campaigns.filter(c => c.status === "running").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Running
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <div className="h-8 w-8 rounded-full bg-green-500 flex items-center justify-center">
                ✓
              </div>
            </div>
            <div className="text-3xl font-bold text-green-600 text-center">
              {campaigns.filter(c => c.status === "completed").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Completed
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <Pause className="h-8 w-8 text-yellow-600" />
            </div>
            <div className="text-3xl font-bold text-yellow-600 text-center">
              {campaigns.filter(c => c.status === "paused" || c.status === "draft").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Paused/Draft
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
export default Firmware;
