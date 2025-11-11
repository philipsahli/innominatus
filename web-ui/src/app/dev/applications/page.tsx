'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Plus, Trash2 } from 'lucide-react';
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
import { DeployWizard } from '@/components/deploy-wizard/deploy-wizard';
import { api, type Application } from '@/lib/api';

export default function ApplicationsPage() {
  const [applications, setApplications] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [wizardOpen, setWizardOpen] = useState(false);

  const loadApplications = async () => {
    try {
      setLoading(true);
      const response = await api.getApplications();
      if (response.success && response.data) {
        setApplications(response.data);
      } else {
        setError(response.error || 'Failed to load applications');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load applications');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadApplications();
  }, []);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">Applications</h1>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Deployed Score specifications
          </p>
        </div>

        <button
          onClick={() => setWizardOpen(true)}
          className="flex items-center gap-2 rounded-lg bg-lime-500 px-4 py-2 text-sm font-medium text-white hover:bg-lime-600"
        >
          <Plus size={16} />
          Deploy New
        </button>
      </div>

      {/* Error State */}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
          {error}
        </div>
      )}

      {/* Applications Table */}
      <DataTable>
        <DataTableHeader>
          <DataTableHeaderCell>Name</DataTableHeaderCell>
          <DataTableHeaderCell>Status</DataTableHeaderCell>
          <DataTableHeaderCell>Environment</DataTableHeaderCell>
          <DataTableHeaderCell>Resources</DataTableHeaderCell>
          <DataTableHeaderCell>Updated</DataTableHeaderCell>
          <DataTableHeaderCell>Actions</DataTableHeaderCell>
        </DataTableHeader>

        <DataTableBody>
          {loading ? (
            <DataTableLoading />
          ) : applications.length === 0 ? (
            <DataTableEmpty message="No applications deployed yet" />
          ) : (
            applications.map((app) => (
              <DataTableRow key={app.name}>
                <DataTableCell mono>
                  <Link
                    href={`/dev/applications/detail/?name=${app.name}`}
                    className="font-medium hover:text-lime-600 dark:hover:text-lime-400"
                  >
                    {app.name}
                  </Link>
                </DataTableCell>

                <DataTableCell>
                  <StatusBadge status={app.status || 'active'} />
                </DataTableCell>

                <DataTableCell>
                  <span className="text-zinc-500">{app.environment}</span>
                </DataTableCell>

                <DataTableCell>
                  <span className="text-zinc-500">{app.resources || 0} resources</span>
                </DataTableCell>

                <DataTableCell mono className="text-zinc-500">
                  {app.lastUpdated ? new Date(app.lastUpdated).toLocaleString() : '—'}
                </DataTableCell>

                <DataTableCell>
                  <div className="flex items-center gap-2">
                    <Link
                      href={`/dev/applications/detail/?name=${app.name}`}
                      className="text-xs text-lime-600 hover:text-lime-700 dark:text-lime-400"
                    >
                      View
                    </Link>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        // TODO: Implement delete
                      }}
                      className="text-xs text-red-600 hover:text-red-700 dark:text-red-400"
                      title="Delete application"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </DataTableCell>
              </DataTableRow>
            ))
          )}
        </DataTableBody>
      </DataTable>

      {/* Stats */}
      {!loading && applications.length > 0 && (
        <div className="text-sm text-zinc-500">
          Showing {applications.length} application{applications.length !== 1 ? 's' : ''}
        </div>
      )}

      {/* Deploy Wizard */}
      <DeployWizard
        isOpen={wizardOpen}
        onClose={() => setWizardOpen(false)}
        onSuccess={() => {
          setWizardOpen(false);
          loadApplications();
        }}
      />
    </div>
  );
}
