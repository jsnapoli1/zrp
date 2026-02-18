import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { 
  Building, 
  Plus, 
  Phone,
  Mail,
  Globe,
  MoreHorizontal,
  Edit,
  Trash2
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Textarea } from "../components/ui/textarea";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "../components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
} from "../components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import { api, type Vendor } from "../lib/api";

function Vendors() {
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingVendor, setEditingVendor] = useState<Vendor | null>(null);

  const [vendorForm, setVendorForm] = useState({
    name: "",
    website: "",
    contact_name: "",
    contact_email: "",
    contact_phone: "",
    notes: "",
    status: "active",
    lead_time_days: 0,
  });

  useEffect(() => {
    fetchVendors();
  }, []);

  const fetchVendors = async () => {
    try {
      setLoading(true);
      const data = await api.getVendors();
      setVendors(data);
    } catch (error) {
      console.error("Failed to fetch vendors:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateVendor = async () => {
    try {
      await api.createVendor(vendorForm);
      setCreateDialogOpen(false);
      resetForm();
      fetchVendors();
    } catch (error) {
      console.error("Failed to create vendor:", error);
    }
  };

  const handleEditVendor = async () => {
    if (!editingVendor) return;
    
    try {
      await api.updateVendor(editingVendor.id, vendorForm);
      setEditDialogOpen(false);
      setEditingVendor(null);
      resetForm();
      fetchVendors();
    } catch (error) {
      console.error("Failed to update vendor:", error);
    }
  };

  const handleDeleteVendor = async (vendorId: string, vendorName: string) => {
    if (!confirm(`Delete vendor "${vendorName}"? This action cannot be undone.`)) {
      return;
    }
    
    try {
      await api.deleteVendor(vendorId);
      fetchVendors();
    } catch (error) {
      console.error("Failed to delete vendor:", error);
    }
  };

  const openEditDialog = (vendor: Vendor) => {
    setEditingVendor(vendor);
    setVendorForm({
      name: vendor.name,
      website: vendor.website || "",
      contact_name: vendor.contact_name || "",
      contact_email: vendor.contact_email || "",
      contact_phone: vendor.contact_phone || "",
      notes: vendor.notes || "",
      status: vendor.status,
      lead_time_days: vendor.lead_time_days,
    });
    setEditDialogOpen(true);
  };

  const resetForm = () => {
    setVendorForm({
      name: "",
      website: "",
      contact_name: "",
      contact_email: "",
      contact_phone: "",
      notes: "",
      status: "active",
      lead_time_days: 0,
    });
  };

  const getStatusBadge = (status: string) => {
    const variant = status === 'active' ? 'default' : 'secondary';
    const color = status === 'active' ? 'text-green-700' : 'text-gray-700';
    
    return (
      <Badge variant={variant}>
        <span className={color}>{status.toUpperCase()}</span>
      </Badge>
    );
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading vendors...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Vendors</h1>
          <p className="text-muted-foreground">
            Manage your supplier relationships and contact information.
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Add Vendor
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Add New Vendor</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="name">Company Name *</Label>
                  <Input
                    id="name"
                    value={vendorForm.name}
                    onChange={(e) => setVendorForm(prev => ({ ...prev, name: e.target.value }))}
                    placeholder="Company name"
                  />
                </div>
                <div>
                  <Label htmlFor="status">Status</Label>
                  <Select value={vendorForm.status} onValueChange={(value) => setVendorForm(prev => ({ ...prev, status: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="active">Active</SelectItem>
                      <SelectItem value="inactive">Inactive</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="contact_name">Contact Name</Label>
                  <Input
                    id="contact_name"
                    value={vendorForm.contact_name}
                    onChange={(e) => setVendorForm(prev => ({ ...prev, contact_name: e.target.value }))}
                    placeholder="Primary contact"
                  />
                </div>
                <div>
                  <Label htmlFor="lead_time_days">Lead Time (Days)</Label>
                  <Input
                    id="lead_time_days"
                    type="number"
                    min="0"
                    value={vendorForm.lead_time_days}
                    onChange={(e) => setVendorForm(prev => ({ ...prev, lead_time_days: parseInt(e.target.value) || 0 }))}
                    placeholder="0"
                  />
                </div>
              </div>

              <div>
                <Label htmlFor="contact_email">Email</Label>
                <Input
                  id="contact_email"
                  type="email"
                  value={vendorForm.contact_email}
                  onChange={(e) => setVendorForm(prev => ({ ...prev, contact_email: e.target.value }))}
                  placeholder="contact@vendor.com"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="contact_phone">Phone</Label>
                  <Input
                    id="contact_phone"
                    value={vendorForm.contact_phone}
                    onChange={(e) => setVendorForm(prev => ({ ...prev, contact_phone: e.target.value }))}
                    placeholder="Phone number"
                  />
                </div>
                <div>
                  <Label htmlFor="website">Website</Label>
                  <Input
                    id="website"
                    value={vendorForm.website}
                    onChange={(e) => setVendorForm(prev => ({ ...prev, website: e.target.value }))}
                    placeholder="https://vendor.com"
                  />
                </div>
              </div>

              <div>
                <Label htmlFor="notes">Notes</Label>
                <Textarea
                  id="notes"
                  value={vendorForm.notes}
                  onChange={(e) => setVendorForm(prev => ({ ...prev, notes: e.target.value }))}
                  placeholder="Additional notes..."
                  rows={3}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreateVendor} disabled={!vendorForm.name}>
                Create Vendor
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Total Vendors</p>
                <p className="text-2xl font-bold">{vendors.length}</p>
              </div>
              <Building className="h-8 w-8 text-blue-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Active</p>
                <p className="text-2xl font-bold text-green-600">
                  {vendors.filter(v => v.status === 'active').length}
                </p>
              </div>
              <Badge variant="default" className="h-8 w-8 rounded-full flex items-center justify-center">
                <span className="text-xs">✓</span>
              </Badge>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Inactive</p>
                <p className="text-2xl font-bold text-gray-600">
                  {vendors.filter(v => v.status === 'inactive').length}
                </p>
              </div>
              <Badge variant="secondary" className="h-8 w-8 rounded-full flex items-center justify-center">
                <span className="text-xs">—</span>
              </Badge>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Vendors Table */}
      <Card>
        <CardHeader>
          <CardTitle>Vendor Directory</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Company</TableHead>
                <TableHead>Contact</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Phone</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Lead Time</TableHead>
                <TableHead>Added</TableHead>
                <TableHead className="w-10"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {vendors.map((vendor) => (
                <TableRow key={vendor.id}>
                  <TableCell>
                    <div>
                      <Link 
                        to={`/vendors/${vendor.id}`}
                        className="font-medium text-blue-600 hover:underline"
                      >
                        {vendor.name}
                      </Link>
                      {vendor.website && (
                        <div className="flex items-center gap-1 mt-1">
                          <Globe className="h-3 w-3 text-muted-foreground" />
                          <a 
                            href={vendor.website.startsWith('http') ? vendor.website : `https://${vendor.website}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs text-blue-600 hover:underline"
                          >
                            Website
                          </a>
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>{vendor.contact_name || "—"}</TableCell>
                  <TableCell>
                    {vendor.contact_email ? (
                      <a 
                        href={`mailto:${vendor.contact_email}`}
                        className="text-blue-600 hover:underline flex items-center gap-1"
                      >
                        <Mail className="h-3 w-3" />
                        {vendor.contact_email}
                      </a>
                    ) : "—"}
                  </TableCell>
                  <TableCell>
                    {vendor.contact_phone ? (
                      <a 
                        href={`tel:${vendor.contact_phone}`}
                        className="text-blue-600 hover:underline flex items-center gap-1"
                      >
                        <Phone className="h-3 w-3" />
                        {vendor.contact_phone}
                      </a>
                    ) : "—"}
                  </TableCell>
                  <TableCell>{getStatusBadge(vendor.status)}</TableCell>
                  <TableCell>
                    {vendor.lead_time_days > 0 ? `${vendor.lead_time_days} days` : "—"}
                  </TableCell>
                  <TableCell>{formatDate(vendor.created_at)}</TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem asChild>
                          <Link to={`/vendors/${vendor.id}`}>View Details</Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => openEditDialog(vendor)}>
                          <Edit className="h-4 w-4 mr-2" />
                          Edit
                        </DropdownMenuItem>
                        <DropdownMenuItem 
                          className="text-red-600"
                          onClick={() => handleDeleteVendor(vendor.id, vendor.name)}
                        >
                          <Trash2 className="h-4 w-4 mr-2" />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
              {vendors.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                    No vendors found
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Edit Vendor Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Vendor</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="edit_name">Company Name *</Label>
                <Input
                  id="edit_name"
                  value={vendorForm.name}
                  onChange={(e) => setVendorForm(prev => ({ ...prev, name: e.target.value }))}
                  placeholder="Company name"
                />
              </div>
              <div>
                <Label htmlFor="edit_status">Status</Label>
                <Select value={vendorForm.status} onValueChange={(value) => setVendorForm(prev => ({ ...prev, status: value }))}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="inactive">Inactive</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="edit_contact_name">Contact Name</Label>
                <Input
                  id="edit_contact_name"
                  value={vendorForm.contact_name}
                  onChange={(e) => setVendorForm(prev => ({ ...prev, contact_name: e.target.value }))}
                  placeholder="Primary contact"
                />
              </div>
              <div>
                <Label htmlFor="edit_lead_time_days">Lead Time (Days)</Label>
                <Input
                  id="edit_lead_time_days"
                  type="number"
                  min="0"
                  value={vendorForm.lead_time_days}
                  onChange={(e) => setVendorForm(prev => ({ ...prev, lead_time_days: parseInt(e.target.value) || 0 }))}
                  placeholder="0"
                />
              </div>
            </div>

            <div>
              <Label htmlFor="edit_contact_email">Email</Label>
              <Input
                id="edit_contact_email"
                type="email"
                value={vendorForm.contact_email}
                onChange={(e) => setVendorForm(prev => ({ ...prev, contact_email: e.target.value }))}
                placeholder="contact@vendor.com"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="edit_contact_phone">Phone</Label>
                <Input
                  id="edit_contact_phone"
                  value={vendorForm.contact_phone}
                  onChange={(e) => setVendorForm(prev => ({ ...prev, contact_phone: e.target.value }))}
                  placeholder="Phone number"
                />
              </div>
              <div>
                <Label htmlFor="edit_website">Website</Label>
                <Input
                  id="edit_website"
                  value={vendorForm.website}
                  onChange={(e) => setVendorForm(prev => ({ ...prev, website: e.target.value }))}
                  placeholder="https://vendor.com"
                />
              </div>
            </div>

            <div>
              <Label htmlFor="edit_notes">Notes</Label>
              <Textarea
                id="edit_notes"
                value={vendorForm.notes}
                onChange={(e) => setVendorForm(prev => ({ ...prev, notes: e.target.value }))}
                placeholder="Additional notes..."
                rows={3}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleEditVendor} disabled={!vendorForm.name}>
              Update Vendor
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
export default Vendors;
