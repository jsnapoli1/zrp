import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Label } from "../components/ui/label";
import { Checkbox } from "../components/ui/checkbox";
import { AlertTriangle, ArrowLeft, FileText, Save } from "lucide-react";
import { api, type NCR } from "../lib/api";

function NCRDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [ncr, setNCR] = useState<NCR | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [formData, setFormData] = useState<Partial<NCR> & { create_eco?: boolean }>({});

  useEffect(() => {
    const fetchNCR = async () => {
      if (!id) return;
      
      try {
        const data = await api.getNCR(id);
        setNCR(data);
        setFormData(data);
      } catch (error) {
        console.error("Failed to fetch NCR:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchNCR();
  }, [id]);

  const handleSave = async () => {
    if (!id) return;
    
    try {
      const updatedNCR = await api.updateNCR(id, formData);
      setNCR(updatedNCR);
      setEditing(false);
    } catch (error) {
      console.error("Failed to update NCR:", error);
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
          <p className="mt-2 text-muted-foreground">Loading NCR...</p>
        </div>
      </div>
    );
  }

  if (!ncr) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/ncrs")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to NCRs
          </Button>
        </div>
        <div className="text-center py-8">
          <h2 className="text-2xl font-semibold mb-2">NCR Not Found</h2>
          <p className="text-muted-foreground">The requested NCR could not be found.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/ncrs")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to NCRs
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{ncr.id}</h1>
            <p className="text-muted-foreground">{ncr.title}</p>
          </div>
        </div>
        <div className="flex gap-2">
          {editing ? (
            <>
              <Button variant="outline" onClick={() => { setEditing(false); setFormData(ncr); }}>
                Cancel
              </Button>
              <Button onClick={handleSave}>
                <Save className="h-4 w-4 mr-2" />
                Save Changes
              </Button>
            </>
          ) : (
            <Button onClick={() => setEditing(true)}>Edit NCR</Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Details */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5" />
                NCR Details
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>Title</Label>
                {editing ? (
                  <Input
                    value={formData.title || ""}
                    onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                    placeholder="NCR title"
                  />
                ) : (
                  <p className="text-sm mt-1">{ncr.title}</p>
                )}
              </div>
              
              <div>
                <Label>Description</Label>
                {editing ? (
                  <Textarea
                    value={formData.description || ""}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    placeholder="Detailed description"
                    rows={3}
                  />
                ) : (
                  <p className="text-sm mt-1 whitespace-pre-wrap">{ncr.description || "No description provided"}</p>
                )}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Severity</Label>
                  {editing ? (
                    <Select value={formData.severity || ""} onValueChange={(value) => setFormData({ ...formData, severity: value })}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="minor">Minor</SelectItem>
                        <SelectItem value="major">Major</SelectItem>
                        <SelectItem value="critical">Critical</SelectItem>
                      </SelectContent>
                    </Select>
                  ) : (
                    <div className="mt-1">
                      <Badge variant={getSeverityBadgeVariant(ncr.severity)} className="flex items-center gap-1 w-fit">
                        <div className={`w-2 h-2 rounded-full ${getSeverityColor(ncr.severity)}`} />
                        {ncr.severity.charAt(0).toUpperCase() + ncr.severity.slice(1)}
                      </Badge>
                    </div>
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
                        <SelectItem value="open">Open</SelectItem>
                        <SelectItem value="investigating">Investigating</SelectItem>
                        <SelectItem value="resolved">Resolved</SelectItem>
                        <SelectItem value="closed">Closed</SelectItem>
                      </SelectContent>
                    </Select>
                  ) : (
                    <div className="mt-1">
                      <Badge variant={ncr.status === "closed" || ncr.status === "resolved" ? "default" : "secondary"}>
                        {ncr.status.charAt(0).toUpperCase() + ncr.status.slice(1)}
                      </Badge>
                    </div>
                  )}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Affected IPN</Label>
                  {editing ? (
                    <Input
                      value={formData.ipn || ""}
                      onChange={(e) => setFormData({ ...formData, ipn: e.target.value })}
                      placeholder="Part number"
                    />
                  ) : (
                    <p className="text-sm mt-1">{ncr.ipn || "Not specified"}</p>
                  )}
                </div>
                
                <div>
                  <Label>Serial Number</Label>
                  {editing ? (
                    <Input
                      value={formData.serial_number || ""}
                      onChange={(e) => setFormData({ ...formData, serial_number: e.target.value })}
                      placeholder="Device serial number"
                    />
                  ) : (
                    <p className="text-sm mt-1">{ncr.serial_number || "Not specified"}</p>
                  )}
                </div>
              </div>

              <div>
                <Label>Defect Type</Label>
                {editing ? (
                  <Input
                    value={formData.defect_type || ""}
                    onChange={(e) => setFormData({ ...formData, defect_type: e.target.value })}
                    placeholder="Type of defect"
                  />
                ) : (
                  <p className="text-sm mt-1">{ncr.defect_type || "Not specified"}</p>
                )}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Root Cause Analysis</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>Root Cause</Label>
                {editing ? (
                  <Textarea
                    value={formData.root_cause || ""}
                    onChange={(e) => setFormData({ ...formData, root_cause: e.target.value })}
                    placeholder="Identified root cause of the issue"
                    rows={3}
                  />
                ) : (
                  <p className="text-sm mt-1 whitespace-pre-wrap">{ncr.root_cause || "Root cause analysis pending"}</p>
                )}
              </div>
              
              <div>
                <Label>Corrective Action</Label>
                {editing ? (
                  <Textarea
                    value={formData.corrective_action || ""}
                    onChange={(e) => setFormData({ ...formData, corrective_action: e.target.value })}
                    placeholder="Actions taken to correct the issue"
                    rows={3}
                  />
                ) : (
                  <p className="text-sm mt-1 whitespace-pre-wrap">{ncr.corrective_action || "Corrective action pending"}</p>
                )}
              </div>

              {editing && formData.corrective_action && (formData.status === "resolved" || formData.status === "closed") && (
                <div className="flex items-center space-x-2 p-4 bg-muted/50 rounded-lg">
                  <Checkbox 
                    id="create_eco" 
                    checked={formData.create_eco || false}
                    onCheckedChange={(checked) => setFormData({ ...formData, create_eco: checked as boolean })}
                  />
                  <Label htmlFor="create_eco" className="text-sm">
                    Create ECO from corrective action
                  </Label>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <Label>Created</Label>
                <p className="text-sm mt-1">{new Date(ncr.created_at).toLocaleString()}</p>
              </div>
              
              {ncr.resolved_at && (
                <div>
                  <Label>Resolved</Label>
                  <p className="text-sm mt-1">{new Date(ncr.resolved_at).toLocaleString()}</p>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Actions</CardTitle>
            </CardHeader>
            <CardContent>
              <Button 
                variant="outline" 
                className="w-full justify-start"
                onClick={() => navigate("/ecos?from_ncr=" + ncr.id)}
              >
                <FileText className="h-4 w-4 mr-2" />
                Create ECO from NCR
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
export default NCRDetail;
