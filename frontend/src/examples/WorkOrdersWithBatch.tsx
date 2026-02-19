import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Wrench, Plus, CheckCircle, XCircle, Trash2, Download } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { api, type WorkOrder } from "../lib/api";
import { BatchSelectionProvider, useBatchSelection } from "../contexts/BatchSelectionContext";
import { BatchActionBar, type BatchAction } from "../components/BatchActionBar";
import { BatchCheckbox, MasterBatchCheckbox } from "../components/BatchCheckbox";
import { BulkEditDialog, type BulkEditField } from "../components/BulkEditDialog";
import { toast } from "sonner";

function WorkOrdersContent() {
  const [workOrders, setWorkOrders] = useState<WorkOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [bulkEditOpen, setBulkEditOpen] = useState(false);
  const { selectedItems, selectedCount, clearSelection } = useBatchSelection();

  useEffect(() => {
    fetchWorkOrders();
  }, []);

  const fetchWorkOrders = async () => {
    try {
      setLoading(true);
      const data = await api.getWorkOrders();
      setWorkOrders(data);
    } catch (error) {
      toast.error("Failed to fetch work orders");
      console.error("Failed to fetch work orders:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleBatchComplete = async (ids: string[]) => {
    const result = await api.batchWorkOrders(ids, 'complete');
    await fetchWorkOrders();
    return result;
  };

  const handleBatchCancel = async (ids: string[]) => {
    const result = await api.batchWorkOrders(ids, 'cancel');
    await fetchWorkOrders();
    return result;
  };

  const handleBatchDelete = async (ids: string[]) => {
    const result = await api.batchWorkOrders(ids, 'delete');
    await fetchWorkOrders();
    return result;
  };

  const handleBulkEdit = async (updates: Record<string, string>) => {
    const ids = Array.from(selectedItems);
    const result = await api.bulkUpdateWorkOrders(ids, updates);
    await fetchWorkOrders();
    clearSelection();
    return result;
  };

  const batchActions: BatchAction[] = [
    {
      id: 'complete',
      label: 'Mark Complete',
      icon: <CheckCircle className="h-4 w-4" />,
      variant: 'default',
      onExecute: handleBatchComplete,
    },
    {
      id: 'cancel',
      label: 'Cancel',
      icon: <XCircle className="h-4 w-4" />,
      variant: 'outline',
      requiresConfirmation: true,
      confirmationTitle: 'Cancel Work Orders?',
      confirmationMessage: 'This will cancel all selected work orders.',
      onExecute: handleBatchCancel,
    },
    {
      id: 'edit',
      label: 'Bulk Edit',
      variant: 'outline',
      onExecute: async () => {
        setBulkEditOpen(true);
        return { success: 0, failed: 0 };
      },
    },
    {
      id: 'delete',
      label: 'Delete',
      icon: <Trash2 className="h-4 w-4" />,
      variant: 'destructive',
      requiresConfirmation: true,
      confirmationTitle: 'Delete Work Orders?',
      confirmationMessage: 'This action cannot be undone. All selected work orders will be permanently deleted.',
      onExecute: handleBatchDelete,
    },
  ];

  const bulkEditFields: BulkEditField[] = [
    {
      key: 'status',
      label: 'Status',
      type: 'select',
      options: [
        { value: 'open', label: 'Open' },
        { value: 'in_progress', label: 'In Progress' },
        { value: 'completed', label: 'Completed' },
        { value: 'cancelled', label: 'Cancelled' },
      ],
    },
    {
      key: 'priority',
      label: 'Priority',
      type: 'select',
      options: [
        { value: 'low', label: 'Low' },
        { value: 'normal', label: 'Normal' },
        { value: 'high', label: 'High' },
        { value: 'urgent', label: 'Urgent' },
      ],
    },
  ];

  const getStatusBadge = (status: string) => {
    const variants = {
      open: 'secondary',
      in_progress: 'default',
      completed: 'default',
      cancelled: 'destructive',
    } as const;
    return <Badge variant={variants[status as keyof typeof variants] || 'secondary'}>{status}</Badge>;
  };

  return (
    <div className="space-y-6">
      <BatchActionBar
        selectedCount={selectedCount}
        totalCount={workOrders.length}
        actions={batchActions}
        onClearSelection={clearSelection}
        selectedIds={Array.from(selectedItems)}
      />

      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Wrench className="h-8 w-8" />
            Work Orders
          </h1>
          <p className="text-muted-foreground mt-1">
            Manage manufacturing work orders
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline">
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            New Work Order
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>All Work Orders</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-center py-8">Loading...</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b">
                    <th className="text-left p-2">
                      <MasterBatchCheckbox allIds={workOrders.map(wo => wo.id)} />
                    </th>
                    <th className="text-left p-2">ID</th>
                    <th className="text-left p-2">Assembly</th>
                    <th className="text-left p-2">Qty</th>
                    <th className="text-left p-2">Status</th>
                    <th className="text-left p-2">Priority</th>
                    <th className="text-left p-2">Created</th>
                    <th className="text-left p-2">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {workOrders.map((wo) => (
                    <tr key={wo.id} className="border-b hover:bg-muted/50">
                      <td className="p-2">
                        <BatchCheckbox id={wo.id} />
                      </td>
                      <td className="p-2">
                        <Link to={`/workorders/${wo.id}`} className="text-primary hover:underline">
                          {wo.id}
                        </Link>
                      </td>
                      <td className="p-2">{wo.assembly_ipn}</td>
                      <td className="p-2">{wo.qty}</td>
                      <td className="p-2">{getStatusBadge(wo.status)}</td>
                      <td className="p-2">{wo.priority}</td>
                      <td className="p-2">{new Date(wo.created_at).toLocaleDateString()}</td>
                      <td className="p-2">
                        <Link to={`/workorders/${wo.id}`}>
                          <Button variant="ghost" size="sm">View</Button>
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <BulkEditDialog
        open={bulkEditOpen}
        onOpenChange={setBulkEditOpen}
        fields={bulkEditFields}
        selectedCount={selectedCount}
        onSubmit={handleBulkEdit}
        title="Bulk Edit Work Orders"
      />
    </div>
  );
}

export default function WorkOrdersWithBatch() {
  return (
    <BatchSelectionProvider>
      <WorkOrdersContent />
    </BatchSelectionProvider>
  );
}
