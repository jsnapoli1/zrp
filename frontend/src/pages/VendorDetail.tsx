import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { 
  ArrowLeft, 
  Building, 
  Phone,
  Mail,
  Globe,
  Calendar,
  DollarSign,
  Package,
  FileText
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "../components/ui/table";
import { api, type Vendor, type PurchaseOrder } from "../lib/api";

interface PriceCatalogItem {
  ipn: string;
  mpn: string;
  unit_price: number;
  lead_time_days?: number;
  last_updated: string;
}

function VendorDetail() {
  const { id } = useParams<{ id: string }>();
  const [vendor, setVendor] = useState<Vendor | null>(null);
  const [purchaseOrders, setPurchaseOrders] = useState<PurchaseOrder[]>([]);
  const [priceCatalog, setPriceCatalog] = useState<PriceCatalogItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      fetchVendorDetail();
      fetchVendorPurchaseOrders();
      fetchPriceCatalog();
    }
  }, [id]);

  const fetchVendorDetail = async () => {
    if (!id) return;
    
    try {
      setLoading(true);
      const data = await api.getVendor(id);
      setVendor(data);
    } catch (error) {
      console.error("Failed to fetch vendor:", error);
    } finally {
      setLoading(false);
    }
  };

  const fetchVendorPurchaseOrders = async () => {
    if (!id) return;
    
    try {
      // Filter POs by vendor ID - the API should support this but for now we'll filter client-side
      const allPOs = await api.getPurchaseOrders();
      const vendorPOs = allPOs.filter(po => po.vendor_id === id);
      setPurchaseOrders(vendorPOs);
    } catch (error) {
      console.error("Failed to fetch purchase orders:", error);
    }
  };

  const fetchPriceCatalog = async () => {
    if (!id) return;
    
    try {
      // This would typically be a separate API endpoint for vendor pricing
      // For now, we'll create mock data based on the vendor
      const mockCatalog: PriceCatalogItem[] = [
        {
          ipn: "RES-001",
          mpn: "RC0603FR-071KL",
          unit_price: 0.05,
          lead_time_days: 2,
          last_updated: "2024-01-15T10:00:00Z"
        },
        {
          ipn: "CAP-002", 
          mpn: "CL10B104KB8NNNC",
          unit_price: 0.08,
          lead_time_days: 3,
          last_updated: "2024-01-10T10:00:00Z"
        }
      ];
      setPriceCatalog(mockCatalog);
    } catch (error) {
      console.error("Failed to fetch price catalog:", error);
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  // Removed unused formatDateTime function

  const getStatusBadge = (status: string) => {
    const variants = {
      draft: "secondary",
      submitted: "default",
      partial: "outline",
      received: "default",
      closed: "secondary"
    } as const;

    const colors = {
      draft: "text-gray-700",
      submitted: "text-blue-700",
      partial: "text-orange-700",
      received: "text-green-700",
      closed: "text-gray-700"
    } as const;

    return (
      <Badge variant={variants[status as keyof typeof variants] || "secondary"}>
        <span className={colors[status as keyof typeof colors] || "text-gray-700"}>
          {status.toUpperCase()}
        </span>
      </Badge>
    );
  };

  const getVendorStatusBadge = (status: string) => {
    const variant = status === 'active' ? 'default' : 'secondary';
    const color = status === 'active' ? 'text-green-700' : 'text-gray-700';
    
    return (
      <Badge variant={variant}>
        <span className={color}>{status.toUpperCase()}</span>
      </Badge>
    );
  };

  const getTotalPOValue = () => {
    return purchaseOrders.reduce((sum, po) => {
      if (!po.lines) return sum;
      return sum + po.lines.reduce((lineSum, line) => 
        lineSum + (line.qty_ordered * (line.unit_price || 0)), 0
      );
    }, 0);
  };

  const getActivePOsCount = () => {
    return purchaseOrders.filter(po => ['submitted', 'partial'].includes(po.status)).length;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading vendor details...</p>
        </div>
      </div>
    );
  }

  if (!vendor) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" asChild>
            <Link to="/vendors">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Vendors
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="p-8 text-center">
            <h3 className="text-lg font-semibold mb-2">Vendor Not Found</h3>
            <p className="text-muted-foreground">
              The vendor "{id}" could not be found.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" asChild>
            <Link to="/vendors">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Vendors
            </Link>
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{vendor.name}</h1>
            <div className="flex items-center gap-2 mt-1">
              {getVendorStatusBadge(vendor.status)}
              <span className="text-sm text-muted-foreground">•</span>
              <span className="text-sm text-muted-foreground">
                Added {formatDate(vendor.created_at)}
              </span>
            </div>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" asChild>
            <Link to={`/procurement?vendor=${vendor.id}`}>
              Create PO
            </Link>
          </Button>
          <Button asChild>
            <Link to={`/vendors/${vendor.id}/edit`}>
              Edit Vendor
            </Link>
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Total POs</p>
                <p className="text-2xl font-bold">{purchaseOrders.length}</p>
              </div>
              <FileText className="h-8 w-8 text-blue-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Active POs</p>
                <p className="text-2xl font-bold text-orange-600">{getActivePOsCount()}</p>
              </div>
              <Package className="h-8 w-8 text-orange-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Total Value</p>
                <p className="text-2xl font-bold">${getTotalPOValue().toFixed(0)}</p>
              </div>
              <DollarSign className="h-8 w-8 text-green-600" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Lead Time</p>
                <p className="text-2xl font-bold">
                  {vendor.lead_time_days > 0 ? `${vendor.lead_time_days}d` : "—"}
                </p>
              </div>
              <Calendar className="h-8 w-8 text-gray-600" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Vendor Information */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Building className="h-5 w-5" />
            Contact Information
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Company Name</p>
                <p className="text-base font-medium">{vendor.name}</p>
              </div>
              
              <div>
                <p className="text-sm font-medium text-muted-foreground">Primary Contact</p>
                <p className="text-base">{vendor.contact_name || "—"}</p>
              </div>

              <div>
                <p className="text-sm font-medium text-muted-foreground">Email</p>
                {vendor.contact_email ? (
                  <a 
                    href={`mailto:${vendor.contact_email}`}
                    className="text-base text-blue-600 hover:underline flex items-center gap-2"
                  >
                    <Mail className="h-4 w-4" />
                    {vendor.contact_email}
                  </a>
                ) : (
                  <p className="text-base">—</p>
                )}
              </div>
            </div>
            
            <div className="space-y-4">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Phone</p>
                {vendor.contact_phone ? (
                  <a 
                    href={`tel:${vendor.contact_phone}`}
                    className="text-base text-blue-600 hover:underline flex items-center gap-2"
                  >
                    <Phone className="h-4 w-4" />
                    {vendor.contact_phone}
                  </a>
                ) : (
                  <p className="text-base">—</p>
                )}
              </div>

              <div>
                <p className="text-sm font-medium text-muted-foreground">Website</p>
                {vendor.website ? (
                  <a 
                    href={vendor.website.startsWith('http') ? vendor.website : `https://${vendor.website}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-base text-blue-600 hover:underline flex items-center gap-2"
                  >
                    <Globe className="h-4 w-4" />
                    {vendor.website}
                  </a>
                ) : (
                  <p className="text-base">—</p>
                )}
              </div>

              <div>
                <p className="text-sm font-medium text-muted-foreground">Lead Time</p>
                <p className="text-base">
                  {vendor.lead_time_days > 0 ? `${vendor.lead_time_days} days` : "Not specified"}
                </p>
              </div>
            </div>
          </div>
          
          {vendor.notes && (
            <div className="mt-6 pt-6 border-t">
              <p className="text-sm font-medium text-muted-foreground mb-2">Notes</p>
              <p className="text-base whitespace-pre-wrap">{vendor.notes}</p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Tabs for Price Catalog and PO History */}
      <Tabs defaultValue="price-catalog" className="space-y-4">
        <TabsList>
          <TabsTrigger value="price-catalog">Price Catalog</TabsTrigger>
          <TabsTrigger value="purchase-orders">Purchase Orders</TabsTrigger>
        </TabsList>

        <TabsContent value="price-catalog" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Price Catalog</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>IPN</TableHead>
                    <TableHead>MPN</TableHead>
                    <TableHead className="text-right">Unit Price</TableHead>
                    <TableHead>Lead Time</TableHead>
                    <TableHead>Last Updated</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {priceCatalog.map((item, index) => (
                    <TableRow key={index}>
                      <TableCell className="font-medium">{item.ipn}</TableCell>
                      <TableCell>{item.mpn}</TableCell>
                      <TableCell className="text-right font-mono">${item.unit_price.toFixed(2)}</TableCell>
                      <TableCell>
                        {item.lead_time_days ? `${item.lead_time_days} days` : "—"}
                      </TableCell>
                      <TableCell>{formatDate(item.last_updated)}</TableCell>
                    </TableRow>
                  ))}
                  {priceCatalog.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                        No price catalog entries found for this vendor
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="purchase-orders" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Purchase Order History</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>PO Number</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Total</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead>Expected</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {purchaseOrders.map((po) => (
                    <TableRow key={po.id}>
                      <TableCell>
                        <Link 
                          to={`/purchase-orders/${po.id}`}
                          className="font-medium text-blue-600 hover:underline"
                        >
                          {po.id}
                        </Link>
                      </TableCell>
                      <TableCell>{getStatusBadge(po.status)}</TableCell>
                      <TableCell className="text-right font-mono">
                        ${po.lines?.reduce((sum, line) => 
                          sum + (line.qty_ordered * (line.unit_price || 0)), 0
                        ).toFixed(2) || "0.00"}
                      </TableCell>
                      <TableCell>{formatDate(po.created_at)}</TableCell>
                      <TableCell>{po.expected_date ? formatDate(po.expected_date) : "—"}</TableCell>
                      <TableCell>
                        <Button variant="outline" size="sm" asChild>
                          <Link to={`/purchase-orders/${po.id}`}>
                            View
                          </Link>
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                  {purchaseOrders.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                        No purchase orders found for this vendor
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
export default VendorDetail;
