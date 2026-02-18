import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Label } from "../components/ui/label";
import { Separator } from "../components/ui/separator";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { FileText, ArrowLeft, Save, Download, Plus, Trash2 } from "lucide-react";
import { api, type Quote, type QuoteLine, type Part } from "../lib/api";

function QuoteDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [quote, setQuote] = useState<Quote | null>(null);
  const [parts, setParts] = useState<Part[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [formData, setFormData] = useState<Partial<Quote>>({});

  useEffect(() => {
    const fetchData = async () => {
      if (!id) return;
      
      try {
        const [quoteData, partsResponse] = await Promise.all([
          api.getQuote(id),
          api.getParts()
        ]);
        
        setQuote(quoteData);
        setFormData(quoteData);
        setParts(partsResponse.data || []);
      } catch (error) {
        console.error("Failed to fetch quote data:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [id]);

  const handleSave = async () => {
    if (!id) return;
    
    try {
      const updatedQuote = await api.updateQuote(id, formData);
      setQuote(updatedQuote);
      setEditing(false);
    } catch (error) {
      console.error("Failed to update quote:", error);
    }
  };

  const handleExportPDF = async () => {
    if (!id) return;
    
    try {
      const blob = await api.exportQuotePDF(id);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.style.display = 'none';
      a.href = url;
      a.download = `quote-${id}.pdf`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Failed to export PDF:", error);
    }
  };

  const addLineItem = () => {
    const newLines = [...(formData.lines || []), {
      id: 0, // Temporary ID for new items
      quote_id: "", // Will be set on save
      ipn: "",
      description: "",
      qty: 1,
      unit_price: 0,
      notes: ""
    }];
    setFormData({ ...formData, lines: newLines });
  };

  const removeLineItem = (index: number) => {
    const newLines = (formData.lines || []).filter((_, i) => i !== index);
    setFormData({ ...formData, lines: newLines });
  };

  const updateLineItem = (index: number, field: keyof QuoteLine, value: any) => {
    const newLines = [...(formData.lines || [])];
    newLines[index] = { ...newLines[index], [field]: value };
    setFormData({ ...formData, lines: newLines });
  };

  const getPartCost = (ipn: string) => {
    const part = parts.find(p => p.ipn === ipn);
    return part?.cost || 0;
  };

  const calculateTotals = () => {
    const lines = quote?.lines || [];
    const totalQuoted = lines.reduce((sum, line) => sum + (line.qty * line.unit_price), 0);
    const totalCost = lines.reduce((sum, line) => sum + (line.qty * getPartCost(line.ipn)), 0);
    const margin = totalQuoted - totalCost;
    const marginPercent = totalQuoted > 0 ? (margin / totalQuoted) * 100 : 0;
    
    return { totalQuoted, totalCost, margin, marginPercent };
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case "accepted":
        return "default";
      case "sent":
        return "secondary";
      case "draft":
        return "outline";
      case "expired":
      case "rejected":
        return "destructive";
      default:
        return "outline";
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading quote...</p>
        </div>
      </div>
    );
  }

  if (!quote) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/quotes")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Quotes
          </Button>
        </div>
        <div className="text-center py-8">
          <h2 className="text-2xl font-semibold mb-2">Quote Not Found</h2>
          <p className="text-muted-foreground">The requested quote could not be found.</p>
        </div>
      </div>
    );
  }

  const totals = calculateTotals();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate("/quotes")}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Quotes
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{quote.id}</h1>
            <p className="text-muted-foreground">{quote.customer}</p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleExportPDF}>
            <Download className="h-4 w-4 mr-2" />
            Export PDF
          </Button>
          {editing ? (
            <>
              <Button variant="outline" onClick={() => { setEditing(false); setFormData(quote); }}>
                Cancel
              </Button>
              <Button onClick={handleSave}>
                <Save className="h-4 w-4 mr-2" />
                Save Changes
              </Button>
            </>
          ) : (
            <Button onClick={() => setEditing(true)}>Edit Quote</Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-3 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FileText className="h-5 w-5" />
                Quote Details
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
                    <p className="text-sm mt-1">{quote.customer}</p>
                  )}
                </div>
                
                <div>
                  <Label>Status</Label>
                  {editing ? (
                    <select 
                      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background"
                      value={formData.status || ""}
                      onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                    >
                      <option value="draft">Draft</option>
                      <option value="sent">Sent</option>
                      <option value="accepted">Accepted</option>
                      <option value="rejected">Rejected</option>
                      <option value="expired">Expired</option>
                    </select>
                  ) : (
                    <div className="mt-1">
                      <Badge variant={getStatusBadgeVariant(quote.status)}>
                        {quote.status.charAt(0).toUpperCase() + quote.status.slice(1)}
                      </Badge>
                    </div>
                  )}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>Valid Until</Label>
                  {editing ? (
                    <Input
                      type="date"
                      value={formData.valid_until?.split('T')[0] || ""}
                      onChange={(e) => setFormData({ ...formData, valid_until: e.target.value })}
                    />
                  ) : (
                    <p className="text-sm mt-1">
                      {quote.valid_until ? new Date(quote.valid_until).toLocaleDateString() : "Not specified"}
                    </p>
                  )}
                </div>
                
                <div>
                  <Label>Accepted At</Label>
                  <p className="text-sm mt-1">
                    {quote.accepted_at ? new Date(quote.accepted_at).toLocaleDateString() : "—"}
                  </p>
                </div>
              </div>

              <div>
                <Label>Notes</Label>
                {editing ? (
                  <Textarea
                    value={formData.notes || ""}
                    onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                    placeholder="Quote notes and terms"
                    rows={3}
                  />
                ) : (
                  <p className="text-sm mt-1 whitespace-pre-wrap">{quote.notes || "No notes"}</p>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Line Items */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                Line Items
                {editing && (
                  <Button onClick={addLineItem} size="sm">
                    <Plus className="h-4 w-4 mr-2" />
                    Add Item
                  </Button>
                )}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>IPN</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead>Qty</TableHead>
                    <TableHead>Unit Cost</TableHead>
                    <TableHead>Unit Price</TableHead>
                    <TableHead>Line Total</TableHead>
                    <TableHead>Margin</TableHead>
                    {editing && <TableHead></TableHead>}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(editing ? formData.lines : quote.lines)?.map((line, index) => {
                    const lineCost = getPartCost(line.ipn) * line.qty;
                    const lineTotal = line.qty * line.unit_price;
                    const lineMargin = lineTotal - lineCost;
                    const lineMarginPercent = lineTotal > 0 ? (lineMargin / lineTotal) * 100 : 0;
                    
                    return (
                      <TableRow key={editing ? index : line.id}>
                        <TableCell>
                          {editing ? (
                            <Input
                              value={line.ipn}
                              onChange={(e) => updateLineItem(index, "ipn", e.target.value)}
                              className="w-28"
                            />
                          ) : (
                            line.ipn
                          )}
                        </TableCell>
                        <TableCell>
                          {editing ? (
                            <Input
                              value={line.description}
                              onChange={(e) => updateLineItem(index, "description", e.target.value)}
                              className="min-w-[200px]"
                            />
                          ) : (
                            line.description
                          )}
                        </TableCell>
                        <TableCell>
                          {editing ? (
                            <Input
                              type="number"
                              value={line.qty}
                              onChange={(e) => updateLineItem(index, "qty", parseInt(e.target.value) || 1)}
                              className="w-16"
                              min="1"
                            />
                          ) : (
                            line.qty
                          )}
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          ${getPartCost(line.ipn).toFixed(2)}
                        </TableCell>
                        <TableCell>
                          {editing ? (
                            <Input
                              type="number"
                              step="0.01"
                              value={line.unit_price}
                              onChange={(e) => updateLineItem(index, "unit_price", parseFloat(e.target.value) || 0)}
                              className="w-24"
                              min="0"
                            />
                          ) : (
                            `$${line.unit_price.toFixed(2)}`
                          )}
                        </TableCell>
                        <TableCell className="font-medium">
                          ${lineTotal.toFixed(2)}
                        </TableCell>
                        <TableCell>
                          <div className={`text-sm ${lineMargin >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                            ${lineMargin.toFixed(2)}
                            <div className="text-xs">
                              ({lineMarginPercent.toFixed(1)}%)
                            </div>
                          </div>
                        </TableCell>
                        {editing && (
                          <TableCell>
                            {(formData.lines?.length || 0) > 1 && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => removeLineItem(index)}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            )}
                          </TableCell>
                        )}
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Quote Summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <Label>Total Cost</Label>
                <p className="text-sm mt-1 text-muted-foreground">${totals.totalCost.toFixed(2)}</p>
              </div>
              
              <div>
                <Label>Total Quoted</Label>
                <p className="text-lg font-semibold mt-1">${totals.totalQuoted.toFixed(2)}</p>
              </div>
              
              <Separator />
              
              <div>
                <Label>Margin</Label>
                <div className="mt-1">
                  <p className={`text-lg font-semibold ${totals.margin >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                    ${totals.margin.toFixed(2)}
                  </p>
                  <p className={`text-sm ${totals.marginPercent >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                    ({totals.marginPercent.toFixed(1)}% margin)
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Timeline</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <Label>Created</Label>
                <p className="text-sm mt-1">{new Date(quote.created_at).toLocaleString()}</p>
              </div>
              
              {quote.valid_until && (
                <div>
                  <Label>Valid Until</Label>
                  <p className="text-sm mt-1">{new Date(quote.valid_until).toLocaleString()}</p>
                </div>
              )}

              {quote.accepted_at && (
                <div>
                  <Label>Accepted</Label>
                  <p className="text-sm mt-1">{new Date(quote.accepted_at).toLocaleString()}</p>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Actions</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <Button variant="outline" className="w-full" onClick={handleExportPDF}>
                <Download className="h-4 w-4 mr-2" />
                Download PDF
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
export default QuoteDetail;
