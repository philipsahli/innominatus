'use client';

import { useState, useEffect } from 'react';
import { Loader2, Database, GitBranch, Activity, Package } from 'lucide-react';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { api } from '@/lib/api';
import { ResourceTable, type Resource } from '@/components/dev/resource-table';
import { WorkflowTable, type Workflow } from '@/components/dev/workflow-table';
import { LiveStepsMonitor } from '@/components/dev/live-steps-monitor';
import { GraphView } from '@/components/dev/graph-view';
import { GraphViewReactFlow } from '@/components/dev/graph-view-reactflow';
import { GraphViewCytoscape } from '@/components/dev/graph-view-cytoscape';
import { GraphViewD3 } from '@/components/dev/graph-view-d3';

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
  const [graphLibrary, setGraphLibrary] = useState<string>('svg');

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

      // Deduplicate workflows: Keep only latest execution per workflow name
      const deduplicatedWorkflows = Object.values(
        workflowsWithSteps.reduce(
          (acc, workflow) => {
            const existingWorkflow = acc[workflow.name];

            // If no workflow with this name exists, or this one is newer, keep it
            if (
              !existingWorkflow ||
              new Date(workflow.timestamp) > new Date(existingWorkflow.timestamp)
            ) {
              acc[workflow.name] = workflow;
            }

            return acc;
          },
          {} as Record<string, Workflow>
        )
      );

      setWorkflows(deduplicatedWorkflows);
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
          <Loader2 className="w-8 h-8 animate-spin text-zinc-400 dark:text-zinc-600 mx-auto mb-2" />
          <div className="text-sm text-zinc-500 dark:text-zinc-400">Loading...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">Graph Explorer</h1>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Explore resources, workflows, and dependencies across applications
          </p>
        </div>

        {/* Stats */}
        <div className="flex items-center gap-6 text-sm text-zinc-600 dark:text-zinc-400">
          <div className="flex items-center gap-2">
            <Package size={16} />
            <span>{stats.applications}</span>
          </div>
          <div className="flex items-center gap-2">
            <Database size={16} />
            <span>{stats.resources}</span>
          </div>
          <div className="flex items-center gap-2">
            <GitBranch size={16} />
            <span>{stats.workflows}</span>
          </div>
          {stats.runningSteps > 0 && (
            <div className="flex items-center gap-2">
              <Activity size={16} className="text-blue-500" />
              <span className="text-blue-500">{stats.runningSteps} live</span>
            </div>
          )}
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <Input
          placeholder="Search..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="max-w-xs"
        />

        <Select value={appFilter} onValueChange={setAppFilter}>
          <SelectTrigger className="w-[200px]">
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

        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[180px]">
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

        {(searchTerm || statusFilter !== 'all' || appFilter !== 'all') && (
          <button
            onClick={() => {
              setSearchTerm('');
              setStatusFilter('all');
              setAppFilter('all');
            }}
            className="text-sm text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
          >
            Clear filters
          </button>
        )}
      </div>

      {/* Tabs */}
      <Tabs defaultValue="resources" className="space-y-6">
        <TabsList>
          <TabsTrigger value="resources">
            <Database size={14} className="mr-2" />
            Resources ({resources.length})
          </TabsTrigger>
          <TabsTrigger value="workflows">
            <GitBranch size={14} className="mr-2" />
            Workflows ({workflows.length})
          </TabsTrigger>
          <TabsTrigger value="live">
            <Activity size={14} className="mr-2" />
            Live Steps ({stats.runningSteps})
          </TabsTrigger>
          <TabsTrigger value="graph">Graph View</TabsTrigger>
        </TabsList>

        <TabsContent value="resources">
          <ResourceTable
            resources={resources}
            searchTerm={searchTerm}
            statusFilter={statusFilter}
            appFilter={appFilter}
          />
        </TabsContent>

        <TabsContent value="workflows">
          <WorkflowTable
            workflows={workflows}
            searchTerm={searchTerm}
            statusFilter={statusFilter}
            appFilter={appFilter}
          />
        </TabsContent>

        <TabsContent value="live">
          <LiveStepsMonitor workflows={workflows} />
        </TabsContent>

        <TabsContent value="graph">
          <div className="space-y-4">
            {/* Library Selector */}
            <div className="flex items-center justify-between">
              <div className="text-sm text-zinc-600 dark:text-zinc-400">
                Choose visualization library:
              </div>
              <Select value={graphLibrary} onValueChange={setGraphLibrary}>
                <SelectTrigger className="w-[250px]">
                  <SelectValue placeholder="Select library" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="svg">Simple SVG (Fixed)</SelectItem>
                  <SelectItem value="reactflow">ReactFlow (Recommended)</SelectItem>
                  <SelectItem value="cytoscape">Cytoscape.js (Advanced)</SelectItem>
                  <SelectItem value="d3">D3-Force (Custom)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Graph View */}
            {graphLibrary === 'svg' && <GraphView applications={applications} />}
            {graphLibrary === 'reactflow' && <GraphViewReactFlow applications={applications} />}
            {graphLibrary === 'cytoscape' && <GraphViewCytoscape applications={applications} />}
            {graphLibrary === 'd3' && <GraphViewD3 applications={applications} />}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
