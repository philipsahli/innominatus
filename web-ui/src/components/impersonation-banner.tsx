'use client';

import { useEffect, useState } from 'react';
import { api, ImpersonationStatus } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { AlertCircle, UserX } from 'lucide-react';
import { Alert } from '@/components/ui/alert';

export function ImpersonationBanner() {
  const [status, setStatus] = useState<ImpersonationStatus | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchStatus = async () => {
    const response = await api.getImpersonationStatus();
    if (response.success && response.data) {
      setStatus(response.data);
    }
  };

  useEffect(() => {
    fetchStatus();
  }, []);

  // Only poll when actually impersonating
  useEffect(() => {
    if (!status?.is_impersonating) {
      return;
    }

    // Poll every 30 seconds to keep status updated (reduced from 5s)
    const interval = setInterval(fetchStatus, 30000);
    return () => clearInterval(interval);
  }, [status?.is_impersonating]);

  const handleStopImpersonation = async () => {
    setLoading(true);
    try {
      const response = await api.stopImpersonation();
      if (response.success) {
        await fetchStatus();
        // Reload the page to refresh all user-specific data
        window.location.reload();
      } else {
        alert(`Failed to stop impersonation: ${response.error || 'Unknown error'}`);
      }
    } catch (error) {
      alert(`Error stopping impersonation: ${error}`);
    } finally {
      setLoading(false);
    }
  };

  if (!status?.is_impersonating) {
    return null;
  }

  return (
    <Alert className="border-yellow-500 bg-yellow-50 dark:bg-yellow-900/20 mb-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <AlertCircle className="h-5 w-5 text-yellow-600" />
          <div>
            <p className="font-semibold text-yellow-800 dark:text-yellow-200">Impersonating User</p>
            <p className="text-sm text-yellow-700 dark:text-yellow-300">
              You are currently acting as <strong>{status.impersonated_user?.username}</strong> (
              {status.impersonated_user?.team} - {status.impersonated_user?.role})
            </p>
            <p className="text-xs text-yellow-600 dark:text-yellow-400 mt-1">
              Original user: <strong>{status.original_user?.username}</strong>
            </p>
          </div>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={handleStopImpersonation}
          disabled={loading}
          className="gap-2 border-yellow-600 text-yellow-700 hover:bg-yellow-100 dark:text-yellow-300 dark:hover:bg-yellow-900/40"
        >
          <UserX className="h-4 w-4" />
          Stop Impersonating
        </Button>
      </div>
    </Alert>
  );
}
