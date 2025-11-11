import { useState } from 'react';
import { StepProps } from './types';

export function StepBasicInfo({ data, onChange, onNext }: StepProps) {
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    // Validate app name
    const trimmedName = data.appName.trim();
    if (!trimmedName) {
      setError('Application name is required');
      return;
    }

    // Validate DNS-compliant name (lowercase letters, numbers, hyphens)
    const namePattern = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
    if (!namePattern.test(trimmedName)) {
      setError('Application name must contain only lowercase letters, numbers, and hyphens');
      return;
    }

    setError(null);
    onNext();
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">Application Basics</h2>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Configure the basic information for your application
        </p>
      </div>

      {/* App Name */}
      <div>
        <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Application Name *
        </label>
        <input
          type="text"
          required
          value={data.appName}
          onChange={(e) => {
            onChange({ appName: e.target.value });
            setError(null); // Clear error when user types
          }}
          placeholder="my-app"
          className={`mt-1 block w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-1 ${
            error
              ? 'border-red-500 focus:border-red-500 focus:ring-red-500'
              : 'border-zinc-300 focus:border-lime-500 focus:ring-lime-500'
          } bg-white dark:border-zinc-700 dark:bg-zinc-950 dark:text-white`}
        />
        {error ? (
          <p className="mt-1 text-xs text-red-600 dark:text-red-400">{error}</p>
        ) : (
          <p className="mt-1 text-xs text-zinc-500">Lowercase letters, numbers, and hyphens only</p>
        )}
      </div>

      {/* Environment */}
      <div>
        <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Environment Type
        </label>
        <select
          value={data.environment}
          onChange={(e) =>
            onChange({
              environment: e.target.value as 'kubernetes' | 'production' | 'staging',
            })
          }
          className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
        >
          <option value="kubernetes">Kubernetes</option>
          <option value="production">Production</option>
          <option value="staging">Staging</option>
        </select>
      </div>

      {/* TTL */}
      <div>
        <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Time to Live (TTL)
        </label>
        <select
          value={data.ttl}
          onChange={(e) => onChange({ ttl: e.target.value })}
          className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
        >
          <option value="1h">1 hour</option>
          <option value="24h">24 hours</option>
          <option value="168h">7 days</option>
        </select>
        <p className="mt-1 text-xs text-zinc-500">How long the environment should remain active</p>
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-3">
        <button
          type="submit"
          className="rounded-lg bg-lime-500 px-4 py-2 text-sm font-medium text-white hover:bg-lime-600"
        >
          Next: Container Config
        </button>
      </div>
    </form>
  );
}
