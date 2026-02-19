import type { ReactNode } from "react";
import { useIsMobile } from "../hooks/use-mobile";
import { Card, CardContent } from "./ui/card";

interface ResponsiveTableWrapperProps<T> {
  data: T[];
  children: ReactNode;
  renderMobileCard?: (item: T, index: number) => ReactNode;
}

/**
 * Responsive table wrapper that switches between table and card view on mobile
 * Usage:
 * <ResponsiveTableWrapper
 *   data={items}
 *   renderMobileCard={(item) => <MobileCard item={item} />}
 * >
 *   <ConfigurableTable ... />
 * </ResponsiveTableWrapper>
 */
export function ResponsiveTableWrapper<T>({
  data,
  children,
  renderMobileCard,
}: ResponsiveTableWrapperProps<T>) {
  const isMobile = useIsMobile();

  // If no mobile card renderer provided, always show table (with horizontal scroll)
  if (!renderMobileCard) {
    return <div className="table-scroll-indicator">{children}</div>;
  }

  // On mobile, render as cards if custom renderer provided
  if (isMobile) {
    return (
      <div className="space-y-3">
        {data.map((item, index) => (
          <div key={index}>{renderMobileCard(item, index)}</div>
        ))}
        {data.length === 0 && (
          <Card>
            <CardContent className="py-8 text-center text-muted-foreground">
              No data found
            </CardContent>
          </Card>
        )}
      </div>
    );
  }

  // Desktop: render table normally
  return <div className="table-scroll-indicator">{children}</div>;
}
