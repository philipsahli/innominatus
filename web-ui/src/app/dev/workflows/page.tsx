'use client';

import { useEffect, useState, useRef, useCallback } from 'react';
import Link from 'next/link';
import { RotateCw } from 'lucide-react';
import {
  DataTable,
  DataTableHeader,
  DataTableHeaderCell,
  DataTableBody,
  DataTableRow,
  DataTableCell,
  DataTableEmpty,
  DataTableLoading,
} from '@/components/dev/data-table';
import { StatusBadge } from '@/components/dev/status-badge';
import { api, type WorkflowExecution } from '@/lib/api';

export default function WorkflowsPage() {
  const [workflows, setWorkflows] = useState<WorkflowExecution[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const hasRunningWorkflowsRef = useRef(false);

  const loadWorkflows = useCallback(async () => {
    try {
      setLoading(true);
      const response = await api.getWorkflows();
      if (response.success && response.data) {
        // Handle both paginated response (data.data) and direct array
        const workflowsData = response.data.data || response.data;
        const workflowsList = Array.isArray(workflowsData) ? workflowsData : [];
        setWorkflows(workflowsList);

        // Update ref without causing re-render
        hasRunningWorkflowsRef.current = workflowsList.some((w) => w.status === 'running');
      } else {
        setError(response.error || 'Failed to load workflows');
        setWorkflows([]);
        hasRunningWorkflowsRef.current = false;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load workflows');
      setWorkflows([]);
      hasRunningWorkflowsRef.current = false;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadWorkflows();

    // Auto-refresh every 5s if there are running workflows
    const interval = setInterval(() => {
      if (hasRunningWorkflowsRef.current) {
        loadWorkflows();
      }
    }, 5000);

    return () => clearInterval(interval);
  }, []); // Empty dependency array - run once on mount

  const formatDuration = (start?: string, end?: string) => {
    if (!start) return '—';
    const startTime = new Date(start).getTime();
    const endTime = end ? new Date(end).getTime() : Date.now();
    const duration = Math.floor((endTime - startTime) / 1000);

    if (duration < 60) return `${duration}s`;
    if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`;
    return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">
            Workflow Executions
          </h1>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Track workflow execution history and status
          </p>
        </div>

        <div className="flex items-center gap-4">
          {workflows?.some((w) => w.status === 'running') && (
            <span className="flex items-center gap-2 text-sm text-zinc-500">
              <div className="h-2 w-2 animate-pulse rounded-full bg-blue-500" />
              Auto-refreshing...
            </span>
          )}
          <button
            onClick={loadWorkflows}
            className="flex items-center gap-2 rounded-lg border border-zinc-300 px-3 py-1.5 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
          >
            <RotateCw size={14} />
            Refresh
          </button>
        </div>
      </div>

      {/* Error State */}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
          {error}
        </div>
      )}

      {/* Workflows Table */}
      <DataTable>
        <DataTableHeader>
          <DataTableHeaderCell className="w-8">{''}</DataTableHeaderCell>
          <DataTableHeaderCell>ID</DataTableHeaderCell>
          <DataTableHeaderCell>Workflow</DataTableHeaderCell>
          <DataTableHeaderCell>Status</DataTableHeaderCell>
          <DataTableHeaderCell>Application</DataTableHeaderCell>
          <DataTableHeaderCell>Started</DataTableHeaderCell>
          <DataTableHeaderCell>Duration</DataTableHeaderCell>
          <DataTableHeaderCell>Actions</DataTableHeaderCell>
        </DataTableHeader>

        <DataTableBody>
          {loading ? (
            <DataTableLoading />
          ) : workflows.length === 0 ? (
            <DataTableEmpty message="No workflow executions yet" />
          ) : (
            workflows.map((workflow) => (
              <DataTableRow key={workflow.id}>
                <DataTableCell>{''}</DataTableCell>

                <DataTableCell mono className="text-zinc-500">
                  {workflow.id.substring(0, 8)}
                </DataTableCell>

                <DataTableCell mono>
                  <span className="font-medium">{workflow.name}</span>
                </DataTableCell>

                <DataTableCell>
                  <StatusBadge status={workflow.status || 'pending'} />
                </DataTableCell>

                <DataTableCell>
                  {workflow.app_name ? (
                    <Link
                      href={`/dev/applications/${workflow.app_name}`}
                      className="text-lime-600 hover:text-lime-700 dark:text-lime-400"
                    >
                      {workflow.app_name}
                    </Link>
                  ) : (
                    <span className="text-zinc-400">—</span>
                  )}
                </DataTableCell>

                <DataTableCell mono className="text-zinc-500">
                  {workflow.timestamp ? new Date(workflow.timestamp).toLocaleString() : '—'}
                </DataTableCell>

                <DataTableCell mono className="text-zinc-500">
                  {workflow.duration || '—'}
                </DataTableCell>

                <DataTableCell>
                  <div className="flex items-center gap-2">
                    {workflow.status === 'failed' && (
                      <button className="text-xs text-lime-600 hover:text-lime-700 dark:text-lime-400">
                        Retry
                      </button>
                    )}
                    <Link
                      href={`/dev/graph?workflow=${workflow.id}`}
                      className="text-xs text-zinc-600 hover:text-lime-600 dark:text-zinc-400"
                    >
                      Graph
                    </Link>
                  </div>
                </DataTableCell>
              </DataTableRow>
            ))
          )}
        </DataTableBody>
      </DataTable>

      {/* Stats */}
      {!loading && workflows && workflows.length > 0 && (
        <div className="flex items-center gap-4 text-sm text-zinc-500">
          <span>Total: {workflows.length}</span>
          <span>Running: {workflows.filter((w) => w.status === 'running').length}</span>
          <span>Completed: {workflows.filter((w) => w.status === 'completed').length}</span>
          {workflows.filter((w) => w.status === 'failed').length > 0 && (
            <span className="text-red-600">
              Failed: {workflows.filter((w) => w.status === 'failed').length}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
