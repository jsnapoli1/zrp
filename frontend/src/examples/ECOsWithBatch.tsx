import { useEffect, useState } from "react";
import { FileText, Plus, CheckCircle, XCircle, Trash2 } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { api, type ECO } from "../lib/api";
import { BatchSelectionProvider, useBatchSelection } from "../contexts/BatchSelectionContext";
import { BatchActionBar, type BatchAction } from "../components/BatchActionBar";
import { BatchCheckbox, MasterBatchCheckbox } from "../components/BatchCheckbox";
import { BulkEditDialog, type BulkEditField } from "../components/BulkEditDialog";
import { toast } from "sonner";

function ECOsContent() {
  const [ecos, setECOs] = useState<ECO[]>([]);
  const [loading, setLoading] = useState(true);
  const [bulkEditOpen, setBulkEditOpen] = useState(false);
  const { selectedItems, selectedCount, clearSelection } = useBatchSelection();

  useEffect(() => {
    fetchECOs();
  }, []);

  const fetchECOs = async () => {
    try {
      setLoading(true);
      const data = await api.getECOs();
      setECOs(data);
    } catch (error) {
      toast.error("Failed to fetch ECOs");
      console.error("Failed to fetch ECOs:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleBatchApprove = async (ids: string[]) => {
    const result = await api.batchECOs(ids, 'approve');
    await fetchECOs();
    return result;
  };

  const handleBatchImplement = async (ids: string[]) => {
    const result = await api.batchECOs(ids, 'implement');
    await fetchECOs();
    return result;
  };

  const handleBatchReject = async (ids: string[]) => {
    const result = await api.batchECOs(ids, 'reject');
    await fetchECOs();
    return result;
  };

  const handleBatchDelete = async (ids: string[]) => {
    const result = await api.batchECOs(ids, 'delete');
    await fetchECOs();
    return result;
  };

  const handleBulkEdit = async (updates: Record<string, string>) => {
    const ids = Array.from(selectedItems);
    const result = await api.batchUpdateECOs(ids, updates);
    await fetchECOs();
    clearSelection();
    return result;
  };

  const batchActions: BatchAction[] = [
    {
      id: 'approve',
      label: 'Approve',
      icon: <CheckCircle className="h-4 w-4" />,
      variant: 'default',
      requiresConfirmation: true,
      confirmationTitle: 'Approve ECOs?',
      confirmationMessage: 'This will approve all selected ECOs and allow them to be implemented.',
      onExecute: handleBatchApprove,
    },
    {
      id: 'implement',
      label: 'Implement',
      variant: 'default',
      requiresConfirmation: true,
      confirmationTitle: 'Implement ECOs?',
      confirmationMessage: 'This will mark all selected ECOs as implemented. Make sure changes have been applied.',
      onExecute: handleBatchImplement,
    },
    {
      id: 'reject',
      label: 'Reject',
      icon: <XCircle className="h-4 w-4" />,
      variant: 'outline',
      requiresConfirmation: true,
      confirmationTitle: 'Reject ECOs?',
      confirmationMessage: 'This will reject all selected ECOs.',
      onExecute: handleBatchReject,
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
      confirmationTitle: 'Delete ECOs?',
      confirmationMessage: 'This action cannot be undone. All selected ECOs will be permanently deleted.',
      onExecute: handleBatchDelete,
    },
  ];

  const bulkEditFields: BulkEditField[] = [
    {
      key: 'status',
      label: 'Status',
      type: 'select',
      options: [
        { value: 'draft', label: 'Draft' },
        { value: 'open', label: 'Open' },
        { value: 'approved', label: 'Approved' },
        { value: 'implemented', label: 'Implemented' },
        { value: 'rejected', label: 'Rejected' },
      ],
    },
    {
      key: 'priority',
      label: 'Priority',
      type: 'select',
      options: [
        { value: 'low', label: 'Low' },
        { value: 'medium', label: 'Medium' },
        { value: 'high', label: 'High' },
        { value: 'critical', label: 'Critical' },
      ],
    },
  ];

  const getStatusBadge = (status: string) => {
    const variants = {
      draft: 'secondary',
      open: 'default',
      approved: 'default',
      implemented: 'outline',
      rejected: 'destructive',
    } as const;
    return <Badge variant={variants[status as keyof typeof variants] || 'secondary'}>{status}</Badge>;
  };

  return (
    <div className="space-y-6">
      <BatchActionBar
        selectedCount={selectedCount}
        totalCount={ecos.length}
        actions={batchActions}
        onClearSelection={clearSelection}
        selectedIds={Array.from(selectedItems)}
      />

      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <FileText className="h-8 w-8" />
            Engineering Change Orders
          </h1>
          <p className="text-muted-foreground mt-1">
            Track and manage engineering changes
          </p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          New ECO
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>All ECOs</CardTitle>
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
                      <MasterBatchCheckbox allIds={ecos.map(eco => eco.id)} />
                    </th>
                    <th className="text-left p-2">ID</th>
                    <th className="text-left p-2">Title</th>
                    <th className="text-left p-2">Status</th>
                    <th className="text-left p-2">Affected Parts</th>
                    <th className="text-left p-2">Created By</th>
                    <th className="text-left p-2">Created</th>
                  </tr>
                </thead>
                <tbody>
                  {ecos.map((eco) => (
                    <tr key={eco.id} className="border-b hover:bg-muted/50">
                      <td className="p-2">
                        <BatchCheckbox id={eco.id} />
                      </td>
                      <td className="p-2">
                        <a href={`/ecos/${eco.id}`} className="text-primary hover:underline">
                          {eco.id}
                        </a>
                      </td>
                      <td className="p-2">{eco.title}</td>
                      <td className="p-2">{getStatusBadge(eco.status)}</td>
                      <td className="p-2">{eco.affected_ipns || '-'}</td>
                      <td className="p-2">{eco.created_by}</td>
                      <td className="p-2">{new Date(eco.created_at).toLocaleDateString()}</td>
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
        title="Bulk Edit ECOs"
      />
    </div>
  );
}

export default function ECOsWithBatch() {
  return (
    <BatchSelectionProvider>
      <ECOsContent />
    </BatchSelectionProvider>
  );
}
