import { CheckCircle2, AlertCircle } from 'lucide-react';
import { StepProps } from './types';
import { generateScoreYaml } from './yaml-generator';
import { useState, useEffect } from 'react';

export function StepReview({
  data,
  onPrev,
  onSubmit,
}: StepProps & { onSubmit: (yaml: string) => Promise<void> }) {
  const [deploying, setDeploying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [yaml, setYaml] = useState<string>('');

  // Generate YAML with error handling whenever data changes
  useEffect(() => {
    try {
      const generatedYaml = generateScoreYaml(data);
      setYaml(generatedYaml);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid configuration');
      setYaml('');
    }
  }, [data]);

  const handleDeploy = async () => {
    if (error || !yaml) {
      return; // Don't deploy if there's a validation error
    }

    try {
      setDeploying(true);
      await onSubmit(yaml);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Deployment failed');
    } finally {
      setDeploying(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">Review & Deploy</h2>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Review your configuration and deploy the application
        </p>
      </div>

      {/* Summary Cards */}
      <div className="space-y-4">
        {/* Basic Info Summary */}
        <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
          <div className="mb-2 flex items-center gap-2">
            <CheckCircle2 size={18} className="text-lime-600" />
            <h3 className="text-sm font-medium text-zinc-900 dark:text-white">Application Info</h3>
          </div>
          <dl className="space-y-1 text-sm">
            <div className="flex justify-between">
              <dt className="text-zinc-600 dark:text-zinc-400">Name:</dt>
              <dd className="font-mono text-zinc-900 dark:text-white">{data.appName}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-zinc-600 dark:text-zinc-400">Environment:</dt>
              <dd className="font-mono text-zinc-900 dark:text-white">{data.environment}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-zinc-600 dark:text-zinc-400">TTL:</dt>
              <dd className="font-mono text-zinc-900 dark:text-white">{data.ttl}</dd>
            </div>
          </dl>
        </div>

        {/* Container Summary */}
        <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
          <div className="mb-2 flex items-center gap-2">
            <CheckCircle2 size={18} className="text-lime-600" />
            <h3 className="text-sm font-medium text-zinc-900 dark:text-white">
              Container Configuration
            </h3>
          </div>
          <dl className="space-y-1 text-sm">
            <div className="flex justify-between">
              <dt className="text-zinc-600 dark:text-zinc-400">Image:</dt>
              <dd className="font-mono text-zinc-900 dark:text-white">{data.container.image}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-zinc-600 dark:text-zinc-400">Port:</dt>
              <dd className="font-mono text-zinc-900 dark:text-white">{data.container.port}</dd>
            </div>
            {Object.keys(data.container.envVars).length > 0 && (
              <div>
                <dt className="text-zinc-600 dark:text-zinc-400">Environment Variables:</dt>
                <dd className="mt-1 space-y-1">
                  {Object.entries(data.container.envVars).map(([key, value]) => (
                    <div key={key} className="font-mono text-xs text-zinc-900 dark:text-white">
                      {key}={value}
                    </div>
                  ))}
                </dd>
              </div>
            )}
            {(data.container.cpuRequest || data.container.memoryRequest) && (
              <div className="flex justify-between">
                <dt className="text-zinc-600 dark:text-zinc-400">Resources:</dt>
                <dd className="font-mono text-zinc-900 dark:text-white">
                  {data.container.cpuRequest && `CPU: ${data.container.cpuRequest}`}
                  {data.container.cpuRequest && data.container.memoryRequest && ', '}
                  {data.container.memoryRequest && `Memory: ${data.container.memoryRequest}`}
                </dd>
              </div>
            )}
          </dl>
        </div>

        {/* Resources Summary */}
        {data.resources.length > 0 && (
          <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
            <div className="mb-2 flex items-center gap-2">
              <CheckCircle2 size={18} className="text-lime-600" />
              <h3 className="text-sm font-medium text-zinc-900 dark:text-white">
                Resource Requirements ({data.resources.length})
              </h3>
            </div>
            <div className="space-y-2">
              {data.resources.map((resource, index) => (
                <div
                  key={index}
                  className="rounded border border-zinc-100 bg-zinc-50 p-2 dark:border-zinc-900 dark:bg-zinc-900"
                >
                  <div className="mb-1 flex items-center justify-between">
                    <span className="font-mono text-sm text-zinc-900 dark:text-white">
                      {resource.name}
                    </span>
                    <span className="text-xs text-zinc-500">{resource.type}</span>
                  </div>
                  {Object.keys(resource.properties).length > 0 && (
                    <div className="text-xs text-zinc-600 dark:text-zinc-400">
                      {Object.entries(resource.properties).map(([key, value]) => (
                        <div key={key} className="font-mono">
                          {key}: {String(value)}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* YAML Preview */}
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
        <h3 className="mb-2 text-sm font-medium text-zinc-900 dark:text-white">
          Generated Score Specification
        </h3>
        <pre className="overflow-x-auto rounded border border-zinc-200 bg-zinc-50 p-3 text-xs dark:border-zinc-800 dark:bg-zinc-900">
          <code className="text-zinc-900 dark:text-white">{yaml}</code>
        </pre>
      </div>

      {/* Error Display */}
      {error && (
        <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-900 dark:bg-red-950">
          <AlertCircle size={18} className="mt-0.5 text-red-600 dark:text-red-400" />
          <div className="flex-1">
            <p className="text-sm font-medium text-red-600 dark:text-red-400">Deployment Failed</p>
            <p className="mt-1 text-xs text-red-600 dark:text-red-400">{error}</p>
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex justify-between gap-3">
        <button
          type="button"
          onClick={onPrev}
          disabled={deploying}
          className="rounded-lg border border-zinc-300 px-4 py-2 text-sm font-medium hover:bg-zinc-50 disabled:opacity-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
        >
          Back
        </button>
        <button
          type="button"
          onClick={handleDeploy}
          disabled={deploying}
          className="flex items-center gap-2 rounded-lg bg-lime-500 px-4 py-2 text-sm font-medium text-white hover:bg-lime-600 disabled:opacity-50"
        >
          {deploying ? (
            <>
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></div>
              Deploying...
            </>
          ) : (
            'Deploy Application'
          )}
        </button>
      </div>
    </div>
  );
}
