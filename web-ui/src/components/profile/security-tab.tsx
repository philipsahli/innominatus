'use client';

import { useEffect, useState } from 'react';
import { api, APIKeyInfo, APIKeyFull } from '@/lib/api';
import { CopyButton } from '@/components/copy-button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Key, Plus, Trash2, AlertCircle } from 'lucide-react';

export default function SecurityTab() {
  const [apiKeys, setApiKeys] = useState<APIKeyInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDialog, setShowDialog] = useState(false);
  const [newKey, setNewKey] = useState<APIKeyFull | null>(null);
  const [keyName, setKeyName] = useState('');
  const [expiryDays, setExpiryDays] = useState(90);
  const [generating, setGenerating] = useState(false);

  useEffect(() => {
    loadAPIKeys();
  }, []);

  const loadAPIKeys = async () => {
    setLoading(true);
    setError(null);

    const response = await api.getAPIKeys();
    if (response.success && response.data) {
      setApiKeys(response.data);
    } else {
      setError(response.error || 'Failed to load API keys');
    }

    setLoading(false);
  };

  const handleGenerateKey = async () => {
    if (!keyName.trim()) {
      setError('API key name is required');
      return;
    }

    setGenerating(true);
    setError(null);

    const response = await api.generateAPIKey(keyName, expiryDays);
    if (response.success && response.data) {
      setNewKey(response.data);
      setShowDialog(false);
      setKeyName('');
      setExpiryDays(90);
      await loadAPIKeys();
    } else {
      setError(response.error || 'Failed to generate API key');
    }

    setGenerating(false);
  };

  const handleRevokeKey = async (name: string) => {
    if (!confirm(`Are you sure you want to revoke the API key "${name}"?`)) {
      return;
    }

    const response = await api.revokeAPIKey(name);
    if (response.success) {
      await loadAPIKeys();
    } else {
      setError(response.error || 'Failed to revoke API key');
    }
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr) return 'Never';
    return new Date(dateStr).toLocaleDateString();
  };

  const isExpired = (dateStr: string) => {
    return new Date(dateStr) < new Date();
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-center">
          <CardTitle className="text-xl flex items-center gap-2">
            <Key className="w-5 h-5" />
            API Keys
          </CardTitle>
          <Button onClick={() => setShowDialog(true)}>
            <Plus className="w-4 h-4 mr-2" />
            Generate New Key
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <div className="flex items-center gap-2 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-red-800 dark:text-red-200">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            <p className="text-sm">{error}</p>
          </div>
        )}

        {newKey && (
          <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded">
            <h3 className="text-lg font-semibold text-yellow-800 dark:text-yellow-200 mb-2 flex items-center gap-2">
              <AlertCircle className="w-5 h-5" />
              Save Your API Key
            </h3>
            <p className="text-sm text-yellow-700 dark:text-yellow-300 mb-3">
              This is the only time you&apos;ll see the full key. Copy it now and store it securely.
            </p>
            <div className="bg-white dark:bg-gray-900 p-3 rounded border border-gray-200 dark:border-gray-700 font-mono text-sm break-all mb-3 text-gray-900 dark:text-gray-100">
              {newKey.key}
            </div>
            <div className="flex gap-2">
              <CopyButton text={newKey.key} label="Copy Key" />
              <Button onClick={() => setNewKey(null)} variant="outline">
                Done
              </Button>
            </div>
          </div>
        )}

        {loading ? (
          <div className="text-center py-8 text-muted-foreground">Loading API keys...</div>
        ) : apiKeys.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            No API keys found. Generate one to get started.
          </div>
        ) : (
          <div className="space-y-3">
            {apiKeys.map((key) => (
              <Card key={key.name}>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <Key className="w-4 h-4 text-gray-500 dark:text-gray-400" />
                        <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                          {key.name}
                        </h3>
                        {isExpired(key.expires_at) && (
                          <Badge variant="destructive" className="text-xs">
                            Expired
                          </Badge>
                        )}
                      </div>
                      <div className="text-sm text-muted-foreground space-y-1 ml-7">
                        <div className="font-mono">{key.masked_key}</div>
                        <div className="flex gap-4">
                          <span>Created: {formatDate(key.created_at)}</span>
                          <span>Expires: {formatDate(key.expires_at)}</span>
                          {key.last_used_at && (
                            <span>Last used: {formatDate(key.last_used_at)}</span>
                          )}
                        </div>
                      </div>
                    </div>
                    <Button
                      onClick={() => handleRevokeKey(key.name)}
                      variant="destructive"
                      size="sm"
                    >
                      <Trash2 className="w-4 h-4 mr-1" />
                      Revoke
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </CardContent>

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Generate New API Key</DialogTitle>
            <DialogDescription>
              Create a new API key for programmatic access. The key will be shown only once.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2">Key Name</label>
              <Input
                type="text"
                value={keyName}
                onChange={(e) => setKeyName(e.target.value)}
                placeholder="e.g., production-api"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">Expiry (days)</label>
              <Input
                type="number"
                value={expiryDays}
                onChange={(e) => setExpiryDays(parseInt(e.target.value) || 90)}
                min="1"
                max="365"
              />
            </div>

            <div className="flex gap-2 pt-2">
              <Button onClick={handleGenerateKey} disabled={generating} className="flex-1">
                {generating ? 'Generating...' : 'Generate'}
              </Button>
              <Button onClick={() => setShowDialog(false)} variant="outline">
                Cancel
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
