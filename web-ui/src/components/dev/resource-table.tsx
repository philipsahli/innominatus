'use client';

import { useState } from 'react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { ChevronDown, ChevronRight } from 'lucide-react';

export interface Resource {
  id: number;
  application_name: string;
  resource_name: string;
  resource_type: string;
  state: string;
  health_status?: string;
  provider_id?: string;
  workflow_execution_id?: number;
  configuration?: any;
  error_message?: string;
  created_at?: string;
  updated_at?: string;
}

interface ResourceTableProps {
  resources: Resource[];
  searchTerm?: string;
  statusFilter?: string;
  appFilter?: string;
}

function ResourceRow({ resource }: { resource: Resource }) {
  const [expanded, setExpanded] = useState(false);

  const getStateColor = (state: string) => {
    switch (state) {
      case 'active':
        return 'bg-lime-500/10 text-lime-500 border-lime-500/20';
      case 'requested':
        return 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20';
      case 'provisioning':
        return 'bg-blue-500/10 text-blue-500 border-blue-500/20 animate-pulse';
      case 'failed':
        return 'bg-red-500/10 text-red-500 border-red-500/20';
      case 'terminating':
        return 'bg-orange-500/10 text-orange-500 border-orange-500/20';
      case 'terminated':
        return 'bg-zinc-500/10 text-zinc-500 border-zinc-500/20';
      default:
        return 'bg-zinc-500/10 text-zinc-500 border-zinc-500/20';
    }
  };

  return (
    <div className="border-b border-zinc-800 last:border-0">
      {/* Main Row */}
      <div
        className="flex items-center gap-3 px-4 py-3 hover:bg-zinc-900/50 cursor-pointer transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        {/* Expand Icon */}
        <div className="flex-shrink-0 text-zinc-600">
          {expanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        </div>

        {/* Resource Info */}
        <div className="flex-1 min-w-0 grid grid-cols-4 gap-4">
          <div className="truncate">
            <div className="text-sm font-medium text-white truncate">{resource.resource_name}</div>
            <div className="text-xs text-zinc-500 truncate">{resource.application_name}</div>
          </div>

          <div>
            <Badge variant="outline" className="text-xs">
              {resource.resource_type}
            </Badge>
          </div>

          <div>
            <Badge variant="outline" className={`text-xs ${getStateColor(resource.state)}`}>
              {resource.state}
            </Badge>
          </div>

          <div className="text-xs text-zinc-500">
            {resource.created_at ? new Date(resource.created_at).toLocaleString() : 'N/A'}
          </div>
        </div>
      </div>

      {/* Expanded Details */}
      {expanded && (
        <div className="px-4 py-3 bg-zinc-900/30 border-t border-zinc-800">
          <div className="grid grid-cols-2 gap-4 text-xs">
            <div>
              <div className="text-zinc-500 mb-1">Resource ID</div>
              <div className="text-white font-mono">{resource.id}</div>
            </div>

            {resource.provider_id && (
              <div>
                <div className="text-zinc-500 mb-1">Provider</div>
                <div className="text-white">{resource.provider_id}</div>
              </div>
            )}

            {resource.workflow_execution_id && (
              <div>
                <div className="text-zinc-500 mb-1">Workflow Execution ID</div>
                <div className="text-white font-mono">{resource.workflow_execution_id}</div>
              </div>
            )}

            {resource.health_status && (
              <div>
                <div className="text-zinc-500 mb-1">Health Status</div>
                <div className="text-white">{resource.health_status}</div>
              </div>
            )}

            {resource.updated_at && (
              <div>
                <div className="text-zinc-500 mb-1">Last Updated</div>
                <div className="text-white">{new Date(resource.updated_at).toLocaleString()}</div>
              </div>
            )}

            {resource.error_message && (
              <div className="col-span-2">
                <div className="text-zinc-500 mb-1">Error</div>
                <div className="text-red-400 font-mono text-xs bg-red-500/5 p-2 rounded border border-red-500/20">
                  {resource.error_message}
                </div>
              </div>
            )}

            {resource.configuration && Object.keys(resource.configuration).length > 0 && (
              <div className="col-span-2">
                <div className="text-zinc-500 mb-1">Configuration</div>
                <pre className="text-white font-mono text-xs bg-zinc-900 p-2 rounded border border-zinc-800 overflow-auto">
                  {JSON.stringify(resource.configuration, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export function ResourceTable({
  resources,
  searchTerm = '',
  statusFilter = 'all',
  appFilter = 'all',
}: ResourceTableProps) {
  // Apply filters
  const filteredResources = resources.filter((r) => {
    // Search filter
    const matchesSearch =
      !searchTerm ||
      r.resource_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      r.resource_type.toLowerCase().includes(searchTerm.toLowerCase()) ||
      r.application_name.toLowerCase().includes(searchTerm.toLowerCase());

    // Status filter
    const matchesStatus = statusFilter === 'all' || r.state === statusFilter;

    // App filter
    const matchesApp = appFilter === 'all' || r.application_name === appFilter;

    return matchesSearch && matchesStatus && matchesApp;
  });

  if (filteredResources.length === 0) {
    return (
      <Card className="p-8 bg-zinc-900/50 border-zinc-800">
        <div className="text-center text-zinc-500">
          {searchTerm || statusFilter !== 'all' || appFilter !== 'all'
            ? 'No resources match your filters'
            : 'No resources found'}
        </div>
      </Card>
    );
  }

  return (
    <Card className="bg-zinc-900/50 border-zinc-800 overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-zinc-800 bg-zinc-900/70">
        <div className="grid grid-cols-4 gap-4 text-xs font-medium text-zinc-400 uppercase tracking-wide">
          <div className="pl-7">Resource / Application</div>
          <div>Type</div>
          <div>State</div>
          <div>Created</div>
        </div>
      </div>

      {/* Rows */}
      <div className="divide-y divide-zinc-800">
        {filteredResources.map((resource) => (
          <ResourceRow key={resource.id} resource={resource} />
        ))}
      </div>
    </Card>
  );
}
