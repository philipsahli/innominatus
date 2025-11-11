import { useState } from 'react';
import { Plus, X } from 'lucide-react';
import { StepProps } from './types';

export function StepContainer({ data, onChange, onNext, onPrev }: StepProps) {
  const [newEnvKey, setNewEnvKey] = useState('');
  const [newEnvValue, setNewEnvValue] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (data.container.image.trim()) {
      onNext();
    }
  };

  const addEnvVar = () => {
    if (newEnvKey.trim() && newEnvValue.trim()) {
      onChange({
        container: {
          ...data.container,
          envVars: {
            ...data.container.envVars,
            [newEnvKey]: newEnvValue,
          },
        },
      });
      setNewEnvKey('');
      setNewEnvValue('');
    }
  };

  const removeEnvVar = (key: string) => {
    const { [key]: _, ...rest } = data.container.envVars;
    onChange({
      container: {
        ...data.container,
        envVars: rest,
      },
    });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">
          Container Configuration
        </h2>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Configure your container image and runtime settings
        </p>
      </div>

      {/* Container Image */}
      <div>
        <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Container Image *
        </label>
        <input
          type="text"
          required
          value={data.container.image}
          onChange={(e) =>
            onChange({
              container: { ...data.container, image: e.target.value },
            })
          }
          placeholder="nginx:latest"
          className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
        />
        <p className="mt-1 text-xs text-zinc-500">
          Docker image name and tag (e.g., nginx:latest, myapp:v1.0)
        </p>
      </div>

      {/* Container Port */}
      <div>
        <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Container Port
        </label>
        <input
          type="number"
          value={data.container.port}
          onChange={(e) =>
            onChange({
              container: { ...data.container, port: parseInt(e.target.value) || 8080 },
            })
          }
          className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
        />
        <p className="mt-1 text-xs text-zinc-500">
          Port your application listens on (default: 8080)
        </p>
      </div>

      {/* Environment Variables */}
      <div>
        <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Environment Variables
        </label>

        {/* Existing Env Vars */}
        {Object.entries(data.container.envVars).length > 0 && (
          <div className="mt-2 space-y-2">
            {Object.entries(data.container.envVars).map(([key, value]) => (
              <div
                key={key}
                className="flex items-center gap-2 rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-2 dark:border-zinc-800 dark:bg-zinc-900"
              >
                <code className="flex-1 text-xs">
                  <span className="font-mono text-zinc-900 dark:text-white">{key}</span>
                  <span className="mx-2 text-zinc-400">=</span>
                  <span className="font-mono text-zinc-600 dark:text-zinc-400">{value}</span>
                </code>
                <button
                  type="button"
                  onClick={() => removeEnvVar(key)}
                  className="text-red-500 hover:text-red-600"
                >
                  <X size={16} />
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Add New Env Var */}
        <div className="mt-2 flex gap-2">
          <input
            type="text"
            value={newEnvKey}
            onChange={(e) => setNewEnvKey(e.target.value)}
            placeholder="KEY"
            className="flex-1 rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
          />
          <input
            type="text"
            value={newEnvValue}
            onChange={(e) => setNewEnvValue(e.target.value)}
            placeholder="value"
            className="flex-1 rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
          />
          <button
            type="button"
            onClick={addEnvVar}
            className="flex items-center gap-1 rounded-lg border border-lime-500 px-3 py-2 text-sm text-lime-600 hover:bg-lime-50 dark:text-lime-400 dark:hover:bg-lime-950"
          >
            <Plus size={16} />
            Add
          </button>
        </div>
        <p className="mt-1 text-xs text-zinc-500">
          Note: Resource connection details will be auto-injected
        </p>
      </div>

      {/* Resource Requests (Optional) */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
            CPU Request
          </label>
          <input
            type="text"
            value={data.container.cpuRequest || ''}
            onChange={(e) =>
              onChange({
                container: { ...data.container, cpuRequest: e.target.value },
              })
            }
            placeholder="100m"
            className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
            Memory Request
          </label>
          <input
            type="text"
            value={data.container.memoryRequest || ''}
            onChange={(e) =>
              onChange({
                container: { ...data.container, memoryRequest: e.target.value },
              })
            }
            placeholder="128Mi"
            className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
          />
        </div>
      </div>

      {/* Actions */}
      <div className="flex justify-between gap-3">
        <button
          type="button"
          onClick={onPrev}
          className="rounded-lg border border-zinc-300 px-4 py-2 text-sm font-medium hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-900"
        >
          Back
        </button>
        <button
          type="submit"
          className="rounded-lg bg-lime-500 px-4 py-2 text-sm font-medium text-white hover:bg-lime-600"
        >
          Next: Resources
        </button>
      </div>
    </form>
  );
}
