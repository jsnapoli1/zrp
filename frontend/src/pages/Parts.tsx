import { useEffect, useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "../components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import { Skeleton } from "../components/ui/skeleton";
import { 
  Package, 
  Search, 
  Filter,
  ChevronLeft,
  ChevronRight,
  RotateCcw
} from "lucide-react";
import { api, type Part, type Category, type ApiResponse } from "../lib/api";

interface PartWithFields extends Part {
  category?: string;
  description?: string;
  cost?: number;
  stock?: number;
  status?: string;
}

function Parts() {
  const navigate = useNavigate();
  const [parts, setParts] = useState<Part[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [totalParts, setTotalParts] = useState(0);
  const pageSize = 50;

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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Parts</h1>
          <p className="text-muted-foreground">
            Manage your parts inventory and specifications
          </p>
        </div>
        <Button variant="outline">
          <Package className="h-4 w-4 mr-2" />
          Add Part
        </Button>
      </div>

      {/* Filters Card */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-base font-medium">Filters</CardTitle>
        </CardHeader>
        <CardContent>
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
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>IPN</TableHead>
                    <TableHead>Category</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead className="text-right">Cost</TableHead>
                    <TableHead className="text-right">Stock</TableHead>
                    <TableHead>Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {displayParts.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                        {searchQuery || selectedCategory !== "all" 
                          ? "No parts found matching your criteria" 
                          : "No parts available"}
                      </TableCell>
                    </TableRow>
                  ) : (
                    displayParts.map((part) => (
                      <TableRow 
                        key={part.ipn}
                        className="cursor-pointer hover:bg-muted/50"
                        onClick={() => handleRowClick(part.ipn)}
                      >
                        <TableCell className="font-mono font-medium">
                          {part.ipn}
                        </TableCell>
                        <TableCell>
                          <Badge variant="secondary" className="capitalize">
                            {part.category}
                          </Badge>
                        </TableCell>
                        <TableCell className="max-w-xs truncate">
                          {part.description || 'No description'}
                        </TableCell>
                        <TableCell className="text-right">
                          {part.cost ? `$${part.cost.toFixed(2)}` : '-'}
                        </TableCell>
                        <TableCell className="text-right">
                          {part.stock !== undefined ? part.stock.toString() : '-'}
                        </TableCell>
                        <TableCell>
                          <Badge 
                            variant={part.status === 'active' ? 'default' : 'secondary'}
                          >
                            {part.status || 'active'}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>

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
