import { useState } from "react";
import { Outlet, Link, useLocation, useNavigate } from "react-router-dom";
import {
  BarChart3,
  Building2,
  Calendar,
  ClipboardList,
  Cog,
  FileText,
  Home,
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
} from "lucide-react";

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
      { title: "Procurement", url: "/procurement", icon: TrendingUp },
    ],
  },
  {
    title: "Manufacturing",
    items: [
      { title: "Work Orders", url: "/work-orders", icon: Wrench },
      { title: "Inventory", url: "/inventory", icon: Package },
      { title: "NCRs", url: "/ncrs", icon: ClipboardList },
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
      { title: "Users", url: "/users", icon: Users },
      { title: "Settings", url: "/settings", icon: Settings },
    ],
  },
];

export function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [darkMode, setDarkMode] = useState(false);

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
            {navigationItems.map((section) => (
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
              <Button variant="ghost" size="icon">
                <Bell className="h-4 w-4" />
                <Badge variant="destructive" className="absolute -top-2 -right-2 h-5 w-5 rounded-full p-0 text-xs">
                  3
                </Badge>
              </Button>

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
                  <DropdownMenuItem>
                    <User className="mr-2 h-4 w-4" />
                    <span>Profile</span>
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <Cog className="mr-2 h-4 w-4" />
                    <span>Settings</span>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem>
                    <span>Log out</span>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </header>

          {/* Main Content Area */}
          <main className="flex-1 overflow-y-auto p-6">
            <Outlet />
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
      </div>
    </SidebarProvider>
  );
}