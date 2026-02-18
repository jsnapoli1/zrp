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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Smartphone, ArrowLeft, Save, History, RotateCcw } from "lucide-react";
import { api, type Device, type RMA } from "../lib/api";

function DeviceDetail() {
  const { serialNumber } = useParams<{ serialNumber: string }>();
  const navigate = useNavigate();
  const [device, setDevice] = useState<Device | null>(null);
  const [relatedRMAs, setRelatedRMAs] = useState<RMA[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [formData, setFormData] = useState<Partial<Device>>({});

  useEffect(() => {
    const fetchData = async () => {
      if (!serialNumber) return;
      
      try {
        const [deviceData, allRMAs] = await Promise.all([
          api.getDevice(serialNumber),
          api.getRMAs()
        ]);
        
        setDevice(deviceData);
        setFormData(deviceData);
        
        // Filter RMAs for this device
        const deviceRMAs = allRMAs.filter(rma => rma.serial_number === serialNumber);
        setRelatedRMAs(deviceRMAs);
        
      } catch (error) {
        console.error("Failed to fetch device data:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [serialNumber]);

  const handleSave = async () => {
    if (!serialNumber) return;
    
    try {
      const updatedDevice = await api.updateDevice(serialNumber, formData);
      setDevice(updatedDevice);
      setEditing(false);
    } catch (error) {
      console.error("Failed to update device:", error);
    }
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case "active":
        return "default";
      case "inactive":
        return "secondary";
      case "maintenance":
        return "outline";
      case "retired":
        return "destructive";
      default:
        return "secondary";
    }
  };

  const getRMAStatusBadgeVariant = (status: string) => {
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
          <p className="mt-2 text-muted-foreground">Loading device...</p>
        </div>
      </div>
    );
  }

  if (!device) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/devices")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Devices
          </Button>
        </div>
        <div className="text-center py-8">
          <h2 className="text-2xl font-semibold mb-2">Device Not Found</h2>
          <p className="text-muted-foreground">The requested device could not be found.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/devices")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Devices
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight font-mono">{device.serial_number}</h1>
            <p className="text-muted-foreground">{device.ipn} - {device.customer}</p>
          </div>
        </div>
        <div className="flex gap-2">
          {editing ? (
            <>
              <Button variant="outline" onClick={() => { setEditing(false); setFormData(device); }}>
                Cancel
              </Button>
              <Button onClick={handleSave}>
                <Save className="h-4 w-4 mr-2" />
                Save Changes
              </Button>
            </>
          ) : (
            <Button onClick={() => setEditing(true)}>Edit Device</Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Details */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Smartphone className="h-5 w-5" />
                Device Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Serial Number</Label>
                  <p className="text-sm mt-1 font-mono">{device.serial_number}</p>
                </div>
                
                <div>
                  <Label>IPN</Label>
                  {editing ? (
                    <Input
                      value={formData.ipn || ""}
                      onChange={(e) => setFormData({ ...formData, ipn: e.target.value })}
                      placeholder="Internal part number"
                    />
                  ) : (
                    <p className="text-sm mt-1">{device.ipn}</p>
                  )}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Firmware Version</Label>
                  {editing ? (
                    <Input
                      value={formData.firmware_version || ""}
                      onChange={(e) => setFormData({ ...formData, firmware_version: e.target.value })}
                      placeholder="e.g., v1.2.3"
                      className="font-mono"
                    />
                  ) : (
                    <p className="text-sm mt-1 font-mono">{device.firmware_version || "Not specified"}</p>
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
                        <SelectItem value="active">Active</SelectItem>
                        <SelectItem value="inactive">Inactive</SelectItem>
                        <SelectItem value="maintenance">Maintenance</SelectItem>
                        <SelectItem value="retired">Retired</SelectItem>
                      </SelectContent>
                    </Select>
                  ) : (
                    <div className="mt-1">
                      <Badge variant={getStatusBadgeVariant(device.status)}>
                        {device.status.charAt(0).toUpperCase() + device.status.slice(1)}
                      </Badge>
                    </div>
                  )}
                </div>
              </div>

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
                    <p className="text-sm mt-1">{device.customer || "Not assigned"}</p>
                  )}
                </div>
                
                <div>
                  <Label>Location</Label>
                  {editing ? (
                    <Input
                      value={formData.location || ""}
                      onChange={(e) => setFormData({ ...formData, location: e.target.value })}
                      placeholder="Physical location"
                    />
                  ) : (
                    <p className="text-sm mt-1">{device.location || "Not specified"}</p>
                  )}
                </div>
              </div>

              <div>
                <Label>Install Date</Label>
                {editing ? (
                  <Input
                    type="date"
                    value={formData.install_date?.split('T')[0] || ""}
                    onChange={(e) => setFormData({ ...formData, install_date: e.target.value })}
                  />
                ) : (
                  <p className="text-sm mt-1">
                    {device.install_date ? new Date(device.install_date).toLocaleDateString() : "Not specified"}
                  </p>
                )}
              </div>

              <div>
                <Label>Notes</Label>
                {editing ? (
                  <Textarea
                    value={formData.notes || ""}
                    onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                    placeholder="Additional notes about this device"
                    rows={3}
                  />
                ) : (
                  <p className="text-sm mt-1 whitespace-pre-wrap">{device.notes || "No notes"}</p>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Related RMAs */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <RotateCcw className="h-5 w-5" />
                  Related RMAs
                </div>
                <Button 
                  size="sm" 
                  onClick={() => navigate(`/rmas?device=${device.serial_number}`)}
                >
                  Create RMA
                </Button>
              </CardTitle>
            </CardHeader>
            <CardContent>
              {relatedRMAs.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  No RMAs found for this device.
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>RMA ID</TableHead>
                      <TableHead>Reason</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead>Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {relatedRMAs.map((rma) => (
                      <TableRow key={rma.id}>
                        <TableCell className="font-medium">{rma.id}</TableCell>
                        <TableCell>{rma.reason}</TableCell>
                        <TableCell>
                          <Badge variant={getRMAStatusBadgeVariant(rma.status)}>
                            {rma.status.charAt(0).toUpperCase() + rma.status.slice(1)}
                          </Badge>
                        </TableCell>
                        <TableCell>{new Date(rma.created_at).toLocaleDateString()}</TableCell>
                        <TableCell>
                          <Button variant="outline" size="sm" onClick={() => navigate(`/rmas/${rma.id}`)}>
                            View
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <History className="h-5 w-5" />
                Device History
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <Label>Created</Label>
                <p className="text-sm mt-1">{new Date(device.created_at).toLocaleString()}</p>
              </div>
              
              {device.last_seen && (
                <div>
                  <Label>Last Seen</Label>
                  <p className="text-sm mt-1">{new Date(device.last_seen).toLocaleString()}</p>
                </div>
              )}

              <Separator />
              
              <div>
                <Label>Total RMAs</Label>
                <p className="text-sm mt-1">{relatedRMAs.length}</p>
              </div>

              <div>
                <Label>Active RMAs</Label>
                <p className="text-sm mt-1">
                  {relatedRMAs.filter(rma => !['closed', 'shipped'].includes(rma.status)).length}
                </p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Firmware Management</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <Label>Current Version</Label>
                <p className="text-sm mt-1 font-mono">{device.firmware_version || "Unknown"}</p>
              </div>
              
              <Separator />
              
              <Button 
                variant="outline" 
                className="w-full"
                onClick={() => navigate(`/firmware?device=${device.serial_number}`)}
              >
                View Firmware Campaigns
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Quick Actions</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <Button 
                variant="outline" 
                className="w-full justify-start"
                onClick={() => navigate(`/rmas?device=${device.serial_number}`)}
              >
                <RotateCcw className="h-4 w-4 mr-2" />
                Create RMA
              </Button>
              
              <Button 
                variant="outline" 
                className="w-full justify-start"
                onClick={() => navigate(`/testing?device=${device.serial_number}`)}
              >
                <History className="h-4 w-4 mr-2" />
                View Test History
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
export default DeviceDetail;
