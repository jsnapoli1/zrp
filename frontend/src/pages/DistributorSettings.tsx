import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Badge } from "../components/ui/badge";
import { Store, Save } from "lucide-react";
import { api } from "../lib/api";

export default function DistributorSettings() {
  const [digikeyKey, setDigikeyKey] = useState("");
  const [digikeyClientId, setDigikeyClientId] = useState("");
  const [mouserKey, setMouserKey] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const settings = await api.getDistributorSettings();
      setDigikeyKey(settings.digikey?.api_key || "");
      setDigikeyClientId(settings.digikey?.client_id || "");
      setMouserKey(settings.mouser?.api_key || "");
      setLoaded(true);
    } catch {
      setLoaded(true);
    }
  };

  const saveDigikey = async () => {
    setSaving(true);
    try {
      await api.updateDigikeySettings({ api_key: digikeyKey, client_id: digikeyClientId });
      setMessage("Digikey settings saved");
      setTimeout(() => setMessage(""), 3000);
    } catch {
      setMessage("Failed to save Digikey settings");
    } finally {
      setSaving(false);
    }
  };

  const saveMouser = async () => {
    setSaving(true);
    try {
      await api.updateMouserSettings({ api_key: mouserKey });
      setMessage("Mouser settings saved");
      setTimeout(() => setMessage(""), 3000);
    } catch {
      setMessage("Failed to save Mouser settings");
    } finally {
      setSaving(false);
    }
  };

  if (!loaded) return null;

  return (
    <div className="space-y-6 p-6 max-w-2xl">
      <div className="flex items-center gap-2">
        <Store className="h-6 w-6" />
        <h1 className="text-2xl font-bold">Distributor API Settings</h1>
      </div>

      {message && (
        <Badge variant="secondary" className="text-sm">{message}</Badge>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Digikey</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="text-sm font-medium">API Key</label>
            <Input
              value={digikeyKey}
              onChange={e => setDigikeyKey(e.target.value)}
              placeholder="Digikey API Key"
              type="password"
            />
          </div>
          <div>
            <label className="text-sm font-medium">Client ID</label>
            <Input
              value={digikeyClientId}
              onChange={e => setDigikeyClientId(e.target.value)}
              placeholder="Digikey Client ID"
            />
          </div>
          <Button onClick={saveDigikey} disabled={saving}>
            <Save className="h-4 w-4 mr-1" /> Save Digikey Settings
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Mouser</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="text-sm font-medium">API Key</label>
            <Input
              value={mouserKey}
              onChange={e => setMouserKey(e.target.value)}
              placeholder="Mouser API Key"
              type="password"
            />
          </div>
          <Button onClick={saveMouser} disabled={saving}>
            <Save className="h-4 w-4 mr-1" /> Save Mouser Settings
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
