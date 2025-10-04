'use client';

import { useEffect, useState } from 'react';
import { api, APIKeyInfo, APIKeyFull } from '@/lib/api';
import { CopyButton } from '@/components/copy-button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';

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
    <div className="bg-gray-800 rounded-lg p-6">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-xl font-semibold">API Keys</h2>
        <button
          onClick={() => setShowDialog(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
        >
          Generate New Key
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-900/50 border border-red-600 rounded text-red-200">
          {error}
        </div>
      )}

      {newKey && (
        <div className="mb-6 p-4 bg-yellow-900/50 border border-yellow-600 rounded">
          <h3 className="text-lg font-semibold text-yellow-200 mb-2">⚠️ Save Your API Key</h3>
          <p className="text-sm text-yellow-100 mb-3">
            This is the only time you&apos;ll see the full key. Copy it now and store it securely.
          </p>
          <div className="bg-gray-900 p-3 rounded font-mono text-sm break-all mb-3">
            {newKey.key}
          </div>
          <div className="flex gap-2">
            <CopyButton text={newKey.key} label="Copy Key" />
            <button
              onClick={() => setNewKey(null)}
              className="px-4 py-2 bg-gray-700 text-white rounded hover:bg-gray-600 transition-colors"
            >
              Done
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="text-center py-8 text-gray-400">Loading API keys...</div>
      ) : apiKeys.length === 0 ? (
        <div className="text-center py-8 text-gray-400">
          No API keys found. Generate one to get started.
        </div>
      ) : (
        <div className="space-y-3">
          {apiKeys.map((key) => (
            <div
              key={key.name}
              className="flex items-center justify-between p-4 bg-gray-700 rounded"
            >
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-2">
                  <h3 className="font-semibold">{key.name}</h3>
                  {isExpired(key.expires_at) && (
                    <span className="text-xs px-2 py-1 bg-red-600 rounded">Expired</span>
                  )}
                </div>
                <div className="text-sm text-gray-400 space-y-1">
                  <div>Key: {key.masked_key}</div>
                  <div className="flex gap-4">
                    <span>Created: {formatDate(key.created_at)}</span>
                    <span>Expires: {formatDate(key.expires_at)}</span>
                    {key.last_used_at && <span>Last used: {formatDate(key.last_used_at)}</span>}
                  </div>
                </div>
              </div>
              <button
                onClick={() => handleRevokeKey(key.name)}
                className="px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
              >
                Revoke
              </button>
            </div>
          ))}
        </div>
      )}

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
              <label className="block text-sm font-medium text-gray-300 mb-2">Key Name</label>
              <input
                type="text"
                value={keyName}
                onChange={(e) => setKeyName(e.target.value)}
                placeholder="e.g., production-api"
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400 focus:outline-none focus:border-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Expiry (days)</label>
              <input
                type="number"
                value={expiryDays}
                onChange={(e) => setExpiryDays(parseInt(e.target.value) || 90)}
                min="1"
                max="365"
                className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white focus:outline-none focus:border-blue-500"
              />
            </div>

            <div className="flex gap-2 pt-2">
              <button
                onClick={handleGenerateKey}
                disabled={generating}
                className="flex-1 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors disabled:opacity-50"
              >
                {generating ? 'Generating...' : 'Generate'}
              </button>
              <button
                onClick={() => setShowDialog(false)}
                className="px-4 py-2 bg-gray-700 text-white rounded hover:bg-gray-600 transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
