import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Separator } from "../components/ui/separator";
import { Skeleton } from "../components/ui/skeleton";
import { 
  ArrowLeft, 
  Package, 
  ChevronDown, 
  ChevronRight,
  DollarSign,
  Layers,
  Info,
  GitBranch,
  RefreshCw,
  Store
} from "lucide-react";
import { Link } from "react-router-dom";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../components/ui/table";
import { api, type Part, type BOMNode, type PartCost, type WhereUsedEntry, type MarketPricingResult } from "../lib/api";
import { useGitPLM } from "../hooks/useGitPLM";
import { ExternalLink } from "lucide-react";

interface PartWithDetails extends Part {
  category?: string;
  description?: string;
  manufacturer?: string;
  mpn?: string;
  cost?: number;
  price?: number;
  stock?: number;
  location?: string;
  status?: string;
  datasheet?: string;
  notes?: string;
}

interface BOMTreeProps {
  node: BOMNode;
  level?: number;
  onPartClick?: (ipn: string) => void;
  gitplmBuildUrl?: (ipn: string) => string | null;
}

function BOMTree({ node, level = 0, onPartClick, gitplmBuildUrl }: BOMTreeProps) {
  const [expanded, setExpanded] = useState(level < 2); // Auto-expand first 2 levels
  const hasChildren = node.children && node.children.length > 0;
  
  const handleToggle = () => {
    if (hasChildren) {
      setExpanded(!expanded);
    }
  };

  const handlePartClick = (ipn: string) => {
    if (onPartClick) {
      onPartClick(ipn);
    }
  };

  return (
    <div className="select-none">
      <div 
        className={`flex items-center py-2 px-3 rounded-md hover:bg-muted/50 cursor-pointer ${
          level > 0 ? 'ml-' + (level * 4) : ''
        }`}
        onClick={handleToggle}
      >
        <div className="flex items-center min-w-0 flex-1">
          {hasChildren ? (
            expanded ? 
              <ChevronDown className="h-4 w-4 text-muted-foreground mr-2 flex-shrink-0" /> :
              <ChevronRight className="h-4 w-4 text-muted-foreground mr-2 flex-shrink-0" />
          ) : (
            <div className="w-6 mr-2 flex-shrink-0" />
          )}
          
          <div className="flex items-center min-w-0 flex-1">
            <span 
              className="font-mono text-sm font-medium text-primary hover:underline mr-1"
              onClick={(e) => {
                e.stopPropagation();
                handlePartClick(node.ipn);
              }}
            >
              {node.ipn}
            </span>
            {gitplmBuildUrl && gitplmBuildUrl(node.ipn) && (
              <a
                href={gitplmBuildUrl(node.ipn)!}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
                className="text-muted-foreground hover:text-primary mr-2"
                title="Open in gitplm"
              >
                <ExternalLink className="h-3 w-3" />
              </a>
            )}
            <span className="text-sm text-muted-foreground truncate">
              {node.description || 'No description'}
            </span>
          </div>
          
          <div className="flex items-center space-x-3 ml-4">
            {node.qty && node.qty > 0 && (
              <Badge variant="outline" className="text-xs">
                Qty: {node.qty}
              </Badge>
            )}
            {node.ref && (
              <Badge variant="secondary" className="text-xs">
                {node.ref}
              </Badge>
            )}
          </div>
        </div>
      </div>
      
      {expanded && hasChildren && (
        <div className="ml-4 border-l-2 border-muted pl-2">
          {node.children.map((child, index) => (
            <BOMTree 
              key={`${child.ipn}-${index}`} 
              node={child} 
              level={level + 1}
              onPartClick={onPartClick}
              gitplmBuildUrl={gitplmBuildUrl}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function PartDetail() {
  const { ipn } = useParams<{ ipn: string }>();
  const navigate = useNavigate();
  const [part, setPart] = useState<PartWithDetails | null>(null);
  const [bom, setBom] = useState<BOMNode | null>(null);
  const [cost, setCost] = useState<PartCost | null>(null);
  const [loading, setLoading] = useState(true);
  const [bomLoading, setBomLoading] = useState(false);
  const [costLoading, setCostLoading] = useState(false);
  const [whereUsed, setWhereUsed] = useState<WhereUsedEntry[]>([]);
  const [whereUsedLoading, setWhereUsedLoading] = useState(false);
  const [marketPricing, setMarketPricing] = useState<MarketPricingResult[]>([]);
  const [marketPricingLoading, setMarketPricingLoading] = useState(false);
  const [marketPricingCached, setMarketPricingCached] = useState(false);
  const [marketPricingError, setMarketPricingError] = useState<string>("");
  const [marketPricingUnconfigured, setMarketPricingUnconfigured] = useState<string[]>([]);
  const { configured: gitplmConfigured, buildUrl: gitplmUrl } = useGitPLM();

  useEffect(() => {
    if (ipn) {
      fetchPartDetails();
    }
  }, [ipn]);

  const fetchPartDetails = async () => {
    if (!ipn) return;
    
    setLoading(true);
    try {
      const partData = await api.getPart(decodeURIComponent(ipn));
      
      // Transform fields for display
      const fields = partData.fields || {};
      const detailedPart: PartWithDetails = {
        ...partData,
        category: fields._category || fields.category,
        description: fields.description || fields.desc,
        manufacturer: fields.manufacturer || fields.mfg,
        mpn: fields.mpn || fields.manufacturer_part_number,
        cost: parseFloat(fields.cost || fields.unit_cost || '0') || undefined,
        price: parseFloat(fields.price || fields.unit_price || '0') || undefined,
        stock: parseFloat(fields.stock || fields.qty_on_hand || fields.current_stock || '0') || undefined,
        location: fields.location,
        status: fields.status || 'active',
        datasheet: fields.datasheet || fields.datasheet_url,
        notes: fields.notes || fields.comments,
      };
      
      setPart(detailedPart);

      // Load BOM if this is an assembly
      const upperIPN = ipn.toUpperCase();
      if (upperIPN.startsWith('PCA-') || upperIPN.startsWith('ASY-')) {
        fetchBOM();
      }

      // Load cost information
      fetchCost();

      // Load where-used
      fetchWhereUsed();

      // Load market pricing if part has MPN
      if (detailedPart.mpn) {
        fetchMarketPricing(false);
      }
    } catch (error) {
      console.error("Failed to fetch part details:", error);
    } finally {
      setLoading(false);
    }
  };

  const fetchBOM = async () => {
    if (!ipn) return;
    
    setBomLoading(true);
    try {
      const bomData = await api.getPartBOM(decodeURIComponent(ipn));
      setBom(bomData);
    } catch (error) {
      console.error("Failed to fetch BOM:", error);
    } finally {
      setBomLoading(false);
    }
  };

  const fetchCost = async () => {
    if (!ipn) return;
    
    setCostLoading(true);
    try {
      const costData = await api.getPartCost(decodeURIComponent(ipn));
      setCost(costData);
    } catch (error) {
      console.error("Failed to fetch cost data:", error);
    } finally {
      setCostLoading(false);
    }
  };

  const fetchWhereUsed = async () => {
    if (!ipn) return;
    setWhereUsedLoading(true);
    try {
      const data = await api.getPartWhereUsed(decodeURIComponent(ipn));
      setWhereUsed(data);
    } catch (error) {
      console.error("Failed to fetch where-used:", error);
    } finally {
      setWhereUsedLoading(false);
    }
  };

  const fetchMarketPricing = async (refresh: boolean) => {
    if (!ipn) return;
    setMarketPricingLoading(true);
    try {
      const data = await api.getMarketPricing(decodeURIComponent(ipn), refresh);
      setMarketPricing(data.results || []);
      setMarketPricingCached(data.cached || false);
      setMarketPricingError(data.error || "");
      setMarketPricingUnconfigured(data.unconfigured || []);
    } catch (error) {
      console.error("Failed to fetch market pricing:", error);
    } finally {
      setMarketPricingLoading(false);
    }
  };

  const handleBOMPartClick = (bomIPN: string) => {
    navigate(`/parts/${encodeURIComponent(bomIPN)}`);
  };

  const isAssembly = ipn && (ipn.toUpperCase().startsWith('PCA-') || ipn.toUpperCase().startsWith('ASY-'));

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Skeleton className="h-10 w-10" />
          <Skeleton className="h-8 w-64" />
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Skeleton className="h-96" />
          <Skeleton className="h-96" />
        </div>
      </div>
    );
  }

  if (!part) {
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" onClick={() => navigate('/parts')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Parts
          </Button>
        </div>
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Package className="h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">Part Not Found</h3>
            <p className="text-muted-foreground text-center">
              The part with IPN "{ipn}" could not be found.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" onClick={() => navigate('/parts')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Parts
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight font-mono">{part.ipn}</h1>
            <p className="text-muted-foreground">
              {part.description || 'No description available'}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          <Badge variant="secondary" className="capitalize">
            {part.category || 'Unknown'}
          </Badge>
          <Badge variant={part.status === 'active' ? 'default' : 'secondary'}>
            {part.status || 'active'}
          </Badge>
          {gitplmConfigured && ipn && (
            <Button variant="outline" size="sm" asChild>
              <a href={gitplmUrl(ipn)!} target="_blank" rel="noopener noreferrer">
                <ExternalLink className="h-4 w-4 mr-2" />
                Edit in gitplm
              </a>
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Part Details */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Info className="h-5 w-5 mr-2" />
              Part Details
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-sm font-medium text-muted-foreground">IPN</label>
                <p className="font-mono">{part.ipn}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-muted-foreground">Category</label>
                <p className="capitalize">{part.category || 'Unknown'}</p>
              </div>
              {part.manufacturer && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Manufacturer</label>
                  <p>{part.manufacturer}</p>
                </div>
              )}
              {part.mpn && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">MPN</label>
                  <p className="font-mono">{part.mpn}</p>
                </div>
              )}
              {part.stock !== undefined && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Stock</label>
                  <p>{part.stock}</p>
                </div>
              )}
              {part.location && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Location</label>
                  <p>{part.location}</p>
                </div>
              )}
            </div>
            
            {part.description && (
              <>
                <Separator />
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Description</label>
                  <p className="mt-1">{part.description}</p>
                </div>
              </>
            )}

            {part.notes && (
              <div>
                <label className="text-sm font-medium text-muted-foreground">Notes</label>
                <p className="mt-1 text-sm">{part.notes}</p>
              </div>
            )}

            {part.datasheet && (
              <div>
                <label className="text-sm font-medium text-muted-foreground">Datasheet</label>
                <div className="mt-1">
                  <Button variant="outline" size="sm" asChild>
                    <a href={part.datasheet} target="_blank" rel="noopener noreferrer">
                      View Datasheet
                    </a>
                  </Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Cost Information */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <DollarSign className="h-5 w-5 mr-2" />
              Cost Information
            </CardTitle>
          </CardHeader>
          <CardContent>
            {costLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </div>
            ) : (
              <div className="space-y-4">
                {part.cost && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Unit Cost</label>
                    <p className="text-2xl font-bold">${part.cost.toFixed(2)}</p>
                  </div>
                )}
                
                {cost?.last_unit_price && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Last Purchase Price</label>
                    <p className="text-lg font-semibold">${cost.last_unit_price.toFixed(2)}</p>
                    {cost.po_id && (
                      <p className="text-sm text-muted-foreground">
                        PO: {cost.po_id}
                        {cost.last_ordered && (
                          <span> • {new Date(cost.last_ordered).toLocaleDateString()}</span>
                        )}
                      </p>
                    )}
                  </div>
                )}

                {cost?.bom_cost && cost.bom_cost > 0 && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">BOM Cost Rollup</label>
                    <p className="text-lg font-semibold">${cost.bom_cost.toFixed(2)}</p>
                    <p className="text-sm text-muted-foreground">
                      Based on latest purchase prices
                    </p>
                  </div>
                )}

                {!cost?.last_unit_price && !part.cost && (
                  <p className="text-muted-foreground">No cost information available</p>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* BOM Tree for Assemblies */}
      {isAssembly && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Layers className="h-5 w-5 mr-2" />
              Bill of Materials
            </CardTitle>
          </CardHeader>
          <CardContent>
            {bomLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            ) : bom ? (
              <div className="border rounded-md p-4">
                <BOMTree node={bom} onPartClick={handleBOMPartClick} gitplmBuildUrl={gitplmConfigured ? gitplmUrl : undefined} />
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                <Layers className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>No BOM data available for this assembly</p>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Market Pricing */}
      {part?.mpn && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center">
                <Store className="h-5 w-5 mr-2" />
                Market Pricing
                {marketPricingCached && (
                  <Badge variant="secondary" className="ml-2 text-xs">Cached</Badge>
                )}
              </CardTitle>
              <Button
                variant="outline"
                size="sm"
                onClick={() => fetchMarketPricing(true)}
                disabled={marketPricingLoading}
                data-testid="refresh-market-pricing"
              >
                <RefreshCw className={`h-4 w-4 mr-1 ${marketPricingLoading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {marketPricingLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 2 }).map((_, i) => (
                  <Skeleton key={i} className="h-24 w-full" />
                ))}
              </div>
            ) : marketPricing.length > 0 ? (
              <div className="space-y-4">
                {marketPricing.map((result, idx) => (
                  <div key={idx} className="border rounded-md p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div>
                        <h4 className="font-semibold">{result.distributor}</h4>
                        <p className="text-sm text-muted-foreground font-mono">{result.distributor_pn}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-sm">
                          Stock: <span className={`font-semibold ${result.stock_qty > 0 ? 'text-green-600' : 'text-red-600'}`}>
                            {result.stock_qty.toLocaleString()}
                          </span>
                        </p>
                        <p className="text-sm text-muted-foreground">
                          Lead time: {result.lead_time_days} days
                        </p>
                      </div>
                    </div>
                    {result.price_breaks && result.price_breaks.length > 0 && (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Qty</TableHead>
                            <TableHead className="text-right">Unit Price</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {result.price_breaks.map((pb, pbIdx) => (
                            <TableRow key={pbIdx}>
                              <TableCell>{pb.qty.toLocaleString()}+</TableCell>
                              <TableCell className="text-right font-mono">
                                ${pb.unit_price.toFixed(4)}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    )}
                    {result.product_url && (
                      <a
                        href={result.product_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-xs text-blue-600 hover:underline mt-2 inline-block"
                      >
                        View on {result.distributor} →
                      </a>
                    )}
                  </div>
                ))}
              </div>
            ) : marketPricingError ? (
              <div className="text-center py-8 text-muted-foreground">
                <Store className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>{marketPricingError}</p>
                {marketPricingUnconfigured.length > 0 && (
                  <p className="mt-2 text-sm">
                    <a href="/distributor-settings" className="text-blue-600 hover:underline">
                      Configure API keys →
                    </a>
                  </p>
                )}
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                <Store className="h-8 w-8 mx-auto mb-2 opacity-50" />
                <p>No market pricing available</p>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Where Used */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center">
            <GitBranch className="h-5 w-5 mr-2" />
            Where Used
          </CardTitle>
        </CardHeader>
        <CardContent>
          {whereUsedLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-8 w-full" />
              ))}
            </div>
          ) : whereUsed.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Assembly</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead className="text-right">Qty Per</TableHead>
                  <TableHead>Reference</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {whereUsed.map((entry, index) => (
                  <TableRow key={index}>
                    <TableCell>
                      <Link
                        to={`/parts/${encodeURIComponent(entry.assembly_ipn)}`}
                        className="font-mono text-blue-600 hover:underline"
                      >
                        {entry.assembly_ipn}
                      </Link>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {entry.description || "—"}
                    </TableCell>
                    <TableCell className="text-right">{entry.qty}</TableCell>
                    <TableCell>
                      {entry.ref ? (
                        <Badge variant="secondary" className="text-xs">
                          {entry.ref}
                        </Badge>
                      ) : (
                        "—"
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <GitBranch className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p>This part is not used in any assemblies</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
export default PartDetail;
