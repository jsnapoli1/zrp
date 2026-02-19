import { useCallback, useEffect } from 'react';
import { toast } from 'sonner';
import { api } from '../lib/api';

interface UndoOptions {
  entityType: string;
  entityId: string;
  action: string;
  description?: string;
}

/**
 * Hook for undo functionality on destructive actions.
 * Shows a toast with an "Undo" button for 5 seconds after a destructive action.
 * Also registers Ctrl+Z keyboard shortcut to undo the last change.
 */
export function useUndo() {
  const showUndoToast = useCallback(
    (undoId: number, options: UndoOptions) => {
      const desc = options.description || `${options.action} ${options.entityType} ${options.entityId}`;
      toast(desc, {
        duration: 5000,
        action: {
          label: 'Undo',
          onClick: async () => {
            try {
              await api.performUndo(undoId);
              toast.success(`Restored ${options.entityType} ${options.entityId}`);
            } catch {
              toast.error('Undo failed');
            }
          },
        },
      });
    },
    []
  );

  /**
   * Shows a toast for change_history based undo with redo support.
   */
  const showChangeUndoToast = useCallback(
    (changeId: number, tableName: string, recordId: string, operation: string) => {
      const desc = `Undone: ${operation} ${tableName} ${recordId}`;
      toast(desc, {
        duration: 5000,
        action: {
          label: 'Redo',
          onClick: async () => {
            try {
              const result = await api.undoChange(changeId);
              toast.success(`Redone: ${result.operation} ${result.table_name} ${result.record_id}`);
            } catch {
              toast.error('Redo failed');
            }
          },
        },
      });
    },
    []
  );

  /**
   * Wraps a destructive API call. If the response contains undo_id,
   * shows a toast with an Undo button.
   */
  const withUndo = useCallback(
    async <T extends Record<string, unknown>>(
      apiCall: () => Promise<T>,
      options: Omit<UndoOptions, 'action'> & { action?: string }
    ): Promise<T> => {
      const result = await apiCall();
      const undoId = (result as Record<string, unknown>)?.undo_id as number | undefined;
      if (undoId) {
        showUndoToast(undoId, {
          action: options.action || 'Deleted',
          ...options,
        });
      }
      return result;
    },
    [showUndoToast]
  );

  return { showUndoToast, showChangeUndoToast, withUndo };
}

/**
 * Hook that registers Ctrl+Z keyboard shortcut to undo the last change.
 * Should be used once in the app layout.
 */
export function useGlobalUndo() {
  useEffect(() => {
    const handleKeyDown = async (e: KeyboardEvent) => {
      // Ctrl+Z or Cmd+Z, but not when typing in an input
      if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
        const target = e.target as HTMLElement;
        const tagName = target.tagName.toLowerCase();
        if (tagName === 'input' || tagName === 'textarea' || target.isContentEditable) {
          return; // Don't interfere with native undo in form fields
        }

        e.preventDefault();
        try {
          const changes = await api.getRecentChanges(1);
          if (changes.length === 0) {
            toast.info('Nothing to undo');
            return;
          }
          const lastChange = changes[0];
          if (lastChange.undone === 1) {
            toast.info('Last change already undone');
            return;
          }
          const result = await api.undoChange(lastChange.id);
          const desc = `Undone: ${lastChange.operation} ${lastChange.table_name} ${lastChange.record_id}`;
          toast(desc, {
            duration: 5000,
            action: {
              label: 'Redo',
              onClick: async () => {
                try {
                  await api.undoChange(result.redo_id);
                  toast.success('Redone successfully');
                } catch {
                  toast.error('Redo failed');
                }
              },
            },
          });
        } catch {
          toast.error('Undo failed');
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);
}
