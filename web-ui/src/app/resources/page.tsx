'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Package,
  RefreshCw,
  Search,
  Database,
  Server,
  HardDrive,
  Activity,
  CheckCircle,
  XCircle,
  Clock,
  AlertTriangle,
  Trash2,
  ArrowUpRight,
  Calendar,
  Zap,
} from 'lucide-react';
import { ProtectedRoute } from '@/components/protected-route';
import { useResources } from '@/hooks/use-api';
import { useState, useEffect } from 'react';
import { useSearchParams } from 'next/navigation';
import type { ResourceInstance } from '@/lib/api';
import { ResourceDetailsPane } from '@/components/resource-details-pane';

function getStatusBadge(state: string, healthStatus: string) {
  const isHealthy = healthStatus === 'healthy';
  const isDegraded = healthStatus === 'degraded';

  switch (state) {
    case 'active':
      if (isHealthy) {
        return (
          <Badge
            variant="default"
            className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
          >
            <CheckCircle className="w-3 h-3 mr-1" />
            Active
          </Badge>
        );
      } else if (isDegraded) {
        return (
          <Badge
            variant="secondary"
            className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
          >
            <AlertTriangle className="w-3 h-3 mr-1" />
            Degraded
          </Badge>
        );
      } else {
        return (
          <Badge variant="destructive">
            <XCircle className="w-3 h-3 mr-1" />
            Unhealthy
          </Badge>
        );
      }
    case 'provisioning':
    case 'scaling':
    case 'updating':
      return (
        <Badge
          variant="secondary"
          className="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
        >
          <Zap className="w-3 h-3 mr-1" />
          {state.charAt(0).toUpperCase() + state.slice(1)}
        </Badge>
      );
    case 'requested':
    case 'pending':
      return (
        <Badge variant="outline">
          <Clock className="w-3 h-3 mr-1" />
          Pending
        </Badge>
      );
    case 'terminating':
      return (
        <Badge
          variant="secondary"
          className="bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200"
        >
          <Trash2 className="w-3 h-3 mr-1" />
          Terminating
        </Badge>
      );
    case 'terminated':
      return (
        <Badge variant="outline" className="text-gray-500">
          <XCircle className="w-3 h-3 mr-1" />
          Terminated
        </Badge>
      );
    case 'failed':
      return (
        <Badge variant="destructive">
          <XCircle className="w-3 h-3 mr-1" />
          Failed
        </Badge>
      );
    default:
      return (
        <Badge variant="outline">
          <Clock className="w-3 h-3 mr-1" />
          {state}
        </Badge>
      );
  }
}

function getResourceTypeIcon(type: string) {
  switch (type.toLowerCase()) {
    case 'postgres':
    case 'postgresql':
    case 'database':
      return <Database className="w-4 h-4" />;
    case 'redis':
    case 'cache':
      return <Server className="w-4 h-4" />;
    case 'volume':
    case 'storage':
      return <HardDrive className="w-4 h-4" />;
    default:
      return <Package className="w-4 h-4" />;
  }
}

function formatTimestamp(timestamp: string) {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) {
    return 'just now';
  } else if (diffMins < 60) {
    return `${diffMins} minute${diffMins === 1 ? '' : 's'} ago`;
  } else if (diffHours < 24) {
    return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
  } else {
    return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`;
  }
}

export default function ResourcesPage() {
  const searchParams = useSearchParams();
  const {
    data: resourcesData,
    loading: resourcesLoading,
    error: resourcesError,
    refetch: refetchResources,
  } = useResources();
  const [searchTerm, setSearchTerm] = useState('');
  const [stateFilter, setStateFilter] = useState('all');
  const [typeFilter, setTypeFilter] = useState('all');
  const [selectedResource, setSelectedResource] = useState<ResourceInstance | null>(null);

  // Flatten resources data into a single array
  const allResources: ResourceInstance[] = resourcesData ? Object.values(resourcesData).flat() : [];

  // Auto-select resource from query parameter
  useEffect(() => {
    const resourceId = searchParams.get('resourceId');
    if (resourceId && allResources.length > 0 && !selectedResource) {
      const resource = allResources.find((r) => r.id === parseInt(resourceId, 10));
      if (resource) {
        setSelectedResource(resource);
      }
    }
  }, [searchParams, allResources, selectedResource]);

  // Filter resources based on search term, state, and type
  const filteredResources = allResources.filter((resource) => {
    const matchesSearch =
      resource.resource_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      resource.application_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      resource.resource_type.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesState = stateFilter === 'all' || resource.state === stateFilter;
    const matchesType = typeFilter === 'all' || resource.resource_type === typeFilter;
    return matchesSearch && matchesState && matchesType;
  });

  const handleRefresh = () => {
    refetchResources();
  };

  // Get resource statistics
  const stats = {
    total: allResources.length,
    active: allResources.filter((r) => r.state === 'active').length,
    provisioning: allResources.filter((r) => r.state === 'provisioning').length,
    failed: allResources.filter((r) => r.state === 'failed').length,
    healthy: allResources.filter((r) => r.health_status === 'healthy').length,
  };

  // Get unique resource types for filter
  const uniqueTypes = [...new Set(allResources.map((r) => r.resource_type))];

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900 flex">
        {/* Main content area */}
        <div className={`flex-1 p-6 overflow-auto ${selectedResource ? 'pr-0' : ''}`}>
          <div className="relative space-y-6">
            {/* Header */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 rounded-lg bg-gray-200 dark:bg-gray-700">
                    <Package className="w-6 h-6 text-gray-900 dark:text-gray-100" />
                  </div>
                  <div>
                    <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
                      Resources
                    </h1>
                    <p className="text-gray-600 dark:text-gray-400">
                      Monitor and manage resource instances across applications
                    </p>
                  </div>
                </div>
                <Button
                  variant="outline"
                  onClick={handleRefresh}
                  disabled={resourcesLoading}
                  className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100"
                >
                  <RefreshCw className={`w-4 h-4 mr-2 ${resourcesLoading ? 'animate-spin' : ''}`} />
                  Refresh
                </Button>
              </div>
            </div>

            {/* Error Display */}
            {resourcesError && (
              <Card className="bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700">
                <CardContent className="pt-6">
                  <p className="text-gray-800 dark:text-gray-200 text-sm">
                    Using offline mode - resource data may not be current. Error: {resourcesError}
                  </p>
                </CardContent>
              </Card>
            )}

            {/* Statistics Cards */}
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
              <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Total</CardTitle>
                  <Package className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{stats.total}</div>
                </CardContent>
              </Card>

              <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Active</CardTitle>
                  <CheckCircle className="h-4 w-4 text-green-500" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                    {stats.active}
                  </div>
                </CardContent>
              </Card>

              <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Provisioning</CardTitle>
                  <Zap className="h-4 w-4 text-blue-500" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                    {stats.provisioning}
                  </div>
                </CardContent>
              </Card>

              <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Failed</CardTitle>
                  <XCircle className="h-4 w-4 text-red-500" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                    {stats.failed}
                  </div>
                </CardContent>
              </Card>

              <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Healthy</CardTitle>
                  <Activity className="h-4 w-4 text-green-500" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                    {stats.healthy}
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Filters and Search */}
            <Card className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm border-white/20 shadow-lg">
              <CardHeader className="pb-4">
                <CardTitle className="text-lg">Filters</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex flex-col sm:flex-row gap-4">
                  <div className="flex-1">
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        placeholder="Search resources, applications, or types..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        className="pl-10"
                      />
                    </div>
                  </div>
                  <div className="sm:w-48">
                    <select
                      value={stateFilter}
                      onChange={(e) => setStateFilter(e.target.value)}
                      className="w-full h-10 px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    >
                      <option value="all">All States</option>
                      <option value="active">Active</option>
                      <option value="provisioning">Provisioning</option>
                      <option value="failed">Failed</option>
                      <option value="requested">Requested</option>
                      <option value="terminating">Terminating</option>
                    </select>
                  </div>
                  <div className="sm:w-48">
                    <select
                      value={typeFilter}
                      onChange={(e) => setTypeFilter(e.target.value)}
                      className="w-full h-10 px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    >
                      <option value="all">All Types</option>
                      {uniqueTypes.map((type) => (
                        <option key={type} value={type}>
                          {type}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Resources Table */}
            <Card className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm border-white/20 shadow-lg">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="text-xl">Resource Instances</CardTitle>
                    <CardDescription>
                      {filteredResources.length} of {allResources.length} resources
                      {searchTerm && ` matching "${searchTerm}"`}
                      {stateFilter !== 'all' && ` with state "${stateFilter}"`}
                      {typeFilter !== 'all' && ` of type "${typeFilter}"`}
                    </CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {resourcesLoading ? (
                  <div className="flex items-center justify-center p-12">
                    <RefreshCw className="w-6 h-6 animate-spin mr-2 text-muted-foreground" />
                    <span className="text-muted-foreground">Loading resources...</span>
                  </div>
                ) : filteredResources.length === 0 ? (
                  <div className="flex items-center justify-center p-12 text-center">
                    <div className="space-y-2">
                      <Package className="w-8 h-8 text-muted-foreground mx-auto" />
                      <p className="text-muted-foreground">
                        {searchTerm || stateFilter !== 'all' || typeFilter !== 'all'
                          ? 'No resources match your filters'
                          : 'No resources found'}
                      </p>
                    </div>
                  </div>
                ) : (
                  <div className="rounded-md border">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-12"></TableHead>
                          <TableHead>Resource</TableHead>
                          <TableHead>Application</TableHead>
                          <TableHead>Type</TableHead>
                          <TableHead>State</TableHead>
                          <TableHead>Health</TableHead>
                          <TableHead>Created</TableHead>
                          <TableHead className="w-12"></TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {filteredResources.map((resource) => (
                          <TableRow
                            key={resource.id}
                            className={`hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer ${
                              selectedResource?.id === resource.id
                                ? 'bg-blue-50 dark:bg-blue-900/20'
                                : ''
                            }`}
                            onClick={() => setSelectedResource(resource)}
                          >
                            <TableCell>
                              <div className="flex items-center justify-center">
                                {getResourceTypeIcon(resource.resource_type)}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="space-y-1">
                                <p className="font-medium text-sm">{resource.resource_name}</p>
                                <p className="text-xs text-muted-foreground">ID: {resource.id}</p>
                              </div>
                            </TableCell>
                            <TableCell>
                              <Badge variant="outline" className="text-xs">
                                {resource.application_name}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <Badge variant="secondary" className="text-xs">
                                {resource.resource_type}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              {getStatusBadge(resource.state, resource.health_status)}
                            </TableCell>
                            <TableCell>
                              <Badge variant="outline" className="text-xs">
                                {resource.health_status}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-1 text-sm text-muted-foreground">
                                <Calendar className="w-3 h-3" />
                                {formatTimestamp(resource.created_at)}
                              </div>
                            </TableCell>
                            <TableCell>
                              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                                <ArrowUpRight className="w-4 h-4" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>

        {/* Details pane on the right */}
        {selectedResource && (
          <div className="w-[500px] h-screen sticky top-0 flex-shrink-0 overflow-auto">
            <ResourceDetailsPane
              resource={selectedResource}
              onClose={() => setSelectedResource(null)}
            />
          </div>
        )}
      </div>
    </ProtectedRoute>
  );
}
