import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "../components/ui/dialog";
import { 
  Key, 
  Plus, 
  Copy, 
  Trash2, 
  AlertTriangle,
  CheckCircle2
} from "lucide-react";

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  full_key?: string; // Only available immediately after creation
  status: 'active' | 'revoked';
  created_at: string;
  last_used?: string;
  created_by: string;
}

interface CreateKeyForm {
  name: string;
}

function APIKeys() {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [showFullKey, setShowFullKey] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [revokeDialogOpen, setRevokeDialogOpen] = useState(false);
  const [keyToRevoke, setKeyToRevoke] = useState<APIKey | null>(null);
  
  const [createForm, setCreateForm] = useState<CreateKeyForm>({
    name: ''
  });

  // Format relative timestamp
  const formatRelativeTime = (timestamp: string): string => {
    const now = new Date();
    const logTime = new Date(timestamp);
    const diffMs = now.getTime() - logTime.getTime();
    
    const seconds = Math.floor(diffMs / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    
    if (days > 0) {
      return `${days} day${days === 1 ? '' : 's'} ago`;
    } else if (hours > 0) {
      return `${hours} hour${hours === 1 ? '' : 's'} ago`;
    } else if (minutes > 0) {
      return `${minutes} minute${minutes === 1 ? '' : 's'} ago`;
    } else {
      return 'Just now';
    }
  };

  // Generate random API key
  const generateApiKey = (): string => {
    const prefix = 'zrp_';
    const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    let result = prefix;
    for (let i = 0; i < 32; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  };

  useEffect(() => {
    const fetchAPIKeys = async () => {
      try {
        setLoading(true);
        
        // Mock data - replace with real API call
        const mockKeys: APIKey[] = [
          {
            id: '1',
            name: 'Production Integration',
            key_prefix: 'zrp_abc123...',
            status: 'active',
            created_at: '2024-01-15T10:00:00Z',
            last_used: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
            created_by: 'admin@example.com'
          },
          {
            id: '2',
            name: 'Mobile App',
            key_prefix: 'zrp_def456...',
            status: 'active',
            created_at: '2024-02-01T14:30:00Z',
            last_used: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
            created_by: 'developer@example.com'
          },
          {
            id: '3',
            name: 'Legacy System',
            key_prefix: 'zrp_ghi789...',
            status: 'revoked',
            created_at: '2023-12-10T09:15:00Z',
            last_used: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
            created_by: 'admin@example.com'
          },
          {
            id: '4',
            name: 'Testing Environment',
            key_prefix: 'zrp_jkl012...',
            status: 'active',
            created_at: '2024-02-10T16:45:00Z',
            created_by: 'tester@example.com'
          },
        ];
        
        setApiKeys(mockKeys);
      } catch (error) {
        console.error("Failed to fetch API keys:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchAPIKeys();
  }, []);

  const handleCreateKey = async () => {
    try {
      const fullKey = generateApiKey();
      const keyPrefix = fullKey.substring(0, 10) + '...';
      
      // Mock create key - replace with real API call
      const newKey: APIKey = {
        id: Math.random().toString(36).substr(2, 9),
        name: createForm.name,
        key_prefix: keyPrefix,
        full_key: fullKey, // Only available at creation time
        status: 'active',
        created_at: new Date().toISOString(),
        created_by: 'current_user@example.com' // Should be current user
      };
      
      setApiKeys(prev => [...prev, newKey]);
      setShowFullKey(newKey.id);
      setCreateDialogOpen(false);
      setCreateForm({ name: '' });
    } catch (error) {
      console.error("Failed to create API key:", error);
    }
  };

  const handleRevokeKey = async () => {
    if (!keyToRevoke) return;
    
    try {
      // Mock revoke key - replace with real API call
      setApiKeys(prev => prev.map(key => 
        key.id === keyToRevoke.id 
          ? { ...key, status: 'revoked' as const }
          : key
      ));
      
      setRevokeDialogOpen(false);
      setKeyToRevoke(null);
    } catch (error) {
      console.error("Failed to revoke API key:", error);
    }
  };

  const copyToClipboard = async (text: string, keyId: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(keyId);
      setTimeout(() => setCopiedKey(null), 2000);
    } catch (error) {
      console.error("Failed to copy to clipboard:", error);
    }
  };

  const openRevokeDialog = (key: APIKey) => {
    setKeyToRevoke(key);
    setRevokeDialogOpen(true);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
          <p className="mt-2 text-muted-foreground">Loading API keys...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">API Keys</h1>
          <p className="text-muted-foreground">
            Manage API keys for programmatic access to ZRP.
          </p>
        </div>
        
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button className="flex items-center gap-2">
              <Plus className="h-4 w-4" />
              Generate New Key
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Generate New API Key</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="key-name">Key Name</Label>
                <Input
                  id="key-name"
                  value={createForm.name}
                  onChange={(e) => setCreateForm(prev => ({ ...prev, name: e.target.value }))}
                  placeholder="Enter a descriptive name"
                />
                <p className="text-xs text-muted-foreground">
                  Choose a name that helps you identify this key's purpose.
                </p>
              </div>
              
              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                <div className="flex items-start gap-2">
                  <AlertTriangle className="h-5 w-5 text-yellow-600 flex-shrink-0 mt-0.5" />
                  <div>
                    <h4 className="text-sm font-medium text-yellow-800">Important</h4>
                    <p className="text-xs text-yellow-700 mt-1">
                      The full API key will only be shown once after creation. 
                      Make sure to copy and store it securely.
                    </p>
                  </div>
                </div>
              </div>
              
              <div className="flex gap-3 pt-4">
                <Button 
                  onClick={handleCreateKey} 
                  className="flex-1"
                  disabled={!createForm.name.trim()}
                >
                  Generate Key
                </Button>
                <Button variant="outline" onClick={() => setCreateDialogOpen(false)} className="flex-1">
                  Cancel
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-3">
              <Key className="h-8 w-8 text-blue-600" />
              <div>
                <div className="text-2xl font-bold">{apiKeys.length}</div>
                <div className="text-sm text-muted-foreground">Total Keys</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-3">
              <CheckCircle2 className="h-8 w-8 text-green-600" />
              <div>
                <div className="text-2xl font-bold">{apiKeys.filter(k => k.status === 'active').length}</div>
                <div className="text-sm text-muted-foreground">Active</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-3">
              <AlertTriangle className="h-8 w-8 text-red-600" />
              <div>
                <div className="text-2xl font-bold">{apiKeys.filter(k => k.status === 'revoked').length}</div>
                <div className="text-sm text-muted-foreground">Revoked</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* New Key Display */}
      {showFullKey && (
        <Card className="border-green-200 bg-green-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-green-800">
              <CheckCircle2 className="h-5 w-5" />
              API Key Generated Successfully
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="bg-white border border-green-200 rounded-lg p-4">
                <Label className="text-sm font-medium text-green-800">Your API Key</Label>
                <div className="flex items-center gap-2 mt-2">
                  <code className="flex-1 px-3 py-2 bg-gray-50 border rounded font-mono text-sm">
                    {apiKeys.find(k => k.id === showFullKey)?.full_key}
                  </code>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => copyToClipboard(
                      apiKeys.find(k => k.id === showFullKey)?.full_key || '', 
                      showFullKey
                    )}
                  >
                    {copiedKey === showFullKey ? (
                      <CheckCircle2 className="h-4 w-4" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              </div>
              
              <div className="bg-amber-50 border border-amber-200 rounded-lg p-3">
                <p className="text-sm text-amber-800">
                  <strong>Warning:</strong> This is the only time the full key will be displayed. 
                  Copy it now and store it securely.
                </p>
              </div>
              
              <Button 
                variant="outline" 
                onClick={() => setShowFullKey(null)}
                className="w-full"
              >
                I've copied the key, hide it
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* API Keys Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="h-5 w-5" />
            API Keys
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Key</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Last Used</TableHead>
                  <TableHead>Created By</TableHead>
                  <TableHead className="w-[120px]">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                      No API keys found. Generate your first key to get started.
                    </TableCell>
                  </TableRow>
                ) : (
                  apiKeys.map((key) => (
                    <TableRow key={key.id}>
                      <TableCell className="font-medium">{key.name}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <code className="font-mono text-sm bg-muted px-2 py-1 rounded">
                            {showFullKey === key.id && key.full_key ? key.full_key : key.key_prefix}
                          </code>
                          {showFullKey === key.id && key.full_key ? (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => copyToClipboard(key.full_key!, key.id)}
                            >
                              {copiedKey === key.id ? (
                                <CheckCircle2 className="h-3 w-3" />
                              ) : (
                                <Copy className="h-3 w-3" />
                              )}
                            </Button>
                          ) : null}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge 
                          variant="secondary"
                          className={key.status === 'active' 
                            ? 'bg-green-100 text-green-800' 
                            : 'bg-red-100 text-red-800'
                          }
                        >
                          {key.status === 'active' ? (
                            <CheckCircle2 className="h-3 w-3 mr-1" />
                          ) : (
                            <AlertTriangle className="h-3 w-3 mr-1" />
                          )}
                          {key.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(key.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {key.last_used ? (
                          <div>
                            <div>{new Date(key.last_used).toLocaleDateString()}</div>
                            <div className="text-xs">{formatRelativeTime(key.last_used)}</div>
                          </div>
                        ) : (
                          'Never'
                        )}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {key.created_by}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          {key.status === 'active' && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => openRevokeDialog(key)}
                              className="flex items-center gap-1 text-red-600 hover:text-red-700"
                            >
                              <Trash2 className="h-3 w-3" />
                              Revoke
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Revoke Confirmation Dialog */}
      <Dialog open={revokeDialogOpen} onOpenChange={setRevokeDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Revoke API Key</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-5 w-5 text-red-600 flex-shrink-0 mt-0.5" />
                <div>
                  <h4 className="text-sm font-medium text-red-800">Confirm Revocation</h4>
                  <p className="text-sm text-red-700 mt-1">
                    Are you sure you want to revoke the API key "<strong>{keyToRevoke?.name}</strong>"?
                  </p>
                  <p className="text-sm text-red-700 mt-2">
                    This action cannot be undone. Any applications using this key will lose access immediately.
                  </p>
                </div>
              </div>
            </div>
            
            <div className="flex gap-3 pt-4">
              <Button 
                variant="destructive" 
                onClick={handleRevokeKey} 
                className="flex-1"
              >
                Yes, Revoke Key
              </Button>
              <Button variant="outline" onClick={() => setRevokeDialogOpen(false)} className="flex-1">
                Cancel
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
export default APIKeys;
