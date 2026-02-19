import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from "react";
import { getMyPermissions, type Permission } from "../lib/api";

interface PermissionsContextType {
  permissions: Permission[];
  loading: boolean;
  hasPermission: (module: string, action: string) => boolean;
  canView: (module: string) => boolean;
  canCreate: (module: string) => boolean;
  canEdit: (module: string) => boolean;
  canDelete: (module: string) => boolean;
  canApprove: (module: string) => boolean;
  refresh: () => Promise<void>;
}

const PermissionsContext = createContext<PermissionsContextType>({
  permissions: [],
  loading: true,
  hasPermission: () => false,
  canView: () => false,
  canCreate: () => false,
  canEdit: () => false,
  canDelete: () => false,
  canApprove: () => false,
  refresh: async () => {},
});

export function usePermissions() {
  return useContext(PermissionsContext);
}

// Quick lookup set for O(1) permission checks
function buildLookup(perms: Permission[]): Set<string> {
  const set = new Set<string>();
  for (const p of perms) {
    set.add(`${p.module}:${p.action}`);
  }
  return set;
}

export function PermissionsProvider({ children }: { children: ReactNode }) {
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [lookup, setLookup] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const perms = await getMyPermissions();
      setPermissions(perms);
      setLookup(buildLookup(perms));
    } catch {
      // Not logged in or error — empty permissions
      setPermissions([]);
      setLookup(new Set());
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const hasPermission = useCallback(
    (module: string, action: string) => lookup.has(`${module}:${action}`),
    [lookup]
  );

  const canView = useCallback((module: string) => lookup.has(`${module}:view`), [lookup]);
  const canCreate = useCallback((module: string) => lookup.has(`${module}:create`), [lookup]);
  const canEdit = useCallback((module: string) => lookup.has(`${module}:edit`), [lookup]);
  const canDelete = useCallback((module: string) => lookup.has(`${module}:delete`), [lookup]);
  const canApprove = useCallback((module: string) => lookup.has(`${module}:approve`), [lookup]);

  return (
    <PermissionsContext.Provider
      value={{ permissions, loading, hasPermission, canView, canCreate, canEdit, canDelete, canApprove, refresh }}
    >
      {children}
    </PermissionsContext.Provider>
  );
}
