'use client';

import React, { useState, useEffect } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { ProtectedRoute } from '@/components/protected-route';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
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
  Search,
  Filter,
  PlayCircle,
  XCircle,
  Clock,
  Database,
  RefreshCw,
  ExternalLink,
} from 'lucide-react';
import { api, type Application } from '@/lib/api';
import { ApplicationDetailsPane } from '@/components/application-details-pane';

export default function ApplicationsPage() {
  const searchParams = useSearchParams();
  const [applications, setApplications] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [environmentFilter, setEnvironmentFilter] = useState<string>('all');
  const [selectedApplication, setSelectedApplication] = useState<Application | null>(null);

  // Auto-select application from query parameter
  useEffect(() => {
    const appName = searchParams.get('appName');
    if (appName && applications.length > 0 && !selectedApplication) {
      const app = applications.find((a) => a.name === appName);
      if (app) {
        setSelectedApplication(app);
      }
    }
  }, [searchParams, applications, selectedApplication]);

  const fetchApplications = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await api.getApplications();

      if (response.success && response.data) {
        setApplications(response.data);
      } else {
        setError(response.error || 'Failed to fetch applications');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch applications');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchApplications();
  }, []);

  // Filter applications
  const filteredApplications = applications.filter((app) => {
    const matchesSearch = app.name.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = statusFilter === 'all' || app.status === statusFilter;
    const matchesEnvironment = environmentFilter === 'all' || app.environment === environmentFilter;

    return matchesSearch && matchesStatus && matchesEnvironment;
  });

  // Calculate statistics
  const stats = {
    total: applications.length,
    running: applications.filter((a) => a.status === 'running').length,
    failed: applications.filter((a) => a.status === 'failed').length,
    pending: applications.filter((a) => a.status === 'pending').length,
  };

  // Get unique environments
  const environments = Array.from(new Set(applications.map((a) => a.environment)));

  const getStatusBadgeVariant = (
    status: string
  ): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'running':
        return 'default';
      case 'pending':
        return 'secondary';
      case 'failed':
        return 'destructive';
      default:
        return 'outline';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <PlayCircle className="w-4 h-4 text-green-600 dark:text-green-400" />;
      case 'failed':
        return <XCircle className="w-4 h-4 text-red-600 dark:text-red-400" />;
      case 'pending':
        return <Clock className="w-4 h-4 text-yellow-600 dark:text-yellow-400" />;
      default:
        return null;
    }
  };

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900 flex">
        <div className={`flex-1 p-6 overflow-auto ${selectedApplication ? 'pr-0' : ''}`}>
          {/* Header */}
          <div className="mb-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 flex items-center gap-2">
                  <Package className="w-8 h-8" />
                  Applications
                </h1>
                <p className="text-gray-600 dark:text-gray-400 mt-1">
                  Manage and monitor your deployed applications
                </p>
              </div>
              <Button onClick={fetchApplications} variant="outline" size="sm">
                <RefreshCw className="w-4 h-4 mr-2" />
                Refresh
              </Button>
            </div>

            {/* Statistics Cards */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                    Total Applications
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                    {stats.total}
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                    Running
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-green-600 dark:text-green-400 flex items-center gap-2">
                    <PlayCircle className="w-5 h-5" />
                    {stats.running}
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                    Failed
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-red-600 dark:text-red-400 flex items-center gap-2">
                    <XCircle className="w-5 h-5" />
                    {stats.failed}
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm font-medium text-gray-600 dark:text-gray-400">
                    Pending
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold text-yellow-600 dark:text-yellow-400 flex items-center gap-2">
                    <Clock className="w-5 h-5" />
                    {stats.pending}
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Filters */}
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Filter className="w-5 h-5" />
                    <CardTitle>Filters</CardTitle>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setSearchTerm('');
                      setStatusFilter('all');
                      setEnvironmentFilter('all');
                    }}
                  >
                    Clear All
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex flex-col md:flex-row gap-4">
                  {/* Search */}
                  <div className="flex-1">
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
                      <Input
                        placeholder="Search applications..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        className="pl-10"
                      />
                    </div>
                  </div>

                  {/* Status Filter */}
                  <Select value={statusFilter} onValueChange={setStatusFilter}>
                    <SelectTrigger className="w-full md:w-48">
                      <SelectValue placeholder="Filter by status" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Statuses</SelectItem>
                      <SelectItem value="running">Running</SelectItem>
                      <SelectItem value="pending">Pending</SelectItem>
                      <SelectItem value="failed">Failed</SelectItem>
                    </SelectContent>
                  </Select>

                  {/* Environment Filter */}
                  <Select value={environmentFilter} onValueChange={setEnvironmentFilter}>
                    <SelectTrigger className="w-full md:w-48">
                      <SelectValue placeholder="Filter by environment" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Environments</SelectItem>
                      {environments.map((env) => (
                        <SelectItem key={env} value={env}>
                          {env}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Applications Table */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Package className="w-5 h-5" />
                Applications
                <span className="text-sm font-normal text-gray-500">
                  ({filteredApplications.length} of {applications.length})
                </span>
              </CardTitle>
              <CardDescription>View and manage all deployed applications</CardDescription>
            </CardHeader>
            <CardContent>
              {loading ? (
                <div className="flex items-center justify-center py-8">
                  <RefreshCw className="w-6 h-6 animate-spin text-gray-400" />
                  <span className="ml-2 text-gray-600 dark:text-gray-400">
                    Loading applications...
                  </span>
                </div>
              ) : error ? (
                <div className="text-center py-8">
                  <p className="text-red-600 dark:text-red-400">{error}</p>
                  <Button onClick={fetchApplications} variant="outline" className="mt-4">
                    Try Again
                  </Button>
                </div>
              ) : filteredApplications.length === 0 ? (
                <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                  {applications.length === 0 ? (
                    <>
                      <Package className="w-12 h-12 mx-auto mb-4 text-gray-300 dark:text-gray-600" />
                      <p className="text-lg font-medium">No applications found</p>
                      <p className="text-sm mt-1">Deploy your first application to get started</p>
                    </>
                  ) : (
                    <>
                      <Search className="w-12 h-12 mx-auto mb-4 text-gray-300 dark:text-gray-600" />
                      <p className="text-lg font-medium">No applications match your filters</p>
                      <p className="text-sm mt-1">Try adjusting your search or filter criteria</p>
                    </>
                  )}
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-12"></TableHead>
                        <TableHead>Name</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Environment</TableHead>
                        <TableHead>Resources</TableHead>
                        <TableHead>Last Updated</TableHead>
                        <TableHead className="text-right">Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredApplications.map((app) => (
                        <TableRow
                          key={app.name}
                          className={`cursor-pointer transition-colors ${
                            selectedApplication?.name === app.name
                              ? 'bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500'
                              : 'hover:bg-gray-50 dark:hover:bg-gray-800'
                          }`}
                          onClick={() => setSelectedApplication(app)}
                        >
                          <TableCell>
                            <div className="flex items-center justify-center">
                              {getStatusIcon(app.status)}
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="font-medium text-gray-900 dark:text-gray-100">
                              {app.name}
                            </div>
                          </TableCell>
                          <TableCell>
                            <Badge variant={getStatusBadgeVariant(app.status)}>{app.status}</Badge>
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline">{app.environment}</Badge>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-1">
                              <Database className="w-4 h-4 text-gray-500" />
                              <span>{app.resources}</span>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="text-sm text-gray-600 dark:text-gray-400">
                              {new Date(app.lastUpdated).toLocaleString()}
                            </div>
                          </TableCell>
                          <TableCell className="text-right">
                            <Link href={`/graph/${app.name}`} onClick={(e) => e.stopPropagation()}>
                              <Button variant="ghost" size="sm">
                                <ExternalLink className="w-4 h-4" />
                              </Button>
                            </Link>
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

        {/* Details Pane */}
        {selectedApplication && (
          <div className="w-[500px] h-screen sticky top-0 flex-shrink-0 overflow-auto">
            <ApplicationDetailsPane
              application={selectedApplication}
              onClose={() => setSelectedApplication(null)}
            />
          </div>
        )}
      </div>
    </ProtectedRoute>
  );
}
