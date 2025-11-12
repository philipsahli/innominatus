'use client';

import { useEffect, useState } from 'react';
import { useSearchParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Plus, Database } from 'lucide-react';
import {
  DataTable,
  DataTableHeader,
  DataTableHeaderCell,
  DataTableBody,
  DataTableRow,
  DataTableCell,
  DataTableEmpty,
} from '@/components/dev/data-table';
import { StatusBadge } from '@/components/dev/status-badge';
import { api, formatDate, type ResourceInstance } from '@/lib/api';

interface ApplicationDetail {
  name: string;
  status: string;
  environment: string;
  spec: any;
}

export default function ApplicationDetailPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const appName = searchParams.get('name') || '';

  const [application, setApplication] = useState<ApplicationDetail | null>(null);
  const [resources, setResources] = useState<ResourceInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAddResource, setShowAddResource] = useState(false);

  // Add resource form state
  const [newResourceName, setNewResourceName] = useState('');
  const [newResourceType, setNewResourceType] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (appName) {
      loadApplication();
    } else {
      setLoading(false);
      setError('No application name provided');
    }
  }, [appName]);

  const loadApplication = async () => {
    try {
      setLoading(true);
      setError(null);

      // Load application data
      const appResponse = await api.getApplication(appName);
      if (!appResponse.success || !appResponse.data) {
        setError(appResponse.error || 'Failed to load application');
        return;
      }

      // Load spec
      const specsResponse = await api.getSpecs();
      let spec = null;
      if (specsResponse.success && specsResponse.data) {
        spec = specsResponse.data[appName];
      }

      setApplication({
        name: appName,
        status: appResponse.data.status || 'unknown',
        environment: appResponse.data.environment || 'unknown',
        spec: spec,
      });

      // Load resources
      const resourcesResponse = await api.getResources(appName);
      if (resourcesResponse.success && resourcesResponse.data) {
        const resourceArray = Object.values(resourcesResponse.data).flat() as ResourceInstance[];
        setResources(resourceArray);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load application');
    } finally {
      setLoading(false);
    }
  };

  const handleAddResource = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!newResourceName || !newResourceType) {
      return;
    }

    try {
      setSubmitting(true);

      // Create a new resource via the resource creation endpoint
      const response = await api.createResource(appName, newResourceName, newResourceType, {});

      if (response.success) {
        // Reload application data
        await loadApplication();

        // Reset form
        setNewResourceName('');
        setNewResourceType('');
        setShowAddResource(false);
      } else {
        setError(response.error || 'Failed to add resource');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add resource');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link
            href="/dev/applications"
            className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
          >
            <ArrowLeft size={20} />
          </Link>
          <div>
            <div className="h-8 w-48 animate-pulse rounded bg-zinc-200 dark:bg-zinc-800" />
          </div>
        </div>
        <div className="rounded-lg border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
          <div className="space-y-4">
            <div className="h-4 w-32 animate-pulse rounded bg-zinc-200 dark:bg-zinc-800" />
            <div className="h-4 w-48 animate-pulse rounded bg-zinc-200 dark:bg-zinc-800" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !application) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Link
            href="/dev/applications"
            className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
          >
            <ArrowLeft size={20} />
          </Link>
          <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">
            Application Not Found
          </h1>
        </div>
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
          {error || 'Application not found'}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link
            href="/dev/applications"
            className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white"
          >
            <ArrowLeft size={20} />
          </Link>
          <div>
            <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">
              {application.name}
            </h1>
            <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">Application Details</p>
          </div>
        </div>

        <StatusBadge status={application.status as any} />
      </div>

      {/* Application Info */}
      <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
        <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <div className="text-sm text-zinc-600 dark:text-zinc-400">Status</div>
          <div className="mt-1 font-medium text-zinc-900 dark:text-white">
            <StatusBadge status={application.status as any} />
          </div>
        </div>

        <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <div className="text-sm text-zinc-600 dark:text-zinc-400">Environment</div>
          <div className="mt-1 font-medium text-zinc-900 dark:text-white">
            {application.environment}
          </div>
        </div>

        <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <div className="text-sm text-zinc-600 dark:text-zinc-400">Resources</div>
          <div className="mt-1 font-medium text-zinc-900 dark:text-white">
            {resources.length} {resources.length === 1 ? 'resource' : 'resources'}
          </div>
        </div>
      </div>

      {/* Resources Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">Resources</h2>
          <button
            onClick={() => setShowAddResource(!showAddResource)}
            className="flex items-center gap-2 rounded-lg bg-lime-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-lime-600"
          >
            <Plus size={16} />
            Add Resource
          </button>
        </div>

        {/* Add Resource Form */}
        {showAddResource && (
          <form
            onSubmit={handleAddResource}
            className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
          >
            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <div>
                <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  Resource Name
                </label>
                <input
                  type="text"
                  value={newResourceName}
                  onChange={(e) => setNewResourceName(e.target.value)}
                  placeholder="my-database"
                  className="mt-1 block w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 placeholder-zinc-400 focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-white dark:placeholder-zinc-500"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
                  Resource Type
                </label>
                <select
                  value={newResourceType}
                  onChange={(e) => setNewResourceType(e.target.value)}
                  className="mt-1 block w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-white"
                  required
                >
                  <option value="">Select type...</option>
                  <option value="postgres">PostgreSQL Database</option>
                  <option value="namespace">Kubernetes Namespace</option>
                  <option value="s3-bucket">S3 Bucket</option>
                  <option value="vault-space">Vault Space</option>
                  <option value="gitea-repo">Gitea Repository</option>
                </select>
              </div>

              <div className="flex items-end gap-2">
                <button
                  type="submit"
                  disabled={submitting}
                  className="rounded-lg bg-lime-500 px-4 py-2 text-sm font-medium text-white hover:bg-lime-600 disabled:opacity-50"
                >
                  {submitting ? 'Adding...' : 'Add'}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowAddResource(false);
                    setNewResourceName('');
                    setNewResourceType('');
                  }}
                  className="rounded-lg border border-zinc-300 px-4 py-2 text-sm font-medium text-zinc-700 hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
                >
                  Cancel
                </button>
              </div>
            </div>
          </form>
        )}

        {/* Resources Table */}
        <DataTable>
          <DataTableHeader>
            <DataTableHeaderCell>Name</DataTableHeaderCell>
            <DataTableHeaderCell>Type</DataTableHeaderCell>
            <DataTableHeaderCell>State</DataTableHeaderCell>
            <DataTableHeaderCell>Provider</DataTableHeaderCell>
            <DataTableHeaderCell>Created</DataTableHeaderCell>
          </DataTableHeader>

          <DataTableBody>
            {resources.length === 0 ? (
              <DataTableEmpty message="No resources yet. Add one to get started." />
            ) : (
              resources.map((resource) => (
                <DataTableRow key={resource.id}>
                  <DataTableCell mono>
                    <div className="flex items-center gap-2">
                      <Database size={16} className="text-zinc-400" />
                      {resource.resource_name}
                    </div>
                  </DataTableCell>

                  <DataTableCell>
                    <span className="rounded bg-zinc-100 px-2 py-1 text-xs font-medium text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                      {resource.resource_type}
                    </span>
                  </DataTableCell>

                  <DataTableCell>
                    <StatusBadge status={resource.state} />
                  </DataTableCell>

                  <DataTableCell>
                    <span className="text-zinc-500">{resource.provider_id || '—'}</span>
                  </DataTableCell>

                  <DataTableCell mono className="text-zinc-500">
                    {formatDate(resource.created_at)}
                  </DataTableCell>
                </DataTableRow>
              ))
            )}
          </DataTableBody>
        </DataTable>
      </div>

      {/* Score Spec (Optional) */}
      {application.spec && (
        <div className="space-y-4">
          <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">
            Score Specification
          </h2>
          <div className="rounded-lg border border-zinc-200 bg-zinc-50 p-4 dark:border-zinc-800 dark:bg-zinc-900">
            <pre className="overflow-x-auto text-xs text-zinc-700 dark:text-zinc-300">
              {typeof application.spec === 'string'
                ? application.spec
                : JSON.stringify(application.spec, null, 2)}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
