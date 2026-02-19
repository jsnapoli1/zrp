import { useEffect, useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Textarea } from "../components/ui/textarea";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "../components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "../components/ui/dialog";
// Table components used by ConfigurableTable internally
import { Skeleton } from "../components/ui/skeleton";
import { 
  Search, 
  Filter,
  ScanLine,
  ChevronLeft,
  ChevronRight,
  RotateCcw,
  Plus
} from "lucide-react";
import { api, type Part, type Category, type ApiResponse } from "../lib/api";
import { ConfigurableTable, type ColumnDef } from "../components/ConfigurableTable";
import { BarcodeScanner } from "../components/BarcodeScanner";

interface PartWithFields extends Part {
  category?: string;
  description?: string;
  cost?: number;
  stock?: number;
  status?: string;
}

interface CreatePartData {
  ipn: string;
  description: string;
  category: string;
  cost: string;
  price: string;
  minimum_stock: string;
  current_stock: string;
  location: string;
  vendor: string;
  status: string;
}

function Parts() {
  const navigate = useNavigate();
  const [parts, setParts] = useState<Part[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [showScanner, setShowScanner] = useState(false);
  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [totalParts, setTotalParts] = useState(0);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const pageSize = 50;

  const [partForm, setPartForm] = useState<CreatePartData>({
    ipn: "",
    description: "",
    category: "",
    cost: "",
    price: "",
    minimum_stock: "",
    current_stock: "",
    location: "",
    vendor: "",
    status: "active"
  });

  // Debounced search effect
  useEffect(() => {
    const timeoutId = setTimeout(() => {
      fetchParts();
    }, 300);
    return () => clearTimeout(timeoutId);
  }, [searchQuery, selectedCategory, currentPage]);

  // Load categories on mount
  useEffect(() => {
    fetchCategories();
  }, []);

  const fetchCategories = async () => {
    try {
      const data = await api.getCategories();
      setCategories(data);
    } catch (error) {
      console.error("Failed to fetch categories:", error);
    }
  };

  const fetchParts = async () => {
    setLoading(true);
    try {
      const params: any = {
        page: currentPage,
        limit: pageSize,
      };
      
      if (searchQuery.trim()) {
        params.q = searchQuery.trim();
      }
      
      if (selectedCategory !== "all") {
        params.category = selectedCategory;
      }

      const response: ApiResponse<Part[]> = await api.getParts(params);
      setParts(response.data || []);
      setTotalParts(response.meta?.total || 0);
    } catch (error) {
      console.error("Failed to fetch parts:", error);
      setParts([]);
      setTotalParts(0);
    } finally {
      setLoading(false);
    }
  };

  const handleRowClick = (ipn: string) => {
    navigate(`/parts/${encodeURIComponent(ipn)}`);
  };

  const handleSearch = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(1); // Reset to first page on search
  };

  const handleCategoryChange = (value: string) => {
    setSelectedCategory(value);
    setCurrentPage(1); // Reset to first page on filter change
  };

  const handleReset = () => {
    setSearchQuery("");
    setSelectedCategory("all");
    setCurrentPage(1);
  };

  const handleCreatePart = async () => {
    setCreating(true);
    try {
      const partData = {
        ipn: partForm.ipn,
        description: partForm.description,
        cost: partForm.cost ? parseFloat(partForm.cost) : undefined,
        price: partForm.price ? parseFloat(partForm.price) : undefined,
        minimum_stock: partForm.minimum_stock ? parseInt(partForm.minimum_stock) : undefined,
        current_stock: partForm.current_stock ? parseInt(partForm.current_stock) : undefined,
        location: partForm.location || undefined,
        vendor: partForm.vendor || undefined,
        status: partForm.status,
        fields: {
          category: partForm.category,
          description: partForm.description,
          cost: partForm.cost,
          stock: partForm.current_stock,
          status: partForm.status
        }
      };
      
      await api.createPart(partData);
      setCreateDialogOpen(false);
      setPartForm({
        ipn: "",
        description: "",
        category: "",
        cost: "",
        price: "",
        minimum_stock: "",
        current_stock: "",
        location: "",
        vendor: "",
        status: "active"
      });
      
      // Refresh the parts list
      fetchParts();
    } catch (error) {
      console.error('Failed to create part:', error);
    } finally {
      setCreating(false);
    }
  };

  // Calculate pagination
  const totalPages = Math.ceil(totalParts / pageSize);
  const hasNextPage = currentPage < totalPages;
  const hasPrevPage = currentPage > 1;

  // Extract fields for display
  const displayParts = useMemo(() => {
    return parts.map(part => {
      const fields = part.fields || {};
      return {
        ...part,
        category: fields._category || fields.category || 'Unknown',
        description: fields.description || fields.desc || '',
        cost: parseFloat(fields.cost || fields.unit_price || '0') || undefined,
        stock: parseFloat(fields.stock || fields.qty_on_hand || fields.current_stock || '0') || undefined,
        status: fields.status || 'active',
      } as PartWithFields;
    });
  }, [parts]);

  const partsColumns: ColumnDef<PartWithFields>[] = [
    {
      id: "ipn",
      label: "IPN",
      accessor: (part) => <span className="font-mono font-medium">{part.ipn}</span>,
      defaultVisible: true,
    },
    {
      id: "category",
      label: "Category",
      accessor: (part) => <Badge variant="secondary" className="capitalize">{part.category}</Badge>,
      defaultVisible: true,
    },
    {
      id: "description",
      label: "Description",
      accessor: (part) => <span className="max-w-xs truncate block">{part.description || "No description"}</span>,
      defaultVisible: true,
    },
    {
      id: "cost",
      label: "Cost",
      accessor: (part) => part.cost ? `$${part.cost.toFixed(2)}` : "—",
      className: "text-right",
      headerClassName: "text-right",
      defaultVisible: true,
    },
    {
      id: "stock",
      label: "Stock",
      accessor: (part) => part.stock !== undefined ? part.stock.toString() : "—",
      className: "text-right",
      headerClassName: "text-right",
      defaultVisible: true,
    },
    {
      id: "status",
      label: "Status",
      accessor: (part) => (
        <Badge variant={part.status === "active" ? "default" : "secondary"}>
          {part.status || "active"}
        </Badge>
      ),
      defaultVisible: true,
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Parts</h1>
          <p className="text-muted-foreground">
            Manage your parts inventory and specifications
          </p>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Add Part
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[600px]">
            <DialogHeader>
              <DialogTitle>Add New Part</DialogTitle>
              <DialogDescription>
                Create a new part in your inventory system.
              </DialogDescription>
            </DialogHeader>
            
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="ipn">IPN *</Label>
                <Input
                  id="ipn"
                  placeholder="Internal Part Number"
                  value={partForm.ipn}
                  onChange={(e) => setPartForm(prev => ({ ...prev, ipn: e.target.value }))}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="category">Category</Label>
                <Select
                  value={partForm.category}
                  onValueChange={(value) => setPartForm(prev => ({ ...prev, category: value }))}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select category" />
                  </SelectTrigger>
                  <SelectContent>
                    {categories.map((category) => (
                      <SelectItem key={category.id} value={category.id}>
                        {category.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              
              <div className="col-span-2 space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  placeholder="Part description..."
                  value={partForm.description}
                  onChange={(e) => setPartForm(prev => ({ ...prev, description: e.target.value }))}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="cost">Cost ($)</Label>
                <Input
                  id="cost"
                  type="number"
                  step="0.01"
                  placeholder="0.00"
                  value={partForm.cost}
                  onChange={(e) => setPartForm(prev => ({ ...prev, cost: e.target.value }))}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="price">Price ($)</Label>
                <Input
                  id="price"
                  type="number"
                  step="0.01"
                  placeholder="0.00"
                  value={partForm.price}
                  onChange={(e) => setPartForm(prev => ({ ...prev, price: e.target.value }))}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="minimum_stock">Minimum Stock</Label>
                <Input
                  id="minimum_stock"
                  type="number"
                  placeholder="0"
                  value={partForm.minimum_stock}
                  onChange={(e) => setPartForm(prev => ({ ...prev, minimum_stock: e.target.value }))}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="current_stock">Current Stock</Label>
                <Input
                  id="current_stock"
                  type="number"
                  placeholder="0"
                  value={partForm.current_stock}
                  onChange={(e) => setPartForm(prev => ({ ...prev, current_stock: e.target.value }))}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="location">Location</Label>
                <Input
                  id="location"
                  placeholder="Storage location"
                  value={partForm.location}
                  onChange={(e) => setPartForm(prev => ({ ...prev, location: e.target.value }))}
                />
              </div>
              
              <div className="space-y-2">
                <Label htmlFor="vendor">Vendor</Label>
                <Input
                  id="vendor"
                  placeholder="Primary vendor"
                  value={partForm.vendor}
                  onChange={(e) => setPartForm(prev => ({ ...prev, vendor: e.target.value }))}
                />
              </div>
            </div>
            
            <DialogFooter>
              <Button 
                type="button" 
                variant="outline" 
                onClick={() => setCreateDialogOpen(false)}
                disabled={creating}
              >
                Cancel
              </Button>
              <Button 
                onClick={handleCreatePart}
                disabled={creating || !partForm.ipn}
              >
                {creating ? 'Creating...' : 'Create Part'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Filters Card */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-base font-medium">Filters</CardTitle>
        </CardHeader>
        <CardContent>
          {showScanner && (
            <div className="mb-4">
              <BarcodeScanner
                onScan={(code) => {
                  handleSearch(code);
                  setShowScanner(false);
                }}
              />
            </div>
          )}
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Search parts by IPN, description..."
                  value={searchQuery}
                  onChange={(e) => handleSearch(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>
            <Button
              variant="outline"
              onClick={() => setShowScanner(!showScanner)}
            >
              <ScanLine className="h-4 w-4 mr-1" />
              Scan
            </Button>
            <div className="w-full sm:w-48">
              <Select value={selectedCategory} onValueChange={handleCategoryChange}>
                <SelectTrigger>
                  <Filter className="h-4 w-4 mr-2" />
                  <SelectValue placeholder="Category" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Categories</SelectItem>
                  {categories.map((category) => (
                    <SelectItem key={category.id} value={category.id}>
                      {category.name} ({category.count})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button variant="outline" onClick={handleReset}>
              <RotateCcw className="h-4 w-4" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Results */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>
            Parts ({totalParts.toLocaleString()})
          </CardTitle>
          <div className="text-sm text-muted-foreground">
            Page {currentPage} of {totalPages}
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <>
              <ConfigurableTable<PartWithFields>
                tableName="parts"
                columns={partsColumns}
                data={displayParts}
                rowKey={(part) => part.ipn}
                onRowClick={(part) => handleRowClick(part.ipn)}
                emptyMessage={
                  searchQuery || selectedCategory !== "all"
                    ? "No parts found matching your criteria"
                    : "No parts available"
                }
              />

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-6">
                  <div className="text-sm text-muted-foreground">
                    Showing {((currentPage - 1) * pageSize) + 1} to {Math.min(currentPage * pageSize, totalParts)} of {totalParts} parts
                  </div>
                  <div className="flex items-center space-x-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(currentPage - 1)}
                      disabled={!hasPrevPage}
                    >
                      <ChevronLeft className="h-4 w-4 mr-1" />
                      Previous
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(currentPage + 1)}
                      disabled={!hasNextPage}
                    >
                      Next
                      <ChevronRight className="h-4 w-4 ml-1" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
export default Parts;
