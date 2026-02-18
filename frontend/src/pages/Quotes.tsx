import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Textarea } from "../components/ui/textarea";
import { Label } from "../components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { FileText, Plus, Trash2 } from "lucide-react";
import { api, type Quote, type QuoteLine } from "../lib/api";

function Quotes() {
  const navigate = useNavigate();
  const [quotes, setQuotes] = useState<Quote[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formData, setFormData] = useState({
    customer: "",
    notes: "",
    valid_until: "",
    lines: [] as Array<Omit<QuoteLine, 'id' | 'quote_id'>>,
  });

  useEffect(() => {
    const fetchQuotes = async () => {
      try {
        const data = await api.getQuotes();
        setQuotes(data);
      } catch (error) {
        console.error("Failed to fetch quotes:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchQuotes();
  }, []);

  const handleCreateQuote = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const newQuote = await api.createQuote({
        customer: formData.customer,
        notes: formData.notes,
        valid_until: formData.valid_until,
        lines: formData.lines.filter(line => line.ipn && line.qty && line.unit_price) as QuoteLine[],
      });
      setQuotes([newQuote, ...quotes]);
      setCreateDialogOpen(false);
      resetForm();
    } catch (error) {
      console.error("Failed to create quote:", error);
    }
  };

  const resetForm = () => {
    setFormData({
      customer: "",
      notes: "",
      valid_until: "",
      lines: [{ ipn: "", description: "", qty: 1, unit_price: 0, notes: "" }],
    });
  };

  const addLineItem = () => {
    setFormData({
      ...formData,
      lines: [...formData.lines, { ipn: "", description: "", qty: 1, unit_price: 0, notes: "" }]
    });
  };

  const removeLineItem = (index: number) => {
    setFormData({
      ...formData,
      lines: formData.lines.filter((_, i) => i !== index)
    });
  };

  const updateLineItem = (index: number, field: keyof Omit<QuoteLine, 'id' | 'quote_id'>, value: any) => {
    const updatedLines = [...formData.lines];
    updatedLines[index] = { ...updatedLines[index], [field]: value };
    setFormData({ ...formData, lines: updatedLines });
  };

  const calculateQuoteTotal = (quote: Quote) => {
    if (!quote.lines) return 0;
    return quote.lines.reduce((sum, line) => sum + (line.qty * line.unit_price), 0);
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

  // Initialize form with one empty line item
  useEffect(() => {
    if (formData.lines.length === 0) {
      setFormData({
        ...formData,
        lines: [{ ipn: "", description: "", qty: 1, unit_price: 0, notes: "" }]
      });
    }
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading quotes...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Quotes</h1>
          <p className="text-muted-foreground">
            Manage customer quotes and proposals
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Quote
            </Button>
          </DialogTrigger>
          <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Create New Quote</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleCreateQuote} className="space-y-6">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="customer">Customer *</Label>
                  <Input
                    id="customer"
                    value={formData.customer}
                    onChange={(e) => setFormData({ ...formData, customer: e.target.value })}
                    placeholder="Customer name"
                    required
                  />
                </div>
                <div>
                  <Label htmlFor="valid_until">Valid Until</Label>
                  <Input
                    id="valid_until"
                    type="date"
                    value={formData.valid_until}
                    onChange={(e) => setFormData({ ...formData, valid_until: e.target.value })}
                  />
                </div>
              </div>

              <div>
                <Label htmlFor="notes">Notes</Label>
                <Textarea
                  id="notes"
                  value={formData.notes}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  placeholder="Quote description and terms"
                  rows={2}
                />
              </div>

              {/* Line Items */}
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <Label>Line Items</Label>
                  <Button type="button" variant="outline" onClick={addLineItem}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Item
                  </Button>
                </div>

                <div className="border rounded-lg overflow-hidden">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>IPN *</TableHead>
                        <TableHead>Description</TableHead>
                        <TableHead>Qty *</TableHead>
                        <TableHead>Unit Price *</TableHead>
                        <TableHead>Total</TableHead>
                        <TableHead></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {formData.lines.map((line, index) => (
                        <TableRow key={index}>
                          <TableCell>
                            <Input
                              value={line.ipn || ""}
                              onChange={(e) => updateLineItem(index, "ipn", e.target.value)}
                              placeholder="Part number"
                              className="min-w-[120px]"
                            />
                          </TableCell>
                          <TableCell>
                            <Input
                              value={line.description || ""}
                              onChange={(e) => updateLineItem(index, "description", e.target.value)}
                              placeholder="Description"
                              className="min-w-[150px]"
                            />
                          </TableCell>
                          <TableCell>
                            <Input
                              type="number"
                              value={line.qty || 1}
                              onChange={(e) => updateLineItem(index, "qty", parseInt(e.target.value) || 1)}
                              min="1"
                              className="w-20"
                            />
                          </TableCell>
                          <TableCell>
                            <Input
                              type="number"
                              step="0.01"
                              value={line.unit_price || 0}
                              onChange={(e) => updateLineItem(index, "unit_price", parseFloat(e.target.value) || 0)}
                              min="0"
                              className="w-24"
                            />
                          </TableCell>
                          <TableCell>
                            <span className="font-medium">
                              ${((line.qty || 0) * (line.unit_price || 0)).toFixed(2)}
                            </span>
                          </TableCell>
                          <TableCell>
                            {formData.lines.length > 1 && (
                              <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                onClick={() => removeLineItem(index)}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                      <TableRow>
                        <TableCell colSpan={4} className="text-right font-medium">
                          Total:
                        </TableCell>
                        <TableCell className="font-bold">
                          ${formData.lines.reduce((sum, line) => sum + ((line.qty || 0) * (line.unit_price || 0)), 0).toFixed(2)}
                        </TableCell>
                        <TableCell></TableCell>
                      </TableRow>
                    </TableBody>
                  </Table>
                </div>
              </div>

              <div className="flex justify-end gap-2">
                <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit">Create Quote</Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            Quote Records
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Quote ID</TableHead>
                <TableHead>Customer</TableHead>
                <TableHead>Total</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Valid Until</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {quotes.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8">
                    <div className="text-muted-foreground">
                      No quotes found. Create your first quote to get started.
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                quotes.map((quote) => (
                  <TableRow key={quote.id} className="cursor-pointer hover:bg-muted/50" onClick={() => navigate(`/quotes/${quote.id}`)}>
                    <TableCell className="font-medium">{quote.id}</TableCell>
                    <TableCell>{quote.customer}</TableCell>
                    <TableCell className="font-medium">
                      ${calculateQuoteTotal(quote).toFixed(2)}
                    </TableCell>
                    <TableCell>
                      <Badge variant={getStatusBadgeVariant(quote.status)}>
                        {quote.status.charAt(0).toUpperCase() + quote.status.slice(1)}
                      </Badge>
                    </TableCell>
                    <TableCell>{new Date(quote.created_at).toLocaleDateString()}</TableCell>
                    <TableCell>
                      {quote.valid_until ? new Date(quote.valid_until).toLocaleDateString() : "—"}
                    </TableCell>
                    <TableCell>
                      <Button variant="outline" size="sm" onClick={(e) => { e.stopPropagation(); navigate(`/quotes/${quote.id}`); }}>
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

      {/* Quote Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <FileText className="h-8 w-8 text-blue-600" />
            </div>
            <div className="text-3xl font-bold text-blue-600 text-center">
              {quotes.length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Total Quotes
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
              {quotes.filter(q => q.status === "accepted").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Accepted
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <div className="h-8 w-8 rounded-full bg-yellow-500 flex items-center justify-center">
                ⏱
              </div>
            </div>
            <div className="text-3xl font-bold text-yellow-600 text-center">
              {quotes.filter(q => q.status === "sent").length}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Pending
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6">
            <div className="flex items-center justify-center mb-2">
              <div className="h-8 w-8 rounded-full bg-primary flex items-center justify-center text-primary-foreground font-bold">
                $
              </div>
            </div>
            <div className="text-3xl font-bold text-primary text-center">
              ${quotes.reduce((sum, quote) => sum + calculateQuoteTotal(quote), 0).toFixed(0)}
            </div>
            <div className="text-sm text-muted-foreground text-center mt-1">
              Total Value
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
export default Quotes;
