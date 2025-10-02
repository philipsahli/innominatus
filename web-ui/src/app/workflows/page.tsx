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
  Activity,
  RefreshCw,
  Search,
  Clock,
  CheckCircle,
  XCircle,
  Zap,
  ArrowUpRight,
  Calendar,
  FileText,
  Play,
  GitBranch,
} from 'lucide-react';
import { ProtectedRoute } from '@/components/protected-route';
import { useWorkflows } from '@/hooks/use-api';
import { useState } from 'react';
import { useRouter } from 'next/navigation';

function getStatusBadge(status: string) {
  return (
    <Badge variant="outline">
      <Clock className="w-3 h-3 mr-1" />
      {status}
    </Badge>
  );
}

function getStatusIcon() {
  return <Activity className="w-4 h-4 text-gray-500" />;
}

function formatDuration(duration: string) {
  // If duration is already formatted (like "2m 15s"), return as is
  if (duration.includes('m') || duration.includes('s')) {
    return duration;
  }

  // Otherwise try to parse and format
  const seconds = parseInt(duration);
  if (isNaN(seconds)) return duration;

  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
}

export default function WorkflowsPage() {
  const router = useRouter();
  const {
    data: workflows,
    loading: workflowsLoading,
    error: workflowsError,
    refetch: refetchWorkflows,
  } = useWorkflows();
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [activeTab, setActiveTab] = useState<'executions' | 'templates'>('executions');

  const displayWorkflows = workflows || [];

  // Filter workflows based on search term and status
  const filteredWorkflows = displayWorkflows.filter((workflow) => {
    const matchesSearch =
      workflow.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (workflow.app_name && workflow.app_name.toLowerCase().includes(searchTerm.toLowerCase()));
    const matchesStatus = statusFilter === 'all' || workflow.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

  const handleRefresh = () => {
    refetchWorkflows();
  };

  // Get workflow statistics
  const stats = {
    total: displayWorkflows.length,
    running: displayWorkflows.filter((w) => w.status === 'running').length,
    completed: displayWorkflows.filter((w) => w.status === 'completed').length,
    failed: displayWorkflows.filter((w) => w.status === 'failed').length,
    pending: displayWorkflows.filter((w) => w.status === 'pending').length,
  };

  return (
    <ProtectedRoute>
      <div className="min-h-screen bg-white dark:bg-gray-900 p-6">
        <div className="relative space-y-6">
          {/* Header */}
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg bg-gray-200 dark:bg-gray-700">
                  <Activity className="w-6 h-6 text-gray-900 dark:text-gray-100" />
                </div>
                <div>
                  <h1 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100">
                    Workflow Management
                  </h1>
                  <p className="text-gray-600 dark:text-gray-400">
                    Manage workflow templates and monitor executions
                  </p>
                </div>
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={() => router.push('/workflows/analyze')}
                  className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100"
                >
                  <GitBranch className="w-4 h-4 mr-2" />
                  Analyze Workflow
                </Button>
                <Button
                  variant="outline"
                  onClick={handleRefresh}
                  disabled={workflowsLoading}
                  className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-900 dark:text-gray-100"
                >
                  <RefreshCw className={`w-4 h-4 mr-2 ${workflowsLoading ? 'animate-spin' : ''}`} />
                  Refresh
                </Button>
              </div>
            </div>
          </div>

          {/* Error Display */}
          {workflowsError && (
            <Card className="bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700">
              <CardContent className="pt-6">
                <p className="text-gray-800 dark:text-gray-200 text-sm">
                  Using offline mode - workflow data may not be current. Error: {workflowsError}
                </p>
              </CardContent>
            </Card>
          )}

          {/* Tabs */}
          <div className="border-b border-gray-200 dark:border-gray-700">
            <nav className="-mb-px flex space-x-8">
              <button
                onClick={() => setActiveTab('executions')}
                className={`py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'executions'
                    ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                }`}
              >
                <Activity className="w-4 h-4 mr-2 inline" />
                Workflow Executions
              </button>
              <button
                onClick={() => setActiveTab('templates')}
                className={`py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'templates'
                    ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                }`}
              >
                <FileText className="w-4 h-4 mr-2 inline" />
                Workflow Templates
              </button>
            </nav>
          </div>

          {activeTab === 'executions' && (
            <>
              {/* Statistics Cards */}
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Total</CardTitle>
                    <Activity className="h-4 w-4 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{stats.total}</div>
                  </CardContent>
                </Card>

                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Running</CardTitle>
                    <Zap className="h-4 w-4 text-gray-500" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                      {stats.running}
                    </div>
                  </CardContent>
                </Card>

                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Completed</CardTitle>
                    <CheckCircle className="h-4 w-4 text-gray-500" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                      {stats.completed}
                    </div>
                  </CardContent>
                </Card>

                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Failed</CardTitle>
                    <XCircle className="h-4 w-4 text-gray-500" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                      {stats.failed}
                    </div>
                  </CardContent>
                </Card>

                <Card className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg">
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">Pending</CardTitle>
                    <Clock className="h-4 w-4 text-gray-500" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                      {stats.pending}
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
                          placeholder="Search workflows or applications..."
                          value={searchTerm}
                          onChange={(e) => setSearchTerm(e.target.value)}
                          className="pl-10"
                        />
                      </div>
                    </div>
                    <div className="sm:w-48">
                      <select
                        value={statusFilter}
                        onChange={(e) => setStatusFilter(e.target.value)}
                        className="w-full h-10 px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      >
                        <option value="all">All Status</option>
                        <option value="running">Running</option>
                        <option value="completed">Completed</option>
                        <option value="failed">Failed</option>
                        <option value="pending">Pending</option>
                      </select>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Workflows Table */}
              <Card className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm border-white/20 shadow-lg">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="text-xl">Workflow Executions</CardTitle>
                      <CardDescription>
                        {filteredWorkflows.length} of {displayWorkflows.length} workflows
                        {searchTerm && ` matching "${searchTerm}"`}
                        {statusFilter !== 'all' && ` with status "${statusFilter}"`}
                      </CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  {workflowsLoading ? (
                    <div className="flex items-center justify-center p-12">
                      <RefreshCw className="w-6 h-6 animate-spin mr-2 text-muted-foreground" />
                      <span className="text-muted-foreground">Loading workflows...</span>
                    </div>
                  ) : filteredWorkflows.length === 0 ? (
                    <div className="flex items-center justify-center p-12 text-center">
                      <div className="space-y-2">
                        <Activity className="w-8 h-8 text-muted-foreground mx-auto" />
                        <p className="text-muted-foreground">
                          {searchTerm || statusFilter !== 'all'
                            ? 'No workflows match your filters'
                            : 'No workflows found'}
                        </p>
                      </div>
                    </div>
                  ) : (
                    <div className="rounded-md border">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead className="w-12"></TableHead>
                            <TableHead>Workflow</TableHead>
                            <TableHead>Application</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Duration</TableHead>
                            <TableHead>Started</TableHead>
                            <TableHead className="w-12"></TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {filteredWorkflows.map((workflow) => (
                            <TableRow
                              key={workflow.id}
                              className="hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer"
                              onClick={() => router.push(`/workflows/${workflow.id}`)}
                            >
                              <TableCell>
                                <div className="flex items-center justify-center">
                                  {getStatusIcon()}
                                </div>
                              </TableCell>
                              <TableCell>
                                <div className="space-y-1">
                                  <p className="font-medium text-sm">{workflow.name}</p>
                                  <p className="text-xs text-muted-foreground">ID: {workflow.id}</p>
                                </div>
                              </TableCell>
                              <TableCell>
                                {workflow.app_name ? (
                                  <Badge variant="outline" className="text-xs">
                                    {workflow.app_name}
                                  </Badge>
                                ) : (
                                  <span className="text-xs text-muted-foreground">-</span>
                                )}
                              </TableCell>
                              <TableCell>{getStatusBadge(workflow.status)}</TableCell>
                              <TableCell>
                                <span className="text-sm font-mono">
                                  {workflow.status === 'pending'
                                    ? '-'
                                    : formatDuration(workflow.duration)}
                                </span>
                              </TableCell>
                              <TableCell>
                                <div className="flex items-center gap-1 text-sm text-muted-foreground">
                                  <Calendar className="w-3 h-3" />
                                  {workflow.timestamp}
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
            </>
          )}

          {activeTab === 'templates' && (
            <Card className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-sm border-white/20 shadow-lg">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="text-xl">Workflow Templates</CardTitle>
                    <CardDescription>
                      Available workflow definitions and golden paths
                    </CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                  {/* Golden Path Templates */}
                  <Card className="border-2 border-dashed border-gray-300 dark:border-gray-600 hover:border-blue-400 dark:hover:border-blue-500 transition-colors">
                    <CardHeader className="pb-3">
                      <div className="flex items-center gap-2">
                        <FileText className="w-4 h-4 text-blue-500" />
                        <CardTitle className="text-lg">deploy-app</CardTitle>
                      </div>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <p className="text-sm text-muted-foreground">
                        Deploy application with full infrastructure provisioning
                      </p>
                      <div className="flex flex-wrap gap-1">
                        <Badge variant="secondary" className="text-xs">
                          terraform
                        </Badge>
                        <Badge variant="secondary" className="text-xs">
                          kubernetes
                        </Badge>
                        <Badge variant="secondary" className="text-xs">
                          ansible
                        </Badge>
                      </div>
                      <Button size="sm" className="w-full">
                        <Play className="w-3 h-3 mr-2" />
                        Run Template
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="border-2 border-dashed border-gray-300 dark:border-gray-600 hover:border-blue-400 dark:hover:border-blue-500 transition-colors">
                    <CardHeader className="pb-3">
                      <div className="flex items-center gap-2">
                        <FileText className="w-4 h-4 text-green-500" />
                        <CardTitle className="text-lg">ephemeral-env</CardTitle>
                      </div>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <p className="text-sm text-muted-foreground">
                        Create temporary environments for testing
                      </p>
                      <div className="flex flex-wrap gap-1">
                        <Badge variant="secondary" className="text-xs">
                          kubernetes
                        </Badge>
                        <Badge variant="secondary" className="text-xs">
                          temporary
                        </Badge>
                      </div>
                      <Button size="sm" className="w-full">
                        <Play className="w-3 h-3 mr-2" />
                        Run Template
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="border-2 border-dashed border-gray-300 dark:border-gray-600 hover:border-blue-400 dark:hover:border-blue-500 transition-colors">
                    <CardHeader className="pb-3">
                      <div className="flex items-center gap-2">
                        <FileText className="w-4 h-4 text-purple-500" />
                        <CardTitle className="text-lg">db-lifecycle</CardTitle>
                      </div>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <p className="text-sm text-muted-foreground">
                        Manage database operations (backup, migration, health check)
                      </p>
                      <div className="flex flex-wrap gap-1">
                        <Badge variant="secondary" className="text-xs">
                          database
                        </Badge>
                        <Badge variant="secondary" className="text-xs">
                          backup
                        </Badge>
                        <Badge variant="secondary" className="text-xs">
                          migration
                        </Badge>
                      </div>
                      <Button size="sm" className="w-full">
                        <Play className="w-3 h-3 mr-2" />
                        Run Template
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="border-2 border-dashed border-gray-300 dark:border-gray-600 hover:border-blue-400 dark:hover:border-blue-500 transition-colors">
                    <CardHeader className="pb-3">
                      <div className="flex items-center gap-2">
                        <FileText className="w-4 h-4 text-orange-500" />
                        <CardTitle className="text-lg">observability-setup</CardTitle>
                      </div>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <p className="text-sm text-muted-foreground">
                        Setup monitoring and observability stack
                      </p>
                      <div className="flex flex-wrap gap-1">
                        <Badge variant="secondary" className="text-xs">
                          monitoring
                        </Badge>
                        <Badge variant="secondary" className="text-xs">
                          grafana
                        </Badge>
                        <Badge variant="secondary" className="text-xs">
                          prometheus
                        </Badge>
                      </div>
                      <Button size="sm" className="w-full">
                        <Play className="w-3 h-3 mr-2" />
                        Run Template
                      </Button>
                    </CardContent>
                  </Card>
                </div>

                <div className="mt-6 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
                  <h4 className="text-sm font-medium mb-2">What are Workflow Templates?</h4>
                  <p className="text-sm text-muted-foreground">
                    Workflow templates are reusable workflow definitions that can be executed
                    multiple times with different parameters. They define the steps, types, and
                    configuration needed to complete complex automation tasks. Golden paths are
                    curated templates that represent the recommended way to accomplish common
                    platform operations.
                  </p>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </ProtectedRoute>
  );
}
