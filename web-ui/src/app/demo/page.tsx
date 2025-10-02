'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ExternalLink, Play, Trash2, RefreshCw } from 'lucide-react';
import { ProtectedRoute } from '@/components/protected-route';
import { useDemoStatus, useDemoActions } from '@/hooks/use-api';

export default function DemoEnvironmentPage() {
  // API hooks
  const { data: demoData, loading: statusLoading, error: statusError, refetch } = useDemoStatus();
  const { runDemoTime, runDemoNuke } = useDemoActions();

  const components = demoData?.components || [];

  const handleDemoTime = async () => {
    const result = await runDemoTime.mutate(undefined);
    if (result.success) {
      console.log('Demo time completed successfully');
      refetch(); // Refresh status after action
    }
  };

  const handleDemoNuke = async () => {
    const result = await runDemoNuke.mutate(undefined);
    if (result.success) {
      console.log('Demo nuke completed successfully');
      refetch(); // Refresh status after action
    }
  };

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
        <div className="relative space-y-6">
          {/* Header */}
          <div className="space-y-4">
            <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
              Demo Environment
            </h1>
            <p className="text-gray-600 dark:text-gray-400 text-lg">
              Manage and monitor your local development platform components. This demo environment
              includes all the tools needed for a complete development workflow.
            </p>
          </div>

          {/* Error Display */}
          {statusError && (
            <Card className="bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700">
              <CardContent className="pt-6">
                <p className="text-gray-800 dark:text-gray-200 text-sm">
                  Using offline mode - demo data may not be current. Error: {statusError}
                </p>
              </CardContent>
            </Card>
          )}

          {/* Demo Components Table */}
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-2xl">Components Status</CardTitle>
                  <CardDescription>
                    Current status of all demo environment components
                    {demoData?.timestamp && (
                      <span className="block mt-1 text-xs text-muted-foreground">
                        Last updated: {new Date(demoData.timestamp).toLocaleString()}
                      </span>
                    )}
                  </CardDescription>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => refetch()}
                  disabled={statusLoading}
                >
                  <RefreshCw className={`w-4 h-4 mr-2 ${statusLoading ? 'animate-spin' : ''}`} />
                  Refresh
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-gray-200 dark:border-gray-700">
                      <th className="text-left py-3 px-4 font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wide text-sm">
                        Component
                      </th>
                      <th className="text-left py-3 px-4 font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wide text-sm">
                        URL
                      </th>
                      <th className="text-left py-3 px-4 font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wide text-sm">
                        Status
                      </th>
                      <th className="text-left py-3 px-4 font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wide text-sm">
                        Credentials
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {statusLoading ? (
                      <tr>
                        <td colSpan={4} className="py-8 text-center text-muted-foreground">
                          <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2" />
                          Loading component status...
                        </td>
                      </tr>
                    ) : components.length === 0 ? (
                      <tr>
                        <td colSpan={4} className="py-8 text-center text-muted-foreground">
                          <div className="space-y-2">
                            <RefreshCw className="w-8 h-8 text-muted-foreground mx-auto" />
                            <p>No demo components found</p>
                          </div>
                        </td>
                      </tr>
                    ) : (
                      components.map((component) => (
                        <tr
                          key={component.name}
                          className="border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                        >
                          <td className="py-4 px-4 font-medium text-gray-900 dark:text-gray-100">
                            {component.name}
                          </td>
                          <td className="py-4 px-4">
                            <a
                              href={component.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-300 inline-flex items-center gap-1 font-medium"
                            >
                              {component.url}
                              <ExternalLink className="w-3 h-3" />
                            </a>
                          </td>
                          <td className="py-4 px-4">
                            <Badge variant="outline">
                              {component.status ? 'Healthy' : 'Unhealthy'}
                            </Badge>
                          </td>
                          <td className="py-4 px-4">
                            <code className="bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded text-sm font-mono text-gray-800 dark:text-gray-300">
                              {component.credentials}
                            </code>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>

          {/* Action Buttons */}
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
            <CardHeader>
              <CardTitle>Demo Environment Actions</CardTitle>
              <CardDescription>Manage your demo environment setup and cleanup</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex gap-4 flex-wrap">
                <Button
                  onClick={handleDemoTime}
                  disabled={runDemoTime.loading || runDemoNuke.loading}
                  className="bg-gray-600 hover:bg-gray-700 text-white shadow-lg"
                >
                  {runDemoTime.loading ? (
                    <>
                      <div className="w-4 h-4 border-2 border-white/20 border-t-white rounded-full animate-spin mr-2" />
                      Running...
                    </>
                  ) : (
                    <>
                      <Play className="w-4 h-4 mr-2" />
                      Run Demo Time
                    </>
                  )}
                </Button>

                <Button
                  onClick={handleDemoNuke}
                  disabled={runDemoTime.loading || runDemoNuke.loading}
                  variant="destructive"
                  className="shadow-lg"
                >
                  {runDemoNuke.loading ? (
                    <>
                      <div className="w-4 h-4 border-2 border-white/20 border-t-white rounded-full animate-spin mr-2" />
                      Nuking...
                    </>
                  ) : (
                    <>
                      <Trash2 className="w-4 h-4 mr-2" />
                      Run Demo Nuke
                    </>
                  )}
                </Button>
              </div>

              {/* Error display for actions */}
              {(runDemoTime.error || runDemoNuke.error) && (
                <div className="mt-4 p-3 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
                  <p className="text-gray-800 dark:text-gray-200 text-sm">
                    {runDemoTime.error || runDemoNuke.error}
                  </p>
                </div>
              )}

              <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
                <div className="flex items-start gap-3">
                  <div className="w-5 h-5 rounded-full bg-gray-500 flex-shrink-0 mt-0.5"></div>
                  <div className="space-y-2">
                    <p className="font-semibold text-gray-900 dark:text-gray-100">
                      Demo Environment Information
                    </p>
                    <div className="text-sm text-gray-800 dark:text-gray-200 space-y-1">
                      <p>
                        <strong>Demo Time:</strong> Install/reconcile the complete demo environment
                        with all components
                      </p>
                      <p>
                        <strong>Demo Nuke:</strong> Completely remove the demo environment and clean
                        up all resources
                      </p>
                      <p>
                        <strong>Prerequisites:</strong> Docker Desktop with Kubernetes enabled,
                        kubectl, and helm installed
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </ProtectedRoute>
  );
}
