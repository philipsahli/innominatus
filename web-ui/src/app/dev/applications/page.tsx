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
import { api, formatDate, type Application } from '@/lib/api';

export default function ApplicationsPage() {
  const [applications, setApplications] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [wizardOpen, setWizardOpen] = useState(false);
  const [selectedTeam, setSelectedTeam] = useState<string>('all');

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

  // Get unique teams from applications
  const teams = Array.from(new Set(applications.map((app) => app.team))).sort();

  // Filter applications by selected team
  const filteredApplications =
    selectedTeam === 'all'
      ? applications
      : applications.filter((app) => app.team === selectedTeam);

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

      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <label htmlFor="team-filter" className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
            Team:
          </label>
          <select
            id="team-filter"
            value={selectedTeam}
            onChange={(e) => setSelectedTeam(e.target.value)}
            className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm text-zinc-900 shadow-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-white"
          >
            <option value="all">All Teams</option>
            {teams.map((team) => (
              <option key={team} value={team}>
                {team}
              </option>
            ))}
          </select>
        </div>
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
          <DataTableHeaderCell>Team</DataTableHeaderCell>
          <DataTableHeaderCell>Status</DataTableHeaderCell>
          <DataTableHeaderCell>Environment</DataTableHeaderCell>
          <DataTableHeaderCell>Resources</DataTableHeaderCell>
          <DataTableHeaderCell>Updated</DataTableHeaderCell>
          <DataTableHeaderCell>Actions</DataTableHeaderCell>
        </DataTableHeader>

        <DataTableBody>
          {loading ? (
            <DataTableLoading />
          ) : filteredApplications.length === 0 ? (
            <DataTableEmpty message={selectedTeam === 'all' ? 'No applications deployed yet' : `No applications found for team "${selectedTeam}"`} />
          ) : (
            filteredApplications.map((app) => (
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
                  <span className="inline-flex items-center rounded-md bg-lime-50 px-2 py-1 text-xs font-medium text-lime-700 ring-1 ring-inset ring-lime-600/20 dark:bg-lime-500/10 dark:text-lime-400 dark:ring-lime-500/30">
                    {app.team}
                  </span>
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
                  {formatDate(app.lastUpdated)}
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
          Showing {filteredApplications.length} of {applications.length} application{applications.length !== 1 ? 's' : ''}
          {selectedTeam !== 'all' && ` (filtered by team: ${selectedTeam})`}
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
