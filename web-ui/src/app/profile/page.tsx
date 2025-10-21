'use client';

import { useEffect, useState } from 'react';
import { ProtectedRoute } from '@/components/protected-route';
import { api, UserProfile } from '@/lib/api';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Users, User, Shield } from 'lucide-react';
import SecurityTab from '@/components/profile/security-tab';

export default function ProfilePage() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadProfile();
  }, []);

  const loadProfile = async () => {
    setLoading(true);
    setError(null);

    const response = await api.getProfile();
    if (response.success && response.data) {
      setProfile(response.data);
    } else {
      setError(response.error || 'Failed to load profile');
    }

    setLoading(false);
  };

  if (loading) {
    return (
      <ProtectedRoute>
        <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
          <div className="text-center py-8 text-gray-600 dark:text-gray-400">
            Loading profile...
          </div>
        </div>
      </ProtectedRoute>
    );
  }

  if (error) {
    return (
      <ProtectedRoute>
        <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
          <div className="text-center py-8 text-red-600 dark:text-red-400">{error}</div>
        </div>
      </ProtectedRoute>
    );
  }

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 rounded-lg bg-gray-200 dark:bg-gray-700">
              <User className="w-6 h-6 text-gray-900 dark:text-gray-100" />
            </div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Profile</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 ml-14">
            Manage your account settings and security
          </p>
        </div>

        <Tabs defaultValue="overview" className="space-y-4">
          <TabsList>
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="security">Security</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-6">
            {/* Account Information Card */}
            <Card>
              <CardHeader>
                <CardTitle className="text-xl flex items-center gap-2">
                  <User className="w-5 h-5" />
                  Account Information
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-6">
                  <div>
                    <label className="text-sm text-muted-foreground flex items-center gap-1">
                      <User className="w-3 h-3" />
                      Username
                    </label>
                    <p className="mt-2 font-medium text-gray-900 dark:text-gray-100">
                      {profile?.username}
                    </p>
                  </div>

                  <div>
                    <label className="text-sm text-muted-foreground flex items-center gap-1">
                      <Shield className="w-3 h-3" />
                      Role
                    </label>
                    <div className="mt-2">
                      <Badge variant="outline" className="capitalize">
                        {profile?.role}
                      </Badge>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Team Membership Card */}
            <Card>
              <CardHeader>
                <CardTitle className="text-xl flex items-center gap-2">
                  <Users className="w-5 h-5" />
                  Team Membership
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div>
                    <label className="text-sm text-muted-foreground flex items-center gap-1 mb-3">
                      Current Team
                    </label>
                    <div className="flex items-center gap-3 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border-2 border-blue-500/50">
                      <div className="w-12 h-12 rounded-full bg-blue-600 flex items-center justify-center flex-shrink-0">
                        <Users className="w-6 h-6 text-white" />
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <p className="font-semibold text-lg text-gray-900 dark:text-gray-100">
                            {profile?.team}
                          </p>
                          <Badge variant="default" className="bg-blue-600 text-white">
                            Active
                          </Badge>
                        </div>
                        <p className="text-sm text-muted-foreground mt-1">
                          You are currently working under this team
                        </p>
                      </div>
                    </div>
                  </div>

                  <div className="text-sm text-muted-foreground bg-gray-100 dark:bg-gray-800 p-3 rounded border border-gray-200 dark:border-gray-700">
                    <p className="flex items-center gap-2">
                      <span>ℹ️</span>
                      <span>
                        Team switching and multi-team support will be available in a future update.
                      </span>
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="security">
            <SecurityTab />
          </TabsContent>
        </Tabs>
      </div>
    </ProtectedRoute>
  );
}
