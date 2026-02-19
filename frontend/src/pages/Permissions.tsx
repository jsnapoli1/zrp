import { useState, useEffect, useCallback } from "react";
import { getPermissions, getPermissionModules, setRolePermissions, type ModuleInfo } from "../lib/api";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Checkbox } from "../components/ui/checkbox";
import { toast } from "sonner";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import { Shield, Save } from "lucide-react";

const MODULE_LABELS: Record<string, string> = {
  parts: "Parts",
  ecos: "ECOs",
  documents: "Documents",
  inventory: "Inventory",
  vendors: "Vendors",
  purchase_orders: "Purchase Orders",
  work_orders: "Work Orders",
  ncrs: "NCRs",
  rmas: "RMAs",
  quotes: "Quotes",
  pricing: "Pricing",
  devices: "Devices",
  firmware: "Firmware",
  shipments: "Shipments",
  field_reports: "Field Reports",
  rfqs: "RFQs",
  reports: "Reports",
  testing: "Testing",
  admin: "Admin (Users, API Keys, Settings)",
};

const ACTION_LABELS: Record<string, string> = {
  view: "View",
  create: "Create",
  edit: "Edit",
  delete: "Delete",
  approve: "Approve",
};

const ROLES = ["admin", "user", "readonly"];

export default function Permissions() {
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [selectedRole, setSelectedRole] = useState("admin");
  const [permSet, setPermSet] = useState<Set<string>>(new Set());
  const [originalPermSet, setOriginalPermSet] = useState<Set<string>>(new Set());
  const [saving, setSaving] = useState(false);

  const loadPermissions = useCallback(async (role: string) => {
    try {
      const [mods, perms] = await Promise.all([
        getPermissionModules(),
        getPermissions(role),
      ]);
      setModules(mods);
      const set = new Set(perms.map((p) => `${p.module}:${p.action}`));
      setPermSet(set);
      setOriginalPermSet(new Set(set));
    } catch (err) {
      toast.error("Failed to load permissions");
    }
  }, []);

  useEffect(() => {
    loadPermissions(selectedRole);
  }, [selectedRole, loadPermissions]);

  const toggle = (module: string, action: string) => {
    const key = `${module}:${action}`;
    setPermSet((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const toggleAllForModule = (module: string, actions: string[]) => {
    const allChecked = actions.every((a) => permSet.has(`${module}:${a}`));
    setPermSet((prev) => {
      const next = new Set(prev);
      for (const a of actions) {
        if (allChecked) {
          next.delete(`${module}:${a}`);
        } else {
          next.add(`${module}:${a}`);
        }
      }
      return next;
    });
  };

  const hasChanges = (() => {
    if (permSet.size !== originalPermSet.size) return true;
    for (const k of permSet) {
      if (!originalPermSet.has(k)) return true;
    }
    return false;
  })();

  const handleSave = async () => {
    setSaving(true);
    try {
      const permissions = Array.from(permSet).map((key) => {
        const [module, action] = key.split(":");
        return { module, action };
      });
      await setRolePermissions(selectedRole, permissions);
      setOriginalPermSet(new Set(permSet));
      toast.success(`Permissions updated for ${selectedRole}`);
    } catch (err: any) {
      toast.error(err.message || "Failed to save permissions");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Shield className="h-6 w-6" />
            Permissions
          </h1>
          <p className="text-muted-foreground">
            Configure module-level permissions for each role
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Select value={selectedRole} onValueChange={setSelectedRole}>
            <SelectTrigger className="w-[180px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {ROLES.map((r) => (
                <SelectItem key={r} value={r}>
                  <Badge variant="outline" className="mr-2">{r}</Badge>
                  {r.charAt(0).toUpperCase() + r.slice(1)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button onClick={handleSave} disabled={!hasChanges || saving}>
            <Save className="h-4 w-4 mr-2" />
            {saving ? "Saving..." : "Save Changes"}
          </Button>
        </div>
      </div>

      <div className="grid gap-4">
        {modules.map((mod) => {
          const allChecked = mod.actions.every((a) =>
            permSet.has(`${mod.module}:${a}`)
          );
          const someChecked =
            !allChecked &&
            mod.actions.some((a) => permSet.has(`${mod.module}:${a}`));

          return (
            <Card key={mod.module}>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="text-base">
                      {MODULE_LABELS[mod.module] || mod.module}
                    </CardTitle>
                    <CardDescription className="text-xs">
                      {mod.module}
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2">
                    <Checkbox
                      checked={allChecked ? true : someChecked ? "indeterminate" : false}
                      onCheckedChange={() => toggleAllForModule(mod.module, mod.actions)}
                    />
                    <span className="text-xs text-muted-foreground">All</span>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-4">
                  {mod.actions.map((action) => {
                    const checked = permSet.has(`${mod.module}:${action}`);
                    return (
                      <label
                        key={action}
                        className="flex items-center gap-2 cursor-pointer"
                      >
                        <Checkbox
                          checked={checked}
                          onCheckedChange={() => toggle(mod.module, action)}
                        />
                        <span className="text-sm">
                          {ACTION_LABELS[action] || action}
                        </span>
                      </label>
                    );
                  })}
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
