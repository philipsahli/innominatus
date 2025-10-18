'use client';

import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Network, RefreshCw, Eye, AlertCircle, Workflow } from 'lucide-react';
import { useApplications } from '@/hooks/use-api';
import { useRouter, usePathname } from 'next/navigation';
import { ProtectedRoute } from '@/components/protected-route';
import { GraphVisualization } from '@/components/graph-visualization';

export default function GraphListPage() {
  const router = useRouter();
  const pathname = usePathname();
  const { data: applications, loading, error, refetch } = useApplications();

  // Extract app name from pathname like /graph/my-app3
  const [appName, setAppName] = useState<string | null>(null);

  useEffect(() => {
    const parts = pathname.split('/').filter(Boolean);
    if (parts.length === 2 && parts[0] === 'graph') {
      setAppName(parts[1]);
    } else {
      setAppName(null);
    }
  }, [pathname]);

  return (
    <ProtectedRoute>
      {/* If we have an app name in the URL, show the graph visualization */}
      {appName ? (
        <GraphVisualization app={appName} />
      ) : (
        <div className="p-6 space-y-6">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold tracking-tight flex items-center gap-3">
                <Network className="w-8 h-8 text-blue-500" />
                Workflow Graphs
              </h1>
              <p className="text-muted-foreground mt-2">
                Visualize orchestration workflows, steps, and resource dependencies
              </p>
            </div>
            <Button
              variant="outline"
              onClick={() => refetch()}
              disabled={loading}
              className="gap-2"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          </div>

          {/* Error Display */}
          {error && (
            <Card className="border-amber-200 bg-amber-50 dark:border-amber-700 dark:bg-amber-950">
              <CardContent className="pt-6 flex items-center gap-3">
                <AlertCircle className="w-5 h-5 text-amber-600" />
                <p className="text-sm text-amber-800 dark:text-amber-200">
                  Failed to load applications: {error}
                </p>
              </CardContent>
            </Card>
          )}

          {/* Application Grid */}
          {loading ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {[...Array(6)].map((_, i) => (
                <Card key={i} className="animate-pulse">
                  <CardHeader>
                    <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded"></div>
                      <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-5/6"></div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : applications && applications.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {applications.map((app) => (
                <Card
                  key={app.name}
                  className="hover:shadow-lg transition-shadow cursor-pointer group"
                  onClick={() => router.push(`/graph/${app.name}`)}
                >
                  <CardHeader>
                    <CardTitle className="flex items-center justify-between">
                      <span className="flex items-center gap-2">
                        <Workflow className="w-5 h-5 text-blue-500" />
                        {app.name}
                      </span>
                      <Badge variant="outline" className="text-xs">
                        {app.status || 'active'}
                      </Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      <p className="text-sm text-muted-foreground line-clamp-2">
                        View workflow execution graph with real-time state updates
                      </p>
                      <div className="flex items-center justify-between pt-2 border-t">
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <Network className="w-3 h-3" />
                          <span>Graph View</span>
                        </div>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="group-hover:bg-blue-50 dark:group-hover:bg-blue-950"
                          onClick={(e) => {
                            e.stopPropagation();
                            router.push(`/graph/${app.name}`);
                          }}
                        >
                          <Eye className="w-4 h-4 mr-2" />
                          View
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <Card>
              <CardContent className="pt-6 text-center py-12">
                <Network className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                <p className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                  No Applications Found
                </p>
                <p className="text-sm text-muted-foreground">
                  Deploy an application to see its workflow graph visualization
                </p>
              </CardContent>
            </Card>
          )}

          {/* Info Card */}
          <Card className="bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800">
            <CardHeader>
              <CardTitle className="text-sm font-medium text-blue-900 dark:text-blue-100">
                About Workflow Graphs
              </CardTitle>
            </CardHeader>
            <CardContent className="text-sm text-blue-800 dark:text-blue-200 space-y-2">
              <p>
                Workflow graphs visualize the orchestration flow from specifications through
                workflows, steps, and resources.
              </p>
              <ul className="list-disc list-inside space-y-1 ml-2">
                <li>Real-time state updates via Server-Sent Events</li>
                <li>Color-coded nodes by type and state</li>
                <li>Interactive graph with zoom and pan controls</li>
                <li>Export graphs as JSON for analysis</li>
              </ul>
            </CardContent>
          </Card>
        </div>
      )}
    </ProtectedRoute>
  );
}
