import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Checkbox } from "../components/ui/checkbox";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { Separator } from "../components/ui/separator";
import { 
  Mail, 
  Send, 
  CheckCircle2, 
  AlertTriangle, 
  Settings,
  Eye,
  EyeOff,
  TestTube2,
  Shield
} from "lucide-react";

interface EmailConfig {
  enabled: boolean;
  smtp_host: string;
  smtp_port: number;
  smtp_security: 'none' | 'tls' | 'ssl';
  smtp_username: string;
  smtp_password: string;
  from_address: string;
  from_name: string;
  test_email?: string;
}

interface TestEmailResult {
  success: boolean;
  message: string;
  timestamp: string;
}

function EmailSettings() {
  const [config, setConfig] = useState<EmailConfig>({
    enabled: false,
    smtp_host: '',
    smtp_port: 587,
    smtp_security: 'tls',
    smtp_username: '',
    smtp_password: '',
    from_address: '',
    from_name: 'ZRP System',
    test_email: ''
  });
  
  const [originalConfig, setOriginalConfig] = useState<EmailConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [testResult, setTestResult] = useState<TestEmailResult | null>(null);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);

  // Common SMTP presets
  const smtpPresets = [
    { name: 'Custom', host: '', port: 587, security: 'tls' as const },
    { name: 'Gmail', host: 'smtp.gmail.com', port: 587, security: 'tls' as const },
    { name: 'Outlook/Hotmail', host: 'smtp-mail.outlook.com', port: 587, security: 'tls' as const },
    { name: 'Yahoo', host: 'smtp.mail.yahoo.com', port: 587, security: 'tls' as const },
    { name: 'Amazon SES', host: 'email-smtp.us-east-1.amazonaws.com', port: 587, security: 'tls' as const },
    { name: 'SendGrid', host: 'smtp.sendgrid.net', port: 587, security: 'tls' as const },
  ];

  useEffect(() => {
    const fetchEmailConfig = async () => {
      try {
        setLoading(true);
        
        // Mock data - replace with real API call
        const mockConfig: EmailConfig = {
          enabled: true,
          smtp_host: 'smtp.example.com',
          smtp_port: 587,
          smtp_security: 'tls',
          smtp_username: 'notifications@example.com',
          smtp_password: '********', // Password should be masked
          from_address: 'noreply@example.com',
          from_name: 'ZRP System',
        };
        
        setConfig(mockConfig);
        setOriginalConfig(mockConfig);
      } catch (error) {
        console.error("Failed to fetch email configuration:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchEmailConfig();
  }, []);

  // Check for unsaved changes
  useEffect(() => {
    if (!originalConfig) return;
    
    const hasChanges = JSON.stringify(config) !== JSON.stringify(originalConfig);
    setHasUnsavedChanges(hasChanges);
  }, [config, originalConfig]);

  const updateConfig = (updates: Partial<EmailConfig>) => {
    setConfig(prev => ({ ...prev, ...updates }));
  };

  const applyPreset = (presetName: string) => {
    const preset = smtpPresets.find(p => p.name === presetName);
    if (!preset || preset.name === 'Custom') return;
    
    updateConfig({
      smtp_host: preset.host,
      smtp_port: preset.port,
      smtp_security: preset.security
    });
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      
      // Mock save - replace with real API call
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      setOriginalConfig(config);
      setTestResult(null); // Clear any previous test results
    } catch (error) {
      console.error("Failed to save email configuration:", error);
    } finally {
      setSaving(false);
    }
  };

  const handleTestEmail = async () => {
    if (!config.test_email) return;
    
    try {
      setTesting(true);
      setTestResult(null);
      
      // Mock test email - replace with real API call
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // Simulate random success/failure for demo
      const success = Math.random() > 0.3;
      
      setTestResult({
        success,
        message: success 
          ? `Test email sent successfully to ${config.test_email}`
          : 'Failed to send test email. Please check your SMTP configuration.',
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      setTestResult({
        success: false,
        message: 'Error sending test email: ' + (error as Error).message,
        timestamp: new Date().toISOString()
      });
    } finally {
      setTesting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading email settings...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Email Settings</h1>
          <p className="text-muted-foreground">
            Configure SMTP settings for system notifications and alerts.
          </p>
        </div>
        
        <div className="flex items-center gap-2">
          {hasUnsavedChanges && (
            <Badge variant="outline" className="text-orange-600">
              Unsaved Changes
            </Badge>
          )}
          <Button 
            onClick={handleSave}
            disabled={saving || !hasUnsavedChanges}
            className="flex items-center gap-2"
          >
            {saving ? (
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
            ) : (
              <Settings className="h-4 w-4" />
            )}
            {saving ? 'Saving...' : 'Save Settings'}
          </Button>
        </div>
      </div>

      {/* Enable/Disable Toggle */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Mail className="h-5 w-5" />
            Email Notifications
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-2">
            <Checkbox
              id="email-enabled"
              checked={config.enabled}
              onCheckedChange={(checked) => updateConfig({ enabled: !!checked })}
            />
            <Label htmlFor="email-enabled" className="text-sm font-medium">
              Enable email notifications
            </Label>
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            When enabled, the system will send notifications for alerts, reminders, and status updates.
          </p>
          
          {config.enabled && (
            <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
              <div className="flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4 text-blue-600" />
                <span className="text-sm text-blue-800">Email notifications are enabled</span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* SMTP Configuration */}
      {config.enabled && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>SMTP Configuration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* SMTP Preset */}
              <div className="space-y-2">
                <Label>SMTP Provider Preset</Label>
                <Select onValueChange={applyPreset}>
                  <SelectTrigger>
                    <SelectValue placeholder="Choose a preset or configure manually" />
                  </SelectTrigger>
                  <SelectContent>
                    {smtpPresets.map(preset => (
                      <SelectItem key={preset.name} value={preset.name}>
                        {preset.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  Select a common provider to auto-fill SMTP settings, or choose "Custom" to configure manually.
                </p>
              </div>

              <Separator />

              {/* SMTP Settings */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="smtp-host">SMTP Host</Label>
                  <Input
                    id="smtp-host"
                    value={config.smtp_host}
                    onChange={(e) => updateConfig({ smtp_host: e.target.value })}
                    placeholder="smtp.example.com"
                  />
                </div>
                
                <div className="space-y-2">
                  <Label htmlFor="smtp-port">SMTP Port</Label>
                  <Input
                    id="smtp-port"
                    type="number"
                    value={config.smtp_port}
                    onChange={(e) => updateConfig({ smtp_port: parseInt(e.target.value) || 587 })}
                    placeholder="587"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label>Security</Label>
                <Select 
                  value={config.smtp_security} 
                  onValueChange={(value: any) => updateConfig({ smtp_security: value })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">None</SelectItem>
                    <SelectItem value="tls">TLS (Recommended)</SelectItem>
                    <SelectItem value="ssl">SSL</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <Separator />

              {/* Authentication */}
              <div className="space-y-4">
                <h4 className="font-medium flex items-center gap-2">
                  <Shield className="h-4 w-4" />
                  Authentication
                </h4>
                
                <div className="space-y-2">
                  <Label htmlFor="smtp-username">Username</Label>
                  <Input
                    id="smtp-username"
                    value={config.smtp_username}
                    onChange={(e) => updateConfig({ smtp_username: e.target.value })}
                    placeholder="username@example.com"
                  />
                </div>
                
                <div className="space-y-2">
                  <Label htmlFor="smtp-password">Password</Label>
                  <div className="relative">
                    <Input
                      id="smtp-password"
                      type={showPassword ? "text" : "password"}
                      value={config.smtp_password}
                      onChange={(e) => updateConfig({ smtp_password: e.target.value })}
                      placeholder="Enter SMTP password"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="absolute right-2 top-1/2 transform -translate-y-1/2 h-8 w-8 p-0"
                      onClick={() => setShowPassword(!showPassword)}
                    >
                      {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    For Gmail and other providers, you may need to use an app-specific password.
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* From Address Configuration */}
          <Card>
            <CardHeader>
              <CardTitle>Sender Configuration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="from-address">From Email Address</Label>
                  <Input
                    id="from-address"
                    type="email"
                    value={config.from_address}
                    onChange={(e) => updateConfig({ from_address: e.target.value })}
                    placeholder="noreply@example.com"
                  />
                </div>
                
                <div className="space-y-2">
                  <Label htmlFor="from-name">From Name</Label>
                  <Input
                    id="from-name"
                    value={config.from_name}
                    onChange={(e) => updateConfig({ from_name: e.target.value })}
                    placeholder="ZRP System"
                  />
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Test Email */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TestTube2 className="h-5 w-5" />
                Test Email Configuration
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="test-email">Test Email Address</Label>
                <div className="flex gap-2">
                  <Input
                    id="test-email"
                    type="email"
                    value={config.test_email || ''}
                    onChange={(e) => updateConfig({ test_email: e.target.value })}
                    placeholder="test@example.com"
                    className="flex-1"
                  />
                  <Button
                    onClick={handleTestEmail}
                    disabled={testing || !config.test_email}
                    className="flex items-center gap-2"
                  >
                    {testing ? (
                      <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                    ) : (
                      <Send className="h-4 w-4" />
                    )}
                    {testing ? 'Sending...' : 'Send Test'}
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">
                  Send a test email to verify your SMTP configuration is working correctly.
                </p>
              </div>

              {/* Test Result */}
              {testResult && (
                <div className={`p-4 rounded-lg border ${
                  testResult.success 
                    ? 'bg-green-50 border-green-200' 
                    : 'bg-red-50 border-red-200'
                }`}>
                  <div className="flex items-start gap-2">
                    {testResult.success ? (
                      <CheckCircle2 className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
                    ) : (
                      <AlertTriangle className="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
                    )}
                    <div className="flex-1">
                      <p className={`text-sm font-medium ${
                        testResult.success ? 'text-green-800' : 'text-red-800'
                      }`}>
                        {testResult.success ? 'Test Successful' : 'Test Failed'}
                      </p>
                      <p className={`text-sm mt-1 ${
                        testResult.success ? 'text-green-700' : 'text-red-700'
                      }`}>
                        {testResult.message}
                      </p>
                      <p className="text-xs text-muted-foreground mt-2">
                        {new Date(testResult.timestamp).toLocaleString()}
                      </p>
                    </div>
                  </div>
                </div>
              )}

              {hasUnsavedChanges && (
                <div className="bg-amber-50 border border-amber-200 rounded-lg p-3">
                  <p className="text-sm text-amber-800">
                    <strong>Note:</strong> You have unsaved changes. Save your configuration before testing.
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
export default EmailSettings;
