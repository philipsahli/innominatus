'use client';

import Link from 'next/link';
import { Package, Database, GitBranch, Network } from 'lucide-react';
import { ActivityFeed } from '@/components/dev/activity-feed';

export default function DevHome() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">
          Developer Dashboard
        </h1>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Minimalistic, data-focused view of your platform
        </p>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-4 gap-4">
        <StatCard
          title="Applications"
          value="—"
          href="/dev/applications"
          icon={<Package size={16} />}
        />
        <StatCard title="Resources" value="—" href="/dev/resources" icon={<Database size={16} />} />
        <StatCard
          title="Workflows"
          value="—"
          href="/dev/workflows"
          icon={<GitBranch size={16} />}
        />
        <StatCard title="Graph" value="Visualize" href="/dev/graph" icon={<Network size={16} />} />
      </div>

      {/* Live Activity Feed */}
      <ActivityFeed maxEvents={15} />
    </div>
  );
}

function StatCard({
  title,
  value,
  href,
  icon,
}: {
  title: string;
  value: string;
  href: string;
  icon: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      className="group rounded-lg border border-zinc-200 p-4 transition-colors hover:border-zinc-300 hover:bg-zinc-50 dark:border-zinc-800 dark:hover:border-zinc-700 dark:hover:bg-zinc-900"
    >
      <div className="flex items-center gap-2 text-zinc-600 dark:text-zinc-400">
        {icon}
        <span className="text-xs font-medium uppercase tracking-wide">{title}</span>
      </div>
      <div className="mt-2 text-2xl font-semibold text-zinc-900 dark:text-white">{value}</div>
    </Link>
  );
}
