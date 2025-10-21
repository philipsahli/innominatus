'use client';

import { ProtectedRoute } from '@/components/protected-route';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import Link from 'next/link';
import {
  Server,
  Activity,
  Users,
  RefreshCw,
  TrendingUp,
  Database,
  Cloud,
  Clock,
  ArrowUpRight,
  Zap,
} from 'lucide-react';
import { useApplications, useWorkflows, useStats, useResources } from '@/hooks/use-api';
import type { ResourceInstance } from '@/lib/api';

function getStatusBadge(status: string) {
  return (
    <Badge variant="outline">
      <Clock className="w-3 h-3 mr-1" />
      {status}
    </Badge>
  );
}

function getResourceStateBadge(state: string) {
  const variants: Record<
    string,
    { variant: 'default' | 'secondary' | 'destructive' | 'outline'; className?: string }
  > = {
    active: {
      variant: 'default',
      className: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    },
    provisioning: { variant: 'secondary' },
    failed: { variant: 'destructive' },
    terminated: { variant: 'outline', className: 'text-gray-500' },
  };
  const config = variants[state] || { variant: 'outline' };
  return (
    <Badge variant={config.variant} className={config.className}>
      {state}
    </Badge>
  );
}

export default function Dashboard() {
  const {
    data: stats,
    loading: statsLoading,
    error: statsError,
    refetch: refetchStats,
  } = useStats();
  const {
    data: applications,
    loading: appsLoading,
    error: appsError,
    refetch: refetchApps,
  } = useApplications();
  const {
    data: workflows,
    loading: workflowsLoading,
    error: workflowsError,
    refetch: refetchWorkflows,
  } = useWorkflows();
  const {
    data: resourcesData,
    loading: resourcesLoading,
    error: resourcesError,
    refetch: refetchResources,
  } = useResources();

  const handleRefresh = () => {
    refetchStats();
    refetchApps();
    refetchWorkflows();
    refetchResources();
  };

  const displayStats = stats || { applications: 0, workflows: 0, resources: 0, users: 0 };
  const displayApps = applications || [];
  const displayWorkflows = workflows?.data || [];

  // Flatten resources data and sort by most recent
  const allResources: ResourceInstance[] = resourcesData ? Object.values(resourcesData).flat() : [];
  const recentResources = allResources
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    .slice(0, 6);

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900">
        <div className="p-6 space-y-8">
          {/* Hero Section */}
          <div className="relative">
            <div className="p-8 rounded-2xl border bg-white dark:bg-gray-900">
              <div className="flex items-center justify-between">
                <div className="space-y-2">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-gray-200 dark:bg-gray-700">
                      <Zap className="w-6 h-6 text-gray-900 dark:text-gray-100" />
                    </div>
                    <h1 className="text-4xl font-bold text-gray-900 dark:text-gray-100">
                      Dashboard
                    </h1>
                  </div>
                  <p className="text-lg text-muted-foreground max-w-2xl">
                    Welcome to the IDP Orchestrator - monitor your applications, workflows, and
                    infrastructure at a glance
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <div className="hidden md:flex items-center gap-2 px-4 py-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-200">
                    <div className="w-2 h-2 rounded-full bg-gray-500 animate-pulse"></div>
                    <span className="text-sm font-medium">System Healthy</span>
                  </div>
                  <Button
                    variant="outline"
                    onClick={handleRefresh}
                    disabled={statsLoading || appsLoading || workflowsLoading}
                    className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700"
                  >
                    <RefreshCw
                      className={`w-4 h-4 mr-2 ${statsLoading || appsLoading || workflowsLoading ? 'animate-spin' : ''}`}
                    />
                    Refresh
                  </Button>
                </div>
              </div>
            </div>
          </div>

          {/* Error Display */}
          {(statsError || appsError || workflowsError) && (
            <Card className="border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
              <CardContent className="pt-6">
                <p className="text-gray-800 dark:text-gray-200 text-sm">
                  API connection issues - some data may not be current.
                  {statsError && ` Stats: ${statsError}`}
                  {appsError && ` Apps: ${appsError}`}
                  {workflowsError && ` Workflows: ${workflowsError}`}
                </p>
              </CardContent>
            </Card>
          )}

          {/* Debug Info for Workflows */}
          {workflowsError && (
            <Card className="border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
              <CardContent className="pt-6">
                <p className="text-gray-800 dark:text-gray-200 text-sm">
                  <strong>Workflows Debug:</strong> {workflowsError}
                  <br />
                  <strong>Workflows data:</strong> {JSON.stringify(workflows)}
                  <br />
                  <strong>Display workflows length:</strong> {displayWorkflows.length}
                </p>
              </CardContent>
            </Card>
          )}

          {/* Stats Overview */}
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
            <Card className="relative overflow-hidden border shadow-lg bg-white dark:bg-gray-800">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 relative z-10">
                <CardTitle className="text-sm font-medium">Applications</CardTitle>
                <Server className="h-5 w-5" />
              </CardHeader>
              <CardContent className="relative z-10">
                <div className="text-3xl font-bold">{displayStats.applications}</div>
                <div className="flex items-center gap-2 mt-2">
                  <TrendingUp className="h-3 w-3 " />
                  <p className="text-xs">Total applications</p>
                </div>
              </CardContent>
            </Card>

            <Card className="relative overflow-hidden border shadow-lg bg-white dark:bg-gray-800">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 relative z-10">
                <CardTitle className="text-sm font-medium ">Active Workflows</CardTitle>
                <Activity className="h-5 w-5 " />
              </CardHeader>
              <CardContent className="relative z-10">
                <div className="text-3xl font-bold">{displayStats.workflows}</div>
                <div className="flex items-center gap-2 mt-2">
                  <Zap className="h-3 w-3" />
                  <p className="text-xs">Running workflows</p>
                </div>
              </CardContent>
            </Card>

            <Card className="relative overflow-hidden border shadow-lg bg-white dark:bg-gray-800">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 relative z-10">
                <CardTitle className="text-sm font-medium ">Resources</CardTitle>
                <Database className="h-5 w-5 " />
              </CardHeader>
              <CardContent className="relative z-10">
                <div className="text-3xl font-bold">{displayStats.resources}</div>
                <div className="flex items-center gap-2 mt-2">
                  <Cloud className="h-3 w-3 " />
                  <p className="text-xs">Across all environments</p>
                </div>
              </CardContent>
            </Card>

            <Card className="relative overflow-hidden border shadow-lg bg-white dark:bg-gray-800">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 relative z-10">
                <CardTitle className="text-sm font-medium ">Users</CardTitle>
                <Users className="h-5 w-5 " />
              </CardHeader>
              <CardContent className="relative z-10">
                <div className="text-3xl font-bold">{displayStats.users}</div>
                <div className="flex items-center gap-2 mt-2">
                  <ArrowUpRight className="h-3 w-3 " />
                  <p className="text-xs ">Platform users</p>
                </div>
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-8 lg:grid-cols-2">
            {/* Applications Table */}
            <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
              <CardHeader className="pb-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-gray-100 dark:bg-gray-700">
                      <Server className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                    </div>
                    <CardTitle className="text-xl">Applications</CardTitle>
                  </div>
                  <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                    <ArrowUpRight className="w-4 h-4" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                {appsLoading ? (
                  <div className="flex items-center justify-center p-12">
                    <RefreshCw className="w-6 h-6 animate-spin mr-2 text-muted-foreground" />
                    <span className="text-muted-foreground">Loading applications...</span>
                  </div>
                ) : displayApps.length === 0 ? (
                  <div className="flex items-center justify-center p-12 text-center">
                    <div className="space-y-2">
                      <Server className="w-8 h-8 text-muted-foreground mx-auto" />
                      <p className="text-muted-foreground">No applications found</p>
                    </div>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {displayApps.map((app) => (
                      <div
                        key={app.name}
                        className="flex items-center justify-between p-4 rounded-lg border bg-white dark:bg-gray-800 hover:shadow-md transition-all"
                      >
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-600 flex items-center justify-center font-semibold text-xs text-gray-900 dark:text-gray-100">
                            {app.name.charAt(0).toUpperCase()}
                          </div>
                          <div>
                            <p className="font-semibold text-sm">{app.name}</p>
                            <p className="text-xs text-muted-foreground">{app.lastUpdated}</p>
                          </div>
                        </div>
                        <div className="flex items-center gap-3">
                          <Badge variant="outline" className="text-xs">
                            {app.environment}
                          </Badge>
                          {getStatusBadge(app.status)}
                          <div className="text-xs text-muted-foreground min-w-0">
                            {app.resources} resources
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Recent Workflows */}
            <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
              <CardHeader className="pb-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-gray-100 dark:bg-gray-700">
                      <Activity className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                    </div>
                    <CardTitle className="text-xl">Recent Workflows</CardTitle>
                  </div>
                  <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                    <ArrowUpRight className="w-4 h-4" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                {workflowsLoading ? (
                  <div className="flex items-center justify-center p-12">
                    <RefreshCw className="w-6 h-6 animate-spin mr-2 text-muted-foreground" />
                    <span className="text-muted-foreground">Loading workflows...</span>
                  </div>
                ) : displayWorkflows.length === 0 ? (
                  <div className="flex items-center justify-center p-12 text-center">
                    <div className="space-y-2">
                      <Activity className="w-8 h-8 text-muted-foreground mx-auto" />
                      <p className="text-muted-foreground">No workflows found</p>
                    </div>
                  </div>
                ) : (
                  <>
                    <div className="space-y-4">
                      {displayWorkflows.slice(0, 4).map((workflow) => (
                        <div
                          key={workflow.id}
                          className="flex items-center justify-between p-4 rounded-lg border bg-white dark:bg-gray-800 hover:shadow-md transition-all"
                        >
                          <div className="flex items-center gap-3">
                            <div className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-600 flex items-center justify-center">
                              <Zap className="w-4 h-4 text-gray-900 dark:text-gray-100" />
                            </div>
                            <div className="min-w-0">
                              <p className="font-semibold text-sm truncate">{workflow.name}</p>
                              <p className="text-xs text-muted-foreground">
                                {workflow.timestamp} • {workflow.duration}
                              </p>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            {getStatusBadge(workflow.status)}
                          </div>
                        </div>
                      ))}
                    </div>
                    <div className="mt-6 pt-4 border-t border-gray-200 dark:border-gray-700">
                      <Link href="/workflows">
                        <Button
                          variant="outline"
                          className="w-full bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                        >
                          <Activity className="w-4 h-4 mr-2" />
                          View All Workflows
                        </Button>
                      </Link>
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Resources Section */}
          <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
            <CardHeader className="pb-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-gray-100 dark:bg-gray-700">
                    <Database className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                  </div>
                  <CardTitle className="text-xl">Recent Resources</CardTitle>
                </div>
                <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                  <ArrowUpRight className="w-4 h-4" />
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {resourcesLoading ? (
                <div className="flex items-center justify-center p-12">
                  <RefreshCw className="w-6 h-6 animate-spin mr-2 text-muted-foreground" />
                  <span className="text-muted-foreground">Loading resources...</span>
                </div>
              ) : recentResources.length === 0 ? (
                <div className="flex items-center justify-center p-12 text-center">
                  <div className="space-y-2">
                    <Database className="w-8 h-8 text-muted-foreground mx-auto" />
                    <p className="text-muted-foreground">No resources found</p>
                  </div>
                </div>
              ) : (
                <>
                  <div className="space-y-3">
                    {recentResources.map((resource) => (
                      <Link
                        key={resource.id}
                        href={`/resources?resourceId=${resource.id}`}
                        className="flex items-start justify-between p-4 rounded-lg border bg-white dark:bg-gray-800 hover:shadow-md hover:border-blue-300 dark:hover:border-blue-700 transition-all cursor-pointer"
                      >
                        <div className="flex items-start gap-3 flex-1 min-w-0">
                          <div className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-600 flex items-center justify-center flex-shrink-0">
                            <Database className="w-4 h-4 text-gray-900 dark:text-gray-100" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 mb-1">
                              <p className="font-semibold text-sm truncate">
                                {resource.resource_name}
                              </p>
                              <Badge variant="outline" className="text-xs shrink-0">
                                {resource.resource_type}
                              </Badge>
                            </div>
                            <Link
                              href={`/graph/${resource.application_name}`}
                              className="text-xs text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 flex items-center gap-1 w-fit"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <span className="truncate">Part of: {resource.application_name}</span>
                              <ArrowUpRight className="w-3 h-3 shrink-0" />
                            </Link>
                            <p className="text-xs text-muted-foreground mt-1">
                              Created: {new Date(resource.created_at).toLocaleString()}
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2 ml-2 shrink-0">
                          {getResourceStateBadge(resource.state)}
                          {resource.health_status && (
                            <Badge
                              variant="outline"
                              className={`text-xs ${
                                resource.health_status === 'healthy'
                                  ? 'border-green-500 text-green-700 dark:text-green-400'
                                  : 'border-yellow-500 text-yellow-700 dark:text-yellow-400'
                              }`}
                            >
                              {resource.health_status}
                            </Badge>
                          )}
                        </div>
                      </Link>
                    ))}
                  </div>
                  <div className="mt-6 pt-4 border-t border-gray-200 dark:border-gray-700">
                    <Link href="/resources">
                      <Button
                        variant="outline"
                        className="w-full bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
                      >
                        <Database className="w-4 h-4 mr-2" />
                        View All Resources
                      </Button>
                    </Link>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </ProtectedRoute>
  );
}
