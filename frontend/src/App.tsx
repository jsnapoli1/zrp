import React, { Suspense } from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { Toaster } from "sonner";
import { AppLayout } from "./layouts/AppLayout";
import { LoadingSpinner } from "./components/LoadingSpinner";
import { WebSocketProvider } from "./contexts/WebSocketContext";
import { ErrorBoundary } from "./components/ErrorBoundary";

// Lazy load all pages for code-splitting
const Dashboard = React.lazy(() => import("./pages/Dashboard"));
const Calendar = React.lazy(() => import("./pages/Calendar"));
const Parts = React.lazy(() => import("./pages/Parts"));
const PartDetail = React.lazy(() => import("./pages/PartDetail"));
const ECOs = React.lazy(() => import("./pages/ECOs"));
const ECODetail = React.lazy(() => import("./pages/ECODetail"));
const Documents = React.lazy(() => import("./pages/Documents"));
const Inventory = React.lazy(() => import("./pages/Inventory"));
const InventoryDetail = React.lazy(() => import("./pages/InventoryDetail"));
const Procurement = React.lazy(() => import("./pages/Procurement"));
const PODetail = React.lazy(() => import("./pages/PODetail"));
const Vendors = React.lazy(() => import("./pages/Vendors"));
const VendorDetail = React.lazy(() => import("./pages/VendorDetail"));
const WorkOrders = React.lazy(() => import("./pages/WorkOrders"));
const WorkOrderDetail = React.lazy(() => import("./pages/WorkOrderDetail"));
const WorkOrderPrint = React.lazy(() => import("./pages/WorkOrderPrint"));
const POPrint = React.lazy(() => import("./pages/POPrint"));
const NCRs = React.lazy(() => import("./pages/NCRs"));
const NCRDetail = React.lazy(() => import("./pages/NCRDetail"));
const RMAs = React.lazy(() => import("./pages/RMAs"));
const RMADetail = React.lazy(() => import("./pages/RMADetail"));
const Testing = React.lazy(() => import("./pages/Testing"));
const Devices = React.lazy(() => import("./pages/Devices"));
const DeviceDetail = React.lazy(() => import("./pages/DeviceDetail"));
const Firmware = React.lazy(() => import("./pages/Firmware"));
const FirmwareDetail = React.lazy(() => import("./pages/FirmwareDetail"));
const Quotes = React.lazy(() => import("./pages/Quotes"));
const QuoteDetail = React.lazy(() => import("./pages/QuoteDetail"));
const Shipments = React.lazy(() => import("./pages/Shipments"));
const ShipmentDetail = React.lazy(() => import("./pages/ShipmentDetail"));
const ShipmentPrint = React.lazy(() => import("./pages/ShipmentPrint"));
const Receiving = React.lazy(() => import("./pages/Receiving"));
const Reports = React.lazy(() => import("./pages/Reports"));
const Audit = React.lazy(() => import("./pages/Audit"));
const Users = React.lazy(() => import("./pages/Users"));
const APIKeys = React.lazy(() => import("./pages/APIKeys"));
const EmailSettings = React.lazy(() => import("./pages/EmailSettings"));
const EmailPreferences = React.lazy(() => import("./pages/EmailPreferences"));
const EmailLog = React.lazy(() => import("./pages/EmailLog"));
const GitPLMSettings = React.lazy(() => import("./pages/GitPLMSettings"));
const DistributorSettings = React.lazy(() => import("./pages/DistributorSettings"));
const Login = React.lazy(() => import("./pages/Login"));
const Backups = React.lazy(() => import("./pages/Backups"));
const UndoHistory = React.lazy(() => import("./pages/UndoHistory"));
const Scan = React.lazy(() => import("./pages/Scan"));
const RFQs = React.lazy(() => import("./pages/RFQs"));
const RFQDetail = React.lazy(() => import("./pages/RFQDetail"));

// Placeholder components for other pages
const PlaceholderPage = ({ title }: { title: string }) => (
  <div className="space-y-6">
    <div>
      <h1 className="text-3xl font-bold tracking-tight">{title}</h1>
      <p className="text-muted-foreground">
        This page is under construction. The React foundation is ready - 
        individual module pages can be built on top of this structure.
      </p>
    </div>
    <div className="rounded-lg border border-dashed p-8 text-center">
      <h3 className="text-lg font-semibold">Coming Soon</h3>
      <p className="text-sm text-muted-foreground mt-2">
        This {title.toLowerCase()} interface will be implemented next.
      </p>
    </div>
  </div>
);

function App() {
  return (
    <Router>
      <Toaster position="bottom-right" richColors closeButton />
      <ErrorBoundary>
      <WebSocketProvider>
      <Suspense fallback={<LoadingSpinner />}>
        <Routes>
          <Route path="/login" element={<Login />} />
          {/* Print routes - outside AppLayout for clean printing */}
          <Route path="/work-orders/:id/print" element={<WorkOrderPrint />} />
          <Route path="/purchase-orders/:id/print" element={<POPrint />} />
          <Route path="/shipments/:id/print" element={<ShipmentPrint />} />
          <Route path="/" element={<AppLayout />}>
            <Route index element={<Dashboard />} />
            <Route path="/dashboard" element={<Dashboard />} />
            
            {/* Engineering */}
            <Route path="/parts" element={<Parts />} />
            <Route path="/parts/:ipn" element={<PartDetail />} />
            <Route path="/ecos" element={<ECOs />} />
            <Route path="/ecos/:id" element={<ECODetail />} />
            <Route path="/documents" element={<Documents />} />
            <Route path="/testing" element={<Testing />} />
            <Route path="/ncrs" element={<NCRs />} />
            <Route path="/ncrs/:id" element={<NCRDetail />} />
            <Route path="/rmas" element={<RMAs />} />
            <Route path="/rmas/:id" element={<RMADetail />} />
            <Route path="/devices" element={<Devices />} />
            <Route path="/devices/:serialNumber" element={<DeviceDetail />} />
            <Route path="/firmware" element={<Firmware />} />
            <Route path="/firmware/:id" element={<FirmwareDetail />} />
            <Route path="/quotes" element={<Quotes />} />
            <Route path="/quotes/:id" element={<QuoteDetail />} />
            
            {/* Supply Chain */}
            <Route path="/vendors" element={<Vendors />} />
            <Route path="/vendors/:id" element={<VendorDetail />} />
            <Route path="/purchase-orders" element={<Procurement />} />
            <Route path="/purchase-orders/:id" element={<PODetail />} />
            <Route path="/procurement" element={<Procurement />} />
            <Route path="/rfqs" element={<RFQs />} />
            <Route path="/rfqs/:id" element={<RFQDetail />} />
            <Route path="/receiving" element={<Receiving />} />
            <Route path="/shipments" element={<Shipments />} />
            <Route path="/shipments/:id" element={<ShipmentDetail />} />
            
            {/* Manufacturing */}
            <Route path="/work-orders" element={<WorkOrders />} />
            <Route path="/work-orders/:id" element={<WorkOrderDetail />} />
            <Route path="/inventory" element={<Inventory />} />
            <Route path="/inventory/:ipn" element={<InventoryDetail />} />
            
            {/* Field & Service */}
            <Route path="/field-reports" element={<PlaceholderPage title="Field Reports" />} />
            <Route path="/pricing" element={<PlaceholderPage title="Pricing" />} />
            
            {/* Reports */}
            <Route path="/reports" element={<Reports />} />
            <Route path="/calendar" element={<Calendar />} />
            
            {/* Admin */}
            <Route path="/users" element={<Users />} />
            <Route path="/audit" element={<Audit />} />
            <Route path="/api-keys" element={<APIKeys />} />
            <Route path="/email-settings" element={<EmailSettings />} />
            <Route path="/email-preferences" element={<EmailPreferences />} />
            <Route path="/email-log" element={<EmailLog />} />
            <Route path="/gitplm-settings" element={<GitPLMSettings />} />
            <Route path="/distributor-settings" element={<DistributorSettings />} />
            <Route path="/backups" element={<Backups />} />
            <Route path="/undo-history" element={<UndoHistory />} />
            <Route path="/scan" element={<Scan />} />
            <Route path="/settings" element={<PlaceholderPage title="Settings" />} />
          </Route>
        </Routes>
      </Suspense>
      </WebSocketProvider>
      </ErrorBoundary>
    </Router>
  );
}

export default App;