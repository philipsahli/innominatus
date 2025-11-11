'use client';

import { useState } from 'react';
import { Search } from 'lucide-react';
import { GraphVisualization } from '@/components/graph-visualization';

export default function GraphPage() {
  const [appName, setAppName] = useState('');
  const [submittedApp, setSubmittedApp] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setSubmittedApp(appName);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white">Dependency Graph</h1>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Visualize relationships between applications, resources, and workflows
        </p>
      </div>

      {/* Search Bar */}
      <form onSubmit={handleSubmit} className="flex items-center gap-2">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" size={16} />
          <input
            type="text"
            value={appName}
            onChange={(e) => setAppName(e.target.value)}
            placeholder="Enter application name..."
            className="w-full rounded-lg border border-zinc-300 bg-white px-10 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-900 dark:text-white"
          />
        </div>
        <button
          type="submit"
          className="rounded-lg bg-lime-500 px-4 py-2 text-sm font-medium text-white hover:bg-lime-600"
        >
          View Graph
        </button>
      </form>

      {/* Graph Visualization */}
      {submittedApp ? (
        <div className="h-[calc(100vh-16rem)] rounded-lg border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
          <GraphVisualization app={submittedApp} />
        </div>
      ) : (
        <div className="flex h-96 items-center justify-center rounded-lg border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900">
          <div className="text-center">
            <p className="text-sm text-zinc-500">
              Enter an application name to view its dependency graph
            </p>
            <p className="mt-1 text-xs text-zinc-400">
              Graph shows relationships between apps, resources, providers, and workflows
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
