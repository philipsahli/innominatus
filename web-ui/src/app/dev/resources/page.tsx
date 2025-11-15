'use client';

import { useEffect, useState } from 'react';
import { Database, RefreshCw, Filter, X } from 'lucide-react';
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
import { formatAsYAML } from '@/lib/formatters';

export default function ResourcesPage() {
  const [resources, setResources] = useState<ResourceInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedResource, setSelectedResource] = useState<ResourceInstance | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [typeFilter, setTypeFilter] = useState<string>('all');
  const [stateFilter, setStateFilter] = useState<string>('all');
  const [appFilter, setAppFilter] = useState<string>('all');

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

  // Auto-refresh effect
  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(async () => {
      try {
        const response = await api.getResources();
        if (response.success && response.data) {
          const allResources = Object.values(response.data).flat();
          setResources(allResources);
        }
      } catch (err) {
        // Silently fail on auto-refresh errors
        console.error('Auto-refresh failed:', err);
      }
    }, 15000); // Refresh every 15 seconds

    return () => clearInterval(interval);
  }, [autoRefresh]);

  // Extract unique values for filters
  const uniqueTypes = Array.from(new Set(resources.map((r) => r.resource_type))).sort();
  const uniqueStates = Array.from(
    new Set(resources.map((r) => r.state || 'active').filter(Boolean))
  ).sort();
  const uniqueApps = Array.from(
    new Set(resources.map((r) => r.application_name).filter(Boolean))
  ).sort();

  // Apply filters
  const filteredResources = resources.filter((resource) => {
    const matchesType = typeFilter === 'all' || resource.resource_type === typeFilter;
    const matchesState = stateFilter === 'all' || (resource.state || 'active') === stateFilter;
    const matchesApp = appFilter === 'all' || resource.application_name === appFilter;
    return matchesType && matchesState && matchesApp;
  });

  // Count active filters
  const activeFilterCount =
    (typeFilter !== 'all' ? 1 : 0) +
    (stateFilter !== 'all' ? 1 : 0) +
    (appFilter !== 'all' ? 1 : 0);

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

        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 text-sm text-zinc-500">
            <Database size={16} />
            {!loading && <span>{resources.filter((r) => r.state === 'active').length} active</span>}
          </div>

          <button
            onClick={() => setAutoRefresh(!autoRefresh)}
            className={`flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm transition-colors ${
              autoRefresh
                ? 'border-lime-200 bg-lime-50 text-lime-700 hover:bg-lime-100 dark:border-lime-900 dark:bg-lime-950 dark:text-lime-400'
                : 'border-zinc-200 bg-white text-zinc-600 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-400'
            }`}
            title={autoRefresh ? 'Auto-refresh enabled (15s)' : 'Auto-refresh disabled'}
          >
            <RefreshCw size={14} className={autoRefresh ? 'animate-spin' : ''} />
            {autoRefresh ? 'Auto' : 'Manual'}
          </button>
        </div>
      </div>

      {/* Error State */}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
          {error}
        </div>
      )}

      {/* Filters */}
      {!loading && resources.length > 0 && (
        <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <Filter size={16} className="text-zinc-500" />
              <span className="text-sm font-medium text-zinc-900 dark:text-white">Filters</span>
              {activeFilterCount > 0 && (
                <span className="rounded-full bg-lime-100 px-2 py-0.5 text-xs font-medium text-lime-700 dark:bg-lime-950 dark:text-lime-400">
                  {activeFilterCount}
                </span>
              )}
            </div>
            {activeFilterCount > 0 && (
              <button
                onClick={() => {
                  setTypeFilter('all');
                  setStateFilter('all');
                  setAppFilter('all');
                }}
                className="flex items-center gap-1 text-xs text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
              >
                <X size={12} />
                Clear all
              </button>
            )}
          </div>

          <div className="flex flex-wrap gap-3">
            {/* Type Filter */}
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
              className="rounded border border-zinc-200 bg-white px-3 py-1.5 text-sm text-zinc-900 hover:border-zinc-300 focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-800 dark:bg-zinc-950 dark:text-white"
            >
              <option value="all">All Types</option>
              {uniqueTypes.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>

            {/* State Filter */}
            <select
              value={stateFilter}
              onChange={(e) => setStateFilter(e.target.value)}
              className="rounded border border-zinc-200 bg-white px-3 py-1.5 text-sm text-zinc-900 hover:border-zinc-300 focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-800 dark:bg-zinc-950 dark:text-white"
            >
              <option value="all">All States</option>
              {uniqueStates.map((state) => (
                <option key={state} value={state}>
                  {state}
                </option>
              ))}
            </select>

            {/* App Filter */}
            <select
              value={appFilter}
              onChange={(e) => setAppFilter(e.target.value)}
              className="rounded border border-zinc-200 bg-white px-3 py-1.5 text-sm text-zinc-900 hover:border-zinc-300 focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-800 dark:bg-zinc-950 dark:text-white"
            >
              <option value="all">All Applications</option>
              {uniqueApps.map((app) => (
                <option key={app} value={app}>
                  {app}
                </option>
              ))}
            </select>
          </div>

          {/* Filter Results Count */}
          {activeFilterCount > 0 && (
            <div className="mt-3 pt-3 border-t border-zinc-200 text-xs text-zinc-500 dark:border-zinc-800">
              Showing {filteredResources.length} of {resources.length} resources
            </div>
          )}
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
          ) : filteredResources.length === 0 ? (
            <DataTableEmpty message="No resources match the current filters" />
          ) : (
            filteredResources.map((resource) => (
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
                      {formatAsYAML(selectedResource.configuration)}
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
