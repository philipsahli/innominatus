'use client';

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { Loader2, Users, Plus, Trash2, Key, Shield, UserCircle, Copy, Check } from 'lucide-react';
import { AdminRouteProtection } from '@/components/admin-route-protection';
import { useToast } from '@/hooks/use-toast';

interface User {
  username: string;
  team: string;
  role: string;
}

interface APIKey {
  name: string;
  key: string;
  created_at: string;
  expires_at: string;
  last_used_at?: string;
}

export default function UsersPage() {
  return (
    <AdminRouteProtection>
      <UsersPageContent />
    </AdminRouteProtection>
  );
}

function UsersPageContent() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showKeysDialog, setShowKeysDialog] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [keysLoading, setKeysLoading] = useState(false);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const { toast } = useToast();

  // Form state for create user
  const [newUser, setNewUser] = useState({
    username: '',
    password: '',
    team: '',
    role: 'user',
  });

  // Form state for generate key
  const [newKey, setNewKey] = useState({
    name: '',
    expiry_days: 90,
  });

  useEffect(() => {
    fetchUsers();
  }, []);

  async function fetchUsers() {
    try {
      setLoading(true);
      const response = await api.listUsers();
      if (response.success && Array.isArray(response.data)) {
        setUsers(response.data);
        setError(null);
      } else {
        setError(response.error || 'Failed to load users');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users');
    } finally {
      setLoading(false);
    }
  }

  async function handleCreateUser() {
    if (!newUser.username || !newUser.password || !newUser.team) {
      toast({
        title: 'Validation Error',
        description: 'Username, password, and team are required',
        variant: 'destructive',
      });
      return;
    }

    try {
      const response = await fetch('/api/admin/users', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(newUser),
      });

      if (response.ok) {
        toast({
          title: 'User Created',
          description: `User '${newUser.username}' was created successfully`,
        });
        setShowCreateDialog(false);
        setNewUser({ username: '', password: '', team: '', role: 'user' });
        fetchUsers();
      } else {
        const error = await response.text();
        toast({
          title: 'Creation Failed',
          description: error || 'Failed to create user',
          variant: 'destructive',
        });
      }
    } catch (err) {
      toast({
        title: 'Error',
        description: err instanceof Error ? err.message : 'Failed to create user',
        variant: 'destructive',
      });
    }
  }

  async function handleDeleteUser() {
    if (!selectedUser) return;

    try {
      const response = await fetch(`/api/admin/users/${selectedUser.username}`, {
        method: 'DELETE',
        credentials: 'include',
      });

      if (response.ok) {
        toast({
          title: 'User Deleted',
          description: `User '${selectedUser.username}' was deleted successfully`,
        });
        setShowDeleteDialog(false);
        setSelectedUser(null);
        fetchUsers();
      } else {
        const error = await response.text();
        toast({
          title: 'Deletion Failed',
          description: error || 'Failed to delete user',
          variant: 'destructive',
        });
      }
    } catch (err) {
      toast({
        title: 'Error',
        description: err instanceof Error ? err.message : 'Failed to delete user',
        variant: 'destructive',
      });
    }
  }

  async function fetchUserKeys(username: string) {
    setKeysLoading(true);
    try {
      const response = await fetch(`/api/admin/users/${username}/api-keys`, {
        credentials: 'include',
      });

      if (response.ok) {
        const data = await response.json();
        setApiKeys(data.api_keys || []);
      } else {
        toast({
          title: 'Failed to Load Keys',
          description: 'Could not retrieve API keys for this user',
          variant: 'destructive',
        });
      }
    } catch (err) {
      toast({
        title: 'Error',
        description: 'Failed to load API keys',
        variant: 'destructive',
      });
    } finally {
      setKeysLoading(false);
    }
  }

  async function handleGenerateKey() {
    if (!selectedUser || !newKey.name) {
      toast({
        title: 'Validation Error',
        description: 'Key name is required',
        variant: 'destructive',
      });
      return;
    }

    try {
      const response = await fetch(`/api/admin/users/${selectedUser.username}/api-keys`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(newKey),
      });

      if (response.ok) {
        const data = await response.json();
        toast({
          title: 'API Key Generated',
          description: 'Save this key now - it will not be shown again',
        });

        // Show the full key temporarily
        const tempKey = { ...data, name: newKey.name, created_at: new Date().toISOString() };
        setApiKeys([...apiKeys, tempKey]);
        setNewKey({ name: '', expiry_days: 90 });

        // Auto-copy the key
        if (data.key) {
          navigator.clipboard.writeText(data.key);
          setCopiedKey(data.key);
          setTimeout(() => setCopiedKey(null), 3000);
        }

        // Refresh keys after a delay to show masked version
        setTimeout(() => fetchUserKeys(selectedUser.username), 2000);
      } else {
        const error = await response.text();
        toast({
          title: 'Generation Failed',
          description: error || 'Failed to generate API key',
          variant: 'destructive',
        });
      }
    } catch (err) {
      toast({
        title: 'Error',
        description: err instanceof Error ? err.message : 'Failed to generate API key',
        variant: 'destructive',
      });
    }
  }

  async function handleRevokeKey(keyName: string) {
    if (!selectedUser) return;

    try {
      const response = await fetch(
        `/api/admin/users/${selectedUser.username}/api-keys/${keyName}`,
        {
          method: 'DELETE',
          credentials: 'include',
        }
      );

      if (response.ok) {
        toast({
          title: 'API Key Revoked',
          description: `Key '${keyName}' was revoked successfully`,
        });
        fetchUserKeys(selectedUser.username);
      } else {
        const error = await response.text();
        toast({
          title: 'Revocation Failed',
          description: error || 'Failed to revoke API key',
          variant: 'destructive',
        });
      }
    } catch (err) {
      toast({
        title: 'Error',
        description: err instanceof Error ? err.message : 'Failed to revoke API key',
        variant: 'destructive',
      });
    }
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text);
    setCopiedKey(text);
    setTimeout(() => setCopiedKey(null), 2000);
    toast({
      title: 'Copied',
      description: 'API key copied to clipboard',
    });
  }

  if (loading) {
    return (
      <div className="p-6 flex items-center justify-center min-h-[400px]">
        <div className="flex flex-col items-center gap-2">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Loading users...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-md bg-destructive/15 p-4 text-destructive">
          <h3 className="font-semibold">Error Loading Users</h3>
          <p className="text-sm mt-1">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Users className="h-8 w-8" />
            User Management
          </h1>
          <p className="text-muted-foreground">Manage platform users and permissions</p>
        </div>
        <Button onClick={() => setShowCreateDialog(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Create User
        </Button>
      </div>

      {/* Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Users</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{users.length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">Admins</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-amber-600">
              {users.filter((u) => u.role === 'admin').length}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">Standard Users</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-blue-600">
              {users.filter((u) => u.role === 'user').length}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Users Table */}
      <Card>
        <CardHeader>
          <CardTitle>All Users</CardTitle>
          <CardDescription>Platform user accounts and their roles</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Team</TableHead>
                <TableHead>Role</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((user) => (
                <TableRow key={user.username}>
                  <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                      <UserCircle className="h-4 w-4 text-muted-foreground" />
                      {user.username}
                    </div>
                  </TableCell>
                  <TableCell>{user.team}</TableCell>
                  <TableCell>
                    <Badge variant={user.role === 'admin' ? 'default' : 'secondary'}>
                      {user.role === 'admin' && <Shield className="h-3 w-3 mr-1" />}
                      {user.role}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setSelectedUser(user);
                          setShowKeysDialog(true);
                          fetchUserKeys(user.username);
                        }}
                      >
                        <Key className="h-4 w-4 mr-1" />
                        API Keys
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => {
                          setSelectedUser(user);
                          setShowDeleteDialog(true);
                        }}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Create User Dialog */}
      <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New User</DialogTitle>
            <DialogDescription>Add a new user to the platform</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={newUser.username}
                onChange={(e) => setNewUser({ ...newUser, username: e.target.value })}
                placeholder="alice"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={newUser.password}
                onChange={(e) => setNewUser({ ...newUser, password: e.target.value })}
                placeholder="Enter password"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="team">Team</Label>
              <Input
                id="team"
                value={newUser.team}
                onChange={(e) => setNewUser({ ...newUser, team: e.target.value })}
                placeholder="platform"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="role">Role</Label>
              <Select value={newUser.role} onValueChange={(value) => setNewUser({ ...newUser, role: value })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="user">User</SelectItem>
                  <SelectItem value="admin">Admin</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreateUser}>Create User</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete User Dialog */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete User</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete user '{selectedUser?.username}'? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeleteDialog(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDeleteUser}>
              Delete User
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* API Keys Dialog */}
      <Dialog open={showKeysDialog} onOpenChange={setShowKeysDialog}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>API Keys - {selectedUser?.username}</DialogTitle>
            <DialogDescription>Manage API keys for this user</DialogDescription>
          </DialogHeader>

          {keysLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin" />
            </div>
          ) : (
            <div className="space-y-4">
              {/* Generate New Key */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm">Generate New API Key</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="grid gap-2">
                    <Label htmlFor="keyName">Key Name</Label>
                    <Input
                      id="keyName"
                      value={newKey.name}
                      onChange={(e) => setNewKey({ ...newKey, name: e.target.value })}
                      placeholder="my-cli-key"
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="expiryDays">Expiry (days)</Label>
                    <Input
                      id="expiryDays"
                      type="number"
                      value={newKey.expiry_days}
                      onChange={(e) =>
                        setNewKey({ ...newKey, expiry_days: parseInt(e.target.value) || 90 })
                      }
                    />
                  </div>
                  <Button onClick={handleGenerateKey} className="w-full">
                    <Plus className="h-4 w-4 mr-2" />
                    Generate Key
                  </Button>
                </CardContent>
              </Card>

              {/* Existing Keys */}
              <div className="space-y-2">
                <h4 className="text-sm font-medium">Existing Keys ({apiKeys.length})</h4>
                {apiKeys.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No API keys for this user</p>
                ) : (
                  <div className="space-y-2">
                    {apiKeys.map((key) => (
                      <Card key={key.name}>
                        <CardContent className="p-4">
                          <div className="flex items-center justify-between">
                            <div className="space-y-1">
                              <div className="flex items-center gap-2">
                                <Key className="h-4 w-4 text-muted-foreground" />
                                <span className="font-medium">{key.name}</span>
                              </div>
                              <div className="flex items-center gap-2 text-xs text-muted-foreground font-mono">
                                <span>{key.key}</span>
                                <Button
                                  size="sm"
                                  variant="ghost"
                                  className="h-6 w-6 p-0"
                                  onClick={() => copyToClipboard(key.key)}
                                >
                                  {copiedKey === key.key ? (
                                    <Check className="h-3 w-3 text-green-600" />
                                  ) : (
                                    <Copy className="h-3 w-3" />
                                  )}
                                </Button>
                              </div>
                              <div className="text-xs text-muted-foreground">
                                Created: {new Date(key.created_at).toLocaleDateString()}
                              </div>
                            </div>
                            <Button
                              size="sm"
                              variant="destructive"
                              onClick={() => handleRevokeKey(key.name)}
                            >
                              Revoke
                            </Button>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowKeysDialog(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
