import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Label } from "../components/ui/label";
import { Separator } from "../components/ui/separator";
import { RotateCcw, ArrowLeft, Save, Truck, CheckCircle } from "lucide-react";
import { api, type RMA } from "../lib/api";

function RMADetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [rma, setRMA] = useState<RMA | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [formData, setFormData] = useState<Partial<RMA>>({});

  useEffect(() => {
    const fetchRMA = async () => {
      if (!id) return;
      
      try {
        const data = await api.getRMA(id);
        setRMA(data);
        setFormData(data);
      } catch (error) {
        console.error("Failed to fetch RMA:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchRMA();
  }, [id]);

  const handleSave = async () => {
    if (!id) return;
    
    try {
      const updatedRMA = await api.updateRMA(id, formData);
      setRMA(updatedRMA);
      setEditing(false);
    } catch (error) {
      console.error("Failed to update RMA:", error);
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

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "shipped":
        return Truck;
      case "closed":
        return CheckCircle;
      case "received":
        return RotateCcw;
      default:
        return RotateCcw;
    }
  };

  const statusWorkflow = [
    { value: "open", label: "Open", description: "RMA request created" },
    { value: "received", label: "Received", description: "Device received for inspection" },
    { value: "investigating", label: "Investigating", description: "Analyzing the defect" },
    { value: "resolved", label: "Resolved", description: "Issue resolved, ready to ship" },
    { value: "shipped", label: "Shipped", description: "Replacement/repaired device shipped" },
    { value: "closed", label: "Closed", description: "RMA completed" },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading RMA...</p>
        </div>
      </div>
    );
  }

  if (!rma) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/rmas")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to RMAs
          </Button>
        </div>
        <div className="text-center py-8">
          <h2 className="text-2xl font-semibold mb-2">RMA Not Found</h2>
          <p className="text-muted-foreground">The requested RMA could not be found.</p>
        </div>
      </div>
    );
  }

  const StatusIcon = getStatusIcon(rma.status);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/rmas")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to RMAs
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{rma.id}</h1>
            <p className="text-muted-foreground">{rma.customer} - {rma.serial_number}</p>
          </div>
        </div>
        <div className="flex gap-2">
          {editing ? (
            <>
              <Button variant="outline" onClick={() => { setEditing(false); setFormData(rma); }}>
                Cancel
              </Button>
              <Button onClick={handleSave}>
                <Save className="h-4 w-4 mr-2" />
                Save Changes
              </Button>
            </>
          ) : (
            <Button onClick={() => setEditing(true)}>Edit RMA</Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Details */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <StatusIcon className="h-5 w-5" />
                RMA Details
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Customer</Label>
                  {editing ? (
                    <Input
                      value={formData.customer || ""}
                      onChange={(e) => setFormData({ ...formData, customer: e.target.value })}
                      placeholder="Customer name"
                    />
                  ) : (
                    <p className="text-sm mt-1">{rma.customer}</p>
                  )}
                </div>
                
                <div>
                  <Label>Device Serial Number</Label>
                  {editing ? (
                    <Input
                      value={formData.serial_number || ""}
                      onChange={(e) => setFormData({ ...formData, serial_number: e.target.value })}
                      placeholder="Serial number"
                      className="font-mono"
                    />
                  ) : (
                    <p className="text-sm mt-1 font-mono">{rma.serial_number}</p>
                  )}
                </div>
              </div>

              <div>
                <Label>Reason for Return</Label>
                {editing ? (
                  <Input
                    value={formData.reason || ""}
                    onChange={(e) => setFormData({ ...formData, reason: e.target.value })}
                    placeholder="Reason for return"
                  />
                ) : (
                  <p className="text-sm mt-1">{rma.reason}</p>
                )}
              </div>
              
              <div>
                <Label>Defect Description</Label>
                {editing ? (
                  <Textarea
                    value={formData.defect_description || ""}
                    onChange={(e) => setFormData({ ...formData, defect_description: e.target.value })}
                    placeholder="Detailed description of the defect"
                    rows={3}
                  />
                ) : (
                  <p className="text-sm mt-1 whitespace-pre-wrap">{rma.defect_description || "No description provided"}</p>
                )}
              </div>

              <div>
                <Label>Status</Label>
                {editing ? (
                  <Select value={formData.status || ""} onValueChange={(value) => setFormData({ ...formData, status: value })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {statusWorkflow.map((status) => (
                        <SelectItem key={status.value} value={status.value}>
                          {status.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <div className="mt-1">
                    <Badge variant={getStatusBadgeVariant(rma.status)} className="mb-2">
                      {rma.status.charAt(0).toUpperCase() + rma.status.slice(1)}
                    </Badge>
                  </div>
                )}
              </div>

              <Separator />

              <div>
                <Label>Resolution</Label>
                {editing ? (
                  <Textarea
                    value={formData.resolution || ""}
                    onChange={(e) => setFormData({ ...formData, resolution: e.target.value })}
                    placeholder="Resolution taken for this RMA"
                    rows={3}
                  />
                ) : (
                  <p className="text-sm mt-1 whitespace-pre-wrap">{rma.resolution || "Resolution pending"}</p>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Status Workflow */}
          <Card>
            <CardHeader>
              <CardTitle>Status Workflow</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {statusWorkflow.map((status, index) => {
                  const isActive = status.value === rma.status;
                  const isCompleted = statusWorkflow.findIndex(s => s.value === rma.status) > index;
                  
                  return (
                    <div key={status.value} className={`flex items-center gap-3 p-2 rounded-lg ${isActive ? 'bg-primary/10 border border-primary/20' : ''}`}>
                      <div className={`w-3 h-3 rounded-full ${isActive ? 'bg-primary' : isCompleted ? 'bg-green-500' : 'bg-muted'}`} />
                      <div>
                        <p className={`text-sm font-medium ${isActive ? 'text-primary' : ''}`}>
                          {status.label}
                        </p>
                        <p className="text-xs text-muted-foreground">{status.description}</p>
                      </div>
                    </div>
                  );
                })}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Timeline</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <Label>Created</Label>
                <p className="text-sm mt-1">{new Date(rma.created_at).toLocaleString()}</p>
              </div>
              
              {rma.received_at && (
                <div>
                  <Label>Received</Label>
                  <p className="text-sm mt-1">{new Date(rma.received_at).toLocaleString()}</p>
                </div>
              )}

              {rma.resolved_at && (
                <div>
                  <Label>Resolved</Label>
                  <p className="text-sm mt-1">{new Date(rma.resolved_at).toLocaleString()}</p>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Device Information</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <p className="text-sm">
                  <span className="font-medium">Serial Number:</span>
                  <br />
                  <span className="font-mono text-xs">{rma.serial_number}</span>
                </p>
                <p className="text-sm text-muted-foreground">
                  Click to view full device details
                </p>
                <Button 
                  variant="outline" 
                  size="sm" 
                  className="w-full"
                  onClick={() => navigate(`/devices/${rma.serial_number}`)}
                >
                  View Device Details
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
export default RMADetail;
