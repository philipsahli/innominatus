'use client';

import { useState } from 'react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { ChevronDown, ChevronRight, Loader2 } from 'lucide-react';
import { api, type WorkflowStepExecution } from '@/lib/api';

export type WorkflowStep = WorkflowStepExecution;

export interface Workflow {
  id: string;
  name: string;
  status: 'completed' | 'running' | 'failed' | 'pending';
  duration: string;
  timestamp: string;
  app_name?: string;
  // Extended fields
  total_steps?: number;
  error_message?: string;
  steps?: WorkflowStep[];
}

interface WorkflowTableProps {
  workflows: Workflow[];
  searchTerm?: string;
  statusFilter?: string;
  appFilter?: string;
}

function WorkflowRow({ workflow }: { workflow: Workflow }) {
  const [expanded, setExpanded] = useState(false);
  const [steps, setSteps] = useState<WorkflowStep[]>([]);
  const [loadingSteps, setLoadingSteps] = useState(false);

  const handleExpand = async () => {
    if (!expanded && steps.length === 0) {
      setLoadingSteps(true);
      try {
        const detailRes = await api.getWorkflow(workflow.id);
        if (detailRes.success && detailRes.data?.steps) {
          setSteps(detailRes.data.steps);
        }
      } catch (error) {
        console.error('Failed to load steps:', error);
      } finally {
        setLoadingSteps(false);
      }
    }
    setExpanded(!expanded);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'bg-lime-500/10 text-lime-500 border-lime-500/20';
      case 'running':
        return 'bg-blue-500/10 text-blue-500 border-blue-500/20 animate-pulse';
      case 'failed':
        return 'bg-red-500/10 text-red-500 border-red-500/20';
      case 'pending':
        return 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20';
      default:
        return 'bg-zinc-500/10 text-zinc-500 border-zinc-500/20';
    }
  };

  return (
    <div className="border-b border-zinc-800 last:border-0">
      {/* Main Row */}
      <div
        className="flex items-center gap-3 px-4 py-3 hover:bg-zinc-900/50 cursor-pointer transition-colors"
        onClick={handleExpand}
      >
        {/* Expand Icon */}
        <div className="flex-shrink-0 text-zinc-600">
          {expanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        </div>

        {/* Workflow Info */}
        <div className="flex-1 min-w-0 grid grid-cols-5 gap-4">
          <div className="truncate">
            <div className="text-sm font-medium text-white truncate">{workflow.name}</div>
            <div className="text-xs text-zinc-500 truncate">{workflow.app_name || 'N/A'}</div>
          </div>

          <div>
            <Badge variant="outline" className={`text-xs ${getStatusColor(workflow.status)}`}>
              {workflow.status}
            </Badge>
          </div>

          <div className="text-xs text-zinc-500">
            {workflow.total_steps ? `${workflow.total_steps} steps` : 'N/A'}
          </div>

          <div className="text-xs text-zinc-500">{workflow.duration || 'N/A'}</div>

          <div className="text-xs text-zinc-500">
            {new Date(workflow.timestamp).toLocaleString()}
          </div>
        </div>
      </div>

      {/* Expanded Details */}
      {expanded && (
        <div className="px-4 py-3 bg-zinc-900/30 border-t border-zinc-800">
          {/* Workflow Metadata */}
          <div className="grid grid-cols-3 gap-4 text-xs mb-4">
            <div>
              <div className="text-zinc-500 mb-1">Workflow ID</div>
              <div className="text-white font-mono">{workflow.id}</div>
            </div>

            <div>
              <div className="text-zinc-500 mb-1">Started At</div>
              <div className="text-white">{new Date(workflow.timestamp).toLocaleString()}</div>
            </div>

            <div>
              <div className="text-zinc-500 mb-1">Duration</div>
              <div className="text-white">{workflow.duration || 'N/A'}</div>
            </div>

            {workflow.error_message && (
              <div className="col-span-3">
                <div className="text-zinc-500 mb-1">Error</div>
                <div className="text-red-400 font-mono text-xs bg-red-500/5 p-2 rounded border border-red-500/20">
                  {workflow.error_message}
                </div>
              </div>
            )}
          </div>

          {/* Steps */}
          <div className="text-zinc-500 text-xs mb-2 font-medium">Workflow Steps</div>
          {loadingSteps ? (
            <div className="flex items-center justify-center py-4 text-zinc-500">
              <Loader2 size={16} className="animate-spin mr-2" />
              Loading steps...
            </div>
          ) : steps.length > 0 ? (
            <div className="space-y-2">
              {steps.map((step) => (
                <div key={step.id} className="bg-zinc-900 rounded border border-zinc-800 p-3">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <div className="text-zinc-500 font-mono text-xs">#{step.step_number}</div>
                      <div className="text-white text-sm font-medium">{step.step_name}</div>
                      <Badge variant="outline" className="text-xs">
                        {step.step_type}
                      </Badge>
                    </div>
                    <Badge variant="outline" className={`text-xs ${getStatusColor(step.status)}`}>
                      {step.status}
                    </Badge>
                  </div>

                  <div className="grid grid-cols-3 gap-4 text-xs text-zinc-500">
                    {step.started_at && (
                      <div>Started: {new Date(step.started_at).toLocaleTimeString()}</div>
                    )}
                    {step.completed_at && (
                      <div>Completed: {new Date(step.completed_at).toLocaleTimeString()}</div>
                    )}
                    {step.duration_ms && (
                      <div>
                        Duration:{' '}
                        {step.duration_ms > 1000
                          ? `${(step.duration_ms / 1000).toFixed(1)}s`
                          : `${step.duration_ms}ms`}
                      </div>
                    )}
                  </div>

                  {step.error_message && (
                    <div className="mt-2 text-red-400 font-mono text-xs bg-red-500/5 p-2 rounded border border-red-500/20">
                      {step.error_message}
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center text-zinc-500 py-4">No steps available</div>
          )}
        </div>
      )}
    </div>
  );
}

export function WorkflowTable({
  workflows,
  searchTerm = '',
  statusFilter = 'all',
  appFilter = 'all',
}: WorkflowTableProps) {
  // Apply filters
  const filteredWorkflows = workflows.filter((w) => {
    // Search filter
    const matchesSearch =
      !searchTerm ||
      w.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (w.app_name && w.app_name.toLowerCase().includes(searchTerm.toLowerCase()));

    // Status filter
    const matchesStatus = statusFilter === 'all' || w.status === statusFilter;

    // App filter
    const matchesApp = appFilter === 'all' || w.app_name === appFilter;

    return matchesSearch && matchesStatus && matchesApp;
  });

  if (filteredWorkflows.length === 0) {
    return (
      <Card className="p-8 bg-zinc-900/50 border-zinc-800">
        <div className="text-center text-zinc-500">
          {searchTerm || statusFilter !== 'all' || appFilter !== 'all'
            ? 'No workflows match your filters'
            : 'No workflows found'}
        </div>
      </Card>
    );
  }

  return (
    <Card className="bg-zinc-900/50 border-zinc-800 overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-zinc-800 bg-zinc-900/70">
        <div className="grid grid-cols-5 gap-4 text-xs font-medium text-zinc-400 uppercase tracking-wide">
          <div className="pl-7">Workflow / Application</div>
          <div>Status</div>
          <div>Steps</div>
          <div>Duration</div>
          <div>Started</div>
        </div>
      </div>

      {/* Rows */}
      <div className="divide-y divide-zinc-800">
        {filteredWorkflows.map((workflow) => (
          <WorkflowRow key={workflow.id} workflow={workflow} />
        ))}
      </div>
    </Card>
  );
}
