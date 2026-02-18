import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { 
  BarChart, 
  FileText, 
  Download, 
  DollarSign, 
  Package,
  Clock,
  TrendingUp,
  AlertTriangle,
  Users,
  ShoppingCart
} from "lucide-react";

interface ReportCard {
  id: string;
  title: string;
  description: string;
  icon: any;
  color: string;
  lastGenerated?: string;
}

interface ReportData {
  columns: string[];
  rows: any[][];
  summary?: {
    total: number;
    currency?: boolean;
  };
}

const reportCards: ReportCard[] = [
  {
    id: 'inventory_valuation',
    title: 'Inventory Valuation',
    description: 'Current inventory value by category and location',
    icon: DollarSign,
    color: 'text-green-600',
  },
  {
    id: 'open_ecos',
    title: 'Open ECOs Report',
    description: 'Engineering change orders by status and priority',
    icon: FileText,
    color: 'text-blue-600',
  },
  {
    id: 'wo_throughput',
    title: 'Work Order Throughput',
    description: 'Completion rates and cycle times by month',
    icon: Clock,
    color: 'text-orange-600',
  },
  {
    id: 'vendor_performance',
    title: 'Vendor Performance',
    description: 'On-time delivery and quality metrics',
    icon: TrendingUp,
    color: 'text-purple-600',
  },
  {
    id: 'low_stock',
    title: 'Low Stock Alert',
    description: 'Parts below minimum stock levels',
    icon: AlertTriangle,
    color: 'text-red-600',
  },
  {
    id: 'user_activity',
    title: 'User Activity',
    description: 'Login frequency and module usage',
    icon: Users,
    color: 'text-teal-600',
  },
  {
    id: 'po_summary',
    title: 'Purchase Order Summary',
    description: 'PO status and spending by vendor',
    icon: ShoppingCart,
    color: 'text-indigo-600',
  },
  {
    id: 'part_usage',
    title: 'Part Usage Analysis',
    description: 'Most/least used parts and consumption trends',
    icon: Package,
    color: 'text-pink-600',
  },
];

function Reports() {
  const [selectedReport, setSelectedReport] = useState<string | null>(null);
  const [reportData, setReportData] = useState<ReportData | null>(null);
  const [loading, setLoading] = useState(false);

  const generateReport = async (reportId: string) => {
    setLoading(true);
    setSelectedReport(reportId);
    
    try {
      // Mock report data - replace with real API calls
      let mockData: ReportData;
      
      switch (reportId) {
        case 'inventory_valuation':
          mockData = {
            columns: ['Category', 'Location', 'Parts Count', 'Total Value'],
            rows: [
              ['Electronics', 'Warehouse A', '1,234', '$125,430'],
              ['Mechanical', 'Warehouse B', '856', '$89,250'],
              ['Consumables', 'Storage C', '2,145', '$15,680'],
              ['Raw Materials', 'Warehouse A', '432', '$45,290'],
            ],
            summary: { total: 275650, currency: true }
          };
          break;
          
        case 'open_ecos':
          mockData = {
            columns: ['ECO ID', 'Title', 'Status', 'Priority', 'Created', 'Assignee'],
            rows: [
              ['ECO-001', 'Widget Improvement v2.1', 'In Review', 'High', '2024-02-15', 'John Doe'],
              ['ECO-002', 'Cost Reduction Initiative', 'Draft', 'Medium', '2024-02-18', 'Jane Smith'],
              ['ECO-003', 'Safety Enhancement', 'Approved', 'High', '2024-02-10', 'Mike Johnson'],
            ]
          };
          break;
          
        case 'wo_throughput':
          mockData = {
            columns: ['Month', 'Completed', 'In Progress', 'Avg Cycle Time (days)', 'On-Time %'],
            rows: [
              ['January 2024', '45', '12', '5.2', '87%'],
              ['February 2024', '52', '8', '4.8', '92%'],
              ['March 2024', '38', '15', '6.1', '81%'],
            ]
          };
          break;
          
        case 'vendor_performance':
          mockData = {
            columns: ['Vendor', 'Orders', 'On-Time Delivery', 'Quality Score', 'Total Spent'],
            rows: [
              ['ABC Electronics', '45', '94%', '4.8/5', '$125,430'],
              ['XYZ Components', '32', '89%', '4.6/5', '$89,250'],
              ['TechParts Inc', '28', '96%', '4.9/5', '$67,890'],
            ]
          };
          break;
          
        default:
          mockData = {
            columns: ['Item', 'Value', 'Status'],
            rows: [
              ['Sample Data 1', '100', 'Active'],
              ['Sample Data 2', '200', 'Pending'],
              ['Sample Data 3', '150', 'Complete'],
            ]
          };
      }
      
      // Simulate API delay
      await new Promise(resolve => setTimeout(resolve, 1000));
      setReportData(mockData);
      
    } catch (error) {
      console.error("Failed to generate report:", error);
    } finally {
      setLoading(false);
    }
  };

  const exportReport = (format: 'csv' | 'html') => {
    if (!reportData || !selectedReport) return;
    
    const selectedCard = reportCards.find(card => card.id === selectedReport);
    const reportTitle = selectedCard?.title || 'Report';
    
    if (format === 'csv') {
      const csvContent = [
        reportData.columns.join(','),
        ...reportData.rows.map(row => row.join(','))
      ].join('\n');
      
      const blob = new Blob([csvContent], { type: 'text/csv' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${reportTitle.replace(/\s+/g, '_')}.csv`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } else if (format === 'html') {
      const htmlContent = `
        <!DOCTYPE html>
        <html>
        <head>
          <title>${reportTitle}</title>
          <style>
            body { font-family: Arial, sans-serif; margin: 20px; }
            h1 { color: #333; }
            table { border-collapse: collapse; width: 100%; }
            th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
            th { background-color: #f2f2f2; }
            .summary { margin-top: 20px; padding: 10px; background-color: #f9f9f9; }
          </style>
        </head>
        <body>
          <h1>${reportTitle}</h1>
          <p>Generated on: ${new Date().toLocaleDateString()}</p>
          <table>
            <thead>
              <tr>${reportData.columns.map(col => `<th>${col}</th>`).join('')}</tr>
            </thead>
            <tbody>
              ${reportData.rows.map(row => 
                `<tr>${row.map(cell => `<td>${cell}</td>`).join('')}</tr>`
              ).join('')}
            </tbody>
          </table>
          ${reportData.summary ? `
            <div class="summary">
              <strong>Total: ${reportData.summary.currency ? '$' : ''}${reportData.summary.total.toLocaleString()}</strong>
            </div>
          ` : ''}
        </body>
        </html>
      `;
      
      const blob = new Blob([htmlContent], { type: 'text/html' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${reportTitle.replace(/\s+/g, '_')}.html`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Reports</h1>
        <p className="text-muted-foreground">
          Generate and export reports across all modules.
        </p>
      </div>

      {/* Report Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {reportCards.map((report) => {
          const Icon = report.icon;
          const isSelected = selectedReport === report.id;
          
          return (
            <Card 
              key={report.id} 
              className={`cursor-pointer transition-colors hover:bg-accent/50 ${
                isSelected ? 'ring-2 ring-primary' : ''
              }`}
              onClick={() => generateReport(report.id)}
            >
              <CardContent className="p-6">
                <div className="flex items-start gap-3">
                  <Icon className={`h-8 w-8 ${report.color} flex-shrink-0`} />
                  <div className="flex-1 min-w-0">
                    <h3 className="font-semibold text-sm mb-1">{report.title}</h3>
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      {report.description}
                    </p>
                    {report.lastGenerated && (
                      <p className="text-xs text-muted-foreground mt-2">
                        Last: {report.lastGenerated}
                      </p>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Report Results */}
      {selectedReport && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="flex items-center gap-2">
              <BarChart className="h-5 w-5" />
              {reportCards.find(card => card.id === selectedReport)?.title}
            </CardTitle>
            {reportData && !loading && (
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => exportReport('csv')}
                  className="flex items-center gap-1"
                >
                  <Download className="h-4 w-4" />
                  CSV
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => exportReport('html')}
                  className="flex items-center gap-1"
                >
                  <Download className="h-4 w-4" />
                  HTML
                </Button>
              </div>
            )}
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="flex items-center justify-center py-12">
                <div className="text-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
                  <p className="mt-2 text-muted-foreground">Generating report...</p>
                </div>
              </div>
            ) : reportData ? (
              <div className="space-y-4">
                <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        {reportData.columns.map((column, index) => (
                          <TableHead key={index}>{column}</TableHead>
                        ))}
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {reportData.rows.map((row, rowIndex) => (
                        <TableRow key={rowIndex}>
                          {row.map((cell, cellIndex) => (
                            <TableCell key={cellIndex}>{cell}</TableCell>
                          ))}
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
                
                {reportData.summary && (
                  <div className="flex justify-end">
                    <Card className="w-fit">
                      <CardContent className="p-4">
                        <div className="text-sm font-medium">
                          Total: {reportData.summary.currency ? '$' : ''}{reportData.summary.total.toLocaleString()}
                        </div>
                      </CardContent>
                    </Card>
                  </div>
                )}
                
                <div className="text-xs text-muted-foreground">
                  Generated on {new Date().toLocaleString()}
                </div>
              </div>
            ) : null}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
export default Reports;
