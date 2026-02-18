import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Label } from "../components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { TestTube, Plus, CheckCircle, XCircle } from "lucide-react";
import { api, type TestRecord } from "../lib/api";

function Testing() {
  const [testRecords, setTestRecords] = useState<TestRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formData, setFormData] = useState({
    serial_number: "",
    ipn: "",
    firmware_version: "",
    test_type: "",
    result: "pass",
    measurements: "",
    notes: "",
    tested_by: "",
  });

  useEffect(() => {
    const fetchTestRecords = async () => {
      try {
        const data = await api.getTestRecords();
        setTestRecords(data);
      } catch (error) {
        console.error("Failed to fetch test records:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchTestRecords();
  }, []);

  const handleCreateTestRecord = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const newTestRecord = await api.createTestRecord({
        ...formData,
        tested_at: new Date().toISOString(),
      });
      setTestRecords([newTestRecord, ...testRecords]);
      setCreateDialogOpen(false);
      setFormData({
        serial_number: "",
        ipn: "",
        firmware_version: "",
        test_type: "",
        result: "pass",
        measurements: "",
        notes: "",
        tested_by: "",
      });
    } catch (error) {
      console.error("Failed to create test record:", error);
    }
  };

  const getResultBadge = (result: string) => {
    if (result === "pass") {
      return (
        <Badge variant="default" className="bg-green-500 hover:bg-green-600">
          <CheckCircle className="h-3 w-3 mr-1" />
          Pass
        </Badge>
      );
    } else {
      return (
        <Badge variant="destructive">
          <XCircle className="h-3 w-3 mr-1" />
          Fail
        </Badge>
      );
    }
  };

  const testTypes = [
    "Functional Test",
    "Performance Test",
    "Environmental Test",
    "Safety Test",
    "Compliance Test",
    "Stress Test",
    "Burn-in Test",
    "Final Test",
    "Incoming Inspection",
    "Quality Audit",
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading test records...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Testing</h1>
          <p className="text-muted-foreground">
            Track device testing results and quality metrics
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Test Record
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Create Test Record</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleCreateTestRecord} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="serial_number">Device Serial Number *</Label>
                  <Input
                    id="serial_number"
                    value={formData.serial_number}
                    onChange={(e) => setFormData({ ...formData, serial_number: e.target.value })}
                    placeholder="Serial number"
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="ipn">IPN *</Label>
                  <Input
                    id="ipn"
                    value={formData.ipn}
                    onChange={(e) => setFormData({ ...formData, ipn: e.target.value })}
                    placeholder="Internal part number"
                    required
                  />
                </div>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="firmware_version">Firmware Version</Label>
                  <Input
                    id="firmware_version"
                    value={formData.firmware_version}
                    onChange={(e) => setFormData({ ...formData, firmware_version: e.target.value })}
                    placeholder="e.g., v1.2.3"
                  />
                </div>
                <div>
                  <Label htmlFor="test_type">Test Type *</Label>
                  <Select value={formData.test_type} onValueChange={(value) => setFormData({ ...formData, test_type: value })}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select test type" />
                    </SelectTrigger>
                    <SelectContent>
                      {testTypes.map((type) => (
                        <SelectItem key={type} value={type}>
                          {type}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="result">Result *</Label>
                  <Select value={formData.result} onValueChange={(value) => setFormData({ ...formData, result: value })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="pass">Pass</SelectItem>
                      <SelectItem value="fail">Fail</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor="tested_by">Tested By</Label>
                  <Input
                    id="tested_by"
                    value={formData.tested_by}
                    onChange={(e) => setFormData({ ...formData, tested_by: e.target.value })}
                    placeholder="Technician name"
                  />
                </div>
              </div>

              <div>
                <Label htmlFor="measurements">Measurements</Label>
                <Textarea
                  id="measurements"
                  value={formData.measurements}
                  onChange={(e) => setFormData({ ...formData, measurements: e.target.value })}
                  placeholder="Test measurements and data"
                  rows={3}
                />
              </div>

              <div>
                <Label htmlFor="notes">Notes</Label>
                <Textarea
                  id="notes"
                  value={formData.notes}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  placeholder="Additional notes or observations"
                  rows={2}
                />
              </div>

              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit">Create Test Record</Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TestTube className="h-5 w-5" />
            Test Records
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Test ID</TableHead>
                <TableHead>Device S/N</TableHead>
                <TableHead>IPN</TableHead>
                <TableHead>Test Type</TableHead>
                <TableHead>Result</TableHead>
                <TableHead>Tested By</TableHead>
                <TableHead>Date</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {testRecords.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8">
                    <div className="text-muted-foreground">
                      No test records found. Create your first test record to get started.
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                testRecords.map((record) => (
                  <TableRow key={record.id} className="hover:bg-muted/50">
                    <TableCell className="font-medium">TEST-{record.id.toString().padStart(4, '0')}</TableCell>
                    <TableCell className="font-mono text-sm">{record.serial_number}</TableCell>
                    <TableCell>{record.ipn}</TableCell>
                    <TableCell>{record.test_type}</TableCell>
                    <TableCell>{getResultBadge(record.result)}</TableCell>
                    <TableCell>{record.tested_by || "—"}</TableCell>
                    <TableCell>{new Date(record.tested_at).toLocaleDateString()}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Test Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <TestTube className="h-8 w-8 text-blue-600" />
            </div>
            <div className="text-3xl font-bold text-blue-600 text-center">
              {testRecords.length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Total Tests
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <CheckCircle className="h-8 w-8 text-green-600" />
            </div>
            <div className="text-3xl font-bold text-green-600 text-center">
              {testRecords.filter(r => r.result === "pass").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Passed
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <XCircle className="h-8 w-8 text-red-600" />
            </div>
            <div className="text-3xl font-bold text-red-600 text-center">
              {testRecords.filter(r => r.result === "fail").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Failed
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <div className="h-8 w-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground font-bold">
                %
              </div>
            </div>
            <div className="text-3xl font-bold text-primary text-center">
              {testRecords.length > 0 
                ? Math.round((testRecords.filter(r => r.result === "pass").length / testRecords.length) * 100)
                : 0}%
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Pass Rate
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
export default Testing;
