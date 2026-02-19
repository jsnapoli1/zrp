import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Badge } from "./ui/badge";
import { DollarSign, Archive } from "lucide-react";
import type { Part } from "../lib/api";

interface PartMobileCardProps {
  part: Part & {
    category?: string;
    description?: string;
    cost?: number;
    stock?: number;
    status?: string;
  };
  onClick?: () => void;
}

/**
 * Mobile-optimized card view for parts list
 * Shows key information in a touch-friendly layout
 */
export function PartMobileCard({ part, onClick }: PartMobileCardProps) {
  return (
    <Card 
      className="cursor-pointer hover:shadow-md transition-shadow"
      onClick={onClick}
    >
      <CardHeader className="pb-3">
        <div className="flex justify-between items-start gap-2">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-base font-semibold truncate">
              {part.ipn}
            </CardTitle>
            {part.category && (
              <p className="text-sm text-muted-foreground mt-0.5">
                {part.category}
              </p>
            )}
          </div>
          {part.status && (
            <Badge variant={part.status === 'active' ? 'default' : 'secondary'}>
              {part.status}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {part.description && (
          <p className="text-sm text-muted-foreground line-clamp-2">
            {part.description}
          </p>
        )}
        
        <div className="grid grid-cols-2 gap-3 pt-2 border-t">
          {/* Stock Level */}
          {typeof part.stock === 'number' && (
            <div className="flex items-center gap-2">
              <Archive className="h-4 w-4 text-muted-foreground" />
              <div className="flex flex-col">
                <span className="text-xs text-muted-foreground">Stock</span>
                <span className="text-sm font-medium">{part.stock}</span>
              </div>
            </div>
          )}

          {/* Cost */}
          {typeof part.cost === 'number' && (
            <div className="flex items-center gap-2">
              <DollarSign className="h-4 w-4 text-muted-foreground" />
              <div className="flex flex-col">
                <span className="text-xs text-muted-foreground">Cost</span>
                <span className="text-sm font-medium">
                  ${part.cost.toFixed(2)}
                </span>
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
