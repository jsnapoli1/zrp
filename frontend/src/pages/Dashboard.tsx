import { useEffect, useState, useCallback } from "react";
import { useWSSubscription } from "../contexts/WebSocketContext";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { 
  Package, 
  FileText, 
  AlertTriangle, 
  Wrench, 
  ShoppingCart, 
  ClipboardList,
  Smartphone,
  RotateCcw,
} from "lucide-react";
import { api, type DashboardStats } from "../lib/api";

interface ExtendedDashboardStats extends DashboardStats {
  open_ecos: number;
  open_pos: number;
  open_ncrs: number;
  total_devices: number;
  open_rmas: number;
}

const kpiCards = [
  {
    title: "Total Parts",
    key: "total_parts",
    icon: Package,
    color: "text-blue-600",
  },
  {
    title: "Open ECOs",
    key: "open_ecos",
    icon: FileText,
    color: "text-yellow-600",
  },
  {
    title: "Low Stock",
    key: "low_stock_alerts",
    icon: AlertTriangle,
    color: "text-red-600",
  },
  {
    title: "Active Work Orders",
    key: "active_work_orders",
    icon: Wrench,
    color: "text-green-600",
  },
  {
    title: "Open POs",
    key: "open_pos",
    icon: ShoppingCart,
    color: "text-purple-600",
  },
  {
    title: "Open NCRs",
    key: "open_ncrs",
    icon: ClipboardList,
    color: "text-orange-600",
  },
  {
    title: "Total Devices",
    key: "total_devices",
    icon: Smartphone,
    color: "text-teal-600",
  },
  {
    title: "Open RMAs",
    key: "open_rmas",
    icon: RotateCcw,
    color: "text-pink-600",
  },
];

interface Activity {
  id: string;
  type: string;
  description: string;
  timestamp: string;
  user: string;
}

function Dashboard() {
  const [stats, setStats] = useState<ExtendedDashboardStats | null>(null);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchDashboardData = useCallback(async () => {
    try {
      const [dashboardData, chartsData] = await Promise.all([
        api.getDashboard(),
        api.getDashboardCharts(),
      ]);
      
      // Extend the dashboard data with additional stats
      const extendedStats: ExtendedDashboardStats = {
        ...dashboardData,
        open_ecos: chartsData?.eco_counts?.reduce((a: number, b: number) => a + b, 0) || 0,
        open_pos: 12, // Mock data - replace with real API call
        open_ncrs: 5, // Mock data - replace with real API call  
        total_devices: 150, // Mock data - replace with real API call
        open_rmas: 3, // Mock data - replace with real API call
      };
      
      setStats(extendedStats);
      
      // Mock activity data - replace with real API call
      setActivities([
        {
          id: "1",
          type: "ECO",
          description: "New ECO created: Widget Improvement v2.1",
          timestamp: "2 hours ago",
          user: "John Doe",
        },
        {
          id: "2",
          type: "Work Order",
          description: "Work Order WO-001 completed",
          timestamp: "4 hours ago",
          user: "Jane Smith",
        },
        {
          id: "3",
          type: "Inventory",
          description: "Low stock alert for part ABC-123",
          timestamp: "6 hours ago",
          user: "System",
        },
      ]);
    } catch (error) {
      console.error("Failed to fetch dashboard data:", error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDashboardData();
  }, [fetchDashboardData]);

  // Real-time updates via WebSocket instead of 30-second polling
  useWSSubscription(
    ["*"],
    useCallback(() => {
      fetchDashboardData();
    }, [fetchDashboardData])
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading dashboard...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Welcome back! Here's what's happening with your operations.
        </p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {kpiCards.map((card) => {
          const Icon = card.icon;
          const value = stats?.[card.key as keyof ExtendedDashboardStats] || 0;
          
          return (
            <Card key={card.key} className="text-center">
              <CardContent className="p-6">
                <div className="flex items-center justify-center mb-2">
                  <Icon className={`h-8 w-8 ${card.color}`} />
                </div>
                <div className={`text-3xl font-bold ${card.color}`}>
                  {typeof value === 'number' ? value.toLocaleString() : value}
                </div>
                <div className="text-sm text-muted-foreground mt-1">
                  {card.title}
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Charts and Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* ECO Status Chart */}
        <Card>
          <CardHeader>
            <CardTitle>ECO Status</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-[200px] flex items-center justify-center text-muted-foreground">
              Chart visualization would go here
              <br />
              (Chart.js integration needed)
            </div>
          </CardContent>
        </Card>

        {/* Recent Activity */}
        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {activities.map((activity) => (
                <div key={activity.id} className="flex items-start space-x-3">
                  <Badge variant="secondary" className="mt-1">
                    {activity.type}
                  </Badge>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium">
                      {activity.description}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {activity.timestamp} by {activity.user}
                    </p>
                  </div>
                </div>
              ))}
              
              {activities.length === 0 && (
                <div className="text-center text-muted-foreground py-8">
                  No recent activity
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
export default Dashboard;
