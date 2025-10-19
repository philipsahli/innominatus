'use client';

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Select } from '@/components/ui/select';
import { UserCheck, AlertTriangle } from 'lucide-react';

interface User {
  username: string;
  team: string;
  role: string;
  is_admin: boolean;
}

export function AdminImpersonation() {
  const [users, setUsers] = useState<User[]>([]);
  const [selectedUser, setSelectedUser] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchUsers();
  }, []);

  const fetchUsers = async () => {
    try {
      const response = await api.listUsers();
      if (response.success && response.data) {
        setUsers(response.data);
      } else {
        setError(response.error || 'Failed to load users');
      }
    } catch (err) {
      setError('Failed to load users');
    }
  };

  const handleImpersonate = async () => {
    if (!selectedUser) {
      alert('Please select a user to impersonate');
      return;
    }

    if (
      !confirm(
        `Are you sure you want to impersonate user "${selectedUser}"?\n\nYou will see the application as this user would see it, with their permissions and access level.`
      )
    ) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await api.startImpersonation(selectedUser);
      if (response.success) {
        // Reload page to refresh all user-specific data
        window.location.reload();
      } else {
        setError(response.error || 'Failed to start impersonation');
      }
    } catch (err) {
      setError('Failed to start impersonation');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <UserCheck className="h-5 w-5" />
          User Impersonation
        </CardTitle>
        <CardDescription>
          Temporarily act as another user to troubleshoot issues or view their experience
        </CardDescription>
      </CardHeader>
      <CardContent>
        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md flex items-start gap-2">
            <AlertTriangle className="h-5 w-5 text-red-600 mt-0.5" />
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">Select User to Impersonate</label>
            <select
              value={selectedUser}
              onChange={(e) => setSelectedUser(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-800 text-sm"
              disabled={loading}
            >
              <option value="">-- Select a user --</option>
              {users.map((user) => (
                <option key={user.username} value={user.username}>
                  {user.username} ({user.team} - {user.role})
                  {user.is_admin ? ' - ADMIN' : ''}
                </option>
              ))}
            </select>
          </div>

          <Button onClick={handleImpersonate} disabled={!selectedUser || loading} className="w-full">
            {loading ? 'Starting Impersonation...' : 'Start Impersonation'}
          </Button>

          <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md">
            <p className="text-sm text-blue-800 dark:text-blue-200">
              <strong>Note:</strong> While impersonating, you will see a yellow banner at the top of
              the page. Click &ldquo;Stop Impersonating&rdquo; to return to your normal session.
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
