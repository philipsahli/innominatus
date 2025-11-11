'use client';

import { useState, useEffect } from 'react';
import { Network, Loader2, Database, GitBranch, Activity, Package } from 'lucide-react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { api } from '@/lib/api';
import { ResourceTable, type Resource } from '@/components/dev/resource-table';
import { WorkflowTable, type Workflow, type WorkflowStep } from '@/components/dev/workflow-table';
import { LiveStepsMonitor } from '@/components/dev/live-steps-monitor';

interface Application {
  name: string;
  score_spec?: any;
  created_at?: string;
}

export default function Graph2Page() {
  const [resources, setResources] = useState<Resource[]>([]);
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [applications, setApplications] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [appFilter, setAppFilter] = useState<string>('all');

  // Load initial data
  useEffect(() => {
    loadAllData();
  }, []);

  // Auto-refresh for running workflows
  useEffect(() => {
    const interval = setInterval(() => {
      refreshRunningWorkflows();
    }, 3000);
    return () => clearInterval(interval);
  }, [workflows]);

  async function loadAllData() {
    setLoading(true);
    try {
      const [appsRes, resourcesRes, workflowsRes] = await Promise.all([
        api.getApplications(),
        api.getResources(),
        api.getWorkflows(undefined, undefined, undefined, 1, 100),
      ]);

      setApplications(appsRes.data || []);

      // Flatten resources from grouped structure
      const resourcesData = resourcesRes.data || {};
      const flatResources = Object.values(resourcesData).flat();
      setResources(flatResources);

      // Load steps for running workflows
      const workflowsData = workflowsRes.data?.data || [];
      const workflowsWithSteps = await Promise.all(
        workflowsData.map(async (workflow) => {
          if (workflow.status === 'running') {
            try {
              const detailRes = await api.getWorkflow(workflow.id);
              if (detailRes.success && detailRes.data) {
                return {
                  ...workflow,
                  steps: detailRes.data.steps,
                  total_steps: detailRes.data.total_steps,
                  error_message: detailRes.data.error_message,
                };
              }
              return workflow;
            } catch (error) {
              return workflow;
            }
          }
          return workflow;
        })
      );

      setWorkflows(workflowsWithSteps);
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  }

  async function refreshRunningWorkflows() {
    try {
      const runningWorkflows = workflows.filter((w) => w.status === 'running');
      if (runningWorkflows.length === 0) return;

      // Refresh steps for running workflows
      const updatedWorkflows = await Promise.all(
        workflows.map(async (workflow) => {
          if (workflow.status === 'running') {
            try {
              const detailRes = await api.getWorkflow(workflow.id);
              if (detailRes.success && detailRes.data) {
                return {
                  ...workflow,
                  steps: detailRes.data.steps,
                  total_steps: detailRes.data.total_steps,
                  error_message: detailRes.data.error_message,
                };
              }
              return workflow;
            } catch (error) {
              return workflow;
            }
          }
          return workflow;
        })
      );

      setWorkflows(updatedWorkflows);
    } catch (error) {
      console.error('Failed to refresh workflows:', error);
    }
  }

  const stats = {
    applications: applications.length,
    resources: resources.length,
    workflows: workflows.length,
    runningSteps: workflows
      .filter((w) => w.status === 'running')
      .reduce((sum, w) => sum + (w.steps?.filter((s) => s.status === 'running').length || 0), 0),
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin text-zinc-600 mx-auto mb-2" />
          <div className="text-sm text-zinc-500">Loading graph data...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white mb-2">Graph Explorer</h1>
        <p className="text-sm text-zinc-500">
          Explore resources, workflows, and real-time execution across all applications
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-4">
        <Card className="bg-zinc-900/50 border-zinc-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400 flex items-center gap-2">
              <Package size={14} />
              Applications
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-white">{stats.applications}</div>
          </CardContent>
        </Card>

        <Card className="bg-zinc-900/50 border-zinc-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400 flex items-center gap-2">
              <Database size={14} />
              Resources
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-white">{stats.resources}</div>
          </CardContent>
        </Card>

        <Card className="bg-zinc-900/50 border-zinc-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400 flex items-center gap-2">
              <GitBranch size={14} />
              Workflows
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-white">{stats.workflows}</div>
          </CardContent>
        </Card>

        <Card className="bg-zinc-900/50 border-zinc-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400 flex items-center gap-2">
              <Activity size={14} />
              Live Steps
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-white flex items-center gap-2">
              {stats.runningSteps}
              {stats.runningSteps > 0 && (
                <span className="w-2 h-2 rounded-full bg-blue-500 animate-pulse" />
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <Card className="bg-zinc-900/50 border-zinc-800">
        <CardContent className="pt-6">
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="text-xs text-zinc-500 mb-2 block">Search</label>
              <Input
                placeholder="Search resources, workflows..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="bg-zinc-900 border-zinc-800"
              />
            </div>

            <div>
              <label className="text-xs text-zinc-500 mb-2 block">Application</label>
              <Select value={appFilter} onValueChange={setAppFilter}>
                <SelectTrigger className="bg-zinc-900 border-zinc-800">
                  <SelectValue placeholder="All applications" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All applications</SelectItem>
                  {applications
                    .filter((app) => app.name && app.name.trim() !== '')
                    .map((app) => (
                      <SelectItem key={app.name} value={app.name}>
                        {app.name}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </div>

            <div>
              <label className="text-xs text-zinc-500 mb-2 block">Status</label>
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="bg-zinc-900 border-zinc-800">
                  <SelectValue placeholder="All statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="requested">Requested</SelectItem>
                  <SelectItem value="provisioning">Provisioning</SelectItem>
                  <SelectItem value="failed">Failed</SelectItem>
                  <SelectItem value="running">Running</SelectItem>
                  <SelectItem value="completed">Completed</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {(searchTerm || statusFilter !== 'all' || appFilter !== 'all') && (
            <div className="mt-4 flex items-center justify-between">
              <div className="text-xs text-zinc-500">
                Filters active: {searchTerm && 'search'} {statusFilter !== 'all' && 'status'}{' '}
                {appFilter !== 'all' && 'application'}
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setSearchTerm('');
                  setStatusFilter('all');
                  setAppFilter('all');
                }}
                className="text-xs text-zinc-500 hover:text-white"
              >
                Clear filters
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Tabs */}
      <Tabs defaultValue="resources" className="space-y-4">
        <TabsList className="bg-zinc-900 border border-zinc-800">
          <TabsTrigger value="resources" className="data-[state=active]:bg-zinc-800">
            <Database size={14} className="mr-2" />
            Resources ({resources.length})
          </TabsTrigger>
          <TabsTrigger value="workflows" className="data-[state=active]:bg-zinc-800">
            <GitBranch size={14} className="mr-2" />
            Workflows ({workflows.length})
          </TabsTrigger>
          <TabsTrigger value="live" className="data-[state=active]:bg-zinc-800">
            <Activity size={14} className="mr-2" />
            Live Steps ({stats.runningSteps})
          </TabsTrigger>
          <TabsTrigger value="graph" className="data-[state=active]:bg-zinc-800">
            <Network size={14} className="mr-2" />
            Graph View
          </TabsTrigger>
        </TabsList>

        <TabsContent value="resources" className="space-y-4">
          <ResourceTable
            resources={resources}
            searchTerm={searchTerm}
            statusFilter={statusFilter}
            appFilter={appFilter}
          />
        </TabsContent>

        <TabsContent value="workflows" className="space-y-4">
          <WorkflowTable
            workflows={workflows}
            searchTerm={searchTerm}
            statusFilter={statusFilter}
            appFilter={appFilter}
          />
        </TabsContent>

        <TabsContent value="live" className="space-y-4">
          <LiveStepsMonitor workflows={workflows} />
        </TabsContent>

        <TabsContent value="graph" className="space-y-4">
          <Card className="p-8 bg-zinc-900/50 border-zinc-800">
            <div className="text-center">
              <Network size={32} className="mx-auto mb-3 text-zinc-600" />
              <div className="text-zinc-500 mb-2">Graph visualization coming soon</div>
              <div className="text-xs text-zinc-600">
                Will integrate GraphVisualization component with application selector
              </div>
            </div>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
