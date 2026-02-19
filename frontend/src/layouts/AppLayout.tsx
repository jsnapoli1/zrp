import { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { useGlobalUndo } from "../hooks/useUndo";
import { useWS } from "../contexts/WebSocketContext";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { Outlet, Link, useLocation, useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import {
  AlertTriangle,
  BarChart3,
  Building2,
  Calendar,
  CheckCircle2,
  ClipboardList,
  Cog,
  FileText,
  Home,
  Info,
  Package,
  Search,
  Settings,
  ShoppingCart,
  TrendingUp,
  Users,
  Wrench,
  Moon,
  Sun,
  User,
  Bell,
  ScanLine,
  RotateCcw,
  Clock,
  XCircle,
  ShieldCheck,
} from "lucide-react";

import { usePermissions } from "../contexts/PermissionsContext";
import { Button } from "../components/ui/button";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from "../components/ui/sidebar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../components/ui/dropdown-menu";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "../components/ui/command";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Badge } from "../components/ui/badge";

const navigationItems = [
  {
    title: "Engineering",
    items: [
      { title: "Dashboard", url: "/", icon: Home },
      { title: "Parts", url: "/parts", icon: Package },
      { title: "ECOs", url: "/ecos", icon: FileText },
      { title: "Documents", url: "/documents", icon: FileText },
      { title: "Testing", url: "/testing", icon: ClipboardList },
    ],
  },
  {
    title: "Supply Chain",
    items: [
      { title: "Vendors", url: "/vendors", icon: Building2 },
      { title: "Purchase Orders", url: "/purchase-orders", icon: ShoppingCart },
      { title: "RFQs", url: "/rfqs", icon: TrendingUp },
      { title: "Procurement", url: "/procurement", icon: TrendingUp },
      { title: "Shipments", url: "/shipments", icon: TrendingUp },
    ],
  },
  {
    title: "Manufacturing",
    items: [
      { title: "Work Orders", url: "/work-orders", icon: Wrench },
      { title: "Inventory", url: "/inventory", icon: Package },
      { title: "NCRs", url: "/ncrs", icon: ClipboardList },
      { title: "CAPAs", url: "/capas", icon: ShieldCheck },
      { title: "Scan", url: "/scan", icon: ScanLine },
    ],
  },
  {
    title: "Field & Service",
    items: [
      { title: "RMAs", url: "/rmas", icon: ClipboardList },
      { title: "Field Reports", url: "/field-reports", icon: FileText },
    ],
  },
  {
    title: "Sales",
    items: [
      { title: "Quotes", url: "/quotes", icon: FileText },
      { title: "Pricing", url: "/pricing", icon: TrendingUp },
    ],
  },
  {
    title: "Reports",
    items: [
      { title: "Analytics", url: "/reports", icon: BarChart3 },
      { title: "Calendar", url: "/calendar", icon: Calendar },
    ],
  },
  {
    title: "Admin",
    items: [
      { title: "Users", url: "/users", icon: Users, module: "admin" },
      { title: "Permissions", url: "/permissions", icon: Settings, module: "admin" },
      { title: "Settings", url: "/settings", icon: Settings, module: "admin" },
    ],
  },
];

// Module mapping for nav items (items without module are always shown)
const NAV_MODULE_MAP: Record<string, string> = {
  "/parts": "parts",
  "/ecos": "ecos",
  "/documents": "documents",
  "/testing": "testing",
  "/vendors": "vendors",
  "/purchase-orders": "purchase_orders",
  "/rfqs": "rfqs",
  "/procurement": "purchase_orders",
  "/shipments": "shipments",
  "/work-orders": "work_orders",
  "/inventory": "inventory",
  "/ncrs": "ncrs",
  "/capas": "ncrs",
  "/rmas": "rmas",
  "/field-reports": "field_reports",
  "/quotes": "quotes",
  "/pricing": "pricing",
  "/reports": "reports",
  "/devices": "devices",
  "/firmware": "firmware",
  "/users": "admin",
  "/permissions": "admin",
  "/settings": "admin",
};

interface Notification {
  id: string;
  title: string;
  severity: "info" | "warning" | "error" | "success";
  type: string;
  link: string;
  timestamp: string;
  read: boolean;
}

const severityConfig = {
  info: { icon: Info, color: "text-blue-500" },
  warning: { icon: AlertTriangle, color: "text-yellow-500" },
  error: { icon: XCircle, color: "text-red-500" },
  success: { icon: CheckCircle2, color: "text-green-500" },
};

const defaultNotifications: Notification[] = [
  { id: "1", title: "Low stock: Resistor 10kΩ (5 remaining)", severity: "warning", type: "inventory", link: "/inventory", timestamp: new Date(Date.now() - 3600000).toISOString(), read: false },
  { id: "2", title: "ECO-2024-042 approved", severity: "success", type: "eco", link: "/ecos", timestamp: new Date(Date.now() - 7200000).toISOString(), read: false },
  { id: "3", title: "Work Order WO-118 overdue", severity: "error", type: "work-order", link: "/work-orders", timestamp: new Date(Date.now() - 18000000).toISOString(), read: false },
];

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [darkMode, setDarkMode] = useState(false);
  const [notifications, setNotifications] = useState<Notification[]>(defaultNotifications);
  const [notifOpen, setNotifOpen] = useState(false);
  const notifRef = useRef<HTMLDivElement>(null);
  const [profileOpen, setProfileOpen] = useState(false);
  const [currentUser, setCurrentUser] = useState<{ id: number; username: string; display_name: string; role: string } | null>(null);
  const { canView } = usePermissions();
  const [pwForm, setPwForm] = useState({ current: "", new_pw: "", confirm: "" });
  const [pwError, setPwError] = useState("");
  const [pwSuccess, setPwSuccess] = useState("");
  const [pwLoading, setPwLoading] = useState(false);
  const { status: wsStatus } = useWS();
  useGlobalUndo();

  // Filter navigation items based on user permissions
  const filteredNav = useMemo(() => {
    return navigationItems
      .map((section) => ({
        ...section,
        items: section.items.filter((item) => {
          const mod = NAV_MODULE_MAP[item.url];
          // Items without a module mapping are always shown (Dashboard, Calendar, Scan)
          if (!mod) return true;
          return canView(mod);
        }),
      }))
      .filter((section) => section.items.length > 0);
  }, [canView]);

  useEffect(() => {
    api.getMe().then((res) => { if (res?.user) setCurrentUser(res.user); });
  }, []);

  const handleLogout = useCallback(async () => {
    try { await api.logout(); } catch { /* ignore */ }
    window.location.href = "/login";
  }, []);

  const handleChangePassword = useCallback(async () => {
    setPwError("");
    setPwSuccess("");
    if (pwForm.new_pw !== pwForm.confirm) { setPwError("Passwords do not match"); return; }
    if (!pwForm.current || !pwForm.new_pw) { setPwError("All fields are required"); return; }
    setPwLoading(true);
    try {
      await api.changePassword(pwForm.current, pwForm.new_pw);
      setPwSuccess("Password changed successfully");
      setPwForm({ current: "", new_pw: "", confirm: "" });
    } catch (e: any) {
      setPwError(e.message || "Failed to change password");
    } finally {
      setPwLoading(false);
    }
  }, [pwForm]);

  const unreadCount = notifications.filter((n) => !n.read).length;

  // Close dropdown on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (notifRef.current && !notifRef.current.contains(e.target as Node)) {
        setNotifOpen(false);
      }
    }
    if (notifOpen) document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [notifOpen]);

  const markAllRead = () => setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
  const dismissNotification = (id: string) => setNotifications((prev) => prev.filter((n) => n.id !== id));
  const handleNotifClick = (n: Notification) => {
    setNotifications((prev) => prev.map((x) => (x.id === n.id ? { ...x, read: true } : x)));
    setNotifOpen(false);
    navigate(n.link);
  };

  const toggleDarkMode = () => {
    setDarkMode(!darkMode);
    document.documentElement.classList.toggle("dark");
  };

  // Command dialog for search
  const handleSearch = () => {
    setOpen(true);
  };

  return (
    <SidebarProvider>
      <div className="flex min-h-screen w-full">
        {/* Sidebar */}
        <Sidebar>
          <SidebarHeader className="border-b p-4">
            <div className="flex items-center space-x-2">
              <Package className="h-6 w-6" />
              <span className="text-lg font-semibold">ZRP</span>
            </div>
          </SidebarHeader>

          <SidebarContent className="flex-1 overflow-y-auto">
            {filteredNav.map((section) => (
              <SidebarGroup key={section.title}>
                <SidebarGroupLabel>{section.title}</SidebarGroupLabel>
                <SidebarGroupContent>
                  <SidebarMenu>
                    {section.items.map((item) => {
                      const isActive = location.pathname === item.url;
                      return (
                        <SidebarMenuItem key={item.title}>
                          <SidebarMenuButton asChild isActive={isActive}>
                            <Link to={item.url}>
                              <item.icon className="h-4 w-4" />
                              <span>{item.title}</span>
                            </Link>
                          </SidebarMenuButton>
                        </SidebarMenuItem>
                      );
                    })}
                  </SidebarMenu>
                </SidebarGroupContent>
              </SidebarGroup>
            ))}
          </SidebarContent>

          <SidebarFooter className="border-t p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">v1.0.0</span>
              <Button variant="ghost" size="sm" onClick={toggleDarkMode}>
                {darkMode ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
              </Button>
            </div>
          </SidebarFooter>
        </Sidebar>

        {/* Main Content */}
        <div className="flex flex-1 flex-col">
          {/* Header */}
          <header className="flex h-16 items-center justify-between border-b bg-background px-6">
            <div className="flex items-center space-x-4">
              <SidebarTrigger />
              <Button
                variant="outline"
                className="relative h-9 w-9 p-0 xl:h-10 xl:w-60 xl:justify-start xl:px-3 xl:py-2"
                onClick={handleSearch}
              >
                <Search className="h-4 w-4 xl:mr-2" />
                <span className="hidden xl:inline-flex">Search...</span>
                <kbd className="pointer-events-none absolute right-1.5 top-2 hidden h-6 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium opacity-100 xl:flex">
                  <span className="text-xs">⌘</span>K
                </kbd>
              </Button>
            </div>

            <div className="flex items-center space-x-4">
              <div className="flex items-center gap-1.5" title={`WebSocket: ${wsStatus}`}>
                <span
                  className={`inline-block h-2 w-2 rounded-full ${
                    wsStatus === "connected"
                      ? "bg-green-500"
                      : wsStatus === "connecting"
                      ? "bg-yellow-500 animate-pulse"
                      : "bg-red-500"
                  }`}
                />
                <span className="text-xs text-muted-foreground hidden sm:inline">
                  {wsStatus === "connected" ? "Live" : wsStatus === "connecting" ? "Connecting" : "Offline"}
                </span>
              </div>

              <Button variant="ghost" size="icon" onClick={() => navigate("/undo-history")} title="Undo History">
                <Clock className="h-4 w-4" />
              </Button>

              <div className="relative" ref={notifRef}>
                <Button variant="ghost" size="icon" onClick={() => setNotifOpen((v) => !v)} aria-label="Notifications">
                  <Bell className="h-4 w-4" />
                  {unreadCount > 0 && (
                    <Badge variant="destructive" className="absolute -top-1 -right-1 h-5 w-5 rounded-full p-0 text-xs flex items-center justify-center">
                      {unreadCount}
                    </Badge>
                  )}
                </Button>
                {notifOpen && (
                  <div className="absolute right-0 top-full mt-2 w-80 rounded-md border bg-popover shadow-lg z-50">
                    <div className="flex items-center justify-between border-b px-4 py-2">
                      <span className="text-sm font-semibold">Notifications</span>
                      {unreadCount > 0 && (
                        <button className="text-xs text-muted-foreground hover:text-foreground" onClick={markAllRead}>
                          Mark all read
                        </button>
                      )}
                    </div>
                    <div className="max-h-72 overflow-y-auto">
                      {notifications.length === 0 ? (
                        <p className="px-4 py-6 text-center text-sm text-muted-foreground">No notifications</p>
                      ) : (
                        notifications.map((n) => {
                          const cfg = severityConfig[n.severity];
                          const Icon = cfg.icon;
                          return (
                            <div
                              key={n.id}
                              className={`flex items-start gap-3 px-4 py-3 hover:bg-accent cursor-pointer ${!n.read ? "bg-accent/40" : ""}`}
                              onClick={() => handleNotifClick(n)}
                              role="button"
                              tabIndex={0}
                            >
                              <Icon className={`h-4 w-4 mt-0.5 shrink-0 ${cfg.color}`} />
                              <div className="flex-1 min-w-0">
                                <p className="text-sm leading-tight truncate">{n.title}</p>
                                <p className="text-xs text-muted-foreground mt-0.5">{timeAgo(n.timestamp)}</p>
                              </div>
                              <button
                                className="text-muted-foreground hover:text-foreground shrink-0"
                                onClick={(e) => { e.stopPropagation(); dismissNotification(n.id); }}
                                aria-label="Dismiss"
                              >
                                <XCircle className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          );
                        })
                      )}
                    </div>
                  </div>
                )}
              </div>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="relative h-8 w-8 rounded-full">
                    <User className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-56" align="end" forceMount>
                  <DropdownMenuLabel className="font-normal">
                    <div className="flex flex-col space-y-1">
                      <p className="text-sm font-medium leading-none">Admin User</p>
                      <p className="text-xs leading-none text-muted-foreground">
                        admin@zrp.com
                      </p>
                    </div>
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={() => setProfileOpen(true)}>
                    <User className="mr-2 h-4 w-4" />
                    <span>Profile</span>
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => navigate("/notification-preferences")}>
                    <Bell className="mr-2 h-4 w-4" />
                    <span>Notification Preferences</span>
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => navigate("/settings")}>
                    <Cog className="mr-2 h-4 w-4" />
                    <span>Settings</span>
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => navigate("/undo-history")}>
                    <RotateCcw className="mr-2 h-4 w-4" />
                    <span>Undo History</span>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={handleLogout}>
                    <span>Log out</span>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </header>

          {/* Main Content Area */}
          <main className="flex-1 overflow-y-auto p-6">
            <ErrorBoundary>
              <Outlet />
            </ErrorBoundary>
          </main>
        </div>

        {/* Command Dialog for Search */}
        <CommandDialog open={open} onOpenChange={setOpen}>
          <CommandInput placeholder="Type a command or search..." />
          <CommandList>
            <CommandEmpty>No results found.</CommandEmpty>
            <CommandGroup heading="Navigation">
              {navigationItems.flatMap((section) =>
                section.items.map((item) => (
                  <CommandItem
                    key={item.url}
                    onSelect={() => {
                      setOpen(false);
                      navigate(item.url);
                    }}
                  >
                    <item.icon className="mr-2 h-4 w-4" />
                    <span>{item.title}</span>
                  </CommandItem>
                ))
              )}
            </CommandGroup>
            <CommandSeparator />
            <CommandGroup heading="Quick Actions">
              <CommandItem onSelect={() => { setOpen(false); navigate("/ecos"); }}>
                <FileText className="mr-2 h-4 w-4" />
                <span>Create ECO</span>
              </CommandItem>
              <CommandItem onSelect={() => { setOpen(false); navigate("/work-orders"); }}>
                <Wrench className="mr-2 h-4 w-4" />
                <span>New Work Order</span>
              </CommandItem>
              <CommandItem onSelect={() => { setOpen(false); navigate("/parts"); }}>
                <Package className="mr-2 h-4 w-4" />
                <span>Add Part</span>
              </CommandItem>
            </CommandGroup>
          </CommandList>
        </CommandDialog>

        {/* Profile Dialog */}
        <Dialog open={profileOpen} onOpenChange={(v) => { setProfileOpen(v); if (!v) { setPwError(""); setPwSuccess(""); setPwForm({ current: "", new_pw: "", confirm: "" }); } }}>
          <DialogContent className="sm:max-w-[425px]">
            <DialogHeader>
              <DialogTitle>Profile</DialogTitle>
              <DialogDescription>Your account information and password management.</DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="grid grid-cols-4 items-center gap-4">
                <Label className="text-right text-muted-foreground">Username</Label>
                <span className="col-span-3 text-sm">{currentUser?.username || "—"}</span>
              </div>
              <div className="grid grid-cols-4 items-center gap-4">
                <Label className="text-right text-muted-foreground">Display Name</Label>
                <span className="col-span-3 text-sm">{currentUser?.display_name || "—"}</span>
              </div>
              <div className="grid grid-cols-4 items-center gap-4">
                <Label className="text-right text-muted-foreground">Role</Label>
                <span className="col-span-3"><Badge variant="outline">{currentUser?.role || "—"}</Badge></span>
              </div>

              <div className="border-t pt-4">
                <h4 className="text-sm font-medium mb-3">Change Password</h4>
                <div className="space-y-3">
                  <div>
                    <Label htmlFor="pw-current">Current Password</Label>
                    <Input id="pw-current" type="password" value={pwForm.current} onChange={(e) => setPwForm((f) => ({ ...f, current: e.target.value }))} />
                  </div>
                  <div>
                    <Label htmlFor="pw-new">New Password</Label>
                    <Input id="pw-new" type="password" value={pwForm.new_pw} onChange={(e) => setPwForm((f) => ({ ...f, new_pw: e.target.value }))} />
                  </div>
                  <div>
                    <Label htmlFor="pw-confirm">Confirm New Password</Label>
                    <Input id="pw-confirm" type="password" value={pwForm.confirm} onChange={(e) => setPwForm((f) => ({ ...f, confirm: e.target.value }))} />
                  </div>
                  {pwError && <p className="text-sm text-destructive">{pwError}</p>}
                  {pwSuccess && <p className="text-sm text-green-600">{pwSuccess}</p>}
                  <Button onClick={handleChangePassword} disabled={pwLoading} className="w-full">
                    {pwLoading ? "Changing..." : "Change Password"}
                  </Button>
                </div>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </SidebarProvider>
  );
}