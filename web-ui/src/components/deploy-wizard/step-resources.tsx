import { useState, useEffect } from 'react';
import { Plus, X, Database, Box, FolderKey, GitBranch } from 'lucide-react';
import { StepProps, ProviderSummary, WorkflowSummary, ResourceRequest } from './types';
import { api } from '@/lib/api';

interface ResourceTypeInfo {
  type: string;
  providerName: string;
  workflow: WorkflowSummary;
}

export function StepResources({ data, onChange, onNext, onPrev }: StepProps) {
  const [providers, setProviders] = useState<ProviderSummary[]>([]);
  const [resourceTypes, setResourceTypes] = useState<ResourceTypeInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadProviders();
  }, []);

  const loadProviders = async () => {
    try {
      setLoading(true);
      const response = await api.getProviders();

      if (response.success && response.data) {
        const providersList = Array.isArray(response.data) ? response.data : [];
        setProviders(providersList);

        // Extract resource types from provisioner workflows
        const types: ResourceTypeInfo[] = [];
        providersList.forEach((provider) => {
          const provisioners =
            provider.workflows?.filter((w) => w.category === 'provisioner') || [];

          provisioners.forEach((workflow) => {
            // Extract resource type from workflow name (e.g., "provision-postgres" -> "postgres")
            const match = workflow.name.match(/provision-(.+)/);
            const resourceType = match ? match[1] : workflow.name;

            types.push({
              type: resourceType,
              providerName: provider.name,
              workflow: workflow,
            });
          });
        });

        setResourceTypes(types);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load providers');
    } finally {
      setLoading(false);
    }
  };

  const addResource = (resourceType: ResourceTypeInfo) => {
    const newResource: ResourceRequest = {
      name: `${resourceType.type}-${data.resources.length + 1}`,
      type: resourceType.type,
      properties: {},
    };

    // Set default values from workflow parameters
    resourceType.workflow.parameters?.forEach((param) => {
      if (param.default !== undefined) {
        newResource.properties[param.name] = param.default;
      }
    });

    onChange({
      resources: [...data.resources, newResource],
    });
  };

  const removeResource = (index: number) => {
    const newResources = data.resources.filter((_, i) => i !== index);
    onChange({ resources: newResources });
  };

  const updateResource = (index: number, updates: Partial<ResourceRequest>) => {
    const newResources = [...data.resources];
    newResources[index] = { ...newResources[index], ...updates };
    onChange({ resources: newResources });
  };

  const updateResourceProperty = (index: number, key: string, value: any) => {
    const newResources = [...data.resources];
    newResources[index].properties[key] = value;
    onChange({ resources: newResources });
  };

  const getResourceTypeInfo = (type: string): ResourceTypeInfo | undefined => {
    return resourceTypes.find((rt) => rt.type === type);
  };

  const getIconForResourceType = (type: string) => {
    if (type.includes('postgres') || type.includes('database')) {
      return <Database size={20} className="text-lime-600" />;
    }
    if (type.includes('s3') || type.includes('storage') || type.includes('bucket')) {
      return <Box size={20} className="text-lime-600" />;
    }
    if (type.includes('vault') || type.includes('secret')) {
      return <FolderKey size={20} className="text-lime-600" />;
    }
    if (type.includes('repo') || type.includes('git')) {
      return <GitBranch size={20} className="text-lime-600" />;
    }
    return <Box size={20} className="text-lime-600" />;
  };

  const renderParameterInput = (
    resource: ResourceRequest,
    resourceIndex: number,
    param: {
      name: string;
      type: string;
      required: boolean;
      default?: any;
      description?: string;
      enum?: string[];
    }
  ) => {
    const value = resource.properties[param.name] ?? param.default ?? '';

    if (param.enum && param.enum.length > 0) {
      return (
        <select
          value={value}
          onChange={(e) => updateResourceProperty(resourceIndex, param.name, e.target.value)}
          className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
        >
          <option value="">Select {param.name}</option>
          {param.enum.map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
      );
    }

    if (param.type === 'boolean') {
      return (
        <label className="mt-1 flex items-center gap-2">
          <input
            type="checkbox"
            checked={value === true || value === 'true'}
            onChange={(e) => updateResourceProperty(resourceIndex, param.name, e.target.checked)}
            className="rounded border-zinc-300 focus:ring-lime-500"
          />
          <span className="text-sm text-zinc-600 dark:text-zinc-400">Enable {param.name}</span>
        </label>
      );
    }

    if (param.type === 'number') {
      return (
        <input
          type="number"
          value={value}
          onChange={(e) =>
            updateResourceProperty(resourceIndex, param.name, parseFloat(e.target.value) || 0)
          }
          className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
        />
      );
    }

    // Default: string input
    return (
      <input
        type="text"
        value={value}
        onChange={(e) => updateResourceProperty(resourceIndex, param.name, e.target.value)}
        placeholder={param.default?.toString() || ''}
        className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
      />
    );
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onNext();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-lime-500 border-r-transparent"></div>
          <p className="mt-4 text-sm text-zinc-600 dark:text-zinc-400">Loading providers...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900 dark:bg-red-950">
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-zinc-900 dark:text-white">
          Resource Requirements
        </h2>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
          Add resources your application needs (databases, storage, etc.)
        </p>
      </div>

      {/* Added Resources */}
      {data.resources.length > 0 && (
        <div className="space-y-4">
          <h3 className="text-sm font-medium text-zinc-700 dark:text-zinc-300">
            Selected Resources ({data.resources.length})
          </h3>
          {data.resources.map((resource, index) => {
            const typeInfo = getResourceTypeInfo(resource.type);
            return (
              <div
                key={index}
                className="rounded-lg border border-zinc-200 bg-zinc-50 p-4 dark:border-zinc-800 dark:bg-zinc-900"
              >
                <div className="mb-3 flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    {getIconForResourceType(resource.type)}
                    <div>
                      <h4 className="text-sm font-medium text-zinc-900 dark:text-white">
                        {resource.type}
                      </h4>
                      {typeInfo && (
                        <p className="text-xs text-zinc-500">via {typeInfo.providerName}</p>
                      )}
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => removeResource(index)}
                    className="text-red-500 hover:text-red-600"
                  >
                    <X size={18} />
                  </button>
                </div>

                {/* Resource Name */}
                <div className="mb-3">
                  <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300">
                    Resource Identifier *
                  </label>
                  <input
                    type="text"
                    value={resource.name}
                    onChange={(e) => updateResource(index, { name: e.target.value })}
                    placeholder="db"
                    className="mt-1 block w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm focus:border-lime-500 focus:outline-none focus:ring-1 focus:ring-lime-500 dark:border-zinc-700 dark:bg-zinc-950 dark:text-white"
                  />
                  <p className="mt-1 text-xs text-zinc-500">
                    Short name used to reference this resource (e.g., &quot;db&quot;,
                    &quot;storage&quot;)
                  </p>
                </div>

                {/* Dynamic Parameters */}
                {typeInfo?.workflow.parameters && typeInfo.workflow.parameters.length > 0 && (
                  <div className="space-y-3">
                    <h5 className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                      Configuration
                    </h5>
                    {typeInfo.workflow.parameters.map((param) => (
                      <div key={param.name}>
                        <label className="block text-xs font-medium text-zinc-700 dark:text-zinc-300">
                          {param.name}
                          {param.required && <span className="text-red-500"> *</span>}
                        </label>
                        {renderParameterInput(resource, index, param)}
                        {param.description && (
                          <p className="mt-1 text-xs text-zinc-500">{param.description}</p>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Available Resource Types */}
      <div>
        <h3 className="mb-3 text-sm font-medium text-zinc-700 dark:text-zinc-300">
          Available Resource Types
        </h3>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {resourceTypes.map((resourceType, index) => (
            <button
              key={index}
              type="button"
              onClick={() => addResource(resourceType)}
              className="flex items-center gap-3 rounded-lg border border-zinc-300 bg-white p-3 text-left transition-colors hover:border-lime-500 hover:bg-lime-50 dark:border-zinc-700 dark:bg-zinc-950 dark:hover:bg-lime-950"
            >
              {getIconForResourceType(resourceType.type)}
              <div className="flex-1">
                <div className="text-sm font-medium text-zinc-900 dark:text-white">
                  {resourceType.type}
                </div>
                <div className="text-xs text-zinc-500">{resourceType.providerName}</div>
                {resourceType.workflow.description && (
                  <div className="mt-1 text-xs text-zinc-600 dark:text-zinc-400">
                    {resourceType.workflow.description}
                  </div>
                )}
              </div>
              <Plus size={16} className="text-lime-600" />
            </button>
          ))}
        </div>
        {resourceTypes.length === 0 && (
          <p className="text-sm text-zinc-500">
            No resource types available. Make sure providers are configured.
          </p>
        )}
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
          Next: Review
        </button>
      </div>
    </form>
  );
}
