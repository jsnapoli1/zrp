import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { api, type RFQ, type RFQCompare, type RFQQuote } from "../lib/api";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Textarea } from "../components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";

const statusColors: Record<string, string> = {
  draft: "secondary",
  sent: "default",
  awarded: "default",
  closed: "outline",
};

export default function RFQDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [rfq, setRfq] = useState<RFQ | null>(null);
  const [compare, setCompare] = useState<RFQCompare | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const [editDueDate, setEditDueDate] = useState("");
  const [editNotes, setEditNotes] = useState("");
  const [emailBody, setEmailBody] = useState<{ subject: string; body: string } | null>(null);
  const [emailDialogOpen, setEmailDialogOpen] = useState(false);
  const [quoteDialogOpen, setQuoteDialogOpen] = useState(false);
  const [newQuote, setNewQuote] = useState({ rfq_vendor_id: 0, rfq_line_id: 0, unit_price: 0, lead_time_days: 0, moq: 0, notes: "" });
  const [awardSelections, setAwardSelections] = useState<Record<number, string>>({});

  const load = async () => {
    if (!id) return;
    const data = await api.getRFQ(id);
    setRfq(data);
    setEditTitle(data.title);
    setEditDueDate(data.due_date || "");
    setEditNotes(data.notes || "");
    try {
      const cmp = await api.compareRFQ(id);
      setCompare(cmp);
    } catch { /* ignore */ }
    setLoading(false);
  };

  useEffect(() => { load(); }, [id]);

  if (loading || !rfq) return <div className="p-6">Loading...</div>;

  const handleSave = async () => {
    await api.updateRFQ(rfq.id, {
      title: editTitle,
      due_date: editDueDate,
      notes: editNotes,
      lines: rfq.lines,
      vendors: rfq.vendors,
    });
    setEditing(false);
    load();
  };

  const handleSend = async () => {
    await api.sendRFQ(rfq.id);
    load();
  };

  const handleClose = async () => {
    await api.closeRFQ(rfq.id);
    load();
  };

  const handleAwardAll = async (vendorId: string) => {
    const result = await api.awardRFQ(rfq.id, vendorId);
    alert(`Awarded! PO created: ${result.po_id}`);
    load();
  };

  const handleAwardPerLine = async () => {
    const awards = Object.entries(awardSelections).map(([lineId, vendorId]) => ({
      line_id: parseInt(lineId),
      vendor_id: vendorId,
    }));
    if (awards.length === 0) { alert("Select vendors for lines first"); return; }
    const result = await api.awardRFQPerLine(rfq.id, awards);
    alert(`Awarded! POs created: ${result.po_ids.join(", ")}`);
    load();
  };

  const handleGenerateEmail = async () => {
    const email = await api.getRFQEmailBody(rfq.id);
    setEmailBody(email);
    setEmailDialogOpen(true);
  };

  const handleCopyEmail = () => {
    if (emailBody) {
      navigator.clipboard.writeText(emailBody.body);
      alert("Email body copied to clipboard!");
    }
  };

  const handleAddQuote = async () => {
    await api.createRFQQuote(rfq.id, newQuote);
    setQuoteDialogOpen(false);
    setNewQuote({ rfq_vendor_id: 0, rfq_line_id: 0, unit_price: 0, lead_time_days: 0, moq: 0, notes: "" });
    load();
  };

  const handleDelete = async () => {
    if (!confirm("Delete this RFQ?")) return;
    await api.deleteRFQ(rfq.id);
    navigate("/rfqs");
  };

  const lines = rfq.lines || [];
  const vendors = rfq.vendors || [];
  const quotes = rfq.quotes || [];

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{rfq.title}</h1>
          <Badge variant={statusColors[rfq.status] as any || "secondary"}>{rfq.status}</Badge>
          <span className="text-sm text-muted-foreground font-mono">{rfq.id}</span>
        </div>
        <div className="flex gap-2">
          {rfq.status === "draft" && <Button onClick={handleSend}>Send to Vendors</Button>}
          {rfq.status === "draft" && <Button variant="outline" onClick={handleGenerateEmail}>Generate Email</Button>}
          {(rfq.status === "sent" || rfq.status === "awarded") && <Button variant="outline" onClick={handleClose}>Close RFQ</Button>}
          {rfq.status === "draft" && <Button variant="outline" onClick={() => setEditing(!editing)}>{editing ? "Cancel" : "Edit"}</Button>}
          <Button variant="destructive" onClick={handleDelete}>Delete</Button>
        </div>
      </div>

      {editing && (
        <div className="border rounded-lg p-4 space-y-3 bg-muted/30">
          <div><Label>Title</Label><Input value={editTitle} onChange={(e) => setEditTitle(e.target.value)} /></div>
          <div><Label>Due Date</Label><Input type="date" value={editDueDate} onChange={(e) => setEditDueDate(e.target.value)} /></div>
          <div><Label>Notes</Label><Textarea value={editNotes} onChange={(e) => setEditNotes(e.target.value)} /></div>
          <Button onClick={handleSave}>Save Changes</Button>
        </div>
      )}

      <div className="text-sm text-muted-foreground">
        {rfq.due_date && <span>Due: {rfq.due_date} · </span>}
        Created by {rfq.created_by} on {rfq.created_at?.split("T")[0]}
        {rfq.notes && <span> · {rfq.notes}</span>}
      </div>

      <Tabs defaultValue="lines" className="w-full">
        <TabsList>
          <TabsTrigger value="lines">Lines ({lines.length})</TabsTrigger>
          <TabsTrigger value="vendors">Vendors ({vendors.length})</TabsTrigger>
          <TabsTrigger value="responses">Responses ({quotes.length})</TabsTrigger>
          <TabsTrigger value="compare">Compare</TabsTrigger>
          <TabsTrigger value="award">Award</TabsTrigger>
        </TabsList>

        <TabsContent value="lines">
          <div className="border rounded-lg">
            <table className="w-full">
              <thead><tr className="border-b bg-muted/50">
                <th className="text-left p-3">IPN</th>
                <th className="text-left p-3">Description</th>
                <th className="text-right p-3">Qty</th>
                <th className="text-left p-3">Unit</th>
              </tr></thead>
              <tbody>
                {lines.map((l) => (
                  <tr key={l.id} className="border-b">
                    <td className="p-3 font-mono text-sm">{l.ipn}</td>
                    <td className="p-3">{l.description}</td>
                    <td className="p-3 text-right">{l.qty}</td>
                    <td className="p-3">{l.unit}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </TabsContent>

        <TabsContent value="vendors">
          <div className="border rounded-lg">
            <table className="w-full">
              <thead><tr className="border-b bg-muted/50">
                <th className="text-left p-3">Vendor</th>
                <th className="text-left p-3">Status</th>
                <th className="text-left p-3">Quoted At</th>
                <th className="text-left p-3">Notes</th>
              </tr></thead>
              <tbody>
                {vendors.map((v) => (
                  <tr key={v.id} className="border-b">
                    <td className="p-3">{v.vendor_name || v.vendor_id}</td>
                    <td className="p-3"><Badge variant="outline">{v.status}</Badge></td>
                    <td className="p-3">{v.quoted_at || "—"}</td>
                    <td className="p-3">{v.notes || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </TabsContent>

        <TabsContent value="responses">
          <div className="flex justify-end mb-2">
            <Dialog open={quoteDialogOpen} onOpenChange={setQuoteDialogOpen}>
              <DialogTrigger asChild><Button size="sm">Add Quote</Button></DialogTrigger>
              <DialogContent>
                <DialogHeader><DialogTitle>Add Vendor Quote</DialogTitle></DialogHeader>
                <div className="space-y-3">
                  <div>
                    <Label>Vendor</Label>
                    <select className="w-full border rounded p-2" value={newQuote.rfq_vendor_id} onChange={(e) => setNewQuote({ ...newQuote, rfq_vendor_id: parseInt(e.target.value) })}>
                      <option value={0}>Select vendor...</option>
                      {vendors.map((v) => <option key={v.id} value={v.id}>{v.vendor_name || v.vendor_id}</option>)}
                    </select>
                  </div>
                  <div>
                    <Label>Line Item</Label>
                    <select className="w-full border rounded p-2" value={newQuote.rfq_line_id} onChange={(e) => setNewQuote({ ...newQuote, rfq_line_id: parseInt(e.target.value) })}>
                      <option value={0}>Select line...</option>
                      {lines.map((l) => <option key={l.id} value={l.id}>{l.ipn} - {l.description}</option>)}
                    </select>
                  </div>
                  <div><Label>Unit Price</Label><Input type="number" step="0.001" value={newQuote.unit_price} onChange={(e) => setNewQuote({ ...newQuote, unit_price: parseFloat(e.target.value) })} /></div>
                  <div><Label>Lead Time (days)</Label><Input type="number" value={newQuote.lead_time_days} onChange={(e) => setNewQuote({ ...newQuote, lead_time_days: parseInt(e.target.value) })} /></div>
                  <div><Label>MOQ</Label><Input type="number" value={newQuote.moq} onChange={(e) => setNewQuote({ ...newQuote, moq: parseInt(e.target.value) })} /></div>
                  <div><Label>Notes</Label><Textarea value={newQuote.notes} onChange={(e) => setNewQuote({ ...newQuote, notes: e.target.value })} /></div>
                  <Button onClick={handleAddQuote} disabled={!newQuote.rfq_vendor_id || !newQuote.rfq_line_id}>Add Quote</Button>
                </div>
              </DialogContent>
            </Dialog>
          </div>
          <div className="border rounded-lg">
            <table className="w-full">
              <thead><tr className="border-b bg-muted/50">
                <th className="text-left p-3">Vendor</th>
                <th className="text-left p-3">Line Item</th>
                <th className="text-right p-3">Unit Price</th>
                <th className="text-right p-3">Lead Time</th>
                <th className="text-right p-3">MOQ</th>
                <th className="text-left p-3">Notes</th>
              </tr></thead>
              <tbody>
                {quotes.map((q) => {
                  const vendor = vendors.find((v) => v.id === q.rfq_vendor_id);
                  const line = lines.find((l) => l.id === q.rfq_line_id);
                  return (
                    <tr key={q.id} className="border-b">
                      <td className="p-3">{vendor?.vendor_name || vendor?.vendor_id || q.rfq_vendor_id}</td>
                      <td className="p-3">{line?.ipn || q.rfq_line_id}</td>
                      <td className="p-3 text-right font-mono">${q.unit_price.toFixed(4)}</td>
                      <td className="p-3 text-right">{q.lead_time_days}d</td>
                      <td className="p-3 text-right">{q.moq}</td>
                      <td className="p-3">{q.notes || "—"}</td>
                    </tr>
                  );
                })}
                {quotes.length === 0 && <tr><td colSpan={6} className="p-3 text-center text-muted-foreground">No quotes yet</td></tr>}
              </tbody>
            </table>
          </div>
        </TabsContent>

        <TabsContent value="compare">
          {compare && compare.vendors.length > 0 ? (
            <div className="border rounded-lg overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="text-left p-3">Line Item</th>
                    {compare.vendors.map((v) => (
                      <th key={v.id} className="text-center p-3 min-w-[150px]">{v.vendor_name || v.vendor_id}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {compare.lines.map((line) => {
                    const matrix = (compare as any).matrix || {};
                    const lineQuotes = matrix[line.id] || {};
                    return (
                      <tr key={line.id} className="border-b">
                        <td className="p-3">
                          <div className="font-mono text-sm">{line.ipn}</div>
                          <div className="text-sm text-muted-foreground">{line.description} × {line.qty}</div>
                        </td>
                        {compare.vendors.map((v) => {
                          const q = lineQuotes[v.id];
                          return (
                            <td key={v.id} className="p-3 text-center">
                              {q ? (
                                <div>
                                  <div className="font-mono font-bold">${q.unit_price.toFixed(4)}</div>
                                  <div className="text-xs text-muted-foreground">{q.lead_time_days}d · MOQ {q.moq}</div>
                                  {q.notes && <div className="text-xs">{q.notes}</div>}
                                </div>
                              ) : <span className="text-muted-foreground">—</span>}
                            </td>
                          );
                        })}
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : <p className="text-muted-foreground">No vendor quotes to compare yet.</p>}
        </TabsContent>

        <TabsContent value="award">
          {rfq.status === "awarded" || rfq.status === "closed" ? (
            <p className="text-muted-foreground">This RFQ has been {rfq.status}.</p>
          ) : (
            <div className="space-y-4">
              <h3 className="font-semibold">Award per Line Item</h3>
              <div className="border rounded-lg">
                <table className="w-full">
                  <thead><tr className="border-b bg-muted/50">
                    <th className="text-left p-3">Line Item</th>
                    <th className="text-left p-3">Award to Vendor</th>
                  </tr></thead>
                  <tbody>
                    {lines.map((l) => (
                      <tr key={l.id} className="border-b">
                        <td className="p-3">{l.ipn} - {l.description}</td>
                        <td className="p-3">
                          <select className="border rounded p-1" value={awardSelections[l.id] || ""} onChange={(e) => setAwardSelections({ ...awardSelections, [l.id]: e.target.value })}>
                            <option value="">Select vendor...</option>
                            {vendors.map((v) => <option key={v.id} value={v.vendor_id}>{v.vendor_name || v.vendor_id}</option>)}
                          </select>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <div className="flex gap-2">
                <Button onClick={handleAwardPerLine}>Award Selected Lines</Button>
                {vendors.length > 0 && (
                  <div className="flex gap-2 items-center ml-4">
                    <span className="text-sm text-muted-foreground">Or award all to:</span>
                    {vendors.map((v) => (
                      <Button key={v.id} variant="outline" size="sm" onClick={() => handleAwardAll(v.vendor_id)}>
                        {v.vendor_name || v.vendor_id}
                      </Button>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </TabsContent>
      </Tabs>

      <Dialog open={emailDialogOpen} onOpenChange={setEmailDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader><DialogTitle>RFQ Email</DialogTitle></DialogHeader>
          {emailBody && (
            <div className="space-y-3">
              <div><Label>Subject</Label><Input readOnly value={emailBody.subject} /></div>
              <div><Label>Body</Label><Textarea readOnly value={emailBody.body} rows={15} className="font-mono text-sm" /></div>
              <Button onClick={handleCopyEmail}>Copy to Clipboard</Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
