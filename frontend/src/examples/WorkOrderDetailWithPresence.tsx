/**
 * Example: Work Order Detail Page with Real-Time Collaboration
 * 
 * This example shows how to integrate:
 * - User presence indicators
 * - Real-time updates when other users make changes
 * - Automatic data refresh
 */

import { useState, useEffect } from "react";
import { usePresence, useResourceUpdates } from "../hooks/usePresence";
import { PresenceIndicator } from "../components/PresenceIndicator";
import { Card, CardHeader, CardTitle, CardContent } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Loader2 } from "lucide-react";

interface WorkOrder {
  id: string;
  assembly_ipn: string;
  qty: number;
  status: string;
  priority: string;
  notes: string;
  created_at: string;
}

export function WorkOrderDetailWithPresence({ id }: { id: string }) {
  const [workOrder, setWorkOrder] = useState<WorkOrder | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

  // Fetch work order data
  const fetchWorkOrder = async () => {
    try {
      const res = await fetch(`/api/v1/work-orders/${id}`);
      if (res.ok) {
        const data = await res.json();
        setWorkOrder(data);
        setLastUpdate(new Date());
      }
    } catch (error) {
      console.error("Failed to fetch work order:", error);
    } finally {
      setLoading(false);
    }
  };

  // Initial fetch
  useEffect(() => {
    fetchWorkOrder();
  }, [id]);

  // Subscribe to real-time updates
  useResourceUpdates("work_order", ({ action, id: updatedId }) => {
    // If this work order was updated, refresh data
    if (String(updatedId) === String(id)) {
      console.log(`Work order ${id} was ${action}ed by another user, refreshing...`);
      fetchWorkOrder();
    }
  });

  // Report our presence (we're viewing this work order)
  // This will show our avatar to other users viewing the same WO
  usePresence("work_order", id, "viewing");

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  if (!workOrder) {
    return <div>Work order not found</div>;
  }

  return (
    <div className="space-y-6">
      {/* Header with presence indicator */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Work Order {workOrder.id}</h1>
          <p className="text-muted-foreground">{workOrder.assembly_ipn}</p>
        </div>
        
        {/* Show who else is viewing this work order */}
        <PresenceIndicator
          resourceType="work_order"
          resourceId={id}
          action="viewing"
        />
      </div>

      {/* Real-time update indicator */}
      {lastUpdate && (
        <Badge variant="outline" className="text-xs">
          Last updated: {lastUpdate.toLocaleTimeString()}
        </Badge>
      )}

      {/* Work order details */}
      <Card>
        <CardHeader>
          <CardTitle>Details</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Assembly IPN</label>
              <p>{workOrder.assembly_ipn}</p>
            </div>
            <div>
              <label className="text-sm font-medium">Quantity</label>
              <p>{workOrder.qty}</p>
            </div>
            <div>
              <label className="text-sm font-medium">Status</label>
              <Badge>{workOrder.status}</Badge>
            </div>
            <div>
              <label className="text-sm font-medium">Priority</label>
              <Badge variant="outline">{workOrder.priority}</Badge>
            </div>
          </div>
          
          {workOrder.notes && (
            <div>
              <label className="text-sm font-medium">Notes</label>
              <p className="text-sm text-muted-foreground">{workOrder.notes}</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * Example: Work Order Editor with "Editing" Presence
 * 
 * When a user opens the edit form, their presence shows as "editing"
 * to warn other users that changes are in progress.
 */
export function WorkOrderEditor({ id }: { id: string }) {
  // Report that we're EDITING (not just viewing)
  // This shows an "editing" indicator to other users
  usePresence("work_order", id, "editing");

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2>Edit Work Order {id}</h2>
        
        {/* Show who else is viewing/editing */}
        <PresenceIndicator
          resourceType="work_order"
          resourceId={id}
          action="editing"
        />
      </div>

      {/* Edit form */}
      <form>
        {/* ... form fields ... */}
      </form>
    </div>
  );
}

/**
 * Example: Work Orders List with Real-Time Updates
 * 
 * The list automatically refreshes when work orders are created, updated, or deleted
 */
export function WorkOrdersList() {
  const [workOrders, setWorkOrders] = useState<WorkOrder[]>([]);
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  const fetchWorkOrders = async () => {
    const res = await fetch("/api/v1/work-orders");
    if (res.ok) {
      const data = await res.json();
      setWorkOrders(data);
      setLastRefresh(new Date());
    }
  };

  useEffect(() => {
    fetchWorkOrders();
  }, []);

  // Auto-refresh when any work order is created/updated/deleted
  useResourceUpdates("work_order", ({ action }) => {
    console.log(`Work order ${action}d, refreshing list...`);
    fetchWorkOrders();
  });

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1>Work Orders</h1>
        <Badge variant="outline" className="text-xs">
          Last refresh: {lastRefresh.toLocaleTimeString()}
        </Badge>
      </div>

      <div className="space-y-2">
        {workOrders.map((wo) => (
          <Card key={wo.id}>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>{wo.id}</CardTitle>
                <p className="text-sm text-muted-foreground">
                  {wo.assembly_ipn}
                </p>
              </div>
              <Badge>{wo.status}</Badge>
            </CardHeader>
          </Card>
        ))}
      </div>
    </div>
  );
}
