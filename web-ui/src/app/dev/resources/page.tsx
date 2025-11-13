'use client';

import { useEffect, useState } from 'react';
import { Database } from 'lucide-react';
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
import { StatusBadge, StatusDot } from '@/components/dev/status-badge';
import { CopyableText } from '@/components/dev/code-block';
import { api, type ResourceInstance } from '@/lib/api';

export default function ResourcesPage() {
  const [resources, setResources] = useState<ResourceInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedResource, setSelectedResource] = useState<ResourceInstance | null>(null);

  useEffect(() => {
    async function loadResources() {
      try {
        setLoading(true);
        const response = await api.getResources();
        if (response.success && response.data) {
          // Flatten the grouped resources
          const allResources = Object.values(response.data).flat();
          setResources(allResources);
        } else {
          setError(response.error || 'Failed to load resources');
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load resources');
      } finally {
        setLoading(false);
      }
    }

    loadResources();
  }, []);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">Resources</h1>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Infrastructure resources provisioned by the platform
          </p>
        </div>

        <div className="flex items-center gap-2 text-sm text-zinc-500">
          <Database size={16} />
          {!loading && <span>{resources.filter((r) => r.state === 'active').length} active</span>}
        </div>
      </div>

      {/* Error State */}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
          {error}
        </div>
      )}

      {/* Resources Table */}
      <DataTable>
        <DataTableHeader>
          <DataTableHeaderCell className="w-8">{''}</DataTableHeaderCell>
          <DataTableHeaderCell>Name</DataTableHeaderCell>
          <DataTableHeaderCell>Type</DataTableHeaderCell>
          <DataTableHeaderCell>State</DataTableHeaderCell>
          <DataTableHeaderCell>Provider</DataTableHeaderCell>
          <DataTableHeaderCell>Hints</DataTableHeaderCell>
          <DataTableHeaderCell>Actions</DataTableHeaderCell>
        </DataTableHeader>

        <DataTableBody>
          {loading ? (
            <DataTableLoading />
          ) : resources.length === 0 ? (
            <DataTableEmpty message="No resources provisioned yet" />
          ) : (
            resources.map((resource) => (
              <DataTableRow key={resource.id}>
                <DataTableCell>
                  <StatusDot status={resource.state || 'active'} />
                </DataTableCell>

                <DataTableCell mono>
                  <button
                    onClick={() => setSelectedResource(resource)}
                    className="font-medium hover:text-lime-600 dark:hover:text-lime-400"
                  >
                    {resource.resource_name}
                  </button>
                </DataTableCell>

                <DataTableCell>
                  <span className="rounded bg-zinc-100 px-2 py-0.5 font-mono text-xs text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                    {resource.resource_type}
                  </span>
                </DataTableCell>

                <DataTableCell>
                  <StatusBadge status={resource.state || 'active'} />
                </DataTableCell>

                <DataTableCell className="text-zinc-500">
                  {resource.provider_id || '—'}
                </DataTableCell>

                <DataTableCell>
                  {resource.hints && Array.isArray(resource.hints) && resource.hints.length > 0 ? (
                    <div className="flex items-center gap-2 text-xs text-zinc-500">
                      <span className="truncate" title={resource.hints[0].label}>
                        {resource.hints[0].label}: {resource.hints[0].value.substring(0, 30)}
                        {resource.hints[0].value.length > 30 && '...'}
                      </span>
                      {resource.hints.length > 1 && (
                        <span className="text-zinc-400">+{resource.hints.length - 1} more</span>
                      )}
                    </div>
                  ) : (
                    <span className="text-zinc-400">—</span>
                  )}
                </DataTableCell>

                <DataTableCell>
                  <button
                    onClick={() => setSelectedResource(resource)}
                    className="text-xs text-lime-600 hover:text-lime-700 dark:text-lime-400"
                  >
                    Details
                  </button>
                </DataTableCell>
              </DataTableRow>
            ))
          )}
        </DataTableBody>
      </DataTable>

      {/* Stats */}
      {!loading && resources.length > 0 && (
        <div className="flex items-center gap-4 text-sm text-zinc-500">
          <span>Total: {resources.length}</span>
          <span>Active: {resources.filter((r) => r.state === 'active').length}</span>
          <span>Provisioning: {resources.filter((r) => r.state === 'provisioning').length}</span>
          {resources.filter((r) => r.state === 'failed').length > 0 && (
            <span className="text-red-600">
              Failed: {resources.filter((r) => r.state === 'failed').length}
            </span>
          )}
        </div>
      )}

      {/* Detail Pane */}
      {selectedResource && (
        <div className="fixed inset-y-0 right-0 z-50 w-full max-w-2xl border-l border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-950">
          <div className="flex h-full flex-col">
            {/* Header */}
            <div className="flex items-center justify-between border-b border-zinc-200 px-6 py-4 dark:border-zinc-800">
              <div>
                <h2 className="font-mono text-lg font-semibold text-zinc-900 dark:text-white">
                  {selectedResource.resource_name}
                </h2>
                <p className="mt-1 text-sm text-zinc-500">{selectedResource.resource_type}</p>
              </div>
              <button
                onClick={() => setSelectedResource(null)}
                className="rounded p-2 hover:bg-zinc-100 dark:hover:bg-zinc-800"
              >
                ✕
              </button>
            </div>

            {/* Content */}
            <div className="flex-1 overflow-y-auto p-6">
              <div className="space-y-6">
                {/* Status */}
                <div>
                  <h3 className="text-xs font-medium uppercase tracking-wide text-zinc-500">
                    Status
                  </h3>
                  <div className="mt-2">
                    <StatusBadge status={selectedResource.state || 'active'} />
                  </div>
                </div>

                {/* Hints */}
                {selectedResource.hints &&
                  Array.isArray(selectedResource.hints) &&
                  selectedResource.hints.length > 0 && (
                    <div>
                      <h3 className="text-xs font-medium uppercase tracking-wide text-zinc-500">
                        Quick Access
                      </h3>
                      <div className="mt-2 grid grid-cols-1 gap-2">
                        {selectedResource.hints.map((hint, index) => (
                          <div
                            key={index}
                            className="flex items-center justify-between rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-2 dark:border-zinc-800 dark:bg-zinc-900"
                          >
                            <div className="flex-1 min-w-0">
                              <div className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                                {hint.label}
                              </div>
                              <div className="mt-1 truncate text-xs text-zinc-500">
                                {hint.value}
                              </div>
                            </div>
                            <button
                              onClick={() => {
                                if (hint.type === 'url' || hint.type === 'dashboard') {
                                  window.open(hint.value, '_blank', 'noopener,noreferrer');
                                } else {
                                  navigator.clipboard.writeText(hint.value);
                                }
                              }}
                              className="ml-2 text-lime-600 hover:text-lime-700 dark:text-lime-400"
                              title={
                                hint.type === 'url' || hint.type === 'dashboard'
                                  ? 'Open in new tab'
                                  : 'Copy to clipboard'
                              }
                            >
                              {hint.type === 'url' || hint.type === 'dashboard' ? '↗' : '📋'}
                            </button>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                {/* Configuration */}
                {selectedResource.configuration && (
                  <div>
                    <h3 className="text-xs font-medium uppercase tracking-wide text-zinc-500">
                      Configuration
                    </h3>
                    <pre className="mt-2 rounded-lg border border-zinc-200 bg-zinc-50 p-4 text-xs dark:border-zinc-800 dark:bg-zinc-900">
                      {JSON.stringify(selectedResource.configuration, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
