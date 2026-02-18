import { useEffect, useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Smartphone, Plus, Upload, Download } from "lucide-react";
import { api, type Device } from "../lib/api";

function Devices() {
  const navigate = useNavigate();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<{ success: number; errors: string[] } | null>(null);

  useEffect(() => {
    const fetchDevices = async () => {
      try {
        const data = await api.getDevices();
        setDevices(data);
      } catch (error) {
        console.error("Failed to fetch devices:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchDevices();
  }, []);

  const handleExport = async () => {
    try {
      const blob = await api.exportDevices();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.style.display = 'none';
      a.href = url;
      a.download = `devices-export-${new Date().toISOString().split('T')[0]}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Failed to export devices:", error);
    }
  };

  const handleImport = async () => {
    if (!importFile) return;

    try {
      const result = await api.importDevices(importFile);
      setImportResult(result);
      
      if (result.success > 0) {
        // Refresh the device list
        const data = await api.getDevices();
        setDevices(data);
      }
    } catch (error) {
      console.error("Failed to import devices:", error);
      setImportResult({ success: 0, errors: ["Import failed: " + (error as Error).message] });
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

  const resetImportDialog = () => {
    setImportDialogOpen(false);
    setImportFile(null);
    setImportResult(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading devices...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Device Registry</h1>
          <p className="text-muted-foreground">
            Manage device inventory and track firmware versions
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleExport}>
            <Download className="h-4 w-4 mr-2" />
            Export CSV
          </Button>
          <Dialog open={importDialogOpen} onOpenChange={resetImportDialog}>
            <DialogTrigger asChild>
              <Button variant="outline">
                <Upload className="h-4 w-4 mr-2" />
                Import CSV
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Import Devices from CSV</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                {!importResult ? (
                  <>
                    <div>
                      <Label htmlFor="csv-file">CSV File</Label>
                      <Input
                        id="csv-file"
                        type="file"
                        accept=".csv"
                        onChange={(e) => setImportFile(e.target.files?.[0] || null)}
                        ref={fileInputRef}
                      />
                      <p className="text-sm text-muted-foreground mt-2">
                        CSV should include columns: serial_number, ipn, firmware_version, customer, location, status
                      </p>
                    </div>
                    <div className="flex justify-end gap-2">
                      <Button variant="outline" onClick={resetImportDialog}>
                        Cancel
                      </Button>
                      <Button onClick={handleImport} disabled={!importFile}>
                        Import
                      </Button>
                    </div>
                  </>
                ) : (
                  <>
                    <div className="space-y-2">
                      <p className="text-sm">
                        <span className="font-medium text-green-600">
                          Successfully imported: {importResult.success} devices
                        </span>
                      </p>
                      {importResult.errors.length > 0 && (
                        <div>
                          <p className="text-sm font-medium text-red-600 mb-1">
                            Errors ({importResult.errors.length}):
                          </p>
                          <div className="max-h-32 overflow-y-auto bg-muted p-2 rounded text-xs">
                            {importResult.errors.map((error, index) => (
                              <div key={index}>{error}</div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                    <div className="flex justify-end">
                      <Button onClick={resetImportDialog}>Close</Button>
                    </div>
                  </>
                )}
              </div>
            </DialogContent>
          </Dialog>
          <Button onClick={() => navigate("/devices/new")}>
            <Plus className="h-4 w-4 mr-2" />
            Add Device
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Smartphone className="h-5 w-5" />
            Device Inventory
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Serial Number</TableHead>
                <TableHead>IPN</TableHead>
                <TableHead>Firmware Version</TableHead>
                <TableHead>Customer</TableHead>
                <TableHead>Location</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Last Seen</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {devices.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8">
                    <div className="text-muted-foreground">
                      No devices found. Add devices manually or import from CSV.
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                devices.map((device) => (
                  <TableRow key={device.serial_number} className="cursor-pointer hover:bg-muted/50" onClick={() => navigate(`/devices/${device.serial_number}`)}>
                    <TableCell className="font-mono font-medium">{device.serial_number}</TableCell>
                    <TableCell>{device.ipn}</TableCell>
                    <TableCell className="font-mono text-sm">{device.firmware_version || "—"}</TableCell>
                    <TableCell>{device.customer || "—"}</TableCell>
                    <TableCell>{device.location || "—"}</TableCell>
                    <TableCell>
                      <Badge variant={getStatusBadgeVariant(device.status)}>
                        {device.status.charAt(0).toUpperCase() + device.status.slice(1)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {device.last_seen ? new Date(device.last_seen).toLocaleDateString() : "—"}
                    </TableCell>
                    <TableCell>
                      <Button variant="outline" size="sm" onClick={(e) => { e.stopPropagation(); navigate(`/devices/${device.serial_number}`); }}>
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

      {/* Device Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <Smartphone className="h-8 w-8 text-blue-600" />
            </div>
            <div className="text-3xl font-bold text-blue-600 text-center">
              {devices.length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Total Devices
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <div className="h-8 w-8 rounded-full bg-green-500 flex items-center justify-center">
                <div className="w-4 h-4 rounded-full bg-white" />
              </div>
            </div>
            <div className="text-3xl font-bold text-green-600 text-center">
              {devices.filter(d => d.status === "active").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Active
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <div className="h-8 w-8 rounded-full bg-yellow-500 flex items-center justify-center">
                <div className="w-4 h-4 rounded-full bg-white" />
              </div>
            </div>
            <div className="text-3xl font-bold text-yellow-600 text-center">
              {devices.filter(d => d.status === "maintenance").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Maintenance
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <div className="h-8 w-8 rounded-full bg-gray-500 flex items-center justify-center">
                <div className="w-4 h-4 rounded-full bg-white" />
              </div>
            </div>
            <div className="text-3xl font-bold text-gray-600 text-center">
              {devices.filter(d => d.status === "inactive" || d.status === "retired").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Inactive
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
export default Devices;
