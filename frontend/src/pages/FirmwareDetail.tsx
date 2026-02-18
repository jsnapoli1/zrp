import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Progress } from "../components/ui/progress";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Separator } from "../components/ui/separator";
import { Label } from "../components/ui/label";
import { Cpu, ArrowLeft, Play, Pause, RotateCcw } from "lucide-react";
import { api, type FirmwareCampaign, type CampaignDevice } from "../lib/api";

function FirmwareDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [campaign, setCampaign] = useState<FirmwareCampaign | null>(null);
  const [devices, setDevices] = useState<CampaignDevice[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      if (!id) return;
      
      try {
        const [campaignData, devicesData] = await Promise.all([
          api.getFirmwareCampaign(id),
          api.getCampaignDevices(id)
        ]);
        
        setCampaign(campaignData);
        setDevices(devicesData);
      } catch (error) {
        console.error("Failed to fetch campaign data:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();

    // Poll for updates every 5 seconds if campaign is running
    const interval = setInterval(() => {
      if (campaign?.status === "running") {
        fetchData();
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [id, campaign?.status]);

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case "completed":
      case "success":
        return "default";
      case "running":
      case "in_progress":
        return "secondary";
      case "pending":
        return "outline";
      case "failed":
      case "error":
        return "destructive";
      default:
        return "outline";
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "completed":
      case "success":
        return "text-green-600";
      case "running":
      case "in_progress":
        return "text-blue-600";
      case "pending":
        return "text-yellow-600";
      case "failed":
      case "error":
        return "text-red-600";
      default:
        return "text-gray-600";
    }
  };

  const calculateProgress = () => {
    if (devices.length === 0) return 0;
    const completedDevices = devices.filter(d => d.status === "completed" || d.status === "success").length;
    return Math.round((completedDevices / devices.length) * 100);
  };

  const getDeviceStats = () => {
    const stats = {
      total: devices.length,
      completed: devices.filter(d => d.status === "completed" || d.status === "success").length,
      in_progress: devices.filter(d => d.status === "in_progress" || d.status === "running").length,
      pending: devices.filter(d => d.status === "pending").length,
      failed: devices.filter(d => d.status === "failed" || d.status === "error").length,
    };
    return stats;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading campaign...</p>
        </div>
      </div>
    );
  }

  if (!campaign) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/firmware")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Firmware
          </Button>
        </div>
        <div className="text-center py-8">
          <h2 className="text-2xl font-semibold mb-2">Campaign Not Found</h2>
          <p className="text-muted-foreground">The requested firmware campaign could not be found.</p>
        </div>
      </div>
    );
  }

  const progress = calculateProgress();
  const stats = getDeviceStats();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/firmware")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Firmware
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{campaign.name}</h1>
            <p className="text-muted-foreground">
              Target Version: <span className="font-mono">{campaign.version}</span>
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          {campaign.status === "running" ? (
            <Button variant="outline">
              <Pause className="h-4 w-4 mr-2" />
              Pause Campaign
            </Button>
          ) : campaign.status === "paused" || campaign.status === "draft" ? (
            <Button>
              <Play className="h-4 w-4 mr-2" />
              Start Campaign
            </Button>
          ) : null}
          
          {campaign.status === "failed" && (
            <Button variant="outline">
              <RotateCcw className="h-4 w-4 mr-2" />
              Retry Failed
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Progress Overview */}
        <div className="lg:col-span-3">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Cpu className="h-5 w-5" />
                Campaign Progress
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span>Overall Progress</span>
                  <span>{progress}% ({stats.completed} of {stats.total} devices)</span>
                </div>
                <Progress value={progress} className="h-3" />
              </div>

              <div className="grid grid-cols-4 gap-4">
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">{stats.completed}</div>
                  <div className="text-sm text-muted-foreground">Completed</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-blue-600">{stats.in_progress}</div>
                  <div className="text-sm text-muted-foreground">In Progress</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-yellow-600">{stats.pending}</div>
                  <div className="text-sm text-muted-foreground">Pending</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-red-600">{stats.failed}</div>
                  <div className="text-sm text-muted-foreground">Failed</div>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Device List */}
          <Card className="mt-6">
            <CardHeader>
              <CardTitle>Device Update Status</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Serial Number</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Updated At</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {devices.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center py-8">
                        <div className="text-muted-foreground">
                          No devices found for this campaign.
                        </div>
                      </TableCell>
                    </TableRow>
                  ) : (
                    devices.map((device) => (
                      <TableRow key={device.serial_number}>
                        <TableCell 
                          className="font-mono cursor-pointer hover:text-blue-600" 
                          onClick={() => navigate(`/devices/${device.serial_number}`)}
                        >
                          {device.serial_number}
                        </TableCell>
                        <TableCell>
                          <Badge 
                            variant={getStatusBadgeVariant(device.status)} 
                            className={getStatusColor(device.status)}
                          >
                            {device.status.charAt(0).toUpperCase() + device.status.slice(1).replace('_', ' ')}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {device.updated_at ? new Date(device.updated_at).toLocaleString() : "—"}
                        </TableCell>
                        <TableCell>
                          <Button 
                            variant="outline" 
                            size="sm"
                            onClick={() => navigate(`/devices/${device.serial_number}`)}
                          >
                            View Device
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

        {/* Sidebar */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Campaign Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <Label>Status</Label>
                <div className="mt-1">
                  <Badge 
                    variant={getStatusBadgeVariant(campaign.status)} 
                    className={getStatusColor(campaign.status)}
                  >
                    {campaign.status.charAt(0).toUpperCase() + campaign.status.slice(1)}
                  </Badge>
                </div>
              </div>

              <div>
                <Label>Category</Label>
                <p className="text-sm mt-1">{campaign.category || "Not specified"}</p>
              </div>

              <div>
                <Label>Target Filter</Label>
                <p className="text-sm mt-1 font-mono text-xs break-all">
                  {campaign.target_filter || "All devices"}
                </p>
              </div>

              <Separator />

              <div>
                <Label>Created</Label>
                <p className="text-sm mt-1">{new Date(campaign.created_at).toLocaleString()}</p>
              </div>

              {campaign.started_at && (
                <div>
                  <Label>Started</Label>
                  <p className="text-sm mt-1">{new Date(campaign.started_at).toLocaleString()}</p>
                </div>
              )}

              {campaign.completed_at && (
                <div>
                  <Label>Completed</Label>
                  <p className="text-sm mt-1">{new Date(campaign.completed_at).toLocaleString()}</p>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Release Notes</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm whitespace-pre-wrap">
                {campaign.notes || "No release notes available."}
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
export default FirmwareDetail;
