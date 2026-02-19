import { useState } from "react";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { Separator } from "./ui/separator";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "./ui/alert-dialog";
import { Progress } from "./ui/progress";
import { X, AlertTriangle } from "lucide-react";
import { toast } from "sonner";

export interface BatchAction {
  id: string;
  label: string;
  icon?: React.ReactNode;
  variant?: "default" | "destructive" | "outline";
  requiresConfirmation?: boolean;
  confirmationTitle?: string;
  confirmationMessage?: string;
  onExecute: (selectedIds: string[]) => Promise<{ success: number; failed: number; errors?: string[] }>;
}

interface BatchActionBarProps {
  selectedCount: number;
  totalCount: number;
  actions: BatchAction[];
  onClearSelection: () => void;
  selectedIds: string[];
}

export function BatchActionBar({
  selectedCount,
  totalCount,
  actions,
  onClearSelection,
  selectedIds,
}: BatchActionBarProps) {
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);
  const [pendingAction, setPendingAction] = useState<BatchAction | null>(null);
  const [executing, setExecuting] = useState(false);
  const [progress, setProgress] = useState(0);

  if (selectedCount === 0) return null;

  const handleActionClick = (action: BatchAction) => {
    if (action.requiresConfirmation) {
      setPendingAction(action);
      setShowConfirmDialog(true);
    } else {
      executeAction(action);
    }
  };

  const executeAction = async (action: BatchAction) => {
    setExecuting(true);
    setProgress(0);
    
    try {
      // Simulate progress for user feedback
      const progressInterval = setInterval(() => {
        setProgress((prev) => Math.min(prev + 10, 90));
      }, 100);

      const result = await action.onExecute(selectedIds);
      
      clearInterval(progressInterval);
      setProgress(100);

      if (result.success > 0) {
        toast.success(
          `${action.label}: ${result.success} item${result.success !== 1 ? "s" : ""} processed successfully`
        );
      }
      if (result.failed > 0) {
        toast.error(
          `${action.label}: ${result.failed} item${result.failed !== 1 ? "s" : ""} failed`,
          {
            description: result.errors?.slice(0, 3).join("\n"),
          }
        );
      }

      // Clear selection on success
      if (result.failed === 0) {
        onClearSelection();
      }
    } catch (error) {
      toast.error(`${action.label} failed: ${error instanceof Error ? error.message : "Unknown error"}`);
    } finally {
      setExecuting(false);
      setProgress(0);
      setPendingAction(null);
      setShowConfirmDialog(false);
    }
  };

  const handleConfirm = () => {
    if (pendingAction) {
      executeAction(pendingAction);
    }
  };

  return (
    <>
      <div className="sticky top-0 z-10 bg-background border-b shadow-md">
        <div className="flex items-center gap-4 p-4">
          <div className="flex items-center gap-2">
            <Badge variant="secondary" className="text-base px-3 py-1">
              {selectedCount} of {totalCount} selected
            </Badge>
            <Button
              variant="ghost"
              size="sm"
              onClick={onClearSelection}
              disabled={executing}
            >
              <X className="h-4 w-4 mr-1" />
              Clear
            </Button>
          </div>

          <Separator orientation="vertical" className="h-8" />

          <div className="flex items-center gap-2 flex-wrap">
            {actions.map((action) => (
              <Button
                key={action.id}
                variant={action.variant || "default"}
                size="sm"
                onClick={() => handleActionClick(action)}
                disabled={executing}
              >
                {action.icon && <span className="mr-1">{action.icon}</span>}
                {action.label}
              </Button>
            ))}
          </div>
        </div>

        {executing && (
          <div className="px-4 pb-3">
            <Progress value={progress} className="h-2" />
            <p className="text-xs text-muted-foreground mt-1">
              Processing {selectedCount} item{selectedCount !== 1 ? "s" : ""}...
            </p>
          </div>
        )}
      </div>

      <AlertDialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              {pendingAction?.confirmationTitle || "Confirm Action"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {pendingAction?.confirmationMessage ||
                `Are you sure you want to ${pendingAction?.label.toLowerCase()} ${selectedCount} item${selectedCount !== 1 ? "s" : ""}? This action cannot be undone.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirm}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Confirm
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
